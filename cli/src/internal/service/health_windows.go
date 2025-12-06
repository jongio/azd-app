//go:build windows

package service

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// processIsRunning checks if a process with the given PID is running.
// On Windows, we use OpenProcess to check if the process exists and is running.
func processIsRunning(pid int) error {
	// Find the process first
	p, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process %d not found: %w", pid, err)
	}

	// On Windows, FindProcess always succeeds, so we need to actually check
	// if the process is still running by calling GetExitCodeProcess

	// Open the process with limited rights just to query its status
	const processQueryLimitedInformation = 0x1000
	handle, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(p.Pid))
	if err != nil {
		return fmt.Errorf("process %d not accessible: %w", pid, err)
	}
	defer syscall.CloseHandle(handle) //nolint:errcheck

	// Check if process is still running using GetExitCodeProcess
	var exitCode uint32
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getExitCodeProcess := kernel32.NewProc("GetExitCodeProcess")

	ret, _, err := getExitCodeProcess.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&exitCode)),
	)
	if ret == 0 {
		return fmt.Errorf("failed to get exit code for process %d: %w", pid, err)
	}

	// stillActive (259/0x103) means the process is still running
	const stillActive = 259
	if exitCode != stillActive {
		return fmt.Errorf("process %d has exited with code %d", pid, exitCode)
	}

	return nil
}
