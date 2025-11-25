//go:build darwin

package notify

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// darwinNotifier implements Notifier for macOS using osascript.
type darwinNotifier struct {
	config Config
}

// newPlatformNotifier creates a macOS-specific notifier.
func newPlatformNotifier(config Config) (Notifier, error) {
	return &darwinNotifier{
		config: config,
	}, nil
}

// Send sends a notification using macOS notification system.
func (d *darwinNotifier) Send(ctx context.Context, notification Notification) error {
	if !d.IsAvailable() {
		return ErrNotAvailable
	}

	// Use timeout from config
	ctx, cancel := context.WithTimeout(ctx, d.config.Timeout)
	defer cancel()

	// Escape quotes in strings
	title := strings.ReplaceAll(notification.Title, "\"", "\\\"")
	message := strings.ReplaceAll(notification.Message, "\"", "\\\"")
	subtitle := strings.ReplaceAll(d.config.AppName, "\"", "\\\"")

	// Build AppleScript to send notification
	script := fmt.Sprintf(`display notification "%s" with title "%s" subtitle "%s"`,
		message, title, subtitle)

	// Execute osascript with the AppleScript
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %v (output: %s)", ErrNotificationFailed, err, string(output))
	}

	return nil
}

// IsAvailable checks if macOS notification system is available.
func (d *darwinNotifier) IsAvailable() bool {
	// Check if osascript is available
	_, err := exec.LookPath("osascript")
	return err == nil
}

// RequestPermission requests notification permissions.
// On macOS, first notification triggers permission prompt automatically.
func (d *darwinNotifier) RequestPermission(ctx context.Context) error {
	if !d.IsAvailable() {
		return ErrNotAvailable
	}

	// Send a test notification to trigger permission prompt
	testNotification := Notification{
		Title:    d.config.AppName,
		Message:  "Notifications enabled",
		Severity: "info",
	}

	return d.Send(ctx, testNotification)
}

// Close cleans up resources (no-op on macOS).
func (d *darwinNotifier) Close() error {
	return nil
}
