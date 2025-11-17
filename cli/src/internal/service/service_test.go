package service_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

func TestParseAzureYaml(t *testing.T) {
	// Create a temporary azure.yaml file
	tmpDir := t.TempDir()
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")

	content := `name: test-app
services:
  web:
    project: ./src/web
    language: js
    host: containerapp
  api:
    project: ./src/api
    language: python
    host: containerapp
    uses:
      - web
resources:
  db:
    type: postgres.database
`

	if err := os.WriteFile(azureYamlPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create test azure.yaml: %v", err)
	}

	// Parse the file
	azureYaml, err := service.ParseAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}

	// Verify services
	if len(azureYaml.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(azureYaml.Services))
	}

	if _, exists := azureYaml.Services["web"]; !exists {
		t.Error("Expected service 'web' not found")
	}

	if _, exists := azureYaml.Services["api"]; !exists {
		t.Error("Expected service 'api' not found")
	}

	// Verify resources
	if len(azureYaml.Resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(azureYaml.Resources))
	}
}

func TestFilterServices(t *testing.T) {
	azureYaml := &service.AzureYaml{
		Services: map[string]service.Service{
			"web": {Host: "containerapp", Project: "./web"},
			"api": {Host: "containerapp", Project: "./api"},
			"db":  {Host: "containerapp", Project: "./db"},
		},
	}

	tests := []struct {
		name     string
		filter   []string
		expected int
	}{
		{"Filter single service", []string{"web"}, 1},
		{"Filter multiple services", []string{"web", "api"}, 2},
		{"Filter all services", []string{"web", "api", "db"}, 3},
		{"Filter non-existent service", []string{"invalid"}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.FilterServices(azureYaml, tt.filter)
			if len(result) != tt.expected {
				t.Errorf("Expected %d services, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestHasServices(t *testing.T) {
	tests := []struct {
		name     string
		yaml     *service.AzureYaml
		expected bool
	}{
		{
			"Has services",
			&service.AzureYaml{
				Services: map[string]service.Service{
					"web": {Host: "containerapp"},
				},
			},
			true,
		},
		{
			"No services",
			&service.AzureYaml{
				Services: map[string]service.Service{},
			},
			false,
		},
		{
			"Nil services",
			&service.AzureYaml{},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.HasServices(tt.yaml)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestPortHealthCheck(t *testing.T) {
	// This test requires a port to be listening
	// Skip if no network is available
	t.Skip("Integration test - requires network setup")
}

func TestBuildDependencyGraph(t *testing.T) {
	services := map[string]service.Service{
		"web": {
			Host: "containerapp",
			Uses: []string{"api"},
		},
		"api": {
			Host: "containerapp",
			Uses: []string{"db"},
		},
	}

	resources := map[string]service.Resource{
		"db": {
			Type: "postgres.database",
		},
	}

	graph, err := service.BuildDependencyGraph(services, resources)
	if err != nil {
		t.Fatalf("Failed to build dependency graph: %v", err)
	}

	if len(graph.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(graph.Nodes))
	}

	// Verify edges
	if len(graph.Edges["web"]) != 1 || graph.Edges["web"][0] != "api" {
		t.Error("Expected web to depend on api")
	}

	if len(graph.Edges["api"]) != 1 || graph.Edges["api"][0] != "db" {
		t.Error("Expected api to depend on db")
	}
}

func TestDetectCycles(t *testing.T) {
	tests := []struct {
		name      string
		services  map[string]service.Service
		shouldErr bool
	}{
		{
			"No cycles",
			map[string]service.Service{
				"web": {Host: "containerapp", Uses: []string{"api"}},
				"api": {Host: "containerapp"},
			},
			false,
		},
		{
			"Simple cycle",
			map[string]service.Service{
				"web": {Host: "containerapp", Uses: []string{"api"}},
				"api": {Host: "containerapp", Uses: []string{"web"}},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph, err := service.BuildDependencyGraph(tt.services, map[string]service.Resource{})
			if err != nil {
				if !tt.shouldErr {
					t.Fatalf("Unexpected error building graph: %v", err)
				}
				return
			}

			err = service.DetectCycles(graph)
			if tt.shouldErr && err == nil {
				t.Error("Expected cycle detection to fail, but it passed")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no cycle, but got error: %v", err)
			}
		})
	}
}

func TestParseAzureYaml_InvalidPath(t *testing.T) {
	_, err := service.ParseAzureYaml("/nonexistent/path/azure.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestParseAzureYaml_InvalidYaml(t *testing.T) {
	tmpDir := t.TempDir()
	invalidYamlPath := filepath.Join(tmpDir, "invalid.yaml")

	invalidContent := `name: test
services:
  web:
    invalid yaml structure with no proper indentation
  - this should fail`

	if err := os.WriteFile(invalidYamlPath, []byte(invalidContent), 0600); err != nil {
		t.Fatalf("Failed to create invalid yaml: %v", err)
	}

	_, err := service.ParseAzureYaml(invalidYamlPath)
	if err == nil {
		t.Error("Expected error parsing invalid YAML")
	}
}

func TestFilterServices_EmptyFilter(t *testing.T) {
	azureYaml := &service.AzureYaml{
		Services: map[string]service.Service{
			"web": {Host: "containerapp", Project: "./web"},
			"api": {Host: "containerapp", Project: "./api"},
		},
	}

	// Empty filter should return all services
	result := service.FilterServices(azureYaml, []string{})
	if len(result) != 2 {
		t.Errorf("Expected all 2 services with empty filter, got %d", len(result))
	}
}

func TestFilterServices_NilYaml(t *testing.T) {
	// FilterServices should handle nil yaml without panicking
	// This was a bug that caused panics - now fixed with nil check
	result := service.FilterServices(nil, []string{"test"})
	if result == nil {
		t.Error("Expected non-nil map, got nil")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty map for nil yaml, got %d services", len(result))
	}
}

func TestBuildDependencyGraph_ComplexDependencies(t *testing.T) {
	services := map[string]service.Service{
		"frontend": {Host: "containerapp", Uses: []string{"api", "auth"}},
		"api":      {Host: "containerapp", Uses: []string{"db"}},
		"auth":     {Host: "containerapp", Uses: []string{"db"}},
	}

	resources := map[string]service.Resource{
		"db": {Type: "postgres.database"},
	}

	graph, err := service.BuildDependencyGraph(services, resources)
	if err != nil {
		t.Fatalf("Failed to build dependency graph: %v", err)
	}

	// Verify frontend depends on both api and auth
	if len(graph.Edges["frontend"]) != 2 {
		t.Errorf("Expected frontend to have 2 dependencies, got %d", len(graph.Edges["frontend"]))
	}

	// Verify both api and auth depend on db
	if len(graph.Edges["api"]) != 1 || graph.Edges["api"][0] != "db" {
		t.Error("Expected api to depend on db")
	}
	if len(graph.Edges["auth"]) != 1 || graph.Edges["auth"][0] != "db" {
		t.Error("Expected auth to depend on db")
	}
}

func TestBuildDependencyGraph_MissingDependency(t *testing.T) {
	services := map[string]service.Service{
		"web": {Host: "containerapp", Uses: []string{"nonexistent"}},
	}

	// The graph should be built but cycles detected on missing dependency
	graph, err := service.BuildDependencyGraph(services, map[string]service.Resource{})

	// Depending on implementation, either build fails or cycle detection fails
	if err == nil && graph != nil {
		// If build succeeds, cycle detection should fail
		err = service.DetectCycles(graph)
	}

	if err == nil {
		t.Error("Expected error for missing dependency")
	}
}

func TestDetectCycles_ThreeServiceCycle(t *testing.T) {
	services := map[string]service.Service{
		"a": {Host: "containerapp", Uses: []string{"b"}},
		"b": {Host: "containerapp", Uses: []string{"c"}},
		"c": {Host: "containerapp", Uses: []string{"a"}},
	}

	// BuildDependencyGraph may detect the cycle immediately
	_, err := service.BuildDependencyGraph(services, map[string]service.Resource{})
	if err == nil {
		t.Error("Expected cycle detection to find 3-service cycle")
	}
}

func TestDetectCycles_SelfReference(t *testing.T) {
	services := map[string]service.Service{
		"web": {Host: "containerapp", Uses: []string{"web"}},
	}

	// BuildDependencyGraph may detect self-reference immediately
	_, err := service.BuildDependencyGraph(services, map[string]service.Resource{})
	if err == nil {
		t.Error("Expected cycle detection to find self-reference")
	}
}

func TestHasServices_NilYaml(t *testing.T) {
	result := service.HasServices(nil)
	if result {
		t.Error("Expected false for nil yaml")
	}
}

func TestParseAzureYaml_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	// ParseAzureYaml expects file to be named exactly "azure.yaml"
	emptyYamlPath := filepath.Join(tmpDir, "azure.yaml")

	if err := os.WriteFile(emptyYamlPath, []byte(""), 0600); err != nil {
		t.Fatalf("Failed to create empty yaml: %v", err)
	}

	azureYaml, err := service.ParseAzureYaml(tmpDir)
	if err != nil {
		t.Fatalf("Failed to parse empty yaml: %v", err)
	}

	if azureYaml.Name != "" {
		t.Errorf("Expected empty name, got %s", azureYaml.Name)
	}
}

func TestParseAzureYaml_OnlyName(t *testing.T) {
	tmpDir := t.TempDir()
	// ParseAzureYaml expects file to be named exactly "azure.yaml"
	simpleYamlPath := filepath.Join(tmpDir, "azure.yaml")

	content := `name: my-app`
	if err := os.WriteFile(simpleYamlPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create simple yaml: %v", err)
	}

	azureYaml, err := service.ParseAzureYaml(tmpDir)
	if err != nil {
		t.Fatalf("Failed to parse simple yaml: %v", err)
	}

	if azureYaml.Name != "my-app" {
		t.Errorf("Expected name 'my-app', got %s", azureYaml.Name)
	}

	if len(azureYaml.Services) != 0 {
		t.Errorf("Expected no services, got %d", len(azureYaml.Services))
	}
}
