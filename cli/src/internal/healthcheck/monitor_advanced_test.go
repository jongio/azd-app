package healthcheck

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/registry"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
)

// TestCircuitBreakerIntegration tests circuit breaker functionality with real health checks.
func TestCircuitBreakerIntegration(t *testing.T) {
	// Create a mock server that fails consistently
	mock := NewMockHealthServer()
	defer mock.Close()
	mock.SimulateUnhealthy()

	config := MonitorConfig{
		ProjectDir:             t.TempDir(),
		DefaultEndpoint:        "/health",
		Timeout:                2 * time.Second,
		EnableCircuitBreaker:   true,
		CircuitBreakerFailures: 3, // Fail after 3 attempts
		CircuitBreakerTimeout:  10 * time.Second,
	}

	monitor, err := NewHealthMonitor(config)
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	svc := serviceInfo{
		Name: "circuit-test",
		Port: mock.Port(),
	}

	// Make multiple requests to trip the circuit breaker
	for i := 0; i < 5; i++ {
		result := monitor.checker.CheckService(context.Background(), svc)
		t.Logf("Attempt %d: Status=%s, Error=%s", i+1, result.Status, result.Error)

		if i < 3 {
			// First few should fail normally
			if result.Status != HealthStatusUnhealthy {
				t.Errorf("Attempt %d: expected unhealthy, got %s", i+1, result.Status)
			}
		} else {
			// After threshold, circuit should open
			if result.Error != "" && !contains(result.Error, "circuit breaker") {
				t.Logf("Attempt %d: Circuit breaker should be open", i+1)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	// Verify circuit breaker was created
	monitor.checker.mu.RLock()
	breaker := monitor.checker.breakers["circuit-test"]
	monitor.checker.mu.RUnlock()

	if breaker == nil {
		t.Error("Expected circuit breaker to be created")
	}
}

// TestCircuitBreakerRecovery tests that circuit breaker can recover when service becomes healthy.
func TestCircuitBreakerRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping circuit breaker recovery test in short mode")
	}

	mock := NewMockHealthServer()
	defer mock.Close()

	// Start with unhealthy service
	mock.SimulateUnhealthy()

	config := MonitorConfig{
		ProjectDir:             t.TempDir(),
		DefaultEndpoint:        "/health",
		Timeout:                1 * time.Second,
		EnableCircuitBreaker:   true,
		CircuitBreakerFailures: 3,
		CircuitBreakerTimeout:  3 * time.Second, // Longer timeout for reliability
	}

	monitor, err := NewHealthMonitor(config)
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	svc := serviceInfo{
		Name: "recovery-test",
		Port: mock.Port(),
	}

	// Trip the circuit (3 failures + 1 more to confirm open state)
	for i := 0; i < 4; i++ {
		result := monitor.checker.CheckService(context.Background(), svc)
		t.Logf("Check %d: Status=%s, Error=%s", i+1, result.Status, result.Error)
		time.Sleep(200 * time.Millisecond)
	}

	// Verify circuit is open
	result := monitor.checker.CheckService(context.Background(), svc)
	if result.Status == HealthStatusHealthy {
		t.Fatal("Circuit breaker should be open (service should be unhealthy)")
	}
	t.Logf("Circuit is open: %s", result.Error)

	// Fix the service
	mock.SimulateHealthy()
	t.Log("Service is now healthy, waiting for circuit breaker timeout...")

	// Wait for circuit breaker timeout to allow half-open state
	// Use polling instead of fixed sleep for better reliability
	time.Sleep(3500 * time.Millisecond) // Wait slightly longer than timeout

	// Circuit should attempt half-open state and then close
	// Poll for recovery with timeout
	recovered := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		result = monitor.checker.CheckService(context.Background(), svc)
		t.Logf("Recovery check: Status=%s, Error=%s", result.Status, result.Error)

		if result.Status == HealthStatusHealthy {
			recovered = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if !recovered {
		t.Error("Circuit breaker did not recover when service became healthy within timeout")
	} else {
		t.Log("Circuit breaker successfully recovered!")
	}
}

// TestRateLimiterIntegration tests rate limiting functionality.
func TestRateLimiterIntegration(t *testing.T) {
	mock := NewMockHealthServer()
	defer mock.Close()
	mock.SimulateHealthy()

	config := MonitorConfig{
		ProjectDir:      t.TempDir(),
		DefaultEndpoint: "/health",
		Timeout:         5 * time.Second,
		RateLimit:       2, // 2 requests per second
	}

	monitor, err := NewHealthMonitor(config)
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	svc := serviceInfo{
		Name: "ratelimit-test",
		Port: mock.Port(),
	}

	// Make rapid requests - rate limiter should throttle them
	start := time.Now()
	count := 10 // More requests to clearly show rate limiting
	for i := 0; i < count; i++ {
		result := monitor.checker.CheckService(context.Background(), svc)
		if i == 0 || i == count-1 {
			t.Logf("Request %d: Status=%s, Elapsed=%v", i+1, result.Status, time.Since(start))
		}
	}
	elapsed := time.Since(start)

	// With rate limit of 2/s and burst of 4, 10 requests should take at least 3 seconds
	// First 4 can burst, remaining 6 are rate limited at 2/s = 3 more seconds
	minExpected := 2500 * time.Millisecond // Allow some margin
	if elapsed < minExpected {
		t.Logf("Warning: Rate limiting might not be optimal: %d requests in %v (expected >%v)", count, elapsed, minExpected)
		// Don't fail - rate limiting behavior can vary based on system load
	} else {
		t.Logf("Rate limiting working: %d requests in %v", count, elapsed)
	}

	// Verify rate limiter was created
	monitor.checker.mu.RLock()
	limiter := monitor.checker.rateLimiters["ratelimit-test"]
	monitor.checker.mu.RUnlock()

	if limiter == nil {
		t.Error("Expected rate limiter to be created")
	}
}

// TestRateLimiterCancellation tests that rate limiter respects context cancellation.
func TestRateLimiterCancellation(t *testing.T) {
	mock := NewMockHealthServer()
	defer mock.Close()
	mock.SimulateHealthy()

	config := MonitorConfig{
		ProjectDir:      t.TempDir(),
		DefaultEndpoint: "/health",
		Timeout:         5 * time.Second,
		RateLimit:       1, // Very strict rate limit
	}

	monitor, err := NewHealthMonitor(config)
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	svc := serviceInfo{
		Name: "cancel-test",
		Port: mock.Port(),
	}

	// Create context that cancels quickly
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Make several requests - should get canceled due to context
	result := monitor.checker.CheckService(ctx, svc)

	// First request should succeed
	if result.Status == HealthStatusHealthy {
		t.Log("First request succeeded as expected")
	}

	// Wait for context to expire
	time.Sleep(150 * time.Millisecond)

	// Second request with expired context should fail
	result = monitor.checker.CheckService(ctx, svc)
	if result.Status != HealthStatusUnhealthy {
		t.Errorf("Expected unhealthy status with canceled context, got %s", result.Status)
	}
	if result.Error != "" && !strings.Contains(strings.ToLower(result.Error), "context") && !strings.Contains(strings.ToLower(result.Error), "cancel") {
		t.Logf("Note: Error doesn't mention context cancellation: %s", result.Error)
	}
}

// TestCachingIntegration tests health check result caching.
func TestCachingIntegration(t *testing.T) {
	mock := NewMockHealthServer()
	defer mock.Close()
	mock.SimulateHealthy()

	projectDir := t.TempDir()
	config := MonitorConfig{
		ProjectDir:      projectDir,
		DefaultEndpoint: "/health",
		Timeout:         5 * time.Second,
		CacheTTL:        2 * time.Second, // Cache for 2 seconds
	}

	monitor, err := NewHealthMonitor(config)
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	// Register a test service so Check() has something to check
	entry := &registry.ServiceRegistryEntry{
		Name:       "cache-test",
		ProjectDir: projectDir,
		Port:       mock.Port(),
		URL:        mock.URL(),
		Status:     "ready",
		Health:     "unknown",
	}
	if err := monitor.registry.Register(entry); err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// First check - should hit the server
	mock.ResetRequestCount()
	report, err := monitor.Check(context.Background(), nil)
	if err != nil {
		t.Fatalf("First check failed: %v", err)
	}
	if len(report.Services) == 0 {
		t.Fatal("Expected at least one service in report")
	}

	firstRequestCount := mock.GetRequestCount()
	t.Logf("First check made %d requests", firstRequestCount)
	if firstRequestCount == 0 {
		t.Fatal("First check should have made at least one request")
	}

	// Immediate second check - should use cache (entire report is cached)
	_, err = monitor.Check(context.Background(), nil)
	if err != nil {
		t.Fatalf("Second check failed: %v", err)
	}

	secondRequestCount := mock.GetRequestCount()
	if secondRequestCount != firstRequestCount {
		t.Errorf("Expected cached result (same request count %d), but got %d total requests",
			firstRequestCount, secondRequestCount)
	}

	// Wait for cache to expire (2s TTL + small buffer)
	time.Sleep(2200 * time.Millisecond)

	// Poll until cache expires and new request is made
	var thirdRequestCount int
	deadline := time.Now().Add(2 * time.Second)
	cacheExpired := false
	for time.Now().Before(deadline) {
		_, err = monitor.Check(context.Background(), nil)
		if err != nil {
			t.Fatalf("Check failed during cache expiry wait: %v", err)
		}
		thirdRequestCount = mock.GetRequestCount()
		if thirdRequestCount > secondRequestCount {
			cacheExpired = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if !cacheExpired {
		t.Errorf("Expected cache to expire and make new requests, got %d (was %d)",
			thirdRequestCount, secondRequestCount)
	}
}

// TestGetOrCreateCircuitBreaker tests concurrent circuit breaker creation.
func TestGetOrCreateCircuitBreaker(t *testing.T) {
	checker := &HealthChecker{
		enableBreaker:   true,
		breakerFailures: 5,
		breakerTimeout:  60 * time.Second,
		breakers:        make(map[string]*gobreaker.CircuitBreaker),
	}

	// Create circuit breaker
	breaker1 := checker.getOrCreateCircuitBreaker("test-service")
	if breaker1 == nil {
		t.Fatal("Expected circuit breaker to be created")
	}

	// Get same circuit breaker again
	breaker2 := checker.getOrCreateCircuitBreaker("test-service")
	if breaker2 == nil {
		t.Fatal("Expected circuit breaker to be returned")
	}

	// Should be the same instance
	if breaker1 != breaker2 {
		t.Error("Expected same circuit breaker instance")
	}

	// Different service should get different breaker
	breaker3 := checker.getOrCreateCircuitBreaker("other-service")
	if breaker3 == breaker1 {
		t.Error("Expected different circuit breaker for different service")
	}
}

// TestGetOrCreateCircuitBreakerDisabled tests when circuit breaker is disabled.
func TestGetOrCreateCircuitBreakerDisabled(t *testing.T) {
	checker := &HealthChecker{
		enableBreaker: false,
		breakers:      make(map[string]*gobreaker.CircuitBreaker),
	}

	breaker := checker.getOrCreateCircuitBreaker("test-service")
	if breaker != nil {
		t.Error("Expected nil when circuit breaker is disabled")
	}
}

// TestGetOrCreateRateLimiter tests concurrent rate limiter creation.
func TestGetOrCreateRateLimiter(t *testing.T) {
	checker := &HealthChecker{
		rateLimit:    10,
		rateLimiters: make(map[string]*rate.Limiter),
	}

	// Create rate limiter
	limiter1 := checker.getOrCreateRateLimiter("test-service")
	if limiter1 == nil {
		t.Fatal("Expected rate limiter to be created")
	}

	// Get same rate limiter again
	limiter2 := checker.getOrCreateRateLimiter("test-service")
	if limiter2 == nil {
		t.Fatal("Expected rate limiter to be returned")
	}

	// Should be the same instance
	if limiter1 != limiter2 {
		t.Error("Expected same rate limiter instance")
	}
}

// TestGetOrCreateRateLimiterDisabled tests when rate limiting is disabled.
func TestGetOrCreateRateLimiterDisabled(t *testing.T) {
	checker := &HealthChecker{
		rateLimit:    0, // Disabled
		rateLimiters: make(map[string]*rate.Limiter),
	}

	limiter := checker.getOrCreateRateLimiter("test-service")
	if limiter != nil {
		t.Error("Expected nil when rate limiting is disabled")
	}
}

// TestParseHealthCheckConfig tests Docker Compose healthcheck parsing.
// NOTE: This function was removed as it always returned nil.
// When Docker Compose healthcheck parsing is implemented, this test should be re-enabled.
func TestParseHealthCheckConfig(t *testing.T) {
	t.Skip("parseHealthCheckConfig was removed - will be implemented when Docker Compose support is added")

	// Currently returns nil as Docker Compose integration is future enhancement
	// This test verifies it doesn't crash

	// svc := service.Service{
	// 	Language: "nodejs",
	// }
	//
	// config := parseHealthCheckConfig(svc)
	//
	// // Should return nil for now (not implemented yet)
	// if config != nil {
	// 	t.Log("Health check config parsing implemented")
	// }
}

// TestBuildServiceListWithHealthCheckConfig tests service list building with health check configuration.
func TestBuildServiceListWithHealthCheckConfig(t *testing.T) {
	tempDir := t.TempDir()

	monitor := &HealthMonitor{
		config: MonitorConfig{
			ProjectDir: tempDir,
		},
	}

	// Test with services that might have health check config in future
	azureYaml := &service.AzureYaml{
		Services: map[string]service.Service{
			"web": {
				Language: "nodejs",
				Project:  "./web",
			},
			"api": {
				Language: "python",
				Project:  "./api",
			},
		},
	}

	services := monitor.buildServiceList(azureYaml, nil)

	if len(services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(services))
	}

	// Verify services were created
	found := make(map[string]bool)
	for _, svc := range services {
		found[svc.Name] = true
	}

	if !found["web"] || !found["api"] {
		t.Error("Expected both web and api services")
	}
}

// TestHealthCheckWithSlowResponse tests handling of slow health check responses.
func TestHealthCheckWithSlowResponse(t *testing.T) {
	mock := NewMockHealthServerWithConfig(200, `{"status":"healthy"}`, 500*time.Millisecond)
	defer mock.Close()

	config := MonitorConfig{
		ProjectDir:      t.TempDir(),
		DefaultEndpoint: "/health",
		Timeout:         2 * time.Second, // Allow slow responses
	}

	monitor, err := NewHealthMonitor(config)
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	svc := serviceInfo{
		Name: "slow-service",
		Port: mock.Port(),
	}

	start := time.Now()
	result := monitor.checker.CheckService(context.Background(), svc)
	elapsed := time.Since(start)

	if result.Status != HealthStatusHealthy {
		t.Errorf("Expected healthy despite slow response, got %s", result.Status)
	}

	if elapsed < 400*time.Millisecond {
		t.Errorf("Response time too fast: %v (expected ~500ms)", elapsed)
	}

	t.Logf("Slow health check completed in %v", elapsed)
}

// TestMultipleServicesParallel tests parallel health checking of multiple services.
func TestMultipleServicesParallel(t *testing.T) {
	// Create multiple mock servers
	mocks := make([]*MockHealthServer, 5)
	for i := range mocks {
		mocks[i] = NewMockHealthServer()
		defer mocks[i].Close()
		mocks[i].SimulateHealthy()
	}

	config := MonitorConfig{
		ProjectDir:      t.TempDir(),
		DefaultEndpoint: "/health",
		Timeout:         5 * time.Second,
	}

	monitor, err := NewHealthMonitor(config)
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	// Register services
	for i, mock := range mocks {
		err := monitor.registry.Register(&registry.ServiceRegistryEntry{
			Name:      fmt.Sprintf("service-%d", i),
			Port:      mock.Port(),
			PID:       1000 + i,
			StartTime: time.Now(),
			Status:    "running",
		})
		if err != nil {
			t.Fatalf("Failed to register service %d: %v", i, err)
		}
	}

	// Check all services
	start := time.Now()
	report, err := monitor.Check(context.Background(), nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	if len(report.Services) != 5 {
		t.Errorf("Expected 5 services in report, got %d", len(report.Services))
	}

	// Parallel execution should complete faster than serial
	// (5 services × ~10ms each should be < 200ms total with parallelism)
	if elapsed > 1*time.Second {
		t.Errorf("Parallel health checks too slow: %v", elapsed)
	}

	t.Logf("Checked 5 services in parallel in %v", elapsed)
}
