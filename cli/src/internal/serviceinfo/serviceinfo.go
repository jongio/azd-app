// Package serviceinfo provides comprehensive service information management,
// combining Azure YAML definitions, runtime state, and Azure environment data.
// It serves as the single source of truth for service info used by both the info command and dashboard.
package serviceinfo

import (
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/resourcealert"
	"github.com/jongio/azd-app/cli/src/internal/resourcesampler"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/registry"
)

var (
	// environmentCache stores the latest environment variables from azd
	// This cache is refreshed when azd fires environment update events (e.g., after provision)
	environmentCache   map[string]string
	environmentCacheMu sync.RWMutex

	// sampleResourceUsage reports CPU/memory for a running process tree. Kept
	// as a package var so tests can exercise the merge path deterministically
	// without depending on live process sampling.
	sampleResourceUsage = resourcesampler.Sample

	// resourceAlertEngine debounces per-service resource threshold warnings so a
	// service hovering over a limit does not flood the console across refreshes.
	resourceAlertEngine = resourcealert.NewEngine(0)
)

func init() {
	environmentCache = make(map[string]string)
}

// RefreshEnvironmentCache updates the cached environment variables from the current process.
// This is called by the listen command when azd fires an "environment updated" event.
// By the time this is called, azd has already updated the process environment.
func RefreshEnvironmentCache() {
	// Build new cache outside the lock to minimize lock duration
	newCache := make(map[string]string)

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		newCache[parts[0]] = parts[1]
	}

	// Atomic swap using copy-on-write pattern
	environmentCacheMu.Lock()
	environmentCache = newCache
	environmentCacheMu.Unlock()
}

// RefreshEnvironmentFromEvent updates the cached environment variables from a provision event.
// This is called by the listen command when azd fires an "environment updated" event.
func RefreshEnvironmentFromEvent(bicepOutputs map[string]any) {
	environmentCacheMu.Lock()
	defer environmentCacheMu.Unlock()

	// Extract environment variables from bicep outputs
	// Bicep outputs are typically in the format: { "outputName": { "value": "actualValue" } }
	for key, val := range bicepOutputs {
		if outputMap, ok := val.(map[string]any); ok {
			if value, ok := outputMap["value"].(string); ok {
				environmentCache[strings.ToUpper(key)] = value
			}
		}
	}
}

// ServiceInfo contains comprehensive information about a service.
type ServiceInfo struct {
	Name string `json:"name"`

	// Azure.yaml definition info
	Host      string `json:"host,omitempty"` // Host type from azure.yaml: "local", "containerapp", "appservice", "function", etc.
	Language  string `json:"language,omitempty"`
	Framework string `json:"framework,omitempty"`
	Project   string `json:"project,omitempty"`

	// Local development info (runtime state)
	Local *LocalServiceInfo `json:"local,omitempty"`

	// Azure environment info
	Azure *AzureServiceInfo `json:"azure,omitempty"`

	// Environment variables (Azure-related)
	EnvironmentVars map[string]string `json:"environmentVariables,omitempty"`
}

// LocalServiceInfo contains local development information.
type LocalServiceInfo struct {
	Status      string     `json:"status"`              // "running", "not-running", "unknown"
	Health      string     `json:"health"`              // "healthy", "unhealthy", "unknown"
	URL         string     `json:"url,omitempty"`       // Auto-discovered local URL
	CustomURL   string     `json:"customUrl,omitempty"` // User-configured custom URL (e.g., ngrok)
	Port        int        `json:"port,omitempty"`
	PID         int        `json:"pid,omitempty"`
	StartTime   *time.Time `json:"startTime,omitempty"`
	LastChecked *time.Time `json:"lastChecked,omitempty"`
	ServiceType string     `json:"serviceType,omitempty"` // "http", "tcp", "process", "container"
	ServiceMode string     `json:"serviceMode,omitempty"` // "watch", "build", "daemon", "task" (for type=process)
	// CPUPercent is the running process tree's CPU usage as a percentage of a
	// single core (can exceed 100 on multi-core workloads). Zero when the
	// service is not running or sampling failed.
	CPUPercent float64 `json:"cpuPercent,omitempty"`
	// MemoryBytes is the running process tree's resident memory in bytes. Zero
	// when the service is not running or sampling failed.
	MemoryBytes uint64 `json:"memoryBytes,omitempty"`
	// CPUThresholdPercent is the configured CPU alert threshold, if any.
	CPUThresholdPercent float64 `json:"cpuThresholdPercent,omitempty"`
	// MemoryThresholdMB is the configured memory alert threshold in MB, if any.
	MemoryThresholdMB uint64 `json:"memoryThresholdMB,omitempty"`
	// CPUOverThreshold is true when live CPU usage exceeds CPUThresholdPercent.
	CPUOverThreshold bool `json:"cpuOverThreshold,omitempty"`
	// MemoryOverThreshold is true when live memory usage exceeds MemoryThresholdMB.
	MemoryOverThreshold bool `json:"memoryOverThreshold,omitempty"`
}

// AzureServiceInfo contains Azure-specific service information.
type AzureServiceInfo struct {
	URL                string `json:"url,omitempty"`                // Auto-discovered Azure deployment URL
	CustomURL          string `json:"customUrl,omitempty"`          // User-configured custom URL
	CustomDomain       string `json:"customDomain,omitempty"`       // User-configured OR SDK-discovered custom domain
	CustomDomainSource string `json:"customDomainSource,omitempty"` // "user" or "azure-sdk"
	ResourceName       string `json:"resourceName,omitempty"`
	ImageName          string `json:"imageName,omitempty"`
}

// GetServiceInfo returns comprehensive service information for a project directory.
// This is the single source of truth for service info used by both the info command and dashboard.
func GetServiceInfo(projectDir string) ([]*ServiceInfo, error) {
	// Parse azure.yaml to get service definitions (if it exists)
	azureYaml, err := parseAzureYaml(projectDir)
	if err != nil {
		// Don't fail if azure.yaml doesn't exist, just return empty
		azureYaml = &service.AzureYaml{Services: make(map[string]service.Service)}
	}

	reg := registry.GetRegistry(projectDir)
	runningServices := reg.ListAll()

	// Get Azure environment values (all values from process + event cache)
	azureEnv := getAzureEnvironmentValues()

	// Extract Azure service information from environment
	azureServiceInfo := extractAzureServiceInfo(azureEnv)

	// Merge azure.yaml services with running services to get complete picture
	allServices := mergeServiceInfo(azureYaml, runningServices, azureServiceInfo, azureEnv)

	return allServices, nil
}

// parseAzureYaml parses azure.yaml from the project directory.
func parseAzureYaml(projectDir string) (*service.AzureYaml, error) {
	// Use service.ParseAzureYaml which handles path resolution correctly
	azureYaml, err := service.ParseAzureYaml(projectDir)
	if err != nil {
		// If azure.yaml not found, return empty structure
		if strings.Contains(err.Error(), "not found") {
			return &service.AzureYaml{Services: make(map[string]service.Service)}, nil
		}
		return nil, err
	}

	return azureYaml, nil
}

// getAzureEnvironmentValues gets Azure environment variables from the process.
// When running as an azd extension, all environment variables are injected by azd.
// We also merge in cached values from provision events.
func getAzureEnvironmentValues() map[string]string {
	envVars := make(map[string]string)

	// Get environment variables from process (azd injects all env vars when running extensions)
	for _, line := range os.Environ() {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			envVars[parts[0]] = parts[1]
		}
	}

	// Merge in the cached environment values from azd events (higher priority)
	// This ensures we have the latest values from provision operations
	environmentCacheMu.RLock()
	for key, value := range environmentCache {
		envVars[key] = value
	}
	environmentCacheMu.RUnlock()

	return envVars
}

// normalizeServiceName converts a service name from environment variable format to azure.yaml format.
// Environment variables use underscores (SERVICE_CONTAINERAPP_API_NAME) while azure.yaml uses hyphens (containerapp-api).
func normalizeServiceName(name string) string {
	// Convert to lowercase and replace underscores with hyphens
	return strings.ReplaceAll(strings.ToLower(name), "_", "-")
}

// extractAzureServiceInfo extracts Azure service information from environment variables.
func extractAzureServiceInfo(envVars map[string]string) map[string]AzureServiceInfo {
	azureServices := make(map[string]AzureServiceInfo)

	for key, value := range envVars {
		keyUpper := strings.ToUpper(key)

		// Skip system variables using prefix/exact matching to avoid false positives
		// (e.g., SERVICE_PIPELINE_URL should not be filtered)
		systemPrefixes := []string{"PATH", "TEMP", "TMP", "HOME", "COMSPEC", "WINDIR", "SYSTEMROOT", "PIPE"}
		isSystemVar := false
		for _, prefix := range systemPrefixes {
			if keyUpper == prefix || strings.HasPrefix(keyUpper, prefix+"_") {
				isSystemVar = true
				break
			}
		}
		if isSystemVar {
			continue
		}

		// Pattern 1 (highest priority): SERVICE_{SERVICE_NAME}_URL -> Azure URL
		if strings.HasPrefix(keyUpper, "SERVICE_") && strings.HasSuffix(keyUpper, "_URL") &&
			(strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")) {
			serviceName := strings.TrimPrefix(keyUpper, "SERVICE_")
			serviceName = strings.TrimSuffix(serviceName, "_URL")
			serviceName = normalizeServiceName(serviceName)

			if serviceName != "" {
				info := azureServices[serviceName]
				info.URL = value
				azureServices[serviceName] = info
			}
			continue
		}

		// Pattern 2: {SERVICE_NAME}_URL -> Azure URL (without SERVICE_ prefix)
		if strings.HasSuffix(keyUpper, "_URL") &&
			(strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")) {
			serviceName := strings.TrimSuffix(keyUpper, "_URL")
			serviceName = normalizeServiceName(serviceName)

			if serviceName != "" {
				// Only set if not already set by higher priority pattern
				if existing, exists := azureServices[serviceName]; !exists || existing.URL == "" {
					info := azureServices[serviceName]
					info.URL = value
					azureServices[serviceName] = info
				}
			}
		}

		// Pattern 1 (highest priority): SERVICE_{SERVICE_NAME}_NAME -> Azure resource name
		if strings.HasPrefix(keyUpper, "SERVICE_") && strings.HasSuffix(keyUpper, "_NAME") && !strings.HasSuffix(keyUpper, "_IMAGE_NAME") {
			serviceName := strings.TrimPrefix(keyUpper, "SERVICE_")
			serviceName = strings.TrimSuffix(serviceName, "_NAME")
			serviceName = normalizeServiceName(serviceName)

			if serviceName != "" {
				info := azureServices[serviceName]
				info.ResourceName = value
				azureServices[serviceName] = info
			}
			continue
		}

		// Pattern 2: {SERVICE_NAME}_NAME -> Azure resource name (without SERVICE_ prefix)
		if strings.HasSuffix(keyUpper, "_NAME") && !strings.HasSuffix(keyUpper, "_IMAGE_NAME") {
			serviceName := strings.TrimSuffix(keyUpper, "_NAME")
			serviceName = normalizeServiceName(serviceName)

			if serviceName != "" {
				// Only set if not already set by higher priority pattern
				if existing, exists := azureServices[serviceName]; !exists || existing.ResourceName == "" {
					info := azureServices[serviceName]
					info.ResourceName = value
					azureServices[serviceName] = info
				}
			}
		}

		// Pattern: SERVICE_{SERVICE_NAME}_IMAGE_NAME -> Docker image
		if strings.HasPrefix(keyUpper, "SERVICE_") && strings.HasSuffix(keyUpper, "_IMAGE_NAME") {
			serviceName := strings.TrimPrefix(keyUpper, "SERVICE_")
			serviceName = strings.TrimSuffix(serviceName, "_IMAGE_NAME")
			serviceName = normalizeServiceName(serviceName)

			if serviceName != "" {
				info := azureServices[serviceName]
				info.ImageName = value
				azureServices[serviceName] = info
			}
		}
	}

	return azureServices
}

// mergeServiceInfo combines azure.yaml services with running services and Azure info.
func mergeServiceInfo(azureYaml *service.AzureYaml, runningServices []*registry.ServiceRegistryEntry, azureServices map[string]AzureServiceInfo, envVars map[string]string) []*ServiceInfo {
	serviceMap := make(map[string]*ServiceInfo)
	thresholds := make(map[string]resourcealert.Threshold)

	// First, add all services from azure.yaml
	if azureYaml != nil && azureYaml.Services != nil {
		for name, svc := range azureYaml.Services {
			// Normalize service name to lowercase for case-insensitive matching
			normalizedName := strings.ToLower(name)
			serviceInfo := &ServiceInfo{
				Name:            name, // Preserve original casing for display
				Host:            svc.Host,
				Language:        svc.Language,
				Project:         svc.Project,
				Framework:       detectFramework(svc),
				EnvironmentVars: envVars, // Include Azure/AZD environment variables
				// Initialize with default local state
				Local: &LocalServiceInfo{
					Status: "not-running",
					Health: "unknown",
				},
			}

			// Extract local.customUrl from service config if present
			if svc.Local != nil && svc.Local.CustomURL != "" {
				serviceInfo.Local.CustomURL = svc.Local.CustomURL
			}

			// Extract Azure config from service config if present
			if svc.Azure != nil {
				// Initialize Azure info if not already present
				if serviceInfo.Azure == nil {
					serviceInfo.Azure = &AzureServiceInfo{}
				}
				if svc.Azure.CustomURL != "" {
					serviceInfo.Azure.CustomURL = svc.Azure.CustomURL
				}
				if svc.Azure.CustomDomain != "" {
					serviceInfo.Azure.CustomDomain = svc.Azure.CustomDomain
					serviceInfo.Azure.CustomDomainSource = svc.Azure.CustomDomainSource
				}
			}

			// Handle deprecated root-level URL field (backward compatibility)
			if svc.URL != "" {
				// Initialize Azure info if not already present
				if serviceInfo.Azure == nil {
					serviceInfo.Azure = &AzureServiceInfo{}
				}
				// Only set CustomURL if not already set from Azure.CustomURL
				if serviceInfo.Azure.CustomURL == "" {
					serviceInfo.Azure.CustomURL = svc.URL
				}
			}

			if svc.Resources != nil {
				thresholds[normalizedName] = resourcealert.Threshold{
					CPUPercent: svc.Resources.CPUPercent,
					MemoryMB:   svc.Resources.MemoryMB,
				}
			}

			serviceMap[normalizedName] = serviceInfo
		}
	}

	// Overlay running service information
	for _, runningSvc := range runningServices {
		normalizedName := strings.ToLower(runningSvc.Name)
		if existing, exists := serviceMap[normalizedName]; exists {
			// Preserve any locally configured custom URL while overlaying runtime state
			existingCustomURL := ""
			if existing.Local != nil {
				existingCustomURL = existing.Local.CustomURL
			}

			existing.Local = &LocalServiceInfo{
				Status:      runningSvc.Status,
				Health:      "", // Health is computed dynamically via health checks, not stored in registry
				URL:         runningSvc.URL,
				Port:        runningSvc.Port,
				PID:         runningSvc.PID,
				StartTime:   &runningSvc.StartTime,
				LastChecked: &runningSvc.LastChecked,
				ServiceType: runningSvc.Type,
				ServiceMode: runningSvc.Mode,
			}

			if existingCustomURL != "" {
				existing.Local.CustomURL = existingCustomURL
			}

			// Sample live CPU/memory for services with a known PID. Fail-soft:
			// a stale PID yields zero usage, which omitempty drops from output.
			if runningSvc.PID > 0 {
				usage := sampleResourceUsage(runningSvc.PID)
				existing.Local.CPUPercent = usage.CPUPercent
				existing.Local.MemoryBytes = usage.MemoryBytes
				if threshold, ok := thresholds[normalizedName]; ok {
					applyResourceThresholds(existing.Local, runningSvc.Name, usage, threshold)
				}
			}
		}
	}

	// Overlay Azure service information (only for services in azure.yaml)
	// Log when we have azure.yaml services but no Azure info discovered
	if len(serviceMap) > 0 && len(azureServices) == 0 {
		slog.Debug("No Azure service information discovered from environment variables",
			"azureYamlServices", len(serviceMap),
			"hint", "Ensure SERVICE_*_URL or SERVICE_*_NAME environment variables are set")
	}

	for serviceName, azureInfo := range azureServices {
		// serviceName from azureServices is already lowercase
		if existing, exists := serviceMap[serviceName]; exists {
			// Preserve custom URLs and domains from config
			var existingCustomURL, existingCustomDomain, existingCustomDomainSource string
			if existing.Azure != nil {
				existingCustomURL = existing.Azure.CustomURL
				existingCustomDomain = existing.Azure.CustomDomain
				existingCustomDomainSource = existing.Azure.CustomDomainSource
			}

			// Replace Azure info with environment-based info
			existing.Azure = &azureInfo

			// Restore custom URLs/domains from config (user config takes precedence over env vars)
			if existingCustomURL != "" {
				existing.Azure.CustomURL = existingCustomURL
			}
			if existingCustomDomain != "" {
				existing.Azure.CustomDomain = existingCustomDomain
				existing.Azure.CustomDomainSource = existingCustomDomainSource
			}
		}
	}

	// Convert map to slice
	result := make([]*ServiceInfo, 0, len(serviceMap))
	for _, svc := range serviceMap {
		result = append(result, svc)
	}

	return result
}

// applyResourceThresholds records the configured thresholds on the local info,
// sets the over-threshold indicators from the current sample, and raises a
// throttled warning for each dimension that is exceeded.
func applyResourceThresholds(local *LocalServiceInfo, serviceName string, usage resourcesampler.Usage, threshold resourcealert.Threshold) {
	if local == nil || !threshold.Configured() {
		return
	}

	local.CPUThresholdPercent = threshold.CPUPercent
	local.MemoryThresholdMB = threshold.MemoryMB

	now := time.Now()

	// Indicator state is a pure comparison, always reflecting the latest sample.
	for _, breach := range threshold.Exceeds(serviceName, usage.CPUPercent, usage.MemoryBytes, now) {
		switch breach.Kind {
		case resourcealert.KindCPU:
			local.CPUOverThreshold = true
		case resourcealert.KindMemory:
			local.MemoryOverThreshold = true
		}
	}

	// Warnings are debounced so a service hovering over a limit does not flood
	// the console across refreshes.
	for _, breach := range resourceAlertEngine.Evaluate(serviceName, usage.CPUPercent, usage.MemoryBytes, threshold, now) {
		switch breach.Kind {
		case resourcealert.KindCPU:
			slog.Warn("Service over CPU threshold",
				"service", serviceName,
				"cpuPercent", breach.Value,
				"thresholdPercent", breach.Limit)
		case resourcealert.KindMemory:
			slog.Warn("Service over memory threshold",
				"service", serviceName,
				"memoryMB", breach.Value,
				"thresholdMB", breach.Limit)
		}
	}
}

// detectFramework attempts to detect framework from service definition.
func detectFramework(svc service.Service) string {
	switch svc.Language {
	case "node":
		return "express"
	case "python":
		return "flask"
	case "dotnet":
		return "aspnetcore"
	default:
		return svc.Language
	}
}
