package commands

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jongio/azd-core/cliout"
)

var allowedCustomPrereqCommands = map[string]struct{}{
	"air":      {},
	"aspire":   {},
	"az":       {},
	"azd":      {},
	"cargo":    {},
	toolDocker: {},
	"dotnet":   {},
	"func":     {},
	"git":      {},
	"go":       {},
	"gradle":   {},
	"java":     {},
	"mvn":      {},
	"node":     {},
	"npm":      {},
	"pip":      {},
	"pipenv":   {},
	"pnpm":     {},
	"poetry":   {},
	"python":   {},
	"uv":       {},
	"yarn":     {},
}

func normalizeAllowedCommandName(command string) string {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" || strings.ContainsAny(trimmed, `\\/`) {
		return ""
	}

	name := strings.ToLower(filepath.Base(trimmed))
	return strings.TrimSuffix(name, ".exe")
}

func isAllowedCustomPrereqCommand(command string) bool {
	name := normalizeAllowedCommandName(command)
	if name == "" {
		return false
	}
	_, allowed := allowedCustomPrereqCommands[name]
	return allowed
}

func validateCustomPrereqCommand(fieldName, command string) error {
	if strings.TrimSpace(command) == "" {
		return nil
	}
	if !isAllowedCustomPrereqCommand(command) {
		return fmt.Errorf("invalid %s %q: only allowlisted tool commands are permitted", fieldName, command)
	}
	return nil
}

// installURLRegistry maps tool names to their installation page URLs.
var installURLRegistry = map[string]string{
	"node":     "https://nodejs.org/",
	"npm":      "https://nodejs.org/",
	"pnpm":     "https://pnpm.io/installation",
	"yarn":     "https://yarnpkg.com/getting-started/install",
	"python":   "https://www.python.org/downloads/",
	"pip":      "https://www.python.org/downloads/",
	"poetry":   "https://python-poetry.org/docs/#installation",
	"uv":       "https://docs.astral.sh/uv/getting-started/installation/",
	"pipenv":   "https://pipenv.pypa.io/en/latest/installation.html",
	"dotnet":   "https://dotnet.microsoft.com/download",
	"aspire":   "https://learn.microsoft.com/dotnet/aspire/fundamentals/setup-tooling",
	toolDocker: "https://www.docker.com/products/docker-desktop",
	"git":      "https://git-scm.com/downloads",
	"go":       "https://go.dev/dl/",
	"azd":      "https://aka.ms/install-azd",
	"az":       "https://aka.ms/installazurecli",
	"air":      "https://github.com/air-verse/air#installation",
	"func":     "https://learn.microsoft.com/azure/azure-functions/functions-run-local#install-the-azure-functions-core-tools",
	"java":     "https://adoptium.net/",
	"mvn":      "https://maven.apache.org/install.html",
	"gradle":   "https://gradle.org/install/",
	"gh":       "https://cli.github.com/",
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
	// Resolve install URL (custom overrides built-in)
	installURL := pc.getInstallURL(prereq)

	result := ReqResult{
		Name:       prereq.Name,
		Required:   prereq.MinVersion,
		Satisfied:  false,
		InstallURL: installURL,
	}

	if err := validateCustomPrereqCommand("command", prereq.Command); err != nil {
		result.Message = err.Error()
		return result
	}
	if prereq.CheckRunning {
		if err := validateCustomPrereqCommand("runningCheckCommand", prereq.RunningCheckCommand); err != nil {
			result.Message = err.Error()
			return result
		}
	}

	installed, version, isPodman := pc.getInstalledVersion(prereq)
	result.Installed = installed
	result.Version = version
	result.IsPodman = isPodman

	if !installed {
		result.Message = "Not installed"
		if !cliout.IsJSON() {
			cliout.ItemError("%s: NOT INSTALLED (required: %s)", prereq.Name, prereq.MinVersion)
			if installURL != "" {
				cliout.Item("   Install: %s", installURL)
			}
		}
		return result
	}

	// When Podman is aliased to Docker, skip version comparison since version schemes differ.
	// Podman uses its own versioning (e.g., 5.7.0) which is not comparable to Docker versions (e.g., 20.10.0).
	if isPodman && prereq.Name == toolDocker {
		result.Message = "Podman detected (version check skipped)"
		if !cliout.IsJSON() {
			cliout.ItemSuccess("%s: %s via Podman (version check skipped)", prereq.Name, version)
		}
		// Continue to check if running if needed, otherwise mark satisfied
		if !prereq.CheckRunning {
			result.Satisfied = true
			return result
		}
	} else if version == "" {
		result.Message = "Version unknown"
		if !cliout.IsJSON() {
			cliout.ItemWarning("%s: INSTALLED (version unknown, required: %s)", prereq.Name, prereq.MinVersion)
		}
		// Continue to check if it's running if needed
	} else {
		versionOk := compareVersions(version, prereq.MinVersion)
		if !versionOk {
			result.Message = fmt.Sprintf("Version %s does not meet minimum %s", version, prereq.MinVersion)
			if !cliout.IsJSON() {
				cliout.ItemError("%s: %s (required: %s)", prereq.Name, version, prereq.MinVersion)
				if installURL != "" {
					cliout.Item("   Install: %s", installURL)
				}
			}
			return result
		}
		if !cliout.IsJSON() {
			cliout.ItemSuccess("%s: %s (required: %s)", prereq.Name, version, prereq.MinVersion)
		}
	}

	// Check if the tool is running (if configured)
	if prereq.CheckRunning {
		result.CheckedRun = true
		isRunning := pc.checkIsRunning(prereq)
		result.Running = isRunning
		if !isRunning {
			result.Message = "Not running"
			if !cliout.IsJSON() {
				cliout.Item("- %s✗%s NOT RUNNING", cliout.Red, cliout.Reset)
			}
			return result
		}
		result.Satisfied = true
		result.Message = "Running"
		if !cliout.IsJSON() {
			cliout.Item("- %s✓%s RUNNING", cliout.Green, cliout.Reset)
		}
		return result
	}

	if version != "" {
		result.Satisfied = true
		result.Message = "Satisfied"
	}
	return result
}

// getInstallURL returns the install URL for a prerequisite.
// Custom InstallURL in prerequisite takes precedence over built-in registry.
func (pc *PrerequisiteChecker) getInstallURL(prereq Prerequisite) string {
	// Custom URL takes precedence
	if prereq.InstallURL != "" {
		return prereq.InstallURL
	}

	// Resolve aliases to canonical name
	tool := prereq.Name
	if canonical, isAlias := pc.aliases[tool]; isAlias {
		tool = canonical
	}

	// Look up in registry
	if url, found := installURLRegistry[tool]; found {
		return url
	}

	return ""
}

// getInstalledVersion gets the installed version of a prerequisite.
// Returns isPodman=true when Podman is detected aliased to Docker.
func (pc *PrerequisiteChecker) getInstalledVersion(prereq Prerequisite) (installed bool, version string, isPodman bool) {
	config := pc.getToolConfig(prereq)

	// #nosec G204 -- Command and args come from toolRegistry or validated azure.yaml prerequisite configuration
	cmd := exec.CommandContext(context.Background(), config.Command, config.Args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, "", false
	}

	outputStr := strings.TrimSpace(string(output))

	// Detect Podman aliased to Docker
	isPodman = strings.Contains(outputStr, "Podman Engine")

	version = extractVersion(config, outputStr)

	return true, version, isPodman
}

// getToolConfig gets the tool configuration for a prerequisite.
func (pc *PrerequisiteChecker) getToolConfig(prereq Prerequisite) ToolConfig {
	// Check if custom configuration is provided in prerequisite
	if prereq.Command != "" {
		return ToolConfig{
			Command:       normalizeAllowedCommandName(prereq.Command),
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
		case toolDocker:
			command = toolDocker
			args = []string{"ps"}
		default:
			// No default running check for this tool
			// Return false to indicate check is not configured properly
			// Users should provide RunningCheckCommand if checkRunning is true
			return false
		}
	}

	if prereq.RunningCheckCommand != "" {
		command = normalizeAllowedCommandName(command)
	}

	// #nosec G204 -- Command and args are validated against an allowlist or come from the built-in Docker check
	cmd := exec.CommandContext(context.Background(), command, args...)
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
