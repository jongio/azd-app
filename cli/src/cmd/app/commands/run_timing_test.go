package commands

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jongio/azd-app/cli/src/internal/config"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/startuptime"
)

func TestBuildRunTimingNilResult(t *testing.T) {
	_, ok := buildRunTiming(nil)
	assert.False(t, ok)
}

func TestBuildRunTimingNoReadyTime(t *testing.T) {
	result := &service.OrchestrationResult{
		Processes: map[string]*service.ServiceProcess{
			"api": {StartTime: time.Now()},
		},
	}
	_, ok := buildRunTiming(result)
	assert.False(t, ok, "a run with no ready time has nothing to record")
}

func TestBuildRunTimingComputesPerServiceDuration(t *testing.T) {
	start := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ready := start.Add(5 * time.Second)

	result := &service.OrchestrationResult{
		ReadyTime: ready,
		Processes: map[string]*service.ServiceProcess{
			"api":     {StartTime: start},
			"web":     {StartTime: start.Add(2 * time.Second)},
			"nostart": {}, // no start time -> skipped
			"nilproc": nil,
		},
	}

	current, ok := buildRunTiming(result)
	require.True(t, ok)
	assert.Equal(t, ready, current.Timestamp)
	assert.Equal(t, 5*time.Second, current.Services["api"])
	assert.Equal(t, 3*time.Second, current.Services["web"])
	assert.NotContains(t, current.Services, "nostart")
	assert.NotContains(t, current.Services, "nilproc")
}

func TestBuildRunTimingClampsNegativeToZero(t *testing.T) {
	ready := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	result := &service.OrchestrationResult{
		ReadyTime: ready,
		Processes: map[string]*service.ServiceProcess{
			// process reports a start after ready (clock skew): clamp to 0.
			"api": {StartTime: ready.Add(time.Second)},
		},
	}

	current, ok := buildRunTiming(result)
	require.True(t, ok)
	assert.Equal(t, time.Duration(0), current.Services["api"])
}

func TestRecordStartupTimingsPersistsHistory(t *testing.T) {
	// Point the config dir (and therefore timing history) at a temp location.
	tmp := t.TempDir()
	origGetConfigPath := config.GetConfigPath
	config.GetConfigPath = func() (string, error) {
		return filepath.Join(tmp, "config.json"), nil
	}
	t.Cleanup(func() { config.GetConfigPath = origGetConfigPath })

	origNoTiming := runNoTiming
	runNoTiming = false
	t.Cleanup(func() { runNoTiming = origNoTiming })

	projectDir := filepath.Join(tmp, "proj")
	start := time.Now()
	result := &service.OrchestrationResult{
		ReadyTime: start.Add(3 * time.Second),
		Processes: map[string]*service.ServiceProcess{
			"api": {StartTime: start},
		},
	}

	recordStartupTimings(projectDir, result)

	path := startuptime.HistoryPath(tmp, projectDir)
	history := startuptime.Load(path)
	require.Len(t, history.Runs, 1)
	assert.Equal(t, 3*time.Second, history.Runs[0].Services["api"])

	// A second run appends rather than replaces.
	recordStartupTimings(projectDir, result)
	assert.Len(t, startuptime.Load(path).Runs, 2)
}

func TestRecordStartupTimingsRespectsNoTiming(t *testing.T) {
	tmp := t.TempDir()
	origGetConfigPath := config.GetConfigPath
	config.GetConfigPath = func() (string, error) {
		return filepath.Join(tmp, "config.json"), nil
	}
	t.Cleanup(func() { config.GetConfigPath = origGetConfigPath })

	origNoTiming := runNoTiming
	runNoTiming = true
	t.Cleanup(func() { runNoTiming = origNoTiming })

	projectDir := filepath.Join(tmp, "proj")
	start := time.Now()
	result := &service.OrchestrationResult{
		ReadyTime: start.Add(3 * time.Second),
		Processes: map[string]*service.ServiceProcess{
			"api": {StartTime: start},
		},
	}

	recordStartupTimings(projectDir, result)

	path := startuptime.HistoryPath(tmp, projectDir)
	assert.Empty(t, startuptime.Load(path).Runs, "no-timing must not write history")
}
