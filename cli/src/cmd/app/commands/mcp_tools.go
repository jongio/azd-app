package commands

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"syscall"
	"time"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
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
	b.AddTool(
		"get_services", handleGetServices,
		azdext.MCPToolOptions{
			Title:       "Get Running Services",
			Description: "Get comprehensive information about all running services in the current azd app project. Returns service status, health, URLs, ports, Azure deployment information, and environment variables.",
			ReadOnly:    true,
			Idempotent:  true,
		},
		mcp.WithOutputSchema[ServicesResult](),
		mcp.WithString(
			"projectDir",
			mcp.Description("Optional project directory path. If not provided, uses current directory."),
		),
	)
}

func handleGetServices(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
	projectDir, err := extractValidatedProjectDir(args)
	if err != nil {
		return mcpErrorResult("Invalid project directory: %v", err), nil
	}

	result, err := getAppInfoForMCP(ctx, projectDir)
	if err != nil {
		return mcpErrorResult("Failed to get services: %v", err), nil
	}

	return marshalToolResult(result)
}

// --- get_service_logs ---

func addGetServiceLogsTool(b *azdext.MCPServerBuilder) {
	b.AddTool(
		"get_service_logs", handleGetServiceLogs,
		azdext.MCPToolOptions{
			Title:       "Get Service Logs",
			Description: "Get logs from running services. Can filter by service name, log level, and time range. Supports both local and Azure cloud logs via the source parameter.",
			ReadOnly:    true,
			Idempotent:  true,
		},
		mcp.WithString(
			"projectDir",
			mcp.Description("Optional project directory path. If not provided, uses current directory."),
		),
		mcp.WithString(
			"serviceName",
			mcp.Description("Optional service name to filter logs. If not provided, shows logs from all services."),
		),
		mcp.WithNumber(
			"tail",
			mcp.Description("Number of recent log lines to retrieve. Default is 100."),
		),
		mcp.WithString(
			"level",
			mcp.Description("Filter by log level: 'info', 'warn', 'error', 'debug', or 'all'. Default is 'all'."),
		),
		mcp.WithString(
			"since",
			mcp.Description("Show logs since duration (e.g., '5m', '1h', '30s'). If provided, overrides tail parameter."),
		),
		mcp.WithString(
			"source",
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
			return mcpErrorResult("Invalid project directory: %v", valErr), nil
		}
		projectDir = validated
	}

	if serviceName := args.OptionalString("serviceName", ""); serviceName != "" {
		if valErr := security.ValidateServiceName(serviceName, true); valErr != nil {
			return mcpErrorResult("%s", valErr.Error()), nil
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
			return mcpErrorResult("%s", valErr.Error()), nil
		}
		opts.level = level
	}

	if since := args.OptionalString("since", ""); since != "" {
		if !isValidDuration(since) {
			return mcpErrorResult("Invalid 'since' format. Use duration like '5m', '1h', '30s'"), nil
		}
		opts.since = since
	}

	if source := args.OptionalString("source", ""); source != "" {
		allowedSources := map[string]bool{"local": true, "azure": true, "both": true}
		if valErr := validateEnumParam(source, allowedSources, "source"); valErr != nil {
			return mcpErrorResult("%s", valErr.Error()), nil
		}
		if source == "both" {
			source = "all"
		}
		opts.source = source
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		return mcpErrorResult("Request canceled: %v", ctxErr), nil
	}

	collectCtx, collectCancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer collectCancel()

	executor := newLogsExecutorForMCP(opts, projectDir)
	collected, err := executor.collect(collectCtx, serviceArgs)
	if err != nil {
		if collectCtx.Err() == context.DeadlineExceeded {
			return mcpErrorResult("Command timed out after %v", defaultCommandTimeout), nil
		}
		return mcpErrorResult("Failed to get logs: %v", err), nil
	}

	return marshalToolResult(collected.Entries)
}

// --- get_service_errors ---

func addGetServiceErrorsTool(b *azdext.MCPServerBuilder) {
	b.AddTool(
		"get_service_errors", handleGetServiceErrors,
		azdext.MCPToolOptions{
			Title:       "Get Service Errors",
			Description: "Get error logs from services with surrounding context for debugging. Optimized for AI-assisted troubleshooting - returns only errors with relevant context to help diagnose issues quickly. Uses the logs command filtered to error level.",
			ReadOnly:    true,
			Idempotent:  true,
		},
		mcp.WithString(
			"projectDir",
			mcp.Description("Optional project directory path. If not provided, uses current directory."),
		),
		mcp.WithString(
			"serviceName",
			mcp.Description("Optional service name to filter errors. If not provided, shows errors from all services."),
		),
		mcp.WithString(
			"since",
			mcp.Description("Show errors since duration (e.g., '5m', '1h', '30s'). Default is '10m'."),
		),
		mcp.WithNumber(
			"tail",
			mcp.Description("Number of log lines to retrieve. Default is 500."),
		),
		mcp.WithNumber(
			"contextLines",
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
			return mcpErrorResult("Invalid project directory: %v", valErr), nil
		}
		projectDir = validated
	}

	if serviceName := args.OptionalString("serviceName", ""); serviceName != "" {
		if valErr := security.ValidateServiceName(serviceName, true); valErr != nil {
			return mcpErrorResult("%s", valErr.Error()), nil
		}
		serviceArgs = append(serviceArgs, serviceName)
	}

	if s := args.OptionalString("since", ""); s != "" {
		if !isValidDuration(s) {
			return mcpErrorResult("Invalid 'since' format. Use duration like '5m', '1h', '30s'"), nil
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
		return mcpErrorResult("Request canceled: %v", err), nil
	}

	collectCtx, collectCancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer collectCancel()

	executor := newLogsExecutorForMCP(opts, projectDir)
	collected, err := executor.collect(collectCtx, serviceArgs)
	if err != nil {
		if collectCtx.Err() == context.DeadlineExceeded {
			return mcpErrorResult("Command timed out after %v", defaultCommandTimeout), nil
		}
		return mcpErrorResult("Failed to get errors: %v", err), nil
	}

	entries := collected.EntriesWithContext
	result := map[string]any{
		"summary": map[string]any{
			"totalErrors": len(entries),
			"since":       opts.since,
		},
		"errors": entries,
	}

	return marshalToolResult(result)
}

// --- get_project_info ---

func addGetProjectInfoTool(b *azdext.MCPServerBuilder) {
	b.AddTool(
		"get_project_info", handleGetProjectInfo,
		azdext.MCPToolOptions{
			Title:       "Get Project Information",
			Description: "Get project metadata and configuration from azure.yaml. Returns project name, directory, and service definitions.",
			ReadOnly:    true,
			Idempotent:  true,
		},
		mcp.WithOutputSchema[ProjectInfo](),
		mcp.WithString(
			"projectDir",
			mcp.Description("Optional project directory path. If not provided, uses current directory."),
		),
	)
}

func handleGetProjectInfo(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
	projectDir, err := extractValidatedProjectDir(args)
	if err != nil {
		return mcpErrorResult("Invalid project directory: %v", err), nil
	}

	result, err := getAppInfoForMCP(ctx, projectDir)
	if err != nil {
		return mcpErrorResult("Failed to get project info: %v", err), nil
	}

	// Extract just project-level info
	projectInfo := map[string]any{
		"project": result["project"],
	}

	// Extract service metadata (name, language, framework, project path)
	if services, ok := result["services"].([]any); ok {
		simplifiedServices := []map[string]any{}
		for _, svc := range services {
			if svcMap, ok := svc.(map[string]any); ok {
				simplified := map[string]any{
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
	b.AddTool(
		"run_services", handleRunServices,
		azdext.MCPToolOptions{
			Title:       "Run Development Services",
			Description: "Start development services defined in azure.yaml, Aspire, or docker compose. By default this starts the application in the background and returns immediately. Set wait=true to block until every service is ready (or timeoutSeconds elapses) and get each service's final state back.",
		},
		mcp.WithString(
			"projectDir",
			mcp.Description("Optional project directory path. If not provided, uses current directory."),
		),
		mcp.WithString(
			"runtime",
			mcp.Description("Optional runtime mode: 'azd' (default), 'aspire', 'pnpm', or 'docker-compose'."),
		),
		mcp.WithBoolean(
			"wait",
			mcp.Description("If true, block until all services are ready or timeoutSeconds elapses before returning. Default false (returns immediately after starting)."),
		),
		mcp.WithNumber(
			"timeoutSeconds",
			mcp.Description("Maximum seconds to wait for readiness when wait is true. Default 120."),
		),
	)
}

func handleRunServices(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
	cmdArgs, err := extractProjectDirArg(args)
	if err != nil {
		return mcpErrorResult("Invalid project directory: %v", err), nil
	}

	if runtime := args.OptionalString("runtime", ""); runtime != "" {
		if valErr := validateEnumParam(runtime, allowedRuntimes, "runtime"); valErr != nil {
			return mcpErrorResult("%s", valErr.Error()), nil
		}
		cmdArgs = append(cmdArgs, "--runtime", runtime)
	}

	wait := args.OptionalBool("wait", false)
	timeoutSeconds := clampRunWaitTimeout(args.OptionalInt("timeoutSeconds", defaultRunWaitTimeoutSeconds))

	// Resolve the directory used for readiness polling up front so an invalid
	// path fails before we start the process.
	pollDir := ""
	if wait {
		pd, pdErr := extractValidatedProjectDir(args)
		if pdErr != nil {
			return mcpErrorResult("Invalid project directory: %v", pdErr), nil
		}
		pollDir = pd
	}

	// Note: azd app run is interactive and long-running, so we run it in a non-blocking way
	// and return information about the command being executed.
	cmd := exec.CommandContext(context.WithoutCancel(ctx), azdCommand, append([]string{appSubcommand, "run"}, cmdArgs...)...)

	// Create pipes to capture startup errors without blocking
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return mcpErrorResult("Failed to create stderr pipe: %v", err), nil
	}

	if err := cmd.Start(); err != nil {
		return mcpErrorResult("Failed to start services: %v", err), nil
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
				return mcpErrorResult("Service failed to start: %s", errMsg), nil
			default:
				return mcpErrorResult("Service failed to start immediately"), nil
			}
		}
	}

	// Release the process so it's not a zombie when parent exits
	go func() {
		_ = cmd.Wait()
	}()

	if wait {
		waitRes := waitForServicesReady(ctx, pollDir, time.Duration(timeoutSeconds)*time.Second, runWaitPollInterval)
		return marshalToolResult(buildRunWaitResult(waitRes, pid, timeoutSeconds))
	}

	result := map[string]any{
		"status":  "started",
		"message": "Services are starting in the background. Use get_services to check their status.",
	}
	if pid > 0 {
		result["pid"] = pid
	}

	return marshalToolResult(result)
}

// Readiness constants and helpers backing the run_services wait behavior.
const (
	defaultRunWaitTimeoutSeconds = 120
	maxRunWaitTimeoutSeconds     = 900
	runWaitPollInterval          = 1 * time.Second

	serviceStatusRunning   = "running"
	serviceHealthUnhealthy = "unhealthy"
)

// serviceReadiness is the per-service state reported by the wait behavior.
type serviceReadiness struct {
	Name   string `json:"name"`
	Status string `json:"status,omitempty"`
	Health string `json:"health,omitempty"`
	Ready  bool   `json:"ready"`
}

// runWaitResult is the structured result returned when run_services waits for
// readiness. TimedOut distinguishes a timeout from a clean all-ready return so
// the caller can tell why it stopped waiting.
type runWaitResult struct {
	Ready    bool               `json:"ready"`
	TimedOut bool               `json:"timedOut"`
	Services []serviceReadiness `json:"services"`
}

// pollServicesReadiness reads the current readiness of every service. It is a
// package var so tests can stub the services source without a live dashboard.
var pollServicesReadiness = collectServiceReadiness

// clampRunWaitTimeout normalizes a caller-supplied readiness timeout (in
// seconds) to the supported range: non-positive values fall back to the
// default and anything above the maximum is capped.
func clampRunWaitTimeout(seconds int) int {
	if seconds <= 0 {
		return defaultRunWaitTimeoutSeconds
	}
	if seconds > maxRunWaitTimeoutSeconds {
		return maxRunWaitTimeoutSeconds
	}
	return seconds
}

// collectServiceReadiness reuses the same service source as get_services and
// info, then derives a ready flag per service via toServiceReadiness.
func collectServiceReadiness(ctx context.Context, projectDir string) ([]serviceReadiness, error) {
	services, err := collectServiceInfoForMCP(ctx, projectDir)
	if err != nil {
		return nil, err
	}
	return toServiceReadiness(services), nil
}

// toServiceReadiness maps service info to per-service readiness. A service is
// ready once it is running and not explicitly unhealthy; a running service
// without a health probe reports "unknown" and counts as ready. Nil entries
// are skipped.
func toServiceReadiness(services []*serviceinfo.ServiceInfo) []serviceReadiness {
	out := make([]serviceReadiness, 0, len(services))
	for _, svc := range services {
		if svc == nil {
			continue
		}
		r := serviceReadiness{Name: svc.Name}
		if svc.Local != nil {
			r.Status = svc.Local.Status
			r.Health = svc.Local.Health
			r.Ready = svc.Local.Status == serviceStatusRunning && svc.Local.Health != serviceHealthUnhealthy
		}
		out = append(out, r)
	}
	return out
}

// allServicesReady reports whether every service is ready. An empty set is not
// ready: it means no service state is available yet, so waiting should continue.
func allServicesReady(rs []serviceReadiness) bool {
	if len(rs) == 0 {
		return false
	}
	for _, r := range rs {
		if !r.Ready {
			return false
		}
	}
	return true
}

// waitForServicesReady polls readiness until every service is ready, the
// timeout elapses, or the context is cancelled. On timeout or cancellation it
// returns the last observed state with TimedOut set, never a bare error, so the
// caller can report which services did and did not come up.
func waitForServicesReady(ctx context.Context, projectDir string, timeout, interval time.Duration) runWaitResult {
	deadline := time.Now().Add(timeout)
	var last []serviceReadiness
	for {
		readiness, err := pollServicesReadiness(ctx, projectDir)
		if err == nil {
			last = readiness
			if allServicesReady(readiness) {
				return runWaitResult{Ready: true, Services: readiness}
			}
		}
		if !time.Now().Before(deadline) {
			return runWaitResult{Ready: false, TimedOut: true, Services: last}
		}
		select {
		case <-ctx.Done():
			return runWaitResult{Ready: false, TimedOut: true, Services: last}
		case <-time.After(interval):
		}
	}
}

// buildRunWaitResult builds the structured result returned by run_services when
// it waits for readiness. It reports each service's final state and a
// status/message that distinguishes a clean all-ready return from a timeout.
func buildRunWaitResult(res runWaitResult, pid, timeoutSeconds int) map[string]any {
	result := map[string]any{
		"ready":    res.Ready,
		"services": res.Services,
	}
	if pid > 0 {
		result["pid"] = pid
	}
	if res.TimedOut {
		result["status"] = "timeout"
		result["message"] = fmt.Sprintf(
			"Timed out after %ds waiting for services to become ready. See services for the current state of each one.",
			timeoutSeconds)
	} else {
		result["status"] = "ready"
		result["message"] = "All services are ready."
	}
	return result
}

// --- stop_services ---

func addStopServicesTool(b *azdext.MCPServerBuilder) {
	b.AddTool(
		"stop_services", handleStopServices,
		azdext.MCPToolOptions{
			Title:       "Stop Running Services",
			Description: "Stop all running development services. This will gracefully shut down services started with run_services.",
			Idempotent:  true,
		},
		mcp.WithString(
			"projectDir",
			mcp.Description("Optional project directory path. If not provided, uses current directory."),
		),
		mcp.WithString(
			"serviceName",
			mcp.Description("Optional specific service to stop. If not provided, stops all running services."),
		),
	)
}

func handleStopServices(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
	projectDir, err := extractValidatedProjectDir(args)
	if err != nil {
		return mcpErrorResult("Invalid project directory: %v", err), nil
	}

	ctrl, err := NewServiceController(projectDir)
	if err != nil {
		return mcpErrorResult("Failed to initialize service controller: %v", err), nil
	}

	if serviceName := args.OptionalString("serviceName", ""); serviceName != "" {
		if valErr := security.ValidateServiceName(serviceName, false); valErr != nil {
			return mcpErrorResult("%s", valErr.Error()), nil
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
	b.AddTool(
		"start_service", handleStartService,
		azdext.MCPToolOptions{
			Title:       "Start Service",
			Description: "Start a specific stopped service. Use this to start individual services that were previously stopped.",
		},
		mcp.WithString(
			"serviceName",
			mcp.Description("Name of the service to start"),
			mcp.Required(),
		),
		mcp.WithString(
			"projectDir",
			mcp.Description("Optional project directory path. If not provided, uses current directory."),
		),
	)
}

func handleStartService(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
	return handleSingleServiceOp(ctx, args, func(ctx context.Context, ctrl *ServiceController, name string) (*mcp.CallToolResult, error) {
		return marshalToolResult(ctrl.StartService(ctx, name))
	})
}

// --- restart_service ---

func addRestartServiceTool(b *azdext.MCPServerBuilder) {
	b.AddTool(
		"restart_service", handleRestartService,
		azdext.MCPToolOptions{
			Title:       "Restart Service",
			Description: "Restart a specific service. This will stop and start the specified service.",
		},
		mcp.WithString(
			"serviceName",
			mcp.Description("Name of the service to restart"),
			mcp.Required(),
		),
		mcp.WithString(
			"projectDir",
			mcp.Description("Optional project directory path. If not provided, uses current directory."),
		),
	)
}

func handleRestartService(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
	return handleSingleServiceOp(ctx, args, func(ctx context.Context, ctrl *ServiceController, name string) (*mcp.CallToolResult, error) {
		return marshalToolResult(ctrl.RestartService(ctx, name))
	})
}

// --- install_dependencies ---

func addInstallDependenciesTool(b *azdext.MCPServerBuilder) {
	b.AddTool(
		"install_dependencies", handleInstallDependencies,
		azdext.MCPToolOptions{
			Title:       "Install Project Dependencies",
			Description: "Install dependencies for all detected projects (Node.js, Python, .NET). Automatically detects package managers (npm/pnpm/yarn, uv/poetry/pip, dotnet) and installs dependencies.",
		},
		mcp.WithString(
			"projectDir",
			mcp.Description("Optional project directory path. If not provided, uses current directory."),
		),
	)
}

func handleInstallDependencies(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
	cmdArgs, err := extractProjectDirArg(args)
	if err != nil {
		return mcpErrorResult("Invalid project directory: %v", err), nil
	}

	// SEC-026 (CWE-829): Verify the target directory is an azd workspace before
	// running package managers. Postinstall scripts execute arbitrary code, so we
	// must not invoke them outside a trusted project directory.
	projectDir, err := extractValidatedProjectDir(args)
	if err != nil {
		return mcpErrorResult("Invalid project directory: %v", err), nil
	}
	azureYamlPath, err := detector.FindAzureYaml(projectDir)
	if err != nil {
		return mcpErrorResult("Error searching for azure.yaml: %v", err), nil
	}
	if azureYamlPath == "" {
		return mcpErrorResult("install_dependencies requires an azure.yaml project; run from a project directory"), nil
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		return mcpErrorResult("Request canceled: %v", ctxErr), nil
	}

	cmdCtx, cancel := context.WithTimeout(ctx, dependencyInstallTimeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, azdCommand, append([]string{appSubcommand, "deps"}, cmdArgs...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return mcpErrorResult("Request was canceled"), nil
		}
		if errors.Is(cmdCtx.Err(), context.DeadlineExceeded) {
			return mcpErrorResult("Dependency installation timed out after %v", dependencyInstallTimeout), nil
		}
		return mcpErrorResult("Failed to install dependencies: %v\nOutput: %s", err, string(output)), nil
	}

	result := map[string]any{
		"status":  "completed",
		"message": "Dependencies installed successfully",
		"output":  string(output),
	}

	return marshalToolResult(result)
}

// --- check_requirements ---

func addCheckRequirementsTool(b *azdext.MCPServerBuilder) {
	b.AddTool(
		"check_requirements", handleCheckRequirements,
		azdext.MCPToolOptions{
			Title:       "Check Prerequisites",
			Description: "Check if all required prerequisites (tools, CLIs, SDKs) defined in azure.yaml are installed and meet minimum version requirements. Returns detailed status of each requirement.",
			ReadOnly:    true,
			Idempotent:  true,
		},
		mcp.WithOutputSchema[RequirementsResult](),
		mcp.WithString(
			"projectDir",
			mcp.Description("Optional project directory path. If not provided, uses current directory."),
		),
	)
}

func handleCheckRequirements(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
	cmdArgs, err := extractProjectDirArg(args)
	if err != nil {
		return mcpErrorResult("Invalid project directory: %v", err), nil
	}

	result, err := executeAzdAppCommand(ctx, "reqs", cmdArgs)
	if err != nil {
		return mcpErrorResult("Failed to check requirements: %v", err), nil
	}

	return marshalToolResult(result)
}

// --- get_environment_variables ---

func addGetEnvironmentVariablesTool(b *azdext.MCPServerBuilder) {
	b.AddTool(
		"get_environment_variables", handleGetEnvironmentVariables,
		azdext.MCPToolOptions{
			Title:       "Get Environment Variables",
			Description: "Get environment variables configured for services. Returns all environment variables that services will use.",
			ReadOnly:    true,
			Idempotent:  true,
		},
		mcp.WithString(
			"serviceName",
			mcp.Description("Optional service name to filter environment variables. If not provided, returns all."),
		),
		mcp.WithString(
			"projectDir",
			mcp.Description("Optional project directory path. If not provided, uses current directory."),
		),
	)
}

func handleGetEnvironmentVariables(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
	projectDir, err := extractValidatedProjectDir(args)
	if err != nil {
		return mcpErrorResult("Invalid project directory: %v", err), nil
	}

	serviceName := args.OptionalString("serviceName", "")
	hasFilter := serviceName != ""
	if hasFilter {
		if valErr := security.ValidateServiceName(serviceName, true); valErr != nil {
			return mcpErrorResult("%s", valErr.Error()), nil
		}
	}

	result, err := getAppInfoForMCP(ctx, projectDir)
	if err != nil {
		return mcpErrorResult("Failed to get environment variables: %v", err), nil
	}

	envVars := make(map[string]any)
	if services, ok := result["services"].([]any); ok {
		for _, svc := range services {
			if svcMap, ok := svc.(map[string]any); ok {
				svcName, _ := svcMap["name"].(string)
				if hasFilter && svcName != serviceName {
					continue
				}
				if env, ok := svcMap["env"].(map[string]any); ok {
					safeEnv := make(map[string]any, len(env))
					for k, v := range env {
						if strVal, ok := v.(string); ok {
							safeEnv[k] = redactEnvVarForMCP(k, strVal)
						} else {
							safeEnv[k] = v
						}
					}
					envVars[svcName] = safeEnv
				}
			}
		}
	}

	return marshalToolResult(envVars)
}

// --- set_environment_variable ---

func addSetEnvironmentVariableTool(b *azdext.MCPServerBuilder) {
	b.AddTool(
		"set_environment_variable", handleSetEnvironmentVariable,
		azdext.MCPToolOptions{
			Title: "Set Environment Variable",
			Description: "Provides guidance on how to set an environment variable for services. " +
				"This tool does NOT modify any files or system state; it returns instructions " +
				"for configuring the variable in azure.yaml, .env files, or the shell. " +
				"Secret-pattern values (keys containing TOKEN, SECRET, KEY, PASSWORD, CREDENTIAL, " +
				"CONNECTION_STRING) are redacted in the response.",
			ReadOnly:   true,
			Idempotent: true,
		},
		mcp.WithString(
			"name",
			mcp.Description("Name of the environment variable"),
			mcp.Required(),
		),
		mcp.WithString(
			"value",
			mcp.Description("Value of the environment variable"),
			mcp.Required(),
		),
		mcp.WithString(
			"serviceName",
			mcp.Description("Optional service name. If not provided, applies to all services."),
		),
	)
}

func handleSetEnvironmentVariable(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
	name, err := args.RequireString("name")
	if err != nil {
		return mcpErrorResult("%s", err.Error()), nil
	}

	if !safeNamePattern.MatchString(name) {
		return mcpErrorResult("Invalid environment variable name: must start with alphanumeric and contain only alphanumeric, underscore, or hyphen"), nil
	}

	value, err := args.RequireString("value")
	if err != nil {
		return mcpErrorResult("%s", err.Error()), nil
	}

	// Redact secret-pattern values before including them in any response.
	// This prevents tokens, passwords, API keys, and credentials from leaking
	// through the MCP tool result (CWE-684).
	displayValue := redactSecretValue(name, value)

	serviceName := args.OptionalString("serviceName", "")
	if serviceName != "" {
		if err := security.ValidateServiceName(serviceName, true); err != nil {
			return mcpErrorResult("%s", err.Error()), nil
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
		name, displayValue,
		serviceName, name, displayValue,
		name, displayValue,
		name, displayValue)

	result := map[string]any{
		"status":   "guidance",
		"message":  guidance,
		"variable": name,
		"value":    displayValue,
	}

	return marshalToolResult(result)
}
