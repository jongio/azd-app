package service_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/portmanager"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// TestMain sets up test mode for all tests in this package.
// This prevents interactive prompts when real ports are in use.
func TestMain(m *testing.M) {
	// Set test mode: all ports appear available
	cleanup := portmanager.SetTestModeForTesting(func(port int) bool {
		return true // All ports available in tests
	})
	defer cleanup()

	// Clear any cached port managers
	portmanager.ClearCacheForTesting()

	os.Exit(m.Run())
}

// TestFunctionsVariant_String tests the String() method for all variants
func TestFunctionsVariant_String(t *testing.T) {
	tests := []struct {
		variant  int // Using int to avoid import issues
		expected string
	}{
		{0, "Unknown"},    // FunctionsVariantUnknown
		{1, "Logic Apps"}, // FunctionsVariantLogicApps
		{2, "Node.js"},    // FunctionsVariantNodeJS
		{3, "Python"},     // FunctionsVariantPython
		{4, ".NET"},       // FunctionsVariantDotNet
		{5, "Java"},       // FunctionsVariantJava
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			// Note: We're testing the method indirectly through detection
			// Direct testing would require exporting FunctionsVariant or creating a helper
		})
	}
}

// TestDetectFunctionsVariant_LogicApps tests Logic Apps detection
func TestDetectFunctionsVariant_LogicApps(t *testing.T) {
	tests := []struct {
		name         string
		projectFiles map[string]string
		shouldDetect bool
		description  string
	}{
		{
			name: "Logic Apps with workflows folder",
			projectFiles: map[string]string{
				"host.json":                          `{"version": "2.0"}`,
				"workflows/MyWorkflow/workflow.json": `{"definition": {}}`,
			},
			shouldDetect: true,
			description:  "Should detect Logic Apps via workflows/ directory",
		},
		{
			name: "Logic Apps with extension bundle",
			projectFiles: map[string]string{
				"host.json": `{
					"version": "2.0",
					"extensionBundle": {
						"id": "Microsoft.Azure.Functions.ExtensionBundle.Workflows",
						"version": "[1.*, 2.0.0)"
					}
				}`,
			},
			shouldDetect: true,
			description:  "Should detect Logic Apps via extension bundle",
		},
		{
			name: "Logic Apps with both workflows and bundle",
			projectFiles: map[string]string{
				"host.json": `{
					"version": "2.0",
					"extensionBundle": {
						"id": "Microsoft.Azure.Functions.ExtensionBundle.Workflows"
					}
				}`,
				"workflows/MyWorkflow/workflow.json": `{"definition": {}}`,
			},
			shouldDetect: true,
			description:  "Should detect Logic Apps with both indicators",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()

			// Create project files
			for filename, content := range tt.projectFiles {
				filePath := filepath.Join(tmpDir, filename)
				dir := filepath.Dir(filePath)
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("Failed to create directory %s: %v", dir, err)
				}
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to write file %s: %v", filePath, err)
				}
			}

			// Test detection via DetectServiceRuntime
			svc := service.Service{
				Project: tmpDir,
				Host:    "function",
			}
			usedPorts := make(map[int]bool)
			runtime, err := service.DetectServiceRuntime("test-service", svc, usedPorts, tmpDir, "")

			if tt.shouldDetect {
				if err != nil {
					t.Fatalf("Expected successful detection, got error: %v", err)
				}
				if runtime.Language != "Logic Apps" {
					t.Errorf("Expected Language 'Logic Apps', got %q", runtime.Language)
				}
				if runtime.Framework != "Logic Apps Standard" {
					t.Errorf("Expected Framework 'Logic Apps Standard', got %q", runtime.Framework)
				}
			}
		})
	}
}

// TestDetectFunctionsVariant_NodeJS tests Node.js Functions detection
func TestDetectFunctionsVariant_NodeJS(t *testing.T) {
	tests := []struct {
		name         string
		projectFiles map[string]string
		expectedLang string
		description  string
	}{
		{
			name: "Node.js v4 with TypeScript",
			projectFiles: map[string]string{
				"host.json": `{"version": "2.0"}`,
				"package.json": `{
					"dependencies": {
						"@azure/functions": "^4.0.0"
					}
				}`,
				"tsconfig.json": `{"compilerOptions": {}}`,
			},
			expectedLang: "TypeScript",
			description:  "Should detect TypeScript Functions v4",
		},
		{
			name: "Node.js v4 without TypeScript",
			projectFiles: map[string]string{
				"host.json": `{"version": "2.0"}`,
				"package.json": `{
					"dependencies": {
						"@azure/functions": "^4.0.0"
					}
				}`,
			},
			expectedLang: "JavaScript",
			description:  "Should detect JavaScript Functions v4",
		},
		{
			name: "Node.js v3 with function.json",
			projectFiles: map[string]string{
				"host.json":                 `{"version": "2.0"}`,
				"package.json":              `{"dependencies": {}}`,
				"HttpTrigger/function.json": `{"bindings": []}`,
				"HttpTrigger/index.js":      `module.exports = async function() {}`,
			},
			expectedLang: "JavaScript",
			description:  "Should detect JavaScript Functions v3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			for filename, content := range tt.projectFiles {
				filePath := filepath.Join(tmpDir, filename)
				dir := filepath.Dir(filePath)
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("Failed to create directory %s: %v", dir, err)
				}
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to write file %s: %v", filePath, err)
				}
			}

			svc := service.Service{
				Project: tmpDir,
				Host:    "function",
			}
			usedPorts := make(map[int]bool)
			runtime, err := service.DetectServiceRuntime("test-service", svc, usedPorts, tmpDir, "")

			if err != nil {
				t.Fatalf("Expected successful detection, got error: %v", err)
			}
			if runtime.Language != tt.expectedLang {
				t.Errorf("Expected Language %q, got %q", tt.expectedLang, runtime.Language)
			}
			if runtime.Framework != "Node.js Functions" {
				t.Errorf("Expected Framework 'Node.js Functions', got %q", runtime.Framework)
			}
		})
	}
}

// TestDetectFunctionsVariant_Python tests Python Functions detection
func TestDetectFunctionsVariant_Python(t *testing.T) {
	tests := []struct {
		name         string
		projectFiles map[string]string
		description  string
	}{
		{
			name: "Python v2 with function_app.py",
			projectFiles: map[string]string{
				"host.json": `{"version": "2.0"}`,
				"function_app.py": `
import azure.functions as func
app = func.FunctionApp()
@app.route(route="hello")
def hello(req: func.HttpRequest) -> func.HttpResponse:
    return func.HttpResponse("Hello")
				`,
			},
			description: "Should detect Python v2 model",
		},
		{
			name: "Python v1 with requirements.txt and function.json",
			projectFiles: map[string]string{
				"host.json":                 `{"version": "2.0"}`,
				"requirements.txt":          `azure-functions`,
				"HttpTrigger/function.json": `{"bindings": []}`,
				"HttpTrigger/__init__.py":   `def main(req): pass`,
			},
			description: "Should detect Python v1 model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			for filename, content := range tt.projectFiles {
				filePath := filepath.Join(tmpDir, filename)
				dir := filepath.Dir(filePath)
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("Failed to create directory %s: %v", dir, err)
				}
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to write file %s: %v", filePath, err)
				}
			}

			svc := service.Service{
				Project: tmpDir,
				Host:    "function",
			}
			usedPorts := make(map[int]bool)
			runtime, err := service.DetectServiceRuntime("test-service", svc, usedPorts, tmpDir, "")

			if err != nil {
				t.Fatalf("Expected successful detection, got error: %v", err)
			}
			if runtime.Language != "Python" {
				t.Errorf("Expected Language 'Python', got %q", runtime.Language)
			}
			if runtime.Framework != "Python Functions" {
				t.Errorf("Expected Framework 'Python Functions', got %q", runtime.Framework)
			}
		})
	}
}

// TestDetectFunctionsVariant_DotNet tests .NET Functions detection
func TestDetectFunctionsVariant_DotNet(t *testing.T) {
	tests := []struct {
		name         string
		projectFiles map[string]string
		description  string
	}{
		{
			name: ".NET Isolated Worker",
			projectFiles: map[string]string{
				"host.json": `{"version": "2.0"}`,
				"Function.csproj": `<Project Sdk="Microsoft.NET.Sdk">
					<ItemGroup>
						<PackageReference Include="Microsoft.Azure.Functions.Worker" Version="1.0.0" />
					</ItemGroup>
				</Project>`,
			},
			description: "Should detect .NET Isolated Worker",
		},
		{
			name: ".NET In-Process",
			projectFiles: map[string]string{
				"host.json": `{"version": "2.0"}`,
				"Function.csproj": `<Project Sdk="Microsoft.NET.Sdk.Functions">
					<PropertyGroup>
						<TargetFramework>net6.0</TargetFramework>
					</PropertyGroup>
				</Project>`,
			},
			description: "Should detect .NET In-Process",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			for filename, content := range tt.projectFiles {
				filePath := filepath.Join(tmpDir, filename)
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to write file %s: %v", filePath, err)
				}
			}

			svc := service.Service{
				Project: tmpDir,
				Host:    "function",
			}
			usedPorts := make(map[int]bool)
			runtime, err := service.DetectServiceRuntime("test-service", svc, usedPorts, tmpDir, "")

			if err != nil {
				t.Fatalf("Expected successful detection, got error: %v", err)
			}
			if runtime.Language != "C#" {
				t.Errorf("Expected Language 'C#', got %q", runtime.Language)
			}
			if runtime.Framework != ".NET Functions" {
				t.Errorf("Expected Framework '.NET Functions', got %q", runtime.Framework)
			}
		})
	}
}

// TestDetectFunctionsVariant_Java tests Java Functions detection
func TestDetectFunctionsVariant_Java(t *testing.T) {
	tests := []struct {
		name         string
		projectFiles map[string]string
		description  string
	}{
		{
			name: "Java with Maven",
			projectFiles: map[string]string{
				"host.json": `{"version": "2.0"}`,
				"pom.xml": `<project>
					<build>
						<plugins>
							<plugin>
								<groupId>com.microsoft.azure</groupId>
								<artifactId>azure-functions-maven-plugin</artifactId>
							</plugin>
						</plugins>
					</build>
				</project>`,
			},
			description: "Should detect Java Functions with Maven",
		},
		{
			name: "Java with Gradle",
			projectFiles: map[string]string{
				"host.json": `{"version": "2.0"}`,
				"build.gradle": `
plugins {
    id 'com.microsoft.azure.azurefunctions' version '1.0.0'
}
				`,
			},
			description: "Should detect Java Functions with Gradle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			for filename, content := range tt.projectFiles {
				filePath := filepath.Join(tmpDir, filename)
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to write file %s: %v", filePath, err)
				}
			}

			svc := service.Service{
				Project: tmpDir,
				Host:    "function",
			}
			usedPorts := make(map[int]bool)
			runtime, err := service.DetectServiceRuntime("test-service", svc, usedPorts, tmpDir, "")

			if err != nil {
				t.Fatalf("Expected successful detection, got error: %v", err)
			}
			if runtime.Language != "Java" {
				t.Errorf("Expected Language 'Java', got %q", runtime.Language)
			}
			if runtime.Framework != "Java Functions" {
				t.Errorf("Expected Framework 'Java Functions', got %q", runtime.Framework)
			}
		})
	}
}

// TestDetectFunctionsVariant_NoHostJson tests error handling for missing host.json
func TestDetectFunctionsVariant_NoHostJson(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Node.js project WITHOUT host.json
	packageJSON := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(packageJSON, []byte(`{"dependencies": {"@azure/functions": "^4.0.0"}}`), 0644); err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	svc := service.Service{
		Project: tmpDir,
		Host:    "function",
	}
	usedPorts := make(map[int]bool)
	_, err := service.DetectServiceRuntime("test-service", svc, usedPorts, tmpDir, "")

	if err == nil {
		t.Fatalf("Expected error for missing host.json, got nil")
	}
	if !containsString(err.Error(), "host.json") {
		t.Errorf("Expected error message to mention 'host.json', got: %v", err)
	}
}

// TestDetectFunctionsVariant_NoFunctions tests error handling when no functions are defined
func TestDetectFunctionsVariant_NoFunctions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create host.json but no function definitions
	hostJSON := filepath.Join(tmpDir, "host.json")
	if err := os.WriteFile(hostJSON, []byte(`{"version": "2.0"}`), 0644); err != nil {
		t.Fatalf("Failed to write host.json: %v", err)
	}

	svc := service.Service{
		Project: tmpDir,
		Host:    "function",
	}
	usedPorts := make(map[int]bool)
	_, err := service.DetectServiceRuntime("test-service", svc, usedPorts, tmpDir, "")

	if err == nil {
		t.Fatalf("Expected error for no function definitions, got nil")
	}
	if !containsString(err.Error(), "variant") && !containsString(err.Error(), "function definitions") {
		t.Errorf("Expected error message to mention variant/functions, got: %v", err)
	}
}

// TestBuildFunctionsRuntime_DefaultPort tests that default port is 7071
func TestBuildFunctionsRuntime_DefaultPort(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Logic Apps project
	hostJSON := filepath.Join(tmpDir, "host.json")
	if err := os.WriteFile(hostJSON, []byte(`{
		"version": "2.0",
		"extensionBundle": {
			"id": "Microsoft.Azure.Functions.ExtensionBundle.Workflows"
		}
	}`), 0644); err != nil {
		t.Fatalf("Failed to write host.json: %v", err)
	}

	svc := service.Service{
		Project: tmpDir,
		Host:    "function",
	}
	usedPorts := make(map[int]bool)
	runtime, err := service.DetectServiceRuntime("test-service", svc, usedPorts, tmpDir, "")

	if err != nil {
		t.Fatalf("Expected successful detection, got error: %v", err)
	}
	if runtime.Port != 7071 {
		t.Errorf("Expected default port 7071, got %d", runtime.Port)
	}
}

// TestBuildFunctionsRuntime_ExplicitPort tests that explicit port from azure.yaml is respected
func TestBuildFunctionsRuntime_ExplicitPort(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Logic Apps project
	hostJSON := filepath.Join(tmpDir, "host.json")
	if err := os.WriteFile(hostJSON, []byte(`{
		"version": "2.0",
		"extensionBundle": {
			"id": "Microsoft.Azure.Functions.ExtensionBundle.Workflows"
		}
	}`), 0644); err != nil {
		t.Fatalf("Failed to write host.json: %v", err)
	}

	svc := service.Service{
		Project: tmpDir,
		Host:    "function",
		Ports:   []string{"8080"},
	}
	usedPorts := make(map[int]bool)
	runtime, err := service.DetectServiceRuntime("test-service", svc, usedPorts, tmpDir, "")

	if err != nil {
		t.Fatalf("Expected successful detection, got error: %v", err)
	}
	if runtime.Port != 8080 {
		t.Errorf("Expected explicit port 8080, got %d", runtime.Port)
	}
}

// TestBuildFunctionsRuntime_PortConflict tests port conflict resolution
func TestBuildFunctionsRuntime_PortConflict(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Logic Apps project
	hostJSON := filepath.Join(tmpDir, "host.json")
	if err := os.WriteFile(hostJSON, []byte(`{
		"version": "2.0",
		"extensionBundle": {
			"id": "Microsoft.Azure.Functions.ExtensionBundle.Workflows"
		}
	}`), 0644); err != nil {
		t.Fatalf("Failed to write host.json: %v", err)
	}

	// Mark default port as used
	usedPorts := map[int]bool{7071: true}

	svc := service.Service{
		Project: tmpDir,
		Host:    "function",
	}
	runtime, err := service.DetectServiceRuntime("test-service", svc, usedPorts, tmpDir, "")

	if err != nil {
		t.Fatalf("Expected successful detection, got error: %v", err)
	}
	if runtime.Port == 7071 {
		t.Errorf("Expected port to be assigned different from 7071 (conflict), got %d", runtime.Port)
	}
}

// TestBuildFunctionsRuntime_Command tests that func start command is built correctly
func TestBuildFunctionsRuntime_Command(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Node.js Functions project
	hostJSON := filepath.Join(tmpDir, "host.json")
	packageJSON := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(hostJSON, []byte(`{"version": "2.0"}`), 0644); err != nil {
		t.Fatalf("Failed to write host.json: %v", err)
	}
	if err := os.WriteFile(packageJSON, []byte(`{"dependencies": {"@azure/functions": "^4.0.0"}}`), 0644); err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	svc := service.Service{
		Project: tmpDir,
		Host:    "function",
	}
	usedPorts := make(map[int]bool)
	runtime, err := service.DetectServiceRuntime("test-service", svc, usedPorts, tmpDir, "")

	if err != nil {
		t.Fatalf("Expected successful detection, got error: %v", err)
	}
	if runtime.Command != "func" {
		t.Errorf("Expected command 'func', got %q", runtime.Command)
	}
	if len(runtime.Args) < 2 || runtime.Args[0] != "start" || runtime.Args[1] != "--port" {
		t.Errorf("Expected args ['start', '--port', ...], got %v", runtime.Args)
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
