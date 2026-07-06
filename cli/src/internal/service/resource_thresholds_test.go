package service_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAzureYaml_ResourceThresholds(t *testing.T) {
	yamlContent := `name: resource-test
services:
  api:
    host: containerapp
    image: nginx:latest
    ports: ["8080"]
    resources:
      cpuPercent: 85
      memoryMB: 1024
  worker:
    host: containerapp
    image: worker:latest
`
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "azure.yaml"), []byte(yamlContent), 0o600))

	parsed, err := service.ParseAzureYaml(tmpDir)
	require.NoError(t, err)

	api, ok := parsed.Services["api"]
	require.True(t, ok)
	require.NotNil(t, api.Resources)
	assert.Equal(t, 85.0, api.Resources.CPUPercent)
	assert.Equal(t, uint64(1024), api.Resources.MemoryMB)

	// A service without a resources block leaves it nil.
	worker, ok := parsed.Services["worker"]
	require.True(t, ok)
	assert.Nil(t, worker.Resources)
}
