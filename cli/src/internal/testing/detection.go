// Package testing provides test execution and coverage aggregation for multi-language projects.
package testing

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/security"
)

// TestTypeDetector detects available test types in a service directory.
type TestTypeDetector struct {
	dir      string
	language string
}

// DetectedTestTypes represents the test types found in a directory.
type DetectedTestTypes struct {
	HasUnit          bool
	HasIntegration   bool
	HasE2E           bool
	UnitPaths        []string
	IntegrationPaths []string
	E2EPaths         []string
}

// NewTestTypeDetector creates a new test type detector.
func NewTestTypeDetector(dir, language string) *TestTypeDetector {
	return &TestTypeDetector{
		dir:      dir,
		language: strings.ToLower(language),
	}
}

// Detect detects all available test types in the service directory.
func (d *TestTypeDetector) Detect() *DetectedTestTypes {
	result := &DetectedTestTypes{
		UnitPaths:        make([]string, 0),
		IntegrationPaths: make([]string, 0),
		E2EPaths:         make([]string, 0),
	}

	// Detect by directory structure
	d.detectByDirectories(result)

	// Detect by file patterns
	d.detectByFilePatterns(result)

	// Detect by markers (for languages that support test markers)
	d.detectByMarkers(result)

	return result
}

// detectByDirectories checks for test type directories.
func (d *TestTypeDetector) detectByDirectories(result *DetectedTestTypes) {
	// Common directory names for each test type
	unitDirs := []string{
		"unit", "tests/unit", "test/unit", "__tests__/unit",
		"src/__tests__/unit", "spec/unit", "Unit", "UnitTests",
	}
	integrationDirs := []string{
		"integration", "tests/integration", "test/integration",
		"__tests__/integration", "spec/integration",
		"Integration", "IntegrationTests",
	}
	e2eDirs := []string{
		"e2e", "tests/e2e", "test/e2e", "__tests__/e2e",
		"spec/e2e", "E2E", "EndToEnd", "end-to-end",
		"cypress", "playwright",
	}

	// Check unit directories
	for _, dir := range unitDirs {
		fullPath := filepath.Join(d.dir, dir)
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			result.HasUnit = true
			result.UnitPaths = append(result.UnitPaths, fullPath)
		}
	}

	// Check integration directories
	for _, dir := range integrationDirs {
		fullPath := filepath.Join(d.dir, dir)
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			result.HasIntegration = true
			result.IntegrationPaths = append(result.IntegrationPaths, fullPath)
		}
	}

	// Check e2e directories
	for _, dir := range e2eDirs {
		fullPath := filepath.Join(d.dir, dir)
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			result.HasE2E = true
			result.E2EPaths = append(result.E2EPaths, fullPath)
		}
	}
}

// detectByFilePatterns scans for test files matching type-specific patterns.
func (d *TestTypeDetector) detectByFilePatterns(result *DetectedTestTypes) {
	patterns := d.getFilePatterns()

	_ = filepath.WalkDir(d.dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// Skip directories
		if entry.IsDir() {
			// Skip node_modules, vendor, etc.
			name := entry.Name()
			if name == "node_modules" || name == "vendor" || name == ".git" ||
				name == "bin" || name == "obj" || name == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}

		filename := entry.Name()
		lowFilename := strings.ToLower(filename)

		// Check unit patterns
		for _, pattern := range patterns.Unit {
			if matched, _ := regexp.MatchString(pattern, lowFilename); matched {
				result.HasUnit = true
				result.UnitPaths = appendUnique(result.UnitPaths, filepath.Dir(path))
			}
		}

		// Check integration patterns
		for _, pattern := range patterns.Integration {
			if matched, _ := regexp.MatchString(pattern, lowFilename); matched {
				result.HasIntegration = true
				result.IntegrationPaths = appendUnique(result.IntegrationPaths, filepath.Dir(path))
			}
		}

		// Check e2e patterns
		for _, pattern := range patterns.E2E {
			if matched, _ := regexp.MatchString(pattern, lowFilename); matched {
				result.HasE2E = true
				result.E2EPaths = appendUnique(result.E2EPaths, filepath.Dir(path))
			}
		}

		return nil
	})
}

// filePatterns holds regex patterns for different test types.
type filePatterns struct {
	Unit        []string
	Integration []string
	E2E         []string
}

// getFilePatterns returns file patterns for the language.
func (d *TestTypeDetector) getFilePatterns() filePatterns {
	switch d.language {
	case "js", "javascript", "typescript", "ts":
		return filePatterns{
			Unit: []string{
				`\.unit\.test\.(ts|tsx|js|jsx)$`,
				`\.unit\.spec\.(ts|tsx|js|jsx)$`,
				`_unit_test\.(ts|tsx|js|jsx)$`,
				`unit\.test\.(ts|tsx|js|jsx)$`,
			},
			Integration: []string{
				`\.integration\.test\.(ts|tsx|js|jsx)$`,
				`\.integration\.spec\.(ts|tsx|js|jsx)$`,
				`_integration_test\.(ts|tsx|js|jsx)$`,
				`integration\.test\.(ts|tsx|js|jsx)$`,
			},
			E2E: []string{
				`\.e2e\.test\.(ts|tsx|js|jsx)$`,
				`\.e2e\.spec\.(ts|tsx|js|jsx)$`,
				`_e2e_test\.(ts|tsx|js|jsx)$`,
				`e2e\.test\.(ts|tsx|js|jsx)$`,
				`\.cy\.(ts|tsx|js|jsx)$`,   // Cypress
				`\.spec\.(ts|tsx|js|jsx)$`, // Playwright
			},
		}

	case "python", "py":
		return filePatterns{
			Unit: []string{
				`test_unit.*\.py$`,
				`unit_test.*\.py$`,
				`.*_unit_test\.py$`,
				`test_.*_unit\.py$`,
			},
			Integration: []string{
				`test_integration.*\.py$`,
				`integration_test.*\.py$`,
				`.*_integration_test\.py$`,
				`test_.*_integration\.py$`,
			},
			E2E: []string{
				`test_e2e.*\.py$`,
				`e2e_test.*\.py$`,
				`.*_e2e_test\.py$`,
				`test_.*_e2e\.py$`,
			},
		}

	case "go", "golang":
		return filePatterns{
			Unit: []string{
				`.*_unit_test\.go$`,
				`unit_test\.go$`,
			},
			Integration: []string{
				`.*_integration_test\.go$`,
				`integration_test\.go$`,
			},
			E2E: []string{
				`.*_e2e_test\.go$`,
				`e2e_test\.go$`,
			},
		}

	case "csharp", "dotnet", "fsharp", "cs", "fs":
		return filePatterns{
			Unit: []string{
				`.*unittests?\.cs$`,
				`.*\.unit\.cs$`,
			},
			Integration: []string{
				`.*integrationtests?\.cs$`,
				`.*\.integration\.cs$`,
			},
			E2E: []string{
				`.*e2etests?\.cs$`,
				`.*\.e2e\.cs$`,
			},
		}

	default:
		return filePatterns{}
	}
}

// detectByMarkers scans files for test markers/attributes.
func (d *TestTypeDetector) detectByMarkers(result *DetectedTestTypes) {
	markers := d.getMarkers()
	if len(markers.Unit) == 0 && len(markers.Integration) == 0 && len(markers.E2E) == 0 {
		return
	}

	_ = filepath.WalkDir(d.dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// Skip directories
		if entry.IsDir() {
			name := entry.Name()
			if name == "node_modules" || name == "vendor" || name == ".git" ||
				name == "bin" || name == "obj" || name == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only check test files
		if !d.isTestFile(entry.Name()) {
			return nil
		}

		// Validate path before reading (security G304 fix)
		if err := security.ValidatePath(path); err != nil {
			return nil
		}

		// Read file content
		// #nosec G304 -- Path validated by security.ValidatePath above
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		contentStr := string(content)

		// Check unit markers
		for _, marker := range markers.Unit {
			if strings.Contains(contentStr, marker) {
				result.HasUnit = true
				result.UnitPaths = appendUnique(result.UnitPaths, filepath.Dir(path))
				break
			}
		}

		// Check integration markers
		for _, marker := range markers.Integration {
			if strings.Contains(contentStr, marker) {
				result.HasIntegration = true
				result.IntegrationPaths = appendUnique(result.IntegrationPaths, filepath.Dir(path))
				break
			}
		}

		// Check e2e markers
		for _, marker := range markers.E2E {
			if strings.Contains(contentStr, marker) {
				result.HasE2E = true
				result.E2EPaths = appendUnique(result.E2EPaths, filepath.Dir(path))
				break
			}
		}

		return nil
	})
}

// markerPatterns holds markers for different test types.
type markerPatterns struct {
	Unit        []string
	Integration []string
	E2E         []string
}

// getMarkers returns test markers for the language.
func (d *TestTypeDetector) getMarkers() markerPatterns {
	switch d.language {
	case "python", "py":
		return markerPatterns{
			Unit: []string{
				"@pytest.mark.unit",
				"@unittest.skipUnless",
				"# unit test",
			},
			Integration: []string{
				"@pytest.mark.integration",
				"# integration test",
			},
			E2E: []string{
				"@pytest.mark.e2e",
				"@pytest.mark.end_to_end",
				"# e2e test",
			},
		}

	case "csharp", "dotnet", "fsharp", "cs", "fs":
		return markerPatterns{
			Unit: []string{
				`[Trait("Category", "Unit")]`,
				`[Category("Unit")]`,
				"[UnitTest]",
			},
			Integration: []string{
				`[Trait("Category", "Integration")]`,
				`[Category("Integration")]`,
				"[IntegrationTest]",
			},
			E2E: []string{
				`[Trait("Category", "E2E")]`,
				`[Trait("Category", "EndToEnd")]`,
				`[Category("E2E")]`,
				"[E2ETest]",
			},
		}

	case "go", "golang":
		return markerPatterns{
			Unit: []string{
				"// +build unit",
				"//go:build unit",
				"TestUnit",
			},
			Integration: []string{
				"// +build integration",
				"//go:build integration",
				"TestIntegration",
			},
			E2E: []string{
				"// +build e2e",
				"//go:build e2e",
				"TestE2E",
			},
		}

	case "js", "javascript", "typescript", "ts":
		return markerPatterns{
			Unit: []string{
				"describe.unit",
				"it.unit",
				"// @unit",
				"/* @unit */",
			},
			Integration: []string{
				"describe.integration",
				"it.integration",
				"// @integration",
				"/* @integration */",
			},
			E2E: []string{
				"describe.e2e",
				"it.e2e",
				"// @e2e",
				"/* @e2e */",
			},
		}

	default:
		return markerPatterns{}
	}
}

// isTestFile checks if a file is a test file based on language.
func (d *TestTypeDetector) isTestFile(filename string) bool {
	lowFilename := strings.ToLower(filename)

	switch d.language {
	case "js", "javascript", "typescript", "ts":
		return strings.Contains(lowFilename, ".test.") ||
			strings.Contains(lowFilename, ".spec.") ||
			strings.Contains(lowFilename, "_test.")

	case "python", "py":
		return strings.HasPrefix(lowFilename, "test_") ||
			strings.HasSuffix(lowFilename, "_test.py")

	case "go", "golang":
		return strings.HasSuffix(lowFilename, "_test.go")

	case "csharp", "dotnet", "fsharp", "cs", "fs":
		return strings.Contains(lowFilename, "test") &&
			(strings.HasSuffix(lowFilename, ".cs") || strings.HasSuffix(lowFilename, ".fs"))

	default:
		return false
	}
}

// GetAvailableTestTypes returns a list of available test types.
func (d *TestTypeDetector) GetAvailableTestTypes() []string {
	detected := d.Detect()
	types := make([]string, 0, 3)

	if detected.HasUnit {
		types = append(types, "unit")
	}
	if detected.HasIntegration {
		types = append(types, "integration")
	}
	if detected.HasE2E {
		types = append(types, "e2e")
	}

	// If no specific types detected, return "all"
	if len(types) == 0 {
		return []string{"all"}
	}

	return types
}

// appendUnique appends a value to a slice if not already present.
func appendUnique(slice []string, value string) []string {
	for _, v := range slice {
		if v == value {
			return slice
		}
	}
	return append(slice, value)
}

// DetectServiceTestTypes detects test types for a service.
// This is a convenience function for use by the orchestrator.
func DetectServiceTestTypes(dir, language string) *DetectedTestTypes {
	detector := NewTestTypeDetector(dir, language)
	return detector.Detect()
}

// SuggestTestTypeConfig generates a suggested TestTypeConfig based on detection.
func SuggestTestTypeConfig(dir, language string) *ServiceTestConfig {
	detected := DetectServiceTestTypes(dir, language)
	config := &ServiceTestConfig{}

	if detected.HasUnit && len(detected.UnitPaths) > 0 {
		config.Unit = &TestTypeConfig{
			Pattern: getPatternForType("unit", language),
		}
	}

	if detected.HasIntegration && len(detected.IntegrationPaths) > 0 {
		config.Integration = &TestTypeConfig{
			Pattern: getPatternForType("integration", language),
		}
	}

	if detected.HasE2E && len(detected.E2EPaths) > 0 {
		config.E2E = &TestTypeConfig{
			Pattern: getPatternForType("e2e", language),
		}
	}

	return config
}

// getPatternForType returns default test patterns for a test type.
func getPatternForType(testType, language string) string {
	lang := strings.ToLower(language)

	switch testType {
	case "unit":
		switch lang {
		case "go", "golang":
			return "^TestUnit"
		case "python", "py":
			return "test_unit"
		case "js", "javascript", "typescript", "ts":
			return "unit"
		case "csharp", "dotnet", "fsharp", "cs", "fs":
			return "Category=Unit"
		}
	case "integration":
		switch lang {
		case "go", "golang":
			return "^TestIntegration"
		case "python", "py":
			return "test_integration"
		case "js", "javascript", "typescript", "ts":
			return "integration"
		case "csharp", "dotnet", "fsharp", "cs", "fs":
			return "Category=Integration"
		}
	case "e2e":
		switch lang {
		case "go", "golang":
			return "^TestE2E"
		case "python", "py":
			return "test_e2e"
		case "js", "javascript", "typescript", "ts":
			return "e2e"
		case "csharp", "dotnet", "fsharp", "cs", "fs":
			return "Category=E2E"
		}
	}

	return ""
}
