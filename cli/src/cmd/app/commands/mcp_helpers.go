package commands

import (
	"context"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/jongio/azd-core/security"
	"github.com/mark3labs/mcp-go/mcp"
)

// mcpServiceHandler is the common signature for single-service operations
// (start, stop, restart) that take a controller and service name.
type mcpServiceHandler func(ctx context.Context, ctrl *ServiceController, serviceName string) (*mcp.CallToolResult, error)

// handleSingleServiceOp extracts and validates serviceName and projectDir from
// MCP tool args, creates a ServiceController, and delegates to the given handler.
// This eliminates the repetitive boilerplate across start/restart/stop handlers.
func handleSingleServiceOp(ctx context.Context, args azdext.ToolArgs, op mcpServiceHandler) (*mcp.CallToolResult, error) {
	serviceName, err := args.RequireString("serviceName")
	if err != nil {
		return mcpErrorResult("%s", err.Error()), nil
	}

	if valErr := security.ValidateServiceName(serviceName, false); valErr != nil {
		return mcpErrorResult("%s", valErr.Error()), nil
	}

	projectDir, err := extractValidatedProjectDir(args)
	if err != nil {
		return mcpErrorResult("Invalid project directory: %v", err), nil
	}

	ctrl, err := NewServiceController(projectDir)
	if err != nil {
		return mcpErrorResult("Failed to initialize service controller: %v", err), nil
	}

	return op(ctx, ctrl, serviceName)
}
