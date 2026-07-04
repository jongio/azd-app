package commands

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/dashboard"
	"github.com/jongio/azd-app/cli/src/internal/runstate"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRunStateServices(t *testing.T) {
	t.Run("sorts, skips nil, and derives url from port", func(t *testing.T) {
		processes := map[string]*service.ServiceProcess{
			"web":     {Port: 3000},                               // URL derived from port
			"api":     {Port: 8080, URL: "http://localhost:8080"}, // explicit URL preserved
			"nilproc": nil,                                        // skipped
			"bare":    {},                                         // no port, no URL
		}

		got := buildRunStateServices(processes)
		require.Len(t, got, 3) // nilproc skipped

		// Results are sorted by name: api, bare, web.
		assert.Equal(t, "api", got[0].Name)
		assert.Equal(t, "bare", got[1].Name)
		assert.Equal(t, "web", got[2].Name)

		assert.Equal(t, "http://localhost:8080", got[0].URL)
		assert.Equal(t, "", got[1].URL)                      // bare stays empty
		assert.Equal(t, "http://localhost:3000", got[2].URL) // derived
		assert.Equal(t, 3000, got[2].Port)
	})

	t.Run("empty map yields empty slice", func(t *testing.T) {
		got := buildRunStateServices(map[string]*service.ServiceProcess{})
		assert.Empty(t, got)
	})
}

func TestWaitForDashboardURL(t *testing.T) {
	t.Run("returns empty after timeout with nil server and no port file", func(t *testing.T) {
		projectDir := t.TempDir()

		start := time.Now()
		url := waitForDashboardURL(projectDir, nil, 30*time.Millisecond)
		assert.Equal(t, "", url)
		assert.GreaterOrEqual(t, time.Since(start), 30*time.Millisecond)
	})

	t.Run("returns empty for a not-started server", func(t *testing.T) {
		projectDir := t.TempDir()
		server := dashboard.GetServer(projectDir) // GetURL() == "" until started

		url := waitForDashboardURL(projectDir, server, 30*time.Millisecond)
		assert.Equal(t, "", url)
	})
}

func TestWriteRunState(t *testing.T) {
	projectDir := t.TempDir()
	t.Cleanup(func() {
		_ = runstate.Remove(projectDir)
		if p, err := runstate.Path(projectDir); err == nil {
			_ = os.RemoveAll(filepath.Dir(p))
		}
	})

	result := &service.OrchestrationResult{
		Processes: map[string]*service.ServiceProcess{
			"api": {Port: 8080, URL: "http://localhost:8080"},
		},
		// Zero StartTime forces writeRunState to fall back to time.Now().
	}

	// A nil dashboard server plus no port file means waitForDashboardURL returns
	// "" after its internal timeout, so DashboardURL is persisted empty.
	writeRunState(projectDir, result, nil)

	st, ok, err := runstate.Read(projectDir)
	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, st)
	assert.Equal(t, os.Getpid(), st.PID)
	assert.False(t, st.StartTime.IsZero())
	assert.Equal(t, "", st.DashboardURL)
	require.Len(t, st.Services, 1)
	assert.Equal(t, "api", st.Services[0].Name)
	assert.Equal(t, "http://localhost:8080", st.Services[0].URL)
}
