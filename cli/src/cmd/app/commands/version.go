package commands

import (
	"github.com/jongio/azd-core/cliout"
	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
var Version = "dev"

// BuildTime is set at build time via -ldflags.
var BuildTime = "unknown"

// Commit is set at build time via -ldflags.
var Commit = "unknown"

// VersionInfo represents version information for JSON cliout.
type VersionInfo struct {
	Version   string `json:"version"`
	BuildTime string `json:"buildTime"`
}

// NewVersionCommand creates the version command.
func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "version",
		Short:        "Show version information",
		Long:         `Display the version of the azd app extension.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// JSON output
			if cliout.IsJSON() {
				return cliout.PrintJSON(VersionInfo{
					Version:   Version,
					BuildTime: BuildTime,
				})
			}

			// Default output
			cliout.CommandHeader("version", "Show version information")
			cliout.Label("Version", Version)
			cliout.Label("Built", BuildTime)
			return nil
		},
	}
}
