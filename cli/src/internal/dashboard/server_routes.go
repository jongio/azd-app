package dashboard

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"
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
		slog.Warn("failed to load static files", "error", err)
		s.mux.HandleFunc("/", s.handleFallback)
		return
	}

	// Connect-RPC handlers own the entire API surface post-PR4.
	// All legacy /api/* REST handlers and /api/ws WebSocket have been
	// removed; dashboard clients use the azdapp.v1.* Connect services.
	rpc.Mount(s.mux, rpc.Dependencies{
		Broadcast:    s.broadcast,
		Version:      version.Version,
		SessionToken: s.sessionToken,
		Project:      rpc.ProjectSourceFunc(service.ParseAzureYaml),
		ProjectDir:   s.projectDir,
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

		// AzureService: 15 RPCs split across 4 narrow sub-stores
		// (config / catalog / logs client / diagnostics) wired in
		// rpc_azure_adapter.go. Closures over s.azureYamlMu serialise
		// azure.yaml writes with the parallel REST handlers.
		Azure: newAzureStoreFuncs(s),
	})

	// Serve session token for dashboard authentication.
	// The React app fetches this on startup and includes it in all RPC calls.
	s.mux.HandleFunc("/api/session-token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(s.sessionToken))
	})

	// Pre-read index.html once for SPA client-side routing fallback
	indexContent, indexReadErr := fs.ReadFile(distFS, "index.html")

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
			if indexReadErr != nil {
				http.NotFound(w, r)
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
