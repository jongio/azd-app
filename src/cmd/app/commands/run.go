package commands

import (
	"github.com/spf13/cobra"
)

// NewRunCommand creates the run command.
func NewRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run the development environment (Aspire, pnpm, or docker compose)",
		Long:  `Automatically detects and runs the appropriate development command: Aspire (AppHost.cs), pnpm dev/start scripts, or docker compose from package.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdOrchestrator.Run("run")
		},
	}
}
