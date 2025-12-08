//go:build integration && !windows

package portmanager

import (
	"os"
	"os/exec"
	"syscall"
)

// setProcessGroupImpl sets process group for clean termination on Unix.
func setProcessGroupImpl(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}

// syscallSignal0Impl returns signal 0 for process existence check on Unix.
func syscallSignal0Impl() os.Signal {
	return syscall.Signal(0)
}
