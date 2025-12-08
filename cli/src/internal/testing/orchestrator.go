// Package testing provides test execution and coverage aggregation for multi-language projects.
package testing

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/logging"
	"github.com/jongio/azd-app/cli/src/internal/security"
	"gopkg.in/yaml.v3"
)

// DefaultTestTimeout is the default timeout for test execution per service.
const DefaultTestTimeout = 10 * time.Minute

// ProgressCallback is a function that gets called during test execution to report progress.
type ProgressCallback func(event ProgressEvent)

// ProgressEvent represents a progress update during test execution.
type ProgressEvent struct {
	// Type indicates the type of progress event
	Type ProgressEventType
	// Service is the name of the service (if applicable)
	Service string
	// Framework is the test framework being used (if applicable)
	Framework string
	// Message is an optional message for the event
	Message string
}

// ProgressEventType represents the type of progress event.
type ProgressEventType int

const (
	// ProgressEventValidationStart indicates validation is starting
	ProgressEventValidationStart ProgressEventType = iota
	// ProgressEventServiceValidated indicates a service has been validated
	ProgressEventServiceValidated
	// ProgressEventValidationComplete indicates validation is complete
	ProgressEventValidationComplete
	// ProgressEventTestStart indicates tests are starting for a service
	ProgressEventTestStart
	// ProgressEventTestComplete indicates tests have completed for a service
	ProgressEventTestComplete
	// ProgressEventServiceSkipped indicates a service was skipped
	ProgressEventServiceSkipped
)

// TestOrchestrator manages test execution across services.
type TestOrchestrator struct {
	config           *TestConfig
	services         []ServiceInfo
	progressCallback ProgressCallback
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
		config:           config,
		services:         make([]ServiceInfo, 0),
		progressCallback: nil,
	}
}

// SetProgressCallback sets the callback function for progress updates.
func (o *TestOrchestrator) SetProgressCallback(callback ProgressCallback) {
	o.progressCallback = callback
}

// emitProgress emits a progress event if a callback is set.
func (o *TestOrchestrator) emitProgress(event ProgressEvent) {
	if o.progressCallback != nil {
		o.progressCallback(event)
	}
}

// GetServices returns the loaded services.
func (o *TestOrchestrator) GetServices() []ServiceInfo {
	return o.services
}

// ValidateAllServices validates all loaded services for testability.
func (o *TestOrchestrator) ValidateAllServices() []ServiceValidation {
	o.emitProgress(ProgressEvent{
		Type:    ProgressEventValidationStart,
		Message: fmt.Sprintf("Validating %d services", len(o.services)),
	})

	validations := make([]ServiceValidation, 0, len(o.services))
	for _, service := range o.services {
		validation := ValidateService(service)
		validations = append(validations, validation)

		o.emitProgress(ProgressEvent{
			Type:      ProgressEventServiceValidated,
			Service:   validation.Name,
			Framework: validation.Framework,
			Message:   validation.SkipReason,
		})
	}

	o.emitProgress(ProgressEvent{
		Type: ProgressEventValidationComplete,
	})

	return validations
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
	azureYamlDirAbs, err := filepath.Abs(azureYamlDir)
	if err != nil {
		return fmt.Errorf("failed to resolve azure.yaml directory: %w", err)
	}

	for name, svc := range azureYaml.Services {
		// Resolve project directory
		projectDir := svc.Project
		if !filepath.IsAbs(projectDir) {
			projectDir = filepath.Join(azureYamlDir, projectDir)
		}

		// Normalize the path
		projectDir = filepath.Clean(projectDir)

		// Security: Validate project directory stays within azure.yaml directory
		// This prevents path traversal attacks via malicious azure.yaml
		projectDirAbs, err := filepath.Abs(projectDir)
		if err != nil {
			return fmt.Errorf("failed to resolve project directory for service %s: %w", name, err)
		}

		// Check that the project directory is under the azure.yaml directory
		if !strings.HasPrefix(projectDirAbs, azureYamlDirAbs) {
			return fmt.Errorf("service %s project path '%s' escapes project boundary", name, svc.Project)
		}

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

	case "go", "golang":
		framework, err := detectGoTestFramework(service.Dir)
		if err != nil {
			return nil, fmt.Errorf("failed to detect Go test framework: %w", err)
		}
		config.Framework = framework

	default:
		return nil, fmt.Errorf("unsupported language: %s", service.Language)
	}

	// Auto-detect test type configurations if not specified
	if config.Unit == nil && config.Integration == nil && config.E2E == nil {
		detectedConfig := SuggestTestTypeConfig(service.Dir, service.Language)
		config.Unit = detectedConfig.Unit
		config.Integration = detectedConfig.Integration
		config.E2E = detectedConfig.E2E
	}

	return config, nil
}

// GetAvailableTestTypesForService returns available test types for a service.
func (o *TestOrchestrator) GetAvailableTestTypesForService(service ServiceInfo) []string {
	detector := NewTestTypeDetector(service.Dir, service.Language)
	return detector.GetAvailableTestTypes()
}

// GetAvailableTestTypes returns a map of available test types per service.
func (o *TestOrchestrator) GetAvailableTestTypes() map[string][]string {
	result := make(map[string][]string)
	for _, service := range o.services {
		result[service.Name] = o.GetAvailableTestTypesForService(service)
	}
	return result
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

	// Initialize coverage aggregator if coverage is enabled
	var coverageAggregator *CoverageAggregator
	if o.config != nil && o.config.CoverageThreshold > 0 {
		outputDir := o.config.OutputDir
		if outputDir == "" {
			outputDir = "./coverage"
		}
		coverageAggregator = NewCoverageAggregator(o.config.CoverageThreshold, outputDir)
	}

	// Execute tests for each service
	for _, service := range services {
		// Emit test start progress event
		o.emitProgress(ProgressEvent{
			Type:    ProgressEventTestStart,
			Service: service.Name,
		})

		testResult, err := o.executeServiceTests(service, testType)
		if err != nil {
			if o.config != nil && o.config.FailFast {
				return nil, fmt.Errorf("test failed for service %s: %w", service.Name, err)
			}
			// Continue with other services
			testResult = &TestResult{
				Service: service.Name,
				Success: false,
				Error:   err.Error(),
			}
		}

		// Emit test complete progress event
		o.emitProgress(ProgressEvent{
			Type:    ProgressEventTestComplete,
			Service: service.Name,
		})

		result.Services = append(result.Services, testResult)
		result.Passed += testResult.Passed
		result.Failed += testResult.Failed
		result.Skipped += testResult.Skipped
		result.Total += testResult.Total
		result.Duration += testResult.Duration

		if !testResult.Success {
			result.Success = false
		}

		// Add coverage if available
		if coverageAggregator != nil && testResult.Coverage != nil {
			if err := coverageAggregator.AddCoverage(service.Name, testResult.Coverage); err != nil {
				log := logging.NewLogger("test")
				log.Warn("failed to add coverage data", "service", service.Name, "error", err.Error())
			}
		}
	}

	// Aggregate coverage and check threshold
	if coverageAggregator != nil {
		result.Coverage = coverageAggregator.Aggregate()

		// Check threshold
		meetsThreshold, percentage := coverageAggregator.CheckThreshold()
		if !meetsThreshold {
			result.Success = false
			result.Error = fmt.Sprintf("Coverage %.1f%% is below threshold %.1f%%", percentage, o.config.CoverageThreshold)
		}

		// Generate coverage reports in multiple formats
		log := logging.NewLogger("test")
		if err := coverageAggregator.GenerateReport("json"); err != nil {
			log.Warn("failed to generate JSON coverage report", "error", err.Error())
		}
		if err := coverageAggregator.GenerateReport("html"); err != nil {
			log.Warn("failed to generate HTML coverage report", "error", err.Error())
		}
		if err := coverageAggregator.GenerateReport("cobertura"); err != nil {
			log.Warn("failed to generate Cobertura coverage report", "error", err.Error())
		}
	}

	return result, nil
}

// ExecuteTestsWithValidation validates services and runs tests only for testable services.
// Returns validation results along with test results.
func (o *TestOrchestrator) ExecuteTestsWithValidation(testType string, serviceFilter []string) (*AggregateResult, []ServiceValidation, error) {
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
		return nil, nil, fmt.Errorf("no services to test")
	}

	// Validate all services first
	o.emitProgress(ProgressEvent{
		Type:    ProgressEventValidationStart,
		Message: fmt.Sprintf("Analyzing %d services", len(services)),
	})

	validations := make([]ServiceValidation, 0, len(services))
	for _, service := range services {
		validation := ValidateService(service)
		validations = append(validations, validation)

		o.emitProgress(ProgressEvent{
			Type:      ProgressEventServiceValidated,
			Service:   validation.Name,
			Framework: validation.Framework,
			Message:   validation.SkipReason,
		})
	}

	o.emitProgress(ProgressEvent{
		Type: ProgressEventValidationComplete,
	})

	// Get testable services
	testableServices := make([]ServiceInfo, 0)
	for i, v := range validations {
		if v.CanTest {
			testableServices = append(testableServices, services[i])
		} else {
			// Emit skip event for non-testable services
			o.emitProgress(ProgressEvent{
				Type:    ProgressEventServiceSkipped,
				Service: v.Name,
				Message: v.SkipReason,
			})
		}
	}

	if len(testableServices) == 0 {
		return result, validations, nil
	}

	// Initialize coverage aggregator if coverage is enabled
	var coverageAggregator *CoverageAggregator
	if o.config != nil && o.config.CoverageThreshold > 0 {
		outputDir := o.config.OutputDir
		if outputDir == "" {
			outputDir = "./coverage"
		}
		coverageAggregator = NewCoverageAggregator(o.config.CoverageThreshold, outputDir)
	}

	// Execute tests for each testable service
	for _, service := range testableServices {
		// Find the validation for this service to get framework info
		var framework string
		for _, v := range validations {
			if v.Name == service.Name {
				framework = v.Framework
				break
			}
		}

		// Emit test start progress event
		o.emitProgress(ProgressEvent{
			Type:      ProgressEventTestStart,
			Service:   service.Name,
			Framework: framework,
		})

		testResult, err := o.executeServiceTests(service, testType)
		if err != nil {
			if o.config != nil && o.config.FailFast {
				return nil, validations, fmt.Errorf("test failed for service %s: %w", service.Name, err)
			}
			// Continue with other services
			testResult = &TestResult{
				Service: service.Name,
				Success: false,
				Error:   err.Error(),
			}
		}

		// Emit test complete progress event
		o.emitProgress(ProgressEvent{
			Type:    ProgressEventTestComplete,
			Service: service.Name,
		})

		result.Services = append(result.Services, testResult)
		result.Passed += testResult.Passed
		result.Failed += testResult.Failed
		result.Skipped += testResult.Skipped
		result.Total += testResult.Total
		result.Duration += testResult.Duration

		if !testResult.Success {
			result.Success = false
		}

		// Add coverage if available
		if coverageAggregator != nil && testResult.Coverage != nil {
			if err := coverageAggregator.AddCoverage(service.Name, testResult.Coverage); err != nil {
				log := logging.NewLogger("test")
				log.Warn("failed to add coverage data", "service", service.Name, "error", err.Error())
			}
		}
	}

	// Aggregate coverage and check threshold
	if coverageAggregator != nil {
		result.Coverage = coverageAggregator.Aggregate()

		// Check threshold
		meetsThreshold, percentage := coverageAggregator.CheckThreshold()
		if !meetsThreshold {
			result.Success = false
			result.Error = fmt.Sprintf("Coverage %.1f%% is below threshold %.1f%%", percentage, o.config.CoverageThreshold)
		}

		// Generate coverage reports in multiple formats
		log := logging.NewLogger("test")
		if err := coverageAggregator.GenerateReport("json"); err != nil {
			log.Warn("failed to generate JSON coverage report", "error", err.Error())
		}
		if err := coverageAggregator.GenerateReport("html"); err != nil {
			log.Warn("failed to generate HTML coverage report", "error", err.Error())
		}
		if err := coverageAggregator.GenerateReport("cobertura"); err != nil {
			log.Warn("failed to generate Cobertura coverage report", "error", err.Error())
		}
	}

	return result, validations, nil
}

// executeServiceTests runs tests for a single service.
func (o *TestOrchestrator) executeServiceTests(service ServiceInfo, testType string) (*TestResult, error) {
	// Detect test configuration
	config, err := o.DetectTestConfig(service)
	if err != nil {
		return nil, fmt.Errorf("failed to detect test config: %w", err)
	}

	// Get test type config for setup/teardown
	var typeConfig *TestTypeConfig
	switch testType {
	case "unit":
		typeConfig = config.Unit
	case "integration":
		typeConfig = config.Integration
	case "e2e":
		typeConfig = config.E2E
	}

	// Execute setup commands
	if typeConfig != nil && len(typeConfig.Setup) > 0 {
		if err := o.executeCommands(service.Dir, typeConfig.Setup, "setup"); err != nil {
			return nil, fmt.Errorf("setup failed: %w", err)
		}
	}

	// Ensure teardown runs even if tests fail
	var result *TestResult
	var testErr error
	defer func() {
		if typeConfig != nil && len(typeConfig.Teardown) > 0 {
			if err := o.executeCommands(service.Dir, typeConfig.Teardown, "teardown"); err != nil {
				log := logging.NewLogger("test")
				log.Warn("teardown failed", "service", service.Name, "error", err.Error())
			}
		}
	}()

	// Create appropriate test runner based on language
	var runner TestRunner
	switch strings.ToLower(service.Language) {
	case "js", "javascript", "typescript", "ts":
		runner = NewNodeTestRunner(service.Dir, config)
	case "python", "py":
		runner = NewPythonTestRunner(service.Dir, config)
	case "csharp", "dotnet", "fsharp", "cs", "fs":
		runner = NewDotnetTestRunner(service.Dir, config)
	case "go", "golang":
		runner = NewGoTestRunner(service.Dir, config)
	default:
		return nil, fmt.Errorf("unsupported language: %s", service.Language)
	}

	// Execute tests (coverage flag from config)
	coverageEnabled := false
	if o.config != nil && o.config.CoverageThreshold > 0 {
		coverageEnabled = true
	}

	// Determine timeout
	timeout := DefaultTestTimeout
	if o.config != nil && o.config.Timeout > 0 {
		timeout = o.config.Timeout
	}

	// Execute tests with timeout
	result, testErr = o.executeWithTimeout(runner, testType, coverageEnabled, timeout)
	if testErr != nil {
		return nil, testErr
	}

	result.Service = service.Name
	return result, nil
}

// executeWithTimeout runs tests with a timeout.
// Returns a clear error message if the timeout is exceeded.
func (o *TestOrchestrator) executeWithTimeout(runner TestRunner, testType string, coverage bool, timeout time.Duration) (*TestResult, error) {
	type runResult struct {
		result *TestResult
		err    error
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Run tests in a goroutine
	resultChan := make(chan runResult, 1)
	go func() {
		result, err := runner.RunTests(testType, coverage)
		resultChan <- runResult{result: result, err: err}
	}()

	// Wait for either completion or timeout
	select {
	case <-ctx.Done():
		// Timeout exceeded
		return nil, fmt.Errorf("test execution timed out after %s", timeout)
	case res := <-resultChan:
		return res.result, res.err
	}
}

// TestRunner interface for language-specific test runners.
type TestRunner interface {
	RunTests(testType string, coverage bool) (*TestResult, error)
}

// GetServicePaths returns the paths of all services for file watching.
func (o *TestOrchestrator) GetServicePaths() ([]string, error) {
	paths := make([]string, 0, len(o.services))
	for _, service := range o.services {
		paths = append(paths, service.Dir)
	}
	return paths, nil
}

// Helper functions

// detectNodeTestFramework detects the Node.js test framework.
func detectNodeTestFramework(dir string) (string, error) {
	// Check for configuration files
	configFiles := map[string]string{
		"jest.config.js":   "jest",
		"jest.config.ts":   "jest",
		"jest.config.json": "jest",
		"vitest.config.js": "vitest",
		"vitest.config.ts": "vitest",
		".mocharc.js":      "mocha",
		".mocharc.json":    "mocha",
		".mocharc.yaml":    "mocha",
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

// detectGoTestFramework detects the Go test framework.
func detectGoTestFramework(dir string) (string, error) {
	// Go only has the standard testing package
	// Check if go.mod exists and there are test files
	goModPath := filepath.Join(dir, "go.mod")
	if _, err := os.Stat(goModPath); err != nil {
		return "", fmt.Errorf("go.mod not found in %s", dir)
	}

	// Check for *_test.go files
	hasTests := false
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), "_test.go") {
				hasTests = true
				break
			}
		}
	}

	// Also check subdirectories for test files
	if !hasTests {
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() && strings.HasSuffix(d.Name(), "_test.go") {
				hasTests = true
				return filepath.SkipAll
			}
			return nil
		})
	}

	if !hasTests {
		return "", fmt.Errorf("no test files found in %s", dir)
	}

	return "gotest", nil
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

// executeCommands executes a list of commands in the specified directory.
func (o *TestOrchestrator) executeCommands(dir string, commands []string, stage string) error {
	log := logging.NewLogger("test").WithOperation(stage)
	for i, cmd := range commands {
		if !logging.IsStructured() {
			fmt.Printf("Running %s command %d/%d: %s\n", stage, i+1, len(commands), cmd)
		}
		log.Debug("executing command", "stage", stage, "index", i+1, "total", len(commands), "command", cmd)

		// Execute command using os/exec
		if err := runCommand(dir, cmd); err != nil {
			return fmt.Errorf("command '%s' failed: %w", cmd, err)
		}
	}
	return nil
}

// runCommand executes a command in the specified directory.
// Parses the command string into command and arguments to avoid shell injection.
func runCommand(dir, cmd string) error {
	parts := parseCommandString(cmd)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	// #nosec G204 -- Command parts are validated and from azure.yaml
	command := exec.Command(parts[0], parts[1:]...)
	command.Dir = dir
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

// parseCommandString parses a command string into command and arguments.
// Handles quoted strings and basic shell-style arguments.
func parseCommandString(cmd string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, r := range cmd {
		switch {
		case r == '"' || r == '\'':
			if inQuote && r == quoteChar {
				// End of quoted section
				inQuote = false
				quoteChar = 0
			} else if !inQuote {
				// Start of quoted section
				inQuote = true
				quoteChar = r
			} else {
				// Different quote inside - treat as literal
				current.WriteRune(r)
			}
		case r == ' ' || r == '\t':
			if inQuote {
				current.WriteRune(r)
			} else if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	// Add the last part if any
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}
