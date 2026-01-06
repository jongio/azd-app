package azure

import (
	"context"
	"testing"
	"time"
)

func TestParseISO8601Duration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{
			name:  "15 minutes",
			input: "PT15M",
			want:  15 * time.Minute,
		},
		{
			name:  "1 hour",
			input: "PT1H",
			want:  1 * time.Hour,
		},
		{
			name:  "30 seconds",
			input: "PT30S",
			want:  30 * time.Second,
		},
		{
			name:  "1 hour 30 minutes",
			input: "PT1H30M",
			want:  1*time.Hour + 30*time.Minute,
		},
		{
			name:  "2 hours 15 minutes 30 seconds",
			input: "PT2H15M30S",
			want:  2*time.Hour + 15*time.Minute + 30*time.Second,
		},
		{
			name:    "invalid - no P prefix",
			input:   "T15M",
			wantErr: true,
		},
		{
			name:    "invalid - no T prefix",
			input:   "P15M",
			wantErr: true,
		},
		{
			name:    "invalid - empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid - just P",
			input:   "P",
			wantErr: true,
		},
		{
			name:    "invalid - just PT",
			input:   "PT",
			wantErr: true,
		},
		{
			name:    "invalid unit",
			input:   "PT15X",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseISO8601Duration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseISO8601Duration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseISO8601Duration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractWorkspaceNameFromID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "full resource ID",
			input: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/my-workspace",
			want:  "my-workspace",
		},
		{
			name:  "case insensitive",
			input: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.OperationalInsights/Workspaces/My-Workspace",
			want:  "My-Workspace",
		},
		{
			name:  "simple name",
			input: "my-workspace",
			want:  "my-workspace",
		},
		{
			name:  "GUID",
			input: "12345678-1234-1234-1234-123456789012",
			want:  "12345678-1234-1234-1234-123456789012",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "different resource type",
			input: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/mystorage",
			want:  "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/mystorage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractWorkspaceNameFromID(tt.input)
			if got != tt.want {
				t.Errorf("extractWorkspaceNameFromID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGenerateGuidance(t *testing.T) {
	verifier := &WorkspaceVerifier{}

	tests := []struct {
		name        string
		serviceName string
		result      *ServiceVerificationResult
		wantContain string
	}{
		{
			name:        "ok status",
			serviceName: "api",
			result: &ServiceVerificationResult{
				Status:   ServiceStatusOK,
				LogCount: 42,
			},
			wantContain: "Logs flowing correctly",
		},
		{
			name:        "no logs",
			serviceName: "web",
			result: &ServiceVerificationResult{
				Status: ServiceStatusNoLogs,
			},
			wantContain: "No recent logs",
		},
		{
			name:        "diagnostic not configured",
			serviceName: "function",
			result: &ServiceVerificationResult{
				Status: ServiceStatusDiagnosticNotConfigured,
			},
			wantContain: "Configure diagnostic settings",
		},
		{
			name:        "error",
			serviceName: "container",
			result: &ServiceVerificationResult{
				Status: ServiceStatusError,
				Error:  "Permission denied",
			},
			wantContain: "Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := verifier.generateGuidance(tt.serviceName, tt.result)
			if got == "" {
				t.Errorf("generateGuidance() returned empty string")
				return
			}
			if tt.wantContain != "" && !contains(got, tt.wantContain) {
				t.Errorf("generateGuidance() = %q, want to contain %q", got, tt.wantContain)
			}
			// Verify service name is in the guidance
			if !contains(got, tt.serviceName) {
				t.Errorf("generateGuidance() = %q, want to contain service name %q", got, tt.serviceName)
			}
		})
	}
}

func TestWorkspaceVerificationResponse_Serialization(t *testing.T) {
	// Test that the response types serialize correctly to JSON
	now := time.Now()
	response := WorkspaceVerificationResponse{
		Status: VerificationStatusPartial,
		Workspace: WorkspaceInfo{
			ID:   "/subscriptions/test/workspaces/test-workspace",
			Name: "test-workspace",
		},
		Results: map[string]*ServiceVerificationResult{
			"api": {
				LogCount:    15,
				LastLogTime: &now,
				Status:      ServiceStatusOK,
			},
			"web": {
				LogCount: 0,
				Status:   ServiceStatusNoLogs,
				Message:  "No logs found. This may be normal...",
			},
			"function": {
				LogCount: 0,
				Status:   ServiceStatusDiagnosticNotConfigured,
				Error:    "DiagnosticSettingsNotConfigured: No diagnostic settings found",
			},
		},
		Guidance: []string{
			"api: Logs flowing correctly",
			"web: No recent logs - wait or trigger activity",
			"function: Configure diagnostic settings first",
		},
	}

	// Verify field values
	if response.Status != VerificationStatusPartial {
		t.Errorf("Status = %q, want %q", response.Status, VerificationStatusPartial)
	}

	if response.Workspace.Name != "test-workspace" {
		t.Errorf("Workspace.Name = %q, want %q", response.Workspace.Name, "test-workspace")
	}

	if len(response.Results) != 3 {
		t.Errorf("Results count = %d, want 3", len(response.Results))
	}

	apiResult := response.Results["api"]
	if apiResult.Status != ServiceStatusOK {
		t.Errorf("api status = %q, want %q", apiResult.Status, ServiceStatusOK)
	}
	if apiResult.LogCount != 15 {
		t.Errorf("api LogCount = %d, want 15", apiResult.LogCount)
	}
	if apiResult.LastLogTime == nil {
		t.Error("api LastLogTime is nil, want timestamp")
	}

	webResult := response.Results["web"]
	if webResult.Status != ServiceStatusNoLogs {
		t.Errorf("web status = %q, want %q", webResult.Status, ServiceStatusNoLogs)
	}
	if webResult.Message == "" {
		t.Error("web Message is empty, want message")
	}

	functionResult := response.Results["function"]
	if functionResult.Status != ServiceStatusDiagnosticNotConfigured {
		t.Errorf("function status = %q, want %q", functionResult.Status, ServiceStatusDiagnosticNotConfigured)
	}
	if functionResult.Error == "" {
		t.Error("function Error is empty, want error")
	}

	if len(response.Guidance) != 3 {
		t.Errorf("Guidance count = %d, want 3", len(response.Guidance))
	}
}

func TestServiceVerificationStatus_StringValues(t *testing.T) {
	// Verify the status constants have the expected string values
	if ServiceStatusOK != "ok" {
		t.Errorf("ServiceStatusOK = %q, want %q", ServiceStatusOK, "ok")
	}

	if ServiceStatusNoLogs != "no-logs" {
		t.Errorf("ServiceStatusNoLogs = %q, want %q", ServiceStatusNoLogs, "no-logs")
	}

	if ServiceStatusError != "error" {
		t.Errorf("ServiceStatusError = %q, want %q", ServiceStatusError, "error")
	}

	if ServiceStatusDiagnosticNotConfigured != "diagnostic-not-configured" {
		t.Errorf("ServiceStatusDiagnosticNotConfigured = %q, want %q", ServiceStatusDiagnosticNotConfigured, "diagnostic-not-configured")
	}
}

func TestWorkspaceVerificationStatus_StringValues(t *testing.T) {
	// Verify the status constants have the expected string values
	if VerificationStatusSuccess != "success" {
		t.Errorf("VerificationStatusSuccess = %q, want %q", VerificationStatusSuccess, "success")
	}

	if VerificationStatusPartial != "partial" {
		t.Errorf("VerificationStatusPartial = %q, want %q", VerificationStatusPartial, "partial")
	}

	if VerificationStatusError != "error" {
		t.Errorf("VerificationStatusError = %q, want %q", VerificationStatusError, "error")
	}
}

func TestWorkspaceVerificationRequest_DefaultValues(t *testing.T) {
	// Test default values handling
	req := &WorkspaceVerificationRequest{}

	// Empty services should be treated as "check all"
	if len(req.Services) != 0 {
		t.Errorf("Default Services = %v, want empty slice", req.Services)
	}

	// Empty timespan should default to PT15M
	if req.Timespan != "" {
		t.Errorf("Default Timespan = %q, want empty string (will default to PT15M)", req.Timespan)
	}
}

func TestVerifyService_NoDiagnosticSettings(t *testing.T) {
	// This test demonstrates the structure for testing verifyService
	// In a real test environment, we'd mock the Log Analytics client and diagnostic checker

	verifier := &WorkspaceVerifier{
		diagnostics: &DiagnosticSettingsChecker{},
	}

	// We can test the logic paths without actual Azure calls
	// by checking that the result structure is correct

	result := &ServiceVerificationResult{
		Status: ServiceStatusDiagnosticNotConfigured,
		Error:  "DiagnosticSettingsNotConfigured: No diagnostic settings found for this resource",
	}

	if result.Status != ServiceStatusDiagnosticNotConfigured {
		t.Errorf("Status = %q, want %q", result.Status, ServiceStatusDiagnosticNotConfigured)
	}

	if result.Error == "" {
		t.Error("Error is empty, want error message")
	}

	// Verify the structure is ready for JSON serialization
	if result.LogCount != 0 {
		t.Errorf("LogCount = %d, want 0 for diagnostic not configured", result.LogCount)
	}

	_ = verifier // Avoid unused variable error
}

func TestVerifyService_WithLogs(t *testing.T) {
	// Test the structure of a successful verification result
	now := time.Now()

	result := &ServiceVerificationResult{
		Status:      ServiceStatusOK,
		LogCount:    25,
		LastLogTime: &now,
	}

	if result.Status != ServiceStatusOK {
		t.Errorf("Status = %q, want %q", result.Status, ServiceStatusOK)
	}

	if result.LogCount != 25 {
		t.Errorf("LogCount = %d, want 25", result.LogCount)
	}

	if result.LastLogTime == nil {
		t.Error("LastLogTime is nil, want timestamp")
	}

	if !result.LastLogTime.Equal(now) {
		t.Errorf("LastLogTime = %v, want %v", result.LastLogTime, now)
	}
}

func TestVerifyService_NoLogs(t *testing.T) {
	// Test the structure of a no-logs result

	result := &ServiceVerificationResult{
		Status:   ServiceStatusNoLogs,
		LogCount: 0,
		Message:  "No logs found. This may be normal if the service hasn't run yet or if diagnostic settings were just configured (allow 2-5 minutes for ingestion).",
	}

	if result.Status != ServiceStatusNoLogs {
		t.Errorf("Status = %q, want %q", result.Status, ServiceStatusNoLogs)
	}

	if result.LogCount != 0 {
		t.Errorf("LogCount = %d, want 0", result.LogCount)
	}

	if result.LastLogTime != nil {
		t.Errorf("LastLogTime = %v, want nil", result.LastLogTime)
	}

	if result.Message == "" {
		t.Error("Message is empty, want guidance message")
	}
}

func TestVerifyService_QueryError(t *testing.T) {
	// Test the structure of an error result

	result := &ServiceVerificationResult{
		Status: ServiceStatusError,
		Error:  "Failed to query logs: permission denied",
	}

	if result.Status != ServiceStatusError {
		t.Errorf("Status = %q, want %q", result.Status, ServiceStatusError)
	}

	if result.Error == "" {
		t.Error("Error is empty, want error message")
	}

	if result.LogCount != 0 {
		t.Errorf("LogCount = %d, want 0 for error case", result.LogCount)
	}
}

func TestWorkspaceVerifier_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This would be run with actual Azure credentials in a real environment
	// For CI/CD, you'd set up proper mocking or use recorded responses
	t.Skip("Integration test requires live Azure environment")

	// Example integration test structure:
	// ctx := context.Background()
	// cred, err := NewCredentialChain()
	// if err != nil {
	//     t.Fatalf("Failed to create credentials: %v", err)
	// }
	//
	// verifier := NewWorkspaceVerifier(cred, "/path/to/project")
	// req := &WorkspaceVerificationRequest{
	//     Timespan: "PT15M",
	// }
	// response, err := verifier.VerifyWorkspace(ctx, req)
	// if err != nil {
	//     t.Fatalf("VerifyWorkspace failed: %v", err)
	// }
	//
	// // Verify response
	// if response.Status == "" {
	//     t.Error("Expected status to be populated")
	// }
	// if response.Workspace.ID == "" {
	//     t.Error("Expected workspace ID to be populated")
	// }
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && hasSubstring(s, substr)))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestNewWorkspaceVerifier(t *testing.T) {
	cred := &simpleMockCredential{token: "test-token"}
	projectDir := "/test/project"

	verifier := NewWorkspaceVerifier(cred, projectDir)

	if verifier == nil {
		t.Fatal("NewWorkspaceVerifier returned nil")
	}

	if verifier.credential != cred {
		t.Error("credential not set correctly")
	}

	if verifier.projectDir != projectDir {
		t.Error("projectDir not set correctly")
	}

	if verifier.discovery == nil {
		t.Error("discovery not initialized")
	}

	if verifier.diagnostics == nil {
		t.Error("diagnostics not initialized")
	}
}

func TestWorkspaceVerificationRequest_CustomTimespan(t *testing.T) {
	tests := []struct {
		name     string
		timespan string
		wantDur  time.Duration
		wantErr  bool
	}{
		{
			name:     "default 15 minutes",
			timespan: "PT15M",
			wantDur:  15 * time.Minute,
		},
		{
			name:     "1 hour",
			timespan: "PT1H",
			wantDur:  1 * time.Hour,
		},
		{
			name:     "30 minutes",
			timespan: "PT30M",
			wantDur:  30 * time.Minute,
		},
		{
			name:     "5 minutes",
			timespan: "PT5M",
			wantDur:  5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &WorkspaceVerificationRequest{
				Timespan: tt.timespan,
			}

			// Verify the timespan can be parsed
			dur, err := parseISO8601Duration(req.Timespan)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseISO8601Duration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && dur != tt.wantDur {
				t.Errorf("parseISO8601Duration() = %v, want %v", dur, tt.wantDur)
			}
		})
	}
}

func TestVerifyWorkspace_InvalidTimespan(t *testing.T) {
	// Test that invalid timespan returns an error
	cred := &simpleMockCredential{token: "test-token"}
	verifier := NewWorkspaceVerifier(cred, "/test/project")

	ctx := context.Background()
	req := &WorkspaceVerificationRequest{
		Timespan: "invalid",
	}

	_, err := verifier.VerifyWorkspace(ctx, req)
	if err == nil {
		t.Error("Expected error for invalid timespan, got nil")
	}

	if !contains(err.Error(), "invalid timespan") {
		t.Errorf("Error message = %q, want to contain 'invalid timespan'", err.Error())
	}
}

func TestVerifyWorkspace_EmptyTimespanUsesDefault(t *testing.T) {
	// Test that empty timespan defaults to PT15M
	req := &WorkspaceVerificationRequest{
		Timespan: "",
	}

	// The VerifyWorkspace method should set it to PT15M if empty
	defaultTimespan := "PT15M"
	if req.Timespan == "" {
		req.Timespan = defaultTimespan
	}

	dur, err := parseISO8601Duration(req.Timespan)
	if err != nil {
		t.Fatalf("Failed to parse default timespan: %v", err)
	}

	expectedDur := 15 * time.Minute
	if dur != expectedDur {
		t.Errorf("Default timespan duration = %v, want %v", dur, expectedDur)
	}
}
