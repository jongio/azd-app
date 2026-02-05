package dashboard

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/registry"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
)

const (
	contentTypeHeader = "Content-Type"
	jsonContentType   = "application/json"
)

// handlePing is a simple health check endpoint to verify the dashboard is running.
func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(contentTypeHeader, jsonContentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// handleGetEnvironment returns environment information for Codespace detection.
func (s *Server) handleGetEnvironment(w http.ResponseWriter, r *http.Request) {

	// Detect GitHub Codespace environment
	codespaceName := os.Getenv("CODESPACE_NAME")
	codespacePortDomain := os.Getenv("GITHUB_CODESPACES_PORT_FORWARDING_DOMAIN")

	// Default domain if not set but in Codespace
	if codespaceName != "" && codespacePortDomain == "" {
		codespacePortDomain = "app.github.dev"
	}

	// Detect if running in VS Code (desktop) vs browser-based Codespace
	// In VS Code desktop (including VS Code connected to Codespace), localhost URLs work natively
	// Only in browser-based Codespace do we need to transform localhost URLs
	isVsCodeDesktop := runningOnVsCodeDesktop()

	// Get Azure environment name if available
	azureEnvName := os.Getenv("AZURE_ENV_NAME")

	response := map[string]interface{}{
		"codespace": map[string]interface{}{
			"enabled":         codespaceName != "",
			"name":            codespaceName,
			"domain":          codespacePortDomain,
			"isVsCodeDesktop": isVsCodeDesktop,
		},
		"environmentName": azureEnvName,
	}

	WriteJSONSuccess(w, response)
}

// runningOnVsCodeDesktop detects if VS Code desktop is available.
// When VS Code desktop is available (including when connected to Codespace),
// localhost URLs work natively without transformation.
// In browser-based Codespace, 'code --status' returns:
// "The --status argument is not yet supported in browsers."
// Reference: azure/azure-dev cli/azd/cmd/auth_login.go runningOnCodespacesBrowser
func runningOnVsCodeDesktop() bool {
	// Check if running in Codespace first - if not, no need to check
	if os.Getenv("CODESPACES") != "true" {
		return false
	}

	// Try to run 'code --status' to detect VS Code desktop vs browser
	// This command returns specific output in browser-based VS Code
	cmd := exec.Command("code", "--status")
	output, err := cmd.Output()
	if err != nil {
		// If code command fails or doesn't exist, we're likely in browser Codespace
		// or some environment where VS Code CLI isn't available
		return false
	}

	// If output contains the browser-specific message, we're in browser Codespace
	// Otherwise, we're in VS Code desktop connected to Codespace
	return !strings.Contains(string(output), "The --status argument is not yet supported in browsers")
}

// handleGetServices returns services for the current project.
func (s *Server) handleGetServices(w http.ResponseWriter, r *http.Request) {
	// Use shared serviceinfo package to get merged service data
	services, err := serviceinfo.GetServiceInfo(s.projectDir)
	if err != nil {
		log.Printf("Warning: Failed to get service info: %v", err)
		InternalError(w, "Failed to get service info", err)
		return
	}

	WriteJSONSuccess(w, services)
}

// handleGetProject returns project metadata from azure.yaml.
func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	azureYaml, err := service.ParseAzureYaml(s.projectDir)
	if err != nil {
		InternalError(w, "Failed to parse azure.yaml", err)
		return
	}

	response := map[string]string{
		"name": azureYaml.Name,
		"dir":  s.projectDir,
	}

	WriteJSONSuccess(w, response)
}

// handleGetLogs returns recent logs for services.
func (s *Server) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	// Add panic recovery with detailed logging
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in handleGetLogs: %v\nStack: %s", r, debug.Stack())
			InternalError(w, "Internal server error", fmt.Errorf("panic: %v", r))
		}
	}()

	serviceName := r.URL.Query().Get("service")
	tailStr := r.URL.Query().Get("tail")

	// Default to 500 lines with bounds checking
	tail := 500
	if tailStr != "" {
		if n, err := fmt.Sscanf(tailStr, "%d", &tail); err != nil || n != 1 {
			tail = 500
		}
	}
	// Enforce reasonable limits to prevent memory exhaustion
	if tail <= 0 {
		tail = 500
	} else if tail > 10000 {
		tail = 10000 // Maximum 10k lines
	}

	logManager := service.GetLogManager(s.projectDir)
	if logManager == nil {
		InternalError(w, "Log manager not initialized", nil)
		return
	}

	var logs []service.LogEntry
	if serviceName != "" {
		// Get logs from specific service
		buffer, exists := logManager.GetBuffer(serviceName)
		if !exists {
			NotFound(w, fmt.Sprintf("Service '%s' not found", serviceName))
			return
		}
		if buffer == nil {
			InternalError(w, "Log buffer is nil", nil)
			return
		}
		logs = buffer.GetRecent(tail)
	} else {
		// Get logs from all services
		logs = logManager.GetAllLogs(tail)
	}

	// Enable gzip compression for large responses
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set(contentTypeHeader, jsonContentType)
		w.WriteHeader(http.StatusOK)

		gz := gzip.NewWriter(w)
		defer func() { _ = gz.Close() }()

		if err := json.NewEncoder(gz).Encode(logs); err != nil {
			log.Printf("Failed to write gzipped JSON response: %v", err)
		}
	} else {
		WriteJSONSuccess(w, logs)
	}
}

// handleStartService handles POST /api/services/start to start a service or all services.
func (s *Server) handleStartService(w http.ResponseWriter, r *http.Request) {
	newServiceOperationHandler(s, opStart).Handle(w, r)
}

// handleStopService handles POST /api/services/stop to stop a service or all services.
func (s *Server) handleStopService(w http.ResponseWriter, r *http.Request) {
	newServiceOperationHandler(s, opStop).Handle(w, r)
}

// handleRestartService handles POST /api/services/restart to restart a service or all services.
func (s *Server) handleRestartService(w http.ResponseWriter, r *http.Request) {
	newServiceOperationHandler(s, opRestart).Handle(w, r)
}

// handleFallback provides a simple HTML page when static files aren't available.
func (s *Server) handleFallback(w http.ResponseWriter, r *http.Request) {
	reg := registry.GetRegistry(s.projectDir)
	services := reg.ListAll()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>AZD App Dashboard</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; max-width: 1200px; margin: 40px auto; padding: 20px; }
        h1 { color: #0078d4; }
        .service { background: #f5f5f5; padding: 15px; margin: 10px 0; border-radius: 8px; }
        .status { display: inline-block; width: 12px; height: 12px; border-radius: 50%%; margin-right: 8px; }
        .ready { background: #107c10; }
        .starting { background: #ffb900; }
        .error { background: #d13438; }
        a { color: #0078d4; text-decoration: none; }
        a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <h1>🚀 AZD App Dashboard</h1>
    <p>Running Services in Current Project</p>
`)

	if len(services) == 0 {
		_, _ = fmt.Fprintf(w, `<p>No services are currently running.</p>`)
	} else {
		for _, svc := range services {
			statusClass := "starting"
			switch svc.Status {
			case "ready":
				statusClass = "ready"
			case "error":
				statusClass = "error"
			}

			// Escape all user-controllable values to prevent XSS
			escapedName := html.EscapeString(svc.Name)
			escapedURL := html.EscapeString(svc.URL)
			escapedFramework := html.EscapeString(svc.Framework)
			escapedLanguage := html.EscapeString(svc.Language)
			escapedStatus := html.EscapeString(svc.Status)
			escapedHealth := "-" // Health is computed dynamically via health checks

			_, _ = fmt.Fprintf(w, `
    <div class="service">
        <h3><span class="status %s"></span>%s</h3>
        <p><strong>URL:</strong> <a href="%s" target="_blank">%s</a></p>
        <p><strong>Framework:</strong> %s (%s)</p>
        <p><strong>Status:</strong> %s | <strong>Health:</strong> %s</p>
        <p><strong>Started:</strong> %s</p>
    </div>
`, statusClass, escapedName, escapedURL, escapedURL, escapedFramework, escapedLanguage, escapedStatus, escapedHealth, svc.StartTime.Format(time.RFC822))
		}
	}

	_, _ = fmt.Fprintf(w, `
    <hr>
    <p style="color: #666; font-size: 14px;">
        <a href="/api/services">View JSON</a> | 
        <a href="/api/services/all">All Projects (JSON)</a>
    </p>
</body>
</html>`)
}
