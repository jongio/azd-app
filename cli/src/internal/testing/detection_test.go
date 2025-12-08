// Package testing provides test execution and coverage aggregation for multi-language projects.
package testing

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewTestTypeDetector(t *testing.T) {
	detector := NewTestTypeDetector("/test/dir", "go")
	if detector.dir != "/test/dir" {
		t.Errorf("expected dir /test/dir, got %s", detector.dir)
	}
	if detector.language != "go" {
		t.Errorf("expected language go, got %s", detector.language)
	}
}

func TestTestTypeDetector_DetectByDirectories(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create test directories
	unitDir := filepath.Join(tmpDir, "unit")
	integrationDir := filepath.Join(tmpDir, "tests", "integration")
	e2eDir := filepath.Join(tmpDir, "e2e")

	if err := os.MkdirAll(unitDir, 0o755); err != nil {
		t.Fatalf("failed to create unit dir: %v", err)
	}
	if err := os.MkdirAll(integrationDir, 0o755); err != nil {
		t.Fatalf("failed to create integration dir: %v", err)
	}
	if err := os.MkdirAll(e2eDir, 0o755); err != nil {
		t.Fatalf("failed to create e2e dir: %v", err)
	}

	detector := NewTestTypeDetector(tmpDir, "go")
	result := detector.Detect()

	if !result.HasUnit {
		t.Error("expected HasUnit to be true")
	}
	if !result.HasIntegration {
		t.Error("expected HasIntegration to be true")
	}
	if !result.HasE2E {
		t.Error("expected HasE2E to be true")
	}
}

func TestTestTypeDetector_DetectByFilePatterns_Go(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	unitFile := filepath.Join(tmpDir, "handler_unit_test.go")
	integrationFile := filepath.Join(tmpDir, "db_integration_test.go")
	e2eFile := filepath.Join(tmpDir, "api_e2e_test.go")

	if err := os.WriteFile(unitFile, []byte("package main"), 0o644); err != nil {
		t.Fatalf("failed to create unit test file: %v", err)
	}
	if err := os.WriteFile(integrationFile, []byte("package main"), 0o644); err != nil {
		t.Fatalf("failed to create integration test file: %v", err)
	}
	if err := os.WriteFile(e2eFile, []byte("package main"), 0o644); err != nil {
		t.Fatalf("failed to create e2e test file: %v", err)
	}

	detector := NewTestTypeDetector(tmpDir, "go")
	result := detector.Detect()

	if !result.HasUnit {
		t.Error("expected HasUnit to be true for Go unit test file")
	}
	if !result.HasIntegration {
		t.Error("expected HasIntegration to be true for Go integration test file")
	}
	if !result.HasE2E {
		t.Error("expected HasE2E to be true for Go e2e test file")
	}
}

func TestTestTypeDetector_DetectByFilePatterns_Python(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	unitFile := filepath.Join(tmpDir, "test_unit_handler.py")
	integrationFile := filepath.Join(tmpDir, "test_integration_db.py")
	e2eFile := filepath.Join(tmpDir, "test_e2e_api.py")

	if err := os.WriteFile(unitFile, []byte("# test"), 0o644); err != nil {
		t.Fatalf("failed to create unit test file: %v", err)
	}
	if err := os.WriteFile(integrationFile, []byte("# test"), 0o644); err != nil {
		t.Fatalf("failed to create integration test file: %v", err)
	}
	if err := os.WriteFile(e2eFile, []byte("# test"), 0o644); err != nil {
		t.Fatalf("failed to create e2e test file: %v", err)
	}

	detector := NewTestTypeDetector(tmpDir, "python")
	result := detector.Detect()

	if !result.HasUnit {
		t.Error("expected HasUnit to be true for Python unit test file")
	}
	if !result.HasIntegration {
		t.Error("expected HasIntegration to be true for Python integration test file")
	}
	if !result.HasE2E {
		t.Error("expected HasE2E to be true for Python e2e test file")
	}
}

func TestTestTypeDetector_DetectByFilePatterns_TypeScript(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	unitFile := filepath.Join(tmpDir, "handler.unit.test.ts")
	integrationFile := filepath.Join(tmpDir, "db.integration.test.ts")
	e2eFile := filepath.Join(tmpDir, "api.e2e.test.ts")

	if err := os.WriteFile(unitFile, []byte("// test"), 0o644); err != nil {
		t.Fatalf("failed to create unit test file: %v", err)
	}
	if err := os.WriteFile(integrationFile, []byte("// test"), 0o644); err != nil {
		t.Fatalf("failed to create integration test file: %v", err)
	}
	if err := os.WriteFile(e2eFile, []byte("// test"), 0o644); err != nil {
		t.Fatalf("failed to create e2e test file: %v", err)
	}

	detector := NewTestTypeDetector(tmpDir, "typescript")
	result := detector.Detect()

	if !result.HasUnit {
		t.Error("expected HasUnit to be true for TypeScript unit test file")
	}
	if !result.HasIntegration {
		t.Error("expected HasIntegration to be true for TypeScript integration test file")
	}
	if !result.HasE2E {
		t.Error("expected HasE2E to be true for TypeScript e2e test file")
	}
}

func TestTestTypeDetector_DetectByMarkers_Go(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file with build tags
	testContent := `//go:build unit

package main

import "testing"

func TestUnit_Handler(t *testing.T) {}
`
	testFile := filepath.Join(tmpDir, "handler_test.go")
	if err := os.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	detector := NewTestTypeDetector(tmpDir, "go")
	result := detector.Detect()

	if !result.HasUnit {
		t.Error("expected HasUnit to be true for Go file with unit build tag")
	}
}

func TestTestTypeDetector_DetectByMarkers_Python(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file with pytest markers
	testContent := `import pytest

@pytest.mark.integration
def test_database_connection():
    pass
`
	testFile := filepath.Join(tmpDir, "test_db.py")
	if err := os.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	detector := NewTestTypeDetector(tmpDir, "python")
	result := detector.Detect()

	if !result.HasIntegration {
		t.Error("expected HasIntegration to be true for Python file with pytest marker")
	}
}

func TestTestTypeDetector_DetectByMarkers_CSharp(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file with xUnit traits
	testContent := `using Xunit;

public class UserTests
{
    [Fact]
    [Trait("Category", "Unit")]
    public void TestUserCreate()
    {
    }
}
`
	testFile := filepath.Join(tmpDir, "UserTests.cs")
	if err := os.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	detector := NewTestTypeDetector(tmpDir, "csharp")
	result := detector.Detect()

	if !result.HasUnit {
		t.Error("expected HasUnit to be true for C# file with Trait attribute")
	}
}

func TestTestTypeDetector_GetAvailableTestTypes(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(dir string) error
		language string
		expected []string
	}{
		{
			name: "all types present",
			setup: func(dir string) error {
				if err := os.MkdirAll(filepath.Join(dir, "unit"), 0o755); err != nil {
					return err
				}
				if err := os.MkdirAll(filepath.Join(dir, "integration"), 0o755); err != nil {
					return err
				}
				return os.MkdirAll(filepath.Join(dir, "e2e"), 0o755)
			},
			language: "go",
			expected: []string{"unit", "integration", "e2e"},
		},
		{
			name: "no types - fallback to all",
			setup: func(dir string) error {
				return nil
			},
			language: "go",
			expected: []string{"all"},
		},
		{
			name: "only unit",
			setup: func(dir string) error {
				return os.MkdirAll(filepath.Join(dir, "unit"), 0o755)
			},
			language: "go",
			expected: []string{"unit"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			if err := tt.setup(tmpDir); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			detector := NewTestTypeDetector(tmpDir, tt.language)
			result := detector.GetAvailableTestTypes()

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d types, got %d: %v", len(tt.expected), len(result), result)
				return
			}

			for i, exp := range tt.expected {
				if result[i] != exp {
					t.Errorf("expected type %s at index %d, got %s", exp, i, result[i])
				}
			}
		})
	}
}

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		filename string
		language string
		expected bool
	}{
		// Go
		{"handler_test.go", "go", true},
		{"handler.go", "go", false},
		// Python
		{"test_handler.py", "python", true},
		{"handler_test.py", "python", true},
		{"handler.py", "python", false},
		// TypeScript
		{"handler.test.ts", "typescript", true},
		{"handler.spec.ts", "typescript", true},
		{"handler.ts", "typescript", false},
		// C#
		{"UserTests.cs", "csharp", true},
		{"User.cs", "csharp", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			detector := NewTestTypeDetector("", tt.language)
			result := detector.isTestFile(tt.filename)
			if result != tt.expected {
				t.Errorf("isTestFile(%s) = %v, expected %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestGetPatternForType(t *testing.T) {
	tests := []struct {
		testType string
		language string
		expected string
	}{
		{"unit", "go", "^TestUnit"},
		{"integration", "go", "^TestIntegration"},
		{"e2e", "go", "^TestE2E"},
		{"unit", "python", "test_unit"},
		{"integration", "python", "test_integration"},
		{"e2e", "python", "test_e2e"},
		{"unit", "typescript", "unit"},
		{"integration", "typescript", "integration"},
		{"e2e", "typescript", "e2e"},
		{"unit", "csharp", "Category=Unit"},
		{"integration", "csharp", "Category=Integration"},
		{"e2e", "csharp", "Category=E2E"},
	}

	for _, tt := range tests {
		t.Run(tt.testType+"_"+tt.language, func(t *testing.T) {
			result := getPatternForType(tt.testType, tt.language)
			if result != tt.expected {
				t.Errorf("getPatternForType(%s, %s) = %s, expected %s",
					tt.testType, tt.language, result, tt.expected)
			}
		})
	}
}

func TestSuggestTestTypeConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directories for all test types
	if err := os.MkdirAll(filepath.Join(tmpDir, "unit"), 0o755); err != nil {
		t.Fatalf("failed to create unit dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "integration"), 0o755); err != nil {
		t.Fatalf("failed to create integration dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "e2e"), 0o755); err != nil {
		t.Fatalf("failed to create e2e dir: %v", err)
	}

	config := SuggestTestTypeConfig(tmpDir, "go")

	if config.Unit == nil {
		t.Error("expected Unit config to be set")
	} else if config.Unit.Pattern != "^TestUnit" {
		t.Errorf("expected Unit pattern ^TestUnit, got %s", config.Unit.Pattern)
	}

	if config.Integration == nil {
		t.Error("expected Integration config to be set")
	} else if config.Integration.Pattern != "^TestIntegration" {
		t.Errorf("expected Integration pattern ^TestIntegration, got %s", config.Integration.Pattern)
	}

	if config.E2E == nil {
		t.Error("expected E2E config to be set")
	} else if config.E2E.Pattern != "^TestE2E" {
		t.Errorf("expected E2E pattern ^TestE2E, got %s", config.E2E.Pattern)
	}
}

func TestAppendUnique(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		value    string
		expected []string
	}{
		{
			name:     "add new value",
			slice:    []string{"a", "b"},
			value:    "c",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "value already exists",
			slice:    []string{"a", "b"},
			value:    "b",
			expected: []string{"a", "b"},
		},
		{
			name:     "empty slice",
			slice:    []string{},
			value:    "a",
			expected: []string{"a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := appendUnique(tt.slice, tt.value)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d items, got %d", len(tt.expected), len(result))
				return
			}
			for i, v := range tt.expected {
				if result[i] != v {
					t.Errorf("expected %s at index %d, got %s", v, i, result[i])
				}
			}
		})
	}
}

func TestDetectServiceTestTypes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create unit directory
	if err := os.MkdirAll(filepath.Join(tmpDir, "unit"), 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	result := DetectServiceTestTypes(tmpDir, "go")

	if !result.HasUnit {
		t.Error("expected HasUnit to be true")
	}
	if result.HasIntegration {
		t.Error("expected HasIntegration to be false")
	}
	if result.HasE2E {
		t.Error("expected HasE2E to be false")
	}
}

func TestDetectByFilePatterns_SkipsNodeModules(t *testing.T) {
	tmpDir := t.TempDir()

	// Create node_modules with a unit test file inside (should be skipped)
	nodeModulesDir := filepath.Join(tmpDir, "node_modules", "some-package")
	if err := os.MkdirAll(nodeModulesDir, 0o755); err != nil {
		t.Fatalf("failed to create node_modules: %v", err)
	}

	// Create a unit test file inside node_modules (should be skipped)
	testFile := filepath.Join(nodeModulesDir, "handler_unit_test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Also create a real unit directory at root (should be detected)
	// Don't create it - we want to verify node_modules is truly skipped
	detector := NewTestTypeDetector(tmpDir, "go")
	result := detector.Detect()

	// Files in node_modules should be skipped by file pattern detection
	// and there's no unit directory at root level, so HasUnit should be false
	if result.HasUnit {
		t.Error("expected HasUnit to be false (node_modules should be skipped)")
	}
}

func TestGetFilePatterns_AllLanguages(t *testing.T) {
	languages := []string{
		"js", "javascript", "typescript", "ts",
		"python", "py",
		"go", "golang",
		"csharp", "dotnet", "fsharp", "cs", "fs",
	}

	for _, lang := range languages {
		t.Run(lang, func(t *testing.T) {
			detector := NewTestTypeDetector("", lang)
			patterns := detector.getFilePatterns()

			// Each language should have at least one pattern for each type
			if len(patterns.Unit) == 0 {
				t.Errorf("no unit patterns for language %s", lang)
			}
			if len(patterns.Integration) == 0 {
				t.Errorf("no integration patterns for language %s", lang)
			}
			if len(patterns.E2E) == 0 {
				t.Errorf("no e2e patterns for language %s", lang)
			}
		})
	}
}

func TestGetMarkers_AllLanguages(t *testing.T) {
	languages := []string{
		"python", "py",
		"go", "golang",
		"csharp", "dotnet",
		"js", "typescript",
	}

	for _, lang := range languages {
		t.Run(lang, func(t *testing.T) {
			detector := NewTestTypeDetector("", lang)
			markers := detector.getMarkers()

			// Each language should have markers defined
			hasAnyMarkers := len(markers.Unit) > 0 ||
				len(markers.Integration) > 0 ||
				len(markers.E2E) > 0

			if !hasAnyMarkers {
				t.Errorf("no markers defined for language %s", lang)
			}
		})
	}
}
