package serviceinfo

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/registry"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

func TestRefreshEnvironmentCache(t *testing.T) {
	// Save original environment
	originalEnv := os.Environ()
	defer func() {
		// Restore original environment
		os.Clearenv()
		for _, env := range originalEnv {
			parts := splitEnv(env)
			if len(parts) == 2 {
				os.Setenv(parts[0], parts[1])
			}
		}
	}()

	tests := []struct {
		name     string
		envVars  map[string]string
		wantKeys []string
	}{
		{
			name: "basic environment variables",
			envVars: map[string]string{
				"SERVICE_API_URL":  "https://api.example.com",
				"SERVICE_WEB_URL":  "https://web.example.com",
				"SERVICE_API_NAME": "my-api",
			},
			wantKeys: []string{"SERVICE_API_URL", "SERVICE_WEB_URL", "SERVICE_API_NAME"},
		},
		{
			name: "mixed case variables",
			envVars: map[string]string{
				"service_api_url": "https://api.example.com",
				"API_URL":         "https://other.example.com",
			},
			wantKeys: []string{"service_api_url", "API_URL"},
		},
		{
			name:     "empty environment",
			envVars:  map[string]string{},
			wantKeys: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Clearenv()

			// Set test environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Refresh cache
			RefreshEnvironmentCache()

			// Verify cache contains expected keys
			environmentCacheMu.RLock()
			defer environmentCacheMu.RUnlock()

			for _, key := range tt.wantKeys {
				if _, exists := environmentCache[key]; !exists {
					t.Errorf("expected cache to contain key %q, but it was missing", key)
				}
			}

			// Verify cache values match environment
			for key, expectedValue := range tt.envVars {
				if cachedValue, exists := environmentCache[key]; !exists {
					t.Errorf("cache missing key %q", key)
				} else if cachedValue != expectedValue {
					t.Errorf("cache[%q] = %q, want %q", key, cachedValue, expectedValue)
				}
			}
		})
	}
}

func TestRefreshEnvironmentCache_ConcurrentAccess(t *testing.T) {
	// Save original environment
	originalEnv := os.Environ()
	defer func() {
		os.Clearenv()
		for _, env := range originalEnv {
			parts := splitEnv(env)
			if len(parts) == 2 {
				os.Setenv(parts[0], parts[1])
			}
		}
	}()

	// Set some test environment variables
	os.Clearenv()
	os.Setenv("TEST_VAR_1", "value1")
	os.Setenv("TEST_VAR_2", "value2")

	// Test concurrent access to cache
	var wg sync.WaitGroup
	iterations := 100

	// Concurrent refreshes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				RefreshEnvironmentCache()
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				getAzureEnvironmentValues("")
			}
		}(i)
	}

	wg.Wait()

	// Verify cache is still consistent
	environmentCacheMu.RLock()
	defer environmentCacheMu.RUnlock()

	if val, exists := environmentCache["TEST_VAR_1"]; !exists || val != "value1" {
		t.Errorf("cache inconsistent after concurrent access: TEST_VAR_1 = %q, exists = %v", val, exists)
	}
}

func TestGetAzureEnvironmentValues_MergesCache(t *testing.T) {
	// Save original environment
	originalEnv := os.Environ()
	defer func() {
		os.Clearenv()
		for _, env := range originalEnv {
			parts := splitEnv(env)
			if len(parts) == 2 {
				os.Setenv(parts[0], parts[1])
			}
		}
		// Clear cache
		environmentCacheMu.Lock()
		environmentCache = make(map[string]string)
		environmentCacheMu.Unlock()
	}()

	// Set process environment
	os.Clearenv()
	os.Setenv("PROCESS_VAR", "from_process")
	os.Setenv("SHARED_VAR", "process_value")

	// Manually populate cache with different values
	environmentCacheMu.Lock()
	environmentCache["CACHE_VAR"] = "from_cache"
	environmentCache["SHARED_VAR"] = "cache_value" // Cache should win
	environmentCacheMu.Unlock()

	// Get merged environment values
	result := getAzureEnvironmentValues("")

	// Verify process-only variable
	if result["PROCESS_VAR"] != "from_process" {
		t.Errorf("PROCESS_VAR = %q, want %q", result["PROCESS_VAR"], "from_process")
	}

	// Verify cache-only variable
	if result["CACHE_VAR"] != "from_cache" {
		t.Errorf("CACHE_VAR = %q, want %q", result["CACHE_VAR"], "from_cache")
	}

	// Verify cache takes priority over process for shared variables
	if result["SHARED_VAR"] != "cache_value" {
		t.Errorf("SHARED_VAR = %q, want %q (cache should override process)", result["SHARED_VAR"], "cache_value")
	}
}

func TestExtractAzureServiceInfo_EnvironmentPatterns(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		wantInfo map[string]AzureServiceInfo
	}{
		{
			name: "SERVICE_ prefix pattern (highest priority)",
			envVars: map[string]string{
				"SERVICE_API_URL":  "https://api.azure.com",
				"SERVICE_API_NAME": "my-api-resource",
			},
			wantInfo: map[string]AzureServiceInfo{
				"api": {
					URL:          "https://api.azure.com",
					ResourceName: "my-api-resource",
				},
			},
		},
		{
			name: "simple pattern without SERVICE_ prefix",
			envVars: map[string]string{
				"API_URL":  "https://api.example.com",
				"WEB_NAME": "my-web-app",
			},
			wantInfo: map[string]AzureServiceInfo{
				"api": {
					URL: "https://api.example.com",
				},
				"web": {
					ResourceName: "my-web-app",
				},
			},
		},
		{
			name: "priority - SERVICE_ prefix wins over simple pattern",
			envVars: map[string]string{
				"SERVICE_API_URL": "https://high-priority.com",
				"API_URL":         "https://low-priority.com",
			},
			wantInfo: map[string]AzureServiceInfo{
				"api": {
					URL: "https://high-priority.com",
				},
			},
		},
		{
			name: "image name pattern",
			envVars: map[string]string{
				"SERVICE_API_IMAGE_NAME": "myregistry.azurecr.io/api:latest",
			},
			wantInfo: map[string]AzureServiceInfo{
				"api": {
					ImageName: "myregistry.azurecr.io/api:latest",
				},
			},
		},
		{
			name: "filters out system variables",
			envVars: map[string]string{
				"SERVICE_API_URL": "https://api.example.com",
				"PATH":            "/usr/bin:/bin",
				"TEMP":            "/tmp",
				"HOME":            "/home/user",
				"PIPE_NAME":       "some-pipe",
			},
			wantInfo: map[string]AzureServiceInfo{
				"api": {
					URL: "https://api.example.com",
				},
			},
		},
		{
			name: "non-URL values ignored for _URL suffix",
			envVars: map[string]string{
				"SERVICE_API_URL": "not-a-url",
				"WEB_URL":         "also-not-a-url",
			},
			wantInfo: map[string]AzureServiceInfo{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractAzureServiceInfo(tt.envVars)

			// Check expected services exist
			for serviceName, expectedInfo := range tt.wantInfo {
				actualInfo, exists := result[serviceName]
				if !exists {
					t.Errorf("expected service %q not found in result", serviceName)
					continue
				}

				if actualInfo.URL != expectedInfo.URL {
					t.Errorf("service %q: URL = %q, want %q", serviceName, actualInfo.URL, expectedInfo.URL)
				}
				if actualInfo.ResourceName != expectedInfo.ResourceName {
					t.Errorf("service %q: ResourceName = %q, want %q", serviceName, actualInfo.ResourceName, expectedInfo.ResourceName)
				}
				if actualInfo.ImageName != expectedInfo.ImageName {
					t.Errorf("service %q: ImageName = %q, want %q", serviceName, actualInfo.ImageName, expectedInfo.ImageName)
				}
			}

			// Check no unexpected services
			for serviceName := range result {
				if _, expected := tt.wantInfo[serviceName]; !expected {
					t.Errorf("unexpected service %q in result", serviceName)
				}
			}
		})
	}
}

func TestRefreshEnvironmentFromEvent(t *testing.T) {
	// Clear cache first
	environmentCacheMu.Lock()
	environmentCache = make(map[string]string)
	environmentCacheMu.Unlock()

	tests := []struct {
		name          string
		bicepOutputs  map[string]interface{}
		wantCacheKeys map[string]string
	}{
		{
			name: "bicep outputs with value field",
			bicepOutputs: map[string]interface{}{
				"apiUrl": map[string]interface{}{
					"value": "https://api.azure.com",
					"type":  "string",
				},
				"webUrl": map[string]interface{}{
					"value": "https://web.azure.com",
				},
			},
			wantCacheKeys: map[string]string{
				"APIURL": "https://api.azure.com",
				"WEBURL": "https://web.azure.com",
			},
		},
		{
			name: "mixed output types - only strings extracted",
			bicepOutputs: map[string]interface{}{
				"apiUrl": map[string]interface{}{
					"value": "https://api.azure.com",
				},
				"port": map[string]interface{}{
					"value": 8080, // Not a string, should be ignored
				},
				"enabled": map[string]interface{}{
					"value": true, // Not a string, should be ignored
				},
			},
			wantCacheKeys: map[string]string{
				"APIURL": "https://api.azure.com",
			},
		},
		{
			name: "outputs without value field ignored",
			bicepOutputs: map[string]interface{}{
				"apiUrl": "just-a-string", // Not a map, should be ignored
				"config": map[string]interface{}{
					"setting": "value", // No "value" key
				},
			},
			wantCacheKeys: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear cache before each test
			environmentCacheMu.Lock()
			environmentCache = make(map[string]string)
			environmentCacheMu.Unlock()

			// Call function
			RefreshEnvironmentFromEvent(tt.bicepOutputs)

			// Verify cache
			environmentCacheMu.RLock()
			defer environmentCacheMu.RUnlock()

			for key, expectedValue := range tt.wantCacheKeys {
				if cachedValue, exists := environmentCache[key]; !exists {
					t.Errorf("expected cache to contain key %q", key)
				} else if cachedValue != expectedValue {
					t.Errorf("cache[%q] = %q, want %q", key, cachedValue, expectedValue)
				}
			}

			// Verify no unexpected keys
			for key := range environmentCache {
				if _, expected := tt.wantCacheKeys[key]; !expected {
					t.Errorf("unexpected key %q in cache", key)
				}
			}
		})
	}
}

// Helper function to split environment variable string
func splitEnv(env string) []string {
	for i := 0; i < len(env); i++ {
		if env[i] == '=' {
			return []string{env[:i], env[i+1:]}
		}
	}
	return []string{env}
}

func TestDetectFramework(t *testing.T) {
	tests := []struct {
		name     string
		language string
		expected string
	}{
		{
			name:     "node language",
			language: "node",
			expected: "express",
		},
		{
			name:     "python language",
			language: "python",
			expected: "flask",
		},
		{
			name:     "dotnet language",
			language: "dotnet",
			expected: "aspnetcore",
		},
		{
			name:     "unknown language returns language itself",
			language: "java",
			expected: "java",
		},
		{
			name:     "empty language",
			language: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.Service{Language: tt.language}
			result := detectFramework(svc)
			if result != tt.expected {
				t.Errorf("detectFramework(%v) = %q, want %q", svc, result, tt.expected)
			}
		})
	}
}

func TestMergeServiceInfo(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name            string
		azureYaml       *service.AzureYaml
		runningServices []*registry.ServiceRegistryEntry
		azureServices   map[string]AzureServiceInfo
		expectedCount   int
		checkService    string
		expectRunning   bool
	}{
		{
			name: "merge azure.yaml with running service",
			azureYaml: &service.AzureYaml{
				Services: map[string]service.Service{
					"api": {Language: "node", Project: "./api"},
				},
			},
			runningServices: []*registry.ServiceRegistryEntry{
				{
					Name:      "api",
					Status:    "running",
					Health:    "healthy",
					Port:      3000,
					StartTime: now,
				},
			},
			azureServices: map[string]AzureServiceInfo{},
			expectedCount: 1,
			checkService:  "api",
			expectRunning: true,
		},
		{
			name: "service in azure.yaml but not running",
			azureYaml: &service.AzureYaml{
				Services: map[string]service.Service{
					"web": {Language: "python", Project: "./web"},
				},
			},
			runningServices: []*registry.ServiceRegistryEntry{},
			azureServices:   map[string]AzureServiceInfo{},
			expectedCount:   1,
			checkService:    "web",
			expectRunning:   false,
		},
		{
			name: "case insensitive service matching",
			azureYaml: &service.AzureYaml{
				Services: map[string]service.Service{
					"API": {Language: "node"},
				},
			},
			runningServices: []*registry.ServiceRegistryEntry{
				{Name: "api", Status: "running", Port: 3000},
			},
			azureServices: map[string]AzureServiceInfo{},
			expectedCount: 1,
			checkService:  "API",
			expectRunning: true,
		},
		{
			name:            "empty inputs",
			azureYaml:       &service.AzureYaml{Services: make(map[string]service.Service)},
			runningServices: []*registry.ServiceRegistryEntry{},
			azureServices:   map[string]AzureServiceInfo{},
			expectedCount:   0,
		},
		{
			name: "with azure service info",
			azureYaml: &service.AzureYaml{
				Services: map[string]service.Service{
					"api": {Language: "node"},
				},
			},
			runningServices: []*registry.ServiceRegistryEntry{},
			azureServices: map[string]AzureServiceInfo{
				"api": {URL: "https://api.azurewebsites.net"},
			},
			expectedCount: 1,
			checkService:  "api",
			expectRunning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeServiceInfo(tt.azureYaml, tt.runningServices, tt.azureServices)
			
			if len(result) != tt.expectedCount {
				t.Errorf("mergeServiceInfo() returned %d services, want %d", len(result), tt.expectedCount)
			}
			
			if tt.checkService != "" {
				found := false
				for _, svc := range result {
					if svc.Name == tt.checkService {
						found = true
						if tt.expectRunning {
							if svc.Local == nil || svc.Local.Status != "running" {
								t.Errorf("service %q should be running", tt.checkService)
							}
						} else {
							if svc.Local != nil && svc.Local.Status == "running" {
								t.Errorf("service %q should not be running", tt.checkService)
							}
						}
					}
				}
				if !found {
					t.Errorf("service %q not found in results", tt.checkService)
				}
			}
		})
	}
}

func TestParseAzureYaml(t *testing.T) {
	tests := []struct {
		name      string
		setupDir  func(t *testing.T) string
		wantError bool
	}{
		{
			name: "valid azure.yaml",
			setupDir: func(t *testing.T) string {
				tmpDir := t.TempDir()
				yamlContent := `name: test-app
services:
  api:
    language: node
    project: ./api
`
				if err := os.WriteFile(filepath.Join(tmpDir, "azure.yaml"), []byte(yamlContent), 0600); err != nil {
					t.Fatalf("failed to create azure.yaml: %v", err)
				}
				return tmpDir
			},
			wantError: false,
		},
		{
			name: "missing azure.yaml returns empty structure",
			setupDir: func(t *testing.T) string {
				return t.TempDir()
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setupDir(t)
			result, err := parseAzureYaml(dir)
			
			if tt.wantError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result == nil {
				t.Error("parseAzureYaml() returned nil result")
			}
		})
	}
}

func TestGetServiceInfo(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create a simple azure.yaml
	yamlContent := `name: test
services:
  api:
    language: node
`
	if err := os.WriteFile(filepath.Join(tmpDir, "azure.yaml"), []byte(yamlContent), 0600); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}
	
	services, err := GetServiceInfo(tmpDir)
	if err != nil {
		t.Errorf("GetServiceInfo() error = %v", err)
	}
	
	if len(services) == 0 {
		t.Error("GetServiceInfo() returned no services")
	}
	
	// Should find the api service
	found := false
	for _, svc := range services {
		if svc.Name == "api" {
			found = true
			if svc.Language != "node" {
				t.Errorf("api service language = %q, want %q", svc.Language, "node")
			}
		}
	}
	if !found {
		t.Error("api service not found in results")
	}
}
