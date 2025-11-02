package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/output"
)

// DefaultTimeout is the default timeout for command execution.
const DefaultTimeout = 30 * time.Minute

// RunWithContext executes a command with context for cancellation and timeout.
// The command inherits all environment variables from the parent process, including
// azd-specific variables like AZD_SERVER, AZD_ACCESS_TOKEN, and environment values.
func RunWithContext(ctx context.Context, name string, args []string, dir string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	// In JSON mode, suppress output from subprocesses to ensure valid JSON
	if output.IsJSON() {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		cmd.Stdin = nil
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
	}

	cmd.Env = os.Environ() // Inherit all environment variables from parent process

	return cmd.Run()
}

// RunWithTimeout executes a command with a timeout.
func RunWithTimeout(name string, args []string, dir string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return RunWithContext(ctx, name, args, dir)
}

// RunCommand executes a command safely with default timeout.
func RunCommand(name string, args []string, dir string) error {
	return RunWithTimeout(name, args, dir, DefaultTimeout)
}

// StartCommand starts a long-running command in the background and returns immediately.
// The command inherits stdout/stderr/stdin from the parent process.
// The command inherits all environment variables including azd context (AZD_SERVER, AZD_ACCESS_TOKEN, etc.).
// Use this for starting servers, Aspire projects, or other long-running processes.
func StartCommand(name string, args []string, dir string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ() // Inherit all environment variables from parent process

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	output.Newline()
	output.Success("Started %s (PID: %d)", name, cmd.Process.Pid)
	output.Item("Output will appear below. Press Ctrl+C to stop it when ready.")
	output.Newline()

	return nil
}

// RunCommandWithOutput executes a command and captures output with timeout.
// The command inherits all environment variables from the parent process.
func RunCommandWithOutput(ctx context.Context, name string, args []string, dir string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ() // Inherit all environment variables from parent process

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("command failed: %w", err)
	}

	return output, nil
}

// OutputLineHandler is called for each line of output from a command.
type OutputLineHandler func(line string) error

// lineWriter wraps an io.Writer and calls a handler for each complete line.
type lineWriter struct {
	output  io.Writer
	handler OutputLineHandler
	buffer  bytes.Buffer
	mu      sync.Mutex
}

func (lw *lineWriter) Write(p []byte) (n int, err error) {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	// Write to the actual output first
	n, err = lw.output.Write(p)
	if err != nil {
		return n, err
	}

	// Add to buffer and process complete lines
	lw.buffer.Write(p)
	for {
		line, err := lw.buffer.ReadString('\n')
		if err != nil {
			// No complete line yet, put it back
			lw.buffer.WriteString(line)
			break
		}
		// Remove trailing newline and call handler
		line = line[:len(line)-1]
		if lw.handler != nil {
			_ = lw.handler(line)
		}
	}

	return n, nil
}

// StartCommandWithOutputMonitoring starts a command and monitors its output line-by-line.
// The handler function is called for each line of stdout/stderr.
// Output is still displayed to the user in real-time.
// The command inherits all environment variables including azd context.
// This function BLOCKS and waits for the command to complete or be interrupted.
func StartCommandWithOutputMonitoring(name string, args []string, dir string, handler OutputLineHandler) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ() // Inherit all environment variables from parent process

	// Wrap stdout and stderr with line handlers
	cmd.Stdout = &lineWriter{output: os.Stdout, handler: handler}
	cmd.Stderr = &lineWriter{output: os.Stderr, handler: handler}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	output.Newline()
	output.Success("Started %s (PID: %d)", name, cmd.Process.Pid)
	output.Item("Press Ctrl+C to stop.")
	output.Newline()

	// Wait for the command to complete (this blocks until the process exits or is killed)
	if err := cmd.Wait(); err != nil {
		// Ignore exit errors from Ctrl+C
		return nil
	}

	return nil
}
