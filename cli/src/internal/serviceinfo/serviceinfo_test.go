package serviceinfo

import "testing"

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
