// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package testing

import (
	"os"
	"path/filepath"
	"testing"
)

// TestIntegration_LoadPolyglotProject tests loading the polyglot test project.
func TestIntegration_LoadPolyglotProject(t *testing.T) {
	// Find the polyglot test project
	projectPath := findPolyglotProject(t)
	if projectPath == "" {
		t.Skip("Polyglot test project not found")
	}

	azureYamlPath := filepath.Join(projectPath, "azure.yaml")
	if _, err := os.Stat(azureYamlPath); os.IsNotExist(err) {
		t.Fatalf("azure.yaml not found at %s", azureYamlPath)
	}

	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	err := orchestrator.LoadServicesFromAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to load services: %v", err)
	}

	// Should have 4 services
	if len(orchestrator.services) != 4 {
		t.Errorf("Expected 4 services, got %d", len(orchestrator.services))
	}

	// Check each service exists
	serviceNames := make(map[string]bool)
	for _, svc := range orchestrator.services {
		serviceNames[svc.Name] = true
	}

	expectedServices := []string{"node-api", "python-worker", "go-service", "dotnet-api"}
	for _, name := range expectedServices {
		if !serviceNames[name] {
			t.Errorf("Expected service '%s' not found", name)
		}
	}
}

// TestIntegration_DetectFrameworks tests framework detection for polyglot project.
func TestIntegration_DetectFrameworks(t *testing.T) {
	projectPath := findPolyglotProject(t)
	if projectPath == "" {
		t.Skip("Polyglot test project not found")
	}

	azureYamlPath := filepath.Join(projectPath, "azure.yaml")
	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	if err := orchestrator.LoadServicesFromAzureYaml(azureYamlPath); err != nil {
		t.Fatalf("Failed to load services: %v", err)
	}

	tests := []struct {
		serviceName       string
		expectedFramework string
	}{
		{"node-api", "vitest"},
		{"python-worker", "pytest"},
		{"go-service", "gotest"},
		{"dotnet-api", "xunit"},
	}

	for _, tt := range tests {
		t.Run(tt.serviceName, func(t *testing.T) {
			var service *ServiceInfo
			for i := range orchestrator.services {
				if orchestrator.services[i].Name == tt.serviceName {
					service = &orchestrator.services[i]
					break
				}
			}

			if service == nil {
				t.Fatalf("Service '%s' not found", tt.serviceName)
			}

			testConfig, err := orchestrator.DetectTestConfig(*service)
			if err != nil {
				t.Fatalf("Failed to detect test config: %v", err)
			}

			if testConfig.Framework != tt.expectedFramework {
				t.Errorf("Expected framework '%s', got '%s'", tt.expectedFramework, testConfig.Framework)
			}
		})
	}
}

// TestIntegration_ServiceFiltering tests filtering services by name.
func TestIntegration_ServiceFiltering(t *testing.T) {
	projectPath := findPolyglotProject(t)
	if projectPath == "" {
		t.Skip("Polyglot test project not found")
	}

	azureYamlPath := filepath.Join(projectPath, "azure.yaml")
	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	if err := orchestrator.LoadServicesFromAzureYaml(azureYamlPath); err != nil {
		t.Fatalf("Failed to load services: %v", err)
	}

	tests := []struct {
		name     string
		filter   []string
		expected int
	}{
		{"no filter", nil, 4},
		{"single service", []string{"node-api"}, 1},
		{"multiple services", []string{"node-api", "go-service"}, 2},
		{"non-existent", []string{"unknown"}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterServices(orchestrator.services, tt.filter)
			if len(filtered) != tt.expected {
				t.Errorf("Expected %d services, got %d", tt.expected, len(filtered))
			}
		})
	}
}

// TestIntegration_GetServicePaths tests getting service paths from polyglot project.
func TestIntegration_GetServicePaths(t *testing.T) {
	projectPath := findPolyglotProject(t)
	if projectPath == "" {
		t.Skip("Polyglot test project not found")
	}

	azureYamlPath := filepath.Join(projectPath, "azure.yaml")
	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	if err := orchestrator.LoadServicesFromAzureYaml(azureYamlPath); err != nil {
		t.Fatalf("Failed to load services: %v", err)
	}

	paths, err := orchestrator.GetServicePaths()
	if err != nil {
		t.Fatalf("GetServicePaths failed: %v", err)
	}

	if len(paths) != 4 {
		t.Errorf("Expected 4 paths, got %d", len(paths))
	}

	// Verify all paths exist
	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Service path does not exist: %s", path)
		}
	}
}

// TestIntegration_GoRunner_HasTests tests Go runner detection on real project.
func TestIntegration_GoRunner_HasTests(t *testing.T) {
	projectPath := findPolyglotProject(t)
	if projectPath == "" {
		t.Skip("Polyglot test project not found")
	}

	goServicePath := filepath.Join(projectPath, "services", "go-service")
	if _, err := os.Stat(goServicePath); os.IsNotExist(err) {
		t.Skip("Go service not found")
	}

	config := &ServiceTestConfig{Framework: "gotest"}
	runner := NewGoTestRunner(goServicePath, config)

	hasTests := runner.HasTests()
	if !hasTests {
		t.Error("Expected Go service to have tests")
	}
}

// TestIntegration_NodeRunner_HasTests tests Node runner detection on real project.
func TestIntegration_NodeRunner_HasTests(t *testing.T) {
	projectPath := findPolyglotProject(t)
	if projectPath == "" {
		t.Skip("Polyglot test project not found")
	}

	nodeServicePath := filepath.Join(projectPath, "services", "node-api")
	if _, err := os.Stat(nodeServicePath); os.IsNotExist(err) {
		t.Skip("Node service not found")
	}

	config := &ServiceTestConfig{Framework: "vitest"}
	runner := NewNodeTestRunner(nodeServicePath, config)

	hasTests := runner.HasTests()
	if !hasTests {
		t.Error("Expected Node service to have tests")
	}
}

// TestIntegration_PythonRunner_HasTests tests Python runner detection on real project.
func TestIntegration_PythonRunner_HasTests(t *testing.T) {
	projectPath := findPolyglotProject(t)
	if projectPath == "" {
		t.Skip("Polyglot test project not found")
	}

	pythonServicePath := filepath.Join(projectPath, "services", "python-worker")
	if _, err := os.Stat(pythonServicePath); os.IsNotExist(err) {
		t.Skip("Python service not found")
	}

	config := &ServiceTestConfig{Framework: "pytest"}
	runner := NewPythonTestRunner(pythonServicePath, config)

	hasTests := runner.HasTests()
	if !hasTests {
		t.Error("Expected Python service to have tests")
	}
}

// TestIntegration_DotnetRunner_HasTests tests .NET runner detection on real project.
func TestIntegration_DotnetRunner_HasTests(t *testing.T) {
	projectPath := findPolyglotProject(t)
	if projectPath == "" {
		t.Skip("Polyglot test project not found")
	}

	dotnetServicePath := filepath.Join(projectPath, "services", "dotnet-api")
	if _, err := os.Stat(dotnetServicePath); os.IsNotExist(err) {
		t.Skip(".NET service not found")
	}

	config := &ServiceTestConfig{Framework: "xunit"}
	runner := NewDotnetTestRunner(dotnetServicePath, config)

	hasTests := runner.HasTests()
	if !hasTests {
		t.Error("Expected .NET service to have tests")
	}
}

// TestIntegration_ReportGenerator_OutputFormats tests report generation.
func TestIntegration_ReportGenerator_OutputFormats(t *testing.T) {
	tmpDir := t.TempDir()

	results := &AggregateResult{
		Services: []*TestResult{
			{
				Service: "test-service",
				Passed:  5,
				Failed:  1,
				Skipped: 1,
				Total:   7,
				Success: false,
				Failures: []TestFailure{
					{Name: "TestFailing", Message: "assertion failed"},
				},
			},
		},
		Passed:   5,
		Failed:   1,
		Skipped:  1,
		Total:    7,
		Duration: 2.5,
		Success:  false,
	}

	formats := []string{"json", "junit"}
	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			gen := NewReportGenerator(format, tmpDir)
			err := gen.GenerateTestReport(results)
			if err != nil {
				t.Fatalf("GenerateTestReport failed for %s: %v", format, err)
			}

			// Verify file was created
			var expectedFile string
			switch format {
			case "json":
				expectedFile = filepath.Join(tmpDir, "test-results.json")
			case "junit":
				expectedFile = filepath.Join(tmpDir, "test-results.xml")
			}

			if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
				t.Errorf("Expected output file not created: %s", expectedFile)
			}
		})
	}
}

// findPolyglotProject searches for the polyglot-test project.
func findPolyglotProject(t *testing.T) string {
	t.Helper()

	// Try relative paths from test location
	candidates := []string{
		"../../../tests/projects/integration/polyglot-test",
		"../../../../tests/projects/integration/polyglot-test",
		"../../../../../cli/tests/projects/integration/polyglot-test",
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	for _, candidate := range candidates {
		path := filepath.Join(cwd, candidate)
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}

		azureYaml := filepath.Join(absPath, "azure.yaml")
		if _, err := os.Stat(azureYaml); err == nil {
			return absPath
		}
	}

	// Try using environment variable or known paths
	if envPath := os.Getenv("POLYGLOT_TEST_PROJECT"); envPath != "" {
		if _, err := os.Stat(filepath.Join(envPath, "azure.yaml")); err == nil {
			return envPath
		}
	}

	return ""
}
