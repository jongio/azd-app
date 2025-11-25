//go:build windows

package notify

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// windowsNotifier implements Notifier for Windows using PowerShell and WinRT.
type windowsNotifier struct {
	config Config
}

// newPlatformNotifier creates a Windows-specific notifier.
func newPlatformNotifier(config Config) (Notifier, error) {
	return &windowsNotifier{
		config: config,
	}, nil
}

// Send sends a notification using Windows Toast Notifications.
func (w *windowsNotifier) Send(ctx context.Context, notification Notification) error {
	if !w.IsAvailable() {
		return ErrNotAvailable
	}

	// Use timeout from config
	ctx, cancel := context.WithTimeout(ctx, w.config.Timeout)
	defer cancel()

	// Build PowerShell script to send toast notification
	script := w.buildToastScript(notification)

	// Execute PowerShell with the script
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %v (output: %s)", ErrNotificationFailed, err, string(output))
	}

	return nil
}

// buildToastScript builds a PowerShell script for Windows toast notifications.
func (w *windowsNotifier) buildToastScript(notification Notification) string {
	// Escape single quotes in strings
	title := strings.ReplaceAll(notification.Title, "'", "''")
	message := strings.ReplaceAll(notification.Message, "'", "''")
	appName := strings.ReplaceAll(w.config.AppName, "'", "''")

	// Build PowerShell script using Windows.UI.Notifications
	script := fmt.Sprintf(`
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null

$APP_ID = '%s'

$template = @"
<toast>
    <visual>
        <binding template='ToastGeneric'>
            <text>%s</text>
            <text>%s</text>
            <text placement='attribution'>%s</text>
        </binding>
    </visual>
    <audio src='ms-winsoundevent:Notification.Default' />
</toast>
"@

$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml($template)

$toast = New-Object Windows.UI.Notifications.ToastNotification $xml
$notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier($APP_ID)
$notifier.Show($toast)
`, w.config.AppID, title, message, appName)

	return script
}

// IsAvailable checks if Windows toast notifications are available.
func (w *windowsNotifier) IsAvailable() bool {
	// Check if PowerShell is available
	_, err := exec.LookPath("powershell.exe")
	return err == nil
}

// RequestPermission requests notification permissions (no-op on Windows).
// Windows doesn't require explicit permission requests for toast notifications.
func (w *windowsNotifier) RequestPermission(ctx context.Context) error {
	// Windows toast notifications don't require explicit permission request
	if !w.IsAvailable() {
		return ErrNotAvailable
	}
	return nil
}

// Close cleans up resources (no-op on Windows).
func (w *windowsNotifier) Close() error {
	return nil
}
