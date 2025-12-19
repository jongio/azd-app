package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// isExpectedCloseError returns true if the error is an expected WebSocket closure.
// These occur when clients disconnect (tab closed, page refresh, navigation).
func isExpectedCloseError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Check for normal closure status
	if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
		return true
	}
	// Check for common expected closure patterns
	expectedPatterns := []string{
		"connection was aborted",
		"wsasend:",                  // Windows socket error
		"broken pipe",               // Unix pipe closed
		"connection reset",          // Connection reset by peer
		"context deadline exceeded", // Write timeout
		"context canceled",          // Context was canceled
		"use of closed network connection",
	}
	for _, pattern := range expectedPatterns {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return true
		}
	}
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// wsClient wraps a coder/websocket connection with a write mutex for safe concurrent writes.
type wsClient struct {
	conn           *websocket.Conn
	writeMu        sync.Mutex
	ctx            context.Context
	cancel         context.CancelFunc
	failureCount   int
	failureCountMu sync.Mutex
}

// connectionRateLimiter tracks connection attempts per IP to prevent spam.
type connectionRateLimiter struct {
	mu             sync.Mutex
	connections    map[string]*connectionTracker
	maxPerIP       int
	maxTotal       int
	totalCount     int
	cleanupTick    *time.Ticker
	stopCleanup    chan struct{}
	cleanupStopped bool
}

type connectionTracker struct {
	count      int
	lastAccess time.Time
}

// newConnectionRateLimiter creates a new rate limiter instance.
func newConnectionRateLimiter() *connectionRateLimiter {
	rl := &connectionRateLimiter{
		connections: make(map[string]*connectionTracker),
		maxPerIP:    100, // Maximum 100 concurrent connections per IP (generous for localhost testing)
		maxTotal:    500, // Maximum 500 total concurrent connections
		stopCleanup: make(chan struct{}),
	}
	// Start cleanup goroutine
	rl.cleanupTick = time.NewTicker(1 * time.Minute)
	go rl.cleanup()
	return rl
}

// checkAndIncrement checks if the IP can make a new connection and increments count.
func (rl *connectionRateLimiter) checkAndIncrement(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Check global connection limit first
	if rl.totalCount >= rl.maxTotal {
		return false
	}

	tracker, exists := rl.connections[ip]
	if !exists {
		rl.connections[ip] = &connectionTracker{count: 1, lastAccess: time.Now()}
		rl.totalCount++
		return true
	}

	if tracker.count >= rl.maxPerIP {
		return false
	}

	tracker.count++
	tracker.lastAccess = time.Now()
	rl.totalCount++
	return true
}

// decrement reduces the connection count for an IP.
func (rl *connectionRateLimiter) decrement(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if tracker, exists := rl.connections[ip]; exists {
		tracker.count--
		rl.totalCount--
		if rl.totalCount < 0 {
			rl.totalCount = 0 // Safety check
		}
		tracker.lastAccess = time.Now()
		if tracker.count <= 0 {
			delete(rl.connections, ip)
		}
	}
}

// cleanup periodically removes stale entries.
func (rl *connectionRateLimiter) cleanup() {
	for {
		select {
		case <-rl.stopCleanup:
			rl.mu.Lock()
			rl.cleanupStopped = true
			rl.mu.Unlock()
			rl.cleanupTick.Stop()
			return
		case <-rl.cleanupTick.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, tracker := range rl.connections {
				// Remove entries with no connections and not accessed in 5 minutes
				if tracker.count == 0 && now.Sub(tracker.lastAccess) > 5*time.Minute {
					delete(rl.connections, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// shutdown stops the cleanup goroutine gracefully.
func (rl *connectionRateLimiter) shutdown() {
	rl.mu.Lock()
	if rl.cleanupStopped {
		rl.mu.Unlock()
		return
	}
	rl.mu.Unlock()

	select {
	case rl.stopCleanup <- struct{}{}:
		// Sent shutdown signal
	default:
		// Channel already closed or blocked
	}
}

// getClientIP extracts the client IP from the request.
// For localhost-only servers, we use RemoteAddr directly to avoid IP spoofing.
// Proxy headers (X-Forwarded-For) are intentionally ignored for security.
func getClientIP(r *http.Request) string {
	return getClientIPDirect(r)
}

// getClientIPDirect extracts IP from RemoteAddr without trusting proxy headers.
func getClientIPDirect(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// checkOrigin validates the WebSocket connection origin.
// Only allows localhost connections to prevent CSWSH attacks.
func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	// Empty origin is allowed for programmatic WebSocket clients (CLI tools, testing)
	// Browsers always send Origin header, so this still protects against CSRF
	if origin == "" {
		// Additional check: ensure connection is from localhost
		// This prevents remote programmatic attacks
		remoteIP := getClientIPDirect(r)
		return remoteIP == "127.0.0.1" || remoteIP == "::1" || remoteIP == "localhost"
	}

	// Parse origin URL properly to handle all cases including IPv6
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	// Get hostname without port
	host := u.Hostname()

	// Allow localhost, 127.0.0.1, and IPv6 localhost (::1)
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// acceptWebSocket accepts a WebSocket connection with origin validation and rate limiting.
func acceptWebSocket(w http.ResponseWriter, r *http.Request, rateLimiter *connectionRateLimiter) (*websocket.Conn, error) {
	// Validate origin before accepting
	if !checkOrigin(r) {
		http.Error(w, "Forbidden: invalid origin", http.StatusForbidden)
		return nil, http.ErrAbortHandler
	}

	// Check rate limit
	clientIP := getClientIP(r)
	if !rateLimiter.checkAndIncrement(clientIP) {
		http.Error(w, "Too many connections from this IP", http.StatusTooManyRequests)
		return nil, http.ErrAbortHandler
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// We handle origin checking manually above for clearer error messages
		InsecureSkipVerify: true,
	})
	if err != nil {
		// Decrement on failure
		rateLimiter.decrement(clientIP)
		return nil, err
	}

	return conn, nil
}

// newWSClientWithContext creates a new WebSocket client wrapper with parent context.
func newWSClientWithContext(ctx context.Context, conn *websocket.Conn) *wsClient {
	clientCtx, cancel := context.WithCancel(ctx)
	return &wsClient{
		conn:   conn,
		ctx:    clientCtx,
		cancel: cancel,
	}
}

// writeJSON safely writes JSON to the WebSocket connection with mutex protection.
// Marshals JSON outside the mutex to reduce lock contention.
func (c *wsClient) writeJSON(data interface{}) error {
	// Marshal OUTSIDE the lock to avoid blocking other writers during CPU work
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	// Use reasonable timeout for local WebSocket connections
	ctx, cancel := context.WithTimeout(c.ctx, service.DefaultWebSocketWriteTimeout)
	defer cancel()

	return c.conn.Write(ctx, websocket.MessageText, jsonBytes)
}

// close closes the WebSocket connection and decrements the rate limiter.
func (c *wsClient) close() error {
	c.cancel()
	return c.conn.Close(websocket.StatusNormalClosure, "server closing")
}

// closeWithRateLimit closes the connection and decrements the rate limiter for the given IP.
func (c *wsClient) closeWithRateLimit(clientIP string, rateLimiter *connectionRateLimiter) error {
	defer func() {
		if rateLimiter != nil {
			rateLimiter.decrement(clientIP)
		}
	}()
	c.cancel()
	return c.conn.Close(websocket.StatusNormalClosure, "server closing")
}

// recordWriteFailure increments the failure counter for backpressure tracking.
func (c *wsClient) recordWriteFailure() int {
	c.failureCountMu.Lock()
	defer c.failureCountMu.Unlock()
	c.failureCount++
	return c.failureCount
}

// readMessage reads a single message from the WebSocket connection.
func readMessage(client *wsClient) error {
	ctx, cancel := context.WithTimeout(client.ctx, service.DefaultWebSocketPongWait)
	defer cancel()

	_, _, err := client.conn.Read(ctx)
	if err != nil {
		// Check if it's a normal closure
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			return nil
		}
		return err
	}
	return nil
}

// wsHealthMonitor manages WebSocket connection health monitoring with ping/pong for coder/websocket.
type wsHealthMonitor struct {
	client     *wsClient
	stopChan   chan struct{}
	pingTicker *time.Ticker
	stopOnce   sync.Once // Ensure stopChan is only closed once
}

// newWSHealthMonitor creates a new WebSocket health monitor.
func newWSHealthMonitor(client *wsClient) *wsHealthMonitor {
	return &wsHealthMonitor{
		client:   client,
		stopChan: make(chan struct{}),
	}
}

// start begins the health monitoring with ping/pong messages.
// Returns an error channel that will receive any fatal errors.
func (m *wsHealthMonitor) start() <-chan error {
	errChan := make(chan error, 1)

	// Start ping ticker - coder/websocket handles pings automatically, but we can configure read deadline
	m.pingTicker = time.NewTicker(service.DefaultWebSocketPingPeriod)

	// Set initial read deadline to detect dead connections
	m.client.conn.SetReadLimit(10 * 1024 * 1024) // 10MB max message size

	// Start ping loop in goroutine
	go func() {
		defer m.pingTicker.Stop()
		defer close(errChan) // Close channel when monitoring stops
		for {
			select {
			case <-m.stopChan:
				return
			case <-m.pingTicker.C:
				// coder/websocket manages read deadlines internally via context
				// The ping/pong mechanism combined with read timeouts detects dead connections

				// Send ping to verify connection is alive
				ctx, cancel := context.WithTimeout(m.client.ctx, service.DefaultWebSocketWriteTimeout)
				err := m.client.conn.Ping(ctx)
				cancel()

				if err != nil {
					slog.Debug("ping failed, connection likely closed", "error", err)
					// Try to send error with timeout to prevent goroutine leak
					select {
					case errChan <- err:
					case <-time.After(100 * time.Millisecond):
						// Receiver not listening, exit to prevent goroutine leak
						slog.Debug("health monitor error send timeout, receiver not listening")
					case <-m.client.ctx.Done():
						// Client context canceled
					}
					return
				}
			}
		}
	}()

	return errChan
}

// stop stops the health monitor.
// Safe to call multiple times - will only close stopChan once.
func (m *wsHealthMonitor) stop() {
	m.stopOnce.Do(func() {
		close(m.stopChan)
	})
}
