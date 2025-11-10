package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/executor"
)

// StartService starts a service and returns the process handle.
func StartService(runtime *ServiceRuntime, env map[string]string, projectDir string) (*ServiceProcess, error) {
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

	// Apply debug flags if debugging is enabled
	if err := ApplyDebugFlags(runtime, cmd); err != nil {
		return nil, fmt.Errorf("failed to apply debug flags: %w", err)
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
	StartLogCollection(process, projectDir)

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
	return StopServiceGraceful(process, 5*time.Second)
}

// StopServiceGraceful stops a service with graceful shutdown timeout.
// On Unix: Sends SIGINT, waits for timeout, then force kills if still running.
// On Windows: Immediately kills the process (graceful signals don't work reliably).
// Returns nil if process stops successfully within timeout.
func StopServiceGraceful(process *ServiceProcess, timeout time.Duration) error {
	if process == nil {
		return fmt.Errorf("process is nil")
	}
	if process.Process == nil {
		return fmt.Errorf("process not started")
	}

	slog.Info("stopping service",
		slog.String("service", process.Name),
		slog.Int("pid", process.Process.Pid),
		slog.Int("port", process.Port),
		slog.Duration("timeout", timeout))

	// Use platform-specific graceful shutdown
	return stopProcessGraceful(process.Process, process.Name, timeout)
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
	ctx, cancel := context.WithTimeout(context.Background(), executor.DefaultTimeout)
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
func StartLogCollection(process *ServiceProcess, projectDir string) {
	// Get or create log manager for this project
	logManager := GetLogManager(projectDir)

	// Create log buffer for this service (1000 entries max, enable file logging)
	buffer, err := logManager.CreateBuffer(process.Name, 1000, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to create log buffer for %s: %v\n", process.Name, err)
		return
	}

	// Start goroutines to collect stdout and stderr
	go collectStreamLogs(process.Stdout, process.Name, buffer, false)
	go collectStreamLogs(process.Stderr, process.Name, buffer, true)
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
