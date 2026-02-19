// Package healthcheck provides health monitoring capabilities for services.
// Exported types are re-exported from azd-core/healthcheck.
// App-specific code (monitor, checker, metrics) remains local.
package healthcheck

import (
	"time"

	core "github.com/jongio/azd-core/healthcheck"
)

// Re-export public types from azd-core/healthcheck.
type HealthStatus = core.HealthStatus
type HealthCheckType = core.HealthCheckType
type HealthCheckResult = core.HealthCheckResult
type HealthReport = core.HealthReport
type HealthSummary = core.HealthSummary
type MonitorConfig = core.MonitorConfig
type ServiceInfo = core.ServiceInfo
type HealthCheckConfig = core.HealthCheckConfig
type HealthProfile = core.HealthProfile
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
