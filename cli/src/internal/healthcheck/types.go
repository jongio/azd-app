// Package healthcheck provides health monitoring capabilities for services.
// Exported types are re-exported from azd-core/healthcheck.
// App-specific code (monitor, checker, metrics) remains local.
package healthcheck

import (
	"time"

	core "github.com/jongio/azd-core/healthcheck"
)

// HealthStatus re-exports the core health status enumeration for monitored services.
type HealthStatus = core.HealthStatus

// HealthCheckType re-exports the core health check type enumeration.
type HealthCheckType = core.HealthCheckType

// HealthCheckResult re-exports the core health check result for an individual monitored service.
type HealthCheckResult = core.HealthCheckResult

// HealthReport re-exports the core health report returned by monitoring operations.
type HealthReport = core.HealthReport

// HealthSummary re-exports the aggregate core health summary across monitored services.
type HealthSummary = core.HealthSummary

// MonitorConfig re-exports the core monitor configuration used to set up health monitoring.
type MonitorConfig = core.MonitorConfig

// ServiceInfo re-exports the core service metadata used during health checks.
type ServiceInfo = core.ServiceInfo

// HealthCheckConfig re-exports the core probe configuration for a single health check.
type HealthCheckConfig = core.HealthCheckConfig

// HealthProfile re-exports the core named health profile definition loaded from configuration.
type HealthProfile = core.HealthProfile

// HealthProfiles re-exports the collection of named core health profiles.
type HealthProfiles = core.HealthProfiles

// Local aliases for backward compatibility with unexported references.
type serviceInfo = core.ServiceInfo
type healthCheckConfig = core.HealthCheckConfig

// Re-export constants from azd-core/healthcheck.
const (
	HealthStatusHealthy   = core.HealthStatusHealthy
	HealthStatusDegraded  = core.HealthStatusDegraded
	HealthStatusUnhealthy = core.HealthStatusUnhealthy
	HealthStatusStarting  = core.HealthStatusStarting
	HealthStatusUnknown   = core.HealthStatusUnknown

	HealthCheckTypeHTTP    = core.HealthCheckTypeHTTP
	HealthCheckTypeTCP     = core.HealthCheckTypeTCP
	HealthCheckTypeProcess = core.HealthCheckTypeProcess

	HTTPIdleConnTimeout       = core.HTTPIdleConnTimeout
	HTTPDialTimeout           = core.HTTPDialTimeout
	HTTPKeepAliveTimeout      = core.HTTPKeepAliveTimeout
	HTTPTLSHandshakeTimeout   = core.HTTPTLSHandshakeTimeout
	HTTPExpectContinueTimeout = core.HTTPExpectContinueTimeout

	ServiceTypeProcess   = core.ServiceTypeProcess
	ServiceTypeContainer = core.ServiceTypeContainer
	ServiceModeWatch     = core.ServiceModeWatch
	ServiceModeBuild     = core.ServiceModeBuild
	ServiceModeTask      = core.ServiceModeTask
)

// Re-export functions from azd-core/healthcheck.
var (
	FilterServices     = core.FilterServices
	LoadHealthProfiles = core.LoadHealthProfiles
	SaveSampleProfiles = core.SaveSampleProfiles
)

// Local alias for backward compatibility with monitor.go's lowercase reference.
var filterServices = core.FilterServices

// --- Unexported definitions kept locally (needed by checker.go and monitor.go) ---

const (
	maxConcurrentChecks     = 10
	maxResponseBodySize     = 1024 * 1024
	defaultPortCheckTimeout = 2 * time.Second
	startupGracePeriod      = 30 * time.Second
	endpointCacheNone       = "__none__"
)

var commonHealthPaths = []string{
	"/health",
	"/healthz",
	"/ready",
	"/alive",
	"/ping",
}

type httpHealthCheckResult struct {
	Endpoint     string
	ResponseTime time.Duration
	StatusCode   int
	Status       HealthStatus
	Details      map[string]interface{}
	Error        string
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
		case HealthStatusStarting:
			summary.Starting++
		default:
			summary.Unknown++
		}
	}

	if summary.Unhealthy > 0 {
		summary.Overall = HealthStatusUnhealthy
	} else if summary.Degraded > 0 {
		summary.Overall = HealthStatusDegraded
	} else if summary.Healthy > 0 {
		summary.Overall = HealthStatusHealthy
	} else if summary.Starting > 0 {
		summary.Overall = HealthStatusStarting
	} else {
		summary.Overall = HealthStatusUnknown
	}

	return summary
}
