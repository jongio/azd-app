package commands

import (
	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/spf13/cobra"
)

// NewMetadataCommand creates the metadata command using the azdext SDK helper.
// rootCmdProvider returns the root command for introspection.
func NewMetadataCommand(rootCmdProvider func() *cobra.Command) *cobra.Command {
	return azdext.NewMetadataCommand("1.0", "jongio.azd.app", rootCmdProvider)
}
