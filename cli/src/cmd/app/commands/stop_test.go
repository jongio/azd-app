package commands

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/runstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetStopFlags restores the package-level stop flag vars so tests do not leak
// state into one another.
func resetStopFlags(t *testing.T) {
	t.Helper()
	prevService, prevAll, prevYes := stopService, stopAll, stopYes
	t.Cleanup(func() {
		stopService, stopAll, stopYes = prevService, prevAll, prevYes
	})
	stopService, stopAll, stopYes = "", false, false
}

func TestNewStopCommandFlags(t *testing.T) {
	cmd := NewStopCommand()

	tests := []struct {
		name       string
		flag       string
		shorthand  string
		defaultVal string
	}{
		{name: "service flag", flag: "service", shorthand: "s", defaultVal: ""},
		{name: "all flag", flag: "all", shorthand: "", defaultVal: "false"},
		{name: "yes flag", flag: "yes", shorthand: "y", defaultVal: "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := cmd.Flags().Lookup(tt.flag)
			require.NotNil(t, f, "flag %q should be defined", tt.flag)
			assert.Equal(t, tt.shorthand, f.Shorthand)
			assert.Equal(t, tt.defaultVal, f.DefValue)
		})
	}
}

func TestRunStopServicesAllNoRunningServices(t *testing.T) {
	resetStopFlags(t)
	t.Chdir(t.TempDir())

	stopAll = true

	// With no registered/running services, --all should report that there is
	// nothing to stop and return without error rather than tearing down.
	err := runStopServices()
	assert.NoError(t, err)
}

func TestRunStopRoutesToServicesWhenServiceSet(t *testing.T) {
	resetStopFlags(t)
	t.Chdir(t.TempDir())

	// --all with an empty registry exercises the service-scoped path, which
	// must succeed (no-op) instead of attempting whole-app teardown.
	stopAll = true
	require.NoError(t, runStop(nil, nil))
}

// seedRunState writes a run state for a throwaway project and returns its dir.
func seedRunState(t *testing.T, pid int) string {
	t.Helper()
	projectDir := filepath.Join(t.TempDir(), "stop-project")
	statePath, err := runstate.Path(projectDir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(statePath)) })
	require.NoError(t, runstate.Write(projectDir, runstate.RunState{PID: pid, StartTime: time.Now()}))
	return projectDir
}

// A failed shutdown against a live manager must keep the state, otherwise the
// only record of the PID is lost and the manager becomes unreachable.
func TestClearRunStateIfManagerDeadKeepsStateForLiveProcess(t *testing.T) {
	projectDir := seedRunState(t, os.Getpid())

	clearRunStateIfManagerDead(projectDir)

	_, found, err := runstate.Read(projectDir)
	require.NoError(t, err)
	assert.True(t, found, "state for a live manager must survive a failed stop")
}

// A manager that crashed leaves state nothing else would clean up, so a failed
// stop should drop it rather than report a phantom run forever.
func TestClearRunStateIfManagerDeadRemovesStateForDeadProcess(t *testing.T) {
	projectDir := seedRunState(t, deadPIDForTest(t))

	clearRunStateIfManagerDead(projectDir)

	_, found, err := runstate.Read(projectDir)
	require.NoError(t, err)
	assert.False(t, found, "stale state from a crashed manager must be cleared")
}

func TestClearRunStateIfManagerDeadIgnoresMissingState(t *testing.T) {
	assert.NotPanics(t, func() {
		clearRunStateIfManagerDead(filepath.Join(t.TempDir(), "no-state"))
	})
}

// deadPIDForTest returns a PID that is not running.
func deadPIDForTest(t *testing.T) int {
	t.Helper()
	for pid := 300000; pid < 300100; pid++ {
		if !runstate.IsRunning(&runstate.RunState{PID: pid}) {
			return pid
		}
	}
	t.Skip("could not find a free PID to represent a dead manager")
	return 0
}
