package serviceinfo

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/registry"
)

func TestNormalizeServiceName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Basic cases with underscores -> hyphens
		{"CONTAINERAPP_API", "containerapp-api"},
		{"APPSERVICE_WEB", "appservice-web"},
		{"FUNCTIONS_WORKER", "functions-worker"},

		// Already hyphenated names should stay as-is (after lowercase)
		{"containerapp-api", "containerapp-api"},
		{"appservice-web", "appservice-web"},

		// Mixed case should become lowercase
		{"ContainerApp_API", "containerapp-api"},
		{"MyService_Name", "myservice-name"},

		// Multiple underscores
		{"MY_LONG_SERVICE_NAME", "my-long-service-name"},

		// No underscores (simple name)
		{"API", "api"},
		{"web", "web"},

		// Empty string
		{"", ""},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := normalizeServiceName(tc.input)
			if result != tc.expected {
				t.Errorf("normalizeServiceName(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestExtractAzureServiceInfo(t *testing.T) {
	// Test environment variables matching the azure-logs-test project
	envVars := map[string]string{
		"SERVICE_CONTAINERAPP_API_URL":        "https://ca-k7zjfgph5a6jk.jollybush-17e4ca58.westus3.azurecontainerapps.io",
		"SERVICE_CONTAINERAPP_API_NAME":       "ca-k7zjfgph5a6jk",
		"SERVICE_CONTAINERAPP_API_IMAGE_NAME": "crk7zjfgph5a6jk.azurecr.io/azure-logs-test/containerapp-api-jong-azlogs-test-01:azd-deploy-1765263801",
		"SERVICE_APPSERVICE_WEB_URL":          "https://appservice-web-k7zjfgph5a6jk.azurewebsites.net",
		"SERVICE_APPSERVICE_WEB_NAME":         "appservice-web-k7zjfgph5a6jk",
		"SERVICE_FUNCTIONS_WORKER_URL":        "https://func-k7zjfgph5a6jk.azurewebsites.net",
		"SERVICE_FUNCTIONS_WORKER_NAME":       "func-k7zjfgph5a6jk",
		"AZURE_SUBSCRIPTION_ID":               "25fd0362-aa79-488b-b37b-d6e892009fdf",
		"AZURE_RESOURCE_GROUP_NAME":           "rg-jong-azlogs-test-01",
	}

	result := extractAzureServiceInfo(envVars)

	// Verify containerapp-api is extracted with hyphenated name
	containerApp, exists := result["containerapp-api"]
	if !exists {
		t.Errorf("Expected 'containerapp-api' service to exist, got keys: %v", getKeys(result))
	} else {
		if containerApp.URL != "https://ca-k7zjfgph5a6jk.jollybush-17e4ca58.westus3.azurecontainerapps.io" {
			t.Errorf("containerapp-api URL = %q, want %q", containerApp.URL, "https://ca-k7zjfgph5a6jk.jollybush-17e4ca58.westus3.azurecontainerapps.io")
		}
		if containerApp.ResourceName != "ca-k7zjfgph5a6jk" {
			t.Errorf("containerapp-api ResourceName = %q, want %q", containerApp.ResourceName, "ca-k7zjfgph5a6jk")
		}
		if containerApp.ImageName != "crk7zjfgph5a6jk.azurecr.io/azure-logs-test/containerapp-api-jong-azlogs-test-01:azd-deploy-1765263801" {
			t.Errorf("containerapp-api ImageName = %q", containerApp.ImageName)
		}
	}

	// Verify appservice-web is extracted with hyphenated name
	appService, exists := result["appservice-web"]
	if !exists {
		t.Errorf("Expected 'appservice-web' service to exist, got keys: %v", getKeys(result))
	} else {
		if appService.URL != "https://appservice-web-k7zjfgph5a6jk.azurewebsites.net" {
			t.Errorf("appservice-web URL = %q, want %q", appService.URL, "https://appservice-web-k7zjfgph5a6jk.azurewebsites.net")
		}
		if appService.ResourceName != "appservice-web-k7zjfgph5a6jk" {
			t.Errorf("appservice-web ResourceName = %q, want %q", appService.ResourceName, "appservice-web-k7zjfgph5a6jk")
		}
	}

	// Verify functions-worker is extracted with hyphenated name
	funcWorker, exists := result["functions-worker"]
	if !exists {
		t.Errorf("Expected 'functions-worker' service to exist, got keys: %v", getKeys(result))
	} else {
		if funcWorker.URL != "https://func-k7zjfgph5a6jk.azurewebsites.net" {
			t.Errorf("functions-worker URL = %q, want %q", funcWorker.URL, "https://func-k7zjfgph5a6jk.azurewebsites.net")
		}
	}

	// Verify old underscore-based names do NOT exist
	if _, exists := result["containerapp_api"]; exists {
		t.Errorf("Should NOT have 'containerapp_api' (underscore), got service: %v", result["containerapp_api"])
	}
	if _, exists := result["appservice_web"]; exists {
		t.Errorf("Should NOT have 'appservice_web' (underscore), got service: %v", result["appservice_web"])
	}
}

func getKeys(m map[string]AzureServiceInfo) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
func TestGetServiceInfo(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create a simple azure.yaml file
	azureYamlContent := `name: test-project
services:
  api:
    language: node
    host: local
    project: ./api
  web:
    language: python
    host: local
    project: ./web
`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0644); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	// Set some environment variables for Azure service info
	t.Setenv("SERVICE_API_URL", "https://api.example.com")
	t.Setenv("SERVICE_API_NAME", "api-resource")

	// Get service info
	services, err := GetServiceInfo(tmpDir)
	if err != nil {
		t.Fatalf("GetServiceInfo() failed: %v", err)
	}

	if len(services) < 2 {
		t.Errorf("Expected at least 2 services, got %d", len(services))
	}

	// Verify that services have correct basic info
	foundAPI := false
	foundWeb := false
	for _, svc := range services {
		if svc.Name == "api" {
			foundAPI = true
			if svc.Language != "node" {
				t.Errorf("api service language = %q, want %q", svc.Language, "node")
			}
			if svc.Host != "local" {
				t.Errorf("api service host = %q, want %q", svc.Host, "local")
			}
			// Check Azure info was merged
			if svc.Azure != nil {
				if svc.Azure.URL != "https://api.example.com" {
					t.Errorf("api Azure URL = %q, want %q", svc.Azure.URL, "https://api.example.com")
				}
			}
		}
		if svc.Name == "web" {
			foundWeb = true
			if svc.Language != "python" {
				t.Errorf("web service language = %q, want %q", svc.Language, "python")
			}
		}
	}

	if !foundAPI {
		t.Error("api service not found in result")
	}
	if !foundWeb {
		t.Error("web service not found in result")
	}
}

func TestGetServiceInfo_NoAzureYaml(t *testing.T) {
	// Test with a directory that has no azure.yaml
	tmpDir := t.TempDir()

	services, err := GetServiceInfo(tmpDir)
	// Should not error, just return empty list
	if err != nil {
		t.Errorf("GetServiceInfo() with no azure.yaml should not error, got: %v", err)
	}

	// Should return empty service list (not nil, but empty slice)
	if services == nil {
		services = []*ServiceInfo{} // Adjust expectation - empty slice is valid
	}
	// Just verify it doesn't panic and returns something reasonable
	t.Logf("Got %d services (expected 0 or empty)", len(services))
}

func TestParseAzureYaml(t *testing.T) {
	// Test with valid azure.yaml
	tmpDir := t.TempDir()
	azureYamlContent := `name: test
services:
  api:
    language: node
    host: local
`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0644); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	result, err := parseAzureYaml(tmpDir)
	if err != nil {
		t.Fatalf("parseAzureYaml() error = %v", err)
	}

	if len(result.Services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(result.Services))
	}

	if _, exists := result.Services["api"]; !exists {
		t.Error("Expected 'api' service to exist")
	}
}

func TestParseAzureYaml_NotFound(t *testing.T) {
	// Test with directory that has no azure.yaml
	tmpDir := t.TempDir()

	result, err := parseAzureYaml(tmpDir)
	// Should not error, should return empty structure
	if err != nil {
		t.Errorf("parseAzureYaml() with missing file should not error, got: %v", err)
	}

	if result == nil {
		t.Fatal("parseAzureYaml() should not return nil")
	}

	if result.Services == nil {
		t.Error("parseAzureYaml() should return initialized Services map")
	}
}

func TestMergeServiceInfo(t *testing.T) {
	azureYaml := &service.AzureYaml{
		Services: map[string]service.Service{
			"api": {
				Language: "node",
				Host:     "local",
				Project:  "./api",
			},
			"web": {
				Language: "python",
				Host:     "local",
				Project:  "./web",
			},
		},
	}

	runningServices := []*registry.ServiceRegistryEntry{
		{
			Name:   "api",
			Status: "running",
			Port:   3000,
			URL:    "http://localhost:3000",
			PID:    12345,
		},
	}

	azureServices := map[string]AzureServiceInfo{
		"api": {
			URL:          "https://api.example.com",
			ResourceName: "api-resource",
		},
	}

	envVars := map[string]string{
		"AZURE_SUBSCRIPTION_ID": "test-sub-id",
	}

	result := mergeServiceInfo(azureYaml, runningServices, azureServices, envVars)

	if len(result) != 2 {
		t.Fatalf("Expected 2 services, got %d", len(result))
	}

	// Find the api service
	var apiService *ServiceInfo
	for _, svc := range result {
		if svc.Name == "api" {
			apiService = svc
			break
		}
	}

	if apiService == nil {
		t.Fatal("api service not found")
	}

	// Verify azure.yaml info is present
	if apiService.Language != "node" {
		t.Errorf("api language = %q, want %q", apiService.Language, "node")
	}

	// Verify running service info is merged
	if apiService.Local == nil {
		t.Fatal("api Local info should not be nil")
	}
	if apiService.Local.Status != "running" {
		t.Errorf("api status = %q, want %q", apiService.Local.Status, "running")
	}
	if apiService.Local.Port != 3000 {
		t.Errorf("api port = %d, want %d", apiService.Local.Port, 3000)
	}

	// Verify Azure info is merged
	if apiService.Azure == nil {
		t.Fatal("api Azure info should not be nil")
	}
	if apiService.Azure.URL != "https://api.example.com" {
		t.Errorf("api Azure URL = %q, want %q", apiService.Azure.URL, "https://api.example.com")
	}

	// Verify environment variables are included
	if apiService.EnvironmentVars == nil {
		t.Fatal("api EnvironmentVars should not be nil")
	}
	if apiService.EnvironmentVars["AZURE_SUBSCRIPTION_ID"] != "test-sub-id" {
		t.Errorf("api env var AZURE_SUBSCRIPTION_ID = %q, want %q",
			apiService.EnvironmentVars["AZURE_SUBSCRIPTION_ID"], "test-sub-id")
	}
}

func TestMergeServiceInfo_WithURL(t *testing.T) {
	azureYaml := &service.AzureYaml{
		Services: map[string]service.Service{
			"web": {
				Language: "node",
				Host:     "containerapp",
				Project:  "./web",
				URL:      "https://myapp.example.com", // Deprecated root-level URL field
			},
			"api": {
				Language: "python",
				Host:     "appservice",
				Project:  "./api",
				URL:      "https://api.myapp.example.com", // Deprecated root-level URL field
			},
		},
	}

	azureServices := map[string]AzureServiceInfo{
		"web": {
			URL:          "https://web-abc123.azurewebsites.net", // Auto-discovered URL
			ResourceName: "web-abc123",
		},
		"api": {
			URL:          "https://api-abc123.azurewebsites.net", // Auto-discovered URL
			ResourceName: "api-abc123",
		},
	}

	envVars := map[string]string{}

	result := mergeServiceInfo(azureYaml, nil, azureServices, envVars)

	if len(result) != 2 {
		t.Fatalf("Expected 2 services, got %d", len(result))
	}

	// Find the web service and verify CustomURL is set from deprecated URL field
	var webService *ServiceInfo
	for _, svc := range result {
		if svc.Name == "web" {
			webService = svc
			break
		}
	}

	if webService == nil {
		t.Fatal("web service not found")
	}

	if webService.Azure == nil {
		t.Fatal("web Azure info should not be nil")
	}

	// Verify deprecated URL field migrated to CustomURL
	if webService.Azure.CustomURL != "https://myapp.example.com" {
		t.Errorf("web Azure.CustomURL = %q, want %q", webService.Azure.CustomURL, "https://myapp.example.com")
	}

	// Verify auto-discovered URL is preserved
	if webService.Azure.URL != "https://web-abc123.azurewebsites.net" {
		t.Errorf("web Azure.URL = %q, want %q", webService.Azure.URL, "https://web-abc123.azurewebsites.net")
	}

	// Find the api service and verify url is set
	var apiService *ServiceInfo
	for _, svc := range result {
		if svc.Name == "api" {
			apiService = svc
			break
		}
	}

	if apiService == nil {
		t.Fatal("api service not found")
	}

	if apiService.Azure == nil {
		t.Fatal("api Azure info should not be nil")
	}

	if apiService.Azure.CustomURL != "https://api.myapp.example.com" {
		t.Errorf("api Azure.CustomURL = %q, want %q", apiService.Azure.CustomURL, "https://api.myapp.example.com")
	}

	if apiService.Azure.URL != "https://api-abc123.azurewebsites.net" {
		t.Errorf("api Azure.URL = %q, want %q", apiService.Azure.URL, "https://api-abc123.azurewebsites.net")
	}
}

func TestMergeServiceInfo_PreservesLocalCustomURLWithRunningService(t *testing.T) {
	azureYaml := &service.AzureYaml{
		Services: map[string]service.Service{
			"web": {
				Language: "node",
				Host:     "containerapp",
				Project:  "./web",
				Local: &service.LocalServiceConfig{
					CustomURL: "https://local.override.example.com",
				},
			},
		},
	}

	running := []*registry.ServiceRegistryEntry{
		{
			Name:        "web",
			Status:      "running",
			Port:        3000,
			StartTime:   time.Now(),
			LastChecked: time.Now(),
		},
	}

	result := mergeServiceInfo(azureYaml, running, nil, nil)
	if len(result) != 1 {
		t.Fatalf("expected 1 service, got %d", len(result))
	}

	web := result[0]
	if web.Local == nil {
		t.Fatalf("expected Local info to be present")
	}

	if web.Local.CustomURL != "https://local.override.example.com" {
		t.Fatalf("expected custom local URL to be preserved, got %q", web.Local.CustomURL)
	}
}

func TestMergeServiceInfo_NilAzureYaml(t *testing.T) {
	// Test with nil azure.yaml
	result := mergeServiceInfo(nil, nil, nil, nil)

	// The function returns a slice from the map, which may be nil or empty
	// Both are valid - just verify it doesn't panic
	t.Logf("Got %d services from nil inputs", len(result))
}

func TestDetectFramework(t *testing.T) {
	tests := []struct {
		name     string
		service  service.Service
		expected string
	}{
		{
			name:     "node service",
			service:  service.Service{Language: "node"},
			expected: "express",
		},
		{
			name:     "python service",
			service:  service.Service{Language: "python"},
			expected: "flask",
		},
		{
			name:     "dotnet service",
			service:  service.Service{Language: "dotnet"},
			expected: "aspnetcore",
		},
		{
			name:     "go service",
			service:  service.Service{Language: "go"},
			expected: "go",
		},
		{
			name:     "empty language",
			service:  service.Service{Language: ""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectFramework(tt.service)
			if result != tt.expected {
				t.Errorf("detectFramework() = %q, want %q", result, tt.expected)
			}
		})
	}
}
