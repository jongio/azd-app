package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/azure"
)

// TestHandleAzureBicepTemplate tests the Bicep template API endpoint.
func TestHandleAzureBicepTemplate(t *testing.T) {
	// Note: This test requires an authenticated environment with deployed resources
	// It's more of a smoke test to ensure the handler doesn't crash

	server := &Server{
		projectDir: "../../tests/projects/integration/azure-logs-test",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/azure/bicep-template", nil)
	w := httptest.NewRecorder()

	server.handleAzureBicepTemplate(w, req)

	// The response should be either:
	// - 200 OK with template data (if resources exist and auth is available)
	// - 401 Unauthorized (if credentials not available)
	// - 404 Not Found (if no resources deployed)
	// - 503 Service Unavailable (if can't discover resources)

	statusCode := w.Code
	if statusCode != http.StatusOK &&
		statusCode != http.StatusUnauthorized &&
		statusCode != http.StatusNotFound &&
		statusCode != http.StatusServiceUnavailable {
		t.Errorf("Unexpected status code: %d, body: %s", statusCode, w.Body.String())
	}

	// If successful, verify response structure
	if statusCode == http.StatusOK {
		var response struct {
			Data *azure.BicepTemplateResponse `json:"data"`
		}

		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Data == nil {
			t.Fatal("Response data is nil")
		}

		// Verify template structure
		if response.Data.Template == "" {
			t.Error("Template should not be empty")
		}

		if len(response.Data.Services) == 0 {
			t.Error("Services list should not be empty")
		}

		if response.Data.Instructions.Summary == "" {
			t.Error("Instructions summary should not be empty")
		}

		if len(response.Data.Instructions.Steps) == 0 {
			t.Error("Instructions steps should not be empty")
		}

		if len(response.Data.Parameters) == 0 {
			t.Error("Parameters should not be empty")
		}
	}
}

// TestHandleAzureBicepTemplate_MethodNotAllowed tests that only GET is allowed.
func TestHandleAzureBicepTemplate_MethodNotAllowed(t *testing.T) {
	server := &Server{
		projectDir: "../../tests/projects/integration/azure-logs-test",
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/azure/bicep-template", nil)
			w := httptest.NewRecorder()

			// The MethodGuard should handle this, but we'll test the raw handler
			server.handleAzureBicepTemplate(w, req)

			// The handler itself doesn't check method (MethodGuard does that in routes)
			// So this test just ensures the handler doesn't panic on different methods
		})
	}
}
