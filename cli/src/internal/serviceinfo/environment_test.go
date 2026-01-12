package serviceinfo

import (
	"os"
	"strings"
	"sync"
	"testing"
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
	// Clear cache before test
	environmentCacheMu.Lock()
	environmentCache = make(map[string]string)
	environmentCacheMu.Unlock()

	defer func() {
		// Clear cache after test
		environmentCacheMu.Lock()
		environmentCache = make(map[string]string)
		environmentCacheMu.Unlock()
	}()

	// Manually populate cache with values
	environmentCacheMu.Lock()
	environmentCache["AZURE_CACHE_VAR"] = "from_cache"
	environmentCache["MY_CUSTOM_VAR"] = "custom_value"
	environmentCacheMu.Unlock()

	// Get merged environment values (azd env get-values may or may not return values
	// depending on whether we're in an azd project, but cache should always be included)
	result := getAzureEnvironmentValues("")

	// Verify cache variables are included
	if result["AZURE_CACHE_VAR"] != "from_cache" {
		t.Errorf("AZURE_CACHE_VAR = %q, want %q", result["AZURE_CACHE_VAR"], "from_cache")
	}

	if result["MY_CUSTOM_VAR"] != "custom_value" {
		t.Errorf("MY_CUSTOM_VAR = %q, want %q", result["MY_CUSTOM_VAR"], "custom_value")
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
		{
			name: "invalid URLs rejected - protocol injection",
			envVars: map[string]string{
				"SERVICE_API_URL": "javascript:alert(1)",
				"WEB_URL":         "file:///etc/passwd",
				"DB_URL":          "data:text/html,<script>alert(1)</script>",
			},
			wantInfo: map[string]AzureServiceInfo{},
		},
		{
			name: "invalid URLs rejected - wrong protocols",
			envVars: map[string]string{
				"SERVICE_API_URL": "ftp://example.com",
				"WEB_URL":         "ssh://example.com",
				"DB_URL":          "telnet://example.com",
			},
			wantInfo: map[string]AzureServiceInfo{},
		},
		{
			name: "invalid URLs rejected - malformed URLs",
			envVars: map[string]string{
				"SERVICE_API_URL": "http://",
				"WEB_URL":         "https://",
				"DB_URL":          "://no-scheme",
			},
			wantInfo: map[string]AzureServiceInfo{},
		},
		{
			name: "invalid URLs rejected - missing scheme",
			envVars: map[string]string{
				"SERVICE_API_URL": "example.com",
				"WEB_URL":         "www.example.com",
			},
			wantInfo: map[string]AzureServiceInfo{},
		},
		{
			name: "valid URLs accepted",
			envVars: map[string]string{
				"SERVICE_API_URL": "https://api.example.com",
				"WEB_URL":         "http://localhost:8080",
			},
			wantInfo: map[string]AzureServiceInfo{
				"api": {
					URL: "https://api.example.com",
				},
				"web": {
					URL: "http://localhost:8080",
				},
			},
		},
		{
			name: "invalid URLs rejected - excessively long URL (>2048 characters)",
			envVars: map[string]string{
				"SERVICE_API_URL": "https://example.com/" + strings.Repeat("a", 2050),
			},
			wantInfo: map[string]AzureServiceInfo{},
		},
		{
			name: "invalid URLs rejected - whitespace-only URL",
			envVars: map[string]string{
				"SERVICE_API_URL": "   ",
				"WEB_URL":         "\t\t",
			},
			wantInfo: map[string]AzureServiceInfo{},
		},
		{
			name: "invalid URLs rejected - empty string",
			envVars: map[string]string{
				"SERVICE_API_URL": "",
			},
			wantInfo: map[string]AzureServiceInfo{},
		},
		{
			name: "invalid URLs rejected - vbscript protocol injection",
			envVars: map[string]string{
				"SERVICE_API_URL": "vbscript:msgbox(1)",
				"WEB_URL":         "vbscript:Execute(\"malicious\")",
			},
			wantInfo: map[string]AzureServiceInfo{},
		},
		{
			name: "mixed valid/invalid URLs - only valid URLs accepted",
			envVars: map[string]string{
				"SERVICE_API_URL":  "https://api.example.com",           // Valid
				"SERVICE_WEB_URL":  "javascript:alert(1)",                // Invalid - protocol injection
				"SERVICE_DB_URL":   "http://example.com",                 // Valid
				"SERVICE_AUTH_URL": "ftp://auth.example.com",             // Invalid - wrong protocol
				"SERVICE_LOG_URL":  "https://log.example.com",            // Valid
				"CACHE_URL":        "file:///etc/passwd",                 // Invalid - protocol injection
			},
			wantInfo: map[string]AzureServiceInfo{
				"api": {
					URL: "https://api.example.com",
				},
				"db": {
					URL: "http://example.com",
				},
				"log": {
					URL: "https://log.example.com",
				},
			},
		},
		{
			name: "invalid URLs rejected - all protocol injection variants",
			envVars: map[string]string{
				"SERVICE_API_URL":   "javascript:alert(1)",
				"SERVICE_WEB_URL":   "file:///etc/passwd",
				"SERVICE_DB_URL":    "data:text/html,<script>alert(1)</script>",
				"SERVICE_AUTH_URL":  "vbscript:msgbox(1)",
				"SERVICE_CACHE_URL": "about:blank",
			},
			wantInfo: map[string]AzureServiceInfo{},
		},
		{
			name: "invalid URLs rejected - whitespace variations",
			envVars: map[string]string{
				"SERVICE_API_URL": " ",
				"WEB_URL":         "  \n  ",
				"DB_URL":          "\t",
			},
			wantInfo: map[string]AzureServiceInfo{},
		},
	{
		name: "custom domain - valid HTTPS",
		envVars: map[string]string{
			"SERVICE_API_CUSTOM_DOMAIN": "https://api.custom.example.com",
			"SERVICE_WEB_CUSTOM_DOMAIN": "https://web.custom.example.com",
		},
		wantInfo: map[string]AzureServiceInfo{
			"api": {
				CustomDomain: stringPtr("https://api.custom.example.com"),
			},
			"web": {
				CustomDomain: stringPtr("https://web.custom.example.com"),
			},
		},
	},
	{
		name: "custom domain - valid HTTP localhost",
		envVars: map[string]string{
			"SERVICE_API_CUSTOM_DOMAIN": "http://localhost:3000",
			"SERVICE_WEB_CUSTOM_DOMAIN": "http://127.0.0.1:8080",
		},
		wantInfo: map[string]AzureServiceInfo{
			"api": {
				CustomDomain: stringPtr("http://localhost:3000"),
			},
			"web": {
				CustomDomain: stringPtr("http://127.0.0.1:8080"),
			},
		},
	},
	{
		name: "custom domain - rejected HTTP non-localhost",
		envVars: map[string]string{
			"SERVICE_API_CUSTOM_DOMAIN": "http://custom.example.com",
			"SERVICE_WEB_CUSTOM_DOMAIN": "http://web.example.com",
		},
		wantInfo: map[string]AzureServiceInfo{},
	},
	{
		name: "custom domain - rejected invalid protocols",
		envVars: map[string]string{
			"SERVICE_API_CUSTOM_DOMAIN": "ftp://custom.example.com",
			"SERVICE_WEB_CUSTOM_DOMAIN": "ssh://web.example.com",
			"SERVICE_DB_CUSTOM_DOMAIN":  "telnet://db.example.com",
		},
		wantInfo: map[string]AzureServiceInfo{},
	},
	{
		name: "custom domain - rejected protocol injection",
		envVars: map[string]string{
			"SERVICE_API_CUSTOM_DOMAIN": "javascript:alert(1)",
			"SERVICE_WEB_CUSTOM_DOMAIN": "file:///etc/passwd",
		},
		wantInfo: map[string]AzureServiceInfo{},
	},
	{
		name: "custom domain - rejected malformed URLs",
		envVars: map[string]string{
			"SERVICE_API_CUSTOM_DOMAIN": "http://",
			"SERVICE_WEB_CUSTOM_DOMAIN": "https://",
			"SERVICE_DB_CUSTOM_DOMAIN":  "not-a-url",
		},
		wantInfo: map[string]AzureServiceInfo{},
	},
	{
		name: "custom domain - rejected excessively long URL",
		envVars: map[string]string{
			"SERVICE_API_CUSTOM_DOMAIN": "https://example.com/" + strings.Repeat("x", 2050),
		},
		wantInfo: map[string]AzureServiceInfo{},
	},
	{
		name: "custom domain - rejected whitespace-only URL",
		envVars: map[string]string{
			"SERVICE_API_CUSTOM_DOMAIN": "   ",
			"SERVICE_WEB_CUSTOM_DOMAIN": "\t\n",
		},
		wantInfo: map[string]AzureServiceInfo{},
	},
	{
		name: "custom domain - rejected empty string",
		envVars: map[string]string{
			"SERVICE_API_CUSTOM_DOMAIN": "",
		},
		wantInfo: map[string]AzureServiceInfo{},
	},
	{
		name: "custom domain - rejected vbscript protocol",
		envVars: map[string]string{
			"SERVICE_API_CUSTOM_DOMAIN": "vbscript:msgbox(1)",
		},
		wantInfo: map[string]AzureServiceInfo{},
	},
	{
		name: "custom domain - mixed valid/invalid with regular URLs",
		envVars: map[string]string{
			"SERVICE_API_URL":           "https://api.azure.com",           // Valid URL
			"SERVICE_API_CUSTOM_DOMAIN": "https://api.custom.com",          // Valid custom domain
			"SERVICE_WEB_URL":           "https://web.azure.com",           // Valid URL
			"SERVICE_WEB_CUSTOM_DOMAIN": "javascript:alert(1)",             // Invalid custom domain
			"SERVICE_DB_URL":            "ftp://db.example.com",            // Invalid URL (ftp not allowed)
			"SERVICE_DB_CUSTOM_DOMAIN":  "https://db.custom.com",           // Valid custom domain (extracted independently)
		},
		wantInfo: map[string]AzureServiceInfo{
			"api": {
				URL:          "https://api.azure.com",
				CustomDomain: stringPtr("https://api.custom.com"),
			},
			"web": {
				URL: "https://web.azure.com",
				// CustomDomain not set because invalid
			},
			"db": {
				// URL not set because invalid (ftp)
				CustomDomain: stringPtr("https://db.custom.com"),
			},
		},
	},
	{
		name: "image name and custom domain combined",
		envVars: map[string]string{
			"SERVICE_API_IMAGE_NAME":    "myregistry.azurecr.io/api:latest",
			"SERVICE_API_CUSTOM_DOMAIN": "https://api.custom.example.com",
		},
		wantInfo: map[string]AzureServiceInfo{
			"api": {
				ImageName:    "myregistry.azurecr.io/api:latest",
				CustomDomain: stringPtr("https://api.custom.example.com"),
			},
		},
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
				
				// Check CustomDomain - handle nil comparisons
				if (actualInfo.CustomDomain == nil) != (expectedInfo.CustomDomain == nil) {
					t.Errorf("service %q: CustomDomain nil mismatch - actual: %v, expected: %v", 
						serviceName, actualInfo.CustomDomain, expectedInfo.CustomDomain)
				} else if actualInfo.CustomDomain != nil && expectedInfo.CustomDomain != nil {
					if *actualInfo.CustomDomain != *expectedInfo.CustomDomain {
						t.Errorf("service %q: CustomDomain = %q, want %q", 
							serviceName, *actualInfo.CustomDomain, *expectedInfo.CustomDomain)
					}
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
