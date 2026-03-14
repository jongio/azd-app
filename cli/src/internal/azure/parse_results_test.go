package azure

import (
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azlogs"
)

// Test constants
const (
	testTimestamp     = "2024-12-17T10:00:00.000Z"
	parseTestService  = "test-service"
	testContainerName = "my-app"
	sourceAzureFuncs  = "Azure Functions"
)

// TestParseResults_WithSourceField tests that parseResults correctly extracts Source field
func TestParseResults_WithSourceField(t *testing.T) {
	client := &LogAnalyticsClient{}

	testCases := []struct {
		name           string
		response       azlogs.QueryWorkspaceResponse
		serviceName    string
		resourceType   ResourceType
		expectedCount  int
		expectedSource string
	}{
		{
			name: "Function app logs with Source field",
			response: createMockLogsResponse([]mockRow{
				{
					TimeGenerated: testTimestamp,
					Source:        sourceAzureFuncs,
					Message:       "Test function executed",
					Level:         "Information",
					FunctionName:  "HttpTrigger",
				},
			}),
			serviceName:    "my-function",
			resourceType:   ResourceTypeFunction,
			expectedCount:  1,
			expectedSource: sourceAzureFuncs,
		},
		{
			name: "Container app logs with Source field",
			response: createMockLogsResponse([]mockRow{
				{
					TimeGenerated:      testTimestamp,
					Source:             "Azure Container Apps",
					Log_s:              "Container started",
					Stream_s:           "stdout",
					ContainerAppName_s: testContainerName,
				},
			}),
			serviceName:    testContainerName,
			resourceType:   ResourceTypeContainerApp,
			expectedCount:  1,
			expectedSource: "Azure Container Apps",
		},
		{
			name: "App service logs with Source field",
			response: createMockLogsResponse([]mockRow{
				{
					TimeGenerated:     testTimestamp,
					Source:            "Azure App Service",
					ResultDescription: "Request processed",
					Level:             "Info",
				},
			}),
			serviceName:    "my-web-app",
			resourceType:   ResourceTypeAppService,
			expectedCount:  1,
			expectedSource: "Azure App Service",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entries, err := client.parseResults(tc.response, tc.serviceName, tc.resourceType)
			if err != nil {
				t.Fatalf("parseResults() failed: %v", err)
			}

			if len(entries) != tc.expectedCount {
				t.Errorf("Expected %d entries, got %d", tc.expectedCount, len(entries))
			}

			if len(entries) > 0 {
				entry := entries[0]
				if entry.Source != tc.expectedSource {
					t.Errorf("Expected Source %q, got %q", tc.expectedSource, entry.Source)
				}
			}
		})
	}
}

// TestParseResults_WithoutSourceField_BackwardCompatibility tests parsing without Source field
func TestParseResults_WithoutSourceField_BackwardCompatibility(t *testing.T) {
	client := &LogAnalyticsClient{}

	// Simulate old query response without Source field
	response := createMockLogsResponseWithoutSource([]mockRowNoSource{
		{
			TimeGenerated: testTimestamp,
			Message:       "Legacy log entry",
			Level:         "Information",
		},
	})

	entries, err := client.parseResults(response, parseTestService, ResourceTypeFunction)
	if err != nil {
		t.Fatalf("parseResults() should handle missing Source field: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	// Source should be empty when not present in response
	if entries[0].Source != "" {
		t.Errorf("Expected empty Source for legacy response, got %q", entries[0].Source)
	}

	// Other fields should still be parsed
	if entries[0].Message != "Legacy log entry" {
		t.Errorf("Expected message to be parsed correctly, got %q", entries[0].Message)
	}
}

// TestParseResults_NullSourceValue tests handling of null Source values
func TestParseResults_NullSourceValue(t *testing.T) {
	client := &LogAnalyticsClient{}

	response := createMockLogsResponseWithNullSource([]mockRowNullSource{
		{
			TimeGenerated: testTimestamp,
			Source:        nil, // null value
			Message:       "Test message",
			Level:         "Info",
		},
	})

	entries, err := client.parseResults(response, parseTestService, ResourceTypeFunction)
	if err != nil {
		t.Fatalf("parseResults() should handle null Source: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	// Should handle null gracefully
	if entries[0].Source != "" {
		t.Errorf("Expected empty string for null Source, got %q", entries[0].Source)
	}
}

// TestParseResults_MultipleRowsDifferentSources tests mixed source types
func TestParseResults_MultipleRowsDifferentSources(t *testing.T) {
	client := &LogAnalyticsClient{}

	response := createMockLogsResponse([]mockRow{
		{
			TimeGenerated: testTimestamp,
			Source:        sourceAzureFuncs,
			Message:       "Function log 1",
			Level:         "Info",
		},
		{
			TimeGenerated: "2024-12-17T10:01:00.000Z",
			Source:        sourceAzureFuncs,
			Message:       "Function log 2",
			Level:         "Info",
		},
	})

	entries, err := client.parseResults(response, parseTestService, ResourceTypeFunction)
	if err != nil {
		t.Fatalf("parseResults() failed: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}

	// Both should have the same Source
	for i, entry := range entries {
		if entry.Source != sourceAzureFuncs {
			t.Errorf("Entry %d: expected Source %q, got %q", i, sourceAzureFuncs, entry.Source)
		}
	}
}

// TestParseResults_EmptyResponse tests handling of empty query results
func TestParseResults_EmptyResponse(t *testing.T) {
	client := &LogAnalyticsClient{}

	response := createMockLogsResponse([]mockRow{})

	entries, err := client.parseResults(response, parseTestService, ResourceTypeFunction)
	if err != nil {
		t.Fatalf("parseResults() should handle empty response: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected 0 entries for empty response, got %d", len(entries))
	}
}

// Helper types and functions for creating mock responses

// mockRow represents Azure Log Analytics columns - field names intentionally match Azure schema
//
//nolint:revive,stylecheck // Field names match Azure Log Analytics column names
type mockRow struct {
	TimeGenerated      string
	Source             string
	Message            string
	Level              string
	FunctionName       string
	Log_s              string //nolint:stylecheck // Azure column name
	Stream_s           string //nolint:stylecheck // Azure column name
	ContainerAppName_s string //nolint:stylecheck // Azure column name
	ResultDescription  string
}

type mockRowNoSource struct {
	TimeGenerated string
	Message       string
	Level         string
}

type mockRowNullSource struct {
	TimeGenerated string
	Source        any // can be nil
	Message       string
	Level         string
}

func createMockLogsResponse(rows []mockRow) azlogs.QueryWorkspaceResponse {
	// Create column definitions
	timeCol := "TimeGenerated"
	sourceCol := "Source"
	messageCol := "Message"
	levelCol := "Level"
	funcCol := "FunctionName"
	logCol := "Log_s"
	streamCol := "Stream_s"
	containerCol := "ContainerAppName_s"
	resultCol := "ResultDescription"

	typeStr := azlogs.ColumnType("string")
	typeDatetime := azlogs.ColumnType("datetime")

	columns := []azlogs.Column{
		{Name: &timeCol, Type: &typeDatetime},
		{Name: &sourceCol, Type: &typeStr},
		{Name: &messageCol, Type: &typeStr},
		{Name: &levelCol, Type: &typeStr},
		{Name: &funcCol, Type: &typeStr},
		{Name: &logCol, Type: &typeStr},
		{Name: &streamCol, Type: &typeStr},
		{Name: &containerCol, Type: &typeStr},
		{Name: &resultCol, Type: &typeStr},
	}

	// Convert rows to interface slices
	rowData := make([]azlogs.Row, 0, len(rows))
	for _, r := range rows {
		row := azlogs.Row{
			r.TimeGenerated,
			r.Source,
			r.Message,
			r.Level,
			r.FunctionName,
			r.Log_s,
			r.Stream_s,
			r.ContainerAppName_s,
			r.ResultDescription,
		}
		rowData = append(rowData, row)
	}

	tableName := "PrimaryResult"
	return azlogs.QueryWorkspaceResponse{
		QueryResults: azlogs.QueryResults{
			Tables: []azlogs.Table{
				{
					Name:    &tableName,
					Columns: columns,
					Rows:    rowData,
				},
			},
		},
	}
}

func createMockLogsResponseWithoutSource(rows []mockRowNoSource) azlogs.QueryWorkspaceResponse {
	timeCol := "TimeGenerated"
	messageCol := "Message"
	levelCol := "Level"

	typeStr := azlogs.ColumnType("string")
	typeDatetime := azlogs.ColumnType("datetime")

	columns := []azlogs.Column{
		{Name: &timeCol, Type: &typeDatetime},
		{Name: &messageCol, Type: &typeStr},
		{Name: &levelCol, Type: &typeStr},
	}

	rowData := make([]azlogs.Row, 0, len(rows))
	for _, r := range rows {
		row := azlogs.Row{
			r.TimeGenerated,
			r.Message,
			r.Level,
		}
		rowData = append(rowData, row)
	}

	tableName := "PrimaryResult"
	return azlogs.QueryWorkspaceResponse{
		QueryResults: azlogs.QueryResults{
			Tables: []azlogs.Table{
				{
					Name:    &tableName,
					Columns: columns,
					Rows:    rowData,
				},
			},
		},
	}
}

func createMockLogsResponseWithNullSource(rows []mockRowNullSource) azlogs.QueryWorkspaceResponse {
	timeCol := "TimeGenerated"
	sourceCol := "Source"
	messageCol := "Message"
	levelCol := "Level"

	typeStr := azlogs.ColumnType("string")
	typeDatetime := azlogs.ColumnType("datetime")

	columns := []azlogs.Column{
		{Name: &timeCol, Type: &typeDatetime},
		{Name: &sourceCol, Type: &typeStr},
		{Name: &messageCol, Type: &typeStr},
		{Name: &levelCol, Type: &typeStr},
	}

	rowData := make([]azlogs.Row, 0, len(rows))
	for _, r := range rows {
		row := azlogs.Row{
			r.TimeGenerated,
			r.Source, // can be nil
			r.Message,
			r.Level,
		}
		rowData = append(rowData, row)
	}

	tableName := "PrimaryResult"
	return azlogs.QueryWorkspaceResponse{
		QueryResults: azlogs.QueryResults{
			Tables: []azlogs.Table{
				{
					Name:    &tableName,
					Columns: columns,
					Rows:    rowData,
				},
			},
		},
	}
}

// TestSubstitutePlaceholders_PreservesSourceField tests placeholder substitution doesn't break Source
func TestSubstitutePlaceholders_PreservesSourceField(t *testing.T) {
	testCases := []struct {
		name        string
		query       string
		serviceName string
		timespan    string
		contains    []string
	}{
		{
			name: "Function app query with Source",
			query: `FunctionAppLogs
| where TimeGenerated > ago({timespan})
| where _ResourceId contains '{serviceName}'
| extend Source = "Azure Functions"
| project TimeGenerated, Source, Message`,
			serviceName: "my-func",
			timespan:    "30m",
			contains:    []string{"my-func", "30m", `Source = "Azure Functions"`},
		},
		{
			name: "Container app query with Source",
			query: `ContainerAppConsoleLogs_CL
| where TimeGenerated > ago({timespan})
| where ContainerAppName_s =~ '{serviceName}'
| extend Source = "Azure Container Apps"
| project TimeGenerated, Source, Log_s`,
			serviceName: "my-app",
			timespan:    "1h",
			contains:    []string{"my-app", "1h", `Source = "Azure Container Apps"`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := SubstitutePlaceholders(tc.query, tc.serviceName, tc.timespan)

			// Verify all expected strings are present
			for _, expected := range tc.contains {
				if !containsSubstring(result, expected) {
					t.Errorf("Expected result to contain %q\nGot:\n%s", expected, result)
				}
			}

			// Verify no placeholders remain
			if containsSubstring(result, "{serviceName}") {
				t.Errorf("Query still contains {serviceName} placeholder:\n%s", result)
			}
			if containsSubstring(result, "{timespan}") {
				t.Errorf("Query still contains {timespan} placeholder:\n%s", result)
			}
		})
	}
}

// TestLogEntry_MissingSourceField_DoesNotCrash tests robustness
func TestLogEntry_MissingSourceField_DoesNotCrash(t *testing.T) {
	// Create LogEntry without Source field set
	entry := LogEntry{
		Service:   parseTestService,
		Message:   "Test message",
		Level:     LogLevelInfo,
		Timestamp: time.Now(),
		// Source is intentionally not set
	}

	// Should not crash when accessing Source
	if entry.Source != "" {
		t.Errorf("Expected empty Source, got %q", entry.Source)
	}

	// Should be able to set Source after creation
	entry.Source = sourceAzureFuncs
	if entry.Source != sourceAzureFuncs {
		t.Errorf("Expected Source to be settable, got %q", entry.Source)
	}
}
