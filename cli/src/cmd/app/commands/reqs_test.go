package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/output"
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
		// NEW: Test cases for the bug fix
		{
			name:      "installed fewer parts but equal (1.2 vs 1.2.0)",
			installed: "1.2",
			required:  "1.2.0",
			expected:  true, // Fixed: Should treat 1.2 as 1.2.0
		},
		{
			name:      "installed fewer parts but equal (18 vs 18.0.0)",
			installed: "18",
			required:  "18.0.0",
			expected:  true, // Fixed: Should treat 18 as 18.0.0
		},
		{
			name:      "installed fewer parts still less than required",
			installed: "1.1",
			required:  "1.2.0",
			expected:  false, // Should still fail if actually less
		},
		{
			name:      "required fewer parts than installed (edge case)",
			installed: "1.2.3",
			required:  "1.2",
			expected:  true, // 1.2.3 >= 1.2.0 (implicit)
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
		{
			name:   "podman aliased to docker",
			config: toolRegistry["docker"],
			output: `Client:       Podman Engine
Version:      5.7.0
API Version:  5.7.0
Go Version:   go1.25.4
Git Commit:   0370128fc8dcae93533334324ef838db8f8da8cb
Built:        Tue Nov 11 10:57:57 2025
OS/Arch:      windows/amd64

Server:       Podman Engine
Version:      5.7.0
API Version:  5.7.0
Go Version:   go1.24.9
Git Commit:   0370128fc8dcae93533334324ef838db8f8da8cb
Built:        Mon Nov 10 16:00:00 2025
OS/Arch:      linux/amd64`,
			expected: "5.7.0",
		},
		{
			name:     "docker native version",
			config:   toolRegistry["docker"],
			output:   "Docker version 28.5.1, build abc123",
			expected: "28.5.1",
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

func TestExtractPodmanVersion(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name: "full podman output",
			output: `Client:       Podman Engine
Version:      5.7.0
API Version:  5.7.0
Go Version:   go1.25.4
Git Commit:   0370128fc8dcae93533334324ef838db8f8da8cb
Built:        Tue Nov 11 10:57:57 2025
OS/Arch:      windows/amd64

Server:       Podman Engine
Version:      5.7.0
API Version:  5.7.0
Go Version:   go1.24.9
Git Commit:   0370128fc8dcae93533334324ef838db8f8da8cb
Built:        Mon Nov 10 16:00:00 2025
OS/Arch:      linux/amd64`,
			expected: "5.7.0",
		},
		{
			name: "client only",
			output: `Client:       Podman Engine
Version:      4.9.3
API Version:  4.9.3`,
			expected: "4.9.3",
		},
		{
			name:     "version with extra spaces",
			output:   "Version:      5.7.0",
			expected: "5.7.0",
		},
		{
			name:     "missing version line",
			output:   "Client:       Podman Engine\nAPI Version:  5.7.0",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPodmanVersion(tt.output)
			if result != tt.expected {
				t.Errorf("extractPodmanVersion(%q) = %q, want %q", tt.output, result, tt.expected)
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
				Name:          "go",
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
				Name:    "nonexistent-tool-xyz",
				Command: "nonexistent-tool-xyz",
				Args:    []string{"--version"},
			},
			expectInstalled:       false,
			expectVersionNonEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewPrerequisiteChecker()
			installed, version := checker.getInstalledVersion(tt.prereq)

			// Just verify the function runs without panicking
			// Actual version detection depends on what's installed on the test machine
			if tt.expectInstalled && !installed {
				t.Logf("Expected %s to be installed but it wasn't (this is OK if not on system)", tt.prereq.Name)
			}

			if installed && tt.expectVersionNonEmpty && version == "" {
				t.Errorf("Tool %s is installed but version is empty", tt.prereq.Name)
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
				Name: "nodejs",
			},
			expected: "node",
		},
		{
			name: "azure-cli alias resolves to az",
			prereq: Prerequisite{
				Name: "azure-cli",
			},
			expected: "az",
		},
		{
			name: "node uses node directly",
			prereq: Prerequisite{
				Name: "node",
			},
			expected: "node",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test just verifies aliases resolve without executing
			tool := tt.prereq.Name
			if canonical, isAlias := toolAliases[tool]; isAlias {
				tool = canonical
			}

			config, found := toolRegistry[tool]
			if !found {
				t.Errorf("Tool %s (from %s) not found in registry", tool, tt.prereq.Name)
				return
			}

			if config.Command != tt.expected {
				t.Errorf("Tool %s resolved to command %s, want %s", tt.prereq.Name, config.Command, tt.expected)
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
		if chdirErr := os.Chdir(originalDir); chdirErr != nil {
			t.Logf("Warning: failed to restore directory: %v", chdirErr)
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
		if chdirErr := os.Chdir(originalDir); chdirErr != nil {
			t.Logf("Warning: failed to restore directory: %v", chdirErr)
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
	tests := []struct {
		name           string
		yamlContent    string
		useJSONOutput  bool
		expectedError  bool
		validateOutput func(t *testing.T)
	}{
		{
			name: "empty reqs array - default output",
			yamlContent: `name: test
reqs: []
`,
			useJSONOutput: false,
			expectedError: false,
		},
		{
			name: "no reqs section - default output",
			yamlContent: `name: test
services:
  - name: web
`,
			useJSONOutput: false,
			expectedError: false,
		},
		{
			name: "empty reqs array - JSON output",
			yamlContent: `name: test
reqs: []
`,
			useJSONOutput: true,
			expectedError: false,
		},
		{
			name: "no reqs section - JSON output",
			yamlContent: `name: test
services:
  - name: api
`,
			useJSONOutput: true,
			expectedError: false,
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
				if chdirErr := os.Chdir(originalDir); chdirErr != nil {
					t.Logf("Warning: failed to restore directory: %v", chdirErr)
				}
			}()

			// Create temporary directory
			tempDir := t.TempDir()
			if err := os.Chdir(tempDir); err != nil {
				t.Fatal(err)
			}

			// Write azure.yaml
			if err := os.WriteFile("azure.yaml", []byte(tt.yamlContent), 0600); err != nil {
				t.Fatal(err)
			}

			// Set output format
			if tt.useJSONOutput {
				if err := output.SetFormat("json"); err != nil {
					t.Fatal(err)
				}
				defer func() {
					_ = output.SetFormat("default")
				}()
			}

			// Run the command
			err = runReqs()

			// Verify error expectation
			if tt.expectedError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Additional validation if provided
			if tt.validateOutput != nil {
				tt.validateOutput(t)
			}
		})
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
		if chdirErr := os.Chdir(originalDir); chdirErr != nil {
			t.Logf("Warning: failed to restore directory: %v", chdirErr)
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
  - name: nonexistent-tool-for-testing
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
				Name:       "nonexistent-xyz-123",
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
	requiredTools := []string{"node", "pnpm", "python", "dotnet", "aspire", "azd", "az", "func"}

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

func TestFuncToolRegistry(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		expected string // Expected command that would be executed
	}{
		{
			name:     "func resolves correctly",
			toolName: "func",
			expected: "func",
		},
		{
			name:     "azure-functions-core-tools alias resolves to func",
			toolName: "azure-functions-core-tools",
			expected: "func",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := tt.toolName
			if canonical, isAlias := toolAliases[tool]; isAlias {
				tool = canonical
			}

			config, found := toolRegistry[tool]
			if !found {
				t.Errorf("Tool %s (from %s) not found in registry", tool, tt.toolName)
				return
			}

			if config.Command != tt.expected {
				t.Errorf("Tool %s resolved to command %s, want %s", tt.toolName, config.Command, tt.expected)
			}
		})
	}
}

func TestFuncVersionExtraction(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "func standard version",
			output:   "4.5.0",
			expected: "4.5.0",
		},
		{
			name:     "func older version",
			output:   "4.0.0",
			expected: "4.0.0",
		},
		{
			name:     "func latest version",
			output:   "4.10.2",
			expected: "4.10.2",
		},
		{
			name:     "func with extra whitespace",
			output:   "  4.5.0  \n",
			expected: "4.5.0",
		},
	}

	config := toolRegistry["func"]

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVersion(config, tt.output)
			if result != tt.expected {
				t.Errorf("extractVersion(func config, %q) = %q, want %q", tt.output, result, tt.expected)
			}
		})
	}
}

func TestFuncVersionComparison(t *testing.T) {
	tests := []struct {
		name      string
		installed string
		required  string
		expected  bool
	}{
		{
			name:      "func 4.5.0 meets 4.0.0 requirement",
			installed: "4.5.0",
			required:  "4.0.0",
			expected:  true,
		},
		{
			name:      "func 4.0.0 meets 4.0.0 requirement",
			installed: "4.0.0",
			required:  "4.0.0",
			expected:  true,
		},
		{
			name:      "func 3.9.0 fails 4.0.0 requirement",
			installed: "3.9.0",
			required:  "4.0.0",
			expected:  false,
		},
		{
			name:      "func 4.10.2 meets 4.5.0 requirement",
			installed: "4.10.2",
			required:  "4.5.0",
			expected:  true,
		},
		{
			name:      "func 4.5.1 meets 4.5.0 requirement",
			installed: "4.5.1",
			required:  "4.5.0",
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

func TestAzureYamlParsing(t *testing.T) {
	tempDir := t.TempDir()
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")

	yamlContent := `name: test-project
reqs:
  - name: node
    minVersion: 18.0.0
  - name: custom-tool
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
	if azureYaml.Reqs[0].Name != "node" {
		t.Errorf("First req ID = %s, want node", azureYaml.Reqs[0].Name)
	}
	if azureYaml.Reqs[0].MinVersion != "18.0.0" {
		t.Errorf("First req MinVersion = %s, want 18.0.0", azureYaml.Reqs[0].MinVersion)
	}
	if azureYaml.Reqs[0].Command != "" {
		t.Errorf("First req should have empty Command for built-in tool")
	}

	// Check second req (custom)
	if azureYaml.Reqs[1].Name != "custom-tool" {
		t.Errorf("Second req ID = %s, want custom-tool", azureYaml.Reqs[1].Name)
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
				Name:         "docker",
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
					Name:                 "custom-service",
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
				Name:                 "failing-service",
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
					Name:                 "echo-service",
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
					Name:                 "echo-service",
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
				Name:         "unknown-tool",
				CheckRunning: true,
			},
			expected: false, // Fixed: Should return false when check is not properly configured
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip != "" {
				t.Skip(tt.skip)
			}

			checker := NewPrerequisiteChecker()
			result := checker.checkIsRunning(tt.prereq)
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
					Name:                 "test-tool",
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
					Name:                 "test-tool",
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
					Name:                 "test-tool",
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
  - name: docker
    minVersion: 20.0.0
    checkRunning: true
  - name: custom-service
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

func TestRunReqsFix_AllSatisfied(t *testing.T) {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if chdirErr := os.Chdir(originalDir); chdirErr != nil {
			t.Logf("Warning: failed to restore directory: %v", chdirErr)
		}
	}()

	// Create temporary directory with valid azure.yaml
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	// Create azure.yaml with a tool that should always be available (echo simulation)
	versionCmd, versionArgs := shellCommand("echo 2.0.0")
	yamlContent := fmt.Sprintf(`name: test
reqs:
  - name: test-tool
    minVersion: 1.0.0
    command: %s
    args: %v
`, versionCmd, yamlArgsString(versionArgs))

	if err := os.WriteFile("azure.yaml", []byte(yamlContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Run fix - should succeed since tool is "installed"
	err = runReqsFix()
	if err != nil {
		t.Logf("Fix completed with message: %v", err)
	}
}

func TestRunReqsFix_NoAzureYaml(t *testing.T) {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if chdirErr := os.Chdir(originalDir); chdirErr != nil {
			t.Logf("Warning: failed to restore directory: %v", chdirErr)
		}
	}()

	// Create temporary directory without azure.yaml
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	err = runReqsFix()
	if err == nil {
		t.Error("Expected error when azure.yaml is missing, got nil")
	}
}

func TestRunReqsFix_PartialSuccess(t *testing.T) {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if chdirErr := os.Chdir(originalDir); chdirErr != nil {
			t.Logf("Warning: failed to restore directory: %v", chdirErr)
		}
	}()

	// Create temporary directory
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	// Create azure.yaml with one satisfiable tool and one that will fail
	versionCmd, versionArgs := shellCommand("echo 2.0.0")
	yamlContent := fmt.Sprintf(`name: test
reqs:
  - name: working-tool
    minVersion: 1.0.0
    command: %s
    args: %v
  - name: nonexistent-tool-xyz-999
    minVersion: 1.0.0
`, versionCmd, yamlArgsString(versionArgs))

	if err := os.WriteFile("azure.yaml", []byte(yamlContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Run fix - should partially succeed
	err = runReqsFix()
	if err == nil {
		t.Error("Expected error when some requirements can't be fixed, got nil")
	}
}

func TestRunReqsFix_NoFailedRequirements(t *testing.T) {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if chdirErr := os.Chdir(originalDir); chdirErr != nil {
			t.Logf("Warning: failed to restore directory: %v", chdirErr)
		}
	}()

	// Create temporary directory
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	// Create azure.yaml with no reqs
	yamlContent := `name: test
reqs: []
`
	if err := os.WriteFile("azure.yaml", []byte(yamlContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Run fix - should fail with appropriate message
	err = runReqsFix()
	if err == nil {
		t.Error("Expected error when no reqs defined, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "no reqs defined") {
		t.Errorf("Expected error about no reqs, got: %v", err)
	}
}

func TestRunReqsFix_JSONOutput(t *testing.T) {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if chdirErr := os.Chdir(originalDir); chdirErr != nil {
			t.Logf("Warning: failed to restore directory: %v", chdirErr)
		}
	}()

	// Create temporary directory
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	// Create azure.yaml with a working tool
	versionCmd, versionArgs := shellCommand("echo 2.0.0")
	yamlContent := fmt.Sprintf(`name: test
reqs:
  - name: test-tool
    minVersion: 1.0.0
    command: %s
    args: %v
`, versionCmd, yamlArgsString(versionArgs))

	if err := os.WriteFile("azure.yaml", []byte(yamlContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Set JSON output format
	if err := output.SetFormat("json"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = output.SetFormat("default")
	}()

	// Run fix - verify it doesn't panic with JSON output
	err = runReqsFix()
	// Don't check error, just verify no panic occurred
	t.Logf("Fix with JSON output completed: %v", err)
}

func TestRunReqsFix_VersionCheckFailsAfterFind(t *testing.T) {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if chdirErr := os.Chdir(originalDir); chdirErr != nil {
			t.Logf("Warning: failed to restore directory: %v", chdirErr)
		}
	}()

	// Create temporary directory
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	// Create azure.yaml with a tool that returns a version but doesn't meet minimum
	versionCmd, versionArgs := shellCommand("echo 0.5.0")
	yamlContent := fmt.Sprintf(`name: test
reqs:
  - name: outdated-tool
    minVersion: 1.0.0
    command: %s
    args: %v
`, versionCmd, yamlArgsString(versionArgs))

	if err := os.WriteFile("azure.yaml", []byte(yamlContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Run fix - should fail because version doesn't meet requirement
	err = runReqsFix()
	if err == nil {
		t.Error("Expected error when version check fails, got nil")
	}
}

func TestRunReqsFix_InvalidYAML(t *testing.T) {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if chdirErr := os.Chdir(originalDir); chdirErr != nil {
			t.Logf("Warning: failed to restore directory: %v", chdirErr)
		}
	}()

	// Create temporary directory
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	// Create invalid azure.yaml
	invalidYAML := `this is not: valid: yaml: [[[`
	if err := os.WriteFile("azure.yaml", []byte(invalidYAML), 0600); err != nil {
		t.Fatal(err)
	}

	// Run fix - should fail with parse error
	err = runReqsFix()
	if err == nil {
		t.Error("Expected error when azure.yaml is invalid, got nil")
	}
}

// yamlArgsString converts args array to YAML string format
func yamlArgsString(args []string) string {
	if len(args) == 0 {
		return "[]"
	}
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = fmt.Sprintf(`"%s"`, arg)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
