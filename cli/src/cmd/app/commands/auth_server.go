package commands

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/authserver"
	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/spf13/cobra"
)

// NewAuthServerCommand creates the auth server command group.
func NewAuthServerCommand() *cobra.Command {
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Manage the authentication server",
		Long:  "Start, stop, or check the status of the authentication server.",
	}

	serverCmd.AddCommand(newAuthServerStartCommand())
	serverCmd.AddCommand(newAuthServerStatusCommand())

	return serverCmd
}

// Server configuration flags
var (
	serverPort          int
	serverEnableTLS     bool
	serverCertFile      string
	serverKeyFile       string
	serverSecret        string
	serverTokenExpiry   int
	serverBindAddress   string
	serverRateLimitReqs int
)

func newAuthServerStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the authentication server",
		Long: `Start the authentication server to provide Azure tokens to client containers.

The server uses your existing Azure credentials (from 'azd auth login') to obtain
Azure access tokens and distributes them securely to authenticated clients.

Security:
- Clients must provide a shared secret (via Authorization header)
- TLS encryption is recommended for production use
- Rate limiting prevents abuse
- Tokens are short-lived and automatically refreshed

Example:
  # Start server with default settings (HTTP on port 8080)
  azd app auth server start --secret mysecret

  # Start with TLS enabled
  azd app auth server start --secret mysecret --tls --cert server.crt --key server.key

  # Start on custom port with custom token expiry
  azd app auth server start --secret mysecret --port 9000 --token-expiry 1800
`,
		RunE: runAuthServerStart,
	}

	// Add flags
	cmd.Flags().IntVar(&serverPort, "port", 8080, "Server port")
	cmd.Flags().BoolVar(&serverEnableTLS, "tls", false, "Enable TLS/HTTPS")
	cmd.Flags().StringVar(&serverCertFile, "cert", "", "TLS certificate file path")
	cmd.Flags().StringVar(&serverKeyFile, "key", "", "TLS key file path")
	cmd.Flags().StringVar(&serverSecret, "secret", "", "Shared secret for authentication (or set AZD_AUTH_SECRET env var)")
	cmd.Flags().IntVar(&serverTokenExpiry, "token-expiry", 900, "Token expiry in seconds (default: 900 = 15 minutes)")
	cmd.Flags().StringVar(&serverBindAddress, "bind", "0.0.0.0", "Network interface to bind to")
	cmd.Flags().IntVar(&serverRateLimitReqs, "rate-limit", 10, "Max requests per minute per client")

	return cmd
}

func runAuthServerStart(cmd *cobra.Command, args []string) error {
	// Get secret from environment if not provided via flag
	if serverSecret == "" {
		serverSecret = os.Getenv("AZD_AUTH_SECRET")
	}

	if serverSecret == "" {
		return fmt.Errorf("shared secret is required (use --secret flag or AZD_AUTH_SECRET environment variable)")
	}

	// Create server configuration
	config := authserver.DefaultConfig()
	config.Port = serverPort
	config.EnableTLS = serverEnableTLS
	config.CertFile = serverCertFile
	config.KeyFile = serverKeyFile
	config.SharedSecret = serverSecret
	config.TokenExpiry = secondsToDuration(serverTokenExpiry)
	config.BindAddress = serverBindAddress
	config.RateLimitRequests = serverRateLimitReqs

	// Validate configuration
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create and start server
	server, err := authserver.NewServer(config)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	if !output.IsJSON() {
		output.Section("🔐", "Starting authentication server")
		output.Info("Port: %d", config.Port)
		output.Info("TLS: %v", config.EnableTLS)
		output.Info("Token Expiry: %d seconds", serverTokenExpiry)
		output.Info("Bind Address: %s", config.BindAddress)
		output.Info("Rate Limit: %d requests/minute", config.RateLimitRequests)
		output.Newline()
	}

	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	protocol := "http"
	if config.EnableTLS {
		protocol = "https"
	}

	if output.IsJSON() {
		return output.PrintJSON(map[string]interface{}{
			"status": "running",
			"url":    server.GetURL(),
			"config": map[string]interface{}{
				"port":         config.Port,
				"tls":          config.EnableTLS,
				"bind_address": config.BindAddress,
			},
		})
	}

	output.Success("Authentication server started successfully!")
	output.Info("Server URL: %s", output.URL(server.GetURL()))
	output.Newline()
	output.Info("📊 Endpoints:")
	output.Item("  GET %s://%s:%d/token?scope=<scope>", protocol, config.BindAddress, config.Port)
	output.Item("  GET %s://%s:%d/health", protocol, config.BindAddress, config.Port)
	output.Newline()
	output.Info("🛑 Press Ctrl+C to stop the server")
	output.Newline()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for interrupt signal
	<-sigChan

	output.Newline()
	output.Warning("Shutting down server...")

	if err := server.Stop(); err != nil {
		return fmt.Errorf("failed to stop server: %w", err)
	}

	output.Success("Server stopped successfully")
	return nil
}

func newAuthServerStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check authentication server status",
		Long:  "Check if the authentication server is running and healthy.",
		RunE:  runAuthServerStatus,
	}

	cmd.Flags().StringVar(&serverSecret, "server", "http://localhost:8080", "Server URL to check")

	return cmd
}

func runAuthServerStatus(cmd *cobra.Command, args []string) error {
	// This is a simple health check command
	// In a production implementation, you might want to:
	// - Check if the server process is running
	// - Verify connectivity
	// - Check system resources

	if output.IsJSON() {
		return output.PrintJSON(map[string]interface{}{
			"message": "Use the client health check to verify server status",
			"example": fmt.Sprintf("azd app auth token get --server %s --secret <your-secret> --health-check", serverSecret),
		})
	}

	output.Info("To check server status, use the client health check:")
	output.Item("azd app auth token get --server %s --secret <your-secret> --health-check", serverSecret)

	return nil
}

// Helper function to convert seconds to Duration
func secondsToDuration(seconds int) time.Duration {
	return time.Duration(seconds) * time.Second
}
