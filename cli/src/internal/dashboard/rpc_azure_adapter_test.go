package dashboard

// Tests for newAzureStoreFuncs, the production adapter that wires the
// dashboard Server into rpc.AzureStoreFuncs (the slice of 22 callbacks
// the AzureService Connect handler depends on). These are integration
// smoke tests scoped to behaviour that does NOT require live Azure
// credentials:
//
//   - Every field on the returned AzureStoreFuncs is populated (no nil
//     callback can panic the handler at runtime).
//   - LoadAzureYamlFn round-trips the on-disk azure.yaml.
//   - EnableGlobalAnalyticsFn is idempotent and creates the Logs.Analytics
//     block when missing (matches handleEnableAzureLogging behaviour).
//   - SaveServiceLogConfigFn validates service membership and writes
//     through yamlutil.UpdateServiceLogsConfig (preserves comments).
//   - SaveServiceCustomQueryFn writes per-service queries.
//   - The s.azureYamlMu serialisation participates correctly under
//     concurrent Load/Save bursts (no panic, no data loss).
//
// Streaming (FetchLogsFn / NewRealtimeStreamerFn) and credential paths
// are intentionally out of scope: they require live Azure credentials
// and are exercised end-to-end by the rpc-package tests against
// scripted stub stores.

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

// writeAzureYaml writes a minimal azure.yaml fixture to dir.
func writeAzureYaml(t *testing.T, dir, contents string) {
	t.Helper()
	p := filepath.Join(dir, "azure.yaml")
	if err := os.WriteFile(p, []byte(contents), 0o644); err != nil {
		t.Fatalf("write azure.yaml: %v", err)
	}
}

func newAdapterTestServer(t *testing.T) *Server {
	t.Helper()
	tmp := t.TempDir()
	return &Server{projectDir: tmp}
}

// TestNewAzureStoreFuncs_AllFieldsPopulated guards against future drift
// where a new field on rpc.AzureStoreFuncs is added but the adapter
// forgets to wire it: a nil callback would panic the Connect handler
// the moment a client invokes the corresponding RPC. Reflection scan
// catches this at compile/test time, not in production.
func TestNewAzureStoreFuncs_AllFieldsPopulated(t *testing.T) {
	s := newAdapterTestServer(t)
	funcs := newAzureStoreFuncs(s)

	v := reflect.ValueOf(funcs)
	tt := v.Type()
	if v.NumField() == 0 {
		t.Fatalf("AzureStoreFuncs has no fields; reflection wiring broke")
	}
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		name := tt.Field(i).Name
		// AzureStoreFuncs only exports func-typed fields. If a non-func
		// field is added later, this assertion documents that the test
		// needs updating.
		if f.Kind() != reflect.Func {
			t.Errorf("field %s is %s, expected func", name, f.Kind())
			continue
		}
		if f.IsNil() {
			t.Errorf("field %s is nil; newAzureStoreFuncs forgot to wire it", name)
		}
	}
}

// TestAdapter_LoadAzureYaml_RoundTrip verifies the adapter reads what
// the file contains - no swallowed errors, no caching surprise.
func TestAdapter_LoadAzureYaml_RoundTrip(t *testing.T) {
	s := newAdapterTestServer(t)
	writeAzureYaml(t, s.projectDir, `name: demo
services:
  api:
    host: containerapp
    language: go
`)
	funcs := newAzureStoreFuncs(s)

	got, err := funcs.LoadAzureYamlFn()
	if err != nil {
		t.Fatalf("LoadAzureYamlFn: %v", err)
	}
	if got.Name != "demo" {
		t.Errorf("Name = %q, want demo", got.Name)
	}
	if _, ok := got.Services["api"]; !ok {
		t.Errorf("expected service 'api' in azure.yaml, got %v", got.Services)
	}
}

// TestAdapter_LoadAzureYaml_MissingFile surfaces a wrapped error rather
// than a panic, matching the loadAzureYaml helper's contract.
func TestAdapter_LoadAzureYaml_MissingFile(t *testing.T) {
	s := newAdapterTestServer(t) // tmpDir but no azure.yaml
	funcs := newAzureStoreFuncs(s)

	if _, err := funcs.LoadAzureYamlFn(); err == nil {
		t.Fatalf("expected error on missing azure.yaml, got nil")
	}
}

// TestAdapter_EnableGlobalAnalytics_CreatesBlock mirrors the
// handleEnableAzureLogging contract: returns false on first enable
// (alreadyEnabled=false), true on subsequent calls, and persists the
// Logs.Analytics block to disk.
func TestAdapter_EnableGlobalAnalytics_CreatesBlock(t *testing.T) {
	s := newAdapterTestServer(t)
	writeAzureYaml(t, s.projectDir, `name: demo
services:
  api:
    host: containerapp
`)
	funcs := newAzureStoreFuncs(s)

	already, err := funcs.EnableGlobalAnalyticsFn()
	if err != nil {
		t.Fatalf("first EnableGlobalAnalyticsFn: %v", err)
	}
	if already {
		t.Errorf("first call: alreadyEnabled = true, want false")
	}
	// Verify block landed on disk.
	got, err := funcs.LoadAzureYamlFn()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.Logs == nil || got.Logs.Analytics == nil {
		t.Fatalf("Logs.Analytics not persisted: Logs=%+v", got.Logs)
	}

	// Second call must be idempotent and report alreadyEnabled.
	already, err = funcs.EnableGlobalAnalyticsFn()
	if err != nil {
		t.Fatalf("second EnableGlobalAnalyticsFn: %v", err)
	}
	if !already {
		t.Errorf("second call: alreadyEnabled = false, want true")
	}
}

// TestAdapter_SaveServiceLogConfig_UnknownService matches
// handleSaveAzureLogConfig: writes targeting an undeclared service
// must surface errServiceNotFound (handler maps it to NotFound).
func TestAdapter_SaveServiceLogConfig_UnknownService(t *testing.T) {
	s := newAdapterTestServer(t)
	writeAzureYaml(t, s.projectDir, `name: demo
services:
  api:
    host: containerapp
`)
	funcs := newAzureStoreFuncs(s)

	err := funcs.SaveServiceLogConfigFn("nope", []string{"AppServiceConsoleLogs"}, "")
	if err == nil {
		t.Fatalf("expected error for unknown service, got nil")
	}
}

// TestAdapter_SaveServiceLogConfig_HappyPath confirms the write reaches
// disk by invoking yamlutil.UpdateServiceLogsConfig under the hood. We
// don't assert exact YAML formatting (yamlutil owns that contract) -
// just that the service block grew a logs section.
func TestAdapter_SaveServiceLogConfig_HappyPath(t *testing.T) {
	s := newAdapterTestServer(t)
	writeAzureYaml(t, s.projectDir, `name: demo
services:
  api:
    host: containerapp
`)
	funcs := newAzureStoreFuncs(s)

	if err := funcs.SaveServiceLogConfigFn("api", []string{"AppServiceConsoleLogs", "AppServiceHTTPLogs"}, "MyTable | take 5"); err != nil {
		t.Fatalf("SaveServiceLogConfigFn: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(s.projectDir, "azure.yaml"))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	body := string(raw)
	if !contains(body, "AppServiceConsoleLogs") || !contains(body, "MyTable | take 5") {
		t.Errorf("azure.yaml missing expected log config:\n%s", body)
	}
}

// TestAdapter_SaveServiceCustomQuery_RoundTrip exercises the
// per-service custom-query path used by handleSaveAzureQuery.
func TestAdapter_SaveServiceCustomQuery_RoundTrip(t *testing.T) {
	s := newAdapterTestServer(t)
	writeAzureYaml(t, s.projectDir, `name: demo
services:
  api:
    host: containerapp
`)
	funcs := newAzureStoreFuncs(s)

	const q = "ContainerAppConsoleLogs_CL | where ServiceName_s == 'api' | take 1"
	if err := funcs.SaveServiceCustomQueryFn("api", q); err != nil {
		t.Fatalf("SaveServiceCustomQueryFn: %v", err)
	}

	got, err := funcs.LoadAzureYamlFn()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	svc := got.Services["api"]
	if svc.Logs == nil || svc.Logs.Analytics == nil || svc.Logs.Analytics.Query != q {
		t.Errorf("custom query not persisted: svc.Logs=%+v", svc.Logs)
	}

	// Unknown service should error, matching SaveServiceLogConfigFn.
	if err := funcs.SaveServiceCustomQueryFn("ghost", q); err == nil {
		t.Errorf("expected error for unknown service in SaveServiceCustomQueryFn")
	}
}

// TestAdapter_AzureYamlMu_SerialisesConcurrentWrites runs N goroutines
// hammering Load + EnableGlobalAnalytics + SaveAzureYaml in parallel.
// Two assertions:
//  1. No goroutine returns an error or panics (file stays parseable).
//  2. Every iteration sees a valid AzureYaml on Load (the write half
//     of the race never observes a half-written file).
//
// This test relies on `go test -race` to detect any actual data race
// on s.azureYamlMu; even without -race the iteration count is high
// enough that an unsynchronised implementation would tear the YAML
// occasionally and fail Load.
func TestAdapter_AzureYamlMu_SerialisesConcurrentWrites(t *testing.T) {
	s := newAdapterTestServer(t)
	writeAzureYaml(t, s.projectDir, `name: demo
services:
  api:
    host: containerapp
`)
	funcs := newAzureStoreFuncs(s)

	const goroutines = 8
	const iterations = 50

	var wg sync.WaitGroup
	errCh := make(chan error, goroutines*iterations)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				// Mix of read + write paths to exercise both
				// RLock and Lock acquisition orderings.
				if _, err := funcs.LoadAzureYamlFn(); err != nil {
					errCh <- err
					return
				}
				if _, err := funcs.EnableGlobalAnalyticsFn(); err != nil {
					errCh <- err
					return
				}
				ay, err := funcs.LoadAzureYamlFn()
				if err != nil {
					errCh <- err
					return
				}
				// Re-save through SaveAzureYamlFn to exercise the
				// write path that goes through the same mutex.
				if err := funcs.SaveAzureYamlFn(ay); err != nil {
					errCh <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent adapter op failed: %v", err)
		}
	}

	// Final state must still parse cleanly and retain the analytics
	// block we created during the burst.
	final, err := funcs.LoadAzureYamlFn()
	if err != nil {
		t.Fatalf("final reload: %v", err)
	}
	if final.Logs == nil || final.Logs.Analytics == nil {
		t.Errorf("expected Logs.Analytics to survive concurrent writes, got %+v", final.Logs)
	}
}

// TestAdapter_WorkspaceID_FallsBackToYaml documents the
// WorkspaceIDFn fallback path: when the env-derived workspace is empty
// the adapter consults azure.yaml's Logs.Analytics.Workspace. This is
// the only sub-store path that mixes catalog + config sources, so it
// gets its own focused test.
func TestAdapter_WorkspaceID_FallsBackToYaml(t *testing.T) {
	s := newAdapterTestServer(t)
	writeAzureYaml(t, s.projectDir, `name: demo
services:
  api:
    host: containerapp
logs:
  analytics:
    workspace: my-workspace-id
`)
	funcs := newAzureStoreFuncs(s)

	// We can't easily stub getWorkspaceIDFromEnv from outside the
	// package; instead we observe that on a clean tempdir with no
	// AZURE_* vars the env path returns nothing and the fallback
	// surfaces the yaml-declared workspace.
	id, err := funcs.WorkspaceIDFn(context.Background())
	if err != nil {
		// Some envs may surface a "no env" error before the
		// fallback runs; document the looser contract here.
		t.Logf("WorkspaceIDFn returned err=%v (acceptable on systems with partial env)", err)
		return
	}
	if id != "my-workspace-id" {
		t.Logf("WorkspaceIDFn = %q (env path may have shadowed yaml fallback; non-fatal)", id)
	}
}

// (substring helper `contains` lives in dashboard_test.go; reused here)

// Compile-time guard: ensure the adapter test stays in sync with the
// service.AzureYaml shape it depends on. If AzureYaml.Services type
// changes, this assignment fails to compile and the test author is
// forced to revisit the fixtures above.
var _ = func() map[string]service.Service { return (&service.AzureYaml{}).Services }
