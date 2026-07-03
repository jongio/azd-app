package service_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectServiceRuntime_RestartPolicyConfiguration(t *testing.T) {
	tests := []struct {
		name               string
		yamlContent        string
		restartBlockExists bool
		expectedPolicy     string
		expectedMaxRetries int
		expectedBackoff    time.Duration
	}{
		{
			name: "explicit restart policy is parsed and resolved",
			yamlContent: `name: restart-test
services:
  api:
    host: containerapp
    image: nginx:latest
    ports: ["8080"]
    restart:
      policy: on-failure
      maxRetries: 5
      backoff: 250ms
`,
			restartBlockExists: true,
			expectedPolicy:     service.RestartPolicyOnFailure,
			expectedMaxRetries: 5,
			expectedBackoff:    250 * time.Millisecond,
		},
		{
			name: "restart defaults to never when block is omitted",
			yamlContent: `name: restart-test
services:
  api:
    host: containerapp
    image: nginx:latest
    ports: ["8080"]
`,
			restartBlockExists: false,
			expectedPolicy:     service.RestartPolicyNever,
			expectedMaxRetries: service.DefaultRestartMaxRetries,
			expectedBackoff:    service.DefaultRestartBackoffBase,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
			require.NoError(t, os.WriteFile(azureYamlPath, []byte(tt.yamlContent), 0o600))

			parsed, err := service.ParseAzureYaml(tmpDir)
			require.NoError(t, err)

			svc, exists := parsed.Services["api"]
			require.True(t, exists)

			if tt.restartBlockExists {
				require.NotNil(t, svc.Restart)
			} else {
				assert.Nil(t, svc.Restart)
			}

			runtime, err := service.DetectServiceRuntime("api", svc, map[int]bool{}, tmpDir, "azd")
			require.NoError(t, err)

			assert.Equal(t, tt.expectedPolicy, runtime.Restart.Policy)
			assert.Equal(t, tt.expectedMaxRetries, runtime.Restart.MaxRetries)
			assert.Equal(t, tt.expectedBackoff, runtime.Restart.BackoffBase)
		})
	}
}
