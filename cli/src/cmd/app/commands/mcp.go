package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

// NewMCPCommand creates the mcp command with subcommands.
func NewMCPCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "mcp",
		Short:  "Model Context Protocol server operations",
		Long:   `Manage the Model Context Protocol (MCP) server for AI assistant integration`,
		Hidden: true, // Hidden from help - primarily used by azd internally
	}

	cmd.AddCommand(newMCPServeCommand())

	return cmd
}

// newMCPServeCommand creates the mcp serve subcommand.
func newMCPServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP server",
		Long:  `Starts the Model Context Protocol server to expose azd app functionality to AI assistants`,
		RunE:  runMCPServe,
	}
}

// runMCPServe starts the MCP server using the Node.js implementation.
func runMCPServe(cmd *cobra.Command, args []string) error {
	// Get the directory where the extension binary is located
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	exeDir := filepath.Dir(exePath)

	// Look for the MCP server in the extension directory
	// The MCP server should be bundled with the extension
	mcpServerPath := filepath.Join(exeDir, "mcp", "dist", "index.bundle.js")

	// Check if the MCP server exists
	if _, err := os.Stat(mcpServerPath); os.IsNotExist(err) {
		return fmt.Errorf("MCP server not found at %s. Please ensure the extension is properly installed", mcpServerPath)
	}

	// Execute the Node.js MCP server
	// Use node to run the MCP server script
	mcpCmd := exec.Command("node", mcpServerPath)

	// Pass through stdin, stdout, stderr for the MCP protocol
	mcpCmd.Stdin = os.Stdin
	mcpCmd.Stdout = os.Stdout
	mcpCmd.Stderr = os.Stderr

	// Set environment variables that the MCP server might need
	mcpCmd.Env = os.Environ()

	// Run the MCP server (this blocks until the server exits)
	if err := mcpCmd.Run(); err != nil {
		return fmt.Errorf("failed to run MCP server: %w", err)
	}

	return nil
}
