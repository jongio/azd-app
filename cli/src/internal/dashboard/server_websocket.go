package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/coder/websocket"
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
	conn, err := acceptWebSocket(w, r, s.rateLimiter)
	if err != nil {
		if err != http.ErrAbortHandler {
			log.Printf("WebSocket upgrade error: %v", err)
		}
		return
	}

	// Use request context to properly handle client disconnection
	client := newWSClientWithContext(r.Context(), conn)
	clientWrapper := &clientConn{client: client}
	clientIP := getClientIP(r)

	// Register client IMMEDIATELY to ensure rate limiter cleanup happens
	s.clientsMu.Lock()
	s.clients[clientWrapper] = true
	s.clientsMu.Unlock()

	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, clientWrapper)
		s.clientsMu.Unlock()
		if closeErr := client.closeWithRateLimit(clientIP, s.rateLimiter); closeErr != nil {
			// Only log unexpected close errors
			if !isExpectedCloseError(closeErr) {
				log.Printf("Failed to close websocket connection: %v", closeErr)
			}
		}
	}()

	// Send initial service data
	services, err := serviceinfo.GetServiceInfo(s.projectDir)
	if err != nil {
		log.Printf("Warning: Failed to get service info: %v", err)
		services = []*serviceinfo.ServiceInfo{} // Empty array on error
	}

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

	// Read messages in separate goroutine to avoid blocking
	readDone := make(chan error, 1)
	go func() {
		for {
			if err := readMessage(client); err != nil {
				readDone <- err
				return
			}
		}
	}()

	// Keep connection alive and listen for client messages
	select {
	case <-s.stopChan:
		return
	case <-healthErrors:
		// Health monitor detected a problem, close connection
		return
	case <-readDone:
		// Client disconnected or read error
		return
	}
}

// BroadcastUpdate sends service updates to all connected WebSocket clients.
// Broadcasts asynchronously with goroutine limiting to prevent resource exhaustion.
func (s *Server) BroadcastUpdate(services []*registry.ServiceRegistryEntry) {
	// Copy client list to avoid holding lock during writes
	s.clientsMu.RLock()
	clients := make([]*clientConn, 0, len(s.clients))
	for client := range s.clients {
		clients = append(clients, client)
	}
	s.clientsMu.RUnlock()

	message := map[string]interface{}{
		"type":     "services",
		"services": services,
	}

	// Marshal once before broadcast to avoid repeated CPU work
	jsonBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal broadcast message: %v", err)
		return
	}

	// Limit concurrent broadcast goroutines to prevent resource exhaustion
	sem := make(chan struct{}, service.WebSocketMaxConcurrentBroadcasts)
	var wg sync.WaitGroup

	for _, clientConn := range clients {
		// Capture client pointer before spawning goroutine to prevent race
		client := clientConn.client
		if client == nil {
			continue
		}

		wg.Add(1)
		go func(wsClient *wsClient, data []byte) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Write pre-marshaled JSON
			wsClient.writeMu.Lock()
			ctx, cancel := context.WithTimeout(wsClient.ctx, service.DefaultWebSocketWriteTimeout)
			err := wsClient.conn.Write(ctx, websocket.MessageText, data)
			cancel()
			wsClient.writeMu.Unlock()

			if err != nil {
				if !isExpectedCloseError(err) {
					log.Printf("WebSocket send error: %v", err)
					// Track consecutive write failures for backpressure
					failures := wsClient.recordWriteFailure()
					if failures >= service.WebSocketMaxWriteFailures {
						log.Printf("WebSocket client exceeded max write failures (%d), will disconnect", failures)
						// Signal to close by canceling context - main handler will clean up
						wsClient.cancel()
					}
				}
			}
		}(client, jsonBytes)
	}

	// Wait for all broadcasts to complete before returning
	wg.Wait()
}

// BroadcastServiceUpdate fetches fresh service info and broadcasts to all connected clients.
// This is called when environment variables are updated (e.g., after azd provision).
// Broadcasts asynchronously with goroutine limiting to prevent resource exhaustion.
func (s *Server) BroadcastServiceUpdate(projectDir string) error {
	// Fetch fresh service info with updated environment variables
	services, err := serviceinfo.GetServiceInfo(projectDir)
	if err != nil {
		return fmt.Errorf("failed to get service info: %w", err)
	}

	// Copy client list to avoid holding lock during writes
	s.clientsMu.RLock()
	clients := make([]*clientConn, 0, len(s.clients))
	for client := range s.clients {
		clients = append(clients, client)
	}
	s.clientsMu.RUnlock()

	message := map[string]interface{}{
		"type":     "services",
		"services": services,
	}

	// Marshal once before broadcast to avoid repeated CPU work
	jsonBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal broadcast message: %v", err)
		return fmt.Errorf("failed to marshal broadcast message: %w", err)
	}

	// Limit concurrent broadcast goroutines to prevent resource exhaustion
	sem := make(chan struct{}, service.WebSocketMaxConcurrentBroadcasts)
	var wg sync.WaitGroup

	for _, clientConn := range clients {
		// Capture client pointer before spawning goroutine to prevent race
		client := clientConn.client
		if client == nil {
			continue
		}

		wg.Add(1)
		go func(wsClient *wsClient, data []byte) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Write pre-marshaled JSON
			wsClient.writeMu.Lock()
			ctx, cancel := context.WithTimeout(wsClient.ctx, service.DefaultWebSocketWriteTimeout)
			err := wsClient.conn.Write(ctx, websocket.MessageText, data)
			cancel()
			wsClient.writeMu.Unlock()

			if err != nil {
				if !isExpectedCloseError(err) {
					log.Printf("WebSocket send error: %v", err)
					// Track consecutive write failures for backpressure
					failures := wsClient.recordWriteFailure()
					if failures >= service.WebSocketMaxWriteFailures {
						log.Printf("WebSocket client exceeded max write failures (%d), will disconnect", failures)
						// Signal to close by canceling context - main handler will clean up
						wsClient.cancel()
					}
				}
			}
		}(client, jsonBytes)
	}

	// Wait for all broadcasts to complete before returning
	wg.Wait()
	return nil
}

// handleLogStream streams logs via WebSocket.
func (s *Server) handleLogStream(w http.ResponseWriter, r *http.Request) {
	serviceName := r.URL.Query().Get("service")

	// Upgrade connection to WebSocket
	rawConn, err := acceptWebSocket(w, r, s.rateLimiter)
	if err != nil {
		if err != http.ErrAbortHandler {
			log.Printf("WebSocket upgrade failed: %v", err)
		}
		return
	}
	// Wrap connection with mutex for safe concurrent writes
	// Use request context to properly handle client disconnection
	client := newWSClientWithContext(r.Context(), rawConn)
	conn := &clientConn{client: client}
	clientIP := getClientIP(r)
	defer func() {
		if err := client.closeWithRateLimit(clientIP, s.rateLimiter); err != nil {
			if !isExpectedCloseError(err) {
				fmt.Fprintf(os.Stderr, "Warning: failed to close websocket connection: %v\n", err)
			}
		}
	}()

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

	// Merge all subscription channels with backpressure handling
	// Use constant for buffer size
	mergedChan := make(chan service.LogEntry, service.WebSocketLogChannelBuffer)
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
					// Try to send with timeout to prevent blocking on slow consumers
					// CRITICAL: Always include stopMerge in select to prevent goroutine leaks
					select {
					case mergedChan <- entry:
						// Successfully sent
					case <-stopMerge:
						return
					default:
						// Channel full, try with timeout
						timer := time.NewTimer(service.WebSocketSlowConsumerTimeout)
						select {
						case mergedChan <- entry:
							// Successfully sent after brief wait
							timer.Stop()
						case <-timer.C:
							// Drop log entry if consumer is too slow
							log.Printf("Warning: Dropped log entry due to slow consumer")
						case <-stopMerge:
							timer.Stop()
							return
						}
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

	// Stream logs to WebSocket with batching to improve throughput
	done := make(chan struct{})
	go func() {
		defer close(done)
		batch := make([]service.LogEntry, 0, 100)       // Batch up to 100 entries
		ticker := time.NewTicker(50 * time.Millisecond) // Flush every 50ms
		defer ticker.Stop()

		flush := func() error {
			if len(batch) == 0 {
				return nil
			}
			// Send as array if batched, single entry if just one
			var payload interface{}
			if len(batch) == 1 {
				payload = batch[0]
			} else {
				payload = batch
			}
			if err := conn.writeWebSocketJSON(payload); err != nil {
				return err
			}
			batch = batch[:0] // Clear batch
			return nil
		}

		for {
			select {
			case entry, ok := <-mergedChan:
				if !ok {
					// Flush remaining batch before closing
					if err := flush(); err != nil && !isExpectedCloseError(err) {
						log.Printf("WebSocket write error: %v", err)
					}
					return
				}
				batch = append(batch, entry)
				// Flush if batch is full
				if len(batch) >= 100 {
					if err := flush(); err != nil {
						if !isExpectedCloseError(err) {
							log.Printf("WebSocket write error: %v", err)
						}
						return
					}
				}
			case <-ticker.C:
				// Periodic flush to ensure low latency
				if err := flush(); err != nil {
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
	// CRITICAL: Close stopMerge FIRST to signal goroutines, THEN wait for them
	// Otherwise goroutines may be blocked and won't see the stop signal
	<-done
	close(stopMerge) // Signal all merger goroutines to stop
	wg.Wait()        // Wait for all merger goroutines to finish cleanup
}
