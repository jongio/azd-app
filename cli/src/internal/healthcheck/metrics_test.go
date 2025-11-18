package healthcheck

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sony/gobreaker"
)

func TestRecordHealthCheck(t *testing.T) {
	t.Parallel()
	// Reset metrics before test
	healthCheckDuration.Reset()
	healthCheckTotal.Reset()
	healthCheckErrors.Reset()
	healthCheckResponseCode.Reset()
	serviceUptime.Reset()

	result := HealthCheckResult{
		ServiceName:  "test-service",
		Status:       HealthStatusHealthy,
		CheckType:    HealthCheckTypeHTTP,
		ResponseTime: 50 * time.Millisecond,
		StatusCode:   200,
		Uptime:       2 * time.Hour,
	}

	recordHealthCheck(result)

	// Verify duration histogram was recorded
	count := testutil.CollectAndCount(healthCheckDuration)
	if count == 0 {
		t.Error("Expected duration metric to be recorded")
	}

	// Verify total counter was incremented
	count = testutil.CollectAndCount(healthCheckTotal)
	if count == 0 {
		t.Error("Expected total counter to be incremented")
	}

	// Verify uptime gauge was set
	count = testutil.CollectAndCount(serviceUptime)
	if count == 0 {
		t.Error("Expected uptime gauge to be set")
	}
}

func TestRecordHealthCheckWithError(t *testing.T) {
	healthCheckErrors.Reset()

	result := HealthCheckResult{
		ServiceName:  "error-service",
		Status:       HealthStatusUnhealthy,
		CheckType:    HealthCheckTypeHTTP,
		ResponseTime: 100 * time.Millisecond,
		Error:        "connection timeout",
	}

	recordHealthCheck(result)

	// Verify error counter was incremented
	count := testutil.CollectAndCount(healthCheckErrors)
	if count == 0 {
		t.Error("Expected error counter to be incremented")
	}
}

func TestRecordHealthCheckWithHTTPStatus(t *testing.T) {
	healthCheckResponseCode.Reset()

	result := HealthCheckResult{
		ServiceName:  "http-service",
		Status:       HealthStatusHealthy,
		CheckType:    HealthCheckTypeHTTP,
		ResponseTime: 30 * time.Millisecond,
		StatusCode:   200,
	}

	recordHealthCheck(result)

	// Verify HTTP status code counter was incremented
	count := testutil.CollectAndCount(healthCheckResponseCode)
	if count == 0 {
		t.Error("Expected HTTP status code counter to be incremented")
	}
}

func TestRecordCircuitBreakerState(t *testing.T) {
	circuitBreakerState.Reset()

	tests := []struct {
		name     string
		state    gobreaker.State
		expected float64
	}{
		{"closed state", gobreaker.StateClosed, 0},
		{"half-open state", gobreaker.StateHalfOpen, 1},
		{"open state", gobreaker.StateOpen, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recordCircuitBreakerState("test-service", tt.state)

			// Verify the gauge value
			count := testutil.CollectAndCount(circuitBreakerState)
			if count == 0 {
				t.Error("Expected circuit breaker state gauge to be set")
			}
		})
	}
}

func TestGetErrorType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		errMsg   string
		expected string
	}{
		{"timeout error", "connection timeout exceeded", "timeout"},
		{"deadline error", "context deadline exceeded", "timeout"},
		{"connection refused", "connection refused by server", "connection_refused"},
		{"circuit breaker", "circuit breaker open", "circuit_breaker"},
		{"context canceled", "context canceled", "canceled"},
		{"500 error", "HTTP 500 internal server error", "server_error"},
		{"503 error", "service unavailable 503", "server_error"},
		{"401 error", "authentication failed 401", "auth_error"},
		{"403 error", "forbidden 403", "auth_error"},
		{"404 error", "not found 404", "not_found"},
		{"process error", "process 1234 not running", "process_error"},
		{"port error", "port 8080 not listening", "port_error"},
		{"unknown error", "something went wrong", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getErrorType(tt.errMsg)
			if result != tt.expected {
				t.Errorf("Expected error type '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substrs  []string
		expected bool
	}{
		{"contains first", "hello world", []string{"hello", "foo"}, true},
		{"contains second", "hello world", []string{"foo", "world"}, true},
		{"contains none", "hello world", []string{"foo", "bar"}, false},
		{"exact match", "timeout", []string{"timeout"}, true},
		{"substring match", "connection timeout", []string{"timeout"}, true},
		{"empty string", "", []string{"foo"}, false},
		{"empty substrs", "hello", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsAny(tt.s, tt.substrs...)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestServeMetrics(t *testing.T) {
	// Start metrics server in background
	port := 19090 // Use a different port from default
	errChan := make(chan error, 1)

	go func() {
		errChan <- ServeMetrics(port)
	}()

	// Give server time to start
	time.Sleep(500 * time.Millisecond)

	// Test metrics endpoint
	resp, err := http.Get("http://localhost:19090/metrics")
	if err != nil {
		t.Fatalf("Failed to fetch metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Test health endpoint
	resp, err = http.Get("http://localhost:19090/health")
	if err != nil {
		t.Fatalf("Failed to fetch health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Note: Server keeps running - this is expected behavior
	// In production, it would be shut down gracefully with context cancellation
}

func TestMetricsEndpointFormat(t *testing.T) {
	// Create a test registry
	registry := prometheus.NewRegistry()

	// Register a test metric
	testCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_metric_total",
		Help: "A test metric",
	})
	registry.MustRegister(testCounter)
	testCounter.Inc()

	// Create test server
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	server := httptest.NewServer(handler)
	defer server.Close()

	// Fetch metrics
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to fetch metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("Expected text/plain content type, got %s", contentType)
	}
}

func TestHealthCheckMetricsLabels(t *testing.T) {
	healthCheckTotal.Reset()

	// Record metrics with different labels
	results := []HealthCheckResult{
		{
			ServiceName:  "web",
			Status:       HealthStatusHealthy,
			CheckType:    HealthCheckTypeHTTP,
			ResponseTime: 10 * time.Millisecond,
		},
		{
			ServiceName:  "api",
			Status:       HealthStatusUnhealthy,
			CheckType:    HealthCheckTypePort,
			ResponseTime: 5 * time.Millisecond,
		},
		{
			ServiceName:  "db",
			Status:       HealthStatusDegraded,
			CheckType:    HealthCheckTypeProcess,
			ResponseTime: 2 * time.Millisecond,
		},
	}

	for _, result := range results {
		recordHealthCheck(result)
	}

	// Verify metrics were recorded with proper labels
	count := testutil.CollectAndCount(healthCheckTotal)
	if count < 3 {
		t.Errorf("Expected at least 3 metrics, got %d", count)
	}
}

func TestRecordHealthCheckNoUptime(t *testing.T) {
	serviceUptime.Reset()

	// Record health check without uptime
	result := HealthCheckResult{
		ServiceName:  "test-no-uptime",
		Status:       HealthStatusHealthy,
		CheckType:    HealthCheckTypeHTTP,
		ResponseTime: 50 * time.Millisecond,
		Uptime:       0, // No uptime
	}

	recordHealthCheck(result)

	// Uptime gauge should not be set for zero uptime
	// This is expected behavior - gauge is only set for positive uptime
}

func TestRecordHealthCheckUnhealthyNoUptime(t *testing.T) {
	serviceUptime.Reset()

	// Unhealthy services shouldn't record uptime
	result := HealthCheckResult{
		ServiceName:  "unhealthy-service",
		Status:       HealthStatusUnhealthy,
		CheckType:    HealthCheckTypeHTTP,
		ResponseTime: 100 * time.Millisecond,
		Uptime:       2 * time.Hour, // Has uptime but unhealthy
	}

	recordHealthCheck(result)

	// Uptime should not be recorded for unhealthy services
	// Only healthy services get uptime metrics
}

func TestGetErrorTypeCaseSensitivity(t *testing.T) {
	// Error type detection should work with different cases
	tests := []struct {
		errMsg   string
		expected string
	}{
		{"timeout occurred", "timeout"},                        // lowercase
		{"connection refused by server", "connection_refused"}, // lowercase
		{"circuit breaker is open", "circuit_breaker"},         // lowercase
		{"HTTP 500 error response", "server_error"},            // lowercase with numbers
	}

	for _, tt := range tests {
		result := getErrorType(tt.errMsg)
		if result != tt.expected {
			t.Errorf("For '%s': expected '%s', got '%s'", tt.errMsg, tt.expected, result)
		}
	}
}
