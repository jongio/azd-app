package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

// ValidationRequest represents the request body for POST /api/editor/validate
type ValidationRequest struct {
	Content string `json:"content"` // YAML content to validate
}

// ValidationIssue represents a single validation issue
type ValidationIssue struct {
	Path    string `json:"path"`
	Message string `json:"message"`
	Level   string `json:"level"` // "error", "warning", "info"
}

// ValidationResponse represents the response for POST /api/editor/validate
type ValidationResponse struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationIssue `json:"errors"`
	Warnings []ValidationIssue `json:"warnings"`
	Info     []ValidationIssue `json:"info"`
}

// handleValidateConfig validates azure.yaml configuration
func (s *Server) handleValidateConfig(w http.ResponseWriter, r *http.Request) {
	var req ValidationRequest
	if !ReadJSONBody(w, r, &req, maxRequestBodySize) {
		return
	}

	// Load schema if not already loaded
	if err := loadSchema(); err != nil {
		InternalError(w, "Failed to load schema", err)
		return
	}

	// Parse YAML to verify it's valid YAML first
	var yamlData interface{}
	if err := yaml.Unmarshal([]byte(req.Content), &yamlData); err != nil {
		response := ValidationResponse{
			Valid: false,
			Errors: []ValidationIssue{
				{
					Path:    "",
					Message: fmt.Sprintf("Invalid YAML syntax: %v", err),
					Level:   "error",
				},
			},
			Warnings: []ValidationIssue{},
			Info:     []ValidationIssue{},
		}
		WriteJSONSuccess(w, response)
		return
	}

	// Convert YAML to JSON for schema validation
	jsonData, err := json.Marshal(yamlData)
	if err != nil {
		InternalError(w, "Failed to convert YAML to JSON", err)
		return
	}

	// Initialize response
	response := ValidationResponse{
		Valid:    true,
		Errors:   []ValidationIssue{},
		Warnings: []ValidationIssue{},
		Info:     []ValidationIssue{},
	}

	// Validate against JSON Schema
	schemaLoader := gojsonschema.NewBytesLoader(azureYamlSchema)
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		InternalError(w, "Schema validation failed", err)
		return
	}

	// Process schema validation errors
	if !result.Valid() {
		response.Valid = false
		for _, err := range result.Errors() {
			response.Errors = append(response.Errors, ValidationIssue{
				Path:    err.Field(),
				Message: formatSchemaError(err),
				Level:   "error",
			})
		}
	}

	// Parse to azure.yaml structure for business rule validation
	var azureYaml service.AzureYaml
	if err := yaml.Unmarshal([]byte(req.Content), &azureYaml); err == nil {
		// Run custom business rule validation
		businessRuleIssues := validateBusinessRules(&azureYaml)
		response.Errors = append(response.Errors, businessRuleIssues.Errors...)
		response.Warnings = append(response.Warnings, businessRuleIssues.Warnings...)
		response.Info = append(response.Info, businessRuleIssues.Info...)

		if len(businessRuleIssues.Errors) > 0 {
			response.Valid = false
		}
	}

	WriteJSONSuccess(w, response)
}

// businessRuleValidation holds validation issues from business rules
type businessRuleValidation struct {
	Errors   []ValidationIssue
	Warnings []ValidationIssue
	Info     []ValidationIssue
}

// validateBusinessRules performs custom validation beyond schema validation
func validateBusinessRules(config *service.AzureYaml) businessRuleValidation {
	result := businessRuleValidation{
		Errors:   []ValidationIssue{},
		Warnings: []ValidationIssue{},
		Info:     []ValidationIssue{},
	}

	if strings.TrimSpace(config.Name) == "" {
		result.Errors = append(result.Errors, ValidationIssue{
			Path:    "name",
			Message: "Application name is required",
			Level:   "error",
		})
	}

	// Check for duplicate service names (already enforced by map but check anyway)
	serviceNames := make(map[string]bool)
	for name := range config.Services {
		if serviceNames[name] {
			result.Errors = append(result.Errors, ValidationIssue{
				Path:    fmt.Sprintf("services.%s", name),
				Message: fmt.Sprintf("Duplicate service name: %s", name),
				Level:   "error",
			})
		}
		serviceNames[name] = true
	}

	// Check for duplicate resource names
	resourceNames := make(map[string]bool)
	for name := range config.Resources {
		if resourceNames[name] {
			result.Errors = append(result.Errors, ValidationIssue{
				Path:    fmt.Sprintf("resources.%s", name),
				Message: fmt.Sprintf("Duplicate resource name: %s", name),
				Level:   "error",
			})
		}
		resourceNames[name] = true
	}

	// Check for port conflicts
	portMap := make(map[string][]string) // port -> list of service names
	for serviceName, svc := range config.Services {
		for _, portMapping := range svc.Ports {
			// Parse port mapping (could be "8080" or "3000:8080")
			parts := strings.Split(portMapping, ":")
			hostPort := parts[0]
			if len(parts) > 1 {
				hostPort = parts[0]
			}

			portMap[hostPort] = append(portMap[hostPort], serviceName)
		}
	}

	for port, services := range portMap {
		if len(services) > 1 {
			result.Warnings = append(result.Warnings, ValidationIssue{
				Path:    "services",
				Message: fmt.Sprintf("Port %s is used by multiple services: %s. This may cause conflicts.", port, strings.Join(services, ", ")),
				Level:   "warning",
			})
		}
	}

	// Check for circular dependencies
	cycles := detectCircularDependencies(config)
	for _, cycle := range cycles {
		result.Errors = append(result.Errors, ValidationIssue{
			Path:    "services",
			Message: fmt.Sprintf("Circular dependency detected: %s", strings.Join(cycle, " → ")),
			Level:   "error",
		})
	}

	// Check for missing health checks (warnings)
	for serviceName, svc := range config.Services {
		// Skip health check warnings for services that explicitly disable it
		if svc.IsHealthcheckDisabled() {
			continue
		}

		// Warn if service has ports but no health check configured
		if len(svc.Ports) > 0 && svc.Healthcheck == nil {
			result.Info = append(result.Info, ValidationIssue{
				Path:    fmt.Sprintf("services.%s", serviceName),
				Message: fmt.Sprintf("Consider adding a health check for service '%s' to monitor availability", serviceName),
				Level:   "info",
			})
		}
	}

	// Check if no resources are defined (info)
	if len(config.Resources) == 0 {
		result.Info = append(result.Info, ValidationIssue{
			Path:    "resources",
			Message: "No resources defined. Consider adding resource definitions for Azure provisioning.",
			Level:   "info",
		})
	}

	return result
}

// detectCircularDependencies finds circular dependency chains
func detectCircularDependencies(config *service.AzureYaml) [][]string {
	var cycles [][]string
	const maxCycles = 100 // Prevent DoS with unbounded slice growth

	// Build dependency graph
	graph := make(map[string][]string)
	allNodes := make(map[string]bool)

	// Add service dependencies
	for serviceName, svc := range config.Services {
		allNodes[serviceName] = true
		if len(svc.Uses) > 0 {
			graph[serviceName] = svc.Uses
		}
	}

	// Add resource dependencies
	for resourceName, res := range config.Resources {
		allNodes[resourceName] = true
		if len(res.Uses) > 0 {
			graph[resourceName] = res.Uses
		}
	}

	// Detect cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(node string, path []string) bool
	dfs = func(node string, path []string) bool {
		visited[node] = true
		recStack[node] = true

		// Create new slice to avoid shared mutation across recursion
		newPath := make([]string, len(path), len(path)+1)
		copy(newPath, path)
		newPath = append(newPath, node)

		for _, neighbor := range graph[node] {
			if !visited[neighbor] {
				if dfs(neighbor, newPath) {
					return true
				}
			} else if recStack[neighbor] {
				// Check if we've hit max cycles limit
				if len(cycles) >= maxCycles {
					return true
				}

				// Found a cycle
				cycleStart := -1
				for i, n := range newPath {
					if n == neighbor {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					// Create new cycle slice with proper size
					cycle := make([]string, len(newPath)-cycleStart+1)
					copy(cycle, newPath[cycleStart:])
					cycle[len(cycle)-1] = neighbor
					cycles = append(cycles, cycle)
				}
				return true
			}
		}

		recStack[node] = false
		return false
	}

	for node := range allNodes {
		if !visited[node] {
			dfs(node, []string{})
		}
	}

	return cycles
}

// formatSchemaError converts a gojsonschema error to a human-readable message
func formatSchemaError(err gojsonschema.ResultError) string {
	context := err.Context().String()
	description := err.Description()

	// Make error messages more user-friendly
	switch err.Type() {
	case "required":
		return fmt.Sprintf("Required field '%s' is missing", err.Field())
	case "invalid_type":
		return fmt.Sprintf("Field '%s' has invalid type: %s", err.Field(), description)
	case "pattern":
		return fmt.Sprintf("Field '%s' does not match required pattern: %s", err.Field(), description)
	case "enum":
		return fmt.Sprintf("Field '%s' must be one of the allowed values: %s", err.Field(), description)
	case "minimum":
		return fmt.Sprintf("Field '%s' value is too small: %s", err.Field(), description)
	case "maximum":
		return fmt.Sprintf("Field '%s' value is too large: %s", err.Field(), description)
	default:
		if context != "" {
			return fmt.Sprintf("%s: %s", context, description)
		}
		return description
	}
}

// registerValidationRoutes adds validation API routes to the server mux
func (s *Server) registerValidationRoutes() {
	if s.endpointLimiter == nil {
		s.endpointLimiter = NewEndpointRateLimits()
	}
	// Wrap validation handler with rate limiting
	s.mux.HandleFunc("/api/editor/validate", RateLimitMiddleware(s.endpointLimiter,
		MethodGuard(s.handleValidateConfig, http.MethodPost)))
}
