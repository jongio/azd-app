package service

import "time"

const (
	// DefaultMaxLogLines is the default maximum number of log lines to buffer per service.
	// Chosen to balance memory usage with sufficient log history for debugging.
	DefaultMaxLogLines = 1000

	// DefaultHealthCheckTimeout is the default timeout for health check operations.
	DefaultHealthCheckTimeout = 30 * time.Second

	// DefaultServiceStartTimeout is the default timeout waiting for a service to start.
	DefaultServiceStartTimeout = 5 * time.Minute

	// DefaultGracefulShutdownTimeout is the time to wait for graceful shutdown before forced kill.
	DefaultGracefulShutdownTimeout = 10 * time.Second
)
