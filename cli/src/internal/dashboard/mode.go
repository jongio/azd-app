// mode.go provides API endpoints for log source mode management.
package dashboard

import (
	"encoding/json"
	"log"
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
func (s *Server) handleGetMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get current mode from server state
	s.modeMu.RLock()
	currentMode := s.currentMode
	s.modeMu.RUnlock()

	// Check if Azure logging is configured (logs.analytics section exists)
	azureEnabled := false
	azureStatus := "disabled"
	azureRealtime := false

	azureYaml, err := loadAzureYaml(s.projectDir)
	if err == nil && azureYaml.Logs != nil && azureYaml.Logs.Analytics != nil {
		// Azure logging is configured
		azureEnabled = true
		azureStatus = "connected" // Assume connected if configured
		azureRealtime = azureYaml.Logs.Analytics.Realtime
	}

	response := ModeResponse{
		Mode:          string(currentMode),
		AzureEnabled:  azureEnabled,
		AzureStatus:   azureStatus,
		AzureRealtime: azureRealtime,
	}

	if err := writeJSON(w, response); err != nil {
		log.Printf("Failed to write mode response: %v", err)
	}
}

// handleSetMode changes the log source mode.
func (s *Server) handleSetMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ModeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate mode
	if req.Mode != string(service.LogModeLocal) && req.Mode != string(service.LogModeAzure) {
		writeJSONError(w, http.StatusBadRequest, "Invalid mode. Must be 'local' or 'azure'", nil)
		return
	}

	// If switching to Azure mode, verify it's configured
	if req.Mode == string(service.LogModeAzure) {
		azureYaml, err := loadAzureYaml(s.projectDir)
		if err != nil || azureYaml.Logs == nil || azureYaml.Logs.Analytics == nil {
			writeJSONError(w, http.StatusBadRequest, "Azure logging not configured. Add logs.analytics section to azure.yaml", nil)
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

	if err := writeJSON(w, response); err != nil {
		log.Printf("Failed to write mode response: %v", err)
	}
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
