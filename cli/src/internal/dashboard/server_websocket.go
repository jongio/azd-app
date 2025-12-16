package dashboard

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/jongio/azd-app/cli/src/internal/registry"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
)

// clientConn wraps a websocket connection with a write mutex for safe concurrent writes.
type clientConn struct {
	client *wsClient // Uses github.com/coder/websocket
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
