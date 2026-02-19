// Package executor provides safe command execution with output monitoring and timeout handling.
// Generic command execution is delegated to github.com/jongio/azd-core/cmdutil.
// This package adds app-specific behavior: JSON mode output suppression, display messages,
// and error-returning OutputLineHandler.
package executor

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/jongio/azd-core/cliout"
	"github.com/jongio/azd-core/cmdutil"
)

// DefaultTimeout is the default timeout for command execution.
const DefaultTimeout = cmdutil.DefaultTimeout

// RunWithContext executes a command with context for cancellation and timeout.
// The command inherits all environment variables from the parent process, including
// azd-specific variables like AZD_SERVER, AZD_ACCESS_TOKEN, and environment values.
func RunWithContext(ctx context.Context, name string, args []string, dir string) error {
	// In JSON mode, suppress output from subprocesses to ensure valid JSON
	if cliout.IsJSON() {
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir = dir
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		cmd.Stdin = nil
		cmd.Env = os.Environ()
		return cmd.Run()
	}

	return cmdutil.RunWithContext(ctx, name, args, dir)
}

// RunWithTimeout executes a command with a timeout.
func RunWithTimeout(name string, args []string, dir string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return RunWithContext(ctx, name, args, dir)
}

// RunCommand executes a command safely with context and default timeout.
// The provided context must be non-nil.
func RunCommand(ctx context.Context, name string, args []string, dir string) error {
	return RunWithContext(ctx, name, args, dir)
}

// StartCommand starts a long-running command in the background and returns immediately.
// The command inherits stdout/stderr/stdin from the parent process.
// The command inherits all environment variables including azd context (AZD_SERVER, AZD_ACCESS_TOKEN, etc.).
// Use this for starting servers, Aspire projects, or other long-running processes.
// The provided context must be non-nil.
func StartCommand(ctx context.Context, name string, args []string, dir string) error {
	cmd, err := cmdutil.StartCommand(ctx, name, args, dir)
	if err != nil {
		return err
	}

	cliout.Newline()
	cliout.Success("Started %s (PID: %d)", name, cmd.Process.Pid)
	cliout.Item("Output will appear below. Press Ctrl+C to stop it when ready.")
	cliout.Newline()

	return nil
}

// RunCommandWithOutput executes a command and captures both stdout and stderr.
// The command inherits all environment variables from the parent process.
func RunCommandWithOutput(ctx context.Context, name string, args []string, dir string) ([]byte, error) {
	return cmdutil.RunCommandWithOutput(ctx, name, args, dir)
}

// OutputLineHandler is called for each line of output from a command.
// Unlike cmdutil.OutputLineHandler, this returns an error for app-level error propagation.
type OutputLineHandler func(line string) error

// StartCommandWithOutputMonitoring starts a command and monitors its output line-by-line.
// The handler function is called for each line of stdout/stderr.
// Output is still displayed to the user in real-time.
// The command inherits all environment variables including azd context.
// This function BLOCKS and waits for the command to complete or be interrupted.
// The provided context must be non-nil.
func StartCommandWithOutputMonitoring(ctx context.Context, name string, args []string, dir string, handler OutputLineHandler) error {
	// Adapt app's error-returning handler to cmdutil's simpler handler
	coreHandler := cmdutil.OutputLineHandler(func(line string) {
		if handler != nil {
			if err := handler(line); err != nil {
				slog.Warn("output handler error", "error", err)
			}
		}
	})

	cmd, err := cmdutil.StartCommandWithOutputMonitoring(ctx, name, args, dir, coreHandler)
	if err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	cliout.Newline()
	cliout.Success("Started %s (PID: %d)", name, cmd.Process.Pid)
	cliout.Item("Press Ctrl+C to stop.")
	cliout.Newline()

	// Wait for the command to complete (this blocks until the process exits or is killed)
	if err := cmd.Wait(); err != nil {
		// Ignore exit errors from Ctrl+C
		return nil
	}

	return nil
}
