package healthcheck

import (
	"os"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

// TestPerformBuildTaskHealthCheck_ProcessRunning tests health checks when process is still running.
func TestPerformBuildTaskHealthCheck_ProcessRunning(t *testing.T) {
	checker := &HealthChecker{}

	tests := []struct {
		name                   string
		mode                   string
		isInStartupGracePeriod bool
		expectedStatus         HealthStatus
		expectedState          string
	}{
		{
			name:                   "build mode running during grace period",
			mode:                   service.ServiceModeBuild,
			isInStartupGracePeriod: true,
			expectedStatus:         HealthStatusStarting,
			expectedState:          "building",
		},
		{
			name:                   "build mode running after grace period",
			mode:                   service.ServiceModeBuild,
			isInStartupGracePeriod: false,
			expectedStatus:         HealthStatusHealthy,
			expectedState:          "building",
		},
		{
			name:                   "task mode running during grace period",
			mode:                   service.ServiceModeTask,
			isInStartupGracePeriod: true,
			expectedStatus:         HealthStatusStarting,
			expectedState:          "running",
		},
		{
			name:                   "task mode running after grace period",
			mode:                   service.ServiceModeTask,
			isInStartupGracePeriod: false,
			expectedStatus:         HealthStatusHealthy,
			expectedState:          "running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use current process PID since it's always running
			currentPID := getCurrentPID()

			svc := serviceInfo{
				Name: "test-service",
				PID:  currentPID,
				Mode: tt.mode,
			}

			result := HealthCheckResult{
				ServiceName: svc.Name,
				Timestamp:   time.Now(),
				CheckType:   HealthCheckTypeProcess,
				ServiceMode: svc.Mode,
			}

			result = checker.performBuildTaskHealthCheck(svc, tt.isInStartupGracePeriod, result)

			if result.Status != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s", tt.expectedStatus, result.Status)
			}

			if state, ok := result.Details["state"].(string); !ok || state != tt.expectedState {
				t.Errorf("Expected state '%s', got '%v'", tt.expectedState, result.Details["state"])
			}

			if result.PID != currentPID {
				t.Errorf("Expected PID %d, got %d", currentPID, result.PID)
			}
		})
	}
}

// TestPerformBuildTaskHealthCheck_ProcessExitedWithCode tests health checks when process has exited with recorded exit code.
func TestPerformBuildTaskHealthCheck_ProcessExitedWithCode(t *testing.T) {
	checker := &HealthChecker{}

	tests := []struct {
		name           string
		mode           string
		exitCode       int
		expectedStatus HealthStatus
		expectedState  string
	}{
		{
			name:           "build mode exited successfully",
			mode:           service.ServiceModeBuild,
			exitCode:       0,
			expectedStatus: HealthStatusHealthy,
			expectedState:  "built",
		},
		{
			name:           "build mode exited with error",
			mode:           service.ServiceModeBuild,
			exitCode:       1,
			expectedStatus: HealthStatusUnhealthy,
			expectedState:  "failed",
		},
		{
			name:           "task mode exited successfully",
			mode:           service.ServiceModeTask,
			exitCode:       0,
			expectedStatus: HealthStatusHealthy,
			expectedState:  "completed",
		},
		{
			name:           "task mode exited with error",
			mode:           service.ServiceModeTask,
			exitCode:       1,
			expectedStatus: HealthStatusUnhealthy,
			expectedState:  "failed",
		},
		{
			name:           "task mode exited with error code 2",
			mode:           service.ServiceModeTask,
			exitCode:       2,
			expectedStatus: HealthStatusUnhealthy,
			expectedState:  "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exitCode := tt.exitCode
			svc := serviceInfo{
				Name:     "test-service",
				PID:      99999, // Use non-existent PID
				Mode:     tt.mode,
				ExitCode: &exitCode,
			}

			result := HealthCheckResult{
				ServiceName: svc.Name,
				Timestamp:   time.Now(),
				CheckType:   HealthCheckTypeProcess,
				ServiceMode: svc.Mode,
			}

			// Grace period shouldn't matter when exit code is recorded
			result = checker.performBuildTaskHealthCheck(svc, true, result)

			if result.Status != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s", tt.expectedStatus, result.Status)
			}

			if state, ok := result.Details["state"].(string); !ok || state != tt.expectedState {
				t.Errorf("Expected state '%s', got '%v'", tt.expectedState, result.Details["state"])
			}

			if ec, ok := result.Details["exitCode"].(int); !ok || ec != tt.exitCode {
				t.Errorf("Expected exitCode %d, got '%v'", tt.exitCode, result.Details["exitCode"])
			}
		})
	}
}

// TestPerformBuildTaskHealthCheck_ProcessExitedNoCode_WithPID tests the critical case:
// Process has exited (has PID but not running), but exit code hasn't been captured yet.
// This is the scenario that was causing services to be stuck in "starting" status.
func TestPerformBuildTaskHealthCheck_ProcessExitedNoCode_WithPID(t *testing.T) {
	checker := &HealthChecker{}

	tests := []struct {
		name                   string
		mode                   string
		isInStartupGracePeriod bool
		expectedStatus         HealthStatus
		expectedState          string
	}{
		{
			name:                   "task mode exited quickly - during grace period",
			mode:                   service.ServiceModeTask,
			isInStartupGracePeriod: true,
			expectedStatus:         HealthStatusHealthy, // Should NOT be "starting"
			expectedState:          "completed",
		},
		{
			name:                   "task mode exited quickly - after grace period",
			mode:                   service.ServiceModeTask,
			isInStartupGracePeriod: false,
			expectedStatus:         HealthStatusHealthy,
			expectedState:          "completed",
		},
		{
			name:                   "build mode exited quickly - during grace period",
			mode:                   service.ServiceModeBuild,
			isInStartupGracePeriod: true,
			expectedStatus:         HealthStatusHealthy, // Should NOT be "starting"
			expectedState:          "built",
		},
		{
			name:                   "build mode exited quickly - after grace period",
			mode:                   service.ServiceModeBuild,
			isInStartupGracePeriod: false,
			expectedStatus:         HealthStatusHealthy,
			expectedState:          "built",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a non-existent PID to simulate exited process
			svc := serviceInfo{
				Name:     "test-service",
				PID:      99999, // Non-existent PID
				Mode:     tt.mode,
				ExitCode: nil, // Exit code not captured yet
			}

			result := HealthCheckResult{
				ServiceName: svc.Name,
				Timestamp:   time.Now(),
				CheckType:   HealthCheckTypeProcess,
				ServiceMode: svc.Mode,
			}

			result = checker.performBuildTaskHealthCheck(svc, tt.isInStartupGracePeriod, result)

			if result.Status != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s (this is the critical regression case)", tt.expectedStatus, result.Status)
			}

			if state, ok := result.Details["state"].(string); !ok || state != tt.expectedState {
				t.Errorf("Expected state '%s', got '%v'", tt.expectedState, result.Details["state"])
			}

			// Should have the "exit code not captured" note
			if note, ok := result.Details["note"].(string); !ok || note != "exit code not captured" {
				t.Errorf("Expected note 'exit code not captured', got '%v'", result.Details["note"])
			}
		})
	}
}

// TestPerformBuildTaskHealthCheck_NoPID tests health checks when process hasn't started yet (no PID).
func TestPerformBuildTaskHealthCheck_NoPID(t *testing.T) {
	checker := &HealthChecker{}

	tests := []struct {
		name                   string
		mode                   string
		isInStartupGracePeriod bool
		expectedStatus         HealthStatus
	}{
		{
			name:                   "no PID during grace period",
			mode:                   service.ServiceModeTask,
			isInStartupGracePeriod: true,
			expectedStatus:         HealthStatusStarting,
		},
		{
			name:                   "no PID after grace period",
			mode:                   service.ServiceModeTask,
			isInStartupGracePeriod: false,
			expectedStatus:         HealthStatusUnknown,
		},
		{
			name:                   "build mode no PID during grace period",
			mode:                   service.ServiceModeBuild,
			isInStartupGracePeriod: true,
			expectedStatus:         HealthStatusStarting,
		},
		{
			name:                   "build mode no PID after grace period",
			mode:                   service.ServiceModeBuild,
			isInStartupGracePeriod: false,
			expectedStatus:         HealthStatusUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := serviceInfo{
				Name:     "test-service",
				PID:      0, // No PID assigned yet
				Mode:     tt.mode,
				ExitCode: nil,
			}

			result := HealthCheckResult{
				ServiceName: svc.Name,
				Timestamp:   time.Now(),
				CheckType:   HealthCheckTypeProcess,
				ServiceMode: svc.Mode,
			}

			result = checker.performBuildTaskHealthCheck(svc, tt.isInStartupGracePeriod, result)

			if result.Status != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s", tt.expectedStatus, result.Status)
			}
		})
	}
}

// TestPerformBuildTaskHealthCheck_RegressionQuicklyExitingTask is a specific regression test
// for the bug where a task mode process (like data-processor) that completes in ~2.5 seconds
// would stay stuck in "starting" status for up to 30 seconds.
func TestPerformBuildTaskHealthCheck_RegressionQuicklyExitingTask(t *testing.T) {
	checker := &HealthChecker{}

	// Simulate a task that:
	// 1. Started and got a PID
	// 2. Exited quickly (~2.5 seconds)
	// 3. Exit code not captured yet (race condition)
	// 4. Still within 30 second startup grace period

	svc := serviceInfo{
		Name:      "data-processor",
		PID:       99999, // Non-existent PID (process exited)
		Mode:      service.ServiceModeTask,
		ExitCode:  nil,                              // Exit code not captured yet
		StartTime: time.Now().Add(-3 * time.Second), // Started 3 seconds ago
	}

	result := HealthCheckResult{
		ServiceName: svc.Name,
		Timestamp:   time.Now(),
		CheckType:   HealthCheckTypeProcess,
		ServiceMode: svc.Mode,
	}

	// This is the critical test: within startup grace period (30 seconds)
	isInStartupGracePeriod := true

	result = checker.performBuildTaskHealthCheck(svc, isInStartupGracePeriod, result)

	// REGRESSION CHECK: Should NOT be "starting" - should be "healthy" with "completed" state
	if result.Status == HealthStatusStarting {
		t.Errorf("REGRESSION: Task mode process with PID that exited should NOT be 'starting', got %s", result.Status)
		t.Errorf("This bug caused task services to be stuck in 'starting' for up to 30 seconds")
	}

	if result.Status != HealthStatusHealthy {
		t.Errorf("Expected status healthy, got %s", result.Status)
	}

	if state, ok := result.Details["state"].(string); !ok || state != "completed" {
		t.Errorf("Expected state 'completed', got '%v'", result.Details["state"])
	}
}

// TestPerformBuildTaskHealthCheck_RegressionQuicklyExitingBuild is similar to the task test
// but for build mode services.
func TestPerformBuildTaskHealthCheck_RegressionQuicklyExitingBuild(t *testing.T) {
	checker := &HealthChecker{}

	svc := serviceInfo{
		Name:      "cli-build",
		PID:       99999,
		Mode:      service.ServiceModeBuild,
		ExitCode:  nil,
		StartTime: time.Now().Add(-5 * time.Second),
	}

	result := HealthCheckResult{
		ServiceName: svc.Name,
		Timestamp:   time.Now(),
		CheckType:   HealthCheckTypeProcess,
		ServiceMode: svc.Mode,
	}

	isInStartupGracePeriod := true

	result = checker.performBuildTaskHealthCheck(svc, isInStartupGracePeriod, result)

	if result.Status == HealthStatusStarting {
		t.Errorf("REGRESSION: Build mode process with PID that exited should NOT be 'starting', got %s", result.Status)
	}

	if result.Status != HealthStatusHealthy {
		t.Errorf("Expected status healthy, got %s", result.Status)
	}

	if state, ok := result.Details["state"].(string); !ok || state != "built" {
		t.Errorf("Expected state 'built', got '%v'", result.Details["state"])
	}
}

// getCurrentPID returns the current process PID for testing with a running process.
func getCurrentPID() int {
	return os.Getpid()
}
