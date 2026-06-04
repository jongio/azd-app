//go:build windows

package service

import (
	"fmt"
	"os"
)

// openLogFile opens a log file with symlink attack protection (CWE-59).
//
// Windows does not expose O_NOFOLLOW, so we use os.Lstat to inspect the
// path before opening. If the entry already exists and is a symbolic link
// we refuse to open it. This approach has a narrow TOCTOU window, but
// Windows symlinks require Developer Mode or elevated privileges to create,
// which significantly limits the practical attack surface compared to Unix.
func openLogFile(path string, flag int, perm os.FileMode) (*os.File, error) {
	fi, err := os.Lstat(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to stat log file path: %w", err)
	}
	if err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("log file path is a symbolic link, refusing to open (CWE-59): %s", path)
	}
	return os.OpenFile(path, flag, perm)
}
