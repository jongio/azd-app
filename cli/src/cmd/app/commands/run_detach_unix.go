//go:build !windows

package commands

import "syscall"

// detachSpawnAttempts returns the single process attribute set used on Unix.
// Setsid puts the child in a new session with no controlling terminal, which is
// all that is needed to survive the parent exiting. There is no job object
// equivalent to break away from, so there is nothing to retry.
func detachSpawnAttempts() []*syscall.SysProcAttr {
	return []*syscall.SysProcAttr{
		{Setsid: true},
	}
}

// isBreakawayRejected always reports false on Unix; see detachSpawnAttempts.
func isBreakawayRejected(error) bool {
	return false
}
