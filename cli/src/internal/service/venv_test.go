package service_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

func TestPythonVenvDetection(t *testing.T) {
	tests := []struct {
		name         string
		framework    string
		createVenv   bool
		venvName     string
		expectVenv   bool
		projectFiles map[string]string
	}{
		{
			name:       "FastAPI with .venv",
			framework:  "FastAPI",
			createVenv: true,
			venvName:   ".venv",
			expectVenv: true,
			projectFiles: map[string]string{
				"requirements.txt": "fastapi\nuvicorn",
				"main.py":          "from fastapi import FastAPI\napp = FastAPI()",
			},
		},
		{
			name:       "FastAPI with venv (alternative)",
			framework:  "FastAPI",
			createVenv: true,
			venvName:   "venv",
			expectVenv: true,
			projectFiles: map[string]string{
				"requirements.txt": "fastapi\nuvicorn",
				"main.py":          "from fastapi import FastAPI\napp = FastAPI()",
			},
		},
		{
			name:       "FastAPI without venv",
			framework:  "FastAPI",
			createVenv: false,
			expectVenv: false,
			projectFiles: map[string]string{
				"requirements.txt": "fastapi\nuvicorn",
				"main.py":          "from fastapi import FastAPI\napp = FastAPI()",
			},
		},
		{
			name:       "Flask with .venv",
			framework:  "Flask",
			createVenv: true,
			venvName:   ".venv",
			expectVenv: true,
			projectFiles: map[string]string{
				"requirements.txt": "flask",
				"app.py":           "from flask import Flask\napp = Flask(__name__)",
			},
		},
		{
			name:       "Streamlit with .venv",
			framework:  "Streamlit",
			createVenv: true,
			venvName:   ".venv",
			expectVenv: true,
			projectFiles: map[string]string{
				"requirements.txt": "streamlit",
				"main.py":          "import streamlit as st",
			},
		},
		{
			name:       "Django with .venv",
			framework:  "Django",
			createVenv: true,
			venvName:   ".venv",
			expectVenv: true,
			projectFiles: map[string]string{
				"requirements.txt": "django",
				"manage.py":        "#!/usr/bin/env python\nimport django",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary project directory
			tmpDir := t.TempDir()

			// Create project files
			for filename, content := range tt.projectFiles {
				filePath := filepath.Join(tmpDir, filename)
				if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
					t.Fatalf("Failed to create file %s: %v", filename, err)
				}
			}

			// Create venv if specified
			if tt.createVenv {
				var venvPythonPath string
				if runtime.GOOS == "windows" {
					venvPythonPath = filepath.Join(tmpDir, tt.venvName, "Scripts", "python.exe")
				} else {
					venvPythonPath = filepath.Join(tmpDir, tt.venvName, "bin", "python")
				}

				// Create venv directory structure
				if err := os.MkdirAll(filepath.Dir(venvPythonPath), 0750); err != nil {
					t.Fatalf("Failed to create venv directory: %v", err)
				}

				// Create a dummy python executable file
				if err := os.WriteFile(venvPythonPath, []byte("#!/usr/bin/env python"), 0755); err != nil {
					t.Fatalf("Failed to create venv python: %v", err)
				}
			}

			// Create azure.yaml
			azureYamlContent := `name: test-app
services:
  api:
    project: .
    language: python
    host: containerapp`
			azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
			if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0600); err != nil {
				t.Fatalf("Failed to create azure.yaml: %v", err)
			}

			// Parse azure.yaml
			azureYaml, err := service.ParseAzureYaml(azureYamlPath)
			if err != nil {
				t.Fatalf("Failed to parse azure.yaml: %v", err)
			}

			svc := azureYaml.Services["api"]

			// Detect runtime
			runtimeInfo, err := service.DetectServiceRuntime("api", svc, map[int]bool{}, tmpDir, "azd")
			if err != nil {
				t.Fatalf("Failed to detect runtime: %v", err)
			}

			// Verify framework
			if runtimeInfo.Framework != tt.framework {
				t.Errorf("Expected framework %q, got %q", tt.framework, runtimeInfo.Framework)
			}

			// Verify command uses venv or system python
			if tt.expectVenv {
				// Should use venv python (absolute path)
				if runtimeInfo.Command == "python" {
					t.Errorf("Expected venv python path, got system python")
				}
				if !filepath.IsAbs(runtimeInfo.Command) {
					t.Errorf("Expected absolute path to venv python, got %q", runtimeInfo.Command)
				}
				// Verify it contains the venv directory
				if tt.venvName != "" && !strings.Contains(runtimeInfo.Command, tt.venvName) {
					t.Errorf("Expected path to contain venv directory %q, got %q", tt.venvName, runtimeInfo.Command)
				}
			} else {
				// Should use system python
				if runtimeInfo.Command != "python" {
					t.Errorf("Expected system python, got %q", runtimeInfo.Command)
				}
			}

			// For frameworks that run as modules, verify -m flag is present
			if tt.framework == "FastAPI" || tt.framework == "Flask" || tt.framework == "Streamlit" {
				if len(runtimeInfo.Args) < 2 || runtimeInfo.Args[0] != "-m" {
					t.Errorf("Expected '-m' flag in args for %s, got: %v", tt.framework, runtimeInfo.Args)
				}
			}
		})
	}
}
