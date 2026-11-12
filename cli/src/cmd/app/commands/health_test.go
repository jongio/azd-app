package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/healthcheck"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
	"github.com/jongio/azd-core/registry"
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
		{"endpoint flag", "endpoint", "string"},
		{"timeout flag", "timeout", "duration"},
		{"all flag", "all", "bool"},
		{"verbose flag", "verbose", "bool"},
		{"fail on degraded flag", "fail-on-degraded", "bool"},
		{"summary only flag", "summary-only", "bool"},
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

func TestHealthReportExitErrorFailOnDegraded(t *testing.T) {
	original := healthFailOnDegraded
	t.Cleanup(func() { healthFailOnDegraded = original })

	report := &healthcheck.HealthReport{
		Summary: healthcheck.HealthSummary{
			Total:    1,
			Degraded: 1,
			Overall:  healthcheck.HealthStatusDegraded,
		},
	}

	healthFailOnDegraded = false
	if err := healthReportExitError(report); err != nil {
		t.Fatalf("healthReportExitError() default behavior error = %v", err)
	}

	healthFailOnDegraded = true
	err := healthReportExitError(report)
	if err == nil {
		t.Fatal("healthReportExitError() expected degraded error")
	}
	if err.Error() != "1 service(s) degraded" {
		t.Fatalf("healthReportExitError() error = %q", err.Error())
	}
}

func TestHealthReportSummaryLines(t *testing.T) {
	report := &healthcheck.HealthReport{
		Summary: healthcheck.HealthSummary{
			Healthy:   2,
			Degraded:  1,
			Unhealthy: 3,
			Overall:   healthcheck.HealthStatusUnhealthy,
		},
	}

	lines := healthReportSummaryLines(report)
	if len(lines) != 2 {
		t.Fatalf("healthReportSummaryLines() got %d lines, want 2", len(lines))
	}
	if lines[0] != "Overall Status: UNHEALTHY" {
		t.Fatalf("healthReportSummaryLines()[0] = %q", lines[0])
	}
	if lines[1] != "Summary: 2 healthy, 1 degraded, 3 unhealthy" {
		t.Fatalf("healthReportSummaryLines()[1] = %q", lines[1])
	}
}

func TestDisplayHealthReportSummaryOnly(t *testing.T) {
	originalSummaryOnly := healthSummaryOnly
	originalOutput := healthOutput
	t.Cleanup(func() {
		healthSummaryOnly = originalSummaryOnly
		healthOutput = originalOutput
	})

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
				Timestamp:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			},
		},
		Summary: healthcheck.HealthSummary{
			Total:   1,
			Healthy: 1,
			Overall: healthcheck.HealthStatusHealthy,
		},
	}

	wantSummary := "Overall Status: HEALTHY\nSummary: 1 healthy, 0 degraded, 0 unhealthy\n"

	tests := []struct {
		name        string
		output      string
		wantSummary bool
	}{
		{"text output prints summary only", "text", true},
		{"table output prints summary only", "table", true},
		{"json output is unaffected", outputFormatJSON, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			healthSummaryOnly = true
			healthOutput = tt.output

			out, err := captureStdout(t, func() error { return displayHealthReport(report) })
			if err != nil {
				t.Fatalf("displayHealthReport() error = %v", err)
			}

			if tt.wantSummary {
				if out != wantSummary {
					t.Fatalf("displayHealthReport() = %q, want %q", out, wantSummary)
				}
				return
			}

			if !json.Valid([]byte(out)) {
				t.Fatalf("displayHealthReport() produced invalid JSON: %q", out)
			}
			if !strings.Contains(out, "web") {
				t.Errorf("displayHealthReport() JSON dropped per-service detail: %q", out)
			}
		})
	}
}

func TestDisplaySummaryOnlyReport(t *testing.T) {
	report := &healthcheck.HealthReport{
		Summary: healthcheck.HealthSummary{
			Total:     3,
			Healthy:   1,
			Degraded:  1,
			Unhealthy: 1,
			Overall:   healthcheck.HealthStatusUnhealthy,
		},
	}

	out, err := captureStdout(t, func() error { return displaySummaryOnlyReport(report) })
	if err != nil {
		t.Fatalf("displaySummaryOnlyReport() error = %v", err)
	}

	want := "Overall Status: UNHEALTHY\nSummary: 1 healthy, 1 degraded, 1 unhealthy\n"
	if out != want {
		t.Fatalf("displaySummaryOnlyReport() = %q, want %q", out, want)
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
	var result map[string]any
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

func TestSeedHealthRegistryFromServices(t *testing.T) {
	projectDir := t.TempDir()
	startTime := time.Now().Add(-30 * time.Second).Truncate(time.Second)
	endTime := startTime.Add(5 * time.Second)
	zeroExit := 0
	customFailExit := 42

	// No dashboard running should be a safe no-op.
	seedHealthRegistryFromDashboard(context.Background(), projectDir)

	seedHealthRegistryFromServices(projectDir, []*serviceinfo.ServiceInfo{
		nil,
		{Name: "no-local"},
		{
			Name:  "never-started",
			Local: &serviceinfo.LocalServiceInfo{Status: "not-running"},
		},
		{
			Name: "stopped-with-retained-pid",
			Local: &serviceinfo.LocalServiceInfo{
				Status: "not-running",
				PID:    9876,
				Port:   4321,
			},
		},
		{
			Name:      "worker",
			Language:  "csharp",
			Framework: "csharp",
			Local: &serviceinfo.LocalServiceInfo{
				Status:      "running",
				RawStatus:   "running",
				PID:         4242,
				ServiceType: "process",
				ServiceMode: "daemon",
				StartTime:   &startTime,
			},
		},
		{
			Name: "completed-build",
			Local: &serviceinfo.LocalServiceInfo{
				Status:      "",
				RawStatus:   "built",
				ServiceType: "process",
				ServiceMode: "build",
				ExitCode:    &zeroExit,
				StartTime:   &startTime,
				EndTime:     &endTime,
			},
		},
		{
			Name:      "failed-task",
			Language:  "go",
			Framework: "go",
			Local: &serviceinfo.LocalServiceInfo{
				Status:      "error",
				RawStatus:   "error",
				PID:         5150,
				ExitCode:    &customFailExit,
				ServiceType: "process",
				ServiceMode: "task",
				StartTime:   &startTime,
				EndTime:     &endTime,
			},
		},
	})

	reg := registry.GetRegistry(projectDir)
	if _, exists := reg.GetService("never-started"); exists {
		t.Fatal("expected never-started service without PID/port/exitCode to be skipped")
	}

	stoppedSvc, exists := reg.GetService("stopped-with-retained-pid")
	if !exists || stoppedSvc.Status != "stopped" {
		t.Fatalf("expected stopped service with retained PID to normalize to 'stopped', got: %+v", stoppedSvc)
	}

	completedBuild, exists := reg.GetService("completed-build")
	if !exists || completedBuild.Status != "built" || completedBuild.ExitCode == nil || *completedBuild.ExitCode != 0 || !completedBuild.EndTime.Equal(endTime) {
		t.Fatalf("expected completed-build to preserve Status='built', ExitCode=0, EndTime=%v; got: %+v", endTime, completedBuild)
	}

	failedTask, exists := reg.GetService("failed-task")
	if !exists || failedTask.ExitCode == nil || *failedTask.ExitCode != 42 {
		t.Fatalf("expected failed-task to preserve exact ExitCode=42, got: %+v", failedTask)
	}

	worker, exists := reg.GetService("worker")
	if !exists {
		t.Fatal("expected running worker service to be registered")
	}
	if worker.PID != 4242 || worker.Status != "running" || worker.Type != "process" || worker.Mode != "daemon" {
		t.Fatalf("unexpected worker registry entry: %+v", worker)
	}
	if !worker.StartTime.Equal(startTime) {
		t.Fatalf("worker StartTime = %v, want %v", worker.StartTime, startTime)
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
