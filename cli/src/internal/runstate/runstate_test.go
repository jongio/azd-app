package runstate

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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
