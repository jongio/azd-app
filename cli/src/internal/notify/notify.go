// Package notify provides cross-platform OS notification support.
package notify

import (
	"context"
	"fmt"
	"time"
)

// Notification represents a notification to be displayed.
type Notification struct {
	// Title is the notification title (typically service name)
	Title string

	// Message is the notification body (status description)
	Message string

	// Severity indicates the notification severity
	Severity string // "critical", "warning", "info"

	// Timestamp when the notification was created
	Timestamp time.Time

	// Actions are optional actions the user can take
	Actions []Action

	// Data contains arbitrary data associated with the notification
	Data map[string]string
}

// Action represents a notification action button.
type Action struct {
	ID    string
	Label string
}

// Notifier is the interface for platform-specific notification systems.
type Notifier interface {
	// Send sends a notification to the OS notification system.
	Send(ctx context.Context, notification Notification) error

	// IsAvailable returns true if OS notifications are available and permitted.
	IsAvailable() bool

	// RequestPermission requests notification permissions from the OS.
	// Returns nil if permissions granted, error otherwise.
	RequestPermission(ctx context.Context) error

	// Close cleans up notification system resources.
	Close() error
}

// Config contains notification system configuration.
type Config struct {
	// AppName is the application name shown in notifications
	AppName string

	// AppID is the platform-specific application identifier
	AppID string

	// Timeout for notification operations
	Timeout time.Duration
}

// DefaultConfig returns default notification configuration.
func DefaultConfig() Config {
	return Config{
		AppName: "Azure Developer CLI",
		AppID:   "com.microsoft.azd",
		Timeout: 5 * time.Second,
	}
}

// New creates a new platform-specific notifier.
func New(config Config) (Notifier, error) {
	return newPlatformNotifier(config)
}

// Error types
var (
	ErrNotAvailable       = fmt.Errorf("OS notifications not available")
	ErrPermissionDenied   = fmt.Errorf("notification permissions denied")
	ErrNotificationFailed = fmt.Errorf("failed to send notification")
	ErrTimeout            = fmt.Errorf("notification timeout")
)
