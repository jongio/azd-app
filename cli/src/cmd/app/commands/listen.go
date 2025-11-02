package commands

import (
	"fmt"
	"os"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/spf13/cobra"
)

// NewListenCommand creates a new listen command that establishes
// a connection with azd for extension framework operations.
func NewListenCommand() *cobra.Command {
	return &cobra.Command{
		Use:    "listen",
		Short:  "Starts the extension and listens for events",
		Hidden: true, // Hidden from help - only invoked by azd internally
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create a context with the AZD access token
			ctx := azdext.WithAccessToken(cmd.Context())

			// Create a new AZD client
			azdClient, err := azdext.NewAzdClient()
			if err != nil {
				return fmt.Errorf("failed to create azd client: %w", err)
			}
			defer azdClient.Close()

			// Create an extension host with NO capabilities registered
			// This tells azd we only support custom-commands (declared in extension.yaml)
			// and prevents azd from trying to invoke us for service-target-provider
			host := azdext.NewExtensionHost(azdClient)

			// Start the extension host
			// This blocks until azd closes the connection
			if err := host.Run(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "Extension host error: %v\n", err)
				return fmt.Errorf("failed to run extension: %w", err)
			}

			return nil
		},
	}
}
