package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/portmanager"
	"github.com/jongio/azd-app/cli/src/internal/registry"
	"github.com/jongio/azd-app/cli/src/internal/security"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// ServiceController provides shared logic for service lifecycle CLI commands.
type ServiceController struct {
	projectDir string
	registry   *registry.ServiceRegistry
	opManager  *service.ServiceOperationManager
}

// ServiceControlResult contains the result of a service control operation.
type ServiceControlResult struct {
	ServiceName string `json:"serviceName"`
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	Status      string `json:"status,omitempty"`
	Error       string `json:"error,omitempty"`
	Duration    string `json:"duration,omitempty"` // Human-readable duration string
}

// BulkServiceControlResult contains results for bulk service operations.
type BulkServiceControlResult struct {
	Success      bool                   `json:"success"`
	Message      string                 `json:"message"`
	Results      []ServiceControlResult `json:"results"`
	SuccessCount int                    `json:"successCount"`
	FailureCount int                    `json:"failureCount"`
	TotalTime    string                 `json:"totalDuration"` // Human-readable duration string
}

// NewServiceController creates a new service controller for the given project directory.
func NewServiceController(projectDir string) (*ServiceController, error) {
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		projectDir = cwd
	}

	// Find azure.yaml to get the project root
	azureYamlPath, err := detector.FindAzureYaml(projectDir)
	if err != nil {
		return nil, fmt.Errorf("error searching for azure.yaml: %w", err)
	}
	if azureYamlPath != "" {
		projectDir = filepath.Dir(azureYamlPath)
	}

	return &ServiceController{
		projectDir: projectDir,
		registry:   registry.GetRegistry(projectDir),
		opManager:  service.GetOperationManager(),
	}, nil
}

// GetRunningServices returns a list of running service names.
func (c *ServiceController) GetRunningServices() []string {
	return c.filterServices(func(status string) bool {
		return status == constants.StatusRunning || status == constants.StatusReady
	})
}

// GetStoppedServices returns a list of stopped service names.
func (c *ServiceController) GetStoppedServices() []string {
	return c.filterServices(func(status string) bool {
		return status == constants.StatusStopped || status == constants.StatusNotRunning || status == constants.StatusError
	})
}

// GetAllServices returns a list of all registered service names.
func (c *ServiceController) GetAllServices() []string {
	return c.filterServices(func(_ string) bool { return true })
}

// filterServices returns service names matching the given status predicate.
func (c *ServiceController) filterServices(predicate func(status string) bool) []string {
	var names []string
	for _, entry := range c.registry.ListAll() {
		if predicate(entry.Status) {
			names = append(names, entry.Name)
		}
	}
	return names
}

// newErrorResult creates a ServiceControlResult with an error.
func newErrorResult(serviceName, errMsg string) *ServiceControlResult {
	return &ServiceControlResult{
		ServiceName: serviceName,
		Success:     false,
		Error:       errMsg,
		Message:     errMsg,
	}
}

// validateAndGetService validates the service name and retrieves the registry entry.
func (c *ServiceController) validateAndGetService(serviceName string) (*registry.ServiceRegistryEntry, *ServiceControlResult) {
	if err := security.ValidateServiceName(serviceName, false); err != nil {
		return nil, newErrorResult(serviceName, err.Error())
	}

	entry, exists := c.registry.GetService(serviceName)
	if !exists {
		return nil, newErrorResult(serviceName, fmt.Sprintf("service '%s' not found", serviceName))
	}

	return entry, nil
}

// isRunning returns true if the service status indicates it's running.
func isRunning(status string) bool {
	return status == constants.StatusRunning || status == constants.StatusReady
}

// isStopped returns true if the service status indicates it's stopped.
func isStopped(status string) bool {
	return status == constants.StatusStopped || status == constants.StatusNotRunning
}

// StartService starts a single service.
func (c *ServiceController) StartService(ctx context.Context, serviceName string) *ServiceControlResult {
	entry, errResult := c.validateAndGetService(serviceName)
	if errResult != nil {
		return errResult
	}

	if isRunning(entry.Status) {
		return newErrorResult(serviceName, fmt.Sprintf("service '%s' is already running", serviceName))
	}

	opResult := c.opManager.ExecuteOperation(ctx, serviceName, service.OpStart, func(ctx context.Context) error {
		return c.performStart(entry, serviceName)
	})

	return c.buildResult(serviceName, opResult, "start", constants.StatusRunning)
}

// StopService stops a single service.
func (c *ServiceController) StopService(ctx context.Context, serviceName string) *ServiceControlResult {
	entry, errResult := c.validateAndGetService(serviceName)
	if errResult != nil {
		return errResult
	}

	if isStopped(entry.Status) {
		return newErrorResult(serviceName, fmt.Sprintf("service '%s' is already stopped", serviceName))
	}

	opResult := c.opManager.ExecuteOperation(ctx, serviceName, service.OpStop, func(ctx context.Context) error {
		return c.performStop(entry, serviceName)
	})

	return c.buildResult(serviceName, opResult, "stop", constants.StatusStopped)
}

// RestartService restarts a single service.
func (c *ServiceController) RestartService(ctx context.Context, serviceName string) *ServiceControlResult {
	entry, errResult := c.validateAndGetService(serviceName)
	if errResult != nil {
		return errResult
	}

	opResult := c.opManager.ExecuteOperation(ctx, serviceName, service.OpRestart, func(ctx context.Context) error {
		if isRunning(entry.Status) {
			if err := c.performStop(entry, serviceName); err != nil {
				return fmt.Errorf("stop phase failed: %w", err)
			}
			entry, _ = c.registry.GetService(serviceName)
		}
		return c.performStart(entry, serviceName)
	})

	return c.buildResult(serviceName, opResult, "restart", constants.StatusRunning)
}

// buildResult constructs a ServiceControlResult from an operation result.
func (c *ServiceController) buildResult(serviceName string, opResult *service.OperationResult, opVerb, successStatus string) *ServiceControlResult {
	result := &ServiceControlResult{
		ServiceName: serviceName,
		Duration:    opResult.Duration.Round(time.Millisecond).String(),
	}

	if opResult.Error != nil {
		result.Success = false
		result.Error = opResult.Error.Error()
		result.Message = fmt.Sprintf("Failed to %s '%s': %s", opVerb, serviceName, opResult.Error)
	} else {
		result.Success = true
		result.Status = successStatus
		result.Message = fmt.Sprintf("Service '%s' %sed", serviceName, opVerb)
	}

	return result
}

// BulkStart starts multiple services.
func (c *ServiceController) BulkStart(ctx context.Context, serviceNames []string) *BulkServiceControlResult {
	return c.bulkOperation(ctx, serviceNames, service.OpStart, c.StartService)
}

// BulkStop stops multiple services.
func (c *ServiceController) BulkStop(ctx context.Context, serviceNames []string) *BulkServiceControlResult {
	return c.bulkOperation(ctx, serviceNames, service.OpStop, c.StopService)
}

// BulkRestart restarts multiple services.
func (c *ServiceController) BulkRestart(ctx context.Context, serviceNames []string) *BulkServiceControlResult {
	return c.bulkOperation(ctx, serviceNames, service.OpRestart, c.RestartService)
}

// bulkOperation executes an operation on multiple services.
func (c *ServiceController) bulkOperation(
	ctx context.Context,
	serviceNames []string,
	op service.OperationType,
	opFunc func(ctx context.Context, serviceName string) *ServiceControlResult,
) *BulkServiceControlResult {
	startTime := time.Now()
	result := &BulkServiceControlResult{
		Results: make([]ServiceControlResult, 0, len(serviceNames)),
	}

	if len(serviceNames) == 0 {
		result.Success = true
		result.Message = fmt.Sprintf("No services to %s", op)
		result.TotalTime = time.Since(startTime).Round(time.Millisecond).String()
		return result
	}

	for _, svcName := range serviceNames {
		svcResult := opFunc(ctx, svcName)
		result.Results = append(result.Results, *svcResult)
		if svcResult.Success {
			result.SuccessCount++
		} else {
			result.FailureCount++
		}
	}

	result.TotalTime = time.Since(startTime).Round(time.Millisecond).String()
	result.Success = result.FailureCount == 0
	result.Message = fmt.Sprintf("%d service(s) %sed, %d failed", result.SuccessCount, op, result.FailureCount)
	return result
}

// performStart executes the start logic for a service.
func (c *ServiceController) performStart(entry *registry.ServiceRegistryEntry, serviceName string) error {
	// Parse azure.yaml to get service configuration
	azureYaml, err := service.ParseAzureYaml(c.projectDir)
	if err != nil {
		return fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	svcDef, exists := azureYaml.Services[serviceName]
	if !exists {
		return fmt.Errorf("service '%s' not found in azure.yaml", serviceName)
	}

	// Detect runtime
	runtime, err := service.DetectServiceRuntime(serviceName, svcDef, map[int]bool{}, c.projectDir, "")
	if err != nil {
		return fmt.Errorf("failed to detect service runtime: %w", err)
	}

	// Update to starting state
	_ = c.registry.UpdateStatus(serviceName, constants.StatusStarting)

	// Load environment variables
	envVars := c.loadEnvVars(runtime)

	// Start the service
	functionsParser := service.NewFunctionsOutputParser(false)
	process, err := service.StartService(runtime, envVars, c.projectDir, functionsParser)
	if err != nil {
		_ = c.registry.UpdateStatus(serviceName, constants.StatusError)
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Validate process was created successfully
	if process == nil || process.Process == nil {
		_ = c.registry.UpdateStatus(serviceName, constants.StatusError)
		return fmt.Errorf("service process not created")
	}

	// Update registry with new process info
	// Health will be determined dynamically by health checks
	updatedEntry := &registry.ServiceRegistryEntry{
		Name:        serviceName,
		ProjectDir:  entry.ProjectDir,
		PID:         process.Process.Pid,
		Port:        runtime.Port,
		URL:         entry.URL,
		AzureURL:    entry.AzureURL,
		Language:    runtime.Language,
		Framework:   runtime.Framework,
		Status:      constants.StatusRunning,
		StartTime:   time.Now(),
		LastChecked: time.Now(),
		Type:        runtime.Type,
		Mode:        runtime.Mode,
	}
	return c.registry.Register(updatedEntry)
}

// performStop executes the stop logic for a service.
// It stops the service by PID and ensures the port is freed to handle stale registry entries.
func (c *ServiceController) performStop(entry *registry.ServiceRegistryEntry, serviceName string) error {
	// Update to stopping state
	_ = c.registry.UpdateStatus(serviceName, constants.StatusStopping)

	// First, try to stop by the registered PID
	if entry.PID > 0 {
		process, err := os.FindProcess(entry.PID)
		if err != nil {
			slog.Debug("could not find process", "pid", entry.PID, "error", err)
		} else {
			serviceProcess := &service.ServiceProcess{
				Name:    serviceName,
				Process: process,
			}
			if err := service.StopServiceGraceful(serviceProcess, service.DefaultStopTimeout); err != nil {
				// Log but continue - the PID might be stale, we'll try by port next
				slog.Debug("error stopping service by PID", "service", serviceName, "pid", entry.PID, "error", err)
			}
		}
	}

	// Also ensure the port is freed - this handles cases where:
	// 1. The registry PID is stale (process crashed and was restarted outside azd)
	// 2. PID was reused by OS for a different process
	// 3. A child process is still holding the port after parent was killed
	if entry.Port > 0 {
		pm := portmanager.GetPortManager(c.projectDir)
		if err := pm.KillProcessOnPort(entry.Port); err != nil {
			// Not a fatal error - port might already be free
			slog.Debug("error freeing port", "port", entry.Port, "service", serviceName, "error", err)
		}
	}

	return c.registry.UpdateStatus(serviceName, constants.StatusStopped)
}

// loadEnvVars loads environment variables for the service.
func (c *ServiceController) loadEnvVars(runtime *service.ServiceRuntime) map[string]string {
	envVars := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			envVars[pair[0]] = pair[1]
		}
	}
	for k, v := range runtime.Env {
		envVars[k] = v
	}
	return envVars
}

// printResult prints a single service control result to the console.
func printResult(result *ServiceControlResult) {
	if result.Success {
		output.ItemSuccess("%s: %s", result.ServiceName, result.Message)
	} else {
		output.ItemError("%s: %s", result.ServiceName, result.Error)
	}
}

// printBulkResult prints bulk operation results to the console.
func printBulkResult(result *BulkServiceControlResult) {
	for _, r := range result.Results {
		printResult(&r)
	}

	output.Newline()
	if result.Success {
		output.Success("%s", result.Message)
	} else {
		output.Warning("%s", result.Message)
	}
}

// setupContextWithSignalHandling creates a context that cancels on SIGINT/SIGTERM.
// Returns the context, cancel function, and a cleanup function that should be deferred.
func setupContextWithSignalHandling() (context.Context, context.CancelFunc, func()) {
	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	cleanup := func() {
		signal.Stop(sigChan)
		cancel()
	}

	return ctx, cancel, cleanup
}

// printNoServicesRegistered outputs the no services registered message.
func printNoServicesRegistered() {
	output.Info("No services are registered")
	output.Item("Run 'azd app run' first to start your development environment")
}

// noServicesRegisteredResult returns a BulkServiceControlResult for when no services are registered.
func noServicesRegisteredResult() BulkServiceControlResult {
	return BulkServiceControlResult{
		Success: false,
		Message: "No services registered. Run 'azd app run' first.",
		Results: []ServiceControlResult{},
	}
}

// noServicesToOperateResult returns a successful BulkServiceControlResult when no services need the operation.
func noServicesToOperateResult(stateDesc, opVerb string) BulkServiceControlResult {
	return BulkServiceControlResult{
		Success: true,
		Message: fmt.Sprintf("No %s services to %s", stateDesc, opVerb),
		Results: []ServiceControlResult{},
	}
}

// handleNoServicesCase handles the common pattern when --all finds no applicable services.
// Returns true if the case was handled (caller should return), false otherwise.
func handleNoServicesCase(ctrl *ServiceController, stateDesc, opVerb string) bool {
	if len(ctrl.GetAllServices()) == 0 {
		printNoServicesRegistered()
		if output.IsJSON() {
			_ = output.PrintJSON(noServicesRegisteredResult())
		}
		return true
	}
	output.Info("No %s services to %s (all services are already %s)", stateDesc, opVerb, oppositeState(stateDesc))
	if output.IsJSON() {
		_ = output.PrintJSON(noServicesToOperateResult(stateDesc, opVerb))
	}
	return true
}

// oppositeState returns the opposite state description for user messaging.
func oppositeState(state string) string {
	switch state {
	case "stopped":
		return "running"
	case "running":
		return "stopped"
	default:
		return "in another state"
	}
}

// confirmBulkOperation prompts for confirmation of bulk operations.
// Returns true if the user confirms or skipConfirm is true, false if cancelled.
func confirmBulkOperation(count int, opVerb string, skipConfirm bool) bool {
	if skipConfirm || output.IsJSON() {
		return true
	}
	return output.Confirm(fmt.Sprintf("%s all %d service(s)?", capitalize(opVerb), count))
}

// capitalize returns the string with its first letter capitalized.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// executeServiceOperation handles single vs bulk operation execution with consistent output.
func executeServiceOperation(
	ctx context.Context,
	services []string,
	singleOp func(ctx context.Context, name string) *ServiceControlResult,
	bulkOp func(ctx context.Context, names []string) *BulkServiceControlResult,
	opVerb string,
) error {
	if len(services) == 1 {
		result := singleOp(ctx, services[0])
		if output.IsJSON() {
			return output.PrintJSON(result)
		}
		printResult(result)
		if !result.Success {
			return fmt.Errorf("failed to %s service: %s", opVerb, result.Error)
		}
		return nil
	}

	result := bulkOp(ctx, services)
	if output.IsJSON() {
		return output.PrintJSON(result)
	}
	printBulkResult(result)
	if !result.Success {
		return fmt.Errorf("failed to %s %d service(s)", opVerb, result.FailureCount)
	}
	return nil
}

// parseServiceList splits a comma-separated service list, trims whitespace, and validates names.
func parseServiceList(services string) ([]string, error) {
	if services == "" {
		return nil, nil
	}
	parts := strings.Split(services, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			if err := security.ValidateServiceName(trimmed, false); err != nil {
				return nil, fmt.Errorf("invalid service name '%s': %w", trimmed, err)
			}
			result = append(result, trimmed)
		}
	}
	return result, nil
}
