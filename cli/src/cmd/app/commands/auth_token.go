package commands

import (
	"fmt"
	"os"

	"github.com/jongio/azd-app/cli/src/internal/authclient"
	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/spf13/cobra"
)

// NewAuthTokenCommand creates the auth token command group.
func NewAuthTokenCommand() *cobra.Command {
	tokenCmd := &cobra.Command{
		Use:   "token",
		Short: "Manage authentication tokens",
		Long:  "Fetch tokens from the authentication server.",
	}

	tokenCmd.AddCommand(newAuthTokenGetCommand())

	return tokenCmd
}

// Token configuration flags
var (
	tokenServerURL   string
	tokenSecret      string
	tokenScope       string
	tokenHealthCheck bool
)

func newAuthTokenGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get an access token from the auth server",
		Long: `Fetch an Azure access token from the authentication server.

The token is obtained from a centralized auth server that has Azure credentials,
eliminating the need for each client container to have its own credentials.

The fetched token can be used with Azure SDKs or CLI tools to access Azure resources.

Example:
  # Get token for Azure Resource Manager
  azd app auth token get --server http://auth-server:8080 --secret mysecret

  # Get token for a specific scope
  azd app auth token get --server http://auth-server:8080 --secret mysecret \
    --scope https://storage.azure.com/.default

  # Check server health
  azd app auth token get --server http://auth-server:8080 --secret mysecret --health-check

  # Use in a script
  export TOKEN=$(azd app auth token get --server http://auth-server:8080 --secret mysecret -o json | jq -r .access_token)
`,
		RunE: runAuthTokenGet,
	}

	// Add flags
	cmd.Flags().StringVar(&tokenServerURL, "server", "", "Authentication server URL (or set AUTH_SERVER_URL env var)")
	cmd.Flags().StringVar(&tokenSecret, "secret", "", "Shared secret for authentication (or set AZD_AUTH_SECRET env var)")
	cmd.Flags().StringVar(&tokenScope, "scope", "https://management.azure.com/.default", "Token scope")
	cmd.Flags().BoolVar(&tokenHealthCheck, "health-check", false, "Only check server health, don't fetch token")

	return cmd
}

func runAuthTokenGet(cmd *cobra.Command, args []string) error {
	// Get server URL from environment if not provided
	if tokenServerURL == "" {
		tokenServerURL = os.Getenv("AUTH_SERVER_URL")
	}

	if tokenServerURL == "" {
		return fmt.Errorf("server URL is required (use --server flag or AUTH_SERVER_URL environment variable)")
	}

	// Get secret from environment if not provided
	if tokenSecret == "" {
		tokenSecret = os.Getenv("AZD_AUTH_SECRET")
	}

	if tokenSecret == "" {
		return fmt.Errorf("shared secret is required (use --secret flag or AZD_AUTH_SECRET environment variable)")
	}

	// Create client configuration
	config := authclient.DefaultConfig(tokenServerURL, tokenSecret)

	// Create client
	client, err := authclient.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// If health check only, check health and return
	if tokenHealthCheck {
		if err := client.HealthCheck(); err != nil {
			if output.IsJSON() {
				return output.PrintJSON(map[string]interface{}{
					"healthy": false,
					"error":   err.Error(),
				})
			}
			return fmt.Errorf("health check failed: %w", err)
		}

		if output.IsJSON() {
			return output.PrintJSON(map[string]interface{}{
				"healthy": true,
				"server":  tokenServerURL,
			})
		}

		output.Success("Authentication server is healthy")
		output.Info("Server: %s", tokenServerURL)
		return nil
	}

	// Fetch token
	if !output.IsJSON() {
		output.Section("🔐", "Fetching token from auth server")
		output.Info("Server: %s", tokenServerURL)
		output.Info("Scope: %s", tokenScope)
		output.Newline()
	}

	token, err := client.GetToken(tokenScope)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	if output.IsJSON() {
		return output.PrintJSON(map[string]interface{}{
			"access_token": token,
			"scope":        tokenScope,
			"server":       tokenServerURL,
		})
	}

	output.Success("Token fetched successfully!")
	output.Newline()
	output.Info("Access Token:")
	
	// Print token (truncated for security in terminal)
	if len(token) > 50 {
		output.Item("%s...%s", token[:25], token[len(token)-25:])
	} else {
		output.Item("%s", token)
	}
	
	output.Newline()
	output.Info("💡 Tip: Use '-o json' to get the full token in scripts")
	output.Item("export TOKEN=$(azd app auth token get --server %s --secret <secret> -o json | jq -r .access_token)", tokenServerURL)

	return nil
}
