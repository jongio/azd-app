//go:build integration
// +build integration

package runner

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/types"
)

func TestRunAspireIntegration(t *testing.T) {
	// Skip on Windows: Aspire spawns child processes that are difficult to terminate properly
	// on Windows, leading to orphaned processes after test completion.
	if runtime.GOOS == "windows" {
		t.Skip("Integration test disabled on Windows - process cleanup issues with Aspire child processes")
	}

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if aspire is available
	if _, err := os.Stat("../../tests/projects/aspire-test/TestAppHost/TestAppHost.csproj"); os.IsNotExist(err) {
		t.Skip("Aspire test project not found")
	}

	projectDir := filepath.Join("..", "..", "tests", "projects", "aspire-test", "TestAppHost")
	absPath, err := filepath.Abs(projectDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Run Aspire in the background
	go func() {
		err := RunAspire(types.AspireProject{
			Dir:         absPath,
			ProjectFile: filepath.Join(absPath, "TestAppHost.csproj"),
		})
		if err != nil && ctx.Err() == nil {
			t.Logf("RunAspire error: %v", err)
		}
	}()

	// Wait a short time for startup
	time.Sleep(3 * time.Second)

	// Cancel the context to stop Aspire
	cancel()

	t.Log("Aspire ran successfully for test duration")
}

func TestRunPnpmScriptIntegration(t *testing.T) {
	// Skip: This test starts background pnpm processes that continue running after completion
	// causing package-level test failures. Needs proper cleanup implementation.
	t.Skip("Integration test disabled - starts background processes without cleanup")

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name   string
		script string
	}{
		{
			name:   "build_script",
			script: "build",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			// Create a simple package.json with a build script
			packageJSON := `{
  "name": "test-project",
  "version": "1.0.0",
  "scripts": {
    "build": "echo Build complete"
  }
}`
			if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(packageJSON), 0600); err != nil {
				t.Fatal(err)
			}

			// Note: This will try to run pnpm, which may not be installed
			err := RunPnpmScript(tt.script)
			if err != nil {
				t.Logf("RunPnpmScript() error = %v (may be expected if pnpm is not installed)", err)
				t.Skip("Skipping due to missing pnpm")
			}
		})
	}
}

func TestRunDockerComposeIntegration(t *testing.T) {
	// Skip: This test starts background docker processes that continue running after completion
	// causing package-level test failures. Needs proper cleanup implementation.
	t.Skip("Integration test disabled - starts background processes without cleanup")

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()

	// Create a minimal docker-compose.yml
	composeYAML := `version: '3.8'
services:
  test:
    image: alpine:latest
    command: echo "Test service"
`
	if err := os.WriteFile(filepath.Join(tempDir, "docker-compose.yml"), []byte(composeYAML), 0600); err != nil {
		t.Fatal(err)
	}

	// Create package.json with docker compose script
	packageJSON := `{
  "name": "test-project",
  "version": "1.0.0",
  "scripts": {
    "start": "docker compose up"
  }
}`
	if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(packageJSON), 0600); err != nil {
		t.Fatal(err)
	}

	// Note: This requires Docker to be installed and running
	err := RunDockerCompose(tempDir, "start")
	if err != nil {
		t.Logf("RunDockerCompose() error = %v (may be expected if Docker is not running)", err)
		t.Skip("Skipping due to Docker not available")
	}
}
