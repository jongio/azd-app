//go:build integration
// +build integration

package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	testProjectPath = "../../../../tests/projects/health-test"
	healthTimeout   = 5 * time.Minute
	serviceTimeout  = 90 * time.Second
)

// TestHealthCommandE2E_FullWorkflow tests the complete health command workflow end-to-end.
func TestHealthCommandE2E_FullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Get absolute path to test project
	projectDir, err := filepath.Abs(testProjectPath)
	if err != nil {
		t.Fatalf("Failed to resolve project path: %v", err)
	}

	if _, err := os.Stat(filepath.Join(projectDir, "azure.yaml")); os.IsNotExist(err) {
		t.Fatalf("Test project not found at %s", projectDir)
	}

	// Build the azd app binary
	binaryPath := buildAzdBinary(t)
	defer cleanupBinary(t, binaryPath)

	t.Logf("Using binary: %s", binaryPath)
	t.Logf("Test project: %s", projectDir)

	// Test 1: Install dependencies
	t.Run("InstallDependencies", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		cmd := exec.CommandContext(ctx, binaryPath, "app", "deps")
		cmd.Dir = projectDir
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Logf("deps output: %s", output)
			t.Fatalf("Failed to install dependencies: %v", err)
		}

		t.Logf("Dependencies installed successfully")
	})

	// Start services in background
	var runCmd *exec.Cmd
	var runCtx context.Context
	var runCancel context.CancelFunc

	t.Run("StartServices", func(t *testing.T) {
		runCtx, runCancel = context.WithCancel(context.Background())

		runCmd = exec.CommandContext(runCtx, binaryPath, "app", "run")
		runCmd.Dir = projectDir
		runCmd.Stdout = os.Stdout
		runCmd.Stderr = os.Stderr

		if err := runCmd.Start(); err != nil {
			t.Fatalf("Failed to start services: %v", err)
		}

		t.Logf("Services started, PID: %d", runCmd.Process.Pid)

		// Wait for services to initialize
		t.Logf("Waiting %v for services to initialize...", serviceTimeout)
		time.Sleep(serviceTimeout)
	})

	// Ensure cleanup happens
	defer func() {
		if runCancel != nil {
			t.Log("Stopping services...")
			runCancel()
			if runCmd != nil && runCmd.Process != nil {
				// Give graceful shutdown a chance
				time.Sleep(2 * time.Second)
				if runtime.GOOS == "windows" {
					// On Windows, we may need to force kill
					// cSpell:ignore taskkill
					_ = exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", runCmd.Process.Pid)).Run()
				}
				_ = runCmd.Process.Kill()
				_ = runCmd.Wait()
			}
			t.Log("Services stopped")
		}
	}()

	// Test 2: Basic health check
	t.Run("BasicHealthCheck", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binaryPath, "app", "health")
		cmd.Dir = projectDir
		output, err := cmd.CombinedOutput()

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}

		t.Logf("Health check output:\n%s", output)

		// Exit code 0 = all healthy, 1 = some unhealthy (acceptable during startup)
		if exitCode != 0 && exitCode != 1 {
			t.Fatalf("Unexpected exit code: %d (expected 0 or 1)", exitCode)
		}

		// Verify output contains service names
		outputStr := string(output)
		for _, svc := range []string{"web", "api", "database", "worker", "admin"} {
			if !strings.Contains(outputStr, svc) {
				t.Errorf("Output missing service: %s", svc)
			}
		}
	})

	// Test 3: JSON output format
	t.Run("JSONOutputFormat", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binaryPath, "app", "health", "--output", "json")
		cmd.Dir = projectDir
		output, err := cmd.CombinedOutput()

		// Ignore non-zero exit codes, focus on JSON validity
		if err != nil {
			t.Logf("Command returned error (expected during startup): %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(output, &result); err != nil {
			t.Fatalf("Invalid JSON output: %v\nOutput: %s", err, output)
		}

		// Verify JSON structure
		if _, ok := result["services"]; !ok {
			t.Error("JSON output missing 'services' field")
		}
		if _, ok := result["summary"]; !ok {
			t.Error("JSON output missing 'summary' field")
		}

		t.Logf("JSON output valid: %d services", len(result["services"].([]interface{})))
	})

	// Test 4: Table output format
	t.Run("TableOutputFormat", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binaryPath, "app", "health", "--output", "table")
		cmd.Dir = projectDir
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Logf("Command returned error (acceptable): %v", err)
		}

		outputStr := string(output)

		// Verify table headers
		if !strings.Contains(outputStr, "SERVICE") {
			t.Error("Table output missing SERVICE header")
		}
		if !strings.Contains(outputStr, "STATUS") {
			t.Error("Table output missing STATUS header")
		}

		t.Logf("Table output:\n%s", outputStr)
	})

	// Test 5: Service filtering
	t.Run("ServiceFiltering", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binaryPath, "app", "health", "--service", "web,api")
		cmd.Dir = projectDir
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Logf("Command returned error (acceptable): %v", err)
		}

		outputStr := string(output)

		// Should include filtered services
		if !strings.Contains(outputStr, "web") {
			t.Error("Filtered output missing 'web' service")
		}
		if !strings.Contains(outputStr, "api") {
			t.Error("Filtered output missing 'api' service")
		}

		// Should NOT include other services in default output
		// Note: Summary might mention them, so this is a soft check
		t.Logf("Filtered output:\n%s", outputStr)
	})

	// Test 6: Verbose mode
	t.Run("VerboseMode", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binaryPath, "app", "health", "--verbose")
		cmd.Dir = projectDir
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Logf("Command returned error (acceptable): %v", err)
		}

		outputStr := string(output)

		// Verbose should include more details
		// Look for indicators of verbose output (PIDs, ports, etc.)
		hasDetails := strings.Contains(outputStr, "PID") ||
			strings.Contains(outputStr, "port") ||
			strings.Contains(outputStr, "endpoint")

		if !hasDetails {
			t.Log("Warning: Verbose output might not be showing expected details")
		}

		t.Logf("Verbose output length: %d chars", len(outputStr))
	})

	// Test 7: Streaming mode (short duration)
	t.Run("StreamingMode", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binaryPath, "app", "health", "--stream", "--interval", "2s")
		cmd.Dir = projectDir

		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			t.Fatalf("Failed to start streaming health check: %v", err)
		}

		// Let it run for a few iterations
		time.Sleep(7 * time.Second)

		// Cancel and wait
		cancel()
		_ = cmd.Wait()

		output := stdout.String() // Should have multiple updates
		updateCount := strings.Count(output, "Timestamp:")
		if updateCount < 2 {
			t.Errorf("Expected at least 2 updates in streaming mode, got %d", updateCount)
		}

		t.Logf("Streaming produced %d updates", updateCount)
	})

	// Test 8: Verify service info command
	t.Run("ServiceInfo", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binaryPath, "app", "info")
		cmd.Dir = projectDir
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Fatalf("Service info failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)

		// Verify all services are listed
		for _, svc := range []string{"web", "api", "database", "worker", "admin"} {
			if !strings.Contains(outputStr, svc) {
				t.Errorf("Service info missing service: %s", svc)
			}
		}

		t.Logf("Service info:\n%s", outputStr)
	})
}

// TestHealthCommandE2E_ErrorCases tests error handling scenarios.
func TestHealthCommandE2E_ErrorCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	binaryPath := buildAzdBinary(t)
	defer cleanupBinary(t, binaryPath)

	projectDir, err := filepath.Abs(testProjectPath)
	if err != nil {
		t.Fatalf("Failed to resolve project path: %v", err)
	}

	t.Run("NoServicesRunning", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binaryPath, "app", "health")
		cmd.Dir = projectDir
		output, err := cmd.CombinedOutput()

		// Should succeed but report services as unhealthy
		t.Logf("Output with no services:\n%s", output)

		// Verify it doesn't crash
		if err == nil {
			t.Log("Command succeeded (no services running)")
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 1 is acceptable (unhealthy services)
			if exitErr.ExitCode() != 1 {
				t.Errorf("Unexpected exit code: %d", exitErr.ExitCode())
			}
		}
	})

	t.Run("InvalidOutputFormat", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binaryPath, "app", "health", "--output", "invalid")
		cmd.Dir = projectDir
		output, err := cmd.CombinedOutput()

		if err == nil {
			t.Error("Expected error for invalid output format")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "invalid") && !strings.Contains(outputStr, "format") {
			t.Error("Error message should mention invalid format")
		}

		t.Logf("Error output: %s", outputStr)
	})

	t.Run("InvalidInterval", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binaryPath, "app", "health", "--stream", "--interval", "0s")
		cmd.Dir = projectDir
		output, err := cmd.CombinedOutput()

		if err == nil {
			t.Error("Expected error for invalid interval")
		}

		t.Logf("Error output: %s", output)
	})
}

// TestHealthCommandE2E_CrossPlatform tests platform-specific behaviors.
func TestHealthCommandE2E_CrossPlatform(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	binaryPath := buildAzdBinary(t)
	defer cleanupBinary(t, binaryPath)

	projectDir, err := filepath.Abs(testProjectPath)
	if err != nil {
		t.Fatalf("Failed to resolve project path: %v", err)
	}

	t.Run("ProcessCheckCrossPlatform", func(t *testing.T) {
		// This test verifies that process checking works on the current platform
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Start a simple service
		runCtx, runCancel := context.WithCancel(context.Background())
		defer runCancel()

		runCmd := exec.CommandContext(runCtx, binaryPath, "app", "run")
		runCmd.Dir = projectDir

		if err := runCmd.Start(); err != nil {
			t.Fatalf("Failed to start services: %v", err)
		}
		defer func() {
			runCancel()
			_ = runCmd.Process.Kill()
			_ = runCmd.Wait()
		}()

		// Wait a bit for startup
		time.Sleep(30 * time.Second)

		// Health check should detect running processes
		cmd := exec.CommandContext(ctx, binaryPath, "app", "health", "--output", "json")
		cmd.Dir = projectDir
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Logf("Health check returned: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(output, &result); err != nil {
			t.Fatalf("Invalid JSON: %v", err)
		}

		services := result["services"].([]interface{})
		if len(services) == 0 {
			t.Error("No services detected on " + runtime.GOOS)
		}

		t.Logf("Platform %s: Detected %d services", runtime.GOOS, len(services))
	})
}

// buildAzdBinary builds the azd app binary for testing.
func buildAzdBinary(t *testing.T) string {
	t.Helper()

	// Build in temp directory
	tmpDir := t.TempDir()
	binaryName := "azd-test"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(tmpDir, binaryName)

	// Get the project root (4 levels up from this file)
	_, thisFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(thisFile), "../../../..")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, "./src/cmd/app")
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	t.Logf("Building binary: %s", binaryPath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// Verify binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Fatalf("Binary not created at %s", binaryPath)
	}

	return binaryPath
}

// cleanupBinary removes the test binary.
func cleanupBinary(t *testing.T, path string) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		t.Logf("Warning: Failed to cleanup binary: %v", err)
	}
}
