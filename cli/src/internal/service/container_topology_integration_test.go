//go:build integration && docker

package service

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dockerInspectField runs `docker inspect -f <format> <id>` and returns trimmed output.
func dockerInspectField(t *testing.T, id, format string) string {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "docker", "inspect", "-f", format, id)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "docker inspect failed: %s", string(out))
	return strings.TrimSpace(string(out))
}

// TestStartContainerService_WebsiteStyleTopology is the AC8 goal-challenge: it
// runs an azurite-like container through the FULL runtime path
// (DetectServiceRuntime -> StartContainerService -> docker run) exercising the
// website's mechanisms together, a string command, THREE published ports, a
// named volume, and attachment to the shared project network, and verifies via
// `docker inspect` that they all took effect.
func TestStartContainerService_WebsiteStyleTopology(t *testing.T) {
	checkDockerAvailable(t)

	client := docker.NewClient()
	const image = "mcr.microsoft.com/azure-storage/azurite:latest"
	if err := client.Pull(image); err != nil {
		t.Skipf("could not pull %s: %v", image, err)
	}

	// Project dir with an azure.yaml so the network name derives from the root.
	projectDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "azure.yaml"), []byte("name: web-itest\n"), 0o600))

	volName := "azd-itest-azurite-data-" + strings.ReplaceAll(stamp(), ".", "")
	svc := Service{
		Host:    "local",
		Image:   image,
		Type:    ServiceTypeContainer,
		Command: "azurite --blobHost 0.0.0.0 --queueHost 0.0.0.0 --tableHost 0.0.0.0 --skipApiVersionCheck",
		Ports:   []string{"11000:10000", "11001:10001", "11002:10002"},
		Volumes: []string{volName + ":/data"},
	}

	rt, err := DetectServiceRuntime("azurite", svc, map[int]bool{}, projectDir, "azd")
	require.NoError(t, err)
	require.Len(t, rt.Ports, 3, "all three ports captured in runtime")

	containerName := "azd-azurite"
	_ = client.Stop(containerName, 3)
	_ = client.Remove(containerName)

	proc, err := StartContainerService(rt, projectDir, true)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = client.Stop(containerName, 3)
		_ = client.Remove(containerName)
		_ = client.RemoveNetwork(DeriveNetworkName(projectDir))
	})
	require.NotEmpty(t, proc.ContainerID)
	assert.True(t, client.IsRunning(proc.ContainerID))

	// Command override reached the container.
	assert.Contains(t, dockerInspectField(t, proc.ContainerID, `{{join .Config.Cmd " "}}`), "--skipApiVersionCheck")

	// All three host ports are published.
	ports := dockerInspectField(t, proc.ContainerID, "{{.NetworkSettings.Ports}}")
	for _, p := range []string{"11000", "11001", "11002"} {
		assert.Contains(t, ports, p, "expected host port %s to be published", p)
	}

	// Named volume is mounted at /data.
	mounts := dockerInspectField(t, proc.ContainerID, "{{range .Mounts}}{{.Name}}:{{.Destination}} {{end}}")
	assert.Contains(t, mounts, volName+":/data")

	// Attached to the derived project network.
	networks := dockerInspectField(t, proc.ContainerID, "{{range $k,$v := .NetworkSettings.Networks}}{{$k}} {{end}}")
	assert.Contains(t, networks, DeriveNetworkName(projectDir))
}
