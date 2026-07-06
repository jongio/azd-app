package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-core/cliout"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEnvCommand(t *testing.T) {
	cmd := NewEnvCommand()
	require.NotNil(t, cmd)
	assert.Equal(t, "env [service]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	require.NotNil(t, cmd.RunE)

	for _, name := range []string{"format", "no-mask", "env-file", "all", "explain"} {
		assert.NotNil(t, cmd.Flags().Lookup(name), "expected --%s flag", name)
	}
}

func TestResolveEnvFormat(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"", envFormatDotenv, false},
		{"dotenv", envFormatDotenv, false},
		{"DOTENV", envFormatDotenv, false},
		{"shell", envFormatShell, false},
		{"json", envFormatJSON, false},
		{" json ", envFormatJSON, false},
		{"yaml", "", true},
	}
	for _, tt := range tests {
		got, err := resolveEnvFormat(tt.in)
		if tt.wantErr {
			require.Error(t, err, "input %q", tt.in)
			continue
		}
		require.NoError(t, err, "input %q", tt.in)
		assert.Equal(t, tt.want, got)
	}
}

func TestMaskEnv(t *testing.T) {
	env := map[string]string{
		"PLAIN":         "value",
		"DB_PASSWORD":   "supersecret",
		"API_TOKEN":     "abcdef",
		"SHORT_SECRET":  "ab",
		"CONNECTIONURL": "postgres://host",
	}

	t.Run("masking on hides secret-shaped values", func(t *testing.T) {
		got := maskEnv(env, true)
		assert.Equal(t, "value", got["PLAIN"])
		assert.Equal(t, "postgres://host", got["CONNECTIONURL"])
		assert.NotEqual(t, "supersecret", got["DB_PASSWORD"])
		assert.Contains(t, got["DB_PASSWORD"], "***")
		assert.Equal(t, "***", got["SHORT_SECRET"])
	})

	t.Run("masking off keeps raw values", func(t *testing.T) {
		got := maskEnv(env, false)
		assert.Equal(t, "supersecret", got["DB_PASSWORD"])
		assert.Equal(t, "ab", got["SHORT_SECRET"])
	})
}

func TestFormatEnv(t *testing.T) {
	env := map[string]string{
		"B_KEY":       "two",
		"A_KEY":       "one",
		"DB_PASSWORD": "supersecret",
	}

	t.Run("dotenv is sorted KEY=value", func(t *testing.T) {
		out := formatEnv(env, envFormatDotenv, false)
		lines := splitNonEmpty(out)
		require.Len(t, lines, 3)
		assert.Equal(t, "A_KEY=one", lines[0])
		assert.Equal(t, "B_KEY=two", lines[1])
		assert.Equal(t, "DB_PASSWORD=supersecret", lines[2])
	})

	t.Run("shell emits quoted export lines", func(t *testing.T) {
		out := formatEnv(env, envFormatShell, false)
		assert.Contains(t, out, `export A_KEY="one"`)
		assert.Contains(t, out, `export DB_PASSWORD="supersecret"`)
	})

	t.Run("masking applies before formatting", func(t *testing.T) {
		out := formatEnv(env, envFormatDotenv, true)
		assert.NotContains(t, out, "supersecret")
		assert.Contains(t, out, "DB_PASSWORD=su***et")
	})
}

func TestShellQuoteDouble(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"plain", `"plain"`},
		{`with "quotes"`, `"with \"quotes\""`},
		{`back\slash`, `"back\\slash"`},
		{"dollar$var", `"dollar\$var"`},
		{"tick`cmd`", "\"tick\\`cmd\\`\""},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, shellQuoteDouble(tt.in), "input %q", tt.in)
	}
}

func TestRunEnvCommand(t *testing.T) {
	tmpDir := t.TempDir()
	writeEnvTestAzureYaml(t, tmpDir)

	// The env command honors the global cliout output format (the --json flag).
	// Other tests in this package set that global to JSON and reset it with an
	// invalid "text" value, which is a no-op, so the process-wide format can
	// still be JSON when these subtests run. Capture and restore it here, and
	// reset it in resetEnvFlags, so these subtests are hermetic regardless of
	// execution order.
	originalFormat := cliout.GetFormat()
	defer func() { _ = cliout.SetFormat(string(originalFormat)) }()

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()
	require.NoError(t, os.Chdir(tmpDir))

	t.Run("no argument lists services without error", func(t *testing.T) {
		resetEnvFlags()
		out, runErr := captureStdout(t, func() error {
			cmd := NewEnvCommand()
			cmd.SetArgs([]string{})
			return cmd.Execute()
		})
		require.NoError(t, runErr)
		assert.Contains(t, out, "api")
		assert.Contains(t, out, "web")
	})

	t.Run("unknown service errors and lists available", func(t *testing.T) {
		resetEnvFlags()
		cmd := NewEnvCommand()
		cmd.SetArgs([]string{"apii"})
		runErr := cmd.Execute()
		require.Error(t, runErr)
		assert.Contains(t, runErr.Error(), `service "apii" not found`)
		assert.Contains(t, runErr.Error(), "Available services")
	})

	t.Run("known service prints its resolved environment", func(t *testing.T) {
		resetEnvFlags()
		t.Setenv("SERVICE_ENV_MARKER", "marker-value")
		out, runErr := captureStdout(t, func() error {
			cmd := NewEnvCommand()
			cmd.SetArgs([]string{"api"})
			return cmd.Execute()
		})
		require.NoError(t, runErr)
		assert.Contains(t, out, "SERVICE_ENV_MARKER=marker-value")
	})

	t.Run("invalid format errors", func(t *testing.T) {
		resetEnvFlags()
		cmd := NewEnvCommand()
		cmd.SetArgs([]string{"api", "--format", "yaml"})
		runErr := cmd.Execute()
		require.Error(t, runErr)
		assert.Contains(t, runErr.Error(), "invalid --format")
	})

	t.Run("all prints every service grouped by header", func(t *testing.T) {
		resetEnvFlags()
		t.Setenv("SERVICE_ENV_MARKER", "marker-value")
		out, runErr := captureStdout(t, func() error {
			cmd := NewEnvCommand()
			cmd.SetArgs([]string{"--all"})
			return cmd.Execute()
		})
		require.NoError(t, runErr)
		assert.Contains(t, out, "# api")
		assert.Contains(t, out, "# web")
		// The api header must come before the web header (services are sorted).
		assert.Less(t, strings.Index(out, "# api"), strings.Index(out, "# web"))
		assert.Contains(t, out, "SERVICE_ENV_MARKER=marker-value")
	})

	t.Run("all with json emits an object keyed by service", func(t *testing.T) {
		resetEnvFlags()
		out, runErr := captureStdout(t, func() error {
			cmd := NewEnvCommand()
			cmd.SetArgs([]string{"--all", "--format", "json"})
			return cmd.Execute()
		})
		require.NoError(t, runErr)
		var parsed map[string]map[string]string
		require.NoError(t, json.Unmarshal([]byte(out), &parsed))
		assert.Contains(t, parsed, "api")
		assert.Contains(t, parsed, "web")
	})

	t.Run("all with a service name errors", func(t *testing.T) {
		resetEnvFlags()
		cmd := NewEnvCommand()
		cmd.SetArgs([]string{"api", "--all"})
		runErr := cmd.Execute()
		require.Error(t, runErr)
		assert.Contains(t, runErr.Error(), "cannot combine --all with a service name")
	})
}

func TestRenderAllEnv(t *testing.T) {
	resolved := map[string]map[string]string{
		"api": {"GREETING": "hello", "DB_PASSWORD": "supersecret"},
		"web": {"REGION": "westus"},
	}
	names := []string{"api", "web"}

	t.Run("dotenv groups services under headers and masks secrets", func(t *testing.T) {
		resetEnvFlags()
		out, runErr := captureStdout(t, func() error {
			return renderAllEnv(resolved, names, envFormatDotenv, true)
		})
		require.NoError(t, runErr)
		assert.Contains(t, out, "# api")
		assert.Contains(t, out, "GREETING=hello")
		assert.Contains(t, out, "# web")
		assert.Contains(t, out, "REGION=westus")
		assert.NotContains(t, out, "supersecret")
	})

	t.Run("no-mask keeps raw secret values", func(t *testing.T) {
		resetEnvFlags()
		out, runErr := captureStdout(t, func() error {
			return renderAllEnv(resolved, names, envFormatDotenv, false)
		})
		require.NoError(t, runErr)
		assert.Contains(t, out, "DB_PASSWORD=supersecret")
	})

	t.Run("json emits object keyed by service", func(t *testing.T) {
		resetEnvFlags()
		out, runErr := captureStdout(t, func() error {
			return renderAllEnv(resolved, names, envFormatJSON, false)
		})
		require.NoError(t, runErr)
		var parsed map[string]map[string]string
		require.NoError(t, json.Unmarshal([]byte(out), &parsed))
		assert.Equal(t, "hello", parsed["api"]["GREETING"])
		assert.Equal(t, "westus", parsed["web"]["REGION"])
	})

	t.Run("no services reports an empty message", func(t *testing.T) {
		resetEnvFlags()
		out, runErr := captureStdout(t, func() error {
			return renderAllEnv(map[string]map[string]string{}, nil, envFormatDotenv, true)
		})
		require.NoError(t, runErr)
		assert.Contains(t, out, "No services are defined")
	})
}

// resetEnvFlags clears the package-level env command flag state between runs so
// values set by one subtest do not leak into the next. It also resets the global
// cliout output format so a JSON format leaked from another test does not force
// the env command into JSON output.
func resetEnvFlags() {
	envFormat = envFormatDotenv
	envNoMask = false
	envFile = ""
	envAll = false
	envExplain = false
	_ = cliout.SetFormat("default")
}

func writeEnvTestAzureYaml(t *testing.T, dir string) {
	t.Helper()
	content := `name: envtestapp
services:
  api:
    host: containerapp
    language: python
    project: ./api
  web:
    host: containerapp
    language: js
    project: ./web
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte(content), 0o600))
}

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)

	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	// Drain the pipe concurrently so writes larger than the pipe buffer do not
	// block the function under test.
	done := make(chan string, 1)
	go func() {
		var sb strings.Builder
		buf := make([]byte, 4096)
		for {
			n, readErr := r.Read(buf)
			if n > 0 {
				sb.Write(buf[:n])
			}
			if readErr != nil {
				break
			}
		}
		done <- sb.String()
	}()

	runErr := fn()
	require.NoError(t, w.Close())
	out := <-done
	require.NoError(t, r.Close())
	return out, runErr
}

func splitNonEmpty(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}
