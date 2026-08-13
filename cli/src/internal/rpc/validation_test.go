package rpc

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
)

// =============================================================================
// validateQuery
// =============================================================================

func TestValidateQuery_OversizedRejected(t *testing.T) {
	huge := strings.Repeat("a", maxQueryBytes+1)
	if err := validateQuery(huge); err == nil {
		t.Fatal("oversized query should be rejected, got nil error")
	}
}

func TestValidateQuery_ExactSizeAccepted(t *testing.T) {
	exact := strings.Repeat("a", maxQueryBytes)
	if err := validateQuery(exact); err != nil {
		t.Fatalf("query at exact size limit should be accepted, got: %v", err)
	}
}

func TestValidateQuery_EmptyAccepted(t *testing.T) {
	// Empty string is valid for validateQuery itself; callers are responsible
	// for rejecting empty strings at a higher level (e.g. "query is required").
	if err := validateQuery(""); err != nil {
		t.Fatalf("empty string should be accepted by validateQuery, got: %v", err)
	}
}

func TestValidateQuery_NormalKQLAccepted(t *testing.T) {
	q := "ContainerAppConsoleLogs_CL\n| where TimeGenerated > ago(1h)\r\n| project Message\t| take 100"
	if err := validateQuery(q); err != nil {
		t.Fatalf("normal KQL with \\n, \\r, \\t should be accepted, got: %v", err)
	}
}

func TestValidateQuery_NonPrintableBytesRejected(t *testing.T) {
	cases := []struct {
		name  string
		query string
	}{
		{"null_byte", "SELECT *\x00FROM table"},
		{"soh_control", "query\x01injection"},
		{"bel_control", "kql\x07string"},
		{"del_0x7f", "query\x7fhere"},
		{"embedded_esc", "\x1bkql"},
		{"leading_control", "\x02ContainerAppConsoleLogs_CL | take 10"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateQuery(tc.query); err == nil {
				t.Fatalf("query %q with non-printable byte should be rejected, got nil error", tc.query)
			}
		})
	}
}

// =============================================================================
// validateTables
// =============================================================================

func TestValidateTables_KnownTablesAccepted(t *testing.T) {
	known := []string{
		"ContainerAppConsoleLogs_CL",
		"ContainerAppSystemLogs_CL",
		"AppServiceConsoleLogs",
		"AppServiceHTTPLogs",
		"FunctionAppLogs",
		"ContainerLogV2",
		"KubeEvents",
		"ContainerInstanceLog_CL",
	}
	if err := validateTables(known); err != nil {
		t.Fatalf("all known tables should be accepted, got: %v", err)
	}
}

func TestValidateTables_UnknownTableRejected(t *testing.T) {
	cases := []struct {
		name   string
		tables []string
	}{
		{"injection_attempt", []string{"ContainerAppConsoleLogs_CL", "MaliciousTable; DROP TABLE logs--"}},
		{"completely_unknown", []string{"AppRequests"}},
		{"single_char", []string{"X"}},
		{"empty_string", []string{""}},
		{"valid_then_invalid", []string{"AppServiceHTTPLogs", "UnsupportedAuditLogs"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateTables(tc.tables); err == nil {
				t.Fatalf("tables %v should be rejected, got nil error", tc.tables)
			}
		})
	}
}

func TestValidateTables_EmptySliceAccepted(t *testing.T) {
	if err := validateTables(nil); err != nil {
		t.Fatalf("nil table slice should be accepted, got: %v", err)
	}
	if err := validateTables([]string{}); err != nil {
		t.Fatalf("empty table slice should be accepted, got: %v", err)
	}
}

// =============================================================================
// Handler-level integration: SaveAzureLogConfig validation
// =============================================================================

func TestSaveAzureLogConfig_UnknownTableRejected(t *testing.T) {
	h := newStubHandler(stubAzureStore())

	_, err := h.SaveAzureLogConfig(context.Background(),
		connect.NewRequest(&v1.SaveAzureLogConfigRequest{
			Service: "api",
			Mode:    v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_TABLES,
			Tables:  []string{"InjectedMaliciousTable"},
		}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("unknown table: got err=%v code=%v, want InvalidArgument", err, connect.CodeOf(err))
	}
}

func TestSaveAzureLogConfig_OversizedQueryRejected(t *testing.T) {
	h := newStubHandler(stubAzureStore())

	_, err := h.SaveAzureLogConfig(context.Background(),
		connect.NewRequest(&v1.SaveAzureLogConfigRequest{
			Service: "api",
			Mode:    v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_CUSTOM,
			Query:   strings.Repeat("x", maxQueryBytes+1),
		}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("oversized query: got err=%v code=%v, want InvalidArgument", err, connect.CodeOf(err))
	}
}

func TestSaveAzureLogConfig_NonPrintableQueryRejected(t *testing.T) {
	h := newStubHandler(stubAzureStore())

	_, err := h.SaveAzureLogConfig(context.Background(),
		connect.NewRequest(&v1.SaveAzureLogConfigRequest{
			Service: "api",
			Mode:    v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_CUSTOM,
			Query:   "ContainerAppConsoleLogs_CL\x00| take 1",
		}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("non-printable byte: got err=%v code=%v, want InvalidArgument", err, connect.CodeOf(err))
	}
}

// TestSaveAzureLogConfig_ValidationGateBlocksStore verifies that the new
// validation runs BEFORE the store is called: a handler wired with a store
// stub that panics on write should never trigger when input is invalid.
func TestSaveAzureLogConfig_ValidationGateBlocksStore(t *testing.T) {
	funcs := stubAzureStore()
	funcs.SaveServiceLogConfigFn = func(string, []string, string) error {
		t.Fatal("store must not be called when input validation fails")
		return nil
	}
	h := newStubHandler(funcs)

	// Unknown table, store must not be called.
	_, err := h.SaveAzureLogConfig(context.Background(),
		connect.NewRequest(&v1.SaveAzureLogConfigRequest{
			Service: "api",
			Mode:    v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_TABLES,
			Tables:  []string{"NotATable"},
		}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("got err=%v code=%v, want InvalidArgument", err, connect.CodeOf(err))
	}
}

// =============================================================================
// Handler-level integration: SaveServiceQuery validation
// =============================================================================

func TestSaveServiceQuery_OversizedQueryRejected(t *testing.T) {
	funcs := stubAzureStore()
	funcs.SaveServiceCustomQueryFn = func(string, string) error {
		t.Fatal("store must not be called when input validation fails")
		return nil
	}
	h := newStubHandler(funcs)

	_, err := h.SaveServiceQuery(context.Background(),
		connect.NewRequest(&v1.SaveServiceQueryRequest{
			Service: "api",
			Query:   strings.Repeat("q", maxQueryBytes+1),
		}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("oversized query: got err=%v code=%v, want InvalidArgument", err, connect.CodeOf(err))
	}
}

func TestSaveServiceQuery_NonPrintableByteRejected(t *testing.T) {
	funcs := stubAzureStore()
	funcs.SaveServiceCustomQueryFn = func(string, string) error {
		t.Fatal("store must not be called when input validation fails")
		return nil
	}
	h := newStubHandler(funcs)

	_, err := h.SaveServiceQuery(context.Background(),
		connect.NewRequest(&v1.SaveServiceQueryRequest{
			Service: "api",
			Query:   "FunctionAppLogs | where Level == \x00'Error'",
		}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("non-printable byte: got err=%v code=%v, want InvalidArgument", err, connect.CodeOf(err))
	}
}

// =============================================================================
// auditMutation, smoke: must not panic or error; slog output verified
// structurally by the slog default handler.
// =============================================================================

func TestAuditMutation_DoesNotPanic(t *testing.T) {
	// Should not panic with empty peer or extra keys.
	auditMutation(context.Background(), "TestOp", "")
	auditMutation(context.Background(), "TestOp", "127.0.0.1:9999",
		"service", "api", "mode", "tables")
}
