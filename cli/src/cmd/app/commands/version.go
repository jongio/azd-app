package commands

import (
	coreversion "github.com/jongio/azd-core/version"
	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
var Version = "dev"

// BuildTime is set at build time via -ldflags.
var BuildTime = "unknown"

// Commit is set at build time via -ldflags.
var Commit = "unknown"

// VersionInfo provides the shared version information for this extension.
var VersionInfo = coreversion.New("jongio.azd.app", "azd app")

func init() {
	VersionInfo.Version = Version
	VersionInfo.BuildDate = BuildTime
	VersionInfo.GitCommit = Commit
}

// NewVersionCommand creates the version command.
func NewVersionCommand(outputFormat *string) *cobra.Command {
	return coreversion.NewCommand(VersionInfo, outputFormat)
}
