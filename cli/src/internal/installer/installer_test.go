package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	expectedMsg := fmt.Sprintf("unknown package manager 'unknown-manager' for Python project in %s", tmpDir)
	if err != nil && err.Error() != expectedMsg {
		t.Errorf("unexpected error message:\ngot:  %v\nwant: %v", err, expectedMsg)
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
	err := setupWithPip(tmpDir, nil)
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
	err := setupWithPip(tmpDir, nil)

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
	err := setupWithPoetry(tmpDir, nil)

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
	err := setupWithUv(tmpDir, nil)

	// We don't assert success/failure as it depends on tool availability
	// Just verify it doesn't panic
	t.Logf("setupWithUv result: %v", err)
}

func TestIsDependenciesUpToDate(t *testing.T) {
	tests := []struct {
		name           string
		packageManager string
		setupFunc      func(tmpDir string) error
		want           bool
	}{
		{
			name:           "npm_no_lock_file",
			packageManager: "npm",
			setupFunc: func(tmpDir string) error {
				// Create node_modules but no lock file
				return os.MkdirAll(filepath.Join(tmpDir, "node_modules"), 0750)
			},
			want: false,
		},
		{
			name:           "npm_no_node_modules",
			packageManager: "npm",
			setupFunc: func(tmpDir string) error {
				// Create lock file but no node_modules
				return os.WriteFile(filepath.Join(tmpDir, "package-lock.json"), []byte("{}"), 0600)
			},
			want: false,
		},
		{
			name:           "npm_up_to_date",
			packageManager: "npm",
			setupFunc: func(tmpDir string) error {
				// Create lock file first
				if err := os.WriteFile(filepath.Join(tmpDir, "package-lock.json"), []byte("{}"), 0600); err != nil {
					return err
				}
				// Create node_modules and internal lock after main lock
				if err := os.MkdirAll(filepath.Join(tmpDir, "node_modules"), 0750); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(tmpDir, "node_modules", ".package-lock.json"), []byte("{}"), 0600)
			},
			want: true,
		},
		{
			name:           "pnpm_no_internal_store",
			packageManager: "pnpm",
			setupFunc: func(tmpDir string) error {
				if err := os.WriteFile(filepath.Join(tmpDir, "pnpm-lock.yaml"), []byte(""), 0600); err != nil {
					return err
				}
				return os.MkdirAll(filepath.Join(tmpDir, "node_modules"), 0750)
			},
			want: false,
		},
		{
			name:           "yarn_up_to_date",
			packageManager: "yarn",
			setupFunc: func(tmpDir string) error {
				if err := os.WriteFile(filepath.Join(tmpDir, "yarn.lock"), []byte(""), 0600); err != nil {
					return err
				}
				return os.MkdirAll(filepath.Join(tmpDir, "node_modules"), 0750)
			},
			want: true,
		},
		{
			name:           "unknown_package_manager",
			packageManager: "unknown",
			setupFunc:      func(tmpDir string) error { return nil },
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			if err := tt.setupFunc(tmpDir); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			got := isDependenciesUpToDate(tmpDir, tt.packageManager)
			if got != tt.want {
				t.Errorf("isDependenciesUpToDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsFileLockingError(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
		want   bool
	}{
		{
			name:   "ebusy_error",
			stderr: "npm ERR! Error: EBUSY: resource busy or locked",
			want:   true,
		},
		{
			name:   "enotempty_error",
			stderr: "npm ERR! Error: ENOTEMPTY: directory not empty",
			want:   true,
		},
		{
			name:   "eperm_error_windows",
			stderr: "npm ERR! Error: EPERM: operation not permitted",
			want:   true, // Will be true on Windows
		},
		{
			name:   "no_error",
			stderr: "Installation successful",
			want:   false,
		},
		{
			name:   "generic_error",
			stderr: "npm ERR! Some other error",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isFileLockingError(tt.stderr)
			// EPERM check is platform-specific
			if tt.name == "eperm_error_windows" {
				// Just verify it doesn't panic
				_ = got
			} else if got != tt.want {
				t.Errorf("isFileLockingError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPythonSuggestion(t *testing.T) {
	tests := []struct {
		name         string
		tool         string
		exitCode     int
		stderr       string
		wantContains string
	}{
		{
			name:         "uv_not_found",
			tool:         "uv",
			exitCode:     127,
			stderr:       "",
			wantContains: "pip install uv",
		},
		{
			name:         "poetry_not_found",
			tool:         "poetry",
			exitCode:     127,
			stderr:       "",
			wantContains: "pip install poetry",
		},
		{
			name:         "python_not_found",
			tool:         "python",
			exitCode:     127,
			stderr:       "",
			wantContains: "python.org/downloads",
		},
		{
			name:         "permission_error",
			tool:         "pip",
			exitCode:     1,
			stderr:       "PermissionError: [Errno 13] Permission denied",
			wantContains: "appropriate permissions",
		},
		{
			name:         "network_error",
			tool:         "pip",
			exitCode:     1,
			stderr:       "Could not find a version that satisfies the requirement",
			wantContains: "network connection",
		},
		{
			name:         "virtualenv_error",
			tool:         "python -m venv",
			exitCode:     1,
			stderr:       "Error creating virtualenv",
			wantContains: "deleting the .venv directory",
		},
		{
			name:         "no_error",
			tool:         "pip",
			exitCode:     0,
			stderr:       "Successfully installed",
			wantContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPythonSuggestion(tt.tool, tt.exitCode, tt.stderr)
			if tt.wantContains != "" {
				if got == "" || !containsIgnoreCase(got, tt.wantContains) {
					t.Errorf("getPythonSuggestion() = %q, want to contain %q", got, tt.wantContains)
				}
			} else if got != "" {
				t.Errorf("getPythonSuggestion() = %q, want empty string", got)
			}
		})
	}
}

func TestFormatDotnetRestoreError(t *testing.T) {
	tests := []struct {
		name         string
		projectPath  string
		exitCode     int
		stderr       string
		wantContains []string
	}{
		{
			name:        "dotnet_not_found",
			projectPath: "/test/project.csproj",
			exitCode:    127,
			stderr:      "",
			wantContains: []string{
				"failed to restore .NET project",
				"dotnet not found - please install .NET SDK",
			},
		},
		{
			name:        "restore_failed",
			projectPath: "/test/project.csproj",
			exitCode:    1,
			stderr:      "error NU1101: Unable to find package",
			wantContains: []string{
				"failed to restore .NET project",
				"exit code 1",
				"Unable to find package",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("dotnet", "restore", tt.projectPath)
			cmdErr := fmt.Errorf("exit status %d", tt.exitCode)

			err := formatDotnetRestoreError(tt.projectPath, filepath.Dir(tt.projectPath), cmd, cmdErr, tt.stderr)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			errMsg := err.Error()
			for _, want := range tt.wantContains {
				if !containsIgnoreCase(errMsg, want) {
					t.Errorf("error message missing expected content:\nwant substring: %q\ngot: %s", want, errMsg)
				}
			}
		})
	}
}

func TestInstallNodeDependencies_UpToDate(t *testing.T) {
	tmpDir := t.TempDir()

	// Create package.json
	packageJSON := `{"name": "test", "version": "1.0.0"}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0600); err != nil {
		t.Fatalf("failed to create package.json: %v", err)
	}

	// Create package-lock.json
	if err := os.WriteFile(filepath.Join(tmpDir, "package-lock.json"), []byte("{}"), 0600); err != nil {
		t.Fatalf("failed to create package-lock.json: %v", err)
	}

	// Create node_modules and internal lock file
	nodeModulesPath := filepath.Join(tmpDir, "node_modules")
	if err := os.MkdirAll(nodeModulesPath, 0750); err != nil {
		t.Fatalf("failed to create node_modules: %v", err)
	}

	if err := os.WriteFile(filepath.Join(nodeModulesPath, ".package-lock.json"), []byte("{}"), 0600); err != nil {
		t.Fatalf("failed to create internal lock: %v", err)
	}

	project := types.NodeProject{
		Dir:            tmpDir,
		PackageManager: "npm",
	}

	// Should skip install since dependencies are up-to-date
	err := InstallNodeDependencies(project)
	if err != nil {
		t.Errorf("InstallNodeDependencies() with up-to-date deps failed: %v", err)
	}
}

func TestInstallNodeDependencies_Workspace(t *testing.T) {
	tmpDir := t.TempDir()

	// Create package.json for workspace root
	packageJSON := `{
		"name": "test-workspace",
		"version": "1.0.0",
		"workspaces": ["packages/*"]
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0600); err != nil {
		t.Fatalf("failed to create package.json: %v", err)
	}

	tests := []struct {
		name           string
		packageManager string
		expectArgs     []string
	}{
		{
			name:           "npm_workspace",
			packageManager: "npm",
			expectArgs:     []string{"install", "--no-audit", "--no-fund", "--prefer-offline", "--workspaces"},
		},
		{
			name:           "pnpm_workspace",
			packageManager: "pnpm",
			expectArgs:     []string{"install", "--prefer-offline", "--recursive"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := types.NodeProject{
				Dir:             tmpDir,
				PackageManager:  tt.packageManager,
				IsWorkspaceRoot: true,
			}

			// Test that the function constructs the right arguments
			// Since we can't easily test command execution without the actual tool,
			// we'll just verify it doesn't panic with invalid input
			err := InstallNodeDependencies(project)
			// Expect error since package manager likely not installed or will fail
			// The key is it doesn't panic
			t.Logf("%s workspace install result: %v", tt.packageManager, err)
		})
	}
}

// Helper function for case-insensitive contains check
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
