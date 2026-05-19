package dashboard

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-app/cli/src/internal/docker"
	"github.com/jongio/azd-app/cli/src/internal/portmanager"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/registry"
)

// serviceOperation defines the type of service operation to perform.
type serviceOperation int

const (
	opStart serviceOperation = iota
	opStop
	opRestart
)

// serviceOperationHandler performs start/stop/restart operations against the
// dashboard's service registry. Historically this type backed the
// /api/services/{start,stop,restart} REST endpoints; those handlers have
// been removed and the type now exists purely as the execution engine that
// the Connect LifecycleService RPCs drive through service_ops_rpc.go.
type serviceOperationHandler struct {
	server    *Server
	operation serviceOperation
}

// newServiceOperationHandler creates a new handler for service operations.
// Called from service_ops_rpc.go to run bulk start/stop/restart operations
// triggered by the Connect LifecycleService.
func newServiceOperationHandler(s *Server, op serviceOperation) *serviceOperationHandler {
	return &serviceOperationHandler{
		server:    s,
		operation: op,
	}
}

// toServiceOperationType converts the internal operation type to the service package type.
func (h *serviceOperationHandler) toServiceOperationType() service.OperationType {
	switch h.operation {
	case opStart:
		return service.OpStart
	case opStop:
		return service.OpStop
	case opRestart:
		return service.OpRestart
	default:
		return service.OpStart
	}
}

// getOperationVerb returns the verb for the operation (start/stop/restart).
func (h *serviceOperationHandler) getOperationVerb() string {
	switch h.operation {
	case opStart:
		return "start"
	case opStop:
		return "stop"
	case opRestart:
		return "restart"
	}
	return "operate"
}

// executeBulkServiceOperation performs the operation for a single service in bulk mode.
// Invoked per-service by service_ops_rpc.go's RunBulk wrapper around
// service.OperationManager.ExecuteBulkOperation.
func (h *serviceOperationHandler) executeBulkServiceOperation(entry *registry.ServiceRegistryEntry, serviceName string, reg *registry.ServiceRegistry) error {
	// For restart, stop the service first
	if h.operation == opRestart && entry.Status != constants.StatusStopped && entry.Status != constants.StatusNotRunning {
		if err := h.stopService(entry, serviceName); err != nil {
			slog.Warn("error during restart stop phase", "service", serviceName, "error", err)
		}
	}

	// For stop operation
	if h.operation == opStop {
		return h.performStopBulk(entry, serviceName, reg)
	}

	// For start/restart
	return h.performStartBulk(entry, serviceName, reg)
}

// performStopBulk handles the stop operation without writing to HTTP response.
func (h *serviceOperationHandler) performStopBulk(entry *registry.ServiceRegistryEntry, serviceName string, reg *registry.ServiceRegistry) error {
	// Update registry to stopping state
	if err := reg.UpdateStatus(serviceName, constants.StatusStopping); err != nil {
		slog.Warn("failed to update status", "error", err)
	}

	if err := h.stopService(entry, serviceName); err != nil {
		slog.Warn("service operation warning", "error", err)
		if regErr := reg.UpdateStatus(serviceName, constants.StatusError); regErr != nil {
			slog.Warn("failed to update status", "error", regErr)
		}
		return err
	}

	// Update registry to stopped state
	if err := reg.UpdateStatus(serviceName, constants.StatusStopped); err != nil {
		slog.Warn("failed to update status", "error", err)
	}

	return nil
}

// performStartBulk handles the start/restart operation without writing to HTTP response.
func (h *serviceOperationHandler) performStartBulk(entry *registry.ServiceRegistryEntry, serviceName string, reg *registry.ServiceRegistry) error {
	// Parse azure.yaml to get service configuration
	azureYaml, err := service.ParseAzureYaml(h.server.projectDir)
	if err != nil {
		return fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	// Find the service definition
	svcDef, exists := azureYaml.Services[serviceName]
	if !exists {
		return fmt.Errorf("service '%s' not found in azure.yaml", serviceName)
	}

	// Detect runtime for the service
	runtime, err := service.DetectServiceRuntime(serviceName, svcDef, map[int]bool{}, h.server.projectDir, "")
	if err != nil {
		return fmt.Errorf("failed to detect service runtime: %w", err)
	}

	// Update registry to starting state
	if updateErr := reg.UpdateStatus(serviceName, constants.StatusStarting); updateErr != nil {
		slog.Warn("failed to update status", "error", updateErr)
	}

	// Start the service - use container runner for container services
	var process *service.ServiceProcess
	if runtime.Type == service.ServiceTypeContainer {
		process, err = service.StartContainerService(runtime, h.server.projectDir, true) // restartContainers=true for restart/start ops
		if err == nil {
			// Start container log collection
			if logErr := service.StartContainerLogCollection(process, h.server.projectDir); logErr != nil {
				slog.Warn("failed to start container log collection", "service", serviceName, "error", logErr)
			}
		}
	} else {
		// Load environment variables for native services
		envVars := h.loadEnvironmentVariables(runtime)
		functionsParser := service.NewFunctionsOutputParser(false)
		process, err = service.StartService(runtime, envVars, h.server.projectDir, functionsParser)
	}

	if err != nil {
		if regErr := reg.UpdateStatus(serviceName, constants.StatusError); regErr != nil {
			slog.Warn("failed to update status", "error", regErr)
		}
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Validate process was created successfully
	// Container services have ContainerID instead of Process.Pid
	if process == nil {
		if regErr := reg.UpdateStatus(serviceName, constants.StatusError); regErr != nil {
			slog.Warn("failed to update status", "error", regErr)
		}
		return fmt.Errorf("service process not created")
	}
	if runtime.Type == service.ServiceTypeContainer {
		if process.ContainerID == "" {
			if regErr := reg.UpdateStatus(serviceName, constants.StatusError); regErr != nil {
				slog.Warn("failed to update status", "error", regErr)
			}
			return fmt.Errorf("container not created")
		}
	} else {
		if process.Process == nil {
			if regErr := reg.UpdateStatus(serviceName, constants.StatusError); regErr != nil {
				slog.Warn("failed to update status", "error", regErr)
			}
			return fmt.Errorf("native service process not created")
		}
	}

	// Create a fresh entry - preserve Type and Mode for container services
	updatedEntry := &registry.ServiceRegistryEntry{
		Name:        serviceName,
		ProjectDir:  entry.ProjectDir,
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
	// Set PID only for native processes
	if process.Process != nil {
		updatedEntry.PID = process.Process.Pid
	}
	if err := reg.Register(updatedEntry); err != nil {
		slog.Warn("failed to register service", "error", err)
	}

	return nil
}

// validateState checks if the operation is valid for the current service state.
func (h *serviceOperationHandler) validateState(entry *registry.ServiceRegistryEntry, serviceName string) error {
	switch h.operation {
	case opStart:
		if entry.Status == constants.StatusRunning || entry.Status == constants.StatusReady || entry.Status == constants.StatusStarting {
			return fmt.Errorf("service '%s' is already %s", serviceName, entry.Status)
		}
	case opStop:
		if entry.Status == constants.StatusStopped || entry.Status == constants.StatusNotRunning {
			return fmt.Errorf("service '%s' is already stopped", serviceName)
		}
	case opRestart:
		// Restart is always valid
	}
	return nil
}

// stopService stops a running service by PID and ensures the port is freed.
// Returns nil if service was stopped successfully or if there was no process to stop.
// This function handles the case where the registry PID is stale but a different
// process is holding the port (e.g., after a crash and manual restart).
// For container services, uses Docker stop by container name instead of PID/port killing.
func (h *serviceOperationHandler) stopService(entry *registry.ServiceRegistryEntry, serviceName string) error {
	// Container services: use Docker stop by name
	if entry.Type == service.ServiceTypeContainer {
		return h.stopContainerByName(serviceName)
	}

	// Native processes: stop by PID and ensure port is freed
	if entry.PID > 0 {
		process, err := os.FindProcess(entry.PID)
		if err != nil {
			slog.Warn("could not find process", "pid", entry.PID, "error", err)
		} else {
			serviceProcess := &service.ServiceProcess{
				Name:    serviceName,
				Process: process,
			}
			if err := service.StopServiceGraceful(serviceProcess, service.DefaultStopTimeout); err != nil {
				// Log but continue - the PID might be stale, we'll try by port next
				slog.Warn("error stopping service by pid", "service", serviceName, "pid", entry.PID, "error", err)
			}
		}
	}

	// Also ensure the port is freed - this handles cases where:
	// 1. The registry PID is stale (process crashed and was restarted outside azd)
	// 2. PID was reused by OS for a different process
	// 3. A child process is still holding the port after parent was killed
	if entry.Port > 0 {
		pm := portmanager.GetPortManager(h.server.projectDir)
		if err := pm.KillProcessOnPort(entry.Port); err != nil {
			// Not a fatal error - port might already be free
			slog.Warn("error freeing port for service", "port", entry.Port, "service", serviceName, "error", err)
		}
	}

	return nil
}

// stopContainerByName stops a Docker container by its deterministic name (azd-{serviceName}).
// This is safer than port killing which might kill Docker's port forwarding proxy.
func (h *serviceOperationHandler) stopContainerByName(serviceName string) error {
	client := docker.NewClient()
	containerName := fmt.Sprintf("azd-%s", serviceName)

	// Stop container with grace period
	if err := client.Stop(containerName, 10); err != nil {
		slog.Warn("failed to stop container", "container", containerName, "error", err)
	}

	// Remove container to allow fresh start
	if err := client.Remove(containerName); err != nil {
		slog.Warn("failed to remove container", "container", containerName, "error", err)
	}

	return nil
}

// loadEnvironmentVariables loads env vars from OS and merges runtime-specific ones.
func (h *serviceOperationHandler) loadEnvironmentVariables(runtime *service.ServiceRuntime) map[string]string {
	envVars := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			envVars[pair[0]] = pair[1]
		}
	}

	// Merge runtime-specific env
	for k, v := range runtime.Env {
		envVars[k] = v
	}

	return envVars
}

// getOperationPastTense returns the past tense of the operation.
func (h *serviceOperationHandler) getOperationPastTense() string {
	switch h.operation {
	case opStart:
		return "started"
	case opStop:
		return "stopped"
	case opRestart:
		return "restarted"
	}
	return "operated"
}
