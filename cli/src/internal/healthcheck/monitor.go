// Package healthcheck provides health monitoring capabilities for services.
package healthcheck

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/service" // for AzureYaml, Service, GetLogManager (app-specific)
	"github.com/jongio/azd-core/registry"
	cache "github.com/patrickmn/go-cache"
	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
	"gopkg.in/yaml.v3"
)

const (
	statusRunning  = "running"
	statusBuilt    = "built"
	statusStarting = "starting"
)

var (
	// metricsEnabled controls whether Prometheus metrics are recorded.
	// Uses atomic.Bool for thread-safe concurrent access from multiple goroutines.
	metricsEnabled atomic.Bool

	// sharedHTTPTransport is a shared HTTP transport for all health checkers
	// to prevent resource exhaustion from creating multiple connection pools
	sharedHTTPTransport = &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     HTTPIdleConnTimeout,
		DisableKeepAlives:   false,
		// Add reasonable timeouts for dial and TLS handshake
		DialContext: (&net.Dialer{
			Timeout:   HTTPDialTimeout,
			KeepAlive: HTTPKeepAliveTimeout,
		}).DialContext,
		TLSHandshakeTimeout:   HTTPTLSHandshakeTimeout,
		ExpectContinueTimeout: HTTPExpectContinueTimeout,
	}
)

// HealthMonitor coordinates health checking operations.
type HealthMonitor struct {
	config          MonitorConfig
	registry        *registry.ServiceRegistry
	checker         *HealthChecker
	cache           *cache.Cache
	failureCount    map[string]int       // Track consecutive failures per service
	lastSuccessTime map[string]time.Time // Track last success time per service
	failureCountMu  sync.RWMutex         // Thread-safe access to failure tracking maps
}

// InitializeLogging configures the slog default logger based on config.
func InitializeLogging(logLevel, logFormat string) {
	var level slog.Level
	switch strings.ToLower(logLevel) {
	case "debug":
		level = slog.LevelDebug
	case "info", "":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	switch logFormat {
	case "json":
		handler = slog.NewJSONHandler(os.Stderr, opts)
	default:
		handler = slog.NewTextHandler(os.Stderr, opts)
	}
	slog.SetDefault(slog.New(handler))
}

// NewHealthMonitor creates a new health monitor.
func NewHealthMonitor(config MonitorConfig) (*HealthMonitor, error) {
	InitializeLogging(config.LogLevel, config.LogFormat)
	metricsEnabled.Store(config.EnableMetrics)

	slog.Debug("Creating health monitor", "project_dir", config.ProjectDir, "endpoint", config.DefaultEndpoint, "timeout", config.Timeout, "metrics", config.EnableMetrics, "circuit_breaker", config.EnableCircuitBreaker)

	reg := registry.GetRegistry(config.ProjectDir)

	var healthCache *cache.Cache
	if config.CacheTTL > 0 {
		healthCache = cache.New(config.CacheTTL, config.CacheTTL*2)
		slog.Debug("Health check caching enabled", "ttl", config.CacheTTL)
	}

	// Determine startup grace period
	gracePeriod := config.StartupGracePeriod
	if gracePeriod == 0 {
		gracePeriod = startupGracePeriod // Use default
	}

	checker := &HealthChecker{
		timeout:            config.Timeout,
		defaultEndpoint:    config.DefaultEndpoint,
		breakers:           make(map[string]*gobreaker.CircuitBreaker),
		rateLimiters:       make(map[string]*rate.Limiter),
		endpointCache:      make(map[string]string),
		enableBreaker:      config.EnableCircuitBreaker,
		breakerFailures:    config.CircuitBreakerFailures,
		breakerTimeout:     config.CircuitBreakerTimeout,
		rateLimit:          config.RateLimit,
		startupGracePeriod: gracePeriod,
		httpClient: &http.Client{
			Timeout:   config.Timeout,
			Transport: sharedHTTPTransport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}

	return &HealthMonitor{
		config:          config,
		registry:        reg,
		checker:         checker,
		cache:           healthCache,
		failureCount:    make(map[string]int),
		lastSuccessTime: make(map[string]time.Time),
	}, nil
}

// Check performs health checks on all or filtered services.
func (m *HealthMonitor) Check(ctx context.Context, serviceFilter []string) (*HealthReport, error) {
	// Create a safe cache key that handles service names with special characters
	// Use a delimiter that's unlikely to appear in service names and escape it if needed
	cacheKey := "health_report"
	if len(serviceFilter) > 0 {
		// Sort to ensure consistent cache keys regardless of filter order
		sortedFilter := make([]string, len(serviceFilter))
		copy(sortedFilter, serviceFilter)
		// Use pipe as delimiter and URL encode service names to avoid collisions
		var encodedServices []string
		for _, svc := range sortedFilter {
			// Replace special characters to prevent cache key collisions
			encoded := strings.ReplaceAll(svc, "|", "%7C")
			encoded = strings.ReplaceAll(encoded, ":", "%3A")
			encodedServices = append(encodedServices, encoded)
		}
		cacheKey = fmt.Sprintf("health_report|%s", strings.Join(encodedServices, "|"))
	}

	if m.cache != nil {
		if cached, found := m.cache.Get(cacheKey); found {
			report, ok := cached.(*HealthReport)
			if !ok {
				return nil, fmt.Errorf("cached health report for %q has unexpected type %T", cacheKey, cached)
			}
			slog.Debug("Returning cached health report", "key", cacheKey)
			return report, nil
		}
	}

	slog.Debug("Performing health checks", "filter", serviceFilter)

	azureYaml, err := m.loadAzureYaml()
	if err != nil && m.config.Verbose {
		slog.Warn("Could not load azure.yaml", "err", err)
	}

	registeredServices := m.registry.ListAll()
	services := m.buildServiceList(azureYaml, registeredServices)

	if len(serviceFilter) > 0 {
		services = filterServices(services, serviceFilter)
	}

	slog.Info("Starting health checks", "total_services", len(services))

	results := make([]HealthCheckResult, len(services))
	resultChan := make(chan struct {
		index  int
		result HealthCheckResult
	}, len(services))

	semaphore := make(chan struct{}, maxConcurrentChecks)

	for i, svc := range services {
		go func(index int, svc serviceInfo) {
			// Check context before attempting to acquire semaphore
			if ctx.Err() != nil {
				resultChan <- struct {
					index  int
					result HealthCheckResult
				}{index, HealthCheckResult{
					ServiceName: svc.Name,
					Timestamp:   time.Now(),
					Status:      HealthStatusUnknown,
					Error:       "context canceled before check started",
				}}
				return
			}

			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				resultChan <- struct {
					index  int
					result HealthCheckResult
				}{index, HealthCheckResult{
					ServiceName: svc.Name,
					Timestamp:   time.Now(),
					Status:      HealthStatusUnknown,
					Error:       "context canceled",
				}}
				return
			}
			defer func() { <-semaphore }()

			// Check context again after acquiring semaphore
			if ctx.Err() != nil {
				resultChan <- struct {
					index  int
					result HealthCheckResult
				}{index, HealthCheckResult{
					ServiceName: svc.Name,
					Timestamp:   time.Now(),
					Status:      HealthStatusUnknown,
					Error:       "context canceled",
				}}
				return
			}

			result := m.checker.CheckService(ctx, svc)
			// Track failures and update result with consecutive failure count
			m.trackFailure(&result)
			resultChan <- struct {
				index  int
				result HealthCheckResult
			}{index, result}
		}(i, svc)
	}

	for i := 0; i < len(services); i++ {
		res := <-resultChan
		results[res.index] = res.result
	}

	summary := calculateSummary(results)

	slog.Info("Health checks completed", "healthy", summary.Healthy, "unhealthy", summary.Unhealthy, "degraded", summary.Degraded, "unknown", summary.Unknown)

	report := &HealthReport{
		Timestamp: time.Now(),
		Project:   m.config.ProjectDir,
		Services:  results,
		Summary:   summary,
	}

	m.updateRegistry(results)

	if m.cache != nil {
		m.cache.Set(cacheKey, report, cache.DefaultExpiration)
		slog.Debug("Cached health report", "key", cacheKey)
	}

	return report, nil
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

	for _, regSvc := range registeredServices {
		serviceMap[regSvc.Name] = serviceInfo{
			Name:           regSvc.Name,
			Port:           regSvc.Port,
			PID:            regSvc.PID,
			StartTime:      regSvc.StartTime,
			RegistryStatus: regSvc.Status,
			Type:           regSvc.Type,
			Mode:           regSvc.Mode,
			ExitCode:       regSvc.ExitCode,
			EndTime:        regSvc.EndTime,
		}
	}

	if azureYaml != nil {
		for name, svc := range azureYaml.Services {
			info, exists := serviceMap[name]
			if !exists {
				info = serviceInfo{Name: name}
			}

			info.HealthCheck = parseHealthCheckConfig(svc)

			if info.Type == "" {
				info.Type = svc.GetServiceType()
			}
			if info.Mode == "" && info.Type == ServiceTypeProcess {
				info.Mode = svc.GetServiceMode()
			}

			if info.Port == 0 {
				hostPort, containerPort, _ := svc.GetPrimaryPort()
				if hostPort > 0 {
					info.Port = hostPort
				} else if containerPort > 0 {
					info.Port = containerPort
				}
			}

			serviceMap[name] = info
		}
	}

	services := make([]serviceInfo, 0, len(serviceMap))
	for _, svc := range serviceMap {
		services = append(services, svc)
	}

	return services
}

func parseHealthCheckConfig(svc service.Service) *healthCheckConfig {
	if svc.IsHealthcheckDisabled() {
		return &healthCheckConfig{
			Test: []string{"NONE"},
			Type: "none",
		}
	}

	if svc.Healthcheck == nil {
		return nil
	}

	config := &healthCheckConfig{
		Retries: 3,
		Type:    svc.Healthcheck.Type,
		Pattern: svc.Healthcheck.Pattern,
	}

	switch t := svc.Healthcheck.Test.(type) {
	case string:
		config.Test = []string{t}
	case []any:
		for _, item := range t {
			if s, ok := item.(string); ok {
				config.Test = append(config.Test, s)
			}
		}
	case []string:
		config.Test = t
	}

	if svc.Healthcheck.Type == "none" {
		config.Test = []string{"NONE"}
	}

	if svc.Healthcheck.Interval != "" {
		if d, err := time.ParseDuration(svc.Healthcheck.Interval); err == nil {
			config.Interval = d
		}
	}

	if svc.Healthcheck.Timeout != "" {
		if d, err := time.ParseDuration(svc.Healthcheck.Timeout); err == nil {
			config.Timeout = d
		}
	}

	if svc.Healthcheck.Retries > 0 {
		config.Retries = svc.Healthcheck.Retries
	}

	if svc.Healthcheck.StartPeriod != "" {
		if d, err := time.ParseDuration(svc.Healthcheck.StartPeriod); err == nil {
			config.StartPeriod = d
		}
	}

	if svc.Healthcheck.StartInterval != "" {
		if d, err := time.ParseDuration(svc.Healthcheck.StartInterval); err == nil {
			config.StartInterval = d
		}
	}

	return config
}

// trackFailure updates failure tracking for a service based on health check result.
// Thread-safe: uses mutex to protect shared failure tracking maps.
func (m *HealthMonitor) trackFailure(result *HealthCheckResult) {
	m.failureCountMu.Lock()
	defer m.failureCountMu.Unlock()

	serviceName := result.ServiceName

	switch result.Status {
	case HealthStatusUnhealthy:
		// Increment failure count
		m.failureCount[serviceName]++
		result.ConsecutiveFailures = m.failureCount[serviceName]

		// Set last success time if we have it
		if lastSuccess, exists := m.lastSuccessTime[serviceName]; exists {
			result.LastSuccessTime = &lastSuccess
		}
	case HealthStatusHealthy:
		// Reset failure count on healthy status
		m.failureCount[serviceName] = 0
		result.ConsecutiveFailures = 0

		// Update last success time
		now := time.Now()
		m.lastSuccessTime[serviceName] = now
		result.LastSuccessTime = &now
	case HealthStatusDegraded, HealthStatusStarting, HealthStatusUnknown:
		// For non-terminal states, include current count without incrementing.
		if count, exists := m.failureCount[serviceName]; exists {
			result.ConsecutiveFailures = count
		}
		if lastSuccess, exists := m.lastSuccessTime[serviceName]; exists {
			result.LastSuccessTime = &lastSuccess
		}
	}
}

func (m *HealthMonitor) updateRegistry(results []HealthCheckResult) {
	for _, result := range results {
		currentEntry, exists := m.registry.GetService(result.ServiceName)
		if exists && currentEntry.Status == "stopped" {
			slog.Debug("Skipping registry update for stopped service", "service", result.ServiceName)
			continue
		}

		if exists && currentEntry.Status == statusRunning && result.Status == HealthStatusStarting {
			slog.Debug("Keeping running status during health check grace period", "service", result.ServiceName)
			continue
		}

		var status string

		if result.ServiceMode == ServiceModeBuild || result.ServiceMode == ServiceModeTask {
			switch result.Status {
			case HealthStatusHealthy:
				if details, ok := result.Details["state"].(string); ok {
					switch details {
					case statusBuilt:
						status = statusBuilt
					case statusCompleted:
						status = statusCompleted
					case "building", statusRunning:
						status = statusRunning
					case "failed":
						status = statusError
					default:
						status = statusRunning
					}
				} else {
					status = statusRunning
				}
			case HealthStatusUnhealthy:
				status = statusError
			case HealthStatusStarting:
				status = statusStarting
			case HealthStatusDegraded, HealthStatusUnknown:
				status = statusRunning
			}
		} else {
			status = statusRunning
			switch result.Status {
			case HealthStatusUnhealthy:
				status = statusError
			case HealthStatusDegraded:
				status = "degraded"
			case HealthStatusStarting:
				status = statusStarting
			case HealthStatusHealthy, HealthStatusUnknown:
				status = statusRunning
			}
		}

		if err := m.registry.UpdateStatus(result.ServiceName, status); err != nil {
			if m.config.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: Failed to update registry for %s: %v\n", result.ServiceName, err)
			}
		}
	}
}
