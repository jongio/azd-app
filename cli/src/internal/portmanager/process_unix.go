//go:build !windows

package portmanager

import "os/exec"

// hideConsoleWindow is a no-op on non-Windows platforms.
func hideConsoleWindow(_ *exec.Cmd) {}
