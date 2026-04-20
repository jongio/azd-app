package rpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
	"github.com/jongio/azd-app/cli/src/internal/azure"
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
