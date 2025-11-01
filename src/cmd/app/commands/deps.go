package commands

import (
	"github.com/spf13/cobra"
)

// NewDepsCommand creates the deps command.
func NewDepsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "deps",
		Short: "Install dependencies for all detected projects",
		Long:  `Automatically detects and installs dependencies for Node.js (npm/pnpm/yarn), Python (uv/poetry/pip), and .NET projects`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdOrchestrator.Run("deps")
		},
	}
}
