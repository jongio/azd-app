package portmanager

import (
	"testing"
	"time"
)

func TestReleasePort(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	// Assign port
	port, _, err := pm.AssignPort("test-service", 9881, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify assignment exists
	if _, exists := pm.GetAssignment("test-service"); !exists {
		t.Fatal("Expected assignment to exist")
	}

	// Release port
	if err := pm.ReleasePort("test-service"); err != nil {
		t.Errorf("failed to release port: %v", err)
	}

	// Verify assignment is gone
	if _, exists := pm.GetAssignment("test-service"); exists {
		t.Error("Expected assignment to be released")
	}

	// Verify can assign same port to different service
	newPort, _, err := pm.AssignPort("other-service", port, false)
	if err != nil {
		t.Fatalf("Expected no error after release, got: %v", err)
	}

	if newPort != port {
		t.Errorf("Expected to reuse released port %d, got %d", port, newPort)
	}
}

func TestReleasePort_NonExistent(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	// Release a port that was never assigned (should not panic)
	if err := pm.ReleasePort("nonexistent-service"); err != nil {
		t.Errorf("ReleasePort should not error for non-existent service: %v", err)
	}

	// Verify no crash and state is consistent
	if _, exists := pm.GetAssignment("nonexistent-service"); exists {
		t.Error("Expected nonexistent service to not have assignment")
	}
}

func TestReleasePort_UpdatesFile(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	// Assign and release
	if _, _, err := pm.AssignPort("temp-service", 9906, false); err != nil {
		t.Fatalf("failed to assign port: %v", err)
	}

	// Verify assignment exists
	if _, exists := pm.GetAssignment("temp-service"); !exists {
		t.Fatal("Expected assignment to exist")
	}

	// Release
	err := pm.ReleasePort("temp-service")
	if err != nil {
		t.Fatalf("Failed to release port: %v", err)
	}

	// Verify assignment removed
	if _, exists := pm.GetAssignment("temp-service"); exists {
		t.Error("Expected assignment to be removed")
	}
}

func TestCleanStalePorts_VeryOldAssignment(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	// Create a very old assignment (8 days ago, beyond 7-day threshold)
	pm.mu.Lock()
	pm.assignments["ancient-service"] = &PortAssignment{
		ServiceName: "ancient-service",
		Port:        9891,
		LastUsed:    time.Now().Add(-8 * 24 * time.Hour),
	}
	if err := pm.save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}
	pm.mu.Unlock()

	// Clean stale ports
	if err := pm.CleanStalePorts(); err != nil {
		t.Fatalf("CleanStalePorts failed: %v", err)
	}

	// Very old assignment should be cleaned
	if _, exists := pm.GetAssignment("ancient-service"); exists {
		t.Error("Expected ancient assignment to be cleaned")
	}
}

func TestPortInUseError(t *testing.T) {
	tests := []struct {
		name     string
		err      *PortInUseError
		expected string
	}{
		{
			name: "with process name",
			err: &PortInUseError{
				Port:        8080,
				PID:         1234,
				ProcessName: "nginx",
				ServiceName: "web-service",
			},
			expected: "port 8080 required by service 'web-service' is in use by nginx (PID 1234)",
		},
		{
			name: "without process name",
			err: &PortInUseError{
				Port:        3000,
				PID:         5678,
				ProcessName: "",
				ServiceName: "api-service",
			},
			expected: "port 3000 required by service 'api-service' is in use by PID 5678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("PortInUseError.Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPortRangeExhaustedError(t *testing.T) {
	err := &PortRangeExhaustedError{
		StartPort: 3000,
		EndPort:   4000,
	}

	expected := "no available ports in range 3000-4000"
	got := err.Error()

	if got != expected {
		t.Errorf("PortRangeExhaustedError.Error() = %q, want %q", got, expected)
	}
}

func TestInvalidPortError(t *testing.T) {
	err := &InvalidPortError{
		Port:   100,
		Reason: "port outside valid range",
	}

	expected := "invalid port 100: port outside valid range"
	got := err.Error()

	if got != expected {
		t.Errorf("InvalidPortError.Error() = %q, want %q", got, expected)
	}
}
