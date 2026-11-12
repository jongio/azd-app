package testing

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewDotnetTestRunner(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{
		Framework: "xunit",
	}

	runner := NewDotnetTestRunner(tmpDir, config)
	if runner == nil {
		t.Fatal("Expected runner to be created")
	}
	if runner.projectDir != tmpDir {
		t.Error("Project dir not set correctly")
	}
	if runner.config != config {
		t.Error("Config not set correctly")
	}
}

func TestDotnetRunnerBuildTestCommand_CustomCommand(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{
		Framework: "xunit",
		Unit: &TestTypeConfig{
			Command: "dotnet test --filter Category=Unit",
		},
	}

	runner := NewDotnetTestRunner(tmpDir, config)
	command, args := runner.buildTestCommand("unit", false)

	if command != "dotnet" {
		t.Errorf("Expected command 'dotnet', got '%s'", command)
	}
	expectedArgs := []string{"test", "--filter", "Category=Unit"}
	if len(args) < len(expectedArgs) {
		t.Errorf("Expected at least %d args, got %d", len(expectedArgs), len(args))
	}
}

func TestDotnetRunnerBuildTestCommand_WithFilter(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{
		Framework: "xunit",
	}

	runner := NewDotnetTestRunner(tmpDir, config)
	command, args := runner.buildTestCommand("unit", false)

	if command != "dotnet" {
		t.Errorf("Expected command 'dotnet', got '%s'", command)
	}

	// Should include filter for test type
	foundFilter := false
	for i, arg := range args {
		if arg == "--filter" && i+1 < len(args) && args[i+1] == "Category=Unit" {
			foundFilter = true
			break
		}
	}
	if !foundFilter {
		t.Error("Expected '--filter Category=Unit' in args")
	}
}

func TestDotnetRunnerBuildTestCommand_WithCoverage(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{
		Framework: "xunit",
	}

	runner := NewDotnetTestRunner(tmpDir, config)
	_, args := runner.buildTestCommand("all", true)

	// Should include coverage collector
	found := false
	for i, arg := range args {
		if arg == "--collect" && i+1 < len(args) && args[i+1] == "XPlat Code Coverage" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected coverage collector in args")
	}
}

func TestDotnetRunnerBuildTestCommand_MicrosoftTestingPlatform(t *testing.T) {
	t.Run("native global.json MTP passes --project and extension flags directly", func(t *testing.T) {
		rootDir := t.TempDir()
		globalJSON := `{"sdk":{"version":"10.0.100"},"test":{"runner":"Microsoft.Testing.Platform"}}`
		if err := os.WriteFile(filepath.Join(rootDir, "global.json"), []byte(globalJSON), 0o600); err != nil {
			t.Fatalf("write global.json: %v", err)
		}

		// Nest 6 directories deep to verify ancestor search walks all the way to rootDir.
		serviceDir := filepath.Join(rootDir, "a", "b", "c", "d", "e", "svc")
		if err := os.MkdirAll(serviceDir, 0o755); err != nil {
			t.Fatalf("mkdir nested serviceDir: %v", err)
		}
		projPath := filepath.Join(serviceDir, "UnitTests.csproj")
		if err := os.WriteFile(projPath, []byte("<Project></Project>"), 0o600); err != nil {
			t.Fatalf("write UnitTests.csproj: %v", err)
		}

		runner := NewDotnetTestRunner(serviceDir, &ServiceTestConfig{Framework: "xunit"})
		_, args := runner.buildTestCommand("unit", true)

		want := []string{
			"test",
			"--project", projPath,
			"--filter", "Category=Unit",
			"--coverage", "--coverage-output-format", "cobertura",
		}
		if len(args) != len(want) {
			t.Fatalf("args = %v, want %v", args, want)
		}
		for i := range want {
			if args[i] != want[i] {
				t.Fatalf("args[%d] = %q, want %q (full: %v)", i, args[i], want[i], args)
			}
		}
	})

	t.Run("MSBuild TestingPlatformDotnetTestSupport forwards filter and coverage after separator", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Place an SDK-only global.json in tmpDir so any ancestor global.json on the host is shadowed.
		if err := os.WriteFile(filepath.Join(tmpDir, "global.json"), []byte(`{"sdk":{"version":"10.0.100"}}`), 0o600); err != nil {
			t.Fatalf("write global.json: %v", err)
		}
		projPath := filepath.Join(tmpDir, "App.Tests.csproj")
		csproj := `<Project><PropertyGroup><TestingPlatformDotnetTestSupport>true</TestingPlatformDotnetTestSupport></PropertyGroup></Project>`
		if err := os.WriteFile(projPath, []byte(csproj), 0o600); err != nil {
			t.Fatalf("write App.Tests.csproj: %v", err)
		}

		runner := NewDotnetTestRunner(tmpDir, &ServiceTestConfig{Framework: "xunit"})
		_, args := runner.buildTestCommand("unit", true)

		want := []string{
			"test",
			projPath,
			"--",
			"--filter", "Category=Unit",
			"--coverage", "--coverage-output-format", "cobertura",
		}
		if len(args) != len(want) {
			t.Fatalf("args = %v, want %v", args, want)
		}
		for i := range want {
			if args[i] != want[i] {
				t.Fatalf("args[%d] = %q, want %q (full: %v)", i, args[i], want[i], args)
			}
		}
	})

	t.Run("nearer non-MTP global.json shadows parent MTP global.json", func(t *testing.T) {
		rootDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(rootDir, "global.json"), []byte(`{"test":{"runner":"Microsoft.Testing.Platform"}}`), 0o600); err != nil {
			t.Fatalf("write root global.json: %v", err)
		}
		childDir := filepath.Join(rootDir, "legacy-service")
		if err := os.MkdirAll(childDir, 0o755); err != nil {
			t.Fatalf("mkdir childDir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(childDir, "global.json"), []byte(`{"sdk":{"version":"8.0.100"}}`), 0o600); err != nil {
			t.Fatalf("write child global.json: %v", err)
		}

		runner := NewDotnetTestRunner(childDir, &ServiceTestConfig{Framework: "xunit"})
		_, args := runner.buildTestCommand("all", true)

		foundCollect := false
		for _, arg := range args {
			if arg == "--collect" {
				foundCollect = true
			}
		}
		if !foundCollect {
			t.Fatalf("expected VSTest --collect when nearer global.json omits MTP runner, got: %v", args)
		}
	})

	t.Run("inspects only selected first project when multiple test projects exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "global.json"), []byte(`{"sdk":{"version":"8.0.100"}}`), 0o600); err != nil {
			t.Fatalf("write global.json: %v", err)
		}
		firstProj := filepath.Join(tmpDir, "Alpha.Tests.csproj")
		secondProj := filepath.Join(tmpDir, "Zeta.Tests.csproj")
		if err := os.WriteFile(firstProj, []byte(`<Project></Project>`), 0o600); err != nil {
			t.Fatalf("write firstProj: %v", err)
		}
		mtpProj := `<Project><PropertyGroup><UseMicrosoftTestingPlatformRunner>true</UseMicrosoftTestingPlatformRunner></PropertyGroup></Project>`
		if err := os.WriteFile(secondProj, []byte(mtpProj), 0o600); err != nil {
			t.Fatalf("write secondProj: %v", err)
		}

		runner := NewDotnetTestRunner(tmpDir, &ServiceTestConfig{
			Framework: "xunit",
			Unit:      &TestTypeConfig{Projects: []string{firstProj, secondProj}},
		})
		_, args := runner.buildTestCommand("all", true)

		foundCollect := false
		for _, arg := range args {
			if arg == "--collect" {
				foundCollect = true
			}
		}
		if !foundCollect {
			t.Fatalf("expected VSTest --collect for non-MTP selected first project, got: %v", args)
		}
	})
}

func TestDotnetRunnerParseCommand(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{}
	runner := NewDotnetTestRunner(tmpDir, config)

	tests := []struct {
		name            string
		command         string
		expectedCommand string
		expectedArgs    []string
	}{
		{
			name:            "Simple command",
			command:         "dotnet test",
			expectedCommand: "dotnet",
			expectedArgs:    []string{"test"},
		},
		{
			name:            "Command with filter",
			command:         "dotnet test --filter Category=Unit",
			expectedCommand: "dotnet",
			expectedArgs:    []string{"test", "--filter", "Category=Unit"},
		},
		{
			name:            "Single word command",
			command:         "dotnet",
			expectedCommand: "dotnet",
			expectedArgs:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, args := runner.parseCommand(tt.command)
			if cmd != tt.expectedCommand {
				t.Errorf("Expected command '%s', got '%s'", tt.expectedCommand, cmd)
			}
			if len(args) != len(tt.expectedArgs) {
				t.Errorf("Expected %d args, got %d", len(tt.expectedArgs), len(args))
			}
			for i, arg := range args {
				if i < len(tt.expectedArgs) && arg != tt.expectedArgs[i] {
					t.Errorf("Expected arg[%d] '%s', got '%s'", i, tt.expectedArgs[i], arg)
				}
			}
		})
	}
}

func TestDotnetRunnerParseTestOutput(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{}
	runner := NewDotnetTestRunner(tmpDir, config)

	tests := []struct {
		name           string
		output         string
		expectedPassed int
		expectedFailed int
		expectedTotal  int
	}{
		{
			name: "dotnet test output",
			output: `Starting test execution, please wait...
A total of 1 test files matched the specified pattern.

Passed!  - Failed:     0, Passed:    10, Skipped:     0, Total:    10, Duration: 123 ms`,
			expectedPassed: 10,
			expectedFailed: 0,
			expectedTotal:  10,
		},
		{
			name: "dotnet test output with failures",
			output: `Starting test execution, please wait...
A total of 1 test files matched the specified pattern.

Failed!  - Failed:     2, Passed:     8, Skipped:     0, Total:    10, Duration: 123 ms`,
			expectedPassed: 8,
			expectedFailed: 2,
			expectedTotal:  10,
		},
		{
			name: "dotnet test output with skipped",
			output: `Test Run Successful.
Total tests: 15
     Passed: 12
     Failed: 0
    Skipped: 3
 Total time: 1.2345 Seconds`,
			expectedPassed: 12,
			expectedFailed: 0,
			expectedTotal:  15,
		},
		{
			name: "Microsoft.Testing.Platform summary output",
			output: `Running tests from App.Tests.dll (net10.0|x64)
App.Tests.dll (net10.0|x64) passed (2s 100ms)

Test run summary: Passed!
  total: 45
  failed: 0
  succeeded: 42
  skipped: 3
  duration: 2s 500ms`,
			expectedPassed: 42,
			expectedFailed: 0,
			expectedTotal:  45,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &TestResult{}
			runner.parseTestOutput(tt.output, result)

			if result.Passed != tt.expectedPassed {
				t.Errorf("Expected %d passed, got %d", tt.expectedPassed, result.Passed)
			}
			if result.Failed != tt.expectedFailed {
				t.Errorf("Expected %d failed, got %d", tt.expectedFailed, result.Failed)
			}
			if result.Total != tt.expectedTotal {
				t.Errorf("Expected %d total, got %d", tt.expectedTotal, result.Total)
			}
		})
	}
}

func TestDotnetRunnerHasTests(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test project file
	testProject := filepath.Join(tmpDir, "Tests.csproj")
	csprojContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <IsPackable>false</IsPackable>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="xunit" Version="2.4.1" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(testProject, []byte(csprojContent), 0o644); err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	config := &ServiceTestConfig{Framework: "xunit"}
	runner := NewDotnetTestRunner(tmpDir, config)

	if !runner.HasTests() {
		t.Error("Expected HasTests to return true")
	}
}

func TestDotnetRunnerHasTests_NoTests(t *testing.T) {
	tmpDir := t.TempDir()

	config := &ServiceTestConfig{Framework: "xunit"}
	runner := NewDotnetTestRunner(tmpDir, config)

	if runner.HasTests() {
		t.Error("Expected HasTests to return false for directory without tests")
	}
}

func TestFindTestProjects(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test project structure
	testsDir := filepath.Join(tmpDir, "tests")
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		t.Fatalf("Failed to create tests dir: %v", err)
	}

	testProject := filepath.Join(testsDir, "UnitTests.csproj")
	if err := os.WriteFile(testProject, []byte("<Project></Project>"), 0o644); err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	config := &ServiceTestConfig{}
	runner := NewDotnetTestRunner(tmpDir, config)

	projects := runner.findTestProjects()
	if len(projects) == 0 {
		t.Error("Expected to find test projects")
	}

	found := false
	for _, proj := range projects {
		if filepath.Base(proj) == "UnitTests.csproj" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find UnitTests.csproj")
	}
}

// TestDotnetRunnerRunTests_Integration tests the full RunTests workflow
func TestDotnetRunnerRunTests_Integration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple .csproj file
	csprojFile := filepath.Join(tmpDir, "Test.csproj")
	content := `<Project Sdk="Microsoft.NET.Sdk">
<PropertyGroup>
<TargetFramework>net8.0</TargetFramework>
</PropertyGroup>
<ItemGroup>
<PackageReference Include="xunit" Version="2.4.2" />
</ItemGroup>
</Project>`
	if err := os.WriteFile(csprojFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to create csproj file: %v", err)
	}

	config := &ServiceTestConfig{
		Framework: "xunit",
	}

	runner := NewDotnetTestRunner(tmpDir, config)
	result, err := runner.RunTests(context.Background(), "unit", false)
	if err != nil {
		t.Logf("RunTests returned error (expected in test env): %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

// TestDotnetRunnerBuildTestCommand_AllTypes tests different test types
func TestDotnetRunnerBuildTestCommand_AllTypes(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		testType string
	}{
		{name: "Unit tests", testType: "unit"},
		{name: "Integration tests", testType: "integration"},
		{name: "E2E tests", testType: "e2e"},
	}

	config := &ServiceTestConfig{Framework: "xunit"}
	runner := NewDotnetTestRunner(tmpDir, config)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command, args := runner.buildTestCommand(tt.testType, false)

			if command == "" {
				t.Error("Expected non-empty command")
			}
			if len(args) == 0 {
				t.Error("Expected non-empty args")
			}
		})
	}
}
