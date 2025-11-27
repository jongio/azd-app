package dashboard

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// wsClient wraps a nhooyr websocket connection with a write mutex for safe concurrent writes.
type wsClient struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
}

// checkOrigin validates the WebSocket connection origin.
// Only allows localhost connections to prevent CSWSH attacks.
func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	// Empty origin is allowed for direct WebSocket connections (non-browser clients like CLI tools)
	// This is a security trade-off: we allow programmatic access but require localhost for browsers
	if origin == "" {
		return true
	}
	return strings.HasPrefix(origin, "http://localhost:") ||
		strings.HasPrefix(origin, "http://127.0.0.1:") ||
		strings.HasPrefix(origin, "https://localhost:") ||
		strings.HasPrefix(origin, "https://127.0.0.1:")
}

// acceptWebSocket accepts a WebSocket connection with origin validation.
func acceptWebSocket(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	// Validate origin before accepting
	if !checkOrigin(r) {
		http.Error(w, "Forbidden: invalid origin", http.StatusForbidden)
		return nil, http.ErrAbortHandler
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// We handle origin checking manually above for clearer error messages
		InsecureSkipVerify: true,
	})
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// newWSClient creates a new WebSocket client wrapper.
func newWSClient(conn *websocket.Conn) *wsClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &wsClient{
		conn:   conn,
		ctx:    ctx,
		cancel: cancel,
	}
}

// writeJSON safely writes JSON to the WebSocket connection with mutex protection.
func (c *wsClient) writeJSON(data interface{}) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(c.ctx, service.DefaultWebSocketWriteTimeout)
	defer cancel()

	return c.conn.Write(ctx, websocket.MessageText, jsonBytes)
}

// close closes the WebSocket connection.
func (c *wsClient) close() error {
	c.cancel()
	return c.conn.Close(websocket.StatusNormalClosure, "server closing")
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

// wsHealthMonitorNhooyr manages WebSocket connection health monitoring with ping/pong for nhooyr/websocket.
type wsHealthMonitorNhooyr struct {
	client     *wsClient
	stopChan   chan struct{}
	pingTicker *time.Ticker
}

// newWSHealthMonitorNhooyr creates a new WebSocket health monitor.
func newWSHealthMonitorNhooyr(client *wsClient) *wsHealthMonitorNhooyr {
	return &wsHealthMonitorNhooyr{
		client:   client,
		stopChan: make(chan struct{}),
	}
}

// start begins the health monitoring with ping/pong messages.
// Returns an error channel that will receive any fatal errors.
func (m *wsHealthMonitorNhooyr) start() <-chan error {
	errChan := make(chan error, 1)

	// Start ping ticker - nhooyr/websocket handles pings automatically, but we can configure read deadline
	m.pingTicker = time.NewTicker(service.DefaultWebSocketPingPeriod)

	// Start ping loop in goroutine
	go func() {
		defer m.pingTicker.Stop()
		for {
			select {
			case <-m.stopChan:
				return
			case <-m.pingTicker.C:
				// nhooyr/websocket handles pings automatically when reading
				// We just need to ensure we're reading periodically
				ctx, cancel := context.WithTimeout(m.client.ctx, service.DefaultWebSocketWriteTimeout)
				err := m.client.conn.Ping(ctx)
				cancel()

				if err != nil {
					slog.Debug("ping failed, connection likely closed", "error", err)
					errChan <- err
					return
				}
			}
		}
	}()

	return errChan
}

// stop stops the health monitor.
func (m *wsHealthMonitorNhooyr) stop() {
	close(m.stopChan)
}
