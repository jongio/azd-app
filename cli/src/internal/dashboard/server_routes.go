package dashboard

import (
	"context"
	"embed"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/healthcheck"
	"github.com/jongio/azd-app/cli/src/internal/rpc"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
	"github.com/jongio/azd-app/cli/src/internal/version"
)

//go:embed dist
var staticFiles embed.FS

// setupRoutes configures HTTP routes.
func (s *Server) setupRoutes() {
	// Serve static files from embedded FS first (before catch-all patterns)
	distFS, err := fs.Sub(staticFiles, "dist")
	if err != nil {
		log.Printf("Warning: Failed to load static files: %v", err)
		s.mux.HandleFunc("/", s.handleFallback)
		return
	}

	// API endpoints (these take precedence over the file server)
	// Wrap handlers with MethodGuard middleware for proper HTTP method validation
	s.mux.HandleFunc("/api/ping", MethodGuard(s.handlePing, http.MethodGet))
	s.mux.HandleFunc("/api/project", MethodGuard(s.handleGetProject, http.MethodGet))
	s.mux.HandleFunc("/api/services", MethodGuard(s.handleGetServices, http.MethodGet))
	s.mux.HandleFunc("/api/services/start", MethodGuard(s.handleStartService, http.MethodPost))
	s.mux.HandleFunc("/api/services/stop", MethodGuard(s.handleStopService, http.MethodPost))
	s.mux.HandleFunc("/api/services/restart", MethodGuard(s.handleRestartService, http.MethodPost))
	s.mux.HandleFunc("/api/logs", MethodGuard(s.handleGetLogs, http.MethodGet))
	s.mux.HandleFunc("/api/logs/stream", MethodGuard(s.handleLogStream, http.MethodGet))
	s.mux.HandleFunc("/api/logs/classifications", s.handleClassificationsRouter)
	s.mux.HandleFunc("/api/logs/classifications/", s.handleClassificationsRouter)
	s.mux.HandleFunc("/api/logs/preferences", s.handlePreferencesRouter)
	s.mux.HandleFunc("/api/mode", s.handleModeRouter)
	// V2 Azure endpoints (request/response model)
	s.mux.HandleFunc("/api/azure/enable", MethodGuard(s.handleEnableAzureLogging, http.MethodPost)) // Enable Azure logging in azure.yaml
	s.mux.HandleFunc("/api/azure/services", MethodGuard(s.handleAzureServices, http.MethodGet))
	s.mux.HandleFunc("/api/azure/logs", MethodGuard(s.handleAzureLogs, http.MethodGet))
	s.mux.HandleFunc("/api/azure/logs/stream", MethodGuard(s.handleAzureLogsStream, http.MethodGet)) // WebSocket streaming for Azure logs
	s.mux.HandleFunc("/api/azure/logs/health", MethodGuard(s.handleAzureLogsHealth, http.MethodGet))
	s.mux.HandleFunc("/api/azure/logs/setup-state", MethodGuard(s.handleAzureSetupState, http.MethodGet))                       // Setup state detection for setup guide
	s.mux.HandleFunc("/api/azure/logs/verify", MethodGuard(s.handleAzureLogsVerify, http.MethodPost))                           // Verify log connectivity for a service
	s.mux.HandleFunc("/api/azure/diagnostic-settings/check", MethodGuard(s.handleAzureDiagnosticSettingsCheck, http.MethodGet)) // Check diagnostic settings for all services
	s.mux.HandleFunc("/api/azure/diagnostics", MethodGuard(s.handleAzureDiagnostics, http.MethodGet))                           // Comprehensive diagnostics for all services
	s.mux.HandleFunc("/api/azure/workspace/verify", MethodGuard(s.handleAzureWorkspaceVerify, http.MethodPost))                 // Verify workspace connection by querying for recent logs
	s.mux.HandleFunc("/api/azure/bicep-template", MethodGuard(s.handleAzureBicepTemplate, http.MethodGet))                      // Generate consolidated Bicep template for all detected services
	s.mux.HandleFunc("/api/azure/logs/config", s.handleAzureLogConfigRouter)                                                    // Get/save log config per service
	s.mux.HandleFunc("/api/azure/tables", MethodGuard(s.handleAzureTables, http.MethodGet))                                     // List available Log Analytics tables
	s.mux.HandleFunc("/api/azure/query", s.handleAzureQueryRouter)
	s.mux.HandleFunc("/api/ws", s.handleWebSocket)
	s.mux.HandleFunc("/api/health", s.handleHealthCheck)
	s.mux.HandleFunc("/api/health/stream", MethodGuard(s.handleHealthStream, http.MethodGet))
	s.mux.HandleFunc("/api/environment", MethodGuard(s.handleGetEnvironment, http.MethodGet))

	// Connect-RPC handlers run side-by-side with the legacy REST/WS routes
	// during the transport migration (ADR-0001). Mount AFTER the REST
	// patterns so the explicit /api/* prefixes take precedence and Connect
	// owns its own /azdapp.v1.* prefix.
	rpc.Mount(s.mux, rpc.Dependencies{
		Broadcast:  s.broadcast,
		Version:    version.Version,
		Project:    rpc.ProjectSourceFunc(service.ParseAzureYaml),
		ProjectDir: s.projectDir,
		Mode: rpc.ModeStoreFuncs{
			Get: s.getCurrentMode,
			Set: s.setCurrentMode,
		},
		ServicesLister:    rpc.ServiceListerFunc(serviceinfo.GetServiceInfo),
		ServicesLifecycle: newServicesLifecycleAdapter(s),
		BicepFactory:      newBicepGeneratorFactory(s),
		// HealthSource constructs a fresh HealthStreamManager per call,
		// matching the legacy /api/health/stream behaviour: each stream
		// gets isolated change-detection state. The constructor is
		// inexpensive (one struct + map allocation) and avoids leaking
		// state across reconnects, including the "stale previousStates
		// after a service rename" failure mode.
		Health: rpc.HealthSourceFunc(func(ctx context.Context, serviceFilter []string) (*healthcheck.HealthReport, error) {
			mgr, err := NewHealthStreamManager(s.projectDir)
			if err != nil {
				return nil, err
			}
			return mgr.PerformHealthCheck(ctx, serviceFilter)
		}),
		// StateTransitions is intentionally left unset: the dashboard
		// does not currently instantiate a monitor.StateMonitor, so
		// HealthService.StreamStateTransitions returns Unimplemented.
		// Wiring the source belongs in a follow-up batch alongside the
		// monitor lifecycle work; stubbing it here would expose a
		// permanently-empty stream that's worse than the explicit
		// Unimplemented signal.

		// Logs wires the LogsService umbrella to the same primitives
		// the legacy REST handlers use: service.GetLogManager for log
		// reads + subscriptions, the package-private loadAzureYaml /
		// saveAzureYaml pair (guarded by classificationsMu so REST and
		// Connect handlers serialise classification writes), and
		// getOrCreateConfigClient for the preferences blob. Closing
		// over those helpers keeps them un-exported and avoids leaking
		// the classifications mutex through a wrapper struct.
		Logs: newLogsStoreFuncs(s),
	})

	// Serve static files
	fileServer := http.FileServer(http.FS(distFS))
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Check if the requested file exists in the embedded FS
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Try to open the file
		f, err := distFS.Open(strings.TrimPrefix(path, "/"))
		if err != nil {
			// File doesn't exist - serve index.html for client-side routing
			// This handles routes like /console, /services, /environment, /metrics
			indexFile, indexErr := distFS.Open("index.html")
			if indexErr != nil {
				http.NotFound(w, r)
				return
			}
			defer func() { _ = indexFile.Close() }()

			indexContent, readErr := io.ReadAll(indexFile)
			if readErr != nil {
				http.Error(w, "Failed to read index.html", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(indexContent)
			return
		}
		_ = f.Close()

		// File exists, serve it normally
		fileServer.ServeHTTP(w, r)
	})
}
