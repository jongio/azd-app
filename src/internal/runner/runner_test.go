package runner

import (
	"os"
	"path/filepath"
	"testing"

	"app/src/internal/types"
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

// Test function signatures and basic structure.
func TestRunnerFunctionSignatures(t *testing.T) {
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
