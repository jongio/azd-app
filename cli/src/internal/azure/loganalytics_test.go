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

func TestExtractWorkspaceID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// GUID only
		{"abc123-def456", "abc123-def456"},
		// Full resource ID
		{"/subscriptions/sub-id/resourceGroups/rg-name/providers/Microsoft.OperationalInsights/workspaces/my-workspace", "my-workspace"},
		// Empty
		{"", ""},
		// Single component
		{"workspace-name", "workspace-name"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := extractWorkspaceID(tc.input)
			if result != tc.expected {
				t.Errorf("extractWorkspaceID(%q) = %q, want %q", tc.input, result, tc.expected)
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
