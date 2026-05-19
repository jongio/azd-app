package azure

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

// simpleMockCredential is a simple mock implementation of azcore.TokenCredential for testing.
type simpleMockCredential struct {
	token string
	err   error
}

func (m *simpleMockCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	if m.err != nil {
		return azcore.AccessToken{}, m.err
	}
	return azcore.AccessToken{
		Token:     m.token,
		ExpiresOn: time.Now().Add(1 * time.Hour),
	}, nil
}

func TestWorkspaceMatches(t *testing.T) {
	checker := &DiagnosticSettingsChecker{}

	tests := []struct {
		name     string
		actual   string
		expected string
		want     bool
	}{
		{
			name:     "exact match",
			actual:   "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/my-workspace",
			expected: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/my-workspace",
			want:     true,
		},
		{
			name:     "case insensitive match",
			actual:   "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/My-Workspace",
			expected: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/my-workspace",
			want:     true,
		},
		{
			name:     "extract name from resource ID - both resource IDs",
			actual:   "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.OperationalInsights/workspaces/shared-workspace",
			expected: "/subscriptions/sub2/resourceGroups/rg2/providers/Microsoft.OperationalInsights/workspaces/shared-workspace",
			want:     true,
		},
		{
			name:     "extract name from resource ID - one name only",
			actual:   "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/my-workspace",
			expected: "my-workspace",
			want:     true,
		},
		{
			name:     "extract name from resource ID - other name only",
			actual:   "my-workspace",
			expected: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/my-workspace",
			want:     true,
		},
		{
			name:     "different workspace names",
			actual:   "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/workspace-a",
			expected: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/workspace-b",
			want:     false,
		},
		{
			name:     "empty strings",
			actual:   "",
			expected: "my-workspace",
			want:     false,
		},
		{
			name:     "both empty",
			actual:   "",
			expected: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checker.workspaceMatches(tt.actual, tt.expected)
			if got != tt.want {
				t.Errorf("workspaceMatches(%q, %q) = %v, want %v", tt.actual, tt.expected, got, tt.want)
			}
		})
	}
}

func TestExtractWorkspaceName(t *testing.T) {
	tests := []struct {
		name       string
		resourceID string
		want       string
	}{
		{
			name:       "full resource ID",
			resourceID: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/my-workspace",
			want:       "my-workspace",
		},
		{
			name:       "case insensitive",
			resourceID: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.OperationalInsights/Workspaces/My-Workspace",
			want:       "my-workspace",
		},
		{
			name:       "workspace name with hyphens and numbers",
			resourceID: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/test-workspace-123",
			want:       "test-workspace-123",
		},
		{
			name:       "not a resource ID",
			resourceID: "my-workspace",
			want:       "",
		},
		{
			name:       "different resource type",
			resourceID: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/mystorage",
			want:       "",
		},
		{
			name:       "empty string",
			resourceID: "",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractWorkspaceName(tt.resourceID)
			if got != tt.want {
				t.Errorf("extractWorkspaceName(%q) = %q, want %q", tt.resourceID, got, tt.want)
			}
		})
	}
}

func TestDiagnosticSettingsResponse_Serialization(t *testing.T) {
	// Test that the response types serialize correctly to JSON
	response := DiagnosticSettingsCheckResponse{
		WorkspaceID: "/subscriptions/test/workspaces/test-workspace",
		Services: map[string]*DiagnosticSettingsCheckResult{
			"api": {
				Status:                DiagnosticSettingsConfigured,
				ResourceID:            "/subscriptions/test/providers/Microsoft.Web/sites/api",
				DiagnosticSettingName: "toLogAnalytics",
				WorkspaceID:           "/subscriptions/test/workspaces/test-workspace",
			},
			"web": {
				Status:     DiagnosticSettingsNotConfigured,
				ResourceID: "/subscriptions/test/providers/Microsoft.Web/sites/web",
				Error:      "No diagnostic settings found",
			},
			"function": {
				Status:     DiagnosticSettingsError,
				ResourceID: "/subscriptions/test/providers/Microsoft.Web/sites/function",
				Error:      "Insufficient permissions",
			},
		},
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	// Deserialize back
	var decoded DiagnosticSettingsCheckResponse
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify
	if decoded.WorkspaceID != response.WorkspaceID {
		t.Errorf("WorkspaceID mismatch: got %q, want %q", decoded.WorkspaceID, response.WorkspaceID)
	}

	if len(decoded.Services) != len(response.Services) {
		t.Errorf("Services count mismatch: got %d, want %d", len(decoded.Services), len(response.Services))
	}

	// Check specific service
	apiService := decoded.Services["api"]
	if apiService == nil {
		t.Fatal("api service not found in decoded response")
	}

	if apiService.Status != DiagnosticSettingsConfigured {
		t.Errorf("api service status mismatch: got %s, want %s", apiService.Status, DiagnosticSettingsConfigured)
	}

	if apiService.DiagnosticSettingName != "toLogAnalytics" {
		t.Errorf("api service setting name mismatch: got %q, want %q", apiService.DiagnosticSettingName, "toLogAnalytics")
	}
}

func TestDiagnosticSettingsStatus_StringValues(t *testing.T) {
	// Verify the status constants have the expected string values
	if DiagnosticSettingsConfigured != "configured" {
		t.Errorf("DiagnosticSettingsConfigured = %q, want %q", DiagnosticSettingsConfigured, "configured")
	}

	if DiagnosticSettingsNotConfigured != "not-configured" {
		t.Errorf("DiagnosticSettingsNotConfigured = %q, want %q", DiagnosticSettingsNotConfigured, "not-configured")
	}

	if DiagnosticSettingsError != "error" {
		t.Errorf("DiagnosticSettingsError = %q, want %q", DiagnosticSettingsError, "error")
	}
}
