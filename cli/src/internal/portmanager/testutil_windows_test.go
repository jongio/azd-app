//go:build integration && windows

package portmanager

import (
	"os"
	"os/exec"
)

// setProcessGroupImpl hides the console window on Windows.
// Windows doesn't use process groups the same way Unix does.
func setProcessGroupImpl(cmd *exec.Cmd) {
	hideConsoleWindow(cmd)
}

// syscallSignal0Impl returns nil on Windows as signal 0 is not used.
func syscallSignal0Impl() os.Signal {
	// Windows doesn't support signal 0, we use a different method to check process existence
	return nil
}
