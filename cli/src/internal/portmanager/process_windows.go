//go:build windows

package portmanager

import (
	"os/exec"
	"syscall"
)

// createNoWindow prevents Windows from allocating a new console window for a
// console subprocess when the parent runs detached or without a console.
const createNoWindow = 0x08000000

// hideConsoleWindow configures cmd so Windows does not create or flash a
// visible console window when spawning subprocesses (e.g., powershell, taskkill)
// from a background or detached process.
func hideConsoleWindow(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags |= createNoWindow
}
