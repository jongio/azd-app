package service

import (
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestValidateRuntime(t *testing.T) {
	tests := []struct {
		name    string
		runtime ServiceRuntime
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid runtime",
			runtime: ServiceRuntime{
				Name:       "test-service",
				WorkingDir: ".",
				Command:    "echo",
				Language:   "shell",
			},
			wantErr: false,
		},
		{
			name: "missing service name",
			runtime: ServiceRuntime{
				WorkingDir: ".",
				Command:    "echo",
				Language:   "shell",
			},
			wantErr: true,
			errMsg:  "service name is required",
		},
		{
			name: "missing working directory",
			runtime: ServiceRuntime{
				Name:     "test-service",
				Command:  "echo",
				Language: "shell",
			},
			wantErr: true,
			errMsg:  "working directory is required",
		},
		{
			name: "missing command",
			runtime: ServiceRuntime{
				Name:       "test-service",
				WorkingDir: ".",
				Language:   "shell",
			},
			wantErr: true,
			errMsg:  "run command is required",
		},
		{
			name: "missing language",
			runtime: ServiceRuntime{
				Name:       "test-service",
				WorkingDir: ".",
				Command:    "echo",
			},
			wantErr: true,
			errMsg:  "language is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRuntime(&tt.runtime)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateRuntime() expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateRuntime() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateRuntime() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestGetProcessStatus(t *testing.T) {
	// Test not started
	process := &ServiceProcess{
		Name:    "test",
		Process: nil,
	}

	got := GetProcessStatus(process)
	if got != "not-started" {
		t.Errorf("GetProcessStatus() = %v, want not-started", got)
	}

	// We can't easily test running processes without actually starting one
	// The integration test TestStartService_Success covers this
}

func TestStartService_InvalidCommand(t *testing.T) {
	runtime := &ServiceRuntime{
		Name:       "test-service",
		WorkingDir: ".",
		Command:    "",
		Language:   "shell",
	}

	_, err := StartService(runtime, nil, ".", nil)
	if err == nil {
		t.Error("StartService() expected error for empty command")
	}
	if !strings.Contains(err.Error(), "no command specified") {
		t.Errorf("StartService() error = %v, want error containing 'no command specified'", err)
	}
}

func TestStartService_NonexistentCommand(t *testing.T) {
	runtime := &ServiceRuntime{
		Name:       "test-service",
		WorkingDir: ".",
		Command:    "nonexistent-command-xyz-123",
		Args:       []string{},
		Language:   "shell",
	}

	_, err := StartService(runtime, nil, ".", nil)
	if err == nil {
		t.Error("StartService() expected error for nonexistent command")
	}
}

func TestStartService_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping process test in short mode")
	}

	tmpDir := t.TempDir()

	runtime := &ServiceRuntime{
		Name:       "test-echo",
		WorkingDir: tmpDir,
		Command:    "timeout",
		Args:       []string{"5"},
		Language:   "shell",
		Port:       8080,
	}

	process, err := StartService(runtime, map[string]string{"TEST_VAR": "value"}, tmpDir, nil)
	if err != nil {
		t.Fatalf("StartService() error = %v", err)
	}

	if process == nil {
		t.Fatal("StartService() returned nil process")
	}

	if process.Name != "test-echo" {
		t.Errorf("Process name = %v, want test-echo", process.Name)
	}

	if process.Port != 8080 {
		t.Errorf("Process port = %v, want 8080", process.Port)
	}

	if process.Process == nil {
		t.Error("Process.Process is nil")
	}

	// Clean up - remove log buffer to release file handles
	logMgr := GetLogManager(tmpDir)
	_ = logMgr.RemoveBuffer(runtime.Name)

	// Now stop the process
	if process.Process != nil {
		_ = StopServiceGraceful(process, DefaultStopTimeout)
		// Give Windows time to release file handles
		time.Sleep(100 * time.Millisecond)
	}
}

func TestStopServiceGraceful_NotStarted(t *testing.T) {
	process := &ServiceProcess{
		Name:    "test",
		Process: nil,
	}

	err := StopServiceGraceful(process, DefaultStopTimeout)
	if err == nil {
		t.Error("StopServiceGraceful() expected error for nil process")
	}
	if !strings.Contains(err.Error(), "process not started") {
		t.Errorf("StopServiceGraceful() error = %v, want error containing 'process not started'", err)
	}
}

func TestStopServiceGraceful_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	if testing.Short() {
		t.Skip("skipping process test in short mode")
	}

	tmpDir := t.TempDir()

	// Start a simple long-running process
	runtime := &ServiceRuntime{
		Name:       "test-stop",
		WorkingDir: tmpDir,
		Command:    "timeout",
		Args:       []string{"30"}, // Will run for 30 seconds
		Language:   "shell",
		Port:       8081,
	}

	process, err := StartService(runtime, nil, tmpDir, nil)
	if err != nil {
		t.Fatalf("StartService() error = %v", err)
	}
	defer func() {
		// Cleanup
		logMgr := GetLogManager(tmpDir)
		_ = logMgr.RemoveBuffer(runtime.Name)
	}()

	// Give process a moment to start
	time.Sleep(100 * time.Millisecond)

	// Verify process is running
	if process.Process == nil {
		t.Fatal("Process not started")
	}

	// Test graceful stop - on Windows this should use Kill() directly without signal errors
	err = StopServiceGraceful(process, 2*time.Second)

	// Should succeed without "not supported by windows" or "invalid argument" errors
	if err != nil && strings.Contains(err.Error(), "not supported") {
		t.Errorf("StopServiceGraceful() should not fail with 'not supported' on Windows: %v", err)
	}
	if err != nil && strings.Contains(err.Error(), "invalid argument") {
		t.Errorf("StopServiceGraceful() should not fail with 'invalid argument' on Windows: %v", err)
	}

	// Process should be stopped
	time.Sleep(100 * time.Millisecond)
}

func TestExecuteCommand(t *testing.T) {
	tests := []struct {
		name    string
		cmd     string
		args    []string
		dir     string
		wantErr bool
	}{
		{
			name:    "valid command",
			cmd:     "go",
			args:    []string{"version"},
			dir:     ".",
			wantErr: false,
		},
		{
			name:    "invalid command",
			cmd:     "nonexistent-command-xyz",
			args:    []string{},
			dir:     ".",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ExecuteCommand(tt.cmd, tt.args, tt.dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInferLogLevel(t *testing.T) {
	tests := []struct {
		message  string
		isStderr bool
		want     LogLevel
	}{
		// Standard info detection (stdout)
		{"Normal log message", false, LogLevelInfo},
		{"INFO: starting service", false, LogLevelInfo},
		{"Starting application", false, LogLevelInfo},

		// Standard error detection
		{"ERROR: something went wrong", false, LogLevelError},
		{"error occurred", false, LogLevelError},
		{"Exception in thread", false, LogLevelError},
		{"FATAL: critical failure", false, LogLevelError},
		{"panic: runtime error", false, LogLevelError},
		{"5 errors found", false, LogLevelError}, // plural still matches

		// Standard warning detection
		{"WARNING: deprecation notice", false, LogLevelWarn},
		{"warn: invalid config", false, LogLevelWarn},

		// Debug detection
		{"DEBUG: verbose output", false, LogLevelDebug},
		{"trace information", false, LogLevelDebug},

		// Word-boundary: identifiers must NOT misfire a level
		{"errorReporter started", false, LogLevelInfo}, // not \berror\b
		{"trace_worker_started", false, LogLevelInfo},  // not \btrace\b
		{"warmup complete", false, LogLevelInfo},       // not \bwarn\b

		// Structured JSON logs are authoritative (emitter's own level wins)
		{`{"level":"info","msg":"trace_worker_started","role":"trace-worker"}`, false, LogLevelInfo},
		{`{"level":"error","msg":"consumer_error"}`, false, LogLevelError},
		{`{"level":"warn","msg":"retrying"}`, false, LogLevelWarn},
		{`{"level":"debug","msg":"tick"}`, false, LogLevelDebug},
		{`{"severity":"ERROR","message":"boom"}`, false, LogLevelError},
		{`{"level":"info","msg":"error handler ready"}`, false, LogLevelInfo}, // structured beats the word "error"

		// Stream-aware default: unclassified stderr surfaces as WARN, stdout as INFO
		{".env not found. Continuing without it.", true, LogLevelWarn},
		{".env not found. Continuing without it.", false, LogLevelInfo},
		{"plain diagnostic line", true, LogLevelWarn},

		// INFO OVERRIDES - These contain "error" but should be INFO
		{"Found 0 errors. Watching for file changes.", false, LogLevelInfo},
		{"2:59:24 PM - Found 0 errors. Watching for file changes.", false, LogLevelInfo},
		{"Build succeeded with 0 error(s)", false, LogLevelInfo},
		{"Compilation succeeded", false, LogLevelInfo},
		{"compiled successfully", false, LogLevelInfo},
		{"0 failed, 10 passed", false, LogLevelInfo},
		{"All tests passed", false, LogLevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			got := inferLogLevel(tt.message, tt.isStderr)
			if got != tt.want {
				t.Errorf("inferLogLevel(%q, isStderr=%v) = %v, want %v", tt.message, tt.isStderr, got, tt.want)
			}
		})
	}
}

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level LogLevel
		want  string
	}{
		{LogLevelDebug, "DEBUG"},
		{LogLevelInfo, "INFO"},
		{LogLevelWarn, "WARN"},
		{LogLevelError, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.level.String()
			if got != tt.want {
				t.Errorf("LogLevel.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadServiceOutput(t *testing.T) {
	// Create a pipe to simulate service output
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	defer func() { _ = pr.Close() }()
	defer func() { _ = pw.Close() }()

	outputChan := make(chan string, 10)

	// Start reading in background
	go ReadServiceOutput(pr, outputChan)

	// Write test lines
	testLines := []string{
		"Line 1",
		"Line 2",
		"Line 3",
	}

	go func() {
		for _, line := range testLines {
			if _, err := pw.WriteString(line + "\n"); err != nil {
				break
			}
		}
		_ = pw.Close()
	}()

	// Collect output with timeout
	timeout := time.After(2 * time.Second)
	collected := []string{}

	for i := 0; i < len(testLines); i++ {
		select {
		case line := <-outputChan:
			collected = append(collected, line)
		case <-timeout:
			t.Fatal("Timeout waiting for output")
		}
	}

	// Verify we got all lines
	if len(collected) != len(testLines) {
		t.Errorf("Got %d lines, want %d", len(collected), len(testLines))
	}

	for i, want := range testLines {
		if i < len(collected) && collected[i] != want {
			t.Errorf("Line %d = %q, want %q", i, collected[i], want)
		}
	}
}
