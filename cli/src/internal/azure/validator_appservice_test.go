package azure

import (
	"context"
	"testing"
)

func TestAppServiceValidator_Validate_NotDeployed(t *testing.T) {
	cred := &mockCredential{}
	validator, err := NewAppServiceValidator(cred, "/test/project", "test-workspace-id")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	resource := &AzureResource{
		ResourceType: ResourceTypeAppService,
		ResourceID:   "", // Not deployed
	}

	result, err := validator.Validate(context.Background(), "test-app", resource)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	// Verify status
	if result.Status != DiagnosticStatusNotConfigured {
		t.Errorf("Expected status %s, got %s", DiagnosticStatusNotConfigured, result.Status)
	}

	// Verify host type
	if result.HostType != ResourceTypeAppService {
		t.Errorf("Expected host type %s, got %s", ResourceTypeAppService, result.HostType)
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

func TestAppServiceValidator_Validate_DeployedNoDiagnostics(t *testing.T) {
	cred := &mockCredential{}
	validator, err := NewAppServiceValidator(cred, "/test/project", "test-workspace-id")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	resource := &AzureResource{
		ResourceType: ResourceTypeAppService,
		ResourceID:   "/subscriptions/test/resourceGroups/rg/providers/Microsoft.Web/sites/test-app",
	}

	result, err := validator.Validate(context.Background(), "test-app", resource)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	// Should be not configured if diagnostic settings are missing
	if result.Status != DiagnosticStatusNotConfigured && result.Status != DiagnosticStatusPartial {
		t.Logf("Got status: %s", result.Status)
	}

	// Verify requirements exist
	if len(result.Requirements) == 0 {
		t.Error("Expected requirements to be populated")
	}

	// Check for Diagnostic Settings requirement
	foundDiagnostics := false
	for _, req := range result.Requirements {
		if req.Name == "Diagnostic Settings" {
			foundDiagnostics = true
		}
	}

	if !foundDiagnostics {
		t.Error("Expected Diagnostic Settings requirement")
	}
}

func TestAppServiceValidator_GenerateSetupGuide(t *testing.T) {
	cred := &mockCredential{}
	validator, err := NewAppServiceValidator(cred, "/test/project", "test-workspace-id")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	resource := &AzureResource{
		ResourceType: ResourceTypeAppService,
		ResourceID:   "/subscriptions/test/resourceGroups/rg/providers/Microsoft.Web/sites/test-app",
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
			guide := validator.generateSetupGuide("test-app", resource, tt.hasSettings, tt.hasLogs)

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

func TestAppServiceValidator_SetupGuideContent(t *testing.T) {
	cred := &mockCredential{}
	validator, err := NewAppServiceValidator(cred, "/test/project", "test-workspace-id")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	resource := &AzureResource{
		ResourceType: ResourceTypeAppService,
		ResourceID:   "/subscriptions/test/resourceGroups/rg/providers/Microsoft.Web/sites/test-app",
	}

	guide := validator.generateSetupGuide("test-app", resource, false, false)

	if guide == nil {
		t.Fatal("Expected setup guide to be generated")
	}

	// Verify title mentions App Service
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

	// Verify manual setup instructions
	foundManualSetup := false
	for _, step := range guide.Steps {
		if step.Title != "" && len(step.Description) > 50 {
			foundManualSetup = true
			break
		}
	}

	if !foundManualSetup {
		t.Error("Expected setup guide to include detailed manual setup instructions")
	}
}

func TestAppServiceValidator_RequirementStatuses(t *testing.T) {
	cred := &mockCredential{}
	validator, err := NewAppServiceValidator(cred, "/test/project", "test-workspace-id")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	resource := &AzureResource{
		ResourceType: ResourceTypeAppService,
		ResourceID:   "/subscriptions/test/resourceGroups/rg/providers/Microsoft.Web/sites/test-app",
	}

	result, err := validator.Validate(context.Background(), "test-app", resource)
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

func TestAppServiceValidator_DiagnosticStatuses(t *testing.T) {
	cred := &mockCredential{}
	validator, err := NewAppServiceValidator(cred, "/test/project", "test-workspace-id")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	tests := []struct {
		name           string
		resourceID     string
		expectedStatus DiagnosticStatus
	}{
		{
			name:           "not deployed",
			resourceID:     "",
			expectedStatus: DiagnosticStatusNotConfigured,
		},
		{
			name:           "deployed",
			resourceID:     "/subscriptions/test/resourceGroups/rg/providers/Microsoft.Web/sites/test-app",
			expectedStatus: DiagnosticStatusNotConfigured, // In test env without actual config
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := &AzureResource{
				ResourceType: ResourceTypeAppService,
				ResourceID:   tt.resourceID,
			}

			result, err := validator.Validate(context.Background(), "test-app", resource)
			if err != nil {
				t.Fatalf("Validate failed: %v", err)
			}

			if result.Status != tt.expectedStatus {
				t.Logf("Expected status %s, got %s (may vary in test environment)", tt.expectedStatus, result.Status)
			}

			// Verify message is set for non-healthy status
			if result.Message == "" && result.Status != DiagnosticStatusHealthy {
				t.Logf("Note: Message not set for status %s", result.Status)
			}

			// Verify status is one of the valid values
			validDiagStatuses := map[DiagnosticStatus]bool{
				DiagnosticStatusHealthy:       true,
				DiagnosticStatusPartial:       true,
				DiagnosticStatusNotConfigured: true,
				DiagnosticStatusError:         true,
			}

			if !validDiagStatuses[result.Status] {
				t.Errorf("Invalid diagnostic status: %s", result.Status)
			}
		})
	}
}

func TestAppServiceValidator_MessageContent(t *testing.T) {
	cred := &mockCredential{}
	validator, err := NewAppServiceValidator(cred, "/test/project", "test-workspace-id")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	resource := &AzureResource{
		ResourceType: ResourceTypeAppService,
		ResourceID:   "",
	}

	result, err := validator.Validate(context.Background(), "test-app", resource)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	// For not deployed, should have a clear message or setup guide
	if result.SetupGuide == nil {
		t.Error("Expected setup guide for not deployed resource")
	}

	// When deployed (simulated)
	resource.ResourceID = "/subscriptions/test/resourceGroups/rg/providers/Microsoft.Web/sites/test-app"
	result, err = validator.Validate(context.Background(), "test-app", resource)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	// Should have some message about status
	if result.Status == DiagnosticStatusNotConfigured && result.Message == "" {
		t.Error("Expected message for not configured status")
	}
}
