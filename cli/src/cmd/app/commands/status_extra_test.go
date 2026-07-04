package commands

import (
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

func TestFormatStatusService(t *testing.T) {
	tests := []struct {
		name string
		svc  runstate.ServiceState
		want string
	}{
		{
			name: "url and port with mismatched port suffix",
			svc:  runstate.ServiceState{Name: "api", URL: "http://example.com", Port: 8080},
			want: "api: http://example.com (port 8080)",
		},
		{
			name: "url already contains port",
			svc:  runstate.ServiceState{Name: "web", URL: "http://localhost:3000", Port: 3000},
			want: "web: http://localhost:3000",
		},
		{
			name: "url only without port",
			svc:  runstate.ServiceState{Name: "gw", URL: "http://gateway"},
			want: "gw: http://gateway",
		},
		{
			name: "port derives localhost url",
			svc:  runstate.ServiceState{Name: "cache", Port: 6379},
			want: "cache: http://localhost:6379",
		},
		{
			name: "name only",
			svc:  runstate.ServiceState{Name: "worker"},
			want: "worker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatStatusService(tt.svc))
		})
	}
}

func TestStatusTextLines(t *testing.T) {
	started := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name        string
		report      statusReport
		wantContain []string
		wantExclude []string
	}{
		{
			name:        "not running",
			report:      statusReport{Running: false},
			wantContain: []string{"App is not running"},
		},
		{
			name:        "running without services",
			report:      statusReport{Running: true, PID: 123},
			wantContain: []string{"App is running", "PID: 123", "Services: none"},
		},
		{
			name: "running with services dashboard and start time",
			report: statusReport{
				Running:      true,
				PID:          42,
				DashboardURL: "http://localhost:4321",
				StartTime:    started,
				Services: []runstate.ServiceState{
					{Name: "api", URL: "http://localhost:8080", Port: 8080},
				},
			},
			wantContain: []string{
				"App is running",
				"PID: 42",
				"Dashboard: http://localhost:4321",
				"Started: " + started.Format(time.RFC3339),
				"Services:",
				"  - api: http://localhost:8080",
			},
			wantExclude: []string{"Services: none"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := statusTextLines(tt.report)
			for _, want := range tt.wantContain {
				assert.Contains(t, lines, want)
			}
			for _, exclude := range tt.wantExclude {
				assert.NotContains(t, lines, exclude)
			}
		})
	}
}

func TestPrintStatusReport(t *testing.T) {
	require.NoError(t, cliout.SetFormat("default"))

	// Running report exercises the Success branch plus the multi-line Plain loop.
	printStatusReport(statusReport{
		Running:      true,
		PID:          7,
		DashboardURL: "http://localhost:4321",
		StartTime:    time.Now(),
		Services:     []runstate.ServiceState{{Name: "api", Port: 8080}},
	})

	// Not-running report exercises the Info branch.
	printStatusReport(statusReport{Running: false})
}

func TestStatusServices(t *testing.T) {
	t.Run("uses running registry entries and derives url from port", func(t *testing.T) {
		projectDir := t.TempDir()
		reg := registry.GetRegistry(projectDir)
		require.NoError(t, reg.Register(&registry.ServiceRegistryEntry{
			Name:   "web",
			Status: constants.StatusReady,
			Port:   3000, // no URL -> should be derived
		}))
		require.NoError(t, reg.Register(&registry.ServiceRegistryEntry{
			Name:   "api",
			Status: constants.StatusRunning,
			Port:   8080,
			URL:    "http://localhost:8080",
		}))
		require.NoError(t, reg.Register(&registry.ServiceRegistryEntry{
			Name:   "stopped",
			Status: constants.StatusStopped,
			Port:   9999,
		}))

		services := statusServices(projectDir, nil)
		require.Len(t, services, 2) // stopped entry excluded
		assert.Equal(t, "api", services[0].Name)
		assert.Equal(t, "web", services[1].Name)
		assert.Equal(t, "http://localhost:3000", services[1].URL) // derived from port
	})

	t.Run("falls back to provided services when registry is empty", func(t *testing.T) {
		projectDir := t.TempDir()
		fallback := []runstate.ServiceState{
			{Name: "b", Port: 2},
			{Name: "a", URL: "http://localhost:1", Port: 1},
		}

		services := statusServices(projectDir, fallback)
		require.Len(t, services, 2)
		assert.Equal(t, "a", services[0].Name) // sorted by name
		assert.Equal(t, "b", services[1].Name)
	})
}

func TestBuildStatusReportNotRunning(t *testing.T) {
	projectDir := t.TempDir() // no run state written
	t.Cleanup(func() { _ = runstate.Remove(projectDir) })

	report, err := buildStatusReport(projectDir)
	require.NoError(t, err)
	assert.False(t, report.Running)
	assert.Empty(t, report.Services)
}

func TestBuildStatusReportReadError(t *testing.T) {
	projectDir := t.TempDir()
	path, err := runstate.Path(projectDir)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte("{ invalid json"), 0o600))
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(path)) })

	_, err = buildStatusReport(projectDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read run state")
}

func TestRunStatusNotRunning(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte("name: test\n"), 0o600))
	t.Chdir(dir)
	t.Cleanup(func() { _ = runstate.Remove(dir) })

	require.NoError(t, cliout.SetFormat("default"))
	require.NoError(t, runStatus(nil, nil))

	require.NoError(t, cliout.SetFormat("json"))
	t.Cleanup(func() { _ = cliout.SetFormat("default") })
	require.NoError(t, runStatus(nil, nil))
}

func TestRunStatusOutsideProject(t *testing.T) {
	// A directory with no azure.yaml (here or in any ancestor) makes
	// findProjectDir fail, so runStatus returns that error.
	t.Chdir(t.TempDir())

	require.NoError(t, cliout.SetFormat("default"))
	err := runStatus(nil, nil)
	require.Error(t, err)
}
