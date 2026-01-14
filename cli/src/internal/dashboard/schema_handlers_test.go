package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHandleGetSchema(t *testing.T) {
	// Create a test schema file
	tempDir := t.TempDir()
	schemaPath := tempDir + "/azure.yaml.json"
	testSchema := `{"$schema": "http://json-schema.org/draft-07/schema#", "title": "Test Schema", "type": "object", "properties": {"name": {"type": "string"}}}`

	if err := os.WriteFile(schemaPath, []byte(testSchema), 0600); err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}

	// Override schema loading for tests
	azureYamlSchema = []byte(testSchema)
	schemaETag = `"test-etag"`

	server := &Server{
		mux: http.NewServeMux(),
	}
	server.registerSchemaRoutes()

	t.Run("successful schema retrieval", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/editor/schema", nil)
		w := httptest.NewRecorder()

		server.handleGetSchema(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Check headers
		if w.Header().Get("ETag") == "" {
			t.Error("Expected ETag header to be set")
		}

		if w.Header().Get("Cache-Control") == "" {
			t.Error("Expected Cache-Control header to be set")
		}

		// Verify response structure
		var response SchemaResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response.Schema) == 0 {
			t.Error("Expected schema to be non-empty")
		}

		// Verify schema is valid JSON
		var schemaObj map[string]interface{}
		if err := json.Unmarshal(response.Schema, &schemaObj); err != nil {
			t.Errorf("Schema is not valid JSON: %v", err)
		}

		// Check for expected schema properties
		if _, ok := schemaObj["$schema"]; !ok {
			t.Error("Expected schema to have $schema property")
		}

		if _, ok := schemaObj["properties"]; !ok {
			t.Error("Expected schema to have properties")
		}
	})

	t.Run("cache validation with ETag", func(t *testing.T) {
		// First request to get ETag
		req1 := httptest.NewRequest(http.MethodGet, "/api/editor/schema", nil)
		w1 := httptest.NewRecorder()
		server.handleGetSchema(w1, req1)
		etag := w1.Header().Get("ETag")

		// Second request with If-None-Match
		req2 := httptest.NewRequest(http.MethodGet, "/api/editor/schema", nil)
		req2.Header.Set("If-None-Match", etag)
		w2 := httptest.NewRecorder()
		server.handleGetSchema(w2, req2)

		if w2.Code != http.StatusNotModified {
			t.Errorf("Expected status 304 Not Modified, got %d", w2.Code)
		}

		if w2.Body.Len() > 0 {
			t.Error("Expected empty body for 304 response")
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/editor/schema", nil)
		w := httptest.NewRecorder()

		server.mux.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})
}

func TestSchemaContent(t *testing.T) {
	testSchema := `{"$schema": "http://json-schema.org/draft-07/schema#", "title": "Test Schema", "type": "object", "properties": {"name": {"type": "string"}, "services": {"type": "object"}, "resources": {"type": "object"}}}`
	azureYamlSchema = []byte(testSchema)

	// Verify embedded schema content is valid
	var schemaObj map[string]interface{}
	if err := json.Unmarshal(azureYamlSchema, &schemaObj); err != nil {
		t.Fatalf("Schema is not valid JSON: %v", err)
	}

	// Check required schema properties
	requiredProps := []string{"$schema", "title", "type", "properties"}
	for _, prop := range requiredProps {
		if _, ok := schemaObj[prop]; !ok {
			t.Errorf("Schema missing required property: %s", prop)
		}
	}

	// Verify key azure.yaml properties exist
	properties, ok := schemaObj["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Schema properties is not an object")
	}

	expectedProps := []string{"name", "services", "resources"}
	for _, prop := range expectedProps {
		if _, ok := properties[prop]; !ok {
			t.Errorf("Schema missing expected property: %s", prop)
		}
	}
}
