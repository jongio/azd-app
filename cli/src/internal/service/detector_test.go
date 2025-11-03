package service_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

func TestEntrypointOverride(t *testing.T) {
	tests := []struct {
		name         string
		framework    string
		entrypoint   string
		projectFiles map[string]string // filename -> content
		checkCmd     func(runtime *service.ServiceRuntime) error
	}{
		{
			name:       "FastAPI with custom entrypoint",
			framework:  "FastAPI",
			entrypoint: "custom_main",
			projectFiles: map[string]string{
				"requirements.txt": "fastapi\nuvicorn",
				"main.py":          "from fastapi import FastAPI\napp = FastAPI()",
				"custom_main.py":   "from fastapi import FastAPI\napp = FastAPI()",
			},
			checkCmd: func(runtime *service.ServiceRuntime) error {
				if runtime.Command != "uvicorn" {
					t.Errorf("Expected command 'uvicorn', got %q", runtime.Command)
				}
				// Check that custom_main is used in args
				argsStr := strings.Join(runtime.Args, " ")
				if !strings.Contains(argsStr, "custom_main:app") {
					t.Errorf("Expected 'custom_main:app' in args, got: %v", runtime.Args)
				}
				return nil
			},
		},
		{
			name:       "FastAPI without entrypoint (auto-detect)",
			framework:  "FastAPI",
			entrypoint: "",
			projectFiles: map[string]string{
				"requirements.txt": "fastapi\nuvicorn",
				"main.py":          "from fastapi import FastAPI\napp = FastAPI()",
			},
			checkCmd: func(runtime *service.ServiceRuntime) error {
				if runtime.Command != "uvicorn" {
					t.Errorf("Expected command 'uvicorn', got %q", runtime.Command)
				}
				// Should auto-detect main.py
				argsStr := strings.Join(runtime.Args, " ")
				if !strings.Contains(argsStr, "main:app") {
					t.Errorf("Expected 'main:app' in args, got: %v", runtime.Args)
				}
				return nil
			},
		},
		{
			name:       "Flask with custom entrypoint",
			framework:  "Flask",
			entrypoint: "server.py",
			projectFiles: map[string]string{
				"requirements.txt": "flask",
				"main.py":          "from flask import Flask\napp = Flask(__name__)",
				"server.py":        "from flask import Flask\napp = Flask(__name__)",
			},
			checkCmd: func(runtime *service.ServiceRuntime) error {
				if runtime.Command != "python" {
					t.Errorf("Expected command 'python', got %q", runtime.Command)
				}
				// Check FLASK_APP env var is set to custom entrypoint
				if runtime.Env["FLASK_APP"] != "server.py" {
					t.Errorf("Expected FLASK_APP='server.py', got %q", runtime.Env["FLASK_APP"])
				}
				if runtime.Env["FLASK_ENV"] != "development" {
					t.Errorf("Expected FLASK_ENV='development', got %q", runtime.Env["FLASK_ENV"])
				}
				return nil
			},
		},
		{
			name:       "Flask without entrypoint (auto-detect)",
			framework:  "Flask",
			entrypoint: "",
			projectFiles: map[string]string{
				"requirements.txt": "flask",
				"app.py":           "from flask import Flask\napp = Flask(__name__)",
			},
			checkCmd: func(runtime *service.ServiceRuntime) error {
				if runtime.Command != "python" {
					t.Errorf("Expected command 'python', got %q", runtime.Command)
				}
				// Should auto-detect app.py
				if runtime.Env["FLASK_APP"] != "app" {
					t.Errorf("Expected FLASK_APP='app', got %q", runtime.Env["FLASK_APP"])
				}
				return nil
			},
		},
		{
			name:       "Streamlit with custom entrypoint",
			framework:  "Streamlit",
			entrypoint: "dashboard",
			projectFiles: map[string]string{
				"requirements.txt": "streamlit",
				"main.py":          "import streamlit as st",
				"dashboard.py":     "import streamlit as st",
			},
			checkCmd: func(runtime *service.ServiceRuntime) error {
				if runtime.Command != "streamlit" {
					t.Errorf("Expected command 'streamlit', got %q", runtime.Command)
				}
				// Check that dashboard.py is used in args
				argsStr := strings.Join(runtime.Args, " ")
				if !strings.Contains(argsStr, "dashboard.py") {
					t.Errorf("Expected 'dashboard.py' in args, got: %v", runtime.Args)
				}
				return nil
			},
		},
		{
			name:       "Streamlit without entrypoint (auto-detect)",
			framework:  "Streamlit",
			entrypoint: "",
			projectFiles: map[string]string{
				"requirements.txt": "streamlit",
				"main.py":          "import streamlit as st",
			},
			checkCmd: func(runtime *service.ServiceRuntime) error {
				if runtime.Command != "streamlit" {
					t.Errorf("Expected command 'streamlit', got %q", runtime.Command)
				}
				// Should auto-detect main.py
				argsStr := strings.Join(runtime.Args, " ")
				if !strings.Contains(argsStr, "main.py") {
					t.Errorf("Expected 'main.py' in args, got: %v", runtime.Args)
				}
				return nil
			},
		},
		{
			name:       "Python with custom entrypoint",
			framework:  "Python",
			entrypoint: "run_app",
			projectFiles: map[string]string{
				"requirements.txt": "requests",
				"run_app.py":       "# Python app",
			},
			checkCmd: func(runtime *service.ServiceRuntime) error {
				if runtime.Command != "python" {
					t.Errorf("Expected command 'python', got %q", runtime.Command)
				}
				// Check that run_app.py is used in args
				argsStr := strings.Join(runtime.Args, " ")
				if !strings.Contains(argsStr, "run_app.py") {
					t.Errorf("Expected 'run_app.py' in args, got: %v", runtime.Args)
				}
				return nil
			},
		},
		{
			name:       "Python without entrypoint (auto-detect)",
			framework:  "Python",
			entrypoint: "",
			projectFiles: map[string]string{
				"requirements.txt": "requests",
				"main.py":          "# Python app",
			},
			checkCmd: func(runtime *service.ServiceRuntime) error {
				if runtime.Command != "python" {
					t.Errorf("Expected command 'python', got %q", runtime.Command)
				}
				// Should auto-detect main.py
				argsStr := strings.Join(runtime.Args, " ")
				if !strings.Contains(argsStr, "main.py") {
					t.Errorf("Expected 'main.py' in args, got: %v", runtime.Args)
				}
				return nil
			},
		},
		{
			name:       "FastAPI with entrypoint in src directory",
			framework:  "FastAPI",
			entrypoint: "src/main",
			projectFiles: map[string]string{
				"requirements.txt": "fastapi\nuvicorn",
				"main.py":          "from fastapi import FastAPI\napp = FastAPI()",
				"src/main.py":      "from fastapi import FastAPI\napp = FastAPI()",
			},
			checkCmd: func(runtime *service.ServiceRuntime) error {
				if runtime.Command != "uvicorn" {
					t.Errorf("Expected command 'uvicorn', got %q", runtime.Command)
				}
				// Check that src/main:app is used in args
				argsStr := strings.Join(runtime.Args, " ")
				if !strings.Contains(argsStr, "src/main:app") {
					t.Errorf("Expected 'src/main:app' in args, got: %v", runtime.Args)
				}
				return nil
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
				// Create parent directory if needed
				if err := os.MkdirAll(filepath.Dir(filePath), 0750); err != nil {
					t.Fatalf("Failed to create directory for %s: %v", filename, err)
				}
				if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
					t.Fatalf("Failed to create file %s: %v", filename, err)
				}
			}

			// Create azure.yaml with entrypoint configuration
			azureYamlContent := `name: test-app
services:
  api:
    project: .
    language: python
    host: containerapp`

			if tt.entrypoint != "" {
				azureYamlContent += "\n    entrypoint: " + tt.entrypoint
			}

			azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
			if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0600); err != nil {
				t.Fatalf("Failed to create azure.yaml: %v", err)
			}

			// Parse azure.yaml
			azureYaml, err := service.ParseAzureYaml(azureYamlPath)
			if err != nil {
				t.Fatalf("Failed to parse azure.yaml: %v", err)
			}

			// Get the service
			svc, exists := azureYaml.Services["api"]
			if !exists {
				t.Fatal("Service 'api' not found in azure.yaml")
			}

			// Detect runtime with entrypoint (using default "azd" mode)
			runtime, err := service.DetectServiceRuntime("api", svc, map[int]bool{}, tmpDir, "azd")
			if err != nil {
				t.Fatalf("Failed to detect runtime: %v", err)
			}

			// Run the check function
			if err := tt.checkCmd(runtime); err != nil {
				t.Errorf("Command check failed: %v", err)
			}
		})
	}
}

func TestEntrypointValidation(t *testing.T) {
	tests := []struct {
		name         string
		entrypoint   string
		projectFiles map[string]string
	}{
		{
			name:       "Valid entrypoint file exists",
			entrypoint: "custom_main",
			projectFiles: map[string]string{
				"custom_main.py":   "# App",
				"requirements.txt": "requests",
			},
		},
		{
			name:       "Entrypoint with .py extension",
			entrypoint: "custom_main.py",
			projectFiles: map[string]string{
				"custom_main.py":   "# App",
				"requirements.txt": "requests",
			},
		},
		{
			name:       "Entrypoint in subdirectory",
			entrypoint: "src/app",
			projectFiles: map[string]string{
				"src/app.py":       "# App",
				"requirements.txt": "requests",
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
				// Create parent directory if needed
				if err := os.MkdirAll(filepath.Dir(filePath), 0750); err != nil {
					t.Fatalf("Failed to create directory for %s: %v", filename, err)
				}
				if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
					t.Fatalf("Failed to create file %s: %v", filename, err)
				}
			}

			// Create azure.yaml with entrypoint
			azureYamlContent := `name: test-app
services:
  api:
    project: .
    language: python
    host: containerapp
    entrypoint: ` + tt.entrypoint

			azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
			if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0600); err != nil {
				t.Fatalf("Failed to create azure.yaml: %v", err)
			}

			// Parse azure.yaml
			azureYaml, err := service.ParseAzureYaml(azureYamlPath)
			if err != nil {
				t.Fatalf("Failed to parse azure.yaml: %v", err)
			}

			// Get the service
			svc, exists := azureYaml.Services["api"]
			if !exists {
				t.Fatal("Service 'api' not found in azure.yaml")
			}

			// Verify entrypoint is parsed correctly
			if svc.Entrypoint != tt.entrypoint {
				t.Errorf("Expected entrypoint %q, got %q", tt.entrypoint, svc.Entrypoint)
			}

			// Detect runtime - should succeed without error (using default "azd" mode)
			_, err = service.DetectServiceRuntime("api", svc, map[int]bool{}, tmpDir, "azd")
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestEntrypointAzureYamlParsing(t *testing.T) {
	tests := []struct {
		name               string
		yamlContent        string
		serviceName        string
		expectedEntrypoint string
	}{
		{
			name: "Service with entrypoint",
			yamlContent: `name: test-app
services:
  web:
    project: ./src
    language: python
    host: containerapp
    entrypoint: custom_app.py`,
			serviceName:        "web",
			expectedEntrypoint: "custom_app.py",
		},
		{
			name: "Service without entrypoint",
			yamlContent: `name: test-app
services:
  web:
    project: ./src
    language: python
    host: containerapp`,
			serviceName:        "web",
			expectedEntrypoint: "",
		},
		{
			name: "Multiple services with different entrypoints",
			yamlContent: `name: test-app
services:
  api:
    project: ./api
    language: python
    host: containerapp
    entrypoint: server
  worker:
    project: ./worker
    language: python
    host: containerapp
    entrypoint: tasks/worker`,
			serviceName:        "worker",
			expectedEntrypoint: "tasks/worker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary azure.yaml
			tmpDir := t.TempDir()
			azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
			if err := os.WriteFile(azureYamlPath, []byte(tt.yamlContent), 0600); err != nil {
				t.Fatalf("Failed to create azure.yaml: %v", err)
			}

			// Parse azure.yaml
			azureYaml, err := service.ParseAzureYaml(azureYamlPath)
			if err != nil {
				t.Fatalf("Failed to parse azure.yaml: %v", err)
			}

			// Get the service
			svc, exists := azureYaml.Services[tt.serviceName]
			if !exists {
				t.Fatalf("Service %q not found in azure.yaml", tt.serviceName)
			}

			// Verify entrypoint
			if svc.Entrypoint != tt.expectedEntrypoint {
				t.Errorf("Expected entrypoint %q, got %q", tt.expectedEntrypoint, svc.Entrypoint)
			}
		})
	}
}

func TestEntrypointMissingFile(t *testing.T) {
	tests := []struct {
		name         string
		framework    string
		entrypoint   string
		projectFiles map[string]string
		shouldError  bool
		errorContain string
	}{
		{
			name:       "FastAPI missing entrypoint file",
			framework:  "FastAPI",
			entrypoint: "missing_file",
			projectFiles: map[string]string{
				"requirements.txt": "fastapi\nuvicorn",
				"main.py":          "from fastapi import FastAPI\napp = FastAPI()",
			},
			shouldError:  true,
			errorContain: "Python entrypoint file not found: missing_file",
		},
		{
			name:       "Flask missing entrypoint file",
			framework:  "Flask",
			entrypoint: "nonexistent",
			projectFiles: map[string]string{
				"requirements.txt": "flask",
				"app.py":           "from flask import Flask\napp = Flask(__name__)",
			},
			shouldError:  true,
			errorContain: "Python entrypoint file not found: nonexistent",
		},
		{
			name:       "Streamlit missing entrypoint file",
			framework:  "Streamlit",
			entrypoint: "dashboard_missing",
			projectFiles: map[string]string{
				"requirements.txt": "streamlit",
				"main.py":          "import streamlit as st",
			},
			shouldError:  true,
			errorContain: "Python entrypoint file not found: dashboard_missing",
		},
		{
			name:       "Python missing auto-detected file",
			framework:  "Python",
			entrypoint: "", // Will try to auto-detect
			projectFiles: map[string]string{
				"requirements.txt": "requests",
				// No main.py or app.py
			},
			shouldError:  true,
			errorContain: "Python entrypoint file not found: main",
		},
		{
			name:       "FastAPI existing file should not error",
			framework:  "FastAPI",
			entrypoint: "server",
			projectFiles: map[string]string{
				"requirements.txt": "fastapi\nuvicorn",
				"main.py":          "from fastapi import FastAPI\napp = FastAPI()",
				"server.py":        "from fastapi import FastAPI\napp = FastAPI()",
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary project directory
			tmpDir := t.TempDir()

			// Create project files
			for filename, content := range tt.projectFiles {
				filePath := filepath.Join(tmpDir, filename)
				if err := os.MkdirAll(filepath.Dir(filePath), 0750); err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}
				if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
					t.Fatalf("Failed to create file %s: %v", filename, err)
				}
			}

			// Create azure.yaml
			azureYamlContent := `name: test-app
services:
  api:
    project: .
    language: python
    host: containerapp`

			if tt.entrypoint != "" {
				azureYamlContent += "\n    entrypoint: " + tt.entrypoint
			}

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

			// Detect runtime - this should validate entrypoint (using default "azd" mode)
			_, err = service.DetectServiceRuntime("api", svc, map[int]bool{}, tmpDir, "azd")

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error for missing entrypoint file, but got none")
				} else if tt.errorContain != "" && !strings.Contains(err.Error(), tt.errorContain) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorContain, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
