package azure

import (
	"testing"
	"time"
)

func TestFormatTimespan(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{1 * time.Hour, "PT1H0M0S"},
		{30 * time.Minute, "PT30M0S"},
		{45 * time.Second, "PT45S"},
		{2*time.Hour + 30*time.Minute + 15*time.Second, "PT2H30M15S"},
		{0, "PT0S"},
	}

	for _, tc := range tests {
		t.Run(tc.duration.String(), func(t *testing.T) {
			result := formatTimespan(tc.duration)
			if result != tc.expected {
				t.Errorf("formatTimespan(%v) = %q, want %q", tc.duration, result, tc.expected)
			}
		})
	}
}

func TestGetStringFromRow(t *testing.T) {
	row := []any{"value1", "value2", nil, "value4"}
	colIndex := map[string]int{
		"col1": 0,
		"col2": 1,
		"col3": 2,
		"col4": 3,
	}

	tests := []struct {
		columns  []string
		expected string
	}{
		{[]string{"col1"}, "value1"},
		{[]string{"col2", "col1"}, "value2"},
		{[]string{"col3", "col4"}, "value4"},
		{[]string{"nonexistent"}, ""},
		{[]string{"col3"}, ""}, // nil value
	}

	for _, tc := range tests {
		result := getStringFromRow(row, colIndex, tc.columns...)
		if result != tc.expected {
			t.Errorf("getStringFromRow with columns %v = %q, want %q", tc.columns, result, tc.expected)
		}
	}
}

func TestGetDefaultQuery(t *testing.T) {
	// Test that default queries exist for each resource type
	types := []ResourceType{
		ResourceTypeContainerApp,
		ResourceTypeAppService,
		ResourceTypeFunction,
		ResourceTypeAKS,
		ResourceTypeContainerInstance,
	}

	for _, rt := range types {
		query := GetDefaultQuery(rt)
		if query == "" {
			t.Errorf("No default query for resource type %v", rt)
		}
		// Verify placeholder is present
		if rt != ResourceTypeUnknown && !containsPlaceholder(query, "{serviceName}") {
			t.Errorf("Default query for %v missing {serviceName} placeholder", rt)
		}
	}
}

func containsPlaceholder(s, placeholder string) bool {
	return len(s) > 0 && (s == "" || indexOf(s, placeholder) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func TestLogLevel(t *testing.T) {
	// Test that log levels have expected values
	if LogLevelInfo != 0 {
		t.Error("LogLevelInfo should be 0")
	}
	if LogLevelWarn != 1 {
		t.Error("LogLevelWarn should be 1")
	}
	if LogLevelError != 2 {
		t.Error("LogLevelError should be 2")
	}
	if LogLevelDebug != 3 {
		t.Error("LogLevelDebug should be 3")
	}
}

func TestLogEntryStruct(t *testing.T) {
	entry := LogEntry{
		Service:       "api",
		Message:       "Test log message",
		Level:         LogLevelInfo,
		Timestamp:     time.Now(),
		ResourceType:  "containerApp",
		ContainerName: "api-container",
		InstanceID:    "instance-123",
	}

	if entry.Service != "api" {
		t.Errorf("Expected Service 'api', got %q", entry.Service)
	}
	if entry.Level != LogLevelInfo {
		t.Errorf("Expected LogLevelInfo, got %v", entry.Level)
	}
}

func TestExtractMessageForFunctionApp(t *testing.T) {
	// Create a client for testing
	client := &LogAnalyticsClient{}

	tests := []struct {
		name     string
		row      []any
		colIndex map[string]int
		expected string
	}{
		{
			name: "function app with function name and message",
			row:  []any{"MyFunction", "User logged in successfully", "Host.Function"},
			colIndex: map[string]int{
				"FunctionName": 0,
				"Message":      1,
				"Category":     2,
			},
			expected: "[MyFunction] User logged in successfully",
		},
		{
			name: "function app with category only (no function name)",
			row:  []any{"", "Starting host", "Host.Startup"},
			colIndex: map[string]int{
				"FunctionName": 0,
				"Message":      1,
				"Category":     2,
			},
			expected: "[Host.Startup] Starting host",
		},
		{
			name: "function app with message only",
			row:  []any{"", "Simple message", ""},
			colIndex: map[string]int{
				"FunctionName": 0,
				"Message":      1,
				"Category":     2,
			},
			expected: "Simple message",
		},
		{
			name: "function app with empty message",
			row:  []any{"MyFunction", "", "Host.Function"},
			colIndex: map[string]int{
				"FunctionName": 0,
				"Message":      1,
				"Category":     2,
			},
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := client.extractMessage(tc.row, tc.colIndex, ResourceTypeFunction)
			if result != tc.expected {
				t.Errorf("extractMessage() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestFunctionAppDefaultQuery(t *testing.T) {
	query := GetDefaultQuery(ResourceTypeFunction)

	// Verify query is not empty
	if query == "" {
		t.Error("Function App default query should not be empty")
	}

	// Verify query contains expected tables
	if !containsSubstring(query, "FunctionAppLogs") {
		t.Error("Function App query should query FunctionAppLogs table")
	}

	// Verify query contains expected fields
	expectedFields := []string{"Message", "Level", "FunctionName"}
	for _, field := range expectedFields {
		if !containsSubstring(query, field) {
			t.Errorf("Function App query should contain field: %s", field)
		}
	}

	// Verify query contains placeholders
	if !containsSubstring(query, "{serviceName}") {
		t.Error("Function App query should contain {serviceName} placeholder")
	}
	if !containsSubstring(query, "{timespan}") {
		t.Error("Function App query should contain {timespan} placeholder")
	}
}

func containsSubstring(s, substr string) bool {
	return indexOf(s, substr) >= 0
}
