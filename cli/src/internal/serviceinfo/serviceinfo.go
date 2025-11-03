package serviceinfo

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/registry"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

var (
	// environmentCache stores the latest environment variables from azd
	// This cache is refreshed when azd fires environment update events (e.g., after provision)
	environmentCache   map[string]string
	environmentCacheMu sync.RWMutex
)

func init() {
	environmentCache = make(map[string]string)
}

// RefreshEnvironmentCache updates the cached environment variables from the current process.
// This is called by the listen command when azd fires an "environment updated" event.
// By the time this is called, azd has already updated the process environment.
func RefreshEnvironmentCache() {
	environmentCacheMu.Lock()
	defer environmentCacheMu.Unlock()

	// Clear and repopulate the cache from current process environment
	environmentCache = make(map[string]string)

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		environmentCache[parts[0]] = parts[1]
	}
}

// RefreshEnvironmentFromEvent updates the cached environment variables from a provision event.
// This is called by the listen command when azd fires an "environment updated" event.
func RefreshEnvironmentFromEvent(bicepOutputs map[string]interface{}) {
	environmentCacheMu.Lock()
	defer environmentCacheMu.Unlock()

	// Extract environment variables from bicep outputs
	// Bicep outputs are typically in the format: { "outputName": { "value": "actualValue" } }
	for key, val := range bicepOutputs {
		if outputMap, ok := val.(map[string]interface{}); ok {
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
	Status      string     `json:"status"` // "running", "not-running", "unknown"
	Health      string     `json:"health"` // "healthy", "unhealthy", "unknown"
	URL         string     `json:"url,omitempty"`
	Port        int        `json:"port,omitempty"`
	PID         int        `json:"pid,omitempty"`
	StartTime   *time.Time `json:"startTime,omitempty"`
	LastChecked *time.Time `json:"lastChecked,omitempty"`
}

// AzureServiceInfo contains Azure-specific service information.
type AzureServiceInfo struct {
	URL          string `json:"url,omitempty"`
	ResourceName string `json:"resourceName,omitempty"`
	ImageName    string `json:"imageName,omitempty"`
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

	// Get Azure environment values
	azureEnv := getAzureEnvironmentValues(projectDir)

	// Extract Azure service information from environment
	azureServiceInfo := extractAzureServiceInfo(azureEnv)

	// Merge azure.yaml services with running services to get complete picture
	allServices := mergeServiceInfo(azureYaml, runningServices, azureServiceInfo)

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

// getAzureEnvironmentValues reads Azure environment variables from the current process environment.
// Since this is an azd extension, all azd environment variables are automatically available.
// Additionally, it merges in values from the event-driven environment cache which is updated
// when azd provision completes.
func getAzureEnvironmentValues(projectDir string) map[string]string {
	envVars := make(map[string]string)

	// Get all environment variables from current process (azd provides these)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]
		envVars[key] = value
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

// extractAzureServiceInfo extracts Azure service information from environment variables.
func extractAzureServiceInfo(envVars map[string]string) map[string]AzureServiceInfo {
	azureServices := make(map[string]AzureServiceInfo)

	for key, value := range envVars {
		keyUpper := strings.ToUpper(key)

		// Skip system variables
		if strings.Contains(keyUpper, "PIPE") || strings.Contains(keyUpper, "PATH") ||
			strings.Contains(keyUpper, "TEMP") || strings.Contains(keyUpper, "HOME") {
			continue
		}

		// Pattern 1 (highest priority): SERVICE_{SERVICE_NAME}_URL -> Azure URL
		if strings.HasPrefix(keyUpper, "SERVICE_") && strings.HasSuffix(keyUpper, "_URL") &&
			(strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")) {
			serviceName := strings.TrimPrefix(keyUpper, "SERVICE_")
			serviceName = strings.TrimSuffix(serviceName, "_URL")
			serviceName = strings.ToLower(serviceName)

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
			serviceName = strings.ToLower(serviceName)

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
			serviceName = strings.ToLower(serviceName)

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
			serviceName = strings.ToLower(serviceName)

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
			serviceName = strings.ToLower(serviceName)

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
func mergeServiceInfo(azureYaml *service.AzureYaml, runningServices []*registry.ServiceRegistryEntry, azureServices map[string]AzureServiceInfo) []*ServiceInfo {
	serviceMap := make(map[string]*ServiceInfo)

	// First, add all services from azure.yaml
	if azureYaml != nil {
		for name, svc := range azureYaml.Services {
			// Normalize service name to lowercase for case-insensitive matching
			normalizedName := strings.ToLower(name)
			serviceMap[normalizedName] = &ServiceInfo{
				Name:      name, // Preserve original casing for display
				Language:  svc.Language,
				Project:   svc.Project,
				Framework: detectFramework(svc),
				// Initialize with default local state
				Local: &LocalServiceInfo{
					Status: "not-running",
					Health: "unknown",
				},
			}
		}
	}

	// Overlay running service information
	for _, runningSvc := range runningServices {
		normalizedName := strings.ToLower(runningSvc.Name)
		if existing, exists := serviceMap[normalizedName]; exists {
			existing.Local = &LocalServiceInfo{
				Status:      runningSvc.Status,
				Health:      runningSvc.Health,
				URL:         runningSvc.URL,
				Port:        runningSvc.Port,
				PID:         runningSvc.PID,
				StartTime:   &runningSvc.StartTime,
				LastChecked: &runningSvc.LastChecked,
			}
		}
	}

	// Overlay Azure service information (only for services in azure.yaml)
	for serviceName, azureInfo := range azureServices {
		// serviceName from azureServices is already lowercase
		if existing, exists := serviceMap[serviceName]; exists {
			existing.Azure = &azureInfo
		}
	}

	// Convert map to slice
	var result []*ServiceInfo
	for _, svc := range serviceMap {
		result = append(result, svc)
	}

	return result
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
