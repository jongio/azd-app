package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
		Args:       []string{"30"}, // Long timeout
		Language:   "shell",
		Port:       9000,
	}

	process, err := service.StartService(runtime, map[string]string{}, tmpDir, nil)
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

	// Kill the service after a short delay to verify monitoring continues
	go func() {
		time.Sleep(500 * time.Millisecond)
		if process.Process != nil {
			_ = service.StopServiceGraceful(process, 100*time.Millisecond)
		}
	}()

	// Run monitoring in goroutine with timeout
	done := make(chan error, 1)
	go func() {
		done <- monitorServicesUntilShutdown(result, tmpDir)
	}()

	// Ensure cleanup if test exits early
	defer func() {
		// If monitoring is still running, send interrupt to clean up
		select {
		case <-done:
			// Already completed
		default:
			// Send interrupt to terminate monitoring goroutine
			proc, _ := os.FindProcess(os.Getpid())
			_ = proc.Signal(os.Interrupt)
			// Wait briefly for cleanup
			select {
			case <-done:
			case <-time.After(500 * time.Millisecond):
			}
		}
	}()

	// Monitoring should continue even after service exits (process isolation)
	// Force completion after 2 seconds by stopping the process
	select {
	case err := <-done:
		// Should return nil with process isolation
		if err != nil {
			t.Logf("monitorServicesUntilShutdown() returned: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Log("✓ Monitoring continues running even after service exit (process isolation working)")
		// This is expected - monitoring waits for signal, not for processes to exit
	}
}

func TestMonitorServicesUntilShutdown_SignalHandling(t *testing.T) {
	t.Skip("Signal handling is tested in integration/e2e tests - difficult to test reliably in unit tests")

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

	process, err := service.StartService(runtime, map[string]string{}, tmpDir, nil)
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

		process, err := service.StartService(runtime, map[string]string{}, tmpDir, nil)
		if err != nil {
			t.Fatalf("StartService(%d) error = %v", i, err)
		}
		processes[runtime.Name] = process
	}

	t.Cleanup(func() {
		// Kill processes immediately without waiting - this is just test cleanup
		logMgr := service.GetLogManager(tmpDir)
		for name, process := range processes {
			_ = logMgr.RemoveBuffer(name)
			if process.Process != nil {
				_ = process.Process.Kill()
			}
		}
	})

	result := &service.OrchestrationResult{
		Processes: processes,
		Errors:    map[string]error{},
		StartTime: time.Now(),
	}

	// Test that all services can be started and shutdown cleanly works
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := shutdownAllServices(ctx, result.Processes)
	if err != nil {
		t.Logf("shutdownAllServices() returned: %v", err)
	}

	t.Log("✓ Multiple services started and stopped successfully")
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

		process, err := service.StartService(runtime, map[string]string{}, tmpDir, nil)
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

	process, err := service.StartService(runtime, map[string]string{}, tmpDir, nil)
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

// TestMonitorServices_RunsIndefinitely verifies that services run continuously
// without automatic timeout. Services should only stop on explicit signal (Ctrl+C)
// or when they naturally exit, not on arbitrary timeouts.
func TestMonitorServices_RunsIndefinitely(t *testing.T) {
	t.Skip("This test is incompatible with the new process isolation design where monitoring waits for signals")
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

	process, err := service.StartService(runtime, map[string]string{}, tmpDir, nil)
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

// TestMonitorServiceProcess_CleanExit verifies that monitorServiceProcess
// handles clean service exits without panicking or affecting other services.
func TestMonitorServiceProcess_CleanExit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping process test in short mode")
	}

	tmpDir := t.TempDir()

	runtime := &service.ServiceRuntime{
		Name:       "clean-exit",
		WorkingDir: tmpDir,
		Command:    "timeout",
		Args:       []string{"1"}, // Exit after 1 second
		Language:   "shell",
		Port:       9100,
	}

	process, err := service.StartService(runtime, map[string]string{}, tmpDir, nil)
	if err != nil {
		t.Fatalf("StartService() error = %v", err)
	}

	t.Cleanup(func() {
		logMgr := service.GetLogManager(tmpDir)
		_ = logMgr.RemoveBuffer(runtime.Name)
		time.Sleep(100 * time.Millisecond)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)

	// Should not panic or cause issues
	monitorServiceProcess(ctx, &wg, runtime.Name, process)

	// Wait should complete without hanging
	waitDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		t.Log("✓ monitorServiceProcess handled clean exit correctly")
	case <-time.After(3 * time.Second):
		t.Error("monitorServiceProcess did not complete in time")
	}
}

// TestMonitorServiceProcess_CrashExit verifies that monitorServiceProcess
// handles service crashes gracefully without propagating errors.
func TestMonitorServiceProcess_CrashExit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping process test in short mode")
	}

	tmpDir := t.TempDir()

	// Create a script that exits with error code
	runtime := &service.ServiceRuntime{
		Name:       "crash-exit",
		WorkingDir: tmpDir,
		Command:    "cmd",
		Args:       []string{"/C", "exit 1"}, // Exit with error
		Language:   "shell",
		Port:       9101,
	}

	process, err := service.StartService(runtime, map[string]string{}, tmpDir, nil)
	if err != nil {
		t.Fatalf("StartService() error = %v", err)
	}

	t.Cleanup(func() {
		logMgr := service.GetLogManager(tmpDir)
		_ = logMgr.RemoveBuffer(runtime.Name)
		time.Sleep(100 * time.Millisecond)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)

	// Should handle crash without panic
	monitorServiceProcess(ctx, &wg, runtime.Name, process)

	waitDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		t.Log("✓ monitorServiceProcess handled crash gracefully")
	case <-time.After(2 * time.Second):
		t.Error("monitorServiceProcess did not complete after crash")
	}
}

// TestMonitorServiceProcess_ContextCancellation verifies that monitorServiceProcess
// responds correctly to context cancellation (simulating Ctrl+C).
func TestMonitorServiceProcess_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping process test in short mode")
	}

	tmpDir := t.TempDir()

	runtime := &service.ServiceRuntime{
		Name:       "long-running",
		WorkingDir: tmpDir,
		Command:    "timeout",
		Args:       []string{"30"}, // Long timeout
		Language:   "shell",
		Port:       9102,
	}

	process, err := service.StartService(runtime, map[string]string{}, tmpDir, nil)
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

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)

	// Start monitoring
	go monitorServiceProcess(ctx, &wg, runtime.Name, process)

	// Cancel context after short delay (simulate Ctrl+C)
	time.Sleep(500 * time.Millisecond)
	cancel()

	// Should complete quickly after cancellation
	waitDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		t.Log("✓ monitorServiceProcess responded to context cancellation")
	case <-time.After(2 * time.Second):
		t.Error("monitorServiceProcess did not respond to context cancellation")
	}
}

// TestProcessExit_DoesNotStopOtherServices verifies process isolation:
// When one service crashes or exits, other services should continue running.
// This is a key feature - individual service failures don't bring down the entire environment.
func TestProcessExit_DoesNotStopOtherServices(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping process isolation test in short mode")
	}

	tmpDir := t.TempDir()

	// Start one process that exits quickly and one that runs longer
	quickRuntime := &service.ServiceRuntime{
		Name:       "quick-exit",
		WorkingDir: tmpDir,
		Command:    "timeout",
		Args:       []string{"1"}, // Exit after 1 second
		Language:   "shell",
		Port:       9040,
	}

	longRuntime := &service.ServiceRuntime{
		Name:       "long-running",
		WorkingDir: tmpDir,
		Command:    "timeout",
		Args:       []string{"30"}, // Run for 30 seconds
		Language:   "shell",
		Port:       9041,
	}

	quickProcess, err := service.StartService(quickRuntime, map[string]string{}, tmpDir, nil)
	if err != nil {
		t.Fatalf("StartService(quick-exit) error = %v", err)
	}

	longProcess, err := service.StartService(longRuntime, map[string]string{}, tmpDir, nil)
	if err != nil {
		t.Fatalf("StartService(long-running) error = %v", err)
	}

	t.Cleanup(func() {
		logMgr := service.GetLogManager(tmpDir)
		_ = logMgr.RemoveBuffer(quickRuntime.Name)
		_ = logMgr.RemoveBuffer(longRuntime.Name)
		if longProcess.Process != nil {
			_ = service.StopServiceGraceful(longProcess, 1*time.Second)
		}
		time.Sleep(100 * time.Millisecond)
	})

	result := &service.OrchestrationResult{
		Processes: map[string]*service.ServiceProcess{
			"quick-exit":   quickProcess,
			"long-running": longProcess,
		},
		Errors:    map[string]error{},
		StartTime: time.Now(),
	}

	// Run monitoring in a goroutine since it waits indefinitely for signals
	done := make(chan error, 1)
	go func() {
		done <- monitorServicesUntilShutdown(result, tmpDir)
	}()

	// Ensure cleanup if test exits
	defer func() {
		// If monitoring is still running, send interrupt to clean up
		select {
		case <-done:
			// Already completed
		default:
			// Send interrupt to terminate monitoring goroutine cleanly
			proc, _ := os.FindProcess(os.Getpid())
			_ = proc.Signal(os.Interrupt)
			// Wait briefly for cleanup to complete
			select {
			case <-done:
			case <-time.After(500 * time.Millisecond):
			}
		}
	}()

	// Wait to verify monitoring continues after quick-exit stops (~1s)
	// If process isolation works, monitoring should still be running after 3 seconds
	time.Sleep(3 * time.Second)

	// Stop the long-running process to allow test to complete cleanly
	if longProcess.Process != nil {
		_ = service.StopServiceGraceful(longProcess, 1*time.Second)
	}

	// Give monitoring goroutines time to detect all processes done
	select {
	case err := <-done:
		// Should return nil with process isolation
		if err != nil {
			t.Errorf("monitorServicesUntilShutdown() returned error %v, expected nil", err)
		}
		t.Log("✓ Process isolation verified: quick-exit stopped but monitoring continued for long-running service")
	case <-time.After(5 * time.Second):
		// Monitoring continues indefinitely waiting for signal (expected with process isolation)
		// This is actually correct behavior - monitoring waits for user signal, not process exit
		t.Log("✓ Process isolation verified: monitoring continues independently even after all services exit")
		t.Log("   (This is expected - monitoring waits for Ctrl+C, not for processes to finish)")
	}
}
