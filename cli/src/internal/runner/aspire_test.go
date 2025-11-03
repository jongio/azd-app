package runner

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAspireOutputCapture(t *testing.T) {
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

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		// Start the command
		if err := cmd.Start(); err != nil {
			t.Fatalf("Failed to start aspire: %v", err)
		}

		// Give it a few seconds to produce output
		time.Sleep(5 * time.Second)

		// Kill it
		if err := cmd.Process.Kill(); err != nil {
			t.Logf("Warning: failed to kill process: %v", err)
		}

		// Check if we got output
		stdoutStr := stdout.String()
		stderrStr := stderr.String()

		t.Logf("STDOUT length: %d", len(stdoutStr))
		t.Logf("STDERR length: %d", len(stderrStr))

		if len(stdoutStr) > 0 {
			t.Logf("First 500 chars of stdout:\n%s", stdoutStr[:min(500, len(stdoutStr))])
		}
		if len(stderrStr) > 0 {
			t.Logf("First 500 chars of stderr:\n%s", stderrStr[:min(500, len(stderrStr))])
		}

		if len(stdoutStr) == 0 && len(stderrStr) == 0 {
			t.Error("No output captured from aspire run")
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

		// Give it time to produce output
		time.Sleep(5 * time.Second)

		// Kill it
		if err := cmd.Process.Kill(); err != nil {
			t.Logf("Warning: failed to kill process: %v", err)
		}

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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
