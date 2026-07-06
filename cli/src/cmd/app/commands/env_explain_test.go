package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/cliout"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJoinSourcesHighestFirst(t *testing.T) {
	// Overrides are recorded lowest-first; display reverses to highest-first.
	got := joinSourcesHighestFirst([]service.EnvSource{service.EnvSourceOS, service.EnvSourceAzd, service.EnvSourceDotEnv})
	assert.Equal(t, ".env, azd, os", got)
}

func TestJoinSourcesHighestFirstSingle(t *testing.T) {
	assert.Equal(t, "azd", joinSourcesHighestFirst([]service.EnvSource{service.EnvSourceAzd}))
}

func TestJoinSourcesHighestFirstEmpty(t *testing.T) {
	assert.Equal(t, "", joinSourcesHighestFirst(nil))
}

func TestNewEnvCommandExplainFlag(t *testing.T) {
	cmd := NewEnvCommand()
	require.NotNil(t, cmd.Flags().Lookup("explain"), "expected --explain flag")
}

// TestRunEnvExplainCommand drives "env <service> --explain" end to end so the
// runEnv --explain branch and runEnvExplain's text, JSON, and error paths are
// all exercised. getAzureEnvironmentValues() copies the full OS environment, so
// every OS variable also appears as an "azd" layer; azure.yaml overrides both.
func TestRunEnvExplainCommand(t *testing.T) {
	tmpDir := t.TempDir()
	writeEnvExplainAzureYaml(t, tmpDir)

	// The explain path honors the global cliout output format (the --json flag).
	// Capture and restore it so these subtests are hermetic regardless of order.
	originalFormat := cliout.GetFormat()
	defer func() { _ = cliout.SetFormat(string(originalFormat)) }()

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()
	require.NoError(t, os.Chdir(tmpDir))

	t.Run("text output shows source and overrides per variable", func(t *testing.T) {
		resetEnvFlags()
		t.Setenv("EXPLAIN_OVERRIDE", "os-val")
		out, runErr := captureStdout(t, func() error {
			cmd := NewEnvCommand()
			cmd.SetArgs([]string{"api", "--explain"})
			return cmd.Execute()
		})
		require.NoError(t, runErr)
		// azure.yaml wins EXPLAIN_OVERRIDE over the os and azd layers; overrides
		// render highest priority first.
		assert.Contains(t, out, "EXPLAIN_OVERRIDE=svc-val\n    source: azure.yaml (overrode: azd, os)\n")
		// A variable defined only in azure.yaml has a single source, no overrides.
		assert.Contains(t, out, "EXPLAIN_SVC_ONLY=only-val\n    source: azure.yaml\n")
	})

	t.Run("json output includes source and overrides", func(t *testing.T) {
		resetEnvFlags()
		require.NoError(t, cliout.SetFormat("json"))
		t.Setenv("EXPLAIN_OVERRIDE", "os-val")
		out, runErr := captureStdout(t, func() error {
			cmd := NewEnvCommand()
			cmd.SetArgs([]string{"api", "--explain"})
			return cmd.Execute()
		})
		require.NoError(t, runErr)

		var got map[string]struct {
			Value     string   `json:"value"`
			Source    string   `json:"source"`
			Overrides []string `json:"overrides"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &got))

		override, ok := got["EXPLAIN_OVERRIDE"]
		require.True(t, ok, "EXPLAIN_OVERRIDE present in JSON output")
		assert.Equal(t, "svc-val", override.Value)
		assert.Equal(t, "azure.yaml", override.Source)
		// JSON serializes overrides lowest priority first (application order).
		assert.Equal(t, []string{"os", "azd"}, override.Overrides)

		svcOnly, ok := got["EXPLAIN_SVC_ONLY"]
		require.True(t, ok, "EXPLAIN_SVC_ONLY present in JSON output")
		assert.Equal(t, "azure.yaml", svcOnly.Source)
		assert.Empty(t, svcOnly.Overrides)
	})

	t.Run("env-file load failure surfaces an error", func(t *testing.T) {
		resetEnvFlags()
		out, runErr := captureStdout(t, func() error {
			cmd := NewEnvCommand()
			cmd.SetArgs([]string{"api", "--explain", "--env-file", filepath.Join(tmpDir, "missing.env")})
			return cmd.Execute()
		})
		require.Error(t, runErr)
		assert.Contains(t, runErr.Error(), "failed to resolve environment")
		assert.Empty(t, out)
	})
}

func writeEnvExplainAzureYaml(t *testing.T, dir string) {
	t.Helper()
	content := `name: envexplainapp
services:
  api:
    host: containerapp
    language: python
    project: ./api
    environment:
      EXPLAIN_OVERRIDE: svc-val
      EXPLAIN_SVC_ONLY: only-val
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte(content), 0o600))
}
