package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildContainerPortMappings_MultiPort(t *testing.T) {
	rt := &ServiceRuntime{
		Port: 10000,
		Ports: []PortMapping{
			{HostPort: 10000, ContainerPort: 10000, Protocol: "tcp"},
			{HostPort: 10001, ContainerPort: 10001},
			{HostPort: 10002, ContainerPort: 10002, Protocol: "tcp"},
		},
	}
	got := buildContainerPortMappings(rt)
	require.Len(t, got, 3)
	assert.Equal(t, 10000, got[0].HostPort)
	assert.Equal(t, 10001, got[1].HostPort)
	assert.Equal(t, 10002, got[2].HostPort)
	// Empty protocol defaults to tcp.
	assert.Equal(t, "tcp", got[1].Protocol)
}

func TestBuildContainerPortMappings_HostContainerDistinct(t *testing.T) {
	rt := &ServiceRuntime{
		Port:  3000,
		Ports: []PortMapping{{HostPort: 3000, ContainerPort: 8080, Protocol: "tcp"}},
	}
	got := buildContainerPortMappings(rt)
	require.Len(t, got, 1)
	assert.Equal(t, 3000, got[0].HostPort)
	assert.Equal(t, 8080, got[0].ContainerPort)
}

func TestBuildContainerPortMappings_FallbackPrimary(t *testing.T) {
	// No explicit Ports list -> fall back to the single primary port.
	rt := &ServiceRuntime{Port: 5432}
	got := buildContainerPortMappings(rt)
	require.Len(t, got, 1)
	assert.Equal(t, 5432, got[0].HostPort)
	assert.Equal(t, 5432, got[0].ContainerPort)
	assert.Equal(t, "tcp", got[0].Protocol)
}

func TestBuildContainerPortMappings_NoPorts(t *testing.T) {
	assert.Empty(t, buildContainerPortMappings(&ServiceRuntime{}))
}

func TestProjectNetworkDir_NormalizesToProjectRoot(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "azure.yaml"), []byte("name: demo\n"), 0o600))
	sub := filepath.Join(root, "services", "api")
	require.NoError(t, os.MkdirAll(sub, 0o755))

	// From a subdirectory, the network dir resolves to the project root, so the
	// derived network name matches the one derived from the root itself.
	assert.Equal(t, DeriveNetworkName(root), DeriveNetworkName(projectNetworkDir(sub)))
	assert.Equal(t, DeriveNetworkName(root), DeriveNetworkName(projectNetworkDir(root)))
}

func TestProjectNetworkDir_FallsBackWhenNoAzureYaml(t *testing.T) {
	// A directory with no azure.yaml (and no ancestor with one) falls back to itself.
	dir := t.TempDir()
	got := projectNetworkDir(dir)
	// Either the dir itself or an ancestor project root, but never empty.
	assert.NotEmpty(t, got)
}

type fakeImageChecker struct {
	exists bool
	calls  int
}

func (f *fakeImageChecker) ImageExists(string) bool {
	f.calls++
	return f.exists
}

func TestShouldPullImage(t *testing.T) {
	t.Run("never never pulls", func(t *testing.T) {
		fc := &fakeImageChecker{exists: false}
		assert.False(t, shouldPullImage(fc, "img", "never"))
		assert.Zero(t, fc.calls, "never must not query image existence")
	})
	t.Run("missing pulls only when absent", func(t *testing.T) {
		assert.True(t, shouldPullImage(&fakeImageChecker{exists: false}, "img", "missing"))
		assert.False(t, shouldPullImage(&fakeImageChecker{exists: true}, "img", "missing"))
	})
	t.Run("always pulls", func(t *testing.T) {
		assert.True(t, shouldPullImage(&fakeImageChecker{exists: true}, "img", "always"))
	})
	t.Run("empty default pulls (best effort)", func(t *testing.T) {
		assert.True(t, shouldPullImage(&fakeImageChecker{exists: true}, "img", ""))
	})
}
