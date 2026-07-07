package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-core/cliout"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildEnvDiff(t *testing.T) {
	a := map[string]string{
		"SHARED":   "avalue",
		"API_ONLY": "1",
		"COMMON":   "same",
	}
	b := map[string]string{
		"SHARED":   "bvalue",
		"WEB_ONLY": "1",
		"COMMON":   "same",
	}

	result := buildEnvDiff("api", "web", a, b)

	assert.Equal(t, "api", result.ServiceA)
	assert.Equal(t, "web", result.ServiceB)
	assert.Equal(t, map[string]string{"API_ONLY": "1"}, result.OnlyInA)
	assert.Equal(t, map[string]string{"WEB_ONLY": "1"}, result.OnlyInB)
	require.Contains(t, result.Changed, "SHARED")
	assert.Equal(t, "avalue", result.Changed["SHARED"].A)
	assert.Equal(t, "bvalue", result.Changed["SHARED"].B)
	assert.Equal(t, 1, result.Same)
}

// writeEnvDiffAzureYaml writes an azure.yaml with two services whose
// service-specific environment blocks differ, so a diff has something to report.
func writeEnvDiffAzureYaml(t *testing.T, dir string) {
	t.Helper()
	content := `name: envdifftest
services:
  api:
    host: containerapp
    language: python
    project: ./api
    environment:
      SHARED: apivalue
      API_ONLY: apionly
      COMMON: same
  web:
    host: containerapp
    language: js
    project: ./web
    environment:
      SHARED: webvalue
      WEB_ONLY: webonly
      COMMON: same
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte(content), 0o600))
}

func TestRunEnvDiff(t *testing.T) {
	originalFormat := cliout.GetFormat()
	defer func() { _ = cliout.SetFormat(string(originalFormat)) }()

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	chdirTemp := func(t *testing.T, dir string) {
		require.NoError(t, os.Chdir(dir))
		t.Cleanup(func() { _ = os.Chdir(originalDir) })
	}

	t.Run("reports only-in and changed values", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeEnvDiffAzureYaml(t, tmpDir)
		chdirTemp(t, tmpDir)
		resetEnvFlags()

		out, runErr := captureStdout(t, func() error {
			cmd := NewEnvCommand()
			cmd.SetArgs([]string{"--diff", "api", "web"})
			return cmd.Execute()
		})
		require.NoError(t, runErr)
		assert.Contains(t, out, "Only in api:")
		assert.Contains(t, out, "API_ONLY=apionly")
		assert.Contains(t, out, "Only in web:")
		assert.Contains(t, out, "WEB_ONLY=webonly")
		assert.Contains(t, out, "Different values:")
		assert.Contains(t, out, "SHARED:")
		assert.Contains(t, out, "api: apivalue")
		assert.Contains(t, out, "web: webvalue")
	})

	t.Run("json output groups the differences", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeEnvDiffAzureYaml(t, tmpDir)
		chdirTemp(t, tmpDir)
		resetEnvFlags()

		out, runErr := captureStdout(t, func() error {
			cmd := NewEnvCommand()
			cmd.SetArgs([]string{"--diff", "api", "web", "--format", "json"})
			return cmd.Execute()
		})
		require.NoError(t, runErr)

		var parsed envDiffResult
		require.NoError(t, json.Unmarshal([]byte(out), &parsed))
		assert.Equal(t, "api", parsed.ServiceA)
		assert.Equal(t, "web", parsed.ServiceB)
		assert.Contains(t, parsed.OnlyInA, "API_ONLY")
		assert.Contains(t, parsed.OnlyInB, "WEB_ONLY")
		assert.Contains(t, parsed.Changed, "SHARED")
	})

	t.Run("identical services report the same environment", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeEnvTestAzureYaml(t, tmpDir)
		chdirTemp(t, tmpDir)
		resetEnvFlags()

		out, runErr := captureStdout(t, func() error {
			cmd := NewEnvCommand()
			cmd.SetArgs([]string{"--diff", "api", "web"})
			return cmd.Execute()
		})
		require.NoError(t, runErr)
		assert.Contains(t, out, "same environment")
	})

	t.Run("one service name errors", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeEnvDiffAzureYaml(t, tmpDir)
		chdirTemp(t, tmpDir)
		resetEnvFlags()

		cmd := NewEnvCommand()
		cmd.SetArgs([]string{"--diff", "api"})
		runErr := cmd.Execute()
		require.Error(t, runErr)
		assert.Contains(t, runErr.Error(), "exactly two service names")
	})

	t.Run("same service twice errors", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeEnvDiffAzureYaml(t, tmpDir)
		chdirTemp(t, tmpDir)
		resetEnvFlags()

		cmd := NewEnvCommand()
		cmd.SetArgs([]string{"--diff", "api", "api"})
		runErr := cmd.Execute()
		require.Error(t, runErr)
		assert.Contains(t, runErr.Error(), "two different service names")
	})

	t.Run("unknown service errors", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeEnvDiffAzureYaml(t, tmpDir)
		chdirTemp(t, tmpDir)
		resetEnvFlags()

		cmd := NewEnvCommand()
		cmd.SetArgs([]string{"--diff", "api", "nope"})
		runErr := cmd.Execute()
		require.Error(t, runErr)
		assert.Contains(t, runErr.Error(), `service "nope" not found`)
	})

	t.Run("diff with all errors", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeEnvDiffAzureYaml(t, tmpDir)
		chdirTemp(t, tmpDir)
		resetEnvFlags()

		cmd := NewEnvCommand()
		cmd.SetArgs([]string{"--diff", "api", "web", "--all"})
		runErr := cmd.Execute()
		require.Error(t, runErr)
		assert.Contains(t, runErr.Error(), "cannot combine --diff with --all")
	})

	t.Run("more than two service names errors", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeEnvDiffAzureYaml(t, tmpDir)
		chdirTemp(t, tmpDir)
		resetEnvFlags()

		cmd := NewEnvCommand()
		cmd.SetArgs([]string{"--diff", "api", "web", "extra"})
		runErr := cmd.Execute()
		require.Error(t, runErr)
	})
}
