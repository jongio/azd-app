//go:build integration
// +build integration

package service_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

// TestPythonVenvIntegration verifies that Python services use virtual environments correctly.
// This is an integration test that creates real venvs and installs real packages.
// Run with: go test -tags=integration ./src/internal/service/...
func TestPythonVenvIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if Python is available
	pythonCmd := "python"

	// Try to find Python executable
	if _, err := exec.LookPath("python3"); err == nil {
		pythonCmd = "python3"
	} else if _, err := exec.LookPath("python"); err == nil {
		pythonCmd = "python"
		// Verify it's actually Python and not a Windows Store stub
		verifyCmd := exec.Command(pythonCmd, "--version")
		if err := verifyCmd.Run(); err != nil {
			t.Skipf("Python found but not functional: %v", err)
		}
	} else {
		t.Skip("Python not found, skipping integration test")
	}

	tests := []struct {
		name           string
		framework      string
		dependencies   []string
		projectFiles   map[string]string
		expectedModule string // e.g., "uvicorn", "flask", "streamlit"
	}{
		{
			name:         "FastAPI with real venv",
			framework:    "FastAPI",
			dependencies: []string{"fastapi", "uvicorn[standard]"},
			projectFiles: map[string]string{
				"requirements.txt": "fastapi\nuvicorn[standard]",
				"main.py": `from fastapi import FastAPI

app = FastAPI()

@app.get("/")
def read_root():
    return {"message": "Hello World"}
`,
				"azure.yaml": `name: test-fastapi
services:
  api:
    project: .
    language: python
    host: containerapp
`,
			},
			expectedModule: "uvicorn",
		},
		{
			name:         "Flask with real venv",
			framework:    "Flask",
			dependencies: []string{"flask"},
			projectFiles: map[string]string{
				"requirements.txt": "flask",
				"app.py": `from flask import Flask

app = Flask(__name__)

@app.route("/")
def hello():
    return {"message": "Hello World"}
`,
				"azure.yaml": `name: test-flask
services:
  api:
    project: .
    language: python
    host: containerapp
`,
			},
			expectedModule: "flask",
		},
		{
			name:         "Streamlit with real venv",
			framework:    "Streamlit",
			dependencies: []string{"streamlit"},
			projectFiles: map[string]string{
				"requirements.txt": "streamlit",
				"main.py": `import streamlit as st

st.title("Test App")
st.write("Hello World")
`,
				"azure.yaml": `name: test-streamlit
services:
  api:
    project: .
    language: python
    host: containerapp
`,
			},
			expectedModule: "streamlit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			// Create real venv
			t.Logf("Creating virtual environment in %s", tmpDir)
			venvCmd := exec.Command(pythonCmd, "-m", "venv", ".venv")
			venvCmd.Dir = tmpDir
			venvCmd.Stdout = os.Stdout
			venvCmd.Stderr = os.Stderr
			if err := venvCmd.Run(); err != nil {
				t.Skipf("Failed to create venv (Python may not be properly installed): %v", err)
			}

			// Determine pip path
			pipPath := filepath.Join(tmpDir, ".venv", "bin", "pip")
			if runtime.GOOS == "windows" {
				pipPath = filepath.Join(tmpDir, ".venv", "Scripts", "pip.exe")
			}

			// Verify venv was created
			if _, err := os.Stat(pipPath); err != nil {
				t.Fatalf("Venv pip not found at %s: %v", pipPath, err)
			}

			// Install dependencies
			t.Logf("Installing dependencies: %v", tt.dependencies)
			installArgs := append([]string{"install", "--quiet"}, tt.dependencies...)
			installCmd := exec.Command(pipPath, installArgs...)
			installCmd.Dir = tmpDir
			installCmd.Stdout = os.Stdout
			installCmd.Stderr = os.Stderr

			// Set timeout for pip install
			installDone := make(chan error, 1)
			go func() {
				installDone <- installCmd.Run()
			}()

			select {
			case err := <-installDone:
				if err != nil {
					t.Fatalf("Failed to install dependencies: %v", err)
				}
			case <-time.After(2 * time.Minute):
				_ = installCmd.Process.Kill()
				t.Fatal("Timeout waiting for pip install")
			}

			// Parse azure.yaml
			azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
			azureYaml, err := service.ParseAzureYaml(azureYamlPath)
			if err != nil {
				t.Fatalf("Failed to parse azure.yaml: %v", err)
			}

			svc := azureYaml.Services["api"]

			// Detect service runtime
			t.Logf("Detecting service runtime")
			runtimeInfo, err := service.DetectServiceRuntime("api", svc, map[int]bool{}, tmpDir, "azd")
			if err != nil {
				t.Fatalf("Failed to detect runtime: %v", err)
			}

			// Verify framework detection
			if runtimeInfo.Framework != tt.framework {
				t.Errorf("Expected framework %q, got %q", tt.framework, runtimeInfo.Framework)
			}

			// Verify uses venv Python (absolute path)
			if !filepath.IsAbs(runtimeInfo.Command) {
				t.Errorf("Expected absolute path to venv Python, got: %s", runtimeInfo.Command)
			}

			// Verify command contains .venv
			if !strings.Contains(runtimeInfo.Command, ".venv") {
				t.Errorf("Expected venv Python path, got: %s", runtimeInfo.Command)
			}

			// Verify the Python executable exists
			if _, err := os.Stat(runtimeInfo.Command); err != nil {
				t.Errorf("Python executable not found at %s: %v", runtimeInfo.Command, err)
			}

			// Verify uses -m flag for module execution
			if len(runtimeInfo.Args) < 2 || runtimeInfo.Args[0] != "-m" {
				t.Errorf("Expected '-m' flag for module execution, got args: %v", runtimeInfo.Args)
			}

			// Verify correct module name
			if len(runtimeInfo.Args) >= 2 && runtimeInfo.Args[1] != tt.expectedModule {
				t.Errorf("Expected module %q, got %q", tt.expectedModule, runtimeInfo.Args[1])
			}

			// Verify the module is actually installed in venv
			t.Logf("Verifying %s is installed in venv", tt.expectedModule)
			venvPython := runtimeInfo.Command
			checkCmd := exec.Command(venvPython, "-c", "import "+tt.expectedModule)
			checkCmd.Dir = tmpDir
			if err := checkCmd.Run(); err != nil {
				t.Errorf("Module %s not importable from venv Python: %v", tt.expectedModule, err)
			}

			// Log the final command that would be executed
			t.Logf("Command: %s %v", runtimeInfo.Command, runtimeInfo.Args)
			t.Logf("Working directory: %s", runtimeInfo.WorkingDir)
		})
	}
}

// TestPythonVenvFallback verifies that services work without a venv (fallback to system Python).
func TestPythonVenvFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if Python is available and functional
	pythonCmd := "python"
	if _, err := exec.LookPath("python3"); err == nil {
		pythonCmd = "python3"
	}

	verifyCmd := exec.Command(pythonCmd, "--version")
	if err := verifyCmd.Run(); err != nil {
		t.Skipf("Python not available or not functional: %v", err)
	}

	tmpDir := t.TempDir()

	// Create FastAPI project WITHOUT venv
	projectFiles := map[string]string{
		"requirements.txt": "fastapi\nuvicorn",
		"main.py": `from fastapi import FastAPI
app = FastAPI()
`,
		"azure.yaml": `name: test-no-venv
services:
  api:
    project: .
    language: python
    host: containerapp
`,
	}

	for filename, content := range projectFiles {
		filePath := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to create file %s: %v", filename, err)
		}
	}

	// Parse azure.yaml
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	azureYaml, err := service.ParseAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}

	svc := azureYaml.Services["api"]

	// Detect service runtime
	runtimeInfo, err := service.DetectServiceRuntime("api", svc, map[int]bool{}, tmpDir, "azd")
	if err != nil {
		t.Fatalf("Failed to detect runtime: %v", err)
	}

	// Should fall back to system "python" command
	if runtimeInfo.Command != "python" {
		t.Errorf("Expected fallback to system 'python', got: %s", runtimeInfo.Command)
	}

	// Should still use -m flag
	if len(runtimeInfo.Args) < 2 || runtimeInfo.Args[0] != "-m" {
		t.Errorf("Expected '-m' flag even without venv, got args: %v", runtimeInfo.Args)
	}

	t.Logf("Fallback command: %s %v", runtimeInfo.Command, runtimeInfo.Args)
}
