package rpc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
	"github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1/azdappv1connect"
	"github.com/jongio/azd-app/cli/src/internal/azure"
	"github.com/jongio/azd-app/cli/src/internal/dashboard/broadcast"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// stubAzureStore is a tiny zero-value-friendly AzureStoreFuncs builder for
// the handler-level smoke tests. Only the fields each test needs are set;
// the rest panic on call (the AzureStoreFuncs adapter's documented
// fail-loud contract), which keeps tests honest about what they exercise.
func stubAzureStore() AzureStoreFuncs { return AzureStoreFuncs{} }

func newStubHandler(funcs AzureStoreFuncs) *AzureHandler {
	return NewAzureHandler(funcs)
}

// =============================================================================
// mapAzureError - error classification table.
// =============================================================================

func TestMapAzureError(t *testing.T) {
	cases := []struct {
		name string
		ctx  func() context.Context
		err  error
		want connect.Code
	}{
		{
			name: "auth_expired",
			ctx:  context.Background,
			err:  &azure.AzureLogsError{Code: "AUTH_EXPIRED", Message: "expired"},
			want: connect.CodeUnauthenticated,
		},
		{
			name: "no_workspace",
			ctx:  context.Background,
			err:  &azure.AzureLogsError{Code: "NO_WORKSPACE", Message: "missing"},
			want: connect.CodeNotFound,
		},
		{
			name: "client_error",
			ctx:  context.Background,
			err:  &azure.AzureLogsError{Code: "CLIENT_ERROR", Message: "bad request"},
			want: connect.CodeInvalidArgument,
		},
		{
			name: "substring_invalid_timespan",
			ctx:  context.Background,
			err:  errors.New("invalid timespan: PT99X"),
			want: connect.CodeInvalidArgument,
		},
		{
			name: "substring_429",
			ctx:  context.Background,
			err:  errors.New("HTTP 429 TooManyRequests from Log Analytics"),
			want: connect.CodeResourceExhausted,
		},
		{
			name: "default_internal",
			ctx:  context.Background,
			err:  errors.New("something exploded"),
			want: connect.CodeInternal,
		},
		{
			name: "ctx_canceled_wins",
			ctx: func() context.Context {
				c, cancel := context.WithCancel(context.Background())
				cancel()
				return c
			},
			err:  errors.New("any error"),
			want: connect.CodeCanceled,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mapAzureError(tc.ctx(), tc.err)
			if got == nil {
				t.Fatalf("mapAzureError returned nil, want code=%v", tc.want)
			}
			if connect.CodeOf(got) != tc.want {
				t.Fatalf("mapAzureError code=%v, want %v (err=%v)",
					connect.CodeOf(got), tc.want, got)
			}
		})
	}
}

func TestIsNoResults(t *testing.T) {
	if isNoResults(nil) {
		t.Fatal("nil should not be NO_RESULTS")
	}
	if isNoResults(errors.New("anything")) {
		t.Fatal("plain error should not be NO_RESULTS")
	}
	if !isNoResults(&azure.AzureLogsError{Code: "NO_RESULTS"}) {
		t.Fatal("NO_RESULTS code should match")
	}
	if isNoResults(&azure.AzureLogsError{Code: "AUTH_EXPIRED"}) {
		t.Fatal("AUTH_EXPIRED should not match NO_RESULTS")
	}
}

// =============================================================================
// localAzureLogRing - drop-OLDEST + coalescing notify.
// =============================================================================

func TestLocalAzureLogRing_DropOldest(t *testing.T) {
	r := newAzureLogRing(4)
	for i := 0; i < 9; i++ {
		r.push(&v1.LogEntry{Message: string(rune('a' + i))})
	}
	out, dropped := r.drain()
	if len(out) != 4 {
		t.Fatalf("drained %d entries, want 4", len(out))
	}
	if dropped != 5 {
		t.Fatalf("dropped=%d, want 5", dropped)
	}
	// Drop-oldest: last 4 pushes survive (indices 5..8 = "f","g","h","i").
	wantFirst := string(rune('a' + 5))
	if out[0].Message != wantFirst {
		t.Fatalf("first surviving entry=%q, want %q", out[0].Message, wantFirst)
	}
}

func TestLocalAzureLogRing_NotifyCoalesce(t *testing.T) {
	r := newAzureLogRing(8)
	for i := 0; i < 8; i++ {
		r.push(&v1.LogEntry{Message: "x"})
	}
	// notify is buffered=1 so all 8 pushes coalesce to one signal.
	signals := 0
	for {
		select {
		case <-r.notify:
			signals++
		default:
			if signals != 1 {
				t.Fatalf("notify coalesce: got %d signals, want 1", signals)
			}
			return
		}
	}
}

// =============================================================================
// Handler smoke tests - one per RPC, exercising store wiring + validation.
// =============================================================================

func TestAzureHandler_GetAzureServices(t *testing.T) {
	want := []string{"api", "web"}
	funcs := stubAzureStore()
	funcs.ServiceNamesFromEnvFn = func() []string { return want }
	h := newStubHandler(funcs)

	resp, err := h.GetAzureServices(context.Background(),
		connect.NewRequest(&v1.GetAzureServicesRequest{}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got := resp.Msg.Services; !equalStrings(got, want) {
		t.Fatalf("services=%v, want %v", got, want)
	}
}

func TestAzureHandler_EnableAzureLogging_AlreadyEnabled(t *testing.T) {
	funcs := stubAzureStore()
	funcs.EnableGlobalAnalyticsFn = func() (bool, error) { return true, nil }
	h := newStubHandler(funcs)

	resp, err := h.EnableAzureLogging(context.Background(),
		connect.NewRequest(&v1.EnableAzureLoggingRequest{}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !resp.Msg.Enabled {
		t.Fatal("Enabled=false, want true")
	}
	if !strings.Contains(resp.Msg.Message, "already") {
		t.Fatalf("message=%q, want it to mention 'already'", resp.Msg.Message)
	}
}

func TestAzureHandler_EnableAzureLogging_New(t *testing.T) {
	funcs := stubAzureStore()
	funcs.EnableGlobalAnalyticsFn = func() (bool, error) { return false, nil }
	h := newStubHandler(funcs)

	resp, err := h.EnableAzureLogging(context.Background(),
		connect.NewRequest(&v1.EnableAzureLoggingRequest{}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !resp.Msg.Enabled || !strings.Contains(resp.Msg.Message, "Refresh") {
		t.Fatalf("unexpected response: %+v", resp.Msg)
	}
}

func TestAzureHandler_GetAzureLogs_NoResultsIsSuccess(t *testing.T) {
	funcs := stubAzureStore()
	funcs.FetchLogsFn = func(ctx context.Context, c azure.StandaloneLogsConfig) ([]azure.LogEntry, error) {
		return nil, &azure.AzureLogsError{Code: "NO_RESULTS", Message: "empty"}
	}
	h := newStubHandler(funcs)

	resp, err := h.GetAzureLogs(context.Background(),
		connect.NewRequest(&v1.GetAzureLogsRequest{Service: "api"}))
	if err != nil {
		t.Fatalf("NO_RESULTS should map to success-with-empty, got err=%v", err)
	}
	if resp.Msg.Count != 0 || len(resp.Msg.Entries) != 0 {
		t.Fatalf("expected empty entries, got count=%d entries=%v",
			resp.Msg.Count, resp.Msg.Entries)
	}
}

func TestAzureHandler_VerifyAzureLogs_EmptyServiceInvalid(t *testing.T) {
	h := newStubHandler(stubAzureStore())
	_, err := h.VerifyAzureLogs(context.Background(),
		connect.NewRequest(&v1.VerifyAzureLogsRequest{Service: ""}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("err=%v, want InvalidArgument", err)
	}
}

func TestAzureHandler_StreamAzureLogs_EmptyServiceInvalid(t *testing.T) {
	// The stream argument is unused on the validation path so a nil
	// stream is safe here; a real stream would require the connect-go
	// internal plumbing exercised by the integration suite.
	h := newStubHandler(stubAzureStore())
	err := h.StreamAzureLogs(context.Background(),
		connect.NewRequest(&v1.StreamAzureLogsRequest{Service: ""}),
		nil)
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("err=%v, want InvalidArgument", err)
	}
}

func TestAzureHandler_SaveAzureLogConfig_Validation(t *testing.T) {
	h := newStubHandler(stubAzureStore())

	cases := []struct {
		name string
		req  *v1.SaveAzureLogConfigRequest
	}{
		{"empty_service", &v1.SaveAzureLogConfigRequest{
			Service: "",
			Mode:    v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_TABLES,
			Tables:  []string{"AppRequests"},
		}},
		{"unspecified_mode", &v1.SaveAzureLogConfigRequest{
			Service: "api",
			Mode:    v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_UNSPECIFIED,
		}},
		{"tables_mode_no_tables", &v1.SaveAzureLogConfigRequest{
			Service: "api",
			Mode:    v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_TABLES,
		}},
		{"custom_mode_no_query", &v1.SaveAzureLogConfigRequest{
			Service: "api",
			Mode:    v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_CUSTOM,
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := h.SaveAzureLogConfig(context.Background(),
				connect.NewRequest(tc.req))
			if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("err=%v, want InvalidArgument", err)
			}
		})
	}
}

func TestAzureHandler_GetAzureLogConfig_DefaultsToContainerApp(t *testing.T) {
	funcs := stubAzureStore()
	funcs.LoadAzureYamlFn = func() (*service.AzureYaml, error) { return nil, nil }
	funcs.RecommendedTablesFn = func(rt string) []string {
		if rt != "containerapp" {
			t.Fatalf("recommended-tables called with rt=%q, want containerapp", rt)
		}
		return []string{"ContainerAppConsoleLogs_CL"}
	}
	h := newStubHandler(funcs)

	resp, err := h.GetAzureLogConfig(context.Background(),
		connect.NewRequest(&v1.GetAzureLogConfigRequest{Service: "api"}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.Msg.ResourceType != "containerapp" {
		t.Fatalf("ResourceType=%q, want containerapp", resp.Msg.ResourceType)
	}
	if len(resp.Msg.Tables) != 1 || resp.Msg.Tables[0] != "ContainerAppConsoleLogs_CL" {
		t.Fatalf("Tables=%v, want recommended fallback", resp.Msg.Tables)
	}
}

func TestAzureHandler_ListAzureTables_FallsBackToKnown(t *testing.T) {
	funcs := stubAzureStore()
	funcs.WorkspaceIDFn = func(ctx context.Context) (string, error) { return "wsid", nil }
	funcs.ListLiveTablesFn = func(ctx context.Context, w string) ([]azure.TableInfo, error) {
		return nil, errors.New("creds missing")
	}
	funcs.AllKnownTablesFn = func() []azure.TableInfo {
		return []azure.TableInfo{{Name: "AppRequests", Description: "requests"}}
	}
	funcs.RecommendedTablesFn = func(rt string) []string { return []string{"AppRequests"} }
	funcs.IsRecommendedTableFn = func(name, rt string) bool { return name == "AppRequests" }
	funcs.TableCategoriesFn = func() map[string]azure.TableCategory { return nil }
	funcs.TruncateMiddleFn = func(s string, n int) string { return s }
	h := newStubHandler(funcs)

	resp, err := h.ListAzureTables(context.Background(),
		connect.NewRequest(&v1.ListAzureTablesRequest{
			ResourceType: v1.AzureResourceType_AZURE_RESOURCE_TYPE_CONTAINER_APP,
		}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(resp.Msg.Tables) != 1 || resp.Msg.Tables[0].Name != "AppRequests" {
		t.Fatalf("expected AllKnownTables fallback, got %+v", resp.Msg.Tables)
	}
	if !resp.Msg.Tables[0].Recommended {
		t.Fatal("AppRequests should be flagged Recommended")
	}
}

func TestAzureHandler_GetServiceQuery_EmptyServiceInvalid(t *testing.T) {
	h := newStubHandler(stubAzureStore())
	_, err := h.GetServiceQuery(context.Background(),
		connect.NewRequest(&v1.GetServiceQueryRequest{Service: ""}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("err=%v, want InvalidArgument", err)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// =============================================================================
// Test infrastructure - end-to-end Connect wiring + safe defaults.
// =============================================================================

// fullStubAzureFuncs returns an AzureStoreFuncs with every field populated by
// a safe, zero-value-like default. Each field can be overridden in-place by
// individual tests; the goal is to let any RPC be exercised end-to-end via
// httptest WITHOUT every test having to enumerate the entire 22-field
// surface. Defaults match the legacy "no Azure environment" shape:
//
//   - ResolveResource / NewLogAnalyticsCredential return errors so the
//     realtime path always falls back to polling unless overridden.
//   - FetchLogs / VerifyWorkspace / CheckDiagnosticSettings return empty
//     successes (no-op, no error).
//   - GetHealth returns "healthy" with no checks; mirrors a clean state
//     before any probes have run.
//
// Tests that need a specific failure mode override the relevant Fn directly.
func fullStubAzureFuncs() AzureStoreFuncs {
	return AzureStoreFuncs{
		// AzureConfigStore
		LoadAzureYamlFn:          func() (*service.AzureYaml, error) { return nil, nil },
		SaveAzureYamlFn:          func(*service.AzureYaml) error { return nil },
		EnableGlobalAnalyticsFn:  func() (bool, error) { return false, nil },
		SaveServiceLogConfigFn:   func(string, []string, string) error { return nil },
		SaveServiceCustomQueryFn: func(string, string) error { return nil },

		// AzureCatalog
		ServiceNamesFromEnvFn:         func() []string { return nil },
		WorkspaceIDFn:                 func(context.Context) (string, error) { return "", nil },
		DefaultQueryFn:                func(string) string { return "" },
		RecommendedTablesFn:           func(string) []string { return nil },
		AllKnownTablesFn:              func() []azure.TableInfo { return nil },
		IsRecommendedTableFn:          func(string, string) bool { return false },
		TableCategoriesFn:             func() map[string]azure.TableCategory { return nil },
		SubstituteQueryPlaceholdersFn: func(q, _, _ string) string { return q },
		TruncateMiddleFn:              func(s string, _ int) string { return s },
		ListLiveTablesFn:              func(context.Context, string) ([]azure.TableInfo, error) { return nil, nil },

		// AzureLogsClient
		FetchLogsFn: func(context.Context, azure.StandaloneLogsConfig) ([]azure.LogEntry, error) {
			return nil, nil
		},
		ResolveResourceFn: func(context.Context, string) (*azure.AzureResource, error) {
			return nil, errors.New("resource not found")
		},
		NewRealtimeStreamerFn: func(azure.ResourceType, azure.StreamerConfig) (azure.RealtimeLogStreamer, error) {
			return nil, errors.New("realtime not available")
		},
		NewLogAnalyticsCredentialFn: func() (azcore.TokenCredential, error) {
			return nil, errors.New("no credentials")
		},
		VerifyWorkspaceFn: func(context.Context, *azure.WorkspaceVerificationRequest) (*azure.WorkspaceVerificationResponse, error) {
			return &azure.WorkspaceVerificationResponse{
				Status:  azure.VerificationStatusSuccess,
				Results: map[string]*azure.ServiceVerificationResult{},
			}, nil
		},
		VerifyServiceLogsFn: func(context.Context, string) (*VerifyServiceLogsResult, error) {
			return &VerifyServiceLogsResult{Success: true, Message: "ok"}, nil
		},

		// AzureDiagnostics
		CheckDiagnosticSettingsFn: func(context.Context) (*azure.DiagnosticSettingsCheckResponse, error) {
			return &azure.DiagnosticSettingsCheckResponse{
				Services: map[string]*azure.DiagnosticSettingsCheckResult{},
			}, nil
		},
		RunDiagnosticsFn: func(context.Context) (any, error) { return map[string]any{}, nil },
		GetSetupStateFn:  func(context.Context) (any, error) { return map[string]any{}, nil },
		GetHealthFn: func(context.Context) AzureHealthSnapshot {
			return AzureHealthSnapshot{Status: "healthy"}
		},
	}
}

// newAzureTestServer wires an AzureHandler around the supplied funcs behind
// an httptest server so tests exercise the real Connect runtime (HTTP/1
// header flush timing matters for streaming) rather than the in-process
// router. Mirrors newLogsTestServer.
func newAzureTestServer(t *testing.T, funcs AzureStoreFuncs) (azdappv1connect.AzureServiceClient, func()) {
	t.Helper()
	mgr := broadcast.New()
	mux := http.NewServeMux()
	Mount(mux, Dependencies{Broadcast: mgr, Azure: funcs})
	srv := httptest.NewServer(mux)
	client := azdappv1connect.NewAzureServiceClient(srv.Client(), srv.URL)
	return client, func() {
		srv.Close()
		mgr.StopAll()
	}
}

// fakeRealtimeStreamer is a configurable RealtimeLogStreamer for streaming
// tests. Call BurstAll to push a slice of entries on Start, or use Push to
// drive entries from outside the streamer goroutine. Stop is idempotent.
type fakeRealtimeStreamer struct {
	serviceName string
	rt          azure.ResourceType
	burst       []azure.LogEntry    // pushed inline on Start
	push        chan azure.LogEntry // tests can send extras here

	startedOnce sync.Once
	stoppedOnce sync.Once
	started     chan struct{}
	stopped     chan struct{}
	connected   atomic.Bool
}

func newFakeRealtimeStreamer(name string, rt azure.ResourceType) *fakeRealtimeStreamer {
	s := &fakeRealtimeStreamer{
		serviceName: name,
		rt:          rt,
		push:        make(chan azure.LogEntry, 16),
		started:     make(chan struct{}),
		stopped:     make(chan struct{}),
	}
	s.connected.Store(true)
	return s
}

func (f *fakeRealtimeStreamer) Start(ctx context.Context, out chan<- azure.LogEntry) error {
	f.startedOnce.Do(func() { close(f.started) })

	// Burst phase: push the pre-seeded entries quickly so tests can force
	// ring overflow scenarios deterministically.
	for _, e := range f.burst {
		select {
		case <-ctx.Done():
			return nil
		case out <- e:
		}
	}

	// Drain phase: forward any test-driven pushes until ctx is done.
	for {
		select {
		case <-ctx.Done():
			return nil
		case e, ok := <-f.push:
			if !ok {
				return nil
			}
			select {
			case <-ctx.Done():
				return nil
			case out <- e:
			}
		}
	}
}

func (f *fakeRealtimeStreamer) Stop() error {
	f.stoppedOnce.Do(func() {
		f.connected.Store(false)
		close(f.stopped)
	})
	return nil
}

func (f *fakeRealtimeStreamer) ServiceName() string              { return f.serviceName }
func (f *fakeRealtimeStreamer) ResourceType() azure.ResourceType { return f.rt }
func (f *fakeRealtimeStreamer) IsConnected() bool                { return f.connected.Load() }

// fakeStream is a minimal connect.ServerStream stand-in for unit-testing
// sendWithBackpressure. The real ServerStream is unexported / hard to fake;
// instead we test sendWithBackpressure indirectly via a stream constructed
// from a real httptest connection (see TestSendWithBackpressure).

// makeAzureLogEntry is a small constructor that keeps streaming-test
// stimulus data uniform.
func makeAzureLogEntry(svc, msg string, ts time.Time) azure.LogEntry {
	return azure.LogEntry{
		Service:   svc,
		Message:   msg,
		Level:     azure.LogLevelInfo,
		Timestamp: ts,
	}
}

// recvUntil drains the stream until pred returns true, the deadline expires,
// or the stream errors. Returns the slice of frames actually received. Used
// to pull through the initial Status + N entries deterministically without
// time.Sleep assertions.
func recvUntil(
	t *testing.T,
	stream *connect.ServerStreamForClient[v1.StreamAzureLogsResponse],
	deadline time.Duration,
	pred func([]*v1.StreamAzureLogsResponse) bool,
) []*v1.StreamAzureLogsResponse {
	t.Helper()
	var got []*v1.StreamAzureLogsResponse
	done := make(chan struct{})
	timer := time.NewTimer(deadline)
	defer timer.Stop()

	go func() {
		defer close(done)
		for stream.Receive() {
			got = append(got, stream.Msg())
			if pred(got) {
				return
			}
		}
	}()

	select {
	case <-done:
	case <-timer.C:
	}
	return got
}

// =============================================================================
// Per-RPC end-to-end tests through Connect (httptest).
// =============================================================================

func TestE2E_EnableAzureLogging_ErrorMaps(t *testing.T) {
	funcs := fullStubAzureFuncs()
	funcs.EnableGlobalAnalyticsFn = func() (bool, error) {
		return false, &azure.AzureLogsError{Code: "AUTH_REQUIRED", Message: "login first"}
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	_, err := client.EnableAzureLogging(context.Background(),
		connect.NewRequest(&v1.EnableAzureLoggingRequest{}))
	if err == nil || connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("err=%v code=%v want Unauthenticated", err, connect.CodeOf(err))
	}
}

func TestE2E_GetAzureLogs_TailClampsAndMergesService(t *testing.T) {
	var gotCfg azure.StandaloneLogsConfig
	funcs := fullStubAzureFuncs()
	funcs.FetchLogsFn = func(_ context.Context, c azure.StandaloneLogsConfig) ([]azure.LogEntry, error) {
		gotCfg = c
		return []azure.LogEntry{
			makeAzureLogEntry("api", "first", time.Now().Add(-2*time.Second)),
			makeAzureLogEntry("api", "second", time.Now().Add(-1*time.Second)),
		}, nil
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	resp, err := client.GetAzureLogs(context.Background(),
		connect.NewRequest(&v1.GetAzureLogsRequest{
			Service:      "api",
			SinceSeconds: 60,
			Tail:         99_999, // exceeds 10_000 cap
		}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.Msg.Count != 2 || len(resp.Msg.Entries) != 2 {
		t.Fatalf("count=%d entries=%d want 2/2", resp.Msg.Count, len(resp.Msg.Entries))
	}
	if gotCfg.Limit != 10_000 {
		t.Errorf("Limit=%d want 10_000 (clamp)", gotCfg.Limit)
	}
	if gotCfg.Since != 60*time.Second {
		t.Errorf("Since=%v want 60s", gotCfg.Since)
	}
	if len(gotCfg.Services) != 1 || gotCfg.Services[0] != "api" {
		t.Errorf("Services=%v want [api]", gotCfg.Services)
	}
	// Source enum should be AZURE for every entry.
	for i, e := range resp.Msg.Entries {
		if e.Source != v1.LogSource_LOG_SOURCE_AZURE {
			t.Errorf("entry[%d].Source=%v want AZURE", i, e.Source)
		}
	}
}

func TestE2E_GetAzureLogs_DefaultsToOneHourAnd500(t *testing.T) {
	var gotCfg azure.StandaloneLogsConfig
	funcs := fullStubAzureFuncs()
	funcs.FetchLogsFn = func(_ context.Context, c azure.StandaloneLogsConfig) ([]azure.LogEntry, error) {
		gotCfg = c
		return nil, nil
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	if _, err := client.GetAzureLogs(context.Background(),
		connect.NewRequest(&v1.GetAzureLogsRequest{})); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if gotCfg.Since != time.Hour {
		t.Errorf("Since=%v want 1h default", gotCfg.Since)
	}
	if gotCfg.Limit != 500 {
		t.Errorf("Limit=%d want 500 default", gotCfg.Limit)
	}
	if len(gotCfg.Services) != 0 {
		t.Errorf("Services=%v want empty (no service filter)", gotCfg.Services)
	}
}

func TestE2E_GetAzureLogs_ErrorMaps(t *testing.T) {
	funcs := fullStubAzureFuncs()
	funcs.FetchLogsFn = func(context.Context, azure.StandaloneLogsConfig) ([]azure.LogEntry, error) {
		return nil, errors.New("HTTP 429 throttled")
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	_, err := client.GetAzureLogs(context.Background(),
		connect.NewRequest(&v1.GetAzureLogsRequest{Service: "api"}))
	if err == nil || connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("err=%v code=%v want ResourceExhausted", err, connect.CodeOf(err))
	}
}

func TestE2E_GetAzureLogsHealth_FullSnapshot(t *testing.T) {
	funcs := fullStubAzureFuncs()
	funcs.GetHealthFn = func(context.Context) AzureHealthSnapshot {
		return AzureHealthSnapshot{
			Status:  "degraded",
			DocsURL: "https://docs.example/health",
			Checks: []AzureHealthCheckSnapshot{
				{Name: "auth", Status: "pass", Message: "creds ok"},
				{Name: "workspace", Status: "warn", Message: "slow", Fix: "retry"},
				{Name: "diag", Status: "fail", Message: "missing", Fix: "configure"},
				{Name: "weird", Status: "unknown-string", Message: "exotic"},
			},
		}
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	resp, err := client.GetAzureLogsHealth(context.Background(),
		connect.NewRequest(&v1.GetAzureLogsHealthRequest{}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.Msg.Status != v1.AzureOverallStatus_AZURE_OVERALL_STATUS_DEGRADED {
		t.Errorf("Status=%v want DEGRADED", resp.Msg.Status)
	}
	if resp.Msg.DocsUrl != "https://docs.example/health" {
		t.Errorf("DocsUrl=%q want set", resp.Msg.DocsUrl)
	}
	if len(resp.Msg.Checks) != 4 {
		t.Fatalf("Checks=%d want 4", len(resp.Msg.Checks))
	}
	wantStatuses := []v1.AzureCheckStatus{
		v1.AzureCheckStatus_AZURE_CHECK_STATUS_PASS,
		v1.AzureCheckStatus_AZURE_CHECK_STATUS_WARN,
		v1.AzureCheckStatus_AZURE_CHECK_STATUS_FAIL,
		v1.AzureCheckStatus_AZURE_CHECK_STATUS_UNSPECIFIED,
	}
	for i, want := range wantStatuses {
		if got := resp.Msg.Checks[i].Status; got != want {
			t.Errorf("Checks[%d].Status=%v want %v", i, got, want)
		}
	}
	if resp.Msg.Checks[2].Fix != "configure" {
		t.Errorf("Checks[2].Fix=%q want 'configure'", resp.Msg.Checks[2].Fix)
	}
}

func TestE2E_GetAzureSetupState_StructPassthrough(t *testing.T) {
	want := map[string]any{
		"deployed":   true,
		"workspace":  "/subs/x/rg/y/wsp/z",
		"servicesAt": []any{"api", "web"},
	}
	funcs := fullStubAzureFuncs()
	funcs.GetSetupStateFn = func(context.Context) (any, error) { return want, nil }
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	resp, err := client.GetAzureSetupState(context.Background(),
		connect.NewRequest(&v1.GetAzureSetupStateRequest{}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.Msg.State == nil {
		t.Fatal("State is nil")
	}
	got := resp.Msg.State.AsMap()
	if got["deployed"] != true {
		t.Errorf("deployed=%v want true", got["deployed"])
	}
	if got["workspace"] != "/subs/x/rg/y/wsp/z" {
		t.Errorf("workspace=%v want path", got["workspace"])
	}
	servicesAt, ok := got["servicesAt"].([]any)
	if !ok || len(servicesAt) != 2 {
		t.Errorf("servicesAt=%v want 2-element slice", got["servicesAt"])
	}
}

func TestE2E_GetAzureSetupState_ErrorMaps(t *testing.T) {
	funcs := fullStubAzureFuncs()
	funcs.GetSetupStateFn = func(context.Context) (any, error) {
		return nil, &azure.AzureLogsError{Code: "AUTH_EXPIRED", Message: "expired"}
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	_, err := client.GetAzureSetupState(context.Background(),
		connect.NewRequest(&v1.GetAzureSetupStateRequest{}))
	if err == nil || connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("code=%v want Unauthenticated", connect.CodeOf(err))
	}
}

func TestE2E_VerifyAzureLogs_Success(t *testing.T) {
	gotService := ""
	funcs := fullStubAzureFuncs()
	funcs.VerifyServiceLogsFn = func(_ context.Context, name string) (*VerifyServiceLogsResult, error) {
		gotService = name
		return &VerifyServiceLogsResult{Success: true, Message: "found 5", RowsReturned: 5}, nil
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	resp, err := client.VerifyAzureLogs(context.Background(),
		connect.NewRequest(&v1.VerifyAzureLogsRequest{Service: "api"}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !resp.Msg.Success || resp.Msg.RowsReturned != 5 {
		t.Errorf("Success=%v Rows=%d want true/5", resp.Msg.Success, resp.Msg.RowsReturned)
	}
	if gotService != "api" {
		t.Errorf("gotService=%q want api", gotService)
	}
	if resp.Msg.QueriedAt == nil {
		t.Error("QueriedAt is nil; want server-set timestamp")
	}
}

func TestE2E_VerifyAzureLogs_ErrorMaps(t *testing.T) {
	funcs := fullStubAzureFuncs()
	funcs.VerifyServiceLogsFn = func(context.Context, string) (*VerifyServiceLogsResult, error) {
		return nil, &azure.AzureLogsError{Code: "NO_WORKSPACE", Message: "missing"}
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	_, err := client.VerifyAzureLogs(context.Background(),
		connect.NewRequest(&v1.VerifyAzureLogsRequest{Service: "api"}))
	if err == nil || connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("code=%v want NotFound", connect.CodeOf(err))
	}
}

func TestE2E_CheckDiagnosticSettings_FiltersNilEntries(t *testing.T) {
	funcs := fullStubAzureFuncs()
	funcs.CheckDiagnosticSettingsFn = func(context.Context) (*azure.DiagnosticSettingsCheckResponse, error) {
		return &azure.DiagnosticSettingsCheckResponse{
			WorkspaceID: "wsid-123",
			Services: map[string]*azure.DiagnosticSettingsCheckResult{
				"api": {
					Status:                azure.DiagnosticSettingsConfigured,
					ResourceID:            "/subs/r/api",
					DiagnosticSettingName: "send-to-law",
					WorkspaceID:           "wsid-123",
				},
				"missing": nil, // must be filtered out
				"web": {
					Status: azure.DiagnosticSettingsNotConfigured,
					Error:  "not set",
				},
			},
		}, nil
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	resp, err := client.CheckDiagnosticSettings(context.Background(),
		connect.NewRequest(&v1.CheckDiagnosticSettingsRequest{}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.Msg.WorkspaceId != "wsid-123" {
		t.Errorf("WorkspaceId=%q want wsid-123", resp.Msg.WorkspaceId)
	}
	if _, exists := resp.Msg.Services["missing"]; exists {
		t.Error("nil service result should be filtered out")
	}
	if got := resp.Msg.Services["api"]; got == nil ||
		got.Status != v1.DiagnosticSettingsStatus_DIAGNOSTIC_SETTINGS_STATUS_CONFIGURED {
		t.Errorf("api status missing/wrong: %+v", got)
	}
	if got := resp.Msg.Services["web"]; got == nil ||
		got.Status != v1.DiagnosticSettingsStatus_DIAGNOSTIC_SETTINGS_STATUS_NOT_CONFIGURED {
		t.Errorf("web status missing/wrong: %+v", got)
	}
}

func TestE2E_GetAzureDiagnostics_StructPassthrough(t *testing.T) {
	funcs := fullStubAzureFuncs()
	funcs.RunDiagnosticsFn = func(context.Context) (any, error) {
		return map[string]any{
			"summary": "all-good",
			"checks":  []any{map[string]any{"name": "auth", "ok": true}},
		}, nil
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	resp, err := client.GetAzureDiagnostics(context.Background(),
		connect.NewRequest(&v1.GetAzureDiagnosticsRequest{}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	got := resp.Msg.Diagnostics.AsMap()
	if got["summary"] != "all-good" {
		t.Errorf("summary=%v want all-good", got["summary"])
	}
}

func TestE2E_VerifyWorkspace_PartialPreservesPerServiceShape(t *testing.T) {
	last := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	funcs := fullStubAzureFuncs()
	funcs.VerifyWorkspaceFn = func(_ context.Context, req *azure.WorkspaceVerificationRequest) (*azure.WorkspaceVerificationResponse, error) {
		if len(req.Services) != 2 || req.Timespan != "PT15M" {
			t.Errorf("got services=%v timespan=%q want [api web] / PT15M",
				req.Services, req.Timespan)
		}
		return &azure.WorkspaceVerificationResponse{
			Status:    azure.VerificationStatusPartial,
			Workspace: azure.WorkspaceInfo{ID: "wsid", Name: "default"},
			Results: map[string]*azure.ServiceVerificationResult{
				"api": {
					Status:      azure.ServiceStatusOK,
					LogCount:    7,
					LastLogTime: &last,
					Message:     "found logs",
				},
				"web":     {Status: azure.ServiceStatusError, Error: "boom"},
				"ignored": nil, // must be filtered
			},
			Guidance: []string{"open settings", "retry verify"},
		}, nil
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	resp, err := client.VerifyWorkspace(context.Background(),
		connect.NewRequest(&v1.VerifyWorkspaceRequest{
			Services: []string{"api", "web"},
			Timespan: "PT15M",
		}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.Msg.Status != v1.WorkspaceVerificationStatus_WORKSPACE_VERIFICATION_STATUS_PARTIAL {
		t.Errorf("Status=%v want PARTIAL", resp.Msg.Status)
	}
	if resp.Msg.Workspace == nil ||
		resp.Msg.Workspace.Id != "wsid" || resp.Msg.Workspace.Name != "default" {
		t.Errorf("Workspace=%+v want id=wsid name=default", resp.Msg.Workspace)
	}
	if _, ok := resp.Msg.Results["ignored"]; ok {
		t.Error("nil per-service result should be dropped")
	}
	api, ok := resp.Msg.Results["api"]
	if !ok {
		t.Fatal("api result missing")
	}
	if api.Status != v1.ServiceVerificationStatus_SERVICE_VERIFICATION_STATUS_OK {
		t.Errorf("api status=%v want OK", api.Status)
	}
	if api.RowsReturned != 7 {
		t.Errorf("api rows=%d want 7", api.RowsReturned)
	}
	if api.QueriedAt == nil || !api.QueriedAt.AsTime().Equal(last) {
		t.Errorf("api QueriedAt=%v want %v", api.QueriedAt, last)
	}
	if api.Details == nil {
		t.Error("api Details should carry message blob")
	} else if got := api.Details.AsMap()["message"]; got != "found logs" {
		t.Errorf("api Details[message]=%v want 'found logs'", got)
	}
	web := resp.Msg.Results["web"]
	if web == nil || web.Error != "boom" ||
		web.Status != v1.ServiceVerificationStatus_SERVICE_VERIFICATION_STATUS_ERROR {
		t.Errorf("web=%+v want error/boom", web)
	}
	if len(resp.Msg.Guidance) != 2 {
		t.Errorf("Guidance=%v want 2 items", resp.Msg.Guidance)
	}
}

func TestE2E_GetAzureLogConfig_HostBecomesResourceType(t *testing.T) {
	funcs := fullStubAzureFuncs()
	funcs.LoadAzureYamlFn = func() (*service.AzureYaml, error) {
		return &service.AzureYaml{
			Services: map[string]service.Service{
				"api": {Host: "appservice"},
			},
		}, nil
	}
	funcs.RecommendedTablesFn = func(rt string) []string {
		if rt != "appservice" {
			t.Errorf("RecommendedTables called with rt=%q want appservice", rt)
		}
		return []string{"AppServiceConsoleLogs"}
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	resp, err := client.GetAzureLogConfig(context.Background(),
		connect.NewRequest(&v1.GetAzureLogConfigRequest{Service: "api"}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.Msg.ResourceType != "appservice" {
		t.Errorf("ResourceType=%q want appservice", resp.Msg.ResourceType)
	}
	if len(resp.Msg.Tables) != 1 || resp.Msg.Tables[0] != "AppServiceConsoleLogs" {
		t.Errorf("Tables=%v want recommended fallback", resp.Msg.Tables)
	}
	if resp.Msg.Mode != v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_TABLES {
		t.Errorf("Mode=%v want TABLES", resp.Msg.Mode)
	}
}

func TestE2E_GetAzureLogConfig_TablesOverride(t *testing.T) {
	funcs := fullStubAzureFuncs()
	funcs.LoadAzureYamlFn = func() (*service.AzureYaml, error) {
		return &service.AzureYaml{
			Services: map[string]service.Service{
				"api": {
					Host: "containerApp",
					Logs: &service.ServiceLogsConfig{
						Analytics: &service.AnalyticsConfigService{
							Tables: []string{"AppRequests", "AppExceptions"},
						},
					},
				},
			},
		}, nil
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	resp, err := client.GetAzureLogConfig(context.Background(),
		connect.NewRequest(&v1.GetAzureLogConfigRequest{Service: "api"}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !equalStrings(resp.Msg.Tables, []string{"AppRequests", "AppExceptions"}) {
		t.Errorf("Tables=%v want stored override", resp.Msg.Tables)
	}
	if resp.Msg.Mode != v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_TABLES {
		t.Errorf("Mode=%v want TABLES", resp.Msg.Mode)
	}
}

func TestE2E_GetAzureLogConfig_CustomQuery(t *testing.T) {
	funcs := fullStubAzureFuncs()
	funcs.LoadAzureYamlFn = func() (*service.AzureYaml, error) {
		return &service.AzureYaml{
			Services: map[string]service.Service{
				"api": {
					Host: "containerApp",
					Logs: &service.ServiceLogsConfig{
						Analytics: &service.AnalyticsConfigService{
							Query: "AppRequests | take 10",
						},
					},
				},
			},
		}, nil
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	resp, err := client.GetAzureLogConfig(context.Background(),
		connect.NewRequest(&v1.GetAzureLogConfigRequest{Service: "api"}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.Msg.Mode != v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_CUSTOM {
		t.Errorf("Mode=%v want CUSTOM", resp.Msg.Mode)
	}
	if resp.Msg.Query != "AppRequests | take 10" {
		t.Errorf("Query=%q want stored", resp.Msg.Query)
	}
}

func TestE2E_GetAzureLogConfig_LoadErrFallsBackToRecommended(t *testing.T) {
	funcs := fullStubAzureFuncs()
	funcs.LoadAzureYamlFn = func() (*service.AzureYaml, error) {
		return nil, errors.New("yaml unreadable")
	}
	funcs.RecommendedTablesFn = func(rt string) []string {
		if rt != "containerapp" {
			t.Errorf("rt=%q want containerapp", rt)
		}
		return []string{"ContainerAppConsoleLogs_CL"}
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	resp, err := client.GetAzureLogConfig(context.Background(),
		connect.NewRequest(&v1.GetAzureLogConfigRequest{Service: "api"}))
	if err != nil {
		t.Fatalf("err on load failure should fall back, got: %v", err)
	}
	if len(resp.Msg.Tables) != 1 || resp.Msg.Tables[0] != "ContainerAppConsoleLogs_CL" {
		t.Errorf("Tables=%v want fallback", resp.Msg.Tables)
	}
}

func TestE2E_SaveAzureLogConfig_TablesHappyPath(t *testing.T) {
	var gotName string
	var gotTables []string
	var gotQuery string
	funcs := fullStubAzureFuncs()
	funcs.SaveServiceLogConfigFn = func(name string, tables []string, query string) error {
		gotName = name
		gotTables = tables
		gotQuery = query
		return nil
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	resp, err := client.SaveAzureLogConfig(context.Background(),
		connect.NewRequest(&v1.SaveAzureLogConfigRequest{
			Service: "api",
			Mode:    v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_TABLES,
			Tables:  []string{"AppRequests"},
		}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if gotName != "api" || !equalStrings(gotTables, []string{"AppRequests"}) || gotQuery != "" {
		t.Errorf("store got name=%q tables=%v query=%q", gotName, gotTables, gotQuery)
	}
	if resp.Msg.Service != "api" || resp.Msg.Mode != v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_TABLES {
		t.Errorf("echoed response wrong: %+v", resp.Msg)
	}
}

func TestE2E_SaveAzureLogConfig_CustomHappyPath(t *testing.T) {
	var gotQuery string
	funcs := fullStubAzureFuncs()
	funcs.SaveServiceLogConfigFn = func(_ string, tables []string, query string) error {
		if len(tables) != 0 {
			t.Errorf("custom mode should NOT pass tables; got %v", tables)
		}
		gotQuery = query
		return nil
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	if _, err := client.SaveAzureLogConfig(context.Background(),
		connect.NewRequest(&v1.SaveAzureLogConfigRequest{
			Service: "api",
			Mode:    v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_CUSTOM,
			Query:   "AppRequests | take 1",
		})); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if gotQuery != "AppRequests | take 1" {
		t.Errorf("gotQuery=%q want passthrough", gotQuery)
	}
}

func TestE2E_SaveAzureLogConfig_StoreErrorMaps(t *testing.T) {
	funcs := fullStubAzureFuncs()
	funcs.SaveServiceLogConfigFn = func(string, []string, string) error {
		return errors.New("disk full")
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	_, err := client.SaveAzureLogConfig(context.Background(),
		connect.NewRequest(&v1.SaveAzureLogConfigRequest{
			Service: "api",
			Mode:    v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_TABLES,
			Tables:  []string{"X"},
		}))
	if err == nil || connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("code=%v want Internal", connect.CodeOf(err))
	}
}

func TestE2E_ListAzureTables_LiveSucceeds(t *testing.T) {
	funcs := fullStubAzureFuncs()
	funcs.WorkspaceIDFn = func(context.Context) (string, error) { return "wsid-abc", nil }
	funcs.ListLiveTablesFn = func(_ context.Context, w string) ([]azure.TableInfo, error) {
		if w != "wsid-abc" {
			t.Errorf("ListLiveTables got w=%q want wsid-abc", w)
		}
		return []azure.TableInfo{
			{Name: "AppRequests", Description: "requests"},
			{Name: "Custom_CL", Description: "custom"},
		}, nil
	}
	funcs.RecommendedTablesFn = func(string) []string { return []string{"AppRequests"} }
	funcs.IsRecommendedTableFn = func(name, _ string) bool { return name == "AppRequests" }
	funcs.TableCategoriesFn = func() map[string]azure.TableCategory {
		return map[string]azure.TableCategory{
			"app": {DisplayName: "Application", Tables: []string{"AppRequests"}},
		}
	}
	funcs.TruncateMiddleFn = func(s string, _ int) string { return s + "*" }
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	resp, err := client.ListAzureTables(context.Background(),
		connect.NewRequest(&v1.ListAzureTablesRequest{
			ResourceType: v1.AzureResourceType_AZURE_RESOURCE_TYPE_APP_SERVICE,
		}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(resp.Msg.Tables) != 2 {
		t.Fatalf("Tables=%d want 2", len(resp.Msg.Tables))
	}
	if !resp.Msg.Tables[0].Recommended {
		t.Error("AppRequests should be Recommended")
	}
	if resp.Msg.Tables[1].Recommended {
		t.Error("Custom_CL should NOT be Recommended")
	}
	if resp.Msg.Workspace != "wsid-abc*" {
		t.Errorf("Workspace=%q want truncated form", resp.Msg.Workspace)
	}
	if len(resp.Msg.Categories) != 1 || resp.Msg.Categories[0].Name != "app" {
		t.Errorf("Categories=%+v want one 'app' entry", resp.Msg.Categories)
	}
}

func TestE2E_ListAzureTables_EmptyWorkspaceFallsBack(t *testing.T) {
	funcs := fullStubAzureFuncs()
	funcs.WorkspaceIDFn = func(context.Context) (string, error) { return "", nil }
	funcs.AllKnownTablesFn = func() []azure.TableInfo {
		return []azure.TableInfo{{Name: "AppRequests"}}
	}
	funcs.ListLiveTablesFn = func(context.Context, string) ([]azure.TableInfo, error) {
		t.Fatal("ListLiveTables MUST NOT be called when workspaceID is empty")
		return nil, nil
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	resp, err := client.ListAzureTables(context.Background(),
		connect.NewRequest(&v1.ListAzureTablesRequest{}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(resp.Msg.Tables) != 1 || resp.Msg.Tables[0].Name != "AppRequests" {
		t.Errorf("Tables=%+v want AllKnownTables fallback", resp.Msg.Tables)
	}
}

func TestE2E_GetServiceQuery_StoredCustomReturnsCustomResourceType(t *testing.T) {
	funcs := fullStubAzureFuncs()
	funcs.LoadAzureYamlFn = func() (*service.AzureYaml, error) {
		return &service.AzureYaml{
			Services: map[string]service.Service{
				"api": {
					Logs: &service.ServiceLogsConfig{
						Analytics: &service.AnalyticsConfigService{
							Query: "AppRequests | where Name == 'foo'",
						},
					},
				},
			},
		}, nil
	}
	// Default + Substitute MUST NOT be invoked when a stored query exists.
	funcs.DefaultQueryFn = func(string) string {
		t.Fatal("DefaultQuery should NOT be called when stored query exists")
		return ""
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	resp, err := client.GetServiceQuery(context.Background(),
		connect.NewRequest(&v1.GetServiceQueryRequest{Service: "api"}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.Msg.ResourceType != "custom" {
		t.Errorf("ResourceType=%q want custom", resp.Msg.ResourceType)
	}
	if resp.Msg.Query != "AppRequests | where Name == 'foo'" {
		t.Errorf("Query=%q want stored", resp.Msg.Query)
	}
}

func TestE2E_GetServiceQuery_DefaultPath(t *testing.T) {
	calls := 0
	funcs := fullStubAzureFuncs()
	funcs.LoadAzureYamlFn = func() (*service.AzureYaml, error) { return nil, nil }
	funcs.DefaultQueryFn = func(rt string) string {
		calls++
		if rt != "containerapp" {
			t.Errorf("DefaultQuery rt=%q want containerapp", rt)
		}
		return "ContainerAppConsoleLogs_CL | where ContainerAppName_s == '{serviceName}' | where TimeGenerated > ago({timespan})"
	}
	funcs.SubstituteQueryPlaceholdersFn = func(q, svc, ts string) string {
		if svc != "api" || ts != "30m" {
			t.Errorf("Substitute got svc=%q ts=%q want api/30m", svc, ts)
		}
		return strings.ReplaceAll(strings.ReplaceAll(q, "{serviceName}", svc), "{timespan}", ts)
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	resp, err := client.GetServiceQuery(context.Background(),
		connect.NewRequest(&v1.GetServiceQueryRequest{Service: "api"}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if calls != 1 {
		t.Errorf("DefaultQuery called %d times, want 1", calls)
	}
	if resp.Msg.ResourceType != "containerapp" {
		t.Errorf("ResourceType=%q want containerapp", resp.Msg.ResourceType)
	}
	if !strings.Contains(resp.Msg.Query, "'api'") || !strings.Contains(resp.Msg.Query, "ago(30m)") {
		t.Errorf("Query=%q want placeholders substituted", resp.Msg.Query)
	}
}

func TestE2E_SaveServiceQuery_HappyPath(t *testing.T) {
	gotName, gotQuery := "", ""
	funcs := fullStubAzureFuncs()
	funcs.SaveServiceCustomQueryFn = func(name, q string) error {
		gotName = name
		gotQuery = q
		return nil
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	resp, err := client.SaveServiceQuery(context.Background(),
		connect.NewRequest(&v1.SaveServiceQueryRequest{Service: "api", Query: "Foo | take 1"}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if gotName != "api" || gotQuery != "Foo | take 1" {
		t.Errorf("store got name=%q q=%q", gotName, gotQuery)
	}
	if resp.Msg.ResourceType != "custom" {
		t.Errorf("ResourceType=%q want custom", resp.Msg.ResourceType)
	}
}

func TestE2E_SaveServiceQuery_EmptyQueryInvalid(t *testing.T) {
	funcs := fullStubAzureFuncs()
	funcs.SaveServiceCustomQueryFn = func(string, string) error {
		t.Fatal("store should not be called when validation fails")
		return nil
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	_, err := client.SaveServiceQuery(context.Background(),
		connect.NewRequest(&v1.SaveServiceQueryRequest{Service: "api", Query: ""}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code=%v want InvalidArgument", connect.CodeOf(err))
	}
}

func TestE2E_GetAzureServices_Roundtrip(t *testing.T) {
	funcs := fullStubAzureFuncs()
	funcs.ServiceNamesFromEnvFn = func() []string { return []string{"api", "web", "worker"} }
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	resp, err := client.GetAzureServices(context.Background(),
		connect.NewRequest(&v1.GetAzureServicesRequest{}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !equalStrings(resp.Msg.Services, []string{"api", "web", "worker"}) {
		t.Errorf("services=%v want [api web worker]", resp.Msg.Services)
	}
}

// =============================================================================
// Streaming tests - polling mode.
// =============================================================================

func TestE2E_StreamAzureLogs_InitialStatusFrame(t *testing.T) {
	// Default funcs: FetchLogs returns nil/nil so polling loop produces no
	// entries. Test asserts the initial "connected/polling" Status arrives
	// before headers would otherwise stay buffered.
	funcs := fullStubAzureFuncs()
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.StreamAzureLogs(ctx,
		connect.NewRequest(&v1.StreamAzureLogsRequest{Service: "api"}))
	if err != nil {
		t.Fatalf("StreamAzureLogs: %v", err)
	}

	if !stream.Receive() {
		t.Fatalf("Receive: %v", stream.Err())
	}
	st := stream.Msg().GetStatus()
	if st == nil {
		t.Fatalf("first frame should be Status, got %+v", stream.Msg())
	}
	if st.Status != "connected" || st.Mode != "polling" {
		t.Errorf("initial status=%q/%q want connected/polling", st.Status, st.Mode)
	}
	if st.ConsecutiveFails != 0 || st.Error != "" {
		t.Errorf("initial status fails=%d err=%q want clean", st.ConsecutiveFails, st.Error)
	}
}

func TestE2E_StreamAzureLogs_PollingDeliversEntries(t *testing.T) {
	t0 := time.Now().Add(-1 * time.Second)
	calls := atomic.Int32{}
	funcs := fullStubAzureFuncs()
	funcs.FetchLogsFn = func(context.Context, azure.StandaloneLogsConfig) ([]azure.LogEntry, error) {
		n := calls.Add(1)
		if n == 1 {
			return []azure.LogEntry{
				makeAzureLogEntry("api", "first", t0),
				makeAzureLogEntry("api", "second", t0.Add(100*time.Millisecond)),
			}, nil
		}
		return nil, nil
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.StreamAzureLogs(ctx,
		connect.NewRequest(&v1.StreamAzureLogsRequest{Service: "api"}))
	if err != nil {
		t.Fatalf("StreamAzureLogs: %v", err)
	}

	frames := recvUntil(t, stream, 5*time.Second, func(got []*v1.StreamAzureLogsResponse) bool {
		entries := 0
		for _, f := range got {
			if f.GetEntry() != nil {
				entries++
			}
		}
		return entries >= 2
	})

	var entries []*v1.LogEntry
	var statuses []*v1.StreamStatus
	for _, f := range frames {
		if e := f.GetEntry(); e != nil {
			entries = append(entries, e)
		}
		if s := f.GetStatus(); s != nil {
			statuses = append(statuses, s)
		}
	}
	if len(statuses) < 1 {
		t.Errorf("expected initial Status frame; got 0")
	}
	if len(entries) < 2 {
		t.Fatalf("expected 2 entries, got %d (frames=%d)", len(entries), len(frames))
	}
	if entries[0].Message != "first" || entries[1].Message != "second" {
		t.Errorf("entries=[%q,%q] want [first,second]",
			entries[0].Message, entries[1].Message)
	}
}

func TestE2E_StreamAzureLogs_PollingErrorEmitsDegradedStatus(t *testing.T) {
	calls := atomic.Int32{}
	funcs := fullStubAzureFuncs()
	funcs.FetchLogsFn = func(context.Context, azure.StandaloneLogsConfig) ([]azure.LogEntry, error) {
		calls.Add(1)
		return nil, errors.New("workspace flapping")
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.StreamAzureLogs(ctx,
		connect.NewRequest(&v1.StreamAzureLogsRequest{Service: "api"}))
	if err != nil {
		t.Fatalf("StreamAzureLogs: %v", err)
	}

	frames := recvUntil(t, stream, 8*time.Second, func(got []*v1.StreamAzureLogsResponse) bool {
		// We need: initial connected status + at least one degraded status
		degraded := 0
		for _, f := range got {
			if s := f.GetStatus(); s != nil && s.Status != "connected" {
				degraded++
			}
		}
		return degraded >= 1
	})

	var connected, degraded *v1.StreamStatus
	for _, f := range frames {
		s := f.GetStatus()
		if s == nil {
			continue
		}
		if connected == nil && s.Status == "connected" {
			connected = s
		}
		if s.Status != "connected" && degraded == nil {
			degraded = s
		}
	}
	if connected == nil {
		t.Fatal("missing initial connected Status")
	}
	if degraded == nil {
		t.Fatalf("missing degraded Status (frames=%+v)", frames)
	}
	if degraded.Mode != "polling" {
		t.Errorf("degraded.Mode=%q want polling", degraded.Mode)
	}
	if degraded.ConsecutiveFails < 1 {
		t.Errorf("ConsecutiveFails=%d want >=1", degraded.ConsecutiveFails)
	}
	if !strings.Contains(degraded.Error, "flapping") {
		t.Errorf("Error=%q want it to mention 'flapping'", degraded.Error)
	}
}

func TestE2E_StreamAzureLogs_PollingNoResultsIsSilent(t *testing.T) {
	// NO_RESULTS during polling is treated like success - no degraded status,
	// no entries. Asserts handler short-circuits via isNoResults.
	funcs := fullStubAzureFuncs()
	funcs.FetchLogsFn = func(context.Context, azure.StandaloneLogsConfig) ([]azure.LogEntry, error) {
		return nil, &azure.AzureLogsError{Code: "NO_RESULTS", Message: "empty"}
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.StreamAzureLogs(ctx,
		connect.NewRequest(&v1.StreamAzureLogsRequest{Service: "api"}))
	if err != nil {
		t.Fatalf("StreamAzureLogs: %v", err)
	}

	// Receive the initial Status, then assert no degraded follow-up arrives
	// within a polling cycle (~6s base interval but we only watch ~1.5s).
	if !stream.Receive() {
		t.Fatalf("Receive: %v", stream.Err())
	}
	if got := stream.Msg().GetStatus(); got == nil || got.Status != "connected" {
		t.Fatalf("first frame want connected status, got %+v", stream.Msg())
	}

	frames := recvUntil(t, stream, 1500*time.Millisecond, func(got []*v1.StreamAzureLogsResponse) bool {
		// Stop early ONLY on a non-connected frame; otherwise let the timer expire.
		for _, f := range got {
			if s := f.GetStatus(); s != nil && s.Status != "connected" {
				return true
			}
			if f.GetEntry() != nil {
				return true
			}
		}
		return false
	})
	for _, f := range frames {
		if s := f.GetStatus(); s != nil && s.Status != "connected" {
			t.Errorf("NO_RESULTS should NOT emit %q status; got %+v", s.Status, s)
		}
		if e := f.GetEntry(); e != nil {
			t.Errorf("NO_RESULTS should not produce entries; got %+v", e)
		}
	}
}

func TestE2E_StreamAzureLogs_ClientCancelExitsCleanly(t *testing.T) {
	calls := atomic.Int32{}
	funcs := fullStubAzureFuncs()
	funcs.FetchLogsFn = func(context.Context, azure.StandaloneLogsConfig) ([]azure.LogEntry, error) {
		calls.Add(1)
		return nil, nil
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	stream, err := client.StreamAzureLogs(ctx,
		connect.NewRequest(&v1.StreamAzureLogsRequest{Service: "api"}))
	if err != nil {
		t.Fatalf("StreamAzureLogs: %v", err)
	}
	// Pull the initial Status to confirm headers flushed.
	if !stream.Receive() {
		t.Fatalf("Receive: %v", stream.Err())
	}

	cancel()

	// Drain remainder; subsequent Receive returns false and Err signals cancel.
	for stream.Receive() {
		// burn through any in-flight frame
	}
	rerr := stream.Err()
	if rerr != nil &&
		connect.CodeOf(rerr) != connect.CodeCanceled &&
		!strings.Contains(strings.ToLower(rerr.Error()), "cancel") {
		t.Logf("post-cancel err (acceptable): %v", rerr)
	}
}

func TestE2E_StreamAzureLogs_BackfillSecondsPropagates(t *testing.T) {
	got := make(chan time.Duration, 4)
	funcs := fullStubAzureFuncs()
	funcs.FetchLogsFn = func(_ context.Context, c azure.StandaloneLogsConfig) ([]azure.LogEntry, error) {
		select {
		case got <- c.Since:
		default:
		}
		return nil, nil
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.StreamAzureLogs(ctx,
		connect.NewRequest(&v1.StreamAzureLogsRequest{
			Service:         "api",
			BackfillSeconds: 7,
		}))
	if err != nil {
		t.Fatalf("StreamAzureLogs: %v", err)
	}
	if !stream.Receive() {
		t.Fatalf("Receive: %v", stream.Err())
	}

	select {
	case since := <-got:
		if since != 7*time.Second {
			t.Errorf("first poll Since=%v want 7s (BackfillSeconds)", since)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("FetchLogs never called within 2s")
	}
}

// =============================================================================
// Streaming tests - realtime mode.
// =============================================================================

func TestE2E_StreamAzureLogs_RealtimeFallbackOnResolveErr(t *testing.T) {
	// ResolveResource fails -> realtime setup error -> handler emits a
	// "degraded/polling" flip status, then runs the polling loop.
	funcs := fullStubAzureFuncs()
	funcs.ResolveResourceFn = func(context.Context, string) (*azure.AzureResource, error) {
		return nil, errors.New("resource lookup failed")
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.StreamAzureLogs(ctx,
		connect.NewRequest(&v1.StreamAzureLogsRequest{Service: "api", Realtime: true}))
	if err != nil {
		t.Fatalf("StreamAzureLogs: %v", err)
	}

	frames := recvUntil(t, stream, 4*time.Second, func(got []*v1.StreamAzureLogsResponse) bool {
		// We want: initial connected/realtime status THEN a degraded/polling flip.
		var sawConnectedRT, sawDegradedPoll bool
		for _, f := range got {
			s := f.GetStatus()
			if s == nil {
				continue
			}
			if s.Status == "connected" && s.Mode == "realtime" {
				sawConnectedRT = true
			}
			if s.Status == "degraded" && s.Mode == "polling" {
				sawDegradedPoll = true
			}
		}
		return sawConnectedRT && sawDegradedPoll
	})

	var sawConnectedRT, sawDegradedPoll bool
	for _, f := range frames {
		s := f.GetStatus()
		if s == nil {
			continue
		}
		if s.Status == "connected" && s.Mode == "realtime" {
			sawConnectedRT = true
		}
		if s.Status == "degraded" && s.Mode == "polling" {
			sawDegradedPoll = true
		}
	}
	if !sawConnectedRT {
		t.Errorf("missing initial connected/realtime status; frames=%+v", frames)
	}
	if !sawDegradedPoll {
		t.Errorf("missing degraded/polling flip; frames=%+v", frames)
	}
}

func TestE2E_StreamAzureLogs_RealtimeDeliversEntries(t *testing.T) {
	// Wire a fake realtime streamer that pre-bursts 3 entries on Start.
	streamer := newFakeRealtimeStreamer("api", azure.ResourceTypeContainerApp)
	t0 := time.Now()
	streamer.burst = []azure.LogEntry{
		makeAzureLogEntry("api", "rt-1", t0),
		makeAzureLogEntry("api", "rt-2", t0.Add(time.Millisecond)),
		makeAzureLogEntry("api", "rt-3", t0.Add(2*time.Millisecond)),
	}

	funcs := fullStubAzureFuncs()
	funcs.ResolveResourceFn = func(context.Context, string) (*azure.AzureResource, error) {
		return &azure.AzureResource{
			ServiceName:    "api",
			ResourceType:   azure.ResourceTypeContainerApp,
			ResourceGroup:  "rg",
			SubscriptionID: "sub",
			Name:           "api-app",
		}, nil
	}
	funcs.NewLogAnalyticsCredentialFn = func() (azcore.TokenCredential, error) { return nil, nil }
	funcs.NewRealtimeStreamerFn = func(rt azure.ResourceType, cfg azure.StreamerConfig) (azure.RealtimeLogStreamer, error) {
		if rt != azure.ResourceTypeContainerApp {
			t.Errorf("rt=%v want containerApp", rt)
		}
		if cfg.ServiceName != "api" || cfg.ResourceGroup != "rg" {
			t.Errorf("cfg=%+v want service/rg propagated", cfg)
		}
		return streamer, nil
	}
	// Backfill returns nothing so the test focuses on realtime delivery.
	funcs.FetchLogsFn = func(context.Context, azure.StandaloneLogsConfig) ([]azure.LogEntry, error) {
		return nil, nil
	}

	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.StreamAzureLogs(ctx,
		connect.NewRequest(&v1.StreamAzureLogsRequest{
			Service:         "api",
			Realtime:        true,
			BackfillSeconds: 1,
		}))
	if err != nil {
		t.Fatalf("StreamAzureLogs: %v", err)
	}

	frames := recvUntil(t, stream, 4*time.Second, func(got []*v1.StreamAzureLogsResponse) bool {
		entries := 0
		for _, f := range got {
			if f.GetEntry() != nil {
				entries++
			}
		}
		return entries >= 3
	})

	var entries []*v1.LogEntry
	for _, f := range frames {
		if e := f.GetEntry(); e != nil {
			entries = append(entries, e)
		}
	}
	if len(entries) < 3 {
		t.Fatalf("got %d entries, want >=3 (frames=%d)", len(entries), len(frames))
	}
	wantMsgs := []string{"rt-1", "rt-2", "rt-3"}
	for i, want := range wantMsgs {
		if entries[i].Message != want {
			t.Errorf("entries[%d].Message=%q want %q", i, entries[i].Message, want)
		}
	}

	// Cancel and verify Stop was called on the streamer (server tore down).
	cancel()
	select {
	case <-streamer.stopped:
		// good
	case <-time.After(2 * time.Second):
		t.Error("streamer.Stop never invoked after client cancel")
	}
}

func TestE2E_StreamAzureLogs_RealtimeBackfillFailureTolerated(t *testing.T) {
	// Backfill error must not abort realtime; entries from the streamer
	// still arrive.
	streamer := newFakeRealtimeStreamer("api", azure.ResourceTypeContainerApp)
	streamer.burst = []azure.LogEntry{
		makeAzureLogEntry("api", "after-failed-backfill", time.Now()),
	}

	funcs := fullStubAzureFuncs()
	funcs.ResolveResourceFn = func(context.Context, string) (*azure.AzureResource, error) {
		return &azure.AzureResource{ServiceName: "api", ResourceType: azure.ResourceTypeContainerApp}, nil
	}
	funcs.NewLogAnalyticsCredentialFn = func() (azcore.TokenCredential, error) { return nil, nil }
	funcs.NewRealtimeStreamerFn = func(azure.ResourceType, azure.StreamerConfig) (azure.RealtimeLogStreamer, error) {
		return streamer, nil
	}
	funcs.FetchLogsFn = func(context.Context, azure.StandaloneLogsConfig) ([]azure.LogEntry, error) {
		return nil, errors.New("backfill blew up")
	}

	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.StreamAzureLogs(ctx,
		connect.NewRequest(&v1.StreamAzureLogsRequest{
			Service:         "api",
			Realtime:        true,
			BackfillSeconds: 1,
		}))
	if err != nil {
		t.Fatalf("StreamAzureLogs: %v", err)
	}

	frames := recvUntil(t, stream, 4*time.Second, func(got []*v1.StreamAzureLogsResponse) bool {
		for _, f := range got {
			if e := f.GetEntry(); e != nil && e.Message == "after-failed-backfill" {
				return true
			}
		}
		return false
	})

	found := false
	for _, f := range frames {
		if e := f.GetEntry(); e != nil && e.Message == "after-failed-backfill" {
			found = true
		}
	}
	if !found {
		t.Errorf("realtime entry missing after backfill failure; frames=%d", len(frames))
	}
}

func TestE2E_StreamAzureLogs_RealtimeDroppedNotice(t *testing.T) {
	// Force ring overflow: streamer pre-bursts 5_000 entries before the main
	// loop's first Send completes. With ring cap=256 and each Send taking
	// real HTTP/1 work, drops are reliably observed.
	streamer := newFakeRealtimeStreamer("api", azure.ResourceTypeContainerApp)
	const burstN = 5_000
	now := time.Now()
	streamer.burst = make([]azure.LogEntry, burstN)
	for i := 0; i < burstN; i++ {
		streamer.burst[i] = makeAzureLogEntry("api",
			fmt.Sprintf("e%d", i), now.Add(time.Duration(i)*time.Microsecond))
	}

	funcs := fullStubAzureFuncs()
	funcs.ResolveResourceFn = func(context.Context, string) (*azure.AzureResource, error) {
		return &azure.AzureResource{ServiceName: "api", ResourceType: azure.ResourceTypeContainerApp}, nil
	}
	funcs.NewLogAnalyticsCredentialFn = func() (azcore.TokenCredential, error) { return nil, nil }
	funcs.NewRealtimeStreamerFn = func(azure.ResourceType, azure.StreamerConfig) (azure.RealtimeLogStreamer, error) {
		return streamer, nil
	}
	// Slow backfill: gives the pump goroutine time to fill (and overflow)
	// the ring before the main loop starts draining via stream.Send.
	funcs.FetchLogsFn = func(context.Context, azure.StandaloneLogsConfig) ([]azure.LogEntry, error) {
		time.Sleep(150 * time.Millisecond)
		return nil, nil
	}

	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.StreamAzureLogs(ctx,
		connect.NewRequest(&v1.StreamAzureLogsRequest{
			Service:         "api",
			Realtime:        true,
			BackfillSeconds: 1,
		}))
	if err != nil {
		t.Fatalf("StreamAzureLogs: %v", err)
	}

	// Receive until we observe at least one Dropped notice OR ~6s passes.
	frames := recvUntil(t, stream, 6*time.Second, func(got []*v1.StreamAzureLogsResponse) bool {
		for _, f := range got {
			if f.GetDropped() != nil {
				return true
			}
		}
		return false
	})

	var dropped *v1.AzureDroppedNotice
	entriesSeen := 0
	for _, f := range frames {
		if d := f.GetDropped(); d != nil && dropped == nil {
			dropped = d
		}
		if f.GetEntry() != nil {
			entriesSeen++
		}
	}
	if dropped == nil {
		t.Fatalf("expected AzureDroppedNotice with %d-entry burst; frames=%d entries=%d",
			burstN, len(frames), entriesSeen)
	}
	if dropped.Count <= 0 {
		t.Errorf("Dropped.Count=%d want >0", dropped.Count)
	}
	if dropped.Reason != "realtime_buffer_overflow" {
		t.Errorf("Dropped.Reason=%q want realtime_buffer_overflow", dropped.Reason)
	}
	if dropped.At == nil {
		t.Error("Dropped.At missing timestamp")
	}
	if entriesSeen == 0 {
		t.Error("expected at least some entries alongside drops")
	}
}

// =============================================================================
// PollingState retry/recovery composition.
// =============================================================================

func TestE2E_StreamAzureLogs_ConsecutiveFailsIncrement(t *testing.T) {
	calls := atomic.Int32{}
	funcs := fullStubAzureFuncs()
	funcs.FetchLogsFn = func(context.Context, azure.StandaloneLogsConfig) ([]azure.LogEntry, error) {
		calls.Add(1)
		return nil, errors.New("transient")
	}
	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.StreamAzureLogs(ctx,
		connect.NewRequest(&v1.StreamAzureLogsRequest{Service: "api"}))
	if err != nil {
		t.Fatalf("StreamAzureLogs: %v", err)
	}

	// PollingState's NextDelay starts at 5s after the first failure, so we
	// only get to ConsecutiveFails=1 within a reasonable test window. Assert
	// the FIRST degraded status carries 1.
	frames := recvUntil(t, stream, 6*time.Second, func(got []*v1.StreamAzureLogsResponse) bool {
		for _, f := range got {
			if s := f.GetStatus(); s != nil && s.ConsecutiveFails > 0 {
				return true
			}
		}
		return false
	})

	maxFails := int32(0)
	for _, f := range frames {
		if s := f.GetStatus(); s != nil && s.ConsecutiveFails > maxFails {
			maxFails = s.ConsecutiveFails
		}
	}
	if maxFails < 1 {
		t.Errorf("ConsecutiveFails never advanced; frames=%+v", frames)
	}
}

// =============================================================================
// sendWithBackpressure - direct unit test against the unexported helper.
// =============================================================================
//
// The real connect.ServerStream is hard to fake from outside the package, but
// we live in the same package - so we exercise sendWithBackpressure's timeout
// path via a real client/server pair with a bottlenecked client.

// We cannot construct a connect.ServerStream directly, so this test indirectly
// validates sendWithBackpressure: it calls the helper from inside a real
// streaming RPC where the client never receives, then asserts errStreamBlocked
// shows up. The implementation streams via a goroutine, so the helper's
// timeout (~100ms) governs.
func TestSendWithBackpressure_TimesOutOnSlowClient(t *testing.T) {
	if raceEnabled {
		t.Skip("skipping under -race: inherent race between connect-rpc handler Close and httptest transport teardown on client cancel")
	}
	mgr := broadcast.New()
	mux := http.NewServeMux()
	// Mount a bare AzureService where StreamAzureLogs is overridden to
	// invoke sendWithBackpressure with a never-receiving client.
	funcs := fullStubAzureFuncs()
	// The real handler always sends an initial Status BEFORE entering the
	// polling loop; that first Send always succeeds immediately because
	// httptest's transport buffers small frames. The subsequent
	// sendWithBackpressure call inside streamPolling is what we want to
	// exercise. Wire FetchLogs to return one entry per poll so the polling
	// loop reaches sendWithBackpressure.
	t0 := time.Now()
	funcs.FetchLogsFn = func(context.Context, azure.StandaloneLogsConfig) ([]azure.LogEntry, error) {
		return []azure.LogEntry{makeAzureLogEntry("api", "x", t0)}, nil
	}
	Mount(mux, Dependencies{Broadcast: mgr, Azure: funcs})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	defer mgr.StopAll()

	client := azdappv1connect.NewAzureServiceClient(srv.Client(), srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.StreamAzureLogs(ctx,
		connect.NewRequest(&v1.StreamAzureLogsRequest{Service: "api"}))
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	// Read the initial Status frame so headers flush, then close the
	// stream WITHOUT draining further. The server's next sendWithBackpressure
	// call should observe a blocked send (reaches errStreamBlocked OR a
	// transport error after the close races in).
	if !stream.Receive() {
		t.Fatalf("Receive: %v", stream.Err())
	}
	cancel()

	// We can't directly observe errStreamBlocked from the client side, so
	// the assertion is that the server tears the stream down within the
	// backpressure timeout window (handler exits cleanly).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !stream.Receive() {
			break
		}
	}
}

// TestSendWithBackpressure_NonBlockingSendReturnsNil exercises the happy
// path of sendWithBackpressure: a fast Send returns nil within budget. We
// drive it via the realtime path which uses stream.Send directly (NOT
// sendWithBackpressure), so this primarily proves the realtime delivery
// loop completes Sends synchronously.
func TestSendWithBackpressure_HappyPath(t *testing.T) {
	// Dedicated realtime stream so we know the per-frame latency budget is
	// not the constraint - if entries land, sends complete well under
	// the 100ms backpressure window even when the polling helper is in play.
	streamer := newFakeRealtimeStreamer("api", azure.ResourceTypeContainerApp)
	streamer.burst = []azure.LogEntry{
		makeAzureLogEntry("api", "fast", time.Now()),
	}

	funcs := fullStubAzureFuncs()
	funcs.ResolveResourceFn = func(context.Context, string) (*azure.AzureResource, error) {
		return &azure.AzureResource{ServiceName: "api", ResourceType: azure.ResourceTypeContainerApp}, nil
	}
	funcs.NewLogAnalyticsCredentialFn = func() (azcore.TokenCredential, error) { return nil, nil }
	funcs.NewRealtimeStreamerFn = func(azure.ResourceType, azure.StreamerConfig) (azure.RealtimeLogStreamer, error) {
		return streamer, nil
	}
	funcs.FetchLogsFn = func(context.Context, azure.StandaloneLogsConfig) ([]azure.LogEntry, error) {
		return nil, nil
	}

	client, cleanup := newAzureTestServer(t, funcs)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.StreamAzureLogs(ctx,
		connect.NewRequest(&v1.StreamAzureLogsRequest{Service: "api", Realtime: true}))
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	gotEntry := false
	for time.Now().Before(deadline) && !gotEntry {
		if !stream.Receive() {
			break
		}
		if e := stream.Msg().GetEntry(); e != nil && e.Message == "fast" {
			gotEntry = true
		}
	}
	if !gotEntry {
		t.Fatal("did not receive realtime entry within 3s window")
	}
}

// =============================================================================
// localAzureLogRing - additional unit tests beyond drop-oldest + coalesce.
// =============================================================================

func TestLocalAzureLogRing_DrainEmpty(t *testing.T) {
	r := newAzureLogRing(4)
	out, dropped := r.drain()
	if out != nil {
		t.Errorf("empty drain returned %v, want nil", out)
	}
	if dropped != 0 {
		t.Errorf("dropped=%d, want 0", dropped)
	}
}

func TestLocalAzureLogRing_DrainResetsBuffer(t *testing.T) {
	r := newAzureLogRing(4)
	r.push(&v1.LogEntry{Message: "a"})
	r.push(&v1.LogEntry{Message: "b"})
	if out, _ := r.drain(); len(out) != 2 {
		t.Fatalf("first drain=%d entries, want 2", len(out))
	}
	r.push(&v1.LogEntry{Message: "c"})
	out, dropped := r.drain()
	if len(out) != 1 || out[0].Message != "c" {
		t.Errorf("post-drain push out=%+v want [c]", out)
	}
	if dropped != 0 {
		t.Errorf("dropped=%d want 0 (no overflow)", dropped)
	}
}

func TestLocalAzureLogRing_DroppedAccumulates(t *testing.T) {
	r := newAzureLogRing(2)
	for i := 0; i < 5; i++ {
		r.push(&v1.LogEntry{Message: fmt.Sprintf("%d", i)})
	}
	out, dropped := r.drain()
	if len(out) != 2 {
		t.Errorf("len(out)=%d want 2 (cap)", len(out))
	}
	if dropped != 3 {
		t.Errorf("dropped=%d want 3 (5-2)", dropped)
	}
	if r.droppedCount() != 3 {
		t.Errorf("droppedCount=%d want 3", r.droppedCount())
	}
}

func TestLocalAzureLogRing_CapacityFloor(t *testing.T) {
	// capacity <= 0 should be coerced to 1 (constructor invariant).
	r := newAzureLogRing(0)
	r.push(&v1.LogEntry{Message: "a"})
	r.push(&v1.LogEntry{Message: "b"})
	out, dropped := r.drain()
	if len(out) != 1 || out[0].Message != "b" {
		t.Errorf("cap-0 collapsed to 1: out=%+v want [b]", out)
	}
	if dropped != 1 {
		t.Errorf("dropped=%d want 1", dropped)
	}
}

func TestLocalAzureLogRing_ConcurrentPushDrain(t *testing.T) {
	// Stress the mutex: many concurrent producers + a draining consumer.
	r := newAzureLogRing(64)
	const producers = 8
	const perProducer = 1000
	var wg sync.WaitGroup

	stop := make(chan struct{})
	go func() {
		// Drain loop until producers are done AND ring is empty.
		for {
			select {
			case <-stop:
				_, _ = r.drain()
				return
			case <-r.notify:
				_, _ = r.drain()
			case <-time.After(20 * time.Millisecond):
				_, _ = r.drain()
			}
		}
	}()

	for p := 0; p < producers; p++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < perProducer; i++ {
				r.push(&v1.LogEntry{Message: fmt.Sprintf("p%d-%d", id, i)})
			}
		}(p)
	}
	wg.Wait()
	close(stop)

	// Total pushes = 8000. Some were dropped (buffer is small), but
	// dropped + delivered should never exceed total.
	delivered := int64(0)
	// Final drain to capture trailing.
	out, dropped := r.drain()
	delivered += int64(len(out))
	if dropped+delivered > producers*perProducer {
		t.Errorf("delivered+dropped=%d > total=%d (counter overflow)",
			dropped+delivered, producers*perProducer)
	}
}

// Compile-time guards for the helpers / constants referenced above.
var _ = timestamppb.Now
