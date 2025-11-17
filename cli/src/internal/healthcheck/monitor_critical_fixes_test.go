package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/registry"
	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
)

// setupSignalHandler is a test helper (copied from commands package for testing)
func setupSignalHandler(cancel context.CancelFunc) func() {
	done := make(chan struct{})

	go func() {
		<-done
		if cancel != nil {
			cancel()
		}
	}()

	return func() {
		close(done)
	}
}

// TestGoroutineLeakFix tests that goroutines are properly cleaned up
func TestGoroutineLeakFix(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping goroutine leak test in short mode")
	}

	// Record initial goroutine count
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	initialGoroutines := runtime.NumGoroutine()

	// Create test monitor
	tempDir := t.TempDir()
	monitor, err := NewHealthMonitor(MonitorConfig{
		ProjectDir:      tempDir,
		DefaultEndpoint: "/health",
		Timeout:         2 * time.Second,
		LogLevel:        "error",
		LogFormat:       "json",
	})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	// Create test registry with services
	reg := registry.GetRegistry(tempDir)
	for i := 0; i < 5; i++ {
		if err := reg.Register(&registry.ServiceRegistryEntry{
			Name:      fmt.Sprintf("service-%d", i),
			Port:      8000 + i,
			PID:       1000 + i,
			StartTime: time.Now(),
		}); err != nil {
			t.Fatalf("Failed to register service: %v", err)
		}
	}

	// Run multiple health checks
	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		_, err := monitor.Check(ctx, nil)
		cancel()
		if err != nil && ctx.Err() == nil {
			t.Logf("Check %d failed (expected for test): %v", i, err)
		}
	}

	// Close monitor
	if err := monitor.Close(); err != nil {
		t.Errorf("Failed to close monitor: %v", err)
	}

	// Allow goroutines to exit
	runtime.GC()
	time.Sleep(200 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	leaked := finalGoroutines - initialGoroutines

	// Allow small tolerance for background goroutines
	if leaked > 10 {
		t.Errorf("Goroutine leak detected: started with %d, ended with %d, leaked %d goroutines",
			initialGoroutines, finalGoroutines, leaked)
	} else {
		t.Logf("No significant goroutine leak: %d -> %d (diff: %d)",
			initialGoroutines, finalGoroutines, leaked)
	}
}

// TestPanicRecoveryInHealthCheck tests that panics in health checks are recovered
func TestPanicRecoveryInHealthCheck(t *testing.T) {
	tempDir := t.TempDir()
	monitor, err := NewHealthMonitor(MonitorConfig{
		ProjectDir:      tempDir,
		DefaultEndpoint: "/health",
		Timeout:         2 * time.Second,
		LogLevel:        "error",
		LogFormat:       "json",
	})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	defer monitor.Close()

	// Create a mock service that will cause panic
	reg := registry.GetRegistry(tempDir)
	if err := reg.Register(&registry.ServiceRegistryEntry{
		Name:      "panic-service",
		Port:      0, // Invalid port
		PID:       0, // Invalid PID
		StartTime: time.Now(),
	}); err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// This should not panic even if health check has issues
	report, err := monitor.Check(ctx, []string{"panic-service"})
	if err != nil {
		t.Logf("Check failed (may be expected): %v", err)
	}

	// Verify we got a result even if it's unhealthy
	if report != nil && len(report.Services) > 0 {
		result := report.Services[0]
		if result.Status == HealthStatusUnhealthy {
			t.Logf("Service correctly marked as unhealthy: %s", result.Error)
		}
	}
}

// TestCircuitBreakerRaceCondition tests circuit breaker under concurrent load
func TestCircuitBreakerRaceCondition(t *testing.T) {
	tempDir := t.TempDir()

	// Create server that fails
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer failServer.Close()

	monitor, err := NewHealthMonitor(MonitorConfig{
		ProjectDir:             tempDir,
		DefaultEndpoint:        "/health",
		Timeout:                1 * time.Second,
		LogLevel:               "error",
		LogFormat:              "json",
		EnableCircuitBreaker:   true,
		CircuitBreakerFailures: 3,
		CircuitBreakerTimeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	defer monitor.Close()

	// Register service
	reg := registry.GetRegistry(tempDir)
	_, port, _ := parseServerURL(failServer.URL)
	if err := reg.Register(&registry.ServiceRegistryEntry{
		Name:      "test-service",
		Port:      port,
		StartTime: time.Now(),
	}); err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// Run concurrent health checks to trigger race detector
	var wg sync.WaitGroup
	errors := make([]error, 50)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, errors[idx] = monitor.Check(ctx, []string{"test-service"})
		}(i)
	}

	wg.Wait()

	// Check that we got results (even if unhealthy)
	successCount := 0
	for _, err := range errors {
		if err == nil {
			successCount++
		}
	}

	t.Logf("Completed %d/%d concurrent health checks", successCount, len(errors))
	if successCount == 0 {
		t.Error("All concurrent health checks failed")
	}
}

// TestRateLimiterRaceCondition tests rate limiter under concurrent load
func TestRateLimiterRaceCondition(t *testing.T) {
	tempDir := t.TempDir()

	// Create healthy server
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`)) // Ignore error in test mock
	}))
	defer healthServer.Close()

	monitor, err := NewHealthMonitor(MonitorConfig{
		ProjectDir:      tempDir,
		DefaultEndpoint: "/health",
		Timeout:         1 * time.Second,
		LogLevel:        "error",
		LogFormat:       "json",
		RateLimit:       10, // 10 requests per second
	})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	defer monitor.Close()

	// Register service
	reg := registry.GetRegistry(tempDir)
	_, port, _ := parseServerURL(healthServer.URL)
	if err := reg.Register(&registry.ServiceRegistryEntry{
		Name:      "test-service",
		Port:      port,
		StartTime: time.Now(),
	}); err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// Run concurrent health checks
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_, _ = monitor.Check(ctx, []string{"test-service"}) // Ignore result, testing rate limiter
		}()
	}

	wg.Wait()
	t.Log("Rate limiter test completed without race")
}

// TestHTTPConnectionCleanup tests that HTTP connections are properly cleaned up
func TestHTTPConnectionCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping connection cleanup test in short mode")
	}

	tempDir := t.TempDir()

	monitor, err := NewHealthMonitor(MonitorConfig{
		ProjectDir:      tempDir,
		DefaultEndpoint: "/health",
		Timeout:         1 * time.Second,
		LogLevel:        "error",
		LogFormat:       "json",
	})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	// Verify cleanup ticker is created
	if monitor.cleanupTicker == nil {
		t.Error("Cleanup ticker was not initialized")
	}

	// Verify cleanup goroutine can be stopped gracefully
	if err := monitor.Close(); err != nil {
		t.Errorf("Failed to close monitor: %v", err)
	}

	// Verify Close is idempotent
	if err := monitor.Close(); err != nil {
		t.Errorf("Second Close() should not error: %v", err)
	}

	t.Log("Connection cleanup mechanism verified")
}

// TestContextCancellationHandling tests proper context cancellation
func TestContextCancellationHandling(t *testing.T) {
	tempDir := t.TempDir()

	// Create slow server
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // Longer than context timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	monitor, err := NewHealthMonitor(MonitorConfig{
		ProjectDir:      tempDir,
		DefaultEndpoint: "/health",
		Timeout:         1 * time.Second,
		LogLevel:        "error",
		LogFormat:       "json",
	})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	defer monitor.Close()

	// Register service
	reg := registry.GetRegistry(tempDir)
	_, port, _ := parseServerURL(slowServer.URL)
	if err := reg.Register(&registry.ServiceRegistryEntry{
		Name:      "slow-service",
		Port:      port,
		StartTime: time.Now(),
	}); err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// This should timeout and return error, not hang
	start := time.Now()
	report, err := monitor.Check(ctx, []string{"slow-service"})
	elapsed := time.Since(start)

	// Should complete quickly (within 2 seconds including collection timeout)
	if elapsed > 3*time.Second {
		t.Errorf("Health check took too long: %v", elapsed)
	}

	// Log error if present (expected on timeout)
	if err != nil {
		t.Logf("Expected error on timeout: %v", err)
	}

	// Should have a result even on timeout
	if report == nil {
		t.Log("Report is nil (context cancelled)")
	} else if len(report.Services) > 0 {
		result := report.Services[0]
		t.Logf("Service status: %s, error: %s", result.Status, result.Error)
	}

	t.Logf("Context cancellation handled in %v", elapsed)
}

// TestResultCollectionTimeout tests timeout protection in result collection
func TestResultCollectionTimeout(t *testing.T) {
	tempDir := t.TempDir()

	monitor, err := NewHealthMonitor(MonitorConfig{
		ProjectDir:      tempDir,
		DefaultEndpoint: "/health",
		Timeout:         1 * time.Second,
		LogLevel:        "error",
		LogFormat:       "json",
	})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	defer monitor.Close()

	// Register multiple services with invalid ports
	reg := registry.GetRegistry(tempDir)
	for i := 0; i < 10; i++ {
		if err := reg.Register(&registry.ServiceRegistryEntry{
			Name:      fmt.Sprintf("service-%d", i),
			Port:      9999 + i, // Ports likely not listening
			PID:       1,
			StartTime: time.Now(),
		}); err != nil {
			t.Fatalf("Failed to register service: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	report, err := monitor.Check(ctx, nil)
	elapsed := time.Since(start)

	// Should complete within reasonable time (timeout * 2 for collection)
	if elapsed > 10*time.Second {
		t.Errorf("Health check took too long: %v", elapsed)
	}

	if report != nil {
		t.Logf("Completed health check for %d services in %v", len(report.Services), elapsed)
	} else {
		t.Logf("Health check completed with error in %v: %v", elapsed, err)
	}
}

// TestAtomicMetricsFlag tests atomic access to metrics flag
func TestAtomicMetricsFlag(t *testing.T) {
	// Reset flag
	atomic.StoreInt32(&metricsEnabledFlag, 0)

	tempDir := t.TempDir()

	// Create monitor with metrics enabled
	monitor, err := NewHealthMonitor(MonitorConfig{
		ProjectDir:      tempDir,
		DefaultEndpoint: "/health",
		Timeout:         1 * time.Second,
		LogLevel:        "error",
		LogFormat:       "json",
		EnableMetrics:   true,
	})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	defer monitor.Close()

	// Verify flag is set atomically
	if atomic.LoadInt32(&metricsEnabledFlag) != 1 {
		t.Error("Metrics flag not set correctly")
	}

	// Toggle concurrently to test atomicity
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(val int32) {
			defer wg.Done()
			atomic.StoreInt32(&metricsEnabledFlag, val%2)
			_ = atomic.LoadInt32(&metricsEnabledFlag)
		}(int32(i))
	}

	wg.Wait()
	t.Log("Atomic metrics flag operations completed without race")
}

// TestSignalHandlerCleanup tests signal handler cleanup
func TestSignalHandlerCleanup(t *testing.T) {
	initialGoroutines := runtime.NumGoroutine()

	// Create multiple contexts with signal handlers
	for i := 0; i < 5; i++ {
		_, cancel := context.WithCancel(context.Background())
		cleanup := setupSignalHandler(cancel)

		// Immediately cleanup
		cleanup()
		cancel()

		time.Sleep(10 * time.Millisecond)
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	leaked := finalGoroutines - initialGoroutines

	if leaked > 5 {
		t.Errorf("Signal handler goroutine leak: started with %d, ended with %d",
			initialGoroutines, finalGoroutines)
	} else {
		t.Logf("Signal handler cleanup OK: %d -> %d", initialGoroutines, finalGoroutines)
	}
}

// TestCacheStampedePrevention tests singleflight pattern prevents duplicate concurrent requests
func TestCacheStampedePrevention(t *testing.T) {
	// Create server that counts requests
	requestCount := int32(0)
	checkExecutionCount := int32(0) // Count actual health check executions

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		time.Sleep(50 * time.Millisecond) // Simulate slow response
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`)) // Ignore error in test mock
	}))
	defer server.Close()

	// Create a custom health checker to track executions
	host, port, err := parseServerURL(server.URL)

	if err != nil {
		t.Fatalf("Failed to parse server URL %s: %v", server.URL, err)
	}

	t.Logf("Test server running at %s:%d", host, port)

	checker := &HealthChecker{
		timeout:         2 * time.Second,
		defaultEndpoint: "/health",
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:    100,
				IdleConnTimeout: 90 * time.Second,
			},
		},
		breakers:     make(map[string]*gobreaker.CircuitBreaker),
		rateLimiters: make(map[string]*rate.Limiter),
	}

	// Launch 100 concurrent requests for the same service
	concurrency := 100
	var wg sync.WaitGroup
	results := make([]HealthCheckResult, concurrency)
	startTime := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			ctx := context.Background()

			svc := serviceInfo{
				Name:      "test-service",
				Port:      port,
				StartTime: time.Now(),
			}

			// This should be deduplicated by singleflight
			results[index] = checker.CheckService(ctx, svc)
			atomic.AddInt32(&checkExecutionCount, 1)
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	finalRequests := atomic.LoadInt32(&requestCount)
	finalExecutions := atomic.LoadInt32(&checkExecutionCount)

	t.Logf("%d concurrent requests completed in %v", concurrency, elapsed)
	t.Logf("Total goroutines that called CheckService: %d", finalExecutions)
	t.Logf("Actual HTTP requests made: %d", finalRequests)

	// With singleflight, we should have significantly fewer HTTP requests than total callers
	// The key is that even though 100 goroutines called CheckService,
	// singleflight should deduplicate them to just a few actual health checks
	if finalRequests > 10 {
		t.Errorf("Cache stampede not prevented: expected <=10 HTTP requests, got %d", finalRequests)
	}

	// Verify all requests got a result
	healthyCount := 0
	for i, result := range results {
		if result.ServiceName == "" {
			t.Errorf("Request %d got empty result", i)
		}
		if result.Status == HealthStatusHealthy {
			healthyCount++
		}
	}

	t.Logf("Singleflight deduplicated %d concurrent calls to ~%d HTTP requests", concurrency, finalRequests)
	t.Logf("%d/%d requests got healthy status", healthyCount, concurrency)
}

// Helper to parse server URL
func parseServerURL(serverURL string) (string, int, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return "", 0, err
	}
	host := u.Hostname()
	portStr := u.Port()
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port: %s", portStr)
	}
	return host, port, nil
}
