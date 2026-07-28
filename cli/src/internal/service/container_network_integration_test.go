//go:build integration && docker

package service

import (
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stamp returns a unique suffix for test resource names.
func stamp() string { return time.Now().Format("20060102150405.000") }

func TestEnsureNetwork_IdempotentAndRemoveTolerant(t *testing.T) {
	checkDockerAvailable(t)
	client := docker.NewClient()

	net := "azd-app-itest-idem-" + stamp()
	t.Cleanup(func() { _ = client.RemoveNetwork(net) })

	require.NoError(t, client.EnsureNetwork(net))
	// Second call must be tolerated (idempotent).
	require.NoError(t, client.EnsureNetwork(net))

	exists, err := client.NetworkExists(net)
	require.NoError(t, err)
	assert.True(t, exists)

	// Removing a non-existent network is not an error.
	require.NoError(t, client.RemoveNetwork("azd-app-itest-missing-"+stamp()))
}

// TestContainerNetwork_InterContainerDNS verifies the core of the networking
// feature: two container services on the shared project network can resolve
// each other by their service-name network alias (Docker Compose parity).
func TestContainerNetwork_InterContainerDNS(t *testing.T) {
	checkDockerAvailable(t)
	client := docker.NewClient()

	// Small, fast image with nslookup available.
	const image = "busybox:latest"
	if err := client.Pull(image); err != nil {
		t.Skipf("could not pull %s: %v", image, err)
	}

	net := "azd-app-itest-dns-" + stamp()
	require.NoError(t, client.EnsureNetwork(net))
	t.Cleanup(func() { _ = client.RemoveNetwork(net) })

	start := func(name, alias string) string {
		_ = client.Stop(name, 2)
		_ = client.Remove(name)
		id, err := client.Run(docker.ContainerConfig{
			Name:           name,
			Image:          image,
			Command:        []string{"sleep", "120"},
			Network:        net,
			NetworkAliases: []string{alias},
		})
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = client.Stop(name, 2)
			_ = client.Remove(name)
		})
		return id
	}

	alpha := "azd-itest-alpha-" + stamp()
	beta := "azd-itest-beta-" + stamp()
	start(alpha, "alpha")
	start(beta, "beta")

	// From beta, resolve alpha by its network alias. Retry for DNS propagation.
	var code int
	var out string
	var err error
	for i := 0; i < 10; i++ {
		code, out, err = client.Exec(beta, []string{"nslookup", "alpha"})
		if err == nil && code == 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	require.NoError(t, err)
	assert.Equal(t, 0, code, "beta must resolve sibling 'alpha' by network alias; nslookup output:\n%s", out)
}

// TestContainerReuse_NetworkConnect verifies ConnectNetwork is idempotent so a
// reused container is (re)attached to the project network without error.
func TestContainerReuse_NetworkConnect(t *testing.T) {
	checkDockerAvailable(t)
	client := docker.NewClient()

	const image = "busybox:latest"
	if err := client.Pull(image); err != nil {
		t.Skipf("could not pull %s: %v", image, err)
	}

	net := "azd-app-itest-connect-" + stamp()
	require.NoError(t, client.EnsureNetwork(net))
	t.Cleanup(func() { _ = client.RemoveNetwork(net) })

	name := "azd-itest-connect-" + stamp()
	_, err := client.Run(docker.ContainerConfig{
		Name:           name,
		Image:          image,
		Command:        []string{"sleep", "120"},
		Network:        net,
		NetworkAliases: []string{"svc"},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = client.Stop(name, 2)
		_ = client.Remove(name)
	})

	// Container is already connected; a repeat connect must be a no-op.
	require.NoError(t, client.ConnectNetwork(net, name, []string{"svc"}))
}
