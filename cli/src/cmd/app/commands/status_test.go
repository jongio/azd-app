package commands

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-app/cli/src/internal/runstate"
	"github.com/jongio/azd-core/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStatusCommand(t *testing.T) {
	cmd := NewStatusCommand()
	require.NotNil(t, cmd)
	assert.Equal(t, "status", cmd.Use)
	assert.Equal(t, "Show whether azd app run is active", cmd.Short)
	require.NotNil(t, cmd.RunE)
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
