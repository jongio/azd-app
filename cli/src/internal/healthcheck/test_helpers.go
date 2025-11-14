//go:build !prod
// +build !prod

package healthcheck

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"time"
)

// MockHealthServer provides a configurable HTTP server for testing health checks.
type MockHealthServer struct {
	server       *httptest.Server
	port         int
	mu           sync.RWMutex
	statusCode   int32
	response     string
	responseTime time.Duration
	requestCount int32
	failAfter    int32 // Fail after N requests
}

// NewMockHealthServer creates a new mock health server with default healthy state.
func NewMockHealthServer() *MockHealthServer {
	mock := &MockHealthServer{
		statusCode: 200,
		response:   `{"status":"healthy"}`,
	}

	mock.server = httptest.NewServer(http.HandlerFunc(mock.handler))
	mock.port = mock.server.Listener.Addr().(*net.TCPAddr).Port

	return mock
}

// NewMockHealthServerWithConfig creates a mock server with specific configuration.
func NewMockHealthServerWithConfig(statusCode int, response string, delay time.Duration) *MockHealthServer {
	mock := &MockHealthServer{
		statusCode:   int32(statusCode),
		response:     response,
		responseTime: delay,
	}

	mock.server = httptest.NewServer(http.HandlerFunc(mock.handler))
	mock.port = mock.server.Listener.Addr().(*net.TCPAddr).Port

	return mock
}

func (m *MockHealthServer) handler(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt32(&m.requestCount, 1)
	count := atomic.LoadInt32(&m.requestCount)
	failAfter := atomic.LoadInt32(&m.failAfter)

	// Simulate response delay
	if m.responseTime > 0 {
		time.Sleep(m.responseTime)
	}

	// Check if we should fail after N requests
	if failAfter > 0 && count > failAfter {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"status":"unhealthy","error":"service degraded"}`))
		return
	}

	statusCode := int(atomic.LoadInt32(&m.statusCode))
	w.WriteHeader(statusCode)

	m.mu.RLock()
	response := m.response
	m.mu.RUnlock()

	_, _ = w.Write([]byte(response))
}

// Port returns the port the mock server is listening on.
func (m *MockHealthServer) Port() int {
	return m.port
}

// URL returns the base URL of the mock server.
func (m *MockHealthServer) URL() string {
	return m.server.URL
}

// Close shuts down the mock server.
func (m *MockHealthServer) Close() {
	m.server.Close()
}

// SetStatus changes the HTTP status code returned by the server.
func (m *MockHealthServer) SetStatus(code int) {
	atomic.StoreInt32(&m.statusCode, int32(code))
}

// SetResponse changes the response body returned by the server.
func (m *MockHealthServer) SetResponse(response string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.response = response
}

// SetResponseTime sets a delay before responding.
func (m *MockHealthServer) SetResponseTime(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responseTime = delay
}

// SetFailAfter makes the server fail after N requests.
func (m *MockHealthServer) SetFailAfter(n int) {
	atomic.StoreInt32(&m.failAfter, int32(n))
}

// GetRequestCount returns the number of requests received.
func (m *MockHealthServer) GetRequestCount() int {
	return int(atomic.LoadInt32(&m.requestCount))
}

// ResetRequestCount resets the request counter.
func (m *MockHealthServer) ResetRequestCount() {
	atomic.StoreInt32(&m.requestCount, 0)
}

// SimulateHealthy makes the server return healthy status.
func (m *MockHealthServer) SimulateHealthy() {
	m.SetStatus(200)
	m.SetResponse(`{"status":"healthy"}`)
}

// SimulateDegraded makes the server return degraded status.
func (m *MockHealthServer) SimulateDegraded() {
	m.SetStatus(200)
	m.SetResponse(`{"status":"degraded","message":"running with reduced capacity"}`)
}

// SimulateUnhealthy makes the server return unhealthy status.
func (m *MockHealthServer) SimulateUnhealthy() {
	m.SetStatus(503)
	m.SetResponse(`{"status":"unhealthy","error":"database connection failed"}`)
}

// SimulateTimeout makes the server delay response beyond typical timeout.
func (m *MockHealthServer) SimulateTimeout(delay time.Duration) {
	m.SetResponseTime(delay)
}

// SimulateIntermittent makes the server fail every other request.
func (m *MockHealthServer) SimulateIntermittent() {
	m.server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&m.requestCount, 1)
		if count%2 == 0 {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"status":"unhealthy"}`))
		} else {
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"status":"healthy"}`))
		}
	})
}

// MockHealthResponse represents a health check response.
type MockHealthResponse struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// NewHealthyResponse creates a healthy response.
func NewHealthyResponse() string {
	resp := MockHealthResponse{Status: "healthy"}
	data, _ := json.Marshal(resp)
	return string(data)
}

// NewDegradedResponse creates a degraded response with message.
func NewDegradedResponse(message string) string {
	resp := MockHealthResponse{
		Status:  "degraded",
		Message: message,
	}
	data, _ := json.Marshal(resp)
	return string(data)
}

// NewUnhealthyResponse creates an unhealthy response with error.
func NewUnhealthyResponse(err string) string {
	resp := MockHealthResponse{
		Status: "unhealthy",
		Error:  err,
	}
	data, _ := json.Marshal(resp)
	return string(data)
}

// NewDetailedHealthResponse creates a response with custom details.
func NewDetailedHealthResponse(status string, details map[string]interface{}) string {
	resp := MockHealthResponse{
		Status:  status,
		Details: details,
	}
	data, _ := json.Marshal(resp)
	return string(data)
}

// MockPortServer provides a simple TCP server for port health checks.
type MockPortServer struct {
	listener net.Listener
	port     int
	closed   bool
	mu       sync.Mutex
}

// NewMockPortServer creates a new TCP server listening on a random port.
func NewMockPortServer() (*MockPortServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	mock := &MockPortServer{
		listener: listener,
		port:     port,
	}

	// Accept connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // Listener closed
			}
			conn.Close() // Immediately close accepted connections
		}
	}()

	return mock, nil
}

// Port returns the port number.
func (m *MockPortServer) Port() int {
	return m.port
}

// Close stops the server.
func (m *MockPortServer) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true
	return m.listener.Close()
}

// WaitForReady waits for the server to be ready to accept connections.
func WaitForReady(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("port %d not ready after %v", port, timeout)
}
