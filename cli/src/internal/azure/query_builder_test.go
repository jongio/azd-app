package azure

import (
	"strings"
	"testing"
)

const (
	testServiceName       = "test-service"
	testTimespan          = "30m"
	testMyApp             = "my-app"
	nonEmptyQueryMessage  = "Build should return non-empty query"
	orderByTimeDescending = "order by TimeGenerated desc"
)

func TestNewQueryBuilder(t *testing.T) {
	qb := NewQueryBuilder(testServiceName, testTimespan)

	if qb == nil {
		t.Fatal("NewQueryBuilder returned nil")
	}
	if qb.serviceName != "test-service" {
		t.Errorf("serviceName = %q, want %q", qb.serviceName, "test-service")
	}
	if qb.timespan != "30m" {
		t.Errorf("timespan = %q, want %q", qb.timespan, "30m")
	}
	if len(qb.tables) != 0 {
		t.Errorf("tables should be empty, got %d tables", len(qb.tables))
	}
}

func TestQueryBuilder_WithTables(t *testing.T) {
	qb := NewQueryBuilder("test-service", "30m")
	tables := []string{"ContainerAppConsoleLogs_CL", "AppServiceConsoleLogs"}

	result := qb.WithTables(tables)

	if result != qb {
		t.Error("WithTables should return the same builder instance")
	}
	if len(qb.tables) != 2 {
		t.Errorf("tables length = %d, want 2", len(qb.tables))
	}
	if qb.tables[0] != "ContainerAppConsoleLogs_CL" {
		t.Errorf("tables[0] = %q, want %q", qb.tables[0], "ContainerAppConsoleLogs_CL")
	}
}

func TestQueryBuilder_Build_EmptyTables(t *testing.T) {
	qb := NewQueryBuilder("test-service", "30m")

	query := qb.Build()

	if query != "" {
		t.Errorf("Build with no tables should return empty string, got %q", query)
	}
}

func TestQueryBuilder_Build_SingleTable(t *testing.T) {
	qb := NewQueryBuilder("test-service", "30m").
		WithTables([]string{"ContainerAppConsoleLogs_CL"})

	query := qb.Build()

	if query == "" {
		t.Fatal("Build should return non-empty query")
	}

	// Verify query structure
	if !strings.Contains(query, "ContainerAppConsoleLogs_CL") {
		t.Error("Query should contain table name")
	}
	if !strings.Contains(query, "where TimeGenerated > ago(30m)") {
		t.Error("Query should contain timespan filter")
	}
	if !strings.Contains(query, "where ContainerAppName_s =~ 'test-service'") {
		t.Error("Query should contain service name filter")
	}
	if !strings.Contains(query, "order by TimeGenerated desc") {
		t.Error("Query should order by TimeGenerated")
	}
	if !strings.Contains(query, "take 1000") {
		t.Error("Query should limit results")
	}
}

func TestQueryBuilder_Build_SingleTable_NoServiceFilter(t *testing.T) {
	qb := NewQueryBuilder("", "1h").
		WithTables([]string{"AppServiceConsoleLogs"})

	query := qb.Build()

	if query == "" {
		t.Fatal("Build should return non-empty query")
	}

	// Should not have service filter when serviceName is empty
	if strings.Contains(query, "where _ResourceId") {
		t.Error("Query should not contain service filter when serviceName is empty")
	}
	if !strings.Contains(query, "where TimeGenerated > ago(1h)") {
		t.Error("Query should still contain timespan filter")
	}
}

func TestQueryBuilder_Build_MultipleTablesUnion(t *testing.T) {
	qb := NewQueryBuilder("test-service", "30m").
		WithTables([]string{"ContainerAppConsoleLogs_CL", "ContainerAppSystemLogs_CL"})

	query := qb.Build()

	if query == "" {
		t.Fatal("Build should return non-empty query")
	}

	// Verify union query structure
	if !strings.Contains(query, "union") {
		t.Error("Query should use union for multiple tables")
	}
	if !strings.Contains(query, "ContainerAppConsoleLogs_CL") {
		t.Error("Query should contain first table")
	}
	if !strings.Contains(query, "ContainerAppSystemLogs_CL") {
		t.Error("Query should contain second table")
	}
	if !strings.Contains(query, "Table=\"ContainerAppConsoleLogs_CL\"") {
		t.Error("Query should project table name")
	}
	if !strings.Contains(query, "order by TimeGenerated desc") {
		t.Error("Query should order by TimeGenerated")
	}
}

func TestGetServiceFilterColumn(t *testing.T) {
	tests := []struct {
		tableName string
		want      string
	}{
		{"ContainerAppConsoleLogs_CL", "ContainerAppName_s"},
		{"ContainerAppSystemLogs_CL", "ContainerAppName_s"},
		{"AppServiceConsoleLogs", "_ResourceId"},
		{"AppServiceHTTPLogs", "_ResourceId"},
		{"FunctionAppLogs", "_ResourceId"},
		{"ContainerLogV2", "PodName"},
		{"ContainerLog", "Name"},
		{"ContainerInstanceLog_CL", "ContainerGroup_s"},
		{"KubeEvents", "Name"},
		{"UnknownTable", ""},
	}

	for _, tt := range tests {
		t.Run(tt.tableName, func(t *testing.T) {
			got := getServiceFilterColumn(tt.tableName)
			if got != tt.want {
				t.Errorf("getServiceFilterColumn(%q) = %q, want %q", tt.tableName, got, tt.want)
			}
		})
	}
}

func TestGetMessageColumn(t *testing.T) {
	tests := []struct {
		tableName string
		want      string
	}{
		{"ContainerAppConsoleLogs_CL", "Log_s"},
		{"ContainerAppSystemLogs_CL", "Log_s"},
		{"AppServiceConsoleLogs", "ResultDescription"},
		{"AppServiceHTTPLogs", "CsUriStem"},
		{"AppServicePlatformLogs", "Message"},
		{"FunctionAppLogs", "Message"},
		{"ContainerLogV2", "LogMessage"},
		{"ContainerLog", "LogEntry"},
		{"ContainerInstanceLog_CL", "Message_s"},
		{"KubeEvents", "Message"},
		{"UnknownTable", ""},
	}

	for _, tt := range tests {
		t.Run(tt.tableName, func(t *testing.T) {
			got := getMessageColumn(tt.tableName)
			if got != tt.want {
				t.Errorf("getMessageColumn(%q) = %q, want %q", tt.tableName, got, tt.want)
			}
		})
	}
}

func TestGetProjectColumns(t *testing.T) {
	qb := NewQueryBuilder("test-service", "30m")

	// Test with known table
	columns := qb.getProjectColumns("ContainerAppConsoleLogs_CL")
	if !strings.Contains(columns, "Log_s") {
		t.Error("Should project Log_s column for ContainerAppConsoleLogs_CL")
	}
	if !strings.Contains(columns, "Stream_s") {
		t.Error("Should project Stream_s column")
	}
	if !strings.Contains(columns, "ContainerAppName_s") {
		t.Error("Should project ContainerAppName_s column")
	}

	// Test with unknown table
	columns = qb.getProjectColumns("UnknownTable")
	if !strings.Contains(columns, "Message") {
		t.Error("Should project Message alias for unknown table")
	}
	if !strings.Contains(columns, "column_ifexists") {
		t.Error("Should use column_ifexists for unknown table")
	}
}

func TestContainsString(t *testing.T) {
	slice := []string{"apple", "banana", "cherry"}

	if !containsString(slice, "banana") {
		t.Error("Should find 'banana' in slice")
	}
	if containsString(slice, "orange") {
		t.Error("Should not find 'orange' in slice")
	}
	if containsString([]string{}, "test") {
		t.Error("Should not find anything in empty slice")
	}
}

func TestBuildQueryFromTables(t *testing.T) {
	tables := []string{"ContainerAppConsoleLogs_CL"}
	query := BuildQueryFromTables(tables, "test-service", "1h")

	if query == "" {
		t.Fatal("BuildQueryFromTables should return non-empty query")
	}

	if !strings.Contains(query, "ContainerAppConsoleLogs_CL") {
		t.Error("Query should contain table name")
	}
	if !strings.Contains(query, "test-service") {
		t.Error("Query should contain service name")
	}
	if !strings.Contains(query, "1h") {
		t.Error("Query should contain timespan")
	}
}

func TestBuildQueryFromTables_MultipleServices(t *testing.T) {
	tables := []string{"AppServiceConsoleLogs", "AppServiceHTTPLogs"}
	query := BuildQueryFromTables(tables, "my-app", "30m")

	if query == "" {
		t.Fatal("BuildQueryFromTables should return non-empty query")
	}

	if !strings.Contains(query, "union") {
		t.Error("Query should use union for multiple tables")
	}
	if !strings.Contains(query, "AppServiceConsoleLogs") {
		t.Error("Query should contain first table")
	}
	if !strings.Contains(query, "AppServiceHTTPLogs") {
		t.Error("Query should contain second table")
	}
}

func TestSubstitutePlaceholders(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		serviceName string
		timespan    string
		wantContain []string
	}{
		{
			name:        "Replace service name",
			query:       "MyTable | where Service == '{serviceName}'",
			serviceName: "test-service",
			timespan:    "30m",
			wantContain: []string{"test-service"},
		},
		{
			name:        "Replace timespan",
			query:       "MyTable | where TimeGenerated > ago({timespan})",
			serviceName: "test-service",
			timespan:    "1h",
			wantContain: []string{"1h"},
		},
		{
			name:        "Replace both placeholders",
			query:       "MyTable | where TimeGenerated > ago({timespan}) | where Service == '{serviceName}'",
			serviceName: "my-app",
			timespan:    "2h",
			wantContain: []string{"2h", "my-app"},
		},
		{
			name:        "No placeholders",
			query:       "MyTable | project TimeGenerated, Message",
			serviceName: "test-service",
			timespan:    "30m",
			wantContain: []string{"MyTable", "TimeGenerated", "Message"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SubstitutePlaceholders(tt.query, tt.serviceName, tt.timespan)

			for _, want := range tt.wantContain {
				if !strings.Contains(result, want) {
					t.Errorf("Result should contain %q, got: %s", want, result)
				}
			}

			// Should not contain unreplaced placeholders
			if strings.Contains(result, "{serviceName}") {
				t.Error("Result should not contain {serviceName} placeholder")
			}
			if strings.Contains(result, "{timespan}") {
				t.Error("Result should not contain {timespan} placeholder")
			}
		})
	}
}

func TestSubstitutePlaceholders_SanitizeServiceName(t *testing.T) {
	// Service names should be sanitized to prevent KQL injection
	query := "MyTable | where Service == '{serviceName}'"
	serviceName := "test'; drop table--"
	timespan := "30m"

	result := SubstitutePlaceholders(query, serviceName, timespan)

	// The sanitizeKQLString function should escape quotes by doubling them
	// The result should be a safe string literal, not an injection
	// Expected: 'test''; drop table--' which is a complete string literal
	// containing the text: test'; drop table--
	expected := "MyTable | where Service == 'test''; drop table--'"
	if result != expected {
		t.Errorf("SubstitutePlaceholders() = %q, want %q", result, expected)
	}

	// Verify that the single quote was doubled for proper escaping
	if !strings.Contains(result, "test''") {
		t.Error("Single quote should be escaped by doubling it")
	}

	// The result should still contain the original dangerous text as a safe string literal
	// This is correct - the entire thing is treated as a string value, not executable code
}

func TestQueryBuilder_Build_DifferentResourceTypes(t *testing.T) {
	tests := []struct {
		name        string
		table       string
		serviceName string
		wantFilter  string
	}{
		{
			name:        "Container App",
			table:       "ContainerAppConsoleLogs_CL",
			serviceName: "my-container-app",
			wantFilter:  "ContainerAppName_s =~ 'my-container-app'",
		},
		{
			name:        "App Service",
			table:       "AppServiceConsoleLogs",
			serviceName: "my-app-service",
			wantFilter:  "_ResourceId =~ 'my-app-service'",
		},
		{
			name:        "Function App",
			table:       "FunctionAppLogs",
			serviceName: "my-function",
			wantFilter:  "_ResourceId =~ 'my-function'",
		},
		{
			name:        "Kubernetes",
			table:       "ContainerLogV2",
			serviceName: "my-pod",
			wantFilter:  "PodName =~ 'my-pod'",
		},
		{
			name:        "Container Instances",
			table:       "ContainerInstanceLog_CL",
			serviceName: "my-container-group",
			wantFilter:  "ContainerGroup_s =~ 'my-container-group'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewQueryBuilder(tt.serviceName, "30m").
				WithTables([]string{tt.table})

			query := qb.Build()

			if !strings.Contains(query, tt.wantFilter) {
				t.Errorf("Query should contain filter %q, got:\n%s", tt.wantFilter, query)
			}
		})
	}
}

func TestQueryBuilder_Build_ResultLimit(t *testing.T) {
	qb := NewQueryBuilder("test-service", "30m").
		WithTables([]string{"ContainerAppConsoleLogs_CL"})

	query := qb.Build()

	// Verify result limit
	if !strings.Contains(query, "take 1000") {
		t.Error("Query should limit results to 1000")
	}
}

func TestQueryBuilder_Build_TimeOrdering(t *testing.T) {
	qb := NewQueryBuilder("test-service", "30m").
		WithTables([]string{"ContainerAppConsoleLogs_CL"})

	query := qb.Build()

	// Verify descending time order (newest first)
	if !strings.Contains(query, "order by TimeGenerated desc") {
		t.Error("Query should order by TimeGenerated descending")
	}
}
