// azure_logs.go provides API endpoints for Azure log streaming.
package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/azure"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// REMOVED: handleAzureStatus - deprecated v1 endpoint

// AzureServicesResponse represents the list of available services.
type AzureServicesResponse struct {
	Services []string `json:"services"`
}

// handleAzureServices returns the list of services that have Azure resources.
// GET /api/azure/services
func (s *Server) handleAzureServices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse SERVICE_*_NAME environment variables to get service list
	serviceNames := extractServiceNamesFromEnv()

	response := AzureServicesResponse{
		Services: serviceNames,
	}

	if err := writeJSON(w, response); err != nil {
		log.Printf("Failed to write azure services response: %v", err)
	}
}

// EnableAzureResponse represents the response from enabling Azure logging.
type EnableAzureResponse struct {
	Enabled bool   `json:"enabled"`
	Message string `json:"message"`
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

	// Check if already enabled (presence of analytics config means enabled)
	if azureYaml.Logs != nil && azureYaml.Logs.Analytics != nil {
		response := EnableAzureResponse{
			Enabled: true,
			Message: "Azure logging is already enabled",
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

	// Initialize global analytics section (presence means enabled)
	if azureYaml.Logs.Analytics == nil {
		azureYaml.Logs.Analytics = &service.AnalyticsConfigGlobal{}
	}

	// Save azure.yaml
	if err := saveAzureYaml(s.projectDir, azureYaml); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to save azure.yaml", err)
		return
	}

	log.Printf("Azure logging enabled in azure.yaml for project: %s", s.projectDir)

	response := EnableAzureResponse{
		Enabled: true,
		Message: "Azure logging enabled! Refresh to start viewing logs.",
	}

	w.WriteHeader(http.StatusOK)
	if err := writeJSON(w, response); err != nil {
		log.Printf("Failed to write enable azure response: %v", err)
	}
}

// REMOVED: handleEnableAzureLogging - deprecated v1 endpoint (configure via azure.yaml directly)

// AzureLogsResponse represents the structured response for Azure logs.
type AzureLogsResponse struct {
	Status    string              `json:"status"`              // "ok" | "error"
	Logs      []service.LogEntry  `json:"logs,omitempty"`      // Log entries
	Count     int                 `json:"count"`               // Number of logs returned
	Timestamp time.Time           `json:"timestamp"`           // Response timestamp
	Error     *ErrorInfo          `json:"error,omitempty"`     // Error details if status=error
}

// ErrorInfo provides actionable error information with documentation links.
type ErrorInfo struct {
	Message string `json:"message"`         // Human-readable error message
	Code    string `json:"code"`            // Error code: "AUTH_EXPIRED", "NOT_DEPLOYED", etc.
	Action  string `json:"action"`          // What the user should do
	Command string `json:"command"`         // CLI command to run (optional)
	DocsURL string `json:"docsUrl"`         // Documentation URL
}

// handleAzureLogs returns recent Azure logs with structured error handling.
// GET /api/azure/logs?service=<name>&since=<duration>
func (s *Server) handleAzureLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	serviceName := r.URL.Query().Get("service")
	sinceStr := r.URL.Query().Get("since")
	tailStr := r.URL.Query().Get("tail")

	// Parse since duration (e.g., "1h", "30m")
	since := 1 * time.Hour // Default to 1 hour
	if sinceStr != "" {
		if d, err := time.ParseDuration(sinceStr); err == nil && d > 0 {
			since = d
		}
	}

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

	// Try to fetch logs using standalone fetcher for reliability
	ctx := r.Context()
	var services []string
	if serviceName != "" {
		services = []string{serviceName}
	}

	config := azure.StandaloneLogsConfig{
		ProjectDir: s.projectDir,
		Services:   services,
		Since:      since,
		Limit:      tail,
	}

	// Debug logging
	log.Printf("[DEBUG] Azure logs request - service: %q, since: %v, tail: %d", serviceName, since, tail)

	azureLogs, err := azure.FetchAzureLogsStandalone(ctx, config)
	
	// Debug logging
	log.Printf("[DEBUG] Azure logs response - count: %d, err: %v", len(azureLogs), err)
	
	response := AzureLogsResponse{
		Timestamp: time.Now(),
	}

	if err != nil {
		// Map error to structured ErrorInfo
		response.Status = "error"
		response.Count = 0
		response.Error = mapAzureErrorToInfo(err)
		
		// Set appropriate HTTP status code
		statusCode := http.StatusInternalServerError
		if response.Error.Code == "AUTH_EXPIRED" || response.Error.Code == "AUTH_REQUIRED" {
			statusCode = http.StatusUnauthorized
		} else if response.Error.Code == "NOT_DEPLOYED" || response.Error.Code == "NO_WORKSPACE" {
			statusCode = http.StatusServiceUnavailable
		} else if response.Error.Code == "NO_PERMISSION" {
			statusCode = http.StatusForbidden
		}
		
		w.WriteHeader(statusCode)
		if err := writeJSON(w, response); err != nil {
			log.Printf("Failed to write azure logs error response: %v", err)
		}
		return
	}

	// Convert azure.LogEntry to service.LogEntry
	logs := make([]service.LogEntry, len(azureLogs))
	for i, azLog := range azureLogs {
		logs[i] = service.LogEntry{
			Service:   azLog.Service,
			Message:   azLog.Message,
			Level:     convertAzureLogLevel(azLog.Level),
			Timestamp: azLog.Timestamp,
			Source:    service.LogSourceAzure,
			AzureMetadata: &service.AzureLogMetadata{
				ResourceType:  azLog.ResourceType,
				ContainerName: azLog.ContainerName,
				InstanceID:    azLog.InstanceID,
			},
		}
	}

	// Success response
	response.Status = "ok"
	response.Logs = logs
	response.Count = len(logs)

	if err := writeJSON(w, response); err != nil {
		log.Printf("Failed to write azure logs response: %v", err)
	}
}

// convertAzureLogLevel converts azure.LogLevel to service.LogLevel.
func convertAzureLogLevel(azLevel azure.LogLevel) service.LogLevel {
	switch azLevel {
	case azure.LogLevelInfo:
		return service.LogLevelInfo
	case azure.LogLevelWarn:
		return service.LogLevelWarn
	case azure.LogLevelError:
		return service.LogLevelError
	case azure.LogLevelDebug:
		return service.LogLevelDebug
	default:
		return service.LogLevelInfo
	}
}

// mapAzureErrorToInfo converts Azure errors to structured ErrorInfo with docs links.
func mapAzureErrorToInfo(err error) *ErrorInfo {
	if azErr, ok := err.(*azure.AzureLogsError); ok {
		info := &ErrorInfo{
			Message: azErr.Message,
			Code:    azErr.Code,
			Action:  azErr.Action,
			Command: azErr.Command,
		}
		
		// Add documentation URLs based on error code
		switch azErr.Code {
		case "AUTH_EXPIRED", "AUTH_REQUIRED":
			info.DocsURL = "https://aka.ms/azd/app/logs/troubleshoot#auth"
		case "NOT_DEPLOYED":
			info.DocsURL = "https://aka.ms/azd/app/logs/setup"
		case "NO_WORKSPACE":
			info.DocsURL = "https://aka.ms/azd/app/logs/configure"
		case "NO_PERMISSION":
			info.DocsURL = "https://aka.ms/azd/app/logs/troubleshoot#permissions"
		default:
			info.DocsURL = "https://aka.ms/azd/app/logs/troubleshoot"
		}
		
		return info
	}
	
	// Generic error
	return &ErrorInfo{
		Message: err.Error(),
		Code:    "UNKNOWN",
		Action:  "Check logs for more details",
		DocsURL: "https://aka.ms/azd/app/logs/troubleshoot",
	}
}

// handleAzureLogsStream handles WebSocket streaming of Azure logs via polling.
// WS /api/azure/logs/stream?service=<name>
func (s *Server) handleAzureLogsStream(w http.ResponseWriter, r *http.Request) {
	serviceName := r.URL.Query().Get("service")

	// Upgrade connection to WebSocket
	rawConn, err := acceptWebSocket(w, r)
	if err != nil {
		if err != http.ErrAbortHandler {
			log.Printf("Azure logs WebSocket upgrade failed: %v", err)
		}
		return
	}

	// Wrap connection with mutex for safe concurrent writes
	client := newWSClient(rawConn)
	conn := &clientConn{client: client}
	defer client.close()

	// Track last seen timestamp to avoid duplicates
	lastTimestamp := time.Now().Add(-30 * time.Minute) // Start with 30m ago

	// Poll for new logs every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	log.Printf("Azure logs WebSocket connected for service: %s", serviceName)

	// Initial fetch
	ctx := r.Context()
	if err := fetchAndSendAzureLogs(ctx, s.projectDir, serviceName, lastTimestamp, conn, &lastTimestamp); err != nil {
		log.Printf("Initial Azure logs fetch failed: %v", err)
	}

	// Stream updates
	for {
		select {
		case <-ticker.C:
			if err := fetchAndSendAzureLogs(ctx, s.projectDir, serviceName, lastTimestamp, conn, &lastTimestamp); err != nil {
				log.Printf("Azure logs fetch failed: %v", err)
				return
			}
		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// fetchAndSendAzureLogs fetches logs since lastTimestamp and sends them via WebSocket.
func fetchAndSendAzureLogs(ctx context.Context, projectDir string, serviceName string, since time.Time, conn *clientConn, lastTimestamp *time.Time) error {
	var services []string
	if serviceName != "" {
		services = []string{serviceName}
	}

	config := azure.StandaloneLogsConfig{
		ProjectDir: projectDir,
		Services:   services,
		Since:      time.Since(since),
		Limit:      100,
	}

	azureLogs, err := azure.FetchAzureLogsStandalone(ctx, config)
	if err != nil {
		// Send error message to client
		errMsg := map[string]string{
			"error": fmt.Sprintf("Failed to fetch Azure logs: %v", err),
		}
		if writeErr := conn.writeWebSocketJSON(errMsg); writeErr != nil {
			return writeErr
		}
		return err
	}

	// Filter logs newer than last timestamp and send them
	newTimestamp := *lastTimestamp
	for _, azLog := range azureLogs {
		if azLog.Timestamp.After(since) {
			entry := service.LogEntry{
				Service:   azLog.Service,
				Message:   azLog.Message,
				Level:     convertAzureLogLevel(azLog.Level),
				Timestamp: azLog.Timestamp,
				Source:    service.LogSourceAzure,
				AzureMetadata: &service.AzureLogMetadata{
					ResourceType:  azLog.ResourceType,
					ContainerName: azLog.ContainerName,
					InstanceID:    azLog.InstanceID,
				},
			}

			if err := conn.writeWebSocketJSON(entry); err != nil {
				if !isExpectedCloseError(err) {
					log.Printf("Azure logs WebSocket write error: %v", err)
				}
				return err
			}

			// Track latest timestamp
			if azLog.Timestamp.After(newTimestamp) {
				newTimestamp = azLog.Timestamp
			}
		}
	}

	// Update last timestamp
	if newTimestamp.After(*lastTimestamp) {
		*lastTimestamp = newTimestamp
	}

	return nil
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

// REMOVED: handleAzureLogsQuery - deprecated v1 endpoint (use /api/azure/logs with query params)

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

	// Check if there's a custom query saved in azure.yaml
	classificationsMu.RLock()
	azureYaml, err := loadAzureYaml(s.projectDir)
	classificationsMu.RUnlock()

	var query string
	var resourceType string

	// Check service-level analytics config first
	if err == nil {
		if svc, ok := azureYaml.Services[serviceName]; ok && svc.Logs != nil && svc.Logs.Analytics != nil {
			if svc.Logs.Analytics.Query != "" {
				query = svc.Logs.Analytics.Query
				resourceType = "custom"
			}
		}
		// Project-level analytics only has workspace/polling/timespan; no Query field
	}

	// Fall back to default query if no custom query
	if query == "" {
		// Get resource type from environment variables
		resourceType = "containerapp" // Default assumption
		query = azure.GetDefaultQuery(azure.ResourceType(resourceType))
		// Substitute placeholders for display
		query = substituteQueryPlaceholders(query, serviceName, "30m")
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
	svc, exists := azureYaml.Services[req.Service]
	if !exists {
		writeJSONError(w, http.StatusNotFound, "Service not found", nil)
		return
	}

	// Initialize service logs.analytics section if needed
	if svc.Logs == nil {
		svc.Logs = &service.ServiceLogsConfig{}
	}
	if svc.Logs.Analytics == nil {
		svc.Logs.Analytics = &service.AnalyticsConfigService{}
	}

	// Save the custom query for this service
	svc.Logs.Analytics.Query = req.Query
	azureYaml.Services[req.Service] = svc

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

// HealthCheckResponse represents the overall health check result.
type HealthCheckResponse struct {
	Status    string        `json:"status"`    // "healthy" | "degraded" | "error"
	Checks    []HealthCheck `json:"checks"`    // Individual health checks
	DocsURL   string        `json:"docsUrl"`   // Documentation URL
	Timestamp time.Time     `json:"timestamp"` // When check was performed
}

// HealthCheck represents an individual health check result.
type HealthCheck struct {
	Name    string `json:"name"`            // Check name
	Status  string `json:"status"`          // "pass" | "warn" | "fail"
	Message string `json:"message"`         // Result message
	Fix     string `json:"fix,omitempty"`   // Fix instructions for failures
}

// handleAzureLogsHealth performs diagnostic checks for Azure logs troubleshooting.
// GET /api/azure/logs/health
func (s *Server) handleAzureLogsHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := HealthCheckResponse{
		Checks:    make([]HealthCheck, 0, 4),
		DocsURL:   "https://aka.ms/azd/app/logs/troubleshoot",
		Timestamp: time.Now(),
	}

	// Check 1: Authentication
	authCheck := s.checkAuthentication()
	response.Checks = append(response.Checks, authCheck)

	// Check 2: Workspace ID
	workspaceCheck := s.checkWorkspaceID()
	response.Checks = append(response.Checks, workspaceCheck)

	// Check 3: Services Deployed
	servicesCheck := s.checkServicesDeployed()
	response.Checks = append(response.Checks, servicesCheck)

	// Check 4: Connectivity
	connectivityCheck := s.checkConnectivity(workspaceCheck.Status == "pass")
	response.Checks = append(response.Checks, connectivityCheck)

	// Compute overall status
	response.Status = s.computeOverallStatus(response.Checks)

	if err := writeJSON(w, response); err != nil {
		log.Printf("Failed to write health check response: %v", err)
	}
}

// checkAuthentication verifies Azure credentials are available.
func (s *Server) checkAuthentication() HealthCheck {
	check := HealthCheck{
		Name: "Authentication",
	}

	// Try to create credentials
	cred, err := azure.NewLogAnalyticsCredential()
	if err != nil {
		check.Status = "fail"
		check.Message = "Azure credentials not available"
		check.Fix = "Run: azd auth login"
		return check
	}

	// Try to get a token to verify credentials work
	ctx, cancel := timeoutContext(5 * time.Second)
	defer cancel()

	err = azure.ValidateCredentials(ctx, cred)
	if err != nil {
		check.Status = "fail"
		check.Message = "Azure credentials invalid or expired"
		check.Fix = "Run: azd auth login"
		return check
	}

	check.Status = "pass"
	check.Message = "Azure credentials valid"
	return check
}

// checkWorkspaceID verifies Log Analytics workspace is configured.
func (s *Server) checkWorkspaceID() HealthCheck {
	check := HealthCheck{
		Name: "Workspace ID",
	}

	workspaceID := azure.GetWorkspaceIDFromEnv(s.projectDir)
	if workspaceID == "" {
		check.Status = "fail"
		check.Message = "Log Analytics workspace not configured"
		check.Fix = "Run: azd env refresh"
		return check
	}

	check.Status = "pass"
	check.Message = fmt.Sprintf("Workspace ID configured: %s", truncateMiddle(workspaceID, 20))
	return check
}

// checkServicesDeployed verifies at least one service is deployed.
func (s *Server) checkServicesDeployed() HealthCheck {
	check := HealthCheck{
		Name: "Services Deployed",
	}

	serviceCount := 0
	
	// Count SERVICE_*_NAME environment variables
	envVars := getAllEnvironmentVars()
	for key := range envVars {
		if strings.HasPrefix(key, "SERVICE_") && strings.HasSuffix(key, "_NAME") {
			serviceCount++
		}
	}

	if serviceCount == 0 {
		check.Status = "fail"
		check.Message = "No deployed services found"
		check.Fix = "Run: azd up"
		return check
	}

	check.Status = "pass"
	check.Message = fmt.Sprintf("Found %d deployed service(s)", serviceCount)
	return check
}

// checkConnectivity verifies ability to create Log Analytics client.
func (s *Server) checkConnectivity(hasWorkspace bool) HealthCheck {
	check := HealthCheck{
		Name: "Connectivity",
	}

	if !hasWorkspace {
		check.Status = "warn"
		check.Message = "Cannot verify connectivity without workspace ID"
		return check
	}

	workspaceID := azure.GetWorkspaceIDFromEnv(s.projectDir)
	cred, err := azure.NewLogAnalyticsCredential()
	if err != nil {
		check.Status = "warn"
		check.Message = "Cannot create credentials for connectivity test"
		return check
	}

	// Try to create client (this doesn't make actual queries)
	_, err = azure.NewLogAnalyticsClient(cred, workspaceID)
	if err != nil {
		check.Status = "fail"
		check.Message = fmt.Sprintf("Failed to create Log Analytics client: %v", err)
		check.Fix = "Check Azure subscription and permissions"
		return check
	}

	check.Status = "pass"
	check.Message = "Log Analytics client created successfully"
	return check
}

// computeOverallStatus determines overall health based on individual checks.
func (s *Server) computeOverallStatus(checks []HealthCheck) string {
	hasError := false
	hasWarn := false

	for _, check := range checks {
		switch check.Status {
		case "fail":
			hasError = true
		case "warn":
			hasWarn = true
		}
	}

	if hasError {
		return "error"
	}
	if hasWarn {
		return "degraded"
	}
	return "healthy"
}

// extractServiceNamesFromEnv parses SERVICE_*_NAME environment variables and returns service names.
// Returns azure.yaml service names (e.g., "api", "web", "worker") not Azure resource names.
func extractServiceNamesFromEnv() []string {
	serviceMap := make(map[string]bool)

	for _, line := range getEnvironment() {
		// Look for SERVICE_*_NAME pattern (e.g., SERVICE_API_NAME, SERVICE_CONTAINERAPP_API_NAME)
		if strings.HasPrefix(line, "SERVICE_") && strings.Contains(line, "_NAME=") && !strings.Contains(line, "_IMAGE_NAME=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				// Extract service name from key: SERVICE_CONTAINERAPP_API_NAME -> containerapp-api
				// or SERVICE_API_NAME -> api
				key := parts[0]
				key = strings.TrimPrefix(key, "SERVICE_")
				key = strings.TrimSuffix(key, "_NAME")
				
				// Convert to lowercase with hyphens (azure.yaml format)
				serviceName := strings.ToLower(strings.ReplaceAll(key, "_", "-"))
				
				if serviceName != "" {
					serviceMap[serviceName] = true
				}
			}
		}
	}

	// Convert map to sorted slice for consistent output
	services := make([]string, 0, len(serviceMap))
	for name := range serviceMap {
		services = append(services, name)
	}
	
	// Sort alphabetically for consistent ordering
	for i := 0; i < len(services); i++ {
		for j := i + 1; j < len(services); j++ {
			if services[i] > services[j] {
				services[i], services[j] = services[j], services[i]
			}
		}
	}

	return services
}

// truncateMiddle truncates a string in the middle, keeping prefix and suffix.
func truncateMiddle(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 3 {
		return s[:maxLen]
	}
	
	prefixLen := (maxLen - 3) / 2
	suffixLen := maxLen - 3 - prefixLen
	return s[:prefixLen] + "..." + s[len(s)-suffixLen:]
}

// getAllEnvironmentVars returns all environment variables as a map.
func getAllEnvironmentVars() map[string]string {
	result := make(map[string]string)
	for _, env := range getEnvironment() {
		if idx := strings.Index(env, "="); idx > 0 {
			key := env[:idx]
			value := env[idx+1:]
			result[key] = value
		}
	}
	return result
}

// =============================================================================
// Tables API - List available Log Analytics tables
// =============================================================================

// TablesResponse represents the response from the tables API.
type TablesResponse struct {
	Tables      []azure.TableInfo `json:"tables"`
	Recommended []string          `json:"recommended"`
	Workspace   string            `json:"workspace"`
	Categories  []TableCategory   `json:"categories"`
}

// TableCategory represents a category of tables.
type TableCategory struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Tables      []string `json:"tables"`
}

// handleAzureTables returns available Log Analytics tables.
// GET /api/azure/tables?resourceType=containerapp
func (s *Server) handleAzureTables(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resourceTypeStr := r.URL.Query().Get("resourceType")
	resourceType := azure.ResourceType(resourceTypeStr)
	if resourceType == "" {
		resourceType = azure.ResourceTypeContainerApp // Default
	}

	ctx := r.Context()
	workspaceID := azure.GetWorkspaceIDFromEnv(s.projectDir)
	
	var tables []azure.TableInfo
	var err error

	// Try to get live tables from Log Analytics
	if workspaceID != "" {
		cred, credErr := azure.NewLogAnalyticsCredential()
		if credErr == nil {
			client, clientErr := azure.NewLogAnalyticsClient(cred, workspaceID)
			if clientErr == nil {
				tables, err = client.ListAvailableTables(ctx)
			}
		}
	}

	// If we couldn't get live tables, use predefined tables
	if len(tables) == 0 || err != nil {
		tables = azure.GetAllKnownTables()
	}

	// Mark recommended tables for this resource type
	recommended := azure.GetRecommendedTables(resourceType)
	for i := range tables {
		tables[i].Recommended = azure.IsRecommendedTable(tables[i].Name, resourceType)
	}

	// Build categories
	categories := make([]TableCategory, 0, len(azure.TableCategories))
	for name, cat := range azure.TableCategories {
		categories = append(categories, TableCategory{
			Name:        name,
			DisplayName: cat.DisplayName,
			Tables:      cat.Tables,
		})
	}

	response := TablesResponse{
		Tables:      tables,
		Recommended: recommended,
		Workspace:   truncateMiddle(workspaceID, 20),
		Categories:  categories,
	}

	if err := writeJSON(w, response); err != nil {
		log.Printf("Failed to write tables response: %v", err)
	}
}

// =============================================================================
// Log Config API - Get/Save log configuration per service
// =============================================================================

// LogConfigResponse represents the log configuration for a service.
type LogConfigResponse struct {
	Service      string   `json:"service"`
	Mode         string   `json:"mode"` // "tables" | "custom"
	Tables       []string `json:"tables,omitempty"`
	Query        string   `json:"query,omitempty"`
	ResourceType string   `json:"resourceType"`
}

// SaveLogConfigRequest represents the request to save log configuration.
type SaveLogConfigRequest struct {
	Service string   `json:"service"`
	Mode    string   `json:"mode"` // "tables" | "custom"
	Tables  []string `json:"tables,omitempty"`
	Query   string   `json:"query,omitempty"`
}

// handleAzureLogConfigRouter routes log config API requests.
// GET /api/azure/logs/config?service=<name> - get config for service
// PUT /api/azure/logs/config - save config for service
func (s *Server) handleAzureLogConfigRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetLogConfig(w, r)
	case http.MethodPut:
		s.handleSaveLogConfig(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetLogConfig returns the log configuration for a service.
// GET /api/azure/logs/config?service=<name>
func (s *Server) handleGetLogConfig(w http.ResponseWriter, r *http.Request) {
	serviceName := r.URL.Query().Get("service")
	if serviceName == "" {
		writeJSONError(w, http.StatusBadRequest, "service parameter required", nil)
		return
	}

	classificationsMu.RLock()
	azureYaml, err := loadAzureYaml(s.projectDir)
	classificationsMu.RUnlock()

	response := LogConfigResponse{
		Service:      serviceName,
		Mode:         "tables",
		ResourceType: "containerapp", // Default
	}

	if err != nil {
		// Return default config
		response.Tables = azure.GetRecommendedTables(azure.ResourceTypeContainerApp)
		if err := writeJSON(w, response); err != nil {
			log.Printf("Failed to write log config response: %v", err)
		}
		return
	}

	// Get resource type from service config
	if svc, ok := azureYaml.Services[serviceName]; ok {
		if svc.Host != "" {
			response.ResourceType = svc.Host
		}
	}

	// Check service-level analytics config first
	if svc, ok := azureYaml.Services[serviceName]; ok && svc.Logs != nil && svc.Logs.Analytics != nil {
		svcConfig := svc.Logs.Analytics
		if len(svcConfig.Tables) > 0 {
			response.Tables = svcConfig.Tables
			response.Mode = "tables"
		}
		if svcConfig.Query != "" {
			response.Query = svcConfig.Query
			response.Mode = "custom"
		}
	}

	// Project-level analytics only has workspace/polling/timespan; no Tables or Query fields
	// Services must specify their own tables or query

	// If still no tables and mode is tables, use recommended
	if response.Mode == "tables" && len(response.Tables) == 0 {
		resourceType := azure.ResourceType(response.ResourceType)
		response.Tables = azure.GetRecommendedTables(resourceType)
	}

	if err := writeJSON(w, response); err != nil {
		log.Printf("Failed to write log config response: %v", err)
	}
}

// handleSaveLogConfig saves log configuration for a service to azure.yaml.
// PUT /api/azure/logs/config
func (s *Server) handleSaveLogConfig(w http.ResponseWriter, r *http.Request) {
	body, err := readLimitedBody(r, maxRequestBodySize)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Failed to read request body", err)
		return
	}

	var req SaveLogConfigRequest
	if err := decodeJSON(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	if req.Service == "" {
		writeJSONError(w, http.StatusBadRequest, "service is required", nil)
		return
	}
	if req.Mode == "" {
		writeJSONError(w, http.StatusBadRequest, "mode is required", nil)
		return
	}
	if req.Mode != "tables" && req.Mode != "custom" {
		writeJSONError(w, http.StatusBadRequest, "mode must be 'tables' or 'custom'", nil)
		return
	}
	if req.Mode == "tables" && len(req.Tables) == 0 {
		writeJSONError(w, http.StatusBadRequest, "tables required when mode is 'tables'", nil)
		return
	}
	if req.Mode == "custom" && req.Query == "" {
		writeJSONError(w, http.StatusBadRequest, "query required when mode is 'custom'", nil)
		return
	}

	classificationsMu.Lock()
	defer classificationsMu.Unlock()

	azureYaml, err := loadAzureYaml(s.projectDir)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load azure.yaml", err)
		return
	}

	// Check if service exists
	svc, exists := azureYaml.Services[req.Service]
	if !exists {
		writeJSONError(w, http.StatusNotFound, "Service not found", nil)
		return
	}

	// Initialize logs config if needed
	if svc.Logs == nil {
		svc.Logs = &service.ServiceLogsConfig{}
	}
	if svc.Logs.Analytics == nil {
		svc.Logs.Analytics = &service.AnalyticsConfigService{}
	}

	// Update the config based on mode
	if req.Mode == "tables" {
		svc.Logs.Analytics.Tables = req.Tables
		svc.Logs.Analytics.Query = "" // Clear custom query
	} else {
		svc.Logs.Analytics.Query = req.Query
		svc.Logs.Analytics.Tables = nil // Clear table selection
	}

	// Save back to the services map
	azureYaml.Services[req.Service] = svc

	// Save azure.yaml
	if err := saveAzureYaml(s.projectDir, azureYaml); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to save azure.yaml", err)
		return
	}

	log.Printf("Saved log config for service %s (mode=%s)", req.Service, req.Mode)

	// Return the saved config
	response := LogConfigResponse{
		Service: req.Service,
		Mode:    req.Mode,
		Tables:  req.Tables,
		Query:   req.Query,
	}

	if err := writeJSON(w, response); err != nil {
		log.Printf("Failed to write save log config response: %v", err)
	}
}
