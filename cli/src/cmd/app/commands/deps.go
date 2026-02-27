// Package commands provides the command-line interface for the azd-app CLI.
package commands

import (
	"fmt"
	"os"
	"sync"

	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-core/cliout"
	types "github.com/jongio/azd-core/projecttype"
	"github.com/spf13/cobra"
)

// DepsOptions holds the options for the deps command.
// Using a struct instead of global variables for better testability and concurrency safety.
type DepsOptions struct {
	Verbose  bool
	Clean    bool
	NoCache  bool
	Force    bool
	DryRun   bool     // Show what would be installed without installing
	Services []string // Filter to specific services by name
}

// depsExecutor encapsulates the deps command execution with injectable dependencies.
// This struct enables unit testing of the deps command logic.
type depsExecutor struct {
	// Dependencies (injectable for testing)
	getWorkingDir   func() (string, error)
	detectNode      func(root string) ([]types.NodeProject, error)
	detectPython    func(root string) ([]types.PythonProject, error)
	detectDotnet    func(root string) ([]types.DotnetProject, error)
	detectFunctions func(root string) ([]types.FunctionAppProject, error)

	// Options from flags
	opts *DepsOptions
}

// newDepsExecutor creates a depsExecutor with production dependencies.
func newDepsExecutor(opts *DepsOptions) *depsExecutor {
	return &depsExecutor{
		getWorkingDir:   os.Getwd,
		detectNode:      detector.FindNodeProjects,
		detectPython:    detector.FindPythonProjects,
		detectDotnet:    detector.FindDotnetProjects,
		detectFunctions: detector.FindFunctionApps,
		opts:            opts,
	}
}

// execute runs the deps command with the configured dependencies and options.
func (e *depsExecutor) execute() error {
	cliout.CommandHeader("deps", "Install project dependencies")

	// Determine search root
	searchRoot, err := getSearchRoot()
	if err != nil {
		return handleDepsError(err, "failed to determine search root")
	}

	// Detect projects from azure.yaml services only (no tree walk)
	nodeProjects, pythonProjects, dotnetProjects, err := detectProjectsFromAzureYaml(searchRoot)
	if err != nil {
		return handleDepsError(err, "failed to detect projects from azure.yaml")
	}

	// Apply service filter if specified (further restricts to named services)
	if len(e.opts.Services) > 0 {
		nodeProjects, pythonProjects, dotnetProjects = e.filterProjectsByService(
			nodeProjects, pythonProjects, dotnetProjects, searchRoot)
	}

	totalProjects := len(nodeProjects) + len(pythonProjects) + len(dotnetProjects)

	// Handle no projects case
	if totalProjects == 0 {
		return e.handleNoProjectsCase(searchRoot)
	}

	// Dry-run mode: show what would be installed and exit
	if e.opts.DryRun {
		return showDryRunSummary(nodeProjects, pythonProjects, dotnetProjects, searchRoot)
	}

	// Clean dependencies if requested
	if e.opts.Clean {
		if err := cleanDependencies(nodeProjects, pythonProjects, dotnetProjects); err != nil {
			return fmt.Errorf("failed to clean dependencies: %w", err)
		}
	}

	// Use parallel installer for concurrent installation with progress bars
	if !cliout.IsJSON() {
		return runParallelInstallation(nodeProjects, pythonProjects, dotnetProjects, e.opts.Verbose)
	}

	// JSON mode: use sequential installer
	return runJSONInstallation(searchRoot, nodeProjects, pythonProjects, dotnetProjects)
}

// filterProjectsByService filters projects to only those matching the specified services.
func (e *depsExecutor) filterProjectsByService(
	nodeProjects []types.NodeProject,
	pythonProjects []types.PythonProject,
	dotnetProjects []types.DotnetProject,
	searchRoot string,
) ([]types.NodeProject, []types.PythonProject, []types.DotnetProject) {
	return filterProjectsByService(nodeProjects, pythonProjects, dotnetProjects, e.opts.Services, searchRoot)
}

// handleNoProjectsCase handles the case when no projects are detected.
func (e *depsExecutor) handleNoProjectsCase(searchRoot string) error {
	// If user specified services but none matched, show a helpful message
	if len(e.opts.Services) > 0 {
		msg := fmt.Sprintf("No projects found matching services: %v", e.opts.Services)
		if cliout.IsJSON() {
			return cliout.PrintJSON(DepsResult{
				Success:  true,
				Projects: []InstallResult{},
				Message:  msg,
			})
		}
		cliout.Info("%s", msg)
		return nil
	}

	// Check if there are Logic Apps projects (which don't need dependency installation)
	functionApps, _ := e.detectFunctions(searchRoot)
	hasLogicAppsOnly := false
	if len(functionApps) > 0 {
		hasLogicAppsOnly = true
		for _, app := range functionApps {
			if app.Variant != "logicapps" {
				hasLogicAppsOnly = false
				break
			}
		}
	}

	if cliout.IsJSON() {
		return cliout.PrintJSON(DepsResult{
			Success:  true,
			Projects: []InstallResult{},
			Message:  msgNoProjectsDetected,
		})
	}

	cliout.Info(msgNoProjectsDetected)
	if hasLogicAppsOnly {
		cliout.Item("Logic Apps projects detected (no dependency installation needed)")
	} else {
		cliout.Item("Supported: Node.js (package.json), Python (requirements.txt/pyproject.toml), .NET (*.csproj)")
	}
	return nil
}

// GetDepsOptions is a legacy getter function for backward compatibility.
// Deprecated: Use executor pattern instead.
func GetDepsOptions() *DepsOptions {
	depsOptionsMutex.RLock()
	defer depsOptionsMutex.RUnlock()

	// Return a deep copy to prevent external mutation
	servicesCopy := make([]string, len(globalDepsOptions.Services))
	copy(servicesCopy, globalDepsOptions.Services)

	return &DepsOptions{
		Verbose:  globalDepsOptions.Verbose,
		Clean:    globalDepsOptions.Clean,
		NoCache:  globalDepsOptions.NoCache,
		Force:    globalDepsOptions.Force,
		DryRun:   globalDepsOptions.DryRun,
		Services: servicesCopy,
	}
}

// Global options for backward compatibility (temporary).
var globalDepsOptions = &DepsOptions{}
var depsOptionsMutex sync.RWMutex

// ResetDepsOptions resets the global options to defaults.
// This is primarily used for testing to ensure clean state.
func ResetDepsOptions() {
	depsOptionsMutex.Lock()
	defer depsOptionsMutex.Unlock()
	globalDepsOptions = &DepsOptions{}
}

// setDepsOptions sets the global options (internal use only).
// Creates a deep copy to prevent external mutation.
func setDepsOptions(opts *DepsOptions) {
	depsOptionsMutex.Lock()
	defer depsOptionsMutex.Unlock()

	// Deep copy to prevent mutation
	servicesCopy := make([]string, len(opts.Services))
	copy(servicesCopy, opts.Services)

	globalDepsOptions = &DepsOptions{
		Verbose:  opts.Verbose,
		Clean:    opts.Clean,
		NoCache:  opts.NoCache,
		Force:    opts.Force,
		DryRun:   opts.DryRun,
		Services: servicesCopy,
	}
}

// NewDepsCommand creates the deps command.
func NewDepsCommand() *cobra.Command {
	// Create options for this command invocation
	opts := &DepsOptions{}

	cmd := &cobra.Command{
		Use:          "deps",
		Short:        "Install dependencies for services defined in azure.yaml",
		Long:         `Installs dependencies for services defined in azure.yaml. Only service project paths are checked (Node.js, Python, .NET). Requires azure.yaml with a 'services' section.`,
		SilenceUsage: true,
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
			// Handle --force flag (combines --clean and --no-cache)
			if opts.Force {
				opts.Clean = true
				opts.NoCache = true
			}

			// Set global options for backward compatibility with orchestrator
			setDepsOptions(opts)

			// Configure cache based on flag
			if opts.NoCache {
				SetCacheEnabled(false)
			}
			// Use orchestrator to run deps (which will automatically run reqs first)
			return cmdOrchestrator.Run("deps")
		},
	}

	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Show full installation output")
	cmd.Flags().BoolVar(&opts.Clean, "clean", false, "Remove existing dependencies before installing (clears node_modules, .venv, etc.)")
	cmd.Flags().BoolVar(&opts.NoCache, "no-cache", false, "Force fresh dependency installation and bypass cached results")
	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Force clean reinstall (combines --clean and --no-cache)")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Show what would be installed without actually installing")
	cmd.Flags().StringSliceVarP(&opts.Services, "service", "s", nil, "Install dependencies only for specific services (can be specified multiple times)")

	return cmd
}
