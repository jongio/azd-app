// Package dashboard provides a web-based user interface for monitoring and managing services.
package dashboard

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/azdconfig"
	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-app/cli/src/internal/dashboard/broadcast"
	"github.com/jongio/azd-app/cli/src/internal/portmanager"
	"github.com/jongio/azd-app/cli/src/internal/rpc"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

var (
	servers   = make(map[string]*Server) // Key: normalized project directory path
	serversMu sync.Mutex
)

// Server represents the dashboard HTTP server.
type Server struct {
	port         int
	mux          *http.ServeMux
	server       *http.Server
	projectDir   string
	stopChan     chan struct{}
	stopOnce     sync.Once     // Ensure stopChan is only closed once
	shutdownChan chan struct{} // Signals the run process to initiate graceful shutdown
	shutdownOnce sync.Once     // Ensure shutdownChan is only closed once
	started      bool          // Track if server was successfully started
	startedMu    sync.Mutex    // Protect started flag
	configClient azdconfig.ConfigClient
	currentMode  service.LogMode // Current log source mode (local or azure)
	modeMu       sync.RWMutex    // Protect currentMode
	azureYamlMu  sync.RWMutex    // Protect azure.yaml read/write across Connect handlers
	sessionToken string          // Per-session auth token for RPC endpoints

	// broadcast fans coarse-grained UI events out to Connect
	// StreamBroadcast subscribers. See cli/src/internal/dashboard/broadcast
	// for back-pressure policy. Always non-nil; constructed in newServer.
	broadcast *broadcast.Manager
}

// GetServer returns the dashboard server instance for the specified project.
// Creates a new instance if one doesn't exist for this project.
func GetServer(projectDir string) *Server {
	serversMu.Lock()
	defer serversMu.Unlock()

	absPath, key := normalizeProjectPath(projectDir)

	// Return existing server if already created
	if srv, exists := servers[key]; exists {
		return srv
	}

	// Create new server instance for this project
	srv := &Server{
		port:         0, // Will be assigned by port manager
		mux:          http.NewServeMux(),
		projectDir:   absPath,
		stopChan:     make(chan struct{}),
		shutdownChan: make(chan struct{}),
		currentMode:  service.LogModeLocal, // Default to local mode
		broadcast:    broadcast.New(),
		sessionToken: rpc.GenerateSessionToken(),
	}
	srv.setupRoutes()
	servers[key] = srv

	return srv
}

// GetURL returns the dashboard URL if the server is started, empty string otherwise.
func (s *Server) GetURL() string {
	s.startedMu.Lock()
	defer s.startedMu.Unlock()
	if !s.started || s.port == 0 {
		return ""
	}
	return fmt.Sprintf("http://localhost:%d", s.port)
}

// ShutdownChan returns a channel that is closed when a remote shutdown is requested.
// The run orchestrator selects on this to initiate graceful shutdown.
func (s *Server) ShutdownChan() <-chan struct{} {
	return s.shutdownChan
}

// RequestShutdown signals the run process to initiate graceful shutdown.
// Safe to call multiple times.
func (s *Server) RequestShutdown() {
	s.shutdownOnce.Do(func() {
		close(s.shutdownChan)
	})
}

// getCurrentMode is a thread-safe accessor for currentMode. Exported via
// rpc.ModeStoreFuncs so the Connect ModeService handler can read state
// without grabbing s.modeMu directly.
func (s *Server) getCurrentMode() service.LogMode {
	s.modeMu.RLock()
	defer s.modeMu.RUnlock()
	return s.currentMode
}

// setCurrentMode is the matching writer. The Connect ModeService handler
// validates the mode (and azure.yaml availability for AZURE) before
// calling this; this function performs no validation of its own and
// simply enforces the mutex contract.
func (s *Server) setCurrentMode(m service.LogMode) {
	s.modeMu.Lock()
	defer s.modeMu.Unlock()
	s.currentMode = m
}

// Start starts the dashboard server on an assigned port.
func (s *Server) Start() (string, error) {
	// Use port manager to get a persistent port for the dashboard
	portMgr := portmanager.GetPortManager(s.projectDir)

	// Get preferred port (either existing or new random port)
	preferredPort, err := s.generatePreferredPort(portMgr)
	if err != nil {
		return "", err
	}

	// Use FindAndReservePort to atomically find and reserve a port
	// This eliminates the TOCTOU race between port checking and binding
	reservation, err := portMgr.FindAndReservePort(constants.DashboardServiceName, preferredPort)
	if err != nil {
		return "", fmt.Errorf("failed to reserve port for dashboard: %w", err)
	}

	port := reservation.Port

	// Release reservation just before server binds
	// The server must bind immediately after this
	if err := reservation.Release(); err != nil {
		slog.Warn("failed to release port reservation", "error", err)
	}

	s.port = port
	s.server = &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", port),
		Handler:           s.buildHandler(),
		ReadHeaderTimeout: 10 * time.Second,
		// WriteTimeout must be 0 (disabled) because Connect-RPC server-streaming
		// handlers (StreamHealth, StreamLocalLogs, StreamBroadcast) are long-lived.
		// Go's WriteTimeout is an absolute deadline from request header read — not
		// a per-write idle timeout — so any non-zero value kills streams after that
		// duration, causing ERR_INCOMPLETE_CHUNKED_ENCODING on the client.
		// Security: this server binds to 127.0.0.1 only (not internet-facing).
		// ReadHeaderTimeout guards against slowloris; IdleTimeout reclaims idle
		// keep-alive connections.
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in background
	errChan := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("dashboard server error", "error", err)
			errChan <- err
		}
	}()

	// Give server time to start
	time.Sleep(constants.ServerStartupDelay)

	// Mark as started
	s.startedMu.Lock()
	s.started = true
	s.startedMu.Unlock()

	// Check if there was an immediate error (like port already in use).
	// This must be the only reader of errChan until it completes: the
	// post-startup monitor below is started afterwards so it cannot consume
	// a bind failure first and leave Start reporting success for a server
	// that never came up.
	select {
	case err := <-errChan:
		// Check if this is a port-in-use error
		if strings.Contains(err.Error(), "bind") || strings.Contains(err.Error(), "address already in use") {
			fmt.Fprintf(os.Stderr, "\n⚠️  Dashboard port %d is already in use.\n", port)
			fmt.Fprintf(os.Stderr, "This might indicate another 'azd app run' instance is already running for this project.\n")
			fmt.Fprintf(os.Stderr, "Attempting to find an alternative port...\n\n")
		}

		// Port binding failed, try to find an alternative port
		if altPort, retryErr := s.retryWithAlternativePort(portMgr); retryErr == nil {
			return fmt.Sprintf("http://localhost:%d", altPort), nil
		}
		return "", fmt.Errorf("dashboard server failed to start: %w", err)
	default:
		// Server started successfully
	}

	// Startup succeeded, so ownership of errChan passes to a long-lived
	// monitor that reports failures occurring after this point.
	go func() {
		select {
		case err := <-errChan:
			if strings.Contains(err.Error(), "bind") || strings.Contains(err.Error(), "address already in use") {
				slog.Warn("dashboard server encountered port conflict after startup", "error", err)
				slog.Warn("another instance may be running; check for other azd app run processes")
			} else {
				slog.Error("dashboard server encountered error after startup", "error", err)
			}
		case <-s.stopChan:
			return
		}
	}()

	url := fmt.Sprintf("http://localhost:%d", port)

	// Store dashboard port in azdconfig for other commands to discover
	s.registerPortInConfig(port)
	// Persist the session token so azd app stop can authenticate across processes.
	writeTokenFile(s.projectDir, s.sessionToken)

	return url, nil
}

// buildHandler returns the composed HTTP handler for the dashboard server.
// All server construction paths (primary start and port retry) must call this
// to ensure security middleware is applied uniformly.
//
// Middleware order (outermost → innermost):
//   - hostAllow:       rejects non-loopback Host headers (CWE-346 DNS rebinding)
//   - securityHeaders: sets defensive response headers (CWE-693)
//   - s.mux:           routes to registered handlers
func (s *Server) buildHandler() http.Handler {
	return hostAllow(securityHeaders(s.mux))
}

// Stop stops the dashboard server and releases its port assignment.
// Safe to call multiple times - will only stop if server was successfully started.
func (s *Server) Stop() error {
	// Check if server was ever started
	s.startedMu.Lock()
	wasStarted := s.started
	s.started = false // Mark as stopped
	s.startedMu.Unlock()

	// Always clean up from servers map, even if never started
	serversMu.Lock()
	_, key := normalizeProjectPath(s.projectDir)
	delete(servers, key)
	serversMu.Unlock()

	if !wasStarted {
		return nil // Server was never started, nothing more to stop
	}

	// Close stopChan only once to prevent panic
	s.stopOnce.Do(func() {
		close(s.stopChan)
	})

	// Clear dashboard port from azdconfig so other commands know it's not running
	s.clearPortFromConfig()
	// Remove the session-token file so stale tokens cannot authenticate.
	removeTokenFile(s.projectDir)

	// Close the HTTP server first to drain in-flight handlers.
	// This ensures no handlers are running when we nil dependent resources.
	if s.server != nil {
		_ = s.server.Close()
	}

	// Tear down Connect StreamBroadcast subscribers AFTER http.Server.Close
	// so any in-flight stream handlers have already started exiting.
	s.broadcast.StopAll()

	// Now safe — no more handlers running
	if s.configClient != nil {
		s.configClient.Close()
		s.configClient = nil
	}

	return nil
}
