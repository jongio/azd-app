// Package dashboard provides API endpoints for the local dashboard.
package dashboard

import (
	"log"
	"net/http"

	"github.com/jongio/azd-app/cli/src/internal/azure"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// AzureQueryResponse represents the KQL query used for a service.
type AzureQueryResponse struct {
	Service      string `json:"service"`
	ResourceType string `json:"resourceType"`
	Query        string `json:"query"`
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
		http.Error(w, errMethodNotAllowed, http.StatusMethodNotAllowed)
	}
}

// handleAzureQuery returns the KQL query being used for a service.
// GET /api/azure/query?service=<name>
func (s *Server) handleAzureQuery(w http.ResponseWriter, r *http.Request) {
	serviceName, ok := RequireQueryParam(w, r, "service")
	if !ok {
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

	WriteJSONSuccess(w, response)
}

// SaveQueryRequest represents the request body for saving a custom query.
type SaveQueryRequest struct {
	Service string `json:"service"`
	Query   string `json:"query"`
}

// handleSaveAzureQuery saves a custom KQL query for a service to azure.yaml.
// PUT /api/azure/query
func (s *Server) handleSaveAzureQuery(w http.ResponseWriter, r *http.Request) {
	var req SaveQueryRequest
	if !ReadJSONBody(w, r, &req, maxRequestBodySize) {
		return
	}

	if req.Service == "" {
		BadRequest(w, errServiceRequired, nil)
		return
	}
	if req.Query == "" {
		BadRequest(w, errQueryRequired, nil)
		return
	}

	classificationsMu.Lock()
	defer classificationsMu.Unlock()

	// Load existing azure.yaml
	azureYaml, err := loadAzureYaml(s.projectDir)
	if err != nil {
		HandleLoadError(w, err)
		return
	}

	// Check if service exists
	svc, exists := azureYaml.Services[req.Service]
	if !exists {
		NotFound(w, "Service not found")
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
		HandleSaveError(w, err)
		return
	}

	log.Printf("Saved custom KQL query for service %s", req.Service)

	// Return updated query info
	response := AzureQueryResponse{
		Service:      req.Service,
		ResourceType: "custom",
		Query:        req.Query,
	}

	WriteJSONSuccess(w, response)
}
