package executor

import (
	"context"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunWithContext(t *testing.T) {
	ctx := context.Background()

	var name string
	var args []string

	if runtime.GOOS == "windows" {
		name = "cmd.exe"
		args = []string{"/c", "echo", "test"}
	} else {
		name = "echo"
		args = []string{"test"}
	}

	err := RunWithContext(ctx, name, args, "")
	if err != nil {
		t.Errorf("RunWithContext() error = %v, want nil", err)
	}
}

func TestRunWithContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	var name string
	var args []string

	if runtime.GOOS == "windows" {
		name = "cmd.exe"
		args = []string{"/c", "timeout", "10"}
	} else {
		name = "sleep"
		args = []string{"10"}
	}

	err := RunWithContext(ctx, name, args, "")
	if err == nil {
		t.Errorf("RunWithContext() with canceled context should fail")
	}
}

func TestRunWithContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var name string
	var args []string

	if runtime.GOOS == "windows" {
		name = "cmd.exe"
		args = []string{"/c", "timeout", "10"}
	} else {
		name = "sleep"
		args = []string{"10"}
	}

	err := RunWithContext(ctx, name, args, "")
	if err == nil {
		t.Errorf("RunWithContext() with timeout should fail")
	}
}

func TestRunWithContextInDirectory(t *testing.T) {
	tempDir := t.TempDir()

	ctx := context.Background()

	var name string
	var args []string

	if runtime.GOOS == "windows" {
		name = "cmd.exe"
		args = []string{"/c", "cd"}
	} else {
		name = "pwd"
		args = []string{}
	}

	err := RunWithContext(ctx, name, args, tempDir)
	if err != nil {
		t.Errorf("RunWithContext() in directory error = %v, want nil", err)
	}
}

func TestRunWithTimeout(t *testing.T) {
	var name string
	var args []string

	if runtime.GOOS == "windows" {
		name = "cmd.exe"
		args = []string{"/c", "echo", "test"}
	} else {
		name = "echo"
		args = []string{"test"}
	}

	err := RunWithTimeout(name, args, "", 5*time.Second)
	if err != nil {
		t.Errorf("RunWithTimeout() error = %v, want nil", err)
	}
}

func TestRunWithTimeoutExceeded(t *testing.T) {
	var name string
	var args []string

	if runtime.GOOS == "windows" {
		name = "cmd.exe"
		args = []string{"/c", "timeout", "10"}
	} else {
		name = "sleep"
		args = []string{"10"}
	}

	err := RunWithTimeout(name, args, "", 100*time.Millisecond)
	if err == nil {
		t.Errorf("RunWithTimeout() should timeout")
	}
}

func TestRunCommand(t *testing.T) {
	var name string
	var args []string

	if runtime.GOOS == "windows" {
		name = "cmd.exe"
		args = []string{"/c", "echo", "test"}
	} else {
		name = "echo"
		args = []string{"test"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := RunCommand(ctx, name, args, "")
	if err != nil {
		t.Errorf("RunCommand() error = %v, want nil", err)
	}
}

func TestRunCommandInvalidCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := RunCommand(ctx, "nonexistent-command-xyz-123", []string{}, "")
	if err == nil {
		t.Errorf("RunCommand() with invalid command should fail")
	}
}

func TestStartCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	var name string
	var args []string

	if runtime.GOOS == "windows" {
		name = "cmd.exe"
		args = []string{"/c", "echo test > " + testFile}
	} else {
		name = "sh"
		args = []string{"-c", "echo test > " + testFile}
	}

	// Note: StartCommand starts in background, so we just verify it doesn't error
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := StartCommand(ctx, name, args, tempDir)
	if err != nil {
		t.Errorf("StartCommand() error = %v, want nil", err)
	}

	// Give the command time to execute
	time.Sleep(500 * time.Millisecond)
}

func TestStartCommandInvalidCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := StartCommand(ctx, "nonexistent-command-xyz-123", []string{}, "")
	if err == nil {
		t.Errorf("StartCommand() with invalid command should fail")
	}
}

func TestRunCommandWithOutput(t *testing.T) {
	ctx := context.Background()

	var name string
	var args []string
	var expectedOutput string

	if runtime.GOOS == "windows" {
		name = "cmd.exe"
		args = []string{"/c", "echo", "test"}
		expectedOutput = "test"
	} else {
		name = "echo"
		args = []string{"test"}
		expectedOutput = "test"
	}

	output, err := RunCommandWithOutput(ctx, name, args, "")
	if err != nil {
		t.Fatalf("RunCommandWithOutput() error = %v, want nil", err)
	}

	outputStr := strings.TrimSpace(string(output))
	if !strings.Contains(outputStr, expectedOutput) {
		t.Errorf("RunCommandWithOutput() output = %q, want to contain %q", outputStr, expectedOutput)
	}
}

func TestRunCommandWithOutputInDirectory(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	var name string
	var args []string

	if runtime.GOOS == "windows" {
		name = "cmd.exe"
		args = []string{"/c", "cd"}
	} else {
		name = "pwd"
		args = []string{}
	}

	output, err := RunCommandWithOutput(ctx, name, args, tempDir)
	if err != nil {
		t.Fatalf("RunCommandWithOutput() error = %v, want nil", err)
	}

	if len(output) == 0 {
		t.Errorf("RunCommandWithOutput() output is empty")
	}
}

func TestRunCommandWithOutputCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	var name string
	var args []string

	if runtime.GOOS == "windows" {
		name = "cmd.exe"
		args = []string{"/c", "timeout", "10"}
	} else {
		name = "sleep"
		args = []string{"10"}
	}

	_, err := RunCommandWithOutput(ctx, name, args, "")
	if err == nil {
		t.Errorf("RunCommandWithOutput() with canceled context should fail")
	}
}

func TestRunCommandWithOutputInvalidCommand(t *testing.T) {
	ctx := context.Background()

	_, err := RunCommandWithOutput(ctx, "nonexistent-command-xyz-123", []string{}, "")
	if err == nil {
		t.Errorf("RunCommandWithOutput() with invalid command should fail")
	}
}

func TestLineWriter(t *testing.T) {
	var lines []string
	handler := func(line string) error {
		lines = append(lines, line)
		return nil
	}

	lw := &lineWriter{
		output:  io.Discard,
		handler: handler,
	}

	// Write complete line
	_, err := lw.Write([]byte("line 1\n"))
	if err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}

	if len(lines) != 1 {
		t.Fatalf("len(lines) = %v, want 1", len(lines))
	}

	if lines[0] != "line 1" {
		t.Errorf("lines[0] = %q, want %q", lines[0], "line 1")
	}

	// Write multiple lines at once
	_, err = lw.Write([]byte("line 2\nline 3\n"))
	if err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}

	if len(lines) != 3 {
		t.Fatalf("len(lines) = %v, want 3", len(lines))
	}

	if lines[1] != "line 2" {
		t.Errorf("lines[1] = %q, want %q", lines[1], "line 2")
	}

	if lines[2] != "line 3" {
		t.Errorf("lines[2] = %q, want %q", lines[2], "line 3")
	}
}

func TestLineWriterIncomplete(t *testing.T) {
	var lines []string
	handler := func(line string) error {
		lines = append(lines, line)
		return nil
	}

	lw := &lineWriter{
		output:  io.Discard,
		handler: handler,
	}

	// Write incomplete line
	_, err := lw.Write([]byte("partial"))
	if err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}

	// No lines should be captured yet
	if len(lines) != 0 {
		t.Fatalf("len(lines) = %v, want 0", len(lines))
	}

	// Complete the line
	_, err = lw.Write([]byte(" line\n"))
	if err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}

	if len(lines) != 1 {
		t.Fatalf("len(lines) = %v, want 1", len(lines))
	}

	if lines[0] != "partial line" {
		t.Errorf("lines[0] = %q, want %q", lines[0], "partial line")
	}
}

func TestStartCommandWithOutputMonitoring(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	var lines []string
	handler := func(line string) error {
		lines = append(lines, line)
		return nil
	}

	var name string
	var args []string

	if runtime.GOOS == "windows" {
		name = "cmd.exe"
		args = []string{"/c", "echo", "test output"}
	} else {
		name = "echo"
		args = []string{"test output"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := StartCommandWithOutputMonitoring(ctx, name, args, "", handler)
	if err != nil {
		t.Fatalf("StartCommandWithOutputMonitoring() error = %v, want nil", err)
	}

	// Verify handler was called
	if len(lines) == 0 {
		t.Errorf("handler was not called, lines is empty")
	}

	// Check if any line contains the expected output
	found := false
	for _, line := range lines {
		if strings.Contains(line, "test output") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("output not captured, lines = %v", lines)
	}
}

func TestStartCommandWithOutputMonitoringInvalidCommand(t *testing.T) {
	handler := func(line string) error {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := StartCommandWithOutputMonitoring(ctx, "nonexistent-command-xyz-123", []string{}, "", handler)
	if err == nil {
		t.Errorf("StartCommandWithOutputMonitoring() with invalid command should fail")
	}
}
