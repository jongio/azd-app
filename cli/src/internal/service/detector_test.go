package service_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

// TestDockerComposeStyleEntrypoint tests Docker Compose style entrypoint + command semantics.
// In Docker Compose: entrypoint is the executable, command is the arguments.
func TestDockerComposeStyleEntrypoint(t *testing.T) {
	tests := []struct {
		name            string
		entrypoint      string
		command         string
		projectFiles    map[string]string
		expectedCommand string
		expectedArgs    []string
	}{
		{
			name:       "FastAPI with uvicorn entrypoint and command args",
			entrypoint: "uvicorn",
			command:    "custom_main:app --reload --host 0.0.0.0 --port 5000",
			projectFiles: map[string]string{
				"requirements.txt": "fastapi\nuvicorn",
				"custom_main.py":   "from fastapi import FastAPI\napp = FastAPI()",
			},
			expectedCommand: "uvicorn",
			expectedArgs:    []string{"custom_main:app", "--reload", "--host", "0.0.0.0", "--port", "5000"},
		},
		{
			name:       "Flask with python entrypoint and flask args",
			entrypoint: "python",
			command:    "-m flask run --host 0.0.0.0 --port 5000",
			projectFiles: map[string]string{
				"requirements.txt": "flask",
				"app.py":           "from flask import Flask\napp = Flask(__name__)",
			},
			expectedCommand: "python",
			expectedArgs:    []string{"-m", "flask", "run", "--host", "0.0.0.0", "--port", "5000"},
		},
		{
			name:       "Node.js with npm entrypoint and dev script",
			entrypoint: "npm",
			command:    "run dev",
			projectFiles: map[string]string{
				"package.json": `{"name":"test","scripts":{"dev":"node server.js"}}`,
			},
			expectedCommand: "npm",
			expectedArgs:    []string{"run", "dev"},
		},
		{
			name:       "Go service with go entrypoint and run command",
			entrypoint: "go",
			command:    "run ./cmd/api",
			projectFiles: map[string]string{
				"go.mod":          "module example.com/app\n\ngo 1.21",
				"cmd/api/main.go": "package main\n\nfunc main() {}",
			},
			expectedCommand: "go",
			expectedArgs:    []string{"run", "./cmd/api"},
		},
		{
			name:       "Entrypoint only (no command)",
			entrypoint: "uvicorn main:app --reload",
			command:    "",
			projectFiles: map[string]string{
				"requirements.txt": "fastapi\nuvicorn",
				"main.py":          "from fastapi import FastAPI\napp = FastAPI()",
			},
			expectedCommand: "uvicorn",
			expectedArgs:    []string{"main:app", "--reload"},
		},
		{
			name:       "Command only (no entrypoint)",
			entrypoint: "",
			command:    "uvicorn main:app --reload --host 0.0.0.0 --port 5000",
			projectFiles: map[string]string{
				"requirements.txt": "fastapi\nuvicorn",
				"main.py":          "from fastapi import FastAPI\napp = FastAPI()",
			},
			expectedCommand: "uvicorn",
			expectedArgs:    []string{"main:app", "--reload", "--host", "0.0.0.0", "--port", "5000"},
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
					t.Fatalf("Failed to create directory for %s: %v", filename, err)
				}
				if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
					t.Fatalf("Failed to create file %s: %v", filename, err)
				}
			}

			// Create azure.yaml with entrypoint and command
			azureYamlContent := `name: test-app
services:
  api:
    project: .
    host: containerapp
    ports:
      - "5000"`

			if tt.entrypoint != "" {
				azureYamlContent += "\n    entrypoint: " + tt.entrypoint
			}
			if tt.command != "" {
				azureYamlContent += "\n    command: " + tt.command
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
			usedPorts := map[int]bool{3000: true, 8000: true, 8080: true}
			runtime, err := service.DetectServiceRuntime("api", svc, usedPorts, tmpDir, "azd")
			if err != nil {
				t.Fatalf("Failed to detect runtime: %v", err)
			}

			// Verify command matches expected
			if runtime.Command != tt.expectedCommand {
				t.Errorf("Expected command %q, got %q", tt.expectedCommand, runtime.Command)
			}

			// Verify args match expected
			if len(runtime.Args) != len(tt.expectedArgs) {
				t.Errorf("Expected args %v, got %v", tt.expectedArgs, runtime.Args)
			} else {
				for i, arg := range tt.expectedArgs {
					if runtime.Args[i] != arg {
						t.Errorf("Expected arg[%d] = %q, got %q", i, arg, runtime.Args[i])
					}
				}
			}
		})
	}
}

// TestAutoDetectWhenNoOverride tests that framework defaults are used when no entrypoint/command specified.
func TestAutoDetectWhenNoOverride(t *testing.T) {
	tests := []struct {
		name         string
		framework    string
		projectFiles map[string]string // filename -> content
		checkCmd     func(runtime *service.ServiceRuntime) error
	}{
		{
			name:      "FastAPI without entrypoint (auto-detect)",
			framework: "FastAPI",
			projectFiles: map[string]string{
				"requirements.txt": "fastapi\nuvicorn",
				"main.py":          "from fastapi import FastAPI\napp = FastAPI()",
			},
			checkCmd: func(runtime *service.ServiceRuntime) error {
				// Should use python (or venv python), not uvicorn directly
				if runtime.Command != "python" && filepath.Base(runtime.Command) != "python" && filepath.Base(runtime.Command) != "python.exe" {
					t.Errorf("Expected python command, got %q", runtime.Command)
				}
				// Check that -m uvicorn is used
				if len(runtime.Args) < 2 || runtime.Args[0] != "-m" || runtime.Args[1] != "uvicorn" {
					t.Errorf("Expected '-m uvicorn' in args, got: %v", runtime.Args)
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
			name:      "Flask without entrypoint (auto-detect)",
			framework: "Flask",
			projectFiles: map[string]string{
				"requirements.txt": "flask",
				"app.py":           "from flask import Flask\napp = Flask(__name__)",
			},
			checkCmd: func(runtime *service.ServiceRuntime) error {
				if runtime.Command != "python" {
					t.Errorf("Expected command 'python', got %q", runtime.Command)
				}
				// Should auto-detect app.py
				if runtime.Env["FLASK_APP"] != "app.py" {
					t.Errorf("Expected FLASK_APP='app.py', got %q", runtime.Env["FLASK_APP"])
				}
				return nil
			},
		},
		{
			name:      "Streamlit without entrypoint (auto-detect)",
			framework: "Streamlit",
			projectFiles: map[string]string{
				"requirements.txt": "streamlit",
				"main.py":          "import streamlit as st",
			},
			checkCmd: func(runtime *service.ServiceRuntime) error {
				// Should use python (or venv python), not streamlit directly
				if runtime.Command != "python" && filepath.Base(runtime.Command) != "python" && filepath.Base(runtime.Command) != "python.exe" {
					t.Errorf("Expected python command, got %q", runtime.Command)
				}
				// Check that -m streamlit is used
				if len(runtime.Args) < 2 || runtime.Args[0] != "-m" || runtime.Args[1] != "streamlit" {
					t.Errorf("Expected '-m streamlit' in args, got: %v", runtime.Args)
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
			name:      "Python without entrypoint (auto-detect)",
			framework: "Python",
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

			// Create azure.yaml without entrypoint (auto-detect test)
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

			// Get the service
			svc, exists := azureYaml.Services["api"]
			if !exists {
				t.Fatal("Service 'api' not found in azure.yaml")
			}

			// Detect runtime with entrypoint (using default "azd" mode)
			// Mark common ports as used to avoid real port conflicts and interactive prompts in tests
			usedPorts := map[int]bool{3000: true, 5000: true, 8000: true, 8080: true}
			runtime, err := service.DetectServiceRuntime("api", svc, usedPorts, tmpDir, "azd")
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

// TestEntrypointValidation tests that azure.yaml parsing correctly handles the entrypoint field.
// With Docker Compose semantics, entrypoint is the executable, not a filename hint.
func TestEntrypointValidation(t *testing.T) {
	tests := []struct {
		name            string
		entrypoint      string
		command         string
		projectFiles    map[string]string
		expectedCommand string
		expectedArgs    []string
	}{
		{
			name:       "Entrypoint as executable (uvicorn)",
			entrypoint: "uvicorn",
			command:    "main:app --reload",
			projectFiles: map[string]string{
				"main.py":          "from fastapi import FastAPI\napp = FastAPI()",
				"requirements.txt": "fastapi\nuvicorn",
			},
			expectedCommand: "uvicorn",
			expectedArgs:    []string{"main:app", "--reload"},
		},
		{
			name:       "Entrypoint as executable (python)",
			entrypoint: "python",
			command:    "main.py",
			projectFiles: map[string]string{
				"main.py":          "print('hello')",
				"requirements.txt": "requests",
			},
			expectedCommand: "python",
			expectedArgs:    []string{"main.py"},
		},
		{
			name:       "Full command in entrypoint (no command field)",
			entrypoint: "uvicorn main:app --reload --host 0.0.0.0",
			command:    "",
			projectFiles: map[string]string{
				"main.py":          "from fastapi import FastAPI\napp = FastAPI()",
				"requirements.txt": "fastapi\nuvicorn",
			},
			expectedCommand: "uvicorn",
			expectedArgs:    []string{"main:app", "--reload", "--host", "0.0.0.0"},
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

			// Create azure.yaml with entrypoint and command
			azureYamlContent := `name: test-app
services:
  api:
    project: .
    language: python
    host: containerapp
    ports:
      - "5000"
    entrypoint: ` + tt.entrypoint
			if tt.command != "" {
				azureYamlContent += "\n    command: " + tt.command
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

			// Verify entrypoint is parsed correctly
			if svc.Entrypoint != tt.entrypoint {
				t.Errorf("Expected entrypoint %q, got %q", tt.entrypoint, svc.Entrypoint)
			}

			// Detect runtime
			usedPorts := map[int]bool{3000: true, 8000: true, 8080: true}
			runtime, err := service.DetectServiceRuntime("api", svc, usedPorts, tmpDir, "azd")
			if err != nil {
				t.Fatalf("Failed to detect runtime: %v", err)
			}

			// Verify command matches expected
			if runtime.Command != tt.expectedCommand {
				t.Errorf("Expected command %q, got %q", tt.expectedCommand, runtime.Command)
			}

			// Verify args
			if len(runtime.Args) != len(tt.expectedArgs) {
				t.Errorf("Expected args %v, got %v", tt.expectedArgs, runtime.Args)
			} else {
				for i, arg := range tt.expectedArgs {
					if runtime.Args[i] != arg {
						t.Errorf("Expected arg[%d] = %q, got %q", i, arg, runtime.Args[i])
					}
				}
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

// TestAutoDetectMissingFile tests that auto-detection properly fails when no default entry file exists.
// With Docker Compose semantics, entrypoint is the executable (not validated as a file),
// but auto-detection still validates that the default entry point files exist.
func TestAutoDetectMissingFile(t *testing.T) {
	tests := []struct {
		name         string
		projectFiles map[string]string
		shouldError  bool
		errorContain string
	}{
		{
			name: "Python missing auto-detected file",
			projectFiles: map[string]string{
				"requirements.txt": "requests",
				// No main.py or app.py
			},
			shouldError:  true,
			errorContain: "python entrypoint file not found: main",
		},
		{
			name: "FastAPI with main.py should not error",
			projectFiles: map[string]string{
				"requirements.txt": "fastapi\nuvicorn",
				"main.py":          "from fastapi import FastAPI\napp = FastAPI()",
			},
			shouldError: false,
		},
		{
			name: "Flask with app.py should not error",
			projectFiles: map[string]string{
				"requirements.txt": "flask",
				"app.py":           "from flask import Flask\napp = Flask(__name__)",
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

			// Create azure.yaml without entrypoint/command (auto-detect)
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

			// Detect runtime - this should validate entrypoint (using default "azd" mode)
			usedPorts := map[int]bool{3000: true, 5000: true, 8000: true, 8080: true}
			_, err = service.DetectServiceRuntime("api", svc, usedPorts, tmpDir, "azd")

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

func TestGoServiceDetection(t *testing.T) {
	tests := []struct {
		name            string
		projectFiles    map[string]string
		entrypoint      string
		expectedCommand string
		expectedArgs    []string
		expectError     bool
	}{
		{
			name: "Go service with go.mod (auto-detect)",
			projectFiles: map[string]string{
				"go.mod":  "module example.com/app\n\ngo 1.21",
				"main.go": "package main\n\nfunc main() {}",
			},
			entrypoint:      "",
			expectedCommand: "go",
			expectedArgs:    []string{"run", "."},
			expectError:     false,
		},
		{
			name: "Go service with custom entrypoint",
			projectFiles: map[string]string{
				"go.mod":          "module example.com/app\n\ngo 1.21",
				"main.go":         "package main\n\nfunc main() {}",
				"cmd/api/main.go": "package main\n\nfunc main() {}",
			},
			entrypoint:      "go run ./cmd/api",
			expectedCommand: "go",
			expectedArgs:    []string{"run", "./cmd/api"},
			expectError:     false,
		},
		{
			name: "Go service with worker entrypoint",
			projectFiles: map[string]string{
				"go.mod":             "module example.com/app\n\ngo 1.21",
				"main.go":            "package main\n\nfunc main() {}",
				"cmd/worker/main.go": "package main\n\nfunc main() {}",
			},
			entrypoint:      "go run ./cmd/worker",
			expectedCommand: "go",
			expectedArgs:    []string{"run", "./cmd/worker"},
			expectError:     false,
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
					t.Fatalf("Failed to create directory for %s: %v", filename, err)
				}
				if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
					t.Fatalf("Failed to create file %s: %v", filename, err)
				}
			}

			// Create azure.yaml
			azureYamlContent := `name: test-go-app
services:
  api:
    project: .
    language: go
    ports:
      - "8080"`
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
			usedPorts := map[int]bool{3000: true, 5000: true, 8000: true}
			runtime, err := service.DetectServiceRuntime("api", svc, usedPorts, tmpDir, "azd")

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify runtime detection
			if runtime.Language != "Go" {
				t.Errorf("Expected language 'Go', got %q", runtime.Language)
			}
			if runtime.Framework != "Go" {
				t.Errorf("Expected framework 'Go', got %q", runtime.Framework)
			}
			if runtime.PackageManager != "go" {
				t.Errorf("Expected package manager 'go', got %q", runtime.PackageManager)
			}
			if runtime.Command != tt.expectedCommand {
				t.Errorf("Expected command %q, got %q", tt.expectedCommand, runtime.Command)
			}
			if len(runtime.Args) != len(tt.expectedArgs) {
				t.Errorf("Expected args %v, got %v", tt.expectedArgs, runtime.Args)
			} else {
				for i, arg := range tt.expectedArgs {
					if runtime.Args[i] != arg {
						t.Errorf("Expected arg[%d] = %q, got %q", i, arg, runtime.Args[i])
					}
				}
			}
		})
	}
}

func TestGoWorkerServiceWithProcessHealthcheck(t *testing.T) {
	// Create temporary project directory
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := "module example.com/app\n\ngo 1.21"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0600); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create main.go
	mainGo := "package main\n\nfunc main() {}"
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0600); err != nil {
		t.Fatalf("Failed to create main.go: %v", err)
	}

	// Create cmd/worker/main.go
	workerDir := filepath.Join(tmpDir, "cmd", "worker")
	if err := os.MkdirAll(workerDir, 0750); err != nil {
		t.Fatalf("Failed to create worker directory: %v", err)
	}
	workerGo := "package main\n\nfunc main() {}"
	if err := os.WriteFile(filepath.Join(workerDir, "main.go"), []byte(workerGo), 0600); err != nil {
		t.Fatalf("Failed to create worker/main.go: %v", err)
	}

	// Create azure.yaml with process-based healthcheck for worker
	azureYamlContent := `name: test-go-app
services:
  api:
    project: .
    language: go
    ports:
      - "8080"
    healthcheck:
      type: http
      path: /health
  worker:
    project: .
    language: go
    entrypoint: go run ./cmd/worker
    healthcheck:
      type: process`

	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	// Parse azure.yaml
	azureYaml, err := service.ParseAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}

	// Test API service
	t.Run("API service with HTTP healthcheck", func(t *testing.T) {
		svc := azureYaml.Services["api"]
		usedPorts := map[int]bool{}
		runtime, err := service.DetectServiceRuntime("api", svc, usedPorts, tmpDir, "azd")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if runtime.HealthCheck.Type != "http" {
			t.Errorf("Expected healthcheck type 'http', got %q", runtime.HealthCheck.Type)
		}
		// Health check path is configured in azure.yaml but may be overridden by framework defaults
		// The important thing is that it's an HTTP healthcheck
		if runtime.Port == 0 {
			t.Error("Expected API service to have a port assigned")
		}
	})

	// Test worker service
	t.Run("Worker service with process healthcheck", func(t *testing.T) {
		svc := azureYaml.Services["worker"]
		usedPorts := map[int]bool{}
		runtime, err := service.DetectServiceRuntime("worker", svc, usedPorts, tmpDir, "azd")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if runtime.HealthCheck.Type != "process" {
			t.Errorf("Expected healthcheck type 'process', got %q", runtime.HealthCheck.Type)
		}
		if runtime.Port != 0 {
			t.Errorf("Expected worker service to have no port (0), got %d", runtime.Port)
		}
		if runtime.Command != "go" {
			t.Errorf("Expected command 'go', got %q", runtime.Command)
		}
		if len(runtime.Args) < 2 || runtime.Args[0] != "run" || runtime.Args[1] != "./cmd/worker" {
			t.Errorf("Expected args ['run', './cmd/worker'], got %v", runtime.Args)
		}
	})
}

func TestLanguageNormalization(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"go", "Go"},
		{"golang", "Go"},
		{"Go", "Go"},
		{"js", "JavaScript"},
		{"javascript", "JavaScript"},
		{"ts", "TypeScript"},
		{"typescript", "TypeScript"},
		{"py", "Python"},
		{"python", "Python"},
		{"csharp", ".NET"},
		{"dotnet", ".NET"},
		{"cs", ".NET"},
		{"c#", ".NET"},
		{"java", "Java"},
		{"rust", "Rust"},
		{"rs", "Rust"},
		{"php", "PHP"},
		// Note: Docker is detected but not fully supported as a runtime framework
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			// Create temporary project directory with appropriate files
			tmpDir := t.TempDir()

			// Create azure.yaml with explicit language
			azureYamlContent := `name: test-app
services:
  api:
    project: .
    language: ` + tt.input + `
    ports:
      - "8080"`

			azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
			if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0600); err != nil {
				t.Fatalf("Failed to create azure.yaml: %v", err)
			}

			// Create language-specific files to support detection
			switch tt.expected {
			case "Go":
				if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n\ngo 1.21"), 0600); err != nil {
					t.Fatalf("Failed to create go.mod: %v", err)
				}
				if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\n\nfunc main() {}"), 0600); err != nil {
					t.Fatalf("Failed to create main.go: %v", err)
				}
			case "JavaScript", "TypeScript":
				if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name":"test","scripts":{"start":"node index.js"}}`), 0600); err != nil {
					t.Fatalf("Failed to create package.json: %v", err)
				}
			case "Python":
				if err := os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte("requests"), 0600); err != nil {
					t.Fatalf("Failed to create requirements.txt: %v", err)
				}
				if err := os.WriteFile(filepath.Join(tmpDir, "main.py"), []byte("print('hello')"), 0600); err != nil {
					t.Fatalf("Failed to create main.py: %v", err)
				}
			case ".NET":
				if err := os.WriteFile(filepath.Join(tmpDir, "test.csproj"), []byte("<Project></Project>"), 0600); err != nil {
					t.Fatalf("Failed to create test.csproj: %v", err)
				}
			case "Java":
				if err := os.WriteFile(filepath.Join(tmpDir, "pom.xml"), []byte("<project></project>"), 0600); err != nil {
					t.Fatalf("Failed to create pom.xml: %v", err)
				}
			case "Rust":
				if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte("[package]\nname = \"test\""), 0600); err != nil {
					t.Fatalf("Failed to create Cargo.toml: %v", err)
				}
			case "PHP":
				if err := os.WriteFile(filepath.Join(tmpDir, "composer.json"), []byte("{}"), 0600); err != nil {
					t.Fatalf("Failed to create composer.json: %v", err)
				}
			}

			// Parse azure.yaml
			azureYaml, err := service.ParseAzureYaml(azureYamlPath)
			if err != nil {
				t.Fatalf("Failed to parse azure.yaml: %v", err)
			}

			svc := azureYaml.Services["api"]
			usedPorts := map[int]bool{}
			runtime, err := service.DetectServiceRuntime("api", svc, usedPorts, tmpDir, "azd")
			if err != nil {
				t.Fatalf("Unexpected error detecting runtime: %v", err)
			}

			if runtime.Language != tt.expected {
				t.Errorf("For input %q: expected language %q, got %q", tt.input, tt.expected, runtime.Language)
			}
		})
	}
}

func TestServiceModeDetection(t *testing.T) {
	tests := []struct {
		name         string
		projectFiles map[string]string
		entrypoint   string
		language     string
		ports        string
		expectedMode string
	}{
		{
			name: "Go with air.toml (watch mode)",
			projectFiles: map[string]string{
				"go.mod":   "module example.com/app\n\ngo 1.21",
				"main.go":  "package main\n\nfunc main() {}",
				"air.toml": "[build]\n  cmd = \"go build\"",
			},
			entrypoint:   "air",
			language:     "go",
			ports:        "",
			expectedMode: "watch",
		},
		{
			name: "Go build command (build mode)",
			projectFiles: map[string]string{
				"go.mod":  "module example.com/app\n\ngo 1.21",
				"main.go": "package main\n\nfunc main() {}",
			},
			entrypoint:   "go build -o app .",
			language:     "go",
			ports:        "",
			expectedMode: "build",
		},
		{
			name: "HTTP service with ports (http type, no mode)",
			projectFiles: map[string]string{
				"go.mod":  "module example.com/app\n\ngo 1.21",
				"main.go": "package main\n\nfunc main() {}",
			},
			entrypoint:   "go run .",
			language:     "go",
			ports:        "8080",
			expectedMode: "", // Empty because it's http type, not process
		},
		{
			name: "TypeScript with nodemon dev script",
			projectFiles: map[string]string{
				"package.json": `{"name":"test","scripts":{"dev":"nodemon src/index.ts","start":"node dist/index.js"},"devDependencies":{"nodemon":"^3.0.0"}}`,
				"src/index.ts": "console.log('hello')",
			},
			entrypoint:   "npm run dev",
			language:     "typescript",
			ports:        "",
			expectedMode: "watch",
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
					t.Fatalf("Failed to create directory for %s: %v", filename, err)
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
    language: ` + tt.language
			if tt.ports != "" {
				azureYamlContent += "\n    ports:\n      - \"" + tt.ports + "\""
			}
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
			usedPorts := map[int]bool{}
			runtime, err := service.DetectServiceRuntime("api", svc, usedPorts, tmpDir, "azd")
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if runtime.Mode != tt.expectedMode {
				t.Errorf("Expected mode %q, got %q", tt.expectedMode, runtime.Mode)
			}
		})
	}
}

func TestServiceTypeConstants(t *testing.T) {
	// Test that constants are defined correctly
	if service.ServiceTypeHTTP != "http" {
		t.Errorf("Expected ServiceTypeHTTP to be 'http', got %q", service.ServiceTypeHTTP)
	}
	if service.ServiceTypeTCP != "tcp" {
		t.Errorf("Expected ServiceTypeTCP to be 'tcp', got %q", service.ServiceTypeTCP)
	}
	if service.ServiceTypeProcess != "process" {
		t.Errorf("Expected ServiceTypeProcess to be 'process', got %q", service.ServiceTypeProcess)
	}
	if service.ServiceModeWatch != "watch" {
		t.Errorf("Expected ServiceModeWatch to be 'watch', got %q", service.ServiceModeWatch)
	}
	if service.ServiceModeBuild != "build" {
		t.Errorf("Expected ServiceModeBuild to be 'build', got %q", service.ServiceModeBuild)
	}
	if service.ServiceModeDaemon != "daemon" {
		t.Errorf("Expected ServiceModeDaemon to be 'daemon', got %q", service.ServiceModeDaemon)
	}
	if service.ServiceModeTask != "task" {
		t.Errorf("Expected ServiceModeTask to be 'task', got %q", service.ServiceModeTask)
	}
}

func TestServiceHelperMethods(t *testing.T) {
	tests := []struct {
		name         string
		svc          service.Service
		expectedType string
		expectedMode string
		isProcess    bool
		isWatch      bool
		isBuild      bool
	}{
		{
			name: "HTTP service with port",
			svc: service.Service{
				Type:  "",
				Mode:  "",
				Ports: []string{"8080"},
			},
			expectedType: "http",
			expectedMode: "",
			isProcess:    false,
			isWatch:      false,
			isBuild:      false,
		},
		{
			name: "Process service with watch mode",
			svc: service.Service{
				Type:  "process",
				Mode:  "watch",
				Ports: []string{},
			},
			expectedType: "process",
			expectedMode: "watch",
			isProcess:    true,
			isWatch:      true,
			isBuild:      false,
		},
		{
			name: "Process service with build mode",
			svc: service.Service{
				Type:  "process",
				Mode:  "build",
				Ports: []string{},
			},
			expectedType: "process",
			expectedMode: "build",
			isProcess:    true,
			isWatch:      false,
			isBuild:      true,
		},
		{
			name: "Service without ports infers process type",
			svc: service.Service{
				Type:  "",
				Mode:  "",
				Ports: []string{},
			},
			expectedType: "process",
			expectedMode: "daemon", // Default mode for process services
			isProcess:    true,
			isWatch:      false,
			isBuild:      false,
		},
		{
			name: "Explicit TCP type",
			svc: service.Service{
				Type:  "tcp",
				Mode:  "",
				Ports: []string{"9000"},
			},
			expectedType: "tcp",
			expectedMode: "",
			isProcess:    false,
			isWatch:      false,
			isBuild:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.svc.GetServiceType(); got != tt.expectedType {
				t.Errorf("GetServiceType() = %q, want %q", got, tt.expectedType)
			}
			if got := tt.svc.GetServiceMode(); got != tt.expectedMode {
				t.Errorf("GetServiceMode() = %q, want %q", got, tt.expectedMode)
			}
			if got := tt.svc.IsProcessService(); got != tt.isProcess {
				t.Errorf("IsProcessService() = %v, want %v", got, tt.isProcess)
			}
			if got := tt.svc.IsWatchMode(); got != tt.isWatch {
				t.Errorf("IsWatchMode() = %v, want %v", got, tt.isWatch)
			}
			if got := tt.svc.IsBuildMode(); got != tt.isBuild {
				t.Errorf("IsBuildMode() = %v, want %v", got, tt.isBuild)
			}
		})
	}
}

// TestContainerServiceDetection tests the IsContainerService and GetContainerImage methods.
func TestContainerServiceDetection(t *testing.T) {
	tests := []struct {
		name          string
		svc           service.Service
		isContainer   bool
		expectedImage string
	}{
		{
			name: "service with direct image is container",
			svc: service.Service{
				Image: "mcr.microsoft.com/azure-storage/azurite",
				Ports: []string{"10000:10000"},
			},
			isContainer:   true,
			expectedImage: "mcr.microsoft.com/azure-storage/azurite",
		},
		{
			name: "service with docker.image is container",
			svc: service.Service{
				Docker: &service.DockerConfig{
					Image: "redis:7-alpine",
				},
				Ports: []string{"6379"},
			},
			isContainer:   true,
			expectedImage: "redis:7-alpine",
		},
		{
			name: "service with project is not container",
			svc: service.Service{
				Project:  "./api",
				Language: "python",
				Ports:    []string{"8000"},
			},
			isContainer:   false,
			expectedImage: "",
		},
		{
			name: "service with both image and project - image takes precedence",
			svc: service.Service{
				Image:   "postgres:16",
				Project: "./db", // This would be ignored
				Ports:   []string{"5432"},
			},
			isContainer:   true,
			expectedImage: "postgres:16",
		},
		{
			name:          "empty service is not container",
			svc:           service.Service{},
			isContainer:   false,
			expectedImage: "",
		},
		{
			name: "service with empty docker config is not container",
			svc: service.Service{
				Docker: &service.DockerConfig{},
			},
			isContainer:   false,
			expectedImage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.svc.IsContainerService(); got != tt.isContainer {
				t.Errorf("IsContainerService() = %v, want %v", got, tt.isContainer)
			}
			if got := tt.svc.GetContainerImage(); got != tt.expectedImage {
				t.Errorf("GetContainerImage() = %q, want %q", got, tt.expectedImage)
			}
		})
	}
}

// TestContainerRuntimeDetection tests that container services get the correct runtime type.
func TestContainerRuntimeDetection(t *testing.T) {
	// Create a temp directory for the test
	tmpDir := t.TempDir()

	// Create azure.yaml for container service
	azureYaml := `name: test-app
services:
  azurite:
    image: mcr.microsoft.com/azure-storage/azurite
    ports:
      - "10000:10000"
      - "10001:10001"
    healthcheck:
      type: tcp
`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(azureYaml), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	// Parse azure.yaml
	parsed, err := service.ParseAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}

	// Get the azurite service
	svc, ok := parsed.Services["azurite"]
	if !ok {
		t.Fatalf("Service 'azurite' not found in azure.yaml")
	}

	// Verify it's detected as a container service
	if !svc.IsContainerService() {
		t.Errorf("Expected azurite to be detected as container service")
	}

	// Detect runtime
	usedPorts := make(map[int]bool)
	runtime, err := service.DetectServiceRuntime("azurite", svc, usedPorts, tmpDir, "azd")
	if err != nil {
		t.Fatalf("DetectServiceRuntime failed: %v", err)
	}

	// Verify runtime type
	if runtime.Type != service.ServiceTypeContainer {
		t.Errorf("Expected Type = %q, got %q", service.ServiceTypeContainer, runtime.Type)
	}

	// Verify image is stored in Command
	if runtime.Command != "mcr.microsoft.com/azure-storage/azurite" {
		t.Errorf("Expected Command (image) = %q, got %q", "mcr.microsoft.com/azure-storage/azurite", runtime.Command)
	}

	// Verify health check type
	if runtime.HealthCheck.Type != "tcp" {
		t.Errorf("Expected HealthCheck.Type = %q, got %q", "tcp", runtime.HealthCheck.Type)
	}

	// Verify port assignment
	if runtime.Port != 10000 {
		t.Errorf("Expected Port = 10000, got %d", runtime.Port)
	}
}

// TestAllWellKnownContainerServices tests parsing all stock container services (azurite, cosmos, redis, postgres).
func TestAllWellKnownContainerServices(t *testing.T) {
	// Create a temp directory for the test
	tmpDir := t.TempDir()

	// Create azure.yaml with all well-known container services
	azureYaml := `name: wellknown-containers-test
services:
  azurite:
    image: mcr.microsoft.com/azure-storage/azurite:latest
    ports:
      - "10000:10000"
      - "10001:10001"
      - "10002:10002"
    healthcheck:
      test: ["CMD", "nc", "-z", "127.0.0.1", "10000"]
      interval: 10s
      timeout: 5s
      retries: 3

  cosmos:
    image: mcr.microsoft.com/cosmosdb/linux/azure-cosmos-emulator:latest
    ports:
      - "8081:8081"
      - "10250:10250"
    environment:
      AZURE_COSMOS_EMULATOR_PARTITION_COUNT: "10"
      AZURE_COSMOS_EMULATOR_ENABLE_DATA_PERSISTENCE: "true"
    healthcheck:
      test: ["CMD", "curl", "-fk", "https://localhost:8081/_explorer/emulator.pem"]
      interval: 30s
      timeout: 10s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 3

  postgres:
    image: postgres:16-alpine
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: testdb
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 3
`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(azureYaml), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	// Parse azure.yaml
	parsed, err := service.ParseAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}

	// Define expected services
	expectedServices := []struct {
		name        string
		image       string
		port        int
		hasEnv      bool
		envVarCheck string
	}{
		{"azurite", "mcr.microsoft.com/azure-storage/azurite:latest", 10000, false, ""},
		{"cosmos", "mcr.microsoft.com/cosmosdb/linux/azure-cosmos-emulator:latest", 8081, true, "AZURE_COSMOS_EMULATOR_PARTITION_COUNT"},
		{"redis", "redis:7-alpine", 6379, false, ""},
		{"postgres", "postgres:16-alpine", 5432, true, "POSTGRES_USER"},
	}

	usedPorts := make(map[int]bool)

	for _, expected := range expectedServices {
		t.Run(expected.name, func(t *testing.T) {
			// Get the service
			svc, ok := parsed.Services[expected.name]
			if !ok {
				t.Fatalf("Service %q not found in azure.yaml", expected.name)
			}

			// Verify it's detected as a container service
			if !svc.IsContainerService() {
				t.Errorf("%s: Expected to be detected as container service", expected.name)
			}

			// Verify image
			if got := svc.GetContainerImage(); got != expected.image {
				t.Errorf("%s: Expected image %q, got %q", expected.name, expected.image, got)
			}

			// Verify environment if expected
			if expected.hasEnv {
				if _, ok := svc.Environment[expected.envVarCheck]; !ok {
					t.Errorf("%s: Expected environment variable %q not found", expected.name, expected.envVarCheck)
				}
			}

			// Verify healthcheck is set
			if svc.Healthcheck == nil {
				t.Errorf("%s: Expected healthcheck to be set", expected.name)
			} else if svc.Healthcheck.Test == nil {
				t.Errorf("%s: Expected healthcheck test command to be set", expected.name)
			}

			// Detect runtime
			runtime, err := service.DetectServiceRuntime(expected.name, svc, usedPorts, tmpDir, "azd")
			if err != nil {
				t.Fatalf("%s: DetectServiceRuntime failed: %v", expected.name, err)
			}

			// Verify runtime type
			if runtime.Type != service.ServiceTypeContainer {
				t.Errorf("%s: Expected Type = %q, got %q", expected.name, service.ServiceTypeContainer, runtime.Type)
			}

			// Verify port
			if runtime.Port != expected.port {
				t.Errorf("%s: Expected Port = %d, got %d", expected.name, expected.port, runtime.Port)
			}

			// Mark port as used
			usedPorts[runtime.Port] = true
		})
	}

	// Verify we have the expected number of services
	if len(parsed.Services) != 4 {
		t.Errorf("Expected 4 services, got %d", len(parsed.Services))
	}
}
