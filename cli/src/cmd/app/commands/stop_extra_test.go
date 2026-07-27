package commands

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/runstate"
	"github.com/stretchr/testify/require"
)

// TestRunStopAppNoRunningApp exercises the whole-app stop path when nothing is
// running: findProjectDir succeeds, but no dashboard port can be discovered, so
// runStopApp returns an error.
func TestRunStopAppNoRunningApp(t *testing.T) {
	resetStopFlags(t)

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte("name: test\n"), 0o600))
	t.Chdir(dir)
	t.Cleanup(func() {
		_ = runstate.Remove(dir)
		if p, err := runstate.Path(dir); err == nil {
			_ = os.RemoveAll(filepath.Dir(p))
		}
	})

	err := runStopApp()
	require.Error(t, err)
}

// A crashed manager leaves run.json behind. The up-front cleanup in runStopApp
// is what frees it: stop cannot reach its normal success-path cleanup because
// dashboard discovery fails first. Without that up-front call the record would
// survive forever and keep `azd app status` reporting an app that is gone
// (#555). This pins the call-site placement, which the helper's own unit tests
// cannot do.
func TestRunStopAppClearsStateForDeadManager(t *testing.T) {
	resetStopFlags(t)

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte("name: test\n"), 0o600))
	t.Chdir(dir)

	statePath, err := runstate.Path(dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(statePath)) })
	require.NoError(t, runstate.Write(dir, runstate.RunState{PID: deadPIDForTest(t), StartTime: time.Now()}))

	// No dashboard is listening, so the stop itself still fails.
	require.Error(t, runStopApp())

	_, found, err := runstate.Read(dir)
	require.NoError(t, err)
	require.False(t, found, "a dead manager's run state must be cleared")
}
