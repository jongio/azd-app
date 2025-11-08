// Package testing provides test execution and coverage aggregation for multi-language projects.
package testing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/security"
	"gopkg.in/yaml.v3"
)

// TestOrchestrator manages test execution across services.
type TestOrchestrator struct {
	config   *TestConfig
	services []ServiceInfo
}

// ServiceInfo represents a service with its test configuration.
type ServiceInfo struct {
	Name     string
	Language string
	Dir      string
	Config   *ServiceTestConfig
}

// NewTestOrchestrator creates a new test orchestrator.
func NewTestOrchestrator(config *TestConfig) *TestOrchestrator {
	return &TestOrchestrator{
		config:   config,
		services: make([]ServiceInfo, 0),
	}
}

// LoadServicesFromAzureYaml loads service information from azure.yaml.
func (o *TestOrchestrator) LoadServicesFromAzureYaml(azureYamlPath string) error {
	// Validate path
	if err := security.ValidatePath(azureYamlPath); err != nil {
		return fmt.Errorf("invalid azure.yaml path: %w", err)
	}

	// Read azure.yaml
	// #nosec G304 -- Path validated by security.ValidatePath above
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		return fmt.Errorf("failed to read azure.yaml: %w", err)
	}

	// Parse YAML
	var azureYaml struct {
		Services map[string]struct {
			Language string                 `yaml:"language"`
			Project  string                 `yaml:"project"`
			Test     *ServiceTestConfig     `yaml:"test"`
			Config   map[string]interface{} `yaml:",inline"`
		} `yaml:"services"`
	}

	if err := yaml.Unmarshal(data, &azureYaml); err != nil {
		return fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	if len(azureYaml.Services) == 0 {
		return fmt.Errorf("no services defined in azure.yaml")
	}

	// Convert to ServiceInfo
	azureYamlDir := filepath.Dir(azureYamlPath)
	for name, svc := range azureYaml.Services {
		// Resolve project directory
		projectDir := svc.Project
		if !filepath.IsAbs(projectDir) {
			projectDir = filepath.Join(azureYamlDir, projectDir)
		}

		// Normalize the path
		projectDir = filepath.Clean(projectDir)

		o.services = append(o.services, ServiceInfo{
			Name:     name,
			Language: svc.Language,
			Dir:      projectDir,
			Config:   svc.Test,
		})
	}

	return nil
}

// DetectTestConfig auto-detects test configuration for a service.
func (o *TestOrchestrator) DetectTestConfig(service ServiceInfo) (*ServiceTestConfig, error) {
	// If config already exists, return it
	if service.Config != nil {
		return service.Config, nil
	}

	// Auto-detect based on language
	config := &ServiceTestConfig{}

	switch strings.ToLower(service.Language) {
	case "js", "javascript", "typescript", "ts":
		framework, err := detectNodeTestFramework(service.Dir)
		if err != nil {
			return nil, fmt.Errorf("failed to detect Node.js test framework: %w", err)
		}
		config.Framework = framework

	case "python", "py":
		framework, err := detectPythonTestFramework(service.Dir)
		if err != nil {
			return nil, fmt.Errorf("failed to detect Python test framework: %w", err)
		}
		config.Framework = framework

	case "csharp", "dotnet", "fsharp", "cs", "fs":
		framework, err := detectDotnetTestFramework(service.Dir)
		if err != nil {
			return nil, fmt.Errorf("failed to detect .NET test framework: %w", err)
		}
		config.Framework = framework

	default:
		return nil, fmt.Errorf("unsupported language: %s", service.Language)
	}

	return config, nil
}

// ExecuteTests runs tests for all services.
func (o *TestOrchestrator) ExecuteTests(testType string, serviceFilter []string) (*AggregateResult, error) {
	result := &AggregateResult{
		Services: make([]*TestResult, 0),
		Success:  true,
	}

	// Filter services if needed
	services := o.services
	if len(serviceFilter) > 0 {
		services = filterServices(o.services, serviceFilter)
	}

	if len(services) == 0 {
		return nil, fmt.Errorf("no services to test")
	}

	// Execute tests for each service
	for _, service := range services {
		testResult, err := o.executeServiceTests(service, testType)
		if err != nil {
			if o.config.FailFast {
				return nil, fmt.Errorf("test failed for service %s: %w", service.Name, err)
			}
			// Continue with other services
			testResult = &TestResult{
				Service: service.Name,
				Success: false,
				Error:   err.Error(),
			}
		}

		result.Services = append(result.Services, testResult)
		result.Passed += testResult.Passed
		result.Failed += testResult.Failed
		result.Skipped += testResult.Skipped
		result.Total += testResult.Total
		result.Duration += testResult.Duration

		if !testResult.Success {
			result.Success = false
		}
	}

	return result, nil
}

// executeServiceTests runs tests for a single service.
func (o *TestOrchestrator) executeServiceTests(service ServiceInfo, testType string) (*TestResult, error) {
	// Detect test configuration
	config, err := o.DetectTestConfig(service)
	if err != nil {
		return nil, fmt.Errorf("failed to detect test config: %w", err)
	}

	// Create appropriate test runner based on language
	var runner TestRunner
	switch strings.ToLower(service.Language) {
	case "js", "javascript", "typescript", "ts":
		runner = NewNodeTestRunner(service.Dir, config)
	case "python", "py":
		runner = NewPythonTestRunner(service.Dir, config)
	case "csharp", "dotnet", "fsharp", "cs", "fs":
		// TODO: Implement .NET runner
		return nil, fmt.Errorf(".NET test runner not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported language: %s", service.Language)
	}

	// Execute tests (coverage flag from config)
	coverageEnabled := false
	if o.config != nil && o.config.CoverageThreshold > 0 {
		coverageEnabled = true
	}

	result, err := runner.RunTests(testType, coverageEnabled)
	if err != nil {
		return nil, err
	}

	result.Service = service.Name
	return result, nil
}

// TestRunner interface for language-specific test runners.
type TestRunner interface {
	RunTests(testType string, coverage bool) (*TestResult, error)
}

// Helper functions

// detectNodeTestFramework detects the Node.js test framework.
func detectNodeTestFramework(dir string) (string, error) {
	// Check for configuration files
	configFiles := map[string]string{
		"jest.config.js":     "jest",
		"jest.config.ts":     "jest",
		"jest.config.json":   "jest",
		"vitest.config.js":   "vitest",
		"vitest.config.ts":   "vitest",
		".mocharc.js":        "mocha",
		".mocharc.json":      "mocha",
		".mocharc.yaml":      "mocha",
	}

	for file, framework := range configFiles {
		if _, err := os.Stat(filepath.Join(dir, file)); err == nil {
			return framework, nil
		}
	}

	// Check package.json for test script and dependencies
	packageJSONPath := filepath.Join(dir, "package.json")
	if _, err := os.Stat(packageJSONPath); err == nil {
		// #nosec G304 -- Path is constructed safely
		data, err := os.ReadFile(packageJSONPath)
		if err == nil {
			content := string(data)
			if strings.Contains(content, `"jest"`) {
				return "jest", nil
			}
			if strings.Contains(content, `"vitest"`) {
				return "vitest", nil
			}
			if strings.Contains(content, `"mocha"`) {
				return "mocha", nil
			}
		}
	}

	// Default to npm test
	return "npm", nil
}

// detectPythonTestFramework detects the Python test framework.
func detectPythonTestFramework(dir string) (string, error) {
	// Check for pytest configuration
	pytestFiles := []string{"pytest.ini", "pyproject.toml", "setup.cfg"}
	for _, file := range pytestFiles {
		if _, err := os.Stat(filepath.Join(dir, file)); err == nil {
			return "pytest", nil
		}
	}

	// Check for tests directory
	if _, err := os.Stat(filepath.Join(dir, "tests")); err == nil {
		return "pytest", nil
	}

	return "pytest", nil // Default to pytest
}

// detectDotnetTestFramework detects the .NET test framework.
func detectDotnetTestFramework(dir string) (string, error) {
	// Find test projects
	testProjects, err := detector.FindDotnetProjects(dir)
	if err != nil {
		return "", err
	}

	for _, proj := range testProjects {
		// Check if it's a test project
		if strings.Contains(strings.ToLower(proj.Path), "test") {
			// Read project file to detect framework
			// #nosec G304 -- Path from detector.FindDotnetProjects
			data, err := os.ReadFile(proj.Path)
			if err == nil {
				content := string(data)
				if strings.Contains(content, "xunit") {
					return "xunit", nil
				}
				if strings.Contains(content, "NUnit") {
					return "nunit", nil
				}
				if strings.Contains(content, "MSTest") {
					return "mstest", nil
				}
			}
		}
	}

	return "xunit", nil // Default to xUnit
}

// filterServices filters services by name.
func filterServices(services []ServiceInfo, filter []string) []ServiceInfo {
	if len(filter) == 0 {
		return services
	}

	filterMap := make(map[string]bool)
	for _, name := range filter {
		filterMap[strings.TrimSpace(name)] = true
	}

	filtered := make([]ServiceInfo, 0)
	for _, svc := range services {
		if filterMap[svc.Name] {
			filtered = append(filtered, svc)
		}
	}

	return filtered
}

// FindAzureYaml finds the azure.yaml file in the current or parent directories.
func FindAzureYaml() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	return detector.FindAzureYaml(cwd)
}
