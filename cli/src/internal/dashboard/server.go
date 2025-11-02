package dashboard

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/portmanager"
	"github.com/jongio/azd-app/cli/src/internal/registry"
	"github.com/jongio/azd-app/cli/src/internal/service"

	"github.com/gorilla/websocket"
)

//go:embed dist
var staticFiles embed.FS

var (
	servers   = make(map[string]*Server) // Key: absolute project directory path
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
	clients    map[*websocket.Conn]bool
	clientsMu  sync.RWMutex
	stopChan   chan struct{}
}

// GetServer returns the dashboard server instance for the specified project.
// Creates a new instance if one doesn't exist for this project.
func GetServer(projectDir string) *Server {
	serversMu.Lock()
	defer serversMu.Unlock()

	// Get absolute path for consistent key
	absPath, err := filepath.Abs(projectDir)
	if err != nil {
		absPath = projectDir
	}

	// Return existing server if already created
	if srv, exists := servers[absPath]; exists {
		return srv
	}

	// Create new server instance for this project
	srv := &Server{
		port:       0, // Will be assigned by port manager
		mux:        http.NewServeMux(),
		projectDir: absPath,
		clients:    make(map[*websocket.Conn]bool),
		stopChan:   make(chan struct{}),
	}
	srv.setupRoutes()
	servers[absPath] = srv

	return srv
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
	s.mux.HandleFunc("/api/ws", s.handleWebSocket)

	// Serve static files
	fileServer := http.FileServer(http.FS(distFS))
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fileServer.ServeHTTP(w, r)
	})
}

// handleGetServices returns services for the current project.
func (s *Server) handleGetServices(w http.ResponseWriter, r *http.Request) {
	reg := registry.GetRegistry(s.projectDir)
	services := reg.ListAll()

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

	s.clientsMu.Lock()
	s.clients[conn] = true
	s.clientsMu.Unlock()

	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, conn)
		s.clientsMu.Unlock()
		conn.Close()
	}()

	// Send initial service data
	reg := registry.GetRegistry(s.projectDir)
	services := reg.ListAll()
	if err := conn.WriteJSON(map[string]interface{}{
		"type":     "services",
		"services": services,
	}); err != nil {
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

	// Assign port for dashboard service (isExplicit=false, cleanStale=true)
	port, err := portMgr.AssignPort("azd-app-dashboard", 3100, false, true)
	if err != nil {
		return "", fmt.Errorf("failed to assign port for dashboard: %w", err)
	}

	s.port = port
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: s.mux,
	}

	// Start server in background
	errChan := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Dashboard server error: %v", err)
			errChan <- err
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Check if there was an immediate error
	select {
	case err := <-errChan:
		return "", fmt.Errorf("dashboard server failed to start: %w", err)
	default:
		// Server started successfully
	}
	time.Sleep(100 * time.Millisecond)

	url := fmt.Sprintf("http://localhost:%d", port)
	return url, nil
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
		if err := client.WriteJSON(message); err != nil {
			log.Printf("WebSocket send error: %v", err)
		}
	}
}

// Stop stops the dashboard server and releases its port assignment.
func (s *Server) Stop() error {
	close(s.stopChan)

	// Release port assignment
	portMgr := portmanager.GetPortManager(s.projectDir)
	portMgr.ReleasePort("azd-app-dashboard")

	// Remove from servers map
	serversMu.Lock()
	absPath, _ := filepath.Abs(s.projectDir)
	delete(servers, absPath)
	serversMu.Unlock()

	if s.server != nil {
		return s.server.Close()
	}
	return nil
}
