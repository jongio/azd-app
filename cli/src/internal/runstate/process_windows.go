//go:build windows

package runstate

import (
	"syscall"
	"unsafe"
)

func pidAlive(pid int) bool {
	if pid <= 0 || pid > 0x7FFFFFFF {
		return false
	}

	const processQueryLimitedInformation = 0x1000
	handle, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(handle) //nolint:errcheck

	var exitCode uint32
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getExitCodeProcess := kernel32.NewProc("GetExitCodeProcess")

	ret, _, _ := getExitCodeProcess.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&exitCode)), //nolint:gosec // Required to query process state.
	)
	if ret == 0 {
		return false
	}

	const stillActive = 259
	return exitCode == stillActive
}
