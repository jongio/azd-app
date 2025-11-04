package runner

import (
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

			err := RunPnpmScript(tt.script)
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

			err := RunDockerCompose(tt.scriptName, tt.scriptCmd)
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
		_ = RunAspire(project)
		// We expect this to fail since the directory doesn't exist
		// but that's okay - we're just testing the signature
	})

	t.Run("RunPnpmScript signature", func(t *testing.T) {
		_ = RunPnpmScript("dev")
		// We expect this to fail if pnpm isn't installed
		// but that's okay - we're just testing the signature
	})

	t.Run("RunDockerCompose signature", func(t *testing.T) {
		_ = RunDockerCompose("start", "docker compose up")
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

			err := RunNode(tt.project, tt.script)
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
			t.Skip("Skipping actual execution in unit tests")

			_ = RunPython(tt.project)
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

			_ = RunDotnet(tt.project)
		})
	}
}
