//go:build !windows

package service

import (
	"log/slog"
	"os"
	"time"
)

// stopProcessGraceful attempts to stop a process gracefully on Unix systems.
// Sends SIGINT first, waits for timeout, then force kills if still running.
func stopProcessGraceful(process *os.Process, serviceName string, timeout time.Duration) error {
	slog.Debug("attempting graceful shutdown with SIGINT",
		slog.String("service", serviceName),
		slog.Int("pid", process.Pid))

	// Try graceful shutdown first with SIGINT
	if err := process.Signal(os.Interrupt); err != nil {
		slog.Debug("SIGINT failed, forcing kill",
			slog.String("service", serviceName),
			slog.String("error", err.Error()))
		// If signal fails (process already dead or doesn't support signals), try kill
		if killErr := process.Kill(); killErr != nil {
			slog.Debug("kill also failed, process may have already exited",
				slog.String("service", serviceName),
				slog.String("error", killErr.Error()))
			return nil
		}
		// Wait for process to exit
		_, _ = process.Wait()
		slog.Info("service stopped (forced)",
			slog.String("service", serviceName))
		return nil
	}

	// Wait for graceful shutdown with timeout
	done := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	select {
	case err := <-done:
		// Process exited within timeout
		slog.Info("service stopped gracefully",
			slog.String("service", serviceName))
		return err
	case <-time.After(timeout):
		// Timeout expired, force kill
		slog.Warn("graceful shutdown timeout, forcing kill",
			slog.String("service", serviceName),
			slog.Duration("timeout", timeout))
		if err := process.Kill(); err != nil {
			slog.Debug("kill failed, process may have already exited",
				slog.String("service", serviceName),
				slog.String("error", err.Error()))
			return nil
		}
		// Wait for kill to complete
		_, waitErr := process.Wait()
		slog.Info("service stopped (forced after timeout)",
			slog.String("service", serviceName))
		return waitErr
	}
}
