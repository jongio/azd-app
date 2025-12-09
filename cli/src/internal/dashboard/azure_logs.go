// azure_logs.go provides API endpoints for Azure log streaming.
package dashboard

import (
	"log"
	"net/http"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

// azureNotConfiguredMessage provides actionable guidance when Azure logging is not configured.
const azureNotConfiguredMessage = `Azure logging not configured. To enable:
1. Add to azure.yaml:
   logs:
     azure:
       enabled: true
2. Restart 'azd app run'

For more info: https://aka.ms/azd-app/azure-logs`

// EnableAzureResponse represents the response from enabling Azure logging.
type EnableAzureResponse struct {
	Enabled bool   `json:"enabled"`
	Message string `json:"message"`
}

// AzureStatusResponse represents the Azure log streaming status.
type AzureStatusResponse struct {
	Mode          string            `json:"mode"`
	Connected     bool              `json:"connected"`
	Enabled       bool              `json:"enabled"`
	ResourceCount int               `json:"resourceCount"`
	Resources     []ResourceSummary `json:"resources,omitempty"`
	WorkspaceID   string            `json:"workspaceId,omitempty"`
}

// ResourceSummary provides a summary of a discovered Azure resource.
type ResourceSummary struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	ResourceID string `json:"resourceId,omitempty"`
	HasLogs    bool   `json:"hasLogs"`
}

// handleAzureStatus returns the Azure log streaming status.
func (s *Server) handleAzureStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logMgr := service.GetLogManager(s.projectDir)
	azBuffer := logMgr.GetAzureLogBuffer()

	response := AzureStatusResponse{
		Mode:      string(service.LogModeLocal),
		Connected: false,
		Enabled:   false,
	}

	if azBuffer != nil {
		status := azBuffer.GetAzureStatus()
		response.Mode = string(status.Mode)
		response.Connected = status.Connected
		response.Enabled = status.Enabled
		response.ResourceCount = status.ResourceCount
	}

	if err := writeJSON(w, response); err != nil {
		log.Printf("Failed to write azure status response: %v", err)
	}
}

// handleEnableAzureLogging enables Azure logging by adding the config to azure.yaml.
// POST /api/azure/enable
func (s *Server) handleEnableAzureLogging(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	classificationsMu.Lock()
	defer classificationsMu.Unlock()

	// Load existing azure.yaml
	azureYaml, err := loadAzureYaml(s.projectDir)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load azure.yaml", err)
		return
	}

	// Check if already enabled in config
	if azureYaml.Logs != nil && azureYaml.Logs.Azure != nil && azureYaml.Logs.Azure.Enabled {
		// Config already has it enabled, but buffer might not be initialized
		// Initialize it now if needed
		logMgr := service.GetLogManager(s.projectDir)
		if logMgr.GetAzureLogBuffer() == nil {
			azBuffer := service.NewAzureLogBuffer(azureYaml.Logs.Azure, s.projectDir)
			logMgr.SetAzureLogBuffer(azBuffer)
			log.Printf("Azure log buffer initialized (was configured but not running): %s", s.projectDir)
		}

		response := EnableAzureResponse{
			Enabled: true,
			Message: "Azure logging is enabled. Switch to Azure mode to view logs.",
		}
		if err := writeJSON(w, response); err != nil {
			log.Printf("Failed to write enable azure response: %v", err)
		}
		return
	}

	// Initialize logs section if needed
	if azureYaml.Logs == nil {
		azureYaml.Logs = &service.LogsConfig{}
	}

	// Initialize azure section if needed
	if azureYaml.Logs.Azure == nil {
		azureYaml.Logs.Azure = &service.AzureLogsConfig{}
	}

	// Enable Azure logging
	azureYaml.Logs.Azure.Enabled = true

	// Save azure.yaml
	if err := saveAzureYaml(s.projectDir, azureYaml); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to save azure.yaml", err)
		return
	}

	log.Printf("Azure logging enabled in azure.yaml for project: %s", s.projectDir)

	// Initialize Azure log buffer immediately so user doesn't need to restart
	logMgr := service.GetLogManager(s.projectDir)
	if logMgr.GetAzureLogBuffer() == nil {
		azBuffer := service.NewAzureLogBuffer(azureYaml.Logs.Azure, s.projectDir)
		logMgr.SetAzureLogBuffer(azBuffer)
		log.Printf("Azure log buffer initialized for project: %s", s.projectDir)
	}

	response := EnableAzureResponse{
		Enabled: true,
		Message: "Azure logging enabled! Switch to Azure mode to view logs.",
	}

	w.WriteHeader(http.StatusOK)
	if err := writeJSON(w, response); err != nil {
		log.Printf("Failed to write enable azure response: %v", err)
	}
}

// handleAzureLogs returns recent Azure logs.
func (s *Server) handleAzureLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	serviceName := r.URL.Query().Get("service")
	tailStr := r.URL.Query().Get("tail")

	// Default to 500 lines with bounds checking
	tail := 500
	if tailStr != "" {
		if n, err := parseIntParam(tailStr); err == nil && n > 0 {
			tail = n
		}
	}
	if tail > 10000 {
		tail = 10000
	}

	logMgr := service.GetLogManager(s.projectDir)
	azBuffer := logMgr.GetAzureLogBuffer()

	if azBuffer == nil {
		writeJSONError(w, http.StatusServiceUnavailable, azureNotConfiguredMessage, nil)
		return
	}

	var logs []service.LogEntry
	if serviceName != "" {
		logs = azBuffer.GetRecentLogs(serviceName, tail)
	} else {
		logs = azBuffer.GetAllRecentLogs(tail)
	}

	if err := writeJSON(w, logs); err != nil {
		log.Printf("Failed to write azure logs response: %v", err)
	}
}

// handleAzureLogsStream streams Azure logs via WebSocket.
func (s *Server) handleAzureLogsStream(w http.ResponseWriter, r *http.Request) {
	// Upgrade connection to WebSocket
	rawConn, err := acceptWebSocket(w, r)
	if err != nil {
		if err != http.ErrAbortHandler {
			log.Printf("WebSocket upgrade failed: %v", err)
		}
		return
	}

	client := newWSClient(rawConn)
	conn := &clientConn{client: client}
	defer client.close()

	logMgr := service.GetLogManager(s.projectDir)
	azBuffer := logMgr.GetAzureLogBuffer()

	if azBuffer == nil {
		if err := conn.writeWebSocketJSON(map[string]string{"error": azureNotConfiguredMessage}); err != nil {
			log.Printf("Failed to write error to websocket: %v", err)
		}
		return
	}

	// Subscribe to Azure logs
	subscription := azBuffer.Subscribe()
	defer azBuffer.Unsubscribe(subscription)

	// Stream logs to WebSocket
	for {
		select {
		case entry, ok := <-subscription:
			if !ok {
				return // Channel closed
			}
			if err := conn.writeWebSocketJSON(entry); err != nil {
				if !isExpectedCloseError(err) {
					log.Printf("WebSocket write error: %v", err)
				}
				return
			}
		case <-s.stopChan:
			return
		}
	}
}

// parseIntParam parses an integer query parameter.
func parseIntParam(s string) (int, error) {
	var n int
	_, err := parseIntParamWithFormat(s, &n)
	return n, err
}

// parseIntParamWithFormat parses using Sscanf.
func parseIntParamWithFormat(s string, n *int) (int, error) {
	return parseIntFromString(s, n)
}

// parseIntFromString parses an integer from a string.
func parseIntFromString(s string, n *int) (int, error) {
	var count int
	var val int
	// Use simple parsing
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		val = val*10 + int(c-'0')
		count++
	}
	if count > 0 {
		*n = val
		return count, nil
	}
	return 0, errInvalidInt
}

var errInvalidInt = &invalidIntError{}

type invalidIntError struct{}

func (e *invalidIntError) Error() string {
	return "invalid integer"
}
