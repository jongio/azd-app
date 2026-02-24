package commands

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"syscall"
	"time"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/security"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerAllTools registers all MCP tools on the builder.
// Rate limiting and ToolArgs parsing are handled automatically by the builder.
func registerAllTools(b *azdext.MCPServerBuilder) {
	// Observability tools
	addGetServicesTool(b)
	addGetServiceLogsTool(b)
	addGetServiceErrorsTool(b)
	addGetProjectInfoTool(b)
	// Operational tools
	addRunServicesTool(b)
	addStopServicesTool(b)
	addStartServiceTool(b)
	addRestartServiceTool(b)
	addInstallDependenciesTool(b)
	addCheckRequirementsTool(b)
	// Configuration tools
	addGetEnvironmentVariablesTool(b)
	addSetEnvironmentVariableTool(b)
}

// --- get_services ---

func addGetServicesTool(b *azdext.MCPServerBuilder) {
	b.AddTool("get_services", handleGetServices,
		azdext.MCPToolOptions{
			Title:       "Get Running Services",
			Description: "Get comprehensive information about all running services in the current azd app project. Returns service status, health, URLs, ports, Azure deployment information, and environment variables.",
			ReadOnly:    true,
			Idempotent:  true,
		},
		mcp.WithOutputSchema[ServiceInfo](),
		mcp.WithString("projectDir",
			mcp.Description("Optional project directory path. If not provided, uses current directory."),
		),
	)
}

func handleGetServices(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
	cmdArgs, err := extractProjectDirArg(args)
	if err != nil {
		return azdext.MCPErrorResult("Invalid project directory: %v", err), nil
	}

	result, err := executeAzdAppCommand(ctx, "info", cmdArgs)
	if err != nil {
		return azdext.MCPErrorResult("Failed to get services: %v", err), nil
	}

	return marshalToolResult(result)
}

// --- get_service_logs ---

func addGetServiceLogsTool(b *azdext.MCPServerBuilder) {
	b.AddTool("get_service_logs", handleGetServiceLogs,
		azdext.MCPToolOptions{
			Title:       "Get Service Logs",
			Description: "Get logs from running services. Can filter by service name, log level, and time range. Supports both local and Azure cloud logs via the source parameter.",
			ReadOnly:    true,
			Idempotent:  true,
		},
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
		mcp.WithString("source",
			mcp.Description("Log source: 'local' for locally running services, 'azure' for Azure cloud services, 'both' for combined logs. Default is 'local'."),
		),
	)
}

func handleGetServiceLogs(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
	opts := &logsOptions{source: "local", tail: 100, level: "all"}
	var serviceArgs []string
	var projectDir string

	if pd := args.OptionalString("projectDir", ""); pd != "" {
		validated, valErr := validateProjectDir(pd)
		if valErr != nil {
			return azdext.MCPErrorResult("Invalid project directory: %v", valErr), nil
		}
		projectDir = validated
	}

	if serviceName := args.OptionalString("serviceName", ""); serviceName != "" {
		if valErr := security.ValidateServiceName(serviceName, true); valErr != nil {
			return azdext.MCPErrorResult("%s", valErr.Error()), nil
		}
		serviceArgs = append(serviceArgs, serviceName)
	}

	if tail := args.OptionalInt("tail", 0); tail > 0 {
		if tail > maxLogTailLines {
			tail = maxLogTailLines
		}
		opts.tail = tail
	}

	if level := args.OptionalString("level", ""); level != "" {
		if valErr := validateEnumParam(level, allowedLogLevels, "level"); valErr != nil {
			return azdext.MCPErrorResult("%s", valErr.Error()), nil
		}
		opts.level = level
	}

	if since := args.OptionalString("since", ""); since != "" {
		if !isValidDuration(since) {
			return azdext.MCPErrorResult("Invalid 'since' format. Use duration like '5m', '1h', '30s'"), nil
		}
		opts.since = since
	}

	if source := args.OptionalString("source", ""); source != "" {
		allowedSources := map[string]bool{"local": true, "azure": true, "both": true}
		if valErr := validateEnumParam(source, allowedSources, "source"); valErr != nil {
			return azdext.MCPErrorResult("%s", valErr.Error()), nil
		}
		if source == "both" {
			source = "all"
		}
		opts.source = source
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		return azdext.MCPErrorResult("Request cancelled: %v", ctxErr), nil
	}

	collectCtx, collectCancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer collectCancel()

	executor := newLogsExecutorForMCP(opts, projectDir)
	collected, err := executor.collect(collectCtx, serviceArgs)
	if err != nil {
		if collectCtx.Err() == context.DeadlineExceeded {
			return azdext.MCPErrorResult("Command timed out after %v", defaultCommandTimeout), nil
		}
		return azdext.MCPErrorResult("Failed to get logs: %v", err), nil
	}

	return marshalToolResult(collected.Entries)
}

// --- get_service_errors ---

func addGetServiceErrorsTool(b *azdext.MCPServerBuilder) {
	b.AddTool("get_service_errors", handleGetServiceErrors,
		azdext.MCPToolOptions{
			Title:       "Get Service Errors",
			Description: "Get error logs from services with surrounding context for debugging. Optimized for AI-assisted troubleshooting - returns only errors with relevant context to help diagnose issues quickly. Uses the logs command filtered to error level.",
			ReadOnly:    true,
			Idempotent:  true,
		},
		mcp.WithString("projectDir",
			mcp.Description("Optional project directory path. If not provided, uses current directory."),
		),
		mcp.WithString("serviceName",
			mcp.Description("Optional service name to filter errors. If not provided, shows errors from all services."),
		),
		mcp.WithString("since",
			mcp.Description("Show errors since duration (e.g., '5m', '1h', '30s'). Default is '10m'."),
		),
		mcp.WithNumber("tail",
			mcp.Description("Number of log lines to retrieve. Default is 500."),
		),
		mcp.WithNumber("contextLines",
			mcp.Description("Number of log lines before and after each error for context. Default is 3, max is 10."),
		),
	)
}

func handleGetServiceErrors(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
	opts := &logsOptions{
		source:       "local",
		tail:         500,
		level:        "error",
		contextLines: service.DefaultContextLines,
		since:        "10m",
	}
	var serviceArgs []string
	var projectDir string

	if pd := args.OptionalString("projectDir", ""); pd != "" {
		validated, valErr := validateProjectDir(pd)
		if valErr != nil {
			return azdext.MCPErrorResult("Invalid project directory: %v", valErr), nil
		}
		projectDir = validated
	}

	if serviceName := args.OptionalString("serviceName", ""); serviceName != "" {
		if valErr := security.ValidateServiceName(serviceName, true); valErr != nil {
			return azdext.MCPErrorResult("%s", valErr.Error()), nil
		}
		serviceArgs = append(serviceArgs, serviceName)
	}

	if s := args.OptionalString("since", ""); s != "" {
		if !isValidDuration(s) {
			return azdext.MCPErrorResult("Invalid 'since' format. Use duration like '5m', '1h', '30s'"), nil
		}
		opts.since = s
	}

	if t := args.OptionalInt("tail", 0); t > 0 {
		if t > maxLogTailLines {
			t = maxLogTailLines
		}
		opts.tail = t
	}

	if cl := args.OptionalInt("contextLines", -1); cl >= 0 {
		if cl > service.MaxContextLines {
			cl = service.MaxContextLines
		}
		opts.contextLines = cl
	}

	if err := ctx.Err(); err != nil {
		return azdext.MCPErrorResult("Request cancelled: %v", err), nil
	}

	collectCtx, collectCancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer collectCancel()

	executor := newLogsExecutorForMCP(opts, projectDir)
	collected, err := executor.collect(collectCtx, serviceArgs)
	if err != nil {
		if collectCtx.Err() == context.DeadlineExceeded {
			return azdext.MCPErrorResult("Command timed out after %v", defaultCommandTimeout), nil
		}
		return azdext.MCPErrorResult("Failed to get errors: %v", err), nil
	}

	entries := collected.EntriesWithContext
	result := map[string]interface{}{
		"summary": map[string]interface{}{
			"totalErrors": len(entries),
			"since":       opts.since,
		},
		"errors": entries,
	}

	return marshalToolResult(result)
}

// --- get_project_info ---

func addGetProjectInfoTool(b *azdext.MCPServerBuilder) {
	b.AddTool("get_project_info", handleGetProjectInfo,
		azdext.MCPToolOptions{
			Title:       "Get Project Information",
			Description: "Get project metadata and configuration from azure.yaml. Returns project name, directory, and service definitions.",
			ReadOnly:    true,
			Idempotent:  true,
		},
		mcp.WithOutputSchema[ProjectInfo](),
		mcp.WithString("projectDir",
			mcp.Description("Optional project directory path. If not provided, uses current directory."),
		),
	)
}

func handleGetProjectInfo(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
	cmdArgs, err := extractProjectDirArg(args)
	if err != nil {
		return azdext.MCPErrorResult("Invalid project directory: %v", err), nil
	}

	result, err := executeAzdAppCommand(ctx, "info", cmdArgs)
	if err != nil {
		return azdext.MCPErrorResult("Failed to get project info: %v", err), nil
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
}

// --- run_services ---

func addRunServicesTool(b *azdext.MCPServerBuilder) {
	b.AddTool("run_services", handleRunServices,
		azdext.MCPToolOptions{
			Title:       "Run Development Services",
			Description: "Start development services defined in azure.yaml, Aspire, or docker compose. This command will start the application in the background and return information about the started services.",
		},
		mcp.WithString("projectDir",
			mcp.Description("Optional project directory path. If not provided, uses current directory."),
		),
		mcp.WithString("runtime",
			mcp.Description("Optional runtime mode: 'azd' (default), 'aspire', 'pnpm', or 'docker-compose'."),
		),
	)
}

func handleRunServices(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
	cmdArgs, err := extractProjectDirArg(args)
	if err != nil {
		return azdext.MCPErrorResult("Invalid project directory: %v", err), nil
	}

	if runtime := args.OptionalString("runtime", ""); runtime != "" {
		if valErr := validateEnumParam(runtime, allowedRuntimes, "runtime"); valErr != nil {
			return azdext.MCPErrorResult("%s", valErr.Error()), nil
		}
		cmdArgs = append(cmdArgs, "--runtime", runtime)
	}

	// Note: azd app run is interactive and long-running, so we run it in a non-blocking way
	// and return information about the command being executed
	// The context is intentionally NOT used here because the process should continue running
	cmd := exec.Command(azdCommand, append([]string{appSubcommand, "run"}, cmdArgs...)...)

	// Create pipes to capture startup errors without blocking
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return azdext.MCPErrorResult("Failed to create stderr pipe: %v", err), nil
	}

	if err := cmd.Start(); err != nil {
		return azdext.MCPErrorResult("Failed to start services: %v", err), nil
	}

	// Capture PID immediately after Start() to avoid race
	pid := 0
	processStarted := false
	if cmd.Process != nil {
		pid = cmd.Process.Pid
		processStarted = true
	}

	// Check for immediate startup failures (first 100ms)
	startupErrChan := make(chan string, 1)
	go func() {
		defer func() {
			_ = stderrPipe.Close()
		}()

		buf := make([]byte, 4096)
		n, _ := stderrPipe.Read(buf)
		if n > 0 {
			startupErrChan <- string(buf[:n])
		}
	}()

	time.Sleep(100 * time.Millisecond)

	if processStarted {
		if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
			select {
			case errMsg := <-startupErrChan:
				return azdext.MCPErrorResult("Service failed to start: %s", errMsg), nil
			default:
				return azdext.MCPErrorResult("Service failed to start immediately"), nil
			}
		}
	}

	// Release the process so it's not a zombie when parent exits
	go func() {
		_ = cmd.Wait()
	}()

	result := map[string]interface{}{
		"status":  "started",
		"message": "Services are starting in the background. Use get_services to check their status.",
	}
	if pid > 0 {
		result["pid"] = pid
	}

	return marshalToolResult(result)
}

// --- stop_services ---

func addStopServicesTool(b *azdext.MCPServerBuilder) {
	b.AddTool("stop_services", handleStopServices,
		azdext.MCPToolOptions{
			Title:       "Stop Running Services",
			Description: "Stop all running development services. This will gracefully shut down services started with run_services.",
			Idempotent:  true,
		},
		mcp.WithString("projectDir",
			mcp.Description("Optional project directory path. If not provided, uses current directory."),
		),
		mcp.WithString("serviceName",
			mcp.Description("Optional specific service to stop. If not provided, stops all running services."),
		),
	)
}

func handleStopServices(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
	projectDir, err := extractValidatedProjectDir(args)
	if err != nil {
		return azdext.MCPErrorResult("Invalid project directory: %v", err), nil
	}

	ctrl, err := NewServiceController(projectDir)
	if err != nil {
		return azdext.MCPErrorResult("Failed to initialize service controller: %v", err), nil
	}

	if serviceName := args.OptionalString("serviceName", ""); serviceName != "" {
		if valErr := security.ValidateServiceName(serviceName, false); valErr != nil {
			return azdext.MCPErrorResult("%s", valErr.Error()), nil
		}
		result := ctrl.StopService(ctx, serviceName)
		return marshalToolResult(result)
	}

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
}

// --- start_service ---

func addStartServiceTool(b *azdext.MCPServerBuilder) {
	b.AddTool("start_service", handleStartService,
		azdext.MCPToolOptions{
			Title:       "Start Service",
			Description: "Start a specific stopped service. Use this to start individual services that were previously stopped.",
		},
		mcp.WithString("serviceName",
			mcp.Description("Name of the service to start"),
			mcp.Required(),
		),
		mcp.WithString("projectDir",
			mcp.Description("Optional project directory path. If not provided, uses current directory."),
		),
	)
}

func handleStartService(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
	serviceName, err := args.RequireString("serviceName")
	if err != nil {
		return azdext.MCPErrorResult("%s", err.Error()), nil
	}

	if valErr := security.ValidateServiceName(serviceName, false); valErr != nil {
		return azdext.MCPErrorResult("%s", valErr.Error()), nil
	}

	projectDir, err := extractValidatedProjectDir(args)
	if err != nil {
		return azdext.MCPErrorResult("Invalid project directory: %v", err), nil
	}

	ctrl, err := NewServiceController(projectDir)
	if err != nil {
		return azdext.MCPErrorResult("Failed to initialize service controller: %v", err), nil
	}

	result := ctrl.StartService(ctx, serviceName)
	return marshalToolResult(result)
}

// --- restart_service ---

func addRestartServiceTool(b *azdext.MCPServerBuilder) {
	b.AddTool("restart_service", handleRestartService,
		azdext.MCPToolOptions{
			Title:       "Restart Service",
			Description: "Restart a specific service. This will stop and start the specified service.",
		},
		mcp.WithString("serviceName",
			mcp.Description("Name of the service to restart"),
			mcp.Required(),
		),
		mcp.WithString("projectDir",
			mcp.Description("Optional project directory path. If not provided, uses current directory."),
		),
	)
}

func handleRestartService(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
	serviceName, err := args.RequireString("serviceName")
	if err != nil {
		return azdext.MCPErrorResult("%s", err.Error()), nil
	}

	if valErr := security.ValidateServiceName(serviceName, false); valErr != nil {
		return azdext.MCPErrorResult("%s", valErr.Error()), nil
	}

	projectDir, err := extractValidatedProjectDir(args)
	if err != nil {
		return azdext.MCPErrorResult("Invalid project directory: %v", err), nil
	}

	ctrl, err := NewServiceController(projectDir)
	if err != nil {
		return azdext.MCPErrorResult("Failed to initialize service controller: %v", err), nil
	}

	result := ctrl.RestartService(ctx, serviceName)
	return marshalToolResult(result)
}

// --- install_dependencies ---

func addInstallDependenciesTool(b *azdext.MCPServerBuilder) {
	b.AddTool("install_dependencies", handleInstallDependencies,
		azdext.MCPToolOptions{
			Title:       "Install Project Dependencies",
			Description: "Install dependencies for all detected projects (Node.js, Python, .NET). Automatically detects package managers (npm/pnpm/yarn, uv/poetry/pip, dotnet) and installs dependencies.",
			Idempotent:  true,
		},
		mcp.WithString("projectDir",
			mcp.Description("Optional project directory path. If not provided, uses current directory."),
		),
	)
}

func handleInstallDependencies(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
	cmdArgs, err := extractProjectDirArg(args)
	if err != nil {
		return azdext.MCPErrorResult("Invalid project directory: %v", err), nil
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		return azdext.MCPErrorResult("Request cancelled: %v", ctxErr), nil
	}

	cmdCtx, cancel := context.WithTimeout(ctx, dependencyInstallTimeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, azdCommand, append([]string{appSubcommand, "deps"}, cmdArgs...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return azdext.MCPErrorResult("Request was cancelled"), nil
		}
		if errors.Is(cmdCtx.Err(), context.DeadlineExceeded) {
			return azdext.MCPErrorResult("Dependency installation timed out after %v", dependencyInstallTimeout), nil
		}
		return azdext.MCPErrorResult("Failed to install dependencies: %v\nOutput: %s", err, string(output)), nil
	}

	result := map[string]interface{}{
		"status":  "completed",
		"message": "Dependencies installed successfully",
		"output":  string(output),
	}

	return marshalToolResult(result)
}

// --- check_requirements ---

func addCheckRequirementsTool(b *azdext.MCPServerBuilder) {
	b.AddTool("check_requirements", handleCheckRequirements,
		azdext.MCPToolOptions{
			Title:       "Check Prerequisites",
			Description: "Check if all required prerequisites (tools, CLIs, SDKs) defined in azure.yaml are installed and meet minimum version requirements. Returns detailed status of each requirement.",
			ReadOnly:    true,
			Idempotent:  true,
		},
		mcp.WithOutputSchema[RequirementsResult](),
		mcp.WithString("projectDir",
			mcp.Description("Optional project directory path. If not provided, uses current directory."),
		),
	)
}

func handleCheckRequirements(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
	cmdArgs, err := extractProjectDirArg(args)
	if err != nil {
		return azdext.MCPErrorResult("Invalid project directory: %v", err), nil
	}

	result, err := executeAzdAppCommand(ctx, "reqs", cmdArgs)
	if err != nil {
		return azdext.MCPErrorResult("Failed to check requirements: %v", err), nil
	}

	return marshalToolResult(result)
}

// --- get_environment_variables ---

func addGetEnvironmentVariablesTool(b *azdext.MCPServerBuilder) {
	b.AddTool("get_environment_variables", handleGetEnvironmentVariables,
		azdext.MCPToolOptions{
			Title:       "Get Environment Variables",
			Description: "Get environment variables configured for services. Returns all environment variables that services will use.",
			ReadOnly:    true,
			Idempotent:  true,
		},
		mcp.WithString("serviceName",
			mcp.Description("Optional service name to filter environment variables. If not provided, returns all."),
		),
		mcp.WithString("projectDir",
			mcp.Description("Optional project directory path. If not provided, uses current directory."),
		),
	)
}

func handleGetEnvironmentVariables(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
	cmdArgs, err := extractProjectDirArg(args)
	if err != nil {
		return azdext.MCPErrorResult("Invalid project directory: %v", err), nil
	}

	serviceName := args.OptionalString("serviceName", "")
	hasFilter := serviceName != ""
	if hasFilter {
		if valErr := security.ValidateServiceName(serviceName, true); valErr != nil {
			return azdext.MCPErrorResult("%s", valErr.Error()), nil
		}
	}

	result, err := executeAzdAppCommand(ctx, "info", cmdArgs)
	if err != nil {
		return azdext.MCPErrorResult("Failed to get environment variables: %v", err), nil
	}

	envVars := make(map[string]interface{})
	if services, ok := result["services"].([]interface{}); ok {
		for _, svc := range services {
			if svcMap, ok := svc.(map[string]interface{}); ok {
				svcName, _ := svcMap["name"].(string)
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
}

// --- set_environment_variable ---

func addSetEnvironmentVariableTool(b *azdext.MCPServerBuilder) {
	b.AddTool("set_environment_variable", handleSetEnvironmentVariable,
		azdext.MCPToolOptions{
			Title:       "Set Environment Variable",
			Description: "Set an environment variable for services. Note: This provides guidance on how to set environment variables, as they must be configured in azure.yaml or .env files.",
			ReadOnly:    true,
			Idempotent:  true,
		},
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
	)
}

func handleSetEnvironmentVariable(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
	name, err := args.RequireString("name")
	if err != nil {
		return azdext.MCPErrorResult("%s", err.Error()), nil
	}

	if !safeNamePattern.MatchString(name) {
		return azdext.MCPErrorResult("Invalid environment variable name: must start with alphanumeric and contain only alphanumeric, underscore, or hyphen"), nil
	}

	value, err := args.RequireString("value")
	if err != nil {
		return azdext.MCPErrorResult("%s", err.Error()), nil
	}

	serviceName := args.OptionalString("serviceName", "")
	if serviceName != "" {
		if err := security.ValidateServiceName(serviceName, true); err != nil {
			return azdext.MCPErrorResult("%s", err.Error()), nil
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
}
