package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/runstate"
	"github.com/jongio/azd-core/cliout"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withRunDetach(t *testing.T, v bool) {
	t.Helper()
	prev := runDetach
	runDetach = v
	t.Cleanup(func() { runDetach = prev })
}

func TestStripDetachFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "removes bare detach flag",
			args: []string{"run", "--detach", "--runtime", "azd"},
			want: []string{"run", "--runtime", "azd"},
		},
		{
			name: "removes key value detach flag",
			args: []string{"run", "--detach=true", "--service", "api"},
			want: []string{"run", "--service", "api"},
		},
		{
			name: "leaves args when no detach flag",
			args: []string{"run", "--runtime", "aspire"},
			want: []string{"run", "--runtime", "aspire"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, stripDetachFlag(tt.args))
		})
	}
}

func TestMaybeStartDetachedRunSkips(t *testing.T) {
	tests := []struct {
		name   string
		detach bool
		envVal string
	}{
		{name: "detach disabled", detach: false, envVal: ""},
		{name: "already running as detached child", detach: true, envVal: "1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withRunDetach(t, tt.detach)
			t.Setenv(detachedRunEnvVar, tt.envVal)

			result, detached, err := maybeStartDetachedRun(t.TempDir())
			require.NoError(t, err)
			assert.False(t, detached)
			assert.Nil(t, result)
		})
	}
}

func TestPrintDetachedStartResult(t *testing.T) {
	result := &detachedRunResult{Detached: true, PID: 4321, LogFile: filepath.Join("tmp", runLogFileName)}

	t.Run("json output", func(t *testing.T) {
		require.NoError(t, cliout.SetFormat("json"))
		t.Cleanup(func() { _ = cliout.SetFormat("default") })
		assert.NoError(t, printDetachedStartResult(result))
	})

	t.Run("text output", func(t *testing.T) {
		require.NoError(t, cliout.SetFormat("default"))
		assert.NoError(t, printDetachedStartResult(result))
	})
}

func TestMaybeStartDetachedRunSpawns(t *testing.T) {
	if os.Getenv(testDetachChildEnv) == "1" {
		t.Skip("running as the spawned detached child")
	}

	withRunDetach(t, true)
	// Ensure the parent is not itself treated as an already-detached child.
	t.Setenv(detachedRunEnvVar, "")
	// The spawned child inherits this sentinel and exits immediately via TestMain.
	t.Setenv(testDetachChildEnv, "1")

	projectDir := filepath.Join(t.TempDir(), "detach-project")
	statePath, err := runstate.Path(projectDir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(statePath)) })

	result, detached, err := maybeStartDetachedRun(projectDir)
	require.NoError(t, err)
	require.True(t, detached)
	require.NotNil(t, result)
	assert.True(t, result.Detached)
	assert.Greater(t, result.PID, 0)
	assert.Equal(t, filepath.Join(filepath.Dir(statePath), runLogFileName), result.LogFile)

	// The log file is created by the parent before spawning, so it must exist.
	_, statErr := os.Stat(result.LogFile)
	assert.NoError(t, statErr)

	// The child exits on its own; release/kill defensively so nothing leaks.
	if proc, ferr := os.FindProcess(result.PID); ferr == nil {
		_ = proc.Kill()
		_ = proc.Release()
	}
}
