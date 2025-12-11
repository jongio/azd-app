// mode.go provides API endpoints for log source mode management.
package dashboard

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

// modeNotConfiguredMessage provides actionable guidance when trying to switch to Azure mode.
const modeNotConfiguredMessage = `Azure logging not configured. To enable:
1. Add to azure.yaml:
   logs:
     azure:
       enabled: true
2. Restart 'azd app run'

For more info: https://aka.ms/azd-app/azure-logs`

// ModeRequest represents a request to change the log source mode.
type ModeRequest struct {
	Mode string `json:"mode"` // "local" or "azure"
}

// ModeResponse represents the current mode state.
type ModeResponse struct {
	Mode              string `json:"mode"`
	AzureEnabled      bool   `json:"azureEnabled"`
	AzureStatus       string `json:"azureStatus"` // "connected", "disconnected", "error"
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

	// Mode switching is deprecated - always return local mode
	response := ModeResponse{
		Mode:         string(service.LogModeLocal),
		AzureEnabled: false,
		AzureStatus:  "disabled",
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

	// Mode switching is deprecated - return error
	writeJSONError(w, http.StatusBadRequest, "Mode switching is deprecated. Use /api/azure/logs endpoint directly.", nil)
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
