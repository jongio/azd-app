package commands

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"app/src/internal/security"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Prerequisite represents a prerequisite from azure.yaml.
type Prerequisite struct {
	ID         string `yaml:"id"`
	MinVersion string `yaml:"minVersion"`
	// Custom tool configuration (optional)
	Command       string   `yaml:"command,omitempty"`       // Override command to execute
	Args          []string `yaml:"args,omitempty"`          // Override arguments
	VersionPrefix string   `yaml:"versionPrefix,omitempty"` // Override version prefix to strip
	VersionField  int      `yaml:"versionField,omitempty"`  // Override which field contains version
	// Runtime check configuration (optional)
	CheckRunning         bool     `yaml:"checkRunning,omitempty"`         // Whether to check if the tool is running
	RunningCheckCommand  string   `yaml:"runningCheckCommand,omitempty"`  // Command to check if tool is running
	RunningCheckArgs     []string `yaml:"runningCheckArgs,omitempty"`     // Arguments for running check command
	RunningCheckExpected string   `yaml:"runningCheckExpected,omitempty"` // Expected substring in output (optional)
	RunningCheckExitCode *int     `yaml:"runningCheckExitCode,omitempty"` // Expected exit code (default: 0)
}

// AzureYaml represents the structure of azure.yaml.
type AzureYaml struct {
	Requirements []Prerequisite `yaml:"reqs"`
}

// ToolConfig defines how to check a specific tool.
type ToolConfig struct {
	Command       string   // The command to execute
	Args          []string // Arguments to get version
	VersionPrefix string   // Prefix to strip from version output (e.g., "v" for node)
	VersionField  int      // Which field contains version (0 = whole output, 1 = second field, etc.)
}

// toolRegistry maps canonical tool names to their configuration.
var toolRegistry = map[string]ToolConfig{
	"node": {
		Command:       "node",
		Args:          []string{"--version"},
		VersionPrefix: "v",
	},
	"pnpm": {
		Command: "pnpm",
		Args:    []string{"--version"},
	},
	"python": {
		Command:      "python",
		Args:         []string{"--version"},
		VersionField: 1, // "Python 3.12.0" -> take field 1
	},
	"dotnet": {
		Command: "dotnet",
		Args:    []string{"--version"},
	},
	"aspire": {
		Command: "aspire",
		Args:    []string{"--version"},
	},
	"azd": {
		Command: "azd",
		Args:    []string{"version"},
	},
	"az": {
		Command: "az",
		Args:    []string{"version", "--output", "tsv", "--query", "\"azure-cli\""},
	},
}

// toolAliases maps alternative names to canonical tool names.
var toolAliases = map[string]string{
	"nodejs":    "node",
	"azure-cli": "az",
}

// NewReqsCommand creates the reqs command.
func NewReqsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "reqs",
		Short: "Check for required requirements",
		Long: `The reqs command verifies that all required requirements defined in azure.yaml
are installed and meet the minimum version requirements.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdOrchestrator.Run("reqs")
		},
	}
}

func runReqs() error {
	// Validate path to azure.yaml
	azureYamlPath := "azure.yaml"
	if err := security.ValidatePath(azureYamlPath); err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// #nosec G304 -- Path validated by security.ValidatePath above
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		return fmt.Errorf("failed to read azure.yaml: %w", err)
	}

	var azureYaml AzureYaml
	if err := yaml.Unmarshal(data, &azureYaml); err != nil {
		return fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	if len(azureYaml.Requirements) == 0 {
		fmt.Println("✅ No requirements defined in azure.yaml")
		return nil
	}

	fmt.Println("🔍 Checking requirements...")
	fmt.Println()

	allPassed := true
	for _, prereq := range azureYaml.Requirements {
		passed := checkPrerequisite(prereq)
		if !passed {
			allPassed = false
		}
	}

	fmt.Println()
	if allPassed {
		fmt.Println("✅ All requirements are satisfied!")
		return nil
	}

	fmt.Println("❌ Some requirements are missing or don't meet minimum version requirements")
	return fmt.Errorf("requirement check failed")
}

func checkPrerequisite(prereq Prerequisite) bool {
	installed, version := getInstalledVersion(prereq)

	if !installed {
		fmt.Printf("❌ %s: NOT INSTALLED (required: %s)\n", prereq.ID, prereq.MinVersion)
		return false
	}

	if version == "" {
		fmt.Printf("⚠️  %s: INSTALLED (version unknown, required: %s)\n", prereq.ID, prereq.MinVersion)
		// Continue to check if it's running if needed
	} else {
		versionOk := compareVersions(version, prereq.MinVersion)
		if !versionOk {
			fmt.Printf("❌ %s: %s (required: %s)\n", prereq.ID, version, prereq.MinVersion)
			return false
		}
		fmt.Printf("✅ %s: %s (required: %s)", prereq.ID, version, prereq.MinVersion)
	}

	// Check if the tool is running (if configured)
	if prereq.CheckRunning {
		isRunning := checkIsRunning(prereq)
		if !isRunning {
			fmt.Printf(" - ❌ NOT RUNNING\n")
			return false
		}
		fmt.Printf(" - ✅ RUNNING\n")
		return true
	}

	if version != "" {
		fmt.Println()
	}
	return true
}

func getInstalledVersion(prereq Prerequisite) (installed bool, version string) {
	var config ToolConfig

	// Check if custom configuration is provided in prerequisite
	if prereq.Command != "" {
		config = ToolConfig{
			Command:       prereq.Command,
			Args:          prereq.Args,
			VersionPrefix: prereq.VersionPrefix,
			VersionField:  prereq.VersionField,
		}
	} else {
		// Use registry-based configuration
		tool := prereq.ID

		// Resolve aliases to canonical name
		if canonical, isAlias := toolAliases[tool]; isAlias {
			tool = canonical
		}

		// Look up tool configuration
		found := false
		config, found = toolRegistry[tool]
		if !found {
			// Fallback: try generic --version with tool ID as command
			config = ToolConfig{
				Command: prereq.ID,
				Args:    []string{"--version"},
			}
		}
	}

	// #nosec G204 -- Command and args come from toolRegistry or validated azure.yaml prerequisite configuration
	cmd := exec.Command(config.Command, config.Args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, ""
	}

	outputStr := strings.TrimSpace(string(output))
	version = extractVersion(config, outputStr)

	return true, version
}

func extractVersion(config ToolConfig, output string) string {
	// Strip prefix if configured
	if config.VersionPrefix != "" {
		output = strings.TrimPrefix(output, config.VersionPrefix)
	}

	// Extract specific field if configured
	if config.VersionField > 0 {
		parts := strings.Fields(output)
		if len(parts) > config.VersionField {
			output = parts[config.VersionField]
		}
	}

	// Handle azd special case (multi-line output)
	if strings.Contains(output, "azd version") {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "azd version") {
				parts := strings.Fields(line)
				for _, part := range parts {
					if v := extractFirstVersion(part); v != "" && v != "version" {
						return v
					}
				}
			}
		}
	}

	return extractFirstVersion(output)
}

func extractFirstVersion(s string) string {
	// Match semantic version pattern (e.g., 1.2.3, 20.0.0, etc.)
	re := regexp.MustCompile(`(\d+\.\d+\.\d+)`)
	matches := re.FindStringSubmatch(s)
	if len(matches) > 1 {
		return matches[1]
	}

	// Try simpler pattern (e.g., 1.2)
	re = regexp.MustCompile(`(\d+\.\d+)`)
	matches = re.FindStringSubmatch(s)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

func compareVersions(installed, required string) bool {
	installedParts := parseVersion(installed)
	requiredParts := parseVersion(required)

	// Compare each part
	for i := 0; i < len(requiredParts); i++ {
		if i >= len(installedParts) {
			// Installed version has fewer parts, assume 0
			return false
		}

		if installedParts[i] > requiredParts[i] {
			return true
		} else if installedParts[i] < requiredParts[i] {
			return false
		}
		// If equal, continue to next part
	}

	return true // All parts equal or installed version is longer
}

func checkIsRunning(prereq Prerequisite) bool {
	// If no custom running check is configured, use defaults based on tool ID
	command := prereq.RunningCheckCommand
	args := prereq.RunningCheckArgs
	expectedExitCode := 0
	if prereq.RunningCheckExitCode != nil {
		expectedExitCode = *prereq.RunningCheckExitCode
	}

	// Default checks for known tools
	if command == "" {
		switch prereq.ID {
		case "docker":
			command = "docker"
			args = []string{"ps"}
		default:
			// No default running check for this tool
			return true
		}
	}

	// #nosec G204 -- Command and args come from azure.yaml running check configuration or default Docker check
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()

	// Check exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Command failed to execute
			return false
		}
	}

	if exitCode != expectedExitCode {
		return false
	}

	// If an expected substring is configured, check for it in the output
	if prereq.RunningCheckExpected != "" {
		outputStr := strings.TrimSpace(string(output))
		return strings.Contains(outputStr, prereq.RunningCheckExpected)
	}

	return true
}

func parseVersion(version string) []int {
	parts := strings.Split(version, ".")
	result := make([]int, 0, len(parts))

	for _, part := range parts {
		var num int
		_, _ = fmt.Sscanf(part, "%d", &num)
		result = append(result, num)
	}

	return result
}
