package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-app/cli/src/internal/executor"
)

// StartService starts a service and returns the process handle.
func StartService(runtime *ServiceRuntime, env map[string]string, projectDir string, parser *FunctionsOutputParser) (*ServiceProcess, error) {
	if runtime.Command == "" {
		return nil, fmt.Errorf("no command specified for service %s", runtime.Name)
	}

	process := &ServiceProcess{
		Name:    runtime.Name,
		Runtime: *runtime,
		Ready:   false,
	}

	cmd, err := createServiceCommand(runtime, env)
	if err != nil {
		return nil, err
	}

	if err := setupProcessPipes(cmd, process); err != nil {
		return nil, err
	}

	// Start process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start service %s: %w", runtime.Name, err)
	}

	process.Process = cmd.Process
	process.Port = runtime.Port

	// Start log collection
	StartLogCollection(process, projectDir, parser)

	return process, nil
}

// createServiceCommand creates an exec.Cmd for the service.
func createServiceCommand(runtime *ServiceRuntime, env map[string]string) (*exec.Cmd, error) {
	// #nosec G204 -- Command and args come from azure.yaml service configuration, validated by service package
	cmd := exec.Command(runtime.Command, runtime.Args...)
	cmd.Dir = runtime.WorkingDir

	// Build environment variable list ensuring azd context is preserved.
	// Start with os.Environ() which includes all azd context variables
	// (AZD_SERVER, AZD_ACCESS_TOKEN, AZURE_*, etc.) inherited from the parent process.
	// The 'env' map parameter contains service-specific and merged environment variables
	// that should override the base environment when there are conflicts.

	// Convert env map to slice format for exec.Cmd
	envSlice := make([]string, 0, len(env))
	for key, value := range env {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", key, value))
	}

	// The env map already includes os.Environ() variables through ResolveEnvironment,
	// so we can use it directly. This ensures azd context variables are preserved.
	cmd.Env = envSlice

	return cmd, nil
}

// setupProcessPipes creates and attaches stdout/stderr pipes to the process.
func setupProcessPipes(cmd *exec.Cmd, process *ServiceProcess) error {
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	process.Stdout = stdoutPipe
	process.Stderr = stderrPipe

	return nil
}

// StopService stops a running service by sending termination signals.
// Deprecated: Use StopServiceGraceful for better timeout control.
func StopService(process *ServiceProcess) error {
	return StopServiceGraceful(process, DefaultStopTimeout)
}

// StopServiceGraceful stops a service with graceful shutdown timeout.
// Sends SIGINT, waits for timeout, then force kills if still running.
// Returns nil if process stops successfully within timeout.
// Note: The dashboard service is protected and will never be killed.
func StopServiceGraceful(process *ServiceProcess, timeout time.Duration) error {
	if process == nil {
		return fmt.Errorf("process is nil")
	}
	if process.Process == nil {
		return fmt.Errorf("process not started")
	}

	// Never kill the dashboard process - it must remain running to manage other services
	if process.Name == constants.DashboardServiceName {
		slog.Debug("skipping stop for dashboard service - dashboard is protected",
			slog.String("service", process.Name))
		return nil
	}

	slog.Info("stopping service",
		slog.String("service", process.Name),
		slog.Int("pid", process.Process.Pid),
		slog.Int("port", process.Port),
		slog.Duration("timeout", timeout))

	// On Windows, we use taskkill with /T flag to kill the entire process tree.
	// This is critical because services often spawn child processes (e.g., npm -> node,
	// electron -> node) that hold ports. Using process.Kill() only kills the parent,
	// leaving child processes running and holding ports, causing port conflicts on restart.
	if runtime.GOOS == "windows" {
		slog.Debug("using taskkill /T for Windows process tree termination",
			slog.String("service", process.Name),
			slog.Int("pid", process.Process.Pid))

		// Use taskkill /F /T to force kill entire process tree
		// /F = Force termination
		// /T = Kill child processes (tree kill)
		// /PID = Target process ID
		// #nosec G204 -- PID is from os.Process which is a validated integer
		cmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", process.Process.Pid))
		if err := cmd.Run(); err != nil {
			// taskkill returns error if process already exited, which is fine
			slog.Debug("taskkill completed with error (process may have already exited)",
				slog.String("service", process.Name),
				slog.String("error", err.Error()))
		}

		// Wait for process to exit - this may fail if taskkill already cleaned up
		_, waitErr := process.Process.Wait()
		// Ignore "Access is denied" - process already exited on Windows
		if waitErr != nil && !isAccessDeniedError(waitErr) {
			slog.Debug("wait completed with error (expected if taskkill succeeded)",
				slog.String("error", waitErr.Error()))
		}
		slog.Info("service stopped",
			slog.String("service", process.Name))
		return nil
	}

	// On Unix/Linux/macOS, try graceful shutdown with SIGINT first
	if err := process.Process.Signal(os.Interrupt); err != nil {
		slog.Warn("graceful shutdown signal failed, forcing kill",
			slog.String("service", process.Name),
			slog.String("error", err.Error()))
		// If signal fails (process already dead or doesn't support signals), try kill
		if killErr := process.Process.Kill(); killErr != nil {
			return fmt.Errorf("failed to kill process: %w", killErr)
		}
		// Wait for process to exit
		_, _ = process.Process.Wait()
		slog.Info("service stopped (forced)",
			slog.String("service", process.Name))
		return nil
	}

	// Wait for graceful shutdown with timeout
	done := make(chan error, 1)
	go func() {
		_, err := process.Process.Wait()
		done <- err
	}()

	select {
	case err := <-done:
		// Process exited within timeout
		slog.Info("service stopped gracefully",
			slog.String("service", process.Name))
		return err
	case <-time.After(timeout):
		// Timeout expired, force kill
		slog.Warn("graceful shutdown timeout, forcing kill",
			slog.String("service", process.Name),
			slog.Duration("timeout", timeout))
		if err := process.Process.Kill(); err != nil {
			return fmt.Errorf("failed to force kill process after timeout: %w", err)
		}
		// Wait for kill to complete
		_, waitErr := process.Process.Wait()
		slog.Info("service stopped (forced after timeout)",
			slog.String("service", process.Name))
		return waitErr
	}
}

// ReadServiceOutput reads and forwards output from a service.
func ReadServiceOutput(reader io.Reader, outputChan chan<- string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		outputChan <- scanner.Text()
	}
}

// ExecuteCommand executes a command using the executor package.
func ExecuteCommand(name string, args []string, dir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCommandTimeout)
	defer cancel()
	return executor.RunCommand(ctx, name, args, dir)
}

// ValidateRuntime validates that a service runtime is properly configured.
func ValidateRuntime(runtime *ServiceRuntime) error {
	switch {
	case runtime.Name == "":
		return fmt.Errorf("service name is required")
	case runtime.WorkingDir == "":
		return fmt.Errorf("working directory is required for service %s", runtime.Name)
	case runtime.Command == "":
		return fmt.Errorf("run command is required for service %s", runtime.Name)
	case runtime.Language == "":
		return fmt.Errorf("language is required for service %s", runtime.Name)
	default:
		return nil
	}
}

// GetProcessStatus returns the status of a service process.
func GetProcessStatus(process *ServiceProcess) string {
	if process.Process == nil {
		return "not-started"
	}

	// Check if process is still running
	err := process.Process.Signal(nil)
	if err != nil {
		return "stopped"
	}

	if process.Ready {
		return "ready"
	}

	return "starting"
}

// StartLogCollection starts collecting logs from a service process.
func StartLogCollection(process *ServiceProcess, projectDir string, parser *FunctionsOutputParser) {
	// Get or create log manager for this project
	logManager := GetLogManager(projectDir)

	// Create log buffer for this service (1000 entries max, enable file logging)
	buffer, err := logManager.CreateBuffer(process.Name, 1000, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to create log buffer for %s: %v\n", process.Name, err)
		return
	}

	// Check if this is a Functions service
	isFunctionsService := strings.Contains(process.Runtime.Framework, "Functions") ||
		strings.Contains(process.Runtime.Framework, "Logic Apps")

	// Start goroutines to collect stdout and stderr
	if isFunctionsService && parser != nil {
		go collectFunctionsStreamLogs(process.Stdout, process.Name, buffer, parser, false)
		go collectFunctionsStreamLogs(process.Stderr, process.Name, buffer, parser, true)
	} else {
		go collectStreamLogs(process.Stdout, process.Name, buffer, false)
		go collectStreamLogs(process.Stderr, process.Name, buffer, true)
	}
}

// collectStreamLogs reads from a stream and adds entries to the log buffer.
func collectStreamLogs(reader io.ReadCloser, serviceName string, buffer *LogBuffer, isStderr bool) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		entry := LogEntry{
			Service:   serviceName,
			Message:   scanner.Text(),
			Timestamp: time.Now(),
			IsStderr:  isStderr,
			Level:     inferLogLevel(scanner.Text()),
		}
		buffer.Add(entry)
	}
}

// collectFunctionsStreamLogs reads from a stream, adds entries to the log buffer, and parses Functions output.
func collectFunctionsStreamLogs(reader io.ReadCloser, serviceName string, buffer *LogBuffer, parser *FunctionsOutputParser, isStderr bool) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		// Add to log buffer
		entry := LogEntry{
			Service:   serviceName,
			Message:   line,
			Timestamp: time.Now(),
			IsStderr:  isStderr,
			Level:     inferLogLevel(line),
		}
		buffer.Add(entry)

		// Also parse for function endpoints
		parser.ParseLine(serviceName, line)
	}
}

// isAccessDeniedError checks if an error is a Windows "Access is denied" error.
// This error occurs when Wait() is called after Kill() on Windows because the
// process has already exited and its handle is no longer valid.
func isAccessDeniedError(err error) bool {
	if err == nil {
		return false
	}
	// Check for Windows-specific "Access is denied" error message
	// Use string matching as Windows syscall errors don't implement errors.Is
	if runtime.GOOS == "windows" {
		errStr := err.Error()
		return strings.Contains(errStr, "Access is denied") ||
			strings.Contains(errStr, "TerminateProcess: Access is denied")
	}
	return false
}
