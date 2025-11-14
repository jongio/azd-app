package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/types"
)

func TestRunAspire(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "runner-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a minimal .csproj file
	csprojContent := `<Project Sdk="Microsoft.NET.Sdk">
		<PropertyGroup>
			<OutputType>Exe</OutputType>
			<TargetFramework>net8.0</TargetFramework>
		</PropertyGroup>
	</Project>`

	csprojPath := filepath.Join(tmpDir, "AppHost.csproj")
	if err := os.WriteFile(csprojPath, []byte(csprojContent), 0600); err != nil {
		t.Fatalf("failed to create .csproj: %v", err)
	}

	_ = types.AspireProject{
		Dir:         tmpDir,
		ProjectFile: csprojPath,
	}

	// Skip actual dotnet run in tests - it would try to run indefinitely
	t.Skip("Skipping actual dotnet run in unit tests - would run indefinitely")
}

func TestRunPnpmScript(t *testing.T) {
	tests := []struct {
		name   string
		script string
	}{
		{
			name:   "dev script",
			script: "dev",
		},
		{
			name:   "start script",
			script: "start",
		},
		{
			name:   "build script",
			script: "build",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip actual pnpm execution in tests
			t.Skip("Skipping actual pnpm execution in unit tests")

			err := RunPnpmScript(context.Background(), tt.script)
			if err != nil {
				t.Errorf("RunPnpmScript() error = %v", err)
			}
		})
	}
}

func TestRunDockerCompose(t *testing.T) {
	tests := []struct {
		name       string
		scriptName string
		scriptCmd  string
	}{
		{
			name:       "docker compose up",
			scriptName: "start",
			scriptCmd:  "docker compose up",
		},
		{
			name:       "docker-compose up with flags",
			scriptName: "dev",
			scriptCmd:  "docker-compose up -d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip actual docker compose execution in tests
			t.Skip("Skipping actual docker compose execution in unit tests")

			err := RunDockerCompose(context.Background(), tt.scriptName, tt.scriptCmd)
			if err != nil {
				t.Errorf("RunDockerCompose() error = %v", err)
			}
		})
	}
}

// TestRunnerFunctionSignatures verifies runner function signatures and basic structure.
// Skip: These functions start background processes without cleanup mechanisms.
func TestRunnerFunctionSignatures(t *testing.T) {
	t.Skip("Unit test disabled - functions start background processes without cleanup")

	// This test verifies that the runner functions have the correct signatures
	// and can be called without errors (even if we skip actual execution)

	t.Run("RunAspire signature", func(t *testing.T) {
		project := types.AspireProject{
			Dir:         "/tmp/test",
			ProjectFile: "/tmp/test/app.csproj",
		}

		// Just verify it compiles and has the right signature
		_ = RunAspire(context.Background(), project)
		// We expect this to fail since the directory doesn't exist
		// but that's okay - we're just testing the signature
	})

	t.Run("RunPnpmScript signature", func(t *testing.T) {
		_ = RunPnpmScript(context.Background(), "dev")
		// We expect this to fail if pnpm isn't installed
		// but that's okay - we're just testing the signature
	})

	t.Run("RunDockerCompose signature", func(t *testing.T) {
		_ = RunDockerCompose(context.Background(), "start", "docker compose up")
		// We expect this to fail if pnpm isn't installed
		// but that's okay - we're just testing the signature
	})
}

func TestRunNode(t *testing.T) {
	tests := []struct {
		name           string
		project        types.NodeProject
		script         string
		expectError    bool
		errorSubstring string
	}{
		{
			name: "valid npm project with dev script",
			project: types.NodeProject{
				Dir:            "/tmp/test",
				PackageManager: "npm",
			},
			script:      "dev",
			expectError: false,
		},
		{
			name: "valid pnpm project with start script",
			project: types.NodeProject{
				Dir:            "/tmp/test",
				PackageManager: "pnpm",
			},
			script:      "start",
			expectError: false,
		},
		{
			name: "invalid script with semicolon",
			project: types.NodeProject{
				Dir:            "/tmp/test",
				PackageManager: "npm",
			},
			script:         "dev; rm -rf /",
			expectError:    true,
			errorSubstring: "invalid script name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip actual execution since we're testing validation
			t.Skip("Skipping actual execution in unit tests")

			err := RunNode(context.Background(), tt.project, tt.script)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestFindPythonEntryPoint(t *testing.T) {
	tests := []struct {
		name          string
		setupFiles    []string // Files to create for the test
		expectedEntry string   // Expected entry point file
		expectError   bool
	}{
		{
			name:          "main.py in root",
			setupFiles:    []string{"main.py"},
			expectedEntry: "main.py",
			expectError:   false,
		},
		{
			name:          "app.py in root",
			setupFiles:    []string{"app.py"},
			expectedEntry: "app.py",
			expectError:   false,
		},
		{
			name:          "agent.py in root",
			setupFiles:    []string{"agent.py"},
			expectedEntry: "agent.py",
			expectError:   false,
		},
		{
			name:          "main.py in src/",
			setupFiles:    []string{"src/main.py"},
			expectedEntry: filepath.Join("src", "main.py"),
			expectError:   false,
		},
		{
			name:          "agent.py in src/agent/",
			setupFiles:    []string{"src/agent/agent.py"},
			expectedEntry: filepath.Join("src", "agent", "agent.py"),
			expectError:   false,
		},
		{
			name:          "main.py in src/app/",
			setupFiles:    []string{"src/app/main.py"},
			expectedEntry: filepath.Join("src", "app", "main.py"),
			expectError:   false,
		},
		{
			name:          "__main__.py in root",
			setupFiles:    []string{"__main__.py"},
			expectedEntry: "__main__.py",
			expectError:   false,
		},
		{
			name:          "run.py in app/",
			setupFiles:    []string{"app/run.py"},
			expectedEntry: filepath.Join("app", "run.py"),
			expectError:   false,
		},
		{
			name:          "server.py in src/",
			setupFiles:    []string{"src/server.py"},
			expectedEntry: filepath.Join("src", "server.py"),
			expectError:   false,
		},
		{
			name:          "prefers main.py over others",
			setupFiles:    []string{"main.py", "app.py", "agent.py"},
			expectedEntry: "main.py",
			expectError:   false,
		},
		{
			name:          "prefers root over src/",
			setupFiles:    []string{"main.py", "src/main.py"},
			expectedEntry: "main.py",
			expectError:   false,
		},
		{
			name:        "no entry point found",
			setupFiles:  []string{"README.md", "requirements.txt"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "python-entry-test-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer func() { _ = os.RemoveAll(tmpDir) }()

			// Create test files
			for _, file := range tt.setupFiles {
				fullPath := filepath.Join(tmpDir, file)
				dir := filepath.Dir(fullPath)

				// Create directory if needed
				if err := os.MkdirAll(dir, 0750); err != nil {
					t.Fatalf("failed to create directory %s: %v", dir, err)
				}

				// Create file
				if err := os.WriteFile(fullPath, []byte("# Python file"), 0600); err != nil {
					t.Fatalf("failed to create file %s: %v", fullPath, err)
				}
			}

			// Test the function
			entry, err := findPythonEntryPoint(tmpDir)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if entry != tt.expectedEntry {
				t.Errorf("expected entry point %q, got %q", tt.expectedEntry, entry)
			}
		})
	}
}

func TestRunPython(t *testing.T) {
	tests := []struct {
		name    string
		project types.PythonProject
	}{
		{
			name: "uv project",
			project: types.PythonProject{
				Dir:            "/tmp/test",
				PackageManager: "uv",
			},
		},
		{
			name: "poetry project",
			project: types.PythonProject{
				Dir:            "/tmp/test",
				PackageManager: "poetry",
			},
		},
		{
			name: "pip project",
			project: types.PythonProject{
				Dir:            "/tmp/test",
				PackageManager: "pip",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip actual execution since we're testing structure
			t.Skip("Skipping actual Python execution in unit tests")

			_ = RunPython(context.Background(), tt.project)
		})
	}
}

func TestRunDotnet(t *testing.T) {
	tests := []struct {
		name    string
		project types.DotnetProject
	}{
		{
			name: "csproj project",
			project: types.DotnetProject{
				Path: "/tmp/test/App.csproj",
			},
		},
		{
			name: "solution file",
			project: types.DotnetProject{
				Path: "/tmp/test/Solution.sln",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip actual execution since we're testing structure
			t.Skip("Skipping actual execution in unit tests")

			_ = RunDotnet(context.Background(), tt.project)
		})
	}
}

func TestRunAspire_InvalidPath(t *testing.T) {
	project := types.AspireProject{
		Dir:         "../../../invalid/path",
		ProjectFile: "AppHost.csproj",
	}

	err := RunAspire(context.Background(), project)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestRunPnpmScript_InvalidScript(t *testing.T) {
	err := RunPnpmScript(context.Background(), "dev; rm -rf /")
	if err == nil {
		t.Error("expected error for invalid script name")
	}
}

func TestRunDockerCompose_InvalidScript(t *testing.T) {
	err := RunDockerCompose(context.Background(), "start; malicious", "docker compose up")
	if err == nil {
		t.Error("expected error for invalid script name")
	}
}

func TestRunNode_InvalidPath(t *testing.T) {
	project := types.NodeProject{
		Dir:            "../../../invalid/path",
		PackageManager: "npm",
	}

	err := RunNode(context.Background(), project, "dev")
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestRunNode_InvalidScript(t *testing.T) {
	tmpDir := t.TempDir()

	project := types.NodeProject{
		Dir:            tmpDir,
		PackageManager: "npm",
	}

	err := RunNode(context.Background(), project, "dev; rm -rf /")
	if err == nil {
		t.Error("expected error for invalid script name")
	}
}

func TestRunNode_InvalidPackageManager(t *testing.T) {
	tmpDir := t.TempDir()

	project := types.NodeProject{
		Dir:            tmpDir,
		PackageManager: "invalid-pm; rm -rf /",
	}

	err := RunNode(context.Background(), project, "dev")
	if err == nil {
		t.Error("expected error for invalid package manager")
	}
}

func TestRunPython_InvalidPath(t *testing.T) {
	project := types.PythonProject{
		Dir:            "../../../invalid/path",
		PackageManager: "pip",
	}

	err := RunPython(context.Background(), project)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestRunPython_InvalidPackageManager(t *testing.T) {
	tmpDir := t.TempDir()

	// Create main.py
	mainPath := filepath.Join(tmpDir, "main.py")
	if err := os.WriteFile(mainPath, []byte("print('hello')"), 0600); err != nil {
		t.Fatalf("failed to create main.py: %v", err)
	}

	project := types.PythonProject{
		Dir:            tmpDir,
		PackageManager: "invalid-pm; rm -rf /",
	}

	err := RunPython(context.Background(), project)
	if err == nil {
		t.Error("expected error for invalid package manager")
	}
}

func TestRunPython_NoEntryPoint(t *testing.T) {
	tmpDir := t.TempDir()

	// Create requirements.txt but no Python entry point
	reqPath := filepath.Join(tmpDir, "requirements.txt")
	if err := os.WriteFile(reqPath, []byte("requests==2.28.0\n"), 0600); err != nil {
		t.Fatalf("failed to create requirements.txt: %v", err)
	}

	project := types.PythonProject{
		Dir:            tmpDir,
		PackageManager: "pip",
	}

	err := RunPython(context.Background(), project)
	if err == nil {
		t.Error("expected error when no entry point found")
	}
}

func TestRunPython_WithExplicitEntrypoint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	tmpDir := t.TempDir()

	// Create custom entry point
	customPath := filepath.Join(tmpDir, "custom_entry.py")
	if err := os.WriteFile(customPath, []byte("print('hello')"), 0600); err != nil {
		t.Fatalf("failed to create custom_entry.py: %v", err)
	}

	project := types.PythonProject{
		Dir:            tmpDir,
		PackageManager: "pip",
		Entrypoint:     "custom_entry.py",
	}

	// This should not error on validation
	// (it will error on execution if python isn't installed, but that's ok)
	_ = RunPython(context.Background(), project)
}

func TestRunPython_UnsupportedPackageManager(t *testing.T) {
	tmpDir := t.TempDir()

	// Create main.py
	mainPath := filepath.Join(tmpDir, "main.py")
	if err := os.WriteFile(mainPath, []byte("print('hello')"), 0600); err != nil {
		t.Fatalf("failed to create main.py: %v", err)
	}

	project := types.PythonProject{
		Dir:            tmpDir,
		PackageManager: "conda", // Not supported
	}

	// Should fail validation before execution
	err := RunPython(context.Background(), project)
	if err == nil {
		t.Error("expected error for unsupported package manager")
	}
}

func TestRunDotnet_InvalidPath(t *testing.T) {
	project := types.DotnetProject{
		Path: "../../../invalid/path.csproj",
	}

	err := RunDotnet(context.Background(), project)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestFindPythonEntryPoint_Priority(t *testing.T) {
	// Test that main.py is preferred over app.py
	tmpDir := t.TempDir()

	// Create both main.py and app.py
	mainPath := filepath.Join(tmpDir, "main.py")
	appPath := filepath.Join(tmpDir, "app.py")

	if err := os.WriteFile(mainPath, []byte("# main"), 0600); err != nil {
		t.Fatalf("failed to create main.py: %v", err)
	}
	if err := os.WriteFile(appPath, []byte("# app"), 0600); err != nil {
		t.Fatalf("failed to create app.py: %v", err)
	}

	entry, err := findPythonEntryPoint(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if entry != "main.py" {
		t.Errorf("expected main.py to be preferred, got %s", entry)
	}
}

func TestFindPythonEntryPoint_DirectoryPriority(t *testing.T) {
	// Test that root directory is preferred over src/
	tmpDir := t.TempDir()

	// Create app.py in both root and src/
	rootPath := filepath.Join(tmpDir, "app.py")
	srcDir := filepath.Join(tmpDir, "src")
	srcPath := filepath.Join(srcDir, "app.py")

	if err := os.MkdirAll(srcDir, 0750); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	if err := os.WriteFile(rootPath, []byte("# root"), 0600); err != nil {
		t.Fatalf("failed to create root app.py: %v", err)
	}
	if err := os.WriteFile(srcPath, []byte("# src"), 0600); err != nil {
		t.Fatalf("failed to create src app.py: %v", err)
	}

	entry, err := findPythonEntryPoint(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if entry != "app.py" {
		t.Errorf("expected root app.py to be preferred, got %s", entry)
	}
}
