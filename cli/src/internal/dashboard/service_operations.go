package dashboard

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-app/cli/src/internal/registry"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// serviceNameRegex validates service names to prevent injection attacks.
// Allows alphanumeric characters, hyphens, underscores, and dots.
// Must start with alphanumeric, max 63 characters (DNS label compatible).
var serviceNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,62}$`)

// validateServiceName validates that a service name is safe and well-formed.
// Returns an error if the name is empty, too long, or contains invalid characters.
func validateServiceName(name string) error {
	if name == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	if len(name) > 63 {
		return fmt.Errorf("service name exceeds maximum length of 63 characters")
	}
	if !serviceNameRegex.MatchString(name) {
		return fmt.Errorf("service name contains invalid characters (use alphanumeric, hyphen, underscore, or dot)")
	}
	// Additional check for path traversal attempts
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("service name contains invalid path characters")
	}
	return nil
}

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

// Handle processes the service operation request.
func (h *serviceOperationHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	serviceName := r.URL.Query().Get("service")
	if serviceName == "" {
		writeJSONError(w, http.StatusNotImplemented, h.getNotImplementedMessage(), nil)
		return
	}

	// Validate service name to prevent injection attacks
	if err := validateServiceName(serviceName); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("Invalid service name: %s", err.Error()), nil)
		return
	}

	reg := registry.GetRegistry(h.server.projectDir)
	entry, exists := reg.GetService(serviceName)
	if !exists {
		writeJSONError(w, http.StatusNotFound, fmt.Sprintf("Service '%s' not found", serviceName), nil)
		return
	}

	// Validate state for operation
	if err := h.validateState(entry, serviceName); err != nil {
		writeJSONError(w, http.StatusConflict, err.Error(), nil)
		return
	}

	// For restart, stop the service first and wait for process exit
	if h.operation == opRestart && entry.Status != constants.StatusStopped && entry.Status != constants.StatusNotRunning {
		if err := h.stopService(entry, serviceName); err != nil {
			log.Printf("Warning: error during restart stop phase: %v", err)
		}
		// Process exit is handled by stopService via StopServiceGraceful
	}

	// For stop operation, just stop and return
	if h.operation == opStop {
		h.performStop(w, entry, serviceName, reg)
		return
	}

	// For start/restart, start the service
	h.performStart(w, entry, serviceName, reg)
}

// validateState checks if the operation is valid for the current service state.
func (h *serviceOperationHandler) validateState(entry *registry.ServiceRegistryEntry, serviceName string) error {
	switch h.operation {
	case opStart:
		if entry.Status == constants.StatusRunning || entry.Status == constants.StatusReady || entry.Status == constants.StatusStarting {
			return fmt.Errorf("Service '%s' is already %s", serviceName, entry.Status)
		}
	case opStop:
		if entry.Status == constants.StatusStopped || entry.Status == constants.StatusNotRunning {
			return fmt.Errorf("Service '%s' is already stopped", serviceName)
		}
	case opRestart:
		// Restart is always valid
	}
	return nil
}

// stopService stops a running service by PID.
// Returns nil if service was stopped successfully or if there was no process to stop.
func (h *serviceOperationHandler) stopService(entry *registry.ServiceRegistryEntry, serviceName string) error {
	if entry.PID <= 0 {
		return nil
	}

	process, err := os.FindProcess(entry.PID)
	if err != nil {
		return fmt.Errorf("could not find process %d: %w", entry.PID, err)
	}

	serviceProcess := &service.ServiceProcess{
		Name:    serviceName,
		Process: process,
	}
	if err := service.StopServiceGraceful(serviceProcess, service.DefaultStopTimeout); err != nil {
		return fmt.Errorf("error stopping service %s: %w", serviceName, err)
	}
	return nil
}

// performStop handles the stop operation.
func (h *serviceOperationHandler) performStop(w http.ResponseWriter, entry *registry.ServiceRegistryEntry, serviceName string, reg *registry.ServiceRegistry) {
	// Update registry to stopping state
	if err := reg.UpdateStatus(serviceName, constants.StatusStopping, entry.Health); err != nil {
		log.Printf("Warning: failed to update status: %v", err)
	}

	if err := h.stopService(entry, serviceName); err != nil {
		log.Printf("Warning: %v", err)
		// Update registry to error state and notify clients
		if regErr := reg.UpdateStatus(serviceName, constants.StatusError, constants.HealthUnknown); regErr != nil {
			log.Printf("Warning: failed to update status: %v", regErr)
		}
		h.broadcastAndRespond(w, serviceName, constants.StatusError, nil)
		return
	}

	// Update registry to stopped state
	if err := reg.UpdateStatus(serviceName, constants.StatusStopped, constants.HealthUnknown); err != nil {
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
	if err := reg.UpdateStatus(serviceName, constants.StatusStarting, constants.HealthUnknown); err != nil {
		log.Printf("Warning: failed to update status: %v", err)
	}

	// Load environment variables
	envVars := h.loadEnvironmentVariables(runtime)

	// Start the service
	functionsParser := service.NewFunctionsOutputParser(false)
	process, err := service.StartService(runtime, envVars, h.server.projectDir, functionsParser)
	if err != nil {
		if regErr := reg.UpdateStatus(serviceName, constants.StatusError, constants.HealthUnknown); regErr != nil {
			log.Printf("Warning: failed to update status: %v", regErr)
		}
		// Broadcast error state to WebSocket clients
		if broadcastErr := h.server.BroadcastServiceUpdate(h.server.projectDir); broadcastErr != nil {
			log.Printf("Warning: failed to broadcast error update: %v", broadcastErr)
		}
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to %s service", h.getOperationVerb()), err)
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
		Health:      constants.HealthHealthy,
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

// getNotImplementedMessage returns the not implemented message for bulk operations.
func (h *serviceOperationHandler) getNotImplementedMessage() string {
	switch h.operation {
	case opStart:
		return "Starting all services not yet implemented"
	case opStop:
		return "Stopping all services not yet implemented"
	case opRestart:
		return "Restarting all services not yet implemented"
	}
	return "Operation not yet implemented"
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
