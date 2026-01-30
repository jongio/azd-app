package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleAzureSetupAutoFix(t *testing.T) {
	tests := []struct {
		name           string
		setupDir       func(t *testing.T, dir string)
		requestBody    string
		wantSuccess    bool
		wantApplied    bool
		wantStatusCode int
	}{
		{
			name: "auto-fix adds outputs to bicep file",
			setupDir: func(t *testing.T, dir string) {
				t.Helper()
				infraDir := filepath.Join(dir, "infra")
				if err := os.MkdirAll(infraDir, 0755); err != nil {
					t.Fatalf("Failed to create infra dir: %v", err)
				}
				bicepContent := `
param location string = resourceGroup().location

module logAnalytics 'br/public:avm/res/operational-insights/workspace:0.12.0' = {
  name: 'log-analytics'
  params: {
    name: 'log-test'
    location: location
  }
}

output AZURE_LOCATION string = location
`
				if err := os.WriteFile(filepath.Join(infraDir, "main.bicep"), []byte(bicepContent), 0644); err != nil {
					t.Fatalf("Failed to write bicep file: %v", err)
				}
			},
			requestBody:    `{"action": "add-bicep-outputs"}`,
			wantSuccess:    true,
			wantApplied:    true,
			wantStatusCode: http.StatusOK,
		},
		{
			name: "outputs already exist",
			setupDir: func(t *testing.T, dir string) {
				t.Helper()
				infraDir := filepath.Join(dir, "infra")
				if err := os.MkdirAll(infraDir, 0755); err != nil {
					t.Fatalf("Failed to create infra dir: %v", err)
				}
				bicepContent := `
module logAnalytics 'br/public:avm/res/operational-insights/workspace:0.12.0' = {}

output AZURE_LOG_ANALYTICS_WORKSPACE_ID string = logAnalytics.outputs.resourceId
output AZURE_LOG_ANALYTICS_WORKSPACE_GUID string = logAnalytics.outputs.logAnalyticsWorkspaceId
`
				if err := os.WriteFile(filepath.Join(infraDir, "main.bicep"), []byte(bicepContent), 0644); err != nil {
					t.Fatalf("Failed to write bicep file: %v", err)
				}
			},
			requestBody:    `{"action": "add-bicep-outputs"}`,
			wantSuccess:    true,
			wantApplied:    false, // Already exists
			wantStatusCode: http.StatusOK,
		},
		{
			name: "no bicep file found",
			setupDir: func(t *testing.T, dir string) {
				t.Helper()
				// No bicep file created
			},
			requestBody:    `{"action": "add-bicep-outputs"}`,
			wantSuccess:    false,
			wantApplied:    false,
			wantStatusCode: http.StatusOK,
		},
		{
			name: "no log analytics module found",
			setupDir: func(t *testing.T, dir string) {
				t.Helper()
				infraDir := filepath.Join(dir, "infra")
				if err := os.MkdirAll(infraDir, 0755); err != nil {
					t.Fatalf("Failed to create infra dir: %v", err)
				}
				bicepContent := `
param location string = resourceGroup().location
output AZURE_LOCATION string = location
`
				if err := os.WriteFile(filepath.Join(infraDir, "main.bicep"), []byte(bicepContent), 0644); err != nil {
					t.Fatalf("Failed to write bicep file: %v", err)
				}
			},
			requestBody:    `{"action": "add-bicep-outputs"}`,
			wantSuccess:    false,
			wantApplied:    false,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "unknown action",
			setupDir:       func(t *testing.T, dir string) {},
			requestBody:    `{"action": "unknown-action"}`,
			wantSuccess:    false,
			wantApplied:    false,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid json",
			setupDir:       func(t *testing.T, dir string) {},
			requestBody:    `{invalid json}`,
			wantSuccess:    false,
			wantApplied:    false,
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setupDir(t, dir)

			server := &Server{projectDir: dir}
			req := httptest.NewRequest(http.MethodPost, "/api/azure/setup/auto-fix", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleAzureSetupAutoFix(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("expected status %d, got %d", tt.wantStatusCode, w.Code)
			}

			if tt.wantStatusCode != http.StatusOK {
				return
			}

			var response AutoFixResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if response.Success != tt.wantSuccess {
				t.Errorf("expected success=%v, got %v. Message: %s", tt.wantSuccess, response.Success, response.Message)
			}

			if response.Applied != tt.wantApplied {
				t.Errorf("expected applied=%v, got %v", tt.wantApplied, response.Applied)
			}

			// If applied, verify the file was modified
			if tt.wantApplied {
				bicepPath := filepath.Join(dir, "infra", "main.bicep")
				content, err := os.ReadFile(bicepPath)
				if err != nil {
					t.Fatalf("failed to read modified bicep file: %v", err)
				}

				contentStr := string(content)
				if !strings.Contains(contentStr, "AZURE_LOG_ANALYTICS_WORKSPACE_ID") {
					t.Error("expected modified file to contain AZURE_LOG_ANALYTICS_WORKSPACE_ID")
				}
				if !strings.Contains(contentStr, "AZURE_LOG_ANALYTICS_WORKSPACE_GUID") {
					t.Error("expected modified file to contain AZURE_LOG_ANALYTICS_WORKSPACE_GUID")
				}
			}
		})
	}
}

func TestDetectLogAnalyticsModuleName(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "AVM module",
			content: `module logAnalytics 'br/public:avm/res/operational-insights/workspace:0.12.0' = {
  name: 'log-analytics'
}`,
			expected: "logAnalytics",
		},
		{
			name: "custom module path",
			content: `module monitoring './modules/log-analytics-workspace.bicep' = {
  name: 'monitoring'
}`,
			expected: "monitoring",
		},
		{
			name: "direct resource",
			content: `resource logWorkspace 'Microsoft.OperationalInsights/workspaces@2022-10-01' = {
  name: 'log-workspace'
}`,
			expected: "logWorkspace",
		},
		{
			name:     "no log analytics",
			content:  `module storage 'br/public:avm/res/storage/storage-account:0.1.0' = {}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectLogAnalyticsModuleName(tt.content)
			if result != tt.expected {
				t.Errorf("detectLogAnalyticsModuleName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestInsertBicepOutputs(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		outputs     string
		wantContain string
	}{
		{
			name: "insert after existing outputs",
			content: `param location string
output AZURE_LOCATION string = location
output AZURE_RG string = resourceGroup().name
`,
			outputs:     "\noutput TEST string = 'test'\n",
			wantContain: "output TEST string = 'test'",
		},
		{
			name: "append to file without outputs",
			content: `param location string
var name = 'test'
`,
			outputs:     "\noutput TEST string = 'test'\n",
			wantContain: "output TEST string = 'test'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, lineNum := insertBicepOutputs(tt.content, tt.outputs)
			if !strings.Contains(result, tt.wantContain) {
				t.Errorf("insertBicepOutputs() result doesn't contain %q", tt.wantContain)
			}
			if lineNum <= 0 {
				t.Errorf("insertBicepOutputs() lineNum = %d, want > 0", lineNum)
			}
		})
	}
}
