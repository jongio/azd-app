// Package dashboard provides API endpoints for the local dashboard.
package dashboard

import (
	"log"
	"net/http"
	"path/filepath"

	"github.com/jongio/azd-app/cli/src/internal/azure"
	"github.com/jongio/azd-core/yamlutil"
)

const resourceTables = "tables"

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

	resourceTypeStr := r.URL.Query().Get("resourceType")
	resourceType := azure.ResourceType(resourceTypeStr)
	if resourceType == "" {
		resourceType = azure.ResourceTypeContainerApp // Default
	}

	ctx := r.Context()
	workspaceID, err := getWorkspaceIDFromEnv(ctx)

	var tables []azure.TableInfo

	// Try to get live tables from Log Analytics
	if err == nil && workspaceID != "" {
		cred, credErr := newLogAnalyticsCredential()
		if credErr == nil {
			client, clientErr := getOrCreateLogAnalyticsClient(ctx, cred, workspaceID)
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

	WriteJSONSuccess(w, response)
}

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
		http.Error(w, errMethodNotAllowed, http.StatusMethodNotAllowed)
	}
}

// handleGetLogConfig returns the log configuration for a service.
// GET /api/azure/logs/config?service=<name>
func (s *Server) handleGetLogConfig(w http.ResponseWriter, r *http.Request) {
	serviceName, ok := RequireQueryParam(w, r, "service")
	if !ok {
		return
	}

	classificationsMu.RLock()
	azureYaml, err := loadAzureYaml(s.projectDir)
	classificationsMu.RUnlock()

	response := LogConfigResponse{
		Service:      serviceName,
		Mode:         resourceTables,
		ResourceType: "containerapp", // Default
	}

	if err != nil {
		// Return default config
		response.Tables = azure.GetRecommendedTables(azure.ResourceTypeContainerApp)
		WriteJSONSuccess(w, response)
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
			response.Mode = resourceTables
		}
		if svcConfig.Query != "" {
			response.Query = svcConfig.Query
			response.Mode = queryTypeCustom
		}
	}

	// If still no tables and mode is tables, use recommended
	if response.Mode == resourceTables && len(response.Tables) == 0 {
		resourceType := azure.ResourceType(response.ResourceType)
		response.Tables = azure.GetRecommendedTables(resourceType)
	}

	WriteJSONSuccess(w, response)
}

// handleSaveLogConfig saves log configuration for a service to azure.yaml.
// PUT /api/azure/logs/config
func (s *Server) handleSaveLogConfig(w http.ResponseWriter, r *http.Request) {
	var req SaveLogConfigRequest
	if !ReadJSONBody(w, r, &req, maxRequestBodySize) {
		return
	}

	if req.Service == "" {
		BadRequest(w, errServiceRequired, nil)
		return
	}
	if req.Mode == "" {
		BadRequest(w, errModeRequired, nil)
		return
	}
	if req.Mode != resourceTables && req.Mode != queryTypeCustom {
		BadRequest(w, "mode must be 'tables' or 'custom'", nil)
		return
	}
	if req.Mode == resourceTables && len(req.Tables) == 0 {
		BadRequest(w, "tables required when mode is 'tables'", nil)
		return
	}
	if req.Mode == queryTypeCustom && req.Query == "" {
		BadRequest(w, "query required when mode is 'custom'", nil)
		return
	}

	classificationsMu.Lock()
	defer classificationsMu.Unlock()

	azureYaml, err := loadAzureYaml(s.projectDir)
	if err != nil {
		HandleLoadError(w, err)
		return
	}

	// Check if service exists
	if _, exists := azureYaml.Services[req.Service]; !exists {
		NotFound(w, "Service not found")
		return
	}

	// Prepare tables and query based on mode
	var tables []string
	var query string
	if req.Mode == resourceTables {
		tables = req.Tables
		query = ""
	} else {
		tables = nil
		query = req.Query
	}

	// Use non-destructive YAML update to preserve schema, comments, and structure
	azureYamlPath := filepath.Join(s.projectDir, "azure.yaml")
	if err := yamlutil.UpdateServiceLogsConfig(azureYamlPath, req.Service, tables, query); err != nil {
		HandleSaveError(w, err)
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

	WriteJSONSuccess(w, response)
}
