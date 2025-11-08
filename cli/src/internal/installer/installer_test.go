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

func TestInstallNodeDependencies_InvalidPath(t *testing.T) {
	project := types.NodeProject{
		Dir:            "../../../invalid/path",
		PackageManager: "npm",
	}

	err := InstallNodeDependencies(project)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestInstallNodeDependencies_InvalidPackageManager(t *testing.T) {
	tmpDir := t.TempDir()

	project := types.NodeProject{
		Dir:            tmpDir,
		PackageManager: "invalid-pm; rm -rf /",
	}

	err := InstallNodeDependencies(project)
	if err == nil {
		t.Error("expected error for invalid package manager")
	}
}

func TestRestoreDotnetProject_InvalidPath(t *testing.T) {
	project := types.DotnetProject{
		Path: "../../../invalid/path.csproj",
	}

	err := RestoreDotnetProject(project)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestSetupPythonVirtualEnv_UnknownPackageManager(t *testing.T) {
	tmpDir := t.TempDir()

	project := types.PythonProject{
		Dir:            tmpDir,
		PackageManager: "unknown-manager",
	}

	err := SetupPythonVirtualEnv(project)
	if err == nil {
		t.Error("expected error for unknown package manager")
	}

	if err != nil && err.Error() != "unknown package manager: unknown-manager" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSetupWithPip_ExistingVenv(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .venv directory to simulate existing environment
	venvDir := filepath.Join(tmpDir, ".venv")
	if err := os.MkdirAll(venvDir, 0750); err != nil {
		t.Fatalf("failed to create .venv: %v", err)
	}

	// Should return nil when venv exists
	err := setupWithPip(tmpDir)
	if err != nil {
		t.Errorf("setupWithPip() with existing venv should not error: %v", err)
	}
}

func TestSetupWithPip_NoRequirementsTxt(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode - requires python")
	}

	tmpDir := t.TempDir()

	// Try to create venv without requirements.txt
	// This will succeed if python is available
	err := setupWithPip(tmpDir)

	// We don't assert success/failure as it depends on python availability
	// Just verify it doesn't panic
	t.Logf("setupWithPip result: %v", err)
}

func TestSetupWithPoetry_EnvExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	tmpDir := t.TempDir()

	// This tests the path where poetry env info succeeds
	// In practice, this requires poetry to be installed
	err := setupWithPoetry(tmpDir)

	// We expect this to either succeed or fallback to pip
	// Just verify it doesn't panic
	t.Logf("setupWithPoetry result: %v", err)
}

func TestSetupWithUv_NoUvInstalled(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	tmpDir := t.TempDir()

	// Create requirements.txt for fallback
	requirementsPath := filepath.Join(tmpDir, "requirements.txt")
	if err := os.WriteFile(requirementsPath, []byte("# empty\n"), 0600); err != nil {
		t.Fatalf("failed to create requirements.txt: %v", err)
	}

	// This will fallback to pip if uv is not installed
	err := setupWithUv(tmpDir)

	// We don't assert success/failure as it depends on tool availability
	// Just verify it doesn't panic
	t.Logf("setupWithUv result: %v", err)
}

// TestIsDependenciesUpToDate tests the isDependenciesUpToDate function
func TestIsDependenciesUpToDate(t *testing.T) {
	tests := []struct {
		name           string
		packageManager string
		setup          func(dir string) error
		expected       bool
	}{
		{
			name:           "npm - no node_modules",
			packageManager: "npm",
			setup: func(dir string) error {
				// Create package-lock.json only
				return os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte("{}"), 0600)
			},
			expected: false,
		},
		{
			name:           "npm - no lock file",
			packageManager: "npm",
			setup: func(dir string) error {
				// Create node_modules only
				return os.MkdirAll(filepath.Join(dir, "node_modules"), 0750)
			},
			expected: false,
		},
		{
			name:           "npm - missing internal lock",
			packageManager: "npm",
			setup: func(dir string) error {
				// Create both lock file and node_modules, but no internal lock
				if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte("{}"), 0600); err != nil {
					return err
				}
				return os.MkdirAll(filepath.Join(dir, "node_modules"), 0750)
			},
			expected: false,
		},
		{
			name:           "npm - up to date",
			packageManager: "npm",
			setup: func(dir string) error {
				// Create lock file
				if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte("{}"), 0600); err != nil {
					return err
				}
				// Create node_modules
				if err := os.MkdirAll(filepath.Join(dir, "node_modules"), 0750); err != nil {
					return err
				}
				// Create internal lock file (newer than lock file)
				return os.WriteFile(filepath.Join(dir, "node_modules", ".package-lock.json"), []byte("{}"), 0600)
			},
			expected: true,
		},
		{
			name:           "pnpm - missing .pnpm directory",
			packageManager: "pnpm",
			setup: func(dir string) error {
				// Create lock file and node_modules, but no .pnpm directory
				if err := os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte(""), 0600); err != nil {
					return err
				}
				return os.MkdirAll(filepath.Join(dir, "node_modules"), 0750)
			},
			expected: false,
		},
		{
			name:           "pnpm - up to date",
			packageManager: "pnpm",
			setup: func(dir string) error {
				// Create lock file
				if err := os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte(""), 0600); err != nil {
					return err
				}
				// Create node_modules
				if err := os.MkdirAll(filepath.Join(dir, "node_modules"), 0750); err != nil {
					return err
				}
				// Create .pnpm directory
				return os.MkdirAll(filepath.Join(dir, "node_modules", ".pnpm"), 0750)
			},
			expected: true,
		},
		{
			name:           "yarn - no lock file",
			packageManager: "yarn",
			setup: func(dir string) error {
				// Create node_modules only
				return os.MkdirAll(filepath.Join(dir, "node_modules"), 0750)
			},
			expected: false,
		},
		{
			name:           "yarn - up to date",
			packageManager: "yarn",
			setup: func(dir string) error {
				// Create lock file
				if err := os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte(""), 0600); err != nil {
					return err
				}
				// Create node_modules
				return os.MkdirAll(filepath.Join(dir, "node_modules"), 0750)
			},
			expected: true,
		},
		{
			name:           "unknown package manager",
			packageManager: "unknown",
			setup:          func(dir string) error { return nil },
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			if err := tt.setup(tmpDir); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			result := isDependenciesUpToDate(tmpDir, tt.packageManager)
			if result != tt.expected {
				t.Errorf("isDependenciesUpToDate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestInstallNodeDependencies_UpToDate tests the early return when dependencies are up-to-date
func TestInstallNodeDependencies_UpToDate(t *testing.T) {
	tmpDir := t.TempDir()

	// Create package.json
	packageJSON := `{"name": "test", "version": "1.0.0"}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0600); err != nil {
		t.Fatalf("failed to create package.json: %v", err)
	}

	// Create npm lock file
	if err := os.WriteFile(filepath.Join(tmpDir, "package-lock.json"), []byte("{}"), 0600); err != nil {
		t.Fatalf("failed to create package-lock.json: %v", err)
	}

	// Create node_modules and internal lock
	if err := os.MkdirAll(filepath.Join(tmpDir, "node_modules"), 0750); err != nil {
		t.Fatalf("failed to create node_modules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "node_modules", ".package-lock.json"), []byte("{}"), 0600); err != nil {
		t.Fatalf("failed to create internal lock: %v", err)
	}

	project := types.NodeProject{
		Dir:            tmpDir,
		PackageManager: "npm",
	}

	// Should return nil without error since dependencies are up-to-date
	err := InstallNodeDependencies(project)
	if err != nil {
		t.Errorf("InstallNodeDependencies() error = %v, want nil", err)
	}
}
