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

	// DefaultStopTimeout is the default timeout for graceful service shutdown.
	DefaultStopTimeout = 5 * time.Second

	// DefaultCommandTimeout is the default timeout for executing service commands.
	DefaultCommandTimeout = 30 * time.Minute

	// DefaultLogSubscriberTimeout is the timeout for sending log entries to slow subscribers.
	DefaultLogSubscriberTimeout = 10 * time.Millisecond

	// DefaultWebSocketPongWait is the timeout for receiving pong messages from WebSocket clients.
	DefaultWebSocketPongWait = 60 * time.Second

	// DefaultWebSocketPingPeriod is the period for sending ping messages (must be less than PongWait).
	DefaultWebSocketPingPeriod = 54 * time.Second // (PongWait * 9) / 10

	// DefaultWebSocketWriteTimeout is the timeout for WebSocket write operations.
	DefaultWebSocketWriteTimeout = 10 * time.Second
)
