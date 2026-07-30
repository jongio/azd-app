package commands

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var errTestTrustDenied = errors.New("trust denied for test gate")

// newTrustGateTestProject creates a minimal detectable project and chdirs into
// it, so testing.FindAzureYaml resolves against the temp dir rather than the
// real repository.
func newTrustGateTestProject(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	azureYaml := "services:\n  web:\n    project: ./web\n    host: localhost\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte(azureYaml), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "web"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "web", "package.json"), []byte(`{"name":"web"}`), 0o600))

	t.Chdir(dir)
	return dir
}

// stubTestTrustGate swaps the package-level trust gate for the duration of a
// test so execute paths can be exercised without touching the real trust store
// in the user's home directory.
func stubTestTrustGate(t *testing.T, fn func(string) error) {
	t.Helper()

	original := ensureTestTrusted
	ensureTestTrusted = fn
	t.Cleanup(func() { ensureTestTrusted = original })
}

// A nil orchestrator is deliberate in these tests. The trust gate must return
// before any dependency command runs, so commandOrchestrator is never
// dereferenced. If the gate were ever moved after commandOrchestrator.Run the
// nil pointer would panic and fail the test.

func TestRunTestsBlocksWhenWorkspaceUntrusted(t *testing.T) {
	newTrustGateTestProject(t)
	stubTestTrustGate(t, func(string) error { return errTestTrustDenied })

	opts := &TestOptions{Type: "unit", OutputFormat: "default", Threshold: 80}

	err := runTests(nil, opts)

	require.ErrorIs(t, err, errTestTrustDenied)
}

func TestRunTestsTrustGateReceivesProjectAzureYaml(t *testing.T) {
	dir := newTrustGateTestProject(t)

	var gotPath string
	stubTestTrustGate(t, func(p string) error {
		gotPath = p
		return errTestTrustDenied
	})

	opts := &TestOptions{Type: "unit", OutputFormat: "default", Threshold: 80}

	err := runTests(nil, opts)
	require.ErrorIs(t, err, errTestTrustDenied)
	require.NotEmpty(t, gotPath)

	want, err := os.Stat(filepath.Join(dir, "azure.yaml"))
	require.NoError(t, err)
	got, err := os.Stat(gotPath)
	require.NoError(t, err)
	require.True(t, os.SameFile(want, got), "gate received %q, which is not the project azure.yaml", gotPath)
}

// Dry runs are gated too: runTests still invokes commandOrchestrator.Run,
// which installs dependencies, so a dry run is not a read-only operation.
func TestRunTestsDryRunIsStillGated(t *testing.T) {
	newTrustGateTestProject(t)

	called := false
	stubTestTrustGate(t, func(string) error {
		called = true
		return errTestTrustDenied
	})

	opts := &TestOptions{Type: "unit", OutputFormat: "default", Threshold: 80, DryRun: true}

	err := runTests(nil, opts)

	require.ErrorIs(t, err, errTestTrustDenied)
	require.True(t, called, "dry run must still consult the trust gate")
}

// Invalid flags must be rejected before the trust gate so users get the
// specific validation error rather than a trust prompt.
func TestRunTestsValidatesFlagsBeforeTrustGate(t *testing.T) {
	newTrustGateTestProject(t)

	called := false
	stubTestTrustGate(t, func(string) error {
		called = true
		return errTestTrustDenied
	})

	opts := &TestOptions{Type: "bogus", OutputFormat: "default", Threshold: 80}

	err := runTests(nil, opts)

	require.Error(t, err)
	require.NotErrorIs(t, err, errTestTrustDenied)
	require.False(t, called, "flag validation must run before the trust gate")
}
