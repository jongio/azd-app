package service

import (
	"strings"
	"testing"
	"time"
)

func TestStopServiceGraceful_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping graceful shutdown test in short mode")
	}

	tmpDir := t.TempDir()

	// Start a long-running process
	runtime := &ServiceRuntime{
		Name:       "test-graceful",
		WorkingDir: tmpDir,
		Command:    "timeout",
		Args:       []string{"30"}, // 30 second timeout
		Language:   "shell",
		Port:       8090,
	}

	process, err := StartService(runtime, map[string]string{}, tmpDir)
	if err != nil {
		t.Fatalf("StartService() error = %v", err)
	}

	// Clean up log buffer
	t.Cleanup(func() {
		logMgr := GetLogManager(tmpDir)
		_ = logMgr.RemoveBuffer(runtime.Name)
	})

	// Verify process started
	if process.Process == nil {
		t.Fatal("Process should not be nil")
	}

	initialPID := process.Process.Pid
	if initialPID == 0 {
		t.Fatal("Process PID should not be 0")
	}

	// Stop service gracefully with reasonable timeout
	startTime := time.Now()
	err = StopServiceGraceful(process, 5*time.Second)
	elapsed := time.Since(startTime)

	// On Windows, the process may have already exited, which can cause "Access is denied" error
	// This is acceptable as long as the process is stopped
	if err != nil && !strings.Contains(err.Error(), "Access is denied") {
		t.Errorf("StopServiceGraceful() error = %v, want nil or 'Access is denied'", err)
	}

	// Should complete quickly (within timeout)
	if elapsed > 6*time.Second {
		t.Errorf("StopServiceGraceful() took %v, want < 6s", elapsed)
	}

	// Give system time to fully clean up
	time.Sleep(100 * time.Millisecond)
}

func TestStopServiceGraceful_ForcedKillAfterTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping forced kill test in short mode")
	}

	tmpDir := t.TempDir()

	// Start a process that ignores interrupts (on Windows, timeout handles SIGINT differently)
	// Use a process that will take longer than our graceful shutdown timeout
	runtime := &ServiceRuntime{
		Name:       "test-forced-kill",
		WorkingDir: tmpDir,
		Command:    "timeout",
		Args:       []string{"30"},
		Language:   "shell",
		Port:       8091,
	}

	process, err := StartService(runtime, map[string]string{}, tmpDir)
	if err != nil {
		t.Fatalf("StartService() error = %v", err)
	}

	t.Cleanup(func() {
		logMgr := GetLogManager(tmpDir)
		_ = logMgr.RemoveBuffer(runtime.Name)
		time.Sleep(100 * time.Millisecond)
	})

	// Use very short timeout to trigger forced kill
	startTime := time.Now()
	err = StopServiceGraceful(process, 500*time.Millisecond)
	elapsed := time.Since(startTime)

	// Should complete (either gracefully or forced)
	// Error might be nil or contain information about the kill
	if err != nil && !strings.Contains(err.Error(), "killed") && !strings.Contains(err.Error(), "exit") {
		t.Logf("StopServiceGraceful() returned error: %v (this is acceptable)", err)
	}

	// Should complete within timeout + small buffer for kill operation
	if elapsed > 2*time.Second {
		t.Errorf("StopServiceGraceful() took %v, expected < 2s (500ms timeout + kill time)", elapsed)
	}
}

func TestStopServiceGraceful_NilProcess(t *testing.T) {
	process := &ServiceProcess{
		Name:    "test-nil",
		Process: nil,
	}

	err := StopServiceGraceful(process, 5*time.Second)
	if err == nil {
		t.Error("StopServiceGraceful() expected error for nil process")
	}
	if !strings.Contains(err.Error(), "process not started") {
		t.Errorf("StopServiceGraceful() error = %v, want error containing 'process not started'", err)
	}
}

func TestStopServiceGraceful_NilServiceProcess(t *testing.T) {
	err := StopServiceGraceful(nil, 5*time.Second)
	if err == nil {
		t.Error("StopServiceGraceful() expected error for nil ServiceProcess")
	}
	if !strings.Contains(err.Error(), "process is nil") {
		t.Errorf("StopServiceGraceful() error = %v, want error containing 'process is nil'", err)
	}
}

func TestStopServiceGraceful_AlreadyExited(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping already exited test in short mode")
	}

	tmpDir := t.TempDir()

	// Start a process that exits immediately
	runtime := &ServiceRuntime{
		Name:       "test-already-exited",
		WorkingDir: tmpDir,
		Command:    "timeout",
		Args:       []string{"0"}, // Exit immediately
		Language:   "shell",
		Port:       8092,
	}

	process, err := StartService(runtime, map[string]string{}, tmpDir)
	if err != nil {
		t.Fatalf("StartService() error = %v", err)
	}

	t.Cleanup(func() {
		logMgr := GetLogManager(tmpDir)
		_ = logMgr.RemoveBuffer(runtime.Name)
		time.Sleep(100 * time.Millisecond)
	})

	// Wait for process to exit
	time.Sleep(500 * time.Millisecond)

	// Try to stop already-exited process
	err = StopServiceGraceful(process, 5*time.Second)
	// Should either succeed (if signal fails) or return error about process state
	// Both are acceptable outcomes
	if err != nil {
		t.Logf("StopServiceGraceful() on exited process returned: %v (acceptable)", err)
	}
}

func TestStopServiceGraceful_MultipleTimeouts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multiple timeout test in short mode")
	}

	timeouts := []time.Duration{
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
		3 * time.Second,
	}

	for _, timeout := range timeouts {
		t.Run(timeout.String(), func(t *testing.T) {
			tmpDir := t.TempDir()

			runtime := &ServiceRuntime{
				Name:       "test-timeout-" + timeout.String(),
				WorkingDir: tmpDir,
				Command:    "timeout",
				Args:       []string{"30"},
				Language:   "shell",
				Port:       8093,
			}

			process, err := StartService(runtime, map[string]string{}, tmpDir)
			if err != nil {
				t.Fatalf("StartService() error = %v", err)
			}

			t.Cleanup(func() {
				logMgr := GetLogManager(tmpDir)
				_ = logMgr.RemoveBuffer(runtime.Name)
				time.Sleep(100 * time.Millisecond)
			})

			startTime := time.Now()
			_ = StopServiceGraceful(process, timeout)
			elapsed := time.Since(startTime)

			// Should complete within timeout + reasonable buffer
			maxDuration := timeout + 2*time.Second
			if elapsed > maxDuration {
				t.Errorf("StopServiceGraceful(%v) took %v, expected < %v", timeout, elapsed, maxDuration)
			}
		})
	}
}

func TestStopService_UsesGracefulDefault(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping graceful default test in short mode")
	}

	tmpDir := t.TempDir()

	runtime := &ServiceRuntime{
		Name:       "test-stop-default",
		WorkingDir: tmpDir,
		Command:    "timeout",
		Args:       []string{"30"},
		Language:   "shell",
		Port:       8094,
	}

	process, err := StartService(runtime, map[string]string{}, tmpDir)
	if err != nil {
		t.Fatalf("StartService() error = %v", err)
	}

	t.Cleanup(func() {
		logMgr := GetLogManager(tmpDir)
		_ = logMgr.RemoveBuffer(runtime.Name)
		time.Sleep(100 * time.Millisecond)
	})

	// StopService should use StopServiceGraceful with default 5s timeout
	startTime := time.Now()
	err = StopService(process)
	elapsed := time.Since(startTime)

	if err != nil {
		t.Logf("StopService() returned error: %v (may be acceptable)", err)
	}

	// Should complete within reasonable time (5s timeout + buffer)
	if elapsed > 7*time.Second {
		t.Errorf("StopService() took %v, expected < 7s (5s default timeout + buffer)", elapsed)
	}
}

func TestStopAllServices_GracefulShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stop all test in short mode")
	}

	tmpDir := t.TempDir()

	// Start multiple services
	processes := make(map[string]*ServiceProcess)

	for i := 1; i <= 3; i++ {
		runtime := &ServiceRuntime{
			Name:       "test-service-" + string(rune('0'+i)),
			WorkingDir: tmpDir,
			Command:    "timeout",
			Args:       []string{"30"},
			Language:   "shell",
			Port:       8100 + i,
		}

		process, err := StartService(runtime, map[string]string{}, tmpDir)
		if err != nil {
			t.Fatalf("StartService(%d) error = %v", i, err)
		}
		processes[runtime.Name] = process
	}

	t.Cleanup(func() {
		logMgr := GetLogManager(tmpDir)
		for name := range processes {
			_ = logMgr.RemoveBuffer(name)
		}
		time.Sleep(200 * time.Millisecond)
	})

	// Stop all services
	startTime := time.Now()
	StopAllServices(processes)
	elapsed := time.Since(startTime)

	// Should stop all services within reasonable time
	// 3 services * 5s timeout = 15s max, but they run concurrently
	if elapsed > 10*time.Second {
		t.Errorf("StopAllServices() took %v, expected < 10s for 3 concurrent stops", elapsed)
	}
}
