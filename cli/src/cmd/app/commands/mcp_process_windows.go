//go:build windows

package commands

import (
	"os/exec"
	"syscall"
)

// setupProcessGroup configures the command to run in its own process group
// On Windows, this creates a new process group for proper cleanup
func setupProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// Create a new process group on Windows
	cmd.SysProcAttr.CreationFlags = syscall.CREATE_NEW_PROCESS_GROUP
}
