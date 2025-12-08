// Package healthcheck provides health monitoring capabilities for services.
package healthcheck

import (
	"time"
)

const (
	// maxConcurrentChecks limits parallel health check execution
	maxConcurrentChecks = 10

	// maxResponseBodySize limits the size of health check response bodies to prevent memory issues
	maxResponseBodySize = 1024 * 1024 // 1MB

	// defaultPortCheckTimeout is the timeout for TCP port checks
	defaultPortCheckTimeout = 2 * time.Second

	// startupGracePeriod is the time during which services are considered "starting"
	// before being marked as unhealthy if health checks fail
	startupGracePeriod = 30 * time.Second
)

// Common health check endpoint paths to try
var commonHealthPaths = []string{
	"/health",
	"/healthz",
	"/ready",
	"/alive",
	"/ping",
}

// endpointCacheNone is a marker value indicating no HTTP health endpoint was found
// during discovery, so future checks should skip HTTP and fall back to TCP/process.
const endpointCacheNone = "__none__"

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
	HealthCheckTypeTCP     HealthCheckType = "tcp"
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
	ServiceType  string                 `json:"serviceType,omitempty"` // "http", "tcp", "process"
	ServiceMode  string                 `json:"serviceMode,omitempty"` // "watch", "build", "daemon", "task" (for type=process)
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
	Starting  int          `json:"starting"`
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
	StartupGracePeriod     time.Duration // Grace period for services during startup (0 = use default)
}

// serviceInfo holds information about a service for health checking.
type serviceInfo struct {
	Name           string
	Port           int
	PID            int
	StartTime      time.Time
	HealthCheck    *healthCheckConfig
	RegistryStatus string // "running", "stopped", "starting", etc.
	Type           string // "http", "tcp", "process"
	Mode           string // "watch", "build", "daemon", "task" (for type=process)
	ExitCode       *int   // Exit code for completed build/task mode services (nil = still running)
	EndTime        time.Time
}

// healthCheckConfig holds custom healthcheck configuration from azure.yaml.
type healthCheckConfig struct {
	Test          []string
	Type          string // "http", "tcp", "process", "output", "none"
	Pattern       string // Regex pattern for output-based health checks
	Interval      time.Duration
	Timeout       time.Duration
	Retries       int
	StartPeriod   time.Duration
	StartInterval time.Duration
}

// httpHealthCheckResult holds the result of an HTTP health check.
type httpHealthCheckResult struct {
	Endpoint     string
	ResponseTime time.Duration
	StatusCode   int
	Status       HealthStatus
	Details      map[string]interface{}
	Error        string
}

// calculateSummary calculates health statistics from a slice of results.
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
		case HealthStatusStarting:
			summary.Starting++
		default:
			summary.Unknown++
		}
	}

	// Determine overall status
	// Starting services are treated as neutral - they don't affect overall health
	if summary.Unhealthy > 0 {
		summary.Overall = HealthStatusUnhealthy
	} else if summary.Degraded > 0 {
		summary.Overall = HealthStatusDegraded
	} else if summary.Healthy > 0 {
		summary.Overall = HealthStatusHealthy
	} else if summary.Starting > 0 {
		// All services are starting - overall is starting
		summary.Overall = HealthStatusStarting
	} else {
		summary.Overall = HealthStatusUnknown
	}

	return summary
}

// filterServices filters a list of services by name.
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
