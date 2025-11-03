package dashboard

import (
	"crypto/rand"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/portmanager"
	"github.com/jongio/azd-app/cli/src/internal/registry"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"

	"github.com/gorilla/websocket"
)

//go:embed dist
var staticFiles embed.FS

// clientConn wraps a websocket connection with a write mutex for safe concurrent writes.
type clientConn struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
}

var (
	servers   = make(map[string]*Server) // Key: normalized project directory path
	serversMu sync.Mutex
	upgrader  = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for local development
		},
	}
)

// Server represents the dashboard HTTP server.
type Server struct {
	port       int
	mux        *http.ServeMux
	server     *http.Server
	projectDir string
	clients    map[*clientConn]bool
	clientsMu  sync.RWMutex
	stopChan   chan struct{}
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
	s.mux.HandleFunc("/api/project", s.handleGetProject)
	s.mux.HandleFunc("/api/services", s.handleGetServices)
	s.mux.HandleFunc("/api/logs", s.handleGetLogs)
	s.mux.HandleFunc("/api/logs/stream", s.handleLogStream)
	s.mux.HandleFunc("/api/ws", s.handleWebSocket)

	// Serve static files
	fileServer := http.FileServer(http.FS(distFS))
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fileServer.ServeHTTP(w, r)
	})
}

// handleGetServices returns services for the current project.
func (s *Server) handleGetServices(w http.ResponseWriter, r *http.Request) {
	// Use shared serviceinfo package to get merged service data
	services, err := serviceinfo.GetServiceInfo(s.projectDir)
	if err != nil {
		log.Printf("Warning: Failed to get service info: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get service info: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(services); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleGetProject returns project metadata from azure.yaml.
func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	azureYaml, err := service.ParseAzureYaml(s.projectDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse azure.yaml: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"name": azureYaml.Name,
		"dir":  s.projectDir,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleWebSocket handles WebSocket connections for live updates.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Wrap connection with mutex for safe concurrent writes
	client := &clientConn{conn: conn}

	s.clientsMu.Lock()
	s.clients[client] = true
	s.clientsMu.Unlock()

	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, client)
		s.clientsMu.Unlock()
		if err := conn.Close(); err != nil {
			log.Printf("Failed to close websocket connection: %v", err)
		}
	}()

	// Send initial service data using shared serviceinfo package
	services, err := serviceinfo.GetServiceInfo(s.projectDir)
	if err != nil {
		log.Printf("Warning: Failed to get service info: %v", err)
		services = []*serviceinfo.ServiceInfo{} // Empty array on error
	}

	// Use the safe write method
	client.writeMu.Lock()
	err = conn.WriteJSON(map[string]interface{}{
		"type":     "services",
		"services": services,
	})
	client.writeMu.Unlock()
	if err != nil {
		return
	}

	// Keep connection alive and listen for client messages
	for {
		select {
		case <-s.stopChan:
			return
		default:
			_, _, err := conn.ReadMessage()
			if err != nil {
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

			fmt.Fprintf(w, `
    <div class="service">
        <h3><span class="status %s"></span>%s</h3>
        <p><strong>URL:</strong> <a href="%s" target="_blank">%s</a></p>
        <p><strong>Framework:</strong> %s (%s)</p>
        <p><strong>Status:</strong> %s | <strong>Health:</strong> %s</p>
        <p><strong>Started:</strong> %s</p>
    </div>
`, statusClass, svc.Name, svc.URL, svc.URL, svc.Framework, svc.Language, svc.Status, svc.Health, svc.StartTime.Format(time.RFC822))
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

// Start starts the dashboard server on an assigned port.
func (s *Server) Start() (string, error) {
	// Use port manager to get a persistent port for the dashboard
	portMgr := portmanager.GetPortManager(s.projectDir)

	// Use a random port in higher range (40000-49999) to avoid common conflicts
	// This range is typically used for ephemeral/dynamic ports
	nBig, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "", fmt.Errorf("failed to generate random port: %w", err)
	}
	preferredPort := 40000 + int(nBig.Int64())

	// Assign port for dashboard service (isExplicit=false, cleanStale=false to avoid prompts)
	port, err := portMgr.AssignPort("azd-app-dashboard", preferredPort, false, false)
	if err != nil {
		return "", fmt.Errorf("failed to assign port for dashboard: %w", err)
	}

	s.port = port
	s.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
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
			log.Printf("Dashboard server encountered error after startup: %v", err)
		case <-s.stopChan:
			return
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Check if there was an immediate error (like port already in use)
	select {
	case err := <-errChan:
		// Port binding failed, try to find an alternative port
		if port, err := s.retryWithAlternativePort(portMgr); err == nil {
			return fmt.Sprintf("http://localhost:%d", port), nil
		}
		return "", fmt.Errorf("dashboard server failed to start: %w", err)
	default:
		// Server started successfully
	}

	url := fmt.Sprintf("http://localhost:%d", port)
	return url, nil
}

// retryWithAlternativePort attempts to start the server on an alternative port.
func (s *Server) retryWithAlternativePort(portMgr *portmanager.PortManager) (int, error) {
	// Release the failed port assignment
	if err := portMgr.ReleasePort("azd-app-dashboard"); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to release port: %v\n", err)
	}

	// Try to find a new port in the higher range with randomization
	for attempt := 0; attempt < 10; attempt++ {
		// Random port in 40000-49999 range
		nBig, err := rand.Int(rand.Reader, big.NewInt(10000))
		if err != nil {
			continue
		}
		preferredPort := 40000 + int(nBig.Int64())
		port, err := portMgr.AssignPort("azd-app-dashboard", preferredPort, false, false)
		if err != nil {
			continue
		}

		s.port = port
		s.server = &http.Server{
			Addr:              fmt.Sprintf(":%d", port),
			Handler:           s.mux,
			ReadHeaderTimeout: 10 * time.Second,
		}

		errChan := make(chan error, 1)
		go func() {
			if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("Dashboard server error on alternative port: %v", err)
				errChan <- err
			}
		}()

		// Monitor for server errors in background
		go func() {
			select {
			case err := <-errChan:
				log.Printf("Dashboard server encountered error after startup on port %d: %v", port, err)
			case <-s.stopChan:
				return
			}
		}()

		time.Sleep(100 * time.Millisecond)

		select {
		case <-errChan:
			// This port also failed, try next
			if err := portMgr.ReleasePort("azd-app-dashboard"); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to release port: %v\n", err)
			}
			continue
		default:
			// Successfully started
			return port, nil
		}
	}

	return 0, fmt.Errorf("failed to find available port for dashboard after 10 attempts")
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
		client.writeMu.Lock()
		err := client.conn.WriteJSON(message)
		client.writeMu.Unlock()
		if err != nil {
			log.Printf("WebSocket send error: %v", err)
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
		client.writeMu.Lock()
		err := client.conn.WriteJSON(message)
		client.writeMu.Unlock()
		if err != nil {
			log.Printf("WebSocket send error: %v", err)
		}
	}

	return nil
}

// handleGetLogs returns recent logs for services.
func (s *Server) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	serviceName := r.URL.Query().Get("service")
	tailStr := r.URL.Query().Get("tail")

	// Default to 500 lines
	tail := 500
	if tailStr != "" {
		if n, err := fmt.Sscanf(tailStr, "%d", &tail); err != nil || n != 1 {
			tail = 500
		}
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

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(logs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleLogStream streams logs via WebSocket.
func (s *Server) handleLogStream(w http.ResponseWriter, r *http.Request) {
	serviceName := r.URL.Query().Get("service")

	// Upgrade connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	logManager := service.GetLogManager(s.projectDir)

	// Create subscriptions for log streams
	subscriptions := make(map[string]chan service.LogEntry)

	if serviceName != "" {
		// Subscribe to specific service
		buffer, exists := logManager.GetBuffer(serviceName)
		if !exists {
			if err := conn.WriteJSON(map[string]string{"error": fmt.Sprintf("Service '%s' not found", serviceName)}); err != nil {
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
	for _, ch := range subscriptions {
		go func(ch chan service.LogEntry) {
			for entry := range ch {
				mergedChan <- entry
			}
		}(ch)
	}

	// Stream logs to WebSocket
	done := make(chan struct{})
	go func() {
		for {
			select {
			case entry := <-mergedChan:
				if err := conn.WriteJSON(entry); err != nil {
					log.Printf("WebSocket write error: %v", err)
					close(done)
					return
				}
			case <-s.stopChan:
				close(done)
				return
			}
		}
	}()

	// Keep connection alive until client disconnects or server stops
	<-done
}

// Stop stops the dashboard server and releases its port assignment.
func (s *Server) Stop() error {
	close(s.stopChan)

	// Release port assignment
	portMgr := portmanager.GetPortManager(s.projectDir)
	if err := portMgr.ReleasePort("azd-app-dashboard"); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to release dashboard port: %v\n", err)
	}

	// Remove from servers map
	serversMu.Lock()
	_, key := normalizeProjectPath(s.projectDir)
	delete(servers, key)
	serversMu.Unlock()

	if s.server != nil {
		return s.server.Close()
	}
	return nil
}
