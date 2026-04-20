package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
	"github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1/azdappv1connect"
	"github.com/jongio/azd-app/cli/src/internal/azure"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// AzureHandler implements azdapp.v1.AzureService via the AzureService
// sub-store umbrella. Behaviour mirrors the legacy REST handlers in
// dashboard/azure_*.go (parity is enforced by ADR-0001's parallel-stack
// rule).
type AzureHandler struct {
	store AzureService
	// Per-RPC timeouts mirror the legacy net/http handlers' timeoutContext
	// values so observable latency caps don't shift during the migration.
	defaultTimeout    time.Duration
	diagnosticTimeout time.Duration
	verifyTimeout     time.Duration
}

// NewAzureHandler builds a handler. store is required; nil panics so a
// misconfigured Mount fails at startup, not at first request.
func NewAzureHandler(store AzureService) *AzureHandler {
	if store == nil {
		panic("rpc: NewAzureHandler: store must not be nil")
	}
	return &AzureHandler{
		store:             store,
		defaultTimeout:    30 * time.Second,
		diagnosticTimeout: 60 * time.Second,
		verifyTimeout:     60 * time.Second,
	}
}

var _ azdappv1connect.AzureServiceHandler = (*AzureHandler)(nil)

// =============================================================================
// Error mapping
// =============================================================================

// mapAzureError translates an internal error into a Connect status. The
// table mirrors the legacy http-status-code switch in
// dashboard/azure_logs_handlers.go (mapAzureErrorToInfo + handleAzureLogs)
// and the substring checks in handleAzureWorkspaceVerify. NO_RESULTS is
// intentionally NOT classified here - call sites that treat it as success
// must short-circuit BEFORE calling mapAzureError (see GetAzureLogs).
//
// Order of precedence:
//  1. ctx errors win (caller saw a deadline / cancel and that's what we
//     report regardless of the underlying error).
//  2. Typed *azure.AzureLogsError (legacy enum-like Code field).
//  3. Substring fallback for the legacy fmt.Errorf paths in
//     azure/standalone_logs.go and azure/verification.go that don't
//     return a typed error.
func mapAzureError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		if errors.Is(ctxErr, context.DeadlineExceeded) {
			return connect.NewError(connect.CodeDeadlineExceeded, err)
		}
		return connect.NewError(connect.CodeCanceled, err)
	}

	var ae *azure.AzureLogsError
	if errors.As(err, &ae) {
		switch ae.Code {
		case "AUTH_EXPIRED", "AUTH_REQUIRED":
			return connect.NewError(connect.CodeUnauthenticated, err)
		case "NO_WORKSPACE", "NO_PERMISSION", "NOT_DEPLOYED":
			return connect.NewError(connect.CodeNotFound, err)
		case "CLIENT_ERROR":
			return connect.NewError(connect.CodeInvalidArgument, err)
		case "QUERY_FAILED", "UNKNOWN":
			return connect.NewError(connect.CodeInternal, err)
		}
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "invalid timespan"):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case strings.Contains(msg, "toomanyrequests"),
		strings.Contains(msg, "429"),
		strings.Contains(msg, "throttl"),
		strings.Contains(msg, "quota exceeded"):
		return connect.NewError(connect.CodeResourceExhausted, err)
	case strings.Contains(msg, "no resources found"),
		strings.Contains(msg, "workspace not found"),
		strings.Contains(msg, "no log analytics workspace"):
		return connect.NewError(connect.CodeNotFound, err)
	case strings.Contains(msg, "credentials"),
		strings.Contains(msg, "unauthenticated"),
		strings.Contains(msg, "unauthorized"):
		return connect.NewError(connect.CodeUnauthenticated, err)
	}

	return connect.NewError(connect.CodeInternal, err)
}

// isNoResults returns true when err signals NO_RESULTS - the legacy
// handler treats this as success-with-empty-list, not an error.
func isNoResults(err error) bool {
	if err == nil {
		return false
	}
	var ae *azure.AzureLogsError
	if errors.As(err, &ae) && ae.Code == "NO_RESULTS" {
		return true
	}
	return false
}

// =============================================================================
// Enum mapping helpers (legacy string -> proto enum)
// =============================================================================

func toProtoCheckStatus(s string) v1.AzureCheckStatus {
	switch s {
	case "pass":
		return v1.AzureCheckStatus_AZURE_CHECK_STATUS_PASS
	case "warn":
		return v1.AzureCheckStatus_AZURE_CHECK_STATUS_WARN
	case "fail":
		return v1.AzureCheckStatus_AZURE_CHECK_STATUS_FAIL
	default:
		return v1.AzureCheckStatus_AZURE_CHECK_STATUS_UNSPECIFIED
	}
}

func toProtoOverallStatus(s string) v1.AzureOverallStatus {
	switch s {
	case "healthy":
		return v1.AzureOverallStatus_AZURE_OVERALL_STATUS_HEALTHY
	case "degraded":
		return v1.AzureOverallStatus_AZURE_OVERALL_STATUS_DEGRADED
	case "error":
		return v1.AzureOverallStatus_AZURE_OVERALL_STATUS_ERROR
	default:
		return v1.AzureOverallStatus_AZURE_OVERALL_STATUS_UNSPECIFIED
	}
}

func toProtoDiagnosticStatus(s string) v1.DiagnosticSettingsStatus {
	switch s {
	case "configured":
		return v1.DiagnosticSettingsStatus_DIAGNOSTIC_SETTINGS_STATUS_CONFIGURED
	case "not-configured":
		return v1.DiagnosticSettingsStatus_DIAGNOSTIC_SETTINGS_STATUS_NOT_CONFIGURED
	case "error":
		return v1.DiagnosticSettingsStatus_DIAGNOSTIC_SETTINGS_STATUS_ERROR
	default:
		return v1.DiagnosticSettingsStatus_DIAGNOSTIC_SETTINGS_STATUS_UNSPECIFIED
	}
}

func toProtoWorkspaceVerificationStatus(s string) v1.WorkspaceVerificationStatus {
	switch s {
	case "success":
		return v1.WorkspaceVerificationStatus_WORKSPACE_VERIFICATION_STATUS_SUCCESS
	case "partial":
		return v1.WorkspaceVerificationStatus_WORKSPACE_VERIFICATION_STATUS_PARTIAL
	case "error":
		return v1.WorkspaceVerificationStatus_WORKSPACE_VERIFICATION_STATUS_ERROR
	default:
		return v1.WorkspaceVerificationStatus_WORKSPACE_VERIFICATION_STATUS_UNSPECIFIED
	}
}

func toProtoServiceVerificationStatus(s string) v1.ServiceVerificationStatus {
	switch s {
	case "ok":
		return v1.ServiceVerificationStatus_SERVICE_VERIFICATION_STATUS_OK
	case "no-logs":
		return v1.ServiceVerificationStatus_SERVICE_VERIFICATION_STATUS_NO_LOGS
	case "error":
		return v1.ServiceVerificationStatus_SERVICE_VERIFICATION_STATUS_ERROR
	case "diagnostic-not-configured":
		return v1.ServiceVerificationStatus_SERVICE_VERIFICATION_STATUS_DIAGNOSTIC_NOT_CONFIGURED
	default:
		return v1.ServiceVerificationStatus_SERVICE_VERIFICATION_STATUS_UNSPECIFIED
	}
}

func azureLogConfigModeToString(m v1.AzureLogConfigMode) string {
	switch m {
	case v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_TABLES:
		return "tables"
	case v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_CUSTOM:
		return "custom"
	default:
		return ""
	}
}

func stringToAzureLogConfigMode(s string) v1.AzureLogConfigMode { //nolint:unused // kept symmetric with azureLogConfigModeToString for round-trip use in future RPCs (e.g. SetServiceQuery returning current mode)
	switch s {
	case "tables":
		return v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_TABLES
	case "custom":
		return v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_CUSTOM
	default:
		return v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_UNSPECIFIED
	}
}

// azureResourceTypeToString maps the proto enum to the azure package's
// stringly-typed ResourceType values. UNSPECIFIED defaults to
// "containerapp" matching legacy ListAzureTables behaviour.
func azureResourceTypeToString(rt v1.AzureResourceType) string {
	switch rt {
	case v1.AzureResourceType_AZURE_RESOURCE_TYPE_CONTAINER_APP:
		return string(azure.ResourceTypeContainerApp)
	case v1.AzureResourceType_AZURE_RESOURCE_TYPE_APP_SERVICE:
		return string(azure.ResourceTypeAppService)
	case v1.AzureResourceType_AZURE_RESOURCE_TYPE_FUNCTION_APP:
		return string(azure.ResourceTypeFunction)
	default:
		return string(azure.ResourceTypeContainerApp)
	}
}

// toStruct round-trips an arbitrary Go value through JSON into a
// *structpb.Struct. Used to passthrough setup-state and diagnostics
// payloads whose schemas are still being shaped (see the Struct Usage
// Inventory note in ADR-0001).
func toStruct(v any) (*structpb.Struct, error) {
	if v == nil {
		return nil, nil
	}
	if s, ok := v.(*structpb.Struct); ok {
		return s, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		// Fall back to wrapping non-object payloads under "value".
		m = map[string]any{"value": json.RawMessage(b)}
	}
	out := &structpb.Struct{}
	if err := protojson.Unmarshal(mustJSON(m), out); err != nil {
		return nil, err
	}
	return out, nil
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		// All inputs reach mustJSON via toStruct's already-validated
		// json.Marshal; a re-marshal failure is a programming error.
		return []byte(`{}`)
	}
	return b
}

// toProtoAzureLogEntry converts an azure.LogEntry to the wire proto. Reuses
// convertAzureLogLevel + toProtoLogLevel from logs.go for consistency with
// the LogsService side of the migration.
func toProtoAzureLogEntry(e azure.LogEntry) *v1.LogEntry {
	return &v1.LogEntry{
		Service:   e.Service,
		Message:   e.Message,
		Level:     toProtoLogLevel(convertAzureLogLevelToService(e.Level)),
		Timestamp: timestamppb.New(e.Timestamp),
		Stream:    v1.LogStream_LOG_STREAM_STDOUT,
		Source:    v1.LogSource_LOG_SOURCE_AZURE,
	}
}

// convertAzureLogLevelToService duplicates dashboard/azure_logs_conversion.go
// so the rpc package doesn't import the dashboard package (would create a
// cycle: dashboard imports rpc to mount handlers).
func convertAzureLogLevelToService(l azure.LogLevel) service.LogLevel {
	switch l {
	case azure.LogLevelInfo:
		return service.LogLevelInfo
	case azure.LogLevelWarn:
		return service.LogLevelWarn
	case azure.LogLevelError:
		return service.LogLevelError
	case azure.LogLevelDebug:
		return service.LogLevelDebug
	default:
		return service.LogLevelInfo
	}
}

// =============================================================================
// Unary RPCs
// =============================================================================

func (h *AzureHandler) EnableAzureLogging(
	ctx context.Context,
	_ *connect.Request[v1.EnableAzureLoggingRequest],
) (*connect.Response[v1.EnableAzureLoggingResponse], error) {
	already, err := h.store.EnableGlobalAnalytics()
	if err != nil {
		return nil, mapAzureError(ctx, err)
	}
	msg := "Azure logging enabled! Refresh to start viewing logs."
	if already {
		msg = "Azure logging is already enabled"
	}
	return connect.NewResponse(&v1.EnableAzureLoggingResponse{
		Enabled: true,
		Message: msg,
	}), nil
}

func (h *AzureHandler) GetAzureServices(
	_ context.Context,
	_ *connect.Request[v1.GetAzureServicesRequest],
) (*connect.Response[v1.GetAzureServicesResponse], error) {
	return connect.NewResponse(&v1.GetAzureServicesResponse{
		Services: h.store.ServiceNamesFromEnv(),
	}), nil
}

func (h *AzureHandler) GetAzureLogs(
	ctx context.Context,
	req *connect.Request[v1.GetAzureLogsRequest],
) (*connect.Response[v1.GetAzureLogsResponse], error) {
	since := time.Hour
	if req.Msg.SinceSeconds > 0 {
		since = time.Duration(req.Msg.SinceSeconds) * time.Second
	}
	tail := 500
	if req.Msg.Tail > 0 {
		tail = int(req.Msg.Tail)
	}
	if tail > 10_000 {
		tail = 10_000
	}

	var services []string
	if req.Msg.Service != "" {
		services = []string{req.Msg.Service}
	}

	cfg := azure.StandaloneLogsConfig{
		Services: services,
		Since:    since,
		Limit:    tail,
	}

	logs, err := h.store.FetchLogs(ctx, cfg)
	if err != nil {
		if isNoResults(err) {
			return connect.NewResponse(&v1.GetAzureLogsResponse{
				Entries:   nil,
				Timestamp: timestamppb.Now(),
				Count:     0,
			}), nil
		}
		return nil, mapAzureError(ctx, err)
	}

	entries := make([]*v1.LogEntry, len(logs))
	for i, l := range logs {
		entries[i] = toProtoAzureLogEntry(l)
	}
	return connect.NewResponse(&v1.GetAzureLogsResponse{
		Entries:   entries,
		Timestamp: timestamppb.Now(),
		Count:     int32(len(entries)),
	}), nil
}

func (h *AzureHandler) GetAzureLogsHealth(
	ctx context.Context,
	_ *connect.Request[v1.GetAzureLogsHealthRequest],
) (*connect.Response[v1.GetAzureLogsHealthResponse], error) {
	snap := h.store.GetHealth(ctx)
	checks := make([]*v1.AzureHealthCheck, 0, len(snap.Checks))
	for _, c := range snap.Checks {
		checks = append(checks, &v1.AzureHealthCheck{
			Name:    c.Name,
			Status:  toProtoCheckStatus(c.Status),
			Message: c.Message,
			Fix:     c.Fix,
		})
	}
	return connect.NewResponse(&v1.GetAzureLogsHealthResponse{
		Status:    toProtoOverallStatus(snap.Status),
		Checks:    checks,
		DocsUrl:   snap.DocsURL,
		Timestamp: timestamppb.Now(),
	}), nil
}

func (h *AzureHandler) GetAzureSetupState(
	ctx context.Context,
	_ *connect.Request[v1.GetAzureSetupStateRequest],
) (*connect.Response[v1.GetAzureSetupStateResponse], error) {
	state, err := h.store.GetSetupState(ctx)
	if err != nil {
		return nil, mapAzureError(ctx, err)
	}
	pb, err := toStruct(state)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.GetAzureSetupStateResponse{State: pb}), nil
}

func (h *AzureHandler) VerifyAzureLogs(
	ctx context.Context,
	req *connect.Request[v1.VerifyAzureLogsRequest],
) (*connect.Response[v1.VerifyAzureLogsResponse], error) {
	if req.Msg.Service == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("service is required"))
	}
	ctx, cancel := context.WithTimeout(ctx, h.verifyTimeout)
	defer cancel()
	r, err := h.store.VerifyServiceLogs(ctx, req.Msg.Service)
	if err != nil {
		return nil, mapAzureError(ctx, err)
	}
	return connect.NewResponse(&v1.VerifyAzureLogsResponse{
		Success:      r.Success,
		Message:      r.Message,
		RowsReturned: r.RowsReturned,
		QueriedAt:    timestamppb.Now(),
	}), nil
}

func (h *AzureHandler) CheckDiagnosticSettings(
	ctx context.Context,
	_ *connect.Request[v1.CheckDiagnosticSettingsRequest],
) (*connect.Response[v1.CheckDiagnosticSettingsResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, h.defaultTimeout)
	defer cancel()
	res, err := h.store.CheckDiagnosticSettings(ctx)
	if err != nil {
		return nil, mapAzureError(ctx, err)
	}
	out := &v1.CheckDiagnosticSettingsResponse{
		WorkspaceId: res.WorkspaceID,
		Services:    make(map[string]*v1.DiagnosticSettingResult, len(res.Services)),
	}
	for name, r := range res.Services {
		if r == nil {
			continue
		}
		out.Services[name] = &v1.DiagnosticSettingResult{
			Status:                toProtoDiagnosticStatus(string(r.Status)),
			ResourceId:            r.ResourceID,
			DiagnosticSettingName: r.DiagnosticSettingName,
			Error:                 r.Error,
			WorkspaceId:           r.WorkspaceID,
		}
	}
	return connect.NewResponse(out), nil
}

func (h *AzureHandler) GetAzureDiagnostics(
	ctx context.Context,
	_ *connect.Request[v1.GetAzureDiagnosticsRequest],
) (*connect.Response[v1.GetAzureDiagnosticsResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, h.diagnosticTimeout)
	defer cancel()
	res, err := h.store.RunDiagnostics(ctx)
	if err != nil {
		return nil, mapAzureError(ctx, err)
	}
	pb, err := toStruct(res)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.GetAzureDiagnosticsResponse{Diagnostics: pb}), nil
}

func (h *AzureHandler) VerifyWorkspace(
	ctx context.Context,
	req *connect.Request[v1.VerifyWorkspaceRequest],
) (*connect.Response[v1.VerifyWorkspaceResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, h.verifyTimeout)
	defer cancel()

	r := &azure.WorkspaceVerificationRequest{
		Services: req.Msg.Services,
		Timespan: req.Msg.Timespan,
	}
	res, err := h.store.VerifyWorkspace(ctx, r)
	if err != nil {
		return nil, mapAzureError(ctx, err)
	}

	results := make(map[string]*v1.ServiceVerificationResult, len(res.Results))
	for name, sr := range res.Results {
		if sr == nil {
			continue
		}
		entry := &v1.ServiceVerificationResult{
			Status:       toProtoServiceVerificationStatus(string(sr.Status)),
			Service:      name,
			RowsReturned: int32(sr.LogCount),
			Error:        sr.Error,
		}
		if sr.LastLogTime != nil {
			entry.QueriedAt = timestamppb.New(*sr.LastLogTime)
		}
		if sr.Message != "" {
			if details, derr := toStruct(map[string]any{"message": sr.Message}); derr == nil {
				entry.Details = details
			}
		}
		results[name] = entry
	}

	return connect.NewResponse(&v1.VerifyWorkspaceResponse{
		Status: toProtoWorkspaceVerificationStatus(string(res.Status)),
		Workspace: &v1.WorkspaceInfo{
			Id:   res.Workspace.ID,
			Name: res.Workspace.Name,
		},
		Results:  results,
		Guidance: res.Guidance,
	}), nil
}

func (h *AzureHandler) GetAzureLogConfig(
	ctx context.Context,
	req *connect.Request[v1.GetAzureLogConfigRequest],
) (*connect.Response[v1.GetAzureLogConfigResponse], error) {
	if req.Msg.Service == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("service is required"))
	}
	resp := &v1.GetAzureLogConfigResponse{
		Service:      req.Msg.Service,
		Mode:         v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_TABLES,
		ResourceType: "containerapp",
	}

	ay, err := h.store.LoadAzureYaml()
	if err != nil || ay == nil {
		resp.Tables = h.store.RecommendedTables("containerapp")
		return connect.NewResponse(resp), nil
	}

	if svc, ok := ay.Services[req.Msg.Service]; ok {
		if svc.Host != "" {
			resp.ResourceType = svc.Host
		}
		if svc.Logs != nil && svc.Logs.Analytics != nil {
			cfg := svc.Logs.Analytics
			if len(cfg.Tables) > 0 {
				resp.Tables = cfg.Tables
				resp.Mode = v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_TABLES
			}
			if cfg.Query != "" {
				resp.Query = cfg.Query
				resp.Mode = v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_CUSTOM
			}
		}
	}

	if resp.Mode == v1.AzureLogConfigMode_AZURE_LOG_CONFIG_MODE_TABLES && len(resp.Tables) == 0 {
		resp.Tables = h.store.RecommendedTables(resp.ResourceType)
	}
	_ = ctx
	return connect.NewResponse(resp), nil
}

func (h *AzureHandler) SaveAzureLogConfig(
	ctx context.Context,
	req *connect.Request[v1.SaveAzureLogConfigRequest],
) (*connect.Response[v1.SaveAzureLogConfigResponse], error) {
	if req.Msg.Service == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("service is required"))
	}
	mode := azureLogConfigModeToString(req.Msg.Mode)
	if mode == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("mode must be tables or custom"))
	}
	if mode == "tables" && len(req.Msg.Tables) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("tables required when mode is tables"))
	}
	if mode == "custom" && req.Msg.Query == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("query required when mode is custom"))
	}

	var tables []string
	var query string
	if mode == "tables" {
		tables = req.Msg.Tables
	} else {
		query = req.Msg.Query
	}

	if err := h.store.SaveServiceLogConfig(req.Msg.Service, tables, query); err != nil {
		return nil, mapAzureError(ctx, err)
	}

	return connect.NewResponse(&v1.SaveAzureLogConfigResponse{
		Service: req.Msg.Service,
		Mode:    req.Msg.Mode,
		Tables:  req.Msg.Tables,
		Query:   req.Msg.Query,
	}), nil
}

func (h *AzureHandler) ListAzureTables(
	ctx context.Context,
	req *connect.Request[v1.ListAzureTablesRequest],
) (*connect.Response[v1.ListAzureTablesResponse], error) {
	resourceType := azureResourceTypeToString(req.Msg.ResourceType)

	workspaceID, _ := h.store.WorkspaceID(ctx)

	var tables []azure.TableInfo
	if workspaceID != "" {
		if live, err := h.store.ListLiveTables(ctx, workspaceID); err == nil && len(live) > 0 {
			tables = live
		}
	}
	if len(tables) == 0 {
		tables = h.store.AllKnownTables()
	}

	out := &v1.ListAzureTablesResponse{
		Tables:      make([]*v1.AzureTableInfo, 0, len(tables)),
		Recommended: h.store.RecommendedTables(resourceType),
		Workspace:   h.store.TruncateMiddle(workspaceID, 20),
	}
	for _, t := range tables {
		out.Tables = append(out.Tables, &v1.AzureTableInfo{
			Name:        t.Name,
			Description: t.Description,
			Recommended: h.store.IsRecommendedTable(t.Name, resourceType),
		})
	}

	cats := h.store.TableCategories()
	out.Categories = make([]*v1.AzureTableCategory, 0, len(cats))
	for name, c := range cats {
		out.Categories = append(out.Categories, &v1.AzureTableCategory{
			Name:        name,
			DisplayName: c.DisplayName,
			Tables:      c.Tables,
		})
	}
	return connect.NewResponse(out), nil
}

func (h *AzureHandler) GetServiceQuery(
	ctx context.Context,
	req *connect.Request[v1.GetServiceQueryRequest],
) (*connect.Response[v1.GetServiceQueryResponse], error) {
	if req.Msg.Service == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("service is required"))
	}
	var query, resourceType string
	if ay, err := h.store.LoadAzureYaml(); err == nil && ay != nil {
		if svc, ok := ay.Services[req.Msg.Service]; ok &&
			svc.Logs != nil && svc.Logs.Analytics != nil &&
			svc.Logs.Analytics.Query != "" {
			query = svc.Logs.Analytics.Query
			resourceType = "custom"
		}
	}
	if query == "" {
		resourceType = "containerapp"
		query = h.store.DefaultQuery(resourceType)
		query = h.store.SubstituteQueryPlaceholders(query, req.Msg.Service, "30m")
	}
	_ = ctx
	return connect.NewResponse(&v1.GetServiceQueryResponse{
		Service:      req.Msg.Service,
		ResourceType: resourceType,
		Query:        query,
	}), nil
}

func (h *AzureHandler) SaveServiceQuery(
	ctx context.Context,
	req *connect.Request[v1.SaveServiceQueryRequest],
) (*connect.Response[v1.SaveServiceQueryResponse], error) {
	if req.Msg.Service == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("service is required"))
	}
	if req.Msg.Query == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("query is required"))
	}
	if err := h.store.SaveServiceCustomQuery(req.Msg.Service, req.Msg.Query); err != nil {
		return nil, mapAzureError(ctx, err)
	}
	return connect.NewResponse(&v1.SaveServiceQueryResponse{
		Service:      req.Msg.Service,
		ResourceType: "custom",
		Query:        req.Msg.Query,
	}), nil
}
