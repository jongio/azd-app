package healthcheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/docker"
	"github.com/jongio/azd-app/cli/src/internal/procutil"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/rs/zerolog/log"
	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
)

// HealthChecker performs individual health checks with circuit breaker and rate limiting.
type HealthChecker struct {
	timeout            time.Duration
	defaultEndpoint    string
	httpClient         *http.Client
	breakers           map[string]*gobreaker.CircuitBreaker
	rateLimiters       map[string]*rate.Limiter
	endpointCache      map[string]string // Maps service:port to successful endpoint path
	mu                 sync.RWMutex
	enableBreaker      bool
	breakerFailures    int
	breakerTimeout     time.Duration
	rateLimit          int
	startupGracePeriod time.Duration
}

// getOrCreateCircuitBreaker gets or creates a circuit breaker for a service.
func (c *HealthChecker) getOrCreateCircuitBreaker(serviceName string) *gobreaker.CircuitBreaker {
	if !c.enableBreaker {
		return nil
	}

	c.mu.RLock()
	breaker, exists := c.breakers[serviceName]
	c.mu.RUnlock()

	if exists {
		return breaker
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if breaker, exists := c.breakers[serviceName]; exists {
		return breaker
	}

	// Create circuit breaker settings
	settings := gobreaker.Settings{
		Name:        serviceName,
		MaxRequests: 3, // Max requests in half-open state
		Interval:    c.breakerTimeout,
		Timeout:     c.breakerTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= uint32(c.breakerFailures) && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.Info().
				Str("service", name).
				Str("from", from.String()).
				Str("to", to.String()).
				Msg("Circuit breaker state changed")

			// Record state change in metrics
			if metricsEnabled.Load() {
				recordCircuitBreakerState(name, to)
			}
		},
	}

	breaker = gobreaker.NewCircuitBreaker(settings)
	c.breakers[serviceName] = breaker
	return breaker
}

// getOrCreateRateLimiter gets or creates a rate limiter for a service.
func (c *HealthChecker) getOrCreateRateLimiter(serviceName string) *rate.Limiter {
	if c.rateLimit <= 0 {
		return nil
	}

	c.mu.RLock()
	limiter, exists := c.rateLimiters[serviceName]
	c.mu.RUnlock()

	if exists {
		return limiter
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists := c.rateLimiters[serviceName]; exists {
		return limiter
	}

	// Create rate limiter with burst capacity
	limiter = rate.NewLimiter(rate.Limit(c.rateLimit), c.rateLimit*2)
	c.rateLimiters[serviceName] = limiter
	log.Debug().
		Str("service", serviceName).
		Int("rate_limit", c.rateLimit).
		Msg("Created rate limiter")

	return limiter
}

// CheckService performs a health check on a single service using cascading strategy.
func (c *HealthChecker) CheckService(ctx context.Context, svc serviceInfo) HealthCheckResult {
	startTime := time.Now()
	serviceName := svc.Name

	// Skip health checks for stopped services - they should remain in their stopped state
	// without being marked as unhealthy
	if svc.RegistryStatus == "stopped" {
		log.Debug().
			Str("service", serviceName).
			Msg("Skipping health check for stopped service")

		return HealthCheckResult{
			ServiceName:  serviceName,
			Timestamp:    time.Now(),
			Status:       HealthStatusUnknown,
			ResponseTime: time.Since(startTime),
			ServiceType:  svc.Type,
			ServiceMode:  svc.Mode,
		}
	}

	log.Debug().
		Str("service", serviceName).
		Int("port", svc.Port).
		Int("pid", svc.PID).
		Msg("Starting health check")

	// Apply rate limiting if configured
	limiter := c.getOrCreateRateLimiter(serviceName)
	if limiter != nil {
		if err := limiter.Wait(ctx); err != nil {
			log.Warn().
				Str("service", serviceName).
				Err(err).
				Msg("Rate limit exceeded")

			return HealthCheckResult{
				ServiceName: serviceName,
				Timestamp:   time.Now(),
				Status:      HealthStatusUnhealthy,
				Error:       "rate limit exceeded",
			}
		}
	}

	// Get circuit breaker if enabled
	breaker := c.getOrCreateCircuitBreaker(serviceName)

	// Perform check with circuit breaker wrapping if enabled
	var result HealthCheckResult

	if breaker != nil {
		// Add panic recovery for circuit breaker operations
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error().
						Str("service", serviceName).
						Interface("panic", r).
						Msg("Panic recovered during circuit breaker operation")
					result = HealthCheckResult{
						ServiceName: serviceName,
						Timestamp:   time.Now(),
						Status:      HealthStatusUnknown,
						Error:       fmt.Sprintf("internal error: panic during health check: %v", r),
					}
				}
			}()

			output, err := breaker.Execute(func() (interface{}, error) {
				res := c.performServiceCheck(ctx, svc)
				if res.Status == HealthStatusUnhealthy {
					return res, fmt.Errorf("health check failed: %s", res.Error)
				}
				return res, nil
			})

			if err != nil {
				if errors.Is(err, gobreaker.ErrOpenState) {
					log.Warn().
						Str("service", serviceName).
						Msg("Circuit breaker open - skipping check")

					result = HealthCheckResult{
						ServiceName: serviceName,
						Timestamp:   time.Now(),
						Status:      HealthStatusUnhealthy,
						Error:       "circuit breaker open - service unavailable",
					}
				} else {
					// Health check failed
					result = HealthCheckResult{
						ServiceName: serviceName,
						Timestamp:   time.Now(),
						Status:      HealthStatusUnhealthy,
						Error:       err.Error(),
					}
				}
			} else {
				// Safe type assertion with ok-check to prevent panic
				if typedResult, ok := output.(HealthCheckResult); ok {
					result = typedResult
				} else {
					// Unexpected type returned from circuit breaker - should never happen
					log.Error().
						Str("service", serviceName).
						Str("type", fmt.Sprintf("%T", output)).
						Msg("Circuit breaker returned unexpected type")
					result = HealthCheckResult{
						ServiceName: serviceName,
						Timestamp:   time.Now(),
						Status:      HealthStatusUnknown,
						Error:       "internal error: unexpected health check result type",
					}
				}
			}
		}()
	} else {
		// No circuit breaker - perform check directly
		result = c.performServiceCheck(ctx, svc)
	}

	// Record metrics if enabled
	duration := time.Since(startTime)
	result.ResponseTime = duration

	if metricsEnabled.Load() {
		recordHealthCheck(result)
	}

	// Include service type and mode in result
	result.ServiceType = svc.Type
	result.ServiceMode = svc.Mode

	log.Debug().
		Str("service", serviceName).
		Str("status", string(result.Status)).
		Dur("duration", duration).
		Msg("Health check completed")

	return result
}

// performServiceCheck executes the actual health check logic without circuit breaker.
func (c *HealthChecker) performServiceCheck(ctx context.Context, svc serviceInfo) HealthCheckResult {
	result := HealthCheckResult{
		ServiceName: svc.Name,
		Timestamp:   time.Now(),
	}

	// Calculate uptime if we have start time
	if !svc.StartTime.IsZero() {
		result.Uptime = time.Since(svc.StartTime)
	}

	// Startup grace period: If the service has been running for less than the configured grace period,
	// keep it in "starting" state unless health checks pass. This prevents services
	// from showing as "unhealthy" during normal startup.
	gracePeriod := c.startupGracePeriod
	if gracePeriod == 0 {
		gracePeriod = startupGracePeriod // Fallback to default
	}
	isInStartupGracePeriod := !svc.StartTime.IsZero() &&
		time.Since(svc.StartTime) < gracePeriod

	// For process-type services, use process-based health checks directly
	// Skip HTTP/port checks since they have no network endpoint
	if svc.Type == service.ServiceTypeProcess {
		return c.performProcessHealthCheck(ctx, svc, isInStartupGracePeriod)
	}

	// Check for custom healthcheck config first
	if svc.HealthCheck != nil && len(svc.HealthCheck.Test) > 0 {
		if httpResult := c.tryCustomHealthCheck(ctx, svc.HealthCheck, svc); httpResult != nil {
			return c.buildResultFromHTTPCheck(result, httpResult, svc.Port, isInStartupGracePeriod)
		}
	}

	// Cascading strategy: HTTP -> Port -> Process

	// 1. Try HTTP health check
	if svc.Port > 0 {
		if httpResult := c.tryHTTPHealthCheck(ctx, svc.Port); httpResult != nil {
			result.Port = svc.Port
			return c.buildResultFromHTTPCheck(result, httpResult, svc.Port, isInStartupGracePeriod)
		}
	}

	// 2. Fall back to TCP port check
	if svc.Port > 0 {
		result.CheckType = HealthCheckTypeTCP
		result.Port = svc.Port
		if c.checkPort(ctx, svc.Port) {
			result.Status = HealthStatusHealthy
		} else {
			if isInStartupGracePeriod {
				result.Status = HealthStatusStarting
			} else {
				result.Status = HealthStatusUnhealthy
			}
			result.Error = fmt.Sprintf("port %d not listening", svc.Port)
		}
		return result
	}

	// 3. Fall back to process check
	if svc.PID > 0 {
		result.CheckType = HealthCheckTypeProcess
		result.PID = svc.PID
		if isProcessRunning(svc.PID) {
			result.Status = HealthStatusHealthy
		} else {
			if isInStartupGracePeriod {
				result.Status = HealthStatusStarting
			} else {
				result.Status = HealthStatusUnhealthy
			}
			result.Error = fmt.Sprintf("process %d not running", svc.PID)
		}
		return result
	}

	// No check available
	result.CheckType = HealthCheckTypeProcess
	if isInStartupGracePeriod {
		result.Status = HealthStatusStarting
	} else {
		result.Status = HealthStatusUnknown
	}
	result.Error = "no health check method available"

	return result
}

// buildResultFromHTTPCheck builds a HealthCheckResult from an HTTP check result.
func (c *HealthChecker) buildResultFromHTTPCheck(result HealthCheckResult, httpResult *httpHealthCheckResult, port int, isInStartupGracePeriod bool) HealthCheckResult {
	result.CheckType = HealthCheckTypeHTTP
	result.Endpoint = httpResult.Endpoint
	result.ResponseTime = httpResult.ResponseTime
	result.StatusCode = httpResult.StatusCode
	result.Status = httpResult.Status
	result.Details = httpResult.Details
	result.Error = httpResult.Error
	if port > 0 {
		result.Port = port
	}
	// If check failed but we're in startup grace period, keep "starting" status
	if isInStartupGracePeriod && result.Status != HealthStatusHealthy {
		result.Status = HealthStatusStarting
	}
	return result
}

// tryCustomHealthCheck performs a health check using custom configuration from azure.yaml.
// For container services (svc.Type == "container"), CMD and CMD-SHELL health checks
// are executed inside the container using docker exec.
func (c *HealthChecker) tryCustomHealthCheck(ctx context.Context, config *healthCheckConfig, svc serviceInfo) *httpHealthCheckResult {
	if len(config.Test) == 0 {
		return nil
	}

	test := config.Test[0]

	// Check if it's an HTTP URL (cross-platform approach)
	if strings.HasPrefix(test, "http://") || strings.HasPrefix(test, "https://") {
		return c.performHTTPCheck(ctx, test)
	}

	// Check for CMD or CMD-SHELL format
	if len(config.Test) > 1 {
		switch config.Test[0] {
		case "CMD":
			return c.performCommandCheck(ctx, config.Test[1:], svc)
		case "CMD-SHELL":
			return c.performShellCheck(ctx, config.Test[1], svc)
		case "NONE":
			return &httpHealthCheckResult{
				Endpoint: "none",
				Status:   HealthStatusHealthy,
			}
		}
	}

	// Single string that's not a URL - treat as shell command
	return c.performShellCheck(ctx, test, svc)
}

// performHTTPCheck performs a direct HTTP health check to a specific URL.
func (c *HealthChecker) performHTTPCheck(ctx context.Context, urlStr string) *httpHealthCheckResult {
	startTime := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return &httpHealthCheckResult{
			Endpoint: urlStr,
			Status:   HealthStatusUnhealthy,
			Error:    fmt.Sprintf("failed to create request: %v", err),
		}
	}

	resp, err := c.httpClient.Do(req)
	responseTime := time.Since(startTime)

	if err != nil {
		return &httpHealthCheckResult{
			Endpoint:     urlStr,
			ResponseTime: responseTime,
			Status:       HealthStatusUnhealthy,
			Error:        fmt.Sprintf("connection failed: %v", err),
		}
	}

	// Read and close body
	limitedReader := io.LimitReader(resp.Body, maxResponseBodySize)
	body, readErr := io.ReadAll(limitedReader)
	if closeErr := resp.Body.Close(); closeErr != nil {
		log.Warn().Err(closeErr).Str("url", urlStr).Msg("Failed to close response body")
	}

	result := &httpHealthCheckResult{
		Endpoint:     urlStr,
		ResponseTime: responseTime,
		StatusCode:   resp.StatusCode,
	}

	// Determine status based on HTTP status code
	result.Status = c.statusFromHTTPCode(resp.StatusCode)

	// Try to parse response body for additional details
	if readErr == nil && len(body) > 0 && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		c.parseHealthResponseBody(body, result)
	}

	return result
}

// performCommandCheck executes a command for health check (CMD format).
// For container services, the command is executed inside the container using docker exec.
func (c *HealthChecker) performCommandCheck(ctx context.Context, args []string, svc serviceInfo) *httpHealthCheckResult {
	if len(args) == 0 {
		return nil
	}

	startTime := time.Now()
	result := &httpHealthCheckResult{
		Endpoint:     strings.Join(args, " "),
		ResponseTime: 0,
	}

	// For container services, execute inside the container
	if svc.Type == service.ServiceTypeContainer {
		containerName := fmt.Sprintf("azd-%s", svc.Name)
		client := docker.NewClient()

		exitCode, output, err := client.Exec(containerName, args)
		result.ResponseTime = time.Since(startTime)

		if err != nil {
			result.Status = HealthStatusUnhealthy
			result.Error = fmt.Sprintf("docker exec failed: %v", err)
		} else if exitCode != 0 {
			result.Status = HealthStatusUnhealthy
			result.Error = fmt.Sprintf("command exited with code %d: %s", exitCode, output)
		} else {
			result.Status = HealthStatusHealthy
		}
		return result
	}

	// For native services, execute on host
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	err := cmd.Run()
	result.ResponseTime = time.Since(startTime)

	if err != nil {
		result.Status = HealthStatusUnhealthy
		result.Error = fmt.Sprintf("command failed: %v", err)
	} else {
		result.Status = HealthStatusHealthy
	}

	return result
}

// performShellCheck executes a shell command for health check (CMD-SHELL format).
// For container services, the command is executed inside the container using docker exec sh -c.
func (c *HealthChecker) performShellCheck(ctx context.Context, command string, svc serviceInfo) *httpHealthCheckResult {
	startTime := time.Now()
	result := &httpHealthCheckResult{
		Endpoint:     command,
		ResponseTime: 0,
	}

	// For container services, execute inside the container
	if svc.Type == service.ServiceTypeContainer {
		containerName := fmt.Sprintf("azd-%s", svc.Name)
		client := docker.NewClient()

		exitCode, output, err := client.ExecShell(containerName, command)
		result.ResponseTime = time.Since(startTime)

		if err != nil {
			result.Status = HealthStatusUnhealthy
			result.Error = fmt.Sprintf("docker exec failed: %v", err)
		} else if exitCode != 0 {
			result.Status = HealthStatusUnhealthy
			result.Error = fmt.Sprintf("command exited with code %d: %s", exitCode, output)
		} else {
			result.Status = HealthStatusHealthy
		}
		return result
	}

	// For native services, execute on host
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	err := cmd.Run()
	result.ResponseTime = time.Since(startTime)

	if err != nil {
		result.Status = HealthStatusUnhealthy
		result.Error = fmt.Sprintf("command failed: %v", err)
	} else {
		result.Status = HealthStatusHealthy
	}

	return result
}

// tryHTTPHealthCheck attempts HTTP health checks using smart endpoint discovery.
// Uses endpoint caching to avoid spamming multiple endpoints on every check.
// Discovery only happens on first check or when cached endpoint fails.
func (c *HealthChecker) tryHTTPHealthCheck(ctx context.Context, port int) *httpHealthCheckResult {
	cacheKey := fmt.Sprintf("port:%d", port)

	// Ensure endpointCache is initialized (for backward compatibility with tests)
	c.mu.Lock()
	if c.endpointCache == nil {
		c.endpointCache = make(map[string]string)
	}
	c.mu.Unlock()

	// Check if we have a cached endpoint for this port
	c.mu.RLock()
	cachedEndpoint, hasCached := c.endpointCache[cacheKey]
	c.mu.RUnlock()

	// If we have a cached endpoint, ONLY check that endpoint first
	if hasCached {
		// Special marker indicates no HTTP endpoint exists - skip to TCP fallback
		if cachedEndpoint == endpointCacheNone {
			log.Debug().
				Int("port", port).
				Msg("Skipping HTTP check - no endpoint found in previous discovery")
			return nil
		}

		result := c.checkSingleEndpoint(ctx, port, cachedEndpoint)
		if result != nil && result.Status == HealthStatusHealthy {
			return result
		}
		// Cached endpoint failed - clear cache and rediscover
		c.mu.Lock()
		delete(c.endpointCache, cacheKey)
		c.mu.Unlock()
		log.Debug().
			Int("port", port).
			Str("cached_endpoint", cachedEndpoint).
			Msg("Cached health endpoint failed, will rediscover on next check")
		// Fall through to discovery - gives one chance to find a working endpoint
	}

	// No cached endpoint - perform endpoint discovery
	log.Debug().
		Int("port", port).
		Msg("Discovering health endpoint (first check or cache miss)")

	// Build list of endpoints to try, prioritizing common ones
	endpoints := []string{c.defaultEndpoint}
	for _, path := range commonHealthPaths {
		if path != c.defaultEndpoint {
			endpoints = append(endpoints, path)
		}
	}

	// Track the last non-nil result in case no healthy endpoint is found
	var lastResult *httpHealthCheckResult

	for _, endpoint := range endpoints {
		// Check context before each attempt
		if ctx.Err() != nil {
			return nil
		}

		result := c.checkSingleEndpoint(ctx, port, endpoint)
		if result != nil {
			// If healthy, cache and return immediately - stop discovery
			if result.Status == HealthStatusHealthy {
				c.mu.Lock()
				c.endpointCache[cacheKey] = endpoint
				c.mu.Unlock()
				log.Debug().
					Int("port", port).
					Str("endpoint", endpoint).
					Msg("Discovered and cached health endpoint")
				return result
			}
			// Keep track of last non-nil result for fallback
			lastResult = result
		}
	}

	// No healthy endpoint found during discovery
	// Cache a marker to skip HTTP checks in future (will fall back to TCP/process checks)
	if lastResult == nil {
		c.mu.Lock()
		c.endpointCache[cacheKey] = endpointCacheNone
		c.mu.Unlock()
		log.Debug().
			Int("port", port).
			Msg("No HTTP health endpoint found, will use TCP fallback")
	}

	return lastResult
}

// checkSingleEndpoint performs a single HTTP health check on a specific endpoint.
func (c *HealthChecker) checkSingleEndpoint(ctx context.Context, port int, endpoint string) *httpHealthCheckResult {
	url := fmt.Sprintf("http://localhost:%d%s", port, endpoint)

	startTime := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil
	}

	resp, err := c.httpClient.Do(req)
	responseTime := time.Since(startTime)

	if err != nil {
		// Check if error is due to context cancellation
		if ctx.Err() != nil {
			return nil
		}
		return nil
	}

	limitedReader := io.LimitReader(resp.Body, maxResponseBodySize)
	body, readErr := io.ReadAll(limitedReader)
	closeErr := resp.Body.Close()
	if closeErr != nil {
		log.Warn().Err(closeErr).Str("url", url).Msg("Failed to close response body")
	}

	// Skip 404 and 400 responses - these indicate endpoint doesn't exist
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest {
		return nil
	}

	result := &httpHealthCheckResult{
		Endpoint:     url,
		ResponseTime: responseTime,
		StatusCode:   resp.StatusCode,
		Status:       c.statusFromHTTPCode(resp.StatusCode),
	}

	if readErr == nil && len(body) > 0 && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		c.parseHealthResponseBody(body, result)
	}

	return result
}

// statusFromHTTPCode determines health status from HTTP status code.
func (c *HealthChecker) statusFromHTTPCode(statusCode int) HealthStatus {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return HealthStatusHealthy
	case statusCode >= 300 && statusCode < 400:
		return HealthStatusHealthy // Redirects OK
	case statusCode >= 500:
		return HealthStatusUnhealthy
	default:
		return HealthStatusDegraded
	}
}

// parseHealthResponseBody parses JSON response body for health details.
func (c *HealthChecker) parseHealthResponseBody(body []byte, result *httpHealthCheckResult) {
	var details map[string]interface{}
	if err := json.Unmarshal(body, &details); err == nil {
		result.Details = details

		if status, ok := details["status"].(string); ok {
			switch strings.ToLower(status) {
			case "healthy", "ok", "up":
				result.Status = HealthStatusHealthy
			case "degraded", "warning":
				result.Status = HealthStatusDegraded
			case "unhealthy", "down", "error":
				result.Status = HealthStatusUnhealthy
			}
		}
	}
}

// performProcessHealthCheck handles health checks for process-type services.
func (c *HealthChecker) performProcessHealthCheck(ctx context.Context, svc serviceInfo, isInStartupGracePeriod bool) HealthCheckResult {
	result := HealthCheckResult{
		ServiceName: svc.Name,
		Timestamp:   time.Now(),
		CheckType:   HealthCheckTypeProcess,
		ServiceMode: svc.Mode,
	}

	if !svc.StartTime.IsZero() {
		if !svc.EndTime.IsZero() {
			result.Uptime = svc.EndTime.Sub(svc.StartTime)
		} else {
			result.Uptime = time.Since(svc.StartTime)
		}
	}

	if svc.Mode == service.ServiceModeBuild || svc.Mode == service.ServiceModeTask {
		return c.performBuildTaskHealthCheck(svc, isInStartupGracePeriod, result)
	}

	if svc.HealthCheck != nil && svc.HealthCheck.Type == "output" && svc.HealthCheck.Pattern != "" {
		return c.performOutputHealthCheck(svc, isInStartupGracePeriod, result)
	}

	if svc.PID > 0 {
		result.PID = svc.PID
		if isProcessRunning(svc.PID) {
			result.Status = HealthStatusHealthy
		} else {
			if isInStartupGracePeriod {
				result.Status = HealthStatusStarting
			} else {
				result.Status = HealthStatusUnhealthy
			}
			result.Error = fmt.Sprintf("process %d not running", svc.PID)
		}
		return result
	}

	if isInStartupGracePeriod {
		result.Status = HealthStatusStarting
	} else {
		result.Status = HealthStatusUnknown
	}
	result.Error = "no process ID available for health check"

	return result
}

// performBuildTaskHealthCheck handles health checks for build and task mode services.
func (c *HealthChecker) performBuildTaskHealthCheck(svc serviceInfo, isInStartupGracePeriod bool, result HealthCheckResult) HealthCheckResult {
	result.PID = svc.PID

	if svc.PID > 0 && isProcessRunning(svc.PID) {
		if isInStartupGracePeriod {
			result.Status = HealthStatusStarting
		} else {
			result.Status = HealthStatusHealthy
		}
		if svc.Mode == service.ServiceModeBuild {
			result.Details = map[string]interface{}{"state": "building"}
		} else {
			result.Details = map[string]interface{}{"state": "running"}
		}
		return result
	}

	if svc.ExitCode != nil {
		if *svc.ExitCode == 0 {
			result.Status = HealthStatusHealthy
			if svc.Mode == service.ServiceModeBuild {
				result.Details = map[string]interface{}{"state": "built", "exitCode": 0}
			} else {
				result.Details = map[string]interface{}{"state": "completed", "exitCode": 0}
			}
		} else {
			result.Status = HealthStatusUnhealthy
			result.Error = fmt.Sprintf("process exited with code %d", *svc.ExitCode)
			result.Details = map[string]interface{}{"state": "failed", "exitCode": *svc.ExitCode}
		}
		return result
	}

	if svc.PID > 0 {
		result.Status = HealthStatusHealthy
		if svc.Mode == service.ServiceModeBuild {
			result.Details = map[string]interface{}{"state": "built", "note": "exit code not captured"}
		} else {
			result.Details = map[string]interface{}{"state": "completed", "note": "exit code not captured"}
		}
		return result
	}

	if isInStartupGracePeriod {
		result.Status = HealthStatusStarting
		return result
	}

	result.Status = HealthStatusUnknown
	result.Error = "no process information available"
	return result
}

// performOutputHealthCheck handles health checks for services using output pattern matching.
func (c *HealthChecker) performOutputHealthCheck(svc serviceInfo, isInStartupGracePeriod bool, result HealthCheckResult) HealthCheckResult {
	pattern := svc.HealthCheck.Pattern
	result.PID = svc.PID
	result.Details = map[string]interface{}{
		"checkType": "output",
		"pattern":   pattern,
	}

	if svc.PID > 0 && !isProcessRunning(svc.PID) {
		if svc.ExitCode != nil {
			if *svc.ExitCode == 0 {
				result.Status = HealthStatusHealthy
				result.Details["state"] = "completed"
				return result
			}
			result.Status = HealthStatusUnhealthy
			result.Error = fmt.Sprintf("process exited with code %d before pattern matched", *svc.ExitCode)
			result.Details["state"] = "failed"
			return result
		}
		if isInStartupGracePeriod {
			result.Status = HealthStatusStarting
		} else {
			result.Status = HealthStatusUnhealthy
			result.Error = "process not running"
		}
		return result
	}

	projectDir, _ := os.Getwd()
	logManager := service.GetLogManager(projectDir)
	buffer, exists := logManager.GetBuffer(svc.Name)

	if !exists {
		if isInStartupGracePeriod {
			result.Status = HealthStatusStarting
			result.Details["state"] = "waiting_for_logs"
		} else {
			result.Status = HealthStatusUnknown
			result.Error = "log buffer not available"
		}
		return result
	}

	if buffer.ContainsPattern(pattern) {
		result.Status = HealthStatusHealthy
		result.Details["state"] = "pattern_matched"
		return result
	}

	if isInStartupGracePeriod {
		result.Status = HealthStatusStarting
		result.Details["state"] = "waiting_for_pattern"
	} else {
		if svc.Mode == service.ServiceModeWatch {
			result.Status = HealthStatusHealthy
			result.Details["state"] = "watching"
		} else {
			result.Status = HealthStatusUnhealthy
			result.Error = fmt.Sprintf("pattern %q not found in output", pattern)
			result.Details["state"] = "pattern_not_matched"
		}
	}

	return result
}

// checkPort checks if a TCP port is listening.
func (c *HealthChecker) checkPort(ctx context.Context, port int) bool {
	address := fmt.Sprintf("localhost:%d", port)
	dialer := net.Dialer{Timeout: defaultPortCheckTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// isProcessRunning delegates to procutil.IsProcessRunning for cross-platform process detection.
func isProcessRunning(pid int) bool {
	return procutil.IsProcessRunning(pid)
}
