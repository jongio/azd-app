package dashboard

import (
	"bytes"
	"context"
	"crypto/subtle"
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

// sessionTokenPlaceholder is the verbatim string present in index.html.
// The server replaces it (with injectSessionToken) before every HTML response,
// injecting the real per-session token into the page it serves.
const sessionTokenPlaceholder = `<meta name="azd-session-token" content="">`

// injectSessionToken returns a copy of indexHTML with the placeholder
// meta-tag content attribute set to token.
//
// Safety: rpc.GenerateSessionToken returns a hex-encoded random string
// (characters 0-9 and a-f only). Those characters are unconditionally safe
// inside an HTML attribute value — no further encoding is needed. If the
// placeholder is absent the original slice is returned unchanged; the client
// will read an empty string and every RPC call will be rejected by the
// server-side auth interceptor, which is the safe failure mode (CWE-306).
func injectSessionToken(indexHTML []byte, token string) []byte {
	replacement := []byte(`<meta name="azd-session-token" content="` + token + `">`)
	return bytes.Replace(indexHTML, []byte(sessionTokenPlaceholder), replacement, 1)
}

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
	if err := rpc.Mount(s.mux, rpc.Dependencies{
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
	}); err != nil {
		// An empty SessionToken is a programming error — the server always
		// generates one at startup. Panicking here surfaces the misconfiguration
		// immediately rather than serving every request unauthenticated.
		panic("rpc.Mount: " + err.Error())
	}

	// Shutdown endpoint: signals the run process to initiate graceful teardown.
	// Used by `azd app stop` from a separate terminal.
	s.mux.HandleFunc("POST /api/shutdown", func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Session-Token")
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.sessionToken)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		s.RequestShutdown()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"shutting_down"}`))
	})

	// Pre-read index.html once. Pre-compute the token-injected variant used for
	// all HTML responses. The /api/session-token HTTP endpoint has been removed
	// (CWE-306/419/352): the token is now delivered exclusively via this meta
	// tag, which is only accessible to same-origin page loads and is invisible
	// to DNS-rebinding or same-machine HTTP sniffing attacks.
	indexContent, indexReadErr := fs.ReadFile(distFS, "index.html")
	var tokenizedIndex []byte
	if indexReadErr == nil {
		tokenizedIndex = injectSessionToken(indexContent, s.sessionToken)
	}

	// serveIndex writes the tokenized index.html with headers that prevent
	// the token-carrying HTML from being stored by browser or proxy caches.
	serveIndex := func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(tokenizedIndex)
	}

	// Serve static files
	fileServer := http.FileServer(http.FS(distFS))
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Check if the requested file exists in the embedded FS
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Root and /index.html always serve the token-injected page so the
		// SPA receives its auth credential on the initial load.
		if path == "/index.html" {
			if indexReadErr != nil {
				http.NotFound(w, r)
				return
			}
			serveIndex(w)
			return
		}

		// Try to open the file from the embedded FS.
		f, err := distFS.Open(strings.TrimPrefix(path, "/"))
		if err != nil {
			// File doesn't exist — serve index.html for client-side routing.
			// This handles routes like /console, /services, /environment, /metrics.
			if indexReadErr != nil {
				http.NotFound(w, r)
				return
			}
			serveIndex(w)
			return
		}
		_ = f.Close()

		// File exists; serve it normally (static assets can be cached freely).
		fileServer.ServeHTTP(w, r)
	})
}
