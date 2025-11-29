// Package constants provides shared constants used across the CLI.
package constants

import "time"

// File permissions
const (
	// DirPermission is the default permission for creating directories (rwxr-x---)
	DirPermission = 0750
	// FilePermission is the default permission for creating files (rw-r--r--)
	FilePermission = 0644
)

// Dashboard configuration
const (
	// DashboardServiceName is the service name used for port manager assignments
	DashboardServiceName = "azd-app-dashboard"
	// DashboardPortRangeMin is the minimum port for dashboard (ephemeral range)
	DashboardPortRangeMin = 40000
	// DashboardPortRangeMax is the maximum port for dashboard (ephemeral range)
	DashboardPortRangeMax = 49999
	// MaxPortRetries is the maximum number of port binding retries
	MaxPortRetries = 15
	// DefaultLogTail is the default number of log lines to return
	DefaultLogTail = 500
	// MaxLogTail is the maximum number of log lines to return
	MaxLogTail = 10000
	// LogChannelBufferSize is the buffer size for log streaming channels
	LogChannelBufferSize = 100
)

// Timeouts
const (
	// DefaultBrowserTimeout is the default timeout for browser launch operations
	DefaultBrowserTimeout = 5 * time.Second
	// DefaultNotificationTimeout is the default timeout for sending notifications
	DefaultNotificationTimeout = 5 * time.Second
	// ServerReadHeaderTimeout is the timeout for reading HTTP request headers
	ServerReadHeaderTimeout = 10 * time.Second
	// GracefulShutdownTimeout is the timeout for graceful service shutdown
	GracefulShutdownTimeout = 10 * time.Second
	// FunctionsStartupDelay is the delay to wait for Functions to start up
	FunctionsStartupDelay = 2 * time.Second
	// ServiceStopWaitTime is the time to wait after stopping a service
	ServiceStopWaitTime = 500 * time.Millisecond
	// CleanupInterval is the interval for cleanup operations
	CleanupInterval = 5 * time.Minute
	// DefaultMonitorInterval is the default interval for state monitoring
	DefaultMonitorInterval = 5 * time.Second
	// ServerStartupDelay is the delay to wait for HTTP server startup
	ServerStartupDelay = 100 * time.Millisecond
	// ToastAutoDismissTimeout is the timeout for auto-dismissing toast notifications
	ToastAutoDismissTimeout = 10 * time.Second
)

// Buffer sizes
const (
	// DefaultPipelineBufferSize is the default buffer size for notification pipelines
	DefaultPipelineBufferSize = 100
	// DefaultNotificationLimit is the default limit for notification queries
	DefaultNotificationLimit = 50
)

// Severity levels
const (
	SeverityCritical = "critical"
	SeverityWarning  = "warning"
	SeverityError    = "error"
	SeverityInfo     = "info"
)

// Pattern/Override source types
const (
	SourceUser = "user"
	SourceApp  = "app"
)

// Service status values
const (
	StatusRunning    = "running"
	StatusStopped    = "stopped"
	StatusStarting   = "starting"
	StatusReady      = "ready"
	StatusNotRunning = "not-running"
	StatusError      = "error"
	StatusStopping   = "stopping"
)

// Service health values
const (
	HealthHealthy   = "healthy"
	HealthUnhealthy = "unhealthy"
	HealthUnknown   = "unknown"
)

// Pattern field limits
const (
	// MaxPatternNameLength is the maximum length for pattern names
	MaxPatternNameLength = 100
	// MaxPatternLength is the maximum length for pattern regex
	MaxPatternLength = 500
	// MaxPatternDescriptionLength is the maximum length for pattern descriptions
	MaxPatternDescriptionLength = 500
	// MaxOverrideTextLength is the maximum length for override text
	MaxOverrideTextLength = 500
	// MaxNotificationTextLength is the maximum length for notification text (title, message, URL)
	MaxNotificationTextLength = 500
)

// UI constraints
const (
	// MinGridColumns is the minimum number of grid columns for logs view
	MinGridColumns = 1
	// MaxGridColumns is the maximum number of grid columns for logs view
	MaxGridColumns = 6
	// DefaultGridColumns is the default number of grid columns
	DefaultGridColumns = 2
)

// WebSocket reconnection
const (
	// WebSocketInitialReconnectDelay is the initial delay before reconnecting
	WebSocketInitialReconnectDelay = 1 * time.Second
	// WebSocketMaxReconnectDelay is the maximum delay between reconnection attempts
	WebSocketMaxReconnectDelay = 30 * time.Second
	// WebSocketMaxReconnectAttempts is the maximum number of reconnection attempts (0 = unlimited)
	WebSocketMaxReconnectAttempts = 0
)
