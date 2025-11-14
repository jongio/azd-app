package portmanager

import "time"

const (
	// PortRangeStart is the minimum port number for dynamic port assignment.
	// Starts at 3000 to avoid well-known ports (0-1023) and registered ports (1024-2999)
	// which often require admin privileges and conflict with system services.
	PortRangeStart = 3000

	// PortRangeEnd is the maximum port number (standard TCP/IP limit).
	PortRangeEnd = 65535

	// ProcessKillGracePeriod is the time to wait after sending kill signal
	// for the OS to reclaim resources and free ports.
	ProcessKillGracePeriod = 100 * time.Millisecond

	// ProcessKillMaxRetries is the maximum number of times to verify
	// a port is freed after killing a process.
	ProcessKillMaxRetries = 10

	// ProcessKillTimeout is the maximum time to wait for a process kill command.
	ProcessKillTimeout = 5 * time.Second

	// StalePortCleanupAge is the age threshold for cleaning up stale port assignments.
	// Assignments older than this are removed during cleanup.
	StalePortCleanupAge = 7 * 24 * time.Hour
)
