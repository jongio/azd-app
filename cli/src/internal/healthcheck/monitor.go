// Package healthcheck provides health monitoring capabilities for services.
package healthcheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/procutil"
	"github.com/jongio/azd-app/cli/src/internal/registry"
	"github.com/jongio/azd-app/cli/src/internal/service"
	cache "github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
	"gopkg.in/yaml.v3"
)

const (
	// maxConcurrentChecks limits parallel health check execution
	maxConcurrentChecks = 10

	// maxResponseBodySize limits the size of health check response bodies to prevent memory issues
	maxResponseBodySize = 1024 * 1024 // 1MB

	// defaultPortCheckTimeout is the timeout for TCP port checks
	defaultPortCheckTimeout = 2 * time.Second
)

// Common health check endpoint paths to try
var commonHealthPaths = []string{
	"/health",
	"/healthz",
	"/ready",
	"/alive",
	"/ping",
}

var (
	// metricsEnabled controls whether Prometheus metrics are recorded.
	// Uses atomic.Bool for thread-safe concurrent access from multiple goroutines.
	metricsEnabled atomic.Bool
)

// HealthStatus represents the health state of a service.
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusStarting  HealthStatus = "starting"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// HealthCheckType indicates the method used for health checking.
type HealthCheckType string

const (
	HealthCheckTypeHTTP    HealthCheckType = "http"
	HealthCheckTypePort    HealthCheckType = "port"
	HealthCheckTypeProcess HealthCheckType = "process"
)

// HealthCheckResult represents the result of a single health check.
type HealthCheckResult struct {
	ServiceName  string                 `json:"serviceName"`
	Status       HealthStatus           `json:"status"`
	CheckType    HealthCheckType        `json:"checkType"`
	Endpoint     string                 `json:"endpoint,omitempty"`
	ResponseTime time.Duration          `json:"responseTime"`
	StatusCode   int                    `json:"statusCode,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	Details      map[string]interface{} `json:"details,omitempty"`
	Port         int                    `json:"port,omitempty"`
	PID          int                    `json:"pid,omitempty"`
	Uptime       time.Duration          `json:"uptime,omitempty"`
}

// HealthReport contains aggregated health check results.
type HealthReport struct {
	Timestamp time.Time           `json:"timestamp"`
	Project   string              `json:"project"`
	Services  []HealthCheckResult `json:"services"`
	Summary   HealthSummary       `json:"summary"`
}

// HealthSummary provides overall health statistics.
type HealthSummary struct {
	Total     int          `json:"total"`
	Healthy   int          `json:"healthy"`
	Degraded  int          `json:"degraded"`
	Unhealthy int          `json:"unhealthy"`
	Unknown   int          `json:"unknown"`
	Overall   HealthStatus `json:"overall"`
}

// MonitorConfig holds configuration for the health monitor.
type MonitorConfig struct {
	ProjectDir             string
	DefaultEndpoint        string
	Timeout                time.Duration
	Verbose                bool
	LogLevel               string
	LogFormat              string
	EnableCircuitBreaker   bool
	CircuitBreakerFailures int
	CircuitBreakerTimeout  time.Duration
	RateLimit              int // Max checks per second per service (0 = unlimited)
	EnableMetrics          bool
	MetricsPort            int
	CacheTTL               time.Duration
}

// HealthMonitor coordinates health checking operations.
type HealthMonitor struct {
	config   MonitorConfig
	registry *registry.ServiceRegistry
	checker  *HealthChecker
	cache    *cache.Cache
}

// HealthChecker performs individual health checks with circuit breaker and rate limiting.
type HealthChecker struct {
	timeout         time.Duration
	defaultEndpoint string
	httpClient      *http.Client
	breakers        map[string]*gobreaker.CircuitBreaker
	rateLimiters    map[string]*rate.Limiter
	mu              sync.RWMutex
	enableBreaker   bool
	breakerFailures int
	breakerTimeout  time.Duration
	rateLimit       int
}

// InitializeLogging configures the zerolog logger based on config.
func InitializeLogging(logLevel, logFormat string) {
	// Set up time format
	zerolog.TimeFieldFormat = time.RFC3339

	// Configure output format - all formats write to stderr to avoid
	// interfering with command output (especially JSON output)
	switch logFormat {
	case "json":
		// JSON output to stderr
		log.Logger = log.Output(os.Stderr)
	case "pretty":
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "15:04:05",
		})
	case "text":
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			NoColor:    true,
			TimeFormat: time.RFC3339,
		})
	default:
		// Default to simple console output
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "15:04:05",
		})
	}

	// Set log level
	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
}

// NewHealthMonitor creates a new health monitor.
func NewHealthMonitor(config MonitorConfig) (*HealthMonitor, error) {
	// Initialize logging
	InitializeLogging(config.LogLevel, config.LogFormat)

	// Set metrics flag atomically for thread-safe access
	metricsEnabled.Store(config.EnableMetrics)

	log.Debug().
		Str("project_dir", config.ProjectDir).
		Str("endpoint", config.DefaultEndpoint).
		Dur("timeout", config.Timeout).
		Bool("metrics", config.EnableMetrics).
		Bool("circuit_breaker", config.EnableCircuitBreaker).
		Msg("Creating health monitor")

	// Get service registry
	reg := registry.GetRegistry(config.ProjectDir)

	// Create cache if TTL is configured
	var healthCache *cache.Cache
	if config.CacheTTL > 0 {
		healthCache = cache.New(config.CacheTTL, config.CacheTTL*2)
		log.Debug().Dur("ttl", config.CacheTTL).Msg("Health check caching enabled")
	}

	// Create health checker with properly configured HTTP client
	checker := &HealthChecker{
		timeout:         config.Timeout,
		defaultEndpoint: config.DefaultEndpoint,
		breakers:        make(map[string]*gobreaker.CircuitBreaker),
		rateLimiters:    make(map[string]*rate.Limiter),
		enableBreaker:   config.EnableCircuitBreaker,
		breakerFailures: config.CircuitBreakerFailures,
		breakerTimeout:  config.CircuitBreakerTimeout,
		rateLimit:       config.RateLimit,
		httpClient: &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				DisableKeepAlives:   false,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}

	return &HealthMonitor{
		config:   config,
		registry: reg,
		checker:  checker,
		cache:    healthCache,
	}, nil
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

// Check performs health checks on all or filtered services.
func (m *HealthMonitor) Check(ctx context.Context, serviceFilter []string) (*HealthReport, error) {
	// Check cache if enabled
	cacheKey := "health_report"
	if len(serviceFilter) > 0 {
		cacheKey = fmt.Sprintf("health_report::%s", strings.Join(serviceFilter, "::"))
	}

	if m.cache != nil {
		if cached, found := m.cache.Get(cacheKey); found {
			log.Debug().Str("key", cacheKey).Msg("Returning cached health report")
			return cached.(*HealthReport), nil
		}
	}

	log.Debug().
		Strs("filter", serviceFilter).
		Msg("Performing health checks")

	// Load azure.yaml to get service definitions
	azureYaml, err := m.loadAzureYaml()
	if err != nil {
		// If no azure.yaml, just use registry
		if m.config.Verbose {
			log.Warn().Err(err).Msg("Could not load azure.yaml")
		}
	}

	// Get services from registry
	registeredServices := m.registry.ListAll()

	// Build service list combining registry and azure.yaml
	services := m.buildServiceList(azureYaml, registeredServices)

	// Apply filter if specified
	if len(serviceFilter) > 0 {
		services = filterServices(services, serviceFilter)
	}

	log.Info().
		Int("total_services", len(services)).
		Msg("Starting health checks")

	// Perform health checks in parallel
	results := make([]HealthCheckResult, len(services))
	resultChan := make(chan struct {
		index  int
		result HealthCheckResult
	}, len(services))

	// Limit concurrency to prevent overwhelming the system
	semaphore := make(chan struct{}, maxConcurrentChecks)

	for i, svc := range services {
		go func(index int, svc serviceInfo) {
			// Use select to check context cancellation when acquiring semaphore
			// This prevents goroutine leak if context is cancelled while waiting
			select {
			case semaphore <- struct{}{}:
				// Acquired semaphore, proceed with check
			case <-ctx.Done():
				// Context cancelled, send cancelled result and exit
				resultChan <- struct {
					index  int
					result HealthCheckResult
				}{index, HealthCheckResult{
					ServiceName: svc.Name,
					Timestamp:   time.Now(),
					Status:      HealthStatusUnknown,
					Error:       "context cancelled",
				}}
				return
			}
			defer func() { <-semaphore }()

			result := m.checker.CheckService(ctx, svc)
			resultChan <- struct {
				index  int
				result HealthCheckResult
			}{index, result}
		}(i, svc)
	}

	// Collect results
	for i := 0; i < len(services); i++ {
		res := <-resultChan
		results[res.index] = res.result
	}

	// Calculate summary
	summary := calculateSummary(results)

	log.Info().
		Int("healthy", summary.Healthy).
		Int("unhealthy", summary.Unhealthy).
		Int("degraded", summary.Degraded).
		Int("unknown", summary.Unknown).
		Msg("Health checks completed")

	report := &HealthReport{
		Timestamp: time.Now(),
		Project:   m.config.ProjectDir,
		Services:  results,
		Summary:   summary,
	}

	// Update registry with health status
	m.updateRegistry(results)

	// Cache the report if caching is enabled
	if m.cache != nil {
		m.cache.Set(cacheKey, report, cache.DefaultExpiration)
		log.Debug().Str("key", cacheKey).Msg("Cached health report")
	}

	return report, nil
}

type serviceInfo struct {
	Name           string
	Port           int
	PID            int
	StartTime      time.Time
	HealthCheck    *healthCheckConfig
	RegistryHealth string
}

type healthCheckConfig struct {
	Test          []string
	Interval      time.Duration
	Timeout       time.Duration
	Retries       int
	StartPeriod   time.Duration
	StartInterval time.Duration
}

func (m *HealthMonitor) loadAzureYaml() (*service.AzureYaml, error) {
	azureYamlPath := filepath.Join(m.config.ProjectDir, "azure.yaml")
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		return nil, err
	}

	var azureYaml service.AzureYaml
	if err := yaml.Unmarshal(data, &azureYaml); err != nil {
		return nil, err
	}

	return &azureYaml, nil
}

func (m *HealthMonitor) buildServiceList(azureYaml *service.AzureYaml, registeredServices []*registry.ServiceRegistryEntry) []serviceInfo {
	serviceMap := make(map[string]serviceInfo)

	// Add services from registry
	for _, regSvc := range registeredServices {
		serviceMap[regSvc.Name] = serviceInfo{
			Name:           regSvc.Name,
			Port:           regSvc.Port,
			PID:            regSvc.PID,
			StartTime:      regSvc.StartTime,
			RegistryHealth: regSvc.Health,
		}
	}

	// Enhance with azure.yaml data if available
	if azureYaml != nil {
		for name, svc := range azureYaml.Services {
			info, exists := serviceMap[name]
			if !exists {
				// Service defined in azure.yaml but not in registry
				info = serviceInfo{Name: name}
			}

			// Parse healthcheck config (Docker Compose format)
			info.HealthCheck = parseHealthCheckConfig(svc)

			serviceMap[name] = info
		}
	}

	// Convert map to slice
	var services []serviceInfo
	for _, svc := range serviceMap {
		services = append(services, svc)
	}

	return services
}

func parseHealthCheckConfig(svc service.Service) *healthCheckConfig {
	// Check if healthcheck is disabled using the helper method
	if svc.IsHealthcheckDisabled() {
		return &healthCheckConfig{
			Test: []string{"NONE"},
		}
	}

	// Docker Compose style healthcheck parsing from azure.yaml
	if svc.Healthcheck == nil {
		return nil
	}

	config := &healthCheckConfig{
		Retries: 3, // Default
	}

	// Parse test field - can be string or array
	switch t := svc.Healthcheck.Test.(type) {
	case string:
		// Single string - could be URL or shell command
		config.Test = []string{t}
	case []interface{}:
		// Array format: ["CMD", "curl", "-f", "..."] or ["CMD-SHELL", "..."]
		for _, item := range t {
			if s, ok := item.(string); ok {
				config.Test = append(config.Test, s)
			}
		}
	case []string:
		config.Test = t
	}

	// Handle type: "none" - convert to ["NONE"] format for compatibility
	if svc.Healthcheck.Type == "none" {
		config.Test = []string{"NONE"}
	}

	// Parse interval
	if svc.Healthcheck.Interval != "" {
		if d, err := time.ParseDuration(svc.Healthcheck.Interval); err == nil {
			config.Interval = d
		}
	}

	// Parse timeout
	if svc.Healthcheck.Timeout != "" {
		if d, err := time.ParseDuration(svc.Healthcheck.Timeout); err == nil {
			config.Timeout = d
		}
	}

	// Parse retries
	if svc.Healthcheck.Retries > 0 {
		config.Retries = svc.Healthcheck.Retries
	}

	// Parse start_period
	if svc.Healthcheck.StartPeriod != "" {
		if d, err := time.ParseDuration(svc.Healthcheck.StartPeriod); err == nil {
			config.StartPeriod = d
		}
	}

	// Parse start_interval
	if svc.Healthcheck.StartInterval != "" {
		if d, err := time.ParseDuration(svc.Healthcheck.StartInterval); err == nil {
			config.StartInterval = d
		}
	}

	return config
}

func filterServices(services []serviceInfo, filter []string) []serviceInfo {
	filterMap := make(map[string]bool)
	for _, name := range filter {
		filterMap[name] = true
	}

	var filtered []serviceInfo
	for _, svc := range services {
		if filterMap[svc.Name] {
			filtered = append(filtered, svc)
		}
	}

	return filtered
}

func (m *HealthMonitor) updateRegistry(results []HealthCheckResult) {
	// Batch status updates to reduce lock contention
	for _, result := range results {
		status := "running"
		if result.Status == HealthStatusUnhealthy {
			status = "error"
		} else if result.Status == HealthStatusDegraded {
			status = "degraded"
		}

		// Update registry with health status
		// Registry has internal locking, so this is safe
		if err := m.registry.UpdateStatus(result.ServiceName, status, string(result.Status)); err != nil {
			if m.config.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: Failed to update registry for %s: %v\n", result.ServiceName, err)
			}
		}
	}
}

func calculateSummary(results []HealthCheckResult) HealthSummary {
	summary := HealthSummary{
		Total: len(results),
	}

	for _, result := range results {
		switch result.Status {
		case HealthStatusHealthy:
			summary.Healthy++
		case HealthStatusDegraded:
			summary.Degraded++
		case HealthStatusUnhealthy:
			summary.Unhealthy++
		default:
			summary.Unknown++
		}
	}

	// Determine overall status
	if summary.Unhealthy > 0 {
		summary.Overall = HealthStatusUnhealthy
	} else if summary.Degraded > 0 {
		summary.Overall = HealthStatusDegraded
	} else if summary.Healthy > 0 {
		summary.Overall = HealthStatusHealthy
	} else {
		summary.Overall = HealthStatusUnknown
	}

	return summary
}

// CheckService performs a health check on a single service using cascading strategy.
func (c *HealthChecker) CheckService(ctx context.Context, svc serviceInfo) HealthCheckResult {
	startTime := time.Now()
	serviceName := svc.Name

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

	// Check for custom healthcheck config first
	if svc.HealthCheck != nil && len(svc.HealthCheck.Test) > 0 {
		if httpResult := c.tryCustomHealthCheck(ctx, svc.HealthCheck); httpResult != nil {
			result.CheckType = HealthCheckTypeHTTP
			result.Endpoint = httpResult.Endpoint
			result.ResponseTime = httpResult.ResponseTime
			result.StatusCode = httpResult.StatusCode
			result.Status = httpResult.Status
			result.Details = httpResult.Details
			result.Error = httpResult.Error
			// Extract port from URL if available
			if u, err := url.Parse(httpResult.Endpoint); err == nil {
				if p := u.Port(); p != "" {
					if port, err := strconv.Atoi(p); err == nil {
						result.Port = port
					}
				}
			}
			return result
		}
	}

	// Cascading strategy: HTTP -> Port -> Process

	// 1. Try HTTP health check
	if svc.Port > 0 {
		if httpResult := c.tryHTTPHealthCheck(ctx, svc.Port); httpResult != nil {
			result.CheckType = HealthCheckTypeHTTP
			result.Endpoint = httpResult.Endpoint
			result.ResponseTime = httpResult.ResponseTime
			result.StatusCode = httpResult.StatusCode
			result.Status = httpResult.Status
			result.Details = httpResult.Details
			result.Error = httpResult.Error
			result.Port = svc.Port
			return result
		}
	}

	// 2. Fall back to port check
	if svc.Port > 0 {
		result.CheckType = HealthCheckTypePort
		result.Port = svc.Port
		if c.checkPort(ctx, svc.Port) {
			result.Status = HealthStatusHealthy
		} else {
			result.Status = HealthStatusUnhealthy
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
			result.Status = HealthStatusUnhealthy
			result.Error = fmt.Sprintf("process %d not running", svc.PID)
		}
		return result
	}

	// No check available
	result.CheckType = HealthCheckTypeProcess
	result.Status = HealthStatusUnknown
	result.Error = "no health check method available"

	return result
}

// tryCustomHealthCheck performs a health check using custom configuration from azure.yaml.
func (c *HealthChecker) tryCustomHealthCheck(ctx context.Context, config *healthCheckConfig) *httpHealthCheckResult {
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
			// CMD format: ["CMD", "curl", "-f", "http://..."]
			// Execute command directly
			return c.performCommandCheck(ctx, config.Test[1:])
		case "CMD-SHELL":
			// CMD-SHELL format: ["CMD-SHELL", "curl -f http://... || exit 1"]
			// Execute through shell
			return c.performShellCheck(ctx, config.Test[1])
		case "NONE":
			// Disable health check - return healthy
			return &httpHealthCheckResult{
				Endpoint: "none",
				Status:   HealthStatusHealthy,
			}
		}
	}

	// Single string that's not a URL - treat as shell command
	return c.performShellCheck(ctx, test)
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
	_ = resp.Body.Close()

	result := &httpHealthCheckResult{
		Endpoint:     urlStr,
		ResponseTime: responseTime,
		StatusCode:   resp.StatusCode,
	}

	// Determine status based on HTTP status code
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		result.Status = HealthStatusHealthy
	case resp.StatusCode >= 300 && resp.StatusCode < 400:
		result.Status = HealthStatusHealthy // Redirects OK
	case resp.StatusCode >= 500:
		result.Status = HealthStatusUnhealthy
	default:
		result.Status = HealthStatusDegraded
	}

	// Try to parse response body for additional details
	if readErr == nil && len(body) > 0 && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var details map[string]interface{}
		if err := json.Unmarshal(body, &details); err == nil {
			result.Details = details

			// Check for explicit status in response
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

	return result
}

// performCommandCheck executes a command for health check (CMD format).
func (c *HealthChecker) performCommandCheck(ctx context.Context, args []string) *httpHealthCheckResult {
	if len(args) == 0 {
		return nil
	}

	startTime := time.Now()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	err := cmd.Run()
	responseTime := time.Since(startTime)

	result := &httpHealthCheckResult{
		Endpoint:     strings.Join(args, " "),
		ResponseTime: responseTime,
	}

	if err != nil {
		result.Status = HealthStatusUnhealthy
		result.Error = fmt.Sprintf("command failed: %v", err)
	} else {
		result.Status = HealthStatusHealthy
	}

	return result
}

// performShellCheck executes a shell command for health check (CMD-SHELL format).
func (c *HealthChecker) performShellCheck(ctx context.Context, command string) *httpHealthCheckResult {
	startTime := time.Now()

	// Use appropriate shell based on OS
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	err := cmd.Run()
	responseTime := time.Since(startTime)

	result := &httpHealthCheckResult{
		Endpoint:     command,
		ResponseTime: responseTime,
	}

	if err != nil {
		result.Status = HealthStatusUnhealthy
		result.Error = fmt.Sprintf("command failed: %v", err)
	} else {
		result.Status = HealthStatusHealthy
	}

	return result
}

type httpHealthCheckResult struct {
	Endpoint     string
	ResponseTime time.Duration
	StatusCode   int
	Status       HealthStatus
	Details      map[string]interface{}
	Error        string
}

func (c *HealthChecker) tryHTTPHealthCheck(ctx context.Context, port int) *httpHealthCheckResult {
	// Try common health endpoints
	endpoints := []string{c.defaultEndpoint}

	// Add other common paths if they're different from default
	for _, path := range commonHealthPaths {
		if path != c.defaultEndpoint {
			endpoints = append(endpoints, path)
		}
	}

	for _, endpoint := range endpoints {
		// Check if context is already cancelled before making request
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		url := fmt.Sprintf("http://localhost:%d%s", port, endpoint)

		startTime := time.Now()
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue
		}

		resp, err := c.httpClient.Do(req)
		responseTime := time.Since(startTime)

		if err != nil {
			// Connection error - likely service not ready
			continue
		}

		// Read and close body immediately (not in defer) to prevent resource leaks
		// when iterating through multiple endpoints. Defer inside loops accumulates.
		limitedReader := io.LimitReader(resp.Body, maxResponseBodySize)
		body, readErr := io.ReadAll(limitedReader)
		closeErr := resp.Body.Close()
		if closeErr != nil {
			// Log error but don't fail health check
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", closeErr)
		}

		// Skip 404 responses - endpoint doesn't exist, try next one
		if resp.StatusCode == http.StatusNotFound {
			continue
		}

		// Found a responding endpoint
		result := &httpHealthCheckResult{
			Endpoint:     url,
			ResponseTime: responseTime,
			StatusCode:   resp.StatusCode,
		}

		// Determine status based on HTTP status code
		switch {
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			result.Status = HealthStatusHealthy
		case resp.StatusCode >= 300 && resp.StatusCode < 400:
			result.Status = HealthStatusHealthy // Redirects OK
		case resp.StatusCode >= 500:
			result.Status = HealthStatusUnhealthy
		default:
			result.Status = HealthStatusDegraded
		}

		// Try to parse response body for additional details
		if readErr == nil && len(body) > 0 && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			var details map[string]interface{}
			if err := json.Unmarshal(body, &details); err == nil {
				result.Details = details

				// Check for explicit status in response
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

		return result
	}

	return nil
}

func (c *HealthChecker) checkPort(ctx context.Context, port int) bool {
	address := fmt.Sprintf("localhost:%d", port)
	// Use dialer with context for proper cancellation support
	dialer := net.Dialer{Timeout: defaultPortCheckTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return false
	}
	_ = conn.Close() // Ignore close error for port check
	return true
}

// isProcessRunning delegates to procutil.IsProcessRunning for cross-platform process detection.
// This wrapper maintains backward compatibility while eliminating code duplication.
func isProcessRunning(pid int) bool {
	return procutil.IsProcessRunning(pid)
}
