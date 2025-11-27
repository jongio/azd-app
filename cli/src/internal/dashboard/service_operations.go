package dashboard

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/registry"
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

	// For restart, stop the service first
	if h.operation == opRestart && entry.Status != "stopped" && entry.Status != "not-running" {
		h.stopService(entry, serviceName)
		time.Sleep(500 * time.Millisecond)
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
		if entry.Status == "running" || entry.Status == "ready" || entry.Status == "starting" {
			return fmt.Errorf("Service '%s' is already %s", serviceName, entry.Status)
		}
	case opStop:
		if entry.Status == "stopped" || entry.Status == "not-running" {
			return fmt.Errorf("Service '%s' is already stopped", serviceName)
		}
	case opRestart:
		// Restart is always valid
	}
	return nil
}

// stopService stops a running service by PID.
func (h *serviceOperationHandler) stopService(entry *registry.ServiceRegistryEntry, serviceName string) {
	if entry.PID <= 0 {
		return
	}

	process, err := os.FindProcess(entry.PID)
	if err != nil {
		log.Printf("Warning: could not find process %d: %v", entry.PID, err)
		return
	}

	serviceProcess := &service.ServiceProcess{
		Name:    serviceName,
		Process: process,
	}
	if err := service.StopService(serviceProcess); err != nil {
		log.Printf("Warning: error stopping service %s: %v", serviceName, err)
	}
}

// performStop handles the stop operation.
func (h *serviceOperationHandler) performStop(w http.ResponseWriter, entry *registry.ServiceRegistryEntry, serviceName string, reg *registry.ServiceRegistry) {
	// Update registry to stopping state
	if err := reg.UpdateStatus(serviceName, "stopping", entry.Health); err != nil {
		log.Printf("Warning: failed to update status: %v", err)
	}

	h.stopService(entry, serviceName)

	// Update registry to stopped state
	if err := reg.UpdateStatus(serviceName, "stopped", "unknown"); err != nil {
		log.Printf("Warning: failed to update status: %v", err)
	}

	h.broadcastAndRespond(w, serviceName, "stopped", nil)
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
	if err := reg.UpdateStatus(serviceName, "starting", "unknown"); err != nil {
		log.Printf("Warning: failed to update status: %v", err)
	}

	// Load environment variables
	envVars := h.loadEnvironmentVariables(runtime)

	// Start the service
	functionsParser := service.NewFunctionsOutputParser(false)
	process, err := service.StartService(runtime, envVars, h.server.projectDir, functionsParser)
	if err != nil {
		if regErr := reg.UpdateStatus(serviceName, "error", "unknown"); regErr != nil {
			log.Printf("Warning: failed to update status: %v", regErr)
		}
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to %s service", h.getOperationVerb()), err)
		return
	}

	// Update registry with running state
	entry.PID = process.Process.Pid
	entry.Status = "running"
	entry.Health = "healthy"
	entry.StartTime = time.Now()
	entry.LastChecked = time.Now()
	if err := reg.Register(entry); err != nil {
		log.Printf("Warning: failed to register service: %v", err)
	}

	h.broadcastAndRespond(w, serviceName, h.getOperationPastTense(), entry)
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
