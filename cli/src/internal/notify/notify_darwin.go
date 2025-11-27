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

// maxAppleScriptStringLength is the maximum allowed length for AppleScript strings
// to prevent potential resource exhaustion.
const maxAppleScriptStringLength = 1000

// sanitizeForAppleScript sanitizes a string for safe use in AppleScript.
// This prevents command injection attacks when executing osascript.
func sanitizeForAppleScript(s string) string {
	// Truncate to prevent resource exhaustion
	if len(s) > maxAppleScriptStringLength {
		s = s[:maxAppleScriptStringLength]
	}

	// Replace backslashes first (before other escapes that add backslashes)
	s = strings.ReplaceAll(s, "\\", "\\\\")
	// Escape double quotes
	s = strings.ReplaceAll(s, "\"", "\\\"")
	// Remove control characters that could be used for injection
	s = strings.Map(func(r rune) rune {
		// Keep printable ASCII and common Unicode, remove control chars
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return -1 // Remove character
		}
		return r
	}, s)

	return s
}

// Send sends a notification using macOS notification system.
func (d *darwinNotifier) Send(ctx context.Context, notification Notification) error {
	if !d.IsAvailable() {
		return ErrNotAvailable
	}

	// Use timeout from config
	ctx, cancel := context.WithTimeout(ctx, d.config.Timeout)
	defer cancel()

	// Sanitize strings to prevent AppleScript injection
	title := sanitizeForAppleScript(notification.Title)
	message := sanitizeForAppleScript(notification.Message)
	subtitle := sanitizeForAppleScript(d.config.AppName)

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
