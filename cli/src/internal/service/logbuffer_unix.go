//go:build !windows

package service

import (
	"fmt"
	"os"
	"syscall"
)

// openLogFile opens a log file with symlink attack protection (CWE-59).
//
// On Unix systems, O_NOFOLLOW is added to the open flags so the kernel
// refuses to follow a symbolic link at the final path component. The
// refusal is atomic - there is no TOCTOU race between checking for a
// symlink and opening the file. The kernel returns ELOOP when the final
// component is a symlink.
func openLogFile(path string, flag int, perm os.FileMode) (*os.File, error) {
	f, err := os.OpenFile(path, flag|syscall.O_NOFOLLOW, perm)
	if err != nil {
		// O_NOFOLLOW causes ELOOP when the target final component is a symlink.
		if pathErr, ok := err.(*os.PathError); ok {
			if errno, ok := pathErr.Err.(syscall.Errno); ok && errno == syscall.ELOOP {
				return nil, fmt.Errorf("log file path is a symbolic link, refusing to open (CWE-59): %s", path)
			}
		}
		return nil, err
	}
	return f, nil
}
