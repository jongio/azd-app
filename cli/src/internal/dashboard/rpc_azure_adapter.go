package dashboard

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"

	"github.com/jongio/azd-app/cli/src/internal/azure"
	"github.com/jongio/azd-app/cli/src/internal/rpc"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/yamlutil"
)

// newAzureStoreFuncs builds the rpc.AzureStoreFuncs adapter that backs the
// AzureService Connect handler. It composes four narrow sub-stores
// (config / catalog / logs client / diagnostics) so each surface stays well
// under the 30-method ceiling and tests can stub one slice without faking
// the whole Azure stack. See rpc/azure_stores.go for the contracts.
//
// All azure.yaml mutations close over s.azureYamlMu so the parallel REST
// surface (handleEnableAzureLogging, handleSaveAzureLogConfig,
// handleSaveAzureQuery) and the Connect surface serialise edits during the
// parallel-stack window described in ADR-0001.
func newAzureStoreFuncs(s *Server) rpc.AzureStoreFuncs {
	cfg := newAzureConfigFuncs(s)
	cat := newAzureCatalogFuncs(s)
	cli := newAzureLogsClientFuncs(s)
	diag := newAzureDiagnosticsFuncs(s)
	return rpc.AzureStoreFuncs{
		// Config
		LoadAzureYamlFn:          cfg.LoadAzureYamlFn,
		SaveAzureYamlFn:          cfg.SaveAzureYamlFn,
		EnableGlobalAnalyticsFn:  cfg.EnableGlobalAnalyticsFn,
		SaveServiceLogConfigFn:   cfg.SaveServiceLogConfigFn,
		SaveServiceCustomQueryFn: cfg.SaveServiceCustomQueryFn,

		// Catalog
		ServiceNamesFromEnvFn:         cat.ServiceNamesFromEnvFn,
		WorkspaceIDFn:                 cat.WorkspaceIDFn,
		DefaultQueryFn:                cat.DefaultQueryFn,
		RecommendedTablesFn:           cat.RecommendedTablesFn,
		AllKnownTablesFn:              cat.AllKnownTablesFn,
		IsRecommendedTableFn:          cat.IsRecommendedTableFn,
		TableCategoriesFn:             cat.TableCategoriesFn,
		SubstituteQueryPlaceholdersFn: cat.SubstituteQueryPlaceholdersFn,
		TruncateMiddleFn:              cat.TruncateMiddleFn,
		ListLiveTablesFn:              cat.ListLiveTablesFn,

		// Logs client
		FetchLogsFn:                 cli.FetchLogsFn,
		ResolveResourceFn:           cli.ResolveResourceFn,
		NewRealtimeStreamerFn:       cli.NewRealtimeStreamerFn,
		NewLogAnalyticsCredentialFn: cli.NewLogAnalyticsCredentialFn,
		VerifyWorkspaceFn:           cli.VerifyWorkspaceFn,
		VerifyServiceLogsFn:         cli.VerifyServiceLogsFn,

		// Diagnostics
		CheckDiagnosticSettingsFn: diag.CheckDiagnosticSettingsFn,
		RunDiagnosticsFn:          diag.RunDiagnosticsFn,
		GetSetupStateFn:           diag.GetSetupStateFn,
		GetHealthFn:               diag.GetHealthFn,
	}
}

// newAzureConfigFuncs returns the AzureConfigStore slice. All paths take
// s.azureYamlMu (RLock for reads, Lock for writes) so the Connect handler
// shares serialisation with the REST handlers in azure_logs_handlers.go,
// azure_logs_tables.go and azure_logs_query.go.
func newAzureConfigFuncs(s *Server) rpc.AzureStoreFuncs {
	return rpc.AzureStoreFuncs{
		LoadAzureYamlFn: func() (*service.AzureYaml, error) {
			s.azureYamlMu.RLock()
			defer s.azureYamlMu.RUnlock()
			return loadAzureYaml(s.projectDir)
		},
		SaveAzureYamlFn: func(ay *service.AzureYaml) error {
			s.azureYamlMu.Lock()
			defer s.azureYamlMu.Unlock()
			return saveAzureYaml(s.projectDir, ay)
		},
		EnableGlobalAnalyticsFn: func() (bool, error) {
			s.azureYamlMu.Lock()
			defer s.azureYamlMu.Unlock()
			ay, err := loadAzureYaml(s.projectDir)
			if err != nil {
				return false, err
			}
			if ay.Logs != nil && ay.Logs.Analytics != nil {
				return true, nil
			}
			if ay.Logs == nil {
				ay.Logs = &service.LogsConfig{}
			}
			if ay.Logs.Analytics == nil {
				ay.Logs.Analytics = &service.AnalyticsConfigGlobal{}
			}
			if err := saveAzureYaml(s.projectDir, ay); err != nil {
				return false, err
			}
			return false, nil
		},
		SaveServiceLogConfigFn: func(serviceName string, tables []string, query string) error {
			s.azureYamlMu.Lock()
			defer s.azureYamlMu.Unlock()
			// Mirror handleSaveAzureLogConfig: validate the service exists,
			// then delegate to yamlutil.UpdateServiceLogsConfig so we
			// preserve comments / structure rather than re-marshalling.
			ay, err := loadAzureYaml(s.projectDir)
			if err != nil {
				return err
			}
			if _, ok := ay.Services[serviceName]; !ok {
				return errServiceNotFound
			}
			path := filepath.Join(s.projectDir, "azure.yaml")
			return yamlutil.UpdateServiceLogsConfig(path, serviceName, tables, query)
		},
		SaveServiceCustomQueryFn: func(serviceName, query string) error {
			s.azureYamlMu.Lock()
			defer s.azureYamlMu.Unlock()
			ay, err := loadAzureYaml(s.projectDir)
			if err != nil {
				return err
			}
			svc, ok := ay.Services[serviceName]
			if !ok {
				return errServiceNotFound
			}
			if svc.Logs == nil {
				svc.Logs = &service.ServiceLogsConfig{}
			}
			if svc.Logs.Analytics == nil {
				svc.Logs.Analytics = &service.AnalyticsConfigService{}
			}
			svc.Logs.Analytics.Query = query
			ay.Services[serviceName] = svc
			return saveAzureYaml(s.projectDir, ay)
		},
	}
}

// errServiceNotFound is returned by the config sub-store when a write
// targets a service that isn't declared in azure.yaml. mapAzureError
// surfaces this through the proto layer; the legacy REST surface returns
// 404 for the same condition.
var errServiceNotFound = errors.New("workspace not found: service not declared in azure.yaml")

// newAzureCatalogFuncs returns the AzureCatalog slice. None of these
// helpers touch Azure directly except ListLiveTables, which is best-effort
// and falls back to AllKnownTables in the handler when credentials are
// missing.
func newAzureCatalogFuncs(s *Server) rpc.AzureStoreFuncs {
	return rpc.AzureStoreFuncs{
		ServiceNamesFromEnvFn: func() []string {
			return extractServiceNamesFromEnv()
		},
		WorkspaceIDFn: func(ctx context.Context) (string, error) {
			id, err := getWorkspaceIDFromEnv(ctx)
			if err == nil && id != "" {
				return id, nil
			}
			// Fall back to azure.yaml just like azure_setup.go:677.
			s.azureYamlMu.RLock()
			defer s.azureYamlMu.RUnlock()
			ay, lerr := loadAzureYaml(s.projectDir)
			if lerr == nil && ay.Logs != nil && ay.Logs.Analytics != nil && ay.Logs.Analytics.Workspace != "" {
				return ay.Logs.Analytics.Workspace, nil
			}
			if err != nil {
				return "", err
			}
			return "", nil
		},
		DefaultQueryFn: func(rt string) string {
			return azure.GetDefaultQuery(azure.ResourceType(rt))
		},
		RecommendedTablesFn: func(rt string) []string {
			return azure.GetRecommendedTables(azure.ResourceType(rt))
		},
		AllKnownTablesFn: func() []azure.TableInfo {
			return azure.GetAllKnownTables()
		},
		IsRecommendedTableFn: func(name, rt string) bool {
			return azure.IsRecommendedTable(name, azure.ResourceType(rt))
		},
		TableCategoriesFn: func() map[string]azure.TableCategory {
			return azure.TableCategories
		},
		SubstituteQueryPlaceholdersFn: func(query, serviceName, timespan string) string {
			return substituteQueryPlaceholders(query, serviceName, timespan)
		},
		TruncateMiddleFn: func(str string, max int) string {
			return truncateMiddle(str, max)
		},
		ListLiveTablesFn: func(ctx context.Context, workspaceID string) ([]azure.TableInfo, error) {
			cred, err := newLogAnalyticsCredential()
			if err != nil {
				return nil, err
			}
			client, err := getOrCreateLogAnalyticsClient(ctx, cred, workspaceID)
			if err != nil {
				return nil, err
			}
			return client.ListAvailableTables(ctx)
		},
	}
}

// newAzureLogsClientFuncs returns the AzureLogsClient slice. Closes over
// the package-level fetch / credential helpers so existing test seams in
// azure_logs_test.go continue to apply to both surfaces.
func newAzureLogsClientFuncs(s *Server) rpc.AzureStoreFuncs {
	return rpc.AzureStoreFuncs{
		FetchLogsFn: func(ctx context.Context, c azure.StandaloneLogsConfig) ([]azure.LogEntry, error) {
			if c.ProjectDir == "" {
				c.ProjectDir = s.projectDir
			}
			return fetchAzureLogsStandalone(ctx, c)
		},
		ResolveResourceFn: func(ctx context.Context, serviceName string) (*azure.AzureResource, error) {
			cred, err := newLogAnalyticsCredential()
			if err != nil {
				return nil, err
			}
			disc := azure.NewResourceDiscovery(cred, s.projectDir)
			return disc.GetResource(ctx, serviceName)
		},
		NewRealtimeStreamerFn: func(rt azure.ResourceType, cfg azure.StreamerConfig) (azure.RealtimeLogStreamer, error) {
			return azure.NewRealtimeStreamer(rt, cfg)
		},
		NewLogAnalyticsCredentialFn: func() (azcore.TokenCredential, error) {
			return newLogAnalyticsCredential()
		},
		VerifyWorkspaceFn: func(ctx context.Context, req *azure.WorkspaceVerificationRequest) (*azure.WorkspaceVerificationResponse, error) {
			cred, err := newLogAnalyticsCredential()
			if err != nil {
				return nil, err
			}
			v := azure.NewWorkspaceVerifier(cred, s.projectDir)
			return v.VerifyWorkspace(ctx, req)
		},
		VerifyServiceLogsFn: func(ctx context.Context, serviceName string) (*rpc.VerifyServiceLogsResult, error) {
			// Project handleAzureLogsVerify (azure_setup.go:656-820) down
			// to the small {Success, Message, RowsReturned} struct the
			// proto carries. Workspace lookup mirrors line 675-693;
			// credential check mirrors 695-706; fetch mirrors 708-756;
			// "no logs" handling mirrors 759-771.
			workspaceID, err := getWorkspaceIDFromEnv(ctx)
			if err != nil || workspaceID == "" {
				s.azureYamlMu.RLock()
				ay, lerr := loadAzureYaml(s.projectDir)
				s.azureYamlMu.RUnlock()
				if lerr == nil && ay.Logs != nil && ay.Logs.Analytics != nil {
					workspaceID = ay.Logs.Analytics.Workspace
				}
			}
			if workspaceID == "" {
				return &rpc.VerifyServiceLogsResult{
					Success: false,
					Message: "Log Analytics workspace not configured",
				}, nil
			}
			if _, err := newLogAnalyticsCredential(); err != nil {
				return &rpc.VerifyServiceLogsResult{
					Success: false,
					Message: "Azure credentials not available",
				}, nil
			}
			cfg := azure.StandaloneLogsConfig{
				ProjectDir: s.projectDir,
				Services:   []string{serviceName},
				Since:      15 * time.Minute,
				Limit:      10,
			}
			logs, err := fetchAzureLogsStandalone(ctx, cfg)
			if err != nil {
				if ctx.Err() == context.DeadlineExceeded {
					return &rpc.VerifyServiceLogsResult{
						Success: false,
						Message: "Query timeout - Log Analytics workspace may be slow to respond",
					}, nil
				}
				msg := err.Error()
				if strings.Contains(msg, "no resources found") || strings.Contains(msg, "workspace not found") {
					return &rpc.VerifyServiceLogsResult{
						Success: false,
						Message: fmt.Sprintf("Service '%s' not found or not deployed", serviceName),
					}, nil
				}
				return &rpc.VerifyServiceLogsResult{
					Success: false,
					Message: fmt.Sprintf("Failed to query logs: %v", err),
				}, nil
			}
			if len(logs) == 0 {
				return &rpc.VerifyServiceLogsResult{
					Success:      false,
					RowsReturned: 0,
					Message:      fmt.Sprintf("No logs found for service '%s' in the last 15 minutes", serviceName),
				}, nil
			}
			return &rpc.VerifyServiceLogsResult{
				Success:      true,
				RowsReturned: int32(len(logs)),
				Message:      fmt.Sprintf("Successfully verified log flow for service '%s'", serviceName),
			}, nil
		},
	}
}

// newAzureDiagnosticsFuncs returns the AzureDiagnostics slice. Setup-state
// and run-diagnostics outputs are returned as opaque structs that the RPC
// layer projects into *structpb.Struct via toStruct.
func newAzureDiagnosticsFuncs(s *Server) rpc.AzureStoreFuncs {
	return rpc.AzureStoreFuncs{
		CheckDiagnosticSettingsFn: func(ctx context.Context) (*azure.DiagnosticSettingsCheckResponse, error) {
			cred, err := newLogAnalyticsCredential()
			if err != nil {
				return nil, err
			}
			return azure.NewDiagnosticSettingsChecker(cred, s.projectDir).CheckAllServices(ctx)
		},
		RunDiagnosticsFn: func(ctx context.Context) (any, error) {
			cred, err := newLogAnalyticsCredential()
			if err != nil {
				return nil, err
			}
			return azure.NewDiagnosticsEngine(cred, s.projectDir).RunDiagnostics(ctx)
		},
		GetSetupStateFn: func(ctx context.Context) (any, error) {
			// Build the same SetupStateResponse the REST surface emits.
			// Reuse the Server's existing helpers so both surfaces stay
			// in lockstep without duplicating the probe logic.
			resp := SetupStateResponse{
				Workspace:      s.checkWorkspaceState(),
				Authentication: s.checkAuthState(ctx),
				Services:       s.checkServicesState(ctx),
				Timestamp:      time.Now(),
			}
			step, status := s.determineSetupStep(resp)
			resp.Step = step
			resp.OverallStatus = status
			resp.Issues = s.collectSetupIssues(resp)
			resp.NextSteps = s.generateNextSteps(resp)
			return resp, nil
		},
		GetHealthFn: func(ctx context.Context) rpc.AzureHealthSnapshot {
			// Mirror handleAzureLogsHealth ordering exactly so the
			// Connect surface produces the same snapshot the REST
			// surface (and existing UI) already exercises.
			authCheck := s.checkAuthentication()
			workspaceCheck := s.checkWorkspaceID()
			servicesCheck := s.checkServicesDeployed()
			connectivityCheck := s.checkConnectivity(workspaceCheck.Status == statusPass)

			checks := []rpc.AzureHealthCheckSnapshot{
				toCheckSnapshot(authCheck),
				toCheckSnapshot(workspaceCheck),
				toCheckSnapshot(servicesCheck),
				toCheckSnapshot(connectivityCheck),
			}
			snap := rpc.AzureHealthSnapshot{
				Checks:  checks,
				DocsURL: logsTroubleshootURL,
			}
			snap.Status = s.computeOverallStatus([]HealthCheck{
				authCheck, workspaceCheck, servicesCheck, connectivityCheck,
			})
			return snap
		},
	}
}

func toCheckSnapshot(h HealthCheck) rpc.AzureHealthCheckSnapshot {
	return rpc.AzureHealthCheckSnapshot{
		Name:    h.Name,
		Status:  h.Status,
		Message: h.Message,
		Fix:     h.Fix,
	}
}
