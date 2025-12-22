// Package dashboard provides API endpoints for the local dashboard.
package dashboard

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// HealthCheckResponse represents the overall health check result.
type HealthCheckResponse struct {
	Status    string        `json:"status"`    // "healthy" | "degraded" | "error"
	Checks    []HealthCheck `json:"checks"`    // Individual health checks
	DocsURL   string        `json:"docsUrl"`   // Documentation URL
	Timestamp time.Time     `json:"timestamp"` // When check was performed
}

// HealthCheck represents an individual health check result.
type HealthCheck struct {
	Name    string `json:"name"`          // Check name
	Status  string `json:"status"`        // "pass" | "warn" | "fail"
	Message string `json:"message"`       // Result message
	Fix     string `json:"fix,omitempty"` // Fix instructions for failures
}

// handleAzureLogsHealth performs diagnostic checks for Azure logs troubleshooting.
// GET /api/azure/logs/health
func (s *Server) handleAzureLogsHealth(w http.ResponseWriter, r *http.Request) {
	response := HealthCheckResponse{
		Checks:    make([]HealthCheck, 0, 4),
		DocsURL:   logsTroubleshootURL,
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

	WriteJSONSuccess(w, response)
}

// checkAuthentication verifies Azure credentials are available.
func (s *Server) checkAuthentication() HealthCheck {
	check := HealthCheck{
		Name: "Authentication",
	}

	// Try to create credentials
	cred, err := newLogAnalyticsCredential()
	if err != nil {
		check.Status = "fail"
		check.Message = "Azure credentials not available"
		check.Fix = "azd auth login"
		return check
	}

	// Try to get a token to verify credentials work
	ctx, cancel := timeoutContext(5 * time.Second)
	defer cancel()

	err = validateCredentials(ctx, cred)
	if err != nil {
		check.Status = "fail"
		check.Message = "Azure credentials invalid or expired"
		check.Fix = "azd auth login"
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

	workspaceID := getWorkspaceIDFromEnv(s.projectDir)
	if workspaceID == "" {
		check.Status = "fail"
		check.Message = "Log Analytics workspace not configured"
		check.Fix = "azd env refresh"
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
		check.Fix = "azd up"
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

	workspaceID := getWorkspaceIDFromEnv(s.projectDir)
	cred, err := newLogAnalyticsCredential()
	if err != nil {
		check.Status = "warn"
		check.Message = "Cannot create credentials for connectivity test"
		return check
	}

	// Try to create client (this doesn't make actual queries)
	ctx := context.Background()
	_, err = getOrCreateLogAnalyticsClient(ctx, cred, workspaceID)
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
