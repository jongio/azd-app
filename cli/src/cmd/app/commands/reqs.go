package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/jongio/azd-core/cliout"

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
	// Install URL configuration (optional)
	InstallURL string `yaml:"installUrl,omitempty"` // URL to installation page (overrides built-in)
}

// ReqsService represents a minimal service definition for reqs parsing.
// Only includes fields needed to detect container services.
type ReqsService struct {
	Image  string            `yaml:"image,omitempty"`
	Docker *ReqsDockerConfig `yaml:"docker,omitempty"`
}

// ReqsDockerConfig represents minimal Docker configuration for reqs parsing.
type ReqsDockerConfig struct {
	Image string `yaml:"image,omitempty"`
}

// AzureYaml represents the structure of azure.yaml.
type AzureYaml struct {
	Reqs     []Prerequisite         `yaml:"reqs"`
	Services map[string]ReqsService `yaml:"services,omitempty"`
}

const (
	toolDocker = "docker"
	osWindows  = "windows"
)

// hasContainerServices returns true if any service is a container service.
func (a *AzureYaml) hasContainerServices() bool {
	for _, svc := range a.Services {
		if svc.Image != "" {
			return true
		}
		if svc.Docker != nil && svc.Docker.Image != "" {
			return true
		}
	}
	return false
}

// hasDockerReq returns true if Docker is already in the reqs list.
func (a *AzureYaml) hasDockerReq() bool {
	for _, req := range a.Reqs {
		if strings.EqualFold(req.Name, toolDocker) {
			return true
		}
	}
	return false
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
	IsPodman   bool   `json:"isPodman,omitempty"`   // True when Podman is aliased to Docker
	InstallURL string `json:"installUrl,omitempty"` // URL to installation page
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
	toolDocker: {
		Command:      toolDocker,
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
		Args:    []string{"version", "--output", "tsv", "--query", `"azure-cli"`},
	},
	"air": {
		Command:       "air",
		Args:          []string{"-v"},
		VersionPrefix: "v",
	},
	"func": {
		Command: "func",
		Args:    []string{"--version"},
	},
	"java": {
		Command:      "java",
		Args:         []string{"-version"},
		VersionField: 2, // "java version \"17.0.1\" ..." or "openjdk version \"17.0.1\" ..." -> take field 2
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
		Use:          "reqs",
		Short:        "Check for required reqs",
		SilenceUsage: true,
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
				return cliout.SetFormat(formatValue)
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

			return newCommandOrchestrator().Run("reqs")
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
