package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/cache"
	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/pathutil"

	"github.com/spf13/cobra"
)

// Prerequisite represents a prerequisite from azure.yaml.
type Prerequisite struct {
	Name       string `yaml:"name"`
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
	Name       string `json:"name"`
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
	"func": {
		Command: "func",
		Args:    []string{"--version"},
	},
	"java": {
		Command:      "java",
		Args:         []string{"-version"},
		VersionField: 2, // "java version "17.0.1" ..." or "openjdk version "17.0.1" ..." -> take field 2
	},
	"mvn": {
		Command:      "mvn",
		Args:         []string{"--version"},
		VersionField: 2, // "Apache Maven 3.9.0 ..." -> take field 2
	},
	"gradle": {
		Command:      "gradle",
		Args:         []string{"--version"},
		VersionField: 1, // "Gradle 8.5" -> take field 1
	},
}

// toolAliases maps alternative names to canonical tool names.
var toolAliases = map[string]string{
	"nodejs":                     "node",
	"azure-cli":                  "az",
	"azure-functions-core-tools": "func",
}

// NewReqsCommand creates the reqs command.
func NewReqsCommand() *cobra.Command {
	var generateMode bool
	var dryRun bool
	var noCache bool
	var clearCache bool
	var fixMode bool

	cmd := &cobra.Command{
		Use:   "reqs",
		Short: "Check for required reqs",
		Long: `The reqs command verifies that all required reqs defined in azure.yaml
are installed and meet the minimum version reqs.

With --generate, it scans your project to detect dependencies and automatically
generates the reqs section in azure.yaml based on what's installed on your machine.

With --fix, it attempts to resolve PATH issues by refreshing the environment and
searching for installed tools that aren't accessible in the current session.

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

			// Configure cache based on flag
			SetCacheEnabled(!noCache)

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

			if fixMode {
				// Disable cache for fix mode to ensure fresh checks
				SetCacheEnabled(false)
				return runReqsFix()
			}

			return cmdOrchestrator.Run("reqs")
		},
	}

	cmd.Flags().BoolVarP(&generateMode, "generate", "g", false, "Generate reqs from detected project dependencies")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without modifying azure.yaml")
	cmd.Flags().BoolVar(&noCache, "no-cache", false, "Force fresh reqs check and bypass cached results")
	cmd.Flags().BoolVar(&clearCache, "clear-cache", false, "Clear cached reqs results")
	cmd.Flags().BoolVar(&fixMode, "fix", false, "Attempt to fix PATH issues for missing tools")

	return cmd
}

func runReqs() error {
	// Use orchestrator to execute reqs check with caching support
	return executeReqs()
}

// PrerequisiteChecker handles checking of prerequisites.
type PrerequisiteChecker struct {
	registry map[string]ToolConfig
	aliases  map[string]string
}

// NewPrerequisiteChecker creates a new prerequisite checker.
func NewPrerequisiteChecker() *PrerequisiteChecker {
	return &PrerequisiteChecker{
		registry: toolRegistry,
		aliases:  toolAliases,
	}
}

// Check checks a prerequisite and returns structured result.
func (pc *PrerequisiteChecker) Check(prereq Prerequisite) ReqResult {
	installed, version := pc.getInstalledVersion(prereq)

	result := ReqResult{
		Name:      prereq.Name,
		Installed: installed,
		Version:   version,
		Required:  prereq.MinVersion,
		Satisfied: false,
	}

	if !installed {
		result.Message = "Not installed"
		if !output.IsJSON() {
			output.ItemError("%s: NOT INSTALLED (required: %s)", prereq.Name, prereq.MinVersion)
		}
		return result
	}

	if version == "" {
		result.Message = "Version unknown"
		if !output.IsJSON() {
			output.ItemWarning("%s: INSTALLED (version unknown, required: %s)", prereq.Name, prereq.MinVersion)
		}
		// Continue to check if it's running if needed
	} else {
		versionOk := compareVersions(version, prereq.MinVersion)
		if !versionOk {
			result.Message = fmt.Sprintf("Version %s does not meet minimum %s", version, prereq.MinVersion)
			if !output.IsJSON() {
				output.ItemError("%s: %s (required: %s)", prereq.Name, version, prereq.MinVersion)
			}
			return result
		}
		if !output.IsJSON() {
			output.ItemSuccess("%s: %s (required: %s)", prereq.Name, version, prereq.MinVersion)
		}
	}

	// Check if the tool is running (if configured)
	if prereq.CheckRunning {
		result.CheckedRun = true
		isRunning := pc.checkIsRunning(prereq)
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

// getInstalledVersion gets the installed version of a prerequisite.
func (pc *PrerequisiteChecker) getInstalledVersion(prereq Prerequisite) (installed bool, version string) {
	config := pc.getToolConfig(prereq)

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

// getToolConfig gets the tool configuration for a prerequisite.
func (pc *PrerequisiteChecker) getToolConfig(prereq Prerequisite) ToolConfig {
	// Check if custom configuration is provided in prerequisite
	if prereq.Command != "" {
		return ToolConfig{
			Command:       prereq.Command,
			Args:          prereq.Args,
			VersionPrefix: prereq.VersionPrefix,
			VersionField:  prereq.VersionField,
		}
	}

	// Use registry-based configuration
	tool := prereq.Name

	// Resolve aliases to canonical name
	if canonical, isAlias := pc.aliases[tool]; isAlias {
		tool = canonical
	}

	// Look up tool configuration
	if config, found := pc.registry[tool]; found {
		return config
	}

	// Fallback: try generic --version with tool ID as command
	return ToolConfig{
		Command: prereq.Name,
		Args:    []string{"--version"},
	}
}

// checkIsRunning checks if a prerequisite tool is currently running.
func (pc *PrerequisiteChecker) checkIsRunning(prereq Prerequisite) bool {
	// If no custom running check is configured, use defaults based on tool ID
	command := prereq.RunningCheckCommand
	args := prereq.RunningCheckArgs
	expectedExitCode := 0
	if prereq.RunningCheckExitCode != nil {
		expectedExitCode = *prereq.RunningCheckExitCode
	}

	// Default checks for known tools
	if command == "" {
		switch prereq.Name {
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

// Deprecated: Use PrerequisiteChecker.Check instead
func checkPrerequisite(prereq Prerequisite) bool {
	checker := NewPrerequisiteChecker()
	result := checker.Check(prereq)
	return result.Satisfied
}

// extractVersion extracts version from command output.
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
		return extractAzdVersion(output)
	}

	return extractFirstVersion(output)
}

// extractAzdVersion extracts version from azd multi-line output.
func extractAzdVersion(output string) string {
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
	return ""
}

// extractFirstVersion finds the first semantic version in a string.
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

// compareVersions compares installed version against required version.
// Returns true if installed >= required.
func compareVersions(installed, required string) bool {
	installedParts := parseVersion(installed)
	requiredParts := parseVersion(required)

	// Compare each part left to right
	for i := 0; i < len(requiredParts); i++ {
		if i >= len(installedParts) {
			// Installed version has fewer parts, assume 0
			return false
		}

		if installedParts[i] > requiredParts[i] {
			return true
		}
		if installedParts[i] < requiredParts[i] {
			return false
		}
		// Equal, continue to next part
	}

	return true // All parts equal or installed version is longer
}

// parseVersion parses a version string into numeric parts.
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

// FixResult represents the result of attempting to fix a requirement.
type FixResult struct {
	Name      string `json:"name"`
	Fixed     bool   `json:"fixed"`
	Found     bool   `json:"found"`
	Path      string `json:"path,omitempty"`
	Message   string `json:"message"`
	Satisfied bool   `json:"satisfied"`
}

// runReqsFix attempts to fix PATH issues for missing tools.
func runReqsFix() error {
	if !output.IsJSON() {
		output.Section(output.IconTool, "Attempting to fix requirement issues...")
	}

	// Load azure.yaml
	azureYamlPath, azureYaml, err := loadAzureYaml()
	if err != nil {
		return err
	}

	if len(azureYaml.Reqs) == 0 {
		return fmt.Errorf("no reqs defined in azure.yaml - run 'azd app reqs --generate' to add them")
	}

	// Step 1: Run initial check to identify issues
	initialChecker := NewPrerequisiteChecker()
	var failedReqs []Prerequisite
	for _, prereq := range azureYaml.Reqs {
		result := initialChecker.Check(prereq)
		if !result.Satisfied {
			failedReqs = append(failedReqs, prereq)
		}
	}

	if len(failedReqs) == 0 {
		if output.IsJSON() {
			return output.PrintJSON(map[string]interface{}{
				"success": true,
				"message": "All requirements already satisfied",
			})
		}
		output.Success("All requirements already satisfied!")
		return nil
	}

	// Step 2: Refresh PATH
	if !output.IsJSON() {
		output.Newline()
		output.Step(output.IconRefresh, "Refreshing environment PATH...")
	}

	_, err = pathutil.RefreshPATH()
	if err != nil {
		if !output.IsJSON() {
			output.Warning("Failed to refresh PATH: %v", err)
		}
	} else {
		if !output.IsJSON() {
			output.ItemSuccess("PATH refreshed successfully")
		}
	}

	// Step 3: Try to find and fix each failed requirement
	fixResults := make([]FixResult, 0, len(failedReqs))
	fixedCount := 0

	for _, prereq := range failedReqs {
		if !output.IsJSON() {
			output.Newline()
			output.Step(output.IconSearch, "Searching for %s...", prereq.Name)
		}

		fixResult := FixResult{
			Name:  prereq.Name,
			Fixed: false,
			Found: false,
		}

		// Get the command name to search for
		config := initialChecker.getToolConfig(prereq)
		toolCommand := config.Command

		// Try to find tool in current PATH (after refresh)
		toolPath := pathutil.FindToolInPath(toolCommand)
		if toolPath != "" {
			fixResult.Found = true
			fixResult.Path = toolPath

			// Re-check if it works now
			result := initialChecker.Check(prereq)
			if result.Satisfied {
				fixResult.Fixed = true
				fixResult.Satisfied = true
				fixResult.Message = fmt.Sprintf("Found and verified: %s", toolPath)
				fixedCount++
				if !output.IsJSON() {
					output.ItemSuccess("Found: %s", toolPath)
					output.ItemSuccess("Version verified successfully")
				}
			} else {
				fixResult.Message = fmt.Sprintf("Found at %s but version check failed: %s", toolPath, result.Message)
				if !output.IsJSON() {
					output.ItemWarning("Found: %s", toolPath)
					output.ItemWarning("Version check failed: %s", result.Message)
				}
			}
		} else {
			// Tool not found in PATH, try searching common locations
			toolPath = pathutil.SearchToolInSystemPath(toolCommand)
			if toolPath != "" {
				fixResult.Found = true
				fixResult.Path = toolPath
				fixResult.Message = fmt.Sprintf("Found at %s but not in PATH - restart terminal may be needed", toolPath)
				if !output.IsJSON() {
					output.ItemWarning("Found: %s", toolPath)
					output.ItemWarning("Tool is installed but not in current PATH")
					output.Info("   %s Restart your terminal to update PATH", output.IconBulb)
				}
			} else {
				// Not found anywhere
				suggestion := pathutil.GetInstallSuggestion(toolCommand)
				fixResult.Message = fmt.Sprintf("Not found - %s", suggestion)
				if !output.IsJSON() {
					output.ItemError("Not found in system PATH")
					output.Info("   %s %s", output.IconBulb, suggestion)
				}
			}
		}

		fixResults = append(fixResults, fixResult)
	}

	// Step 4: Invalidate cache so next check gets fresh results
	if fixedCount > 0 {
		// Use same azure.yaml path for cache clearing
		cacheDir := filepath.Join(filepath.Dir(azureYamlPath), ".azure", "cache")
		cacheManager, err := cache.NewCacheManagerWithOptions(cache.CacheOptions{
			Enabled:  true,
			CacheDir: cacheDir,
		})
		if err == nil {
			if err := cacheManager.ClearCache(); err != nil {
				// Log but don't fail on cache clear error
				if !output.IsJSON() {
					output.Warning("Failed to clear cache: %v", err)
				}
			}
		}
	}

	// Step 5: Re-check all requirements
	if !output.IsJSON() {
		output.Newline()
		output.Section(output.IconCheck, "Re-checking requirements...")
	}

	checker := NewPrerequisiteChecker()
	allResults := make([]ReqResult, 0, len(azureYaml.Reqs))
	allSatisfied := true

	for _, prereq := range azureYaml.Reqs {
		result := checker.Check(prereq)
		allResults = append(allResults, result)
		if !result.Satisfied {
			allSatisfied = false
		}
	}

	// JSON output
	if output.IsJSON() {
		return output.PrintJSON(map[string]interface{}{
			"success":      fixedCount > 0,
			"fixed":        fixedCount,
			"total":        len(failedReqs),
			"allSatisfied": allSatisfied,
			"fixes":        fixResults,
			"results":      allResults,
		})
	}

	// Default output - summary
	output.Newline()
	if fixedCount > 0 {
		output.Success("Fixed %d of %d issues!", fixedCount, len(failedReqs))
	} else {
		output.Warning("Could not automatically fix any issues")
	}

	if !allSatisfied {
		output.Newline()
		output.Info("%s Next steps:", output.IconBulb)
		output.Item("1. Run suggested install commands above")
		output.Item("2. Restart your terminal to refresh PATH")
		output.Item("3. Run 'azd app reqs' again to verify")
		return fmt.Errorf("not all requirements satisfied")
	}

	output.Newline()
	output.Success("All requirements now satisfied!")
	output.Newline()
	output.Info("ℹ️  Note: Tools may not be available in THIS terminal session")

	// Provide platform-specific refresh instructions
	if runtime.GOOS == "windows" {
		output.Info("   To refresh PATH in your current PowerShell session, run:")
		output.Info("   %s$env:PATH = [System.Environment]::GetEnvironmentVariable(\"Path\",\"Machine\") + \";\" + [System.Environment]::GetEnvironmentVariable(\"Path\",\"User\")%s", output.Dim, output.Reset)
		output.Info("   Or simply restart your terminal")
	} else {
		output.Info("   To use the tools immediately, restart your terminal or source your shell profile")
	}

	return nil
}
