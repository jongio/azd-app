package dashboard

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/wellknown"
)

// WellKnownServiceResponse represents a service definition in the API response
type WellKnownServiceResponse struct {
	Name              string            `json:"name"`
	DisplayName       string            `json:"displayName"`
	Description       string            `json:"description"`
	Category          string            `json:"category"`
	Icon              string            `json:"icon,omitempty"`
	Host              string            `json:"host"`
	Image             string            `json:"image"`
	Ports             []string          `json:"ports,omitempty"`
	Environment       map[string]string `json:"environment,omitempty"`
	Healthcheck       interface{}       `json:"healthcheck,omitempty"`
	ConnectionStrings map[string]string `json:"connectionStrings,omitempty"`
	DocsURL           string            `json:"docsUrl,omitempty"`
}

// WellKnownServicesListResponse represents the response for GET /api/editor/wellknown
type WellKnownServicesListResponse struct {
	Services []WellKnownServiceResponse `json:"services"`
}

// serviceIcons maps service names to emoji icons
var serviceIcons = map[string]string{
	"azurite":  "📦",
	"cosmos":   "🌍",
	"redis":    "🔴",
	"postgres": "🐘",
	"mongodb":  "🍃",
	"mysql":    "🐬",
}

// serviceDocsURLs maps service names to documentation URLs
var serviceDocsURLs = map[string]string{
	"azurite":  "https://learn.microsoft.com/azure/storage/common/storage-use-azurite",
	"cosmos":   "https://learn.microsoft.com/azure/cosmos-db/docker-emulator-linux",
	"redis":    "https://redis.io/docs/",
	"postgres": "https://www.postgresql.org/docs/",
	"mongodb":  "https://www.mongodb.com/docs/",
	"mysql":    "https://dev.mysql.com/doc/",
}

// handleGetWellKnownServices returns all well-known service definitions
func (s *Server) handleGetWellKnownServices(w http.ResponseWriter, r *http.Request) {
	services := make([]WellKnownServiceResponse, 0, len(wellknown.Registry))

	for _, def := range wellknown.Registry {
		services = append(services, toWellKnownServiceResponse(def))
	}

	response := WellKnownServicesListResponse{
		Services: services,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		InternalError(w, "Failed to encode response", err)
		return
	}
}

// handleGetWellKnownService returns a specific well-known service by name
func (s *Server) handleGetWellKnownService(w http.ResponseWriter, r *http.Request) {
	// Extract service name from path: /api/editor/wellknown/{name}
	name := strings.TrimPrefix(r.URL.Path, "/api/editor/wellknown/")
	if name == "" {
		BadRequest(w, "Service name is required", nil)
		return
	}

	def := wellknown.Get(name)
	if def == nil {
		NotFound(w, "Service not found")
		return
	}

	response := toWellKnownServiceResponse(*def)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		InternalError(w, "Failed to encode response", err)
		return
	}
}

// handleWellKnownRouter routes well-known service requests
func (s *Server) handleWellKnownRouter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path

	// Handle /api/editor/wellknown (list all)
	if path == "/api/editor/wellknown" {
		s.handleGetWellKnownServices(w, r)
		return
	}

	// Handle /api/editor/wellknown/{name} (get specific service)
	if strings.HasPrefix(path, "/api/editor/wellknown/") {
		s.handleGetWellKnownService(w, r)
		return
	}

	http.NotFound(w, r)
}

// toWellKnownServiceResponse converts a ServiceDefinition to API response format
func toWellKnownServiceResponse(def wellknown.ServiceDefinition) WellKnownServiceResponse {
	response := WellKnownServiceResponse{
		Name:              def.Name,
		DisplayName:       def.DisplayName,
		Description:       def.Description,
		Category:          def.Category,
		Host:              def.Host,
		Image:             def.Image,
		Ports:             def.Ports,
		Environment:       def.Environment,
		ConnectionStrings: def.ConnectionStrings,
	}

	// Add icon if available
	if icon, ok := serviceIcons[def.Name]; ok {
		response.Icon = icon
	}

	// Add docs URL if available
	if docsURL, ok := serviceDocsURLs[def.Name]; ok {
		response.DocsURL = docsURL
	}

	// Add healthcheck if present
	if def.Healthcheck != nil {
		response.Healthcheck = def.Healthcheck
	}

	return response
}
