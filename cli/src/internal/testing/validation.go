// Package testing provides test execution and coverage aggregation for multi-language projects.
package testing

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/security"
)

// ServiceValidation represents the validation result for a service's testability.
type ServiceValidation struct {
	// Name is the service name
	Name string
	// Language is the programming language
	Language string
	// Framework is the detected test framework
	Framework string
	// TestFiles is the count of test files found
	TestFiles int
	// CanTest indicates if the service can be tested
	CanTest bool
	// SkipReason explains why a service cannot be tested (if CanTest is false)
	SkipReason string
}

// ValidateService checks if a service is testable and returns validation details.
func ValidateService(service ServiceInfo) ServiceValidation {
	validation := ServiceValidation{
		Name:     service.Name,
		Language: service.Language,
		CanTest:  false,
	}

	// Validate the service directory path
	if err := security.ValidatePath(service.Dir); err != nil {
		validation.SkipReason = "Invalid service directory path"
		return validation
	}

	// Check if directory exists
	if _, err := os.Stat(service.Dir); os.IsNotExist(err) {
		validation.SkipReason = "Service directory does not exist"
		return validation
	}

	// Validate based on language
	switch strings.ToLower(service.Language) {
	case "js", "javascript", "typescript", "ts":
		return validateNodeService(service, validation)
	case "python", "py":
		return validatePythonService(service, validation)
	case "go", "golang":
		return validateGoService(service, validation)
	case "csharp", "dotnet", "fsharp", "cs", "fs":
		return validateDotnetService(service, validation)
	default:
		validation.SkipReason = "Unsupported language: " + service.Language
		return validation
	}
}

// validateNodeService validates a Node.js service for testability.
func validateNodeService(service ServiceInfo, validation ServiceValidation) ServiceValidation {
	// Check for test framework config files
	frameworkConfigs := map[string]string{
		"jest.config.js":   "jest",
		"jest.config.ts":   "jest",
		"jest.config.json": "jest",
		"vitest.config.js": "vitest",
		"vitest.config.ts": "vitest",
		".mocharc.js":      "mocha",
		".mocharc.json":    "mocha",
		".mocharc.yaml":    "mocha",
	}

	for file, framework := range frameworkConfigs {
		if _, err := os.Stat(filepath.Join(service.Dir, file)); err == nil {
			validation.Framework = framework
			break
		}
	}

	// Check package.json for test script and dependencies
	packageJSONPath := filepath.Join(service.Dir, "package.json")
	hasTestScript := false
	// #nosec G304 -- Path is constructed from validated service.Dir
	if data, err := os.ReadFile(packageJSONPath); err == nil {
		content := string(data)

		// Check for test script
		if strings.Contains(content, `"test"`) && strings.Contains(content, `"scripts"`) {
			hasTestScript = true
		}

		// Detect framework from dependencies if not already detected
		if validation.Framework == "" {
			if strings.Contains(content, `"vitest"`) {
				validation.Framework = "vitest"
			} else if strings.Contains(content, `"jest"`) {
				validation.Framework = "jest"
			} else if strings.Contains(content, `"mocha"`) {
				validation.Framework = "mocha"
			}
		}
	}

	// Count test files
	testPatterns := []string{
		"**/*.test.js",
		"**/*.test.ts",
		"**/*.spec.js",
		"**/*.spec.ts",
		"**/*.test.jsx",
		"**/*.test.tsx",
		"**/*.spec.jsx",
		"**/*.spec.tsx",
	}

	testFileCount := countTestFiles(service.Dir, testPatterns)
	validation.TestFiles = testFileCount

	// Determine if service can be tested
	if testFileCount > 0 || hasTestScript {
		validation.CanTest = true
		if validation.Framework == "" {
			validation.Framework = "npm"
		}
	} else {
		validation.SkipReason = "No test script in package.json and no test files found"
	}

	return validation
}

// validatePythonService validates a Python service for testability.
func validatePythonService(service ServiceInfo, validation ServiceValidation) ServiceValidation {
	// Check for pytest configuration
	pytestConfigs := []string{"pytest.ini", "pyproject.toml", "setup.cfg", "tox.ini"}
	hasPytestConfig := false
	for _, config := range pytestConfigs {
		configPath := filepath.Join(service.Dir, config)
		if _, err := os.Stat(configPath); err == nil {
			// For pyproject.toml, check if it contains pytest config
			if config == "pyproject.toml" {
				// #nosec G304 -- Path is constructed from validated service.Dir
				if data, err := os.ReadFile(configPath); err == nil {
					if strings.Contains(string(data), "[tool.pytest") {
						hasPytestConfig = true
						validation.Framework = "pytest"
						break
					}
				}
			} else {
				hasPytestConfig = true
				validation.Framework = "pytest"
				break
			}
		}
	}

	// Check for tests directory
	testDirs := []string{"tests", "test"}
	hasTestDir := false
	for _, dir := range testDirs {
		testPath := filepath.Join(service.Dir, dir)
		if info, err := os.Stat(testPath); err == nil && info.IsDir() {
			hasTestDir = true
			break
		}
	}

	// Count test files
	testPatterns := []string{
		"**/test_*.py",
		"**/*_test.py",
		"tests/**/*.py",
		"test/**/*.py",
	}

	testFileCount := countTestFiles(service.Dir, testPatterns)
	validation.TestFiles = testFileCount

	// Determine if service can be tested
	if testFileCount > 0 || hasTestDir || hasPytestConfig {
		validation.CanTest = true
		if validation.Framework == "" {
			validation.Framework = "pytest"
		}
	} else {
		validation.SkipReason = "No pytest configuration and no test files found"
	}

	return validation
}

// validateGoService validates a Go service for testability.
func validateGoService(service ServiceInfo, validation ServiceValidation) ServiceValidation {
	// Check for go.mod
	goModPath := filepath.Join(service.Dir, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		validation.SkipReason = "No go.mod file found"
		return validation
	}

	// Count *_test.go files
	testFileCount := 0
	_ = filepath.WalkDir(service.Dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		// Skip vendor directory
		if d.IsDir() && d.Name() == "vendor" {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), "_test.go") {
			testFileCount++
		}
		return nil
	})

	validation.TestFiles = testFileCount
	validation.Framework = "gotest"

	if testFileCount > 0 {
		validation.CanTest = true
	} else {
		validation.SkipReason = "No *_test.go files found"
	}

	return validation
}

// validateDotnetService validates a .NET service for testability.
func validateDotnetService(service ServiceInfo, validation ServiceValidation) ServiceValidation {
	// Find test projects (projects with "Test" or "Tests" in name)
	testProjectCount := 0
	framework := ""

	_ = filepath.WalkDir(service.Dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		// Skip bin and obj directories
		if d.IsDir() && (d.Name() == "bin" || d.Name() == "obj") {
			return filepath.SkipDir
		}

		if !d.IsDir() && (strings.HasSuffix(d.Name(), ".csproj") || strings.HasSuffix(d.Name(), ".fsproj")) {
			fileName := strings.ToLower(d.Name())
			if strings.Contains(fileName, "test") {
				testProjectCount++

				// Try to detect framework from project file
				if framework == "" {
					// #nosec G304 -- Path is from filepath.WalkDir of validated service.Dir
					if data, err := os.ReadFile(path); err == nil {
						content := string(data)
						if strings.Contains(content, "xunit") {
							framework = "xunit"
						} else if strings.Contains(content, "NUnit") {
							framework = "nunit"
						} else if strings.Contains(content, "MSTest") {
							framework = "mstest"
						}
					}
				}
			}
		}
		return nil
	})

	validation.TestFiles = testProjectCount
	if framework != "" {
		validation.Framework = framework
	} else if testProjectCount > 0 {
		validation.Framework = "dotnet"
	}

	if testProjectCount > 0 {
		validation.CanTest = true
	} else {
		validation.SkipReason = "No test projects found (projects with 'Test' in name)"
	}

	return validation
}

// countTestFiles counts files matching the given glob patterns in a directory.
func countTestFiles(dir string, patterns []string) int {
	count := 0
	seen := make(map[string]bool)

	for _, pattern := range patterns {
		// Handle simple patterns
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err == nil {
			for _, match := range matches {
				if !seen[match] {
					seen[match] = true
					count++
				}
			}
		}

		// Handle recursive patterns (**/)
		if strings.Contains(pattern, "**") {
			// Walk the directory tree
			basePattern := strings.Replace(pattern, "**/", "", 1)
			_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				// Skip node_modules, vendor, and other common ignore dirs
				if d.IsDir() {
					switch d.Name() {
					case "node_modules", "vendor", ".git", "bin", "obj", "__pycache__", ".pytest_cache":
						return filepath.SkipDir
					}
				}
				if !d.IsDir() {
					// Check if file matches the base pattern
					matched, _ := filepath.Match(basePattern, d.Name())
					if matched && !seen[path] {
						seen[path] = true
						count++
					}
				}
				return nil
			})
		}
	}

	return count
}

// ValidateServices validates all services and returns validation results.
func ValidateServices(services []ServiceInfo) []ServiceValidation {
	validations := make([]ServiceValidation, 0, len(services))
	for _, service := range services {
		validations = append(validations, ValidateService(service))
	}
	return validations
}

// GetTestableServices returns only the services that can be tested.
func GetTestableServices(validations []ServiceValidation) []ServiceValidation {
	testable := make([]ServiceValidation, 0)
	for _, v := range validations {
		if v.CanTest {
			testable = append(testable, v)
		}
	}
	return testable
}

// GetSkippedServices returns only the services that cannot be tested.
func GetSkippedServices(validations []ServiceValidation) []ServiceValidation {
	skipped := make([]ServiceValidation, 0)
	for _, v := range validations {
		if !v.CanTest {
			skipped = append(skipped, v)
		}
	}
	return skipped
}
