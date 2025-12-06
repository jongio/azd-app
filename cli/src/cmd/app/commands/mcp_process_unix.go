//go:build !windows

package commands

import (
	"os/exec"
	"syscall"
)

// setupProcessGroup configures the command to run in its own process group
// This allows killing the entire process tree when cancelling
func setupProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// Create a new process group
	cmd.SysProcAttr.Setpgid = true
	// Prevent the process from receiving signals from the terminal
	cmd.SysProcAttr.Setsid = true
}
