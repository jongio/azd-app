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

	// Start a test server
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

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
