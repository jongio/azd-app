package azure

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

// WorkspaceVerificationStatus represents the overall status of workspace verification.
type WorkspaceVerificationStatus string

const (
	VerificationStatusSuccess WorkspaceVerificationStatus = "success"
	VerificationStatusPartial WorkspaceVerificationStatus = "partial"
	VerificationStatusError   WorkspaceVerificationStatus = "error"
)

// ServiceVerificationStatus represents the verification status of a single service.
type ServiceVerificationStatus string

const (
	ServiceStatusOK                      ServiceVerificationStatus = "ok"
	ServiceStatusNoLogs                  ServiceVerificationStatus = "no-logs"
	ServiceStatusError                   ServiceVerificationStatus = "error"
	ServiceStatusDiagnosticNotConfigured ServiceVerificationStatus = "diagnostic-not-configured"
)

// WorkspaceInfo contains information about the Log Analytics workspace.
type WorkspaceInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ServiceVerificationResult represents the verification result for a single service.
type ServiceVerificationResult struct {
	LogCount    int                       `json:"logCount"`
	LastLogTime *time.Time                `json:"lastLogTime,omitempty"`
	Status      ServiceVerificationStatus `json:"status"`
	Message     string                    `json:"message,omitempty"`
	Error       string                    `json:"error,omitempty"`
}

// WorkspaceVerificationRequest contains the parameters for workspace verification.
type WorkspaceVerificationRequest struct {
	Services []string `json:"services,omitempty"` // Optional: specific services to check
	Timespan string   `json:"timespan,omitempty"` // Optional: ISO 8601 duration (default: PT15M)
}

// WorkspaceVerificationResponse represents the complete verification response.
type WorkspaceVerificationResponse struct {
	Status    WorkspaceVerificationStatus           `json:"status"`
	Workspace WorkspaceInfo                         `json:"workspace"`
	Results   map[string]*ServiceVerificationResult `json:"results"`
	Guidance  []string                              `json:"guidance"`
}

// WorkspaceVerifier handles workspace verification operations.
type WorkspaceVerifier struct {
	credential  azcore.TokenCredential
	projectDir  string
	discovery   *ResourceDiscovery
	diagnostics *DiagnosticSettingsChecker
}

// NewWorkspaceVerifier creates a new workspace verifier.
func NewWorkspaceVerifier(credential azcore.TokenCredential, projectDir string) *WorkspaceVerifier {
	return &WorkspaceVerifier{
		credential:  credential,
		projectDir:  projectDir,
		discovery:   NewResourceDiscovery(credential, projectDir),
		diagnostics: NewDiagnosticSettingsChecker(credential, projectDir),
	}
}

// VerifyWorkspace verifies the workspace connection by querying for recent logs.
func (v *WorkspaceVerifier) VerifyWorkspace(ctx context.Context, req *WorkspaceVerificationRequest) (*WorkspaceVerificationResponse, error) {
	// Set default timespan if not provided
	timespan := req.Timespan
	if timespan == "" {
		timespan = "PT15M" // 15 minutes default
	}

	// Parse the timespan to duration
	duration, err := parseISO8601Duration(timespan)
	if err != nil {
		return nil, fmt.Errorf("invalid timespan format: %w", err)
	}

	// Discover resources
	discoveryResult, err := v.discovery.Discover(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover resources: %w", err)
	}

	if discoveryResult.LogAnalyticsWorkspaceID == "" {
		return nil, fmt.Errorf("no Log Analytics workspace configured for this environment")
	}

	// Create Log Analytics client
	laClient, err := NewLogAnalyticsClient(v.credential, discoveryResult.LogAnalyticsWorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to create Log Analytics client: %w", err)
	}

	response := &WorkspaceVerificationResponse{
		Workspace: WorkspaceInfo{
			ID:   discoveryResult.LogAnalyticsWorkspaceID,
			Name: extractWorkspaceNameFromID(discoveryResult.LogAnalyticsWorkspaceID),
		},
		Results:  make(map[string]*ServiceVerificationResult),
		Guidance: make([]string, 0),
	}

	// Determine which services to check
	servicesToCheck := req.Services
	if len(servicesToCheck) == 0 {
		// Check all discovered services
		for serviceName := range discoveryResult.Resources {
			servicesToCheck = append(servicesToCheck, serviceName)
		}
	}

	// Track overall status
	successCount := 0
	errorCount := 0
	noLogsCount := 0

	// Check each service
	for _, serviceName := range servicesToCheck {
		slog.Debug("verifying service", "service", serviceName, "timespan", timespan)

		result := v.verifyService(ctx, laClient, serviceName, discoveryResult, duration)
		response.Results[serviceName] = result

		// Update counters
		switch result.Status {
		case ServiceStatusOK:
			successCount++
		case ServiceStatusNoLogs:
			noLogsCount++
		case ServiceStatusError, ServiceStatusDiagnosticNotConfigured:
			errorCount++
		}

		// Generate guidance for this service
		guidance := v.generateGuidance(serviceName, result)
		if guidance != "" {
			response.Guidance = append(response.Guidance, guidance)
		}
	}

	// Determine overall status
	if errorCount > 0 && successCount == 0 {
		response.Status = VerificationStatusError
	} else if successCount > 0 && (noLogsCount > 0 || errorCount > 0) {
		response.Status = VerificationStatusPartial
	} else if successCount > 0 {
		response.Status = VerificationStatusSuccess
	} else {
		response.Status = VerificationStatusError
	}

	slog.Debug("workspace verification complete",
		"status", response.Status,
		"success", successCount,
		"noLogs", noLogsCount,
		"errors", errorCount)

	return response, nil
}

// verifyService verifies a single service by querying for recent logs.
func (v *WorkspaceVerifier) verifyService(
	ctx context.Context,
	laClient *LogAnalyticsClient,
	serviceName string,
	discoveryResult *DiscoveryResult,
	duration time.Duration,
) *ServiceVerificationResult {
	result := &ServiceVerificationResult{
		Status:   ServiceStatusNoLogs,
		LogCount: 0,
	}

	// Get resource info
	resource, ok := discoveryResult.Resources[serviceName]
	if !ok || resource.ResourceID == "" {
		result.Status = ServiceStatusError
		result.Error = "Service not found or not deployed"
		return result
	}

	// First check if diagnostic settings are configured
	diagnosticResult := v.diagnostics.checkDiagnosticSettings(
		ctx,
		serviceName,
		resource.ResourceID,
		discoveryResult.LogAnalyticsWorkspaceID,
	)

	if diagnosticResult.Status == DiagnosticSettingsNotConfigured {
		result.Status = ServiceStatusDiagnosticNotConfigured
		result.Error = "DiagnosticSettingsNotConfigured: No diagnostic settings found for this resource"
		result.Message = "Configure diagnostic settings to send logs to Log Analytics workspace"
		return result
	}

	if diagnosticResult.Status == DiagnosticSettingsError {
		result.Status = ServiceStatusError
		result.Error = diagnosticResult.Error
		return result
	}

	// Query for logs
	logs, err := laClient.QueryLogs(ctx, serviceName, resource.ResourceType, duration, "")
	if err != nil {
		result.Status = ServiceStatusError
		result.Error = fmt.Sprintf("Failed to query logs: %v", err)
		slog.Debug("log query failed", "service", serviceName, "error", err)
		return result
	}

	result.LogCount = len(logs)

	if result.LogCount > 0 {
		// Found logs - service is working
		result.Status = ServiceStatusOK
		// Find the most recent log timestamp
		var mostRecent time.Time
		for _, log := range logs {
			if log.Timestamp.After(mostRecent) {
				mostRecent = log.Timestamp
			}
		}
		result.LastLogTime = &mostRecent
	} else {
		// No logs found
		result.Status = ServiceStatusNoLogs
		result.Message = "No logs found. This may be normal if the service hasn't run yet or if diagnostic settings were just configured (allow 2-5 minutes for ingestion)."
	}

	return result
}

// generateGuidance generates a human-readable guidance message for a service verification result.
func (v *WorkspaceVerifier) generateGuidance(serviceName string, result *ServiceVerificationResult) string {
	switch result.Status {
	case ServiceStatusOK:
		return fmt.Sprintf("%s: Logs flowing correctly (%d logs found)", serviceName, result.LogCount)
	case ServiceStatusNoLogs:
		return fmt.Sprintf("%s: No recent logs - wait or trigger activity", serviceName)
	case ServiceStatusDiagnosticNotConfigured:
		return fmt.Sprintf("%s: Configure diagnostic settings first", serviceName)
	case ServiceStatusError:
		errorMsg := result.Error
		if len(errorMsg) > 80 {
			errorMsg = errorMsg[:80] + "..."
		}
		return fmt.Sprintf("%s: Error - %s", serviceName, errorMsg)
	default:
		return ""
	}
}

// parseISO8601Duration parses an ISO 8601 duration string to time.Duration.
// Examples: PT15M (15 minutes), PT1H (1 hour), PT30S (30 seconds)
func parseISO8601Duration(s string) (time.Duration, error) {
	if !strings.HasPrefix(s, "P") {
		return 0, fmt.Errorf("invalid ISO 8601 duration: must start with 'P'")
	}

	s = s[1:] // Remove 'P' prefix

	// Check if it has a time component
	if !strings.HasPrefix(s, "T") {
		return 0, fmt.Errorf("invalid ISO 8601 duration: must have time component starting with 'T'")
	}

	s = s[1:] // Remove 'T' prefix

	// Check if we have any duration components after 'T'
	if s == "" {
		return 0, fmt.Errorf("invalid ISO 8601 duration: empty time component")
	}

	var duration time.Duration

	// Parse hours (H), minutes (M), seconds (S)
	for s != "" {
		i := 0
		// Find the first non-digit character
		for i < len(s) && (s[i] >= '0' && s[i] <= '9') {
			i++
		}

		if i == 0 || i >= len(s) {
			return 0, fmt.Errorf("invalid ISO 8601 duration format")
		}

		// Parse the number
		var value int
		_, err := fmt.Sscanf(s[:i], "%d", &value)
		if err != nil {
			return 0, fmt.Errorf("invalid number in duration: %w", err)
		}

		// Get the unit
		unit := s[i]
		s = s[i+1:]

		switch unit {
		case 'H':
			duration += time.Duration(value) * time.Hour
		case 'M':
			duration += time.Duration(value) * time.Minute
		case 'S':
			duration += time.Duration(value) * time.Second
		default:
			return 0, fmt.Errorf("invalid duration unit: %c", unit)
		}
	}

	return duration, nil
}

// extractWorkspaceNameFromID extracts the workspace name from a resource ID or GUID.
// If the input is a resource ID, it extracts the name.
// If it's a GUID or simple name, it returns it as-is.
func extractWorkspaceNameFromID(id string) string {
	if id == "" {
		return ""
	}

	// Check if it's a resource ID (case-insensitive)
	idLower := strings.ToLower(id)
	if strings.Contains(idLower, "/workspaces/") {
		parts := strings.Split(id, "/")
		for i, part := range parts {
			if strings.EqualFold(part, "workspaces") && i+1 < len(parts) {
				return parts[i+1]
			}
		}
	}

	// Otherwise, return the ID as-is (could be GUID or name)
	return id
}
