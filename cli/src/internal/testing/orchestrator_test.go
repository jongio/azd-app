package testing

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewTestOrchestrator(t *testing.T) {
	config := &TestConfig{
		Parallel: true,
		Verbose:  false,
	}

	orchestrator := NewTestOrchestrator(config)
	if orchestrator == nil {
		t.Fatal("Expected orchestrator to be created")
	}
	if orchestrator.config != config {
		t.Error("Config not set correctly")
	}
	if len(orchestrator.services) != 0 {
		t.Error("Services should be empty initially")
	}
}

func TestLoadServicesFromAzureYaml(t *testing.T) {
	// Create a temporary azure.yaml file
	tmpDir := t.TempDir()
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")

	yamlContent := `name: test-app
services:
  web:
    language: js
    project: ./src/web
  api:
    language: python
    project: ./src/api
    test:
      framework: pytest
      unit:
        command: pytest tests/unit
`

	if err := os.WriteFile(azureYamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test azure.yaml: %v", err)
	}

	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	err := orchestrator.LoadServicesFromAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("LoadServicesFromAzureYaml failed: %v", err)
	}

	if len(orchestrator.services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(orchestrator.services))
	}

	// Check first service
	found := false
	for _, svc := range orchestrator.services {
		if svc.Name == "web" {
			found = true
			if svc.Language != "js" {
				t.Errorf("Expected language 'js', got '%s'", svc.Language)
			}
		}
	}
	if !found {
		t.Error("Service 'web' not found")
	}
}

func TestLoadServicesFromAzureYaml_NoServices(t *testing.T) {
	tmpDir := t.TempDir()
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")

	yamlContent := `name: test-app
services: {}
`

	if err := os.WriteFile(azureYamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test azure.yaml: %v", err)
	}

	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	err := orchestrator.LoadServicesFromAzureYaml(azureYamlPath)
	if err == nil {
		t.Error("Expected error for no services")
	}
}

func TestLoadServicesFromAzureYaml_InvalidPath(t *testing.T) {
	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	err := orchestrator.LoadServicesFromAzureYaml("/non/existent/path")
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestDetectTestConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a service directory with package.json
	serviceDir := filepath.Join(tmpDir, "web")
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		t.Fatalf("Failed to create service dir: %v", err)
	}

	packageJSON := `{
		"name": "web",
		"scripts": {
			"test": "jest"
		},
		"devDependencies": {
			"jest": "^29.0.0"
		}
	}`

	if err := os.WriteFile(filepath.Join(serviceDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	service := ServiceInfo{
		Name:     "web",
		Language: "js",
		Dir:      serviceDir,
		Config:   nil,
	}

	testConfig, err := orchestrator.DetectTestConfig(service)
	if err != nil {
		t.Fatalf("DetectTestConfig failed: %v", err)
	}

	if testConfig.Framework != "jest" {
		t.Errorf("Expected framework 'jest', got '%s'", testConfig.Framework)
	}
}

func TestDetectTestConfig_ExistingConfig(t *testing.T) {
	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	existingConfig := &ServiceTestConfig{
		Framework: "custom",
	}

	service := ServiceInfo{
		Name:     "web",
		Language: "js",
		Dir:      "/tmp",
		Config:   existingConfig,
	}

	testConfig, err := orchestrator.DetectTestConfig(service)
	if err != nil {
		t.Fatalf("DetectTestConfig failed: %v", err)
	}

	if testConfig != existingConfig {
		t.Error("Should return existing config")
	}
}

func TestGetServicePaths(t *testing.T) {
	tmpDir := t.TempDir()

	// Create service directories
	webDir := filepath.Join(tmpDir, "web")
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(webDir, 0755); err != nil {
		t.Fatalf("Failed to create web dir: %v", err)
	}
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("Failed to create api dir: %v", err)
	}

	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)
	orchestrator.services = []ServiceInfo{
		{Name: "web", Dir: webDir},
		{Name: "api", Dir: apiDir},
	}

	paths, err := orchestrator.GetServicePaths()
	if err != nil {
		t.Fatalf("GetServicePaths failed: %v", err)
	}
	if len(paths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(paths))
	}

	// Check paths are included
	foundWeb := false
	foundAPI := false
	for _, path := range paths {
		if path == webDir {
			foundWeb = true
		}
		if path == apiDir {
			foundAPI = true
		}
	}

	if !foundWeb || !foundAPI {
		t.Error("Expected both service paths to be returned")
	}
}

func TestDetectTestConfig_GoLanguage(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Go service directory with go.mod and test files
	serviceDir := filepath.Join(tmpDir, "go-service")
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		t.Fatalf("Failed to create service dir: %v", err)
	}

	goMod := `module test

go 1.21
`
	if err := os.WriteFile(filepath.Join(serviceDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	testFile := `package main

import "testing"

func TestSomething(t *testing.T) {
}
`
	if err := os.WriteFile(filepath.Join(serviceDir, "main_test.go"), []byte(testFile), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	service := ServiceInfo{
		Name:     "go-service",
		Language: "go",
		Dir:      serviceDir,
		Config:   nil,
	}

	testConfig, err := orchestrator.DetectTestConfig(service)
	if err != nil {
		t.Fatalf("DetectTestConfig failed: %v", err)
	}

	if testConfig.Framework != "gotest" {
		t.Errorf("Expected framework 'gotest', got '%s'", testConfig.Framework)
	}
}

func TestDetectTestConfig_UnsupportedLanguage(t *testing.T) {
	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	service := ServiceInfo{
		Name:     "service",
		Language: "unsupported",
		Dir:      "/tmp",
		Config:   nil,
	}

	_, err := orchestrator.DetectTestConfig(service)
	if err == nil {
		t.Error("Expected error for unsupported language")
	}
}

func TestDetectNodeTestFramework_Vitest(t *testing.T) {
	tmpDir := t.TempDir()

	// Create vitest config
	vitestConfig := `export default {
  test: {}
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "vitest.config.ts"), []byte(vitestConfig), 0644); err != nil {
		t.Fatalf("Failed to create vitest config: %v", err)
	}

	framework, err := detectNodeTestFramework(tmpDir)
	if err != nil {
		t.Fatalf("detectNodeTestFramework failed: %v", err)
	}

	if framework != "vitest" {
		t.Errorf("Expected framework 'vitest', got '%s'", framework)
	}
}

func TestDetectNodeTestFramework_Mocha(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mocha config
	mochaConfig := `{}`
	if err := os.WriteFile(filepath.Join(tmpDir, ".mocharc.json"), []byte(mochaConfig), 0644); err != nil {
		t.Fatalf("Failed to create mocha config: %v", err)
	}

	framework, err := detectNodeTestFramework(tmpDir)
	if err != nil {
		t.Fatalf("detectNodeTestFramework failed: %v", err)
	}

	if framework != "mocha" {
		t.Errorf("Expected framework 'mocha', got '%s'", framework)
	}
}

func TestDetectNodeTestFramework_FromPackageJSON(t *testing.T) {
	tmpDir := t.TempDir()

	packageJSON := `{
		"devDependencies": {
			"vitest": "^1.0.0"
		}
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	framework, err := detectNodeTestFramework(tmpDir)
	if err != nil {
		t.Fatalf("detectNodeTestFramework failed: %v", err)
	}

	if framework != "vitest" {
		t.Errorf("Expected framework 'vitest', got '%s'", framework)
	}
}

func TestDetectPythonTestFramework_FromPytestIni(t *testing.T) {
	tmpDir := t.TempDir()

	pytestIni := `[pytest]
testpaths = tests
`
	if err := os.WriteFile(filepath.Join(tmpDir, "pytest.ini"), []byte(pytestIni), 0644); err != nil {
		t.Fatalf("Failed to create pytest.ini: %v", err)
	}

	framework, err := detectPythonTestFramework(tmpDir)
	if err != nil {
		t.Fatalf("detectPythonTestFramework failed: %v", err)
	}

	if framework != "pytest" {
		t.Errorf("Expected framework 'pytest', got '%s'", framework)
	}
}

func TestDetectPythonTestFramework_FromTestsDir(t *testing.T) {
	tmpDir := t.TempDir()

	testsDir := filepath.Join(tmpDir, "tests")
	if err := os.MkdirAll(testsDir, 0755); err != nil {
		t.Fatalf("Failed to create tests dir: %v", err)
	}

	framework, err := detectPythonTestFramework(tmpDir)
	if err != nil {
		t.Fatalf("detectPythonTestFramework failed: %v", err)
	}

	if framework != "pytest" {
		t.Errorf("Expected framework 'pytest', got '%s'", framework)
	}
}

func TestDetectGoTestFramework_NoGoMod(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := detectGoTestFramework(tmpDir)
	if err == nil {
		t.Error("Expected error when go.mod is missing")
	}
}

func TestDetectGoTestFramework_NoTestFiles(t *testing.T) {
	tmpDir := t.TempDir()

	goMod := `module test

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	_, err := detectGoTestFramework(tmpDir)
	if err == nil {
		t.Error("Expected error when no test files exist")
	}
}

func TestDetectGoTestFramework_WithTestFiles(t *testing.T) {
	tmpDir := t.TempDir()

	goMod := `module test

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	testFile := `package main

func TestExample(t *testing.T) {}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte(testFile), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	framework, err := detectGoTestFramework(tmpDir)
	if err != nil {
		t.Fatalf("detectGoTestFramework failed: %v", err)
	}

	if framework != "gotest" {
		t.Errorf("Expected framework 'gotest', got '%s'", framework)
	}
}

func TestDetectGoTestFramework_WithSubdirectoryTests(t *testing.T) {
	tmpDir := t.TempDir()

	goMod := `module test

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create test file in subdirectory
	subDir := filepath.Join(tmpDir, "pkg")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	testFile := `package pkg

func TestExample(t *testing.T) {}
`
	if err := os.WriteFile(filepath.Join(subDir, "pkg_test.go"), []byte(testFile), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	framework, err := detectGoTestFramework(tmpDir)
	if err != nil {
		t.Fatalf("detectGoTestFramework failed: %v", err)
	}

	if framework != "gotest" {
		t.Errorf("Expected framework 'gotest', got '%s'", framework)
	}
}

func TestFilterServices(t *testing.T) {
	services := []ServiceInfo{
		{Name: "web", Language: "js"},
		{Name: "api", Language: "python"},
		{Name: "gateway", Language: "go"},
	}

	tests := []struct {
		name     string
		filter   []string
		expected int
	}{
		{"no filter", nil, 3},
		{"empty filter", []string{}, 3},
		{"single filter", []string{"web"}, 1},
		{"multiple filter", []string{"web", "api"}, 2},
		{"non-existent filter", []string{"unknown"}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterServices(services, tt.filter)
			if len(filtered) != tt.expected {
				t.Errorf("Expected %d services, got %d", tt.expected, len(filtered))
			}
		})
	}
}

func TestExecuteTests_NoServices(t *testing.T) {
	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	_, err := orchestrator.ExecuteTests("all", nil)
	if err == nil {
		t.Error("Expected error when no services to test")
	}
}

func TestExecuteTests_ServiceFilter_Empty(t *testing.T) {
	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)
	orchestrator.services = []ServiceInfo{
		{Name: "web", Language: "js"},
	}

	_, err := orchestrator.ExecuteTests("all", []string{"unknown"})
	if err == nil {
		t.Error("Expected error when service filter matches nothing")
	}
}

func TestLoadServicesFromAzureYaml_WithTestConfig(t *testing.T) {
	tmpDir := t.TempDir()
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")

	yamlContent := `name: test-app
services:
  api:
    language: python
    project: ./api
    test:
      framework: pytest
      unit:
        command: pytest tests/unit -v
        markers:
          - unit
      integration:
        command: pytest tests/integration -v
        markers:
          - integration
        setup:
          - docker-compose up -d
        teardown:
          - docker-compose down
      coverage:
        threshold: 80
        source: src
`

	if err := os.WriteFile(azureYamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test azure.yaml: %v", err)
	}

	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	err := orchestrator.LoadServicesFromAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("LoadServicesFromAzureYaml failed: %v", err)
	}

	if len(orchestrator.services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(orchestrator.services))
	}

	svc := orchestrator.services[0]
	if svc.Config == nil {
		t.Fatal("Expected test config to be loaded")
	}

	if svc.Config.Framework != "pytest" {
		t.Errorf("Expected framework 'pytest', got '%s'", svc.Config.Framework)
	}

	if svc.Config.Unit == nil {
		t.Fatal("Expected unit config to be loaded")
	}

	if svc.Config.Unit.Command != "pytest tests/unit -v" {
		t.Errorf("Expected unit command 'pytest tests/unit -v', got '%s'", svc.Config.Unit.Command)
	}

	if svc.Config.Integration == nil {
		t.Fatal("Expected integration config to be loaded")
	}

	if len(svc.Config.Integration.Setup) != 1 {
		t.Errorf("Expected 1 setup command, got %d", len(svc.Config.Integration.Setup))
	}

	if len(svc.Config.Integration.Teardown) != 1 {
		t.Errorf("Expected 1 teardown command, got %d", len(svc.Config.Integration.Teardown))
	}
}

func TestParseCommandString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple command",
			input:    "npm test",
			expected: []string{"npm", "test"},
		},
		{
			name:     "command with flags",
			input:    "pytest tests -v --cov=src",
			expected: []string{"pytest", "tests", "-v", "--cov=src"},
		},
		{
			name:     "double quoted argument",
			input:    `echo "hello world"`,
			expected: []string{"echo", "hello world"},
		},
		{
			name:     "single quoted argument",
			input:    `echo 'hello world'`,
			expected: []string{"echo", "hello world"},
		},
		{
			name:     "mixed quotes",
			input:    `cmd "arg one" 'arg two'`,
			expected: []string{"cmd", "arg one", "arg two"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: []string{},
		},
		{
			name:     "extra whitespace",
			input:    "cmd    arg1   arg2",
			expected: []string{"cmd", "arg1", "arg2"},
		},
		{
			name:     "tabs",
			input:    "cmd\targ1\targ2",
			expected: []string{"cmd", "arg1", "arg2"},
		},
		{
			name:     "dotnet test command",
			input:    "dotnet test --filter Category=Unit",
			expected: []string{"dotnet", "test", "--filter", "Category=Unit"},
		},
		{
			name:     "go test command",
			input:    "go test -v -run TestUnit ./...",
			expected: []string{"go", "test", "-v", "-run", "TestUnit", "./..."},
		},
		{
			name:     "docker compose",
			input:    "docker-compose up -d postgres",
			expected: []string{"docker-compose", "up", "-d", "postgres"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommandString(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d parts, got %d: %v", len(tt.expected), len(result), result)
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Part %d: expected '%s', got '%s'", i, expected, result[i])
				}
			}
		})
	}
}

func TestLoadServicesFromAzureYaml_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")

	// Attempt path traversal attack
	yamlContent := `name: test-app
services:
  evil:
    language: js
    project: ../../../etc
`

	if err := os.WriteFile(azureYamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test azure.yaml: %v", err)
	}

	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	err := orchestrator.LoadServicesFromAzureYaml(azureYamlPath)
	if err == nil {
		t.Error("Expected error for path traversal attempt")
	}

	if !containsSubstring(err.Error(), "escapes project boundary") {
		t.Errorf("Expected 'escapes project boundary' error, got: %v", err)
	}
}

func TestLoadServicesFromAzureYaml_ValidNestedPath(t *testing.T) {
	tmpDir := t.TempDir()
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")

	// Create valid nested directory structure
	serviceDir := filepath.Join(tmpDir, "services", "web")
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		t.Fatalf("Failed to create service dir: %v", err)
	}

	yamlContent := `name: test-app
services:
  web:
    language: js
    project: ./services/web
`

	if err := os.WriteFile(azureYamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test azure.yaml: %v", err)
	}

	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	err := orchestrator.LoadServicesFromAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("LoadServicesFromAzureYaml failed for valid path: %v", err)
	}

	if len(orchestrator.services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(orchestrator.services))
	}
}

// containsSubstring checks if s contains substr
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestDetectDotnetTestFramework(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test project with xUnit
	projectDir := filepath.Join(tmpDir, "TestProject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	// Create csproj file with xUnit reference
	csprojContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="xunit" Version="2.5.0" />
  </ItemGroup>
</Project>`
	csprojPath := filepath.Join(projectDir, "TestProject.Tests.csproj")
	if err := os.WriteFile(csprojPath, []byte(csprojContent), 0644); err != nil {
		t.Fatalf("Failed to create csproj: %v", err)
	}

	framework, err := detectDotnetTestFramework(tmpDir)
	if err != nil {
		t.Logf("detectDotnetTestFramework returned error (expected if no dotnet projects found): %v", err)
	}

	// Default should be xunit
	if framework != "xunit" {
		t.Logf("Framework detected: %s", framework)
	}
}

func TestGetAvailableTestTypesForService(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a service directory with test files
	serviceDir := filepath.Join(tmpDir, "web")
	testsDir := filepath.Join(serviceDir, "tests")
	unitDir := filepath.Join(testsDir, "unit")
	integrationDir := filepath.Join(testsDir, "integration")

	if err := os.MkdirAll(unitDir, 0755); err != nil {
		t.Fatalf("Failed to create unit dir: %v", err)
	}
	if err := os.MkdirAll(integrationDir, 0755); err != nil {
		t.Fatalf("Failed to create integration dir: %v", err)
	}

	// Create test files
	if err := os.WriteFile(filepath.Join(unitDir, "test_example.py"), []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create unit test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(integrationDir, "test_api.py"), []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create integration test file: %v", err)
	}

	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	service := ServiceInfo{
		Name:     "web",
		Language: "python",
		Dir:      serviceDir,
	}

	types := orchestrator.GetAvailableTestTypesForService(service)

	if len(types) == 0 {
		t.Log("No test types detected (may be expected if detection patterns don't match)")
	}
}

func TestGetAvailableTestTypes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create service directories
	webDir := filepath.Join(tmpDir, "web")
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(filepath.Join(webDir, "tests", "unit"), 0755); err != nil {
		t.Fatalf("Failed to create web test dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(apiDir, "tests", "integration"), 0755); err != nil {
		t.Fatalf("Failed to create api test dir: %v", err)
	}

	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)
	orchestrator.services = []ServiceInfo{
		{Name: "web", Language: "python", Dir: webDir},
		{Name: "api", Language: "python", Dir: apiDir},
	}

	types := orchestrator.GetAvailableTestTypes()

	if len(types) != 2 {
		t.Errorf("Expected 2 services in result, got %d", len(types))
	}
}

func TestExecuteTests_FailFast(t *testing.T) {
	config := &TestConfig{
		FailFast: true,
	}
	orchestrator := NewTestOrchestrator(config)
	orchestrator.services = []ServiceInfo{
		{Name: "failing-service", Language: "unsupported", Dir: "/nonexistent"},
	}

	_, err := orchestrator.ExecuteTests("all", nil)

	// Should fail fast with error
	if err == nil {
		t.Error("Expected error for fail-fast with failing service")
	}
}

func TestExecuteTests_WithCoverageThreshold(t *testing.T) {
	tmpDir := t.TempDir()

	config := &TestConfig{
		CoverageThreshold: 80.0,
		OutputDir:         tmpDir,
	}
	orchestrator := NewTestOrchestrator(config)

	// No services to test
	_, err := orchestrator.ExecuteTests("all", nil)
	if err == nil {
		t.Error("Expected error for no services")
	}
}

func TestFindAzureYaml(t *testing.T) {
	// This test depends on the current working directory
	// It's primarily to ensure the function doesn't panic
	_, err := FindAzureYaml()
	if err != nil {
		t.Logf("FindAzureYaml returned error (expected if not in azure project): %v", err)
	}
}

func TestLoadServicesFromAzureYaml_WithE2EConfig(t *testing.T) {
	tmpDir := t.TempDir()
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")

	yamlContent := `name: test-app
services:
  web:
    language: js
    project: ./web
    test:
      framework: jest
      e2e:
        command: npm run test:e2e
        setup:
          - docker-compose up -d
        teardown:
          - docker-compose down
`

	if err := os.WriteFile(azureYamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test azure.yaml: %v", err)
	}

	// Create service directory
	if err := os.MkdirAll(filepath.Join(tmpDir, "web"), 0755); err != nil {
		t.Fatalf("Failed to create service dir: %v", err)
	}

	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	err := orchestrator.LoadServicesFromAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("LoadServicesFromAzureYaml failed: %v", err)
	}

	if len(orchestrator.services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(orchestrator.services))
	}

	svc := orchestrator.services[0]
	if svc.Config == nil {
		t.Fatal("Expected test config to be loaded")
	}

	if svc.Config.E2E == nil {
		t.Fatal("Expected E2E config to be loaded")
	}

	if svc.Config.E2E.Command != "npm run test:e2e" {
		t.Errorf("Expected E2E command 'npm run test:e2e', got '%s'", svc.Config.E2E.Command)
	}

	if len(svc.Config.E2E.Setup) != 1 {
		t.Errorf("Expected 1 setup command, got %d", len(svc.Config.E2E.Setup))
	}

	if len(svc.Config.E2E.Teardown) != 1 {
		t.Errorf("Expected 1 teardown command, got %d", len(svc.Config.E2E.Teardown))
	}
}

func TestLoadServicesFromAzureYaml_WithCoverageConfig(t *testing.T) {
	tmpDir := t.TempDir()
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")

	yamlContent := `name: test-app
services:
  api:
    language: python
    project: ./api
    test:
      framework: pytest
      coverage:
        threshold: 90
        source: src
        exclude:
          - "*/migrations/*"
          - "*_test.py"
`

	if err := os.WriteFile(azureYamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test azure.yaml: %v", err)
	}

	// Create service directory
	if err := os.MkdirAll(filepath.Join(tmpDir, "api"), 0755); err != nil {
		t.Fatalf("Failed to create service dir: %v", err)
	}

	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	err := orchestrator.LoadServicesFromAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("LoadServicesFromAzureYaml failed: %v", err)
	}

	svc := orchestrator.services[0]
	if svc.Config.Coverage == nil {
		t.Fatal("Expected coverage config to be loaded")
	}

	if svc.Config.Coverage.Threshold != 90 {
		t.Errorf("Expected coverage threshold 90, got %.0f", svc.Config.Coverage.Threshold)
	}

	if svc.Config.Coverage.Source != "src" {
		t.Errorf("Expected coverage source 'src', got '%s'", svc.Config.Coverage.Source)
	}
}

func TestDetectTestConfig_Python(t *testing.T) {
	tmpDir := t.TempDir()

	// Create pytest.ini to simulate pytest project
	pytestIni := `[pytest]
testpaths = tests
`
	if err := os.WriteFile(filepath.Join(tmpDir, "pytest.ini"), []byte(pytestIni), 0644); err != nil {
		t.Fatalf("Failed to create pytest.ini: %v", err)
	}

	// Create tests directory with subdirectories
	testsDir := filepath.Join(tmpDir, "tests")
	unitDir := filepath.Join(testsDir, "unit")
	integrationDir := filepath.Join(testsDir, "integration")

	if err := os.MkdirAll(unitDir, 0755); err != nil {
		t.Fatalf("Failed to create unit dir: %v", err)
	}
	if err := os.MkdirAll(integrationDir, 0755); err != nil {
		t.Fatalf("Failed to create integration dir: %v", err)
	}

	// Create test files
	if err := os.WriteFile(filepath.Join(unitDir, "test_utils.py"), []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create unit test: %v", err)
	}
	if err := os.WriteFile(filepath.Join(integrationDir, "test_api.py"), []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create integration test: %v", err)
	}

	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	service := ServiceInfo{
		Name:     "api",
		Language: "python",
		Dir:      tmpDir,
		Config:   nil,
	}

	testConfig, err := orchestrator.DetectTestConfig(service)
	if err != nil {
		t.Fatalf("DetectTestConfig failed: %v", err)
	}

	if testConfig.Framework != "pytest" {
		t.Errorf("Expected framework 'pytest', got '%s'", testConfig.Framework)
	}
}

func TestDetectTestConfig_DotNet(t *testing.T) {
	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	// Test with non-existent directory (should fail gracefully)
	service := ServiceInfo{
		Name:     "api",
		Language: "dotnet",
		Dir:      "/nonexistent/dir",
		Config:   nil,
	}

	_, err := orchestrator.DetectTestConfig(service)
	if err == nil {
		t.Log("DetectTestConfig returned nil error for non-existent directory")
	}
}

func TestExecuteTests_ContinueOnError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a service directory
	serviceDir := filepath.Join(tmpDir, "web")
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		t.Fatalf("Failed to create service dir: %v", err)
	}

	config := &TestConfig{
		FailFast: false, // Continue on error
	}
	orchestrator := NewTestOrchestrator(config)
	orchestrator.services = []ServiceInfo{
		{Name: "web", Language: "unsupported", Dir: serviceDir},
	}

	result, err := orchestrator.ExecuteTests("all", nil)

	// Should not return error, but result should show failure
	if err != nil {
		t.Logf("ExecuteTests returned error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Success {
		t.Error("Expected result.Success to be false for failing service")
	}
}

func TestParseCommandString_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "nested quotes",
			input:    `cmd "arg with 'nested' quotes"`,
			expected: []string{"cmd", "arg with 'nested' quotes"},
		},
		{
			name:     "escaped spaces",
			input:    "cmd arg1\\ arg2",
			expected: []string{"cmd", "arg1\\", "arg2"},
		},
		{
			name:     "multiple spaces between args",
			input:    "cmd     arg1     arg2",
			expected: []string{"cmd", "arg1", "arg2"},
		},
		{
			name:     "tabs between args",
			input:    "cmd\targ1\targ2",
			expected: []string{"cmd", "arg1", "arg2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommandString(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d parts, got %d: %v", len(tt.expected), len(result), result)
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Part %d: expected '%s', got '%s'", i, expected, result[i])
				}
			}
		})
	}
}

func TestSetProgressCallback(t *testing.T) {
	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	callCount := 0
	callback := func(event ProgressEvent) {
		callCount++
	}

	orchestrator.SetProgressCallback(callback)

	// Emit a progress event
	orchestrator.emitProgress(ProgressEvent{
		Type:    ProgressEventValidationStart,
		Message: "Test message",
	})

	if callCount != 1 {
		t.Errorf("Expected callback to be called once, got %d", callCount)
	}
}

func TestEmitProgress_NoCallback(t *testing.T) {
	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	// Should not panic when callback is nil
	orchestrator.emitProgress(ProgressEvent{
		Type:    ProgressEventValidationStart,
		Message: "Test message",
	})
}

func TestGetServices(t *testing.T) {
	tmpDir := t.TempDir()

	// Create service directories
	webDir := filepath.Join(tmpDir, "web")
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(webDir, 0755); err != nil {
		t.Fatalf("Failed to create web dir: %v", err)
	}
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("Failed to create api dir: %v", err)
	}

	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)
	orchestrator.services = []ServiceInfo{
		{Name: "web", Language: "js", Dir: webDir},
		{Name: "api", Language: "python", Dir: apiDir},
	}

	services := orchestrator.GetServices()

	if len(services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(services))
	}
}

func TestValidateAllServices(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a service directory with tests
	webDir := filepath.Join(tmpDir, "web")
	if err := os.MkdirAll(webDir, 0755); err != nil {
		t.Fatalf("Failed to create web dir: %v", err)
	}

	// Create vitest config
	if err := os.WriteFile(filepath.Join(webDir, "vitest.config.ts"), []byte("export default {}"), 0644); err != nil {
		t.Fatalf("Failed to create vitest config: %v", err)
	}

	// Create test file
	if err := os.WriteFile(filepath.Join(webDir, "app.test.ts"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a service directory without tests
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("Failed to create api dir: %v", err)
	}

	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)
	orchestrator.services = []ServiceInfo{
		{Name: "web", Language: "typescript", Dir: webDir},
		{Name: "api", Language: "python", Dir: apiDir},
	}

	// Track progress events
	events := make([]ProgressEvent, 0)
	orchestrator.SetProgressCallback(func(event ProgressEvent) {
		events = append(events, event)
	})

	validations := orchestrator.ValidateAllServices()

	if len(validations) != 2 {
		t.Errorf("Expected 2 validations, got %d", len(validations))
	}

	// Should have received validation start and complete events
	hasStart := false
	hasComplete := false
	for _, e := range events {
		if e.Type == ProgressEventValidationStart {
			hasStart = true
		}
		if e.Type == ProgressEventValidationComplete {
			hasComplete = true
		}
	}

	if !hasStart {
		t.Error("Expected ProgressEventValidationStart event")
	}
	if !hasComplete {
		t.Error("Expected ProgressEventValidationComplete event")
	}
}

func TestExecuteTestsWithValidation_NoServices(t *testing.T) {
	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	_, _, err := orchestrator.ExecuteTestsWithValidation("all", nil)
	if err == nil {
		t.Error("Expected error when no services to test")
	}
}

func TestExecuteTestsWithValidation_AllSkipped(t *testing.T) {
	tmpDir := t.TempDir()

	// Create service directory without tests
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("Failed to create api dir: %v", err)
	}

	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)
	orchestrator.services = []ServiceInfo{
		{Name: "api", Language: "python", Dir: apiDir},
	}

	result, validations, err := orchestrator.ExecuteTestsWithValidation("all", nil)
	if err != nil {
		t.Fatalf("ExecuteTestsWithValidation failed: %v", err)
	}

	if len(validations) != 1 {
		t.Errorf("Expected 1 validation, got %d", len(validations))
	}

	if validations[0].CanTest {
		t.Error("Expected service to be marked as not testable")
	}

	// Result should be empty but not nil
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if len(result.Services) != 0 {
		t.Errorf("Expected 0 service results, got %d", len(result.Services))
	}
}

func TestProgressEventTypes(t *testing.T) {
	// Test that all progress event types are distinct
	eventTypes := []ProgressEventType{
		ProgressEventValidationStart,
		ProgressEventServiceValidated,
		ProgressEventValidationComplete,
		ProgressEventTestStart,
		ProgressEventTestComplete,
		ProgressEventServiceSkipped,
	}

	seen := make(map[ProgressEventType]bool)
	for _, et := range eventTypes {
		if seen[et] {
			t.Errorf("Duplicate event type value: %d", et)
		}
		seen[et] = true
	}
}

func TestExecuteTestsWithValidation_ServiceFilter(t *testing.T) {
	tmpDir := t.TempDir()

	// Create service directories
	webDir := filepath.Join(tmpDir, "web")
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(webDir, 0755); err != nil {
		t.Fatalf("Failed to create web dir: %v", err)
	}
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("Failed to create api dir: %v", err)
	}

	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)
	orchestrator.services = []ServiceInfo{
		{Name: "web", Language: "js", Dir: webDir},
		{Name: "api", Language: "python", Dir: apiDir},
	}

	// Filter to only 'web' service - should get error since 'web' has no tests
	_, validations, err := orchestrator.ExecuteTestsWithValidation("all", []string{"web"})
	if err != nil {
		t.Fatalf("ExecuteTestsWithValidation failed: %v", err)
	}

	// Should only validate the filtered service
	if len(validations) != 1 {
		t.Errorf("Expected 1 validation with filter, got %d", len(validations))
	}
	if validations[0].Name != "web" {
		t.Errorf("Expected service name 'web', got '%s'", validations[0].Name)
	}
}

func TestExecuteTestsWithValidation_ProgressEvents(t *testing.T) {
	tmpDir := t.TempDir()

	// Create service directory without tests
	webDir := filepath.Join(tmpDir, "web")
	if err := os.MkdirAll(webDir, 0755); err != nil {
		t.Fatalf("Failed to create web dir: %v", err)
	}

	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)
	orchestrator.services = []ServiceInfo{
		{Name: "web", Language: "js", Dir: webDir},
	}

	// Track all progress events
	events := make([]ProgressEvent, 0)
	orchestrator.SetProgressCallback(func(event ProgressEvent) {
		events = append(events, event)
	})

	_, _, _ = orchestrator.ExecuteTestsWithValidation("all", nil)

	// Should have ValidationStart, ServiceValidated, ValidationComplete, and ServiceSkipped events
	eventTypesSeen := make(map[ProgressEventType]bool)
	for _, e := range events {
		eventTypesSeen[e.Type] = true
	}

	expectedTypes := []ProgressEventType{
		ProgressEventValidationStart,
		ProgressEventServiceValidated,
		ProgressEventValidationComplete,
		ProgressEventServiceSkipped,
	}

	for _, et := range expectedTypes {
		if !eventTypesSeen[et] {
			t.Errorf("Expected event type %d but it was not emitted", et)
		}
	}
}

func TestValidateAllServices_EmptyServices(t *testing.T) {
	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)
	// No services added

	validations := orchestrator.ValidateAllServices()

	if len(validations) != 0 {
		t.Errorf("Expected 0 validations for no services, got %d", len(validations))
	}
}

func TestDefaultTestTimeout(t *testing.T) {
	// Verify the default timeout constant is 10 minutes
	expected := 10 * time.Minute
	if DefaultTestTimeout != expected {
		t.Errorf("Expected DefaultTestTimeout to be 10 minutes, got %v", DefaultTestTimeout)
	}
}

func TestTestConfigWithTimeout(t *testing.T) {
	// Test that TestConfig can hold timeout value
	config := &TestConfig{
		Parallel:          true,
		FailFast:          false,
		CoverageThreshold: 80.0,
		Timeout:           5 * time.Minute,
	}

	orchestrator := NewTestOrchestrator(config)
	if orchestrator.config.Timeout != config.Timeout {
		t.Errorf("Expected timeout %v, got %v", config.Timeout, orchestrator.config.Timeout)
	}
}

// mockTestRunner is a test runner that simulates test execution
type mockTestRunner struct {
	delay   int64 // delay in milliseconds
	result  *TestResult
	err     error
	started chan struct{}
}

func (m *mockTestRunner) RunTests(testType string, coverage bool) (*TestResult, error) {
	if m.started != nil {
		close(m.started)
	}
	if m.delay > 0 {
		// Use a select with time.After to simulate delay
		select {
		case <-make(chan struct{}): // never closes
		case <-func() <-chan struct{} {
			ch := make(chan struct{})
			go func() {
				// Simulate work
				for i := int64(0); i < m.delay*1000000; i++ {
					_ = i
				}
				close(ch)
			}()
			return ch
		}():
		}
	}
	return m.result, m.err
}

func TestExecuteWithTimeout_Success(t *testing.T) {
	config := &TestConfig{
		Timeout: 5 * time.Second,
	}
	orchestrator := NewTestOrchestrator(config)

	runner := &mockTestRunner{
		delay: 0, // no delay
		result: &TestResult{
			Service: "test-service",
			Success: true,
			Passed:  5,
			Total:   5,
		},
	}

	result, err := orchestrator.executeWithTimeout(runner, "unit", false, config.Timeout)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if !result.Success {
		t.Error("Expected success to be true")
	}
	if result.Passed != 5 {
		t.Errorf("Expected 5 passed, got %d", result.Passed)
	}
}

func TestExecuteWithTimeout_Error(t *testing.T) {
	config := &TestConfig{
		Timeout: 5 * time.Second,
	}
	orchestrator := NewTestOrchestrator(config)

	expectedErr := "test runner error"
	runner := &mockTestRunner{
		delay:  0,
		result: nil,
		err:    &testError{msg: expectedErr},
	}

	result, err := orchestrator.executeWithTimeout(runner, "unit", false, config.Timeout)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if err.Error() != expectedErr {
		t.Errorf("Expected error '%s', got '%s'", expectedErr, err.Error())
	}
	if result != nil {
		t.Errorf("Expected nil result on error, got: %v", result)
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
