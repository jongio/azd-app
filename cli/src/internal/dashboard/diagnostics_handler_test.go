package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/jongio/azd-app/cli/src/internal/azure"
)

func TestHandleAzureDiagnostics_Success(t *testing.T) {
	oldCred := newLogAnalyticsCredential
	t.Cleanup(func() {
		newLogAnalyticsCredential = oldCred
	})

	// Mock credential
	newLogAnalyticsCredential = func() (azcore.TokenCredential, error) {
		return fakeTokenCredential{}, nil
	}

	srv := GetServer(t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/api/azure/diagnostics", nil)
	w := httptest.NewRecorder()

	srv.handleAzureDiagnostics(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var result azure.DiagnosticsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response structure
	if result.Services == nil {
		t.Error("Expected services map to be present")
	}

	// WorkspaceID may be empty in test environment
	t.Logf("WorkspaceID: %s", result.WorkspaceID)
}

func TestHandleAzureDiagnostics_NoCredentials(t *testing.T) {
	oldCred := newLogAnalyticsCredential
	t.Cleanup(func() {
		newLogAnalyticsCredential = oldCred
	})

	// Mock credential failure
	newLogAnalyticsCredential = func() (azcore.TokenCredential, error) {
		return nil, &mockError{message: "credentials not available"}
	}

	srv := GetServer(t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/api/azure/diagnostics", nil)
	w := httptest.NewRecorder()

	srv.handleAzureDiagnostics(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected status 401, got %d", resp.StatusCode)
	}

	var result struct {
		Error   string `json:"error"`
		Details string `json:"details,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if result.Error == "" {
		t.Error("Expected error field to be set")
	}
}

func TestHandleAzureDiagnostics_Timeout(t *testing.T) {
	// This test verifies timeout handling
	// In actual implementation, would need to mock a slow operation
	t.Skip("Timeout test requires mocking slow operations")
}

func TestHandleAzureDiagnostics_MethodGuard(t *testing.T) {
	srv := GetServer(t.TempDir())

	// Test POST method (should be rejected)
	req := httptest.NewRequest(http.MethodPost, "/api/azure/diagnostics", nil)
	w := httptest.NewRecorder()

	srv.mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("Expected status 405 for POST, got %d", resp.StatusCode)
	}

	// Test GET method (should be allowed)
	req = httptest.NewRequest(http.MethodGet, "/api/azure/diagnostics", nil)
	w = httptest.NewRecorder()

	// This will fail auth in test env, but should not fail method guard
	srv.mux.ServeHTTP(w, req)

	resp = w.Result()
	// Should not be 405
	if resp.StatusCode == http.StatusMethodNotAllowed {
		t.Error("GET method should be allowed")
	}
}

func TestDiagnosticsResponse_JSONSerialization(t *testing.T) {
	// Test that DiagnosticsResponse serializes correctly to JSON
	response := azure.DiagnosticsResponse{
		WorkspaceID:   "/subscriptions/test/workspaces/test-workspace",
		WorkspaceName: "test-workspace",
		Services: map[string]*azure.ServiceDiagnosticResult{
			"api": {
				HostType: azure.ResourceTypeContainerApp,
				Status:   azure.DiagnosticStatusHealthy,
				LogCount: 100,
				Requirements: []azure.Requirement{
					{
						Name:        "Diagnostic Settings",
						Status:      azure.RequirementStatusMet,
						Description: "Configured correctly",
					},
				},
				Message: "Logs flowing",
			},
			"web": {
				HostType: azure.ResourceTypeAppService,
				Status:   azure.DiagnosticStatusNotConfigured,
				LogCount: 0,
				Requirements: []azure.Requirement{
					{
						Name:        "Diagnostic Settings",
						Status:      azure.RequirementStatusNotMet,
						Description: "Not configured",
						HowToFix:    "Run azd up",
					},
				},
				SetupGuide: &azure.SetupGuide{
					Title:       "Setup Guide",
					Description: "Follow these steps",
					Steps: []azure.SetupGuideStep{
						{
							Title:       "Step 1",
							Description: "Configure diagnostic settings",
							Command:     "azd up",
						},
					},
				},
				Message: "Diagnostic settings not configured",
			},
		},
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	// Deserialize back
	var decoded azure.DiagnosticsResponse
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify basic fields
	if decoded.WorkspaceID != response.WorkspaceID {
		t.Errorf("WorkspaceID mismatch: got %q, want %q", decoded.WorkspaceID, response.WorkspaceID)
	}

	if decoded.WorkspaceName != response.WorkspaceName {
		t.Errorf("WorkspaceName mismatch: got %q, want %q", decoded.WorkspaceName, response.WorkspaceName)
	}

	// Verify services
	if len(decoded.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(decoded.Services))
	}

	// Verify API service
	apiService := decoded.Services["api"]
	if apiService == nil {
		t.Fatal("API service not found")
	}

	if apiService.Status != azure.DiagnosticStatusHealthy {
		t.Errorf("API status mismatch: got %s, want %s", apiService.Status, azure.DiagnosticStatusHealthy)
	}

	if apiService.LogCount != 100 {
		t.Errorf("API log count mismatch: got %d, want %d", apiService.LogCount, 100)
	}

	// Verify web service
	webService := decoded.Services["web"]
	if webService == nil {
		t.Fatal("Web service not found")
	}

	if webService.SetupGuide == nil {
		t.Error("Web service should have setup guide")
	}

	if len(webService.SetupGuide.Steps) != 1 {
		t.Errorf("Expected 1 setup step, got %d", len(webService.SetupGuide.Steps))
	}

	// Verify requirements
	if len(apiService.Requirements) != 1 {
		t.Errorf("Expected 1 requirement for api, got %d", len(apiService.Requirements))
	}

	if len(webService.Requirements) != 1 {
		t.Errorf("Expected 1 requirement for web, got %d", len(webService.Requirements))
	}
}

func TestDiagnosticStatus_AllValidValues(t *testing.T) {
	validStatuses := []azure.DiagnosticStatus{
		azure.DiagnosticStatusHealthy,
		azure.DiagnosticStatusPartial,
		azure.DiagnosticStatusNotConfigured,
		azure.DiagnosticStatusError,
	}

	for _, status := range validStatuses {
		// Test JSON marshaling
		data, err := json.Marshal(status)
		if err != nil {
			t.Errorf("Failed to marshal status %s: %v", status, err)
		}

		var decoded azure.DiagnosticStatus
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Errorf("Failed to unmarshal status %s: %v", status, err)
		}

		if decoded != status {
			t.Errorf("Status mismatch after JSON roundtrip: got %s, want %s", decoded, status)
		}
	}
}

func TestRequirementStatus_AllValidValues(t *testing.T) {
	validStatuses := []azure.RequirementStatus{
		azure.RequirementStatusMet,
		azure.RequirementStatusNotMet,
		azure.RequirementStatusUnknown,
	}

	for _, status := range validStatuses {
		// Test JSON marshaling
		data, err := json.Marshal(status)
		if err != nil {
			t.Errorf("Failed to marshal status %s: %v", status, err)
		}

		var decoded azure.RequirementStatus
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Errorf("Failed to unmarshal status %s: %v", status, err)
		}

		if decoded != status {
			t.Errorf("Status mismatch after JSON roundtrip: got %s, want %s", decoded, status)
		}
	}
}

type mockError struct {
	message string
}

func (e *mockError) Error() string {
	return e.message
}
