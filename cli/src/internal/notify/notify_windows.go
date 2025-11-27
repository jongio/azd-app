//go:build windows

package notify

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jongio/azd-app/cli/src/internal/constants"
)

const (
	// appUserModelID is the unique identifier for Azure Developer CLI notifications
	appUserModelID = "Microsoft.AzureDeveloperCLI"
	// shortcutName is the name of the Start Menu shortcut
	shortcutName = "Azure Developer CLI.lnk"
)

var (
	// registrationOnce ensures we only register the app once per process
	registrationOnce sync.Once
	// registrationError stores any error from registration
	registrationError error
)

// windowsNotifier implements Notifier for Windows using PowerShell and WinRT.
type windowsNotifier struct {
	config Config
}

// newPlatformNotifier creates a Windows-specific notifier.
func newPlatformNotifier(config Config) (Notifier, error) {
	notifier := &windowsNotifier{
		config: config,
	}

	// Register the app for notifications on first use
	registrationOnce.Do(func() {
		registrationError = notifier.ensureAppRegistered()
	})

	// Don't fail if registration fails - we'll try alternative methods
	if registrationError != nil {
		// Log but don't fail - notifications may still work
		fmt.Fprintf(os.Stderr, "Note: Could not register notification app: %v\n", registrationError)
	}

	return notifier, nil
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

// ensureAppRegistered registers Azure Developer CLI for Windows notifications.
// This creates a Start Menu shortcut with the proper AppUserModelID so that
// toast notifications appear as "Azure Developer CLI" instead of PowerShell.
func (w *windowsNotifier) ensureAppRegistered() error {
	// Get the Start Menu Programs path
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return fmt.Errorf("APPDATA environment variable not set")
	}

	startMenuPath := filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs")
	shortcutPath := filepath.Join(startMenuPath, shortcutName)

	// Check if shortcut already exists
	if _, err := os.Stat(shortcutPath); err == nil {
		return nil // Already exists
	}

	// Get the path to azd executable (prefer azd over azd-app)
	exePath := w.findAzdExecutable()
	if exePath == "" {
		// Fall back to current executable
		var err error
		exePath, err = os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}
	}

	// Sanitize paths to prevent injection
	safeShortcutPath := sanitizeForPowerShell(shortcutPath)
	safeExePath := sanitizeForPowerShell(exePath)

	// Create shortcut with AppUserModelID using PowerShell
	script := fmt.Sprintf(`
$shortcutPath = '%s'
$targetPath = '%s'
$appId = '%s'

# Create the shortcut using WScript.Shell
$WshShell = New-Object -ComObject WScript.Shell
$Shortcut = $WshShell.CreateShortcut($shortcutPath)
$Shortcut.TargetPath = $targetPath
$Shortcut.Description = 'Azure Developer CLI - Service Notifications'
$Shortcut.Save()

# Set AppUserModelID using PropertyStore COM interface
$shellApp = New-Object -ComObject Shell.Application
$folder = $shellApp.NameSpace((Split-Path $shortcutPath -Parent))
$item = $folder.ParseName((Split-Path $shortcutPath -Leaf))

# Use extended property to set AppUserModelID
# Property store approach via PowerShell interop
Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
using System.Runtime.InteropServices.ComTypes;

namespace ShortcutUtils {
    [ComImport, Guid("886D8EEB-8CF2-4446-8D02-CDBA1DBDCF99")]
    [InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    public interface IPropertyStore {
        int GetCount(out uint cProps);
        int GetAt(uint iProp, out PropertyKey pkey);
        int GetValue(ref PropertyKey key, out PropVariant pv);
        int SetValue(ref PropertyKey key, ref PropVariant pv);
        int Commit();
    }

    [StructLayout(LayoutKind.Sequential, Pack = 4)]
    public struct PropertyKey {
        public Guid formatId;
        public uint propertyId;
        public PropertyKey(Guid formatId, uint propertyId) {
            this.formatId = formatId;
            this.propertyId = propertyId;
        }
    }

    [StructLayout(LayoutKind.Explicit)]
    public struct PropVariant {
        [FieldOffset(0)] public ushort variantType;
        [FieldOffset(8)] public IntPtr pointerValue;

        public void SetString(string val) {
            variantType = 31; // VT_LPWSTR
            pointerValue = Marshal.StringToCoTaskMemUni(val);
        }
    }

    [ComImport, Guid("00021401-0000-0000-C000-000000000046")]
    public class ShellLink { }

    public static class ShortcutHelper {
        // PKEY_AppUserModel_ID = {9F4C2855-9F79-4B39-A8D0-E1D42DE1D5F3}, 5
        static readonly PropertyKey PKEY_AppUserModel_ID = new PropertyKey(
            new Guid("9F4C2855-9F79-4B39-A8D0-E1D42DE1D5F3"), 5);

        public static void SetAppUserModelId(string shortcutPath, string appId) {
            var shellLink = (IPersistFile)new ShellLink();
            shellLink.Load(shortcutPath, 0);

            var propStore = (IPropertyStore)shellLink;
            var pv = new PropVariant();
            pv.SetString(appId);

            propStore.SetValue(ref PKEY_AppUserModel_ID, ref pv);
            propStore.Commit();

            shellLink.Save(shortcutPath, true);
            Marshal.ReleaseComObject(shellLink);
        }
    }
}
'@

try {
    [ShortcutUtils.ShortcutHelper]::SetAppUserModelId($shortcutPath, $appId)
} catch {
    # Fallback: shortcut exists but without custom AUMID
    # Notifications will still work but may show as PowerShell
}
`,
		safeShortcutPath,
		safeExePath,
		appUserModelID)

	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to register app: %v (output: %s)", err, string(output))
	}

	return nil
}

// findAzdExecutable tries to find the azd executable in common locations
func (w *windowsNotifier) findAzdExecutable() string {
	// Try to find azd in PATH first
	if path, err := exec.LookPath("azd"); err == nil {
		return path
	}

	// Check common installation locations
	locations := []string{
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Azure Dev CLI", "azd.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "Azure Dev CLI", "azd.exe"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Azure Dev CLI", "azd.exe"),
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	return ""
}

// sanitizeForPowerShell sanitizes a string for safe use in PowerShell scripts.
// It escapes single quotes and removes potentially dangerous characters.
//
// Security considerations:
//   - Removes control characters (ASCII 0-31 except tab, newline, carriage return)
//   - Escapes single quotes (PowerShell string delimiter)
//   - Removes backticks (PowerShell escape character)
//   - Removes $ (PowerShell variable expansion)
//   - Removes semicolons (command separator)
//   - Removes parentheses (subexpression operator)
//   - Removes braces (script block delimiters)
//   - Removes pipe and redirect operators
//   - Enforces maximum length to prevent buffer issues
func sanitizeForPowerShell(s string) string {
	// Limit length to prevent buffer issues
	if len(s) > constants.MaxNotificationTextLength {
		s = s[:constants.MaxNotificationTextLength]
	}
	// Remove null bytes and other control characters that could cause issues
	s = strings.Map(func(r rune) rune {
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			return -1 // Remove control characters
		}
		return r
	}, s)
	// Escape single quotes for PowerShell strings
	s = strings.ReplaceAll(s, "'", "''")
	// Remove backticks which are PowerShell escape characters
	s = strings.ReplaceAll(s, "`", "")
	// Remove $ which could trigger variable expansion
	s = strings.ReplaceAll(s, "$", "")
	// Remove semicolons which could separate commands
	s = strings.ReplaceAll(s, ";", "")
	// Remove parentheses which could be used for subexpressions
	s = strings.ReplaceAll(s, "(", "")
	s = strings.ReplaceAll(s, ")", "")
	// Remove braces which could define script blocks
	s = strings.ReplaceAll(s, "{", "")
	s = strings.ReplaceAll(s, "}", "")
	// Remove pipe operator
	s = strings.ReplaceAll(s, "|", "")
	// Remove redirect operators
	s = strings.ReplaceAll(s, ">", "")
	s = strings.ReplaceAll(s, "<", "")
	return s
}

// sanitizeForXML sanitizes a string for safe use in XML content.
// This is used for toast notification templates which are XML-based.
//
// The function first applies PowerShell sanitization, then escapes XML entities.
// Note: < and > are already removed by sanitizeForPowerShell, but we escape
// them here as a defense-in-depth measure.
func sanitizeForXML(s string) string {
	// Limit length first
	if len(s) > constants.MaxNotificationTextLength {
		s = s[:constants.MaxNotificationTextLength]
	}
	// Escape XML special characters FIRST (before PowerShell sanitization)
	// This order is important: & must be escaped before other entities
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	// Now apply PowerShell sanitization for the script context
	// Note: We skip calling sanitizeForPowerShell here since we've already
	// done XML escaping. Instead, we remove remaining dangerous characters.
	s = strings.Map(func(r rune) rune {
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			return -1
		}
		return r
	}, s)
	// Remove PowerShell-specific dangerous characters
	s = strings.ReplaceAll(s, "`", "")
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, ";", "")
	s = strings.ReplaceAll(s, "|", "")
	return s
}

// buildToastScript builds a PowerShell script for Windows toast notifications.
func (w *windowsNotifier) buildToastScript(notification Notification) string {
	// Sanitize inputs to prevent injection attacks
	title := sanitizeForXML(notification.Title)
	message := sanitizeForXML(notification.Message)
	url := sanitizeForPowerShell(notification.URL)

	// Determine icon based on severity
	icon := "Information"
	tipIcon := "Info"
	if notification.Severity == "critical" || notification.Severity == "error" {
		icon = "Error"
		tipIcon = "Error"
	} else if notification.Severity == "warning" {
		icon = "Warning"
		tipIcon = "Warning"
	}

	// Build toast template - include launch URL if provided
	var toastTemplate string
	if url != "" {
		// Toast with clickable action to open browser
		toastTemplate = fmt.Sprintf(`
<toast activationType="protocol" launch="%s">
    <visual>
        <binding template='ToastGeneric'>
            <text>%s</text>
            <text>%s</text>
            <text placement='attribution'>Click to view logs</text>
        </binding>
    </visual>
    <actions>
        <action content='View Logs' activationType='protocol' arguments='%s' />
        <action content='Dismiss' activationType='system' arguments='dismiss' />
    </actions>
    <audio src='ms-winsoundevent:Notification.Default' />
</toast>
`, url, title, message, url)
	} else {
		// Toast without URL
		toastTemplate = fmt.Sprintf(`
<toast activationType="protocol">
    <visual>
        <binding template='ToastGeneric'>
            <text>%s</text>
            <text>%s</text>
        </binding>
    </visual>
    <audio src='ms-winsoundevent:Notification.Default' />
</toast>
`, title, message)
	}

	// Escape the template for PowerShell here-string
	toastTemplate = strings.ReplaceAll(toastTemplate, "'", "''")

	// Build PowerShell script using Windows.UI.Notifications
	// First try our registered AppUserModelID, then fall back to alternatives
	script := fmt.Sprintf(`
$ErrorActionPreference = 'SilentlyContinue'

$loaded = $false
try {
    [void][Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime]
    [void][Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime]
    $loaded = $true
} catch { }

if (-not $loaded) {
    # WinRT not available, use balloon notification
    Add-Type -AssemblyName System.Windows.Forms
    $balloon = New-Object System.Windows.Forms.NotifyIcon
    $balloon.Icon = [System.Drawing.SystemIcons]::%s
    $balloon.BalloonTipIcon = [System.Windows.Forms.ToolTipIcon]::%s
    $balloon.BalloonTipText = '%s'
    $balloon.BalloonTipTitle = '%s'
    $balloon.Visible = $true
    $balloon.ShowBalloonTip(10000)
    Start-Sleep -Milliseconds 500
    $balloon.Dispose()
    exit 0
}

# App: %s
# App IDs to try in order of preference
$appIds = @(
    '%s',
    'Microsoft.WindowsTerminal_8wekyb3d8bbwe!App',
    '{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe'
)

$template = '%s'

$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml($template)

$success = $false
foreach ($appId in $appIds) {
    try {
        $notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier($appId)
        $toast = New-Object Windows.UI.Notifications.ToastNotification $xml
        $notifier.Show($toast)
        $success = $true
        break
    } catch {
        continue
    }
}

if (-not $success) {
    # All toast methods failed, use balloon notification as last resort
    Add-Type -AssemblyName System.Windows.Forms
    $balloon = New-Object System.Windows.Forms.NotifyIcon
    $balloon.Icon = [System.Drawing.SystemIcons]::%s
    $balloon.BalloonTipIcon = [System.Windows.Forms.ToolTipIcon]::%s
    $balloon.BalloonTipText = '%s'
    $balloon.BalloonTipTitle = '%s'
    $balloon.Visible = $true
    $balloon.ShowBalloonTip(10000)
    Start-Sleep -Milliseconds 500
    $balloon.Dispose()
}
`,
		// First fallback icons
		icon, tipIcon, message, title,
		// App name comment
		w.config.AppName,
		// Our app ID (use config if set, otherwise default)
		w.getAppID(),
		// Toast template (already escaped)
		toastTemplate,
		// Second fallback icons
		icon, tipIcon, message, title)

	return script
}

// getAppID returns the App ID to use for notifications.
// Uses the config value if set, otherwise falls back to the default.
func (w *windowsNotifier) getAppID() string {
	if w.config.AppID != "" {
		return w.config.AppID
	}
	return appUserModelID
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
