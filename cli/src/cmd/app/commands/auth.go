package commands

import (
	"github.com/spf13/cobra"
)

// NewAuthCommand creates the auth command group.
func NewAuthCommand() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication server for container-to-container communication",
		Long: `The auth command group provides tools for secure authentication in containerized environments.

It enables you to:
- Run an authentication server that provides Azure tokens to containers
- Fetch tokens from the auth server in client containers
- Enable secure container-to-container communication without duplicating credentials

This is useful for distributed applications running in Docker, Podman, or Kubernetes.`,
	}

	// Add subcommands
	authCmd.AddCommand(NewAuthServerCommand())
	authCmd.AddCommand(NewAuthTokenCommand())

	return authCmd
}
