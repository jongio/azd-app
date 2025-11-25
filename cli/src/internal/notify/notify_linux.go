//go:build linux

package notify

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// linuxNotifier implements Notifier for Linux using notify-send.
type linuxNotifier struct {
	config Config
}

// newPlatformNotifier creates a Linux-specific notifier.
func newPlatformNotifier(config Config) (Notifier, error) {
	return &linuxNotifier{
		config: config,
	}, nil
}

// Send sends a notification using libnotify (notify-send).
func (l *linuxNotifier) Send(ctx context.Context, notification Notification) error {
	if !l.IsAvailable() {
		return ErrNotAvailable
	}

	// Use timeout from config
	ctx, cancel := context.WithTimeout(ctx, l.config.Timeout)
	defer cancel()

	// Map severity to urgency level
	urgency := "normal"
	switch notification.Severity {
	case "critical":
		urgency = "critical"
	case "warning":
		urgency = "normal"
	case "info":
		urgency = "low"
	}

	// Build notify-send command
	// notify-send [options] <summary> [body]
	args := []string{
		"--app-name=" + l.config.AppName,
		"--urgency=" + urgency,
		"--expire-time=5000", // 5 seconds for normal, -1 for critical means stays until dismissed
		notification.Title,
		notification.Message,
	}

	// Critical notifications should stay visible
	if urgency == "critical" {
		args[2] = "--expire-time=-1"
	}

	cmd := exec.CommandContext(ctx, "notify-send", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %v (output: %s)", ErrNotificationFailed, err, string(output))
	}

	return nil
}

// IsAvailable checks if notify-send is available.
func (l *linuxNotifier) IsAvailable() bool {
	// Check if notify-send is available
	_, err := exec.LookPath("notify-send")
	if err != nil {
		return false
	}

	// Check if D-Bus session bus is available (required for notifications)
	if sessionBus := os.Getenv("DBUS_SESSION_BUS_ADDRESS"); sessionBus == "" {
		return false
	}

	return true
}

// RequestPermission requests notification permissions (no-op on Linux).
// Linux doesn't require explicit permission requests.
func (l *linuxNotifier) RequestPermission(ctx context.Context) error {
	if !l.IsAvailable() {
		return ErrNotAvailable
	}
	return nil
}

// Close cleans up resources (no-op on Linux).
func (l *linuxNotifier) Close() error {
	return nil
}
