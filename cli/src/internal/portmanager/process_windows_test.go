//go:build windows

package portmanager

import (
	"context"
	"os/exec"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHideConsoleWindow_Windows(t *testing.T) {
	t.Run("initializes SysProcAttr and sets HideWindow and CREATE_NO_WINDOW", func(t *testing.T) {
		cmd := exec.Command("powershell")
		require.Nil(t, cmd.SysProcAttr)

		hideConsoleWindow(cmd)

		require.NotNil(t, cmd.SysProcAttr)
		assert.True(t, cmd.SysProcAttr.HideWindow)
		assert.Equal(t, uint32(createNoWindow), cmd.SysProcAttr.CreationFlags&createNoWindow)
	})

	t.Run("preserves existing SysProcAttr fields", func(t *testing.T) {
		cmd := exec.Command("taskkill")
		cmd.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
		}

		hideConsoleWindow(cmd)

		require.NotNil(t, cmd.SysProcAttr)
		assert.True(t, cmd.SysProcAttr.HideWindow)
		assert.Equal(t, uint32(syscall.CREATE_NEW_PROCESS_GROUP|createNoWindow), cmd.SysProcAttr.CreationFlags)
	})

	t.Run("newProcessCommand configures hidden console window", func(t *testing.T) {
		cmd, args := buildGetProcessOnPortCommand(8080)
		execCmd := newProcessCommand(context.Background(), cmd, args...)

		require.NotNil(t, execCmd.SysProcAttr)
		assert.True(t, execCmd.SysProcAttr.HideWindow)
		assert.Equal(t, uint32(createNoWindow), execCmd.SysProcAttr.CreationFlags&createNoWindow)
	})
}
