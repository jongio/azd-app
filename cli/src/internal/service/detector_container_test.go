package service

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectContainerRuntime_WiresContainerFields(t *testing.T) {
	projectDir := t.TempDir()
	svc := Service{
		Host:        "local",
		Image:       "postgres:16-alpine",
		CommandArgs: []string{"postgres", "-c", "max_connections=200"},
		PullPolicy:  "missing",
		Volumes:     []string{"pgdata:/var/lib/postgresql/data", "./init.sql:/docker-entrypoint-initdb.d/init.sql"},
		Ports:       []string{"5432:5432"},
		Type:        ServiceTypeContainer,
	}

	rt, err := detectContainerRuntime("db", svc, map[int]bool{}, projectDir)
	require.NoError(t, err)

	// Command tokens become the run args.
	assert.Equal(t, []string{"postgres", "-c", "max_connections=200"}, rt.Args)
	// Pull policy flows through.
	assert.Equal(t, "missing", rt.PullPolicy)
	// Primary port + health check port.
	assert.Equal(t, 5432, rt.Port)
	assert.Equal(t, 5432, rt.HealthCheck.Port)
	require.Len(t, rt.Ports, 1)
	assert.Equal(t, 5432, rt.Ports[0].HostPort)

	// Named volume passes through; relative bind resolves to an absolute path.
	// Use canonical project dir because resolveVolumeSpec canonicalizes symlinks.
	canonical, cErr := filepath.EvalSymlinks(projectDir)
	if cErr != nil {
		canonical = projectDir
	}
	require.Len(t, rt.Volumes, 2)
	assert.Equal(t, "pgdata:/var/lib/postgresql/data", rt.Volumes[0])
	wantBind := filepath.Join(canonical, "init.sql") + ":/docker-entrypoint-initdb.d/init.sql"
	assert.Equal(t, wantBind, rt.Volumes[1])
}

func TestDetectContainerRuntime_MultiPort(t *testing.T) {
	svc := Service{
		Host:  "local",
		Image: "mcr.microsoft.com/azure-storage/azurite:latest",
		Ports: []string{"10000:10000", "10001:10001", "10002:10002"},
		Type:  ServiceTypeContainer,
	}
	rt, err := detectContainerRuntime("azurite", svc, map[int]bool{}, t.TempDir())
	require.NoError(t, err)

	require.Len(t, rt.Ports, 3, "all three azurite ports should be captured")
	assert.Equal(t, 10000, rt.Port, "primary port is the first mapping")
	assert.Equal(t, 10000, rt.HealthCheck.Port)
}

func TestDetectServiceRuntime_ProcessArrayCommandPreservesWhitespace(t *testing.T) {
	// Array-form command on a non-container (process) service must preserve
	// argument tokens containing whitespace (regression guard).
	projectDir := t.TempDir()
	svc := Service{
		Host:        "local",
		Language:    "node",
		Project:     projectDir,
		Type:        ServiceTypeProcess,
		CommandArgs: []string{"node", "-e", "console.log('hello world')"},
		Command:     "node -e console.log('hello world')", // joined form as UnmarshalYAML sets it
	}
	rt, err := DetectServiceRuntime("worker", svc, map[int]bool{}, projectDir, "azd")
	require.NoError(t, err)
	assert.Equal(t, "node", rt.Command)
	assert.Equal(t, []string{"-e", "console.log('hello world')"}, rt.Args)
}

func TestDetectContainerRuntime_RejectsEscapingBind(t *testing.T) {
	svc := Service{
		Host:    "local",
		Image:   "busybox",
		Volumes: []string{"../../etc/passwd:/etc/passwd"},
		Type:    ServiceTypeContainer,
	}
	_, err := detectContainerRuntime("bad", svc, map[int]bool{}, t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "escapes")
}

func TestDetectServiceRuntime_DockerServiceWithCommandRunsAsProcess(t *testing.T) {
	// A deploy-oriented docker service (docker.image + Dockerfile) with an
	// explicit local command must run as a PROCESS for `azd app run`, not be
	// pulled/run as a container. docker.* stays deploy-only.
	projectDir := t.TempDir()
	svc := Service{
		Host:     "appservice",
		Language: "docker",
		Project:  projectDir,
		Docker:   &DockerConfig{Path: "./Dockerfile", Image: "myapp/web"},
		Command:  "npm run dev:web",
		Type:     ServiceTypeHTTP,
		Ports:    []string{"3000"},
	}
	rt, err := DetectServiceRuntime("web", svc, map[int]bool{}, projectDir, "azd")
	require.NoError(t, err)
	assert.NotEqual(t, ServiceTypeContainer, rt.Type, "docker service with a local command must run as a process")
	assert.Empty(t, rt.Image, "must not be treated as a container image to pull")
	assert.Equal(t, "npm", rt.Command)
	assert.Equal(t, []string{"run", "dev:web"}, rt.Args)
}

func TestDetectServiceRuntime_DockerServiceWithoutCommandStaysContainer(t *testing.T) {
	// Without a local command, a docker.image service keeps the (deploy default)
	// container behavior, unchanged.
	svc := Service{
		Host:     "containerapp",
		Language: "docker",
		Project:  ".",
		Docker:   &DockerConfig{Image: "myapp/web"},
	}
	rt, err := DetectServiceRuntime("web", svc, map[int]bool{}, t.TempDir(), "azd")
	require.NoError(t, err)
	assert.Equal(t, ServiceTypeContainer, rt.Type)
	assert.Equal(t, "myapp/web", rt.Image)
}
