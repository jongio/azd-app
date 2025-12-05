package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/security"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

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
				if err := security.ValidateServiceName(serviceName, true); err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				cmdArgs = append(cmdArgs, serviceName)
			}

			if tail, ok := getFloat64Param(args, "tail"); ok && tail > 0 {
				// Cap tail at reasonable maximum
				if tail > float64(maxLogTailLines) {
					tail = float64(maxLogTailLines)
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
			mcp.WithString("serviceName",
				mcp.Description("Optional specific service to stop. If not provided, stops all running services."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgsMap(request)

			// Get project directory
			projectDir, err := extractValidatedProjectDir(args)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Invalid project directory: %v", err)), nil
			}

			// Create service controller
			ctrl, err := NewServiceController(projectDir)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to initialize service controller: %v", err)), nil
			}

			// Check if a specific service was requested
			if serviceName, ok := getStringParam(args, "serviceName"); ok {
				if err := security.ValidateServiceName(serviceName, false); err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				result := ctrl.StopService(ctx, serviceName)
				return marshalToolResult(result)
			}

			// Stop all running services
			runningServices := ctrl.GetRunningServices()
			if len(runningServices) == 0 {
				return marshalToolResult(BulkServiceControlResult{
					Success: true,
					Message: "No running services to stop",
					Results: []ServiceControlResult{},
				})
			}

			result := ctrl.BulkStop(ctx, runningServices)
			return marshalToolResult(result)
		},
	}
}

// newStartServiceTool creates the start_service tool
func newStartServiceTool() server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(
			"start_service",
			mcp.WithTitleAnnotation("Start Service"),
			mcp.WithDescription("Start a specific stopped service. Use this to start individual services that were previously stopped."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("serviceName",
				mcp.Description("Name of the service to start"),
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
			if err := security.ValidateServiceName(serviceName, false); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			// Get project directory
			projectDir, err := extractValidatedProjectDir(args)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Invalid project directory: %v", err)), nil
			}

			// Create service controller
			ctrl, err := NewServiceController(projectDir)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to initialize service controller: %v", err)), nil
			}

			result := ctrl.StartService(ctx, serviceName)
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
			if err := security.ValidateServiceName(serviceName, false); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			// Get project directory
			projectDir, err := extractValidatedProjectDir(args)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Invalid project directory: %v", err)), nil
			}

			// Create service controller
			ctrl, err := NewServiceController(projectDir)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to initialize service controller: %v", err)), nil
			}

			result := ctrl.RestartService(ctx, serviceName)
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
				if err := security.ValidateServiceName(serviceName, true); err != nil {
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
				if err := security.ValidateServiceName(serviceName, true); err != nil {
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
