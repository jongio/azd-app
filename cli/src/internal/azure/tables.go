// Package azure provides Azure-specific functionality for the azd app extension.
package azure

// TableInfo represents metadata about a Log Analytics table.
type TableInfo struct {
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Columns     []string `json:"columns,omitempty"`
	Recommended bool     `json:"recommended,omitempty"`
}

// TableCategory represents a category of related Log Analytics tables.
type TableCategory struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Tables      []string `json:"tables"`
}

// TableCategories defines predefined table categories for Azure resource types.
var TableCategories = map[string]TableCategory{
	"containerapp": {
		Name:        "containerapp",
		DisplayName: "Container Apps",
		Tables: []string{
			"ContainerAppConsoleLogs_CL",
			"ContainerAppSystemLogs_CL",
		},
	},
	"appservice": {
		Name:        "appservice",
		DisplayName: "App Service",
		Tables: []string{
			"AppServiceConsoleLogs",
			"AppServiceHTTPLogs",
			"AppServicePlatformLogs",
			"AppServiceAppLogs",
			"AppServiceAuditLogs",
		},
	},
	"function": {
		Name:        "function",
		DisplayName: "Azure Functions",
		Tables: []string{
			"FunctionAppLogs",
			"AppServiceConsoleLogs",
		},
	},
	"aks": {
		Name:        "aks",
		DisplayName: "Kubernetes Service",
		Tables: []string{
			"ContainerLogV2",
			"ContainerLog",
			"KubeEvents",
			"KubePodInventory",
			"KubeNodeInventory",
			"KubeServices",
			"InsightsMetrics",
		},
	},
	"aci": {
		Name:        "aci",
		DisplayName: "Container Instances",
		Tables: []string{
			"ContainerInstanceLog_CL",
			"ContainerEvent_CL",
		},
	},
}

// TableDescriptions provides human-readable descriptions for common tables.
var TableDescriptions = map[string]string{
	// Container Apps
	"ContainerAppConsoleLogs_CL": "Console output (stdout/stderr) from Container Apps",
	"ContainerAppSystemLogs_CL":  "System-level logs from Container Apps platform",

	// App Service
	"AppServiceConsoleLogs":  "Console output from App Service applications",
	"AppServiceHTTPLogs":     "HTTP request logs from App Service",
	"AppServicePlatformLogs": "Platform-level logs from App Service",
	"AppServiceAppLogs":      "Application logs from App Service",
	"AppServiceAuditLogs":    "Audit logs from App Service",

	// Functions
	"FunctionAppLogs": "Logs from Azure Functions executions",

	// AKS
	"ContainerLogV2":    "Container logs (v2 schema) from Kubernetes",
	"ContainerLog":      "Container logs (legacy schema) from Kubernetes",
	"KubeEvents":        "Kubernetes cluster events",
	"KubePodInventory":  "Inventory of Kubernetes pods",
	"KubeNodeInventory": "Inventory of Kubernetes nodes",
	"KubeServices":      "Kubernetes service definitions",
	"InsightsMetrics":   "Performance metrics from Container Insights",

	// Container Instances
	"ContainerInstanceLog_CL": "Console output from Container Instances",
	"ContainerEvent_CL":       "Events from Container Instances",
}

// TableColumns provides common columns for known tables.
var TableColumns = map[string][]string{
	"ContainerAppConsoleLogs_CL": {"TimeGenerated", "Log_s", "Stream_s", "ContainerAppName_s", "ContainerName_s", "RevisionName_s"},
	"ContainerAppSystemLogs_CL":  {"TimeGenerated", "Log_s", "Type_s", "ContainerAppName_s", "RevisionName_s"},
	"AppServiceConsoleLogs":      {"TimeGenerated", "ResultDescription", "Level", "Host"},
	"AppServiceHTTPLogs":         {"TimeGenerated", "CsMethod", "CsUriStem", "ScStatus", "TimeTaken", "CIp"},
	"AppServicePlatformLogs":     {"TimeGenerated", "Message", "Level", "ContainerName"},
	"FunctionAppLogs":            {"TimeGenerated", "Source", "Message", "Level", "FunctionName", "Category", "HostInstanceId", "AzureResourceId"},
	"ContainerLogV2":             {"TimeGenerated", "LogMessage", "LogSource", "PodName", "PodNamespace", "ContainerName"},
	"ContainerLog":               {"TimeGenerated", "LogEntry", "LogEntrySource", "Name", "ContainerID"},
	"KubeEvents":                 {"TimeGenerated", "Name", "Namespace", "Reason", "Message", "Type"},
	"ContainerInstanceLog_CL":    {"TimeGenerated", "Message_s", "ContainerName_s", "ContainerGroup_s"},
}

// DefaultTablesByResourceType returns the default (recommended) tables for a resource type.
var DefaultTablesByResourceType = map[ResourceType][]string{
	ResourceTypeContainerApp:      {"ContainerAppConsoleLogs_CL"},
	ResourceTypeAppService:        {"AppServiceConsoleLogs"},
	ResourceTypeFunction:          {"FunctionAppLogs"},
	ResourceTypeAKS:               {"ContainerLogV2"},
	ResourceTypeContainerInstance: {"ContainerInstanceLog_CL"},
}

// GetTableInfo returns detailed information about a table.
func GetTableInfo(tableName string) TableInfo {
	info := TableInfo{
		Name:    tableName,
		Columns: TableColumns[tableName],
	}

	// Add description if available
	if desc, ok := TableDescriptions[tableName]; ok {
		info.Description = desc
	}

	// Determine category
	info.Category = GetTableCategory(tableName)

	return info
}

// GetTableCategory determines the category for a table name.
// When a table appears in multiple categories, it returns the primary category
// based on a predefined priority order.
func GetTableCategory(tableName string) string {
	// Priority order for categories
	categoryPriority := []string{"containerapp", "appservice", "function", "aks", "aci"}

	// Check each category in priority order
	for _, category := range categoryPriority {
		if cat, ok := TableCategories[category]; ok {
			for _, table := range cat.Tables {
				if table == tableName {
					return category
				}
			}
		}
	}

	return "other"
}

// GetRecommendedTables returns recommended tables for a resource type.
func GetRecommendedTables(resourceType ResourceType) []string {
	if tables, ok := DefaultTablesByResourceType[resourceType]; ok {
		return tables
	}
	// Default to Container App tables
	return DefaultTablesByResourceType[ResourceTypeContainerApp]
}

// GetTablesForCategory returns all tables in a category.
func GetTablesForCategory(category string) []string {
	if cat, ok := TableCategories[category]; ok {
		return cat.Tables
	}
	return nil
}

// GetAllKnownTables returns all known table names across all categories.
func GetAllKnownTables() []TableInfo {
	seen := make(map[string]bool)
	var tables []TableInfo

	for _, cat := range TableCategories {
		for _, tableName := range cat.Tables {
			if !seen[tableName] {
				seen[tableName] = true
				tables = append(tables, GetTableInfo(tableName))
			}
		}
	}

	return tables
}

// IsRecommendedTable checks if a table is recommended for a resource type.
func IsRecommendedTable(tableName string, resourceType ResourceType) bool {
	recommended := GetRecommendedTables(resourceType)
	for _, t := range recommended {
		if t == tableName {
			return true
		}
	}
	return false
}
