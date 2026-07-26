//go:build !windows

package commands

import (
	"errors"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetachSpawnAttempts(t *testing.T) {
	attempts := detachSpawnAttempts()
	require.Len(t, attempts, 1, "Unix has no job object, so there is nothing to retry")
	require.NotNil(t, attempts[0])
	assert.True(t, attempts[0].Setsid)
}

func TestIsBreakawayRejected(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "nil", err: nil},
		{name: "permission denied", err: syscall.EACCES},
		{name: "unrelated error", err: errors.New("boom")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.False(t, isBreakawayRejected(tt.err), "Unix never retries a spawn")
		})
	}
}
