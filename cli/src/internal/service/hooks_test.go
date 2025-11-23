package service

import (
	"os"
	"path/filepath"
	"strings"
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

func TestHooks_GetPrerun(t *testing.T) {
	tests := []struct {
		name    string
		hooks   *Hooks
		wantNil bool
	}{
		{
			name:    "nil hooks",
			hooks:   nil,
			wantNil: true,
		},
		{
			name:    "hooks with nil prerun",
			hooks:   &Hooks{Prerun: nil},
			wantNil: true,
		},
		{
			name: "hooks with prerun",
			hooks: &Hooks{
				Prerun: &Hook{Run: "echo test"},
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.hooks.GetPrerun()
			
			if tt.wantNil {
				if result != nil {
					t.Error("Expected nil prerun hook")
				}
			} else {
				if result == nil {
					t.Error("Expected non-nil prerun hook")
				}
			}
		})
	}
}

func TestHooks_GetPostrun(t *testing.T) {
	tests := []struct {
		name    string
		hooks   *Hooks
		wantNil bool
	}{
		{
			name:    "nil hooks",
			hooks:   nil,
			wantNil: true,
		},
		{
			name:    "hooks with nil postrun",
			hooks:   &Hooks{Postrun: nil},
			wantNil: true,
		},
		{
			name: "hooks with postrun",
			hooks: &Hooks{
				Postrun: &Hook{Run: "echo test"},
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.hooks.GetPostrun()
			
			if tt.wantNil {
				if result != nil {
					t.Error("Expected nil postrun hook")
				}
			} else {
				if result == nil {
					t.Error("Expected non-nil postrun hook")
				}
			}
		})
	}
}

func TestParseAzureYaml_ComplexHookScenarios(t *testing.T) {
	yamlContent := `name: test-app

hooks:
  prerun:
    run: |
      echo "Multi-line script"
      echo "Line 2"
      echo "Line 3"
    shell: bash
    continueOnError: true
  postrun:
    run: echo "Simple postrun"
    windows:
      run: Write-Host "Windows postrun"
      shell: pwsh
      interactive: true

services:
  web:
    language: TypeScript
    project: .
`

	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "azure.yaml")

	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	azureYaml, err := ParseAzureYaml(yamlPath)
	if err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}

	// Verify prerun multi-line script
	if azureYaml.Hooks.Prerun == nil {
		t.Fatal("Expected prerun hook to be non-nil")
	}
	if !strings.Contains(azureYaml.Hooks.Prerun.Run, "Multi-line script") {
		t.Error("Expected multi-line script in prerun hook")
	}
	if !strings.Contains(azureYaml.Hooks.Prerun.Run, "Line 2") {
		t.Error("Expected Line 2 in prerun hook")
	}

	// Verify postrun Windows override with interactive
	if azureYaml.Hooks.Postrun == nil {
		t.Fatal("Expected postrun hook to be non-nil")
	}
	if azureYaml.Hooks.Postrun.Windows == nil {
		t.Fatal("Expected Windows platform hook to be non-nil")
	}
	if azureYaml.Hooks.Postrun.Windows.Interactive == nil {
		t.Fatal("Expected Windows interactive to be non-nil")
	}
	if !*azureYaml.Hooks.Postrun.Windows.Interactive {
		t.Error("Expected Windows interactive=true")
	}
}

func TestParseAzureYaml_HooksWithAllFields(t *testing.T) {
	yamlContent := `name: test-app

hooks:
  prerun:
    run: echo "prerun"
    shell: bash
    continueOnError: true
    interactive: false
    windows:
      run: Write-Host "windows prerun"
      shell: pwsh
      continueOnError: false
      interactive: true
    posix:
      run: echo "posix prerun"
      shell: zsh
      continueOnError: true
      interactive: false
  postrun:
    run: echo "postrun"
    shell: sh

services:
  web:
    language: TypeScript
    project: .
`

	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "azure.yaml")

	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	azureYaml, err := ParseAzureYaml(yamlPath)
	if err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}

	// Verify all fields are parsed correctly
	prerun := azureYaml.Hooks.Prerun
	if prerun.Run != "echo \"prerun\"" {
		t.Errorf("Expected Run='echo \"prerun\"', got: %s", prerun.Run)
	}
	if prerun.Shell != "bash" {
		t.Errorf("Expected Shell='bash', got: %s", prerun.Shell)
	}
	if !prerun.ContinueOnError {
		t.Error("Expected ContinueOnError=true")
	}
	if prerun.Interactive {
		t.Error("Expected Interactive=false")
	}

	// Verify Windows override
	if prerun.Windows.Run != "Write-Host \"windows prerun\"" {
		t.Errorf("Expected Windows Run, got: %s", prerun.Windows.Run)
	}
	if prerun.Windows.Shell != "pwsh" {
		t.Errorf("Expected Windows Shell='pwsh', got: %s", prerun.Windows.Shell)
	}

	// Verify POSIX override
	if prerun.Posix.Run != "echo \"posix prerun\"" {
		t.Errorf("Expected POSIX Run, got: %s", prerun.Posix.Run)
	}
	if prerun.Posix.Shell != "zsh" {
		t.Errorf("Expected POSIX Shell='zsh', got: %s", prerun.Posix.Shell)
	}
}

