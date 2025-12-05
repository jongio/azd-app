// Package commands provides the command-line interface for the azd-app CLI.
package commands

import (
	"sync"

	"github.com/jongio/azd-app/cli/src/internal/output"
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

// depsOptions holds the current deps command options (set by flag parsing).
// Note: This uses sync for thread-safety in concurrent scenarios.
var (
	depsOptions     = &DepsOptions{}
	depsOptionsLock = &sync.Mutex{}
)

// GetDepsOptions returns a copy of the current deps options (for use by executeDeps).
// Returns a copy to prevent external modification of the shared state.
func GetDepsOptions() *DepsOptions {
	depsOptionsLock.Lock()
	defer depsOptionsLock.Unlock()
	// Return a copy to prevent external modification
	optsCopy := *depsOptions
	optsCopy.Services = make([]string, len(depsOptions.Services))
	copy(optsCopy.Services, depsOptions.Services)
	return &optsCopy
}

// setDepsOptions sets the current deps options (thread-safe).
func setDepsOptions(opts *DepsOptions) {
	depsOptionsLock.Lock()
	defer depsOptionsLock.Unlock()
	depsOptions = opts
}

// ResetDepsOptions resets the deps options to defaults (useful for testing).
func ResetDepsOptions() {
	depsOptionsLock.Lock()
	defer depsOptionsLock.Unlock()
	depsOptions = &DepsOptions{}
}

// NewDepsCommand creates the deps command.
func NewDepsCommand() *cobra.Command {
	// Reset options for each command creation to avoid stale state
	opts := &DepsOptions{}

	cmd := &cobra.Command{
		Use:          "deps",
		Short:        "Install dependencies for all detected projects",
		Long:         `Automatically detects and installs dependencies for Node.js (npm/pnpm/yarn), Python (uv/poetry/pip), and .NET projects`,
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
				return output.SetFormat(formatValue)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle --force flag (combines --clean and --no-cache)
			if opts.Force {
				opts.Clean = true
				opts.NoCache = true
			}

			// Set options using thread-safe setter for executeDeps to access
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
