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

	// Get the azure log buffer from log manager
	logMgr := service.GetLogManager(s.projectDir)
	azBuffer := logMgr.GetAzureLogBuffer()

	response := ModeResponse{
		Mode:         string(service.LogModeLocal),
		AzureEnabled: false,
	}

	if azBuffer != nil {
		status := azBuffer.GetAzureStatus()
		response.Mode = string(status.Mode)
		response.AzureEnabled = status.Enabled
		response.ResourceCount = status.ResourceCount
		response.ConnectionIssue = status.ConnectionIssue
		response.ConnectionMessage = status.ConnectionMessage

		if status.Connected {
			response.AzureStatus = "connected"
		} else if status.Enabled {
			response.AzureStatus = "disconnected"
		} else {
			response.AzureStatus = "disabled"
		}
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
	var mode service.LogMode
	switch req.Mode {
	case "local":
		mode = service.LogModeLocal
	case "azure":
		mode = service.LogModeAzure
	default:
		writeJSONError(w, http.StatusBadRequest, "Invalid mode. Use 'local' or 'azure'", nil)
		return
	}

	// Get the azure log buffer from log manager
	logMgr := service.GetLogManager(s.projectDir)
	azBuffer := logMgr.GetAzureLogBuffer()

	if azBuffer == nil {
		writeJSONError(w, http.StatusServiceUnavailable, modeNotConfiguredMessage, nil)
		return
	}

	// Set the mode
	if err := azBuffer.SetMode(mode); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to set mode", err)
		return
	}

	// Return updated status
	status := azBuffer.GetAzureStatus()
	response := ModeResponse{
		Mode:          string(status.Mode),
		AzureEnabled:  status.Enabled,
		ResourceCount: status.ResourceCount,
	}

	if status.Connected {
		response.AzureStatus = "connected"
	} else if status.Enabled {
		response.AzureStatus = "disconnected"
	} else {
		response.AzureStatus = "disabled"
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
