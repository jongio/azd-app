// azure_logs.go provides API endpoints for Azure log streaming.
package dashboard

import (
	"log"
	"net/http"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

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
	Name         string `json:"name"`
	Type         string `json:"type"`
	ResourceID   string `json:"resourceId,omitempty"`
	HasLogs      bool   `json:"hasLogs"`
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
		writeJSONError(w, http.StatusServiceUnavailable, "Azure logging not configured", nil)
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
		if err := conn.writeWebSocketJSON(map[string]string{"error": "Azure logging not configured"}); err != nil {
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
