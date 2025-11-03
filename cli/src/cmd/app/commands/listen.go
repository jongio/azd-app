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

			// Create an extension host and subscribe to environment update events
			// This allows us to push real-time updates to the dashboard when azd provision completes
			host := azdext.NewExtensionHost(azdClient).
				WithServiceEventHandler("environment updated", handleEnvironmentUpdate, nil)

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

// handleEnvironmentUpdate is called when azd environment variables are updated (e.g., after provision).
// This handler refreshes the dashboard to show the latest environment values.
func handleEnvironmentUpdate(ctx context.Context, args *azdext.ServiceEventArgs) error {
	log.Printf("[azd-app] Environment updated event received for service: %s", args.Service.GetName())

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
