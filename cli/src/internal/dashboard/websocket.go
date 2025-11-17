package dashboard

import (
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// wsHealthMonitor manages WebSocket connection health monitoring with ping/pong.
type wsHealthMonitor struct {
	conn       *websocket.Conn
	writeMu    *sync.Mutex
	stopChan   chan struct{}
	pingTicker *time.Ticker
}

// newWSHealthMonitor creates a new WebSocket health monitor.
func newWSHealthMonitor(conn *websocket.Conn, writeMu *sync.Mutex) *wsHealthMonitor {
	return &wsHealthMonitor{
		conn:     conn,
		writeMu:  writeMu,
		stopChan: make(chan struct{}),
	}
}

// start begins the health monitoring with ping/pong messages.
// Returns an error channel that will receive any fatal errors.
func (m *wsHealthMonitor) start() <-chan error {
	errChan := make(chan error, 1)

	// Configure read deadline and pong handler
	if err := m.conn.SetReadDeadline(time.Now().Add(service.DefaultWebSocketPongWait)); err != nil {
		slog.Warn("failed to set read deadline", "error", err)
		errChan <- err
		return errChan
	}

	m.conn.SetPongHandler(func(string) error {
		return m.conn.SetReadDeadline(time.Now().Add(service.DefaultWebSocketPongWait))
	})

	// Start ping ticker
	m.pingTicker = time.NewTicker(service.DefaultWebSocketPingPeriod)

	// Start ping loop in goroutine
	go func() {
		defer m.pingTicker.Stop()
		for {
			select {
			case <-m.stopChan:
				return
			case <-m.pingTicker.C:
				// Send ping message
				m.writeMu.Lock()
				err := m.conn.WriteControl(
					websocket.PingMessage,
					[]byte{},
					time.Now().Add(service.DefaultWebSocketWriteTimeout),
				)
				m.writeMu.Unlock()

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
func (m *wsHealthMonitor) stop() {
	close(m.stopChan)
}

// configureWebSocket applies standard WebSocket configuration.
func configureWebSocket(conn *websocket.Conn) error {
	// Set initial read deadline
	if err := conn.SetReadDeadline(time.Now().Add(service.DefaultWebSocketPongWait)); err != nil {
		return err
	}

	// Set pong handler to extend deadline on each pong received
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(service.DefaultWebSocketPongWait))
	})

	return nil
}
