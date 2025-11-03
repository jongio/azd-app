package service

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/executor"
)

// StartService starts a service and returns the process handle.
func StartService(runtime *ServiceRuntime, env map[string]string, projectDir string) (*ServiceProcess, error) {
	process := &ServiceProcess{
		Name:    runtime.Name,
		Runtime: *runtime,
		Ready:   false,
	}

	// Build command
	if runtime.Command == "" {
		return nil, fmt.Errorf("no command specified for service %s", runtime.Name)
	}

	// Create command
	args := runtime.Args
	// #nosec G204 -- Command and args come from azure.yaml service configuration, validated by service package
	cmd := exec.Command(runtime.Command, args...)
	cmd.Dir = runtime.WorkingDir

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Create pipes for stdout and stderr
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start service %s: %w", runtime.Name, err)
	}

	process.Process = cmd.Process
	process.Stdout = stdoutPipe
	process.Stderr = stderrPipe
	process.Port = runtime.Port

	// Start log collection
	StartLogCollection(process, projectDir)

	return process, nil
}

// StopService stops a running service.
func StopService(process *ServiceProcess) error {
	if process.Process == nil {
		return fmt.Errorf("process not started")
	}

	// Try graceful shutdown first
	if err := process.Process.Signal(os.Interrupt); err != nil {
		// If interrupt fails, force kill
		if killErr := process.Process.Kill(); killErr != nil {
			return fmt.Errorf("failed to kill process: %w", killErr)
		}
	}

	// Wait for process to exit
	_, err := process.Process.Wait()
	return err
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
	return executor.RunCommand(name, args, dir)
}

// ValidateRuntime validates that a service runtime is properly configured.
func ValidateRuntime(runtime *ServiceRuntime) error {
	if runtime.Name == "" {
		return fmt.Errorf("service name is required")
	}

	if runtime.WorkingDir == "" {
		return fmt.Errorf("working directory is required for service %s", runtime.Name)
	}

	if runtime.Command == "" {
		return fmt.Errorf("run command is required for service %s", runtime.Name)
	}

	if runtime.Language == "" {
		return fmt.Errorf("language is required for service %s", runtime.Name)
	}

	return nil
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
