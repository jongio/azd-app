package commands

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

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
		{
			name: "keeps positional args after the terminator verbatim",
			args: []string{"run", "--detach", "--", "--detach", "api"},
			want: []string{"run", "--", "--detach", "api"},
		},
		{
			name: "keeps a lone terminator",
			args: []string{"run", "--detach", "--"},
			want: []string{"run", "--"},
		},
		{
			name: "does not strip flags that merely start with the name",
			args: []string{"run", "--detach-timeout", "5"},
			want: []string{"run", "--detach-timeout", "5"},
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
			resetDetachedChildCache(t)

			result, detached, err := maybeStartDetachedRun(t.TempDir())
			require.NoError(t, err)
			assert.False(t, detached)
			assert.Nil(t, result)
		})
	}
}

// resetDetachedChildCache clears the one-time detached marker read so each test
// observes its own environment, and clears it again afterwards so a cached
// value cannot leak into unrelated tests.
func resetDetachedChildCache(t *testing.T) {
	t.Helper()
	detachedChildOnce = new(sync.Once)
	detachedChild = false
	t.Cleanup(func() {
		detachedChildOnce = new(sync.Once)
		detachedChild = false
	})
}

func TestIsDetachedChild(t *testing.T) {
	tests := []struct {
		name   string
		envVal string
		want   bool
	}{
		{name: "unset", envVal: "", want: false},
		{name: "detached child", envVal: "1", want: true},
		{name: "other value is not a detached child", envVal: "0", want: false},
		{name: "truthy string is not a detached child", envVal: "true", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(detachedRunEnvVar, tt.envVal)
			resetDetachedChildCache(t)
			assert.Equal(t, tt.want, IsDetachedChild())
		})
	}
}

// The marker must not survive in the environment, because services and
// lifecycle hooks inherit os.Environ(). A nested `azd app run` that still saw
// the marker would skip loading the azd environment it was given with -e.
func TestIsDetachedChildConsumesMarker(t *testing.T) {
	t.Setenv(detachedRunEnvVar, detachedRunEnvValue)
	resetDetachedChildCache(t)

	require.True(t, IsDetachedChild())
	assert.Empty(t, os.Getenv(detachedRunEnvVar), "marker must be removed so children do not inherit it")

	// Later callers still need the answer after the variable is gone.
	assert.True(t, IsDetachedChild(), "result must be cached once the marker is consumed")
}

// A foreground run must not touch the environment it was handed.
func TestIsDetachedChildLeavesForeignValueIntact(t *testing.T) {
	t.Setenv(detachedRunEnvVar, "0")
	resetDetachedChildCache(t)

	assert.False(t, IsDetachedChild())
	assert.Equal(t, "0", os.Getenv(detachedRunEnvVar))
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
	resetDetachedChildCache(t)
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

	// Run state must exist as soon as the parent returns so that `azd app
	// status` and `azd app stop` work during the child's startup window,
	// instead of only after every service and the dashboard are ready.
	state, found, readErr := runstate.Read(projectDir)
	require.NoError(t, readErr)
	require.True(t, found, "detached run state must be written before the parent returns")
	assert.Equal(t, result.PID, state.PID)
	assert.False(t, state.StartTime.IsZero())

	// The child exits on its own; release/kill defensively so nothing leaks.
	if proc, ferr := os.FindProcess(result.PID); ferr == nil {
		_ = proc.Kill()
		_ = proc.Release()
	}
}

func TestRecordDetachedRunState(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "record-project")
	statePath, err := runstate.Path(projectDir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(statePath)) })

	before := time.Now()
	recordDetachedRunState(projectDir, 4242)

	state, found, err := runstate.Read(projectDir)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, 4242, state.PID)
	assert.False(t, state.StartTime.Before(before.Truncate(time.Second)))
	// The child fills these in once it is ready.
	assert.Empty(t, state.DashboardURL)
	assert.Empty(t, state.Services)
}

func TestRecordDetachedRunStateSurvivesWriteFailure(t *testing.T) {
	// A run state write failure must never take down an already-running
	// background process, so this must return normally rather than panic.
	assert.NotPanics(t, func() {
		recordDetachedRunState(string([]byte{0}), 1)
	})
}

func TestSpawnWithAttempts(t *testing.T) {
	sentinel := errors.New("spawn failed")

	t.Run("returns the process from the first successful attempt", func(t *testing.T) {
		calls := 0
		want := &os.Process{Pid: 77}

		got, err := spawnWithAttempts(
			[]*syscall.SysProcAttr{{}, {}},
			func(*syscall.SysProcAttr) (*os.Process, error) {
				calls++
				return want, nil
			},
		)

		require.NoError(t, err)
		assert.Same(t, want, got)
		assert.Equal(t, 1, calls, "must not keep trying after success")
	})

	t.Run("stops on a failure that is not a rejected breakaway", func(t *testing.T) {
		calls := 0

		got, err := spawnWithAttempts(
			[]*syscall.SysProcAttr{{}, {}},
			func(*syscall.SysProcAttr) (*os.Process, error) {
				calls++
				return nil, sentinel
			},
		)

		require.Error(t, err)
		assert.Nil(t, got)
		assert.ErrorIs(t, err, sentinel)
		assert.Contains(t, err.Error(), "start detached run process")
		assert.Equal(t, 1, calls, "a retry would fail identically")
	})

	t.Run("reports a clear error when no attempts are configured", func(t *testing.T) {
		got, err := spawnWithAttempts(nil, func(*syscall.SysProcAttr) (*os.Process, error) {
			t.Fatal("start must not be called without attempts")
			return nil, nil
		})

		require.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "no spawn attempts configured")
	})

	t.Run("passes each attempt's attributes through to the starter", func(t *testing.T) {
		attempts := detachSpawnAttempts()
		var seen *syscall.SysProcAttr

		_, err := spawnWithAttempts(attempts, func(attr *syscall.SysProcAttr) (*os.Process, error) {
			seen = attr
			return &os.Process{Pid: 1}, nil
		})

		require.NoError(t, err)
		assert.Same(t, attempts[0], seen, "the preferred attempt must be tried first")
	})
}

// The child publishes richer state than the parent seed. If the child wins the
// race, the seed must not reduce it back to a bare PID.
func TestRecordDetachedRunStateDoesNotClobberChildState(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "clobber-project")
	statePath, err := runstate.Path(projectDir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(statePath)) })

	full := runstate.RunState{
		PID:          4242,
		DashboardURL: "http://localhost:5000",
		Services:     []runstate.ServiceState{{Name: "api", Port: 3000}},
		StartTime:    time.Now(),
	}
	require.NoError(t, runstate.Write(projectDir, full))

	recordDetachedRunState(projectDir, 4242)

	state, found, err := runstate.Read(projectDir)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "http://localhost:5000", state.DashboardURL, "child state must survive a late seed")
	assert.Len(t, state.Services, 1)
}

// State left by a previous run under a different PID is stale and must be
// replaced, otherwise a new detached run would inherit a dead dashboard URL.
func TestRecordDetachedRunStateReplacesStaleStateFromAnotherPID(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "stale-project")
	statePath, err := runstate.Path(projectDir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(statePath)) })

	require.NoError(t, runstate.Write(projectDir, runstate.RunState{
		PID:          1111,
		DashboardURL: "http://localhost:4999",
		StartTime:    time.Now(),
	}))

	recordDetachedRunState(projectDir, 2222)

	state, found, err := runstate.Read(projectDir)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, 2222, state.PID)
	assert.Empty(t, state.DashboardURL, "stale dashboard URL must not carry over")
}

// The child may publish services before the dashboard URL is known. Gating the
// seed on the PID alone keeps that partial-but-real state from being erased.
func TestRecordDetachedRunStateDoesNotClobberChildStateWithoutDashboard(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "no-dashboard-project")
	statePath, err := runstate.Path(projectDir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(statePath)) })

	require.NoError(t, runstate.Write(projectDir, runstate.RunState{
		PID:       7777,
		Services:  []runstate.ServiceState{{Name: "api", Port: 3000}},
		StartTime: time.Now(),
	}))

	recordDetachedRunState(projectDir, 7777)

	state, found, err := runstate.Read(projectDir)
	require.NoError(t, err)
	require.True(t, found)
	assert.Len(t, state.Services, 1, "child services must survive a late seed")
}
