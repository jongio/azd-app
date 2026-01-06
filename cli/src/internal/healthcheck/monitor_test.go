package healthcheck

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

func TestHealthStatus(t *testing.T) {
	statuses := []HealthStatus{
		HealthStatusHealthy,
		HealthStatusDegraded,
		HealthStatusUnhealthy,
		HealthStatusStarting,
		HealthStatusUnknown,
	}

	for _, status := range statuses {
		if string(status) == "" {
			t.Errorf("HealthStatus should not be empty")
		}
	}
}

func TestHealthCheckType(t *testing.T) {
	types := []HealthCheckType{
		HealthCheckTypeHTTP,
		HealthCheckTypeTCP,
		HealthCheckTypeProcess,
	}

	for _, checkType := range types {
		if string(checkType) == "" {
			t.Errorf("HealthCheckType should not be empty")
		}
	}
}

func TestCalculateSummary(t *testing.T) {
	tests := []struct {
		name     string
		results  []HealthCheckResult
		expected HealthSummary
	}{
		{
			name: "all healthy",
			results: []HealthCheckResult{
				{Status: HealthStatusHealthy},
				{Status: HealthStatusHealthy},
			},
			expected: HealthSummary{
				Total:   2,
				Healthy: 2,
				Overall: HealthStatusHealthy,
			},
		},
		{
			name: "mixed status",
			results: []HealthCheckResult{
				{Status: HealthStatusHealthy},
				{Status: HealthStatusDegraded},
			},
			expected: HealthSummary{
				Total:    2,
				Healthy:  1,
				Degraded: 1,
				Overall:  HealthStatusDegraded,
			},
		},
		{
			name: "has unhealthy",
			results: []HealthCheckResult{
				{Status: HealthStatusHealthy},
				{Status: HealthStatusUnhealthy},
			},
			expected: HealthSummary{
				Total:     2,
				Healthy:   1,
				Unhealthy: 1,
				Overall:   HealthStatusUnhealthy,
			},
		},
		{
			name:    "empty",
			results: []HealthCheckResult{},
			expected: HealthSummary{
				Total:   0,
				Overall: HealthStatusUnknown,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := calculateSummary(tt.results)

			if summary.Total != tt.expected.Total {
				t.Errorf("Expected total %d, got %d", tt.expected.Total, summary.Total)
			}
			if summary.Healthy != tt.expected.Healthy {
				t.Errorf("Expected healthy %d, got %d", tt.expected.Healthy, summary.Healthy)
			}
			if summary.Degraded != tt.expected.Degraded {
				t.Errorf("Expected degraded %d, got %d", tt.expected.Degraded, summary.Degraded)
			}
			if summary.Unhealthy != tt.expected.Unhealthy {
				t.Errorf("Expected unhealthy %d, got %d", tt.expected.Unhealthy, summary.Unhealthy)
			}
			if summary.Overall != tt.expected.Overall {
				t.Errorf("Expected overall %s, got %s", tt.expected.Overall, summary.Overall)
			}
		})
	}
}

func TestHTTPHealthCheck(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedStatus HealthStatus
	}{
		{
			name:           "200 OK",
			statusCode:     200,
			responseBody:   `{"status":"healthy"}`,
			expectedStatus: HealthStatusHealthy,
		},
		{
			name:           "200 degraded",
			statusCode:     200,
			responseBody:   `{"status":"degraded"}`,
			expectedStatus: HealthStatusDegraded,
		},
		{
			name:           "500 error",
			statusCode:     500,
			responseBody:   `{"error":"internal server error"}`,
			expectedStatus: HealthStatusUnhealthy,
		},
		{
			name:           "302 redirect",
			statusCode:     302,
			responseBody:   "",
			expectedStatus: HealthStatusHealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// Extract port from server URL
			port := server.Listener.Addr().(*net.TCPAddr).Port

			checker := &HealthChecker{
				timeout:         5 * time.Second,
				defaultEndpoint: "/health",
				httpClient: &http.Client{
					Timeout: 5 * time.Second,
					CheckRedirect: func(req *http.Request, via []*http.Request) error {
						return http.ErrUseLastResponse
					},
				},
			}

			result := checker.tryHTTPHealthCheck(context.Background(), port)

			if result == nil {
				t.Fatal("Expected result, got nil")
			}

			if result.Status != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s", tt.expectedStatus, result.Status)
			}

			if result.StatusCode != tt.statusCode {
				t.Errorf("Expected status code %d, got %d", tt.statusCode, result.StatusCode)
			}
		})
	}
}

func TestPortCheck(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	port := server.Listener.Addr().(*net.TCPAddr).Port

	checker := &HealthChecker{
		timeout: 5 * time.Second,
	}

	// Test port that's listening
	if !checker.checkPort(context.Background(), port) {
		t.Error("Expected port to be listening")
	}

	// Test port that's not listening
	if checker.checkPort(context.Background(), 64999) {
		t.Error("Expected port to not be listening")
	}
}

func TestFilterServices(t *testing.T) {
	services := []serviceInfo{
		{Name: "web"},
		{Name: "api"},
		{Name: "db"},
	}

	filter := []string{"web", "api"}
	filtered := filterServices(services, filter)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 services, got %d", len(filtered))
	}

	for _, svc := range filtered {
		if svc.Name != "web" && svc.Name != "api" {
			t.Errorf("Unexpected service in filtered list: %s", svc.Name)
		}
	}
}

func TestCheckService(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer server.Close()

	port := server.Listener.Addr().(*net.TCPAddr).Port

	checker := &HealthChecker{
		timeout:         5 * time.Second,
		defaultEndpoint: "/health",
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}

	svc := serviceInfo{
		Name: "test-service",
		Port: port,
	}

	result := checker.CheckService(context.Background(), svc)

	if result.ServiceName != "test-service" {
		t.Errorf("Expected service name 'test-service', got '%s'", result.ServiceName)
	}

	if result.Status != HealthStatusHealthy {
		t.Errorf("Expected status healthy, got %s", result.Status)
	}

	if result.CheckType != HealthCheckTypeHTTP {
		t.Errorf("Expected check type HTTP, got %s", result.CheckType)
	}
}

func TestCheckServiceFallback(t *testing.T) {
	checker := &HealthChecker{
		timeout:         5 * time.Second,
		defaultEndpoint: "/health",
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// Service with port that's not listening - should fall back to port check
	svc := serviceInfo{
		Name: "test-service",
		Port: 64998, // Not listening
	}

	result := checker.CheckService(context.Background(), svc)

	if result.CheckType != HealthCheckTypeTCP {
		t.Errorf("Expected check type tcp, got %s", result.CheckType)
	}

	if result.Status != HealthStatusUnhealthy {
		t.Errorf("Expected status unhealthy, got %s", result.Status)
	}
}

func TestParseHealthCheckConfig(t *testing.T) {
	tests := []struct {
		name            string
		healthcheck     *service.HealthcheckConfig
		expectedTest    []string
		expectedNil     bool
		expectedRetries int
	}{
		{
			name:        "nil healthcheck",
			healthcheck: nil,
			expectedNil: true,
		},
		{
			name: "URL string test",
			healthcheck: &service.HealthcheckConfig{
				Test:     "http://localhost:8080/health",
				Interval: "30s",
				Timeout:  "10s",
				Retries:  5,
			},
			expectedTest:    []string{"http://localhost:8080/health"},
			expectedRetries: 5,
		},
		{
			name: "CMD array test",
			healthcheck: &service.HealthcheckConfig{
				Test: []interface{}{"CMD", "curl", "-f", "http://localhost/health"},
			},
			expectedTest:    []string{"CMD", "curl", "-f", "http://localhost/health"},
			expectedRetries: 3, // default
		},
		{
			name: "CMD-SHELL test",
			healthcheck: &service.HealthcheckConfig{
				Test: []interface{}{"CMD-SHELL", "curl -f http://localhost/health || exit 1"},
			},
			expectedTest:    []string{"CMD-SHELL", "curl -f http://localhost/health || exit 1"},
			expectedRetries: 3, // default
		},
		{
			name: "default retries when not specified",
			healthcheck: &service.HealthcheckConfig{
				Test: "http://localhost:8080/ready",
			},
			expectedTest:    []string{"http://localhost:8080/ready"},
			expectedRetries: 3,
		},
		{
			name: "disable healthcheck",
			healthcheck: &service.HealthcheckConfig{
				Disable: true,
			},
			expectedTest:    []string{"NONE"},
			expectedRetries: 0, // not set when disabled
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.Service{
				Healthcheck: tt.healthcheck,
			}

			config := parseHealthCheckConfig(svc)

			if tt.expectedNil {
				if config != nil {
					t.Errorf("Expected nil config, got %+v", config)
				}
				return
			}

			if config == nil {
				t.Fatal("Expected non-nil config")
			}

			if len(config.Test) != len(tt.expectedTest) {
				t.Errorf("Expected test length %d, got %d", len(tt.expectedTest), len(config.Test))
			}

			for i, expected := range tt.expectedTest {
				if i < len(config.Test) && config.Test[i] != expected {
					t.Errorf("Expected test[%d] = %q, got %q", i, expected, config.Test[i])
				}
			}

			if config.Retries != tt.expectedRetries {
				t.Errorf("Expected retries %d, got %d", tt.expectedRetries, config.Retries)
			}
		})
	}
}

func TestCustomHealthCheck_HTTPUrl(t *testing.T) {
	// Create a test server that responds healthy
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ready" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"healthy","connections":4}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	checker := &HealthChecker{
		timeout:         5 * time.Second,
		defaultEndpoint: "/health",
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	config := &healthCheckConfig{
		Test: []string{server.URL + "/ready"},
	}

	result := checker.tryCustomHealthCheck(context.Background(), config, serviceInfo{})

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Status != HealthStatusHealthy {
		t.Errorf("Expected healthy status, got %s", result.Status)
	}

	if result.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", result.StatusCode)
	}
}

func TestCustomHealthCheck_HTTPUrl_Unhealthy(t *testing.T) {
	// Create a test server that responds with 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	checker := &HealthChecker{
		timeout:         5 * time.Second,
		defaultEndpoint: "/health",
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	config := &healthCheckConfig{
		Test: []string{server.URL + "/health"},
	}

	result := checker.tryCustomHealthCheck(context.Background(), config, serviceInfo{})

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Status != HealthStatusUnhealthy {
		t.Errorf("Expected unhealthy status, got %s", result.Status)
	}
}

func TestTryHTTPHealthCheck_Skips404(t *testing.T) {
	// Create a test server that returns 404 for /health but 200 for /ready
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ready":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"healthy"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Extract port from test server
	_, portStr, _ := net.SplitHostPort(server.Listener.Addr().String())
	port := 0
	_, _ = fmt.Sscanf(portStr, "%d", &port)

	checker := &HealthChecker{
		timeout:         5 * time.Second,
		defaultEndpoint: "/health",
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	result := checker.tryHTTPHealthCheck(context.Background(), port)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Should have found /ready endpoint (which is in commonHealthPaths)
	if result.Status != HealthStatusHealthy {
		t.Errorf("Expected healthy status, got %s (endpoint: %s)", result.Status, result.Endpoint)
	}
}

func TestTryHTTPHealthCheck_Skips400BadRequest(t *testing.T) {
	// Create a test server that returns 400 for all endpoints
	// This simulates a non-HTTP service like Node.js inspector on a debug port
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Bad Request"))
	}))
	defer server.Close()

	// Extract port from test server
	_, portStr, _ := net.SplitHostPort(server.Listener.Addr().String())
	port := 0
	_, _ = fmt.Sscanf(portStr, "%d", &port)

	checker := &HealthChecker{
		timeout:         5 * time.Second,
		defaultEndpoint: "/health",
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	result := checker.tryHTTPHealthCheck(context.Background(), port)

	// Should return nil since all endpoints returned 400 (not an HTTP service)
	// This allows cascading to port/process check
	if result != nil {
		t.Errorf("Expected nil result for 400 responses (cascade to port check), got status: %s", result.Status)
	}
}

func TestCheckService_StoppedService(t *testing.T) {
	// Test that stopped services skip health checks and return unknown status
	// instead of being marked as unhealthy
	checker := &HealthChecker{
		timeout:         5 * time.Second,
		defaultEndpoint: "/health",
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// Service with stopped status - should skip health check
	svc := serviceInfo{
		Name:           "stopped-service",
		Port:           64998, // Port that isn't listening
		RegistryStatus: "stopped",
	}

	result := checker.CheckService(context.Background(), svc)

	// Stopped services should return unknown status, not unhealthy
	if result.Status != HealthStatusUnknown {
		t.Errorf("Expected status unknown for stopped service, got %s", result.Status)
	}

	if result.ServiceName != "stopped-service" {
		t.Errorf("Expected service name 'stopped-service', got '%s'", result.ServiceName)
	}
}

func TestCheckService_RunningService(t *testing.T) {
	// Test that running services still get normal health checks
	checker := &HealthChecker{
		timeout:         5 * time.Second,
		defaultEndpoint: "/health",
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// Service with running status and non-listening port - should be unhealthy
	svc := serviceInfo{
		Name:           "running-service",
		Port:           64998, // Port that isn't listening
		RegistryStatus: "running",
	}

	result := checker.CheckService(context.Background(), svc)

	// Running services with non-listening ports should be marked as unhealthy
	if result.Status != HealthStatusUnhealthy {
		t.Errorf("Expected status unhealthy for running service with dead port, got %s", result.Status)
	}
}

// TestTrackFailure tests consecutive failure tracking
func TestTrackFailure(t *testing.T) {
	monitor := &HealthMonitor{
		failureCount:    make(map[string]int),
		lastSuccessTime: make(map[string]time.Time),
	}

	serviceName := "test-service"

	// Test: First unhealthy check should set consecutive failures to 1
	result := HealthCheckResult{
		ServiceName: serviceName,
		Status:      HealthStatusUnhealthy,
	}
	monitor.trackFailure(&result)

	if result.ConsecutiveFailures != 1 {
		t.Errorf("Expected consecutive failures to be 1, got %d", result.ConsecutiveFailures)
	}

	// Test: Second unhealthy check should increment to 2
	result2 := HealthCheckResult{
		ServiceName: serviceName,
		Status:      HealthStatusUnhealthy,
	}
	monitor.trackFailure(&result2)

	if result2.ConsecutiveFailures != 2 {
		t.Errorf("Expected consecutive failures to be 2, got %d", result2.ConsecutiveFailures)
	}

	// Test: Third unhealthy check should increment to 3
	result3 := HealthCheckResult{
		ServiceName: serviceName,
		Status:      HealthStatusUnhealthy,
	}
	monitor.trackFailure(&result3)

	if result3.ConsecutiveFailures != 3 {
		t.Errorf("Expected consecutive failures to be 3, got %d", result3.ConsecutiveFailures)
	}

	// Test: Healthy check should reset consecutive failures to 0
	result4 := HealthCheckResult{
		ServiceName: serviceName,
		Status:      HealthStatusHealthy,
	}
	monitor.trackFailure(&result4)

	if result4.ConsecutiveFailures != 0 {
		t.Errorf("Expected consecutive failures to be reset to 0, got %d", result4.ConsecutiveFailures)
	}
	if result4.LastSuccessTime == nil {
		t.Error("Expected last success time to be set")
	}

	// Test: Another unhealthy check after reset should start at 1 again
	result5 := HealthCheckResult{
		ServiceName: serviceName,
		Status:      HealthStatusUnhealthy,
	}
	monitor.trackFailure(&result5)

	if result5.ConsecutiveFailures != 1 {
		t.Errorf("Expected consecutive failures to be 1 after reset, got %d", result5.ConsecutiveFailures)
	}
	if result5.LastSuccessTime == nil {
		t.Error("Expected last success time to be preserved from previous healthy check")
	}
}

// TestTrackFailure_DegradedStatus tests that degraded status doesn't increment failure count
func TestTrackFailure_DegradedStatus(t *testing.T) {
	monitor := &HealthMonitor{
		failureCount:    make(map[string]int),
		lastSuccessTime: make(map[string]time.Time),
	}

	serviceName := "test-service"

	// Set initial failure count
	monitor.failureCount[serviceName] = 3

	// Degraded status should preserve but not increment count
	result := HealthCheckResult{
		ServiceName: serviceName,
		Status:      HealthStatusDegraded,
	}
	monitor.trackFailure(&result)

	if result.ConsecutiveFailures != 3 {
		t.Errorf("Expected consecutive failures to remain 3, got %d", result.ConsecutiveFailures)
	}
	if monitor.failureCount[serviceName] != 3 {
		t.Errorf("Expected failure count to remain 3, got %d", monitor.failureCount[serviceName])
	}
}

// TestTrackFailure_ConcurrentAccess tests thread safety of failure tracking
func TestTrackFailure_ConcurrentAccess(t *testing.T) {
	monitor := &HealthMonitor{
		failureCount:    make(map[string]int),
		lastSuccessTime: make(map[string]time.Time),
	}

	serviceName := "test-service"
	concurrency := 100

	// Run concurrent failure tracking
	done := make(chan bool, concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			result := HealthCheckResult{
				ServiceName: serviceName,
				Status:      HealthStatusUnhealthy,
			}
			monitor.trackFailure(&result)
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < concurrency; i++ {
		<-done
	}

	// Failure count should be exactly the number of concurrent calls
	if monitor.failureCount[serviceName] != concurrency {
		t.Errorf("Expected failure count to be %d, got %d (race condition detected)",
			concurrency, monitor.failureCount[serviceName])
	}
}

// TestTrackFailure_MultipleServices tests tracking failures for multiple services independently
func TestTrackFailure_MultipleServices(t *testing.T) {
	monitor := &HealthMonitor{
		failureCount:    make(map[string]int),
		lastSuccessTime: make(map[string]time.Time),
	}

	// Service 1: 3 failures
	for i := 0; i < 3; i++ {
		result := HealthCheckResult{
			ServiceName: "service1",
			Status:      HealthStatusUnhealthy,
		}
		monitor.trackFailure(&result)
	}

	// Service 2: 5 failures
	for i := 0; i < 5; i++ {
		result := HealthCheckResult{
			ServiceName: "service2",
			Status:      HealthStatusUnhealthy,
		}
		monitor.trackFailure(&result)
	}

	// Verify counts are independent
	if monitor.failureCount["service1"] != 3 {
		t.Errorf("Expected service1 failure count to be 3, got %d", monitor.failureCount["service1"])
	}
	if monitor.failureCount["service2"] != 5 {
		t.Errorf("Expected service2 failure count to be 5, got %d", monitor.failureCount["service2"])
	}

	// Reset service1, should not affect service2
	result := HealthCheckResult{
		ServiceName: "service1",
		Status:      HealthStatusHealthy,
	}
	monitor.trackFailure(&result)

	if monitor.failureCount["service1"] != 0 {
		t.Errorf("Expected service1 failure count to be reset to 0, got %d", monitor.failureCount["service1"])
	}
	if monitor.failureCount["service2"] != 5 {
		t.Errorf("Expected service2 failure count to remain 5, got %d", monitor.failureCount["service2"])
	}
}

// TestTrackFailure_LastSuccessTime tests last success time tracking
func TestTrackFailure_LastSuccessTime(t *testing.T) {
	monitor := &HealthMonitor{
		failureCount:    make(map[string]int),
		lastSuccessTime: make(map[string]time.Time),
	}

	serviceName := "test-service"

	// First healthy check should set last success time
	result1 := HealthCheckResult{
		ServiceName: serviceName,
		Status:      HealthStatusHealthy,
	}
	monitor.trackFailure(&result1)

	if result1.LastSuccessTime == nil {
		t.Fatal("Expected last success time to be set")
	}
	firstSuccess := *result1.LastSuccessTime

	// Sleep briefly to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Unhealthy check should preserve last success time
	result2 := HealthCheckResult{
		ServiceName: serviceName,
		Status:      HealthStatusUnhealthy,
	}
	monitor.trackFailure(&result2)

	if result2.LastSuccessTime == nil {
		t.Fatal("Expected last success time to be preserved")
	}
	if !result2.LastSuccessTime.Equal(firstSuccess) {
		t.Error("Expected last success time to match first success")
	}

	// Another healthy check should update last success time
	result3 := HealthCheckResult{
		ServiceName: serviceName,
		Status:      HealthStatusHealthy,
	}
	monitor.trackFailure(&result3)

	if result3.LastSuccessTime == nil {
		t.Fatal("Expected last success time to be updated")
	}
	if result3.LastSuccessTime.Before(firstSuccess) {
		t.Error("Expected last success time to be more recent")
	}
}
