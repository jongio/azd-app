package healthcheck

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
)

const (
	statusError     = "error"
	statusCompleted = "completed"
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
			// Validate breakerFailures to prevent underflow/overflow
			if c.breakerFailures < 0 || int64(c.breakerFailures) > int64(^uint32(0)) {
				return false
			}
			if counts.Requests == 0 {
				return false
			}
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= uint32(c.breakerFailures) && failureRatio >= 0.6 //nolint:gosec // G115 - breakerFailures bounds checked above
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			slog.Info("Circuit breaker state changed", "service", name, "from", from.String(), "to", to.String())

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
	slog.Debug("Created rate limiter", "service", serviceName, "rate_limit", c.rateLimit)

	return limiter
}

// CheckService performs a health check on a single service using cascading strategy.
func (c *HealthChecker) CheckService(ctx context.Context, svc serviceInfo) HealthCheckResult {
	startTime := time.Now()
	serviceName := svc.Name

	// Skip health checks for stopped services - they should remain in their stopped state
	// without being marked as unhealthy
	if svc.RegistryStatus == "stopped" {
		slog.Debug("Skipping health check for stopped service", "service", serviceName)

		return HealthCheckResult{
			ServiceName:  serviceName,
			Timestamp:    time.Now(),
			Status:       HealthStatusUnknown,
			ResponseTime: time.Since(startTime),
			ServiceType:  svc.Type,
			ServiceMode:  svc.Mode,
		}
	}

	slog.Debug("Starting health check", "service", serviceName, "port", svc.Port, "pid", svc.PID)

	// Apply rate limiting if configured
	limiter := c.getOrCreateRateLimiter(serviceName)
	if limiter != nil {
		if err := limiter.Wait(ctx); err != nil {
			slog.Warn("Rate limit exceeded", "service", serviceName, "err", err)

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
					slog.Error("Panic recovered during circuit breaker operation", "service", serviceName, "panic", r)
					result = HealthCheckResult{
						ServiceName: serviceName,
						Timestamp:   time.Now(),
						Status:      HealthStatusUnknown,
						Error:       fmt.Sprintf("internal error: panic during health check: %v", r),
					}
				}
			}()

			output, err := breaker.Execute(func() (any, error) {
				res := c.performServiceCheck(ctx, svc)
				if res.Status == HealthStatusUnhealthy {
					return res, fmt.Errorf("health check failed: %s", res.Error)
				}
				return res, nil
			})

			if err != nil {
				if errors.Is(err, gobreaker.ErrOpenState) {
					slog.Warn("Circuit breaker open - skipping check", "service", serviceName)

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
					slog.Error("Circuit breaker returned unexpected type", "service", serviceName, "type", fmt.Sprintf("%T", output))
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

	slog.Debug("Health check completed", "service", serviceName, "status", string(result.Status), "duration", duration)

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
	if svc.Type == ServiceTypeProcess {
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
		result.Details = make(map[string]any)

		// Create a context with timeout for port check
		portCtx, cancel := context.WithTimeout(ctx, defaultPortCheckTimeout)
		defer cancel()

		address := fmt.Sprintf("localhost:%d", svc.Port)
		dialer := net.Dialer{Timeout: defaultPortCheckTimeout}
		conn, err := dialer.DialContext(portCtx, "tcp", address)

		if err == nil {
			_ = conn.Close()
			result.Status = HealthStatusHealthy
		} else {
			if isInStartupGracePeriod {
				result.Status = HealthStatusStarting
			} else {
				result.Status = HealthStatusUnhealthy
			}
			result.Error = fmt.Sprintf("port %d not listening", svc.Port)
			// Add actionable suggestion
			result.Details["suggestion"] = suggestTCPErrorAction(err, svc.Port)
			result.Details["port"] = svc.Port
		}
		return result
	}

	// 3. Fall back to process check
	if svc.PID > 0 {
		result.CheckType = HealthCheckTypeProcess
		result.PID = svc.PID
		result.Details = make(map[string]any)

		isRunning := isProcessRunning(svc.PID)
		if isRunning {
			result.Status = HealthStatusHealthy
			result.Details["pid"] = svc.PID
		} else {
			if isInStartupGracePeriod {
				result.Status = HealthStatusStarting
			} else {
				result.Status = HealthStatusUnhealthy
			}
			result.Error = fmt.Sprintf("process %d not running", svc.PID)
			// Add actionable suggestion
			result.Details["suggestion"] = suggestProcessErrorAction(svc.PID, isRunning, svc.Mode)
			result.Details["pid"] = svc.PID
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
