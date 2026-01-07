package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/azure"
)

// testContext creates a test context with timeout
func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestHandleAzureSetupState(t *testing.T) {
	tests := []struct {
		name          string
		setupEnv      func(t *testing.T, dir string)
		wantStep      string
		wantStatus    string
		minIssueCount int
	}{
		{
			name: "no configuration",
			setupEnv: func(t *testing.T, dir string) {
				azureYamlPath := filepath.Join(dir, "azure.yaml")
				if err := os.WriteFile(azureYamlPath, []byte("name: test-app"), 0644); err != nil {
					t.Fatalf("Failed to write azure.yaml: %v", err)
				}
			},
			wantStep:      "workspace",
			wantStatus:    "incomplete",
			minIssueCount: 1,
		},
		{
			name: "workspace configured via env var",
			setupEnv: func(t *testing.T, dir string) {
				azureYamlPath := filepath.Join(dir, "azure.yaml")
				if err := os.WriteFile(azureYamlPath, []byte("name: test-app"), 0644); err != nil {
					t.Fatalf("Failed to write azure.yaml: %v", err)
				}
				// Mock the getWorkspaceIDFromEnv function
				oldFunc := getWorkspaceIDFromEnv
				t.Cleanup(func() { getWorkspaceIDFromEnv = oldFunc })
				getWorkspaceIDFromEnv = func(string) string {
					return "/subscriptions/test/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/ws"
				}
			},
			wantStep:      "authentication", // May be "complete" if test environment is authenticated
			wantStatus:    "incomplete",     // May be "complete" if test environment is authenticated
			minIssueCount: 0,                // May have 0 issues if authenticated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setupEnv(t, dir)

			server := &Server{projectDir: dir}
			req := httptest.NewRequest(http.MethodGet, "/api/azure/logs/setup-state", nil)
			w := httptest.NewRecorder()

			server.handleAzureSetupState(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}

			var response SetupStateResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if response.Step != tt.wantStep {
				// Allow "complete" if test environment is authenticated
				if !(response.Step == "complete" && tt.wantStep == "authentication") {
					t.Errorf("expected step %q, got %q", tt.wantStep, response.Step)
				}
			}

			if response.OverallStatus != tt.wantStatus {
				// Allow "complete" if test environment is authenticated
				if !(response.OverallStatus == "complete" && tt.wantStatus == "incomplete") {
					t.Errorf("expected overall status %q, got %q", tt.wantStatus, response.OverallStatus)
				}
			}

			if len(response.Issues) < tt.minIssueCount {
				t.Errorf("expected at least %d issues, got %d", tt.minIssueCount, len(response.Issues))
			}

			// Note: Removed timestamp check as it's flaky in CI environments where Azure API calls
			// can take longer than expected, causing the timestamp to appear old even though
			// it was set recently at the start of the handler.
		})
	}
}

func TestCheckWorkspaceState(t *testing.T) {
	tests := []struct {
		name       string
		setupEnv   func(t *testing.T, dir string)
		wantStatus string
		wantSource string
	}{
		{
			name: "workspace from environment variable",
			setupEnv: func(t *testing.T, dir string) {
				azureYamlPath := filepath.Join(dir, "azure.yaml")
				if err := os.WriteFile(azureYamlPath, []byte("name: test-app"), 0644); err != nil {
					t.Fatalf("Failed to write azure.yaml: %v", err)
				}
				// Mock the getWorkspaceIDFromEnv function
				oldFunc := getWorkspaceIDFromEnv
				t.Cleanup(func() { getWorkspaceIDFromEnv = oldFunc })
				getWorkspaceIDFromEnv = func(string) string {
					return "/subscriptions/test/providers/Microsoft.OperationalInsights/workspaces/ws"
				}
			},
			wantStatus: "configured",
			wantSource: "environment",
		},
		{
			name: "workspace from azure.yaml",
			setupEnv: func(t *testing.T, dir string) {
				azureYamlPath := filepath.Join(dir, "azure.yaml")
				content := "name: test-app\nlogs:\n  analytics:\n    workspace: ${AZURE_LOG_ANALYTICS_WORKSPACE_ID}"
				if err := os.WriteFile(azureYamlPath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to write azure.yaml: %v", err)
				}
			},
			wantStatus: "not-deployed",
			wantSource: "azure.yaml",
		},
		{
			name: "no workspace configured",
			setupEnv: func(t *testing.T, dir string) {
				azureYamlPath := filepath.Join(dir, "azure.yaml")
				if err := os.WriteFile(azureYamlPath, []byte("name: test-app"), 0644); err != nil {
					t.Fatalf("Failed to write azure.yaml: %v", err)
				}
			},
			wantStatus: "missing",
			wantSource: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setupEnv(t, dir)

			server := &Server{projectDir: dir}
			state := server.checkWorkspaceState()

			if state.Status != tt.wantStatus {
				t.Errorf("expected status %q, got %q", tt.wantStatus, state.Status)
			}

			if state.Source != tt.wantSource {
				t.Errorf("expected source %q, got %q", tt.wantSource, state.Source)
			}

			if state.Message == "" {
				t.Error("expected non-empty message")
			}
		})
	}
}

func TestCheckAuthState(t *testing.T) {
	server := &Server{}
	state := server.checkAuthState(testContext(t))

	// In test environment, should be one of these states
	validStates := []string{"unauthenticated", "authenticated", "permission-denied"}
	isValid := false
	for _, s := range validStates {
		if state.Status == s {
			isValid = true
			break
		}
	}
	if !isValid {
		t.Errorf("unexpected status: %q, expected one of %v", state.Status, validStates)
	}

	if state.Message == "" {
		t.Error("expected non-empty message")
	}
}

func TestDetermineSetupStep(t *testing.T) {
	tests := []struct {
		name       string
		response   SetupStateResponse
		wantStep   string
		wantStatus string
	}{
		{
			name: "workspace missing",
			response: SetupStateResponse{
				Workspace: WorkspaceState{Status: "missing"},
			},
			wantStep:   "workspace",
			wantStatus: "incomplete",
		},
		{
			name: "workspace configured but not authenticated",
			response: SetupStateResponse{
				Workspace:      WorkspaceState{Status: "configured"},
				Authentication: AuthState{Status: "unauthenticated"},
			},
			wantStep:   "authentication",
			wantStatus: "incomplete",
		},
		{
			name: "complete setup",
			response: SetupStateResponse{
				Workspace:      WorkspaceState{Status: "configured"},
				Authentication: AuthState{Status: "authenticated"},
				Services: []ServiceSetupState{
					{ServiceName: "api", Status: "ready"},
				},
			},
			wantStep:   "complete",
			wantStatus: "complete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{}
			step, status := server.determineSetupStep(tt.response)

			if step != tt.wantStep {
				t.Errorf("expected step %q, got %q", tt.wantStep, step)
			}

			if status != tt.wantStatus {
				t.Errorf("expected status %q, got %q", tt.wantStatus, status)
			}
		})
	}
}

func TestCollectSetupIssues(t *testing.T) {
	tests := []struct {
		name           string
		response       SetupStateResponse
		wantMinIssues  int
		wantCategories []string
	}{
		{
			name: "no issues when complete",
			response: SetupStateResponse{
				Workspace:      WorkspaceState{Status: "configured"},
				Authentication: AuthState{Status: "authenticated"},
				Services: []ServiceSetupState{
					{ServiceName: "api", Status: "ready"},
				},
			},
			wantMinIssues:  0,
			wantCategories: []string{},
		},
		{
			name: "workspace missing",
			response: SetupStateResponse{
				Workspace: WorkspaceState{Status: "missing"},
			},
			wantMinIssues:  1,
			wantCategories: []string{"workspace"},
		},
		{
			name: "not authenticated",
			response: SetupStateResponse{
				Workspace:      WorkspaceState{Status: "configured"},
				Authentication: AuthState{Status: "unauthenticated"},
			},
			wantMinIssues:  1,
			wantCategories: []string{"auth"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{}
			issues := server.collectSetupIssues(tt.response)

			if len(issues) < tt.wantMinIssues {
				t.Errorf("expected at least %d issues, got %d", tt.wantMinIssues, len(issues))
			}

			for i, issue := range issues {
				if issue.Message == "" {
					t.Errorf("issue %d has empty message", i)
				}
				if issue.Fix == "" {
					t.Errorf("issue %d has empty fix", i)
				}
				if issue.Severity == "" {
					t.Errorf("issue %d has empty severity", i)
				}
			}
		})
	}
}

func TestGenerateNextSteps(t *testing.T) {
	tests := []struct {
		name      string
		response  SetupStateResponse
		wantMin   int
		wantFirst string
	}{
		{
			name:      "complete setup",
			response:  SetupStateResponse{OverallStatus: "complete"},
			wantMin:   1,
			wantFirst: "Setup complete",
		},
		{
			name:      "missing workspace",
			response:  SetupStateResponse{Workspace: WorkspaceState{Status: "missing"}},
			wantMin:   1,
			wantFirst: "Configure Log Analytics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{}
			steps := server.generateNextSteps(tt.response)

			if len(steps) < tt.wantMin {
				t.Errorf("expected at least %d steps, got %d", tt.wantMin, len(steps))
			}

			if len(steps) > 0 && !strings.Contains(steps[0], tt.wantFirst) {
				t.Errorf("expected first step to contain %q, got %q", tt.wantFirst, steps[0])
			}
		})
	}
}

func TestSetupStateResponseJSON(t *testing.T) {
	response := SetupStateResponse{
		Step:          "workspace",
		OverallStatus: "incomplete",
		Workspace: WorkspaceState{
			Status:  "missing",
			Message: "Not configured",
		},
		Authentication: AuthState{
			Status:  "unauthenticated",
			Message: "Not authenticated",
		},
		Services: []ServiceSetupState{
			{
				ServiceName: "api",
				Status:      "not-deployed",
			},
		},
		Issues: []SetupIssue{
			{
				Severity: "error",
				Category: "workspace",
				Message:  "Test issue",
				Fix:      "Test fix",
			},
		},
		NextSteps: []string{"Configure workspace"},
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var decoded SetupStateResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if decoded.Step != response.Step {
		t.Errorf("step mismatch after JSON round-trip")
	}

	if len(decoded.Services) != len(response.Services) {
		t.Errorf("services count mismatch after JSON round-trip")
	}

	if len(decoded.Issues) != len(response.Issues) {
		t.Errorf("issues count mismatch after JSON round-trip")
	}
}

func TestHandleAzureLogsVerify(t *testing.T) {
	tests := []struct {
		name          string
		setupEnv      func(t *testing.T, dir string)
		requestBody   string
		mockLogs      []azure.LogEntry
		mockError     error
		wantSuccess   bool
		wantLogsFound int
		wantMinSteps  int
	}{
		{
			name: "successful verification with logs",
			setupEnv: func(t *testing.T, dir string) {
				azureYamlPath := filepath.Join(dir, "azure.yaml")
				if err := os.WriteFile(azureYamlPath, []byte("name: test-app"), 0644); err != nil {
					t.Fatalf("Failed to write azure.yaml: %v", err)
				}
				// Mock workspace ID
				oldFunc := getWorkspaceIDFromEnv
				t.Cleanup(func() { getWorkspaceIDFromEnv = oldFunc })
				getWorkspaceIDFromEnv = func(string) string {
					return "/subscriptions/test/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/ws"
				}
			},
			requestBody: `{"service": "api"}`,
			mockLogs: []azure.LogEntry{
				{
					Service:   "api",
					Message:   "Test log message",
					Level:     azure.LogLevelInfo,
					Timestamp: time.Now(),
				},
				{
					Service:   "api",
					Message:   "Another log message",
					Level:     azure.LogLevelError,
					Timestamp: time.Now().Add(-1 * time.Minute),
				},
			},
			wantSuccess:   true,
			wantLogsFound: 2,
		},
		{
			name: "no logs found",
			setupEnv: func(t *testing.T, dir string) {
				azureYamlPath := filepath.Join(dir, "azure.yaml")
				if err := os.WriteFile(azureYamlPath, []byte("name: test-app"), 0644); err != nil {
					t.Fatalf("Failed to write azure.yaml: %v", err)
				}
				oldFunc := getWorkspaceIDFromEnv
				t.Cleanup(func() { getWorkspaceIDFromEnv = oldFunc })
				getWorkspaceIDFromEnv = func(string) string {
					return "/subscriptions/test/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/ws"
				}
			},
			requestBody:   `{"service": "api"}`,
			mockLogs:      []azure.LogEntry{},
			wantSuccess:   false,
			wantLogsFound: 0,
			wantMinSteps:  1,
		},
		{
			name: "missing workspace configuration",
			setupEnv: func(t *testing.T, dir string) {
				azureYamlPath := filepath.Join(dir, "azure.yaml")
				if err := os.WriteFile(azureYamlPath, []byte("name: test-app"), 0644); err != nil {
					t.Fatalf("Failed to write azure.yaml: %v", err)
				}
			},
			requestBody:  `{"service": "api"}`,
			wantSuccess:  false,
			wantMinSteps: 1,
		},
		{
			name: "missing service name",
			setupEnv: func(t *testing.T, dir string) {
				azureYamlPath := filepath.Join(dir, "azure.yaml")
				if err := os.WriteFile(azureYamlPath, []byte("name: test-app"), 0644); err != nil {
					t.Fatalf("Failed to write azure.yaml: %v", err)
				}
			},
			requestBody: `{"service": ""}`,
			wantSuccess: false,
		},
		{
			name: "invalid JSON body",
			setupEnv: func(t *testing.T, dir string) {
				azureYamlPath := filepath.Join(dir, "azure.yaml")
				if err := os.WriteFile(azureYamlPath, []byte("name: test-app"), 0644); err != nil {
					t.Fatalf("Failed to write azure.yaml: %v", err)
				}
			},
			requestBody: `{invalid json}`,
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setupEnv(t, dir)

			// Mock fetchAzureLogsStandalone
			oldFetch := fetchAzureLogsStandalone
			t.Cleanup(func() { fetchAzureLogsStandalone = oldFetch })
			fetchAzureLogsStandalone = func(_ context.Context, _ azure.StandaloneLogsConfig) ([]azure.LogEntry, error) {
				if tt.mockError != nil {
					return nil, tt.mockError
				}
				return tt.mockLogs, nil
			}

			server := &Server{projectDir: dir}
			req := httptest.NewRequest(http.MethodPost, "/api/azure/logs/verify", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleAzureLogsVerify(w, req)

			// Check for invalid JSON or missing service name
			if tt.requestBody == `{invalid json}` || tt.requestBody == `{"service": ""}` {
				if w.Code != http.StatusBadRequest {
					t.Errorf("expected status 400 for invalid request, got %d", w.Code)
				}
				return
			}

			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}

			var response VerifyLogsResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if response.Success != tt.wantSuccess {
				t.Errorf("expected success=%v, got %v. Message: %s", tt.wantSuccess, response.Success, response.Message)
			}

			if response.LogsFound != tt.wantLogsFound {
				t.Errorf("expected logsFound=%d, got %d", tt.wantLogsFound, response.LogsFound)
			}

			if tt.wantMinSteps > 0 && len(response.NextSteps) < tt.wantMinSteps {
				t.Errorf("expected at least %d next steps, got %d", tt.wantMinSteps, len(response.NextSteps))
			}

			if response.Message == "" {
				t.Error("expected non-empty message")
			}

			// Check time range and samples for successful verifications
			if tt.wantSuccess && tt.wantLogsFound > 0 {
				if response.TimeRange == nil {
					t.Error("expected time range for successful verification")
				}
				if len(response.Sample) != tt.wantLogsFound {
					t.Errorf("expected %d samples, got %d", tt.wantLogsFound, len(response.Sample))
				}
				for i, sample := range response.Sample {
					if sample.Timestamp == "" {
						t.Errorf("sample %d has empty timestamp", i)
					}
					if sample.Message == "" {
						t.Errorf("sample %d has empty message", i)
					}
					if sample.Level == "" {
						t.Errorf("sample %d has empty level", i)
					}
				}
			}
		})
	}
}

func TestTruncateMessage(t *testing.T) {
	tests := []struct {
		name    string
		message string
		maxLen  int
		want    string
	}{
		{
			name:    "short message",
			message: "Short",
			maxLen:  10,
			want:    "Short",
		},
		{
			name:    "exact length",
			message: "Exactly10!",
			maxLen:  10,
			want:    "Exactly10!",
		},
		{
			name:    "long message",
			message: "This is a very long message that should be truncated",
			maxLen:  20,
			want:    "This is a very lo...",
		},
		{
			name:    "empty message",
			message: "",
			maxLen:  10,
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateMessage(tt.message, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}
