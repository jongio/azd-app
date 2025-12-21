package dashboard

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// TestServer_DoubleStop verifies that calling Stop() multiple times doesn't panic.
func TestServer_DoubleStop(t *testing.T) {
	tempDir := t.TempDir()
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte("name: test\nservices:\n  api:\n    language: python\n"), 0600); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	srv := GetServer(tempDir)
	url, err := srv.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	_ = url

	// First stop
	if err := srv.Stop(); err != nil {
		t.Errorf("first Stop() failed: %v", err)
	}

	// Second stop should not panic
	if err := srv.Stop(); err != nil {
		t.Errorf("second Stop() failed: %v", err)
	}

	// Third stop for good measure
	if err := srv.Stop(); err != nil {
		t.Errorf("third Stop() failed: %v", err)
	}
}

// TestServer_ConcurrentStops verifies that concurrent Stop() calls don't panic.
func TestServer_ConcurrentStops(t *testing.T) {
	tempDir := t.TempDir()
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte("name: test\nservices:\n  api:\n    language: python\n"), 0600); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	srv := GetServer(tempDir)
	url, err := srv.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	_ = url

	// Stop concurrently from multiple goroutines
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = srv.Stop()
		}()
	}

	wg.Wait()
}

// TestServer_BroadcastDuringShutdown verifies graceful handling of broadcasts during shutdown.
func TestServer_BroadcastDuringShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	tempDir := t.TempDir()
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte("name: test\nservices:\n  api:\n    language: python\n"), 0600); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	srv := GetServer(tempDir)
	url, err := srv.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Connect a client
	wsURL := strings.Replace(url, "http://", "ws://", 1) + "/api/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ws, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect WebSocket: %v", err)
	}
	defer ws.Close(websocket.StatusNormalClosure, "test complete")

	// Read initial message
	var initialMsg map[string]interface{}
	if err := wsjson.Read(ctx, ws, &initialMsg); err != nil {
		t.Fatalf("failed to read initial message: %v", err)
	}

	// Start broadcasting in background
	var wg sync.WaitGroup
	stopBroadcast := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stopBroadcast:
				return
			case <-ticker.C:
				// Broadcast should not panic even during shutdown
				_ = srv.BroadcastServiceUpdate(tempDir)
			}
		}
	}()

	// Let broadcasts run for a bit
	time.Sleep(100 * time.Millisecond)

	// Stop server while broadcasts are happening
	if err := srv.Stop(); err != nil {
		t.Errorf("Stop() during broadcasts failed: %v", err)
	}

	close(stopBroadcast)
	wg.Wait()
}

// TestServer_ClientDisconnectDuringBroadcast verifies graceful handling when client disconnects during broadcast.
func TestServer_ClientDisconnectDuringBroadcast(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	tempDir := t.TempDir()
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte("name: test\nservices:\n  api:\n    language: python\n"), 0600); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	srv := GetServer(tempDir)
	url, err := srv.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	// Connect multiple clients
	numClients := 5
	clients := make([]*websocket.Conn, numClients)
	wsURL := strings.Replace(url, "http://", "ws://", 1) + "/api/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for i := 0; i < numClients; i++ {
		ws, _, err := websocket.Dial(ctx, wsURL, nil)
		if err != nil {
			t.Fatalf("failed to connect client %d: %v", i, err)
		}
		clients[i] = ws

		// Read initial message
		var initialMsg map[string]interface{}
		if err := wsjson.Read(ctx, ws, &initialMsg); err != nil {
			t.Fatalf("client %d failed to read initial message: %v", i, err)
		}
	}

	// Start disconnecting clients randomly while broadcasting
	var wg sync.WaitGroup
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			time.Sleep(time.Duration(idx*20) * time.Millisecond)
			_ = clients[idx].Close(websocket.StatusNormalClosure, "test disconnect")
		}(i)
	}

	// Broadcast repeatedly
	for i := 0; i < 10; i++ {
		// This should not panic even though clients are disconnecting
		_ = srv.BroadcastServiceUpdate(tempDir)
		time.Sleep(10 * time.Millisecond)
	}

	wg.Wait()
}

// TestRateLimiter_ConnectionLimit verifies rate limiting prevents excessive connections.
func TestRateLimiter_ConnectionLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping rate limit test in short mode")
	}

	tempDir := t.TempDir()
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte("name: test\nservices:\n  api:\n    language: python\n"), 0600); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	srv := GetServer(tempDir)
	url, err := srv.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	wsURL := strings.Replace(url, "http://", "ws://", 1) + "/api/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Try to open more connections than the rate limit allows (maxPerIP = 100)
	maxConnections := 120
	successCount := 0
	failCount := 0
	connections := make([]*websocket.Conn, 0)

	for i := 0; i < maxConnections; i++ {
		ws, resp, err := websocket.Dial(ctx, wsURL, nil)
		if err != nil {
			failCount++
			if resp != nil && resp.StatusCode == 429 {
				// Expected rate limit error
				continue
			}
			t.Logf("Connection %d failed: %v (status: %v)", i, err, resp)
			continue
		}
		successCount++
		connections = append(connections, ws)

		// Read initial message to keep connection alive
		go func(w *websocket.Conn) {
			var msg map[string]interface{}
			_ = wsjson.Read(ctx, w, &msg)
		}(ws)
	}

	// Clean up connections
	for _, conn := range connections {
		_ = conn.Close(websocket.StatusNormalClosure, "test complete")
	}

	t.Logf("Successful connections: %d, Failed: %d", successCount, failCount)

	// We should have hit the rate limit
	if successCount >= maxConnections {
		t.Errorf("Rate limiter did not prevent excessive connections: %d/%d succeeded", successCount, maxConnections)
	}

	// Should have some successful connections (at least the rate limit)
	if successCount < 90 {
		t.Errorf("Too few connections succeeded: %d (expected >= 90)", successCount)
	}
}

// TestServer_ConcurrentBroadcasts verifies that concurrent broadcasts don't cause issues.
func TestServer_ConcurrentBroadcasts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	tempDir := t.TempDir()
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte("name: test\nservices:\n  api:\n    language: python\n"), 0600); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	srv := GetServer(tempDir)
	url, err := srv.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	// Connect multiple clients
	numClients := 3
	clients := make([]*websocket.Conn, numClients)
	wsURL := strings.Replace(url, "http://", "ws://", 1) + "/api/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for i := 0; i < numClients; i++ {
		ws, _, err := websocket.Dial(ctx, wsURL, nil)
		if err != nil {
			t.Fatalf("failed to connect client %d: %v", i, err)
		}
		defer ws.Close(websocket.StatusNormalClosure, "test complete")
		clients[i] = ws

		// Read initial message
		var initialMsg map[string]interface{}
		if err := wsjson.Read(ctx, ws, &initialMsg); err != nil {
			t.Fatalf("client %d failed to read initial message: %v", i, err)
		}

		// Start reading messages in background
		go func(idx int, w *websocket.Conn) {
			for {
				var msg map[string]interface{}
				if err := wsjson.Read(ctx, w, &msg); err != nil {
					return
				}
			}
		}(i, ws)
	}

	// Broadcast concurrently from multiple goroutines
	numBroadcasters := 5
	broadcastsPerGoroutine := 10
	var wg sync.WaitGroup

	for i := 0; i < numBroadcasters; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < broadcastsPerGoroutine; j++ {
				if err := srv.BroadcastServiceUpdate(tempDir); err != nil {
					t.Errorf("broadcaster %d broadcast %d failed: %v", idx, j, err)
				}
				time.Sleep(5 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	t.Logf("Successfully completed %d concurrent broadcasts to %d clients",
		numBroadcasters*broadcastsPerGoroutine, numClients)
}

// TestServer_SlowClient verifies that slow clients don't block other clients.
func TestServer_SlowClient(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow client test in short mode")
	}

	tempDir := t.TempDir()
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte("name: test\nservices:\n  api:\n    language: python\n"), 0600); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	srv := GetServer(tempDir)
	url, err := srv.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	wsURL := strings.Replace(url, "http://", "ws://", 1) + "/api/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Connect slow client (doesn't read messages)
	slowWS, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect slow client: %v", err)
	}
	defer slowWS.Close(websocket.StatusNormalClosure, "test complete")

	// Read initial message from slow client, then stop reading
	var initialMsg map[string]interface{}
	if readErr := wsjson.Read(ctx, slowWS, &initialMsg); readErr != nil {
		t.Fatalf("slow client failed to read initial message: %v", readErr)
	}
	// Slow client stops reading here

	// Connect fast client
	fastWS, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect fast client: %v", err)
	}
	defer fastWS.Close(websocket.StatusNormalClosure, "test complete")

	// Read initial message from fast client
	if err := wsjson.Read(ctx, fastWS, &initialMsg); err != nil {
		t.Fatalf("fast client failed to read initial message: %v", err)
	}

	// Start reading from fast client in background
	fastClientMessages := 0
	fastClientDone := make(chan struct{})
	go func() {
		defer close(fastClientDone)
		for {
			var msg map[string]interface{}
			if err := wsjson.Read(ctx, fastWS, &msg); err != nil {
				return
			}
			fastClientMessages++
		}
	}()

	// Broadcast multiple messages
	numBroadcasts := 10
	for i := 0; i < numBroadcasts; i++ {
		if err := srv.BroadcastServiceUpdate(tempDir); err != nil {
			t.Errorf("broadcast %d failed: %v", i, err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Give fast client time to receive messages
	time.Sleep(500 * time.Millisecond)

	// Fast client should have received most/all broadcasts despite slow client
	if fastClientMessages < numBroadcasts/2 {
		t.Errorf("Fast client only received %d/%d messages, slow client may be blocking",
			fastClientMessages, numBroadcasts)
	}

	t.Logf("Fast client received %d/%d broadcasts (slow client didn't block)",
		fastClientMessages, numBroadcasts)
}

// TestConnectionRateLimiter_Cleanup verifies that stale entries are cleaned up.
func TestConnectionRateLimiter_Cleanup(t *testing.T) {
	rl := &connectionRateLimiter{
		connections: make(map[string]*connectionTracker),
		maxPerIP:    10,
		stopCleanup: make(chan struct{}),
	}

	// Add some old entries
	rl.connections["192.168.1.1"] = &connectionTracker{
		count:      0,
		lastAccess: time.Now().Add(-10 * time.Minute),
	}
	rl.connections["192.168.1.2"] = &connectionTracker{
		count:      1,
		lastAccess: time.Now(),
	}
	rl.connections["192.168.1.3"] = &connectionTracker{
		count:      0,
		lastAccess: time.Now().Add(-1 * time.Minute),
	}

	// Start cleanup with fast ticker for testing
	rl.cleanupTick = time.NewTicker(100 * time.Millisecond)
	go rl.cleanup()
	defer close(rl.stopCleanup)

	// Wait for cleanup to run
	time.Sleep(200 * time.Millisecond)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Old entry with no connections should be removed
	if _, exists := rl.connections["192.168.1.1"]; exists {
		t.Error("Stale entry was not cleaned up")
	}

	// Active entry should remain
	if _, exists := rl.connections["192.168.1.2"]; !exists {
		t.Error("Active entry was incorrectly cleaned up")
	}

	// Recent entry with no connections should remain (not old enough)
	if _, exists := rl.connections["192.168.1.3"]; !exists {
		t.Error("Recent entry was incorrectly cleaned up")
	}
}

// TestGetClientIP verifies IP extraction from requests.
func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name          string
		remoteAddr    string
		xForwardedFor string
		xRealIP       string
		expectedIP    string
	}{
		{
			name:       "RemoteAddr only",
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "192.168.1.1",
		},
		{
			name:          "X-Forwarded-For ignored (security)",
			remoteAddr:    "10.0.0.1:12345",
			xForwardedFor: "203.0.113.1",
			expectedIP:    "10.0.0.1", // Now uses RemoteAddr for security
		},
		{
			name:          "X-Forwarded-For multiple ignored (security)",
			remoteAddr:    "10.0.0.1:12345",
			xForwardedFor: "203.0.113.1, 198.51.100.1, 192.0.2.1",
			expectedIP:    "10.0.0.1", // Now uses RemoteAddr for security
		},
		{
			name:       "X-Real-IP ignored (security)",
			remoteAddr: "10.0.0.1:12345",
			xRealIP:    "203.0.113.2",
			expectedIP: "10.0.0.1", // Now uses RemoteAddr for security
		},
		{
			name:          "Proxy headers ignored for localhost security",
			remoteAddr:    "10.0.0.1:12345",
			xForwardedFor: "203.0.113.1",
			xRealIP:       "203.0.113.2",
			expectedIP:    "10.0.0.1", // Now uses RemoteAddr for security
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{
				RemoteAddr: tt.remoteAddr,
				Header:     make(map[string][]string),
			}
			if tt.xForwardedFor != "" {
				r.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				r.Header.Set("X-Real-IP", tt.xRealIP)
			}

			ip := getClientIP(r)
			if ip != tt.expectedIP {
				t.Errorf("getClientIP() = %q, want %q", ip, tt.expectedIP)
			}
		})
	}
}

// BenchmarkBroadcast measures broadcast performance with multiple clients.
func BenchmarkBroadcast(b *testing.B) {
	tempDir := b.TempDir()
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte("name: test\nservices:\n  api:\n    language: python\n"), 0600); err != nil {
		b.Fatalf("failed to create azure.yaml: %v", err)
	}

	srv := GetServer(tempDir)
	url, err := srv.Start()
	if err != nil {
		b.Fatalf("failed to start server: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	// Connect multiple clients
	numClients := 10
	clients := make([]*websocket.Conn, numClients)
	wsURL := strings.Replace(url, "http://", "ws://", 1) + "/api/ws"
	ctx := context.Background()

	for i := 0; i < numClients; i++ {
		ws, _, err := websocket.Dial(ctx, wsURL, nil)
		if err != nil {
			b.Fatalf("failed to connect client %d: %v", i, err)
		}
		defer ws.Close(websocket.StatusNormalClosure, "benchmark complete")
		clients[i] = ws

		// Read initial message
		var initialMsg map[string]interface{}
		if err := wsjson.Read(ctx, ws, &initialMsg); err != nil {
			b.Fatalf("client %d failed to read initial message: %v", i, err)
		}

		// Start reading in background
		go func(w *websocket.Conn) {
			for {
				var msg map[string]interface{}
				if err := wsjson.Read(ctx, w, &msg); err != nil {
					return
				}
			}
		}(ws)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := srv.BroadcastServiceUpdate(tempDir); err != nil {
			b.Errorf("broadcast failed: %v", err)
		}
	}
}

func init() {
	// Suppress log output during tests
	if testing.Testing() {
		fmt.Fprintf(os.Stderr, "Running WebSocket concurrency tests\n")
	}
}
