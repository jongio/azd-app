package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"gopkg.in/yaml.v3"
)

// DiagnosticSettingsStatus represents the status of diagnostic settings for a service.
type DiagnosticSettingsStatus string

// DiagnosticSettingsConfigured and related constants describe whether diagnostic settings are available for a service.
const (
	DiagnosticSettingsConfigured    DiagnosticSettingsStatus = "configured"
	DiagnosticSettingsNotConfigured DiagnosticSettingsStatus = "not-configured"
	DiagnosticSettingsError         DiagnosticSettingsStatus = "error"
)

// DiagnosticSettingsCheckResult represents the result of checking diagnostic settings for a single service.
type DiagnosticSettingsCheckResult struct {
	Status                DiagnosticSettingsStatus `json:"status"`
	ResourceID            string                   `json:"resourceId,omitempty"`
	DiagnosticSettingName string                   `json:"diagnosticSettingName,omitempty"`
	Error                 string                   `json:"error,omitempty"`
	WorkspaceID           string                   `json:"workspaceId,omitempty"` // The workspace this service sends logs to
}

// DiagnosticSettingsCheckResponse represents the response for checking all services.
type DiagnosticSettingsCheckResponse struct {
	WorkspaceID string                                    `json:"workspaceId"` // Expected workspace ID
	Services    map[string]*DiagnosticSettingsCheckResult `json:"services"`    // Map of serviceName -> result
}

// DiagnosticSettingsChecker handles checking diagnostic settings for Azure resources.
type DiagnosticSettingsChecker struct {
	credential azcore.TokenCredential
	projectDir string
	discovery  *ResourceDiscovery
}

// NewDiagnosticSettingsChecker creates a new diagnostic settings checker.
func NewDiagnosticSettingsChecker(credential azcore.TokenCredential, projectDir string) *DiagnosticSettingsChecker {
	return &DiagnosticSettingsChecker{
		credential: credential,
		projectDir: projectDir,
		discovery:  NewResourceDiscovery(credential, projectDir),
	}
}

// CheckAllServices checks diagnostic settings for all discovered services.
// When logs.analytics is configured, services may not need explicit diagnostic settings
// because logs are queried directly from Log Analytics workspace.
func (c *DiagnosticSettingsChecker) CheckAllServices(ctx context.Context) (*DiagnosticSettingsCheckResponse, error) {
	// Load azure.yaml to check for logs.analytics configuration
	yamlConfig, err := c.loadAzureYaml()
	if err != nil {
		slog.Debug("failed to load azure.yaml", "error", err)
		yamlConfig = &azureYamlConfig{}
	}

	// Discover resources
	discoveryResult, err := c.discovery.Discover(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover resources: %w", err)
	}

	response := &DiagnosticSettingsCheckResponse{
		WorkspaceID: discoveryResult.LogAnalyticsWorkspaceID,
		Services:    make(map[string]*DiagnosticSettingsCheckResult),
	}

	// If global logs.analytics is configured, all services are considered configured
	// because azd queries logs directly from Log Analytics workspace
	hasGlobalAnalytics := yamlConfig.hasGlobalLogsAnalytics()

	// Check each service
	for serviceName, resource := range discoveryResult.Resources {
		slog.Debug("checking diagnostic settings", "service", serviceName, "resourceId", resource.ResourceID, "hasGlobalAnalytics", hasGlobalAnalytics)

		// If global analytics is configured or service has service-level analytics, it's configured
		if hasGlobalAnalytics || yamlConfig.hasServiceLogsAnalytics(serviceName) {
			response.Services[serviceName] = &DiagnosticSettingsCheckResult{
				Status:     DiagnosticSettingsConfigured,
				ResourceID: resource.ResourceID,
			}
			slog.Debug("service configured via logs.analytics", "service", serviceName)
			continue
		}

		if resource.ResourceID == "" {
			// Resource not deployed or not found
			response.Services[serviceName] = &DiagnosticSettingsCheckResult{
				Status: DiagnosticSettingsNotConfigured,
				Error:  "Resource not deployed or not found",
			}
			continue
		}

		// Otherwise check if diagnostic settings exist in Azure
		result := c.checkDiagnosticSettings(ctx, serviceName, resource.ResourceID, discoveryResult.LogAnalyticsWorkspaceID)
		response.Services[serviceName] = result
	}

	return response, nil
}

// checkDiagnosticSettings checks diagnostic settings for a single resource using Azure Management API.
func (c *DiagnosticSettingsChecker) checkDiagnosticSettings(ctx context.Context, serviceName, resourceID, expectedWorkspaceID string) *DiagnosticSettingsCheckResult {
	result := &DiagnosticSettingsCheckResult{
		Status:     DiagnosticSettingsNotConfigured,
		ResourceID: resourceID,
	}

	// Query Azure Management API for diagnostic settings
	// https://management.azure.com/{resourceUri}/providers/Microsoft.Insights/diagnosticSettings?api-version=2021-05-01-preview
	url := fmt.Sprintf("https://management.azure.com%s/providers/Microsoft.Insights/diagnosticSettings?api-version=2021-05-01-preview", resourceID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		result.Status = DiagnosticSettingsError
		result.Error = fmt.Sprintf("Failed to create request: %v", err)
		return result
	}

	// Get access token
	token, err := c.credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		result.Status = DiagnosticSettingsError
		result.Error = fmt.Sprintf("Failed to get access token: %v", err)
		slog.Debug("diagnostic settings auth error", "service", serviceName, "error", err)
		return result
	}

	req.Header.Set("Authorization", "Bearer "+token.Token)
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		result.Status = DiagnosticSettingsError
		result.Error = fmt.Sprintf("Failed to query diagnostic settings: %v", err)
		slog.Debug("diagnostic settings request error", "service", serviceName, "error", err)
		return result
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		// Resource exists but no diagnostic settings configured
		result.Status = DiagnosticSettingsNotConfigured
		result.Error = "No diagnostic settings found for this resource"
		return result
	}

	if resp.StatusCode == http.StatusForbidden {
		// Permission denied
		result.Status = DiagnosticSettingsError
		result.Error = "Insufficient permissions to check diagnostic settings"
		slog.Debug("diagnostic settings permission denied", "service", serviceName, "statusCode", resp.StatusCode)
		return result
	}

	if resp.StatusCode != http.StatusOK {
		// Other error
		bodyBytes, _ := io.ReadAll(resp.Body)
		result.Status = DiagnosticSettingsError
		result.Error = fmt.Sprintf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
		slog.Debug("diagnostic settings API error", "service", serviceName, "statusCode", resp.StatusCode, "body", string(bodyBytes))
		return result
	}

	// Parse response
	var diagnosticSettings diagnosticSettingsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&diagnosticSettings); err != nil {
		result.Status = DiagnosticSettingsError
		result.Error = fmt.Sprintf("Failed to parse response: %v", err)
		return result
	}

	// Check if any diagnostic setting points to the expected workspace
	if len(diagnosticSettings.Value) == 0 {
		result.Status = DiagnosticSettingsNotConfigured
		result.Error = "No diagnostic settings configured"
		return result
	}

	// Find a diagnostic setting that sends to the Log Analytics workspace
	for _, setting := range diagnosticSettings.Value {
		if setting.Properties.WorkspaceID != "" {
			// Found a setting with workspace configured
			result.DiagnosticSettingName = setting.Name
			result.WorkspaceID = setting.Properties.WorkspaceID

			// Check if it matches the expected workspace
			// The workspace ID could be in different formats:
			// - Full resource ID: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.OperationalInsights/workspaces/{name}
			// - Just the workspace name
			// - GUID
			if c.workspaceMatches(setting.Properties.WorkspaceID, expectedWorkspaceID) {
				result.Status = DiagnosticSettingsConfigured
				slog.Debug("diagnostic settings found and configured correctly",
					"service", serviceName,
					"settingName", setting.Name,
					"workspaceId", setting.Properties.WorkspaceID)
				return result
			}
		}
	}

	// Found diagnostic settings but not pointing to the expected workspace
	if result.DiagnosticSettingName != "" {
		result.Status = DiagnosticSettingsError
		result.Error = fmt.Sprintf("Diagnostic settings configured but not sending to the expected workspace (expected: %s, actual: %s)",
			expectedWorkspaceID, result.WorkspaceID)
		return result
	}

	// Found diagnostic settings but no workspace configured
	result.Status = DiagnosticSettingsNotConfigured
	result.Error = "Diagnostic settings exist but no Log Analytics workspace configured"
	return result
}

// workspaceMatches checks if two workspace identifiers refer to the same workspace.
// Handles different formats: resource IDs, workspace names, GUIDs.
func (c *DiagnosticSettingsChecker) workspaceMatches(actual, expected string) bool {
	if actual == "" || expected == "" {
		return false
	}

	// Exact match
	if actual == expected {
		return true
	}

	// Normalize both to lowercase for comparison
	actualLower := strings.ToLower(actual)
	expectedLower := strings.ToLower(expected)

	if actualLower == expectedLower {
		return true
	}

	// Extract workspace name from resource IDs if present
	actualName := extractWorkspaceName(actualLower)
	expectedName := extractWorkspaceName(expectedLower)

	// If we successfully extracted names from both, compare them
	if actualName != "" && expectedName != "" {
		return actualName == expectedName
	}

	// One might be a full resource ID and the other just a name
	if actualName != "" && actualName == expectedLower {
		return true
	}
	if expectedName != "" && expectedName == actualLower {
		return true
	}

	return false
}

// extractWorkspaceName extracts the workspace name from a resource ID.
// Example: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.OperationalInsights/workspaces/{name} -> {name}
func extractWorkspaceName(resourceID string) string {
	// Normalize to lowercase for comparison
	resourceIDLower := strings.ToLower(resourceID)
	if !strings.Contains(resourceIDLower, "/workspaces/") {
		return ""
	}

	parts := strings.Split(resourceID, "/")
	for i, part := range parts {
		if strings.ToLower(part) == "workspaces" && i+1 < len(parts) {
			return strings.ToLower(parts[i+1])
		}
	}

	return ""
}

// diagnosticSettingsListResponse represents the API response for listing diagnostic settings.
type diagnosticSettingsListResponse struct {
	Value []diagnosticSetting `json:"value"`
}

// diagnosticSetting represents a single diagnostic setting configuration.
type diagnosticSetting struct {
	ID         string                      `json:"id"`
	Name       string                      `json:"name"`
	Type       string                      `json:"type"`
	Properties diagnosticSettingProperties `json:"properties"`
}

// diagnosticSettingProperties contains the actual configuration of the diagnostic setting.
type diagnosticSettingProperties struct {
	WorkspaceID        string             `json:"workspaceId,omitempty"`
	StorageAccountID   string             `json:"storageAccountId,omitempty"`
	EventHubName       string             `json:"eventHubName,omitempty"`
	EventHubAuthRuleID string             `json:"eventHubAuthorizationRuleId,omitempty"`
	Logs               []diagnosticLog    `json:"logs,omitempty"`
	Metrics            []diagnosticMetric `json:"metrics,omitempty"`
}

// diagnosticLog represents a log category configuration.
type diagnosticLog struct {
	Category        string `json:"category,omitempty"`
	CategoryGroup   string `json:"categoryGroup,omitempty"`
	Enabled         bool   `json:"enabled"`
	RetentionPolicy *struct {
		Enabled bool `json:"enabled"`
		Days    int  `json:"days"`
	} `json:"retentionPolicy,omitempty"`
}

// diagnosticMetric represents a metric configuration.
type diagnosticMetric struct {
	Category        string `json:"category,omitempty"`
	Enabled         bool   `json:"enabled"`
	RetentionPolicy *struct {
		Enabled bool `json:"enabled"`
		Days    int  `json:"days"`
	} `json:"retentionPolicy,omitempty"`
}

// CheckSingleService checks diagnostic settings for a specific service by name.
func (c *DiagnosticSettingsChecker) CheckSingleService(ctx context.Context, serviceName string) (*DiagnosticSettingsCheckResult, error) {
	// Get the resource for this service
	resource, err := c.discovery.GetResource(ctx, serviceName)
	if err != nil {
		return &DiagnosticSettingsCheckResult{
			Status: DiagnosticSettingsError,
			Error:  fmt.Sprintf("Service not found: %v", err),
		}, nil
	}

	if resource.ResourceID == "" {
		return &DiagnosticSettingsCheckResult{
			Status: DiagnosticSettingsNotConfigured,
			Error:  "Resource not deployed",
		}, nil
	}

	// Get workspace ID from discovery
	discoveryResult, err := c.discovery.Discover(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover workspace: %w", err)
	}

	return c.checkDiagnosticSettings(ctx, serviceName, resource.ResourceID, discoveryResult.LogAnalyticsWorkspaceID), nil
}

// azureYamlConfig represents a minimal view of azure.yaml for checking logs.analytics configuration.
type azureYamlConfig struct {
	Services map[string]*serviceConfig `yaml:"services,omitempty"`
	Logs     *logsConfig               `yaml:"logs,omitempty"`
}

// serviceConfig represents a minimal service definition.
type serviceConfig struct {
	Logs *serviceLogsConfig `yaml:"logs,omitempty"`
}

// logsConfig represents the project-level logs configuration.
type logsConfig struct {
	Analytics *analyticsConfig `yaml:"analytics,omitempty"`
}

// serviceLogsConfig represents service-level logs configuration.
type serviceLogsConfig struct {
	Analytics *analyticsConfig `yaml:"analytics,omitempty"`
}

// analyticsConfig represents either global or service-level analytics config.
// We don't need the full structure, just to know if it exists.
type analyticsConfig struct {
	Workspace string `yaml:"workspace,omitempty"`
}

// loadAzureYaml loads the azure.yaml file from the project directory.
func (c *DiagnosticSettingsChecker) loadAzureYaml() (*azureYamlConfig, error) {
	yamlPath := filepath.Join(c.projectDir, "azure.yaml")

	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read azure.yaml: %w", err)
	}

	var config azureYamlConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	return &config, nil
}

// hasGlobalLogsAnalytics checks if global logs.analytics is configured.
func (c *azureYamlConfig) hasGlobalLogsAnalytics() bool {
	return c.Logs != nil && c.Logs.Analytics != nil
}

// hasServiceLogsAnalytics checks if service-level logs.analytics is configured.
func (c *azureYamlConfig) hasServiceLogsAnalytics(serviceName string) bool {
	if c.Services != nil {
		if service, ok := c.Services[serviceName]; ok {
			return service.Logs != nil && service.Logs.Analytics != nil
		}
	}
	return false
}

// CheckDiagnosticSettingsWithPipeline is an alternative implementation using Azure SDK's runtime.Pipeline.
// This is more "SDK-native" but requires more boilerplate. Keeping both for reference.
func (c *DiagnosticSettingsChecker) CheckDiagnosticSettingsWithPipeline(ctx context.Context, resourceID string) (*DiagnosticSettingsCheckResult, error) {
	result := &DiagnosticSettingsCheckResult{
		Status:     DiagnosticSettingsNotConfigured,
		ResourceID: resourceID,
	}

	// Create a pipeline for making authenticated requests
	pipeline := runtime.NewPipeline("azd-app", "1.0",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				runtime.NewBearerTokenPolicy(c.credential, []string{"https://management.azure.com/.default"}, nil),
			},
		},
		nil,
	)

	// Build the request
	url := fmt.Sprintf("https://management.azure.com%s/providers/Microsoft.Insights/diagnosticSettings?api-version=2021-05-01-preview", resourceID)
	req, err := runtime.NewRequest(ctx, http.MethodGet, url)
	if err != nil {
		result.Status = DiagnosticSettingsError
		result.Error = fmt.Sprintf("Failed to create request: %v", err)
		return result, nil
	}

	// Execute the request
	resp, err := pipeline.Do(req)
	if err != nil {
		result.Status = DiagnosticSettingsError
		result.Error = fmt.Sprintf("Failed to execute request: %v", err)
		return result, nil
	}
	defer func() { _ = resp.Body.Close() }()

	// Handle response (same as checkDiagnosticSettings)
	if resp.StatusCode == http.StatusNotFound {
		result.Status = DiagnosticSettingsNotConfigured
		result.Error = "No diagnostic settings found"
		return result, nil
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		result.Status = DiagnosticSettingsError
		result.Error = fmt.Sprintf("API error %d: %s", resp.StatusCode, string(bodyBytes))
		return result, nil
	}

	var diagnosticSettings diagnosticSettingsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&diagnosticSettings); err != nil {
		result.Status = DiagnosticSettingsError
		result.Error = fmt.Sprintf("Failed to parse response: %v", err)
		return result, nil
	}

	if len(diagnosticSettings.Value) > 0 {
		// Found at least one diagnostic setting
		result.Status = DiagnosticSettingsConfigured
		result.DiagnosticSettingName = diagnosticSettings.Value[0].Name
		if diagnosticSettings.Value[0].Properties.WorkspaceID != "" {
			result.WorkspaceID = diagnosticSettings.Value[0].Properties.WorkspaceID
		}
	}

	return result, nil
}
