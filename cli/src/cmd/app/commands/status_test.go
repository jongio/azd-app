package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-app/cli/src/internal/runstate"
	"github.com/jongio/azd-core/cliout"
	"github.com/jongio/azd-core/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// restoreStatusFlags snapshots the package-level vars that NewStatusCommand
// binds its flags to and restores them when the test ends. Constructing or
// executing the status command mutates these globals, which otherwise leaks
// into every subsequent test in this binary.
func restoreStatusFlags(t *testing.T) {
	t.Helper()
	watch, interval, service := statusWatch, statusInterval, statusService
	t.Cleanup(func() {
		statusWatch, statusInterval, statusService = watch, interval, service
	})
}

func TestNewStatusCommand(t *testing.T) {
	restoreStatusFlags(t)

	cmd := NewStatusCommand()
	require.NotNil(t, cmd)
	assert.Equal(t, "status", cmd.Use)
	assert.Equal(t, "Show whether azd app run is active", cmd.Short)
	require.NotNil(t, cmd.RunE)
	assert.NotNil(t, cmd.Flags().Lookup("watch"), "expected --watch flag")
	assert.NotNil(t, cmd.Flags().Lookup("interval"), "expected --interval flag")
	assert.NotNil(t, cmd.Flags().Lookup("service"), "expected --service flag")
}

func TestRenderStatusReport(t *testing.T) {
	var buf bytes.Buffer
	report := statusReport{Running: true, PID: 4321, DashboardURL: "http://localhost:40000"}
	renderStatusReport(&buf, report, time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC))
	out := buf.String()
	// Each frame clears the screen and prints a refresh header.
	assert.Contains(t, out, "\033[H\033[2J")
	assert.Contains(t, out, "refreshed 15:04:05")
	assert.Contains(t, out, "App is running")
	assert.Contains(t, out, "PID: 4321")
}

func TestWatchStatusStopsOnContextCancel(t *testing.T) {
	projectDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already canceled: render one frame, then return via ctx.Done()

	var buf bytes.Buffer
	err := watchStatus(ctx, &buf, projectDir, time.Second, nil)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "App is not running")
	assert.Contains(t, out, "Stopped watching.")
}

func TestWatchStatusAppliesServiceFilter(t *testing.T) {
	projectDir := t.TempDir()
	t.Cleanup(func() {
		_ = runstate.Remove(projectDir)
	})

	require.NoError(t, runstate.Write(projectDir, runstate.RunState{
		PID:       os.Getpid(),
		StartTime: time.Now().UTC().Round(time.Second),
		Services: []runstate.ServiceState{
			{Name: "api", URL: "http://localhost:8080", Port: 8080},
			{Name: "web", URL: "http://localhost:3000", Port: 3000},
		},
	}))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	err := watchStatus(ctx, &buf, projectDir, time.Second, []string{"api"})
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "api: http://localhost:8080")
	assert.NotContains(t, out, "web: http://localhost:3000")
}

func TestRunStatusWatchIntervalTooSmall(t *testing.T) {
	_ = cliout.SetFormat("default")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte("name: statustest\nservices: {}\n"), 0o600))
	t.Chdir(dir)

	// NewStatusCommand binds its flags to package-level vars, so executing it
	// here leaves statusWatch/statusInterval set for every later test in this
	// binary. Restore them so repeated runs (go test -count=2) stay isolated.
	restoreStatusFlags(t)

	cmd := NewStatusCommand()
	cmd.SetArgs([]string{"--watch", "--interval", "100ms"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--interval must be at least")
}

func TestBuildStatusReportRunning(t *testing.T) {
	projectDir := t.TempDir()
	t.Cleanup(func() {
		_ = runstate.Remove(projectDir)
	})

	startedAt := time.Now().UTC().Round(time.Second)
	require.NoError(t, runstate.Write(projectDir, runstate.RunState{
		PID:          os.Getpid(),
		DashboardURL: "http://localhost:40000",
		StartTime:    startedAt,
		Services: []runstate.ServiceState{
			{Name: "fallback", URL: "http://localhost:9000", Port: 9000},
		},
	}))

	reg := registry.GetRegistry(projectDir)
	require.NoError(t, reg.Register(&registry.ServiceRegistryEntry{
		Name:   "api",
		Status: constants.StatusRunning,
		Port:   8080,
		URL:    "http://localhost:8080",
	}))

	report, err := buildStatusReport(projectDir)
	require.NoError(t, err)
	assert.True(t, report.Running)
	assert.Equal(t, os.Getpid(), report.PID)
	assert.Equal(t, "http://localhost:40000", report.DashboardURL)
	assert.Equal(t, startedAt, report.StartTime)
	require.Len(t, report.Services, 1)
	assert.Equal(t, "api", report.Services[0].Name)
	assert.Equal(t, "http://localhost:8080", report.Services[0].URL)
	assert.Equal(t, 8080, report.Services[0].Port)

	text := statusTextLines(report)
	assert.Contains(t, text, "App is running")
	assert.Contains(t, text, "Dashboard: http://localhost:40000")
	assert.Contains(t, text, "Services:")
	assert.Contains(t, text, "  - api: http://localhost:8080")

	payload, err := json.Marshal(report)
	require.NoError(t, err)
	assert.Contains(t, string(payload), `"running":true`)
	assert.Contains(t, string(payload), `"dashboardUrl":"http://localhost:40000"`)
	assert.Contains(t, string(payload), `"name":"api"`)
}

func TestFilterStatusReport(t *testing.T) {
	report := statusReport{
		Running: true,
		PID:     123,
		Services: []runstate.ServiceState{
			{Name: "api", URL: "http://localhost:8080", Port: 8080},
			{Name: "worker", Port: 9000},
			{Name: "web", URL: "http://localhost:3000", Port: 3000},
		},
	}

	filtered, err := filterStatusReport(report, []string{"web", "api"})
	require.NoError(t, err)
	require.Len(t, filtered.Services, 2)
	assert.Equal(t, "web", filtered.Services[0].Name)
	assert.Equal(t, "api", filtered.Services[1].Name)

	_, err = filterStatusReport(report, []string{"webb"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `service "webb" not found`)
	assert.Contains(t, err.Error(), `Did you mean "web"?`)
}

func TestFilterStatusReportNotRunning(t *testing.T) {
	report := statusReport{Running: false}

	filtered, err := filterStatusReport(report, []string{"api"})
	require.NoError(t, err)
	assert.False(t, filtered.Running)
	assert.Empty(t, filtered.Services)
}

func TestBuildStatusReportStaleStateRemovesFile(t *testing.T) {
	projectDir := t.TempDir()
	t.Cleanup(func() {
		_ = runstate.Remove(projectDir)
	})

	require.NoError(t, runstate.Write(projectDir, runstate.RunState{
		PID:          2_147_483_646,
		DashboardURL: "http://localhost:49999",
		StartTime:    time.Now().UTC().Round(time.Second),
	}))

	report, err := buildStatusReport(projectDir)
	require.NoError(t, err)
	assert.False(t, report.Running)

	_, exists, err := runstate.Read(projectDir)
	require.NoError(t, err)
	assert.False(t, exists)
}
