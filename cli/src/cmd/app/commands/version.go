package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
var Version = "dev"

// BuildTime is set at build time via -ldflags.
var BuildTime = "unknown"

// NewVersionCommand creates the version command.
func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Display the version of the azd app extension.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("azd app extension\n")
			fmt.Printf("  Version: %s\n", Version)
			fmt.Printf("  Built: %s\n", BuildTime)
			return nil
		},
	}
}
