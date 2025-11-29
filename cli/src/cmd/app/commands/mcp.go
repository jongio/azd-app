package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

// Timeout constants
const (
	defaultCommandTimeout    = 30 * time.Second
	dependencyInstallTimeout = 5 * time.Minute
)

// Command constants
const (
	azdCommand     = "azd"
	appSubcommand  = "app"
	jsonOutputFlag = "--output"
	jsonOutputVal  = "json"
	cwdFlag        = "--cwd"
	projectFlag    = "--project"
)

// Allowed values for validation
var (
	allowedLogLevels = map[string]bool{"info": true, "warn": true, "error": true, "debug": true, "all": true}
	allowedRuntimes  = map[string]bool{"azd": true, "aspire": true, "pnpm": true, "docker-compose": true}
	// safeNamePattern validates service names and other identifiers to prevent injection
	safeNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)
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
	// System instructions to guide AI on how to use the tools
	// This server is part of the azd extension framework and provides runtime operations
	instructions := `This MCP server is provided by the azd app extension and focuses on runtime operations for azd projects.

**Extension Role:**
This server complements azd's core MCP capabilities:
- azd core MCP: Project planning, architecture discovery, infrastructure generation
- azd app MCP (this): Runtime operations, service monitoring, log analysis

**Best Practices:**
1. Always use get_services to check current state before starting/stopping services
2. Use check_requirements before installing dependencies to see what's needed
3. Use get_service_logs to diagnose issues when services fail to start
4. Read azure.yaml resource to understand project structure before operations

**Tool Categories:**
- Observability: get_services, get_service_logs, get_project_info
- Operations: run_services, stop_services, restart_service, install_dependencies
- Configuration: check_requirements, get_environment_variables, set_environment_variable

**Integration Notes:**
- Works with projects created by azd init or azd templates
- Monitors services started by azd app run
- Complements azd's built-in deployment and provisioning workflows`

	// Create MCP server with all capabilities
	// Server name follows azd extension naming convention: {namespace}-mcp-server
	s := server.NewMCPServer(
		"app-mcp-server", "0.1.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(false, true), // subscribe=false, listChanged=true
		server.WithPromptCapabilities(false),         // listChanged=false
		server.WithInstructions(instructions),
	)

	// Add tools
	tools := []server.ServerTool{
		// Observability tools
		newGetServicesTool(),
		newGetServiceLogsTool(),
		newGetProjectInfoTool(),
		// Operational tools
		newRunServicesTool(),
		newStopServicesTool(),
		newRestartServiceTool(),
		newInstallDependenciesTool(),
		newCheckRequirementsTool(),
		// Configuration tools
		newGetEnvironmentVariablesTool(),
		newSetEnvironmentVariableTool(),
	}

	s.AddTools(tools...)

	// Add resources
	resources := []server.ServerResource{
		newAzureYamlResource(),
		newServiceConfigResource(),
	}

	s.AddResources(resources...)

	// Start the server using stdio transport
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		return err
	}

	return nil
}

// executeAzdAppCommand executes an azd app command and returns JSON output
func executeAzdAppCommand(ctx context.Context, command string, args []string) (map[string]interface{}, error) {
	return executeAzdAppCommandWithTimeout(ctx, command, args, defaultCommandTimeout)
}

// executeAzdAppCommandWithTimeout executes an azd app command with a custom timeout
func executeAzdAppCommandWithTimeout(ctx context.Context, command string, args []string, timeout time.Duration) (map[string]interface{}, error) {
	// Check if context is already cancelled
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("command cancelled before execution: %w", err)
	}

	cmdArgs := append([]string{command}, args...)
	cmdArgs = append(cmdArgs, jsonOutputFlag, jsonOutputVal)

	// Use context with timeout for command execution
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, azdCommand, append([]string{appSubcommand}, cmdArgs...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if parent context was cancelled
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil, fmt.Errorf("command cancelled: %w", ctx.Err())
		}
		// Check if command context timed out
		if errors.Is(cmdCtx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("command timed out after %v", timeout)
		}
		return nil, fmt.Errorf("failed to execute azd app %s: %w\nOutput: %s", command, err, string(output))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON output: %w\nRaw output: %s", err, string(output))
	}

	return result, nil
}

// Helper functions for common MCP patterns

// getArgsMap safely extracts the arguments map from a request
func getArgsMap(request mcp.CallToolRequest) map[string]interface{} {
	if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
		return args
	}
	return make(map[string]interface{})
}

// getStringParam safely extracts a string parameter from request arguments
func getStringParam(args map[string]interface{}, key string) (string, bool) {
	if val, ok := args[key].(string); ok && val != "" {
		return val, true
	}
	return "", false
}

// getFloat64Param safely extracts a float64 parameter from request arguments
func getFloat64Param(args map[string]interface{}, key string) (float64, bool) {
	if val, ok := args[key].(float64); ok {
		return val, true
	}
	return 0, false
}

// marshalToolResult marshals data to JSON and returns an MCP tool result
func marshalToolResult(data interface{}) (*mcp.CallToolResult, error) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// extractProjectDirArg extracts projectDir argument and returns command args with --cwd flag
// Returns the args and any validation error
func extractProjectDirArg(args map[string]interface{}) ([]string, error) {
	var cmdArgs []string
	if projectDir, ok := getStringParam(args, "projectDir"); ok {
		validatedPath, err := validateProjectDir(projectDir)
		if err != nil {
			return nil, err
		}
		cmdArgs = append(cmdArgs, cwdFlag, validatedPath)
	}
	return cmdArgs, nil
}

// validateRequiredParam validates that a required parameter exists and returns the value
func validateRequiredParam(args map[string]interface{}, key string) (string, error) {
	val, ok := args[key].(string)
	if !ok || val == "" {
		return "", fmt.Errorf("%s parameter is required", key)
	}
	return val, nil
}

// validateEnumParam validates that a parameter value is in allowed set
func validateEnumParam(value string, allowed map[string]bool, paramName string) error {
	if value == "" {
		return nil // Empty is OK for optional params
	}
	if !allowed[value] {
		var validValues []string
		for k := range allowed {
			validValues = append(validValues, k)
		}
		return fmt.Errorf("invalid %s: '%s'. Valid values: %s", paramName, value, strings.Join(validValues, ", "))
	}
	return nil
}

// isValidDuration validates a duration string format (e.g., "5m", "1h", "30s")
func isValidDuration(s string) bool {
	if s == "" {
		return false
	}
	// Simple validation: must end with s, m, or h and have a numeric prefix
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return false
	}
	suffix := s[len(s)-1]
	if suffix != 's' && suffix != 'm' && suffix != 'h' {
		return false
	}
	prefix := s[:len(s)-1]
	for _, c := range prefix {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(prefix) > 0
}

// validateServiceName validates that a service name is safe and doesn't contain injection characters
func validateServiceName(name string) error {
	if name == "" {
		return nil // Empty is OK for optional params
	}
	if len(name) > 128 {
		return fmt.Errorf("service name too long (max 128 characters)")
	}
	if !safeNamePattern.MatchString(name) {
		return fmt.Errorf("invalid service name: must start with alphanumeric and contain only alphanumeric, underscore, or hyphen")
	}
	return nil
}

// validateProjectDir validates that the project directory path is safe
// Prevents path traversal attacks and ensures the directory exists
func validateProjectDir(dir string) (string, error) {
	if dir == "" || dir == "." {
		return ".", nil
	}

	// Clean the path to resolve any . or .. components
	cleanPath := filepath.Clean(dir)

	// Get absolute path
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("invalid project directory path: %w", err)
	}

	// Check if directory exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("project directory does not exist: %s", absPath)
		}
		return "", fmt.Errorf("cannot access project directory: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", absPath)
	}

	return absPath, nil
}

// getProjectDir gets the project directory from AZD_APP_PROJECT_DIR environment variable or defaults to current directory
// This environment variable is set by azd when invoking the extension's MCP server
func getProjectDir() string {
	// First try AZD_APP_PROJECT_DIR (set via extension.yaml mcp.serve.env)
	projectDir := os.Getenv("AZD_APP_PROJECT_DIR")
	if projectDir == "" {
		// Fall back to PROJECT_DIR for backwards compatibility
		projectDir = os.Getenv("PROJECT_DIR")
	}
	if projectDir == "" {
		projectDir = "."
	}
	return projectDir
}

// ServiceInfo represents the output schema for get_services tool
type ServiceInfo struct {
	Project  map[string]interface{} `json:"project" jsonschema:"description=Project metadata including name and directory"`
	Services []ServiceDetails       `json:"services" jsonschema:"description=List of services with their status and configuration"`
}

// ServiceDetails represents individual service information
type ServiceDetails struct {
	Name      string            `json:"name" jsonschema:"description=Service name"`
	Language  string            `json:"language" jsonschema:"description=Programming language (e.g. python, javascript, dotnet)"`
	Framework string            `json:"framework" jsonschema:"description=Framework used (e.g. flask, express, aspnet)"`
	Project   string            `json:"project" jsonschema:"description=Path to the project directory"`
	Status    string            `json:"status,omitempty" jsonschema:"description=Current running status"`
	Health    string            `json:"health,omitempty" jsonschema:"description=Health check status"`
	URL       string            `json:"url,omitempty" jsonschema:"description=Local URL where service is running"`
	Port      int               `json:"port,omitempty" jsonschema:"description=Port number the service is listening on"`
	PID       int               `json:"pid,omitempty" jsonschema:"description=Process ID of the running service"`
	Env       map[string]string `json:"env,omitempty" jsonschema:"description=Environment variables configured for the service"`
}

// ProjectInfo represents the output schema for get_project_info tool
type ProjectInfo struct {
	Project  map[string]interface{}  `json:"project" jsonschema:"description=Project metadata"`
	Services []ProjectServiceSummary `json:"services" jsonschema:"description=Summary of services defined in the project"`
}

// ProjectServiceSummary represents a simplified service summary
type ProjectServiceSummary struct {
	Name      string `json:"name" jsonschema:"description=Service name"`
	Language  string `json:"language" jsonschema:"description=Programming language"`
	Framework string `json:"framework" jsonschema:"description=Framework used"`
	Project   string `json:"project" jsonschema:"description=Project directory path"`
}

// RequirementsResult represents the output schema for check_requirements tool
type RequirementsResult struct {
	Requirements []RequirementStatus `json:"requirements" jsonschema:"description=List of requirements and their status"`
	AllMet       bool                `json:"allMet" jsonschema:"description=Whether all requirements are satisfied"`
}

// RequirementStatus represents individual requirement check result
type RequirementStatus struct {
	Name           string `json:"name" jsonschema:"description=Requirement name (e.g. node, python, docker)"`
	Required       string `json:"required" jsonschema:"description=Required version"`
	Installed      string `json:"installed,omitempty" jsonschema:"description=Installed version if found"`
	Met            bool   `json:"met" jsonschema:"description=Whether requirement is satisfied"`
	InstallCommand string `json:"installCommand,omitempty" jsonschema:"description=Command to install if missing"`
}

// newGetServicesTool creates the get_services tool
func newGetServicesTool() server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(
			"get_services",
			mcp.WithTitleAnnotation("Get Running Services"),
			mcp.WithDescription("Get comprehensive information about all running services in the current azd app project. Returns service status, health, URLs, ports, Azure deployment information, and environment variables."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOutputSchema[ServiceInfo](),
			mcp.WithString("projectDir",
				mcp.Description("Optional project directory path. If not provided, uses current directory."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgsMap(request)

			cmdArgs, err := extractProjectDirArg(args)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Invalid project directory: %v", err)), nil
			}

			result, err := executeAzdAppCommand(ctx, "info", cmdArgs)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get services: %v", err)), nil
			}

			return marshalToolResult(result)
		},
	}
}

// newGetServiceLogsTool creates the get_service_logs tool
func newGetServiceLogsTool() server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(
			"get_service_logs",
			mcp.WithTitleAnnotation("Get Service Logs"),
			mcp.WithDescription("Get logs from running services. Can filter by service name, log level, and time range. Supports both recent logs and live streaming."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("projectDir",
				mcp.Description("Optional project directory path. If not provided, uses current directory."),
			),
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
			args := getArgsMap(request)

			// Start with --cwd if projectDir is specified
			cmdArgs, err := extractProjectDirArg(args)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Invalid project directory: %v", err)), nil
			}

			if serviceName, ok := getStringParam(args, "serviceName"); ok {
				// Validate service name to prevent injection
				if err := validateServiceName(serviceName); err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				cmdArgs = append(cmdArgs, serviceName)
			}

			if tail, ok := getFloat64Param(args, "tail"); ok && tail > 0 {
				// Cap tail at reasonable maximum
				if tail > 10000 {
					tail = 10000
				}
				cmdArgs = append(cmdArgs, "--tail", fmt.Sprintf("%.0f", tail))
			}

			if level, ok := getStringParam(args, "level"); ok {
				// Validate level parameter
				if err := validateEnumParam(level, allowedLogLevels, "level"); err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				if level != "all" {
					cmdArgs = append(cmdArgs, "--level", level)
				}
			}

			if since, ok := getStringParam(args, "since"); ok {
				// Validate since format (should be like 5m, 1h, 30s)
				if !isValidDuration(since) {
					return mcp.NewToolResultError("Invalid 'since' format. Use duration like '5m', '1h', '30s'"), nil
				}
				cmdArgs = append(cmdArgs, "--since", since)
			}

			// Add format flag for JSON output
			cmdArgs = append(cmdArgs, "--format", "json")

			// Check context before starting
			if err := ctx.Err(); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Request cancelled: %v", err)), nil
			}

			// Execute logs command with context
			cmdCtx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
			defer cancel()

			cmd := exec.CommandContext(cmdCtx, azdCommand, append([]string{appSubcommand, "logs"}, cmdArgs...)...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				if errors.Is(ctx.Err(), context.Canceled) {
					return mcp.NewToolResultError("Request was cancelled"), nil
				}
				if errors.Is(cmdCtx.Err(), context.DeadlineExceeded) {
					return mcp.NewToolResultError(fmt.Sprintf("Command timed out after %v", defaultCommandTimeout)), nil
				}
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get logs: %v\nOutput: %s", err, string(output))), nil
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

			return marshalToolResult(logEntries)
		},
	}
}

// newGetProjectInfoTool creates the get_project_info tool
func newGetProjectInfoTool() server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(
			"get_project_info",
			mcp.WithTitleAnnotation("Get Project Information"),
			mcp.WithDescription("Get project metadata and configuration from azure.yaml. Returns project name, directory, and service definitions."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOutputSchema[ProjectInfo](),
			mcp.WithString("projectDir",
				mcp.Description("Optional project directory path. If not provided, uses current directory."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgsMap(request)

			cmdArgs, err := extractProjectDirArg(args)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Invalid project directory: %v", err)), nil
			}

			result, err := executeAzdAppCommand(ctx, "info", cmdArgs)
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

			return marshalToolResult(projectInfo)
		},
	}
}

// newRunServicesTool creates the run_services tool
func newRunServicesTool() server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(
			"run_services",
			mcp.WithTitleAnnotation("Run Development Services"),
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
			args := getArgsMap(request)

			cmdArgs, err := extractProjectDirArg(args)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Invalid project directory: %v", err)), nil
			}

			if runtime, ok := getStringParam(args, "runtime"); ok {
				// Validate runtime parameter
				if err := validateEnumParam(runtime, allowedRuntimes, "runtime"); err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				cmdArgs = append(cmdArgs, "--runtime", runtime)
			}

			// Note: azd app run is interactive and long-running, so we run it in a non-blocking way
			// and return information about the command being executed
			// The context is intentionally NOT used here because the process should continue running
			cmd := exec.Command(azdCommand, append([]string{appSubcommand, "run"}, cmdArgs...)...)

			// Detach process from current process group so it survives after MCP server exits
			cmd.Stdout = nil
			cmd.Stderr = nil
			cmd.Stdin = nil

			// Start the command but don't wait for it
			if err := cmd.Start(); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to start services: %v", err)), nil
			}

			// Capture PID immediately after Start() to avoid race
			// cmd.Process is guaranteed to be set after successful Start()
			pid := cmd.Process.Pid

			// Release the process so it's not a zombie when parent exits
			// The process will be orphaned and adopted by init/systemd
			go func() {
				_ = cmd.Wait() // Ignore error, just clean up zombie
			}()

			result := map[string]interface{}{
				"status":  "started",
				"message": "Services are starting in the background. Use get_services to check their status.",
				"pid":     pid,
			}

			return marshalToolResult(result)
		},
	}
}

// newInstallDependenciesTool creates the install_dependencies tool
func newInstallDependenciesTool() server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(
			"install_dependencies",
			mcp.WithTitleAnnotation("Install Project Dependencies"),
			mcp.WithDescription("Install dependencies for all detected projects (Node.js, Python, .NET). Automatically detects package managers (npm/pnpm/yarn, uv/poetry/pip, dotnet) and installs dependencies."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("projectDir",
				mcp.Description("Optional project directory path. If not provided, uses current directory."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgsMap(request)

			cmdArgs, err := extractProjectDirArg(args)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Invalid project directory: %v", err)), nil
			}

			// Check context before starting long operation
			if err := ctx.Err(); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Request cancelled: %v", err)), nil
			}

			// Use longer timeout for dependency installation (can be slow)
			cmdCtx, cancel := context.WithTimeout(ctx, dependencyInstallTimeout)
			defer cancel()

			// Execute deps command with context
			cmd := exec.CommandContext(cmdCtx, azdCommand, append([]string{appSubcommand, "deps"}, cmdArgs...)...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				if errors.Is(ctx.Err(), context.Canceled) {
					return mcp.NewToolResultError("Request was cancelled"), nil
				}
				if errors.Is(cmdCtx.Err(), context.DeadlineExceeded) {
					return mcp.NewToolResultError(fmt.Sprintf("Dependency installation timed out after %v", dependencyInstallTimeout)), nil
				}
				return mcp.NewToolResultError(fmt.Sprintf("Failed to install dependencies: %v\nOutput: %s", err, string(output))), nil
			}

			result := map[string]interface{}{
				"status":  "completed",
				"message": "Dependencies installed successfully",
				"output":  string(output),
			}

			return marshalToolResult(result)
		},
	}
}

// newCheckRequirementsTool creates the check_requirements tool
func newCheckRequirementsTool() server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(
			"check_requirements",
			mcp.WithTitleAnnotation("Check Prerequisites"),
			mcp.WithDescription("Check if all required prerequisites (tools, CLIs, SDKs) defined in azure.yaml are installed and meet minimum version requirements. Returns detailed status of each requirement."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOutputSchema[RequirementsResult](),
			mcp.WithString("projectDir",
				mcp.Description("Optional project directory path. If not provided, uses current directory."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgsMap(request)

			cmdArgs, err := extractProjectDirArg(args)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Invalid project directory: %v", err)), nil
			}

			result, err := executeAzdAppCommand(ctx, "reqs", cmdArgs)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to check requirements: %v", err)), nil
			}

			return marshalToolResult(result)
		},
	}
}

// newStopServicesTool creates the stop_services tool
func newStopServicesTool() server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(
			"stop_services",
			mcp.WithTitleAnnotation("Stop Running Services"),
			mcp.WithDescription("Stop all running development services. This will gracefully shut down services started with run_services."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("projectDir",
				mcp.Description("Optional project directory path. If not provided, uses current directory."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Note: azd app doesn't have a direct stop command, so we provide guidance
			result := map[string]interface{}{
				"status":  "info",
				"message": "To stop services, use Ctrl+C in the terminal running 'azd app run', or use system tools to kill the process.",
				"tip":     "You can use get_services to find the PID of running services.",
			}

			return marshalToolResult(result)
		},
	}
}

// newRestartServiceTool creates the restart_service tool
func newRestartServiceTool() server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(
			"restart_service",
			mcp.WithTitleAnnotation("Restart Service"),
			mcp.WithDescription("Restart a specific service. This will stop and start the specified service."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("serviceName",
				mcp.Description("Name of the service to restart"),
				mcp.Required(),
			),
			mcp.WithString("projectDir",
				mcp.Description("Optional project directory path. If not provided, uses current directory."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgsMap(request)

			serviceName, err := validateRequiredParam(args, "serviceName")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			// Validate service name to prevent injection
			if err := validateServiceName(serviceName); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			result := map[string]interface{}{
				"status":  "info",
				"message": fmt.Sprintf("To restart service '%s', first stop it (Ctrl+C or kill PID), then use run_services to start it again.", serviceName),
				"tip":     "Use get_services to find the current PID of the service.",
			}

			return marshalToolResult(result)
		},
	}
}

// newGetEnvironmentVariablesTool creates the get_environment_variables tool
func newGetEnvironmentVariablesTool() server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(
			"get_environment_variables",
			mcp.WithTitleAnnotation("Get Environment Variables"),
			mcp.WithDescription("Get environment variables configured for services. Returns all environment variables that services will use."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("serviceName",
				mcp.Description("Optional service name to filter environment variables. If not provided, returns all."),
			),
			mcp.WithString("projectDir",
				mcp.Description("Optional project directory path. If not provided, uses current directory."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgsMap(request)

			cmdArgs, err := extractProjectDirArg(args)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Invalid project directory: %v", err)), nil
			}

			// Validate service name if provided
			serviceName, hasFilter := getStringParam(args, "serviceName")
			if hasFilter {
				if err := validateServiceName(serviceName); err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
			}

			// Get service info which includes environment variables
			result, err := executeAzdAppCommand(ctx, "info", cmdArgs)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get environment variables: %v", err)), nil
			}

			// Extract environment variables from services
			envVars := make(map[string]interface{})
			if services, ok := result["services"].([]interface{}); ok {
				for _, svc := range services {
					if svcMap, ok := svc.(map[string]interface{}); ok {
						svcName, _ := svcMap["name"].(string)

						// Skip if filtering and name doesn't match
						if hasFilter && svcName != serviceName {
							continue
						}

						if env, ok := svcMap["env"].(map[string]interface{}); ok {
							envVars[svcName] = env
						}
					}
				}
			}

			return marshalToolResult(envVars)
		},
	}
}

// newSetEnvironmentVariableTool creates the set_environment_variable tool
func newSetEnvironmentVariableTool() server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(
			"set_environment_variable",
			mcp.WithTitleAnnotation("Set Environment Variable"),
			mcp.WithDescription("Set an environment variable for services. Note: This provides guidance on how to set environment variables, as they must be configured in azure.yaml or .env files."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("name",
				mcp.Description("Name of the environment variable"),
				mcp.Required(),
			),
			mcp.WithString("value",
				mcp.Description("Value of the environment variable"),
				mcp.Required(),
			),
			mcp.WithString("serviceName",
				mcp.Description("Optional service name. If not provided, applies to all services."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgsMap(request)

			name, err := validateRequiredParam(args, "name")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			// Validate env var name format
			if !safeNamePattern.MatchString(name) {
				return mcp.NewToolResultError("Invalid environment variable name: must start with alphanumeric and contain only alphanumeric, underscore, or hyphen"), nil
			}

			value, err := validateRequiredParam(args, "value")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			serviceName, _ := getStringParam(args, "serviceName")
			if serviceName != "" {
				if err := validateServiceName(serviceName); err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
			} else {
				serviceName = "<service-name>"
			}

			guidance := fmt.Sprintf(`To set environment variable '%s=%s':

**Option 1: Update azure.yaml**
Add to the service configuration:
services:
  %s:
    env:
      %s: "%s"

**Option 2: Use .env file**
Create/update .env file in project root:
%s=%s

**Option 3: System environment**
Export in your shell:
export %s="%s"

After updating, restart services for changes to take effect.`,
				name, value,
				serviceName, name, value,
				name, value,
				name, value)

			result := map[string]interface{}{
				"status":   "guidance",
				"message":  guidance,
				"variable": name,
				"value":    value,
			}

			return marshalToolResult(result)
		},
	}
}

// newAzureYamlResource creates a resource for reading azure.yaml
func newAzureYamlResource() server.ServerResource {
	return server.ServerResource{
		Resource: mcp.NewResource(
			"azure://project/azure.yaml",
			"azure.yaml",
			mcp.WithResourceDescription("The azure.yaml configuration file that defines the project structure, services, and dependencies."),
			mcp.WithAnnotations([]mcp.Role{mcp.RoleUser, mcp.RoleAssistant}, 0.9),
			mcp.WithMIMEType("application/x-yaml"),
		),
		Handler: func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			// Check context
			if err := ctx.Err(); err != nil {
				return nil, fmt.Errorf("request cancelled: %w", err)
			}

			// Get and validate project directory
			projectDir := getProjectDir()
			validatedDir, err := validateProjectDir(projectDir)
			if err != nil {
				return nil, fmt.Errorf("invalid project directory: %w", err)
			}

			// Find and read azure.yaml from project directory
			azureYamlPath := filepath.Join(validatedDir, "azure.yaml")

			// Verify the file path is still within the project directory (defense in depth)
			cleanPath := filepath.Clean(azureYamlPath)
			if !strings.HasPrefix(cleanPath, validatedDir) && validatedDir != "." {
				return nil, fmt.Errorf("azure.yaml path escapes project directory")
			}

			content, err := os.ReadFile(cleanPath)
			if err != nil {
				if os.IsNotExist(err) {
					return nil, fmt.Errorf("azure.yaml not found in project directory: %s", validatedDir)
				}
				return nil, fmt.Errorf("failed to read azure.yaml: %w", err)
			}

			return []mcp.ResourceContents{
				&mcp.TextResourceContents{
					URI:      request.Params.URI,
					Text:     string(content),
					MIMEType: "application/x-yaml",
				},
			}, nil
		},
	}
}

// newServiceConfigResource creates a resource for reading service configurations
func newServiceConfigResource() server.ServerResource {
	return server.ServerResource{
		Resource: mcp.NewResource(
			"azure://project/services/configs",
			"service-configs",
			mcp.WithResourceDescription("Configuration details for all services including environment variables, ports, and runtime settings."),
			mcp.WithAnnotations([]mcp.Role{mcp.RoleAssistant}, 0.7),
			mcp.WithMIMEType("application/json"),
		),
		Handler: func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			// Check context
			if err := ctx.Err(); err != nil {
				return nil, fmt.Errorf("request cancelled: %w", err)
			}

			// Get service configurations from project directory
			var cmdArgs []string
			projectDir := getProjectDir()
			if projectDir != "." {
				validatedDir, err := validateProjectDir(projectDir)
				if err != nil {
					return nil, fmt.Errorf("invalid project directory: %w", err)
				}
				cmdArgs = append(cmdArgs, cwdFlag, validatedDir)
			}

			result, err := executeAzdAppCommand(ctx, "info", cmdArgs)
			if err != nil {
				return nil, fmt.Errorf("failed to get service configs: %w", err)
			}

			// Extract just the configuration parts (not runtime status)
			configs := make(map[string]interface{})
			if services, ok := result["services"].([]interface{}); ok {
				for _, svc := range services {
					if svcMap, ok := svc.(map[string]interface{}); ok {
						svcName, _ := svcMap["name"].(string)
						if svcName == "" {
							continue // Skip services without names
						}
						config := map[string]interface{}{
							"name":      svcMap["name"],
							"language":  svcMap["language"],
							"framework": svcMap["framework"],
							"project":   svcMap["project"],
							"env":       svcMap["env"],
						}
						configs[svcName] = config
					}
				}
			}

			jsonBytes, err := json.MarshalIndent(configs, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to marshal configs: %w", err)
			}

			return []mcp.ResourceContents{
				&mcp.TextResourceContents{
					URI:      request.Params.URI,
					Text:     string(jsonBytes),
					MIMEType: "application/json",
				},
			}, nil
		},
	}
}
