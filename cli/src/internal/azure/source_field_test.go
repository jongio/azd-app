package azure

import (
	"strings"
	"testing"
	"time"
)

// TestDefaultQueriesContainSource verifies all default queries include Source field
func TestDefaultQueriesContainSource(t *testing.T) {
	resourceTypes := []struct {
		resourceType   ResourceType
		expectedSource string
	}{
		{ResourceTypeFunction, "Azure Functions"},
		{ResourceTypeContainerApp, "Azure Container Apps"},
		{ResourceTypeAppService, "Azure App Service"},
	}

	for _, tc := range resourceTypes {
		t.Run(string(tc.resourceType), func(t *testing.T) {
			query := GetDefaultQuery(tc.resourceType)

			// Verify query is not empty
			if query == "" {
				t.Errorf("Default query for %s should not be empty", tc.resourceType)
				return
			}

			// Verify query contains extend clause with Source
			if !strings.Contains(query, "extend") && !strings.Contains(query, "Source") {
				t.Errorf("Query for %s should contain 'extend' with Source field:\n%s", tc.resourceType, query)
			}

			// Verify query projects Source field
			if !strings.Contains(query, "Source") {
				t.Errorf("Query for %s should project Source field:\n%s", tc.resourceType, query)
			}
		})
	}
}

// TestStandaloneQueryContainsSource verifies standalone query builder includes Source field
func TestStandaloneQueryContainsSource(t *testing.T) {
	testCases := []struct {
		name           string
		resourceType   ResourceType
		services       []string
		expectedSource string
	}{
		{
			name:           "Function App with service",
			resourceType:   ResourceTypeFunction,
			services:       []string{"func-test"},
			expectedSource: "Azure Functions",
		},
		{
			name:           "Function App without service",
			resourceType:   ResourceTypeFunction,
			services:       []string{},
			expectedSource: "Azure Functions",
		},
		{
			name:           "Container App with service",
			resourceType:   ResourceTypeContainerApp,
			services:       []string{"app-test"},
			expectedSource: "Azure Container Apps",
		},
		{
			name:           "App Service with service",
			resourceType:   ResourceTypeAppService,
			services:       []string{"web-test"},
			expectedSource: "Azure App Service",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			query := buildStandaloneQueryForType(tc.resourceType, tc.services, 30*time.Minute, 100)

			// Verify query is not empty
			if query == "" {
				t.Errorf("Standalone query for %s should not be empty", tc.resourceType)
				return
			}

			// Verify query contains extend clause with Source
			if !strings.Contains(query, "| extend Source =") {
				t.Errorf("Standalone query for %s should contain '| extend Source =':\n%s", tc.resourceType, query)
			}

			// Verify query contains expected source value
			if !strings.Contains(query, tc.expectedSource) {
				t.Errorf("Standalone query for %s should contain source %q:\n%s", tc.resourceType, tc.expectedSource, query)
			}

			// Verify query projects Source field
			if !strings.Contains(query, "| project") {
				t.Errorf("Standalone query for %s should contain project clause:\n%s", tc.resourceType, query)
			}

			// Verify Source is in the project list (after extend)
			projectIndex := strings.Index(query, "| project")
			extendIndex := strings.Index(query, "| extend Source =")
			if projectIndex != -1 && extendIndex != -1 {
				if projectIndex < extendIndex {
					t.Errorf("In query for %s, 'project' should come after 'extend Source':\n%s", tc.resourceType, query)
				}

				// Get the project clause
				projectClause := query[projectIndex:]
				if !strings.Contains(projectClause, "Source") {
					t.Errorf("Project clause for %s should include Source field:\n%s", tc.resourceType, projectClause)
				}
			}
		})
	}
}

// TestLogEntrySourceFieldExtraction verifies Source field is extracted from query results
func TestLogEntrySourceFieldExtraction(t *testing.T) {
	testCases := []struct {
		name           string
		row            []any
		colIndex       map[string]int
		resourceType   ResourceType
		expectedSource string
	}{
		{
			name: "Function app with Source field",
			row:  []any{"2024-12-17T10:00:00Z", "Azure Functions", "Test message", "INFO", "TestFunc"},
			colIndex: map[string]int{
				"TimeGenerated": 0,
				"Source":        1,
				"Message":       2,
				"Level":         3,
				"FunctionName":  4,
			},
			resourceType:   ResourceTypeFunction,
			expectedSource: "Azure Functions",
		},
		{
			name: "Container app with Source field",
			row:  []any{"2024-12-17T10:00:00Z", "Azure Container Apps", "Container log", "INFO"},
			colIndex: map[string]int{
				"TimeGenerated": 0,
				"Source":        1,
				"Log_s":         2,
				"Stream_s":      3,
			},
			resourceType:   ResourceTypeContainerApp,
			expectedSource: "Azure Container Apps",
		},
		{
			name: "App Service with Source field",
			row:  []any{"2024-12-17T10:00:00Z", "Azure App Service", "App log", "INFO"},
			colIndex: map[string]int{
				"TimeGenerated":     0,
				"Source":            1,
				"ResultDescription": 2,
				"Level":             3,
			},
			resourceType:   ResourceTypeAppService,
			expectedSource: "Azure App Service",
		},
		{
			name: "No Source field present",
			row:  []any{"2024-12-17T10:00:00Z", "Test message", "INFO"},
			colIndex: map[string]int{
				"TimeGenerated": 0,
				"Message":       1,
				"Level":         2,
			},
			resourceType:   ResourceTypeFunction,
			expectedSource: "", // Should be empty when not present
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Extract Source using getStringFromRow (same logic as parseResults)
			source := getStringFromRow(tc.row, tc.colIndex, "Source")

			if source != tc.expectedSource {
				t.Errorf("Expected Source %q, got %q", tc.expectedSource, source)
			}
		})
	}
}

// TestLogEntryStructHasSourceField verifies LogEntry struct includes Source field
func TestLogEntryStructHasSourceField(t *testing.T) {
	entry := LogEntry{
		Service:       "test-service",
		Message:       "Test message",
		Level:         LogLevelInfo,
		Timestamp:     time.Now(),
		Source:        "Azure Functions",
		ResourceType:  "function",
		ContainerName: "",
		InstanceID:    "instance-123",
	}

	if entry.Source != "Azure Functions" {
		t.Errorf("Expected Source field to be 'Azure Functions', got %q", entry.Source)
	}

	// Verify Source field is accessible and can be updated
	entry.Source = "Azure Container Apps"
	if entry.Source != "Azure Container Apps" {
		t.Errorf("Expected Source field to be updatable, got %q", entry.Source)
	}
}

// TestSourceFieldInQueryResults verifies Source field appears correctly in different query scenarios
func TestSourceFieldInQueryResults(t *testing.T) {
	testCases := []struct {
		name         string
		resourceType ResourceType
		serviceName  string
		query        string
	}{
		{
			name:         "Function App query with service filter",
			resourceType: ResourceTypeFunction,
			serviceName:  "my-function",
			query: `FunctionAppLogs
| where TimeGenerated > ago(30m)
| where _ResourceId contains 'my-function'
| extend Source = "Azure Functions", AzureService = "functions"
| project TimeGenerated, Source, Message, Level, FunctionName
| take 100`,
		},
		{
			name:         "Container App query with service filter",
			resourceType: ResourceTypeContainerApp,
			serviceName:  "my-app",
			query: `ContainerAppConsoleLogs_CL
| where TimeGenerated > ago(30m)
| where ContainerAppName_s contains 'my-app'
| extend Source = "Azure Container Apps", AzureService = "containerapp"
| project TimeGenerated, Source, Log_s, Stream_s
| take 100`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Verify query structure
			lines := strings.Split(tc.query, "\n")

			var hasExtend, hasProject, hasSource bool
			var projectLine string

			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "| extend Source =") {
					hasExtend = true
				}
				if strings.HasPrefix(trimmed, "| project") {
					hasProject = true
					projectLine = trimmed
					if strings.Contains(trimmed, "Source") {
						hasSource = true
					}
				}
			}

			if !hasExtend {
				t.Errorf("Query for %s should have extend clause with Source:\n%s", tc.resourceType, tc.query)
			}

			if !hasProject {
				t.Errorf("Query for %s should have project clause:\n%s", tc.resourceType, tc.query)
			}

			if !hasSource {
				t.Errorf("Query for %s should project Source field. Project line: %s", tc.resourceType, projectLine)
			}
		})
	}
}

// TestSourceFieldConsistency verifies Source values are consistent across query types
func TestSourceFieldConsistency(t *testing.T) {
	testCases := []struct {
		resourceType   ResourceType
		expectedSource string
	}{
		{ResourceTypeFunction, "Azure Functions"},
		{ResourceTypeContainerApp, "Azure Container Apps"},
		{ResourceTypeAppService, "Azure App Service"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.resourceType), func(t *testing.T) {
			// Check default query
			defaultQuery := GetDefaultQuery(tc.resourceType)
			if !strings.Contains(defaultQuery, tc.expectedSource) {
				t.Errorf("Default query for %s should contain source %q", tc.resourceType, tc.expectedSource)
			}

			// Check standalone query
			standaloneQuery := buildStandaloneQueryForType(tc.resourceType, []string{"test"}, 30*time.Minute, 100)
			if !strings.Contains(standaloneQuery, tc.expectedSource) {
				t.Errorf("Standalone query for %s should contain source %q", tc.resourceType, tc.expectedSource)
			}

			// Verify both use the same source value
			if !strings.Contains(defaultQuery, tc.expectedSource) || !strings.Contains(standaloneQuery, tc.expectedSource) {
				t.Errorf("Source value mismatch for %s between default and standalone queries", tc.resourceType)
			}
		})
	}
}
