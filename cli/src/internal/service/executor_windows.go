package service

import (
	"log/slog"
	"os"
	"time"
)

// stopProcessGraceful attempts to stop a process gracefully on Windows.
// On Windows, os.Signal doesn't work reliably for child processes, so we:
// 1. Try Kill() immediately (which terminates the process tree on Windows)
// 2. Wait for the process to exit
func stopProcessGraceful(process *os.Process, serviceName string, timeout time.Duration) error {
	// On Windows, there's no reliable way to send graceful shutdown signals to child processes.
	// Process.Signal(os.Interrupt) doesn't work for processes that aren't in the same console group.
	// The most reliable approach is to use Kill() which terminates the entire process tree.
	
	slog.Debug("attempting to stop process on Windows",
		slog.String("service", serviceName),
		slog.Int("pid", process.Pid))

	if err := process.Kill(); err != nil {
		// On Windows, "invalid argument" often means the process already exited
		slog.Debug("kill failed, process may have already exited",
			slog.String("service", serviceName),
			slog.String("error", err.Error()))
		return nil
	}

	// Wait for process to exit with timeout
	done := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	select {
	case err := <-done:
		slog.Info("service stopped",
			slog.String("service", serviceName))
		return err
	case <-time.After(timeout):
		// This should rarely happen since Kill() is forceful
		slog.Warn("process did not exit after kill within timeout",
			slog.String("service", serviceName),
			slog.Duration("timeout", timeout))
		return nil
	}
}
