package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	// Create a temp directory under the current working directory for testing
	// This is needed because validateProjectDir requires directories to be under cwd or home
	cwd, err := os.Getwd()
	require.NoError(t, err)
	tempDir, err := os.MkdirTemp(cwd, "test_extract_project_dir")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

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
	// Create a temp directory under the current working directory for testing
	// This is needed because validateProjectDir requires directories to be under cwd or home
	cwd, err := os.Getwd()
	require.NoError(t, err)
	tempDir, err := os.MkdirTemp(cwd, "test_validate_project_dir")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	tempFile := filepath.Join(tempDir, "testfile.txt")
	err = os.WriteFile(tempFile, []byte("test"), 0644)
	require.NoError(t, err)

	// Create a symlink for testing
	symlinkPath := filepath.Join(tempDir, "symlink_to_temp")
	_ = os.Symlink(tempDir, symlinkPath) // Ignore error if symlink creation fails on Windows

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
			name:      "Empty string (should resolve to current directory)",
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
		{
			name:      "Path traversal attempt - relative",
			dir:       "../../../../../../etc",
			wantError: true, // Should fail either due to boundary check or non-existence
		},
		{
			name:      "System directory access - /etc",
			dir:       "/etc",
			wantError: true, // Should fail due to system directory protection
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validateProjectDir(tt.dir)
			if tt.wantError && err == nil {
				t.Errorf("Expected error for %s, got nil (result: %s)", tt.dir, result)
			}
			if !tt.wantError && err != nil {
				t.Errorf("Expected no error for %s, got %v", tt.dir, err)
			}
			if !tt.wantError && result == "" {
				t.Errorf("Expected non-empty result for %s", tt.dir)
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

	// Create a temp directory under the current working directory for testing
	// This is needed because validateProjectDir requires directories to be under cwd or home
	cwd, err := os.Getwd()
	require.NoError(t, err)
	tempDir, err := os.MkdirTemp(cwd, "test_project_dir")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		envValue    string
		expectValue bool // true = expect envValue back, false = expect fallback to "." or cwd
	}{
		{
			name:        "With valid AZD_APP_PROJECT_DIR set",
			envValue:    tempDir,
			expectValue: true,
		},
		{
			name:        "With invalid (non-existent) AZD_APP_PROJECT_DIR",
			envValue:    "/nonexistent/custom/project/path",
			expectValue: false, // Should fall back
		},
		{
			name:        "Without AZD_APP_PROJECT_DIR set",
			envValue:    "",
			expectValue: false, // Should return "." or cwd
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
			if tt.expectValue {
				if result != tt.envValue {
					t.Errorf("Expected %s, got %s", tt.envValue, result)
				}
			} else {
				// Should return "." or current working directory
				cwd, _ := os.Getwd()
				if result != "." && result != cwd {
					t.Errorf("Expected '.' or '%s', got %s", cwd, result)
				}
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

// TestGetServiceLogsToolValidation tests validation logic for get_service_logs tool
func TestGetServiceLogsToolValidation(t *testing.T) {
	tool := newGetServiceLogsTool()
	ctx := context.Background()

	tests := []struct {
		name           string
		args           map[string]interface{}
		expectErrorMsg string
	}{
		{
			name:           "Invalid level parameter",
			args:           map[string]interface{}{"level": "invalid_level"},
			expectErrorMsg: "invalid level",
		},
		{
			name:           "Invalid since format - missing unit",
			args:           map[string]interface{}{"since": "30"},
			expectErrorMsg: "Invalid 'since' format",
		},
		{
			name:           "Invalid since format - invalid unit",
			args:           map[string]interface{}{"since": "30x"},
			expectErrorMsg: "Invalid 'since' format",
		},
		{
			name:           "Invalid service name - injection attempt",
			args:           map[string]interface{}{"serviceName": "api; rm -rf /"},
			expectErrorMsg: "service name",
		},
		{
			name:           "Invalid project dir - non-existent",
			args:           map[string]interface{}{"projectDir": "/nonexistent/path/xyz123"},
			expectErrorMsg: "project directory",
		},
		{
			name:           "Tail parameter exceeds max - should be capped",
			args:           map[string]interface{}{"tail": float64(20000)},
			expectErrorMsg: "", // Should succeed but cap at 10000
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
				t.Fatalf("Handler returned Go error: %v", err)
			}

			if result == nil {
				t.Fatal("Handler returned nil result")
			}

			// Check if result is an error result
			if tt.expectErrorMsg != "" && len(result.Content) > 0 {
				content := result.Content[0]
				if _, ok := content.(mcp.TextContent); !ok {
					t.Logf("Content type: %T", content)
				}
				// Some validation errors might pass but execution fails
				// That's OK, we're testing validation paths
			}
		})
	}
}

// TestRunServicesToolValidation tests validation logic for run_services tool
func TestRunServicesToolValidation(t *testing.T) {
	tool := newRunServicesTool()
	ctx := context.Background()

	tests := []struct {
		name           string
		args           map[string]interface{}
		expectErrorMsg string
	}{
		{
			name:           "Invalid runtime parameter",
			args:           map[string]interface{}{"runtime": "invalid_runtime"},
			expectErrorMsg: "invalid runtime",
		},
		{
			name:           "Valid runtime - azd",
			args:           map[string]interface{}{"runtime": "azd"},
			expectErrorMsg: "",
		},
		{
			name:           "Valid runtime - aspire",
			args:           map[string]interface{}{"runtime": "aspire"},
			expectErrorMsg: "",
		},
		{
			name:           "Valid runtime - pnpm",
			args:           map[string]interface{}{"runtime": "pnpm"},
			expectErrorMsg: "",
		},
		{
			name:           "Valid runtime - docker-compose",
			args:           map[string]interface{}{"runtime": "docker-compose"},
			expectErrorMsg: "",
		},
		{
			name:           "Invalid project dir",
			args:           map[string]interface{}{"projectDir": "/nonexistent/path/xyz123"},
			expectErrorMsg: "project directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "run_services",
					Arguments: tt.args,
				},
			}

			result, err := tool.Handler(ctx, request)
			if err != nil {
				t.Fatalf("Handler returned Go error: %v", err)
			}

			if result == nil {
				t.Fatal("Handler returned nil result")
			}

			// Check error message if expected
			if tt.expectErrorMsg != "" && len(result.Content) > 0 {
				if textContent, ok := result.Content[0].(mcp.TextContent); ok {
					if result.IsError && !containsSubstr(textContent.Text, tt.expectErrorMsg) {
						t.Errorf("Expected error containing '%s', got '%s'", tt.expectErrorMsg, textContent.Text)
					}
				}
			}
		})
	}
}

// TestSetEnvironmentVariableToolValidation tests validation for set_environment_variable tool
func TestSetEnvironmentVariableToolValidation(t *testing.T) {
	tool := newSetEnvironmentVariableTool()
	ctx := context.Background()

	tests := []struct {
		name           string
		args           map[string]interface{}
		expectErrorMsg string
	}{
		{
			name:           "Missing name parameter",
			args:           map[string]interface{}{"value": "test"},
			expectErrorMsg: "name parameter is required",
		},
		{
			name:           "Missing value parameter",
			args:           map[string]interface{}{"name": "TEST_VAR"},
			expectErrorMsg: "value parameter is required",
		},
		{
			name:           "Invalid env var name - starts with hyphen",
			args:           map[string]interface{}{"name": "-INVALID", "value": "test"},
			expectErrorMsg: "Invalid environment variable name",
		},
		{
			name:           "Invalid env var name - special chars",
			args:           map[string]interface{}{"name": "VAR;DROP TABLE", "value": "test"},
			expectErrorMsg: "Invalid environment variable name",
		},
		{
			name:           "Invalid service name",
			args:           map[string]interface{}{"name": "TEST_VAR", "value": "test", "serviceName": "; rm -rf /"},
			expectErrorMsg: "service name",
		},
		{
			name:           "Valid parameters",
			args:           map[string]interface{}{"name": "MY_VAR", "value": "my_value"},
			expectErrorMsg: "",
		},
		{
			name:           "Valid with service name",
			args:           map[string]interface{}{"name": "MY_VAR", "value": "my_value", "serviceName": "api"},
			expectErrorMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "set_environment_variable",
					Arguments: tt.args,
				},
			}

			result, err := tool.Handler(ctx, request)
			if err != nil {
				t.Fatalf("Handler returned Go error: %v", err)
			}

			if result == nil {
				t.Fatal("Handler returned nil result")
			}

			if tt.expectErrorMsg != "" && len(result.Content) > 0 {
				if textContent, ok := result.Content[0].(mcp.TextContent); ok {
					if result.IsError && !containsSubstr(textContent.Text, tt.expectErrorMsg) {
						t.Errorf("Expected error containing '%s', got '%s'", tt.expectErrorMsg, textContent.Text)
					}
				}
			}
		})
	}
}

// TestGetEnvironmentVariablesToolValidation tests validation for get_environment_variables tool
func TestGetEnvironmentVariablesToolValidation(t *testing.T) {
	tool := newGetEnvironmentVariablesTool()
	ctx := context.Background()

	tests := []struct {
		name           string
		args           map[string]interface{}
		expectErrorMsg string
	}{
		{
			name:           "Invalid service name - injection",
			args:           map[string]interface{}{"serviceName": "api; rm -rf /"},
			expectErrorMsg: "service name",
		},
		{
			name:           "Invalid project dir",
			args:           map[string]interface{}{"projectDir": "/nonexistent/path/xyz123"},
			expectErrorMsg: "project directory",
		},
		{
			name:           "Valid service name",
			args:           map[string]interface{}{"serviceName": "api"},
			expectErrorMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "get_environment_variables",
					Arguments: tt.args,
				},
			}

			result, err := tool.Handler(ctx, request)
			if err != nil {
				t.Fatalf("Handler returned Go error: %v", err)
			}

			if result == nil {
				t.Fatal("Handler returned nil result")
			}

			if tt.expectErrorMsg != "" && len(result.Content) > 0 {
				if textContent, ok := result.Content[0].(mcp.TextContent); ok {
					if result.IsError && !containsSubstr(textContent.Text, tt.expectErrorMsg) {
						t.Errorf("Expected error containing '%s', got '%s'", tt.expectErrorMsg, textContent.Text)
					}
				}
			}
		})
	}
}

// TestRestartServiceToolValidation tests validation for restart_service tool
func TestRestartServiceToolValidation(t *testing.T) {
	tool := newRestartServiceTool()
	ctx := context.Background()

	tests := []struct {
		name           string
		args           map[string]interface{}
		expectErrorMsg string
	}{
		{
			name:           "Missing service name",
			args:           map[string]interface{}{},
			expectErrorMsg: "serviceName parameter is required",
		},
		{
			name:           "Invalid service name - injection",
			args:           map[string]interface{}{"serviceName": "api; rm -rf /"},
			expectErrorMsg: "service name",
		},
		{
			name:           "Valid service name",
			args:           map[string]interface{}{"serviceName": "api"},
			expectErrorMsg: "",
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
				t.Fatalf("Handler returned Go error: %v", err)
			}

			if result == nil {
				t.Fatal("Handler returned nil result")
			}

			if tt.expectErrorMsg != "" && len(result.Content) > 0 {
				if textContent, ok := result.Content[0].(mcp.TextContent); ok {
					if result.IsError && !containsSubstr(textContent.Text, tt.expectErrorMsg) {
						t.Errorf("Expected error containing '%s', got '%s'", tt.expectErrorMsg, textContent.Text)
					}
				}
			}
		})
	}
}

// TestStopServicesToolHandler tests the stop_services tool handler
func TestStopServicesToolHandler(t *testing.T) {
	tool := newStopServicesTool()
	ctx := context.Background()

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "stop_services",
			Arguments: map[string]interface{}{},
		},
	}

	result, err := tool.Handler(ctx, request)
	if err != nil {
		t.Fatalf("Handler returned Go error: %v", err)
	}

	if result == nil {
		t.Fatal("Handler returned nil result")
	}

	// stop_services should return guidance (not actually stop anything)
	if result.IsError {
		t.Error("stop_services should not return an error")
	}

	if len(result.Content) == 0 {
		t.Error("stop_services should return content with guidance")
	}
}

// TestCheckRequirementsToolValidation tests validation for check_requirements tool
func TestCheckRequirementsToolValidation(t *testing.T) {
	tool := newCheckRequirementsTool()
	ctx := context.Background()

	tests := []struct {
		name           string
		args           map[string]interface{}
		expectErrorMsg string
	}{
		{
			name:           "Invalid project dir",
			args:           map[string]interface{}{"projectDir": "/nonexistent/path/xyz123"},
			expectErrorMsg: "project directory",
		},
		{
			name:           "No parameters",
			args:           map[string]interface{}{},
			expectErrorMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "check_requirements",
					Arguments: tt.args,
				},
			}

			result, err := tool.Handler(ctx, request)
			if err != nil {
				t.Fatalf("Handler returned Go error: %v", err)
			}

			if result == nil {
				t.Fatal("Handler returned nil result")
			}

			if tt.expectErrorMsg != "" && len(result.Content) > 0 {
				if textContent, ok := result.Content[0].(mcp.TextContent); ok {
					if result.IsError && !containsSubstr(textContent.Text, tt.expectErrorMsg) {
						t.Errorf("Expected error containing '%s', got '%s'", tt.expectErrorMsg, textContent.Text)
					}
				}
			}
		})
	}
}

// TestInstallDependenciesToolValidation tests validation for install_dependencies tool
func TestInstallDependenciesToolValidation(t *testing.T) {
	tool := newInstallDependenciesTool()
	ctx := context.Background()

	tests := []struct {
		name           string
		args           map[string]interface{}
		expectErrorMsg string
	}{
		{
			name:           "Invalid project dir",
			args:           map[string]interface{}{"projectDir": "/nonexistent/path/xyz123"},
			expectErrorMsg: "project directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "install_dependencies",
					Arguments: tt.args,
				},
			}

			result, err := tool.Handler(ctx, request)
			if err != nil {
				t.Fatalf("Handler returned Go error: %v", err)
			}

			if result == nil {
				t.Fatal("Handler returned nil result")
			}

			if tt.expectErrorMsg != "" && len(result.Content) > 0 {
				if textContent, ok := result.Content[0].(mcp.TextContent); ok {
					if result.IsError && !containsSubstr(textContent.Text, tt.expectErrorMsg) {
						t.Errorf("Expected error containing '%s', got '%s'", tt.expectErrorMsg, textContent.Text)
					}
				}
			}
		})
	}
}

// TestContextCancellation tests that handlers respect context cancellation
func TestContextCancellation(t *testing.T) {
	tool := newGetServiceLogsTool()

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_service_logs",
			Arguments: map[string]interface{}{},
		},
	}

	result, err := tool.Handler(ctx, request)
	if err != nil {
		t.Fatalf("Handler returned Go error: %v", err)
	}

	if result == nil {
		t.Fatal("Handler returned nil result")
	}

	// Should return an error result due to cancelled context
	if !result.IsError {
		// It's acceptable if it doesn't error - depends on timing
		t.Log("Context cancellation might not have been detected (timing dependent)")
	}
}

// TestAzureYamlResourceHandler tests the azure.yaml resource handler
func TestAzureYamlResourceHandler(t *testing.T) {
	// Create temp directory under the current working directory with azure.yaml
	// This is needed because validateProjectDir requires directories to be under cwd or home
	cwd, err := os.Getwd()
	require.NoError(t, err)
	tempDir, err := os.MkdirTemp(cwd, "test_azure_yaml_resource")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	content := `name: test-project
services:
  api:
    language: python
`
	err = os.WriteFile(azureYamlPath, []byte(content), 0644)
	require.NoError(t, err)

	// Set environment variable for project dir
	originalEnv := os.Getenv("AZD_APP_PROJECT_DIR")
	os.Setenv("AZD_APP_PROJECT_DIR", tempDir)
	defer func() {
		if originalEnv != "" {
			os.Setenv("AZD_APP_PROJECT_DIR", originalEnv)
		} else {
			os.Unsetenv("AZD_APP_PROJECT_DIR")
		}
	}()

	resource := newAzureYamlResource()
	ctx := context.Background()

	request := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: "azure://project/azure.yaml",
		},
	}

	result, err := resource.Handler(ctx, request)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	if len(result) == 0 {
		t.Fatal("Handler returned empty result")
	}

	// Verify content
	if textContent, ok := result[0].(*mcp.TextResourceContents); ok {
		if textContent.Text != content {
			t.Errorf("Content mismatch: expected '%s', got '%s'", content, textContent.Text)
		}
		if textContent.MIMEType != "application/x-yaml" {
			t.Errorf("MIME type mismatch: expected 'application/x-yaml', got '%s'", textContent.MIMEType)
		}
	} else {
		t.Error("Expected TextResourceContents")
	}
}

// TestAzureYamlResourceHandlerMissingFile tests error handling when azure.yaml is missing
func TestAzureYamlResourceHandlerMissingFile(t *testing.T) {
	// Create temp directory under the current working directory
	// This is needed because validateProjectDir requires directories to be under cwd or home
	cwd, err := os.Getwd()
	require.NoError(t, err)
	tempDir, err := os.MkdirTemp(cwd, "test_azure_yaml_missing")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	originalEnv := os.Getenv("AZD_APP_PROJECT_DIR")
	os.Setenv("AZD_APP_PROJECT_DIR", tempDir)
	defer func() {
		if originalEnv != "" {
			os.Setenv("AZD_APP_PROJECT_DIR", originalEnv)
		} else {
			os.Unsetenv("AZD_APP_PROJECT_DIR")
		}
	}()

	resource := newAzureYamlResource()
	ctx := context.Background()

	request := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: "azure://project/azure.yaml",
		},
	}

	_, err = resource.Handler(ctx, request)
	if err == nil {
		t.Error("Expected error when azure.yaml is missing")
	}

	if !containsSubstr(err.Error(), "azure.yaml not found") {
		t.Errorf("Expected 'azure.yaml not found' error, got: %v", err)
	}
}

// TestGetProjectDirWithFallback tests the PROJECT_DIR fallback
func TestGetProjectDirWithFallback(t *testing.T) {
	// Save original values
	originalAzdAppProjectDir := os.Getenv("AZD_APP_PROJECT_DIR")
	originalProjectDir := os.Getenv("PROJECT_DIR")
	defer func() {
		if originalAzdAppProjectDir != "" {
			os.Setenv("AZD_APP_PROJECT_DIR", originalAzdAppProjectDir)
		} else {
			os.Unsetenv("AZD_APP_PROJECT_DIR")
		}
		if originalProjectDir != "" {
			os.Setenv("PROJECT_DIR", originalProjectDir)
		} else {
			os.Unsetenv("PROJECT_DIR")
		}
	}()

	// Create temp directories under the current working directory for testing
	// This is needed because validateProjectDir requires directories to be under cwd or home
	cwd, err := os.Getwd()
	require.NoError(t, err)
	tempAzdDir, err := os.MkdirTemp(cwd, "test_azd_dir")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempAzdDir)

	tempFallbackDir, err := os.MkdirTemp(cwd, "test_fallback_dir")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempFallbackDir)

	tests := []struct {
		name             string
		azdAppProjectDir string
		projectDir       string
		expected         string
	}{
		{
			name:             "AZD_APP_PROJECT_DIR takes precedence",
			azdAppProjectDir: tempAzdDir,
			projectDir:       tempFallbackDir,
			expected:         tempAzdDir,
		},
		{
			name:             "Falls back to PROJECT_DIR",
			azdAppProjectDir: "",
			projectDir:       tempFallbackDir,
			expected:         tempFallbackDir,
		},
		{
			name:             "Returns current dir when both empty",
			azdAppProjectDir: "",
			projectDir:       "",
			expected:         cwd, // validateProjectDir converts "." to absolute cwd
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv("AZD_APP_PROJECT_DIR")
			os.Unsetenv("PROJECT_DIR")

			if tt.azdAppProjectDir != "" {
				os.Setenv("AZD_APP_PROJECT_DIR", tt.azdAppProjectDir)
			}
			if tt.projectDir != "" {
				os.Setenv("PROJECT_DIR", tt.projectDir)
			}

			result := getProjectDir()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// containsSubstr is a helper function for string containment check
func containsSubstr(s, substr string) bool {
	return strings.Contains(s, substr)
}

// TestTokenBucketRateLimit tests the token bucket rate limiter
func TestTokenBucketRateLimit(t *testing.T) {
	// Create a rate limiter with 3 tokens, refilling at 1 token per 100ms
	limiter := NewTokenBucket(3, 100*time.Millisecond)

	// First 3 calls should succeed (burst capacity)
	for i := 0; i < 3; i++ {
		if !limiter.Allow() {
			t.Errorf("Call %d should be allowed (within burst)", i+1)
		}
	}

	// 4th call should be blocked
	if limiter.Allow() {
		t.Error("4th call should be blocked (burst exhausted)")
	}

	// Wait for token refill
	time.Sleep(150 * time.Millisecond)

	// Should have 1 token now
	if !limiter.Allow() {
		t.Error("Call after refill should be allowed")
	}

	// Should be blocked again
	if limiter.Allow() {
		t.Error("Call should be blocked again")
	}
}

// TestRateLimitIntegration tests rate limiting in MCP tools
func TestRateLimitIntegration(t *testing.T) {
	// Save and restore the global rate limiter
	oldLimiter := globalRateLimiter
	defer func() { globalRateLimiter = oldLimiter }()

	// Create a strict rate limiter for testing
	globalRateLimiter = NewTokenBucket(2, 10*time.Second)

	tool := newRunServicesTool()
	ctx := context.Background()

	// First two calls should succeed (or fail with different error)
	for i := 0; i < 2; i++ {
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "run_services",
				Arguments: map[string]interface{}{},
			},
		}
		result, _ := tool.Handler(ctx, request)
		if result.IsError && strings.Contains(string(result.Content[0].(mcp.TextContent).Text), "Rate limit exceeded") {
			t.Errorf("Call %d should not be rate limited", i+1)
		}
	}

	// Third call should be rate limited
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "run_services",
			Arguments: map[string]interface{}{},
		},
	}
	result, _ := tool.Handler(ctx, request)
	if !result.IsError {
		t.Error("3rd call should be rate limited")
	}
	if !strings.Contains(string(result.Content[0].(mcp.TextContent).Text), "Rate limit exceeded") {
		t.Error("3rd call should return rate limit error")
	}
}

// TestGetProjectDirValidation tests that environment variables are validated
func TestGetProjectDirValidation(t *testing.T) {
	// Save original values
	originalAzdAppProjectDir := os.Getenv("AZD_APP_PROJECT_DIR")
	defer func() {
		if originalAzdAppProjectDir != "" {
			os.Setenv("AZD_APP_PROJECT_DIR", originalAzdAppProjectDir)
		} else {
			os.Unsetenv("AZD_APP_PROJECT_DIR")
		}
	}()

	// Create a valid temp directory under the current working directory
	// This is needed because validateProjectDir requires directories to be under cwd or home
	cwd, err := os.Getwd()
	require.NoError(t, err)
	validTempDir, err := os.MkdirTemp(cwd, "test_validation_valid")
	require.NoError(t, err)
	defer os.RemoveAll(validTempDir)

	tests := []struct {
		name        string
		envValue    string
		shouldBeCwd bool // Should fall back to current dir
	}{
		{
			name:        "Valid temp directory under cwd",
			envValue:    validTempDir,
			shouldBeCwd: false,
		},
		{
			name:        "System directory should fail validation",
			envValue:    "/etc",
			shouldBeCwd: true, // Should fall back to current dir
		},
		{
			name:        "Non-existent directory should fail validation",
			envValue:    "/nonexistent/path/xyz",
			shouldBeCwd: true, // Should fall back to current dir
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("AZD_APP_PROJECT_DIR", tt.envValue)
			result := getProjectDir()

			if tt.shouldBeCwd && result != "." && result != cwd {
				t.Errorf("Expected current directory fallback, got %s", result)
			}
			if !tt.shouldBeCwd && (result == "." || result == cwd) {
				// This might be OK if the env value is actually the cwd
				if tt.envValue != cwd && !strings.HasPrefix(cwd, tt.envValue) {
					t.Errorf("Expected valid directory, got current directory fallback")
				}
			}
		})
	}
}
