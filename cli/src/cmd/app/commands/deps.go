// Package commands provides the command-line interface for the azd-app CLI.
package commands

import (
	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/spf13/cobra"
)

var (
	depsVerbose bool
	depsClean   bool
	depsNoCache bool
	depsForce   bool
)

// NewDepsCommand creates the deps command.
func NewDepsCommand() *cobra.Command {
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
			if depsForce {
				depsClean = true
				depsNoCache = true
			}

			// Configure cache based on flag
			if depsNoCache {
				SetCacheEnabled(false)
			}
			// Use orchestrator to run deps (which will automatically run reqs first)
			return cmdOrchestrator.Run("deps")
		},
	}

	cmd.Flags().BoolVarP(&depsVerbose, "verbose", "v", false, "Show full installation output")
	cmd.Flags().BoolVar(&depsClean, "clean", false, "Remove existing dependencies before installing (clears node_modules, .venv, etc.)")
	cmd.Flags().BoolVar(&depsNoCache, "no-cache", false, "Force fresh dependency installation and bypass cached results")
	cmd.Flags().BoolVarP(&depsForce, "force", "f", false, "Force clean reinstall (combines --clean and --no-cache)")

	return cmd
}
