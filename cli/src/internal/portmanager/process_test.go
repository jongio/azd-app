package portmanager

import (
	"context"
	"os/exec"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildGetProcessOnPortCommand(t *testing.T) {
	cmd, args := buildGetProcessOnPortCommand(8080)

	if runtime.GOOS == osWindows {
		assert.Equal(t, "powershell", cmd)
		require.GreaterOrEqual(t, len(args), 7)
		assert.Equal(t, []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command"}, args[:len(args)-1])
		script := args[len(args)-1]
		assert.Contains(t, script, ":8080 ")
		assert.Contains(t, script, "LISTENING")
		return
	}

	assert.Equal(t, "sh", cmd)
	require.Len(t, args, 2)
	assert.Equal(t, "-c", args[0])
	assert.Contains(t, args[1], "lsof -t -i:8080")
}

func TestBuildGetProcessNameCommand(t *testing.T) {
	cmd, args := buildGetProcessNameCommand(4321)

	if runtime.GOOS == osWindows {
		assert.Equal(t, "powershell", cmd)
		require.GreaterOrEqual(t, len(args), 7)
		assert.Equal(t, []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command"}, args[:len(args)-1])
		script := args[len(args)-1]
		assert.Contains(t, script, "Get-Process -Id 4321")
		return
	}

	assert.Equal(t, "sh", cmd)
	require.Len(t, args, 2)
	assert.Equal(t, "-c", args[0])
	assert.Contains(t, args[1], "ps -p 4321 -o comm=")
}

func TestBuildKillProcessCommand(t *testing.T) {
	cmd, args := buildKillProcessCommand(4321)

	if runtime.GOOS == osWindows {
		assert.Equal(t, "taskkill", cmd)
		assert.Equal(t, []string{"/F", "/T", "/PID", "4321"}, args)
		return
	}

	assert.Equal(t, "sh", cmd)
	require.Len(t, args, 2)
	assert.Equal(t, "-c", args[0])
	assert.Contains(t, args[1], "kill -9")
}

func TestNewProcessCommand(t *testing.T) {
	cmd, args := buildGetProcessOnPortCommand(8080)
	execCmd := newProcessCommand(context.Background(), cmd, args...)

	require.NotNil(t, execCmd)
	assert.Nil(t, execCmd.Stdin)
	if runtime.GOOS == osWindows {
		require.NotNil(t, execCmd.SysProcAttr)
	}
}

func TestHideConsoleWindow_NilSafe(t *testing.T) {
	assert.NotPanics(t, func() {
		hideConsoleWindow(nil)
	})

	cmd := exec.Command("echo", "test")
	assert.NotPanics(t, func() {
		hideConsoleWindow(cmd)
	})
	if runtime.GOOS == osWindows {
		require.NotNil(t, cmd.SysProcAttr)
	}
}
