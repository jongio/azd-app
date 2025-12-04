package dashboard

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-app/cli/src/internal/portmanager"
	"github.com/jongio/azd-app/cli/src/internal/registry"
	"github.com/jongio/azd-app/cli/src/internal/security"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// serviceOperation defines the type of service operation to perform.
type serviceOperation int

const (
	opStart serviceOperation = iota
	opStop
	opRestart
)

// serviceOperationHandler handles start/stop/restart operations with shared logic.
type serviceOperationHandler struct {
	server    *Server
	operation serviceOperation
}

// newServiceOperationHandler creates a new handler for service operations.
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

// Handle processes the service operation request.
// If no service name is provided, performs bulk operation on all applicable services.
func (h *serviceOperationHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	serviceName := r.URL.Query().Get("service")
	if serviceName == "" {
		// Bulk operation - handle all applicable services
		h.handleBulkOperation(w, r)
		return
	}

	// Single service operation
	h.handleSingleOperation(w, r, serviceName)
}

// handleSingleOperation handles operations on a single service.
func (h *serviceOperationHandler) handleSingleOperation(w http.ResponseWriter, r *http.Request, serviceName string) {
	// Validate service name to prevent injection attacks
	if err := security.ValidateServiceName(serviceName, false); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("Invalid service name: %s", err.Error()), nil)
		return
	}

	reg := registry.GetRegistry(h.server.projectDir)
	entry, exists := reg.GetService(serviceName)
	if !exists {
		writeJSONError(w, http.StatusNotFound, fmt.Sprintf("Service '%s' not found", serviceName), nil)
		return
	}

	// Check if operation is already in progress via operation manager
	opMgr := service.GetOperationManager()
	if opMgr.IsOperationInProgress(serviceName) {
		writeJSONError(w, http.StatusConflict, fmt.Sprintf("Operation already in progress for service '%s'", serviceName), nil)
		return
	}

	// Validate state for operation
	if err := h.validateState(entry, serviceName); err != nil {
		writeJSONError(w, http.StatusConflict, err.Error(), nil)
		return
	}

	// Execute operation with the operation manager for concurrency control
	// Use request context for proper cancellation
	ctx := r.Context()
	opType := h.toServiceOperationType()

	result := opMgr.ExecuteOperation(ctx, serviceName, opType, func(ctx context.Context) error {
		return h.executeServiceOperation(w, entry, serviceName, reg)
	})

	if result.Error != nil {
		writeJSONError(w, http.StatusInternalServerError, result.Error.Error(), nil)
	}
}

// executeServiceOperation performs the actual service operation.
func (h *serviceOperationHandler) executeServiceOperation(w http.ResponseWriter, entry *registry.ServiceRegistryEntry, serviceName string, reg *registry.ServiceRegistry) error {
	// For restart, stop the service first and wait for process exit
	if h.operation == opRestart && entry.Status != constants.StatusStopped && entry.Status != constants.StatusNotRunning {
		if err := h.stopService(entry, serviceName); err != nil {
			log.Printf("Warning: error during restart stop phase: %v", err)
		}
	}

	// For stop operation, just stop and return
	if h.operation == opStop {
		h.performStop(w, entry, serviceName, reg)
		return nil
	}

	// For start/restart, start the service
	h.performStart(w, entry, serviceName, reg)
	return nil
}

// handleBulkOperation handles operations on all applicable services.
func (h *serviceOperationHandler) handleBulkOperation(w http.ResponseWriter, r *http.Request) {
	reg := registry.GetRegistry(h.server.projectDir)
	allServices := reg.ListAll()

	// Filter services based on operation type
	var applicableServices []string
	for _, entry := range allServices {
		switch h.operation {
		case opStart:
			// Start only stopped/errored services
			if entry.Status == constants.StatusStopped || entry.Status == constants.StatusNotRunning || entry.Status == constants.StatusError {
				applicableServices = append(applicableServices, entry.Name)
			}
		case opStop:
			// Stop only running services
			if entry.Status == constants.StatusRunning || entry.Status == constants.StatusReady || entry.Status == constants.StatusStarting {
				applicableServices = append(applicableServices, entry.Name)
			}
		case opRestart:
			// Restart all services that are running or stopped
			applicableServices = append(applicableServices, entry.Name)
		}
	}

	if len(applicableServices) == 0 {
		response := map[string]interface{}{
			"success":  true,
			"message":  fmt.Sprintf("No services to %s", h.getOperationVerb()),
			"services": []interface{}{},
		}
		if err := writeJSON(w, response); err != nil {
			log.Printf("Failed to write JSON response: %v", err)
		}
		return
	}

	// Execute bulk operation
	// Use request context for proper cancellation
	opMgr := service.GetOperationManager()
	ctx := r.Context()
	opType := h.toServiceOperationType()

	// Create operation function factory for each service
	operationFactory := func(svcName string) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			entry, exists := reg.GetService(svcName)
			if !exists {
				return fmt.Errorf("service '%s' not found", svcName)
			}
			return h.executeBulkServiceOperation(entry, svcName, reg)
		}
	}

	result := opMgr.ExecuteBulkOperation(ctx, applicableServices, opType, operationFactory)

	// Broadcast update after bulk operation
	if err := h.server.BroadcastServiceUpdate(h.server.projectDir); err != nil {
		log.Printf("Warning: failed to broadcast update: %v", err)
	}

	// Build response with results
	serviceResults := make([]map[string]interface{}, 0, len(result.Results))
	for _, opResult := range result.Results {
		svcResult := map[string]interface{}{
			"name":     opResult.ServiceName,
			"success":  opResult.Success,
			"duration": opResult.Duration.String(),
		}
		if opResult.Error != nil {
			svcResult["error"] = opResult.Error.Error()
		}
		serviceResults = append(serviceResults, svcResult)
	}

	response := map[string]interface{}{
		"success":      result.FailureCount == 0,
		"message":      fmt.Sprintf("%d service(s) %s, %d failed", result.SuccessCount, h.getOperationPastTense(), result.FailureCount),
		"services":     serviceResults,
		"successCount": result.SuccessCount,
		"failureCount": result.FailureCount,
		"duration":     result.TotalDuration.String(),
	}

	if err := writeJSON(w, response); err != nil {
		log.Printf("Failed to write JSON response: %v", err)
	}
}

// executeBulkServiceOperation performs the operation for a single service in bulk mode.
// Unlike executeServiceOperation, this doesn't write to the response writer.
func (h *serviceOperationHandler) executeBulkServiceOperation(entry *registry.ServiceRegistryEntry, serviceName string, reg *registry.ServiceRegistry) error {
	// For restart, stop the service first
	if h.operation == opRestart && entry.Status != constants.StatusStopped && entry.Status != constants.StatusNotRunning {
		if err := h.stopService(entry, serviceName); err != nil {
			log.Printf("Warning: error during restart stop phase for %s: %v", serviceName, err)
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
		log.Printf("Warning: failed to update status: %v", err)
	}

	if err := h.stopService(entry, serviceName); err != nil {
		log.Printf("Warning: %v", err)
		if regErr := reg.UpdateStatus(serviceName, constants.StatusError); regErr != nil {
			log.Printf("Warning: failed to update status: %v", regErr)
		}
		return err
	}

	// Update registry to stopped state
	if err := reg.UpdateStatus(serviceName, constants.StatusStopped); err != nil {
		log.Printf("Warning: failed to update status: %v", err)
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
	if err := reg.UpdateStatus(serviceName, constants.StatusStarting); err != nil {
		log.Printf("Warning: failed to update status: %v", err)
	}

	// Load environment variables
	envVars := h.loadEnvironmentVariables(runtime)

	// Start the service
	functionsParser := service.NewFunctionsOutputParser(false)
	process, err := service.StartService(runtime, envVars, h.server.projectDir, functionsParser)
	if err != nil {
		if regErr := reg.UpdateStatus(serviceName, constants.StatusError); regErr != nil {
			log.Printf("Warning: failed to update status: %v", regErr)
		}
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Validate process was created successfully
	if process == nil || process.Process == nil {
		if regErr := reg.UpdateStatus(serviceName, constants.StatusError); regErr != nil {
			log.Printf("Warning: failed to update status: %v", regErr)
		}
		return fmt.Errorf("service process not created")
	}

	// Create a fresh entry
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
	}
	if err := reg.Register(updatedEntry); err != nil {
		log.Printf("Warning: failed to register service: %v", err)
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
func (h *serviceOperationHandler) stopService(entry *registry.ServiceRegistryEntry, serviceName string) error {
	// First, try to stop by the registered PID
	if entry.PID > 0 {
		process, err := os.FindProcess(entry.PID)
		if err != nil {
			log.Printf("Warning: could not find process %d: %v", entry.PID, err)
		} else {
			serviceProcess := &service.ServiceProcess{
				Name:    serviceName,
				Process: process,
			}
			if err := service.StopServiceGraceful(serviceProcess, service.DefaultStopTimeout); err != nil {
				// Log but continue - the PID might be stale, we'll try by port next
				log.Printf("Warning: error stopping service %s by PID %d: %v", serviceName, entry.PID, err)
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
			log.Printf("Warning: error freeing port %d for service %s: %v", entry.Port, serviceName, err)
		}
	}

	return nil
}

// performStop handles the stop operation.
func (h *serviceOperationHandler) performStop(w http.ResponseWriter, entry *registry.ServiceRegistryEntry, serviceName string, reg *registry.ServiceRegistry) {
	// Update registry to stopping state
	if err := reg.UpdateStatus(serviceName, constants.StatusStopping); err != nil {
		log.Printf("Warning: failed to update status: %v", err)
	}

	if err := h.stopService(entry, serviceName); err != nil {
		log.Printf("Warning: %v", err)
		// Update registry to error state and notify clients
		if regErr := reg.UpdateStatus(serviceName, constants.StatusError); regErr != nil {
			log.Printf("Warning: failed to update status: %v", regErr)
		}
		h.broadcastAndRespond(w, serviceName, constants.StatusError, nil)
		return
	}

	// Update registry to stopped state
	if err := reg.UpdateStatus(serviceName, constants.StatusStopped); err != nil {
		log.Printf("Warning: failed to update status: %v", err)
	}

	h.broadcastAndRespond(w, serviceName, constants.StatusStopped, nil)
}

// performStart handles the start/restart operation.
func (h *serviceOperationHandler) performStart(w http.ResponseWriter, entry *registry.ServiceRegistryEntry, serviceName string, reg *registry.ServiceRegistry) {
	// Parse azure.yaml to get service configuration
	azureYaml, err := service.ParseAzureYaml(h.server.projectDir)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to parse azure.yaml", err)
		return
	}

	// Find the service definition
	svcDef, exists := azureYaml.Services[serviceName]
	if !exists {
		writeJSONError(w, http.StatusNotFound, fmt.Sprintf("Service '%s' not found in azure.yaml", serviceName), nil)
		return
	}

	// Detect runtime for the service
	runtime, err := service.DetectServiceRuntime(serviceName, svcDef, map[int]bool{}, h.server.projectDir, "")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to detect service runtime", err)
		return
	}

	// Update registry to starting state
	if err := reg.UpdateStatus(serviceName, constants.StatusStarting); err != nil {
		log.Printf("Warning: failed to update status: %v", err)
	}

	// Load environment variables
	envVars := h.loadEnvironmentVariables(runtime)

	// Start the service
	functionsParser := service.NewFunctionsOutputParser(false)
	process, err := service.StartService(runtime, envVars, h.server.projectDir, functionsParser)
	if err != nil {
		if regErr := reg.UpdateStatus(serviceName, constants.StatusError); regErr != nil {
			log.Printf("Warning: failed to update status: %v", regErr)
		}
		// Broadcast error state to WebSocket clients
		if broadcastErr := h.server.BroadcastServiceUpdate(h.server.projectDir); broadcastErr != nil {
			log.Printf("Warning: failed to broadcast error update: %v", broadcastErr)
		}
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to %s service", h.getOperationVerb()), err)
		return
	}

	// Validate process was created successfully
	if process == nil || process.Process == nil {
		if regErr := reg.UpdateStatus(serviceName, constants.StatusError); regErr != nil {
			log.Printf("Warning: failed to update status: %v", regErr)
		}
		writeJSONError(w, http.StatusInternalServerError, "Service process not created", nil)
		return
	}

	// Create a fresh entry to avoid race conditions with the copy from GetService
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
	}
	if err := reg.Register(updatedEntry); err != nil {
		log.Printf("Warning: failed to register service: %v", err)
	}

	h.broadcastAndRespond(w, serviceName, h.getOperationPastTense(), updatedEntry)
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

// broadcastAndRespond broadcasts update to WebSocket clients and sends HTTP response.
func (h *serviceOperationHandler) broadcastAndRespond(w http.ResponseWriter, serviceName, action string, entry *registry.ServiceRegistryEntry) {
	// Broadcast update to WebSocket clients
	if err := h.server.BroadcastServiceUpdate(h.server.projectDir); err != nil {
		log.Printf("Warning: failed to broadcast update: %v", err)
	}

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Service '%s' %s successfully", serviceName, action),
	}
	if entry != nil {
		response["service"] = entry
	}

	if err := writeJSON(w, response); err != nil {
		log.Printf("Failed to write JSON response: %v", err)
	}
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
