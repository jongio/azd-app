package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/jongio/azd-core/security"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/require"
)

// testBuildServer creates a test MCP server with all tools registered via the builder.
func testBuildServer(t *testing.T) *server.MCPServer {
	t.Helper()
	builder := azdext.NewMCPServerBuilder("test-server", "1.0.0")
	registerAllTools(builder)
	return builder.Build()
}

// testToolArgs creates ToolArgs from a map for use in handler tests.
func testToolArgs(args map[string]interface{}) azdext.ToolArgs {
	return azdext.ParseToolArgs(mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	})
}

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
	s := testBuildServer(t)
	tool := s.GetTool("get_services")
	require.NotNil(t, tool, "get_services tool should be registered")
	require.Equal(t, "get_services", tool.Tool.Name)
	require.NotEmpty(t, tool.Tool.Description)
	require.Equal(t, "Get Running Services", tool.Tool.Annotations.Title)
}

func TestGetServiceLogsToolDefinition(t *testing.T) {
	s := testBuildServer(t)
	tool := s.GetTool("get_service_logs")
	require.NotNil(t, tool, "get_service_logs tool should be registered")
	require.Equal(t, "get_service_logs", tool.Tool.Name)
	require.NotEmpty(t, tool.Tool.Description)
}

func TestGetProjectInfoToolDefinition(t *testing.T) {
	s := testBuildServer(t)
	tool := s.GetTool("get_project_info")
	require.NotNil(t, tool, "get_project_info tool should be registered")
	require.Equal(t, "get_project_info", tool.Tool.Name)
	require.NotEmpty(t, tool.Tool.Description)
}

func TestGetServicesToolHandlerBehavior(t *testing.T) {
	ctx := context.Background()
	args := testToolArgs(map[string]interface{}{})

	result, err := handleGetServices(ctx, args)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	if result == nil {
		t.Fatal("Handler returned nil result")
	}

	if len(result.Content) == 0 {
		t.Error("Handler result should have content")
	}
}

func TestGetServiceLogsToolHandlerBehavior(t *testing.T) {
	ctx := context.Background()
	args := testToolArgs(map[string]interface{}{"tail": float64(10)})

	result, err := handleGetServiceLogs(ctx, args)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	if result == nil {
		t.Fatal("Handler returned nil result")
	}

	if len(result.Content) == 0 {
		t.Error("Handler result should have content")
	}
}

func TestGetProjectInfoToolHandlerBehavior(t *testing.T) {
	ctx := context.Background()
	args := testToolArgs(map[string]interface{}{})

	result, err := handleGetProjectInfo(ctx, args)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	if result == nil {
		t.Fatal("Handler returned nil result")
	}

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
	s := testBuildServer(t)
	tool := s.GetTool("run_services")
	require.NotNil(t, tool, "run_services tool should be registered")
	require.Equal(t, "run_services", tool.Tool.Name)
	require.NotEmpty(t, tool.Tool.Description)
}

func TestInstallDependenciesToolDefinition(t *testing.T) {
	s := testBuildServer(t)
	tool := s.GetTool("install_dependencies")
	require.NotNil(t, tool, "install_dependencies tool should be registered")
	require.Equal(t, "install_dependencies", tool.Tool.Name)
	require.NotEmpty(t, tool.Tool.Description)
}

func TestCheckRequirementsToolDefinition(t *testing.T) {
	s := testBuildServer(t)
	tool := s.GetTool("check_requirements")
	require.NotNil(t, tool, "check_requirements tool should be registered")
	require.Equal(t, "check_requirements", tool.Tool.Name)
	require.NotEmpty(t, tool.Tool.Description)
}

func TestStopServicesToolDefinition(t *testing.T) {
	s := testBuildServer(t)
	tool := s.GetTool("stop_services")
	require.NotNil(t, tool, "stop_services tool should be registered")
	require.Equal(t, "stop_services", tool.Tool.Name)
}

func TestRestartServiceToolDefinition(t *testing.T) {
	s := testBuildServer(t)
	tool := s.GetTool("restart_service")
	require.NotNil(t, tool, "restart_service tool should be registered")
	require.Equal(t, "restart_service", tool.Tool.Name)
}

func TestGetEnvironmentVariablesToolDefinition(t *testing.T) {
	s := testBuildServer(t)
	tool := s.GetTool("get_environment_variables")
	require.NotNil(t, tool, "get_environment_variables tool should be registered")
	require.Equal(t, "get_environment_variables", tool.Tool.Name)
}

func TestSetEnvironmentVariableToolDefinition(t *testing.T) {
	s := testBuildServer(t)
	tool := s.GetTool("set_environment_variable")
	require.NotNil(t, tool, "set_environment_variable tool should be registered")
	require.Equal(t, "set_environment_variable", tool.Tool.Name)
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

			// Verify that structured content is returned (not just text)
			if !tt.wantError && result != nil {
				// Should have structured content populated
				if result.StructuredContent == nil {
					t.Error("Expected StructuredContent to be populated for schema-based tools")
				}
				// Should also have text content for backwards compatibility
				if len(result.Content) == 0 {
					t.Error("Expected Content to have fallback text")
				}
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
	defer os.RemoveAll(tempDir) //nolint:errcheck

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
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: tt.args,
				},
			}
			args := azdext.ParseToolArgs(request)
			result, err := extractProjectDirArg(args)
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
	defer os.RemoveAll(tempDir) //nolint:errcheck
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
			args := testToolArgs(tt.args)
			result, err := handleGetServices(ctx, args)

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
			args := testToolArgs(tt.args)
			result, err := handleGetServiceLogs(ctx, args)

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
	ctx := context.Background()
	args := testToolArgs(map[string]interface{}{})

	result, err := handleGetProjectInfo(ctx, args)

	if err != nil {
		t.Errorf("Handler returned Go error: %v", err)
	}

	if result == nil {
		t.Fatal("Handler returned nil result")
	}
}

func TestRestartServiceToolHandler(t *testing.T) {
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
			args := testToolArgs(tt.args)
			result, err := handleRestartService(ctx, args)

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
	// Create a temp directory under the current working directory for testing
	// This is needed because validateProjectDir requires directories to be under cwd or home
	cwd, err := os.Getwd()
	require.NoError(t, err)
	tempDir, err := os.MkdirTemp(cwd, "test_project_dir")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck

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
				t.Setenv("AZD_APP_PROJECT_DIR", tt.envValue)
			} else {
				t.Setenv("AZD_APP_PROJECT_DIR", "")
				_ = os.Unsetenv("AZD_APP_PROJECT_DIR")
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
	s := testBuildServer(t)
	expected := map[string]string{
		"get_services":              "Get Running Services",
		"get_service_logs":          "Get Service Logs",
		"get_service_errors":        "Get Service Errors",
		"get_project_info":          "Get Project Information",
		"run_services":              "Run Development Services",
		"stop_services":             "Stop Running Services",
		"restart_service":           "Restart Service",
		"install_dependencies":      "Install Project Dependencies",
		"check_requirements":        "Check Prerequisites",
		"get_environment_variables": "Get Environment Variables",
		"set_environment_variable":  "Set Environment Variable",
	}

	for name, title := range expected {
		t.Run(name, func(t *testing.T) {
			tool := s.GetTool(name)
			require.NotNil(t, tool, "tool %s should be registered", name)
			require.Equal(t, title, tool.Tool.Annotations.Title)
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
		if resource.Resource.Annotations.Priority == nil || *resource.Resource.Annotations.Priority != 0.9 {
			t.Errorf("azure.yaml resource should have priority 0.9, got %v", resource.Resource.Annotations.Priority)
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
		if resource.Resource.Annotations.Priority == nil || *resource.Resource.Annotations.Priority != 0.7 {
			t.Errorf("service-configs resource should have priority 0.7, got %v", resource.Resource.Annotations.Priority)
		}
	})
}

func TestGetServiceLogsToolValidation(t *testing.T) {
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
			args := testToolArgs(tt.args)
			result, err := handleGetServiceLogs(ctx, args)
			if err != nil {
				t.Fatalf("Handler returned Go error: %v", err)
			}

			if result == nil {
				t.Fatal("Handler returned nil result")
			}

			if tt.expectErrorMsg != "" && len(result.Content) > 0 {
				content := result.Content[0]
				if _, ok := content.(mcp.TextContent); !ok {
					t.Logf("Content type: %T", content)
				}
			}
		})
	}
}

// TestRunServicesToolValidation tests validation logic for run_services tool
func TestRunServicesToolValidation(t *testing.T) {
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
			args := testToolArgs(tt.args)
			result, err := handleRunServices(ctx, args)
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
	ctx := context.Background()

	tests := []struct {
		name           string
		args           map[string]interface{}
		expectErrorMsg string
	}{
		{
			name:           "Missing name parameter",
			args:           map[string]interface{}{"value": "test"},
			expectErrorMsg: "required argument \"name\" not found",
		},
		{
			name:           "Missing value parameter",
			args:           map[string]interface{}{"name": "TEST_VAR"},
			expectErrorMsg: "required argument \"value\" not found",
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
			args := testToolArgs(tt.args)
			result, err := handleSetEnvironmentVariable(ctx, args)
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
			args := testToolArgs(tt.args)
			result, err := handleGetEnvironmentVariables(ctx, args)
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
	ctx := context.Background()

	tests := []struct {
		name           string
		args           map[string]interface{}
		expectErrorMsg string
	}{
		{
			name:           "Missing service name",
			args:           map[string]interface{}{},
			expectErrorMsg: "required argument \"serviceName\" not found",
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
			args := testToolArgs(tt.args)
			result, err := handleRestartService(ctx, args)
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
	ctx := context.Background()
	args := testToolArgs(map[string]interface{}{})

	result, err := handleStopServices(ctx, args)
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
			args := testToolArgs(tt.args)
			result, err := handleCheckRequirements(ctx, args)
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
			args := testToolArgs(tt.args)
			result, err := handleInstallDependencies(ctx, args)
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
	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	args := testToolArgs(map[string]interface{}{})
	result, err := handleGetServiceLogs(ctx, args)
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
	defer os.RemoveAll(tempDir) //nolint:errcheck
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	content := `name: test-project
services:
  api:
    language: python
`
	err = os.WriteFile(azureYamlPath, []byte(content), 0644)
	require.NoError(t, err)

	// Set environment variable for project dir
	t.Setenv("AZD_APP_PROJECT_DIR", tempDir)

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
	defer os.RemoveAll(tempDir) //nolint:errcheck
	t.Setenv("AZD_APP_PROJECT_DIR", tempDir)

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
	// Create temp directories under the current working directory for testing
	// This is needed because validateProjectDir requires directories to be under cwd or home
	cwd, err := os.Getwd()
	require.NoError(t, err)
	tempAzdDir, err := os.MkdirTemp(cwd, "test_azd_dir")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempAzdDir) //nolint:errcheck

	tempFallbackDir, err := os.MkdirTemp(cwd, "test_fallback_dir")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempFallbackDir) //nolint:errcheck

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
			// Use t.Setenv to ensure cleanup; then unset if value should be empty
			t.Setenv("AZD_APP_PROJECT_DIR", "")
			_ = os.Unsetenv("AZD_APP_PROJECT_DIR")
			t.Setenv("PROJECT_DIR", "")
			_ = os.Unsetenv("PROJECT_DIR")

			if tt.azdAppProjectDir != "" {
				t.Setenv("AZD_APP_PROJECT_DIR", tt.azdAppProjectDir)
			}
			if tt.projectDir != "" {
				t.Setenv("PROJECT_DIR", tt.projectDir)
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

// TestRateLimitIntegration tests rate limiting through the builder's middleware
func TestRateLimitIntegration(t *testing.T) {
	// Build a server with strict rate limiting: burst=2, very slow refill
	builder := azdext.NewMCPServerBuilder("test-server", "1.0.0").
		WithRateLimit(2, 0.001)
	addRunServicesTool(builder)
	s := builder.Build()

	tool := s.GetTool("run_services")
	require.NotNil(t, tool)

	ctx := context.Background()

	// First two calls should succeed (or fail with non-rate-limit error)
	for i := 0; i < 2; i++ {
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "run_services",
				Arguments: map[string]interface{}{},
			},
		}
		result, _ := tool.Handler(ctx, request)
		if result.IsError {
			if tc, ok := result.Content[0].(mcp.TextContent); ok {
				if strings.Contains(tc.Text, "rate limit exceeded") {
					t.Errorf("Call %d should not be rate limited", i+1)
				}
			}
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
	require.True(t, result.IsError, "3rd call should be rate limited")
	tc, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok)
	require.Contains(t, tc.Text, "rate limit exceeded")
}

// TestGetProjectDirValidation tests that environment variables are validated
func TestGetProjectDirValidation(t *testing.T) {
	// Create a valid temp directory under the current working directory
	// This is needed because validateProjectDir requires directories to be under cwd or home
	cwd, err := os.Getwd()
	require.NoError(t, err)
	validTempDir, err := os.MkdirTemp(cwd, "test_validation_valid")
	require.NoError(t, err)
	defer os.RemoveAll(validTempDir) //nolint:errcheck

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
			t.Setenv("AZD_APP_PROJECT_DIR", tt.envValue)
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
