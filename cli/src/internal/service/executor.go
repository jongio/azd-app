package service

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/jongio/azd-app/cli/src/internal/executor"
)

// StartService starts a service and returns the process handle.
func StartService(runtime *ServiceRuntime, env map[string]string) (*ServiceProcess, error) {
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
