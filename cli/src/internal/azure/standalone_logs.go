// Package azure provides Azure cloud integration for log streaming and resource discovery.
package azure

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
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
		workspaceID = getWorkspaceIDFromEnv(config.ProjectDir)
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
		return nil, &AzureLogsError{
			Code:    "AUTH_REQUIRED",
			Message: "Azure authentication required",
			Action:  "Run 'azd auth login' to authenticate",
			Command: "azd auth login",
		}
	}

	// Create client
	client, err := NewLogAnalyticsClient(cred, workspaceID)
	if err != nil {
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

	// Debug: log service grouping
	if os.Getenv("AZD_APP_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Target services: %v\n", targetServices)
		fmt.Fprintf(os.Stderr, "[DEBUG] Services by type: %v\n", servicesByType)
	}

	// Query each resource type and collect all entries
	var allEntries []LogEntry
	for resourceType, services := range servicesByType {
		// Get Azure names for filtering
		var azureNames []string
		for _, svc := range services {
			azureNames = append(azureNames, svc.AzureName)
		}

		query := buildStandaloneQueryForType(resourceType, azureNames, since, limit)

		if os.Getenv("AZD_APP_DEBUG") == "true" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Query for %s: %s\n", resourceType, strings.ReplaceAll(query, "\n", " | "))
		}

		entries, err := client.QueryLogs(ctx, "", resourceType, since, query)
		if err != nil {
			// Log error but continue with other resource types
			if os.Getenv("AZD_APP_DEBUG") == "true" {
				fmt.Fprintf(os.Stderr, "[DEBUG] Query failed for %s: %v\n", resourceType, err)
			}
			continue
		}

		allEntries = append(allEntries, entries...)
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

// getWorkspaceIDFromEnv attempts to get the workspace GUID from azd environment.
// Uses environment variables directly since the azd extension framework provides them.
func getWorkspaceIDFromEnv(projectDir string) string {
	// When running as an azd extension, environment variables are already available.
	// Try AZURE_LOG_ANALYTICS_WORKSPACE_GUID first (set by azd provision)
	if guid := os.Getenv("AZURE_LOG_ANALYTICS_WORKSPACE_GUID"); guid != "" {
		return guid
	}

	// Try the .env file directly
	envFile := filepath.Join(projectDir, ".azure", getDefaultEnvName(projectDir), ".env")
	if content, err := os.ReadFile(envFile); err == nil {
		for _, line := range strings.Split(string(content), "\n") {
			if strings.HasPrefix(line, "AZURE_LOG_ANALYTICS_WORKSPACE_GUID=") {
				return strings.TrimPrefix(line, "AZURE_LOG_ANALYTICS_WORKSPACE_GUID=")
			}
		}
	}

	return ""
}

// getDefaultEnvName gets the default environment name for azd.
func getDefaultEnvName(projectDir string) string {
	// Try to read from .azure/.env
	defaultEnvFile := filepath.Join(projectDir, ".azure", ".env")
	if content, err := os.ReadFile(defaultEnvFile); err == nil {
		for _, line := range strings.Split(string(content), "\n") {
			if strings.HasPrefix(line, "AZURE_ENV_NAME=") {
				return strings.TrimPrefix(line, "AZURE_ENV_NAME=")
			}
		}
	}
	return ""
}

// resolveServiceNames maps azure.yaml service names to Azure resource names.
// For example, "containerapp-api" -> "ca-k7zjfgph5a6jk" using SERVICE_*_NAME env vars.
func resolveServiceNames(projectDir string, services []string) []string {
	if len(services) == 0 {
		return services
	}

	// Build lookup map from environment
	serviceNameMap := make(map[string]string)

	// Get env values from environment (provided by azd extension framework)
	for _, line := range os.Environ() {
		// Look for SERVICE_*_NAME entries
		// Format: SERVICE_CONTAINERAPP_API_NAME="ca-k7zjfgph5a6jk"
		if strings.HasPrefix(line, "SERVICE_") && strings.Contains(line, "_NAME=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				// Extract service name from key: SERVICE_CONTAINERAPP_API_NAME -> containerapp-api
				key := parts[0]
				key = strings.TrimPrefix(key, "SERVICE_")
				key = strings.TrimSuffix(key, "_NAME")
				key = strings.ToLower(strings.ReplaceAll(key, "_", "-"))

				// Get the Azure resource name
				value := strings.Trim(parts[1], "\"")
				if value != "" {
					serviceNameMap[key] = value
				}
			}
		}
	}

	// Resolve service names
	resolved := make([]string, 0, len(services))
	for _, svc := range services {
		svcLower := strings.ToLower(svc)
		if azureName, ok := serviceNameMap[svcLower]; ok {
			resolved = append(resolved, azureName)
		} else {
			// Keep original if no mapping found
			resolved = append(resolved, svc)
		}
	}

	return resolved
}

// buildStandaloneQuery builds a KQL query for standalone log fetching (Container Apps only - legacy).
func buildStandaloneQuery(services []string, since time.Duration, limit int) string {
	return buildStandaloneQueryForType(ResourceTypeContainerApp, services, since, limit)
}

// buildStandaloneQueryForType builds a KQL query for a specific resource type.
func buildStandaloneQueryForType(resourceType ResourceType, services []string, since time.Duration, limit int) string {
	var sb strings.Builder

	switch resourceType {
	case ResourceTypeContainerApp:
		sb.WriteString("ContainerAppConsoleLogs_CL\n")
		sb.WriteString(fmt.Sprintf("| where TimeGenerated > ago(%s)\n", formatKQLDuration(since)))
		if len(services) > 0 {
			var conditions []string
			for _, svc := range services {
				conditions = append(conditions, fmt.Sprintf("ContainerAppName_s =~ '%s'", sanitizeKQLString(svc)))
				conditions = append(conditions, fmt.Sprintf("ContainerName_s =~ '%s'", sanitizeKQLString(svc)))
			}
			sb.WriteString(fmt.Sprintf("| where %s\n", strings.Join(conditions, " or ")))
		}
		sb.WriteString("| project TimeGenerated, Log_s, Stream_s, ContainerAppName_s, ContainerName_s, RevisionName_s\n")

	case ResourceTypeAppService:
		sb.WriteString("AppServiceConsoleLogs\n")
		sb.WriteString(fmt.Sprintf("| where TimeGenerated > ago(%s)\n", formatKQLDuration(since)))
		if len(services) > 0 {
			var conditions []string
			for _, svc := range services {
				// For App Service, filter by _ResourceId which contains the app name
				conditions = append(conditions, fmt.Sprintf("_ResourceId contains '%s'", sanitizeKQLString(svc)))
			}
			sb.WriteString(fmt.Sprintf("| where %s\n", strings.Join(conditions, " or ")))
		}
		sb.WriteString("| project TimeGenerated, Message=ResultDescription, Level, _ResourceId\n")

	case ResourceTypeFunction:
		sb.WriteString("FunctionAppLogs\n")
		sb.WriteString(fmt.Sprintf("| where TimeGenerated > ago(%s)\n", formatKQLDuration(since)))
		if len(services) > 0 {
			var conditions []string
			for _, svc := range services {
				// For Functions, filter by _ResourceId which contains the function app name
				conditions = append(conditions, fmt.Sprintf("_ResourceId contains '%s'", sanitizeKQLString(svc)))
			}
			sb.WriteString(fmt.Sprintf("| where %s\n", strings.Join(conditions, " or ")))
		}
		sb.WriteString("| project TimeGenerated, Message, Level, FunctionName, _ResourceId\n")

	case ResourceTypeAKS:
		sb.WriteString("ContainerLogV2\n")
		sb.WriteString(fmt.Sprintf("| where TimeGenerated > ago(%s)\n", formatKQLDuration(since)))
		if len(services) > 0 {
			var conditions []string
			for _, svc := range services {
				conditions = append(conditions, fmt.Sprintf("PodName contains '%s'", sanitizeKQLString(svc)))
				conditions = append(conditions, fmt.Sprintf("ContainerName contains '%s'", sanitizeKQLString(svc)))
			}
			sb.WriteString(fmt.Sprintf("| where %s\n", strings.Join(conditions, " or ")))
		}
		sb.WriteString("| project TimeGenerated, LogMessage, PodName, ContainerName, PodNamespace\n")

	case ResourceTypeContainerInstance:
		sb.WriteString("ContainerInstanceLog_CL\n")
		sb.WriteString(fmt.Sprintf("| where TimeGenerated > ago(%s)\n", formatKQLDuration(since)))
		if len(services) > 0 {
			var conditions []string
			for _, svc := range services {
				conditions = append(conditions, fmt.Sprintf("ContainerGroup_s contains '%s'", sanitizeKQLString(svc)))
			}
			sb.WriteString(fmt.Sprintf("| where %s\n", strings.Join(conditions, " or ")))
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

// mapQueryError maps Azure SDK errors to actionable AzureLogsError.
func mapQueryError(err error) *AzureLogsError {
	errStr := err.Error()

	switch {
	case strings.Contains(errStr, "AADSTS") || strings.Contains(errStr, "401"):
		return &AzureLogsError{
			Code:    "AUTH_EXPIRED",
			Message: "Authentication expired",
			Action:  "Run this command to fix:",
			Command: "azd auth login",
		}

	case strings.Contains(errStr, "ResourceNotFound") || strings.Contains(errStr, "404"):
		return &AzureLogsError{
			Code:    "NOT_DEPLOYED",
			Message: "Azure resources not found",
			Action:  "Deploy your app first:",
			Command: "azd up",
		}

	case strings.Contains(errStr, "AuthorizationFailed") || strings.Contains(errStr, "403"):
		return &AzureLogsError{
			Code:    "NO_PERMISSION",
			Message: "Missing permissions on Log Analytics workspace",
			Action:  "Grant 'Log Analytics Reader' role in Azure Portal",
		}

	case strings.Contains(errStr, "WorkspaceNotFound"):
		return &AzureLogsError{
			Code:    "NO_WORKSPACE",
			Message: "Log Analytics workspace not found",
			Action:  "Check AZURE_LOG_ANALYTICS_WORKSPACE_GUID is correct",
		}

	default:
		return &AzureLogsError{
			Code:    "UNKNOWN",
			Message: errStr,
			Action:  "Check Azure Portal for details",
		}
	}
}

// StreamConfig holds configuration for standalone Azure log streaming.
type StreamConfig struct {
	ProjectDir   string
	WorkspaceID  string        // Log Analytics workspace GUID
	Services     []string      // Service names to filter (empty = all)
	PollInterval time.Duration // How often to poll (default 30s)
}

// StreamAzureLogsStandalone streams Azure logs by polling Log Analytics.
// Logs are sent to the provided channel. The function blocks until ctx is cancelled.
// This enables `azd app logs -f --source azure` without requiring `azd app run`.
func StreamAzureLogsStandalone(ctx context.Context, config StreamConfig, logs chan<- LogEntry) error {
	// Get workspace ID from environment if not provided
	workspaceID := config.WorkspaceID
	if workspaceID == "" {
		workspaceID = getWorkspaceIDFromEnv(config.ProjectDir)
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
		return &AzureLogsError{
			Code:    "AUTH_REQUIRED",
			Message: "Azure authentication required",
			Action:  "Run 'azd auth login' to authenticate",
			Command: "azd auth login",
		}
	}

	// Create client
	client, err := NewLogAnalyticsClient(cred, workspaceID)
	if err != nil {
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

	// Track last seen timestamp to avoid duplicates
	// Use 24h initial window to catch recent logs, even if container has been idle
	lastSeen := time.Now().Add(-24 * time.Hour) // Start with last 24 hours

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
	sinceAgo := time.Since(since)
	if sinceAgo < time.Minute {
		sinceAgo = time.Minute // Minimum 1 minute window
	}

	// Collect all entries from all resource types
	var allEntries []LogEntry
	for resourceType, services := range servicesByType {
		var azureNames []string
		for _, svc := range services {
			azureNames = append(azureNames, svc.AzureName)
		}

		query := buildStreamingQueryForType(resourceType, azureNames, sinceAgo)

		if os.Getenv("AZD_APP_DEBUG") == "true" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Streaming query for %s: %s\n", resourceType, strings.ReplaceAll(query, "\n", " | "))
		}

		entries, err := client.QueryLogs(ctx, "", resourceType, sinceAgo, query)
		if err != nil {
			if os.Getenv("AZD_APP_DEBUG") == "true" {
				fmt.Fprintf(os.Stderr, "[DEBUG] Query failed for %s: %v\n", resourceType, err)
			}
			continue
		}

		allEntries = append(allEntries, entries...)
	}

	// Sort by timestamp ascending for streaming (oldest first)
	sortLogEntriesByTimeAsc(allEntries)

	if os.Getenv("AZD_APP_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Fetched %d total entries, lastSeen=%v, sinceAgo=%v\n", len(allEntries), lastSeen.Format(time.RFC3339), sinceAgo)
	}

	// Send new entries (in chronological order - oldest to newest)
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

	// Show last poll timestamp
	fmt.Fprintf(os.Stderr, "[%s] Last polled\n", time.Now().Format("15:04:05"))

	return nil
}

// fetchAndSendLogs fetches logs newer than lastSeen and sends them to the channel (legacy - Container Apps only).
func fetchAndSendLogs(ctx context.Context, client *LogAnalyticsClient, services []string, since time.Time, logs chan<- LogEntry, lastSeen *time.Time) error {
	// Query for logs since last seen time
	sinceAgo := time.Since(since)
	if sinceAgo < time.Minute {
		sinceAgo = time.Minute // Minimum 1 minute window
	}

	query := buildStreamingQuery(services, sinceAgo)

	// Debug: log query
	if os.Getenv("AZD_APP_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Query: %s\n", strings.ReplaceAll(query, "\n", " | "))
	}

	entries, err := client.QueryLogs(ctx, "", ResourceTypeContainerApp, sinceAgo, query)
	if err != nil {
		return mapQueryError(err)
	}

	// Debug: log fetch results
	if os.Getenv("AZD_APP_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Fetched %d entries, lastSeen=%v, sinceAgo=%v\n", len(entries), lastSeen.Format(time.RFC3339), sinceAgo)
	}

	// Send new entries (in chronological order - oldest to newest)
	sentCount := 0
	for _, entry := range entries {
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

	// Debug: log sent count
	if os.Getenv("AZD_APP_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Sent %d entries to channel\n", sentCount)
	}

	return nil
}

// buildStreamingQuery builds a KQL query optimized for streaming (ordered by time asc) - legacy Container Apps only.
func buildStreamingQuery(services []string, since time.Duration) string {
	return buildStreamingQueryForType(ResourceTypeContainerApp, services, since)
}

// buildStreamingQueryForType builds a KQL query optimized for streaming for a specific resource type.
func buildStreamingQueryForType(resourceType ResourceType, services []string, since time.Duration) string {
	var sb strings.Builder

	switch resourceType {
	case ResourceTypeContainerApp:
		sb.WriteString("ContainerAppConsoleLogs_CL\n")
		sb.WriteString(fmt.Sprintf("| where TimeGenerated > ago(%s)\n", formatKQLDuration(since)))
		if len(services) > 0 {
			var conditions []string
			for _, svc := range services {
				conditions = append(conditions, fmt.Sprintf("ContainerAppName_s =~ '%s'", sanitizeKQLString(svc)))
				conditions = append(conditions, fmt.Sprintf("ContainerName_s =~ '%s'", sanitizeKQLString(svc)))
			}
			sb.WriteString(fmt.Sprintf("| where %s\n", strings.Join(conditions, " or ")))
		}
		sb.WriteString("| project TimeGenerated, Log_s, Stream_s, ContainerAppName_s, ContainerName_s, RevisionName_s\n")

	case ResourceTypeAppService:
		sb.WriteString("AppServiceConsoleLogs\n")
		sb.WriteString(fmt.Sprintf("| where TimeGenerated > ago(%s)\n", formatKQLDuration(since)))
		if len(services) > 0 {
			var conditions []string
			for _, svc := range services {
				conditions = append(conditions, fmt.Sprintf("_ResourceId contains '%s'", sanitizeKQLString(svc)))
			}
			sb.WriteString(fmt.Sprintf("| where %s\n", strings.Join(conditions, " or ")))
		}
		sb.WriteString("| project TimeGenerated, Message=ResultDescription, Level, _ResourceId\n")

	case ResourceTypeFunction:
		sb.WriteString("FunctionAppLogs\n")
		sb.WriteString(fmt.Sprintf("| where TimeGenerated > ago(%s)\n", formatKQLDuration(since)))
		if len(services) > 0 {
			var conditions []string
			for _, svc := range services {
				conditions = append(conditions, fmt.Sprintf("_ResourceId contains '%s'", sanitizeKQLString(svc)))
			}
			sb.WriteString(fmt.Sprintf("| where %s\n", strings.Join(conditions, " or ")))
		}
		sb.WriteString("| project TimeGenerated, Message, Level, FunctionName, _ResourceId\n")

	case ResourceTypeAKS:
		sb.WriteString("ContainerLogV2\n")
		sb.WriteString(fmt.Sprintf("| where TimeGenerated > ago(%s)\n", formatKQLDuration(since)))
		if len(services) > 0 {
			var conditions []string
			for _, svc := range services {
				conditions = append(conditions, fmt.Sprintf("PodName contains '%s'", sanitizeKQLString(svc)))
				conditions = append(conditions, fmt.Sprintf("ContainerName contains '%s'", sanitizeKQLString(svc)))
			}
			sb.WriteString(fmt.Sprintf("| where %s\n", strings.Join(conditions, " or ")))
		}
		sb.WriteString("| project TimeGenerated, LogMessage, PodName, ContainerName, PodNamespace\n")

	case ResourceTypeContainerInstance:
		sb.WriteString("ContainerInstanceLog_CL\n")
		sb.WriteString(fmt.Sprintf("| where TimeGenerated > ago(%s)\n", formatKQLDuration(since)))
		if len(services) > 0 {
			var conditions []string
			for _, svc := range services {
				conditions = append(conditions, fmt.Sprintf("ContainerGroup_s contains '%s'", sanitizeKQLString(svc)))
			}
			sb.WriteString(fmt.Sprintf("| where %s\n", strings.Join(conditions, " or ")))
		}
		sb.WriteString("| project TimeGenerated, Message_s, ContainerName_s\n")

	default:
		return buildStreamingQueryForType(ResourceTypeContainerApp, services, since)
	}

	sb.WriteString("| order by TimeGenerated asc\n") // Chronological for streaming
	sb.WriteString("| take 500")                     // Reasonable batch size

	return sb.String()
}
