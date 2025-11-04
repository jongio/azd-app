package commands

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/cache"
	"github.com/jongio/azd-app/cli/src/internal/output"

	"github.com/spf13/cobra"
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
	Reqs []Prerequisite `yaml:"reqs"`
}

// ReqResult represents the result of checking a requirement.
type ReqResult struct {
	ID         string `json:"id"`
	Installed  bool   `json:"installed"`
	Version    string `json:"version,omitempty"`
	Required   string `json:"required"`
	Satisfied  bool   `json:"satisfied"`
	Running    bool   `json:"running,omitempty"`
	CheckedRun bool   `json:"checkedRunning,omitempty"`
	Message    string `json:"message,omitempty"`
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
	"npm": {
		Command: "npm",
		Args:    []string{"--version"},
	},
	"pnpm": {
		Command: "pnpm",
		Args:    []string{"--version"},
	},
	"yarn": {
		Command: "yarn",
		Args:    []string{"--version"},
	},
	"python": {
		Command:      "python",
		Args:         []string{"--version"},
		VersionField: 1, // "Python 3.12.0" -> take field 1
	},
	"pip": {
		Command:      "pip",
		Args:         []string{"--version"},
		VersionField: 1, // "pip 25.2 from ..." -> take field 1
	},
	"poetry": {
		Command:      "poetry",
		Args:         []string{"--version"},
		VersionField: 2, // "Poetry (version 2.2.1)" -> take field 2
	},
	"uv": {
		Command: "uv",
		Args:    []string{"--version"},
	},
	"pipenv": {
		Command: "pipenv",
		Args:    []string{"--version"},
	},
	"dotnet": {
		Command: "dotnet",
		Args:    []string{"--version"},
	},
	"aspire": {
		Command: "aspire",
		Args:    []string{"--version"},
	},
	"docker": {
		Command:      "docker",
		Args:         []string{"--version"},
		VersionField: 2, // "Docker version 28.5.1, build ..." -> take field 2
	},
	"git": {
		Command:      "git",
		Args:         []string{"--version"},
		VersionField: 2, // "git version 2.51.2.windows.1" -> take field 2
	},
	"go": {
		Command:      "go",
		Args:         []string{"version"},
		VersionField: 2, // "go version go1.25.3 windows/amd64" -> take field 2
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
	var generateMode bool
	var dryRun bool
	var noCache bool
	var clearCache bool

	cmd := &cobra.Command{
		Use:   "reqs",
		Short: "Check for required reqs",
		Long: `The reqs command verifies that all required reqs defined in azure.yaml
are installed and meet the minimum version reqs.

With --generate, it scans your project to detect dependencies and automatically
generates the reqs section in azure.yaml based on what's installed on your machine.

The command caches results in .azure/cache/ to improve performance on subsequent runs.
Use --no-cache to force a fresh check and bypass cached results.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Try to get the output flag from parent or self
			var formatValue string
			if flag := cmd.InheritedFlags().Lookup("output"); flag != nil {
				formatValue = flag.Value.String()
			} else if flag := cmd.Flags().Lookup("output"); flag != nil {
				formatValue = flag.Value.String()
			}
			if formatValue != "" {
				return output.SetFormat(formatValue)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle clear cache flag
			if clearCache {
				return runClearCache()
			}

			// Set global cache disable flag (defined in core.go)
			setDisableCache(noCache)

			if generateMode {
				// Get current working directory
				workingDir, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get working directory: %w", err)
				}

				config := GenerateConfig{
					DryRun:     dryRun,
					WorkingDir: workingDir,
				}
				return runGenerate(config)
			}
			return cmdOrchestrator.Run("reqs")
		},
	}

	cmd.Flags().BoolVarP(&generateMode, "generate", "g", false, "Generate reqs from detected project dependencies")
	cmd.Flags().BoolVar(&generateMode, "gen", false, "Alias for --generate")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without modifying azure.yaml")
	cmd.Flags().BoolVar(&noCache, "no-cache", false, "Force fresh reqs check and bypass cached results")
	cmd.Flags().BoolVar(&clearCache, "clear-cache", false, "Clear cached reqs results")

	return cmd
}

func runReqs() error {
	// Use orchestrator to execute reqs check with caching support
	return executeReqs()
}

// checkPrerequisiteWithResult checks a prerequisite and returns structured result.
func checkPrerequisiteWithResult(prereq Prerequisite) ReqResult {
	installed, version := getInstalledVersion(prereq)

	result := ReqResult{
		ID:        prereq.ID,
		Installed: installed,
		Version:   version,
		Required:  prereq.MinVersion,
		Satisfied: false,
	}

	if !installed {
		result.Message = "Not installed"
		if !output.IsJSON() {
			output.ItemError("%s: NOT INSTALLED (required: %s)", prereq.ID, prereq.MinVersion)
		}
		return result
	}

	if version == "" {
		result.Message = "Version unknown"
		if !output.IsJSON() {
			output.ItemWarning("%s: INSTALLED (version unknown, required: %s)", prereq.ID, prereq.MinVersion)
		}
		// Continue to check if it's running if needed
	} else {
		versionOk := compareVersions(version, prereq.MinVersion)
		if !versionOk {
			result.Message = fmt.Sprintf("Version %s does not meet minimum %s", version, prereq.MinVersion)
			if !output.IsJSON() {
				output.ItemError("%s: %s (required: %s)", prereq.ID, version, prereq.MinVersion)
			}
			return result
		}
		if !output.IsJSON() {
			output.ItemSuccess("%s: %s (required: %s)", prereq.ID, version, prereq.MinVersion)
		}
	}

	// Check if the tool is running (if configured)
	if prereq.CheckRunning {
		result.CheckedRun = true
		isRunning := checkIsRunning(prereq)
		result.Running = isRunning
		if !isRunning {
			result.Message = "Not running"
			if !output.IsJSON() {
				output.Item("- %s✗%s NOT RUNNING", output.Red, output.Reset)
			}
			return result
		}
		result.Satisfied = true
		result.Message = "Running"
		if !output.IsJSON() {
			output.Item("- %s✓%s RUNNING", output.Green, output.Reset)
		}
		return result
	}

	if version != "" {
		result.Satisfied = true
		result.Message = "Satisfied"
	}
	return result
}

func checkPrerequisite(prereq Prerequisite) bool {
	result := checkPrerequisiteWithResult(prereq)
	return result.Satisfied
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

// runClearCache clears the reqs cache.
func runClearCache() error {
	cacheManager, err := cache.NewCacheManager()
	if err != nil {
		return fmt.Errorf("failed to initialize cache manager: %w", err)
	}

	if err := cacheManager.ClearCache(); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	if output.IsJSON() {
		return output.PrintJSON(map[string]interface{}{
			"success": true,
			"message": "Reqs cache cleared successfully",
		})
	}

	output.Success("Reqs cache cleared successfully")
	return nil
}
