package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/runstate"
	"github.com/stretchr/testify/require"
)

// TestRunStopAppNoRunningApp exercises the whole-app stop path when nothing is
// running: findProjectDir succeeds (the deferred run-state cleanup is armed),
// but no dashboard port can be discovered, so runStopApp returns an error.
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
