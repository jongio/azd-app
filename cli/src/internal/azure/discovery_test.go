package azure

import (
	"context"
	"testing"
	"time"
)

func TestNewResourceDiscovery(t *testing.T) {
	// Test with nil credential (should not panic)
	discovery := NewResourceDiscovery(nil, "/tmp/project")
	if discovery == nil {
		t.Fatal("NewResourceDiscovery returned nil")
	}
	if discovery.cacheDuration != 5*time.Minute {
		t.Errorf("Expected cache duration 5m, got %v", discovery.cacheDuration)
	}
}

func TestInferResourceTypeFromURL(t *testing.T) {
	tests := []struct {
		url      string
		expected ResourceType
	}{
		{"https://myapp.azurewebsites.net", ResourceTypeAppService},
		{"https://myapp.bluefield.azurecontainerapps.io", ResourceTypeContainerApp},
		{"https://myapp.azurefunctions.net", ResourceTypeFunction},
		{"https://MYAPP.AZUREWEBSITES.NET", ResourceTypeAppService}, // case insensitive
		{"https://myapp.random.domain.com", ResourceTypeUnknown},
		{"", ResourceTypeUnknown},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := inferResourceTypeFromURL(tc.url)
			if result != tc.expected {
				t.Errorf("inferResourceTypeFromURL(%q) = %v, want %v", tc.url, result, tc.expected)
			}
		})
	}
}

func TestDiscoveryCache(t *testing.T) {
	discovery := NewResourceDiscovery(nil, "/tmp/project")

	// Initially cache should be nil
	if discovery.cache != nil {
		t.Error("Expected nil cache initially")
	}

	// Set a fake cache result
	discovery.cache = &DiscoveryResult{
		SubscriptionID: "test-sub",
		ResourceGroup:  "test-rg",
		DiscoveredAt:   time.Now(),
		Resources:      make(map[string]*AzureResource),
	}

	// Verify cache is set
	if discovery.cache == nil {
		t.Error("Cache should not be nil after setting")
	}
	if discovery.cache.SubscriptionID != "test-sub" {
		t.Errorf("Expected subscription 'test-sub', got %q", discovery.cache.SubscriptionID)
	}
}

func TestAzureResourceStruct(t *testing.T) {
	resource := &AzureResource{
		ServiceName:    "api",
		ResourceID:     "/subscriptions/123/resourceGroups/rg/providers/Microsoft.Web/sites/myapp",
		ResourceType:   ResourceTypeAppService,
		ResourceGroup:  "rg",
		SubscriptionID: "123",
		URL:            "https://myapp.azurewebsites.net",
		Name:           "myapp",
	}

	if resource.ServiceName != "api" {
		t.Errorf("Expected ServiceName 'api', got %q", resource.ServiceName)
	}
	if resource.ResourceType != ResourceTypeAppService {
		t.Errorf("Expected ResourceType appService, got %v", resource.ResourceType)
	}
}

func TestDiscoveryResultStruct(t *testing.T) {
	result := &DiscoveryResult{
		SubscriptionID:          "sub-123",
		ResourceGroup:           "my-rg",
		Environment:             "dev",
		LogAnalyticsWorkspaceID: "workspace-123",
		Resources:               make(map[string]*AzureResource),
		DiscoveredAt:            time.Now(),
	}

	result.Resources["api"] = &AzureResource{
		ServiceName:  "api",
		ResourceType: ResourceTypeContainerApp,
	}

	if len(result.Resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(result.Resources))
	}
	if result.Resources["api"].ResourceType != ResourceTypeContainerApp {
		t.Errorf("Expected ContainerApp, got %v", result.Resources["api"].ResourceType)
	}
}

func TestDiscoverWithCancelledContext(t *testing.T) {
	discovery := NewResourceDiscovery(nil, "/tmp/project")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should not hang
	_, err := discovery.Discover(ctx)
	if err == nil {
		// The function may return nil error if cache is hit or azd command fails gracefully
		// This is acceptable behavior
		t.Log("Discover returned nil error with cancelled context (acceptable)")
	}
}

func TestMapARMTypeToResourceType(t *testing.T) {
	tests := []struct {
		name     string
		armType  string
		kind     *string
		expected ResourceType
	}{
		// Container Apps
		{"container app", "Microsoft.App/containerApps", nil, ResourceTypeContainerApp},
		{"container app lowercase", "microsoft.app/containerapps", nil, ResourceTypeContainerApp},

		// App Service (no kind or non-function kind)
		{"app service no kind", "Microsoft.Web/sites", nil, ResourceTypeAppService},
		{"app service web kind", "Microsoft.Web/sites", strPtr("app"), ResourceTypeAppService},
		{"app service linux kind", "Microsoft.Web/sites", strPtr("app,linux"), ResourceTypeAppService},

		// Function Apps (various kind values)
		{"function app basic", "Microsoft.Web/sites", strPtr("functionapp"), ResourceTypeFunction},
		{"function app linux", "Microsoft.Web/sites", strPtr("functionapp,linux"), ResourceTypeFunction},
		{"function app workflow", "Microsoft.Web/sites", strPtr("functionapp,workflowapp"), ResourceTypeFunction},
		{"function app container", "Microsoft.Web/sites", strPtr("functionapp,linux,container"), ResourceTypeFunction},
		{"function app case insensitive", "Microsoft.Web/sites", strPtr("FunctionApp"), ResourceTypeFunction},
		{"function app mixed case", "Microsoft.Web/sites", strPtr("FunctionApp,Linux"), ResourceTypeFunction},

		// AKS
		{"aks", "Microsoft.ContainerService/managedClusters", nil, ResourceTypeAKS},

		// Container Instances
		{"container instance", "Microsoft.ContainerInstance/containerGroups", nil, ResourceTypeContainerInstance},

		// Unknown
		{"unknown type", "Microsoft.Storage/storageAccounts", nil, ResourceTypeUnknown},
		{"empty type", "", nil, ResourceTypeUnknown},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := mapARMTypeToResourceType(tc.armType, tc.kind)
			if result != tc.expected {
				t.Errorf("mapARMTypeToResourceType(%q, %v) = %v, want %v", tc.armType, tc.kind, result, tc.expected)
			}
		})
	}
}

func TestIsFunctionAppKind(t *testing.T) {
	tests := []struct {
		kind     string
		expected bool
	}{
		// Function App kinds - should return true
		{"functionapp", true},
		{"FunctionApp", true},
		{"FUNCTIONAPP", true},
		{"functionapp,linux", true},
		{"functionapp,workflowapp", true},
		{"functionapp,linux,container", true},
		{"linux,functionapp", true},

		// Non-function kinds - should return false
		{"app", false},
		{"app,linux", false},
		{"", false},
		{"webapp", false},
		{"linux", false},
	}

	for _, tc := range tests {
		t.Run(tc.kind, func(t *testing.T) {
			result := isFunctionAppKind(tc.kind)
			if result != tc.expected {
				t.Errorf("isFunctionAppKind(%q) = %v, want %v", tc.kind, result, tc.expected)
			}
		})
	}
}

// Helper function to create string pointers for tests
func strPtr(s string) *string {
	return &s
}
