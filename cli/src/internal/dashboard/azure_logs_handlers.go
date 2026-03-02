// Package dashboard provides API endpoints for the local dashboard.
package dashboard

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/azure"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/security"
)

// AzureServicesResponse represents the list of available services.
type AzureServicesResponse struct {
	Services []string `json:"services"`
}

// handleAzureServices returns the list of services that have Azure resources.
// GET /api/azure/services
func (s *Server) handleAzureServices(w http.ResponseWriter, r *http.Request) {
	// Parse SERVICE_*_NAME environment variables to get service list
	serviceNames := extractServiceNamesFromEnv()

	response := AzureServicesResponse{
		Services: serviceNames,
	}

	WriteJSONSuccess(w, response)
}

// EnableAzureResponse represents the response from enabling Azure logging.
type EnableAzureResponse struct {
	Enabled bool   `json:"enabled"`
	Message string `json:"message"`
}

// handleEnableAzureLogging enables Azure logging by adding the config to azure.yaml.
// POST /api/azure/enable
func (s *Server) handleEnableAzureLogging(w http.ResponseWriter, r *http.Request) {
	classificationsMu.Lock()
	defer classificationsMu.Unlock()

	// Load existing azure.yaml
	azureYaml, err := loadAzureYaml(s.projectDir)
	if err != nil {
		HandleLoadError(w, err)
		return
	}

	// Check if already enabled (presence of analytics config means enabled)
	if azureYaml.Logs != nil && azureYaml.Logs.Analytics != nil {
		response := EnableAzureResponse{
			Enabled: true,
			Message: "Azure logging is already enabled",
		}
		WriteJSONSuccess(w, response)
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
		HandleSaveError(w, err)
		return
	}

	log.Printf("Azure logging enabled in azure.yaml for project: %s", s.projectDir)

	response := EnableAzureResponse{
		Enabled: true,
		Message: "Azure logging enabled! Refresh to start viewing logs.",
	}

	w.WriteHeader(http.StatusOK)
	WriteJSONSuccess(w, response)
}

// AzureLogsResponse represents the structured response for Azure logs.
type AzureLogsResponse struct {
	Status    string             `json:"status"`          // "ok" | "error"
	Logs      []service.LogEntry `json:"logs,omitempty"`  // Log entries
	Count     int                `json:"count"`           // Number of logs returned
	Timestamp time.Time          `json:"timestamp"`       // Response timestamp
	Error     *ErrorInfo         `json:"error,omitempty"` // Error details if status=error
}

// handleAzureLogs returns recent Azure logs with structured error handling.
// GET /api/azure/logs?service=<name>&since=<duration>
func (s *Server) handleAzureLogs(w http.ResponseWriter, r *http.Request) {

	serviceName := r.URL.Query().Get("service")
	sinceStr := r.URL.Query().Get("since")
	tailStr := r.URL.Query().Get("tail")

	// Validate service name to prevent injection attacks
	if err := security.ValidateServiceName(serviceName, true); err != nil {
		BadRequest(w, "Invalid service name", nil)
		return
	}

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

	azureLogs, err := fetchAzureLogsStandalone(ctx, config)

	response := AzureLogsResponse{
		Timestamp: time.Now(),
	}

	if err != nil {
		response.Status = "error"
		response.Count = 0
		response.Error = mapAzureErrorToInfo(err)

		// Set appropriate HTTP status code
		statusCode := http.StatusInternalServerError
		switch response.Error.Code {
		case "AUTH_EXPIRED", "AUTH_REQUIRED":
			statusCode = http.StatusUnauthorized
		case "NOT_DEPLOYED", "NO_WORKSPACE":
			statusCode = http.StatusServiceUnavailable
		case "NO_PERMISSION":
			statusCode = http.StatusForbidden
		case "NO_RESULTS":
			// NO_RESULTS is not an error - it means the query succeeded but found no logs
			// Return 200 with empty logs array instead of error
			response.Status = "ok"
			response.Logs = []service.LogEntry{}
			response.Count = 0
			response.Error = nil
			WriteJSONSuccess(w, response)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
			log.Printf("Failed to write azure logs error response: %v", encodeErr)
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
			IsStderr:  false,
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

	WriteJSONSuccess(w, response)
}

// handleAzureDiagnosticSettingsCheck checks diagnostic settings for all services.
// GET /api/azure/diagnostic-settings/check
func (s *Server) handleAzureDiagnosticSettingsCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := timeoutContext(30 * time.Second)
	defer cancel()

	// Create credentials
	cred, err := newLogAnalyticsCredential()
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized,
			MsgAzureCredsNotAvailable, err)
		return
	}

	// Create diagnostic settings checker
	checker := azure.NewDiagnosticSettingsChecker(cred, s.projectDir)

	// Check all services
	result, err := checker.CheckAllServices(ctx)
	if err != nil {
		// Check if this is a context timeout
		if ctx.Err() != nil {
			writeJSONError(w, http.StatusGatewayTimeout,
				"Request timed out while checking diagnostic settings", err)
			return
		}

		// Other errors (discovery failed, etc.)
		InternalError(w, "Failed to check diagnostic settings", err)
		return
	}

	// Return the results
	WriteJSONSuccess(w, result)
}

// handleAzureDiagnostics runs comprehensive diagnostics on all services.
// GET /api/azure/diagnostics
func (s *Server) handleAzureDiagnostics(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := timeoutContext(60 * time.Second)
	defer cancel()

	// Create credentials
	cred, err := newLogAnalyticsCredential()
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized,
			MsgAzureCredsNotAvailable, err)
		return
	}

	// Create diagnostics engine
	engine := azure.NewDiagnosticsEngine(cred, s.projectDir)

	// Run diagnostics
	result, err := engine.RunDiagnostics(ctx)
	if err != nil {
		// Check if this is a context timeout
		if ctx.Err() != nil {
			writeJSONError(w, http.StatusGatewayTimeout,
				"Request timed out while running diagnostics", err)
			return
		}

		// Other errors (discovery failed, etc.)
		InternalError(w, "Failed to run diagnostics", err)
		return
	}

	// Return the results
	WriteJSONSuccess(w, result)
}

// handleAzureWorkspaceVerify verifies workspace connection by querying for recent logs.
// POST /api/azure/workspace/verify
func (s *Server) handleAzureWorkspaceVerify(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := timeoutContext(60 * time.Second) // Longer timeout for log queries
	defer cancel()

	// Parse request body
	var req azure.WorkspaceVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest,
			"Invalid request body", err)
		return
	}

	// Create credentials
	cred, err := newLogAnalyticsCredential()
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized,
			MsgAzureCredsNotAvailable, err)
		return
	}

	// Create workspace verifier
	verifier := azure.NewWorkspaceVerifier(cred, s.projectDir)

	// Verify workspace
	result, err := verifier.VerifyWorkspace(ctx, &req)
	if err != nil {
		// Check if this is a context timeout
		if ctx.Err() != nil {
			writeJSONError(w, http.StatusGatewayTimeout,
				"Request timed out while verifying workspace", err)
			return
		}

		// Check for specific error types
		errMsg := err.Error()
		if strings.Contains(errMsg, "no Log Analytics workspace") {
			writeJSONError(w, http.StatusServiceUnavailable,
				"No Log Analytics workspace configured for this environment", err)
			return
		}

		if strings.Contains(errMsg, "invalid timespan") {
			writeJSONError(w, http.StatusBadRequest,
				"Invalid timespan format. Use ISO 8601 duration (e.g., PT15M)", err)
			return
		}

		// Other errors (discovery failed, etc.)
		InternalError(w, "Failed to verify workspace", err)
		return
	}

	// Return the results
	WriteJSONSuccess(w, result)
}

// handleAzureBicepTemplate generates a consolidated Bicep template for all detected services.
// GET /api/azure/bicep-template
func (s *Server) handleAzureBicepTemplate(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := timeoutContext(30 * time.Second)
	defer cancel()

	// Create credentials
	cred, err := newLogAnalyticsCredential()
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized,
			MsgAzureCredsNotAvailable, err)
		return
	}

	// Create resource discovery
	discovery := azure.NewResourceDiscovery(cred, s.projectDir)

	// Create Bicep generator
	generator := azure.NewBicepGenerator(discovery)

	// Generate template
	result, err := generator.GenerateTemplate(ctx)
	if err != nil {
		// Check if this is a context timeout
		if ctx.Err() != nil {
			writeJSONError(w, http.StatusGatewayTimeout,
				"Request timed out while generating Bicep template", err)
			return
		}

		// Check for specific error types
		errMsg := err.Error()
		if strings.Contains(errMsg, "no Azure resources found") {
			writeJSONError(w, http.StatusNotFound,
				"No Azure resources found. Deploy your application with 'azd up' first.", err)
			return
		}

		if strings.Contains(errMsg, "failed to discover resources") {
			writeJSONError(w, http.StatusServiceUnavailable,
				"Unable to discover Azure resources. Ensure your environment is deployed.", err)
			return
		}

		// Other errors
		InternalError(w, "Failed to generate Bicep template", err)
		return
	}

	// Return the generated template
	WriteJSONSuccess(w, result)
}
