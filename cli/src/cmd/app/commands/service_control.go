package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/output"
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
	var running []string
	for _, entry := range c.registry.ListAll() {
		if entry.Status == constants.StatusRunning || entry.Status == constants.StatusReady {
			running = append(running, entry.Name)
		}
	}
	return running
}

// GetStoppedServices returns a list of stopped service names.
func (c *ServiceController) GetStoppedServices() []string {
	var stopped []string
	for _, entry := range c.registry.ListAll() {
		if entry.Status == constants.StatusStopped || entry.Status == constants.StatusNotRunning || entry.Status == constants.StatusError {
			stopped = append(stopped, entry.Name)
		}
	}
	return stopped
}

// GetAllServices returns a list of all registered service names.
func (c *ServiceController) GetAllServices() []string {
	var all []string
	for _, entry := range c.registry.ListAll() {
		all = append(all, entry.Name)
	}
	return all
}

// StartService starts a single service.
func (c *ServiceController) StartService(ctx context.Context, serviceName string) *ServiceControlResult {
	result := &ServiceControlResult{
		ServiceName: serviceName,
		Success:     false,
	}

	// Validate service name to prevent injection
	if err := security.ValidateServiceName(serviceName, false); err != nil {
		result.Error = err.Error()
		result.Message = result.Error
		return result
	}

	entry, exists := c.registry.GetService(serviceName)
	if !exists {
		result.Error = fmt.Sprintf("service '%s' not found", serviceName)
		result.Message = result.Error
		return result
	}

	// Check if already running
	if entry.Status == constants.StatusRunning || entry.Status == constants.StatusReady {
		result.Error = fmt.Sprintf("service '%s' is already running", serviceName)
		result.Message = result.Error
		return result
	}

	// Execute with operation manager
	opResult := c.opManager.ExecuteOperation(ctx, serviceName, service.OpStart, func(ctx context.Context) error {
		return c.performStart(entry, serviceName)
	})

	if opResult.Error != nil {
		result.Error = opResult.Error.Error()
		result.Message = fmt.Sprintf("Failed to start '%s': %s", serviceName, opResult.Error)
	} else {
		result.Success = true
		result.Status = constants.StatusRunning
		result.Message = fmt.Sprintf("Service '%s' started", serviceName)
	}
	result.Duration = opResult.Duration.Round(time.Millisecond).String()
	return result
}

// StopService stops a single service.
func (c *ServiceController) StopService(ctx context.Context, serviceName string) *ServiceControlResult {
	result := &ServiceControlResult{
		ServiceName: serviceName,
		Success:     false,
	}

	// Validate service name to prevent injection
	if err := security.ValidateServiceName(serviceName, false); err != nil {
		result.Error = err.Error()
		result.Message = result.Error
		return result
	}

	entry, exists := c.registry.GetService(serviceName)
	if !exists {
		result.Error = fmt.Sprintf("service '%s' not found", serviceName)
		result.Message = result.Error
		return result
	}

	// Check if already stopped
	if entry.Status == constants.StatusStopped || entry.Status == constants.StatusNotRunning {
		result.Error = fmt.Sprintf("service '%s' is already stopped", serviceName)
		result.Message = result.Error
		return result
	}

	// Execute with operation manager
	opResult := c.opManager.ExecuteOperation(ctx, serviceName, service.OpStop, func(ctx context.Context) error {
		return c.performStop(entry, serviceName)
	})

	if opResult.Error != nil {
		result.Error = opResult.Error.Error()
		result.Message = fmt.Sprintf("Failed to stop '%s': %s", serviceName, opResult.Error)
	} else {
		result.Success = true
		result.Status = constants.StatusStopped
		result.Message = fmt.Sprintf("Service '%s' stopped", serviceName)
	}
	result.Duration = opResult.Duration.Round(time.Millisecond).String()
	return result
}

// RestartService restarts a single service.
func (c *ServiceController) RestartService(ctx context.Context, serviceName string) *ServiceControlResult {
	result := &ServiceControlResult{
		ServiceName: serviceName,
		Success:     false,
	}

	// Validate service name to prevent injection
	if err := security.ValidateServiceName(serviceName, false); err != nil {
		result.Error = err.Error()
		result.Message = result.Error
		return result
	}

	entry, exists := c.registry.GetService(serviceName)
	if !exists {
		result.Error = fmt.Sprintf("service '%s' not found", serviceName)
		result.Message = result.Error
		return result
	}

	// Execute with operation manager
	opResult := c.opManager.ExecuteOperation(ctx, serviceName, service.OpRestart, func(ctx context.Context) error {
		// Stop if running
		if entry.Status == constants.StatusRunning || entry.Status == constants.StatusReady {
			if err := c.performStop(entry, serviceName); err != nil {
				return fmt.Errorf("stop phase failed: %w", err)
			}
			// Refresh entry after stop
			entry, _ = c.registry.GetService(serviceName)
		}
		// Start
		return c.performStart(entry, serviceName)
	})

	if opResult.Error != nil {
		result.Error = opResult.Error.Error()
		result.Message = fmt.Sprintf("Failed to restart '%s': %s", serviceName, opResult.Error)
	} else {
		result.Success = true
		result.Status = constants.StatusRunning
		result.Message = fmt.Sprintf("Service '%s' restarted", serviceName)
	}
	result.Duration = opResult.Duration.Round(time.Millisecond).String()
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
	_ = c.registry.UpdateStatus(serviceName, constants.StatusStarting, constants.HealthUnknown)

	// Load environment variables
	envVars := c.loadEnvVars(runtime)

	// Start the service
	functionsParser := service.NewFunctionsOutputParser(false)
	process, err := service.StartService(runtime, envVars, c.projectDir, functionsParser)
	if err != nil {
		_ = c.registry.UpdateStatus(serviceName, constants.StatusError, constants.HealthUnknown)
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Update registry with new process info
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
		Health:      constants.HealthHealthy,
		StartTime:   time.Now(),
		LastChecked: time.Now(),
	}
	return c.registry.Register(updatedEntry)
}

// performStop executes the stop logic for a service.
func (c *ServiceController) performStop(entry *registry.ServiceRegistryEntry, serviceName string) error {
	if entry.PID <= 0 {
		_ = c.registry.UpdateStatus(serviceName, constants.StatusStopped, constants.HealthUnknown)
		return nil
	}

	// Update to stopping state
	_ = c.registry.UpdateStatus(serviceName, constants.StatusStopping, entry.Health)

	process, err := os.FindProcess(entry.PID)
	if err != nil {
		_ = c.registry.UpdateStatus(serviceName, constants.StatusStopped, constants.HealthUnknown)
		return nil // Process already gone
	}

	serviceProcess := &service.ServiceProcess{
		Name:    serviceName,
		Process: process,
	}
	if err := service.StopServiceGraceful(serviceProcess, service.DefaultStopTimeout); err != nil {
		_ = c.registry.UpdateStatus(serviceName, constants.StatusError, constants.HealthUnknown)
		return fmt.Errorf("error stopping service: %w", err)
	}

	return c.registry.UpdateStatus(serviceName, constants.StatusStopped, constants.HealthUnknown)
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
func printResult(result *ServiceControlResult, operation string) {
	if result.Success {
		output.ItemSuccess("%s: %s", result.ServiceName, result.Message)
	} else {
		output.ItemError("%s: %s", result.ServiceName, result.Error)
	}
}

// printBulkResult prints bulk operation results to the console.
func printBulkResult(result *BulkServiceControlResult, operation string) {
	for _, r := range result.Results {
		printResult(&r, operation)
	}

	output.Newline()
	if result.Success {
		output.Success("%s", result.Message)
	} else {
		output.Warning("%s", result.Message)
	}
}
