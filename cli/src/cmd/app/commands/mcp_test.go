package commands

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
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

func TestGetServicesToolHandler(t *testing.T) {
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

func TestGetServiceLogsToolHandler(t *testing.T) {
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

func TestGetProjectInfoToolHandler(t *testing.T) {
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
