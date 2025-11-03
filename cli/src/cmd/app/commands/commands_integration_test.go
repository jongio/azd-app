//go:build integration
// +build integration

package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunReqsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name          string
		azureYAML     string
		expectError   bool
		errorContains string
	}{
		{
			name: "valid_prerequisites",
			azureYAML: `name: test-project
reqs:
  - id: node
    minVersion: 18.0.0
  - id: azd
    minVersion: 1.0.0
`,
			expectError: false,
		},
		{
			name: "missing_prerequisite",
			azureYAML: `name: test-project
reqs:
  - id: nonexistent-tool-xyz-123
    minVersion: 1.0.0
`,
			expectError:   true,
			errorContains: "NOT INSTALLED",
		},
		{
			name: "docker_with_running_check",
			azureYAML: `name: test-project
reqs:
  - id: docker
    minVersion: 20.0.0
    checkRunning: true
`,
			expectError: false, // May fail if Docker not running, but that's expected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save current directory
			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := os.Chdir(originalDir); err != nil {
					t.Logf("Warning: failed to restore directory: %v", err)
				}
			}()

			// Create temporary directory with azure.yaml
			tempDir := t.TempDir()
			if err := os.Chdir(tempDir); err != nil {
				t.Fatal(err)
			}

			azureYamlPath := filepath.Join(tempDir, "azure.yaml")
			if err := os.WriteFile(azureYamlPath, []byte(tt.azureYAML), 0600); err != nil {
				t.Fatal(err)
			}

			err = runReqs()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errorContains)
				}
			} else {
				if err != nil {
					t.Logf("runReqs() returned error: %v (may be expected)", err)
				}
			}
		})
	}
}

func TestCheckPrerequisiteIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name     string
		prereq   Prerequisite
		wantPass bool
	}{
		{
			name: "node_installed",
			prereq: Prerequisite{
				ID:         "node",
				MinVersion: "14.0.0",
			},
			wantPass: true, // Assumes Node.js is installed
		},
		{
			name: "go_installed",
			prereq: Prerequisite{
				ID:         "go",
				MinVersion: "1.20.0",
			},
			wantPass: true, // Assumes Go is installed
		},
		{
			name: "custom_tool_not_installed",
			prereq: Prerequisite{
				ID:         "nonexistent-tool-integration-test",
				MinVersion: "1.0.0",
			},
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed := checkPrerequisite(tt.prereq)

			if tt.wantPass && !passed {
				t.Errorf("checkPrerequisite() failed, want success")
			}

			if !tt.wantPass && passed {
				t.Errorf("checkPrerequisite() succeeded, want failure")
			}
		})
	}
}

func TestCheckIsRunningIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name    string
		prereq  Prerequisite
		skipMsg string
	}{
		{
			name: "docker_running_check",
			prereq: Prerequisite{
				ID:           "docker",
				CheckRunning: true,
			},
			skipMsg: "Docker may not be installed or running",
		},
		{
			name: "custom_check_echo",
			prereq: Prerequisite{
				ID:                   "test-service",
				CheckRunning:         true,
				RunningCheckCommand:  "cmd",
				RunningCheckArgs:     []string{"/c", "echo", "active"},
				RunningCheckExpected: "active",
			},
			skipMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipMsg != "" {
				// Try the check but skip if it fails (expected in some environments)
				result := checkIsRunning(tt.prereq)
				if !result {
					t.Skip(tt.skipMsg)
				}
				t.Logf("Running check passed for %s", tt.prereq.ID)
			} else {
				result := checkIsRunning(tt.prereq)
				if !result {
					t.Errorf("checkIsRunning() = false, want true")
				}
			}
		})
	}
}
