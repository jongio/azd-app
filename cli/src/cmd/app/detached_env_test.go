package main

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/jongio/azd-core/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingRunner captures the argv of every command the env loader would run
// so a test can assert that no `azd env get-values` subprocess was attempted.
type recordingRunner struct {
	mu    sync.Mutex
	calls [][]string
	out   []byte
}

func (r *recordingRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, append([]string{name}, args...))
	return r.out, nil
}

func (r *recordingRunner) invocations() [][]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

// installRecordingRunner swaps in a runner that answers instantly and restores
// the real one afterwards, so these tests never spawn a real azd process.
func installRecordingRunner(t *testing.T) *recordingRunner {
	t.Helper()
	runner := &recordingRunner{out: []byte(`{"FROM_AZD":"yes"}`)}
	prev := env.SetCommandRunner(runner)
	t.Cleanup(func() { env.SetCommandRunner(prev) })
	return runner
}

// withDetachedChild forces the detached-child answer for one test and restores
// the production detection afterwards.
func withDetachedChild(t *testing.T, detached bool) {
	t.Helper()
	prev := isDetachedChild
	isDetachedChild = func() bool { return detached }
	t.Cleanup(func() { isDetachedChild = prev })
}

// runPreRun builds the real command tree, applies -e, and runs the chained
// PersistentPreRunE exactly as the binary would.
func runPreRun(t *testing.T, envName string) error {
	t.Helper()
	cmd := newRootCmd()
	require.NoError(t, cmd.PersistentFlags().Set("environment", envName))
	require.NotNil(t, cmd.PersistentPreRunE, "root command must keep a PersistentPreRunE")
	return cmd.PersistentPreRunE(cmd, nil)
}

// TestDetachedChildSkipsEnvironmentLoad pins the guard that keeps
// `azd app run --detach` from hanging.
//
// The detached child outlives the azd host that served its parent, so shelling
// out to `azd env get-values` blocks until the context gives up and produces a
// silent, empty run.log. The child already inherits the values its parent
// resolved, so the load is redundant as well as harmful.
//
// This is a regression pin, not an aspiration: if a future refactor drops the
// guard, believing the azdext SDK now handles environment selection, this test
// fails. The SDK does not handle it. azdext.LoadAzdEnvironment shells out to the
// same `azd env get-values` and does not even pass -e.
func TestDetachedChildSkipsEnvironmentLoad(t *testing.T) {
	runner := installRecordingRunner(t)
	withDetachedChild(t, true)

	require.NoError(t, runPreRun(t, "staging"))

	assert.Empty(t, runner.invocations(),
		"a detached child must not shell out to azd; doing so hangs until the dead host times out")
}

// TestForegroundRunLoadsNamedEnvironment is the other half of the guard. The
// foreground invocation must still reload values for the environment it was
// given with -e, because azd injects the default environment into the extension
// process and only then passes -e through.
func TestForegroundRunLoadsNamedEnvironment(t *testing.T) {
	runner := installRecordingRunner(t)
	withDetachedChild(t, false)

	require.NoError(t, runPreRun(t, "staging"))

	calls := runner.invocations()
	require.Len(t, calls, 1, "foreground run must load the named environment exactly once")

	argv := strings.Join(calls[0], " ")
	assert.Contains(t, argv, "env")
	assert.Contains(t, argv, "get-values")
	assert.Contains(t, argv, "staging",
		"the -e value must reach azd, otherwise the default environment is read instead")
}

// TestNoEnvironmentFlagSkipsLoad confirms the guard's first condition. With no
// -e there is nothing to reload, so no subprocess should run at all.
func TestNoEnvironmentFlagSkipsLoad(t *testing.T) {
	runner := installRecordingRunner(t)
	withDetachedChild(t, false)

	require.NoError(t, runPreRun(t, ""))

	assert.Empty(t, runner.invocations(),
		"without -e there is no named environment to reload")
}
