// Package dashboard provides a web-based user interface for monitoring and managing services.
package dashboard

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/azdconfig"
	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-app/cli/src/internal/portmanager"
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
	clients      map[*clientConn]bool
	clientsMu    sync.RWMutex
	rateLimiter  *connectionRateLimiter // Per-server rate limiter
	stopChan     chan struct{}
	stopOnce     sync.Once  // Ensure stopChan is only closed once
	started      bool       // Track if server was successfully started
	startedMu    sync.Mutex // Protect started flag
	configClient azdconfig.ConfigClient
	currentMode  service.LogMode // Current log source mode (local or azure)
	modeMu       sync.RWMutex    // Protect currentMode
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
		port:        0, // Will be assigned by port manager
		mux:         http.NewServeMux(),
		projectDir:  absPath,
		clients:     make(map[*clientConn]bool),
		rateLimiter: newConnectionRateLimiter(),
		stopChan:    make(chan struct{}),
		currentMode: service.LogModeLocal, // Default to local mode
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
		log.Printf("Warning: failed to release port reservation: %v", err)
	}

	s.port = port
	s.server = &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", port),
		Handler:           securityHeaders(s.mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Start server in background
	errChan := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Dashboard server error: %v", err)
			errChan <- err
		}
	}()

	// Monitor for server errors in background
	go func() {
		select {
		case err := <-errChan:
			if strings.Contains(err.Error(), "bind") || strings.Contains(err.Error(), "address already in use") {
				log.Printf("Dashboard server encountered port conflict after startup: %v", err)
				log.Printf("Another instance may be running. Check for other 'azd app run' processes.")
			} else {
				log.Printf("Dashboard server encountered error after startup: %v", err)
			}
		case <-s.stopChan:
			return
		}
	}()

	// Give server time to start
	time.Sleep(constants.ServerStartupDelay)

	// Mark as started
	s.startedMu.Lock()
	s.started = true
	s.startedMu.Unlock()

	// Check if there was an immediate error (like port already in use)
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

	url := fmt.Sprintf("http://localhost:%d", port)

	// Store dashboard port in azdconfig for other commands to discover
	s.registerPortInConfig(port)

	return url, nil
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

	// Gracefully close all active WebSocket connections with timeout
	done := make(chan struct{})
	go func() {
		s.clientsMu.Lock()
		for client := range s.clients {
			_ = client.client.close()
			delete(s.clients, client)
		}
		s.clientsMu.Unlock()
		close(done)
	}()

	// Wait for graceful shutdown with timeout
	select {
	case <-done:
		// All clients closed gracefully
	case <-time.After(5 * time.Second):
		log.Printf("Warning: WebSocket shutdown timeout, some connections may not have closed gracefully")
	}

	// Clear dashboard port from azdconfig so other commands know it's not running
	s.clearPortFromConfig()

	// Close the config client if it was created
	if s.configClient != nil {
		s.configClient.Close()
		s.configClient = nil
	}

	// Shutdown rate limiter
	if s.rateLimiter != nil {
		s.rateLimiter.shutdown()
		s.rateLimiter = nil
	}

	if s.server != nil {
		return s.server.Close()
	}
	return nil
}
