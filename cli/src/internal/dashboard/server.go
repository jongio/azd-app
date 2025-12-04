// Package dashboard provides a web-based user interface for monitoring and managing services.
// It includes a dashboard server with WebSocket support for real-time updates,
// log streaming capabilities, and REST API endpoints for service management.
package dashboard

import (
	"context"
	"crypto/rand"
	"embed"
	"fmt"
	"html"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/azdconfig"
	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-app/cli/src/internal/portmanager"
	"github.com/jongio/azd-app/cli/src/internal/registry"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
)

//go:embed dist
var staticFiles embed.FS

// clientConn wraps a websocket connection with a write mutex for safe concurrent writes.
type clientConn struct {
	client *wsClient // Uses github.com/coder/websocket
}

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
	stopChan     chan struct{}
	started      bool       // Track if server was successfully started
	startedMu    sync.Mutex // Protect started flag
	configClient azdconfig.ConfigClient
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
		port:       0, // Will be assigned by port manager
		mux:        http.NewServeMux(),
		projectDir: absPath,
		clients:    make(map[*clientConn]bool),
		stopChan:   make(chan struct{}),
	}
	srv.setupRoutes()
	servers[key] = srv

	return srv
}

func normalizeProjectPath(projectDir string) (string, string) {
	absPath := projectDir
	if v, err := filepath.Abs(projectDir); err == nil {
		absPath = v
	}
	if v, err := filepath.EvalSymlinks(absPath); err == nil {
		absPath = v
	}
	absPath = filepath.Clean(absPath)

	key := absPath
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}

	return absPath, key
}

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
	s.mux.HandleFunc("/api/ping", s.handlePing)
	s.mux.HandleFunc("/api/project", s.handleGetProject)
	s.mux.HandleFunc("/api/services", s.handleGetServices)
	s.mux.HandleFunc("/api/services/start", s.handleStartService)
	s.mux.HandleFunc("/api/services/stop", s.handleStopService)
	s.mux.HandleFunc("/api/services/restart", s.handleRestartService)
	s.mux.HandleFunc("/api/logs", s.handleGetLogs)
	s.mux.HandleFunc("/api/logs/stream", s.handleLogStream)
	s.mux.HandleFunc("/api/logs/classifications", s.handleClassificationsRouter)
	s.mux.HandleFunc("/api/logs/classifications/", s.handleClassificationsRouter)
	s.mux.HandleFunc("/api/logs/preferences", s.handlePreferencesRouter)
	s.mux.HandleFunc("/api/ws", s.handleWebSocket)
	s.mux.HandleFunc("/api/health", s.handleHealthCheck)
	s.mux.HandleFunc("/api/health/stream", s.handleHealthStream)

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
			defer indexFile.Close()

			indexContent, readErr := io.ReadAll(indexFile)
			if readErr != nil {
				http.Error(w, "Failed to read index.html", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(indexContent)
			return
		}
		f.Close()

		// File exists, serve it normally
		fileServer.ServeHTTP(w, r)
	})
}

// handlePing is a simple health check endpoint to verify the dashboard is running.
func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// handleGetServices returns services for the current project.
func (s *Server) handleGetServices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Use shared serviceinfo package to get merged service data
	services, err := serviceinfo.GetServiceInfo(s.projectDir)
	if err != nil {
		log.Printf("Warning: Failed to get service info: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to get service info", err)
		return
	}

	if err := writeJSON(w, services); err != nil {
		log.Printf("Failed to write JSON response: %v", err)
	}
}

// handleGetProject returns project metadata from azure.yaml.
func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	azureYaml, err := service.ParseAzureYaml(s.projectDir)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to parse azure.yaml", err)
		return
	}

	response := map[string]string{
		"name": azureYaml.Name,
		"dir":  s.projectDir,
	}

	if err := writeJSON(w, response); err != nil {
		log.Printf("Failed to write JSON response: %v", err)
	}
}

// handleWebSocket handles WebSocket connections for live updates.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := acceptWebSocket(w, r)
	if err != nil {
		if err != http.ErrAbortHandler {
			log.Printf("WebSocket upgrade error: %v", err)
		}
		return
	}

	client := newWSClient(conn)
	clientWrapper := &clientConn{client: client}

	s.clientsMu.Lock()
	s.clients[clientWrapper] = true
	s.clientsMu.Unlock()

	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, clientWrapper)
		s.clientsMu.Unlock()
		if err := client.close(); err != nil {
			// Only log unexpected close errors
			if !isExpectedCloseError(err) {
				log.Printf("Failed to close websocket connection: %v", err)
			}
		}
	}()

	// Send initial service data using shared serviceinfo package
	services, err := serviceinfo.GetServiceInfo(s.projectDir)
	if err != nil {
		log.Printf("Warning: Failed to get service info: %v", err)
		services = []*serviceinfo.ServiceInfo{} // Empty array on error
	}

	// Use the safe write method
	if err := clientWrapper.writeWebSocketJSON(map[string]interface{}{
		"type":     "services",
		"services": services,
	}); err != nil {
		log.Printf("Failed to send initial services: %v", err)
		return
	}

	// Start health monitoring
	monitor := newWSHealthMonitor(client)
	healthErrors := monitor.start()
	defer monitor.stop()

	// Keep connection alive and listen for client messages
	for {
		select {
		case <-s.stopChan:
			return
		case <-healthErrors:
			// Health monitor detected a problem, close connection
			return
		default:
			if err := readMessage(client); err != nil {
				return
			}
		}
	}
}

// handleFallback provides a simple HTML page when static files aren't available.
func (s *Server) handleFallback(w http.ResponseWriter, r *http.Request) {
	reg := registry.GetRegistry(s.projectDir)
	services := reg.ListAll()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>AZD App Dashboard</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; max-width: 1200px; margin: 40px auto; padding: 20px; }
        h1 { color: #0078d4; }
        .service { background: #f5f5f5; padding: 15px; margin: 10px 0; border-radius: 8px; }
        .status { display: inline-block; width: 12px; height: 12px; border-radius: 50%%; margin-right: 8px; }
        .ready { background: #107c10; }
        .starting { background: #ffb900; }
        .error { background: #d13438; }
        a { color: #0078d4; text-decoration: none; }
        a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <h1>🚀 AZD App Dashboard</h1>
    <p>Running Services in Current Project</p>
`)

	if len(services) == 0 {
		fmt.Fprintf(w, `<p>No services are currently running.</p>`)
	} else {
		for _, svc := range services {
			statusClass := "starting"
			if svc.Status == "ready" {
				statusClass = "ready"
			} else if svc.Status == "error" {
				statusClass = "error"
			}

			// Escape all user-controllable values to prevent XSS
			escapedName := html.EscapeString(svc.Name)
			escapedURL := html.EscapeString(svc.URL)
			escapedFramework := html.EscapeString(svc.Framework)
			escapedLanguage := html.EscapeString(svc.Language)
			escapedStatus := html.EscapeString(svc.Status)
			escapedHealth := "-" // Health is computed dynamically via health checks

			fmt.Fprintf(w, `
    <div class="service">
        <h3><span class="status %s"></span>%s</h3>
        <p><strong>URL:</strong> <a href="%s" target="_blank">%s</a></p>
        <p><strong>Framework:</strong> %s (%s)</p>
        <p><strong>Status:</strong> %s | <strong>Health:</strong> %s</p>
        <p><strong>Started:</strong> %s</p>
    </div>
`, statusClass, escapedName, escapedURL, escapedURL, escapedFramework, escapedLanguage, escapedStatus, escapedHealth, svc.StartTime.Format(time.RFC822))
		}
	}

	fmt.Fprintf(w, `
    <hr>
    <p style="color: #666; font-size: 14px;">
        <a href="/api/services">View JSON</a> | 
        <a href="/api/services/all">All Projects (JSON)</a>
    </p>
</body>
</html>`)
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

	// Check for existing persisted port first to maintain URL consistency across runs
	var preferredPort int
	if existingPort, exists := portMgr.GetAssignment(constants.DashboardServiceName); exists && existingPort > 0 {
		// Use persisted port as preferred - same workspace gets same dashboard URL
		preferredPort = existingPort
	} else {
		// First run: generate random port in dashboard range (40000-49999)
		// This range is typically used for ephemeral/dynamic ports to avoid common conflicts
		nBig, err := rand.Int(rand.Reader, big.NewInt(10000))
		if err != nil {
			return "", fmt.Errorf("failed to generate random port: %w", err)
		}
		preferredPort = 40000 + int(nBig.Int64())
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
		Handler:           s.mux,
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
		if port, err := s.retryWithAlternativePort(portMgr); err == nil {
			return fmt.Sprintf("http://localhost:%d", port), nil
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

// getOrCreateConfigClient returns the cached config client, creating it lazily if needed.
func (s *Server) getOrCreateConfigClient() azdconfig.ConfigClient {
	if s.configClient != nil {
		return s.configClient
	}

	client, err := azdconfig.NewClient(context.Background())
	if err != nil {
		slog.Debug("failed to create azdconfig client, using in-memory fallback", "error", err)
		s.configClient = azdconfig.NewInMemoryClient()
		return s.configClient
	}

	s.configClient = client
	return client
}

// registerPortInConfig stores the dashboard port in azdconfig for discovery by other commands.
func (s *Server) registerPortInConfig(port int) {
	client := s.getOrCreateConfigClient()

	projectHash := azdconfig.ProjectHash(s.projectDir)
	if err := client.SetDashboardPort(projectHash, port); err != nil {
		slog.Debug("failed to register dashboard port in config", "error", err)
	} else {
		slog.Debug("registered dashboard port in config", "port", port, "projectHash", projectHash)
	}
}

// clearPortFromConfig removes the dashboard port from azdconfig.
func (s *Server) clearPortFromConfig() {
	client := s.getOrCreateConfigClient()

	projectHash := azdconfig.ProjectHash(s.projectDir)
	if err := client.ClearDashboardPort(projectHash); err != nil {
		slog.Debug("failed to clear dashboard port from config", "error", err)
	} else {
		slog.Debug("cleared dashboard port from config", "projectHash", projectHash)
	}
}

// retryWithAlternativePort attempts to start the server on an alternative port.
func (s *Server) retryWithAlternativePort(portMgr *portmanager.PortManager) (int, error) {
	// Release the failed port assignment
	if err := portMgr.ReleasePort(constants.DashboardServiceName); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to release port: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "Searching for an available dashboard port...\n")

	// Try to find a new port in the higher range with randomization
	for attempt := 0; attempt < 15; attempt++ {
		var preferredPort int
		if attempt < 5 {
			// First 5 attempts: random ports in 40000-49999 range
			nBig, err := rand.Int(rand.Reader, big.NewInt(10000))
			if err != nil {
				continue
			}
			preferredPort = 40000 + int(nBig.Int64())
		} else {
			// After 5 failed random attempts, try sequential ports
			preferredPort = 40000 + (attempt * 100)
		}

		// Use port reservation to prevent TOCTOU race
		reservation, err := portMgr.FindAndReservePort(constants.DashboardServiceName, preferredPort)
		if err != nil {
			continue
		}

		port := reservation.Port
		s.port = port
		s.server = &http.Server{
			Addr:              fmt.Sprintf("127.0.0.1:%d", port),
			Handler:           s.mux,
			ReadHeaderTimeout: 10 * time.Second,
		}

		// Release reservation just before binding - this is the atomic handoff
		_ = reservation.Release()

		errChan := make(chan error, 1)
		go func() {
			if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("Dashboard server error on alternative port: %v", err)
				errChan <- err
			}
		}()

		// Monitor for server errors in background
		go func(port int) {
			select {
			case err := <-errChan:
				if strings.Contains(err.Error(), "bind") || strings.Contains(err.Error(), "address already in use") {
					log.Printf("Dashboard server encountered port conflict on port %d: %v", port, err)
				} else {
					log.Printf("Dashboard server encountered error after startup on port %d: %v", port, err)
				}
			case <-s.stopChan:
				return
			}
		}(port)

		time.Sleep(100 * time.Millisecond)

		select {
		case <-errChan:
			// This port also failed, try next
			if err := portMgr.ReleasePort(constants.DashboardServiceName); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to release port: %v\n", err)
			}
			continue
		default:
			// Successfully started - register the new port in azdconfig
			s.registerPortInConfig(port)
			fmt.Fprintf(os.Stderr, "✓ Dashboard started on alternative port %d\n\n", port)
			return port, nil
		}
	}

	return 0, fmt.Errorf("failed to find available port for dashboard after 15 attempts")
}

// BroadcastUpdate sends service updates to all connected WebSocket clients.
func (s *Server) BroadcastUpdate(services []*registry.ServiceRegistryEntry) {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	message := map[string]interface{}{
		"type":     "services",
		"services": services,
	}

	for client := range s.clients {
		if err := client.writeWebSocketJSON(message); err != nil {
			if !isExpectedCloseError(err) {
				log.Printf("WebSocket send error: %v", err)
			}
		}
	}
}

// BroadcastServiceUpdate fetches fresh service info and broadcasts to all connected clients.
// This is called when environment variables are updated (e.g., after azd provision).
func (s *Server) BroadcastServiceUpdate(projectDir string) error {
	// Fetch fresh service info with updated environment variables
	services, err := serviceinfo.GetServiceInfo(projectDir)
	if err != nil {
		return fmt.Errorf("failed to get service info: %w", err)
	}

	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	message := map[string]interface{}{
		"type":     "services",
		"services": services,
	}

	for client := range s.clients {
		if err := client.writeWebSocketJSON(message); err != nil {
			if !isExpectedCloseError(err) {
				log.Printf("WebSocket send error: %v", err)
			}
		}
	}

	return nil
}

// handleGetLogs returns recent logs for services.
func (s *Server) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	serviceName := r.URL.Query().Get("service")
	tailStr := r.URL.Query().Get("tail")

	// Default to 500 lines with bounds checking
	tail := 500
	if tailStr != "" {
		if n, err := fmt.Sscanf(tailStr, "%d", &tail); err != nil || n != 1 {
			tail = 500
		}
	}
	// Enforce reasonable limits to prevent memory exhaustion
	if tail <= 0 {
		tail = 500
	} else if tail > 10000 {
		tail = 10000 // Maximum 10k lines
	}

	logManager := service.GetLogManager(s.projectDir)

	var logs []service.LogEntry
	if serviceName != "" {
		// Get logs from specific service
		buffer, exists := logManager.GetBuffer(serviceName)
		if !exists {
			http.Error(w, fmt.Sprintf("Service '%s' not found", serviceName), http.StatusNotFound)
			return
		}
		logs = buffer.GetRecent(tail)
	} else {
		// Get logs from all services
		logs = logManager.GetAllLogs(tail)
	}

	if err := writeJSON(w, logs); err != nil {
		log.Printf("Failed to write logs JSON: %v", err)
	}
}

// handleLogStream streams logs via WebSocket.
func (s *Server) handleLogStream(w http.ResponseWriter, r *http.Request) {
	serviceName := r.URL.Query().Get("service")

	// Upgrade connection to WebSocket
	rawConn, err := acceptWebSocket(w, r)
	if err != nil {
		if err != http.ErrAbortHandler {
			log.Printf("WebSocket upgrade failed: %v", err)
		}
		return
	}
	// Wrap connection with mutex for safe concurrent writes
	client := newWSClient(rawConn)
	conn := &clientConn{client: client}
	defer client.close()

	logManager := service.GetLogManager(s.projectDir)

	// Create subscriptions for log streams
	subscriptions := make(map[string]chan service.LogEntry)

	if serviceName != "" {
		// Subscribe to specific service
		buffer, exists := logManager.GetBuffer(serviceName)
		if !exists {
			if err := conn.writeWebSocketJSON(map[string]string{"error": fmt.Sprintf("Service '%s' not found", serviceName)}); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to write error to websocket: %v\n", err)
			}
			return
		}
		subscriptions[serviceName] = buffer.Subscribe()
	} else {
		// Subscribe to all services
		for name, buffer := range logManager.GetAllBuffers() {
			subscriptions[name] = buffer.Subscribe()
		}
	}

	// Cleanup function
	defer func() {
		for svcName, ch := range subscriptions {
			if buffer, exists := logManager.GetBuffer(svcName); exists {
				buffer.Unsubscribe(ch)
			}
		}
	}()

	// Merge all subscription channels
	mergedChan := make(chan service.LogEntry, 100)
	stopMerge := make(chan struct{})
	var wg sync.WaitGroup

	for _, ch := range subscriptions {
		wg.Add(1)
		go func(ch chan service.LogEntry) {
			defer wg.Done()
			for {
				select {
				case entry, ok := <-ch:
					if !ok {
						return
					}
					select {
					case mergedChan <- entry:
					case <-stopMerge:
						return
					}
				case <-stopMerge:
					return
				}
			}
		}(ch)
	}

	// Close merged channel when all goroutines finish
	go func() {
		wg.Wait()
		close(mergedChan)
	}()

	// Stream logs to WebSocket
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case entry, ok := <-mergedChan:
				if !ok {
					return
				}
				if err := conn.writeWebSocketJSON(entry); err != nil {
					// Only log unexpected errors - client disconnects are normal
					if !isExpectedCloseError(err) {
						log.Printf("WebSocket write error: %v", err)
					}
					return
				}
			case <-s.stopChan:
				return
			}
		}
	}()

	// Keep connection alive until client disconnects or server stops
	<-done
	close(stopMerge)
	wg.Wait()
}

// handleStartService handles POST /api/services/start to start a service or all services.
func (s *Server) handleStartService(w http.ResponseWriter, r *http.Request) {
	newServiceOperationHandler(s, opStart).Handle(w, r)
}

// handleStopService handles POST /api/services/stop to stop a service or all services.
func (s *Server) handleStopService(w http.ResponseWriter, r *http.Request) {
	newServiceOperationHandler(s, opStop).Handle(w, r)
}

// handleRestartService handles POST /api/services/restart to restart a service or all services.
func (s *Server) handleRestartService(w http.ResponseWriter, r *http.Request) {
	newServiceOperationHandler(s, opRestart).Handle(w, r)
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

	close(s.stopChan)

	// Clear dashboard port from azdconfig so other commands know it's not running
	s.clearPortFromConfig()

	// Close the config client if it was created
	if s.configClient != nil {
		s.configClient.Close()
		s.configClient = nil
	}

	if s.server != nil {
		return s.server.Close()
	}
	return nil
}
