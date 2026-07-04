package commands

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-core/cliout"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCertCommand(t *testing.T) {
	t.Parallel()

	cmd := NewCertCommand()
	require.NotNil(t, cmd)
	assert.Equal(t, "cert", cmd.Use)

	tests := []struct {
		name     string
		flagName string
		flagType string
	}{
		{
			name:     "host flag",
			flagName: "host",
			flagType: "stringSlice",
		},
		{
			name:     "force flag",
			flagName: "force",
			flagType: "bool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			require.NotNil(t, flag)
			assert.Equal(t, tt.flagType, flag.Value.Type())
		})
	}
}

// withCertGlobals overrides the package-level cert globals for a test and
// restores them (and the cliout format) once the test finishes.
func withCertGlobals(t *testing.T, dirFn func() (string, error), hosts []string, force bool) {
	t.Helper()

	origDir := getCertsDir
	origHosts := certHosts
	origForce := certForce
	t.Cleanup(func() {
		getCertsDir = origDir
		certHosts = origHosts
		certForce = origForce
		_ = cliout.SetFormat("default")
	})

	getCertsDir = dirFn
	certHosts = hosts
	certForce = force
}

func TestRunCert_TextMode(t *testing.T) {
	dir := t.TempDir()
	withCertGlobals(t, func() (string, error) { return dir, nil }, nil, false)
	require.NoError(t, cliout.SetFormat("default"))

	out, err := captureStdout(t, func() error { return runCert(nil, nil) })
	require.NoError(t, err)
	assert.Contains(t, out, "Generated local certificate files")
	assert.FileExists(t, filepath.Join(dir, "ca.crt"))
	assert.FileExists(t, filepath.Join(dir, "cert.crt"))
	assert.FileExists(t, filepath.Join(dir, "cert.key"))

	// A second run with the same hosts reuses the existing certificate files.
	reused, err := captureStdout(t, func() error { return runCert(nil, nil) })
	require.NoError(t, err)
	assert.Contains(t, reused, "Reused existing local certificate files")

	// --force always regenerates the leaf certificate.
	certForce = true
	forced, err := captureStdout(t, func() error { return runCert(nil, nil) })
	require.NoError(t, err)
	assert.Contains(t, forced, "Generated local certificate files")
}

func TestRunCert_JSONMode(t *testing.T) {
	dir := t.TempDir()
	withCertGlobals(t, func() (string, error) { return dir, nil }, []string{"api.local.test"}, false)
	require.NoError(t, cliout.SetFormat("json"))

	out, err := captureStdout(t, func() error { return runCert(nil, nil) })
	require.NoError(t, err)

	var parsed certCommandOutput
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Contains(t, parsed.Hosts, "api.local.test")
	assert.Contains(t, parsed.Hosts, "localhost")
	assert.Contains(t, parsed.Hosts, "127.0.0.1")
	assert.NotEmpty(t, parsed.CACertPath)
	assert.NotEmpty(t, parsed.CertPath)
	assert.NotEmpty(t, parsed.KeyPath)
	assert.NotEmpty(t, parsed.TrustCommand)
	assert.False(t, parsed.Reused)
}

func TestRunCert_GetCertsDirError(t *testing.T) {
	withCertGlobals(t, func() (string, error) { return "", fmt.Errorf("home boom") }, nil, false)
	require.NoError(t, cliout.SetFormat("default"))

	err := runCert(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "home boom")
}

func TestRunCert_GenerateError(t *testing.T) {
	dir := t.TempDir()
	// A host containing a port separator makes certs.Generate fail during host
	// normalization, exercising runCert's error-wrapping path.
	withCertGlobals(t, func() (string, error) { return dir, nil }, []string{"bad:host"}, false)
	require.NoError(t, cliout.SetFormat("default"))

	err := runCert(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "generate certificates")
}

func TestGetCertsDir(t *testing.T) {
	dir, err := getCertsDir()
	require.NoError(t, err)
	assert.Contains(t, filepath.ToSlash(dir), ".azd/app/certs")
}

func TestTrustCommandFor(t *testing.T) {
	t.Parallel()

	const caPath = "/tmp/ca.crt"
	tests := []struct {
		goos     string
		contains string
	}{
		{goos: "windows", contains: "certutil -addstore"},
		{goos: "darwin", contains: "security add-trusted-cert"},
		{goos: "linux", contains: "update-ca-certificates"},
		{goos: "plan9", contains: "into your system trust store"},
	}

	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			t.Parallel()
			cmd := trustCommandFor(tt.goos, caPath)
			assert.Contains(t, cmd, tt.contains)
			assert.Contains(t, cmd, caPath)
		})
	}
}
