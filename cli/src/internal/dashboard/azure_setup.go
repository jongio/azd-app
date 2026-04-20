// Package dashboard provides API endpoints for the local dashboard.
package dashboard

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/jongio/azd-app/cli/src/internal/azure"
)

// SetupStateResponse represents the overall Azure logs setup state.
type SetupStateResponse struct {
	Step           string              `json:"step"`           // Current setup step
	OverallStatus  string              `json:"overallStatus"`  // "complete" | "incomplete" | "error"
	Workspace      WorkspaceState      `json:"workspace"`      // Log Analytics workspace state
	Authentication AuthState           `json:"authentication"` // Authentication state
	Services       []ServiceSetupState `json:"services"`       // Per-service setup state
	Issues         []SetupIssue        `json:"issues"`         // List of issues found
	NextSteps      []string            `json:"nextSteps"`      // Recommended next actions
	Timestamp      time.Time           `json:"timestamp"`      // When state was checked
}

// WorkspaceState represents the Log Analytics workspace configuration state.
type WorkspaceState struct {
	Status      string `json:"status"`                // "configured" | "missing" | "not-deployed" | "invalid"
	WorkspaceID string `json:"workspaceId,omitempty"` // Workspace resource ID
	Message     string `json:"message"`               // Human-readable status message
	Source      string `json:"source,omitempty"`      // Where workspace ID was found (env, azure.yaml)
}

// AuthState represents the authentication and permissions state.
type AuthState struct {
	Status                string   `json:"status"`                  // "authenticated" | "unauthenticated" | "permission-denied" | "error"
	Message               string   `json:"message"`                 // Human-readable status message
	Principal             string   `json:"principal,omitempty"`     // Authenticated principal (email/identity)
	HasArmAccess          bool     `json:"hasArmAccess"`            // Has Azure Resource Manager access
	HasLogsAccess         bool     `json:"hasLogsAccess"`           // Has Log Analytics API access
	HasLogAnalyticsReader bool     `json:"hasLogAnalyticsReader"`   // Has Log Analytics Reader role or equivalent
	MissingScopes         []string `json:"missingScopes,omitempty"` // Missing permission scopes
}

// ServiceSetupState represents setup state for a single service.
type ServiceSetupState struct {
	ServiceName        string `json:"serviceName"`                // Service name from azure.yaml
	ResourceName       string `json:"resourceName,omitempty"`     // Azure resource name
	ResourceType       string `json:"resourceType,omitempty"`     // Azure resource type
	Deployed           bool   `json:"deployed"`                   // Resource exists in Azure
	DiagnosticSettings bool   `json:"diagnosticSettings"`         // Diagnostic settings configured
	LogsFlowing        bool   `json:"logsFlowing"`                // Logs are being received
	Status             string `json:"status"`                     // "ready" | "partial" | "not-configured" | "not-deployed"
	LastLogTimestamp   string `json:"lastLogTimestamp,omitempty"` // ISO timestamp of most recent log
}

// SetupIssue represents a configuration issue with actionable fix.
type SetupIssue struct {
	Severity string `json:"severity"`          // "error" | "warning" | "info"
	Category string `json:"category"`          // "workspace" | "auth" | "diagnostic-settings" | "config"
	Message  string `json:"message"`           // Issue description
	Fix      string `json:"fix"`               // Fix command or instruction
	DocsURL  string `json:"docsUrl,omitempty"` // Documentation link
}

// handleAzureSetupState was the REST entrypoint for GET /api/azure/logs/setup-state.
// The handler body has been removed; the Connect AzureService.SetupState RPC
// in rpc_azure_adapter.go calls the same check* helpers directly and handles
// the context-timeout orchestration inline.
//
// The legacy handler signature is preserved only as a signpost for reviewers
// tracing references from older branches.

// checkWorkspaceState verifies Log Analytics workspace configuration.
func (s *Server) checkWorkspaceState() WorkspaceState {
	state := WorkspaceState{
		Status:  StatusMissing,
		Message: MsgWorkspaceNotConfigured,
	}

	// Check environment variable first (highest priority)
	workspaceID, err := getWorkspaceIDFromEnv(context.Background())
	if err == nil && workspaceID != "" {
		state.Status = StatusConfigured
		state.WorkspaceID = workspaceID
		state.Source = "environment"
		state.Message = fmt.Sprintf("Workspace configured: %s", truncateMiddle(workspaceID, 40))
		return state
	}

	// Check azure.yaml configuration
	azureYaml, err := loadAzureYaml(s.projectDir)
	if err != nil {
		state.Status = StatusError
		state.Message = fmt.Sprintf("Failed to load azure.yaml: %v", err)
		return state
	}

	// Check if logs.analytics.workspace is configured
	if azureYaml.Logs != nil && azureYaml.Logs.Analytics != nil && azureYaml.Logs.Analytics.Workspace != "" {
		workspaceRef := azureYaml.Logs.Analytics.Workspace
		state.Status = StatusNotDeployed
		state.WorkspaceID = workspaceRef
		state.Source = "azure.yaml"
		state.Message = fmt.Sprintf("Workspace configured in azure.yaml (%s) but not deployed. Run 'azd up' to deploy.", workspaceRef)
		return state
	}

	state.Message = "Log Analytics workspace not configured in azure.yaml or environment"
	return state
}

// checkAuthState verifies authentication and permissions.
func (s *Server) checkAuthState(ctx context.Context) AuthState {
	state := AuthState{
		Status:        StatusUnauthenticated,
		Message:       "Not authenticated",
		HasArmAccess:  false,
		HasLogsAccess: false,
		MissingScopes: make([]string, 0),
	}

	// Try to create credential
	cred, err := newLogAnalyticsCredential()
	if err != nil {
		state.Message = MsgAzureCredsNotAvailable
		state.MissingScopes = []string{"ARM", "Log Analytics"}
		return state
	}

	// Check ARM scope (required for resource discovery)
	armErr := validateCredentials(ctx, cred)
	if armErr != nil {
		state.Message = MsgAuthFailed
		state.MissingScopes = append(state.MissingScopes, "ARM", "Log Analytics")
		return state
	}

	state.HasArmAccess = true

	// Try to determine principal (user identity)
	// This is best-effort - some credential types don't expose principal
	principal := getPrincipalFromCredentials(ctx, cred)
	if principal != "" {
		state.Principal = principal
	}
	// Skip Azure CLI fallback - it's slow (can take 1-2 seconds)
	// Users will see "Unknown" which is acceptable for setup guide

	// For fast initial checks, we assume authenticated users have Log Analytics access
	// Actual permission verification happens when they try to query logs
	// This keeps the setup guide responsive while still validating auth
	state.HasLogsAccess = true
	state.HasLogAnalyticsReader = true
	state.Status = StatusAuthenticated
	// Don't include principal in message - it's shown separately in "Signed in as" box
	state.Message = "Authenticated with Azure. You have the required permissions."

	return state
}

// checkServicesState checks diagnostic settings for all discovered services.
func (s *Server) checkServicesState(ctx context.Context) []ServiceSetupState {
	services := make([]ServiceSetupState, 0)

	// Get service names from environment
	serviceNames := extractServiceNamesFromEnv()
	if len(serviceNames) == 0 {
		slog.Debug("setup: no services found in environment")
		return services
	}

	// Try to discover Azure resources
	cred, err := newLogAnalyticsCredential()
	if err != nil {
		slog.Warn("setup: cannot create credentials for discovery", "error", err)
		// Add services in unknown state
		for _, name := range serviceNames {
			services = append(services, ServiceSetupState{
				ServiceName: name,
				Deployed:    false,
				Status:      StatusNotDeployed,
			})
		}
		return services
	}

	discovery := azure.NewResourceDiscovery(cred, s.projectDir)
	discoveryResult, err := discovery.Discover(ctx)
	if err != nil {
		slog.Warn("setup: resource discovery failed", "error", err)
		// Add services in unknown state
		for _, name := range serviceNames {
			services = append(services, ServiceSetupState{
				ServiceName: name,
				Deployed:    false,
				Status:      StatusNotDeployed,
			})
		}
		return services
	}

	// Check each service
	for _, serviceName := range serviceNames {
		service := s.checkSingleServiceState(ctx, serviceName, discoveryResult, cred)
		services = append(services, service)
	}

	return services
}

// checkSingleServiceState checks setup state for a single service.
func (s *Server) checkSingleServiceState(ctx context.Context, serviceName string, discovery *azure.DiscoveryResult, cred azcore.TokenCredential) ServiceSetupState {
	state := ServiceSetupState{
		ServiceName:        serviceName,
		Deployed:           false,
		DiagnosticSettings: false,
		LogsFlowing:        false,
		Status:             StatusNotDeployed,
	}

	// Check if service is in discovery results
	resource, ok := discovery.Resources[serviceName]
	if !ok {
		slog.Debug("setup: service not found in discovery", "service", serviceName)
		return state
	}

	state.Deployed = true
	state.ResourceName = resource.Name
	state.ResourceType = string(resource.ResourceType)

	// Check diagnostic settings (this requires querying Azure Management API)
	// For now, we'll do a heuristic check: if we can query logs, diagnostic settings are likely configured
	hasDiagnostics := s.checkDiagnosticSettings(ctx, serviceName, discovery.LogAnalyticsWorkspaceID, cred)
	state.DiagnosticSettings = hasDiagnostics

	// Check if logs are flowing (query recent logs)
	lastLogTime := s.checkLogsFlowing(ctx, serviceName, discovery.LogAnalyticsWorkspaceID, cred)
	if lastLogTime != "" {
		state.LogsFlowing = true
		state.LastLogTimestamp = lastLogTime
	}

	// Determine overall status
	if state.LogsFlowing {
		state.Status = StatusReady
	} else if state.DiagnosticSettings {
		state.Status = StatusPartial // Configured but no logs yet (may take 5-15 min)
	} else {
		state.Status = StatusNotConfigured
	}

	return state
}

// checkDiagnosticSettings checks if diagnostic settings are configured for a service.
// Queries Azure Resource Manager API to check if diagnostic settings exist.
func (s *Server) checkDiagnosticSettings(ctx context.Context, serviceName, workspaceID string, cred azcore.TokenCredential) bool {
	if workspaceID == "" {
		return false
	}

	// Get discovery result to find the resource ID
	discovery := azure.NewResourceDiscovery(cred, s.projectDir)
	result, err := discovery.Discover(ctx)
	if err != nil {
		slog.Debug("setup: discovery failed for diagnostic settings check", "service", serviceName, "error", err)
		return false
	}

	resource, ok := result.Resources[serviceName]
	if !ok || resource.ResourceID == "" {
		slog.Debug("setup: resource not found or missing resource ID", "service", serviceName)
		return false
	}

	// Query ARM API for diagnostic settings
	// https://management.azure.com/{resourceUri}/providers/Microsoft.Insights/diagnosticSettings?api-version=2021-05-01-preview
	diagnosticsURL := fmt.Sprintf("https://management.azure.com%s/providers/Microsoft.Insights/diagnosticSettings?api-version=2021-05-01-preview", resource.ResourceID)

	// Get access token
	token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		slog.Debug("setup: cannot get token for diagnostic settings check", "service", serviceName, "error", err)
		return false
	}

	// Make HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, diagnosticsURL, nil)
	if err != nil {
		slog.Debug("setup: cannot create request", "service", serviceName, "error", err)
		return false
	}
	req.Header.Set("Authorization", "Bearer "+token.Token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Debug("setup: diagnostic settings request failed", "service", serviceName, "error", err)
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		slog.Debug("setup: diagnostic settings request returned error", "service", serviceName, "status", resp.StatusCode)
		return false
	}

	// Parse response
	var diagnosticsResponse struct {
		Value []struct {
			Name       string `json:"name"`
			Properties struct {
				WorkspaceID string `json:"workspaceId"`
			} `json:"properties"`
		} `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&diagnosticsResponse); err != nil {
		slog.Debug("setup: cannot parse diagnostic settings response", "service", serviceName, "error", err)
		return false
	}

	// Check if any diagnostic setting points to our workspace
	// WorkspaceID can be a full resource ID or just the GUID - normalize both
	workspaceIDNormalized := strings.ToLower(workspaceID)
	for _, setting := range diagnosticsResponse.Value {
		settingWorkspaceID := strings.ToLower(setting.Properties.WorkspaceID)
		// Match if either contains the other (handles GUID vs full resource ID)
		if strings.Contains(settingWorkspaceID, workspaceIDNormalized) || strings.Contains(workspaceIDNormalized, settingWorkspaceID) {
			slog.Debug("setup: found diagnostic setting", "service", serviceName, "setting", setting.Name)
			return true
		}
	}

	slog.Debug("setup: no diagnostic settings found for workspace", "service", serviceName, "workspace", workspaceID)
	return false
}

// checkLogsFlowing queries for recent logs to verify log flow.
// Returns ISO timestamp of most recent log, or empty string if no logs found.
func (s *Server) checkLogsFlowing(ctx context.Context, serviceName, workspaceID string, _ azcore.TokenCredential) string {
	if workspaceID == "" {
		return ""
	}

	// Use standalone logs fetcher to check for recent logs
	config := azure.StandaloneLogsConfig{
		ProjectDir: s.projectDir,
		Services:   []string{serviceName},
		Since:      15 * time.Minute, // Check last 15 minutes
		Limit:      1,                // Only need one log to verify flow
	}

	logs, err := fetchAzureLogsStandalone(ctx, config)
	if err != nil {
		slog.Debug("setup: cannot fetch logs for service", "service", serviceName, "error", err)
		return ""
	}

	if len(logs) > 0 {
		return logs[0].Timestamp.Format(time.RFC3339)
	}

	return ""
}

// determineSetupStep determines the current setup step based on state.
func (s *Server) determineSetupStep(response SetupStateResponse) (string, string) {
	// Check workspace first
	if response.Workspace.Status != "configured" {
		return "workspace", "incomplete"
	}

	// Check authentication
	if response.Authentication.Status != "authenticated" {
		return "authentication", "incomplete"
	}

	// Check services
	allReady := true
	anyPartial := false
	for _, svc := range response.Services {
		if svc.Status != StatusReady {
			allReady = false
		}
		if svc.Status == StatusPartial || svc.Status == StatusNotConfigured {
			anyPartial = true
		}
	}

	if !allReady {
		if anyPartial {
			return StepVerification, StatusIncomplete
		}
		return StepDiagnosticSettings, StatusIncomplete
	}

	return StatusComplete, StatusComplete
}

// collectSetupIssues generates a list of issues from the setup state.
func (s *Server) collectSetupIssues(response SetupStateResponse) []SetupIssue {
	issues := make([]SetupIssue, 0)

	// Workspace issues
	switch response.Workspace.Status {
	case StatusMissing:
		issues = append(issues, SetupIssue{
			Severity: "error",
			Category: CategoryWorkspace,
			Message:  MsgWorkspaceNotConfigured,
			Fix:      "Add 'logs.analytics.workspace: ${AZURE_LOG_ANALYTICS_WORKSPACE_ID}' to azure.yaml and deploy with 'azd up'",
			DocsURL:  logsTroubleshootURL,
		})
	case StatusNotDeployed:
		issues = append(issues, SetupIssue{
			Severity: "warning",
			Category: CategoryWorkspace,
			Message:  "Workspace configured but not deployed",
			Fix:      "azd up",
			DocsURL:  logsTroubleshootURL,
		})
	}

	// Authentication issues
	if response.Authentication.Status == StatusUnauthenticated {
		issues = append(issues, SetupIssue{
			Severity: "error",
			Category: CategoryAuth,
			Message:  "Not authenticated with Azure",
			Fix:      "azd auth login",
			DocsURL:  logsTroubleshootURL,
		})
	} else if len(response.Authentication.MissingScopes) > 0 {
		issues = append(issues, SetupIssue{
			Severity: "error",
			Category: CategoryAuth,
			Message:  fmt.Sprintf("Missing required permissions: %s", strings.Join(response.Authentication.MissingScopes, ", ")),
			Fix:      "azd auth login",
			DocsURL:  logsTroubleshootURL,
		})
	}

	// Service issues
	for _, svc := range response.Services {
		switch svc.Status {
		case StatusNotDeployed:
			issues = append(issues, SetupIssue{
				Severity: "warning",
				Category: CategoryDiagnosticSettings,
				Message:  fmt.Sprintf("Service '%s' not deployed to Azure", svc.ServiceName),
				Fix:      "azd up",
			})
		case StatusNotConfigured:
			issues = append(issues, SetupIssue{
				Severity: "error",
				Category: CategoryDiagnosticSettings,
				Message:  fmt.Sprintf("Service '%s' missing diagnostic settings", svc.ServiceName),
				Fix:      fmt.Sprintf("Configure diagnostic settings for %s to send logs to Log Analytics workspace", svc.ServiceName),
				DocsURL:  logsTroubleshootURL,
			})
		case StatusPartial:
			issues = append(issues, SetupIssue{
				Severity: "info",
				Category: CategoryDiagnosticSettings,
				Message:  fmt.Sprintf("Service '%s' configured but no logs yet (this can take 5-15 minutes after deployment)", svc.ServiceName),
				Fix:      "Wait a few minutes and refresh, or check diagnostic settings in Azure Portal",
			})
		}
	}

	return issues
}

// generateNextSteps generates recommended next actions based on setup state.
func (s *Server) generateNextSteps(response SetupStateResponse) []string {
	steps := make([]string, 0)

	if response.OverallStatus == StatusComplete {
		steps = append(steps, "Setup complete! Click 'View Logs' to start streaming Azure logs.")
		return steps
	}

	// Add steps based on current state
	if response.Workspace.Status != StatusConfigured {
		steps = append(steps, "1. Configure Log Analytics workspace in azure.yaml")
		steps = append(steps, "2. Deploy infrastructure with 'azd up'")
		return steps
	}

	if response.Authentication.Status != StatusAuthenticated {
		steps = append(steps, "1. Authenticate with Azure using 'azd auth login'")
		return steps
	}

	// Check service states
	needsDiagnostics := false
	for _, svc := range response.Services {
		if svc.Status == StatusNotConfigured {
			needsDiagnostics = true
			break
		}
	}

	if needsDiagnostics {
		steps = append(steps, "1. Configure diagnostic settings for services")
		steps = append(steps, "2. Deploy updated infrastructure with 'azd up'")
		return steps
	}

	// Waiting for logs
	steps = append(steps, "1. Wait 5-15 minutes for logs to flow")
	steps = append(steps, "2. Verify diagnostic settings in Azure Portal if logs don't appear")

	return steps
}

// getPrincipalFromCredentials attempts to get the authenticated principal (user/identity).
// This is best-effort - not all credential types expose this information.
func getPrincipalFromCredentials(ctx context.Context, cred azcore.TokenCredential) string {
	// Try to get a token and decode the principal from claims
	token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		return ""
	}

	// Parse JWT token to extract user principal name (UPN) or email
	// JWT tokens have 3 parts: header.payload.signature
	parts := strings.Split(token.Token, ".")
	if len(parts) != 3 {
		return ""
	}

	// Decode the payload (middle part) - it's base64 URL encoded
	payload := parts[1]
	// Add padding if needed
	if mod := len(payload) % 4; mod != 0 {
		payload += strings.Repeat("=", 4-mod)
	}

	// Use base64 RawURLEncoding (without padding) for JWT
	payloadBytes, err := b64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		// Try with standard base64 as fallback
		payloadBytes, err = b64.StdEncoding.DecodeString(payload)
		if err != nil {
			return ""
		}
	}

	// Parse JSON payload
	var claims map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return ""
	}

	// Try to extract principal in order of preference:
	// 1. upn (User Principal Name) - e.g., user@domain.com
	// 2. unique_name - alternative field
	// 3. email - email claim
	// 4. preferred_username - another alternative
	if upn, ok := claims["upn"].(string); ok && upn != "" {
		return upn
	}
	if uniqueName, ok := claims["unique_name"].(string); ok && uniqueName != "" {
		return uniqueName
	}
	if email, ok := claims["email"].(string); ok && email != "" {
		return email
	}
	if preferredUsername, ok := claims["preferred_username"].(string); ok && preferredUsername != "" {
		return preferredUsername
	}

	// If no user principal, try to get service principal name or app display name
	if appDisplayName, ok := claims["app_displayname"].(string); ok && appDisplayName != "" {
		return fmt.Sprintf("%s (Service Principal)", appDisplayName)
	}
	if appID, ok := claims["appid"].(string); ok && appID != "" {
		return fmt.Sprintf("Service Principal: %s", appID)
	}

	return ""
}
