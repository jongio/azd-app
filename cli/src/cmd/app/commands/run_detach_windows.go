//go:build windows

package commands

import (
	"errors"
	"syscall"
)

// Windows process creation flags used when spawning the background run.
const (
	detachedProcess        = 0x00000008
	createNewProcessGroup  = 0x00000200
	createBreakawayFromJob = 0x01000000
)

// detachSpawnAttempts returns the process attributes to try, most preferred first.
//
// DETACHED_PROCESS only releases the console; it does not remove the child from
// the parent's job object. Terminals, IDEs, and CI agents commonly run azd
// inside a job with JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE, so the "detached" child
// is killed the moment the parent exits. CREATE_BREAKAWAY_FROM_JOB escapes that
// job, but it fails with ERROR_ACCESS_DENIED when the job does not set
// JOB_OBJECT_LIMIT_BREAKAWAY_OK. The second attempt drops the flag so those
// environments keep the previous behaviour instead of failing to start at all.
func detachSpawnAttempts() []*syscall.SysProcAttr {
	return []*syscall.SysProcAttr{
		{CreationFlags: createBreakawayFromJob | detachedProcess | createNewProcessGroup},
		{CreationFlags: detachedProcess | createNewProcessGroup},
	}
}

// isBreakawayRejected reports whether a start failure was the job object
// refusing the breakaway request, which means the next attempt should be tried.
func isBreakawayRejected(err error) bool {
	return errors.Is(err, syscall.ERROR_ACCESS_DENIED)
}
