package commands

import (
	"encoding/json"
	"fmt"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/spf13/cobra"
)

func NewMetadataCommand() *cobra.Command {
	return &cobra.Command{
		Use:    "metadata",
		Short:  "Generate extension metadata",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			rootCmd := cmd.Root()
			metadata := azdext.GenerateExtensionMetadata(
				"1.0",
				"jongio.azd.app",
				rootCmd,
			)

			jsonBytes, err := json.MarshalIndent(metadata, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal metadata: %w", err)
			}

			fmt.Println(string(jsonBytes))
			return nil
		},
	}
}
