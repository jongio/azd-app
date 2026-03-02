// Package azure provides Azure-specific functionality for the azd app extension.
package azure

import (
	"fmt"
	"regexp"
	"strings"
)

// kqlIdentifierPattern validates KQL table and column names.
// KQL identifiers contain only letters, digits, and underscores.
var kqlIdentifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

const (
	// defaultQueryResultLimit is the default limit for query results to prevent excessive data transfer.
	// Set to 1000 rows as a balance between completeness and performance.
	defaultQueryResultLimit = 1000
)

// QueryBuilder helps construct KQL queries from table selections.
type QueryBuilder struct {
	tables      []string
	serviceName string
	timespan    string
}

// kqlTimespanPattern validates KQL ago() timespan values.
// Valid formats: 30m, 1h, 2d, 1h30m, etc. Only digits and time unit suffixes.
var kqlTimespanPattern = regexp.MustCompile(`^[0-9]+[dhms]([0-9]+[dhms])?$`)

// NewQueryBuilder creates a new query builder for the given service.
func NewQueryBuilder(serviceName string, timespan string) *QueryBuilder {
	// Sanitize serviceName for KQL string interpolation
	sanitizedName := sanitizeKQLString(serviceName)
	// Validate timespan format to prevent injection
	if !kqlTimespanPattern.MatchString(timespan) {
		timespan = "30m" // Safe default
	}
	return &QueryBuilder{
		serviceName: sanitizedName,
		timespan:    timespan,
	}
}

// WithTables sets the tables to query.
// Table names are validated to prevent KQL injection.
func (qb *QueryBuilder) WithTables(tables []string) *QueryBuilder {
	validated := make([]string, 0, len(tables))
	for _, t := range tables {
		if isValidKQLIdentifier(t) {
			validated = append(validated, t)
		}
	}
	qb.tables = validated
	return qb
}

// isValidKQLIdentifier checks that a string is a safe KQL identifier (table or column name).
// Valid identifiers contain only letters, digits, and underscores, and start with a letter or underscore.
func isValidKQLIdentifier(s string) bool {
	return s != "" && kqlIdentifierPattern.MatchString(s)
}

// Build generates the KQL query from the configuration.
func (qb *QueryBuilder) Build() string {
	if len(qb.tables) == 0 {
		return ""
	}

	if len(qb.tables) == 1 {
		return qb.buildSingleTableQuery(qb.tables[0])
	}

	return qb.buildUnionQuery()
}

// buildSingleTableQuery generates a query for a single table.
func (qb *QueryBuilder) buildSingleTableQuery(tableName string) string {
	filter := qb.getServiceFilter(tableName)

	query := fmt.Sprintf(`%s
| where TimeGenerated > ago(%s)
%s
| project TimeGenerated, %s
| order by TimeGenerated desc
| take %d`,
		tableName,
		qb.timespan,
		filter,
		qb.getProjectColumns(tableName),
		defaultQueryResultLimit,
	)

	return strings.TrimSpace(query)
}

// buildUnionQuery generates a union query for multiple tables.
func (qb *QueryBuilder) buildUnionQuery() string {
	var parts []string

	for _, tableName := range qb.tables {
		filter := qb.getServiceFilter(tableName)
		part := fmt.Sprintf(`(%s | where TimeGenerated > ago(%s) %s | project TimeGenerated, Table="%s", %s)`,
			tableName,
			qb.timespan,
			filter,
			tableName,
			qb.getProjectColumns(tableName),
		)
		parts = append(parts, part)
	}

	query := fmt.Sprintf(`union %s
| order by TimeGenerated desc
| take %d`,
		strings.Join(parts, ", "),
		defaultQueryResultLimit,
	)

	return strings.TrimSpace(query)
}

// getServiceFilter returns the appropriate WHERE clause for filtering by service name.
func (qb *QueryBuilder) getServiceFilter(tableName string) string {
	if qb.serviceName == "" {
		return ""
	}

	// Get the appropriate column for filtering based on table
	filterCol := getServiceFilterColumn(tableName)
	if filterCol == "" {
		return ""
	}

	// Use case-insensitive comparison.
	// serviceName is already sanitized by NewQueryBuilder, safe for KQL string interpolation.
	return fmt.Sprintf("| where %s =~ '%s'", filterCol, qb.serviceName)
}

// getServiceFilterColumn returns the column to filter by service name for a table.
func getServiceFilterColumn(tableName string) string {
	switch tableName {
	case "ContainerAppConsoleLogs_CL", "ContainerAppSystemLogs_CL":
		return "ContainerAppName_s"
	case "AppServiceConsoleLogs", "AppServiceHTTPLogs", "AppServicePlatformLogs", "AppServiceAppLogs":
		return "_ResourceId"
	case "FunctionAppLogs":
		return "_ResourceId"
	case "ContainerLogV2":
		return "PodName"
	case "ContainerLog":
		return "Name"
	case "ContainerInstanceLog_CL":
		return "ContainerGroup_s"
	case "KubeEvents":
		return "Name"
	default:
		return ""
	}
}

// getProjectColumns returns the columns to project for a table.
func (qb *QueryBuilder) getProjectColumns(tableName string) string {
	columns := TableColumns[tableName]
	if len(columns) == 0 {
		// Default columns if table is not known
		return "Message=tostring(column_ifexists('Message', '')), Level=tostring(column_ifexists('Level', 'INFO'))"
	}

	// Build projection with column aliases for common fields
	var parts []string
	for _, col := range columns {
		if col == "TimeGenerated" {
			continue // Already in the base projection
		}
		parts = append(parts, col)
	}

	// Add Message alias for display
	messageCol := getMessageColumn(tableName)
	if messageCol != "" && !containsString(parts, "Message") {
		parts = append(parts, fmt.Sprintf("Message=%s", messageCol))
	}

	return strings.Join(parts, ", ")
}

// getMessageColumn returns the primary message column for a table.
func getMessageColumn(tableName string) string {
	switch tableName {
	case "ContainerAppConsoleLogs_CL":
		return "Log_s"
	case "ContainerAppSystemLogs_CL":
		return "Log_s"
	case "AppServiceConsoleLogs":
		return "ResultDescription"
	case "AppServiceHTTPLogs":
		return "CsUriStem"
	case "AppServicePlatformLogs":
		return "Message"
	case "FunctionAppLogs":
		return "Message"
	case "ContainerLogV2":
		return "LogMessage"
	case "ContainerLog":
		return "LogEntry"
	case "ContainerInstanceLog_CL":
		return "Message_s"
	case "KubeEvents":
		return "Message"
	default:
		return ""
	}
}

// containsString checks if a slice contains a string.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// BuildQueryFromTables generates a KQL query for the given tables.
// This is a convenience function that creates a QueryBuilder and builds the query.
func BuildQueryFromTables(tables []string, serviceName, timespan string) string {
	return NewQueryBuilder(serviceName, timespan).
		WithTables(tables).
		Build()
}

// SubstitutePlaceholders replaces {serviceName} and {timespan} placeholders in a query.
// serviceName is sanitized to prevent KQL injection. timespan is validated against a safe pattern.
func SubstitutePlaceholders(query, serviceName, timespan string) string {
	query = strings.ReplaceAll(query, "{serviceName}", sanitizeKQLString(serviceName))
	// Validate timespan format to prevent injection
	if !kqlTimespanPattern.MatchString(timespan) {
		timespan = "30m" // Safe default
	}
	query = strings.ReplaceAll(query, "{timespan}", timespan)
	return query
}
