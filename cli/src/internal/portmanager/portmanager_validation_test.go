package portmanager

import (
	"strings"
	"testing"
)

func TestAssignPort_ValidationErrors(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	tests := []struct {
		name         string
		serviceName  string
		port         int
		isExplicit   bool
		wantErr      bool
		errSubstring string
	}{
		{
			name:         "empty service name",
			serviceName:  "",
			port:         3000,
			isExplicit:   false,
			wantErr:      true,
			errSubstring: "serviceName cannot be empty",
		},
		{
			name:         "explicit port zero",
			serviceName:  "test-service",
			port:         0,
			isExplicit:   true,
			wantErr:      true,
			errSubstring: "explicit port must be between 1-65535",
		},
		{
			name:         "explicit port negative",
			serviceName:  "test-service",
			port:         -1,
			isExplicit:   true,
			wantErr:      true,
			errSubstring: "explicit port must be between 1-65535",
		},
		{
			name:         "explicit port too high",
			serviceName:  "test-service",
			port:         65536,
			isExplicit:   true,
			wantErr:      true,
			errSubstring: "explicit port must be between 1-65535",
		},
		{
			name:        "flexible port zero is okay",
			serviceName: "test-service",
			port:        0,
			isExplicit:  false,
			wantErr:     false,
		},
		{
			name:        "valid explicit port",
			serviceName: "test-service",
			port:        8080,
			isExplicit:  true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port, _, err := pm.AssignPort(tt.serviceName, tt.port, tt.isExplicit)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("Expected error, got port=%d", port)
				}
				if tt.errSubstring != "" && !strings.Contains(err.Error(), tt.errSubstring) {
					t.Errorf("Error %q does not contain %q", err.Error(), tt.errSubstring)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if port <= 0 {
					t.Errorf("Expected valid port, got %d", port)
				}
			}
		})
	}
}

func TestAssignPort_ServiceNameWithSpecialChars(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	// Test that service names are properly handled in JSON
	serviceName := "test-service-with-dash_and_underscore"
	port, _, err := pm.AssignPort(serviceName, 3000, false)
	if err != nil {
		t.Fatalf("Failed to assign port: %v", err)
	}

	// Verify assignment was saved correctly
	retrievedPort, exists := pm.GetAssignment(serviceName)
	if !exists {
		t.Fatal("Assignment not found after save")
	}

	if retrievedPort != port {
		t.Errorf("Retrieved port %d, expected %d", retrievedPort, port)
	}
}
