package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// indexOf returns the first index of val in args, or -1.
func indexOf(args []string, val string) int {
	for i, a := range args {
		if a == val {
			return i
		}
	}
	return -1
}

// flagValue returns the argument immediately following the first occurrence of
// flag, or "" if not present / at end.
func flagValue(args []string, flag string) string {
	i := indexOf(args, flag)
	if i < 0 || i+1 >= len(args) {
		return ""
	}
	return args[i+1]
}

func TestBuildRunArgs_Volumes(t *testing.T) {
	cfg := ContainerConfig{
		Name:  "azd-postgres",
		Image: "postgres:16-alpine",
		Volumes: []string{
			"pgdata:/var/lib/postgresql/data",
			"/abs/host/config.json:/app/config.json",
		},
	}
	args := buildRunArgs(cfg)

	// Each volume must appear as a discrete `-v <spec>` pair.
	require.Contains(t, args, "pgdata:/var/lib/postgresql/data")
	require.Contains(t, args, "/abs/host/config.json:/app/config.json")

	vi := indexOf(args, "pgdata:/var/lib/postgresql/data")
	require.Positive(t, vi)
	assert.Equal(t, "-v", args[vi-1], "volume spec must be preceded by -v")
}

func TestBuildRunArgs_CommandAfterImage(t *testing.T) {
	cfg := ContainerConfig{
		Name:    "azd-postgres",
		Image:   "postgres:16-alpine",
		Command: []string{"postgres", "-c", "max_connections=200"},
	}
	args := buildRunArgs(cfg)

	imgIdx := indexOf(args, "postgres:16-alpine")
	require.Positive(t, imgIdx)
	// Command tokens follow the image, in order, as the final args.
	assert.Equal(t, []string{"postgres", "-c", "max_connections=200"}, args[imgIdx+1:])
}

func TestBuildRunArgs_NoCommandOmitsTokens(t *testing.T) {
	cfg := ContainerConfig{Name: "azd-x", Image: "redis:7-alpine"}
	args := buildRunArgs(cfg)
	imgIdx := indexOf(args, "redis:7-alpine")
	require.Positive(t, imgIdx)
	assert.Equal(t, imgIdx, len(args)-1, "image should be the last arg when no command")
}

func TestBuildRunArgs_NetworkAndAliases(t *testing.T) {
	cfg := ContainerConfig{
		Name:           "azd-azurite",
		Image:          "mcr.microsoft.com/azure-storage/azurite:latest",
		Network:        "azd-app-web-abcd1234",
		NetworkAliases: []string{"azurite"},
	}
	args := buildRunArgs(cfg)

	assert.Equal(t, "azd-app-web-abcd1234", flagValue(args, "--network"))
	assert.Equal(t, "azurite", flagValue(args, "--network-alias"))
}

func TestBuildRunArgs_NoNetworkWhenUnset(t *testing.T) {
	cfg := ContainerConfig{Name: "azd-x", Image: "redis:7-alpine"}
	args := buildRunArgs(cfg)
	assert.NotContains(t, args, "--network")
	assert.NotContains(t, args, "--network-alias")
}

func TestBuildRunArgs_PullNeverEmitsFlag(t *testing.T) {
	assert.Equal(t, "never", flagValue(buildRunArgs(ContainerConfig{Image: "x", PullPolicy: PullNever}), "--pull"))
	// missing / always / "" do not add a run-time --pull flag.
	for _, p := range []string{"", PullMissing, PullAlways} {
		args := buildRunArgs(ContainerConfig{Image: "x", PullPolicy: p})
		assert.NotContains(t, args, "--pull", "policy %q must not add --pull", p)
	}
}

func TestBuildRunArgs_MultiPort(t *testing.T) {
	cfg := ContainerConfig{
		Name:  "azd-azurite",
		Image: "azurite",
		Ports: []PortMapping{
			{HostPort: 10000, ContainerPort: 10000, Protocol: "tcp"},
			{HostPort: 10001, ContainerPort: 10001, Protocol: "tcp"},
			{HostPort: 10002, ContainerPort: 10002, Protocol: "tcp"},
		},
	}
	args := buildRunArgs(cfg)
	// All three ports must be published.
	count := 0
	for i, a := range args {
		if a == "-p" && i+1 < len(args) {
			count++
		}
	}
	assert.Equal(t, 3, count, "expected three -p mappings")
	assert.Contains(t, args, "10000:10000/tcp")
	assert.Contains(t, args, "10001:10001/tcp")
	assert.Contains(t, args, "10002:10002/tcp")
}

func TestBuildRunArgs_Environment(t *testing.T) {
	// Single entry avoids map-ordering concerns.
	cfg := ContainerConfig{
		Name:        "azd-pg",
		Image:       "postgres:16-alpine",
		Environment: map[string]string{"POSTGRES_DB": "app"},
	}
	args := buildRunArgs(cfg)
	ei := indexOf(args, "-e")
	require.Positive(t, ei)
	assert.Contains(t, args, "POSTGRES_DB=app")
}

func TestStderrClassifiers(t *testing.T) {
	// EnsureNetwork idempotency.
	assert.True(t, isAlreadyExistsError(`Error response from daemon: network with name azd-app-x already exists`))
	assert.False(t, isAlreadyExistsError(`Error: something else`))

	// RemoveNetwork tolerance.
	assert.True(t, isNetworkNotFoundError(`Error: No such network: azd-app-x`))
	assert.True(t, isNetworkNotFoundError(`Error response from daemon: network azd-app-x not found`))
	assert.False(t, isNetworkNotFoundError(`Error: permission denied`))

	// ConnectNetwork idempotency (both known Docker phrasings).
	assert.True(t, isAlreadyConnectedError(`Error response from daemon: endpoint with name azd-svc already exists in network azd-app-x`))
	assert.True(t, isAlreadyConnectedError(`Error: container is already connected to network azd-app-x`))
	assert.False(t, isAlreadyConnectedError(`Error: no such container`))
}

func TestValidateVolumeSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		wantErr bool
	}{
		{"named volume", "pgdata:/var/lib/postgresql/data", false},
		{"bind mount", "/abs/host:/container", false},
		{"anonymous", "/data", false},
		{"empty", "", true},
		{"whitespace only", "   ", true},
		{"newline injection", "pgdata:/data\nmalicious", true},
		{"null byte", "pgdata:/data\x00", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVolumeSpec(tt.spec)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePullPolicy(t *testing.T) {
	for _, p := range []string{"", PullMissing, PullAlways, PullNever} {
		assert.NoError(t, ValidatePullPolicy(p), "policy %q should be valid", p)
	}
	assert.Error(t, ValidatePullPolicy("sometimes"))
	assert.Error(t, ValidatePullPolicy("Always"))
}

func TestValidateNetworkName(t *testing.T) {
	assert.NoError(t, ValidateNetworkName(""))
	assert.NoError(t, ValidateNetworkName("azd-app-web-abcd1234"))
	assert.Error(t, ValidateNetworkName("-badstart"))
	assert.Error(t, ValidateNetworkName("has space"))
}

func TestContainerConfigValidate_RejectsBadFields(t *testing.T) {
	base := ContainerConfig{Name: "azd-x", Image: "redis:7-alpine"}

	bad := base
	bad.PullPolicy = "nope"
	assert.Error(t, bad.Validate())

	bad2 := base
	bad2.Network = "bad name"
	assert.Error(t, bad2.Validate())

	bad3 := base
	bad3.Volumes = []string{"ok:/data", "bad\nvol"}
	assert.Error(t, bad3.Validate())

	ok := base
	ok.PullPolicy = PullMissing
	ok.Network = "azd-app-x-1234abcd"
	ok.Volumes = []string{"data:/var/lib/x"}
	assert.NoError(t, ok.Validate())
}
