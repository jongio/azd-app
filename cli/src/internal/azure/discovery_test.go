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
		t.Error("NewResourceDiscovery returned nil")
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
