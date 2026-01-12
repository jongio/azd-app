package serviceinfo

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/registry"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/env"
	"github.com/jongio/azd-core/urlutil"
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
	Status      string     `json:"status"` // "running", "not-running", "unknown"
	Health      string     `json:"health"` // "healthy", "unhealthy", "unknown"
	URL         string     `json:"url,omitempty"`
	CustomURL   *string    `json:"customUrl"` // Custom URL from local.url config (always present, null if not set)
	Port        int        `json:"port,omitempty"`
	PID         int        `json:"pid,omitempty"`
	StartTime   *time.Time `json:"startTime,omitempty"`
	LastChecked *time.Time `json:"lastChecked,omitempty"`
	ServiceType string     `json:"serviceType,omitempty"` // "http", "tcp", "process", "container"
	ServiceMode string     `json:"serviceMode,omitempty"` // "watch", "build", "daemon", "task" (for type=process)
}

// AzureServiceInfo contains Azure-specific service information.
type AzureServiceInfo struct {
	URL          string  `json:"url,omitempty"`        // System-generated Azure URL (e.g., *.azurewebsites.net)
	CustomURL    *string `json:"customUrl"`            // Custom URL from azure.url config (always present, null if not set)
	CustomDomain *string `json:"customDomain"`         // Auto-detected custom domain URL (always present, null if not detected)
	ResourceName string  `json:"resourceName,omitempty"`
	ImageName    string  `json:"imageName,omitempty"`
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

	// Get Azure environment values (all values from azd env get-values)
	azureEnv := getAzureEnvironmentValues(projectDir)

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

// getAzureEnvironmentValues reads Azure environment variables from the process environment.
// When running as an azd extension, all Azure environment variables are already available
// via os.Environ() - no need to shell out to 'azd env get-values'.
// Additionally, it merges in values from the event-driven environment cache which is updated
// when azd provision completes.
func getAzureEnvironmentValues(projectDir string) map[string]string {
	envVars := make(map[string]string)

	// Get Azure environment variables from the process environment
	// The azd extension framework provides these automatically: AZURE_*, SERVICE_*
	// Use env.FilterByPrefixSlice for efficient filtering
	azureVars := env.FilterByPrefixSlice(os.Environ(), "AZURE_")
	serviceVars := env.FilterByPrefixSlice(os.Environ(), "SERVICE_")
	
	// Convert filtered slices to map
	for _, envVar := range append(azureVars, serviceVars...) {
		parts := strings.SplitN(envVar, "=", 2)
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

// extractAzureServiceInfo extracts Azure service information from environment variables.
func extractAzureServiceInfo(envVars map[string]string) map[string]AzureServiceInfo {
	azureServices := make(map[string]AzureServiceInfo)

	// Extract SERVICE_*_URL (highest priority)
	serviceURLs := env.ExtractPattern(envVars, env.PatternOptions{
		Prefix:     "SERVICE_",
		Suffix:     "_URL",
		TrimPrefix: true,
		TrimSuffix: true,
		Transform:  env.NormalizeServiceName,
		Validator: func(value string) bool {
			return urlutil.Validate(value) == nil
		},
	})

	// Filter out SERVICE_* keys for fallback extraction
	nonServiceEnvVars := make(map[string]string)
	for k, v := range envVars {
		if !strings.HasPrefix(strings.ToUpper(k), "SERVICE_") {
			nonServiceEnvVars[k] = v
		}
	}

	// Extract *_URL (lower priority) - only from non-SERVICE_ keys
	fallbackURLs := env.ExtractPattern(nonServiceEnvVars, env.PatternOptions{
		Suffix:     "_URL",
		TrimSuffix: true,
		Transform:  env.NormalizeServiceName,
		Validator: func(value string) bool {
			return urlutil.Validate(value) == nil
		},
	})

	// Merge with priority (serviceURLs overrides fallbackURLs)
	for name, url := range serviceURLs {
		info := azureServices[name]
		info.URL = url
		azureServices[name] = info
	}
	for name, url := range fallbackURLs {
		if existing, exists := azureServices[name]; !exists || existing.URL == "" {
			info := azureServices[name]
			info.URL = url
			azureServices[name] = info
		}
	}

	// Filter out _IMAGE_NAME keys and system variables for _NAME extraction
	filteredEnvVars := make(map[string]string)
	for k, v := range envVars {
		keyUpper := strings.ToUpper(k)
		// Skip _IMAGE_NAME suffix
		if strings.HasSuffix(keyUpper, "_IMAGE_NAME") {
			continue
		}
		// Skip system variables
		if strings.Contains(keyUpper, "PIPE") || strings.Contains(keyUpper, "PATH") ||
			strings.Contains(keyUpper, "TEMP") || strings.Contains(keyUpper, "HOME") {
			continue
		}
		filteredEnvVars[k] = v
	}

	// Extract SERVICE_*_NAME (highest priority)
	serviceNames := env.ExtractPattern(filteredEnvVars, env.PatternOptions{
		Prefix:     "SERVICE_",
		Suffix:     "_NAME",
		TrimPrefix: true,
		TrimSuffix: true,
		Transform:  env.NormalizeServiceName,
	})

	// Filter out SERVICE_* keys for fallback extraction
	nonServiceFilteredEnvVars := make(map[string]string)
	for k, v := range filteredEnvVars {
		if !strings.HasPrefix(strings.ToUpper(k), "SERVICE_") {
			nonServiceFilteredEnvVars[k] = v
		}
	}

	// Extract *_NAME (lower priority) - only from non-SERVICE_ keys
	fallbackNames := env.ExtractPattern(nonServiceFilteredEnvVars, env.PatternOptions{
		Suffix:     "_NAME",
		TrimSuffix: true,
		Transform:  env.NormalizeServiceName,
	})

	// Merge with priority (serviceNames overrides fallbackNames)
	for name, resourceName := range serviceNames {
		info := azureServices[name]
		info.ResourceName = resourceName
		azureServices[name] = info
	}
	for name, resourceName := range fallbackNames {
		if existing, exists := azureServices[name]; !exists || existing.ResourceName == "" {
			info := azureServices[name]
			info.ResourceName = resourceName
			azureServices[name] = info
		}
	}

	// Extract SERVICE_*_IMAGE_NAME
	imageNames := env.ExtractPattern(envVars, env.PatternOptions{
		Prefix:     "SERVICE_",
		Suffix:     "_IMAGE_NAME",
		TrimPrefix: true,
		TrimSuffix: true,
		Transform:  env.NormalizeServiceName,
	})

	for name, imageName := range imageNames {
		info := azureServices[name]
		info.ImageName = imageName
		azureServices[name] = info
	}

	// Extract SERVICE_*_CUSTOM_DOMAIN with HTTPS validation
	customDomains := env.ExtractPattern(envVars, env.PatternOptions{
		Prefix:     "SERVICE_",
		Suffix:     "_CUSTOM_DOMAIN",
		TrimPrefix: true,
		TrimSuffix: true,
		Transform:  env.NormalizeServiceName,
		Validator: func(value string) bool {
			return urlutil.ValidateHTTPSOnly(value) == nil
		},
	})

	for name, customDomain := range customDomains {
		info := azureServices[name]
		info.CustomDomain = &customDomain
		azureServices[name] = info
	}

	return azureServices
}

// mergeServiceInfo combines azure.yaml services with running services and Azure info.
func mergeServiceInfo(azureYaml *service.AzureYaml, runningServices []*registry.ServiceRegistryEntry, azureServices map[string]AzureServiceInfo, envVars map[string]string) []*ServiceInfo {
	serviceMap := make(map[string]*ServiceInfo)

	// First, add all services from azure.yaml
	if azureYaml != nil {
		for name, svc := range azureYaml.Services {
			// Normalize service name to lowercase for case-insensitive matching
			normalizedName := strings.ToLower(name)
			// Map Local.CustomURL from azure.yaml config (null if not set)
			var localCustomURL *string
			if svc.Local != nil && svc.Local.CustomURL != "" {
				localCustomURL = &svc.Local.CustomURL
			}

			// Map Azure.CustomURL from azure.yaml config (null if not set)
			var azureCustomURL *string
			if svc.Azure != nil && svc.Azure.CustomURL != "" {
				azureCustomURL = &svc.Azure.CustomURL
			}

			// Map user-provided Azure.CustomDomain from azure.yaml config (null if not set)
			// User-provided customDomain takes precedence over auto-detected
			var azureCustomDomain *string
			if svc.Azure != nil && svc.Azure.CustomDomain != "" {
				azureCustomDomain = &svc.Azure.CustomDomain
			}

			serviceInfo := &ServiceInfo{
				Name:            name, // Preserve original casing for display
				Host:            svc.Host,
				Language:        svc.Language,
				Project:         svc.Project,
				Framework:       detectFramework(svc),
				EnvironmentVars: envVars, // Include Azure/AZD environment variables
				// Initialize with default local state
				Local: &LocalServiceInfo{
					Status:    "not-running",
					Health:    "unknown",
					CustomURL: localCustomURL, // Always present, null if not set
				},
				// Initialize Azure info with CustomURL and CustomDomain from config
				Azure: &AzureServiceInfo{
					CustomURL:    azureCustomURL,    // Always present, null if not set
					CustomDomain: azureCustomDomain, // User-provided OR auto-detected (set here if user-provided)
				},
			}

			serviceMap[normalizedName] = serviceInfo
		}
	}

	// Overlay running service information
	for _, runningSvc := range runningServices {
		normalizedName := strings.ToLower(runningSvc.Name)
		if existing, exists := serviceMap[normalizedName]; exists {
			// Preserve customUrl from config
			customURL := existing.Local.CustomURL

			existing.Local = &LocalServiceInfo{
				Status:      runningSvc.Status,
				Health:      "", // Health is computed dynamically via health checks, not stored in registry
				URL:         runningSvc.URL,
				CustomURL:   customURL, // Preserve from config
				Port:        runningSvc.Port,
				PID:         runningSvc.PID,
				StartTime:   &runningSvc.StartTime,
				LastChecked: &runningSvc.LastChecked,
				ServiceType: runningSvc.Type,
				ServiceMode: runningSvc.Mode,
			}
		}
	}

	// Overlay Azure service information (only for services in azure.yaml)
	for serviceName, azureInfo := range azureServices {
		// serviceName from azureServices is already lowercase
		if existing, exists := serviceMap[serviceName]; exists {
			// Preserve CustomURL from config - NEVER from environment
			// CustomDomain: user-provided (from config) takes precedence over auto-detected
			if existing.Azure == nil {
				existing.Azure = &AzureServiceInfo{}
			}

			// Preserve user-provided CustomURL from config (already set during initialization)
			customURL := existing.Azure.CustomURL
			// Preserve user-provided CustomDomain from config
			userProvidedCustomDomain := existing.Azure.CustomDomain

			// Update with environment-based info
			existing.Azure.URL = azureInfo.URL // System-generated URL from environment
			existing.Azure.ResourceName = azureInfo.ResourceName
			existing.Azure.ImageName = azureInfo.ImageName

			// Restore CustomURL - always from config, never from environment
			existing.Azure.CustomURL = customURL

			// Set CustomDomain: user-provided takes precedence over auto-detected
			// User-provided: already set from config (non-nil if user specified it)
			// Auto-detected: comes from azureInfo (will be set here if user didn't provide)
			if userProvidedCustomDomain != nil {
				// User provided a customDomain in azure.yaml - use that
				existing.Azure.CustomDomain = userProvidedCustomDomain
			} else if azureInfo.CustomDomain != nil {
				// No user-provided value, use auto-detected from environment
				existing.Azure.CustomDomain = azureInfo.CustomDomain
			} else {
				// Neither user-provided nor auto-detected - set to nil
				existing.Azure.CustomDomain = nil
			}
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
