// Package azure provides Azure cloud integration for log streaming and resource discovery.
package azure

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"gopkg.in/yaml.v3"
)

// KQL query constants
const (
	kqlWhereTimeAgo      = "| where TimeGenerated > ago(%s)\n"
	kqlWhereTimeDateTime = "| where TimeGenerated > datetime('%s')\n"
	kqlWhere             = "| where %s\n"
	kqlResourceIDFilter  = "_ResourceId contains '%s'"
	azureDirName         = ".azure"
)

// StandaloneLogsConfig holds configuration for standalone Azure log fetching.
type StandaloneLogsConfig struct {
	ProjectDir  string
	WorkspaceID string        // Log Analytics workspace GUID
	Services    []string      // Service names to filter (empty = all)
	Since       time.Duration // Time range to query
	Limit       int           // Max number of logs
}

// ServiceInfo holds information about a service for log querying.
type ServiceInfo struct {
	Name         string       // azure.yaml service name
	AzureName    string       // Azure resource name from SERVICE_*_NAME
	Host         string       // azure.yaml host type
	ResourceType ResourceType // Mapped resource type
}

// HostToResourceType maps azure.yaml host values to ResourceType.
var HostToResourceType = map[string]ResourceType{
	"containerapp": ResourceTypeContainerApp,
	"appservice":   ResourceTypeAppService,
	"function":     ResourceTypeFunction,
	"aks":          ResourceTypeAKS,
	"aci":          ResourceTypeContainerInstance,
}

// getServicesFromAzureYAML reads azure.yaml and returns service info including host types.
func getServicesFromAzureYAML(projectDir string) ([]ServiceInfo, error) {
	azureYAMLPath := filepath.Join(projectDir, "azure.yaml")
	content, err := os.ReadFile(azureYAMLPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read azure.yaml: %w", err)
	}

	var config struct {
		Services map[string]struct {
			Host string `yaml:"host"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	// Get service name mappings from env
	serviceNameMap := getServiceNameMap(projectDir)

	// Debug: log service name mapping
	if os.Getenv("AZD_APP_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Service name map from environment: %v\n", serviceNameMap)
	}

	var services []ServiceInfo
	for name, svc := range config.Services {
		// Skip local-only services
		if svc.Host == "local" || svc.Host == "" {
			continue
		}

		info := ServiceInfo{
			Name: name,
			Host: svc.Host,
		}

		// Map host to resource type
		if rt, ok := HostToResourceType[svc.Host]; ok {
			info.ResourceType = rt
		} else {
			info.ResourceType = ResourceTypeContainerApp // default fallback
		}

		// Get Azure resource name from env
		if azureName, ok := serviceNameMap[strings.ToLower(name)]; ok {
			info.AzureName = azureName
		} else {
			info.AzureName = name // fallback to azure.yaml name
		}

		// Debug: log each service mapping
		if os.Getenv("AZD_APP_DEBUG") == "true" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Service %s: host=%s, resourceType=%s, azureName=%s\n",
				name, svc.Host, info.ResourceType, info.AzureName)
		}

		services = append(services, info)
	}

	return services, nil
}

// getServiceNameMap returns a map of azure.yaml service names to Azure resource names.
// Uses environment variables directly since the azd extension framework provides them.
func getServiceNameMap(projectDir string) map[string]string {
	serviceNameMap := make(map[string]string)

	// When running as an azd extension, all environment variables are already available
	// via os.Environ(). No need to shell out to 'azd env get-values'.
	for _, line := range os.Environ() {
		if strings.HasPrefix(line, "SERVICE_") && strings.Contains(line, "_NAME=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := parts[0]
				key = strings.TrimPrefix(key, "SERVICE_")
				key = strings.TrimSuffix(key, "_NAME")
				key = strings.ToLower(strings.ReplaceAll(key, "_", "-"))
				value := strings.Trim(parts[1], "\"")
				if value != "" {
					serviceNameMap[key] = value
				}
			}
		}
	}

	return serviceNameMap
}

// FetchAzureLogsStandalone fetches Azure logs directly without requiring the dashboard.
// This is used by `azd app logs --source azure` when no dashboard is running.
func FetchAzureLogsStandalone(ctx context.Context, config StandaloneLogsConfig) ([]LogEntry, error) {
	// Get workspace ID from environment if not provided
	workspaceID := config.WorkspaceID
	if workspaceID == "" {
		if wsID, err := GetWorkspaceIDFromEnv(ctx); err == nil {
			workspaceID = wsID
		}

		if workspaceID == "" {
			// Try auto-discovery
			if discovered, wasDiscovered, err := DiscoverAndStoreWorkspaceID(ctx); err == nil && wasDiscovered {
				workspaceID = discovered
				if os.Getenv("AZD_APP_DEBUG") == "true" {
					fmt.Fprintf(os.Stderr, "[DEBUG] Auto-discovered workspace ID: %s\n", workspaceID)
				}
			} else if err != nil && os.Getenv("AZD_APP_DEBUG") == "true" {
				fmt.Fprintf(os.Stderr, "[DEBUG] Workspace discovery failed: %v\n", err)
			}
		}

		if workspaceID == "" {
			return nil, &AzureLogsError{
				Code:    "NO_WORKSPACE",
				Message: "Log Analytics workspace not configured",
				Action:  "Deploy with 'azd up' or set AZURE_LOG_ANALYTICS_WORKSPACE_GUID",
			}
		}
	}

	// Get credential
	cred, err := NewLogAnalyticsCredential()
	if err != nil {
		// Clear token cache on credential errors
		ClearTokenCacheOnError(err)
		return nil, &AzureLogsError{
			Code:    "AUTH_REQUIRED",
			Message: "Azure authentication required",
			Action:  "Run 'azd auth login' to authenticate",
			Command: "azd auth login",
		}
	}

	// Get or create cached client
	client, err := GetOrCreateLogAnalyticsClient(ctx, cred, workspaceID)
	if err != nil {
		// Clear token cache on client creation errors
		ClearTokenCacheOnError(err)
		return nil, &AzureLogsError{
			Code:    "CLIENT_ERROR",
			Message: fmt.Sprintf("Failed to create Log Analytics client: %v", err),
			Action:  "Check your Azure configuration",
		}
	}

	// Set defaults
	since := config.Since
	if since == 0 {
		since = 1 * time.Hour
	}
	limit := config.Limit
	if limit == 0 {
		limit = 500
	}

	// Get all services from azure.yaml with their host types
	allServices, err := getServicesFromAzureYAML(config.ProjectDir)
	if err != nil {
		slog.Warn("Failed to read azure.yaml services", "error", err)
		// Fall back to Container App only if we can't read azure.yaml
		allServices = []ServiceInfo{{ResourceType: ResourceTypeContainerApp}}
	}

	slog.Debug("Read azure.yaml services", "count", len(allServices), "services", allServices)

	// Filter services if specific ones requested
	var targetServices []ServiceInfo
	if len(config.Services) > 0 {
		serviceMap := make(map[string]bool)
		for _, s := range config.Services {
			serviceMap[strings.ToLower(s)] = true
		}
		for _, svc := range allServices {
			if serviceMap[strings.ToLower(svc.Name)] {
				targetServices = append(targetServices, svc)
			}
		}
		slog.Debug("Filtered services", "requested", config.Services, "matched", targetServices)
	} else {
		targetServices = allServices
	}

	// Group services by resource type
	servicesByType := make(map[ResourceType][]ServiceInfo)
	for _, svc := range targetServices {
		servicesByType[svc.ResourceType] = append(servicesByType[svc.ResourceType], svc)
	}

	// Debug: log service grouping
	if os.Getenv("AZD_APP_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Target services: %v\n", targetServices)
		fmt.Fprintf(os.Stderr, "[DEBUG] Services by type: %v\n", servicesByType)
	}

	// Query each resource type and collect all entries
	var allEntries []LogEntry
	var queryErrors []string
	var successCount int
	for resourceType, services := range servicesByType {
		// Get Azure names for filtering
		var azureNames []string
		for _, svc := range services {
			azureNames = append(azureNames, svc.AzureName)
		}

		query := buildStandaloneQueryForType(resourceType, azureNames, since, limit)

		slog.Debug("Executing KQL query", "resourceType", resourceType, "services", services, "azureNames", azureNames, "query", query)

		entries, err := client.QueryLogs(ctx, "", resourceType, since, query)
		if err != nil {
			// Don't log context cancellation - it's expected when client aborts request
			if ctx.Err() == nil {
				// Log error but continue with other resource types
				slog.Warn("Query failed for resource type", "resourceType", resourceType, "error", err)
				if os.Getenv("AZD_APP_DEBUG") == "true" {
					fmt.Fprintf(os.Stderr, "[DEBUG] Query failed for %s: %v\n", resourceType, err)
				}
			}
			queryErrors = append(queryErrors, fmt.Sprintf("%s: %v", resourceType, err))
			continue
		}

		successCount++

		// Map Azure resource names back to logical service names from azure.yaml
		entries = mapServiceNames(entries, services)

		slog.Debug("Query succeeded", "resourceType", resourceType, "services", services, "entries", len(entries))
		allEntries = append(allEntries, entries...)
	}

	slog.Debug("Total entries collected", "count", len(allEntries))

	// Fix 2: Bubble error when all queries fail with actionable guidance
	if successCount == 0 {
		errMsg := "Azure log queries failed for all resource types"
		if len(queryErrors) > 0 {
			errMsg += fmt.Sprintf(":\n  %s", strings.Join(queryErrors, "\n  "))
		}
		return nil, &AzureLogsError{
			Code:    "QUERY_FAILED",
			Message: errMsg,
			Action:  "Verify Log Analytics workspace access and service permissions",
		}
	}

	// Fix 2: Provide actionable guidance when queries succeed but return zero results
	if len(allEntries) == 0 {
		if len(queryErrors) > 0 {
			errMsg := fmt.Sprintf("Azure log queries returned no results. Some queries failed:\n  %s", strings.Join(queryErrors, "\n  "))
			return nil, &AzureLogsError{
				Code:    "NO_RESULTS",
				Message: errMsg,
				Action:  "Check service names, verify workspace has data (ingestion delay is 1-5 minutes), and confirm permissions",
			}
		} else if len(config.Services) > 0 {
			return nil, &AzureLogsError{
				Code:    "NO_RESULTS",
				Message: fmt.Sprintf("Azure logs returned no results for service(s): %s", strings.Join(config.Services, ", ")),
				Action:  "Verify service names match azure.yaml, check if services are deployed and generating logs, and note that ingestion delay is 1-5 minutes",
			}
		} else {
			return nil, &AzureLogsError{
				Code:    "NO_RESULTS",
				Message: "Azure logs returned no results",
				Action:  "Verify services are deployed and generating logs. Note: Azure logs have a 1-5 minute ingestion delay",
			}
		}
	}

	// Sort all entries by timestamp descending
	sortLogEntriesByTimeDesc(allEntries)

	// Apply limit
	if len(allEntries) > limit {
		allEntries = allEntries[:limit]
	}

	return allEntries, nil
}

// sortLogEntriesByTimeDesc sorts log entries by timestamp in descending order.
func sortLogEntriesByTimeDesc(entries []LogEntry) {
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].Timestamp.After(entries[i].Timestamp) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
}

// sortLogEntriesByTimeAsc sorts log entries by timestamp in ascending order.
func sortLogEntriesByTimeAsc(entries []LogEntry) {
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].Timestamp.Before(entries[i].Timestamp) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
}

// AzureLogsError represents an error with actionable guidance.
type AzureLogsError struct {
	Code    string // Error code: AUTH_REQUIRED, NO_WORKSPACE, etc.
	Message string // Human-readable message
	Action  string // What the user should do
	Command string // CLI command to run (optional)
}

func (e *AzureLogsError) Error() string {
	if e.Command != "" {
		return fmt.Sprintf("%s\n\n%s\n  %s", e.Message, e.Action, e.Command)
	}
	if e.Action != "" {
		return fmt.Sprintf("%s\n\n%s", e.Message, e.Action)
	}
	return e.Message
}

// GetWorkspaceIDFromEnv attempts to get the workspace GUID from azd environment.
// Uses the azd extension framework's Environment service.
func GetWorkspaceIDFromEnv(ctx context.Context) (string, error) {
	azdClient, err := azdext.NewAzdClient()
	if err != nil {
		return "", fmt.Errorf("failed to create azd client: %w", err)
	}
	defer azdClient.Close()

	ctx = azdext.WithAccessToken(ctx)

	// Get current environment name
	resp, err := azdClient.Environment().GetCurrent(ctx, &azdext.EmptyRequest{})
	if err != nil {
		return "", fmt.Errorf("failed to get current environment: %w", err)
	}
	if resp.Environment == nil || resp.Environment.Name == "" {
		return "", fmt.Errorf("no current environment set")
	}
	envName := resp.Environment.Name

	// Try AZURE_LOG_ANALYTICS_WORKSPACE_GUID first (set by azd provision)
	guidResp, err := azdClient.Environment().GetValue(ctx, &azdext.GetEnvRequest{
		EnvName: envName,
		Key:     "AZURE_LOG_ANALYTICS_WORKSPACE_GUID",
	})
	if err == nil && guidResp.Value != "" {
		return guidResp.Value, nil
	}

	// Next: if we have a workspace resource ID, resolve its customerId via ARM
	wsIDResp, err := azdClient.Environment().GetValue(ctx, &azdext.GetEnvRequest{
		EnvName: envName,
		Key:     "AZURE_LOG_ANALYTICS_WORKSPACE_ID",
	})
	if err == nil && wsIDResp.Value != "" {
		if cred, err := NewAzureCredential(); err == nil {
			if guid, err := ResolveWorkspaceCustomerID(ctx, cred, wsIDResp.Value); err == nil {
				// Store it for future use
				_, _ = azdClient.Environment().SetValue(ctx, &azdext.SetEnvRequest{
					EnvName: envName,
					Key:     "AZURE_LOG_ANALYTICS_WORKSPACE_GUID",
					Value:   guid,
				})
				return guid, nil
			}
		}
	}

	return "", fmt.Errorf("workspace GUID not found in environment")
}

// DiscoverAndStoreWorkspaceID attempts to find Log Analytics workspace ID and store it.
// Returns (workspaceGUID, wasDiscovered, error)
func DiscoverAndStoreWorkspaceID(ctx context.Context) (string, bool, error) {
	azdClient, err := azdext.NewAzdClient()
	if err != nil {
		return "", false, fmt.Errorf("failed to create azd client: %w", err)
	}
	defer azdClient.Close()

	ctx = azdext.WithAccessToken(ctx)

	// Get current environment name
	resp, err := azdClient.Environment().GetCurrent(ctx, &azdext.EmptyRequest{})
	if err != nil {
		return "", false, fmt.Errorf("failed to get current environment: %w", err)
	}
	if resp.Environment == nil || resp.Environment.Name == "" {
		return "", false, fmt.Errorf("no current environment set")
	}
	envName := resp.Environment.Name

	// Check if already set
	guidResp, err := azdClient.Environment().GetValue(ctx, &azdext.GetEnvRequest{
		EnvName: envName,
		Key:     "AZURE_LOG_ANALYTICS_WORKSPACE_GUID",
	})
	if err == nil && guidResp.Value != "" {
		return guidResp.Value, false, nil
	}

	// Get resource group from environment
	rgResp, err := azdClient.Environment().GetValue(ctx, &azdext.GetEnvRequest{
		EnvName: envName,
		Key:     "AZURE_RESOURCE_GROUP",
	})
	resourceGroup := ""
	if err == nil {
		resourceGroup = rgResp.Value
	}
	if resourceGroup == "" {
		rgResp, err = azdClient.Environment().GetValue(ctx, &azdext.GetEnvRequest{
			EnvName: envName,
			Key:     "AZURE_RESOURCE_GROUP_NAME",
		})
		if err == nil {
			resourceGroup = rgResp.Value
		}
	}
	if resourceGroup == "" {
		return "", false, fmt.Errorf("AZURE_RESOURCE_GROUP_NAME not set")
	}

	// Try to discover workspace using az CLI
	if os.Getenv("AZD_APP_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Attempting to discover workspace in resource group: %s\n", resourceGroup)
	}

	workspaceID, err := discoverWorkspaceViaAzCLI(ctx, resourceGroup)
	if err != nil {
		return "", false, err
	}

	if workspaceID == "" {
		return "", false, fmt.Errorf("no workspace found in resource group %s", resourceGroup)
	}

	// Store workspace GUID in environment using azd Environment service
	_, err = azdClient.Environment().SetValue(ctx, &azdext.SetEnvRequest{
		EnvName: envName,
		Key:     "AZURE_LOG_ANALYTICS_WORKSPACE_GUID",
		Value:   workspaceID,
	})
	if err != nil {
		return "", false, fmt.Errorf("failed to store workspace ID: %w", err)
	}

	if os.Getenv("AZD_APP_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Discovered and stored workspace ID: %s\n", workspaceID)
	}

	return workspaceID, true, nil
}

// discoverWorkspaceViaAzCLI calls az CLI to discover Log Analytics workspace.
func discoverWorkspaceViaAzCLI(ctx context.Context, resourceGroup string) (string, error) {
	// Build command: az monitor log-analytics workspace list --resource-group <rg> --query "[0].customerId" -o tsv
	args := []string{
		"monitor", "log-analytics", "workspace", "list",
		"--resource-group", resourceGroup,
		"--query", "[0].customerId",
		"-o", "tsv",
	}

	// Execute with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, "az", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if az CLI is not available
		if strings.Contains(err.Error(), "executable file not found") {
			return "", fmt.Errorf("az CLI not found in PATH")
		}
		return "", fmt.Errorf("az CLI failed: %w (output: %s)", err, string(output))
	}

	workspaceID := strings.TrimSpace(string(output))
	if workspaceID == "" {
		return "", fmt.Errorf("no workspace found in output")
	}

	return workspaceID, nil
}

// buildStandaloneQueryForType builds a KQL query for a specific resource type.
func buildStandaloneQueryForType(resourceType ResourceType, services []string, since time.Duration, limit int) string {
	var sb strings.Builder

	switch resourceType {
	case ResourceTypeContainerApp:
		sb.WriteString("ContainerAppConsoleLogs_CL\n")
		sb.WriteString(fmt.Sprintf(kqlWhereTimeAgo, formatKQLDuration(since)))
		if len(services) > 0 {
			var conditions []string
			for _, svc := range services {
				conditions = append(conditions, fmt.Sprintf("ContainerAppName_s =~ '%s'", sanitizeKQLString(svc)))
				conditions = append(conditions, fmt.Sprintf("ContainerName_s =~ '%s'", sanitizeKQLString(svc)))
			}
			sb.WriteString(fmt.Sprintf(kqlWhere, strings.Join(conditions, " or ")))
		}
		sb.WriteString("| extend Source = \"Azure Container Apps\", AzureService = \"containerapp\"\n")
		sb.WriteString("| project TimeGenerated, Source, Log_s, Stream_s, ContainerAppName_s, ContainerName_s, RevisionName_s\n")

	case ResourceTypeAppService:
		sb.WriteString("AppServiceConsoleLogs\n")
		sb.WriteString(fmt.Sprintf(kqlWhereTimeAgo, formatKQLDuration(since)))
		if len(services) > 0 {
			var conditions []string
			for _, svc := range services {
				// For App Service, filter by _ResourceId which contains the app name
				conditions = append(conditions, fmt.Sprintf(kqlResourceIDFilter, sanitizeKQLString(svc)))
			}
			sb.WriteString(fmt.Sprintf(kqlWhere, strings.Join(conditions, " or ")))
		}
		sb.WriteString("| extend Source = \"Azure App Service\", AzureService = \"appservice\"\n")
		sb.WriteString("| project TimeGenerated, Source, Message=ResultDescription, Level, _ResourceId\n")

	case ResourceTypeFunction:
		sb.WriteString("FunctionAppLogs\n")
		sb.WriteString(fmt.Sprintf(kqlWhereTimeAgo, formatKQLDuration(since)))
		if len(services) > 0 {
			var conditions []string
			for _, svc := range services {
				// For Functions, filter by _ResourceId which contains the function app name
				conditions = append(conditions, fmt.Sprintf(kqlResourceIDFilter, sanitizeKQLString(svc)))
			}
			sb.WriteString(fmt.Sprintf(kqlWhere, strings.Join(conditions, " or ")))
		}
		sb.WriteString("| extend Source = \"Azure Functions\", AzureService = \"functions\"\n")
		sb.WriteString("| project TimeGenerated, Source, Message, Level, FunctionName, _ResourceId\n")

	case ResourceTypeAKS:
		sb.WriteString("ContainerLogV2\n")
		sb.WriteString(fmt.Sprintf(kqlWhereTimeAgo, formatKQLDuration(since)))
		if len(services) > 0 {
			var conditions []string
			for _, svc := range services {
				conditions = append(conditions, fmt.Sprintf("PodName contains '%s'", sanitizeKQLString(svc)))
				conditions = append(conditions, fmt.Sprintf("ContainerName contains '%s'", sanitizeKQLString(svc)))
			}
			sb.WriteString(fmt.Sprintf(kqlWhere, strings.Join(conditions, " or ")))
		}
		sb.WriteString("| project TimeGenerated, LogMessage, PodName, ContainerName, PodNamespace\n")

	case ResourceTypeContainerInstance:
		sb.WriteString("ContainerInstanceLog_CL\n")
		sb.WriteString(fmt.Sprintf(kqlWhereTimeAgo, formatKQLDuration(since)))
		if len(services) > 0 {
			var conditions []string
			for _, svc := range services {
				conditions = append(conditions, fmt.Sprintf("ContainerGroup_s contains '%s'", sanitizeKQLString(svc)))
			}
			sb.WriteString(fmt.Sprintf(kqlWhere, strings.Join(conditions, " or ")))
		}
		sb.WriteString("| project TimeGenerated, Message_s, ContainerName_s\n")

	default:
		// Fallback to Container App
		return buildStandaloneQueryForType(ResourceTypeContainerApp, services, since, limit)
	}

	sb.WriteString("| order by TimeGenerated desc\n")
	sb.WriteString(fmt.Sprintf("| take %d", limit))

	return sb.String()
}

// buildTimestampQuery builds a KQL query using precise timestamp filtering instead of ago().
// This is more efficient for polling as it avoids re-fetching overlapping data.
func buildTimestampQuery(resourceType ResourceType, services []string, since time.Time) string {
	var sb strings.Builder
	timestamp := since.UTC().Format(time.RFC3339Nano)

	switch resourceType {
	case ResourceTypeContainerApp:
		sb.WriteString("ContainerAppConsoleLogs_CL\n")
		sb.WriteString(fmt.Sprintf(kqlWhereTimeDateTime, timestamp))
		if len(services) > 0 {
			var conditions []string
			for _, svc := range services {
				conditions = append(conditions, fmt.Sprintf("ContainerAppName_s =~ '%s'", sanitizeKQLString(svc)))
				conditions = append(conditions, fmt.Sprintf("ContainerName_s =~ '%s'", sanitizeKQLString(svc)))
			}
			sb.WriteString(fmt.Sprintf(kqlWhere, strings.Join(conditions, " or ")))
		}
		sb.WriteString("| extend Source = \"Azure Container Apps\", AzureService = \"containerapp\"\n")
		sb.WriteString("| project TimeGenerated, Source, Log_s, Stream_s, ContainerAppName_s, ContainerName_s, RevisionName_s\n")

	case ResourceTypeAppService:
		sb.WriteString("AppServiceConsoleLogs\n")
		sb.WriteString(fmt.Sprintf(kqlWhereTimeDateTime, timestamp))
		if len(services) > 0 {
			var conditions []string
			for _, svc := range services {
				conditions = append(conditions, fmt.Sprintf(kqlResourceIDFilter, sanitizeKQLString(svc)))
			}
			sb.WriteString(fmt.Sprintf(kqlWhere, strings.Join(conditions, " or ")))
		}
		sb.WriteString("| extend Source = \"Azure App Service\", AzureService = \"appservice\"\n")
		sb.WriteString("| project TimeGenerated, Source, Message=ResultDescription, Level, _ResourceId\n")

	case ResourceTypeFunction:
		sb.WriteString("FunctionAppLogs\n")
		sb.WriteString(fmt.Sprintf(kqlWhereTimeDateTime, timestamp))
		if len(services) > 0 {
			var conditions []string
			for _, svc := range services {
				conditions = append(conditions, fmt.Sprintf(kqlResourceIDFilter, sanitizeKQLString(svc)))
			}
			sb.WriteString(fmt.Sprintf(kqlWhere, strings.Join(conditions, " or ")))
		}
		sb.WriteString("| extend Source = \"Azure Functions\", AzureService = \"functions\"\n")
		sb.WriteString("| project TimeGenerated, Source, Message, Level, FunctionName, _ResourceId\n")

	case ResourceTypeAKS:
		sb.WriteString("ContainerLogV2\n")
		sb.WriteString(fmt.Sprintf(kqlWhereTimeDateTime, timestamp))
		if len(services) > 0 {
			var conditions []string
			for _, svc := range services {
				conditions = append(conditions, fmt.Sprintf("PodName contains '%s'", sanitizeKQLString(svc)))
				conditions = append(conditions, fmt.Sprintf("ContainerName contains '%s'", sanitizeKQLString(svc)))
			}
			sb.WriteString(fmt.Sprintf(kqlWhere, strings.Join(conditions, " or ")))
		}
		sb.WriteString("| project TimeGenerated, LogMessage, PodName, ContainerName, PodNamespace\n")

	case ResourceTypeContainerInstance:
		sb.WriteString("ContainerInstanceLog_CL\n")
		sb.WriteString(fmt.Sprintf(kqlWhereTimeDateTime, timestamp))
		if len(services) > 0 {
			var conditions []string
			for _, svc := range services {
				conditions = append(conditions, fmt.Sprintf("ContainerGroup_s contains '%s'", sanitizeKQLString(svc)))
			}
			sb.WriteString(fmt.Sprintf(kqlWhere, strings.Join(conditions, " or ")))
		}
		sb.WriteString("| project TimeGenerated, Message_s, ContainerName_s\n")

	default:
		// Fallback to Container App
		return buildTimestampQuery(ResourceTypeContainerApp, services, since)
	}

	sb.WriteString("| order by TimeGenerated asc\n")
	sb.WriteString("| take 500")

	return sb.String()
}

// formatKQLDuration formats a duration for KQL ago() function.
func formatKQLDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours >= 24 {
		days := hours / 24
		return fmt.Sprintf("%dd", days)
	}
	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%dh%dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	seconds := int(d.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	return fmt.Sprintf("%ds", seconds)
}

// StreamConfig holds configuration for standalone Azure log streaming.
type StreamConfig struct {
	ProjectDir    string
	WorkspaceID   string        // Log Analytics workspace GUID
	Services      []string      // Service names to filter (empty = all)
	PollInterval  time.Duration // How often to poll (default 30s)
	InitialWindow time.Duration // How far back to look on first poll
}

// StreamAzureLogsStandalone streams Azure logs by polling Log Analytics.
// Logs are sent to the provided channel. The function blocks until ctx is cancelled.
// This enables `azd app logs -f --source azure` without requiring `azd app run`.
func StreamAzureLogsStandalone(ctx context.Context, config StreamConfig, logs chan<- LogEntry) error {
	// Get workspace ID from environment if not provided
	workspaceID := config.WorkspaceID
	if workspaceID == "" {
		if wsID, err := GetWorkspaceIDFromEnv(ctx); err == nil {
			workspaceID = wsID
		}

		if workspaceID == "" {
			// Try auto-discovery
			if discovered, wasDiscovered, err := DiscoverAndStoreWorkspaceID(ctx); err == nil && wasDiscovered {
				workspaceID = discovered
				if os.Getenv("AZD_APP_DEBUG") == "true" {
					fmt.Fprintf(os.Stderr, "[DEBUG] Auto-discovered workspace ID: %s\n", workspaceID)
				}
			} else if err != nil && os.Getenv("AZD_APP_DEBUG") == "true" {
				fmt.Fprintf(os.Stderr, "[DEBUG] Workspace discovery failed: %v\n", err)
			}
		}

		if workspaceID == "" {
			return &AzureLogsError{
				Code:    "NO_WORKSPACE",
				Message: "Log Analytics workspace not configured",
				Action:  "Deploy with 'azd up' or set AZURE_LOG_ANALYTICS_WORKSPACE_GUID",
			}
		}
	}

	// Get credential
	cred, err := NewLogAnalyticsCredential()
	if err != nil {
		// Clear token cache on credential errors
		ClearTokenCacheOnError(err)
		return &AzureLogsError{
			Code:    "AUTH_REQUIRED",
			Message: "Azure authentication required",
			Action:  "Run 'azd auth login' to authenticate",
			Command: "azd auth login",
		}
	}

	// Get or create cached client
	client, err := GetOrCreateLogAnalyticsClient(ctx, cred, workspaceID)
	if err != nil {
		// Clear token cache on client creation errors
		ClearTokenCacheOnError(err)
		return &AzureLogsError{
			Code:    "CLIENT_ERROR",
			Message: fmt.Sprintf("Failed to create Log Analytics client: %v", err),
			Action:  "Check your Azure configuration",
		}
	}

	// Set defaults
	pollInterval := config.PollInterval
	if pollInterval == 0 {
		pollInterval = 30 * time.Second
	}

	// Get all services from azure.yaml with their host types
	allServices, err := getServicesFromAzureYAML(config.ProjectDir)
	if err != nil {
		// Fall back to Container App only if we can't read azure.yaml
		allServices = []ServiceInfo{{ResourceType: ResourceTypeContainerApp}}
	}

	// Filter services if specific ones requested
	var targetServices []ServiceInfo
	if len(config.Services) > 0 {
		serviceMap := make(map[string]bool)
		for _, s := range config.Services {
			serviceMap[strings.ToLower(s)] = true
		}
		for _, svc := range allServices {
			if serviceMap[strings.ToLower(svc.Name)] {
				targetServices = append(targetServices, svc)
			}
		}
	} else {
		targetServices = allServices
	}

	// Group services by resource type
	servicesByType := make(map[ResourceType][]ServiceInfo)
	for _, svc := range targetServices {
		servicesByType[svc.ResourceType] = append(servicesByType[svc.ResourceType], svc)
	}

	// Fix 4: Align streaming window with user-requested time range
	// Use InitialWindow if provided, otherwise default to 1 hour (not 24h)
	window := config.InitialWindow
	if window <= 0 {
		window = 1 * time.Hour
	}
	if os.Getenv("AZD_APP_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Streaming initial window: %v\n", window)
	}
	lastSeen := time.Now().Add(-window)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Do initial fetch immediately
	if err := fetchAndSendLogsMultiType(ctx, client, servicesByType, lastSeen, logs, &lastSeen); err != nil {
		// Log error but continue - transient failures shouldn't stop streaming
		if os.Getenv("AZD_APP_DEBUG") == "true" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Initial fetch failed: %v\n", err)
		}
		// Don't return - continue to poll loop
	}

	// Poll loop
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := fetchAndSendLogsMultiType(ctx, client, servicesByType, lastSeen, logs, &lastSeen); err != nil {
				// For streaming, we don't return on transient errors
				// Just skip this poll cycle
				if os.Getenv("AZD_APP_DEBUG") == "true" {
					fmt.Fprintf(os.Stderr, "[DEBUG] Poll fetch failed: %v\n", err)
				}
				continue
			}
		}
	}
}

// fetchAndSendLogsMultiType fetches logs from multiple resource types and sends them to the channel.
func fetchAndSendLogsMultiType(ctx context.Context, client *LogAnalyticsClient, servicesByType map[ResourceType][]ServiceInfo, since time.Time, logs chan<- LogEntry, lastSeen *time.Time) error {
	// Use precise timestamp filtering instead of ago() to avoid duplicate fetches
	// This queries: TimeGenerated > lastSeen instead of TimeGenerated > ago(Nm)

	// Collect all entries from all resource types
	var allEntries []LogEntry
	var successCount int
	var errorCount int
	for resourceType, services := range servicesByType {
		var azureNames []string
		for _, svc := range services {
			azureNames = append(azureNames, svc.AzureName)
		}

		// Build query with timestamp-based filtering
		query := buildTimestampQuery(resourceType, azureNames, *lastSeen)

		if os.Getenv("AZD_APP_DEBUG") == "true" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Streaming query for %s: %s\n", resourceType, strings.ReplaceAll(query, "\n", " | "))
		}

		// Query using custom query (bypasses ago() duration logic)
		entries, err := client.QueryLogs(ctx, "", resourceType, 0, query)
		if err != nil {
			if os.Getenv("AZD_APP_DEBUG") == "true" {
				fmt.Fprintf(os.Stderr, "[DEBUG] Query failed for %s: %v\n", resourceType, err)
			}
			errorCount++
			continue
		}

		successCount++
		entries = mapServiceNames(entries, services)

		allEntries = append(allEntries, entries...)
	}

	if successCount == 0 && errorCount > 0 {
		return fmt.Errorf("azure log queries failed for all resource types")
	}

	// Sort by timestamp ascending for streaming (oldest first)
	sortLogEntriesByTimeAsc(allEntries)

	if os.Getenv("AZD_APP_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Fetched %d total entries, lastSeen=%v\n", len(allEntries), lastSeen.Format(time.RFC3339))
	}

	// Send new entries (in chronological order - oldest to newest)
	// Deduplication: only send entries with timestamp > lastSeen
	sentCount := 0
	for _, entry := range allEntries {
		if entry.Timestamp.After(*lastSeen) {
			select {
			case logs <- entry:
				sentCount++
				*lastSeen = entry.Timestamp
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	if os.Getenv("AZD_APP_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Sent %d entries to channel\n", sentCount)
	}

	// Fix 3: Last poll timestamp only shown in debug mode (was always shown, spamming stderr)
	if os.Getenv("AZD_APP_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "[DEBUG] [%s] Last polled (sent %d entries)\n", time.Now().Format("15:04:05"), sentCount)
	}

	return nil
}

// mapServiceNames remaps Azure resource names to logical service names from azure.yaml.
func mapServiceNames(entries []LogEntry, services []ServiceInfo) []LogEntry {
	if len(entries) == 0 || len(services) == 0 {
		return entries
	}

	mapping := make(map[string]string, len(services)*2)
	for _, svc := range services {
		logical := strings.ToLower(svc.Name)
		if svc.AzureName != "" {
			mapping[strings.ToLower(svc.AzureName)] = logical
		}
		mapping[logical] = logical
	}

	for i := range entries {
		entry := &entries[i]
		candidates := []string{entry.Service, entry.ContainerName, entry.InstanceID}
		for _, candidate := range candidates {
			if candidate == "" {
				continue
			}
			if logical, ok := mapping[strings.ToLower(candidate)]; ok {
				entry.Service = logical
				break
			}
		}
	}

	return entries
}
