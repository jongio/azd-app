package rpc

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"

	"github.com/jongio/azd-app/cli/src/internal/azure"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// AzureService is the umbrella store backing AzureHandler. It is split into
// four narrow sub-stores (config, catalog, logs client, diagnostics) so each
// surface stays well below the 30-method ceiling and tests can stub one slice
// without faking the whole Azure stack.
//
// Sub-store rationale (per ADR-0001 commit-B-1 plan):
//   - AzureConfigStore   — azure.yaml read / write helpers, all guarded by
//     dashboard's azureYamlMu in production.
//   - AzureCatalog       — read-only metadata about workspaces, tables, default
//     queries; no I/O against Azure.
//   - AzureLogsClient    — Azure-side reads (fetch logs, verify workspace,
//     create realtime streamer / credentials).
//   - AzureDiagnostics   — multi-step diagnostics + setup probes that return
//     opaque JSON (passed through *structpb.Struct on the wire).
type AzureService interface {
	AzureConfigStore
	AzureCatalog
	AzureLogsClient
	AzureDiagnostics
}

// AzureConfigStore covers the azure.yaml read / write surface. Production
// closes over loadAzureYaml/saveAzureYaml under azureYamlMu so REST and
// Connect handlers serialise edits to azure.yaml.
type AzureConfigStore interface {
	// LoadAzureYaml returns the parsed azure.yaml (or an empty struct
	// when none exists). Implementations MUST take a read lock on the
	// shared mutex for the duration of the read.
	LoadAzureYaml() (*service.AzureYaml, error)
	// SaveAzureYaml persists the supplied azure.yaml atomically under
	// the write lock.
	SaveAzureYaml(ay *service.AzureYaml) error
	// EnableGlobalAnalytics initialises the logs.analytics block if
	// missing. The bool reports whether logging was already on; the
	// handler reuses the legacy "already enabled" message in that case.
	EnableGlobalAnalytics() (alreadyEnabled bool, err error)
	// SaveServiceLogConfig writes a per-service tables-or-query config
	// non-destructively (mirrors yamlutil.UpdateServiceLogsConfig).
	// `tables` is nil when mode is custom; `query` empty when tables.
	SaveServiceLogConfig(serviceName string, tables []string, query string) error
	// SaveServiceCustomQuery writes a custom KQL query for one service
	// to azure.yaml (services.<name>.logs.analytics.query). Mirrors
	// handleSaveAzureQuery's full RMW path.
	SaveServiceCustomQuery(serviceName, query string) error
}

// AzureCatalog covers read-only metadata used by the dashboard's table-picker
// and query-editor. None of these helpers touch Azure directly; they are
// pure functions over compiled-in tables and the local environment.
type AzureCatalog interface {
	ServiceNamesFromEnv() []string
	WorkspaceID(ctx context.Context) (string, error)
	DefaultQuery(resourceType string) string
	RecommendedTables(resourceType string) []string
	AllKnownTables() []azure.TableInfo
	IsRecommendedTable(name, resourceType string) bool
	TableCategories() map[string]azure.TableCategory
	SubstituteQueryPlaceholders(query, serviceName, timespan string) string
	TruncateMiddle(s string, max int) string
	// ListLiveTables attempts an ARM call against the workspace; returns
	// (nil, err) when credentials are unavailable, allowing the handler
	// to fall back to AllKnownTables (matches legacy handleAzureTables).
	ListLiveTables(ctx context.Context, workspaceID string) ([]azure.TableInfo, error)
}

// AzureLogsClient covers Azure-side reads for the logs surface. All paths
// depend on a TokenCredential; missing credentials surface as
// connect.CodeUnauthenticated through mapAzureError.
type AzureLogsClient interface {
	FetchLogs(ctx context.Context, config azure.StandaloneLogsConfig) ([]azure.LogEntry, error)
	ResolveResource(ctx context.Context, serviceName string) (*azure.AzureResource, error)
	NewRealtimeStreamer(resourceType azure.ResourceType, config azure.StreamerConfig) (azure.RealtimeLogStreamer, error)
	NewLogAnalyticsCredential() (azcore.TokenCredential, error)
	VerifyWorkspace(ctx context.Context, req *azure.WorkspaceVerificationRequest) (*azure.WorkspaceVerificationResponse, error)
	VerifyServiceLogs(ctx context.Context, serviceName string) (*VerifyServiceLogsResult, error)
}

// VerifyServiceLogsResult is the small projection of azure_setup.go's
// VerifyLogsResponse the proto needs.
type VerifyServiceLogsResult struct {
	Success      bool
	Message      string
	RowsReturned int32
}

// AzureDiagnostics covers the diagnostics / setup-state probes. Outputs are
// returned as opaque map[string]any blobs so the handler can wrap them in
// *structpb.Struct without leaking proto types into this layer.
type AzureDiagnostics interface {
	CheckDiagnosticSettings(ctx context.Context) (*azure.DiagnosticSettingsCheckResponse, error)
	RunDiagnostics(ctx context.Context) (any, error)
	GetSetupState(ctx context.Context) (any, error)
	GetHealth(ctx context.Context) AzureHealthSnapshot
}

// AzureHealthSnapshot is the typed form of /api/azure/logs/health.
// Mirrors HealthCheckResponse exactly; the handler maps Status+per-check
// strings into the proto enums.
type AzureHealthSnapshot struct {
	Status  string // "healthy" | "degraded" | "error"
	Checks  []AzureHealthCheckSnapshot
	DocsURL string
}

// AzureHealthCheckSnapshot is one row of the health table.
type AzureHealthCheckSnapshot struct {
	Name    string
	Status  string // "pass" | "warn" | "fail"
	Message string
	Fix     string
}

// AzureStoreFuncs adapts plain function values to AzureService. Mirrors
// LogsStoreFuncs: every field MUST be set; nil values cause a panic at
// call time which is the fail-loud signal we want for misconfigured
// wiring. The dashboard adapter (rpc_azure_adapter.go) populates these
// from its package-private helpers without exporting them.
type AzureStoreFuncs struct {
	// AzureConfigStore
	LoadAzureYamlFn          func() (*service.AzureYaml, error)
	SaveAzureYamlFn          func(ay *service.AzureYaml) error
	EnableGlobalAnalyticsFn  func() (bool, error)
	SaveServiceLogConfigFn   func(serviceName string, tables []string, query string) error
	SaveServiceCustomQueryFn func(serviceName, query string) error

	// AzureCatalog
	ServiceNamesFromEnvFn         func() []string
	WorkspaceIDFn                 func(ctx context.Context) (string, error)
	DefaultQueryFn                func(resourceType string) string
	RecommendedTablesFn           func(resourceType string) []string
	AllKnownTablesFn              func() []azure.TableInfo
	IsRecommendedTableFn          func(name, resourceType string) bool
	TableCategoriesFn             func() map[string]azure.TableCategory
	SubstituteQueryPlaceholdersFn func(query, serviceName, timespan string) string
	TruncateMiddleFn              func(s string, max int) string
	ListLiveTablesFn              func(ctx context.Context, workspaceID string) ([]azure.TableInfo, error)

	// AzureLogsClient
	FetchLogsFn                 func(ctx context.Context, config azure.StandaloneLogsConfig) ([]azure.LogEntry, error)
	ResolveResourceFn           func(ctx context.Context, serviceName string) (*azure.AzureResource, error)
	NewRealtimeStreamerFn       func(rt azure.ResourceType, cfg azure.StreamerConfig) (azure.RealtimeLogStreamer, error)
	NewLogAnalyticsCredentialFn func() (azcore.TokenCredential, error)
	VerifyWorkspaceFn           func(ctx context.Context, req *azure.WorkspaceVerificationRequest) (*azure.WorkspaceVerificationResponse, error)
	VerifyServiceLogsFn         func(ctx context.Context, serviceName string) (*VerifyServiceLogsResult, error)

	// AzureDiagnostics
	CheckDiagnosticSettingsFn func(ctx context.Context) (*azure.DiagnosticSettingsCheckResponse, error)
	RunDiagnosticsFn          func(ctx context.Context) (any, error)
	GetSetupStateFn           func(ctx context.Context) (any, error)
	GetHealthFn               func(ctx context.Context) AzureHealthSnapshot
}

// --- AzureConfigStore ---

func (f AzureStoreFuncs) LoadAzureYaml() (*service.AzureYaml, error) { return f.LoadAzureYamlFn() }
func (f AzureStoreFuncs) SaveAzureYaml(ay *service.AzureYaml) error  { return f.SaveAzureYamlFn(ay) }
func (f AzureStoreFuncs) EnableGlobalAnalytics() (bool, error)       { return f.EnableGlobalAnalyticsFn() }
func (f AzureStoreFuncs) SaveServiceLogConfig(s string, t []string, q string) error {
	return f.SaveServiceLogConfigFn(s, t, q)
}
func (f AzureStoreFuncs) SaveServiceCustomQuery(s, q string) error {
	return f.SaveServiceCustomQueryFn(s, q)
}

// --- AzureCatalog ---

func (f AzureStoreFuncs) ServiceNamesFromEnv() []string { return f.ServiceNamesFromEnvFn() }
func (f AzureStoreFuncs) WorkspaceID(ctx context.Context) (string, error) {
	return f.WorkspaceIDFn(ctx)
}
func (f AzureStoreFuncs) DefaultQuery(rt string) string        { return f.DefaultQueryFn(rt) }
func (f AzureStoreFuncs) RecommendedTables(rt string) []string { return f.RecommendedTablesFn(rt) }
func (f AzureStoreFuncs) AllKnownTables() []azure.TableInfo    { return f.AllKnownTablesFn() }
func (f AzureStoreFuncs) IsRecommendedTable(n, rt string) bool { return f.IsRecommendedTableFn(n, rt) }
func (f AzureStoreFuncs) TableCategories() map[string]azure.TableCategory {
	return f.TableCategoriesFn()
}
func (f AzureStoreFuncs) SubstituteQueryPlaceholders(q, s, ts string) string {
	return f.SubstituteQueryPlaceholdersFn(q, s, ts)
}
func (f AzureStoreFuncs) TruncateMiddle(s string, m int) string { return f.TruncateMiddleFn(s, m) }
func (f AzureStoreFuncs) ListLiveTables(ctx context.Context, w string) ([]azure.TableInfo, error) {
	return f.ListLiveTablesFn(ctx, w)
}

// --- AzureLogsClient ---

func (f AzureStoreFuncs) FetchLogs(ctx context.Context, c azure.StandaloneLogsConfig) ([]azure.LogEntry, error) {
	return f.FetchLogsFn(ctx, c)
}
func (f AzureStoreFuncs) ResolveResource(ctx context.Context, s string) (*azure.AzureResource, error) {
	return f.ResolveResourceFn(ctx, s)
}
func (f AzureStoreFuncs) NewRealtimeStreamer(rt azure.ResourceType, c azure.StreamerConfig) (azure.RealtimeLogStreamer, error) {
	return f.NewRealtimeStreamerFn(rt, c)
}
func (f AzureStoreFuncs) NewLogAnalyticsCredential() (azcore.TokenCredential, error) {
	return f.NewLogAnalyticsCredentialFn()
}
func (f AzureStoreFuncs) VerifyWorkspace(ctx context.Context, req *azure.WorkspaceVerificationRequest) (*azure.WorkspaceVerificationResponse, error) {
	return f.VerifyWorkspaceFn(ctx, req)
}
func (f AzureStoreFuncs) VerifyServiceLogs(ctx context.Context, s string) (*VerifyServiceLogsResult, error) {
	return f.VerifyServiceLogsFn(ctx, s)
}

// --- AzureDiagnostics ---

func (f AzureStoreFuncs) CheckDiagnosticSettings(ctx context.Context) (*azure.DiagnosticSettingsCheckResponse, error) {
	return f.CheckDiagnosticSettingsFn(ctx)
}
func (f AzureStoreFuncs) RunDiagnostics(ctx context.Context) (any, error) {
	return f.RunDiagnosticsFn(ctx)
}
func (f AzureStoreFuncs) GetSetupState(ctx context.Context) (any, error) {
	return f.GetSetupStateFn(ctx)
}
func (f AzureStoreFuncs) GetHealth(ctx context.Context) AzureHealthSnapshot {
	return f.GetHealthFn(ctx)
}
