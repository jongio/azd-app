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
		HealthCheckTypePort,
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

	if result.CheckType != HealthCheckTypePort {
		t.Errorf("Expected check type port, got %s", result.CheckType)
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

	result := checker.tryCustomHealthCheck(context.Background(), config)

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

	result := checker.tryCustomHealthCheck(context.Background(), config)

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
