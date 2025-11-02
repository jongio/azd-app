package installer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/types"
)

func TestInstallNodeDependencies(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "installer-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create package.json
	packageJSON := `{
		"name": "test-project",
		"version": "1.0.0",
		"dependencies": {}
	}`

	packagePath := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(packagePath, []byte(packageJSON), 0600); err != nil {
		t.Fatalf("failed to create package.json: %v", err)
	}

	tests := []struct {
		name            string
		project         types.NodeProject
		expectError     bool
		skipRealInstall bool
	}{
		{
			name: "npm project",
			project: types.NodeProject{
				Dir:            tmpDir,
				PackageManager: "npm",
			},
			expectError:     false,
			skipRealInstall: true, // Skip actual npm install in tests
		},
		{
			name: "pnpm project",
			project: types.NodeProject{
				Dir:            tmpDir,
				PackageManager: "pnpm",
			},
			expectError:     false,
			skipRealInstall: true,
		},
		{
			name: "yarn project",
			project: types.NodeProject{
				Dir:            tmpDir,
				PackageManager: "yarn",
			},
			expectError:     false,
			skipRealInstall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipRealInstall {
				t.Skip("Skipping actual package manager execution in unit tests")
			}

			err := InstallNodeDependencies(tt.project)
			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRestoreDotnetProject(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "installer-test-*")
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

	csprojPath := filepath.Join(tmpDir, "test.csproj")
	if err := os.WriteFile(csprojPath, []byte(csprojContent), 0600); err != nil {
		t.Fatalf("failed to create .csproj: %v", err)
	}

	_ = types.DotnetProject{
		Path: csprojPath,
	}

	// Skip actual dotnet restore in tests
	t.Skip("Skipping actual dotnet restore in unit tests")
}

func TestSetupPythonVirtualEnv(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "installer-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create requirements.txt
	requirementsPath := filepath.Join(tmpDir, "requirements.txt")
	if err := os.WriteFile(requirementsPath, []byte("six==1.16.0\n"), 0600); err != nil {
		t.Fatalf("failed to create requirements.txt: %v", err)
	}

	tests := []struct {
		name            string
		project         types.PythonProject
		setupFiles      map[string]string
		expectError     bool
		skipRealInstall bool
	}{
		{
			name: "pip project",
			project: types.PythonProject{
				Dir:            tmpDir,
				PackageManager: "pip",
			},
			setupFiles:      map[string]string{"requirements.txt": "six==1.16.0\n"},
			expectError:     false,
			skipRealInstall: true,
		},
		{
			name: "poetry project",
			project: types.PythonProject{
				Dir:            tmpDir,
				PackageManager: "poetry",
			},
			setupFiles: map[string]string{
				"pyproject.toml": "[tool.poetry]\nname = \"test\"\nversion = \"0.1.0\"\n\n[tool.poetry.dependencies]\npython = \"^3.8\"",
				"poetry.lock":    "",
			},
			expectError:     false,
			skipRealInstall: true,
		},
		{
			name: "uv project",
			project: types.PythonProject{
				Dir:            tmpDir,
				PackageManager: "uv",
			},
			setupFiles: map[string]string{
				"pyproject.toml": "[project]\nname = \"test\"\nversion = \"0.1.0\"",
				"uv.lock":        "",
			},
			expectError:     false,
			skipRealInstall: true,
		},
		{
			name: "unknown package manager",
			project: types.PythonProject{
				Dir:            tmpDir,
				PackageManager: "unknown",
			},
			setupFiles:      map[string]string{},
			expectError:     true,
			skipRealInstall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh temp dir for this test
			testDir, err := os.MkdirTemp("", "installer-test-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer func() { _ = os.RemoveAll(testDir) }()

			// Create setup files
			for filename, content := range tt.setupFiles {
				path := filepath.Join(testDir, filename)
				if err := os.WriteFile(path, []byte(content), 0600); err != nil {
					t.Fatalf("failed to create %s: %v", filename, err)
				}
			}

			// Update project dir
			tt.project.Dir = testDir

			if tt.skipRealInstall {
				// For unknown package manager, we want to test the error path
				if tt.project.PackageManager == "unknown" {
					err := SetupPythonVirtualEnv(tt.project)
					if err == nil {
						t.Error("expected error for unknown package manager")
					}
					return
				}

				t.Skip("Skipping actual Python environment setup in unit tests")
			}

			err = SetupPythonVirtualEnv(tt.project)
			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// Test that we can detect when a virtual environment already exists.
func TestSetupWithPip_VenvExists(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "installer-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create .venv directory to simulate existing environment
	venvDir := filepath.Join(tmpDir, ".venv")
	if err := os.MkdirAll(venvDir, 0750); err != nil {
		t.Fatalf("failed to create .venv: %v", err)
	}

	// Create requirements.txt
	requirementsPath := filepath.Join(tmpDir, "requirements.txt")
	if err := os.WriteFile(requirementsPath, []byte("six==1.16.0\n"), 0600); err != nil {
		t.Fatalf("failed to create requirements.txt: %v", err)
	}

	// Test with existing venv - should not fail
	// This tests the early return path when venv exists
	t.Skip("Skipping actual Python environment check in unit tests")
}

// Test package manager fallback behavior.
func TestSetupWithUv_FallbackToPip(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "installer-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create requirements.txt for pip fallback
	requirementsPath := filepath.Join(tmpDir, "requirements.txt")
	if err := os.WriteFile(requirementsPath, []byte("six==1.16.0\n"), 0600); err != nil {
		t.Fatalf("failed to create requirements.txt: %v", err)
	}

	// This would test the fallback when uv is not installed
	// In a real test, we'd mock exec.LookPath to return an error
	t.Skip("Skipping fallback tests - would require mocking exec.LookPath")
}

func TestSetupWithPoetry_FallbackToPip(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "installer-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create requirements.txt for pip fallback
	requirementsPath := filepath.Join(tmpDir, "requirements.txt")
	if err := os.WriteFile(requirementsPath, []byte("six==1.16.0\n"), 0600); err != nil {
		t.Fatalf("failed to create requirements.txt: %v", err)
	}

	// This would test the fallback when poetry is not installed
	// In a real test, we'd mock exec.LookPath to return an error
	t.Skip("Skipping fallback tests - would require mocking exec.LookPath")
}
