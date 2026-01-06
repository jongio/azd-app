package azure

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

// mockCredential is a test credential that doesn't make actual API calls.
type mockCredentialForBicep struct{}

func (m *mockCredentialForBicep) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{
		Token:     "mock-token",
		ExpiresOn: time.Now().Add(1 * time.Hour),
	}, nil
}

// TestGenerateTemplate_SingleContainerApp tests template generation for a single Container App.
func TestGenerateTemplate_SingleContainerApp(t *testing.T) {
	// Create discovery with mock data
	discovery := &ResourceDiscovery{
		credential: &mockCredentialForBicep{},
		projectDir: "/test",
		cache: &DiscoveryResult{
			SubscriptionID:          "test-sub",
			ResourceGroup:           "test-rg",
			Environment:             "test-env",
			LogAnalyticsWorkspaceID: "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.OperationalInsights/workspaces/test-workspace",
			Resources: map[string]*AzureResource{
				"api": {
					ServiceName:    "api",
					ResourceID:     "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.App/containerApps/api",
					ResourceType:   ResourceTypeContainerApp,
					ResourceGroup:  "test-rg",
					SubscriptionID: "test-sub",
					Name:           "api",
				},
			},
			DiscoveredAt: time.Now(),
		},
		cacheDuration: 5 * time.Minute,
	}

	generator := NewBicepGenerator(discovery)
	result, err := generator.GenerateTemplate(context.Background())

	if err != nil {
		t.Fatalf("GenerateTemplate failed: %v", err)
	}

	// Verify response structure
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Verify services
	if len(result.Services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(result.Services))
	}
	if result.Services[0] != "api" {
		t.Errorf("Expected service 'api', got '%s'", result.Services[0])
	}

	// Verify template content
	template := result.Template
	if !strings.Contains(template, "param logAnalyticsWorkspaceId string") {
		t.Error("Template missing logAnalyticsWorkspaceId parameter")
	}
	if !strings.Contains(template, "param containerAppName string") {
		t.Error("Template missing containerAppName parameter")
	}
	if !strings.Contains(template, "Microsoft.App/containerApps") {
		t.Error("Template missing Container App resource type")
	}
	if !strings.Contains(template, "ContainerAppConsoleLogs") {
		t.Error("Template missing ContainerAppConsoleLogs category")
	}
	if !strings.Contains(template, "ContainerAppSystemLogs") {
		t.Error("Template missing ContainerAppSystemLogs category")
	}

	// Verify instructions
	if result.Instructions.Summary == "" {
		t.Error("Instructions summary is empty")
	}
	if len(result.Instructions.Steps) == 0 {
		t.Error("Instructions steps are empty")
	}

	// Verify parameters
	if len(result.Parameters) == 0 {
		t.Error("Parameters are empty")
	}
	foundWorkspaceParam := false
	for _, param := range result.Parameters {
		if param.Name == "logAnalyticsWorkspaceId" {
			foundWorkspaceParam = true
			if param.Description == "" {
				t.Error("Workspace parameter missing description")
			}
			if param.Example == "" {
				t.Error("Workspace parameter missing example")
			}
		}
	}
	if !foundWorkspaceParam {
		t.Error("Missing logAnalyticsWorkspaceId parameter info")
	}
}

// TestGenerateTemplate_SingleAppService tests template generation for a single App Service.
func TestGenerateTemplate_SingleAppService(t *testing.T) {
	discovery := &ResourceDiscovery{
		credential: &mockCredentialForBicep{},
		projectDir: "/test",
		cache: &DiscoveryResult{
			SubscriptionID:          "test-sub",
			ResourceGroup:           "test-rg",
			Environment:             "test-env",
			LogAnalyticsWorkspaceID: "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.OperationalInsights/workspaces/test-workspace",
			Resources: map[string]*AzureResource{
				"web": {
					ServiceName:    "web",
					ResourceID:     "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Web/sites/web",
					ResourceType:   ResourceTypeAppService,
					ResourceGroup:  "test-rg",
					SubscriptionID: "test-sub",
					Name:           "web",
				},
			},
			DiscoveredAt: time.Now(),
		},
		cacheDuration: 5 * time.Minute,
	}

	generator := NewBicepGenerator(discovery)
	result, err := generator.GenerateTemplate(context.Background())

	if err != nil {
		t.Fatalf("GenerateTemplate failed: %v", err)
	}

	template := result.Template
	if !strings.Contains(template, "param appServiceName string") {
		t.Error("Template missing appServiceName parameter")
	}
	if !strings.Contains(template, "Microsoft.Web/sites") {
		t.Error("Template missing App Service resource type")
	}
	if !strings.Contains(template, "AppServiceHTTPLogs") {
		t.Error("Template missing AppServiceHTTPLogs category")
	}
	if !strings.Contains(template, "AppServiceConsoleLogs") {
		t.Error("Template missing AppServiceConsoleLogs category")
	}
	if !strings.Contains(template, "AppServiceAppLogs") {
		t.Error("Template missing AppServiceAppLogs category")
	}
	if !strings.Contains(template, "AppServicePlatformLogs") {
		t.Error("Template missing AppServicePlatformLogs category")
	}
}

// TestGenerateTemplate_SingleFunction tests template generation for a single Function App.
func TestGenerateTemplate_SingleFunction(t *testing.T) {
	discovery := &ResourceDiscovery{
		credential: &mockCredentialForBicep{},
		projectDir: "/test",
		cache: &DiscoveryResult{
			SubscriptionID:          "test-sub",
			ResourceGroup:           "test-rg",
			Environment:             "test-env",
			LogAnalyticsWorkspaceID: "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.OperationalInsights/workspaces/test-workspace",
			Resources: map[string]*AzureResource{
				"func": {
					ServiceName:    "func",
					ResourceID:     "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Web/sites/func",
					ResourceType:   ResourceTypeFunction,
					ResourceGroup:  "test-rg",
					SubscriptionID: "test-sub",
					Name:           "func",
				},
			},
			DiscoveredAt: time.Now(),
		},
		cacheDuration: 5 * time.Minute,
	}

	generator := NewBicepGenerator(discovery)
	result, err := generator.GenerateTemplate(context.Background())

	if err != nil {
		t.Fatalf("GenerateTemplate failed: %v", err)
	}

	template := result.Template
	if !strings.Contains(template, "param functionAppName string") {
		t.Error("Template missing functionAppName parameter")
	}
	if !strings.Contains(template, "Microsoft.Web/sites") {
		t.Error("Template missing Function App resource type")
	}
	if !strings.Contains(template, "FunctionAppLogs") {
		t.Error("Template missing FunctionAppLogs category")
	}
}

// TestGenerateTemplate_MultipleServices tests template generation for multiple service types.
func TestGenerateTemplate_MultipleServices(t *testing.T) {
	discovery := &ResourceDiscovery{
		credential: &mockCredentialForBicep{},
		projectDir: "/test",
		cache: &DiscoveryResult{
			SubscriptionID:          "test-sub",
			ResourceGroup:           "test-rg",
			Environment:             "test-env",
			LogAnalyticsWorkspaceID: "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.OperationalInsights/workspaces/test-workspace",
			Resources: map[string]*AzureResource{
				"api": {
					ServiceName:    "api",
					ResourceID:     "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.App/containerApps/api",
					ResourceType:   ResourceTypeContainerApp,
					ResourceGroup:  "test-rg",
					SubscriptionID: "test-sub",
					Name:           "api",
				},
				"web": {
					ServiceName:    "web",
					ResourceID:     "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Web/sites/web",
					ResourceType:   ResourceTypeAppService,
					ResourceGroup:  "test-rg",
					SubscriptionID: "test-sub",
					Name:           "web",
				},
				"func": {
					ServiceName:    "func",
					ResourceID:     "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Web/sites/func",
					ResourceType:   ResourceTypeFunction,
					ResourceGroup:  "test-rg",
					SubscriptionID: "test-sub",
					Name:           "func",
				},
			},
			DiscoveredAt: time.Now(),
		},
		cacheDuration: 5 * time.Minute,
	}

	generator := NewBicepGenerator(discovery)
	result, err := generator.GenerateTemplate(context.Background())

	if err != nil {
		t.Fatalf("GenerateTemplate failed: %v", err)
	}

	// Verify all three services are included
	if len(result.Services) != 3 {
		t.Errorf("Expected 3 services, got %d", len(result.Services))
	}

	// Verify services are sorted
	expectedOrder := []string{"api", "func", "web"}
	for i, expected := range expectedOrder {
		if result.Services[i] != expected {
			t.Errorf("Service at index %d: expected '%s', got '%s'", i, expected, result.Services[i])
		}
	}

	// Verify template contains all service types
	template := result.Template
	requiredStrings := []string{
		"param containerAppName string",
		"param appServiceName string",
		"param functionAppName string",
		"Microsoft.App/containerApps",
		"Microsoft.Web/sites",
		"ContainerAppConsoleLogs",
		"AppServiceHTTPLogs",
		"FunctionAppLogs",
	}

	for _, required := range requiredStrings {
		if !strings.Contains(template, required) {
			t.Errorf("Template missing required string: %s", required)
		}
	}
}

// TestGenerateTemplate_NoResources tests error handling when no resources are found.
func TestGenerateTemplate_NoResources(t *testing.T) {
	discovery := &ResourceDiscovery{
		credential: &mockCredentialForBicep{},
		projectDir: "/test",
		cache: &DiscoveryResult{
			SubscriptionID:          "test-sub",
			ResourceGroup:           "test-rg",
			Environment:             "test-env",
			LogAnalyticsWorkspaceID: "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.OperationalInsights/workspaces/test-workspace",
			Resources:               map[string]*AzureResource{}, // Empty resources
			DiscoveredAt:            time.Now(),
		},
		cacheDuration: 5 * time.Minute,
	}

	generator := NewBicepGenerator(discovery)
	result, err := generator.GenerateTemplate(context.Background())

	if err == nil {
		t.Fatal("Expected error when no resources found, got nil")
	}

	if result != nil {
		t.Error("Expected nil result when error occurs")
	}

	expectedError := "no Azure resources found"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

// TestGenerateTemplate_TemplateStructure tests the overall structure of generated templates.
func TestGenerateTemplate_TemplateStructure(t *testing.T) {
	discovery := &ResourceDiscovery{
		credential: &mockCredentialForBicep{},
		projectDir: "/test",
		cache: &DiscoveryResult{
			SubscriptionID:          "test-sub",
			ResourceGroup:           "test-rg",
			Environment:             "test-env",
			LogAnalyticsWorkspaceID: "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.OperationalInsights/workspaces/test-workspace",
			Resources: map[string]*AzureResource{
				"api": {
					ServiceName:    "api",
					ResourceID:     "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.App/containerApps/api",
					ResourceType:   ResourceTypeContainerApp,
					ResourceGroup:  "test-rg",
					SubscriptionID: "test-sub",
					Name:           "api",
				},
			},
			DiscoveredAt: time.Now(),
		},
		cacheDuration: 5 * time.Minute,
	}

	generator := NewBicepGenerator(discovery)
	result, err := generator.GenerateTemplate(context.Background())

	if err != nil {
		t.Fatalf("GenerateTemplate failed: %v", err)
	}

	template := result.Template

	// Verify template starts with header comments
	if !strings.HasPrefix(template, "// Diagnostic Settings Module") {
		t.Error("Template should start with header comment")
	}

	// Verify template contains required sections
	requiredSections := []string{
		"// Parameters",
		"@description",
		"param logAnalyticsWorkspaceId string",
		"resource",
		"Microsoft.Insights/diagnosticSettings",
		"workspaceId: logAnalyticsWorkspaceId",
		"logs:",
		"metrics:",
		"retentionPolicy:",
		"days: 30",
	}

	for _, section := range requiredSections {
		if !strings.Contains(template, section) {
			t.Errorf("Template missing required section: %s", section)
		}
	}
}

// TestGenerateTemplate_RetentionPolicy tests that retention policies are included.
func TestGenerateTemplate_RetentionPolicy(t *testing.T) {
	discovery := &ResourceDiscovery{
		credential: &mockCredentialForBicep{},
		projectDir: "/test",
		cache: &DiscoveryResult{
			SubscriptionID:          "test-sub",
			ResourceGroup:           "test-rg",
			Environment:             "test-env",
			LogAnalyticsWorkspaceID: "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.OperationalInsights/workspaces/test-workspace",
			Resources: map[string]*AzureResource{
				"api": {
					ServiceName:    "api",
					ResourceID:     "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.App/containerApps/api",
					ResourceType:   ResourceTypeContainerApp,
					ResourceGroup:  "test-rg",
					SubscriptionID: "test-sub",
					Name:           "api",
				},
			},
			DiscoveredAt: time.Now(),
		},
		cacheDuration: 5 * time.Minute,
	}

	generator := NewBicepGenerator(discovery)
	result, err := generator.GenerateTemplate(context.Background())

	if err != nil {
		t.Fatalf("GenerateTemplate failed: %v", err)
	}

	template := result.Template

	// Count occurrences of retention policy - should appear for each log category and metrics
	retentionCount := strings.Count(template, "retentionPolicy:")
	if retentionCount == 0 {
		t.Error("Template should include retention policies")
	}

	// Verify 30 day retention
	if !strings.Contains(template, "days: 30") {
		t.Error("Template should include 30 day retention period")
	}
}

// TestBuildInstructions tests instruction generation.
func TestBuildInstructions(t *testing.T) {
	generator := &BicepGenerator{}

	serviceTypes := map[ResourceType]bool{
		ResourceTypeContainerApp: true,
		ResourceTypeAppService:   true,
	}

	instructions := generator.buildInstructions(serviceTypes)

	if instructions.Summary == "" {
		t.Error("Instructions summary should not be empty")
	}

	if len(instructions.Steps) == 0 {
		t.Error("Instructions should contain steps")
	}

	// Verify key steps are included
	hasInfraStep := false
	hasDeployStep := false

	for _, step := range instructions.Steps {
		if strings.Contains(step, "infra/modules") {
			hasInfraStep = true
		}
		if strings.Contains(step, "azd up") {
			hasDeployStep = true
		}
	}

	if !hasInfraStep {
		t.Error("Instructions should mention saving to infra/modules")
	}

	if !hasDeployStep {
		t.Error("Instructions should mention running azd up")
	}
}

// TestBuildParameters tests parameter documentation generation.
func TestBuildParameters(t *testing.T) {
	generator := &BicepGenerator{}

	params := generator.buildParameters()

	if len(params) == 0 {
		t.Fatal("Should return at least one parameter")
	}

	// Verify workspace parameter exists
	var workspaceParam *BicepParameterInfo
	for i := range params {
		if params[i].Name == "logAnalyticsWorkspaceId" {
			workspaceParam = &params[i]
			break
		}
	}

	if workspaceParam == nil {
		t.Fatal("Should include logAnalyticsWorkspaceId parameter")
	}

	if workspaceParam.Description == "" {
		t.Error("Parameter should have description")
	}

	if workspaceParam.Example == "" {
		t.Error("Parameter should have example")
	}

	// Verify example contains expected format
	if !strings.Contains(workspaceParam.Example, "/subscriptions/") {
		t.Error("Example should contain subscription path format")
	}
}
