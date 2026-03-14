package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

// DiagnosticStatus represents the overall health status of a service's logging configuration.
type DiagnosticStatus string

// DiagnosticStatusHealthy and related constants define logging diagnostic outcomes for a service.
const (
	DiagnosticStatusHealthy       DiagnosticStatus = "healthy"        // Logs flowing correctly
	DiagnosticStatusPartial       DiagnosticStatus = "partial"        // Configured but no logs
	DiagnosticStatusNotConfigured DiagnosticStatus = "not-configured" // Missing configuration
	DiagnosticStatusError         DiagnosticStatus = "error"          // Error during validation
)

// RequirementStatus represents whether a specific requirement is met.
type RequirementStatus string

// RequirementStatusMet and related constants define whether a diagnostic requirement is satisfied.
const (
	RequirementStatusMet     RequirementStatus = "met"
	RequirementStatusNotMet  RequirementStatus = "not-met"
	RequirementStatusUnknown RequirementStatus = "unknown"
)

// Requirement represents a single configuration requirement for a service.
type Requirement struct {
	Name        string            `json:"name"`
	Status      RequirementStatus `json:"status"`
	Description string            `json:"description"`
	HowToFix    string            `json:"howToFix,omitempty"`
}

// SetupGuideStep represents a single step in the setup guide.
type SetupGuideStep struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Command     string `json:"command,omitempty"`
	Code        string `json:"code,omitempty"`
	CodeLang    string `json:"codeLang,omitempty"`
}

// SetupGuide provides step-by-step instructions for configuring a service.
type SetupGuide struct {
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Steps       []SetupGuideStep `json:"steps"`
}

// ServiceDiagnosticResult contains comprehensive diagnostic information for a single service.
type ServiceDiagnosticResult struct {
	HostType     ResourceType     `json:"hostType"`
	Status       DiagnosticStatus `json:"status"`
	LogCount     int              `json:"logCount"`
	LastLogTime  *time.Time       `json:"lastLogTime,omitempty"`
	Requirements []Requirement    `json:"requirements"`
	SetupGuide   *SetupGuide      `json:"setupGuide,omitempty"`
	Message      string           `json:"message,omitempty"`
	Error        string           `json:"error,omitempty"`
}

// DiagnosticsResponse represents the complete diagnostics response for all services.
type DiagnosticsResponse struct {
	WorkspaceID   string                              `json:"workspaceId"`
	WorkspaceName string                              `json:"workspaceName"`
	LastChecked   time.Time                           `json:"lastChecked"`
	Services      map[string]*ServiceDiagnosticResult `json:"services"`
}

// ServiceValidator is the interface that all host-type validators must implement.
type ServiceValidator interface {
	// Validate performs comprehensive validation of a service's logging configuration.
	// Returns diagnostic result with status, requirements, and setup guide if needed.
	Validate(ctx context.Context, serviceName string, resource *AzureResource) (*ServiceDiagnosticResult, error)
}

// DiagnosticsEngine orchestrates validation across all services.
type DiagnosticsEngine struct {
	credential azcore.TokenCredential
	projectDir string
	discovery  *ResourceDiscovery
	validators map[ResourceType]ServiceValidator
}

// NewDiagnosticsEngine creates a new diagnostics engine.
func NewDiagnosticsEngine(credential azcore.TokenCredential, projectDir string) *DiagnosticsEngine {
	engine := &DiagnosticsEngine{
		credential: credential,
		projectDir: projectDir,
		discovery:  NewResourceDiscovery(credential, projectDir),
		validators: make(map[ResourceType]ServiceValidator),
	}

	// Register validators for each host type
	// TODO: Register validators as they are implemented
	// engine.validators[ResourceTypeContainerApp] = NewContainerAppValidator(credential, projectDir)
	// engine.validators[ResourceTypeFunction] = NewFunctionValidator(credential, projectDir)
	// engine.validators[ResourceTypeAppService] = NewAppServiceValidator(credential, projectDir)

	return engine
}

// RunDiagnostics performs comprehensive diagnostics on all discovered services.
func (e *DiagnosticsEngine) RunDiagnostics(ctx context.Context) (*DiagnosticsResponse, error) {
	// Discover resources
	discoveryResult, err := e.discovery.Discover(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover resources: %w", err)
	}

	// Initialize validators with workspace ID
	if discoveryResult.LogAnalyticsWorkspaceID != "" {
		e.initializeValidators(discoveryResult.LogAnalyticsWorkspaceID)
	}

	response := &DiagnosticsResponse{
		WorkspaceID:   discoveryResult.LogAnalyticsWorkspaceID,
		WorkspaceName: extractWorkspaceNameFromID(discoveryResult.LogAnalyticsWorkspaceID),
		LastChecked:   time.Now(),
		Services:      make(map[string]*ServiceDiagnosticResult),
	}

	// Validate each discovered service
	for serviceName, resource := range discoveryResult.Resources {
		result := e.validateService(ctx, serviceName, resource)
		response.Services[serviceName] = result
	}

	return response, nil
}

// initializeValidators creates and registers validators for supported host types.
func (e *DiagnosticsEngine) initializeValidators(workspaceID string) {
	// Container Apps validator
	if caValidator, err := NewContainerAppValidator(e.credential, e.projectDir, workspaceID); err == nil {
		e.validators[ResourceTypeContainerApp] = caValidator
	}

	// Functions validator
	if funcValidator, err := NewFunctionValidator(e.credential, e.projectDir, workspaceID); err == nil {
		e.validators[ResourceTypeFunction] = funcValidator
	}

	// App Service validator
	if appValidator, err := NewAppServiceValidator(e.credential, e.projectDir, workspaceID); err == nil {
		e.validators[ResourceTypeAppService] = appValidator
	}
}

// validateService validates a single service using the appropriate validator.
func (e *DiagnosticsEngine) validateService(ctx context.Context, serviceName string, resource *AzureResource) *ServiceDiagnosticResult {
	// Get validator for this resource type
	validator, ok := e.validators[resource.ResourceType]
	if !ok {
		// No validator registered for this resource type
		return &ServiceDiagnosticResult{
			HostType: resource.ResourceType,
			Status:   DiagnosticStatusError,
			Message:  fmt.Sprintf("Validation not yet implemented for host type: %s", resource.ResourceType),
			Requirements: []Requirement{
				{
					Name:        "Validator Support",
					Status:      RequirementStatusNotMet,
					Description: fmt.Sprintf("Diagnostic validation for %s is not yet implemented", resource.ResourceType),
				},
			},
		}
	}

	// Run validation
	result, err := validator.Validate(ctx, serviceName, resource)
	if err != nil {
		return &ServiceDiagnosticResult{
			HostType: resource.ResourceType,
			Status:   DiagnosticStatusError,
			Error:    fmt.Sprintf("Validation failed: %v", err),
		}
	}

	return result
}

// RegisterValidator registers a validator for a specific resource type.
// This allows validators to be registered after engine creation.
func (e *DiagnosticsEngine) RegisterValidator(resourceType ResourceType, validator ServiceValidator) {
	e.validators[resourceType] = validator
}
