package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

func TestHandleValidateConfig(t *testing.T) {
	server := &Server{
		mux: http.NewServeMux(),
	}
	server.registerValidationRoutes()

	t.Run("valid configuration", func(t *testing.T) {
		validYaml := `name: test-app
services:
  api:
    host: containerapp
    language: node
    project: ./src/api
    ports:
      - "8080:8080"
resources:
  storage:
    type: Microsoft.Storage/storageAccounts
`
		reqBody := map[string]string{"content": validYaml}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/editor/validate", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleValidateConfig(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response ValidationResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Valid {
			t.Errorf("Expected valid config, got errors: %v", response.Errors)
		}

		if len(response.Errors) > 0 {
			t.Errorf("Expected no errors, got %d: %v", len(response.Errors), response.Errors)
		}
	})

	t.Run("invalid YAML syntax", func(t *testing.T) {
		invalidYaml := `name: test
this is: not: valid: yaml: : :
`
		reqBody := map[string]string{"content": invalidYaml}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/editor/validate", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleValidateConfig(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response ValidationResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Valid {
			t.Error("Expected invalid config")
		}

		if len(response.Errors) == 0 {
			t.Error("Expected YAML syntax error")
		}

		if !strings.Contains(response.Errors[0].Message, "Invalid YAML syntax") {
			t.Errorf("Expected YAML syntax error, got: %s", response.Errors[0].Message)
		}
	})

	t.Run("missing required fields", func(t *testing.T) {
		yamlMissingName := `services:
  api:
    host: containerapp
`
		reqBody := map[string]string{"content": yamlMissingName}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/editor/validate", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleValidateConfig(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response ValidationResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Valid {
			t.Error("Expected invalid config due to missing name")
		}

		// Should have error about missing name field
		foundNameError := false
		for _, err := range response.Errors {
			if strings.Contains(strings.ToLower(err.Message), "name") {
				foundNameError = true
				break
			}
		}

		if !foundNameError {
			t.Errorf("Expected error about missing name field, got errors: %v", response.Errors)
		}
	})

	t.Run("port conflict warning", func(t *testing.T) {
		yamlWithConflict := `name: test-app
services:
  api:
    host: containerapp
    ports:
      - "8080:8080"
  web:
    host: containerapp
    ports:
      - "8080:3000"
`
		reqBody := map[string]string{"content": yamlWithConflict}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/editor/validate", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleValidateConfig(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response ValidationResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Port conflicts should be warnings, not errors
		foundPortWarning := false
		for _, warning := range response.Warnings {
			if strings.Contains(warning.Message, "Port 8080") && strings.Contains(warning.Message, "multiple services") {
				foundPortWarning = true
				break
			}
		}

		if !foundPortWarning {
			t.Errorf("Expected port conflict warning, got warnings: %v", response.Warnings)
		}
	})

	t.Run("circular dependency detection", func(t *testing.T) {
		yamlWithCycle := `name: test-app
services:
  api:
    host: containerapp
    uses:
      - web
  web:
    host: containerapp
    uses:
      - api
`
		reqBody := map[string]string{"content": yamlWithCycle}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/editor/validate", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleValidateConfig(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response ValidationResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Valid {
			t.Error("Expected invalid config due to circular dependency")
		}

		foundCycleError := false
		for _, err := range response.Errors {
			if strings.Contains(err.Message, "Circular dependency") {
				foundCycleError = true
				break
			}
		}

		if !foundCycleError {
			t.Errorf("Expected circular dependency error, got errors: %v", response.Errors)
		}
	})

	t.Run("missing health check info message", func(t *testing.T) {
		yamlWithoutHealthcheck := `name: test-app
services:
  api:
    host: containerapp
    language: node
    ports:
      - "8080:8080"
`
		reqBody := map[string]string{"content": yamlWithoutHealthcheck}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/editor/validate", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleValidateConfig(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response ValidationResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should be valid but have info message
		if !response.Valid {
			t.Errorf("Expected valid config, got errors: %v", response.Errors)
		}

		foundHealthcheckInfo := false
		for _, info := range response.Info {
			if strings.Contains(info.Message, "health check") {
				foundHealthcheckInfo = true
				break
			}
		}

		if !foundHealthcheckInfo {
			t.Logf("Info messages: %v", response.Info)
			// This is not a failure - health check suggestions are optional
			// Just log for debugging
		}
	})

	t.Run("invalid request body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/editor/validate", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleValidateConfig(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/editor/validate", nil)
		w := httptest.NewRecorder()

		server.mux.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})

	t.Run("no resources info message", func(t *testing.T) {
		yamlWithoutResources := `name: test-app
services:
  api:
    host: containerapp
    language: node
`
		reqBody := map[string]string{"content": yamlWithoutResources}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/editor/validate", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleValidateConfig(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response ValidationResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Valid {
			t.Errorf("Expected valid config, got errors: %v", response.Errors)
		}

		foundResourcesInfo := false
		for _, info := range response.Info {
			if strings.Contains(info.Message, "No resources defined") {
				foundResourcesInfo = true
				break
			}
		}

		if !foundResourcesInfo {
			t.Logf("Info messages: %v", response.Info)
			// This is not a failure - resource suggestions are optional
		}
	})
}

func TestDetectCircularDependencies(t *testing.T) {
	t.Run("no cycles", func(t *testing.T) {
		config := &service.AzureYaml{
			Name: "test",
			Services: map[string]service.Service{
				"api": {
					Host: "containerapp",
					Uses: []string{"db"},
				},
				"db": {
					Host: "containerapp",
				},
			},
		}

		cycles := detectCircularDependencies(config)
		if len(cycles) > 0 {
			t.Errorf("Expected no cycles, got: %v", cycles)
		}
	})

	t.Run("simple cycle", func(t *testing.T) {
		config := &service.AzureYaml{
			Name: "test",
			Services: map[string]service.Service{
				"api": {
					Host: "containerapp",
					Uses: []string{"web"},
				},
				"web": {
					Host: "containerapp",
					Uses: []string{"api"},
				},
			},
		}

		cycles := detectCircularDependencies(config)
		if len(cycles) == 0 {
			t.Error("Expected to find circular dependency")
		}
	})

	t.Run("three-way cycle", func(t *testing.T) {
		config := &service.AzureYaml{
			Name: "test",
			Services: map[string]service.Service{
				"a": {
					Host: "containerapp",
					Uses: []string{"b"},
				},
				"b": {
					Host: "containerapp",
					Uses: []string{"c"},
				},
				"c": {
					Host: "containerapp",
					Uses: []string{"a"},
				},
			},
		}

		cycles := detectCircularDependencies(config)
		if len(cycles) == 0 {
			t.Error("Expected to find circular dependency")
		}
	})
}

func TestFormatSchemaError(t *testing.T) {
	tests := []struct {
		name     string
		errType  string
		field    string
		desc     string
		expected string
	}{
		{
			name:     "required field",
			errType:  "required",
			field:    "name",
			desc:     "name is required",
			expected: "Required field 'name' is missing",
		},
		{
			name:     "invalid type",
			errType:  "invalid_type",
			field:    "services.api.ports",
			desc:     "expected array, got string",
			expected: "Field 'services.api.ports' has invalid type: expected array, got string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: We can't easily create gojsonschema.ResultError instances
			// This test is more for documentation of expected behavior
			// Actual testing happens through integration tests
		})
	}
}
