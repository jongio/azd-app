package commands

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/jongio/azd-app/cli/src/internal/dashboard"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
	"github.com/spf13/cobra"
)

// NewListenCommand creates a new listen command that establishes
// a connection with azd for extension framework operations.
func NewListenCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "listen",
		Short:        "Start the extension server (required by azd framework)",
		Long:         `Internal command used by the azd CLI to communicate with this extension via JSON-RPC over stdio.`,
		Hidden:       true,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create a context with the AZD access token
			ctx := azdext.WithAccessToken(cmd.Context())

			// Create a new AZD client
			azdClient, err := azdext.NewAzdClient()
			if err != nil {
				return fmt.Errorf("failed to create azd client: %w", err)
			}
			defer azdClient.Close()

			// Create an extension host and subscribe to postprovision events
			// This allows us to push real-time updates to the dashboard when azd provision completes
			// Note: We don't filter by Host or Language, so this will apply to all services
			host := azdext.NewExtensionHost(azdClient).
				WithServiceEventHandler("postprovision", handlePostProvision, &azdext.ServiceEventOptions{})

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

// handlePostProvision is called after azd provision completes for each service.
// This handler refreshes the dashboard to show the latest environment values.
func handlePostProvision(ctx context.Context, args *azdext.ServiceEventArgs) error {
	log.Printf("[azd-app] Post-provision event received for service: %s", args.Service.GetName())

	// The environment variables are now updated in the process by azd
	// We just need to trigger a refresh of the cached environment and broadcast to dashboards

	// Get the project directory
	projectDir := args.Project.GetPath()

	// Refresh the environment cache from the current process environment
	// (azd has already updated os.Environ() by the time this handler is called)
	serviceinfo.RefreshEnvironmentCache()
	log.Printf("[azd-app] Refreshed environment cache from updated process environment")

	// Broadcast updated service info to all connected dashboard clients
	srv := dashboard.GetServer(projectDir)
	if srv != nil {
		if err := srv.BroadcastServiceUpdate(projectDir); err != nil {
			log.Printf("[azd-app] Warning: Failed to broadcast service update: %v", err)
		} else {
			log.Printf("[azd-app] Successfully broadcasted environment update to dashboard clients")
		}
	}

	return nil
}
