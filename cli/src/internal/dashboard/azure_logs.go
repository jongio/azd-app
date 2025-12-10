// azure_logs.go provides API endpoints for Azure log streaming.
package dashboard

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/azure"
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

// AzureQueryResponse represents the KQL query used for a service.
type AzureQueryResponse struct {
	Service      string `json:"service"`
	ResourceType string `json:"resourceType"`
	Query        string `json:"query"`
}

// HistoricalQueryRequest represents a request to query historical logs.
type HistoricalQueryRequest struct {
	Service  string `json:"service"`
	Timespan string `json:"timespan"` // ISO 8601 duration, e.g., "PT1H"
	Query    string `json:"query"`    // Custom KQL query (optional)
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
}

// HistoricalQueryResponse represents the response from a historical log query.
type HistoricalQueryResponse struct {
	Logs          []service.LogEntry `json:"logs"`
	Total         int                `json:"total"`
	HasMore       bool               `json:"hasMore"`
	ExecutionTime int64              `json:"executionTime"` // milliseconds
}

// handleAzureLogsQuery executes a historical log query against Azure Log Analytics.
// POST /api/azure/logs/query
func (s *Server) handleAzureLogsQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := readLimitedBody(r, maxRequestBodySize)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Failed to read request body", err)
		return
	}

	var req HistoricalQueryRequest
	if err := decodeJSON(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	if req.Service == "" {
		writeJSONError(w, http.StatusBadRequest, "service is required", nil)
		return
	}

	// Parse timespan (ISO 8601 duration)
	timespan, err := parseISODuration(req.Timespan)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid timespan format", err)
		return
	}

	// Apply defaults
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	// Get Azure log buffer
	logMgr := service.GetLogManager(s.projectDir)
	azBuffer := logMgr.GetAzureLogBuffer()

	if azBuffer == nil {
		writeJSONError(w, http.StatusServiceUnavailable, azureNotConfiguredMessage, nil)
		return
	}

	// Execute the query
	ctx := r.Context()
	result, err := azBuffer.QueryHistoricalLogs(ctx, req.Service, timespan, req.Query, limit, offset)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Query failed: "+err.Error(), err)
		return
	}

	response := HistoricalQueryResponse{
		Logs:          result.Logs,
		Total:         result.Total,
		HasMore:       result.HasMore,
		ExecutionTime: result.ExecutionTime,
	}

	if err := writeJSON(w, response); err != nil {
		log.Printf("Failed to write historical query response: %v", err)
	}
}

// parseISODuration parses an ISO 8601 duration string like "PT1H" or "PT30M".
func parseISODuration(s string) (time.Duration, error) {
	if s == "" {
		return time.Hour, nil // Default to 1 hour
	}

	// Simple parser for common durations: PT1H, PT30M, PT15M, PT6H, PT24H, P1D
	if len(s) < 3 {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}

	if s[0] != 'P' {
		return 0, fmt.Errorf("duration must start with P: %s", s)
	}

	// Handle P1D (days) format
	if s[len(s)-1] == 'D' {
		var days int
		if _, err := parseIntFromString(s[1:len(s)-1], &days); err != nil {
			return 0, fmt.Errorf("invalid day value: %s", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	// Handle PTnH, PTnM, PTnS format
	if len(s) < 4 || s[1] != 'T' {
		return 0, fmt.Errorf("invalid duration format: %s", s)
	}

	remaining := s[2:] // After "PT"
	var duration time.Duration

	for len(remaining) > 0 {
		// Find the number
		numEnd := 0
		for numEnd < len(remaining) && remaining[numEnd] >= '0' && remaining[numEnd] <= '9' {
			numEnd++
		}

		if numEnd == 0 || numEnd >= len(remaining) {
			return 0, fmt.Errorf("invalid duration format: %s", s)
		}

		var value int
		if _, err := parseIntFromString(remaining[:numEnd], &value); err != nil {
			return 0, fmt.Errorf("invalid numeric value in duration: %s", s)
		}

		unit := remaining[numEnd]
		remaining = remaining[numEnd+1:]

		switch unit {
		case 'H':
			duration += time.Duration(value) * time.Hour
		case 'M':
			duration += time.Duration(value) * time.Minute
		case 'S':
			duration += time.Duration(value) * time.Second
		default:
			return 0, fmt.Errorf("unknown duration unit: %c", unit)
		}
	}

	if duration == 0 {
		return time.Hour, nil // Default fallback
	}

	return duration, nil
}

// handleAzureQueryRouter routes query API requests.
// GET /api/azure/query?service=<name> - get query for service
// PUT /api/azure/query - save custom query for service
func (s *Server) handleAzureQueryRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleAzureQuery(w, r)
	case http.MethodPut:
		s.handleSaveAzureQuery(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAzureQuery returns the KQL query being used for a service.
// GET /api/azure/query?service=<name>
func (s *Server) handleAzureQuery(w http.ResponseWriter, r *http.Request) {

	serviceName := r.URL.Query().Get("service")
	if serviceName == "" {
		writeJSONError(w, http.StatusBadRequest, "service parameter required", nil)
		return
	}

	logMgr := service.GetLogManager(s.projectDir)
	azBuffer := logMgr.GetAzureLogBuffer()

	if azBuffer == nil {
		writeJSONError(w, http.StatusServiceUnavailable, azureNotConfiguredMessage, nil)
		return
	}

	// Get the resource type for this service
	resourceType, resourceName := azBuffer.GetServiceResourceInfo(serviceName)
	if resourceType == "" {
		writeJSONError(w, http.StatusNotFound, "Service not found or not discovered", nil)
		return
	}

	// Check if there's a custom query saved in azure.yaml
	classificationsMu.RLock()
	azureYaml, err := loadAzureYaml(s.projectDir)
	classificationsMu.RUnlock()

	var query string
	isCustom := false

	if err == nil && azureYaml.Logs != nil && azureYaml.Logs.Azure != nil &&
		azureYaml.Logs.Azure.Queries != nil {
		if customQuery, exists := azureYaml.Logs.Azure.Queries[serviceName]; exists && customQuery != "" {
			query = customQuery
			isCustom = true
			resourceType = "custom"
		}
	}

	// Fall back to default query if no custom query
	if !isCustom {
		query = azure.GetDefaultQuery(azure.ResourceType(resourceType))
		// Substitute placeholders for display
		query = substituteQueryPlaceholders(query, resourceName, "30m")
	}

	response := AzureQueryResponse{
		Service:      serviceName,
		ResourceType: resourceType,
		Query:        query,
	}

	if err := writeJSON(w, response); err != nil {
		log.Printf("Failed to write azure query response: %v", err)
	}
}

// substituteQueryPlaceholders replaces placeholders in a query for display.
func substituteQueryPlaceholders(query, serviceName, timespan string) string {
	query = replaceAll(query, "{serviceName}", serviceName)
	query = replaceAll(query, "{timespan}", timespan)
	return query
}

// replaceAll is a simple string replacement (avoids importing strings package).
func replaceAll(s, old, new string) string {
	result := ""
	for {
		i := indexOf(s, old)
		if i < 0 {
			return result + s
		}
		result += s[:i] + new
		s = s[i+len(old):]
	}
}

// indexOf finds the first occurrence of substr in s.
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// SaveQueryRequest represents the request body for saving a custom query.
type SaveQueryRequest struct {
	Service string `json:"service"`
	Query   string `json:"query"`
}

// handleSaveAzureQuery saves a custom KQL query for a service to azure.yaml.
// PUT /api/azure/query
func (s *Server) handleSaveAzureQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := readLimitedBody(r, maxRequestBodySize)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Failed to read request body", err)
		return
	}

	var req SaveQueryRequest
	if err := decodeJSON(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	if req.Service == "" {
		writeJSONError(w, http.StatusBadRequest, "service is required", nil)
		return
	}
	if req.Query == "" {
		writeJSONError(w, http.StatusBadRequest, "query is required", nil)
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

	// Check if service exists
	if _, exists := azureYaml.Services[req.Service]; !exists {
		writeJSONError(w, http.StatusNotFound, "Service not found", nil)
		return
	}

	// Initialize logs.azure section if needed
	if azureYaml.Logs == nil {
		azureYaml.Logs = &service.LogsConfig{}
	}
	if azureYaml.Logs.Azure == nil {
		azureYaml.Logs.Azure = &service.AzureLogsConfig{Enabled: true}
	}
	if azureYaml.Logs.Azure.Queries == nil {
		azureYaml.Logs.Azure.Queries = make(map[string]string)
	}

	// Save the custom query for this service
	azureYaml.Logs.Azure.Queries[req.Service] = req.Query

	// Save azure.yaml
	if err := saveAzureYaml(s.projectDir, azureYaml); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to save azure.yaml", err)
		return
	}

	log.Printf("Saved custom KQL query for service %s", req.Service)

	// Return updated query info
	response := AzureQueryResponse{
		Service:      req.Service,
		ResourceType: "custom",
		Query:        req.Query,
	}

	if err := writeJSON(w, response); err != nil {
		log.Printf("Failed to write save query response: %v", err)
	}
}

// readLimitedBody reads up to maxSize bytes from the request body.
func readLimitedBody(r *http.Request, maxSize int64) ([]byte, error) {
	return readBodyWithLimit(r.Body, maxSize)
}

// readBodyWithLimit reads up to maxSize from a reader.
func readBodyWithLimit(reader interface{ Read([]byte) (int, error) }, maxSize int64) ([]byte, error) {
	data := make([]byte, 0, 1024)
	buf := make([]byte, 1024)
	var total int64
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			total += int64(n)
			if total > maxSize {
				return nil, errBodyTooLarge
			}
			data = append(data, buf[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, err
		}
	}
	return data, nil
}

var errBodyTooLarge = &bodyTooLargeError{}

type bodyTooLargeError struct{}

func (e *bodyTooLargeError) Error() string {
	return "request body too large"
}

// decodeJSON unmarshals JSON from bytes.
func decodeJSON(data []byte, v interface{}) error {
	return jsonUnmarshal(data, v)
}

// jsonUnmarshal is a wrapper for JSON unmarshaling.
func jsonUnmarshal(data []byte, v interface{}) error {
	i := 0
	return unmarshalJSONValue(data, &i, v)
}

// unmarshalJSONValue is a simple JSON parser (delegate to encoding/json in practice).
// For production use, this would use encoding/json.Unmarshal
func unmarshalJSONValue(data []byte, _ *int, v interface{}) error {
	// Use standard library
	return parseJSONStandard(data, v)
}

// parseJSONStandard uses standard encoding/json.
func parseJSONStandard(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
