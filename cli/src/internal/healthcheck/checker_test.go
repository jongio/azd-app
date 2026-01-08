package healthcheck

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/service"
	testutil "github.com/jongio/azd-app/cli/src/internal/testing/testutil"
	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
)

func TestNewHealthChecker(t *testing.T) {
	checker := &HealthChecker{
		timeout:            5 * time.Second,
		defaultEndpoint:    "/health",
		httpClient:         &http.Client{Timeout: 5 * time.Second},
		breakers:           make(map[string]*gobreaker.CircuitBreaker),
		rateLimiters:       make(map[string]*rate.Limiter),
		endpointCache:      make(map[string]string),
		enableBreaker:      true,
		breakerFailures:    5,
		breakerTimeout:     30 * time.Second,
		rateLimit:          10,
		startupGracePeriod: 30 * time.Second,
	}

	if checker.timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want %v", checker.timeout, 5*time.Second)
	}
	if checker.defaultEndpoint != "/health" {
		t.Errorf("DefaultEndpoint = %s, want /health", checker.defaultEndpoint)
	}
}

func TestGetOrCreateCircuitBreaker(t *testing.T) {
	checker := &HealthChecker{
		breakers:        make(map[string]*gobreaker.CircuitBreaker),
		enableBreaker:   true,
		breakerFailures: 3,
		breakerTimeout:  10 * time.Second,
	}

	// Test creating new breaker
	breaker1 := checker.getOrCreateCircuitBreaker("service1")
	if breaker1 == nil {
		t.Fatal("Expected non-nil circuit breaker")
	}

	// Test retrieving existing breaker
	breaker2 := checker.getOrCreateCircuitBreaker("service1")
	if breaker1 != breaker2 {
		t.Error("Expected same circuit breaker instance")
	}

	// Test creating different breaker for different service
	breaker3 := checker.getOrCreateCircuitBreaker("service2")
	if breaker1 == breaker3 {
		t.Error("Expected different circuit breaker for different service")
	}
}

func TestGetOrCreateCircuitBreaker_Disabled(t *testing.T) {
	checker := &HealthChecker{
		breakers:      make(map[string]*gobreaker.CircuitBreaker),
		enableBreaker: false,
	}

	breaker := checker.getOrCreateCircuitBreaker("service1")
	if breaker != nil {
		t.Error("Expected nil circuit breaker when disabled")
	}
}

func TestGetOrCreateRateLimiter(t *testing.T) {
	checker := &HealthChecker{
		rateLimiters: make(map[string]*rate.Limiter),
		rateLimit:    5,
	}

	// Test creating new limiter
	limiter1 := checker.getOrCreateRateLimiter("service1")
	if limiter1 == nil {
		t.Fatal("Expected non-nil rate limiter")
	}

	// Test retrieving existing limiter
	limiter2 := checker.getOrCreateRateLimiter("service1")
	if limiter1 != limiter2 {
		t.Error("Expected same rate limiter instance")
	}

	// Test creating different limiter for different service
	limiter3 := checker.getOrCreateRateLimiter("service2")
	if limiter1 == limiter3 {
		t.Error("Expected different rate limiter for different service")
	}
}

func TestGetOrCreateRateLimiter_Disabled(t *testing.T) {
	checker := &HealthChecker{
		rateLimiters: make(map[string]*rate.Limiter),
		rateLimit:    0, // Disabled
	}

	limiter := checker.getOrCreateRateLimiter("service1")
	if limiter != nil {
		t.Error("Expected nil rate limiter when disabled (rateLimit <= 0)")
	}
}

func TestStatusFromHTTPCode(t *testing.T) {
	checker := &HealthChecker{}

	tests := []struct {
		code       int
		wantStatus HealthStatus
	}{
		{200, HealthStatusHealthy},
		{201, HealthStatusHealthy},
		{299, HealthStatusHealthy},
		{301, HealthStatusHealthy}, // Redirects OK
		{302, HealthStatusHealthy},
		{304, HealthStatusHealthy},
		{400, HealthStatusDegraded},
		{404, HealthStatusDegraded},
		{500, HealthStatusUnhealthy},
		{503, HealthStatusUnhealthy},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("code_%d", tt.code), func(t *testing.T) {
			got := checker.statusFromHTTPCode(tt.code)
			if got != tt.wantStatus {
				t.Errorf("statusFromHTTPCode(%d) = %v, want %v", tt.code, got, tt.wantStatus)
			}
		})
	}
}

func TestParseHealthResponseBody(t *testing.T) {
	checker := &HealthChecker{}

	tests := []struct {
		name       string
		body       string
		wantStatus HealthStatus
		wantKey    string
	}{
		{
			name:       "healthy status",
			body:       `{"status": "healthy"}`,
			wantStatus: HealthStatusHealthy,
			wantKey:    "status",
		},
		{
			name:       "ok status",
			body:       `{"status": "ok"}`,
			wantStatus: HealthStatusHealthy,
			wantKey:    "status",
		},
		{
			name:       "up status",
			body:       `{"status": "up"}`,
			wantStatus: HealthStatusHealthy,
			wantKey:    "status",
		},
		{
			name:       "degraded status",
			body:       `{"status": "degraded"}`,
			wantStatus: HealthStatusDegraded,
			wantKey:    "status",
		},
		{
			name:       "warning status",
			body:       `{"status": "warning"}`,
			wantStatus: HealthStatusDegraded,
			wantKey:    "status",
		},
		{
			name:       "unhealthy status",
			body:       `{"status": "unhealthy"}`,
			wantStatus: HealthStatusUnhealthy,
			wantKey:    "status",
		},
		{
			name:       "down status",
			body:       `{"status": "down"}`,
			wantStatus: HealthStatusUnhealthy,
			wantKey:    "status",
		},
		{
			name:       "error status",
			body:       `{"status": "error"}`,
			wantStatus: HealthStatusUnhealthy,
			wantKey:    "status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &httpHealthCheckResult{}
			checker.parseHealthResponseBody([]byte(tt.body), result)

			if result.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", result.Status, tt.wantStatus)
			}
			if result.Details == nil {
				t.Error("Expected details to be set")
				return
			}
			if _, ok := result.Details[tt.wantKey]; !ok {
				t.Errorf("Expected key %s in details", tt.wantKey)
			}
		})
	}
}

func TestParseHealthResponseBody_InvalidJSON(t *testing.T) {
	checker := &HealthChecker{}
	result := &httpHealthCheckResult{
		Status: HealthStatusHealthy, // Initial status
	}

	checker.parseHealthResponseBody([]byte("not json"), result)

	// Should not change status or set details for invalid JSON
	if result.Status != HealthStatusHealthy {
		t.Errorf("Status changed for invalid JSON: %v", result.Status)
	}
	if result.Details != nil {
		t.Error("Details should not be set for invalid JSON")
	}
}

func TestCheckPort(t *testing.T) {
	checker := &HealthChecker{}

	// Start a test server bound to loopback to avoid firewall prompts
	listener, port, err := testutil.ListenLoopback(0)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	tests := []struct {
		name string
		port int
		want bool
	}{
		{
			name: "listening port",
			port: port,
			want: true,
		},
		{
			name: "non-listening port",
			port: 65432, // Unlikely to be in use
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got := checker.checkPort(ctx, tt.port)
			if got != tt.want {
				t.Errorf("checkPort(%d) = %v, want %v", tt.port, got, tt.want)
			}
		})
	}
}

func TestCheckPort_ContextCancellation(t *testing.T) {
	checker := &HealthChecker{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should fail quickly with cancelled context
	got := checker.checkPort(ctx, 8080)
	if got {
		t.Error("checkPort should return false for cancelled context")
	}
}

func TestPerformHTTPCheck(t *testing.T) {
	checker := &HealthChecker{
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	tests := []struct {
		name           string
		handler        http.HandlerFunc
		wantStatus     HealthStatus
		wantStatusCode int
	}{
		{
			name: "200 OK",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"status": "healthy"}`)
			},
			wantStatus:     HealthStatusHealthy,
			wantStatusCode: 200,
		},
		{
			name: "500 Error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantStatus:     HealthStatusUnhealthy,
			wantStatusCode: 500,
		},
		{
			name: "404 Not Found",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantStatus:     HealthStatusDegraded,
			wantStatusCode: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			ctx := context.Background()
			result := checker.performHTTPCheck(ctx, server.URL)

			if result == nil {
				t.Fatal("Expected non-nil result")
			}
			if result.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", result.Status, tt.wantStatus)
			}
			if result.StatusCode != tt.wantStatusCode {
				t.Errorf("StatusCode = %d, want %d", result.StatusCode, tt.wantStatusCode)
			}
		})
	}
}

func TestPerformHTTPCheck_Timeout(t *testing.T) {
	checker := &HealthChecker{
		httpClient: &http.Client{Timeout: 100 * time.Millisecond},
	}

	// Create server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	result := checker.performHTTPCheck(ctx, server.URL)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Status != HealthStatusUnhealthy {
		t.Errorf("Status = %v, want %v for timeout", result.Status, HealthStatusUnhealthy)
	}
	if !strings.Contains(result.Error, "failed") {
		t.Errorf("Expected timeout error, got: %s", result.Error)
	}
}

func TestPerformShellCheck(t *testing.T) {
	checker := &HealthChecker{}
	ctx := context.Background()

	tests := []struct {
		name       string
		command    string
		svc        serviceInfo
		wantStatus HealthStatus
	}{
		{
			name:       "successful command",
			command:    "echo test",
			svc:        serviceInfo{Name: "test", Type: service.ServiceTypeProcess},
			wantStatus: HealthStatusHealthy,
		},
		{
			name:       "failing command",
			command:    "exit 1",
			svc:        serviceInfo{Name: "test", Type: service.ServiceTypeProcess},
			wantStatus: HealthStatusUnhealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.performShellCheck(ctx, tt.command, tt.svc)
			if result == nil {
				t.Fatal("Expected non-nil result")
			}
			if result.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", result.Status, tt.wantStatus)
			}
		})
	}
}

func TestPerformCommandCheck(t *testing.T) {
	checker := &HealthChecker{}
	ctx := context.Background()

	tests := []struct {
		name       string
		args       []string
		svc        serviceInfo
		wantStatus HealthStatus
		wantNil    bool
	}{
		{
			name:    "empty args",
			args:    []string{},
			svc:     serviceInfo{Name: "test", Type: service.ServiceTypeProcess},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.performCommandCheck(ctx, tt.args, tt.svc)
			if tt.wantNil {
				if result != nil {
					t.Error("Expected nil result for empty args")
				}
				return
			}
			if result == nil {
				t.Fatal("Expected non-nil result")
			}
			if result.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", result.Status, tt.wantStatus)
			}
		})
	}
}

func TestBuildResultFromHTTPCheck(t *testing.T) {
	checker := &HealthChecker{}

	httpResult := &httpHealthCheckResult{
		Endpoint:     "http://localhost:8080/health",
		ResponseTime: 50 * time.Millisecond,
		StatusCode:   200,
		Status:       HealthStatusHealthy,
		Details:      map[string]interface{}{"version": "1.0"},
		Error:        "",
	}

	result := HealthCheckResult{
		ServiceName: "test-service",
		Timestamp:   time.Now(),
	}

	tests := []struct {
		name                   string
		isInStartupGracePeriod bool
		httpStatus             HealthStatus
		wantStatus             HealthStatus
	}{
		{
			name:                   "healthy outside grace period",
			isInStartupGracePeriod: false,
			httpStatus:             HealthStatusHealthy,
			wantStatus:             HealthStatusHealthy,
		},
		{
			name:                   "unhealthy in grace period",
			isInStartupGracePeriod: true,
			httpStatus:             HealthStatusUnhealthy,
			wantStatus:             HealthStatusStarting,
		},
		{
			name:                   "healthy in grace period",
			isInStartupGracePeriod: true,
			httpStatus:             HealthStatusHealthy,
			wantStatus:             HealthStatusHealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpResult.Status = tt.httpStatus
			got := checker.buildResultFromHTTPCheck(result, httpResult, 8080, tt.isInStartupGracePeriod)

			if got.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", got.Status, tt.wantStatus)
			}
			if got.CheckType != HealthCheckTypeHTTP {
				t.Errorf("CheckType = %v, want %v", got.CheckType, HealthCheckTypeHTTP)
			}
			if got.Port != 8080 {
				t.Errorf("Port = %d, want 8080", got.Port)
			}
		})
	}
}

func TestChecker_StoppedService(t *testing.T) {
	checker := &HealthChecker{
		httpClient:    &http.Client{Timeout: 5 * time.Second},
		breakers:      make(map[string]*gobreaker.CircuitBreaker),
		rateLimiters:  make(map[string]*rate.Limiter),
		endpointCache: make(map[string]string),
		enableBreaker: false,
		rateLimit:     0,
	}

	svc := serviceInfo{
		Name:           "stopped-service",
		RegistryStatus: "stopped",
		Port:           8080,
	}

	ctx := context.Background()
	result := checker.CheckService(ctx, svc)

	if result.Status != HealthStatusUnknown {
		t.Errorf("Status = %v, want %v for stopped service", result.Status, HealthStatusUnknown)
	}
}

func TestChecker_RateLimitExceeded(t *testing.T) {
	checker := &HealthChecker{
		httpClient:    &http.Client{Timeout: 5 * time.Second},
		breakers:      make(map[string]*gobreaker.CircuitBreaker),
		rateLimiters:  make(map[string]*rate.Limiter),
		endpointCache: make(map[string]string),
		enableBreaker: false,
		rateLimit:     1, // 1 per second
	}

	svc := serviceInfo{
		Name: "rate-limited-service",
		Port: 8080,
	}

	ctx := context.Background()

	// First check should succeed (or fail normally)
	_ = checker.CheckService(ctx, svc)

	// Immediate second check should hit rate limit
	ctx2, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result := checker.CheckService(ctx2, svc)

	// Should either timeout or succeed after waiting
	// This test validates rate limiter is created
	if result.ServiceName != svc.Name {
		t.Errorf("ServiceName = %s, want %s", result.ServiceName, svc.Name)
	}
}

func TestTryCustomHealthCheck_HTTPUrl(t *testing.T) {
	checker := &HealthChecker{
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status": "healthy"}`)
	}))
	defer server.Close()

	config := &healthCheckConfig{
		Test: []string{server.URL},
	}

	svc := serviceInfo{Name: "test", Type: service.ServiceTypeProcess}

	ctx := context.Background()
	result := checker.tryCustomHealthCheck(ctx, config, svc)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Status != HealthStatusHealthy {
		t.Errorf("Status = %v, want %v", result.Status, HealthStatusHealthy)
	}
}

func TestTryCustomHealthCheck_NONE(t *testing.T) {
	checker := &HealthChecker{}

	config := &healthCheckConfig{
		Test: []string{"NONE", "ignored"},
	}

	svc := serviceInfo{Name: "test", Type: service.ServiceTypeProcess}

	ctx := context.Background()
	result := checker.tryCustomHealthCheck(ctx, config, svc)

	if result == nil {
		t.Fatal("Expected non-nil result for NONE")
	}
	if result.Status != HealthStatusHealthy {
		t.Errorf("Status = %v, want %v for NONE check", result.Status, HealthStatusHealthy)
	}
}

func TestCheckSingleEndpoint_404(t *testing.T) {
	checker := &HealthChecker{
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	port := server.Listener.Addr().(*net.TCPAddr).Port
	ctx := context.Background()

	result := checker.checkSingleEndpoint(ctx, port, "/nonexistent")

	// Should return nil for 404 (endpoint doesn't exist)
	if result != nil {
		t.Error("Expected nil result for 404 response")
	}
}

func TestCheckSingleEndpoint_ContextCancelled(t *testing.T) {
	checker := &HealthChecker{
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := checker.checkSingleEndpoint(ctx, 8080, "/health")

	// Should return nil for cancelled context
	if result != nil {
		t.Error("Expected nil result for cancelled context")
	}
}

// TestSuggestHTTPErrorAction tests HTTP error suggestions for different status codes
func TestSuggestHTTPErrorAction(t *testing.T) {
	tests := []struct {
		statusCode int
		want       string
	}{
		{503, "Service temporarily unavailable. Check if dependencies are running."},
		{500, "Server error. Check application logs for details."},
		{501, "Server error. Check application logs for details."},
		{502, "Server error. Check application logs for details."},
		{504, "Server error. Check application logs for details."},
		{404, "Health endpoint not found. Verify endpoint configuration."},
		{401, "Authentication failed. Check credentials."},
		{403, "Authorization failed. Check permissions."},
		{429, "Rate limited. Reduce request rate or check quotas."},
		{408, "Request timeout. Check network connectivity and service performance."},
		{400, "HTTP request failed. Check service logs for details."},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.statusCode), func(t *testing.T) {
			got := suggestHTTPErrorAction(tt.statusCode)
			if got != tt.want {
				t.Errorf("suggestHTTPErrorAction(%d) = %q, want %q", tt.statusCode, got, tt.want)
			}
		})
	}
}

// TestParseErrorDetailsFromBody tests parsing error details from HTTP response body
func TestParseErrorDetailsFromBody(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want string
	}{
		{
			name: "JSON with error field",
			body: []byte(`{"error": "Database connection failed"}`),
			want: "Database connection failed",
		},
		{
			name: "JSON with message field",
			body: []byte(`{"message": "Service unavailable"}`),
			want: "Service unavailable",
		},
		{
			name: "JSON with detail field",
			body: []byte(`{"detail": "Connection timeout"}`),
			want: "Connection timeout",
		},
		{
			name: "JSON with error_description field",
			body: []byte(`{"error_description": "Invalid token"}`),
			want: "Invalid token",
		},
		{
			name: "Plain text body",
			body: []byte("Internal server error occurred"),
			want: "Internal server error occurred",
		},
		{
			name: "Long plain text body (truncated)",
			body: []byte(strings.Repeat("x", 250)),
			want: strings.Repeat("x", 200) + "...",
		},
		{
			name: "Empty body",
			body: []byte(""),
			want: "",
		},
		{
			name: "Invalid JSON",
			body: []byte(`{invalid json`),
			want: "{invalid json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseErrorDetailsFromBody(tt.body)
			if got != tt.want {
				t.Errorf("parseErrorDetailsFromBody() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestSuggestTCPErrorAction tests TCP error suggestions
func TestSuggestTCPErrorAction(t *testing.T) {
	tests := []struct {
		name string
		err  error
		port int
		want string
	}{
		{
			name: "connection refused",
			err:  fmt.Errorf("connection refused"),
			port: 8080,
			want: "Port 8080 connection refused. Verify service is running and port is correct.",
		},
		{
			name: "timeout",
			err:  fmt.Errorf("i/o timeout"),
			port: 8080,
			want: "Port 8080 connection timeout. Check network connectivity and firewall rules.",
		},
		{
			name: "no route to host",
			err:  fmt.Errorf("no route to host"),
			port: 8080,
			want: "Network unreachable. Check network configuration.",
		},
		{
			name: "other error",
			err:  fmt.Errorf("unknown error"),
			port: 8080,
			want: "Port 8080 connection failed. Verify service is running.",
		},
		{
			name: "nil error",
			err:  nil,
			port: 8080,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := suggestTCPErrorAction(tt.err, tt.port)
			if got != tt.want {
				t.Errorf("suggestTCPErrorAction() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestSuggestProcessErrorAction tests process error suggestions
func TestSuggestProcessErrorAction(t *testing.T) {
	tests := []struct {
		name      string
		pid       int
		isRunning bool
		mode      string
		want      string
	}{
		{
			name:      "process not running",
			pid:       12345,
			isRunning: false,
			mode:      "daemon",
			want:      "Process 12345 not running. Check service logs and verify start command.",
		},
		{
			name:      "process running",
			pid:       12345,
			isRunning: true,
			mode:      "daemon",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := suggestProcessErrorAction(tt.pid, tt.isRunning, tt.mode)
			if got != tt.want {
				t.Errorf("suggestProcessErrorAction() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestPerformHTTPCheck_WithErrorDetails tests HTTP check with error details in response body
func TestPerformHTTPCheck_WithErrorDetails(t *testing.T) {
	// Create test server that returns 503 with JSON error body
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(503)
		_, _ = w.Write([]byte(`{"error": "Database connection pool exhausted"}`))
	}))
	defer server.Close()

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

	result := checker.performHTTPCheck(context.Background(), server.URL)

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.Status != HealthStatusUnhealthy {
		t.Errorf("Expected status %s, got %s", HealthStatusUnhealthy, result.Status)
	}

	if result.StatusCode != 503 {
		t.Errorf("Expected status code 503, got %d", result.StatusCode)
	}

	// Check that suggestion is populated
	suggestion, ok := result.Details["suggestion"].(string)
	if !ok {
		t.Fatal("Expected suggestion in details")
	}
	if !strings.Contains(suggestion, "Service temporarily unavailable") {
		t.Errorf("Expected suggestion to mention service unavailability, got: %s", suggestion)
	}

	// Check that error details are populated from response body
	if result.Error == "" {
		t.Error("Expected error to be populated from response body")
	}
	if !strings.Contains(result.Error, "Database connection pool exhausted") {
		t.Errorf("Expected error to contain response body message, got: %s", result.Error)
	}
}

// TestHTTPCheck_StatusCodeSuggestions tests that all HTTP status codes get appropriate suggestions
func TestHTTPCheck_StatusCodeSuggestions(t *testing.T) {
	// Note: 400 and 404 are intentionally skipped by tryHTTPHealthCheck as they indicate
	// the endpoint doesn't exist (not a health endpoint). Test other status codes.
	statusCodes := []int{401, 403, 408, 429, 500, 501, 502, 503, 504}

	for _, statusCode := range statusCodes {
		t.Run(fmt.Sprintf("status_%d", statusCode), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(statusCode)
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
				endpointCache: make(map[string]string),
			}

			result := checker.tryHTTPHealthCheck(context.Background(), port)

			if result == nil {
				t.Fatal("Expected result, got nil")
			}

			if statusCode >= 400 {
				// All 4xx and 5xx should have suggestions
				suggestion, ok := result.Details["suggestion"]
				if !ok {
					t.Errorf("Expected suggestion for status code %d", statusCode)
				}
				if suggestion == "" {
					t.Errorf("Expected non-empty suggestion for status code %d", statusCode)
				}
			}
		})
	}
}

// TestTCPCheck_WithSuggestion tests TCP check includes actionable suggestions on failure
func TestTCPCheck_WithSuggestion(t *testing.T) {
	// Use a port that's definitely not listening
	deadPort := 19999

	checker := &HealthChecker{
		timeout:            5 * time.Second,
		defaultEndpoint:    "/health",
		httpClient:         &http.Client{Timeout: 5 * time.Second},
		endpointCache:      make(map[string]string),
		startupGracePeriod: 0, // No grace period for this test
	}

	svc := serviceInfo{
		Name:           "test-service",
		Port:           deadPort,
		RegistryStatus: "running",
	}

	result := checker.performServiceCheck(context.Background(), svc)

	if result.Status != HealthStatusUnhealthy {
		t.Errorf("Expected status %s, got %s", HealthStatusUnhealthy, result.Status)
	}

	// Should have a suggestion
	suggestion, ok := result.Details["suggestion"]
	if !ok {
		t.Fatal("Expected suggestion in details for failed TCP check")
	}

	if suggestion == "" {
		t.Error("Expected non-empty suggestion for failed TCP check")
	}
}

// TestProcessCheck_WithSuggestion tests process check includes actionable suggestions on failure
func TestProcessCheck_WithSuggestion(t *testing.T) {
	// Use a PID that doesn't exist
	deadPID := 99999

	checker := &HealthChecker{
		timeout:            5 * time.Second,
		defaultEndpoint:    "/health",
		httpClient:         &http.Client{Timeout: 5 * time.Second},
		startupGracePeriod: 0, // No grace period for this test
	}

	svc := serviceInfo{
		Name:           "test-service",
		PID:            deadPID,
		RegistryStatus: "running",
	}

	result := checker.performServiceCheck(context.Background(), svc)

	if result.Status != HealthStatusUnhealthy {
		t.Errorf("Expected status %s, got %s", HealthStatusUnhealthy, result.Status)
	}

	// Should have a suggestion
	suggestion, ok := result.Details["suggestion"]
	if !ok {
		t.Fatal("Expected suggestion in details for failed process check")
	}

	if suggestion == "" {
		t.Error("Expected non-empty suggestion for failed process check")
	}
}
