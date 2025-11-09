package healthcheck

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/registry"
)

// TestCheckServiceNoPortNoPID tests service with no port or PID
func TestCheckServiceNoPortNoPID(t *testing.T) {
	checker := &HealthChecker{
		timeout:         5 * time.Second,
		defaultEndpoint: "/health",
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	svc := serviceInfo{
		Name: "test-service",
	}

	result := checker.CheckService(context.Background(), svc)

	if result.Status != HealthStatusUnknown {
		t.Errorf("Expected status unknown, got %s", result.Status)
	}

	if result.CheckType != HealthCheckTypeProcess {
		t.Errorf("Expected check type process, got %s", result.CheckType)
	}

	if result.Error == "" {
		t.Error("Expected error message for service with no check method")
	}
}

// TestCheckServiceWithPIDOnly tests process-only health check
func TestCheckServiceWithPIDOnly(t *testing.T) {
	checker := &HealthChecker{
		timeout:         5 * time.Second,
		defaultEndpoint: "/health",
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// Use current process PID for testing
	currentPID := os.Getpid()

	svc := serviceInfo{
		Name: "test-service",
		PID:  currentPID,
	}

	result := checker.CheckService(context.Background(), svc)

	if result.CheckType != HealthCheckTypeProcess {
		t.Errorf("Expected check type process, got %s", result.CheckType)
	}

	// On Unix-like systems, current process should be running
	// On Windows, the check might not work reliably
	if runtime.GOOS != "windows" {
		// The isProcessRunning function may have platform-specific behavior
		// Just verify that we got a result
		if result.Status != HealthStatusHealthy && result.Status != HealthStatusUnhealthy {
			t.Errorf("Expected status healthy or unhealthy, got %s", result.Status)
		}
	}
}

// TestCheckServiceWithDeadProcess tests process check with non-existent PID
func TestCheckServiceWithDeadProcess(t *testing.T) {
	checker := &HealthChecker{
		timeout:         5 * time.Second,
		defaultEndpoint: "/health",
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// Use a very high PID that likely doesn't exist
	deadPID := 999999

	svc := serviceInfo{
		Name: "test-service",
		PID:  deadPID,
	}

	result := checker.CheckService(context.Background(), svc)

	if result.CheckType != HealthCheckTypeProcess {
		t.Errorf("Expected check type process, got %s", result.CheckType)
	}

	if result.Status != HealthStatusUnhealthy {
		t.Errorf("Expected status unhealthy for dead process, got %s", result.Status)
	}
}

// TestTryHTTPHealthCheckContextCancellation tests context cancellation during HTTP check
func TestTryHTTPHealthCheckContextCancellation(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(200)
	}))
	defer server.Close()

	port := server.Listener.Addr().(*net.TCPAddr).Port

	checker := &HealthChecker{
		timeout:         5 * time.Second,
		defaultEndpoint: "/health",
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// Create context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := checker.tryHTTPHealthCheck(ctx, port)

	// Should return nil when context is cancelled
	if result != nil {
		t.Error("Expected nil result when context is cancelled")
	}
}

// TestTryHTTPHealthCheckInvalidJSON tests handling of invalid JSON in health response
func TestTryHTTPHealthCheckInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{invalid json`))
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

	result := checker.tryHTTPHealthCheck(context.Background(), port)

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Should still be healthy based on status code, despite invalid JSON
	if result.Status != HealthStatusHealthy {
		t.Errorf("Expected status healthy, got %s", result.Status)
	}

	// Details should be nil or empty due to invalid JSON
	if result.Details != nil && len(result.Details) > 0 {
		t.Error("Expected empty details for invalid JSON")
	}
}

// TestTryHTTPHealthCheckLargeResponse tests handling of very large response bodies
func TestTryHTTPHealthCheckLargeResponse(t *testing.T) {
	// Create a response larger than maxResponseBodySize
	largeData := make([]byte, maxResponseBodySize+1000)
	for i := range largeData {
		largeData[i] = 'a'
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write(largeData)
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

	result := checker.tryHTTPHealthCheck(context.Background(), port)

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Should still be healthy based on status code
	if result.Status != HealthStatusHealthy {
		t.Errorf("Expected status healthy, got %s", result.Status)
	}
}

// TestTryHTTPHealthCheckDifferentEndpoints tests trying multiple health endpoints
func TestTryHTTPHealthCheckDifferentEndpoints(t *testing.T) {
	// Create a server that only responds to /healthz
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(200)
			w.Write([]byte(`{"status":"healthy"}`))
		}
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

	result := checker.tryHTTPHealthCheck(context.Background(), port)

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Should successfully get health status from one of the endpoints
	if result.Status == HealthStatusUnknown {
		t.Errorf("Expected status not to be unknown, got %s", result.Status)
	}
}

// TestCheckServiceUptime tests uptime calculation
func TestCheckServiceUptime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
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

	startTime := time.Now().Add(-10 * time.Minute)

	svc := serviceInfo{
		Name:      "test-service",
		Port:      port,
		StartTime: startTime,
	}

	result := checker.CheckService(context.Background(), svc)

	if result.Uptime <= 0 {
		t.Error("Expected positive uptime")
	}

	if result.Uptime < 9*time.Minute || result.Uptime > 11*time.Minute {
		t.Errorf("Expected uptime around 10 minutes, got %v", result.Uptime)
	}
}

// TestFilterServicesEmptyFilter tests filtering with empty filter list
func TestFilterServicesEmptyFilter(t *testing.T) {
	services := []serviceInfo{
		{Name: "web"},
		{Name: "api"},
	}

	filtered := filterServices(services, []string{})

	if len(filtered) != 0 {
		t.Errorf("Expected 0 services with empty filter, got %d", len(filtered))
	}
}

// TestFilterServicesNonExistent tests filtering with non-existent service names
func TestFilterServicesNonExistent(t *testing.T) {
	services := []serviceInfo{
		{Name: "web"},
		{Name: "api"},
	}

	filter := []string{"database", "cache"}
	filtered := filterServices(services, filter)

	if len(filtered) != 0 {
		t.Errorf("Expected 0 services for non-existent filter, got %d", len(filtered))
	}
}

// TestBuildServiceListNilAzureYaml tests service list building with nil azure.yaml
func TestBuildServiceListNilAzureYaml(t *testing.T) {
	monitor := &HealthMonitor{
		config: MonitorConfig{
			ProjectDir: "/tmp",
		},
	}

	registeredServices := []*registry.ServiceRegistryEntry{
		{
			Name: "web",
			Port: 8080,
			PID:  1234,
		},
	}

	services := monitor.buildServiceList(nil, registeredServices)

	if len(services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(services))
	}

	if services[0].Name != "web" {
		t.Errorf("Expected service name 'web', got '%s'", services[0].Name)
	}
}

// TestCalculateSummaryUnknownStatus tests summary with unknown status
func TestCalculateSummaryUnknownStatus(t *testing.T) {
	results := []HealthCheckResult{
		{Status: HealthStatusHealthy},
		{Status: HealthStatusUnknown},
	}

	summary := calculateSummary(results)

	if summary.Total != 2 {
		t.Errorf("Expected total 2, got %d", summary.Total)
	}

	if summary.Healthy != 1 {
		t.Errorf("Expected healthy 1, got %d", summary.Healthy)
	}

	if summary.Unknown != 1 {
		t.Errorf("Expected unknown 1, got %d", summary.Unknown)
	}

	// Overall should be healthy when some services are healthy and rest are unknown
	// (unknown doesn't degrade overall status, only unhealthy/degraded do)
	if summary.Overall != HealthStatusHealthy {
		t.Errorf("Expected overall healthy, got %s", summary.Overall)
	}
}
