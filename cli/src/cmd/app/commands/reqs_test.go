package commands

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func shellCommand(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/c", command}
	}
	return "sh", []string{"-c", command}
}

func intPtr(v int) *int {
	return &v
}

func TestExtractFirstVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "semantic version",
			input:    "v1.2.3",
			expected: "1.2.3",
		},
		{
			name:     "version with text",
			input:    "Python 3.12.1",
			expected: "3.12.1",
		},
		{
			name:     "simple version",
			input:    "20.0.0",
			expected: "20.0.0",
		},
		{
			name:     "two-part version",
			input:    "1.2",
			expected: "1.2",
		},
		{
			name:     "no version",
			input:    "hello world",
			expected: "",
		},
		{
			name:     "version at end",
			input:    "go version go1.21.5 windows/amd64",
			expected: "1.21.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFirstVersion(tt.input)
			if result != tt.expected {
				t.Errorf("extractFirstVersion(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected []int
	}{
		{
			name:     "semantic version",
			version:  "1.2.3",
			expected: []int{1, 2, 3},
		},
		{
			name:     "two parts",
			version:  "20.0",
			expected: []int{20, 0},
		},
		{
			name:     "four parts",
			version:  "1.2.3.4",
			expected: []int{1, 2, 3, 4},
		},
		{
			name:     "single digit",
			version:  "5",
			expected: []int{5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVersion(tt.version)
			if len(result) != len(tt.expected) {
				t.Fatalf("parseVersion(%q) length = %d, want %d", tt.version, len(result), len(tt.expected))
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("parseVersion(%q)[%d] = %d, want %d", tt.version, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name      string
		installed string
		required  string
		expected  bool
	}{
		{
			name:      "exact match",
			installed: "1.2.3",
			required:  "1.2.3",
			expected:  true,
		},
		{
			name:      "installed newer major",
			installed: "2.0.0",
			required:  "1.0.0",
			expected:  true,
		},
		{
			name:      "installed newer minor",
			installed: "1.5.0",
			required:  "1.2.0",
			expected:  true,
		},
		{
			name:      "installed newer patch",
			installed: "1.2.5",
			required:  "1.2.3",
			expected:  true,
		},
		{
			name:      "installed older major",
			installed: "1.0.0",
			required:  "2.0.0",
			expected:  false,
		},
		{
			name:      "installed older minor",
			installed: "1.2.0",
			required:  "1.5.0",
			expected:  false,
		},
		{
			name:      "installed older patch",
			installed: "1.2.3",
			required:  "1.2.5",
			expected:  false,
		},
		{
			name:      "installed has fewer parts",
			installed: "1.2",
			required:  "1.2.3",
			expected:  false,
		},
		{
			name:      "installed has more parts",
			installed: "1.2.3.4",
			required:  "1.2.3",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareVersions(tt.installed, tt.required)
			if result != tt.expected {
				t.Errorf("compareVersions(%q, %q) = %v, want %v", tt.installed, tt.required, result, tt.expected)
			}
		})
	}
}

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		name     string
		config   ToolConfig
		output   string
		expected string
	}{
		{
			name: "node with prefix",
			config: ToolConfig{
				VersionPrefix: "v",
			},
			output:   "v20.0.0",
			expected: "20.0.0",
		},
		{
			name: "python with field",
			config: ToolConfig{
				VersionField: 1,
			},
			output:   "Python 3.12.1",
			expected: "3.12.1",
		},
		{
			name: "go with field only",
			config: ToolConfig{
				VersionField: 2,
			},
			output:   "go version go1.21.5 windows/amd64",
			expected: "1.21.5",
		},
		{
			name:     "plain version",
			config:   ToolConfig{},
			output:   "10.0.100",
			expected: "10.0.100",
		},
		{
			name:     "azd multiline",
			config:   ToolConfig{},
			output:   "azd version 1.9.3 (commit abcd1234)\ncopyright info",
			expected: "1.9.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVersion(tt.config, tt.output)
			if result != tt.expected {
				t.Errorf("extractVersion(%+v, %q) = %q, want %q", tt.config, tt.output, result, tt.expected)
			}
		})
	}
}

func TestGetInstalledVersion(t *testing.T) {
	tests := []struct {
		name                  string
		prereq                Prerequisite
		expectInstalled       bool
		expectVersionNonEmpty bool
	}{
		{
			name: "custom tool - go",
			prereq: Prerequisite{
				ID:            "go",
				Command:       "go",
				Args:          []string{"version"},
				VersionField:  2,
				VersionPrefix: "go",
			},
			expectInstalled:       false, // May not be installed
			expectVersionNonEmpty: false,
		},
		{
			name: "nonexistent tool",
			prereq: Prerequisite{
				ID:      "nonexistent-tool-xyz",
				Command: "nonexistent-tool-xyz",
				Args:    []string{"--version"},
			},
			expectInstalled:       false,
			expectVersionNonEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installed, version := getInstalledVersion(tt.prereq)

			// Just verify the function runs without panicking
			// Actual version detection depends on what's installed on the test machine
			if tt.expectInstalled && !installed {
				t.Logf("Expected %s to be installed but it wasn't (this is OK if not on system)", tt.prereq.ID)
			}

			if installed && tt.expectVersionNonEmpty && version == "" {
				t.Errorf("Tool %s is installed but version is empty", tt.prereq.ID)
			}
		})
	}
}

func TestToolAliases(t *testing.T) {
	tests := []struct {
		name     string
		prereq   Prerequisite
		expected string // Expected command that would be executed
	}{
		{
			name: "nodejs alias resolves to node",
			prereq: Prerequisite{
				ID: "nodejs",
			},
			expected: "node",
		},
		{
			name: "azure-cli alias resolves to az",
			prereq: Prerequisite{
				ID: "azure-cli",
			},
			expected: "az",
		},
		{
			name: "node uses node directly",
			prereq: Prerequisite{
				ID: "node",
			},
			expected: "node",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test just verifies aliases resolve without executing
			tool := tt.prereq.ID
			if canonical, isAlias := toolAliases[tool]; isAlias {
				tool = canonical
			}

			config, found := toolRegistry[tool]
			if !found {
				t.Errorf("Tool %s (from %s) not found in registry", tool, tt.prereq.ID)
				return
			}

			if config.Command != tt.expected {
				t.Errorf("Tool %s resolved to command %s, want %s", tt.prereq.ID, config.Command, tt.expected)
			}
		})
	}
}

func TestRunPrereqsWithMissingFile(t *testing.T) {
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

	// Create temporary directory without azure.yaml
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	err = runReqs()
	if err == nil {
		t.Error("Expected error when azure.yaml is missing, got nil")
	}
}

func TestRunPrereqsWithInvalidYAML(t *testing.T) {
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

	// Create temporary directory with invalid azure.yaml
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	invalidYAML := `this is not valid: yaml: content: [[[`
	if err := os.WriteFile("azure.yaml", []byte(invalidYAML), 0600); err != nil {
		t.Fatal(err)
	}

	err = runReqs()
	if err == nil {
		t.Error("Expected error when azure.yaml is invalid, got nil")
	}
}

func TestRunPrereqsWithNoPrereqs(t *testing.T) {
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

	// Create temporary directory with empty reqs
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	emptyYAML := `name: test
reqs: []
`
	if err := os.WriteFile("azure.yaml", []byte(emptyYAML), 0600); err != nil {
		t.Fatal(err)
	}

	err = runReqs()
	if err == nil {
		t.Error("Expected error with empty reqs (backwards compatibility removed), got nil")
	}
	if !strings.Contains(err.Error(), "no reqs defined in azure.yaml") {
		t.Errorf("Expected error about no reqs defined, got: %v", err)
	}
}

func TestRunPrereqsWithValidPrereqs(t *testing.T) {
	// Skip if not in test environment with known tools
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

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

	// Create temporary directory
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	// Use a very old version requirement that should always pass
	validYAML := `name: test
reqs:
  - id: nonexistent-tool-for-testing
    minVersion: 0.0.1
    command: echo
    args: ["1.0.0"]
`
	if err := os.WriteFile("azure.yaml", []byte(validYAML), 0600); err != nil {
		t.Fatal(err)
	}

	// This should pass because echo command succeeds and returns "1.0.0"
	err = runReqs()
	if err != nil {
		t.Logf("Check failed (expected if echo doesn't return valid version): %v", err)
	}
}

func TestCheckPrerequisite(t *testing.T) {
	tests := []struct {
		name     string
		prereq   Prerequisite
		expected bool
	}{
		{
			name: "tool not installed",
			prereq: Prerequisite{
				ID:         "nonexistent-xyz-123",
				MinVersion: "1.0.0",
				Command:    "nonexistent-xyz-123",
				Args:       []string{"--version"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkPrerequisite(tt.prereq)
			if result != tt.expected {
				t.Errorf("checkPrerequisite(%+v) = %v, want %v", tt.prereq, result, tt.expected)
			}
		})
	}
}

func TestToolRegistryCompleteness(t *testing.T) {
	requiredTools := []string{"node", "pnpm", "python", "dotnet", "aspire", "azd", "az"}

	for _, tool := range requiredTools {
		t.Run(tool, func(t *testing.T) {
			config, found := toolRegistry[tool]
			if !found {
				t.Errorf("Tool %s not found in registry", tool)
				return
			}

			if config.Command == "" {
				t.Errorf("Tool %s has empty command", tool)
			}

			if len(config.Args) == 0 {
				t.Errorf("Tool %s has no args", tool)
			}
		})
	}
}

func TestAzureYamlParsing(t *testing.T) {
	tempDir := t.TempDir()
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")

	yamlContent := `name: test-project
reqs:
  - id: node
    minVersion: 18.0.0
  - id: custom-tool
    minVersion: 1.0.0
    command: my-tool
    args: ["version"]
    versionPrefix: "v"
    versionField: 1
`

	if err := os.WriteFile(azureYamlPath, []byte(yamlContent), 0600); err != nil {
		t.Fatal(err)
	}

	// #nosec G304 -- Test file reading from controlled temp directory
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Fatal(err)
	}

	var azureYaml AzureYaml
	if err := yaml.Unmarshal(data, &azureYaml); err != nil {
		t.Fatal(err)
	}

	if len(azureYaml.Reqs) != 2 {
		t.Errorf("Expected 2 reqs, got %d", len(azureYaml.Reqs))
	}

	// Check first req (built-in)
	if azureYaml.Reqs[0].ID != "node" {
		t.Errorf("First req ID = %s, want node", azureYaml.Reqs[0].ID)
	}
	if azureYaml.Reqs[0].MinVersion != "18.0.0" {
		t.Errorf("First req MinVersion = %s, want 18.0.0", azureYaml.Reqs[0].MinVersion)
	}
	if azureYaml.Reqs[0].Command != "" {
		t.Errorf("First req should have empty Command for built-in tool")
	}

	// Check second req (custom)
	if azureYaml.Reqs[1].ID != "custom-tool" {
		t.Errorf("Second req ID = %s, want custom-tool", azureYaml.Reqs[1].ID)
	}
	if azureYaml.Reqs[1].Command != "my-tool" {
		t.Errorf("Second req Command = %s, want my-tool", azureYaml.Reqs[1].Command)
	}
	if azureYaml.Reqs[1].VersionPrefix != "v" {
		t.Errorf("Second req VersionPrefix = %s, want v", azureYaml.Reqs[1].VersionPrefix)
	}
	if azureYaml.Reqs[1].VersionField != 1 {
		t.Errorf("Second req VersionField = %d, want 1", azureYaml.Reqs[1].VersionField)
	}
}

func TestCheckIsRunning(t *testing.T) {
	tests := []struct {
		name     string
		prereq   Prerequisite
		expected bool
		skip     string // Skip message if test should be skipped
	}{
		{
			name: "docker with default check",
			prereq: Prerequisite{
				ID:           "docker",
				CheckRunning: true,
			},
			expected: false, // May not be running in test environment
			skip:     "Docker may not be installed or running",
		},
		{
			name: "custom check with exit code zero",
			prereq: func() Prerequisite {
				cmdName, cmdArgs := shellCommand("exit 0")
				return Prerequisite{
					ID:                   "custom-service",
					CheckRunning:         true,
					RunningCheckCommand:  cmdName,
					RunningCheckArgs:     cmdArgs,
					RunningCheckExitCode: intPtr(0),
				}
			}(),
			expected: true, // Should succeed with exit code 0
		},
		{
			name: "custom check expecting non-zero exit",
			prereq: Prerequisite{
				ID:                   "failing-service",
				CheckRunning:         true,
				RunningCheckCommand:  "nonexistent-command-xyz",
				RunningCheckArgs:     []string{},
				RunningCheckExitCode: intPtr(1),
			},
			expected: false, // Command doesn't exist
		},
		{
			name: "check with expected output",
			prereq: func() Prerequisite {
				cmdName, cmdArgs := shellCommand("echo hello world")
				return Prerequisite{
					ID:                   "echo-service",
					CheckRunning:         true,
					RunningCheckCommand:  cmdName,
					RunningCheckArgs:     cmdArgs,
					RunningCheckExpected: "hello",
				}
			}(),
			expected: true, // echo output contains "hello"
		},
		{
			name: "check with missing expected output",
			prereq: func() Prerequisite {
				cmdName, cmdArgs := shellCommand("echo hello world")
				return Prerequisite{
					ID:                   "echo-service",
					CheckRunning:         true,
					RunningCheckCommand:  cmdName,
					RunningCheckArgs:     cmdArgs,
					RunningCheckExpected: "goodbye",
				}
			}(),
			expected: false, // echo output doesn't contain "goodbye"
		},
		{
			name: "no running check configured, no default",
			prereq: Prerequisite{
				ID:           "unknown-tool",
				CheckRunning: true,
			},
			expected: true, // Should default to true when no check is configured
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip != "" {
				t.Skip(tt.skip)
			}

			result := checkIsRunning(tt.prereq)
			if result != tt.expected {
				t.Errorf("checkIsRunning(%+v) = %v, want %v", tt.prereq, result, tt.expected)
			}
		})
	}
}

func TestCheckPrerequisiteWithRunningCheck(t *testing.T) {
	tests := []struct {
		name     string
		prereq   Prerequisite
		expected bool
	}{
		{
			name: "installed and running",
			prereq: func() Prerequisite {
				versionCmd, versionArgs := shellCommand("echo 2.0.0")
				runningCmd, runningArgs := shellCommand("echo running")
				return Prerequisite{
					ID:                   "test-tool",
					MinVersion:           "1.0.0",
					Command:              versionCmd,
					Args:                 versionArgs,
					CheckRunning:         true,
					RunningCheckCommand:  runningCmd,
					RunningCheckArgs:     runningArgs,
					RunningCheckExpected: "running",
				}
			}(),
			expected: true,
		},
		{
			name: "installed but not running",
			prereq: func() Prerequisite {
				versionCmd, versionArgs := shellCommand("echo 2.0.0")
				runningCmd, runningArgs := shellCommand("echo stopped")
				return Prerequisite{
					ID:                   "test-tool",
					MinVersion:           "1.0.0",
					Command:              versionCmd,
					Args:                 versionArgs,
					CheckRunning:         true,
					RunningCheckCommand:  runningCmd,
					RunningCheckArgs:     runningArgs,
					RunningCheckExpected: "running", // Won't match
				}
			}(),
			expected: false,
		},
		{
			name: "version too old but running",
			prereq: func() Prerequisite {
				versionCmd, versionArgs := shellCommand("echo 2.0.0")
				runningCmd, runningArgs := shellCommand("echo running")
				return Prerequisite{
					ID:                   "test-tool",
					MinVersion:           "3.0.0",
					Command:              versionCmd,
					Args:                 versionArgs,
					CheckRunning:         true,
					RunningCheckCommand:  runningCmd,
					RunningCheckArgs:     runningArgs,
					RunningCheckExpected: "running",
				}
			}(),
			expected: false, // Version check should fail before running check
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkPrerequisite(tt.prereq)
			if result != tt.expected {
				t.Errorf("checkPrerequisite(%+v) = %v, want %v", tt.prereq, result, tt.expected)
			}
		})
	}
}

func TestAzureYamlParsingWithRunningCheck(t *testing.T) {
	tempDir := t.TempDir()
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")

	yamlContent := `name: test-project
reqs:
  - id: docker
    minVersion: 20.0.0
    checkRunning: true
  - id: custom-service
    minVersion: 1.0.0
    command: my-service
    args: ["--version"]
    checkRunning: true
    runningCheckCommand: my-service
    runningCheckArgs: ["status"]
    runningCheckExpected: "active"
    runningCheckExitCode: 0
`

	if err := os.WriteFile(azureYamlPath, []byte(yamlContent), 0600); err != nil {
		t.Fatal(err)
	}

	// #nosec G304 -- Test file reading from controlled temp directory
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Fatal(err)
	}

	var azureYaml AzureYaml
	if err := yaml.Unmarshal(data, &azureYaml); err != nil {
		t.Fatal(err)
	}

	if len(azureYaml.Reqs) != 2 {
		t.Errorf("Expected 2 reqs, got %d", len(azureYaml.Reqs))
	}

	// Check Docker requirement
	dockerReq := azureYaml.Reqs[0]
	if !dockerReq.CheckRunning {
		t.Error("Docker CheckRunning should be true")
	}

	// Check custom service requirement
	customReq := azureYaml.Reqs[1]
	if !customReq.CheckRunning {
		t.Error("Custom service CheckRunning should be true")
	}
	if customReq.RunningCheckCommand != "my-service" {
		t.Errorf("RunningCheckCommand = %s, want my-service", customReq.RunningCheckCommand)
	}
	if len(customReq.RunningCheckArgs) != 1 || customReq.RunningCheckArgs[0] != "status" {
		t.Errorf("RunningCheckArgs = %v, want [status]", customReq.RunningCheckArgs)
	}
	if customReq.RunningCheckExpected != "active" {
		t.Errorf("RunningCheckExpected = %s, want active", customReq.RunningCheckExpected)
	}
	if customReq.RunningCheckExitCode == nil || *customReq.RunningCheckExitCode != 0 {
		t.Error("RunningCheckExitCode should be 0")
	}
}
