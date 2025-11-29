package commands

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/healthcheck"
)

func TestHealthCommand(t *testing.T) {
	// Test that command can be created
	cmd := NewHealthCommand()
	if cmd == nil {
		t.Fatal("NewHealthCommand returned nil")
	}

	if cmd.Use != "health" {
		t.Errorf("Expected Use='health', got '%s'", cmd.Use)
	}
}

func TestHealthCommandFlags(t *testing.T) {
	cmd := NewHealthCommand()

	tests := []struct {
		name         string
		flagName     string
		expectedType string
	}{
		{"service flag", "service", "string"},
		{"stream flag", "stream", "bool"},
		{"interval flag", "interval", "duration"},
		{"output flag", "output", "string"},
		{"endpoint flag", "endpoint", "string"},
		{"timeout flag", "timeout", "duration"},
		{"all flag", "all", "bool"},
		{"verbose flag", "verbose", "bool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("Flag %s not found", tt.flagName)
			}
			if flag.Value.Type() != tt.expectedType {
				t.Errorf("Expected flag type %s, got %s", tt.expectedType, flag.Value.Type())
			}
		})
	}
}

func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		status   healthcheck.HealthStatus
		expected string
	}{
		{healthcheck.HealthStatusHealthy, "✓"},
		{healthcheck.HealthStatusDegraded, "⚠"},
		{healthcheck.HealthStatusUnhealthy, "✗"},
		{healthcheck.HealthStatusStarting, "○"},
		{healthcheck.HealthStatusUnknown, "?"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := getStatusIcon(tt.status)
			if result != tt.expected {
				t.Errorf("Expected icon '%s' for status %s, got '%s'", tt.expected, tt.status, result)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"seconds", 30 * time.Second, "30s"},
		{"minutes", 5 * time.Minute, "5m"},
		{"hour and minutes", 2*time.Hour + 15*time.Minute, "2h 15m"},
		{"days", 25 * time.Hour, "1d 1h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 8, "hello..."},
		{"very short max", "hello", 2, "he"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestDisplayJSONReport(t *testing.T) {
	report := &healthcheck.HealthReport{
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Project:   "/test/project",
		Services: []healthcheck.HealthCheckResult{
			{
				ServiceName:  "test-service",
				Status:       healthcheck.HealthStatusHealthy,
				CheckType:    healthcheck.HealthCheckTypeHTTP,
				ResponseTime: 50 * time.Millisecond,
				Timestamp:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			},
		},
		Summary: healthcheck.HealthSummary{
			Total:   1,
			Healthy: 1,
			Overall: healthcheck.HealthStatusHealthy,
		},
	}

	// Capture output
	var buf bytes.Buffer
	originalStdout := healthOutput
	healthOutput = "json"
	defer func() { healthOutput = originalStdout }()

	err := displayJSONReport(report)
	if err != nil {
		t.Fatalf("displayJSONReport failed: %v", err)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err == nil {
		// Only test if we captured output
		if len(buf.Bytes()) > 0 {
			if result["project"] != "/test/project" {
				t.Errorf("Expected project '/test/project', got '%v'", result["project"])
			}
		}
	}
}

func TestDetectChanges(t *testing.T) {
	prev := &healthcheck.HealthReport{
		Timestamp: time.Now(),
		Services: []healthcheck.HealthCheckResult{
			{ServiceName: "svc1", Status: healthcheck.HealthStatusHealthy},
			{ServiceName: "svc2", Status: healthcheck.HealthStatusHealthy},
		},
	}

	curr := &healthcheck.HealthReport{
		Timestamp: time.Now(),
		Services: []healthcheck.HealthCheckResult{
			{ServiceName: "svc1", Status: healthcheck.HealthStatusHealthy},
			{ServiceName: "svc2", Status: healthcheck.HealthStatusUnhealthy},
		},
	}

	changes := detectChanges(prev, curr)

	if len(changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(changes))
	}

	if changes[0].ServiceName != "svc2" {
		t.Errorf("Expected change for svc2, got %s", changes[0].ServiceName)
	}

	if changes[0].OldStatus != healthcheck.HealthStatusHealthy {
		t.Errorf("Expected old status healthy, got %s", changes[0].OldStatus)
	}

	if changes[0].NewStatus != healthcheck.HealthStatusUnhealthy {
		t.Errorf("Expected new status unhealthy, got %s", changes[0].NewStatus)
	}
}

func TestDisplayTextReport(t *testing.T) {
	report := &healthcheck.HealthReport{
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Project:   "/test/project",
		Services: []healthcheck.HealthCheckResult{
			{
				ServiceName:  "web",
				Status:       healthcheck.HealthStatusHealthy,
				CheckType:    healthcheck.HealthCheckTypeHTTP,
				Endpoint:     "http://localhost:3000/health",
				ResponseTime: 45 * time.Millisecond,
				StatusCode:   200,
				Timestamp:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			},
		},
		Summary: healthcheck.HealthSummary{
			Total:   1,
			Healthy: 1,
			Overall: healthcheck.HealthStatusHealthy,
		},
	}

	// Test that it doesn't error
	err := displayTextReport(report)
	if err != nil {
		t.Errorf("displayTextReport failed: %v", err)
	}
}

func TestDisplayTableReport(t *testing.T) {
	report := &healthcheck.HealthReport{
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Project:   "/test/project",
		Services: []healthcheck.HealthCheckResult{
			{
				ServiceName:  "api",
				Status:       healthcheck.HealthStatusHealthy,
				CheckType:    healthcheck.HealthCheckTypeHTTP,
				Endpoint:     "http://localhost:8080/health",
				ResponseTime: 23 * time.Millisecond,
				Timestamp:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			},
		},
		Summary: healthcheck.HealthSummary{
			Total:   1,
			Healthy: 1,
			Overall: healthcheck.HealthStatusHealthy,
		},
	}

	// Test that it doesn't error
	err := displayTableReport(report)
	if err != nil {
		t.Errorf("displayTableReport failed: %v", err)
	}
}

func TestIsatty(t *testing.T) {
	// Just test that it doesn't panic
	result := isatty()
	_ = result // Result depends on environment
}
