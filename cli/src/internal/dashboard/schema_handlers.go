package dashboard

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"
)

// azureYamlSchema is loaded from file at runtime
var (
	azureYamlSchema     []byte
	azureYamlSchemaOnce sync.Once
	azureYamlSchemaErr  error
)

// SchemaResponse represents the response for GET /api/editor/schema
type SchemaResponse struct {
	Schema json.RawMessage `json:"schema"`
}

var (
	schemaETag string
)

// loadSchema loads the schema file from disk
func loadSchema() error {
	azureYamlSchemaOnce.Do(func() {
		// Try multiple possible schema paths
		possiblePaths := []string{
			"../../schemas/v1.1/azure.yaml.json",           // From cli/src/internal/dashboard
			"../../../schemas/v1.1/azure.yaml.json",        // Alternative path
			"../../../../schemas/v1.1/azure.yaml.json",     // From compiled binary
			"schemas/v1.1/azure.yaml.json",                 // From project root
		}
		
		for _, schemaPath := range possiblePaths {
			data, err := os.ReadFile(schemaPath)
			if err == nil {
				azureYamlSchema = data
				// Calculate ETag from schema content
				schemaETag = `"v1.1-` + time.Now().Format("20060102") + `"`
				return
			}
		}
		
		// If all paths fail, set error
		azureYamlSchemaErr = os.ErrNotExist
	})
	
	return azureYamlSchemaErr
}

// handleGetSchema returns the azure.yaml JSON Schema
func (s *Server) handleGetSchema(w http.ResponseWriter, r *http.Request) {
	// Load schema if not already loaded
	if err := loadSchema(); err != nil {
		InternalError(w, "Failed to load schema", err)
		return
	}
	
	// Set ETag header for cache validation
	w.Header().Set("ETag", schemaETag)
	w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year
	
	// Check if client has cached version
	if match := r.Header.Get("If-None-Match"); match == schemaETag && schemaETag != "" {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// Return schema
	response := SchemaResponse{
		Schema: json.RawMessage(azureYamlSchema),
	}

	WriteJSONSuccess(w, response)
}

// registerSchemaRoutes adds schema API routes to the server mux
func (s *Server) registerSchemaRoutes() {
	s.mux.HandleFunc("/api/editor/schema", MethodGuard(s.handleGetSchema, http.MethodGet))
}
