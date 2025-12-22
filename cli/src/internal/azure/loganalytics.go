package azure

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azlogs"
)

// LogLevel represents the severity of a log message.
type LogLevel int

const (
	LogLevelInfo LogLevel = iota
	LogLevelWarn
	LogLevelError
	LogLevelDebug
)

// LogEntry represents a log entry from Azure Log Analytics.
type LogEntry struct {
	Service       string
	Message       string
	Level         LogLevel
	Timestamp     time.Time
	Source        string
	ResourceType  string
	ContainerName string
	InstanceID    string
}

// Note: defaultQueries uses hardcoded "1000" in the KQL templates because:
// 1. These are string templates, not executed queries
// 2. The limit is part of the KQL syntax, not a Go constant
// 3. Changing this would require string replacement or fmt.Sprintf, adding complexity
// 4. The value aligns with defaultQueryResultLimit constant in query_builder.go

// Default KQL queries for different resource types.
var defaultQueries = map[ResourceType]string{
	ResourceTypeContainerApp: `
ContainerAppConsoleLogs_CL
| where ContainerAppName_s =~ "{serviceName}" or ContainerName_s =~ "{serviceName}"
| where TimeGenerated > ago({timespan})
| extend Source = "Azure Container Apps", AzureService = "containerapp"
| project TimeGenerated, Source, Log_s, Stream_s, ContainerAppName_s, ContainerName_s, RevisionName_s
| order by TimeGenerated desc
| take 1000`,

	ResourceTypeAppService: `
AppServiceConsoleLogs
| where _ResourceId contains "{serviceName}"
| where TimeGenerated > ago({timespan})
| extend Source = "Azure App Service", AzureService = "appservice"
| project TimeGenerated, Source, Message=ResultDescription, Level
| order by TimeGenerated desc
| take 1000`,

	ResourceTypeFunction: `
FunctionAppLogs
| where _ResourceId contains "{serviceName}"
| where TimeGenerated > ago({timespan})
| extend Source = "Azure Functions", 
         AzureService = "functions",
         AzureResourceId = _ResourceId
| project TimeGenerated, 
          Source, 
          Message, 
          Level=case(Level == "Error", "ERROR", Level == "Warning", "WARN", Level == "Information", "INFO", Level == "Debug", "DEBUG", Level == "Trace", "TRACE", "INFO"), 
          FunctionName,
          Category,
          HostInstanceId,
          AzureResourceId
| order by TimeGenerated desc
| take 1000`,

	ResourceTypeAKS: `
ContainerLogV2
| where PodName contains "{serviceName}" or ContainerName contains "{serviceName}"
| where TimeGenerated > ago({timespan})
| project TimeGenerated, LogMessage, PodName, ContainerName, PodNamespace
| order by TimeGenerated desc
| take 1000`,

	ResourceTypeContainerInstance: `
ContainerInstanceLog_CL
| where ContainerGroup_s contains "{serviceName}"
| where TimeGenerated > ago({timespan})
| project TimeGenerated, Message_s, ContainerName_s
| order by TimeGenerated desc
| take 1000`,
}

// LogAnalyticsClient provides methods to query Azure Log Analytics.
type LogAnalyticsClient struct {
	client        *azlogs.Client
	workspaceID   string
	credential    azcore.TokenCredential
	workspaceGUID string
	resolveOnce   sync.Once
	resolveErr    error
}

// NewLogAnalyticsClient creates a new Log Analytics client.
func NewLogAnalyticsClient(credential azcore.TokenCredential, workspaceID string) (*LogAnalyticsClient, error) {
	client, err := azlogs.NewClient(credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Log Analytics client: %w", err)
	}

	return &LogAnalyticsClient{
		client:      client,
		workspaceID: workspaceID,
		credential:  credential,
	}, nil
}

func (c *LogAnalyticsClient) getWorkspaceGUID(ctx context.Context) (string, error) {
	c.resolveOnce.Do(func() {
		c.workspaceGUID, c.resolveErr = NormalizeWorkspaceID(ctx, c.credential, c.workspaceID)
	})
	return c.workspaceGUID, c.resolveErr
}

// QueryLogs queries logs for a specific service.
func (c *LogAnalyticsClient) QueryLogs(ctx context.Context, serviceName string, resourceType ResourceType, since time.Duration, customQuery string) ([]LogEntry, error) {
	query := c.buildQuery(serviceName, resourceType, since, customQuery)

	workspaceID, err := c.getWorkspaceGUID(ctx)
	if err != nil {
		return nil, fmt.Errorf("invalid Log Analytics workspace configuration: %w", err)
	}

	// Format timespan as ISO8601 duration (e.g., "PT1H" for 1 hour)
	timespan := azlogs.TimeInterval(formatTimespan(since))

	slog.Debug("executing KQL query", "workspace", workspaceID, "service", serviceName, "resourceType", resourceType, "timespan", timespan, "query", query)

	resp, err := c.client.QueryWorkspace(ctx, workspaceID, azlogs.QueryBody{
		Query:    &query,
		Timespan: &timespan,
	}, &azlogs.QueryWorkspaceOptions{
		Options: &azlogs.QueryOptions{
			Wait:       toPtrInt(600),   // 10 minute timeout for complex queries
			Statistics: toPtrBool(true), // Enable query performance statistics
		},
	})
	if err != nil {
		// Don't log cancellation errors - they're expected when client aborts request
		if ctx.Err() == nil {
			slog.Debug("KQL query failed", "error", err)
		}
		// Clear token cache on auth errors to force fresh token on next attempt
		ClearTokenCacheOnError(err)
		return nil, fmt.Errorf("log analytics query failed: %w", err)
	}

	entries, err := c.parseResults(resp, serviceName, resourceType)
	slog.Debug("KQL query results", "service", serviceName, "rowCount", len(entries))
	return entries, err
}

// QueryLogsSince queries logs since a specific timestamp.
func (c *LogAnalyticsClient) QueryLogsSince(ctx context.Context, serviceName string, resourceType ResourceType, sinceTime time.Time, customQuery string) ([]LogEntry, error) {
	since := time.Since(sinceTime)
	if since < time.Second {
		since = time.Second
	}
	return c.QueryLogs(ctx, serviceName, resourceType, since, customQuery)
}

// sanitizeKQLString escapes special characters to prevent KQL injection.
func sanitizeKQLString(s string) string {
	// Escape single quotes (KQL string delimiter)
	s = strings.ReplaceAll(s, "'", "''")
	// Escape backslashes
	s = strings.ReplaceAll(s, "\\", "\\\\")
	return s
}

// buildQuery constructs the KQL query with substituted placeholders.
func (c *LogAnalyticsClient) buildQuery(serviceName string, resourceType ResourceType, timespan time.Duration, customQuery string) string {
	var query string
	var querySource string
	if customQuery != "" {
		query = customQuery
		querySource = "custom"
	} else {
		query = defaultQueries[resourceType]
		querySource = string(resourceType)
		if query == "" {
			query = defaultQueries[ResourceTypeContainerApp] // fallback
			querySource = "fallback-containerApp"
		}
	}

	slog.Debug("building KQL query", "serviceName", serviceName, "resourceType", resourceType, "querySource", querySource)

	// Replace placeholders with sanitized values
	query = strings.ReplaceAll(query, "{serviceName}", sanitizeKQLString(serviceName))
	query = strings.ReplaceAll(query, "{timespan}", formatKQLTimespan(timespan))

	return strings.TrimSpace(query)
}

// parseResults converts the Log Analytics response to LogEntry slice.
func (c *LogAnalyticsClient) parseResults(resp azlogs.QueryWorkspaceResponse, serviceName string, resourceType ResourceType) ([]LogEntry, error) {
	// Check for query-level errors returned by Azure
	if resp.Error != nil {
		return nil, fmt.Errorf("query error: %s (code: %s)", resp.Error.Error(), resp.Error.Code)
	}

	// Log query statistics for performance debugging
	if len(resp.Statistics) > 0 {
		slog.Debug("query performance statistics", "stats", string(resp.Statistics))
	}

	var entries []LogEntry

	for _, table := range resp.Tables {
		// Build column index map
		colIndex := make(map[string]int)
		for i, col := range table.Columns {
			if col.Name != nil {
				colIndex[*col.Name] = i
			}
		}

		// Process rows
		for _, row := range table.Rows {
			entry := LogEntry{
				Service:      serviceName,
				ResourceType: string(resourceType),
			}

			// Extract timestamp
			if idx, ok := colIndex["TimeGenerated"]; ok && idx < len(row) {
				if ts, ok := row[idx].(string); ok {
					if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
						entry.Timestamp = t
					}
				}
			}

			// Extract message based on resource type
			entry.Message = c.extractMessage(row, colIndex, resourceType)

			// Extract level
			entry.Level = c.extractLevel(row, colIndex, entry.Message, resourceType)

			// Extract source
			entry.Source = getStringFromRow(row, colIndex, "Source")

			// Extract container/instance info
			entry.ContainerName = getStringFromRow(row, colIndex, "ContainerName_s", "ContainerName")
			entry.InstanceID = getStringFromRow(row, colIndex, "RevisionName_s", "PodName", "InstanceId")

			// Extract service name from resource id when absent
			if entry.Service == "" {
				if resourceID := getStringFromRow(row, colIndex, "_ResourceId", "ResourceId", "resourceId"); resourceID != "" {
					parts := strings.Split(resourceID, "/")
					if len(parts) > 0 {
						entry.Service = parts[len(parts)-1]
					}
				}
			}

			// If service name is empty, try to extract from data
			if entry.Service == "" {
				entry.Service = getStringFromRow(row, colIndex, "ContainerAppName_s", "ContainerName_s", "ServiceName")
			}

			if entry.Message != "" {
				entries = append(entries, entry)
			}
		}
	}

	return entries, nil
}

// extractMessage extracts the log message from the row based on resource type.
func (c *LogAnalyticsClient) extractMessage(row []any, colIndex map[string]int, resourceType ResourceType) string {
	switch resourceType {
	case ResourceTypeContainerApp:
		return getStringFromRow(row, colIndex, "Log_s", "Message")
	case ResourceTypeAppService:
		return getStringFromRow(row, colIndex, "ResultDescription", "Message")
	case ResourceTypeFunction:
		// For Function Apps, include FunctionName context in the message if available
		msg := getStringFromRow(row, colIndex, "Message")
		funcName := getStringFromRow(row, colIndex, "FunctionName")
		category := getStringFromRow(row, colIndex, "Category")

		if funcName != "" && msg != "" {
			// Prepend function name to message for context
			return "[" + funcName + "] " + msg
		}
		if category != "" && msg != "" && funcName == "" {
			// If no function name but has category, include it
			return "[" + category + "] " + msg
		}
		return msg
	case ResourceTypeAKS:
		return getStringFromRow(row, colIndex, "LogMessage", "Message")
	case ResourceTypeContainerInstance:
		return getStringFromRow(row, colIndex, "Message_s", "Message")
	default:
		return getStringFromRow(row, colIndex, "Message", "Log_s", "ResultDescription", "LogMessage")
	}
}

// extractLevel extracts the log level from the row.
func (c *LogAnalyticsClient) extractLevel(row []any, colIndex map[string]int, message string, resourceType ResourceType) LogLevel {
	// Try to get explicit level field
	levelStr := getStringFromRow(row, colIndex, "Level", "Stream_s")
	levelStr = strings.ToLower(levelStr)

	switch levelStr {
	case "error", "err", "stderr", "critical", "fatal":
		return LogLevelError
	case "warning", "warn":
		return LogLevelWarn
	case "debug", "trace", "verbose":
		return LogLevelDebug
	case "info", "information", "stdout":
		return LogLevelInfo
	}

	// Infer from message content using same logic as local logs
	return inferLogLevelFromMessage(message)
}

// inferLogLevelFromMessage infers log level from message content.
// Uses the same patterns as local log inference for consistency.
func inferLogLevelFromMessage(message string) LogLevel {
	msgLower := strings.ToLower(message)

	// Check for patterns that should always be INFO (overrides error/warning detection)
	// These are success messages that contain words like "error" but aren't actually errors
	infoOverrides := []string{
		"found 0 errors",
		"0 error(s)",
		"0 errors",
		"build succeeded",
		"compilation succeeded",
		"compiled successfully",
		"0 failed",
		"all tests passed",
	}
	for _, pattern := range infoOverrides {
		if strings.Contains(msgLower, pattern) {
			return LogLevelInfo
		}
	}

	// Check for error indicators
	if strings.Contains(msgLower, "error") || strings.Contains(msgLower, "exception") ||
		strings.Contains(msgLower, "fatal") || strings.Contains(msgLower, "panic") ||
		strings.Contains(msgLower, "failed") {
		return LogLevelError
	}

	// Check for warning indicators
	if strings.Contains(msgLower, "warn") || strings.Contains(msgLower, "warning") {
		return LogLevelWarn
	}

	// Check for debug indicators
	if strings.Contains(msgLower, "debug") || strings.Contains(msgLower, "trace") {
		return LogLevelDebug
	}

	return LogLevelInfo
}

// GetDefaultQuery returns the default KQL query for a resource type.
func GetDefaultQuery(resourceType ResourceType) string {
	return strings.TrimSpace(defaultQueries[resourceType])
}

// Helper functions

func getStringFromRow(row []any, colIndex map[string]int, columns ...string) string {
	for _, col := range columns {
		if idx, ok := colIndex[col]; ok && idx < len(row) {
			if s, ok := row[idx].(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

func formatTimespan(d time.Duration) string {
	// Azure Monitor API uses ISO 8601 duration format for the timespan parameter
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("PT%dH%dM%dS", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("PT%dM%dS", minutes, seconds)
	}
	return fmt.Sprintf("PT%dS", seconds)
}

// toPtrInt returns a pointer to the given int value.
func toPtrInt(v int) *int {
	return &v
}

// toPtrBool returns a pointer to the given bool value.
func toPtrBool(v bool) *bool {
	return &v
}

// formatKQLTimespan formats a duration for use in KQL ago() function.
// KQL expects formats like "30m", "1h", "1d", not ISO 8601.
func formatKQLTimespan(d time.Duration) string {
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
	return fmt.Sprintf("%ds", seconds)
}

// ListAvailableTables queries the Log Analytics workspace to discover available tables.
// It uses a search query to find distinct table names that have data.
func (c *LogAnalyticsClient) ListAvailableTables(ctx context.Context) ([]TableInfo, error) {
	// Query to get distinct table names from the workspace
	// Using search * to find all tables with data in the last 24 hours
	query := `search * 
| where TimeGenerated > ago(24h)
| distinct $table
| order by $table asc
| take 100`

	workspaceID, err := c.getWorkspaceGUID(ctx)
	if err != nil {
		return nil, fmt.Errorf("invalid Log Analytics workspace configuration: %w", err)
	}
	timespan := azlogs.TimeInterval("P1D") // Last 24 hours

	slog.Debug("listing available tables", "workspace", workspaceID)

	resp, err := c.client.QueryWorkspace(ctx, workspaceID, azlogs.QueryBody{
		Query:    &query,
		Timespan: &timespan,
	}, nil)
	if err != nil {
		slog.Debug("failed to list tables", "error", err)
		ClearTokenCacheOnError(err)
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}

	// Parse the results to extract table names
	var tables []TableInfo
	seenTables := make(map[string]bool)

	for _, table := range resp.Tables {
		// Find the $table column
		tableColIdx := -1
		for i, col := range table.Columns {
			if col.Name != nil && *col.Name == "$table" {
				tableColIdx = i
				break
			}
		}

		if tableColIdx < 0 {
			continue
		}

		// Extract table names from rows
		for _, row := range table.Rows {
			if tableColIdx < len(row) {
				if tableName, ok := row[tableColIdx].(string); ok && tableName != "" {
					if !seenTables[tableName] {
						seenTables[tableName] = true
						info := GetTableInfo(tableName)
						tables = append(tables, info)
					}
				}
			}
		}
	}

	slog.Debug("found available tables", "count", len(tables))
	return tables, nil
}

// ListTablesWithFallback returns available tables, falling back to known tables if query fails.
func (c *LogAnalyticsClient) ListTablesWithFallback(ctx context.Context, resourceType ResourceType) []TableInfo {
	// Try to query available tables
	tables, err := c.ListAvailableTables(ctx)
	if err != nil {
		slog.Debug("falling back to known tables", "error", err, "resourceType", resourceType)
		// Fall back to known tables for the resource type
		return GetAllKnownTables()
	}

	// Mark recommended tables
	for i := range tables {
		tables[i].Recommended = IsRecommendedTable(tables[i].Name, resourceType)
	}

	return tables
}
