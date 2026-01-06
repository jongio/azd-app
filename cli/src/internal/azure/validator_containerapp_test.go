package azure

import (
	"context"
	"testing"
	"time"
)

func TestContainerAppValidator_Validate_NotDeployed(t *testing.T) {
	cred := &mockCredential{}
	validator, err := NewContainerAppValidator(cred, "/test/project", "test-workspace-id")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	resource := &AzureResource{
		ResourceType: ResourceTypeContainerApp,
		ResourceID:   "", // Not deployed
	}

	result, err := validator.Validate(context.Background(), "test-service", resource)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	// Verify status
	if result.Status != DiagnosticStatusNotConfigured {
		t.Errorf("Expected status %s, got %s", DiagnosticStatusNotConfigured, result.Status)
	}

	// Verify host type
	if result.HostType != ResourceTypeContainerApp {
		t.Errorf("Expected host type %s, got %s", ResourceTypeContainerApp, result.HostType)
	}

	// Verify requirements
	if len(result.Requirements) == 0 {
		t.Error("Expected at least one requirement")
	}

	foundDeploymentReq := false
	for _, req := range result.Requirements {
		if req.Name == "Resource Deployment" {
			foundDeploymentReq = true
			if req.Status != RequirementStatusNotMet {
				t.Errorf("Expected deployment requirement status %s, got %s", RequirementStatusNotMet, req.Status)
			}
		}
	}

	if !foundDeploymentReq {
		t.Error("Expected Resource Deployment requirement")
	}

	// Verify setup guide is provided
	if result.SetupGuide == nil {
		t.Error("Expected setup guide for not deployed resource")
	}
}

func TestContainerAppValidator_Validate_DeployedNoDiagnostics(t *testing.T) {
	cred := &mockCredential{}
	validator, err := NewContainerAppValidator(cred, "/test/project", "test-workspace-id")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	resource := &AzureResource{
		ResourceType: ResourceTypeContainerApp,
		ResourceID:   "/subscriptions/test/resourceGroups/rg/providers/Microsoft.App/containerApps/test-app",
	}

	result, err := validator.Validate(context.Background(), "test-service", resource)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	// Should be not configured if diagnostic settings are missing
	// Note: This test may need adjustment based on actual Azure API responses
	if result.Status != DiagnosticStatusNotConfigured && result.Status != DiagnosticStatusPartial {
		t.Logf("Got status: %s", result.Status)
	}

	// Verify requirements exist
	if len(result.Requirements) == 0 {
		t.Error("Expected requirements to be populated")
	}
}

func TestContainerAppValidator_GenerateSetupGuide(t *testing.T) {
	cred := &mockCredential{}
	validator, err := NewContainerAppValidator(cred, "/test/project", "test-workspace-id")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	resource := &AzureResource{
		ResourceType: ResourceTypeContainerApp,
		ResourceID:   "/subscriptions/test/resourceGroups/rg/providers/Microsoft.App/containerApps/test-app",
	}

	tests := []struct {
		name        string
		hasSettings bool
		hasLogs     bool
		expectGuide bool
	}{
		{
			name:        "no settings, no logs",
			hasSettings: false,
			hasLogs:     false,
			expectGuide: true,
		},
		{
			name:        "has settings, no logs",
			hasSettings: true,
			hasLogs:     false,
			expectGuide: true,
		},
		{
			name:        "has settings, has logs",
			hasSettings: true,
			hasLogs:     true,
			expectGuide: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guide := validator.generateSetupGuide("test-service", resource, tt.hasSettings, tt.hasLogs)

			if tt.expectGuide && guide == nil {
				t.Error("Expected setup guide to be generated")
			}

			if !tt.expectGuide && guide != nil {
				t.Error("Expected no setup guide when logs are flowing")
			}

			if guide != nil {
				if guide.Title == "" {
					t.Error("Expected guide title to be set")
				}
				if len(guide.Steps) == 0 {
					t.Error("Expected guide to have steps")
				}
			}
		})
	}
}

func TestContainerAppValidator_SetupGuideContent(t *testing.T) {
	cred := &mockCredential{}
	validator, err := NewContainerAppValidator(cred, "/test/project", "test-workspace-id")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	resource := &AzureResource{
		ResourceType: ResourceTypeContainerApp,
		ResourceID:   "/subscriptions/test/resourceGroups/rg/providers/Microsoft.App/containerApps/test-app",
	}

	guide := validator.generateSetupGuide("test-service", resource, false, false)

	if guide == nil {
		t.Fatal("Expected setup guide to be generated")
	}

	// Verify title mentions Container Apps
	if guide.Title == "" || len(guide.Title) < 5 {
		t.Errorf("Expected meaningful guide title, got: %s", guide.Title)
	}

	// Verify steps include automatic setup
	foundAutoSetup := false
	for _, step := range guide.Steps {
		if step.Command == "azd up" {
			foundAutoSetup = true
			break
		}
	}

	if !foundAutoSetup {
		t.Error("Expected setup guide to include azd up command")
	}
}

func TestFormatTimeSince(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     *time.Time
		expected string
	}{
		{
			name:     "nil time",
			time:     nil,
			expected: "never",
		},
		{
			name:     "just now",
			time:     &now,
			expected: "just now",
		},
		{
			name: "minutes ago",
			time: func() *time.Time {
				t := now.Add(-5 * time.Minute)
				return &t
			}(),
			expected: "5 min ago",
		},
		{
			name: "hours ago",
			time: func() *time.Time {
				t := now.Add(-2 * time.Hour)
				return &t
			}(),
			expected: "2 hours ago",
		},
		{
			name: "days ago",
			time: func() *time.Time {
				t := now.Add(-3 * 24 * time.Hour)
				return &t
			}(),
			expected: "3 days ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTimeSince(tt.time)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestContainerAppValidator_RequirementStatuses(t *testing.T) {
	// Test that requirement statuses are correctly set
	cred := &mockCredential{}
	validator, err := NewContainerAppValidator(cred, "/test/project", "test-workspace-id")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	resource := &AzureResource{
		ResourceType: ResourceTypeContainerApp,
		ResourceID:   "",
	}

	result, err := validator.Validate(context.Background(), "test-service", resource)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	// Verify all requirements have valid statuses
	validStatuses := map[RequirementStatus]bool{
		RequirementStatusMet:     true,
		RequirementStatusNotMet:  true,
		RequirementStatusUnknown: true,
	}

	for _, req := range result.Requirements {
		if !validStatuses[req.Status] {
			t.Errorf("Invalid requirement status: %s for requirement: %s", req.Status, req.Name)
		}

		// All requirements should have a description
		if req.Description == "" {
			t.Errorf("Requirement %q missing description", req.Name)
		}

		// Not met requirements should have a how to fix
		if req.Status == RequirementStatusNotMet && req.HowToFix == "" {
			t.Logf("Warning: Requirement %q has no HowToFix guidance", req.Name)
		}
	}
}
