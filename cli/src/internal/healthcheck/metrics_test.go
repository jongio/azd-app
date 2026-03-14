package healthcheck

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/sony/gobreaker"
)

func TestGetErrorType(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		wantType string
	}{
		{
			name:     "timeout error",
			errMsg:   "request timeout exceeded",
			wantType: "timeout",
		},
		{
			name:     "connection refused",
			errMsg:   "connection refused by server",
			wantType: "connection_refused",
		},
		{
			name:     "circuit breaker",
			errMsg:   "circuit breaker open",
			wantType: "circuit_breaker",
		},
		{
			name:     "context canceled",
			errMsg:   "context canceled by user",
			wantType: "canceled",
		},
		{
			name:     "server error 500",
			errMsg:   "server returned 500 internal error",
			wantType: "server_error",
		},
		{
			name:     "auth error",
			errMsg:   "401 unauthorized access",
			wantType: "auth_error",
		},
		{
			name:     "not found",
			errMsg:   "404 endpoint not found",
			wantType: "not_found",
		},
		{
			name:     "process error",
			errMsg:   "process with PID 1234 died",
			wantType: "process_error",
		},
		{
			name:     "port error",
			errMsg:   "port 8080 is not listening",
			wantType: "port_error",
		},
		{
			name:     "unknown error",
			errMsg:   "some random error message",
			wantType: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getErrorType(tt.errMsg)
			if got != tt.wantType {
				t.Errorf("getErrorType(%q) = %q, want %q", tt.errMsg, got, tt.wantType)
			}
		})
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		substrs []string
		want    bool
	}{
		{
			name:    "contains first substring",
			s:       "hello world",
			substrs: []string{"hello", "foo", "bar"},
			want:    true,
		},
		{
			name:    "contains middle substring",
			s:       "hello world",
			substrs: []string{"foo", "world", "bar"},
			want:    true,
		},
		{
			name:    "contains no substring",
			s:       "hello world",
			substrs: []string{"foo", "bar", "baz"},
			want:    false,
		},
		{
			name:    "empty string",
			s:       "",
			substrs: []string{"hello"},
			want:    false,
		},
		{
			name:    "empty substrs",
			s:       "hello",
			substrs: []string{},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsAny(tt.s, tt.substrs...)
			if got != tt.want {
				t.Errorf("containsAny(%q, %v) = %v, want %v", tt.s, tt.substrs, got, tt.want)
			}
		})
	}
}

func TestRecordHealthCheck(t *testing.T) {
	// This test mainly ensures recordHealthCheck doesn't panic
	// Actual metric values would require more complex integration testing

	result := HealthCheckResult{
		ServiceName:  "test-service",
		Status:       HealthStatusHealthy,
		CheckType:    HealthCheckTypeHTTP,
		ResponseTime: 50 * time.Millisecond,
		StatusCode:   200,
		Uptime:       10 * time.Second,
	}

	// Should not panic
	recordHealthCheck(result)

	// Test with error
	resultWithError := HealthCheckResult{
		ServiceName:  "test-service",
		Status:       HealthStatusUnhealthy,
		CheckType:    HealthCheckTypeHTTP,
		ResponseTime: 100 * time.Millisecond,
		Error:        "connection timeout",
	}

	recordHealthCheck(resultWithError)
}

func TestRecordCircuitBreakerState(t *testing.T) {
	// This test mainly ensures recordCircuitBreakerState doesn't panic

	states := []gobreaker.State{
		gobreaker.StateClosed,
		gobreaker.StateHalfOpen,
		gobreaker.StateOpen,
	}

	for _, state := range states {
		// Should not panic
		recordCircuitBreakerState("test-service", state)
	}
}

func TestCreateMetricsServer(t *testing.T) {
	server := CreateMetricsServer(9999)

	if server == nil {
		t.Fatal("Expected non-nil server")
	}

	if server.Addr != ":9999" {
		t.Errorf("Server address = %s, want :9999", server.Addr)
	}

	if server.ReadTimeout != 10*time.Second {
		t.Errorf("ReadTimeout = %v, want 10s", server.ReadTimeout)
	}

	if server.WriteTimeout != 10*time.Second {
		t.Errorf("WriteTimeout = %v, want 10s", server.WriteTimeout)
	}

	if server.IdleTimeout != 60*time.Second {
		t.Errorf("IdleTimeout = %v, want 60s", server.IdleTimeout)
	}
}

func TestCreateMetricsServer_HealthEndpoint(t *testing.T) {
	server := CreateMetricsServer(0) // Use port 0 for testing

	// Create a test request to /health
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Create a response recorder
	rr := &testResponseWriter{
		headers: make(http.Header),
		body:    []byte{},
	}

	// Serve the request
	server.Handler.ServeHTTP(rr, req)

	if rr.statusCode != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", rr.statusCode, http.StatusOK)
	}

	if string(rr.body) != "OK" {
		t.Errorf("Handler returned unexpected body: got %v want OK", string(rr.body))
	}
}

// testResponseWriter is a simple implementation for testing
type testResponseWriter struct {
	headers    http.Header
	body       []byte
	statusCode int
}

func (w *testResponseWriter) Header() http.Header {
	return w.headers
}

func (w *testResponseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return len(b), nil
}

func (w *testResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}
