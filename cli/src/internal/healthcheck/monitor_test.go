package healthcheck

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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
			name:     "empty",
			results:  []HealthCheckResult{},
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
				w.Write([]byte(tt.responseBody))
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
		w.Write([]byte(`{"status":"healthy"}`))
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
