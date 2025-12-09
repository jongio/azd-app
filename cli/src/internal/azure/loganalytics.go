package azure

import (
	"context"
	"fmt"
	"strings"
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
	ResourceType  string
	ContainerName string
	InstanceID    string
}

// Default KQL queries for different resource types.
var defaultQueries = map[ResourceType]string{
	ResourceTypeContainerApp: `
ContainerAppConsoleLogs_CL
| where ContainerAppName_s == "{serviceName}" or ContainerName_s contains "{serviceName}"
| where TimeGenerated > ago({timespan})
| project TimeGenerated, Log_s, Stream_s, ContainerAppName_s, ContainerName_s, RevisionName_s
| order by TimeGenerated desc
| take 1000`,

	ResourceTypeAppService: `
AppServiceConsoleLogs
| where _ResourceId contains "{serviceName}"
| where TimeGenerated > ago({timespan})
| project TimeGenerated, ResultDescription, Level
| order by TimeGenerated desc
| take 1000`,

	ResourceTypeFunction: `
FunctionAppLogs
| where _ResourceId contains "{serviceName}"
| where TimeGenerated > ago({timespan})
| project TimeGenerated, Message, Level, FunctionName, Category, HostInstanceId, ExceptionDetails
| union (
    traces
    | where _ResourceId contains "{serviceName}" or cloud_RoleName contains "{serviceName}"
    | where TimeGenerated > ago({timespan})
    | project TimeGenerated, Message=message, Level=tostring(severityLevel), FunctionName=operation_Name, Category="traces", HostInstanceId="", ExceptionDetails=""
)
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
	client      *azlogs.Client
	workspaceID string
	credential  azcore.TokenCredential
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

// QueryLogs queries logs for a specific service.
func (c *LogAnalyticsClient) QueryLogs(ctx context.Context, serviceName string, resourceType ResourceType, since time.Duration, customQuery string) ([]LogEntry, error) {
	query := c.buildQuery(serviceName, resourceType, since, customQuery)

	// Extract workspace ID from full resource ID if needed
	workspaceID := extractWorkspaceID(c.workspaceID)

	// Format timespan as ISO8601 duration (e.g., "PT1H" for 1 hour)
	timespan := azlogs.TimeInterval(formatTimespan(since))

	resp, err := c.client.QueryWorkspace(ctx, workspaceID, azlogs.QueryBody{
		Query:    &query,
		Timespan: &timespan,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("Log Analytics query failed: %w", err)
	}

	return c.parseResults(resp, serviceName, resourceType)
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
	if customQuery != "" {
		query = customQuery
	} else {
		query = defaultQueries[resourceType]
		if query == "" {
			query = defaultQueries[ResourceTypeContainerApp] // fallback
		}
	}

	// Replace placeholders with sanitized values
	query = strings.ReplaceAll(query, "{serviceName}", sanitizeKQLString(serviceName))
	query = strings.ReplaceAll(query, "{timespan}", formatTimespan(timespan))

	return strings.TrimSpace(query)
}

// parseResults converts the Log Analytics response to LogEntry slice.
func (c *LogAnalyticsClient) parseResults(resp azlogs.QueryWorkspaceResponse, serviceName string, resourceType ResourceType) ([]LogEntry, error) {
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

			// Extract container/instance info
			entry.ContainerName = getStringFromRow(row, colIndex, "ContainerName_s", "ContainerName")
			entry.InstanceID = getStringFromRow(row, colIndex, "RevisionName_s", "PodName", "InstanceId")

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

	// Infer from message content
	msgLower := strings.ToLower(message)
	if strings.Contains(msgLower, "error") || strings.Contains(msgLower, "exception") || strings.Contains(msgLower, "failed") {
		return LogLevelError
	}
	if strings.Contains(msgLower, "warning") || strings.Contains(msgLower, "warn") {
		return LogLevelWarn
	}
	if strings.Contains(msgLower, "debug") {
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
	// Azure Monitor uses ISO 8601 duration format
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

func extractWorkspaceID(fullID string) string {
	// If it's already just the GUID, return as-is
	if !strings.Contains(fullID, "/") {
		return fullID
	}

	// Extract workspace ID from full resource ID
	// Format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.OperationalInsights/workspaces/{name}
	parts := strings.Split(fullID, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return fullID
}
