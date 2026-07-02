package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetStopFlags restores the package-level stop flag vars so tests do not leak
// state into one another.
func resetStopFlags(t *testing.T) {
	t.Helper()
	prevService, prevAll, prevYes := stopService, stopAll, stopYes
	t.Cleanup(func() {
		stopService, stopAll, stopYes = prevService, prevAll, prevYes
	})
	stopService, stopAll, stopYes = "", false, false
}

func TestNewStopCommandFlags(t *testing.T) {
	cmd := NewStopCommand()

	tests := []struct {
		name       string
		flag       string
		shorthand  string
		defaultVal string
	}{
		{name: "service flag", flag: "service", shorthand: "s", defaultVal: ""},
		{name: "all flag", flag: "all", shorthand: "", defaultVal: "false"},
		{name: "yes flag", flag: "yes", shorthand: "y", defaultVal: "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := cmd.Flags().Lookup(tt.flag)
			require.NotNil(t, f, "flag %q should be defined", tt.flag)
			assert.Equal(t, tt.shorthand, f.Shorthand)
			assert.Equal(t, tt.defaultVal, f.DefValue)
		})
	}
}

func TestRunStopServicesAllNoRunningServices(t *testing.T) {
	resetStopFlags(t)
	t.Chdir(t.TempDir())

	stopAll = true

	// With no registered/running services, --all should report that there is
	// nothing to stop and return without error rather than tearing down.
	err := runStopServices()
	assert.NoError(t, err)
}

func TestRunStopRoutesToServicesWhenServiceSet(t *testing.T) {
	resetStopFlags(t)
	t.Chdir(t.TempDir())

	// --all with an empty registry exercises the service-scoped path, which
	// must succeed (no-op) instead of attempting whole-app teardown.
	stopAll = true
	require.NoError(t, runStop(nil, nil))
}
