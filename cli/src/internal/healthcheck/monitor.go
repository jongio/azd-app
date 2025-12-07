// Package healthcheck provides health monitoring capabilities for services.
package healthcheck

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/registry"
	"github.com/jongio/azd-app/cli/src/internal/service"
	cache "github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
	"gopkg.in/yaml.v3"
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
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
		// Add reasonable timeouts for dial and TLS handshake
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
)

// HealthMonitor coordinates health checking operations.
type HealthMonitor struct {
	config   MonitorConfig
	registry *registry.ServiceRegistry
	checker  *HealthChecker
	cache    *cache.Cache
}

// InitializeLogging configures the zerolog logger based on config.
func InitializeLogging(logLevel, logFormat string) {
	zerolog.TimeFieldFormat = time.RFC3339

	switch logFormat {
	case "json":
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
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "15:04:05",
		})
	}

	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
}

// NewHealthMonitor creates a new health monitor.
func NewHealthMonitor(config MonitorConfig) (*HealthMonitor, error) {
	InitializeLogging(config.LogLevel, config.LogFormat)
	metricsEnabled.Store(config.EnableMetrics)

	log.Debug().
		Str("project_dir", config.ProjectDir).
		Str("endpoint", config.DefaultEndpoint).
		Dur("timeout", config.Timeout).
		Bool("metrics", config.EnableMetrics).
		Bool("circuit_breaker", config.EnableCircuitBreaker).
		Msg("Creating health monitor")

	reg := registry.GetRegistry(config.ProjectDir)

	var healthCache *cache.Cache
	if config.CacheTTL > 0 {
		healthCache = cache.New(config.CacheTTL, config.CacheTTL*2)
		log.Debug().Dur("ttl", config.CacheTTL).Msg("Health check caching enabled")
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
		config:   config,
		registry: reg,
		checker:  checker,
		cache:    healthCache,
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
			log.Debug().Str("key", cacheKey).Msg("Returning cached health report")
			return cached.(*HealthReport), nil
		}
	}

	log.Debug().Strs("filter", serviceFilter).Msg("Performing health checks")

	azureYaml, err := m.loadAzureYaml()
	if err != nil && m.config.Verbose {
		log.Warn().Err(err).Msg("Could not load azure.yaml")
	}

	registeredServices := m.registry.ListAll()
	services := m.buildServiceList(azureYaml, registeredServices)

	if len(serviceFilter) > 0 {
		services = filterServices(services, serviceFilter)
	}

	log.Info().Int("total_services", len(services)).Msg("Starting health checks")

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
					Error:       "context cancelled before check started",
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
					Error:       "context cancelled",
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
					Error:       "context cancelled",
				}}
				return
			}

			result := m.checker.CheckService(ctx, svc)
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

	m.updateRegistry(results)

	if m.cache != nil {
		m.cache.Set(cacheKey, report, cache.DefaultExpiration)
		log.Debug().Str("key", cacheKey).Msg("Cached health report")
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
			if info.Mode == "" && info.Type == service.ServiceTypeProcess {
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

	var services []serviceInfo
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
	case []interface{}:
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

func (m *HealthMonitor) updateRegistry(results []HealthCheckResult) {
	for _, result := range results {
		currentEntry, exists := m.registry.GetService(result.ServiceName)
		if exists && currentEntry.Status == "stopped" {
			log.Debug().
				Str("service", result.ServiceName).
				Msg("Skipping registry update for stopped service")
			continue
		}

		if exists && currentEntry.Status == "running" && result.Status == HealthStatusStarting {
			log.Debug().
				Str("service", result.ServiceName).
				Msg("Keeping running status during health check grace period")
			continue
		}

		var status string

		if result.ServiceMode == service.ServiceModeBuild || result.ServiceMode == service.ServiceModeTask {
			if result.Status == HealthStatusHealthy {
				if details, ok := result.Details["state"].(string); ok {
					switch details {
					case "built":
						status = "built"
					case "completed":
						status = "completed"
					case "building", "running":
						status = "running"
					case "failed":
						status = "error"
					default:
						status = "running"
					}
				} else {
					status = "running"
				}
			} else if result.Status == HealthStatusUnhealthy {
				status = "error"
			} else if result.Status == HealthStatusStarting {
				status = "starting"
			} else {
				status = "running"
			}
		} else {
			status = "running"
			if result.Status == HealthStatusUnhealthy {
				status = "error"
			} else if result.Status == HealthStatusDegraded {
				status = "degraded"
			} else if result.Status == HealthStatusStarting {
				status = "starting"
			}
		}

		if err := m.registry.UpdateStatus(result.ServiceName, status); err != nil {
			if m.config.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: Failed to update registry for %s: %v\n", result.ServiceName, err)
			}
		}
	}
}
