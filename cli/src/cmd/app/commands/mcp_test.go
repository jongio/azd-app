package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/security"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/require"
)

func TestNewMCPCommand(t *testing.T) {
	cmd := NewMCPCommand()

	if cmd == nil {
		t.Fatal("NewMCPCommand returned nil")
	}

	if cmd.Use != "mcp" {
		t.Errorf("Expected command use 'mcp', got '%s'", cmd.Use)
	}

	if !cmd.Hidden {
		t.Error("MCP command should be hidden")
	}

	// Check for serve subcommand
	serveCmd := cmd.Commands()
	if len(serveCmd) == 0 {
		t.Fatal("MCP command should have subcommands")
	}

	foundServe := false
	for _, c := range serveCmd {
		if c.Use == "serve" {
			foundServe = true
			break
		}
	}

	if !foundServe {
		t.Error("MCP command should have 'serve' subcommand")
	}
}

func TestNewMCPServeCommand(t *testing.T) {
	cmd := newMCPServeCommand()

	if cmd == nil {
		t.Fatal("newMCPServeCommand returned nil")
	}

	if cmd.Use != "serve" {
		t.Errorf("Expected command use 'serve', got '%s'", cmd.Use)
	}

	if cmd.RunE == nil {
		t.Error("serve command should have RunE function")
	}
}

func TestGetServicesToolDefinition(t *testing.T) {
	tool := newGetServicesTool()

	if tool.Tool.Name != "get_services" {
		t.Errorf("Expected tool name 'get_services', got '%s'", tool.Tool.Name)
	}

	if tool.Handler == nil {
		t.Error("get_services tool should have a handler")
	}

	// Verify tool metadata
	if tool.Tool.Description == "" {
		t.Error("get_services tool should have a description")
	}

	// Verify tool has title annotation (MCP spec compliance)
	if tool.Tool.Annotations.Title == "" {
		t.Error("get_services tool should have a title annotation")
	}
}

func TestGetServiceLogsToolDefinition(t *testing.T) {
	tool := newGetServiceLogsTool()

	if tool.Tool.Name != "get_service_logs" {
		t.Errorf("Expected tool name 'get_service_logs', got '%s'", tool.Tool.Name)
	}

	if tool.Handler == nil {
		t.Error("get_service_logs tool should have a handler")
	}

	if tool.Tool.Description == "" {
		t.Error("get_service_logs tool should have a description")
	}
}

func TestGetProjectInfoToolDefinition(t *testing.T) {
	tool := newGetProjectInfoTool()

	if tool.Tool.Name != "get_project_info" {
		t.Errorf("Expected tool name 'get_project_info', got '%s'", tool.Tool.Name)
	}

	if tool.Handler == nil {
		t.Error("get_project_info tool should have a handler")
	}

	if tool.Tool.Description == "" {
		t.Error("get_project_info tool should have a description")
	}
}

func TestGetServicesToolHandlerBehavior(t *testing.T) {
	tool := newGetServicesTool()
	ctx := context.Background()

	// Test with empty arguments
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_services",
			Arguments: map[string]interface{}{},
		},
	}

	result, err := tool.Handler(ctx, request)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	if result == nil {
		t.Fatal("Handler returned nil result")
	}

	// Result should have content
	if len(result.Content) == 0 {
		t.Error("Handler result should have content")
	}
}

func TestGetServiceLogsToolHandlerBehavior(t *testing.T) {
	tool := newGetServiceLogsTool()
	ctx := context.Background()

	// Test with tail parameter
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "get_service_logs",
			Arguments: map[string]interface{}{
				"tail": float64(10),
			},
		},
	}

	result, err := tool.Handler(ctx, request)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	if result == nil {
		t.Fatal("Handler returned nil result")
	}

	// Result should have content
	if len(result.Content) == 0 {
		t.Error("Handler result should have content")
	}
}

func TestGetProjectInfoToolHandlerBehavior(t *testing.T) {
	tool := newGetProjectInfoTool()
	ctx := context.Background()

	// Test with empty arguments
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_project_info",
			Arguments: map[string]interface{}{},
		},
	}

	result, err := tool.Handler(ctx, request)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	if result == nil {
		t.Fatal("Handler returned nil result")
	}

	// Result should have content
	if len(result.Content) == 0 {
		t.Error("Handler result should have content")
	}
}

// TestMCPToolsNoDuplication verifies that our tools don't duplicate azd's MCP functionality
func TestMCPToolsNoDuplication(t *testing.T) {
	// azd's MCP tools are focused on:
	// - architecture_planning
	// - azure_yaml_generation
	// - discovery_analysis
	// - docker_generation
	// - error_troubleshooting
	// - iac_generation_rules
	// - infrastructure_generation
	// - plan_init
	// - project_validation
	// - yaml_schema
	//
	// Our tools focus on runtime observability:
	// - get_services (runtime service status)
	// - get_service_logs (live application logs)
	// - get_project_info (project metadata)
	//
	// These are complementary, not duplicative.

	ourTools := []string{"get_services", "get_service_logs", "get_project_info"}
	azdTools := []string{
		"architecture_planning",
		"azure_yaml_generation",
		"discovery_analysis",
		"docker_generation",
		"error_troubleshooting",
		"iac_generation_rules",
		"infrastructure_generation",
		"plan_init",
		"project_validation",
		"yaml_schema",
	}

	for _, ourTool := range ourTools {
		for _, azdTool := range azdTools {
			if ourTool == azdTool {
				t.Errorf("Tool '%s' duplicates azd MCP functionality", ourTool)
			}
		}
	}
}

func TestRunServicesToolDefinition(t *testing.T) {
	tool := newRunServicesTool()

	if tool.Tool.Name != "run_services" {
		t.Errorf("Expected tool name 'run_services', got '%s'", tool.Tool.Name)
	}

	if tool.Handler == nil {
		t.Error("run_services tool should have a handler")
	}

	if tool.Tool.Description == "" {
		t.Error("run_services tool should have a description")
	}
}

func TestInstallDependenciesToolDefinition(t *testing.T) {
	tool := newInstallDependenciesTool()

	if tool.Tool.Name != "install_dependencies" {
		t.Errorf("Expected tool name 'install_dependencies', got '%s'", tool.Tool.Name)
	}

	if tool.Handler == nil {
		t.Error("install_dependencies tool should have a handler")
	}

	if tool.Tool.Description == "" {
		t.Error("install_dependencies tool should have a description")
	}
}

func TestCheckRequirementsToolDefinition(t *testing.T) {
	tool := newCheckRequirementsTool()

	if tool.Tool.Name != "check_requirements" {
		t.Errorf("Expected tool name 'check_requirements', got '%s'", tool.Tool.Name)
	}

	if tool.Handler == nil {
		t.Error("check_requirements tool should have a handler")
	}

	if tool.Tool.Description == "" {
		t.Error("check_requirements tool should have a description")
	}
}

func TestStopServicesToolDefinition(t *testing.T) {
	tool := newStopServicesTool()

	if tool.Tool.Name != "stop_services" {
		t.Errorf("Expected tool name 'stop_services', got '%s'", tool.Tool.Name)
	}

	if tool.Handler == nil {
		t.Error("stop_services tool should have a handler")
	}
}

func TestRestartServiceToolDefinition(t *testing.T) {
	tool := newRestartServiceTool()

	if tool.Tool.Name != "restart_service" {
		t.Errorf("Expected tool name 'restart_service', got '%s'", tool.Tool.Name)
	}

	if tool.Handler == nil {
		t.Error("restart_service tool should have a handler")
	}
}

func TestGetEnvironmentVariablesToolDefinition(t *testing.T) {
	tool := newGetEnvironmentVariablesTool()

	if tool.Tool.Name != "get_environment_variables" {
		t.Errorf("Expected tool name 'get_environment_variables', got '%s'", tool.Tool.Name)
	}

	if tool.Handler == nil {
		t.Error("get_environment_variables tool should have a handler")
	}
}

func TestSetEnvironmentVariableToolDefinition(t *testing.T) {
	tool := newSetEnvironmentVariableTool()

	if tool.Tool.Name != "set_environment_variable" {
		t.Errorf("Expected tool name 'set_environment_variable', got '%s'", tool.Tool.Name)
	}

	if tool.Handler == nil {
		t.Error("set_environment_variable tool should have a handler")
	}
}

func TestAzureYamlResourceDefinition(t *testing.T) {
	resource := newAzureYamlResource()

	if resource.Resource.Name != "azure.yaml" {
		t.Errorf("Expected resource name 'azure.yaml', got '%s'", resource.Resource.Name)
	}

	if resource.Handler == nil {
		t.Error("azure.yaml resource should have a handler")
	}
}

func TestServiceConfigResourceDefinition(t *testing.T) {
	resource := newServiceConfigResource()

	if resource.Resource.Name != "service-configs" {
		t.Errorf("Expected resource name 'service-configs', got '%s'", resource.Resource.Name)
	}

	if resource.Handler == nil {
		t.Error("service-configs resource should have a handler")
	}
}

// Tests for helper functions

func TestGetStringParam(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]interface{}
		key      string
		expected string
		found    bool
	}{
		{
			name:     "Valid string parameter",
			args:     map[string]interface{}{"key": "value"},
			key:      "key",
			expected: "value",
			found:    true,
		},
		{
			name:     "Empty string parameter",
			args:     map[string]interface{}{"key": ""},
			key:      "key",
			expected: "",
			found:    false,
		},
		{
			name:     "Missing parameter",
			args:     map[string]interface{}{},
			key:      "key",
			expected: "",
			found:    false,
		},
		{
			name:     "Wrong type parameter",
			args:     map[string]interface{}{"key": 123},
			key:      "key",
			expected: "",
			found:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, found := getStringParam(tt.args, tt.key)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
			if found != tt.found {
				t.Errorf("Expected found=%v, got %v", tt.found, found)
			}
		})
	}
}

func TestGetFloat64Param(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]interface{}
		key      string
		expected float64
		found    bool
	}{
		{
			name:     "Valid float64 parameter",
			args:     map[string]interface{}{"key": float64(42)},
			key:      "key",
			expected: 42.0,
			found:    true,
		},
		{
			name:     "Missing parameter",
			args:     map[string]interface{}{},
			key:      "key",
			expected: 0,
			found:    false,
		},
		{
			name:     "Wrong type parameter",
			args:     map[string]interface{}{"key": "not a number"},
			key:      "key",
			expected: 0,
			found:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, found := getFloat64Param(tt.args, tt.key)
			if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
			if found != tt.found {
				t.Errorf("Expected found=%v, got %v", tt.found, found)
			}
		})
	}
}

func TestMarshalToolResult(t *testing.T) {
	tests := []struct {
		name      string
		data      interface{}
		wantError bool
	}{
		{
			name:      "Valid map",
			data:      map[string]interface{}{"key": "value"},
			wantError: false,
		},
		{
			name:      "Valid slice",
			data:      []string{"a", "b", "c"},
			wantError: false,
		},
		{
			name:      "Empty map",
			data:      map[string]interface{}{},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := marshalToolResult(tt.data)
			if tt.wantError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantError && result == nil {
				t.Error("Expected result, got nil")
			}
		})
	}
}

func TestExtractProjectDirArg(t *testing.T) {
	// Create a temp directory for testing
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		args      map[string]interface{}
		wantLen   int
		wantError bool
	}{
		{
			name:      "With valid project dir",
			args:      map[string]interface{}{"projectDir": tempDir},
			wantLen:   2, // --cwd and the path
			wantError: false,
		},
		{
			name:      "Without project dir",
			args:      map[string]interface{}{},
			wantLen:   0,
			wantError: false,
		},
		{
			name:      "Empty project dir",
			args:      map[string]interface{}{"projectDir": ""},
			wantLen:   0,
			wantError: false,
		},
		{
			name:      "With non-existent project dir",
			args:      map[string]interface{}{"projectDir": "/nonexistent/path/that/does/not/exist"},
			wantLen:   0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractProjectDirArg(tt.args)
			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if len(result) != tt.wantLen {
					t.Errorf("Expected %d args, got %d", tt.wantLen, len(result))
				}
			}
		})
	}
}

func TestValidateServiceName(t *testing.T) {
	tests := []struct {
		name       string
		service    string
		allowEmpty bool
		wantError  bool
	}{
		{
			name:       "Valid service name",
			service:    "my-service",
			allowEmpty: false,
			wantError:  false,
		},
		{
			name:       "Valid with underscore",
			service:    "my_service_123",
			allowEmpty: false,
			wantError:  false,
		},
		{
			name:       "Empty string allowed",
			service:    "",
			allowEmpty: true,
			wantError:  false,
		},
		{
			name:       "Empty string not allowed",
			service:    "",
			allowEmpty: false,
			wantError:  true,
		},
		{
			name:       "Invalid - starts with hyphen",
			service:    "-service",
			allowEmpty: false,
			wantError:  true,
		},
		{
			name:       "Invalid - contains special chars",
			service:    "service; rm -rf /",
			allowEmpty: false,
			wantError:  true,
		},
		{
			name:       "Invalid - too long",
			service:    string(make([]byte, 100)),
			allowEmpty: false,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := security.ValidateServiceName(tt.service, tt.allowEmpty)
			if tt.wantError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestValidateProjectDir(t *testing.T) {
	// Create a temp directory and file for testing
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "testfile.txt")
	err := os.WriteFile(tempFile, []byte("test"), 0644)
	require.NoError(t, err)

	tests := []struct {
		name      string
		dir       string
		wantError bool
	}{
		{
			name:      "Valid directory",
			dir:       tempDir,
			wantError: false,
		},
		{
			name:      "Current directory",
			dir:       ".",
			wantError: false,
		},
		{
			name:      "Empty string",
			dir:       "",
			wantError: false,
		},
		{
			name:      "Non-existent directory",
			dir:       "/nonexistent/path/that/does/not/exist",
			wantError: true,
		},
		{
			name:      "Path is a file not directory",
			dir:       tempFile,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateProjectDir(tt.dir)
			if tt.wantError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestIsValidDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration string
		want     bool
	}{
		{"Valid seconds", "30s", true},
		{"Valid minutes", "5m", true},
		{"Valid hours", "1h", true},
		{"Empty string", "", false},
		{"Missing unit", "30", false},
		{"Invalid unit", "30x", false},
		{"No number", "s", false},
		{"Negative", "-5m", false},
		{"Float", "1.5m", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidDuration(tt.duration); got != tt.want {
				t.Errorf("isValidDuration(%q) = %v, want %v", tt.duration, got, tt.want)
			}
		})
	}
}

func TestGetArgsMap(t *testing.T) {
	tests := []struct {
		name    string
		request mcp.CallToolRequest
		wantLen int
	}{
		{
			name: "Valid arguments map",
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "test",
					Arguments: map[string]interface{}{"key": "value"},
				},
			},
			wantLen: 1,
		},
		{
			name: "Nil arguments",
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "test",
					Arguments: nil,
				},
			},
			wantLen: 0,
		},
		{
			name: "Wrong type arguments",
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "test",
					Arguments: "not a map",
				},
			},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getArgsMap(tt.request)
			if len(result) != tt.wantLen {
				t.Errorf("Expected %d keys, got %d", tt.wantLen, len(result))
			}
		})
	}
}

func TestValidateRequiredParam(t *testing.T) {
	tests := []struct {
		name          string
		args          map[string]interface{}
		key           string
		expectedValue string
		wantError     bool
	}{
		{
			name:          "Valid required parameter",
			args:          map[string]interface{}{"key": "value"},
			key:           "key",
			expectedValue: "value",
			wantError:     false,
		},
		{
			name:          "Missing required parameter",
			args:          map[string]interface{}{},
			key:           "key",
			expectedValue: "",
			wantError:     true,
		},
		{
			name:          "Empty required parameter",
			args:          map[string]interface{}{"key": ""},
			key:           "key",
			expectedValue: "",
			wantError:     true,
		},
		{
			name:          "Wrong type parameter",
			args:          map[string]interface{}{"key": 123},
			key:           "key",
			expectedValue: "",
			wantError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := validateRequiredParam(tt.args, tt.key)
			if tt.wantError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			if !tt.wantError && val != tt.expectedValue {
				t.Errorf("Expected value '%s', got '%s'", tt.expectedValue, val)
			}
		})
	}
}

func TestValidateEnumParam(t *testing.T) {
	allowed := map[string]bool{"a": true, "b": true, "c": true}

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{
			name:      "Valid value",
			value:     "a",
			wantError: false,
		},
		{
			name:      "Empty value (optional)",
			value:     "",
			wantError: false,
		},
		{
			name:      "Invalid value",
			value:     "invalid",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEnumParam(tt.value, allowed, "test")
			if tt.wantError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

// Tests for tool handlers with mock data

func TestGetServicesToolHandlerWithParams(t *testing.T) {
	tool := newGetServicesTool()
	ctx := context.Background()

	tests := []struct {
		name string
		args map[string]interface{}
	}{
		{
			name: "With project dir",
			args: map[string]interface{}{"projectDir": "/test/project"},
		},
		{
			name: "Without project dir",
			args: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "get_services",
					Arguments: tt.args,
				},
			}

			result, err := tool.Handler(ctx, request)

			// Handler should return error result, not Go error
			if err != nil {
				t.Errorf("Handler returned Go error (should use mcp.NewToolResultError): %v", err)
			}

			if result == nil {
				t.Fatal("Handler returned nil result")
			}
		})
	}
}

func TestGetServiceLogsToolHandlerWithParams(t *testing.T) {
	tool := newGetServiceLogsTool()
	ctx := context.Background()

	tests := []struct {
		name string
		args map[string]interface{}
	}{
		{
			name: "With service name",
			args: map[string]interface{}{"serviceName": "api"},
		},
		{
			name: "With tail parameter",
			args: map[string]interface{}{"tail": float64(50)},
		},
		{
			name: "With level parameter",
			args: map[string]interface{}{"level": "error"},
		},
		{
			name: "With since parameter",
			args: map[string]interface{}{"since": "5m"},
		},
		{
			name: "No parameters",
			args: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "get_service_logs",
					Arguments: tt.args,
				},
			}

			result, err := tool.Handler(ctx, request)

			if err != nil {
				t.Errorf("Handler returned Go error: %v", err)
			}

			if result == nil {
				t.Fatal("Handler returned nil result")
			}
		})
	}
}

func TestGetProjectInfoToolHandlerWithParams(t *testing.T) {
	tool := newGetProjectInfoTool()
	ctx := context.Background()

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_project_info",
			Arguments: map[string]interface{}{},
		},
	}

	result, err := tool.Handler(ctx, request)

	if err != nil {
		t.Errorf("Handler returned Go error: %v", err)
	}

	if result == nil {
		t.Fatal("Handler returned nil result")
	}
}

func TestRestartServiceToolHandler(t *testing.T) {
	tool := newRestartServiceTool()
	ctx := context.Background()

	tests := []struct {
		name        string
		args        map[string]interface{}
		expectError bool
	}{
		{
			name:        "With service name",
			args:        map[string]interface{}{"serviceName": "api"},
			expectError: false,
		},
		{
			name:        "Without service name (should show guidance)",
			args:        map[string]interface{}{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "restart_service",
					Arguments: tt.args,
				},
			}

			result, err := tool.Handler(ctx, request)

			if err != nil {
				t.Errorf("Handler returned Go error: %v", err)
			}

			if result == nil {
				t.Fatal("Handler returned nil result")
			}
		})
	}
}

func TestGetProjectDir(t *testing.T) {
	// Save original value
	originalProjectDir := os.Getenv("AZD_APP_PROJECT_DIR")
	defer func() {
		if originalProjectDir != "" {
			os.Setenv("AZD_APP_PROJECT_DIR", originalProjectDir)
		} else {
			os.Unsetenv("AZD_APP_PROJECT_DIR")
		}
	}()

	tests := []struct {
		name     string
		envValue string
		expected string
	}{
		{
			name:     "With AZD_APP_PROJECT_DIR set",
			envValue: "/custom/project/path",
			expected: "/custom/project/path",
		},
		{
			name:     "Without AZD_APP_PROJECT_DIR set",
			envValue: "",
			expected: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("AZD_APP_PROJECT_DIR", tt.envValue)
			} else {
				os.Unsetenv("AZD_APP_PROJECT_DIR")
			}

			result := getProjectDir()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestAllToolsHaveTitles verifies that all MCP tools have title annotations (MCP spec compliance)
func TestAllToolsHaveTitles(t *testing.T) {
	tools := []struct {
		name     string
		tool     func() server.ServerTool
		expected string
	}{
		{"get_services", newGetServicesTool, "Get Running Services"},
		{"get_service_logs", newGetServiceLogsTool, "Get Service Logs"},
		{"get_project_info", newGetProjectInfoTool, "Get Project Information"},
		{"run_services", newRunServicesTool, "Run Development Services"},
		{"stop_services", newStopServicesTool, "Stop Running Services"},
		{"restart_service", newRestartServiceTool, "Restart Service"},
		{"install_dependencies", newInstallDependenciesTool, "Install Project Dependencies"},
		{"check_requirements", newCheckRequirementsTool, "Check Prerequisites"},
		{"get_environment_variables", newGetEnvironmentVariablesTool, "Get Environment Variables"},
		{"set_environment_variable", newSetEnvironmentVariableTool, "Set Environment Variable"},
	}

	for _, tt := range tools {
		t.Run(tt.name, func(t *testing.T) {
			tool := tt.tool()
			if tool.Tool.Annotations.Title != tt.expected {
				t.Errorf("Tool %s: expected title '%s', got '%s'", tt.name, tt.expected, tool.Tool.Annotations.Title)
			}
		})
	}
}

// TestResourcesHaveAnnotations verifies that MCP resources have proper annotations
func TestResourcesHaveAnnotations(t *testing.T) {
	t.Run("azure.yaml resource", func(t *testing.T) {
		resource := newAzureYamlResource()

		if resource.Resource.Name != "azure.yaml" {
			t.Errorf("Expected resource name 'azure.yaml', got '%s'", resource.Resource.Name)
		}

		// Verify annotations exist
		if resource.Resource.Annotations == nil {
			t.Error("azure.yaml resource should have annotations")
			return
		}

		// Verify audience includes both user and assistant
		if len(resource.Resource.Annotations.Audience) != 2 {
			t.Errorf("Expected 2 audience roles, got %d", len(resource.Resource.Annotations.Audience))
		}

		// Verify priority is set (0.9 for high importance)
		if resource.Resource.Annotations.Priority != 0.9 {
			t.Errorf("azure.yaml resource should have priority 0.9, got %f", resource.Resource.Annotations.Priority)
		}
	})

	t.Run("service-configs resource", func(t *testing.T) {
		resource := newServiceConfigResource()

		if resource.Resource.Name != "service-configs" {
			t.Errorf("Expected resource name 'service-configs', got '%s'", resource.Resource.Name)
		}

		// Verify annotations exist
		if resource.Resource.Annotations == nil {
			t.Error("service-configs resource should have annotations")
			return
		}

		// Verify audience includes assistant
		if len(resource.Resource.Annotations.Audience) != 1 {
			t.Errorf("Expected 1 audience role, got %d", len(resource.Resource.Annotations.Audience))
		}

		// Verify priority is set (0.7 for medium importance)
		if resource.Resource.Annotations.Priority != 0.7 {
			t.Errorf("service-configs resource should have priority 0.7, got %f", resource.Resource.Annotations.Priority)
		}
	})
}
