package main

import (
	"context"
	"errors"
	"os/exec"
	"testing"

	"github.com/jongio/azd-core/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type environmentRunner struct {
	output []byte
	err    error
	calls  [][]string
}

func (r *environmentRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	return r.output, r.err
}

func TestMissingSelectedEnvironment(t *testing.T) {
	exitErr := &exec.ExitError{}
	tests := []struct {
		name    string
		list    string
		listErr error
		loadErr error
		want    string
		missing bool
	}{
		{name: "available environments", list: `[{"Name":"stage","IsDefault":true},{"Name":"production","IsDefault":false}]`, loadErr: exitErr, want: "stage (default)", missing: true},
		{name: "no environments", list: `[]`, loadErr: exitErr, want: "Available environments: none", missing: true},
		{name: "existing environment failure", list: `[{"Name":"doesnotexist"}]`, loadErr: exitErr},
		{name: "listing failure", listErr: errors.New("list failed"), loadErr: exitErr, want: "unable to list available environments"},
		{name: "malformed listing", list: `not JSON`, loadErr: exitErr, want: "unable to parse available environments"},
		{name: "invalid listing", list: `[{}]`, loadErr: exitErr, want: "invalid environment list"},
		{name: "null listing", list: `null`, loadErr: exitErr, want: "invalid environment list"},
		{name: "non subprocess error", loadErr: errors.New("invalid environment values")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			loader := &environmentRunner{err: tc.loadErr}
			previous := env.SetCommandRunner(loader)
			t.Cleanup(func() { env.SetCommandRunner(previous) })
			lister := &environmentRunner{output: []byte(tc.list), err: tc.listErr}
			previousLister := environmentListRunner
			environmentListRunner = lister
			t.Cleanup(func() { environmentListRunner = previousLister })
			withDetachedChild(t, false)

			err := runPreRun(t, "doesnotexist")
			require.Error(t, err)
			assert.ErrorIs(t, err, tc.loadErr)
			var missing *missingEnvironmentError
			assert.Equal(t, tc.missing, errors.As(err, &missing))
			if tc.want != "" {
				assert.Contains(t, err.Error(), tc.want)
			}
			if tc.missing {
				assert.Contains(t, err.Error(), `environment "doesnotexist" does not exist`)
				assert.Contains(t, err.Error(), "azd env new doesnotexist")
				assert.Contains(t, err.Error(), "Run without -e")
			}
			if tc.name == "non subprocess error" {
				assert.Empty(t, lister.calls)
			} else {
				assert.Equal(t, [][]string{{"azd", "env", "list", "--output", "json"}}, lister.calls)
			}
		})
	}
}

func TestCancelledEnvironmentLoadSkipsDiagnosis(t *testing.T) {
	loader := &environmentRunner{err: &exec.ExitError{}}
	previous := env.SetCommandRunner(loader)
	t.Cleanup(func() { env.SetCommandRunner(previous) })
	lister := &environmentRunner{}
	previousLister := environmentListRunner
	environmentListRunner = lister
	t.Cleanup(func() { environmentListRunner = previousLister })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.Error(t, loadSelectedEnvironment(ctx, "stage"))
	assert.Empty(t, lister.calls)
}
