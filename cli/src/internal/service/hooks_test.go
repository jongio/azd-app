package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseAzureYaml_WithHooks(t *testing.T) {
	yamlContent := `name: test-app

hooks:
  prerun:
    run: echo "prerun"
    shell: sh
    continueOnError: false
    interactive: false
  postrun:
    run: echo "postrun"
    shell: bash

services:
  web:
    language: TypeScript
    project: ./frontend
    ports:
      - "3000"
`

	// Create temporary file
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "azure.yaml")
	
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Parse the YAML
	azureYaml, err := ParseAzureYaml(yamlPath)
	if err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}

	// Verify basic properties
	if azureYaml.Name != "test-app" {
		t.Errorf("Expected name='test-app', got: %s", azureYaml.Name)
	}

	// Verify hooks are parsed
	if azureYaml.Hooks == nil {
		t.Fatal("Expected hooks to be non-nil")
	}

	// Verify prerun hook
	if azureYaml.Hooks.Prerun == nil {
		t.Fatal("Expected prerun hook to be non-nil")
	}
	if azureYaml.Hooks.Prerun.Run != "echo \"prerun\"" {
		t.Errorf("Expected prerun run='echo \"prerun\"', got: %s", azureYaml.Hooks.Prerun.Run)
	}
	if azureYaml.Hooks.Prerun.Shell != "sh" {
		t.Errorf("Expected prerun shell='sh', got: %s", azureYaml.Hooks.Prerun.Shell)
	}
	if azureYaml.Hooks.Prerun.ContinueOnError {
		t.Error("Expected prerun continueOnError=false")
	}
	if azureYaml.Hooks.Prerun.Interactive {
		t.Error("Expected prerun interactive=false")
	}

	// Verify postrun hook
	if azureYaml.Hooks.Postrun == nil {
		t.Fatal("Expected postrun hook to be non-nil")
	}
	if azureYaml.Hooks.Postrun.Run != "echo \"postrun\"" {
		t.Errorf("Expected postrun run='echo \"postrun\"', got: %s", azureYaml.Hooks.Postrun.Run)
	}
	if azureYaml.Hooks.Postrun.Shell != "bash" {
		t.Errorf("Expected postrun shell='bash', got: %s", azureYaml.Hooks.Postrun.Shell)
	}

	// Verify services still parsed
	if len(azureYaml.Services) != 1 {
		t.Errorf("Expected 1 service, got: %d", len(azureYaml.Services))
	}
}

func TestParseAzureYaml_WithPlatformSpecificHooks(t *testing.T) {
	yamlContent := `name: test-app

hooks:
  prerun:
    run: echo "default"
    shell: sh
    windows:
      run: echo "windows"
      shell: pwsh
    posix:
      run: echo "posix"
      shell: bash

services:
  web:
    language: TypeScript
    project: .
    ports:
      - "3000"
`

	// Create temporary file
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "azure.yaml")
	
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Parse the YAML
	azureYaml, err := ParseAzureYaml(yamlPath)
	if err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}

	// Verify base hook
	if azureYaml.Hooks == nil || azureYaml.Hooks.Prerun == nil {
		t.Fatal("Expected prerun hook to be non-nil")
	}

	hook := azureYaml.Hooks.Prerun
	if hook.Run != "echo \"default\"" {
		t.Errorf("Expected base run='echo \"default\"', got: %s", hook.Run)
	}

	// Verify Windows override
	if hook.Windows == nil {
		t.Fatal("Expected Windows platform hook to be non-nil")
	}
	if hook.Windows.Run != "echo \"windows\"" {
		t.Errorf("Expected Windows run='echo \"windows\"', got: %s", hook.Windows.Run)
	}
	if hook.Windows.Shell != "pwsh" {
		t.Errorf("Expected Windows shell='pwsh', got: %s", hook.Windows.Shell)
	}

	// Verify POSIX override
	if hook.Posix == nil {
		t.Fatal("Expected POSIX platform hook to be non-nil")
	}
	if hook.Posix.Run != "echo \"posix\"" {
		t.Errorf("Expected POSIX run='echo \"posix\"', got: %s", hook.Posix.Run)
	}
	if hook.Posix.Shell != "bash" {
		t.Errorf("Expected POSIX shell='bash', got: %s", hook.Posix.Shell)
	}
}

func TestParseAzureYaml_WithoutHooks(t *testing.T) {
	yamlContent := `name: test-app

services:
  web:
    language: TypeScript
    project: .
    ports:
      - "3000"
`

	// Create temporary file
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "azure.yaml")
	
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Parse the YAML
	azureYaml, err := ParseAzureYaml(yamlPath)
	if err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}

	// Verify hooks is nil (not specified)
	if azureYaml.Hooks != nil {
		t.Error("Expected hooks to be nil when not specified")
	}
}

func TestParseAzureYaml_WithOnlyPrerunHook(t *testing.T) {
	yamlContent := `name: test-app

hooks:
  prerun:
    run: echo "prerun only"
    shell: sh

services:
  web:
    language: TypeScript
    project: .
`

	// Create temporary file
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "azure.yaml")
	
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Parse the YAML
	azureYaml, err := ParseAzureYaml(yamlPath)
	if err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}

	// Verify prerun exists
	if azureYaml.Hooks == nil || azureYaml.Hooks.Prerun == nil {
		t.Fatal("Expected prerun hook to be non-nil")
	}

	// Verify postrun is nil
	if azureYaml.Hooks.Postrun != nil {
		t.Error("Expected postrun hook to be nil when not specified")
	}
}

func TestParseAzureYaml_HookWithBooleanOverrides(t *testing.T) {
	yamlContent := `name: test-app

hooks:
  prerun:
    run: echo "test"
    shell: sh
    continueOnError: true
    interactive: true
    windows:
      run: echo "windows test"
      continueOnError: false
      interactive: false

services:
  web:
    language: TypeScript
    project: .
`

	// Create temporary file
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "azure.yaml")
	
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Parse the YAML
	azureYaml, err := ParseAzureYaml(yamlPath)
	if err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}

	// Verify base hook
	hook := azureYaml.Hooks.Prerun
	if !hook.ContinueOnError {
		t.Error("Expected base continueOnError=true")
	}
	if !hook.Interactive {
		t.Error("Expected base interactive=true")
	}

	// Verify Windows override can set to false
	if hook.Windows == nil {
		t.Fatal("Expected Windows platform hook to be non-nil")
	}
	if hook.Windows.ContinueOnError == nil {
		t.Fatal("Expected Windows continueOnError to be non-nil")
	}
	if *hook.Windows.ContinueOnError {
		t.Error("Expected Windows continueOnError=false")
	}
	if hook.Windows.Interactive == nil {
		t.Fatal("Expected Windows interactive to be non-nil")
	}
	if *hook.Windows.Interactive {
		t.Error("Expected Windows interactive=false")
	}
}
