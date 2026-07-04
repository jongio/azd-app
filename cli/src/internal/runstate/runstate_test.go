package runstate

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/azdconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withBaseDir(t *testing.T, dir string) {
	t.Helper()
	prev := baseDir
	baseDir = dir
	t.Cleanup(func() {
		baseDir = prev
	})
}

func TestWriteReadRoundTrip(t *testing.T) {
	withBaseDir(t, t.TempDir())

	projectDir := filepath.Join(t.TempDir(), "project")
	startedAt := time.Now().UTC().Round(time.Second)
	expected := RunState{
		PID:          os.Getpid(),
		DashboardURL: "http://localhost:4321",
		StartTime:    startedAt,
		Services: []ServiceState{
			{Name: "api", URL: "http://localhost:8080", Port: 8080},
			{Name: "web", URL: "http://localhost:3000", Port: 3000},
		},
	}

	require.NoError(t, Write(projectDir, expected))

	actual, ok, err := Read(projectDir)
	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, actual)
	assert.Equal(t, expected, *actual)
}

func TestReadMissingReturnsFalse(t *testing.T) {
	withBaseDir(t, t.TempDir())

	projectDir := filepath.Join(t.TempDir(), "missing-project")
	actual, ok, err := Read(projectDir)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, actual)
}

func TestRemoveIsIdempotent(t *testing.T) {
	withBaseDir(t, t.TempDir())

	projectDir := filepath.Join(t.TempDir(), "remove-project")
	require.NoError(t, Remove(projectDir))

	state := RunState{
		PID:          os.Getpid(),
		DashboardURL: "http://localhost:4321",
		StartTime:    time.Now().UTC().Round(time.Second),
	}
	require.NoError(t, Write(projectDir, state))
	require.NoError(t, Remove(projectDir))
	require.NoError(t, Remove(projectDir))

	actual, ok, err := Read(projectDir)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, actual)
}

func TestIsRunning(t *testing.T) {
	tests := []struct {
		name string
		st   *RunState
		want bool
	}{
		{
			name: "current process is running",
			st:   &RunState{PID: os.Getpid()},
			want: true,
		},
		{
			name: "invalid pid is not running",
			st:   &RunState{PID: 2_147_483_646},
			want: false,
		},
		{
			name: "nil state is not running",
			st:   nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsRunning(tt.st))
		})
	}
}

func TestPathDefaultsToHomeDir(t *testing.T) {
	// An empty baseDir must fall back to ~/.azd/azd-app.
	withBaseDir(t, "")

	projectDir := filepath.Join(t.TempDir(), "project")
	path, err := Path(projectDir)
	require.NoError(t, err)

	home, err := os.UserHomeDir()
	require.NoError(t, err)
	want := filepath.Join(home, ".azd", "azd-app", azdconfig.ProjectHash(projectDir), "run.json")
	assert.Equal(t, want, path)
}

func TestWriteErrors(t *testing.T) {
	t.Run("mkdir fails when parent path is a file", func(t *testing.T) {
		base := t.TempDir()
		withBaseDir(t, base)

		projectDir := filepath.Join(t.TempDir(), "project")
		// Occupy the per-project state directory location with a file so
		// MkdirAll cannot create the directory.
		require.NoError(t, os.WriteFile(filepath.Join(base, azdconfig.ProjectHash(projectDir)), []byte("x"), 0o600))

		err := Write(projectDir, RunState{PID: 1})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create run state directory")
	})

	t.Run("write fails when target path is a directory", func(t *testing.T) {
		withBaseDir(t, t.TempDir())

		projectDir := filepath.Join(t.TempDir(), "project")
		path, err := Path(projectDir)
		require.NoError(t, err)
		// Create run.json as a directory so WriteFile fails.
		require.NoError(t, os.MkdirAll(path, 0o700))

		err = Write(projectDir, RunState{PID: 1})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "write run state file")
	})
}

func TestReadErrors(t *testing.T) {
	t.Run("invalid json returns unmarshal error", func(t *testing.T) {
		withBaseDir(t, t.TempDir())

		projectDir := filepath.Join(t.TempDir(), "project")
		path, err := Path(projectDir)
		require.NoError(t, err)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte("{ not valid json"), 0o600))

		st, ok, err := Read(projectDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unmarshal run state file")
		assert.False(t, ok)
		assert.Nil(t, st)
	})

	t.Run("directory path returns read error", func(t *testing.T) {
		withBaseDir(t, t.TempDir())

		projectDir := filepath.Join(t.TempDir(), "project")
		path, err := Path(projectDir)
		require.NoError(t, err)
		// Create run.json as a directory so ReadFile fails with a non-NotExist error.
		require.NoError(t, os.MkdirAll(path, 0o700))

		st, ok, err := Read(projectDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "read run state file")
		assert.False(t, ok)
		assert.Nil(t, st)
	})
}

func TestRemoveError(t *testing.T) {
	withBaseDir(t, t.TempDir())

	projectDir := filepath.Join(t.TempDir(), "project")
	path, err := Path(projectDir)
	require.NoError(t, err)
	// A non-empty directory at the run.json path makes os.Remove fail with a
	// non-NotExist error.
	require.NoError(t, os.MkdirAll(filepath.Join(path, "child"), 0o700))

	err = Remove(projectDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remove run state file")
}
