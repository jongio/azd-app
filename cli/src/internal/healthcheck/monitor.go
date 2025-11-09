// Package healthcheck provides health monitoring capabilities for services.
package healthcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/registry"
	"github.com/jongio/azd-app/cli/src/internal/service"
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
	ProjectDir      string
	DefaultEndpoint string
	Timeout         time.Duration
	Verbose         bool
}

// HealthMonitor coordinates health checking operations.
type HealthMonitor struct {
	config   MonitorConfig
	registry *registry.ServiceRegistry
	checker  *HealthChecker
}

// NewHealthMonitor creates a new health monitor.
func NewHealthMonitor(config MonitorConfig) (*HealthMonitor, error) {
	// Get service registry
	reg := registry.GetRegistry(config.ProjectDir)

	// Create health checker
	checker := &HealthChecker{
		timeout:         config.Timeout,
		defaultEndpoint: config.DefaultEndpoint,
		httpClient: &http.Client{
			Timeout: config.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}

	return &HealthMonitor{
		config:   config,
		registry: reg,
		checker:  checker,
	}, nil
}

// Check performs health checks on all or filtered services.
func (m *HealthMonitor) Check(ctx context.Context, serviceFilter []string) (*HealthReport, error) {
	// Load azure.yaml to get service definitions
	azureYaml, err := m.loadAzureYaml()
	if err != nil {
		// If no azure.yaml, just use registry
		if m.config.Verbose {
			fmt.Fprintf(os.Stderr, "Warning: Could not load azure.yaml: %v\n", err)
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
			semaphore <- struct{}{}
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

	report := &HealthReport{
		Timestamp: time.Now(),
		Project:   m.config.ProjectDir,
		Services:  results,
		Summary:   summary,
	}

	// Update registry with health status
	m.updateRegistry(results)

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
	// Docker Compose style healthcheck parsing
	// Note: This requires the Service type to have a HealthCheck field
	// which should be added in future when Docker Compose integration is implemented.
	// For now, we check if any health-related configuration exists.

	// Check if service has explicit health configuration (future enhancement)
	// When Service type includes healthcheck field from Docker Compose format:
	// type Service struct {
	//     HealthCheck struct {
	//         Test        []string      `yaml:"test"`
	//         Interval    time.Duration `yaml:"interval"`
	//         Timeout     time.Duration `yaml:"timeout"`
	//         Retries     int           `yaml:"retries"`
	//         StartPeriod time.Duration `yaml:"start_period"`
	//     } `yaml:"healthcheck"`
	// }

	// Return nil for now - caller handles gracefully
	return nil
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
	for _, result := range results {
		status := "running"
		if result.Status == HealthStatusUnhealthy {
			status = "error"
		}

		// Update registry with health status
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

// HealthChecker performs health checks on services.
type HealthChecker struct {
	timeout         time.Duration
	defaultEndpoint string
	httpClient      *http.Client
}

// CheckService performs a health check on a single service using cascading strategy.
func (c *HealthChecker) CheckService(ctx context.Context, svc serviceInfo) HealthCheckResult {
	result := HealthCheckResult{
		ServiceName: svc.Name,
		Timestamp:   time.Now(),
	}

	// Calculate uptime if we have start time
	if !svc.StartTime.IsZero() {
		result.Uptime = time.Since(svc.StartTime)
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
		defer resp.Body.Close()

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
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
			if err == nil {
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
		}

		return result
	}

	return nil
}

func (c *HealthChecker) checkPort(ctx context.Context, port int) bool {
	address := fmt.Sprintf("localhost:%d", port)
	conn, err := net.DialTimeout("tcp", address, defaultPortCheckTimeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func isProcessRunning(pid int) bool {
	// Try to find the process
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, signal 0 can be used to check if process exists
	// On Windows, this doesn't work reliably, so we just return true if FindProcess succeeded
	if err := process.Signal(os.Signal(nil)); err != nil {
		return false
	}

	return true
}
