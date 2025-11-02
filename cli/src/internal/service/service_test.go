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
