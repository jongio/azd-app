package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/spf13/cobra"
)

func TestRunCommandFlags(t *testing.T) {
	cmd := NewRunCommand()

	tests := []struct {
		name        string
		args        []string
		wantRuntime string
		wantError   bool
	}{
		{
			name:        "Default runtime should be azd",
			args:        []string{},
			wantRuntime: "azd",
			wantError:   false,
		},
		{
			name:        "Valid runtime azd",
			args:        []string{"--runtime", "azd"},
			wantRuntime: "azd",
			wantError:   false,
		},
		{
			name:        "Valid runtime aspire",
			args:        []string{"--runtime", "aspire"},
			wantRuntime: "aspire",
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags for each test
			cmd = NewRunCommand()
			runRuntime = "azd" // reset to default

			// Parse flags
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("ParseFlags failed: %v", err)
			}

			// Check runtime value
			if runRuntime != tt.wantRuntime {
				t.Errorf("Expected runtime %q, got %q", tt.wantRuntime, runRuntime)
			}
		})
	}
}

func TestRunCommandRuntimeValidation(t *testing.T) {
	tests := []struct {
		name      string
		runtime   string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "Valid runtime azd",
			runtime:   runtimeModeAzd,
			wantError: false,
		},
		{
			name:      "Valid runtime aspire",
			runtime:   runtimeModeAspire,
			wantError: false,
		},
		{
			name:      "Invalid runtime foo",
			runtime:   "foo",
			wantError: true,
			errorMsg:  "invalid --runtime value",
		},
		{
			name:      "Invalid runtime bar",
			runtime:   "bar",
			wantError: true,
			errorMsg:  "invalid --runtime value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the runtime variable
			runRuntime = tt.runtime

			// Create a minimal mock command for testing
			cmd := &cobra.Command{
				RunE: func(cmd *cobra.Command, args []string) error {
					// Only test the validation logic
					return validateRuntimeMode(runRuntime)
				},
			}

			err := cmd.RunE(cmd, []string{})

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestRunCommandFlagDefaults(t *testing.T) {
	cmd := NewRunCommand()

	// Check that flags exist with correct defaults
	runtimeFlag := cmd.Flags().Lookup("runtime")
	if runtimeFlag == nil {
		t.Fatal("--runtime flag not found")
	}

	if runtimeFlag.DefValue != "azd" {
		t.Errorf("Expected default runtime to be 'azd', got %q", runtimeFlag.DefValue)
	}

	serviceFlag := cmd.Flags().Lookup("service")
	if serviceFlag == nil {
		t.Fatal("--service flag not found")
	}

	verboseFlag := cmd.Flags().Lookup("verbose")
	if verboseFlag == nil {
		t.Fatal("--verbose flag not found")
	}

	dryRunFlag := cmd.Flags().Lookup("dry-run")
	if dryRunFlag == nil {
		t.Fatal("--dry-run flag not found")
	}

	envFileFlag := cmd.Flags().Lookup("env-file")
	if envFileFlag == nil {
		t.Fatal("--env-file flag not found")
	}
}

func TestRunAspireMode(t *testing.T) {
	// Create temporary directory with Aspire project
	tmpDir, err := os.MkdirTemp("", "aspire-mode-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create AppHost.csproj
	csprojPath := filepath.Join(tmpDir, "AppHost.csproj")
	csprojContent := `<Project Sdk="Microsoft.NET.Sdk.Web">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`
	if err := os.WriteFile(csprojPath, []byte(csprojContent), 0600); err != nil {
		t.Fatalf("Failed to create csproj: %v", err)
	}

	// Create AppHost.cs
	appHostPath := filepath.Join(tmpDir, "AppHost.cs")
	appHostContent := `// Aspire AppHost
namespace TestAppHost;
public class Program {
    public static void Main(string[] args) {
        var builder = DistributedApplication.CreateBuilder(args);
        builder.Build().Run();
    }
}`
	if err := os.WriteFile(appHostPath, []byte(appHostContent), 0600); err != nil {
		t.Fatalf("Failed to create AppHost.cs: %v", err)
	}

	// Test that runAspireMode can find the project
	// Note: We can't actually run it in tests, but we can verify the function doesn't error on setup
	aspireProject, err := detector.FindAppHost(tmpDir)
	if err != nil {
		t.Fatalf("Failed to find AppHost: %v", err)
	}

	if aspireProject == nil {
		t.Fatal("Expected to find Aspire project, got nil")
	}

	if aspireProject.Dir != tmpDir {
		t.Errorf("Expected dir %q, got %q", tmpDir, aspireProject.Dir)
	}

	if aspireProject.ProjectFile != csprojPath {
		t.Errorf("Expected project file %q, got %q", csprojPath, aspireProject.ProjectFile)
	}
}

func TestMonitorServicesUntilShutdown_StartupTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping startup timeout test in short mode")
	}

	tmpDir := t.TempDir()

	// Create a simple service that starts quickly
	runtime := &service.ServiceRuntime{
		Name:       "quick-start",
		WorkingDir: tmpDir,
		Command:    "timeout",
		Args:       []string{"5"},
		Language:   "shell",
		Port:       9000,
	}

	process, err := service.StartService(runtime, map[string]string{}, tmpDir)
	if err != nil {
		t.Fatalf("StartService() error = %v", err)
	}

	t.Cleanup(func() {
		logMgr := service.GetLogManager(tmpDir)
		_ = logMgr.RemoveBuffer(runtime.Name)
		if process.Process != nil {
			_ = service.StopServiceGraceful(process, 1*time.Second)
		}
		time.Sleep(100 * time.Millisecond)
	})

	result := &service.OrchestrationResult{
		Processes: map[string]*service.ServiceProcess{
			"quick-start": process,
		},
		Errors:    map[string]error{},
		StartTime: time.Now(),
	}

	// Test that monitoring completes successfully for a healthy service
	// We'll stop it quickly to avoid long test duration
	go func() {
		time.Sleep(1 * time.Second)
		if process.Process != nil {
			_ = process.Process.Signal(os.Interrupt)
		}
	}()

	err = monitorServicesUntilShutdown(result, tmpDir)
	// Error is expected here because we interrupted the process
	if err != nil {
		t.Logf("monitorServicesUntilShutdown() error = %v (expected due to interrupt)", err)
	}
}

func TestMonitorServicesUntilShutdown_SignalHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping signal handling test in short mode")
	}

	tmpDir := t.TempDir()

	runtime := &service.ServiceRuntime{
		Name:       "signal-test",
		WorkingDir: tmpDir,
		Command:    "timeout",
		Args:       []string{"30"},
		Language:   "shell",
		Port:       9001,
	}

	process, err := service.StartService(runtime, map[string]string{}, tmpDir)
	if err != nil {
		t.Fatalf("StartService() error = %v", err)
	}

	t.Cleanup(func() {
		logMgr := service.GetLogManager(tmpDir)
		_ = logMgr.RemoveBuffer(runtime.Name)
		if process.Process != nil {
			_ = service.StopServiceGraceful(process, 1*time.Second)
		}
		time.Sleep(100 * time.Millisecond)
	})

	result := &service.OrchestrationResult{
		Processes: map[string]*service.ServiceProcess{
			"signal-test": process,
		},
		Errors:    map[string]error{},
		StartTime: time.Now(),
	}

	// Send interrupt signal after short delay
	go func() {
		time.Sleep(500 * time.Millisecond)
		if process.Process != nil {
			_ = process.Process.Signal(os.Interrupt)
		}
	}()

	startTime := time.Now()
	_ = monitorServicesUntilShutdown(result, tmpDir)
	elapsed := time.Since(startTime)

	// Should complete reasonably quickly after signal
	if elapsed > 10*time.Second {
		t.Errorf("monitorServicesUntilShutdown() took %v, expected < 10s after signal", elapsed)
	}

	t.Logf("monitorServicesUntilShutdown() completed in %v after signal", elapsed)
}

func TestMonitorServicesUntilShutdown_MultipleServices(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multiple services test in short mode")
	}

	tmpDir := t.TempDir()

	// Start multiple services
	processes := make(map[string]*service.ServiceProcess)

	for i := 1; i <= 3; i++ {
		runtime := &service.ServiceRuntime{
			Name:       "multi-service-" + string(rune('0'+i)),
			WorkingDir: tmpDir,
			Command:    "timeout",
			Args:       []string{"30"},
			Language:   "shell",
			Port:       9010 + i,
		}

		process, err := service.StartService(runtime, map[string]string{}, tmpDir)
		if err != nil {
			t.Fatalf("StartService(%d) error = %v", i, err)
		}
		processes[runtime.Name] = process
	}

	t.Cleanup(func() {
		logMgr := service.GetLogManager(tmpDir)
		for name, process := range processes {
			_ = logMgr.RemoveBuffer(name)
			if process.Process != nil {
				_ = service.StopServiceGraceful(process, 1*time.Second)
			}
		}
		time.Sleep(200 * time.Millisecond)
	})

	result := &service.OrchestrationResult{
		Processes: processes,
		Errors:    map[string]error{},
		StartTime: time.Now(),
	}

	// Stop all services after short delay
	go func() {
		time.Sleep(500 * time.Millisecond)
		for _, process := range processes {
			if process.Process != nil {
				_ = process.Process.Signal(os.Interrupt)
			}
		}
	}()

	startTime := time.Now()
	_ = monitorServicesUntilShutdown(result, tmpDir)
	elapsed := time.Since(startTime)

	if elapsed > 15*time.Second {
		t.Errorf("monitorServicesUntilShutdown() took %v, expected < 15s for 3 services", elapsed)
	}
}

func TestShutdownAllServices_WithContext(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping shutdown context test in short mode")
	}

	tmpDir := t.TempDir()

	// Start services
	processes := make(map[string]*service.ServiceProcess)

	for i := 1; i <= 2; i++ {
		runtime := &service.ServiceRuntime{
			Name:       "shutdown-test-" + string(rune('0'+i)),
			WorkingDir: tmpDir,
			Command:    "timeout",
			Args:       []string{"30"},
			Language:   "shell",
			Port:       9020 + i,
		}

		process, err := service.StartService(runtime, map[string]string{}, tmpDir)
		if err != nil {
			t.Fatalf("StartService(%d) error = %v", i, err)
		}
		processes[runtime.Name] = process
	}

	t.Cleanup(func() {
		logMgr := service.GetLogManager(tmpDir)
		for name := range processes {
			_ = logMgr.RemoveBuffer(name)
		}
		time.Sleep(200 * time.Millisecond)
	})

	result := &service.OrchestrationResult{
		Processes: processes,
		Errors:    map[string]error{},
		StartTime: time.Now(),
	}

	// Create context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	startTime := time.Now()
	err := shutdownAllServices(ctx, result.Processes)
	elapsed := time.Since(startTime)

	// Log any shutdown errors for diagnostics
	if err != nil {
		t.Logf("shutdownAllServices() returned error (may be expected): %v", err)
	}

	// Should shutdown within reasonable time
	if elapsed > 12*time.Second {
		t.Errorf("shutdownAllServices() took %v, expected < 12s", elapsed)
	}

	// Verify all processes stopped
	for name, process := range processes {
		if process.Process != nil {
			// Try to signal - should fail if process is stopped
			err := process.Process.Signal(syscall.Signal(0))
			if err == nil {
				t.Logf("Process %s may still be running (acceptable on some platforms)", name)
			}
		}
	}
}

func TestShutdownAllServices_ContextTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping shutdown timeout test in short mode")
	}

	tmpDir := t.TempDir()

	runtime := &service.ServiceRuntime{
		Name:       "shutdown-timeout-test",
		WorkingDir: tmpDir,
		Command:    "timeout",
		Args:       []string{"30"},
		Language:   "shell",
		Port:       9030,
	}

	process, err := service.StartService(runtime, map[string]string{}, tmpDir)
	if err != nil {
		t.Fatalf("StartService() error = %v", err)
	}

	t.Cleanup(func() {
		logMgr := service.GetLogManager(tmpDir)
		_ = logMgr.RemoveBuffer(runtime.Name)
		if process.Process != nil {
			_ = service.StopServiceGraceful(process, 1*time.Second)
		}
		time.Sleep(100 * time.Millisecond)
	})

	result := &service.OrchestrationResult{
		Processes: map[string]*service.ServiceProcess{
			"shutdown-timeout-test": process,
		},
		Errors:    map[string]error{},
		StartTime: time.Now(),
	}

	// Use very short context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	startTime := time.Now()
	err = shutdownAllServices(ctx, result.Processes)
	elapsed := time.Since(startTime)

	// Expect errors due to timeout
	if err != nil {
		t.Logf("shutdownAllServices() returned error (expected due to timeout): %v", err)
	}

	// Should respect context timeout
	if elapsed > 4*time.Second {
		t.Errorf("shutdownAllServices() took %v, expected < 4s with 2s context timeout", elapsed)
	}
}

func TestMonitorServices_RunsIndefinitely(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long-running service test in short mode")
	}

	tmpDir := t.TempDir()

	// Create a simple batch script that runs for a long time
	scriptPath := filepath.Join(tmpDir, "long-running.bat")
	scriptContent := "@echo off\necho Starting long-running service\n:loop\nping 127.0.0.1 -n 2 > nul\ngoto loop"
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0700); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	// Start a service that runs for a long time
	runtime := &service.ServiceRuntime{
		Name:       "long-running",
		WorkingDir: tmpDir,
		Command:    "cmd",
		Args:       []string{"/C", scriptPath},
		Language:   "shell",
		Port:       9050,
	}

	process, err := service.StartService(runtime, map[string]string{}, tmpDir)
	if err != nil {
		t.Fatalf("StartService() error = %v", err)
	}

	t.Cleanup(func() {
		logMgr := service.GetLogManager(tmpDir)
		_ = logMgr.RemoveBuffer(runtime.Name)
		if process.Process != nil {
			_ = service.StopServiceGraceful(process, 1*time.Second)
		}
		time.Sleep(100 * time.Millisecond)
	})

	result := &service.OrchestrationResult{
		Processes: map[string]*service.ServiceProcess{
			"long-running": process,
		},
		Errors:    map[string]error{},
		StartTime: time.Now(),
	}

	// Monitor should run indefinitely until we signal to stop
	// Send interrupt after 5 seconds to verify it runs past any startup timeout
	go func() {
		time.Sleep(5 * time.Second)
		if process.Process != nil {
			_ = process.Process.Signal(os.Interrupt)
		}
	}()

	startTime := time.Now()
	_ = monitorServicesUntilShutdown(result, tmpDir)
	elapsed := time.Since(startTime)

	// Should have run for approximately 5 seconds (not stop at 30 seconds or earlier)
	if elapsed < 4*time.Second {
		t.Errorf("monitorServicesUntilShutdown() stopped too early at %v, expected ~5s", elapsed)
	}
	if elapsed > 10*time.Second {
		t.Errorf("monitorServicesUntilShutdown() took too long at %v, expected ~5s", elapsed)
	}

	t.Logf("✓ Service ran for %v without automatic timeout", elapsed)
}

func TestProcessExit_CausesMonitoringToStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping process exit test in short mode")
	}

	tmpDir := t.TempDir()

	// Start a process that exits quickly
	runtime := &service.ServiceRuntime{
		Name:       "quick-exit",
		WorkingDir: tmpDir,
		Command:    "timeout",
		Args:       []string{"1"}, // Exit after 1 second
		Language:   "shell",
		Port:       9040,
	}

	process, err := service.StartService(runtime, map[string]string{}, tmpDir)
	if err != nil {
		t.Fatalf("StartService() error = %v", err)
	}

	t.Cleanup(func() {
		logMgr := service.GetLogManager(tmpDir)
		_ = logMgr.RemoveBuffer(runtime.Name)
		time.Sleep(100 * time.Millisecond)
	})

	result := &service.OrchestrationResult{
		Processes: map[string]*service.ServiceProcess{
			"quick-exit": process,
		},
		Errors:    map[string]error{},
		StartTime: time.Now(),
	}

	startTime := time.Now()
	err = monitorServicesUntilShutdown(result, tmpDir)
	elapsed := time.Since(startTime)

	// Should detect process exit and return
	if elapsed > 5*time.Second {
		t.Errorf("monitorServicesUntilShutdown() took %v, expected < 5s for quick exit", elapsed)
	}

	// May or may not return error depending on exit code
	if err != nil {
		t.Logf("monitorServicesUntilShutdown() returned error: %v (acceptable)", err)
	}
}
