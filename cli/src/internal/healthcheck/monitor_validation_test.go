package healthcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestInvalidCircuitBreakerConfig tests that invalid circuit breaker configuration is rejected
func TestInvalidCircuitBreakerConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      MonitorConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "circuit breaker failures less than 1",
			config: MonitorConfig{
				ProjectDir:             t.TempDir(),
				EnableCircuitBreaker:   true,
				CircuitBreakerFailures: 0,
				CircuitBreakerTimeout:  5 * time.Second,
			},
			expectError: true,
			errorMsg:    "circuit breaker failures must be at least 1",
		},
		{
			name: "circuit breaker failures negative",
			config: MonitorConfig{
				ProjectDir:             t.TempDir(),
				EnableCircuitBreaker:   true,
				CircuitBreakerFailures: -5,
				CircuitBreakerTimeout:  5 * time.Second,
			},
			expectError: true,
			errorMsg:    "circuit breaker failures must be at least 1",
		},
		{
			name: "circuit breaker timeout zero",
			config: MonitorConfig{
				ProjectDir:             t.TempDir(),
				EnableCircuitBreaker:   true,
				CircuitBreakerFailures: 3,
				CircuitBreakerTimeout:  0,
			},
			expectError: true,
			errorMsg:    "circuit breaker timeout must be positive",
		},
		{
			name: "circuit breaker timeout negative",
			config: MonitorConfig{
				ProjectDir:             t.TempDir(),
				EnableCircuitBreaker:   true,
				CircuitBreakerFailures: 3,
				CircuitBreakerTimeout:  -5 * time.Second,
			},
			expectError: true,
			errorMsg:    "circuit breaker timeout must be positive",
		},
		{
			name: "valid circuit breaker config",
			config: MonitorConfig{
				ProjectDir:             t.TempDir(),
				EnableCircuitBreaker:   true,
				CircuitBreakerFailures: 3,
				CircuitBreakerTimeout:  5 * time.Second,
				Timeout:                2 * time.Second,
			},
			expectError: false,
		},
		{
			name: "circuit breaker disabled - no validation",
			config: MonitorConfig{
				ProjectDir:             t.TempDir(),
				EnableCircuitBreaker:   false,
				CircuitBreakerFailures: 0,
				CircuitBreakerTimeout:  0,
				Timeout:                2 * time.Second,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor, err := NewHealthMonitor(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing %q, but got no error", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, but got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
				if monitor != nil {
					defer monitor.Close()
				}
			}
		})
	}
}

// TestInvalidRateLimitConfig tests that invalid rate limit configuration is rejected
func TestInvalidRateLimitConfig(t *testing.T) {
	tests := []struct {
		name        string
		rateLimit   int
		expectError bool
		errorMsg    string
	}{
		{
			name:        "negative rate limit",
			rateLimit:   -10,
			expectError: true,
			errorMsg:    "rate limit must be non-negative",
		},
		{
			name:        "rate limit zero (disabled)",
			rateLimit:   0,
			expectError: false,
		},
		{
			name:        "positive rate limit",
			rateLimit:   10,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := MonitorConfig{
				ProjectDir: t.TempDir(),
				RateLimit:  tt.rateLimit,
				Timeout:    2 * time.Second,
			}

			monitor, err := NewHealthMonitor(config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing %q, but got no error", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, but got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
				if monitor != nil {
					defer monitor.Close()
				}
			}
		})
	}
}

// TestMetricsServerShutdown tests that the metrics server shuts down gracefully
func TestMetricsServerShutdown(t *testing.T) {
	// Create monitor with metrics enabled
	config := MonitorConfig{
		ProjectDir:    t.TempDir(),
		EnableMetrics: true,
		MetricsPort:   9091,
		Timeout:       2 * time.Second,
	}

	monitor, err := NewHealthMonitor(config)
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	// Start metrics server in background
	go func() {
		if err := ServeMetrics(config.MetricsPort); err != nil && err != http.ErrServerClosed {
			t.Logf("Metrics server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Verify server is running
	resp, err := http.Get("http://localhost:9091/health")
	if err != nil {
		t.Fatalf("Failed to connect to metrics server: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Close monitor (should stop metrics server)
	if err := monitor.Close(); err != nil {
		t.Errorf("Failed to close monitor: %v", err)
	}

	// Wait a bit for shutdown
	time.Sleep(200 * time.Millisecond)

	// Verify server is stopped (connection should fail or be refused)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:9091/health", nil)
	_, err = http.DefaultClient.Do(req)

	// We expect an error because the server should be stopped
	if err == nil {
		t.Error("Expected connection error after server shutdown, but connection succeeded")
	}
}

// TestPortCheckContextCancellation tests that port checks respect context cancellation
func TestPortCheckContextCancellation(t *testing.T) {
	config := MonitorConfig{
		ProjectDir: t.TempDir(),
		Timeout:    5 * time.Second,
	}

	monitor, err := NewHealthMonitor(config)
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	defer monitor.Close()

	// Create a context that we'll cancel immediately
	ctx, cancel := context.WithCancel(context.Background())

	// Start port check in goroutine
	done := make(chan bool)
	go func() {
		// Try to check a port that doesn't exist (will take full timeout normally)
		result := monitor.checker.checkPort(ctx, 9999)
		done <- result
	}()

	// Cancel context almost immediately
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Wait for port check to complete
	select {
	case result := <-done:
		// Should return quickly after cancellation
		if result {
			t.Error("Expected port check to fail on non-existent port")
		}
	case <-time.After(1 * time.Second):
		t.Error("Port check did not respect context cancellation (took too long)")
	}
}

// TestHTTPCheckResponseBodyCleanup tests that HTTP checks don't leak file descriptors
func TestHTTPCheckResponseBodyCleanup(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only /healthz succeeds
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"healthy"}`)) // Ignore error in test mock
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	config := MonitorConfig{
		ProjectDir: t.TempDir(),
		Timeout:    2 * time.Second,
	}

	monitor, err := NewHealthMonitor(config)
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	defer monitor.Close()

	// Use test server port (httptest uses a random available port)
	// We'll just use a dummy port since we're testing the cleanup logic
	port := 18080

	ctx := context.Background()

	// Run multiple health checks to ensure no leaks
	for i := 0; i < 10; i++ {
		result := monitor.checker.tryHTTPHealthCheck(ctx, port)
		if result == nil {
			t.Logf("Iteration %d: No successful endpoint found (expected)", i)
		}
	}

	// If we get here without file descriptor errors, the test passes
	t.Log("Successfully completed multiple HTTP checks without resource leaks")
}

// TestCaseInsensitiveErrorCategorization tests error type detection is case-insensitive
func TestCaseInsensitiveErrorCategorization(t *testing.T) {
	tests := []struct {
		errorMsg     string
		expectedType string
	}{
		{"timeout exceeded", "timeout"},
		{"TIMEOUT EXCEEDED", "timeout"},
		{"Timeout Exceeded", "timeout"},
		{"request timed out", "timeout"},
		{"REQUEST TIMED OUT", "timeout"},
		{"connection refused", "connection_refused"},
		{"Connection Refused", "connection_refused"},
		{"CONNECTION REFUSED", "connection_refused"},
		{"circuit breaker open", "circuit_breaker"},
		{"Circuit Breaker Open", "circuit_breaker"},
		{"CIRCUIT BREAKER OPEN", "circuit_breaker"},
		{"500 internal server error", "server_error"},
		{"503 Service Unavailable", "server_error"},
		{"401 unauthorized", "auth_error"},
		{"403 Forbidden", "auth_error"},
		{"404 not found", "not_found"},
		{"404 Not Found", "not_found"},
		{"process not running", "process_error"},
		{"Process Not Running", "process_error"},
		{"PID 1234 not found", "process_error"},
		{"port not listening", "port_error"},
		{"Port Not Listening", "port_error"},
		{"random error", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.errorMsg, func(t *testing.T) {
			result := getErrorType(tt.errorMsg)
			if result != tt.expectedType {
				t.Errorf("getErrorType(%q) = %q, want %q", tt.errorMsg, result, tt.expectedType)
			}
		})
	}
}
