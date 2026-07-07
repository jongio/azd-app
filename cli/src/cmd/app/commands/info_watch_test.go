package commands

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
	"github.com/jongio/azd-core/cliout"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInfoCommandWatchFlags(t *testing.T) {
	cmd := NewInfoCommand()
	require.NotNil(t, cmd)

	watchFlag := cmd.Flags().Lookup("watch")
	require.NotNil(t, watchFlag, "expected --watch flag")

	intervalFlag := cmd.Flags().Lookup("interval")
	require.NotNil(t, intervalFlag, "expected --interval flag")
	assert.Equal(t, "2s", intervalFlag.DefValue, "expected default interval of 2s")
}

func TestInfoWatchLines(t *testing.T) {
	services := []*serviceinfo.ServiceInfo{
		{
			Name: "api",
			Local: &serviceinfo.LocalServiceInfo{
				Status:      "running",
				Health:      "healthy",
				URL:         "http://localhost:8000",
				Port:        8000,
				PID:         4321,
				CPUPercent:  12.5,
				MemoryBytes: 64 * 1024 * 1024,
			},
		},
		{
			Name: "web",
			Local: &serviceinfo.LocalServiceInfo{
				Status: "stopped",
			},
		},
	}

	lines := infoWatchLines(services)
	joined := ""
	for _, l := range lines {
		joined += l + "\n"
	}

	assert.Contains(t, joined, "api  [running/healthy]")
	assert.Contains(t, joined, "port 8000")
	assert.Contains(t, joined, "pid 4321")
	assert.Contains(t, joined, "cpu 12.5%")
	assert.Contains(t, joined, "mem 64.0 MiB")
	assert.Contains(t, joined, "http://localhost:8000")
	// A stopped service shows its status but no runtime detail line.
	assert.Contains(t, joined, "web  [stopped/unknown]")
}

func TestInfoWatchLinesEmpty(t *testing.T) {
	lines := infoWatchLines(nil)
	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], "No services defined")
}

func TestRenderInfoFrame(t *testing.T) {
	var buf bytes.Buffer
	services := []*serviceinfo.ServiceInfo{
		{Name: "api", Local: &serviceinfo.LocalServiceInfo{Status: "running", Health: "healthy"}},
	}

	renderInfoFrame(&buf, "/root/project", services, time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC))
	out := buf.String()

	// Each frame clears the screen and prints a refresh header.
	assert.Contains(t, out, "\033[H\033[2J")
	assert.Contains(t, out, "refreshed 15:04:05")
	assert.Contains(t, out, "Project: /root/project")
	assert.Contains(t, out, "api  [running/healthy]")
}

func TestWatchInfoStopsOnContextCancel(t *testing.T) {
	cwd := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already canceled: render one frame, then return via ctx.Done()

	var buf bytes.Buffer
	err := watchInfo(ctx, &buf, cwd, nil, time.Second)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "azd app info")
	assert.Contains(t, out, "Stopped watching.")
}

func TestRunInfoWatchIntervalTooSmall(t *testing.T) {
	_ = cliout.SetFormat("default")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte("name: infotest\nservices: {}\n"), 0o600))
	t.Chdir(dir)

	cmd := NewInfoCommand()
	cmd.SetArgs([]string{"--watch", "--interval", "100ms"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--interval must be at least")
}
