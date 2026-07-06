package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectExposureFindingsFromServiceEnv(t *testing.T) {
	services := map[string]service.Service{
		"api": {
			Environment: service.Environment{
				"HOST":         "0.0.0.0",
				"BIND_ADDRESS": "127.0.0.1",
			},
		},
		"worker": {
			Environment: service.Environment{
				"ASPNETCORE_URLS": "http://+:5000",
			},
		},
	}

	findings := collectExposureFindings(services, t.TempDir())

	require.Len(t, findings, 2)
	// Sorted by service name: api before worker.
	assert.Equal(t, "azure.yaml (service: api)", findings[0].Source)
	assert.Equal(t, "HOST", findings[0].Key)
	assert.Equal(t, "azure.yaml (service: worker)", findings[1].Source)
	assert.Equal(t, "ASPNETCORE_URLS", findings[1].Key)
}

func TestCollectExposureFindingsFromDotEnv(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(envPath, []byte("HOST=0.0.0.0\nPORT=8080\n"), 0o600))

	findings := collectExposureFindings(map[string]service.Service{}, dir)

	require.Len(t, findings, 1)
	assert.Equal(t, ".env", findings[0].Source)
	assert.Equal(t, "HOST", findings[0].Key)
	assert.Equal(t, "0.0.0.0", findings[0].Value)
}

func TestCollectExposureFindingsServiceDotEnv(t *testing.T) {
	dir := t.TempDir()
	svcDir := filepath.Join(dir, "api")
	require.NoError(t, os.MkdirAll(svcDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, ".env"), []byte("LISTEN_ADDR=[::]:9000\n"), 0o600))

	services := map[string]service.Service{
		"api": {Project: "api"},
	}

	findings := collectExposureFindings(services, dir)

	require.Len(t, findings, 1)
	assert.Equal(t, filepath.Join("api", ".env"), findings[0].Source)
	assert.Equal(t, "LISTEN_ADDR", findings[0].Key)
}

func TestCollectExposureFindingsNoFindings(t *testing.T) {
	services := map[string]service.Service{
		"api": {
			Environment: service.Environment{
				"HOST": "127.0.0.1",
				"PORT": "8080",
			},
		},
	}

	assert.Empty(t, collectExposureFindings(services, t.TempDir()))
}

func TestExposureEnvFilesSkipsMissing(t *testing.T) {
	dir := t.TempDir()
	services := map[string]service.Service{
		"api": {Project: "api"},
	}

	assert.Empty(t, exposureEnvFiles(services, dir))
}
