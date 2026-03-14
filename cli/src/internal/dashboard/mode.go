// Package dashboard provides API endpoints for log source mode management.
package dashboard

import (
	"encoding/json"
	"net/http"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

// ModeRequest represents a request to change the log source mode.
type ModeRequest struct {
	Mode string `json:"mode"` // "local" or "azure"
}

// ModeResponse represents the current mode state.
type ModeResponse struct {
	Mode              string `json:"mode"`
	AzureEnabled      bool   `json:"azureEnabled"`
	AzureStatus       string `json:"azureStatus"` // "connected", "disconnected", "error"
	AzureRealtime     bool   `json:"azureRealtime"`
	ResourceCount     int    `json:"resourceCount"`
	ConnectionIssue   string `json:"connectionIssue,omitempty"`
	ConnectionMessage string `json:"connectionMessage,omitempty"`
}

// handleGetMode returns the current log source mode.
func (s *Server) handleGetMode(w http.ResponseWriter, _ *http.Request) {
	// Get current mode from server state
	s.modeMu.RLock()
	currentMode := s.currentMode
	s.modeMu.RUnlock()

	// Check if Azure logging is configured (logs.analytics section exists)
	azureEnabled := false
	azureStatus := "disabled"
	azureRealtime := false
	connectionMessage := ""

	azureYaml, err := loadAzureYaml(s.projectDir)
	if err == nil && azureYaml.Logs != nil && azureYaml.Logs.Analytics != nil {
		// Azure logging is configured
		azureEnabled = true
		azureStatus = "connected" // Assume connected if configured
		azureRealtime = azureYaml.Logs.Analytics.Realtime
	} else if err != nil {
		// Error loading azure.yaml
		connectionMessage = "Could not load azure.yaml: " + err.Error()
	} else if azureYaml.Logs == nil || azureYaml.Logs.Analytics == nil {
		// Azure logging not configured
		connectionMessage = "Azure logging not configured in azure.yaml"
	}

	response := ModeResponse{
		Mode:              string(currentMode),
		AzureEnabled:      azureEnabled,
		AzureStatus:       azureStatus,
		AzureRealtime:     azureRealtime,
		ConnectionMessage: connectionMessage,
	}

	WriteJSONSuccess(w, response)
}

// handleSetMode changes the log source mode.
func (s *Server) handleSetMode(w http.ResponseWriter, r *http.Request) {
	var req ModeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "Invalid request body", err)
		return
	}

	// Validate mode
	if req.Mode != string(service.LogModeLocal) && req.Mode != string(service.LogModeAzure) {
		BadRequest(w, "Invalid mode. Must be 'local' or 'azure'", nil)
		return
	}

	// If switching to Azure mode, verify it's configured
	if req.Mode == string(service.LogModeAzure) {
		azureYaml, err := loadAzureYaml(s.projectDir)
		if err != nil || azureYaml.Logs == nil || azureYaml.Logs.Analytics == nil {
			BadRequest(w, "Azure logging not configured. Add logs.analytics section to azure.yaml", nil)
			return
		}
	}

	// Mode is tracked client-side only - just return success with current status
	// The frontend will use the mode to determine which API endpoints to call

	// Store the mode in server state
	s.modeMu.Lock()
	s.currentMode = service.LogMode(req.Mode)
	s.modeMu.Unlock()

	azureEnabled := false
	azureStatus := "disabled"
	azureRealtime := false

	azureYaml, err := loadAzureYaml(s.projectDir)
	if err == nil && azureYaml.Logs != nil && azureYaml.Logs.Analytics != nil {
		azureEnabled = true
		azureStatus = "connected"
		azureRealtime = azureYaml.Logs.Analytics.Realtime
	}

	response := ModeResponse{
		Mode:          req.Mode,
		AzureEnabled:  azureEnabled,
		AzureStatus:   azureStatus,
		AzureRealtime: azureRealtime,
	}

	WriteJSONSuccess(w, response)
}

// handleModeRouter routes mode requests to the appropriate handler.
func (s *Server) handleModeRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetMode(w, r)
	case http.MethodPut, http.MethodPost:
		s.handleSetMode(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
