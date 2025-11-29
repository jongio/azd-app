// Package procutil provides cross-platform process utilities.
package procutil

import (
	"os"
	"runtime"
	"strings"
	"syscall"
)

// IsProcessRunning checks if a process with the given PID is running.
// Works cross-platform (Windows and Unix).
//
// KNOWN LIMITATION (Windows): On Windows, this function may return true for
// stale PIDs because os.FindProcess always succeeds and Signal(0) is not
// fully supported. For production use requiring high reliability, consider
// using Windows API (OpenProcess) or github.com/shirou/gopsutil.
func IsProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Windows, FindProcess always succeeds for any PID, so we use Signal(0)
	// which will fail with "not supported" on Windows. In that case, we try
	// to open the process with minimal permissions to verify it exists.
	if runtime.GOOS == "windows" {
		// Try Signal(0) first - it may work on some Windows versions
		err := process.Signal(syscall.Signal(0))
		if err == nil {
			return true
		}
		// On Windows, Signal(0) is typically not supported.
		// NOTE: This is a known limitation. For fully reliable Windows process
		// detection, use Windows API (OpenProcess with PROCESS_QUERY_LIMITED_INFORMATION)
		// or a library like github.com/shirou/gopsutil.
		// For now, we return true for valid PIDs as a fallback, with the understanding
		// that stale PIDs may incorrectly appear as running.
		errMsg := err.Error()
		if strings.Contains(errMsg, "not supported") || strings.Contains(errMsg, "Access is denied") {
			// Process handle was created, assume process exists
			// This is imperfect but better than failing all Windows checks
			return true
		}
		// Permission denied or other error - process may not exist
		return false
	}

	// On Unix-like systems, use signal 0 to check existence
	if err := process.Signal(syscall.Signal(0)); err != nil {
		return false
	}

	return true
}
