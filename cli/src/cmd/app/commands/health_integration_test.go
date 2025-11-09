package commands

import (
	"context"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/healthcheck"
)

func TestRunHealthValidation(t *testing.T) {
	tests := []struct {
		name          string
		interval      time.Duration
		timeout       time.Duration
		output        string
		expectedError string
	}{
		{
			name:          "interval too short",
			interval:      500 * time.Millisecond,
			timeout:       5 * time.Second,
			output:        "text",
			expectedError: "interval must be at least 1 second",
		},
		{
			name:          "timeout too short",
			interval:      5 * time.Second,
			timeout:       500 * time.Millisecond,
			output:        "text",
			expectedError: "timeout must be between 1s and 60s",
		},
		{
			name:          "timeout too long",
			interval:      5 * time.Second,
			timeout:       61 * time.Second,
			output:        "text",
			expectedError: "timeout must be between 1s and 60s",
		},
		{
			name:          "invalid output format",
			interval:      5 * time.Second,
			timeout:       5 * time.Second,
			output:        "xml",
			expectedError: "output must be 'text', 'json', or 'table'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global vars
			healthInterval = tt.interval
			healthTimeout = tt.timeout
			healthOutput = tt.output

			cmd := NewHealthCommand()
			err := cmd.RunE(cmd, []string{})

			if err == nil {
				t.Errorf("Expected error containing '%s', got nil", tt.expectedError)
			} else if err.Error() != tt.expectedError {
				t.Errorf("Expected error '%s', got '%s'", tt.expectedError, err.Error())
			}
		})
	}
}

func TestPerformStreamCheck(t *testing.T) {
	// Create a mock health monitor
	config := healthcheck.MonitorConfig{
		ProjectDir:      t.TempDir(),
		DefaultEndpoint: "/health",
		Timeout:         5 * time.Second,
		Verbose:         false,
	}

	monitor, err := healthcheck.NewHealthMonitor(config)
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	checkCount := 0
	var prevReport *healthcheck.HealthReport

	err = performStreamCheck(
		context.Background(),
		monitor,
		nil,
		&checkCount,
		&prevReport,
		false, // non-TTY mode
	)

	if err != nil {
		t.Errorf("performStreamCheck failed: %v", err)
	}

	if checkCount != 1 {
		t.Errorf("Expected check count 1, got %d", checkCount)
	}

	if prevReport == nil {
		t.Error("Expected prevReport to be set")
	}
}

func TestDisplayHealthReportJSON(t *testing.T) {
	report := &healthcheck.HealthReport{
		Timestamp: time.Now(),
		Project:   "/test",
		Services:  []healthcheck.HealthCheckResult{},
		Summary: healthcheck.HealthSummary{
			Total:   0,
			Overall: healthcheck.HealthStatusUnknown,
		},
	}

	// Save original and set to json
	original := healthOutput
	healthOutput = "json"
	defer func() { healthOutput = original }()

	err := displayHealthReport(report)
	if err != nil {
		t.Errorf("displayHealthReport failed: %v", err)
	}
}

func TestDisplayHealthReportTable(t *testing.T) {
	report := &healthcheck.HealthReport{
		Timestamp: time.Now(),
		Project:   "/test",
		Services: []healthcheck.HealthCheckResult{
			{
				ServiceName:  "test",
				Status:       healthcheck.HealthStatusHealthy,
				CheckType:    healthcheck.HealthCheckTypeHTTP,
				ResponseTime: 50 * time.Millisecond,
			},
		},
		Summary: healthcheck.HealthSummary{
			Total:   1,
			Healthy: 1,
			Overall: healthcheck.HealthStatusHealthy,
		},
	}

	// Save original and set to table
	original := healthOutput
	healthOutput = "table"
	defer func() { healthOutput = original }()

	err := displayHealthReport(report)
	if err != nil {
		t.Errorf("displayHealthReport failed: %v", err)
	}
}

func TestDisplayHealthReportText(t *testing.T) {
	report := &healthcheck.HealthReport{
		Timestamp: time.Now(),
		Project:   "/test",
		Services: []healthcheck.HealthCheckResult{
			{
				ServiceName:  "api",
				Status:       healthcheck.HealthStatusHealthy,
				CheckType:    healthcheck.HealthCheckTypeHTTP,
				Endpoint:     "http://localhost:8080/health",
				ResponseTime: 45 * time.Millisecond,
				StatusCode:   200,
				Port:         8080,
				Uptime:       2 * time.Hour,
			},
		},
		Summary: healthcheck.HealthSummary{
			Total:   1,
			Healthy: 1,
			Overall: healthcheck.HealthStatusHealthy,
		},
	}

	// Save original and set to text
	original := healthOutput
	healthOutput = "text"
	defer func() { healthOutput = original }()

	err := displayHealthReport(report)
	if err != nil {
		t.Errorf("displayHealthReport failed: %v", err)
	}
}

func TestDisplayStreamStatus(t *testing.T) {
	report := &healthcheck.HealthReport{
		Timestamp: time.Now(),
		Services: []healthcheck.HealthCheckResult{
			{
				ServiceName:  "web",
				Status:       healthcheck.HealthStatusHealthy,
				ResponseTime: 30 * time.Millisecond,
				Uptime:       1 * time.Hour,
			},
		},
	}

	// Just test that it doesn't panic
	displayStreamStatus(report, 5)
}

func TestDisplayStreamHeader(t *testing.T) {
	// Save original and set interval
	original := healthInterval
	healthInterval = 5 * time.Second
	defer func() { healthInterval = original }()

	// Just test that it doesn't panic
	displayStreamHeader()
}

func TestDisplayStreamFooter(t *testing.T) {
	// Just test that it doesn't panic
	displayStreamFooter(10)
}

func TestFormatDurationVariations(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"5 seconds", 5 * time.Second, "5s"},
		{"30 seconds", 30 * time.Second, "30s"},
		{"1 minute", 1 * time.Minute, "1m"},
		{"5 minutes", 5 * time.Minute, "5m"},
		{"1 hour 15 minutes", 75 * time.Minute, "1h 15m"},
		{"2 hours", 2 * time.Hour, "2h 0m"},
		{"25 hours", 25 * time.Hour, "1d 1h"},
		{"48 hours", 48 * time.Hour, "2d 0h"},
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

func TestTruncateEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"empty string", "", 5, ""},
		{"exactly max", "hello", 5, "hello"},
		{"one over", "hello!", 5, "he..."},
		{"maxLen 3", "hello", 3, "hel"},
		{"maxLen 1", "hello", 1, "h"},
		{"maxLen 0", "hello", 0, ""},
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
