package testing

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateService_NodeJS_WithVitest(t *testing.T) {
	tmpDir := t.TempDir()

	// Create vitest config
	vitestConfig := `export default { test: {} }`
	if err := os.WriteFile(filepath.Join(tmpDir, "vitest.config.ts"), []byte(vitestConfig), 0644); err != nil {
		t.Fatalf("Failed to create vitest config: %v", err)
	}

	// Create test file
	testsDir := filepath.Join(tmpDir, "tests")
	if err := os.MkdirAll(testsDir, 0755); err != nil {
		t.Fatalf("Failed to create tests dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testsDir, "example.test.ts"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	service := ServiceInfo{
		Name:     "web",
		Language: "typescript",
		Dir:      tmpDir,
	}

	validation := ValidateService(service)

	if !validation.CanTest {
		t.Errorf("Expected CanTest to be true, got false. SkipReason: %s", validation.SkipReason)
	}
	if validation.Framework != "vitest" {
		t.Errorf("Expected framework 'vitest', got '%s'", validation.Framework)
	}
	if validation.Name != "web" {
		t.Errorf("Expected name 'web', got '%s'", validation.Name)
	}
}

func TestValidateService_NodeJS_WithJest(t *testing.T) {
	tmpDir := t.TempDir()

	// Create jest config
	jestConfig := `module.exports = { testEnvironment: 'node' }`
	if err := os.WriteFile(filepath.Join(tmpDir, "jest.config.js"), []byte(jestConfig), 0644); err != nil {
		t.Fatalf("Failed to create jest config: %v", err)
	}

	service := ServiceInfo{
		Name:     "api",
		Language: "js",
		Dir:      tmpDir,
	}

	validation := ValidateService(service)

	if validation.Framework != "jest" {
		t.Errorf("Expected framework 'jest', got '%s'", validation.Framework)
	}
}

func TestValidateService_NodeJS_WithPackageJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create package.json with test script
	packageJSON := `{
		"name": "test-app",
		"scripts": {
			"test": "jest"
		},
		"devDependencies": {
			"jest": "^29.0.0"
		}
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	service := ServiceInfo{
		Name:     "web",
		Language: "javascript",
		Dir:      tmpDir,
	}

	validation := ValidateService(service)

	if !validation.CanTest {
		t.Errorf("Expected CanTest to be true, got false. SkipReason: %s", validation.SkipReason)
	}
	if validation.Framework != "jest" {
		t.Errorf("Expected framework 'jest', got '%s'", validation.Framework)
	}
}

func TestValidateService_NodeJS_NoTests(t *testing.T) {
	tmpDir := t.TempDir()

	// Create package.json without test script
	packageJSON := `{
		"name": "test-app",
		"scripts": {
			"start": "node index.js"
		}
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	service := ServiceInfo{
		Name:     "admin-server",
		Language: "js",
		Dir:      tmpDir,
	}

	validation := ValidateService(service)

	if validation.CanTest {
		t.Error("Expected CanTest to be false")
	}
	if validation.SkipReason == "" {
		t.Error("Expected SkipReason to be set")
	}
}

func TestValidateService_Python_WithPytest(t *testing.T) {
	tmpDir := t.TempDir()

	// Create pytest.ini
	pytestIni := `[pytest]
testpaths = tests
`
	if err := os.WriteFile(filepath.Join(tmpDir, "pytest.ini"), []byte(pytestIni), 0644); err != nil {
		t.Fatalf("Failed to create pytest.ini: %v", err)
	}

	// Create tests directory with test file
	testsDir := filepath.Join(tmpDir, "tests")
	if err := os.MkdirAll(testsDir, 0755); err != nil {
		t.Fatalf("Failed to create tests dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testsDir, "test_example.py"), []byte("def test_one(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	service := ServiceInfo{
		Name:     "api",
		Language: "python",
		Dir:      tmpDir,
	}

	validation := ValidateService(service)

	if !validation.CanTest {
		t.Errorf("Expected CanTest to be true, got false. SkipReason: %s", validation.SkipReason)
	}
	if validation.Framework != "pytest" {
		t.Errorf("Expected framework 'pytest', got '%s'", validation.Framework)
	}
	if validation.TestFiles == 0 {
		t.Error("Expected TestFiles > 0")
	}
}

func TestValidateService_Python_WithTestsDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create tests directory
	testsDir := filepath.Join(tmpDir, "tests")
	if err := os.MkdirAll(testsDir, 0755); err != nil {
		t.Fatalf("Failed to create tests dir: %v", err)
	}

	service := ServiceInfo{
		Name:     "service",
		Language: "py",
		Dir:      tmpDir,
	}

	validation := ValidateService(service)

	if !validation.CanTest {
		t.Errorf("Expected CanTest to be true, got false. SkipReason: %s", validation.SkipReason)
	}
	if validation.Framework != "pytest" {
		t.Errorf("Expected framework 'pytest', got '%s'", validation.Framework)
	}
}

func TestValidateService_Python_NoTests(t *testing.T) {
	tmpDir := t.TempDir()

	// Create just a main.py
	if err := os.WriteFile(filepath.Join(tmpDir, "main.py"), []byte("print('hello')"), 0644); err != nil {
		t.Fatalf("Failed to create main.py: %v", err)
	}

	service := ServiceInfo{
		Name:     "service",
		Language: "python",
		Dir:      tmpDir,
	}

	validation := ValidateService(service)

	if validation.CanTest {
		t.Error("Expected CanTest to be false")
	}
	if validation.SkipReason == "" {
		t.Error("Expected SkipReason to be set")
	}
}

func TestValidateService_Go_WithTests(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module example.com/test

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create test file
	testFile := `package main

import "testing"

func TestExample(t *testing.T) {}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte(testFile), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	service := ServiceInfo{
		Name:     "api",
		Language: "go",
		Dir:      tmpDir,
	}

	validation := ValidateService(service)

	if !validation.CanTest {
		t.Errorf("Expected CanTest to be true, got false. SkipReason: %s", validation.SkipReason)
	}
	if validation.Framework != "gotest" {
		t.Errorf("Expected framework 'gotest', got '%s'", validation.Framework)
	}
	if validation.TestFiles != 1 {
		t.Errorf("Expected TestFiles to be 1, got %d", validation.TestFiles)
	}
}

func TestValidateService_Go_NoGoMod(t *testing.T) {
	tmpDir := t.TempDir()

	service := ServiceInfo{
		Name:     "api",
		Language: "golang",
		Dir:      tmpDir,
	}

	validation := ValidateService(service)

	if validation.CanTest {
		t.Error("Expected CanTest to be false")
	}
	if validation.SkipReason != "No go.mod file found" {
		t.Errorf("Expected SkipReason 'No go.mod file found', got '%s'", validation.SkipReason)
	}
}

func TestValidateService_Go_NoTestFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module example.com/test

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create non-test file
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create main.go: %v", err)
	}

	service := ServiceInfo{
		Name:     "api",
		Language: "go",
		Dir:      tmpDir,
	}

	validation := ValidateService(service)

	if validation.CanTest {
		t.Error("Expected CanTest to be false")
	}
	if validation.SkipReason != "No *_test.go files found" {
		t.Errorf("Expected SkipReason 'No *_test.go files found', got '%s'", validation.SkipReason)
	}
}

func TestValidateService_Dotnet_WithXUnit(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test project directory
	testProjDir := filepath.Join(tmpDir, "MyApp.Tests")
	if err := os.MkdirAll(testProjDir, 0755); err != nil {
		t.Fatalf("Failed to create test project dir: %v", err)
	}

	// Create csproj with xunit
	csproj := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="xunit" Version="2.5.0" />
  </ItemGroup>
</Project>`
	if err := os.WriteFile(filepath.Join(testProjDir, "MyApp.Tests.csproj"), []byte(csproj), 0644); err != nil {
		t.Fatalf("Failed to create csproj: %v", err)
	}

	service := ServiceInfo{
		Name:     "myapp",
		Language: "dotnet",
		Dir:      tmpDir,
	}

	validation := ValidateService(service)

	if !validation.CanTest {
		t.Errorf("Expected CanTest to be true, got false. SkipReason: %s", validation.SkipReason)
	}
	if validation.Framework != "xunit" {
		t.Errorf("Expected framework 'xunit', got '%s'", validation.Framework)
	}
	if validation.TestFiles != 1 {
		t.Errorf("Expected TestFiles to be 1, got %d", validation.TestFiles)
	}
}

func TestValidateService_Dotnet_NoTestProjects(t *testing.T) {
	tmpDir := t.TempDir()

	// Create non-test project
	projDir := filepath.Join(tmpDir, "MyApp")
	if err := os.MkdirAll(projDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	csproj := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`
	if err := os.WriteFile(filepath.Join(projDir, "MyApp.csproj"), []byte(csproj), 0644); err != nil {
		t.Fatalf("Failed to create csproj: %v", err)
	}

	service := ServiceInfo{
		Name:     "myapp",
		Language: "csharp",
		Dir:      tmpDir,
	}

	validation := ValidateService(service)

	if validation.CanTest {
		t.Error("Expected CanTest to be false")
	}
	if validation.SkipReason == "" {
		t.Error("Expected SkipReason to be set")
	}
}

func TestValidateService_UnsupportedLanguage(t *testing.T) {
	tmpDir := t.TempDir()

	service := ServiceInfo{
		Name:     "service",
		Language: "rust",
		Dir:      tmpDir,
	}

	validation := ValidateService(service)

	if validation.CanTest {
		t.Error("Expected CanTest to be false")
	}
	if validation.SkipReason != "Unsupported language: rust" {
		t.Errorf("Expected SkipReason 'Unsupported language: rust', got '%s'", validation.SkipReason)
	}
}

func TestValidateService_NonExistentDirectory(t *testing.T) {
	service := ServiceInfo{
		Name:     "service",
		Language: "js",
		Dir:      "/nonexistent/path/12345",
	}

	validation := ValidateService(service)

	if validation.CanTest {
		t.Error("Expected CanTest to be false")
	}
	if validation.SkipReason == "" {
		t.Error("Expected SkipReason to be set")
	}
}

func TestValidateServices(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two service directories
	webDir := filepath.Join(tmpDir, "web")
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(webDir, 0755); err != nil {
		t.Fatalf("Failed to create web dir: %v", err)
	}
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("Failed to create api dir: %v", err)
	}

	// Add vitest config to web
	if err := os.WriteFile(filepath.Join(webDir, "vitest.config.ts"), []byte("export default {}"), 0644); err != nil {
		t.Fatalf("Failed to create vitest config: %v", err)
	}

	// Add test file to web
	if err := os.WriteFile(filepath.Join(webDir, "app.test.ts"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// api has no tests

	services := []ServiceInfo{
		{Name: "web", Language: "typescript", Dir: webDir},
		{Name: "api", Language: "python", Dir: apiDir},
	}

	validations := ValidateServices(services)

	if len(validations) != 2 {
		t.Errorf("Expected 2 validations, got %d", len(validations))
	}
}

func TestGetTestableServices(t *testing.T) {
	validations := []ServiceValidation{
		{Name: "web", CanTest: true, Framework: "vitest"},
		{Name: "api", CanTest: false, SkipReason: "No tests"},
		{Name: "gateway", CanTest: true, Framework: "gotest"},
	}

	testable := GetTestableServices(validations)

	if len(testable) != 2 {
		t.Errorf("Expected 2 testable services, got %d", len(testable))
	}

	// Check both testable services are returned
	names := make(map[string]bool)
	for _, v := range testable {
		names[v.Name] = true
	}
	if !names["web"] || !names["gateway"] {
		t.Error("Expected 'web' and 'gateway' to be testable")
	}
}

func TestGetSkippedServices(t *testing.T) {
	validations := []ServiceValidation{
		{Name: "web", CanTest: true, Framework: "vitest"},
		{Name: "api", CanTest: false, SkipReason: "No tests"},
		{Name: "gateway", CanTest: true, Framework: "gotest"},
	}

	skipped := GetSkippedServices(validations)

	if len(skipped) != 1 {
		t.Errorf("Expected 1 skipped service, got %d", len(skipped))
	}

	if skipped[0].Name != "api" {
		t.Errorf("Expected skipped service 'api', got '%s'", skipped[0].Name)
	}
}

func TestCountTestFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	testFiles := []string{
		"app.test.ts",
		"utils.test.ts",
		"helper.spec.js",
	}

	for _, f := range testFiles {
		if err := os.WriteFile(filepath.Join(tmpDir, f), []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", f, err)
		}
	}

	// Create nested directory with test file
	nestedDir := filepath.Join(tmpDir, "src", "components")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create nested dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "button.test.tsx"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create nested test file: %v", err)
	}

	patterns := []string{
		"*.test.ts",
		"*.spec.js",
		"**/*.test.tsx",
	}

	count := countTestFiles(tmpDir, patterns)

	// Should find app.test.ts, utils.test.ts, helper.spec.js, and button.test.tsx
	if count < 3 {
		t.Errorf("Expected at least 3 test files, got %d", count)
	}
}

func TestCountTestFiles_SkipsNodeModules(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file in root
	if err := os.WriteFile(filepath.Join(tmpDir, "app.test.ts"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create test file in node_modules (should be skipped)
	nodeModulesDir := filepath.Join(tmpDir, "node_modules", "some-package")
	if err := os.MkdirAll(nodeModulesDir, 0755); err != nil {
		t.Fatalf("Failed to create node_modules dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nodeModulesDir, "index.test.js"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file in node_modules: %v", err)
	}

	patterns := []string{"**/*.test.ts", "**/*.test.js"}
	count := countTestFiles(tmpDir, patterns)

	// Should only find app.test.ts, not the one in node_modules
	if count != 1 {
		t.Errorf("Expected 1 test file (excluding node_modules), got %d", count)
	}
}

func TestValidateService_Go_WithSubdirectoryTests(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module example.com/test

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create test file in subdirectory
	subDir := filepath.Join(tmpDir, "pkg", "utils")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	testFile := `package utils

import "testing"

func TestHelper(t *testing.T) {}
`
	if err := os.WriteFile(filepath.Join(subDir, "utils_test.go"), []byte(testFile), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	service := ServiceInfo{
		Name:     "api",
		Language: "go",
		Dir:      tmpDir,
	}

	validation := ValidateService(service)

	if !validation.CanTest {
		t.Errorf("Expected CanTest to be true, got false. SkipReason: %s", validation.SkipReason)
	}
	if validation.TestFiles != 1 {
		t.Errorf("Expected TestFiles to be 1, got %d", validation.TestFiles)
	}
}

func TestValidateService_Go_SkipsVendor(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module example.com/test

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create test file in root
	if err := os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create test file in vendor (should be skipped)
	vendorDir := filepath.Join(tmpDir, "vendor", "somelib")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatalf("Failed to create vendor dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(vendorDir, "lib_test.go"), []byte("package somelib"), 0644); err != nil {
		t.Fatalf("Failed to create vendor test file: %v", err)
	}

	service := ServiceInfo{
		Name:     "api",
		Language: "go",
		Dir:      tmpDir,
	}

	validation := ValidateService(service)

	if !validation.CanTest {
		t.Errorf("Expected CanTest to be true, got false. SkipReason: %s", validation.SkipReason)
	}
	// Should only count 1 test file (excluding vendor)
	if validation.TestFiles != 1 {
		t.Errorf("Expected TestFiles to be 1 (excluding vendor), got %d", validation.TestFiles)
	}
}

func TestValidateService_Dotnet_WithNUnit(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test project directory
	testProjDir := filepath.Join(tmpDir, "MyApp.Tests")
	if err := os.MkdirAll(testProjDir, 0755); err != nil {
		t.Fatalf("Failed to create test project dir: %v", err)
	}

	// Create csproj with NUnit
	csproj := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="NUnit" Version="3.14.0" />
  </ItemGroup>
</Project>`
	if err := os.WriteFile(filepath.Join(testProjDir, "MyApp.Tests.csproj"), []byte(csproj), 0644); err != nil {
		t.Fatalf("Failed to create csproj: %v", err)
	}

	service := ServiceInfo{
		Name:     "myapp",
		Language: "cs",
		Dir:      tmpDir,
	}

	validation := ValidateService(service)

	if !validation.CanTest {
		t.Errorf("Expected CanTest to be true, got false. SkipReason: %s", validation.SkipReason)
	}
	if validation.Framework != "nunit" {
		t.Errorf("Expected framework 'nunit', got '%s'", validation.Framework)
	}
}

func TestValidateService_Python_WithPyprojectToml(t *testing.T) {
	tmpDir := t.TempDir()

	// Create pyproject.toml with pytest config
	pyproject := `[tool.poetry]
name = "myapp"
version = "0.1.0"

[tool.pytest.ini_options]
testpaths = ["tests"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte(pyproject), 0644); err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	service := ServiceInfo{
		Name:     "myapp",
		Language: "python",
		Dir:      tmpDir,
	}

	validation := ValidateService(service)

	if !validation.CanTest {
		t.Errorf("Expected CanTest to be true, got false. SkipReason: %s", validation.SkipReason)
	}
	if validation.Framework != "pytest" {
		t.Errorf("Expected framework 'pytest', got '%s'", validation.Framework)
	}
}

func TestValidateService_NodeJS_WithTestFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files without any config
	if err := os.WriteFile(filepath.Join(tmpDir, "app.test.js"), []byte("test('works', () => {})"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	service := ServiceInfo{
		Name:     "web",
		Language: "js",
		Dir:      tmpDir,
	}

	validation := ValidateService(service)

	if !validation.CanTest {
		t.Errorf("Expected CanTest to be true, got false. SkipReason: %s", validation.SkipReason)
	}
	if validation.TestFiles != 1 {
		t.Errorf("Expected TestFiles to be 1, got %d", validation.TestFiles)
	}
}

func TestValidateService_NodeJS_WithMocha(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mocha config
	if err := os.WriteFile(filepath.Join(tmpDir, ".mocharc.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create mocha config: %v", err)
	}

	// Create test file
	if err := os.WriteFile(filepath.Join(tmpDir, "app.test.js"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	service := ServiceInfo{
		Name:     "api",
		Language: "javascript",
		Dir:      tmpDir,
	}

	validation := ValidateService(service)

	if !validation.CanTest {
		t.Errorf("Expected CanTest to be true, got false. SkipReason: %s", validation.SkipReason)
	}
	if validation.Framework != "mocha" {
		t.Errorf("Expected framework 'mocha', got '%s'", validation.Framework)
	}
}

func TestValidateService_NodeJS_WithVitestInPackageJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create package.json with vitest dependency but no config file
	packageJSON := `{
		"name": "test-app",
		"scripts": {
			"test": "vitest"
		},
		"devDependencies": {
			"vitest": "^1.0.0"
		}
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	service := ServiceInfo{
		Name:     "web",
		Language: "ts",
		Dir:      tmpDir,
	}

	validation := ValidateService(service)

	if !validation.CanTest {
		t.Errorf("Expected CanTest to be true, got false. SkipReason: %s", validation.SkipReason)
	}
	if validation.Framework != "vitest" {
		t.Errorf("Expected framework 'vitest', got '%s'", validation.Framework)
	}
}

func TestValidateService_Dotnet_WithMSTest(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test project directory
	testProjDir := filepath.Join(tmpDir, "MyApp.Tests")
	if err := os.MkdirAll(testProjDir, 0755); err != nil {
		t.Fatalf("Failed to create test project dir: %v", err)
	}

	// Create csproj with MSTest
	csproj := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="MSTest.TestFramework" Version="3.0.0" />
  </ItemGroup>
</Project>`
	if err := os.WriteFile(filepath.Join(testProjDir, "MyApp.Tests.csproj"), []byte(csproj), 0644); err != nil {
		t.Fatalf("Failed to create csproj: %v", err)
	}

	service := ServiceInfo{
		Name:     "myapp",
		Language: "fsharp",
		Dir:      tmpDir,
	}

	validation := ValidateService(service)

	if !validation.CanTest {
		t.Errorf("Expected CanTest to be true, got false. SkipReason: %s", validation.SkipReason)
	}
	if validation.Framework != "mstest" {
		t.Errorf("Expected framework 'mstest', got '%s'", validation.Framework)
	}
}

func TestValidateService_Python_WithTestFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file with test_ prefix
	if err := os.WriteFile(filepath.Join(tmpDir, "test_main.py"), []byte("def test_one(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	service := ServiceInfo{
		Name:     "api",
		Language: "py",
		Dir:      tmpDir,
	}

	validation := ValidateService(service)

	if !validation.CanTest {
		t.Errorf("Expected CanTest to be true, got false. SkipReason: %s", validation.SkipReason)
	}
	if validation.TestFiles == 0 {
		t.Error("Expected TestFiles > 0")
	}
}

func TestCountTestFiles_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	patterns := []string{"*.test.ts", "*.spec.ts"}
	count := countTestFiles(tmpDir, patterns)

	if count != 0 {
		t.Errorf("Expected 0 test files in empty directory, got %d", count)
	}
}

func TestCountTestFiles_NonRecursivePattern(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file in root
	if err := os.WriteFile(filepath.Join(tmpDir, "app.test.ts"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Only non-recursive pattern
	patterns := []string{"*.test.ts"}
	count := countTestFiles(tmpDir, patterns)

	if count != 1 {
		t.Errorf("Expected 1 test file, got %d", count)
	}
}

func TestValidateService_Dotnet_WithFSharpTestProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Create F# test project directory
	testProjDir := filepath.Join(tmpDir, "MyApp.Tests")
	if err := os.MkdirAll(testProjDir, 0755); err != nil {
		t.Fatalf("Failed to create test project dir: %v", err)
	}

	// Create fsproj with xunit
	fsproj := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="xunit" Version="2.5.0" />
  </ItemGroup>
</Project>`
	if err := os.WriteFile(filepath.Join(testProjDir, "MyApp.Tests.fsproj"), []byte(fsproj), 0644); err != nil {
		t.Fatalf("Failed to create fsproj: %v", err)
	}

	service := ServiceInfo{
		Name:     "myapp",
		Language: "fs",
		Dir:      tmpDir,
	}

	validation := ValidateService(service)

	if !validation.CanTest {
		t.Errorf("Expected CanTest to be true, got false. SkipReason: %s", validation.SkipReason)
	}
	if validation.Framework != "xunit" {
		t.Errorf("Expected framework 'xunit', got '%s'", validation.Framework)
	}
}
