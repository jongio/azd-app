package dashboard

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

// createTestAzureYaml creates a temporary azure.yaml for testing
func createTestAzureYaml(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")

	if content != "" {
		if err := os.WriteFile(azureYamlPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test azure.yaml: %v", err)
		}
	}

	return tmpDir
}

// TestHandleGetClassifications tests the GET /api/logs/classifications endpoint
func TestHandleGetClassifications(t *testing.T) {
	tests := []struct {
		name           string
		azureYaml      string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "no azure.yaml returns empty list",
			azureYaml:      "",
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name: "empty logs section returns empty list",
			azureYaml: `name: test-project
services:
  api:
    host: container
`,
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name: "returns classifications from azure.yaml",
			azureYaml: `name: test-project
logs:
  classifications:
    - text: "Connection refused"
      level: error
    - text: "cache miss"
      level: info
`,
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := createTestAzureYaml(t, tt.azureYaml)

			server := &Server{projectDir: tmpDir}

			req := httptest.NewRequest(http.MethodGet, "/api/logs/classifications", nil)
			w := httptest.NewRecorder()

			server.handleGetClassifications(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			var result ClassificationsResponse
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if len(result.Classifications) != tt.expectedCount {
				t.Errorf("Expected %d classifications, got %d", tt.expectedCount, len(result.Classifications))
			}
		})
	}
}

// TestHandleCreateClassification tests the POST /api/logs/classifications endpoint
func TestHandleCreateClassification(t *testing.T) {
	tests := []struct {
		name           string
		azureYaml      string
		requestBody    string
		expectedStatus int
		expectedText   string
		expectedLevel  string
	}{
		{
			name: "adds new classification",
			azureYaml: `name: test-project
`,
			requestBody:    `{"text": "Connection refused", "level": "error"}`,
			expectedStatus: http.StatusCreated,
			expectedText:   "Connection refused",
			expectedLevel:  "error",
		},
		{
			name: "adds to existing classifications",
			azureYaml: `name: test-project
logs:
  classifications:
    - text: "existing"
      level: info
`,
			requestBody:    `{"text": "new error", "level": "warning"}`,
			expectedStatus: http.StatusCreated,
			expectedText:   "new error",
			expectedLevel:  "warning",
		},
		{
			name: "updates duplicate classification",
			azureYaml: `name: test-project
logs:
  classifications:
    - text: "Connection refused"
      level: warning
`,
			requestBody:    `{"text": "Connection refused", "level": "error"}`,
			expectedStatus: http.StatusOK, // Update returns 200, not 201
			expectedText:   "Connection refused",
			expectedLevel:  "error",
		},
		{
			name: "rejects empty text",
			azureYaml: `name: test-project
`,
			requestBody:    `{"text": "", "level": "error"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "rejects whitespace-only text",
			azureYaml: `name: test-project
`,
			requestBody:    `{"text": "   ", "level": "error"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "rejects invalid level",
			azureYaml: `name: test-project
`,
			requestBody:    `{"text": "test", "level": "critical"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "rejects invalid JSON",
			azureYaml: `name: test-project
`,
			requestBody:    `{invalid json`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := createTestAzureYaml(t, tt.azureYaml)

			server := &Server{projectDir: tmpDir}

			req := httptest.NewRequest(http.MethodPost, "/api/logs/classifications", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleCreateClassification(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, resp.StatusCode, string(body))
				return
			}

			if tt.expectedStatus == http.StatusCreated || tt.expectedStatus == http.StatusOK {
				var result service.LogClassification
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if result.Text != tt.expectedText {
					t.Errorf("Expected text %q, got %q", tt.expectedText, result.Text)
				}
				if result.Level != tt.expectedLevel {
					t.Errorf("Expected level %q, got %q", tt.expectedLevel, result.Level)
				}

				// Verify it was saved to azure.yaml
				data, err := os.ReadFile(filepath.Join(tmpDir, "azure.yaml"))
				if err != nil {
					t.Fatalf("Failed to read azure.yaml: %v", err)
				}
				if !bytes.Contains(data, []byte(tt.expectedText)) {
					t.Errorf("Classification not found in saved azure.yaml")
				}
			}
		})
	}
}

// TestHandleDeleteClassification tests the DELETE /api/logs/classifications/{index} endpoint
func TestHandleDeleteClassification(t *testing.T) {
	tests := []struct {
		name           string
		azureYaml      string
		index          string
		expectedStatus int
		remainingCount int
	}{
		{
			name: "deletes first classification",
			azureYaml: `name: test-project
logs:
  classifications:
    - text: "first"
      level: info
    - text: "second"
      level: error
`,
			index:          "0",
			expectedStatus: http.StatusNoContent,
			remainingCount: 1,
		},
		{
			name: "deletes last classification",
			azureYaml: `name: test-project
logs:
  classifications:
    - text: "first"
      level: info
    - text: "second"
      level: error
`,
			index:          "1",
			expectedStatus: http.StatusNoContent,
			remainingCount: 1,
		},
		{
			name: "deletes only classification (cleans up logs section)",
			azureYaml: `name: test-project
logs:
  classifications:
    - text: "only"
      level: info
`,
			index:          "0",
			expectedStatus: http.StatusNoContent,
			remainingCount: 0,
		},
		{
			name: "returns 404 for out of range index",
			azureYaml: `name: test-project
logs:
  classifications:
    - text: "only"
      level: info
`,
			index:          "99",
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "returns 404 for negative index",
			azureYaml: `name: test-project
logs:
  classifications:
    - text: "only"
      level: info
`,
			index:          "-1",
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "returns 400 for non-numeric index",
			azureYaml: `name: test-project
logs:
  classifications:
    - text: "only"
      level: info
`,
			index:          "abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "returns 404 when no classifications exist",
			azureYaml: `name: test-project
`,
			index:          "0",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := createTestAzureYaml(t, tt.azureYaml)

			server := &Server{projectDir: tmpDir}

			req := httptest.NewRequest(http.MethodDelete, "/api/logs/classifications/"+tt.index, nil)
			w := httptest.NewRecorder()

			server.handleDeleteClassification(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, resp.StatusCode, string(body))
				return
			}

			if tt.expectedStatus == http.StatusNoContent {
				// Verify remaining count
				azureYaml, err := loadAzureYaml(tmpDir)
				if err != nil && tt.remainingCount > 0 {
					t.Fatalf("Failed to load azure.yaml after delete: %v", err)
				}

				actualCount := 0
				if azureYaml != nil && azureYaml.Logs != nil {
					actualCount = len(azureYaml.Logs.Classifications)
				}

				if actualCount != tt.remainingCount {
					t.Errorf("Expected %d remaining classifications, got %d", tt.remainingCount, actualCount)
				}
			}
		})
	}
}

// TestHandleClassificationsRouter tests the router
func TestHandleClassificationsRouter(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "GET /api/logs/classifications routes correctly",
			method:         http.MethodGet,
			path:           "/api/logs/classifications",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST /api/logs/classifications routes correctly",
			method:         http.MethodPost,
			path:           "/api/logs/classifications",
			expectedStatus: http.StatusBadRequest, // No body, so bad request
		},
		{
			name:           "DELETE /api/logs/classifications/0 routes correctly",
			method:         http.MethodDelete,
			path:           "/api/logs/classifications/0",
			expectedStatus: http.StatusNotFound, // No classifications
		},
		{
			name:           "PUT /api/logs/classifications not allowed",
			method:         http.MethodPut,
			path:           "/api/logs/classifications",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Unknown subpath returns 405 for wrong method",
			method:         http.MethodGet,
			path:           "/api/logs/classifications/0",
			expectedStatus: http.StatusMethodNotAllowed, // GET not allowed on /classifications/{id}
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := createTestAzureYaml(t, `name: test-project`)

			server := &Server{projectDir: tmpDir}

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			server.handleClassificationsRouter(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, resp.StatusCode, string(body))
			}
		})
	}
}

// TestValidateClassificationLevel tests the level validation function
func TestValidateClassificationLevel(t *testing.T) {
	tests := []struct {
		level    string
		expected bool
	}{
		{"info", true},
		{"warning", true},
		{"error", true},
		{"INFO", false},    // Case sensitive
		{"Warning", false}, // Case sensitive
		{"ERROR", false},   // Case sensitive
		{"debug", false},
		{"critical", false},
		{"", false},
		{" info ", false}, // No trimming
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			result := service.ValidateClassificationLevel(tt.level)
			if result != tt.expected {
				t.Errorf("ValidateClassificationLevel(%q) = %v, expected %v", tt.level, result, tt.expected)
			}
		})
	}
}
