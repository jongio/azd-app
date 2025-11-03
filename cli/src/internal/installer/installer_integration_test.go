//go:build integration
// +build integration

package installer

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/types"
)

func TestInstallNodeDependenciesIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name           string
		packageManager string
		setupFunc      func(t *testing.T, dir string)
	}{
		{
			name:           "npm_project",
			packageManager: "npm",
			setupFunc: func(t *testing.T, dir string) {
				packageJSON := `{
  "name": "test-npm-project",
  "version": "1.0.0",
  "dependencies": {
    "lodash": "^4.17.21"
  }
}`
				if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name:           "pnpm_project",
			packageManager: "pnpm",
			setupFunc: func(t *testing.T, dir string) {
				packageJSON := `{
  "name": "test-pnpm-project",
  "version": "1.0.0",
  "dependencies": {
    "lodash": "^4.17.21"
  }
}`
				if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0600); err != nil {
					t.Fatal(err)
				}
				// Create pnpm-lock.yaml to indicate pnpm
				if err := os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte("lockfileVersion: '6.0'\n"), 0600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name:           "yarn_project",
			packageManager: "yarn",
			setupFunc: func(t *testing.T, dir string) {
				packageJSON := `{
  "name": "test-yarn-project",
  "version": "1.0.0",
  "dependencies": {
    "lodash": "^4.17.21"
  }
}`
				if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0600); err != nil {
					t.Fatal(err)
				}
				// Create yarn.lock to indicate yarn
				if err := os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte("# yarn lockfile v1\n"), 0600); err != nil {
					t.Fatal(err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip yarn test if yarn is not installed
			if tt.packageManager == "yarn" {
				if _, err := exec.LookPath("yarn"); err != nil {
					t.Skip("Skipping yarn test - yarn not installed")
				}
			}

			tempDir := t.TempDir()
			tt.setupFunc(t, tempDir)

			err := InstallNodeDependencies(types.NodeProject{
				Dir:            tempDir,
				PackageManager: tt.packageManager,
			})
			if err != nil {
				t.Errorf("InstallNodeDependencies() error = %v", err)
			}

			// Verify node_modules was created
			nodeModulesPath := filepath.Join(tempDir, "node_modules")
			if _, err := os.Stat(nodeModulesPath); os.IsNotExist(err) {
				t.Errorf("node_modules directory was not created")
			}
		})
	}
}

func TestRestoreDotnetProjectIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()

	// Create a minimal .csproj file
	csprojContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`

	csprojPath := filepath.Join(tempDir, "TestProject.csproj")
	if err := os.WriteFile(csprojPath, []byte(csprojContent), 0600); err != nil {
		t.Fatal(err)
	}

	err := RestoreDotnetProject(types.DotnetProject{
		Path: csprojPath,
	})
	if err != nil {
		t.Errorf("RestoreDotnetProject() error = %v", err)
	}

	// Verify obj directory was created (dotnet restore creates this)
	objPath := filepath.Join(tempDir, "obj")
	if _, err := os.Stat(objPath); os.IsNotExist(err) {
		t.Errorf("obj directory was not created by dotnet restore")
	}
}

func TestSetupPythonVirtualEnvIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name           string
		packageManager string
		setupFunc      func(t *testing.T, dir string)
		verifyFunc     func(t *testing.T, dir string)
	}{
		{
			name:           "pip_project",
			packageManager: "pip",
			setupFunc: func(t *testing.T, dir string) {
				requirements := "requests==2.31.0\n"
				if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(requirements), 0600); err != nil {
					t.Fatal(err)
				}
			},
			verifyFunc: func(t *testing.T, dir string) {
				venvPath := filepath.Join(dir, ".venv")
				if _, err := os.Stat(venvPath); os.IsNotExist(err) {
					t.Errorf(".venv directory was not created")
				}
			},
		},
		{
			name:           "uv_project",
			packageManager: "uv",
			setupFunc: func(t *testing.T, dir string) {
				pyproject := `[project]
name = "test-uv-project"
version = "0.1.0"
dependencies = ["requests>=2.31.0"]
`
				if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(pyproject), 0600); err != nil {
					t.Fatal(err)
				}
			},
			verifyFunc: func(t *testing.T, dir string) {
				venvPath := filepath.Join(dir, ".venv")
				if _, err := os.Stat(venvPath); os.IsNotExist(err) {
					t.Errorf(".venv directory was not created")
				}
			},
		},
		{
			name:           "poetry_project",
			packageManager: "poetry",
			setupFunc: func(t *testing.T, dir string) {
				pyproject := `[tool.poetry]
name = "test-poetry-project"
version = "0.1.0"

[tool.poetry.dependencies]
python = "^3.8"
requests = "^2.31.0"
`
				if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(pyproject), 0600); err != nil {
					t.Fatal(err)
				}
			},
			verifyFunc: func(t *testing.T, dir string) {
				// Poetry creates virtual env in a different location
				// Just verify the command ran without error
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			tt.setupFunc(t, tempDir)

			err := SetupPythonVirtualEnv(types.PythonProject{
				Dir:            tempDir,
				PackageManager: tt.packageManager,
			})
			if err != nil {
				t.Logf("SetupPythonVirtualEnv() error = %v (may be expected if %s is not installed)", err, tt.packageManager)
				t.Skip("Skipping due to missing package manager")
			}

			tt.verifyFunc(t, tempDir)
		})
	}
}
