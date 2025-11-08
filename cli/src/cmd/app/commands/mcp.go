package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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

// runMCPServe starts the MCP server using Go implementation.
func runMCPServe(cmd *cobra.Command, args []string) error {
	return runMCPServer(cmd.Context())
}

// runMCPServer implements the MCP server logic
func runMCPServer(ctx context.Context) error {
	// Create MCP server
	s := server.NewMCPServer(
		"azd-app-mcp-server", "0.1.0",
		server.WithToolCapabilities(true),
	)

	// Add tools
	tools := []server.ServerTool{
		newGetServicesTool(),
		newGetServiceLogsTool(),
		newGetProjectInfoTool(),
		newRunServicesTool(),
		newInstallDependenciesTool(),
		newCheckRequirementsTool(),
	}

	s.AddTools(tools...)

	// Start the server using stdio transport
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		return err
	}

	return nil
}

// executeAzdAppCommand executes an azd app command and returns JSON output
func executeAzdAppCommand(command string, args []string) (map[string]interface{}, error) {
	cmdArgs := append([]string{command}, args...)
	cmdArgs = append(cmdArgs, "--output", "json")

	cmd := exec.Command("azd", append([]string{"app"}, cmdArgs...)...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute azd app %s: %w", command, err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON output: %w", err)
	}

	return result, nil
}

// newGetServicesTool creates the get_services tool
func newGetServicesTool() server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(
			"get_services",
			mcp.WithDescription("Get comprehensive information about all running services in the current azd app project. Returns service status, health, URLs, ports, Azure deployment information, and environment variables."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("projectDir",
				mcp.Description("Optional project directory path. If not provided, uses current directory."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})

			var cmdArgs []string
			if projectDir, ok := args["projectDir"].(string); ok && projectDir != "" {
				cmdArgs = append(cmdArgs, "--project", projectDir)
			}

			result, err := executeAzdAppCommand("info", cmdArgs)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get services: %v", err)), nil
			}

			// Convert result to JSON string
			jsonBytes, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
			}

			return mcp.NewToolResultText(string(jsonBytes)), nil
		},
	}
}

// newGetServiceLogsTool creates the get_service_logs tool
func newGetServiceLogsTool() server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(
			"get_service_logs",
			mcp.WithDescription("Get logs from running services. Can filter by service name, log level, and time range. Supports both recent logs and live streaming."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("serviceName",
				mcp.Description("Optional service name to filter logs. If not provided, shows logs from all services."),
			),
			mcp.WithNumber("tail",
				mcp.Description("Number of recent log lines to retrieve. Default is 100."),
			),
			mcp.WithString("level",
				mcp.Description("Filter by log level: 'info', 'warn', 'error', 'debug', or 'all'. Default is 'all'."),
			),
			mcp.WithString("since",
				mcp.Description("Show logs since duration (e.g., '5m', '1h', '30s'). If provided, overrides tail parameter."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})

			var cmdArgs []string

			if serviceName, ok := args["serviceName"].(string); ok && serviceName != "" {
				cmdArgs = append(cmdArgs, serviceName)
			}

			if tail, ok := args["tail"].(float64); ok && tail > 0 {
				cmdArgs = append(cmdArgs, "--tail", fmt.Sprintf("%.0f", tail))
			}

			if level, ok := args["level"].(string); ok && level != "" && level != "all" {
				cmdArgs = append(cmdArgs, "--level", level)
			}

			if since, ok := args["since"].(string); ok && since != "" {
				cmdArgs = append(cmdArgs, "--since", since)
			}

			// Add format flag for JSON output
			cmdArgs = append(cmdArgs, "--format", "json")

			// Execute logs command
			cmd := exec.Command("azd", append([]string{"app", "logs"}, cmdArgs...)...)
			output, err := cmd.Output()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get logs: %v", err)), nil
			}

			// Parse line-by-line JSON output
			logEntries := []map[string]interface{}{}
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			for _, line := range lines {
				if line == "" {
					continue
				}
				var entry map[string]interface{}
				if err := json.Unmarshal([]byte(line), &entry); err == nil {
					logEntries = append(logEntries, entry)
				}
			}

			// Convert to JSON string
			jsonBytes, err := json.MarshalIndent(logEntries, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal logs: %v", err)), nil
			}

			return mcp.NewToolResultText(string(jsonBytes)), nil
		},
	}
}

// newGetProjectInfoTool creates the get_project_info tool
func newGetProjectInfoTool() server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(
			"get_project_info",
			mcp.WithDescription("Get project metadata and configuration from azure.yaml. Returns project name, directory, and service definitions."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("projectDir",
				mcp.Description("Optional project directory path. If not provided, uses current directory."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})

			var cmdArgs []string
			if projectDir, ok := args["projectDir"].(string); ok && projectDir != "" {
				cmdArgs = append(cmdArgs, "--project", projectDir)
			}

			result, err := executeAzdAppCommand("info", cmdArgs)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get project info: %v", err)), nil
			}

			// Extract just project-level info
			projectInfo := map[string]interface{}{
				"project": result["project"],
			}

			// Extract service metadata (name, language, framework, project path)
			if services, ok := result["services"].([]interface{}); ok {
				simplifiedServices := []map[string]interface{}{}
				for _, svc := range services {
					if svcMap, ok := svc.(map[string]interface{}); ok {
						simplified := map[string]interface{}{
							"name":      svcMap["name"],
							"language":  svcMap["language"],
							"framework": svcMap["framework"],
							"project":   svcMap["project"],
						}
						simplifiedServices = append(simplifiedServices, simplified)
					}
				}
				projectInfo["services"] = simplifiedServices
			}

			// Convert to JSON string
			jsonBytes, err := json.MarshalIndent(projectInfo, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal project info: %v", err)), nil
			}

			return mcp.NewToolResultText(string(jsonBytes)), nil
		},
	}
}

// newRunServicesTool creates the run_services tool
func newRunServicesTool() server.ServerTool {
return server.ServerTool{
Tool: mcp.NewTool(
"run_services",
mcp.WithDescription("Start development services defined in azure.yaml, Aspire, or docker compose. This command will start the application in the background and return information about the started services."),
mcp.WithReadOnlyHintAnnotation(false),
mcp.WithIdempotentHintAnnotation(false),
mcp.WithDestructiveHintAnnotation(false),
mcp.WithString("projectDir",
mcp.Description("Optional project directory path. If not provided, uses current directory."),
),
mcp.WithString("runtime",
mcp.Description("Optional runtime mode: 'azd' (default), 'aspire', 'pnpm', or 'docker-compose'."),
),
),
Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
args, _ := request.Params.Arguments.(map[string]interface{})

var cmdArgs []string

if projectDir, ok := args["projectDir"].(string); ok && projectDir != "" {
cmdArgs = append(cmdArgs, "--project", projectDir)
}

if runtime, ok := args["runtime"].(string); ok && runtime != "" {
cmdArgs = append(cmdArgs, "--runtime", runtime)
}

// Note: azd app run is interactive and long-running, so we run it in a non-blocking way
// and return information about the command being executed
cmd := exec.Command("azd", append([]string{"app", "run"}, cmdArgs...)...)

// Start the command but don't wait for it
if err := cmd.Start(); err != nil {
return mcp.NewToolResultError(fmt.Sprintf("Failed to start services: %v", err)), nil
}

result := map[string]interface{}{
"status":  "started",
"message": "Services are starting in the background. Use get_services to check their status.",
"pid":     cmd.Process.Pid,
}

jsonBytes, err := json.MarshalIndent(result, "", "  ")
if err != nil {
return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
}

return mcp.NewToolResultText(string(jsonBytes)), nil
},
}
}

// newInstallDependenciesTool creates the install_dependencies tool
func newInstallDependenciesTool() server.ServerTool {
return server.ServerTool{
Tool: mcp.NewTool(
"install_dependencies",
mcp.WithDescription("Install dependencies for all detected projects (Node.js, Python, .NET). Automatically detects package managers (npm/pnpm/yarn, uv/poetry/pip, dotnet) and installs dependencies."),
mcp.WithReadOnlyHintAnnotation(false),
mcp.WithIdempotentHintAnnotation(true),
mcp.WithDestructiveHintAnnotation(false),
mcp.WithString("projectDir",
mcp.Description("Optional project directory path. If not provided, uses current directory."),
),
),
Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
args, _ := request.Params.Arguments.(map[string]interface{})

var cmdArgs []string

if projectDir, ok := args["projectDir"].(string); ok && projectDir != "" {
cmdArgs = append(cmdArgs, "--project", projectDir)
}

// Execute deps command
cmd := exec.Command("azd", append([]string{"app", "deps"}, cmdArgs...)...)
output, err := cmd.CombinedOutput()
if err != nil {
return mcp.NewToolResultError(fmt.Sprintf("Failed to install dependencies: %v\nOutput: %s", err, string(output))), nil
}

result := map[string]interface{}{
"status":  "completed",
"message": "Dependencies installed successfully",
"output":  string(output),
}

jsonBytes, err := json.MarshalIndent(result, "", "  ")
if err != nil {
return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
}

return mcp.NewToolResultText(string(jsonBytes)), nil
},
}
}

// newCheckRequirementsTool creates the check_requirements tool
func newCheckRequirementsTool() server.ServerTool {
return server.ServerTool{
Tool: mcp.NewTool(
"check_requirements",
mcp.WithDescription("Check if all required prerequisites (tools, CLIs, SDKs) defined in azure.yaml are installed and meet minimum version requirements. Returns detailed status of each requirement."),
mcp.WithReadOnlyHintAnnotation(true),
mcp.WithIdempotentHintAnnotation(true),
mcp.WithDestructiveHintAnnotation(false),
mcp.WithString("projectDir",
mcp.Description("Optional project directory path. If not provided, uses current directory."),
),
),
Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
args, _ := request.Params.Arguments.(map[string]interface{})

var cmdArgs []string

if projectDir, ok := args["projectDir"].(string); ok && projectDir != "" {
cmdArgs = append(cmdArgs, "--project", projectDir)
}

result, err := executeAzdAppCommand("reqs", cmdArgs)
if err != nil {
return mcp.NewToolResultError(fmt.Sprintf("Failed to check requirements: %v", err)), nil
}

jsonBytes, err := json.MarshalIndent(result, "", "  ")
if err != nil {
return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
}

return mcp.NewToolResultText(string(jsonBytes)), nil
},
}
}
