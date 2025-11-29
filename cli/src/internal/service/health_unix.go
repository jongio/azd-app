//go:build !windows

package service

import (
	"fmt"
	"os"
	"syscall"
)

// processIsRunning checks if a process with the given PID is running.
// On Unix systems, we send signal 0 which doesn't affect the process
// but returns an error if the process doesn't exist.
func processIsRunning(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process %d not found: %w", pid, err)
	}

	// On Unix, Signal(0) checks if process exists without affecting it
	if err := p.Signal(syscall.Signal(0)); err != nil {
		return fmt.Errorf("process %d not running: %w", pid, err)
	}

	return nil
}
