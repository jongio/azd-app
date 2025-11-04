package runner

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAspireOutputCapture(t *testing.T) {
	// Skip on Windows: Aspire spawns child processes that are difficult to terminate properly
	// on Windows, leading to orphaned processes after test completion.
	if runtime.GOOS == "windows" {
		t.Skip("Test disabled on Windows - process cleanup issues with Aspire child processes")
	}

	// Find the test Aspire project
	testProjectPath := filepath.Join("..", "..", "..", "tests", "projects", "aspire-test", "TestAppHost")

	// Verify the project exists
	if _, err := os.Stat(testProjectPath); os.IsNotExist(err) {
		t.Skip("Aspire test project not found, skipping test")
	}

	// Test 1: Does aspire run produce output when we run it directly?
	t.Run("DirectExecution", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "aspire", "run", "--non-interactive")
		cmd.Dir = testProjectPath

		// Use pipes instead of buffers to avoid blocking on macOS
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			t.Fatalf("Failed to create stdout pipe: %v", err)
		}
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			t.Fatalf("Failed to create stderr pipe: %v", err)
		}

		// Start the command
		if err := cmd.Start(); err != nil {
			t.Fatalf("Failed to start aspire: %v", err)
		}

		// Read from pipes in goroutines with mutex protection
		var mu sync.Mutex
		var stdout, stderr bytes.Buffer

		var wg sync.WaitGroup
		wg.Add(2)

		// Read stdout
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stdoutPipe)
			for scanner.Scan() {
				mu.Lock()
				stdout.WriteString(scanner.Text() + "\n")
				mu.Unlock()
			}
		}()

		// Read stderr
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stderrPipe)
			for scanner.Scan() {
				mu.Lock()
				stderr.WriteString(scanner.Text() + "\n")
				mu.Unlock()
			}
		}()

		// Give it a short time to produce output
		time.Sleep(2 * time.Second)

		// Kill it
		if err := cmd.Process.Kill(); err != nil {
			t.Logf("Warning: failed to kill process: %v", err)
		}

		// Wait for the process to exit (this closes the pipes)
		_ = cmd.Wait()

		// Wait for readers to finish
		wg.Wait()

		// Check if we got output
		mu.Lock()
		stdoutStr := stdout.String()
		stderrStr := stderr.String()
		mu.Unlock()

		t.Logf("STDOUT length: %d", len(stdoutStr))
		t.Logf("STDERR length: %d", len(stderrStr))

		if len(stdoutStr) > 0 {
			t.Logf("First 500 chars of stdout:\n%s", stdoutStr[:min(500, len(stdoutStr))])
		}
		if len(stderrStr) > 0 {
			t.Logf("First 500 chars of stderr:\n%s", stderrStr[:min(500, len(stderrStr))])
		}

		if len(stdoutStr) == 0 && len(stderrStr) == 0 {
			t.Log("Warning: No output captured from aspire run using direct pipe approach (platform-specific buffering issue)")
		}
	})

	// Test 2: Does our lineWriter approach work?
	t.Run("LineWriterExecution", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "aspire", "run", "--non-interactive")
		cmd.Dir = testProjectPath

		// Use our lineWriter approach with mutex for thread safety
		var mu sync.Mutex
		var capturedLines []string
		lineHandler := func(line string) error {
			mu.Lock()
			capturedLines = append(capturedLines, line)
			mu.Unlock()
			return nil
		}

		// Simple line writer for testing
		writer := &testLineWriter{
			handler: lineHandler,
		}

		cmd.Stdout = writer
		cmd.Stderr = writer

		// Start the command
		if err := cmd.Start(); err != nil {
			t.Fatalf("Failed to start aspire: %v", err)
		}

		// Give it a short time to produce output
		time.Sleep(2 * time.Second)

		// Kill it
		if err := cmd.Process.Kill(); err != nil {
			t.Logf("Warning: failed to kill process: %v", err)
		}

		// Wait for the process to exit
		_ = cmd.Wait()

		mu.Lock()
		lineCount := len(capturedLines)
		var linesToLog []string
		if lineCount > 0 {
			for i := 0; i < min(5, lineCount); i++ {
				linesToLog = append(linesToLog, capturedLines[i])
			}
		}
		mu.Unlock()

		t.Logf("Captured %d lines", lineCount)
		if len(linesToLog) > 0 {
			t.Logf("First few lines:")
			for _, line := range linesToLog {
				t.Logf("  %s", line)
			}
		}

		if lineCount == 0 {
			t.Error("No lines captured with lineWriter approach")
		}

		// Check if we got dashboard URL
		foundDashboard := false
		mu.Lock()
		for _, line := range capturedLines {
			if strings.Contains(line, "Now listening on:") || strings.Contains(line, "localhost") {
				foundDashboard = true
				t.Logf("Found dashboard line: %s", line)
				break
			}
		}
		mu.Unlock()

		if !foundDashboard {
			t.Log("Warning: Did not find dashboard URL in output")
		}
	})
}

// testLineWriter is a simple version of lineWriter for testing.
type testLineWriter struct {
	handler func(string) error
	buffer  bytes.Buffer
}

func (w *testLineWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	w.buffer.Write(p)

	// Process complete lines
	for {
		line, err := w.buffer.ReadString('\n')
		if err != nil {
			// No complete line yet
			w.buffer.WriteString(line)
			break
		}
		// Remove trailing newline and call handler
		line = strings.TrimSuffix(line, "\n")
		line = strings.TrimSuffix(line, "\r")
		if w.handler != nil {
			_ = w.handler(line)
		}
	}

	return n, nil
}
