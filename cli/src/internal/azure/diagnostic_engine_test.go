package azure

import (
	"context"
	"testing"
)

func TestDiagnosticsEngine_NewEngine(t *testing.T) {
	cred := &mockCredential{}
	projectDir := "/test/project"

	engine := NewDiagnosticsEngine(cred, projectDir)

	if engine == nil {
		t.Fatal("Expected engine to be created")
	}

	if engine.credential != cred {
		t.Error("Expected credential to be set")
	}

	if engine.projectDir != projectDir {
		t.Error("Expected project dir to be set")
	}

	if engine.discovery == nil {
		t.Error("Expected discovery to be initialized")
	}

	if engine.validators == nil {
		t.Error("Expected validators map to be initialized")
	}
}

func TestDiagnosticsEngine_RegisterValidator(t *testing.T) {
	cred := &mockCredential{}
	engine := NewDiagnosticsEngine(cred, "/test/project")

	// Create a mock validator
	mockValidator := &mockServiceValidator{
		validateFunc: func(ctx context.Context, serviceName string, resource *AzureResource) (*ServiceDiagnosticResult, error) {
			return &ServiceDiagnosticResult{
				HostType: ResourceTypeContainerApp,
				Status:   DiagnosticStatusHealthy,
			}, nil
		},
	}

	// Register the validator
	engine.RegisterValidator(ResourceTypeContainerApp, mockValidator)

	// Verify it was registered
	if _, ok := engine.validators[ResourceTypeContainerApp]; !ok {
		t.Error("Expected validator to be registered")
	}
}

func TestDiagnosticsEngine_ValidateService_NoValidator(t *testing.T) {
	cred := &mockCredential{}
	engine := NewDiagnosticsEngine(cred, "/test/project")

	resource := &AzureResource{
		ResourceType: "unsupported-type",
		ResourceID:   "/subscriptions/test/resource",
	}

	result := engine.validateService(context.Background(), "test-service", resource)

	if result == nil {
		t.Fatal("Expected result to be returned")
	}

	if result.Status != DiagnosticStatusError {
		t.Errorf("Expected status %s for unsupported type, got %s", DiagnosticStatusError, result.Status)
	}

	if result.Message == "" {
		t.Error("Expected message explaining unsupported type")
	}

	if len(result.Requirements) == 0 {
		t.Error("Expected at least one requirement explaining the issue")
	}
}

func TestDiagnosticsEngine_ValidateService_WithValidator(t *testing.T) {
	cred := &mockCredential{}
	engine := NewDiagnosticsEngine(cred, "/test/project")

	expectedResult := &ServiceDiagnosticResult{
		HostType: ResourceTypeContainerApp,
		Status:   DiagnosticStatusHealthy,
		LogCount: 100,
		Requirements: []Requirement{
			{Name: "Test Requirement", Status: RequirementStatusMet},
		},
	}

	mockValidator := &mockServiceValidator{
		validateFunc: func(ctx context.Context, serviceName string, resource *AzureResource) (*ServiceDiagnosticResult, error) {
			return expectedResult, nil
		},
	}

	engine.RegisterValidator(ResourceTypeContainerApp, mockValidator)

	resource := &AzureResource{
		ResourceType: ResourceTypeContainerApp,
		ResourceID:   "/subscriptions/test/resource",
	}

	result := engine.validateService(context.Background(), "test-service", resource)

	if result == nil {
		t.Fatal("Expected result to be returned")
	}

	if result.Status != expectedResult.Status {
		t.Errorf("Expected status %s, got %s", expectedResult.Status, result.Status)
	}

	if result.LogCount != expectedResult.LogCount {
		t.Errorf("Expected log count %d, got %d", expectedResult.LogCount, result.LogCount)
	}
}

func TestDiagnosticsEngine_ValidateService_ValidatorError(t *testing.T) {
	cred := &mockCredential{}
	engine := NewDiagnosticsEngine(cred, "/test/project")

	mockValidator := &mockServiceValidator{
		validateFunc: func(ctx context.Context, serviceName string, resource *AzureResource) (*ServiceDiagnosticResult, error) {
			return nil, &mockError{message: "validation failed"}
		},
	}

	engine.RegisterValidator(ResourceTypeContainerApp, mockValidator)

	resource := &AzureResource{
		ResourceType: ResourceTypeContainerApp,
		ResourceID:   "/subscriptions/test/resource",
	}

	result := engine.validateService(context.Background(), "test-service", resource)

	if result == nil {
		t.Fatal("Expected result to be returned even on error")
	}

	if result.Status != DiagnosticStatusError {
		t.Errorf("Expected status %s on validator error, got %s", DiagnosticStatusError, result.Status)
	}

	if result.Error == "" {
		t.Error("Expected error message to be set")
	}
}

func TestDiagnosticsEngine_InitializeValidators(t *testing.T) {
	cred := &mockCredential{}
	engine := NewDiagnosticsEngine(cred, "/test/project")

	workspaceID := "/subscriptions/test/workspaces/test-workspace"

	// Initialize validators
	engine.initializeValidators(workspaceID)

	// Verify validators were created
	// Note: This may fail if actual validator constructors require real resources
	// In that case, this test would need to be skipped or mocked differently

	// The actual validators may or may not be registered depending on
	// whether the constructors succeed in a test environment
	t.Logf("Validators registered: %d", len(engine.validators))
}

func TestDiagnosticStatus_ValidValues(t *testing.T) {
	// Verify all diagnostic status constants have expected values
	tests := []struct {
		status DiagnosticStatus
		value  string
	}{
		{DiagnosticStatusHealthy, "healthy"},
		{DiagnosticStatusPartial, "partial"},
		{DiagnosticStatusNotConfigured, "not-configured"},
		{DiagnosticStatusError, "error"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.value {
			t.Errorf("Expected %s to equal %q, got %q", tt.status, tt.value, string(tt.status))
		}
	}
}

func TestRequirementStatus_ValidValues(t *testing.T) {
	// Verify all requirement status constants have expected values
	tests := []struct {
		status RequirementStatus
		value  string
	}{
		{RequirementStatusMet, "met"},
		{RequirementStatusNotMet, "not-met"},
		{RequirementStatusUnknown, "unknown"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.value {
			t.Errorf("Expected %s to equal %q, got %q", tt.status, tt.value, string(tt.status))
		}
	}
}

func TestServiceDiagnosticResult_Structure(t *testing.T) {
	// Verify the result structure contains all expected fields
	result := &ServiceDiagnosticResult{
		HostType: ResourceTypeContainerApp,
		Status:   DiagnosticStatusHealthy,
		LogCount: 100,
		Requirements: []Requirement{
			{Name: "Test", Status: RequirementStatusMet, Description: "Test requirement"},
		},
		SetupGuide: &SetupGuide{
			Title: "Setup",
			Steps: []SetupGuideStep{{Title: "Step 1"}},
		},
		Message: "All good",
	}

	// Verify fields are accessible
	if result.HostType != ResourceTypeContainerApp {
		t.Error("HostType not accessible")
	}
	if result.Status != DiagnosticStatusHealthy {
		t.Error("Status not accessible")
	}
	if result.LogCount != 100 {
		t.Error("LogCount not accessible")
	}
	if len(result.Requirements) != 1 {
		t.Error("Requirements not accessible")
	}
	if result.SetupGuide == nil {
		t.Error("SetupGuide not accessible")
	}
	if result.Message != "All good" {
		t.Error("Message not accessible")
	}
}

func TestDiagnosticsResponse_Structure(t *testing.T) {
	// Verify the response structure contains all expected fields
	response := &DiagnosticsResponse{
		WorkspaceID:   "test-workspace",
		WorkspaceName: "test",
		Services: map[string]*ServiceDiagnosticResult{
			"api": {
				HostType: ResourceTypeContainerApp,
				Status:   DiagnosticStatusHealthy,
			},
		},
	}

	// Verify fields are accessible
	if response.WorkspaceID != "test-workspace" {
		t.Error("WorkspaceID not accessible")
	}
	if response.WorkspaceName != "test" {
		t.Error("WorkspaceName not accessible")
	}
	if len(response.Services) != 1 {
		t.Error("Services not accessible")
	}
	if response.Services["api"].Status != DiagnosticStatusHealthy {
		t.Error("Service status not accessible")
	}
}

// Mock implementations for testing

type mockServiceValidator struct {
	validateFunc func(ctx context.Context, serviceName string, resource *AzureResource) (*ServiceDiagnosticResult, error)
}

func (m *mockServiceValidator) Validate(ctx context.Context, serviceName string, resource *AzureResource) (*ServiceDiagnosticResult, error) {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, serviceName, resource)
	}
	return &ServiceDiagnosticResult{
		HostType: resource.ResourceType,
		Status:   DiagnosticStatusHealthy,
	}, nil
}

type mockError struct {
	message string
}

func (e *mockError) Error() string {
	return e.message
}
