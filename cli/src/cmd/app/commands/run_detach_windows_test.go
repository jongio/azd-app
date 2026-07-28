//go:build windows

package commands

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetachSpawnAttempts(t *testing.T) {
	attempts := detachSpawnAttempts()
	require.Len(t, attempts, 2, "must offer a breakaway attempt and a fallback")

	assert.Equal(
		t,
		uint32(createBreakawayFromJob|detachedProcess|createNewProcessGroup),
		attempts[0].CreationFlags,
		"first attempt must request breakaway so a kill-on-close job cannot take the child down",
	)
	assert.Equal(
		t,
		uint32(detachedProcess|createNewProcessGroup),
		attempts[1].CreationFlags,
		"fallback must preserve the previous flags for jobs that forbid breakaway",
	)
}

func TestIsBreakawayRejected(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "access denied", err: syscall.ERROR_ACCESS_DENIED, want: true},
		{
			name: "access denied wrapped by os/exec",
			err:  &fs.PathError{Op: "fork/exec", Path: "azd.exe", Err: syscall.ERROR_ACCESS_DENIED},
			want: true,
		},
		{name: "file not found", err: syscall.ERROR_FILE_NOT_FOUND, want: false},
		{name: "unrelated error", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isBreakawayRejected(tt.err))
		})
	}
}

// A job object without JOB_OBJECT_LIMIT_BREAKAWAY_OK rejects the breakaway
// request with ERROR_ACCESS_DENIED. The spawn must fall back instead of
// reporting a failure to the user.
func TestSpawnWithAttemptsFallsBackWhenBreakawayRejected(t *testing.T) {
	rejected := &fs.PathError{Op: "fork/exec", Path: "azd.exe", Err: syscall.ERROR_ACCESS_DENIED}
	attempts := detachSpawnAttempts()
	want := &os.Process{Pid: 99}

	var seen []*syscall.SysProcAttr
	got, err := spawnWithAttempts(attempts, func(attr *syscall.SysProcAttr) (*os.Process, error) {
		seen = append(seen, attr)
		if len(seen) == 1 {
			return nil, rejected
		}
		return want, nil
	})

	require.NoError(t, err)
	assert.Same(t, want, got)
	require.Len(t, seen, 2, "must retry once the breakaway request is rejected")
	assert.Equal(t, uint32(createBreakawayFromJob|detachedProcess|createNewProcessGroup), seen[0].CreationFlags)
	assert.Equal(t, uint32(detachedProcess|createNewProcessGroup), seen[1].CreationFlags)
}

func TestSpawnWithAttemptsReportsLastErrorWhenAllRejected(t *testing.T) {
	calls := 0
	got, err := spawnWithAttempts(
		detachSpawnAttempts(),
		func(*syscall.SysProcAttr) (*os.Process, error) {
			calls++
			return nil, syscall.ERROR_ACCESS_DENIED
		},
	)

	require.Error(t, err)
	assert.Nil(t, got)
	assert.Equal(t, 2, calls, "every attempt must be tried before giving up")
	assert.ErrorIs(t, err, syscall.ERROR_ACCESS_DENIED)
	assert.Contains(t, err.Error(), "start detached run process")
}
