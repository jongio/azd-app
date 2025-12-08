// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package testing

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDiscovery_LoadDiscoveryProject tests loading the discovery test project.
func TestDiscovery_LoadDiscoveryProject(t *testing.T) {
	projectPath := findDiscoveryProject(t)
	if projectPath == "" {
		t.Skip("Discovery test project not found")
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

	// Should have 6 services
	if len(orchestrator.services) != 6 {
		t.Errorf("Expected 6 services, got %d", len(orchestrator.services))
	}
}

// TestDiscovery_ValidateServices tests the validation of services for testability.
func TestDiscovery_ValidateServices(t *testing.T) {
	projectPath := findDiscoveryProject(t)
	if projectPath == "" {
		t.Skip("Discovery test project not found")
	}

	azureYamlPath := filepath.Join(projectPath, "azure.yaml")
	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	if err := orchestrator.LoadServicesFromAzureYaml(azureYamlPath); err != nil {
		t.Fatalf("Failed to load services: %v", err)
	}

	validations := orchestrator.ValidateAllServices()

	// Expected results
	expected := map[string]struct {
		canTest   bool
		framework string
	}{
		"web":     {canTest: true, framework: "vitest"},
		"api":     {canTest: true, framework: "jest"},
		"backend": {canTest: true, framework: "pytest"},
		"gateway": {canTest: true, framework: "gotest"},
		"config":  {canTest: false, framework: ""},
		"nested":  {canTest: true, framework: "jest"},
	}

	for _, v := range validations {
		exp, ok := expected[v.Name]
		if !ok {
			t.Errorf("Unexpected service: %s", v.Name)
			continue
		}

		if v.CanTest != exp.canTest {
			t.Errorf("Service %s: expected CanTest=%v, got %v (reason: %s)",
				v.Name, exp.canTest, v.CanTest, v.SkipReason)
		}

		if exp.canTest && v.Framework != exp.framework {
			t.Errorf("Service %s: expected framework=%s, got %s",
				v.Name, exp.framework, v.Framework)
		}
	}
}

// TestDiscovery_FrameworkDetection tests framework detection for various scenarios.
func TestDiscovery_FrameworkDetection(t *testing.T) {
	projectPath := findDiscoveryProject(t)
	if projectPath == "" {
		t.Skip("Discovery test project not found")
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
		description       string
	}{
		{"web", "vitest", "vitest.config.ts detection"},
		{"api", "jest", "jest.config.js detection"},
		{"backend", "pytest", "pyproject.toml pytest detection"},
		{"gateway", "gotest", "go.mod + *_test.go detection"},
		{"nested", "jest", "package.json jest dependency"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
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

// TestDiscovery_TestFileCount tests counting test files for each service.
func TestDiscovery_TestFileCount(t *testing.T) {
	projectPath := findDiscoveryProject(t)
	if projectPath == "" {
		t.Skip("Discovery test project not found")
	}

	azureYamlPath := filepath.Join(projectPath, "azure.yaml")
	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	if err := orchestrator.LoadServicesFromAzureYaml(azureYamlPath); err != nil {
		t.Fatalf("Failed to load services: %v", err)
	}

	validations := orchestrator.ValidateAllServices()

	// All testable services should have at least 1 test file
	for _, v := range validations {
		if v.CanTest && v.TestFiles == 0 {
			t.Errorf("Service %s is testable but has 0 test files", v.Name)
		}
		if !v.CanTest && v.Name == "config" && v.TestFiles > 0 {
			t.Errorf("Service config should have 0 test files, got %d", v.TestFiles)
		}
	}
}

// TestDiscovery_SkipReason tests skip reasons are properly set.
func TestDiscovery_SkipReason(t *testing.T) {
	projectPath := findDiscoveryProject(t)
	if projectPath == "" {
		t.Skip("Discovery test project not found")
	}

	azureYamlPath := filepath.Join(projectPath, "azure.yaml")
	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	if err := orchestrator.LoadServicesFromAzureYaml(azureYamlPath); err != nil {
		t.Fatalf("Failed to load services: %v", err)
	}

	validations := orchestrator.ValidateAllServices()

	// Find the config service (should be skipped)
	for _, v := range validations {
		if v.Name == "config" {
			if v.CanTest {
				t.Error("config service should not be testable")
			}
			if v.SkipReason == "" {
				t.Error("config service should have a skip reason")
			}
			t.Logf("config skip reason: %s", v.SkipReason)
		}
	}
}

// TestDiscovery_NestedTestFiles tests detection of deeply nested test files.
func TestDiscovery_NestedTestFiles(t *testing.T) {
	projectPath := findDiscoveryProject(t)
	if projectPath == "" {
		t.Skip("Discovery test project not found")
	}

	nestedPath := filepath.Join(projectPath, "nested")
	if _, err := os.Stat(nestedPath); os.IsNotExist(err) {
		t.Skip("Nested service not found")
	}

	// Validate the nested service
	service := ServiceInfo{
		Name:     "nested",
		Language: "ts",
		Dir:      nestedPath,
	}

	validation := ValidateService(service)

	if !validation.CanTest {
		t.Errorf("Nested service should be testable: %s", validation.SkipReason)
	}

	if validation.TestFiles == 0 {
		t.Error("Nested service should have at least 1 test file in __tests__ folder")
	}

	if validation.Framework != "jest" {
		t.Errorf("Expected jest framework, got %s", validation.Framework)
	}
}

// TestDiscovery_ConfigAutoDetection tests that auto-detected services are identified.
func TestDiscovery_ConfigAutoDetection(t *testing.T) {
	projectPath := findDiscoveryProject(t)
	if projectPath == "" {
		t.Skip("Discovery test project not found")
	}

	azureYamlPath := filepath.Join(projectPath, "azure.yaml")
	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	if err := orchestrator.LoadServicesFromAzureYaml(azureYamlPath); err != nil {
		t.Fatalf("Failed to load services: %v", err)
	}

	validations := orchestrator.ValidateAllServices()
	services := orchestrator.GetServices()

	// All services in discovery-test project have no config in azure.yaml
	// so GetAutoDetectedServices should return all testable ones
	autoDetected := GetAutoDetectedServices(validations, services)

	// Should have 5 auto-detected services (all except config)
	expectedCount := 5
	if len(autoDetected) != expectedCount {
		t.Errorf("Expected %d auto-detected services, got %d", expectedCount, len(autoDetected))
		for _, v := range autoDetected {
			t.Logf("  - %s (%s)", v.Name, v.Framework)
		}
	}
}

// TestDiscovery_GenerateYAML tests YAML config generation.
func TestDiscovery_GenerateYAML(t *testing.T) {
	projectPath := findDiscoveryProject(t)
	if projectPath == "" {
		t.Skip("Discovery test project not found")
	}

	azureYamlPath := filepath.Join(projectPath, "azure.yaml")
	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	if err := orchestrator.LoadServicesFromAzureYaml(azureYamlPath); err != nil {
		t.Fatalf("Failed to load services: %v", err)
	}

	validations := orchestrator.ValidateAllServices()
	services := orchestrator.GetServices()
	autoDetected := GetAutoDetectedServices(validations, services)

	yaml := GenerateTestConfigYAML(autoDetected, services)

	// Verify YAML contains expected services
	expectedServices := []string{"web", "api", "backend", "gateway", "nested"}
	for _, svc := range expectedServices {
		if !containsString(yaml, svc+":") {
			t.Errorf("Generated YAML missing service: %s", svc)
		}
	}

	// Verify YAML contains expected frameworks
	expectedFrameworks := []string{"vitest", "jest", "pytest", "gotest"}
	for _, fw := range expectedFrameworks {
		if !containsString(yaml, fw) {
			t.Errorf("Generated YAML missing framework: %s", fw)
		}
	}

	// Should NOT contain config service
	if containsString(yaml, "config:") {
		t.Error("Generated YAML should not contain config service (not testable)")
	}
}

// TestDiscovery_OutputModeSelection tests output mode selection logic.
func TestDiscovery_OutputModeSelection(t *testing.T) {
	// Clear CI variables to test TTY behavior
	ciVars := []string{"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI", "TRAVIS", "JENKINS_URL", "TEAMCITY_VERSION", "TF_BUILD", "BUILDKITE", "CODEBUILD_BUILD_ID"}
	oldVars := make(map[string]string)
	for _, v := range ciVars {
		oldVars[v] = os.Getenv(v)
		os.Unsetenv(v)
	}
	defer func() {
		for k, v := range oldVars {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	tests := []struct {
		name         string
		opts         OutputModeOptions
		serviceCount int
		isTTY        bool
		expectedMode OutputMode
	}{
		{
			name:         "single service streams directly",
			opts:         OutputModeOptions{Parallel: true},
			serviceCount: 1,
			isTTY:        true,
			expectedMode: OutputModeStream,
		},
		{
			name:         "multiple parallel uses progress",
			opts:         OutputModeOptions{Parallel: true},
			serviceCount: 3,
			isTTY:        true,
			expectedMode: OutputModeProgress,
		},
		{
			name:         "multiple sequential uses prefixed",
			opts:         OutputModeOptions{Parallel: false},
			serviceCount: 3,
			isTTY:        true,
			expectedMode: OutputModeStreamPrefixed,
		},
		{
			name:         "non-TTY always streams with prefix for multiple",
			opts:         OutputModeOptions{Parallel: true},
			serviceCount: 3,
			isTTY:        false,
			expectedMode: OutputModeStreamPrefixed,
		},
		{
			name:         "stream flag forces streaming",
			opts:         OutputModeOptions{Parallel: true, ForceStream: true},
			serviceCount: 3,
			isTTY:        true,
			expectedMode: OutputModeStream,
		},
		{
			name:         "no-stream flag forces progress",
			opts:         OutputModeOptions{Parallel: false, ForceProgress: true},
			serviceCount: 1,
			isTTY:        true,
			expectedMode: OutputModeProgress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := SelectOutputMode(tt.opts, tt.serviceCount, tt.isTTY)
			if mode != tt.expectedMode {
				t.Errorf("Expected mode %v, got %v", tt.expectedMode, mode)
			}
		})
	}
}

// containsString checks if a string contains a substring.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[0:len(substr)] == substr || containsString(s[1:], substr)))
}

// findDiscoveryProject searches for the discovery-test project.
func findDiscoveryProject(t *testing.T) string {
	t.Helper()

	// Try relative paths from test location
	candidates := []string{
		"../../../tests/projects/discovery-test",
		"../../../../tests/projects/discovery-test",
		"../../../../../cli/tests/projects/discovery-test",
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

	return ""
}
