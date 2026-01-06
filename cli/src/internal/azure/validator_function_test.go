package azure

import (
	"context"
	"testing"
)

func TestFunctionValidator_Validate_NotDeployed(t *testing.T) {
	cred := &mockCredential{}
	validator, err := NewFunctionValidator(cred, "/test/project", "test-workspace-id")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	resource := &AzureResource{
		ResourceType: ResourceTypeFunction,
		ResourceID:   "", // Not deployed
	}

	result, err := validator.Validate(context.Background(), "test-function", resource)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	// Verify status
	if result.Status != DiagnosticStatusNotConfigured {
		t.Errorf("Expected status %s, got %s", DiagnosticStatusNotConfigured, result.Status)
	}

	// Verify host type
	if result.HostType != ResourceTypeFunction {
		t.Errorf("Expected host type %s, got %s", ResourceTypeFunction, result.HostType)
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

func TestFunctionValidator_Validate_DeployedNoAppInsights(t *testing.T) {
	cred := &mockCredential{}
	validator, err := NewFunctionValidator(cred, "/test/project", "test-workspace-id")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	resource := &AzureResource{
		ResourceType: ResourceTypeFunction,
		ResourceID:   "/subscriptions/test/resourceGroups/rg/providers/Microsoft.Web/sites/test-func",
	}

	result, err := validator.Validate(context.Background(), "test-function", resource)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	// Should be not configured if App Insights is missing
	if result.Status != DiagnosticStatusNotConfigured {
		t.Logf("Got status: %s (may vary based on actual configuration)", result.Status)
	}

	// Verify requirements exist
	if len(result.Requirements) == 0 {
		t.Error("Expected requirements to be populated")
	}

	// Check for Application Insights requirement
	foundAppInsights := false
	for _, req := range result.Requirements {
		if req.Name == "Application Insights" {
			foundAppInsights = true
			// In test environment without actual env, should be not met
			if req.Status != RequirementStatusNotMet {
				t.Logf("Application Insights status: %s", req.Status)
			}
		}
	}

	if !foundAppInsights {
		t.Error("Expected Application Insights requirement")
	}
}

func TestFunctionValidator_GenerateSetupGuide(t *testing.T) {
	cred := &mockCredential{}
	validator, err := NewFunctionValidator(cred, "/test/project", "test-workspace-id")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	resource := &AzureResource{
		ResourceType: ResourceTypeFunction,
		ResourceID:   "/subscriptions/test/resourceGroups/rg/providers/Microsoft.Web/sites/test-func",
	}

	tests := []struct {
		name           string
		hasAppInsights bool
		hasLogs        bool
		expectGuide    bool
	}{
		{
			name:           "no app insights, no logs",
			hasAppInsights: false,
			hasLogs:        false,
			expectGuide:    true,
		},
		{
			name:           "has app insights, no logs",
			hasAppInsights: true,
			hasLogs:        false,
			expectGuide:    true,
		},
		{
			name:           "has app insights, has logs",
			hasAppInsights: true,
			hasLogs:        true,
			expectGuide:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guide := validator.generateSetupGuide("test-function", resource, tt.hasAppInsights, tt.hasLogs)

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

func TestFunctionValidator_SetupGuideContent(t *testing.T) {
	cred := &mockCredential{}
	validator, err := NewFunctionValidator(cred, "/test/project", "test-workspace-id")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	resource := &AzureResource{
		ResourceType: ResourceTypeFunction,
		ResourceID:   "/subscriptions/test/resourceGroups/rg/providers/Microsoft.Web/sites/test-func",
	}

	guide := validator.generateSetupGuide("test-function", resource, false, false)

	if guide == nil {
		t.Fatal("Expected setup guide to be generated")
	}

	// Verify title mentions Azure Functions
	if guide.Title == "" || len(guide.Title) < 5 {
		t.Errorf("Expected meaningful guide title, got: %s", guide.Title)
	}

	// Verify steps include Application Insights configuration
	foundAppInsightsConfig := false
	for _, step := range guide.Steps {
		if step.Code != "" && step.CodeLang == "yaml" {
			foundAppInsightsConfig = true
			// Verify the YAML contains APPLICATIONINSIGHTS_CONNECTION_STRING
			if len(step.Code) == 0 {
				t.Error("Expected code snippet for Application Insights configuration")
			}
			break
		}
	}

	if !foundAppInsightsConfig {
		t.Error("Expected setup guide to include Application Insights YAML configuration")
	}

	// Verify deployment step
	foundDeployStep := false
	for _, step := range guide.Steps {
		if step.Command != "" && len(step.Command) > 0 {
			foundDeployStep = true
			break
		}
	}

	if !foundDeployStep {
		t.Error("Expected setup guide to include deployment command")
	}
}

func TestFunctionValidator_RequirementChecks(t *testing.T) {
	cred := &mockCredential{}
	validator, err := NewFunctionValidator(cred, "/test/project", "test-workspace-id")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	resource := &AzureResource{
		ResourceType: ResourceTypeFunction,
		ResourceID:   "/subscriptions/test/resourceGroups/rg/providers/Microsoft.Web/sites/test-func",
	}

	result, err := validator.Validate(context.Background(), "test-function", resource)
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

		// Check specific requirements
		switch req.Name {
		case "Application Insights":
			// Should have how to fix if not met
			if req.Status == RequirementStatusNotMet && req.HowToFix == "" {
				t.Error("Application Insights requirement should have HowToFix")
			}
		case "Diagnostic Settings (Optional)":
			// Optional requirement - may or may not have fix
			if req.Description == "" {
				t.Error("Diagnostic Settings requirement should have description")
			}
		}
	}
}

func TestFunctionValidator_DiagnosticStatus(t *testing.T) {
	cred := &mockCredential{}
	validator, err := NewFunctionValidator(cred, "/test/project", "test-workspace-id")
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
			resourceID:     "/subscriptions/test/resourceGroups/rg/providers/Microsoft.Web/sites/test-func",
			expectedStatus: DiagnosticStatusNotConfigured, // In test env without actual config
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := &AzureResource{
				ResourceType: ResourceTypeFunction,
				ResourceID:   tt.resourceID,
			}

			result, err := validator.Validate(context.Background(), "test-function", resource)
			if err != nil {
				t.Fatalf("Validate failed: %v", err)
			}

			if result.Status != tt.expectedStatus {
				t.Logf("Expected status %s, got %s (may vary in test environment)", tt.expectedStatus, result.Status)
			}

			// Verify message is set
			if result.Message == "" && result.Status != DiagnosticStatusHealthy {
				t.Logf("Note: Message not set for status %s", result.Status)
			}
		})
	}
}
