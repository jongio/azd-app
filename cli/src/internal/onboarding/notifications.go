// Package onboarding provides first-run notification setup experience
package onboarding

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/config"
)

// NotificationOnboarding manages first-run notification setup
type NotificationOnboarding struct {
	reader *bufio.Reader
}

// New creates a new notification onboarding instance
func New() *NotificationOnboarding {
	return &NotificationOnboarding{
		reader: bufio.NewReader(os.Stdin),
	}
}

// Run executes the onboarding flow
func (o *NotificationOnboarding) Run(ctx context.Context) error {
	fmt.Println("")
	fmt.Println("🔔 Welcome to Azure Dev Notifications!")
	fmt.Println("")
	fmt.Println("Stay informed about your services with real-time notifications for:")
	fmt.Println("  • Service state changes (starting, running, stopped)")
	fmt.Println("  • Health check failures")
	fmt.Println("  • Deployment completion")
	fmt.Println("  • Critical errors and warnings")
	fmt.Println("")

	// Ask if user wants to enable notifications
	fmt.Print("Enable desktop notifications? (Y/n): ")
	response, _ := o.reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	enableNotifications := response == "" || response == "y" || response == "yes"

	prefs := config.DefaultNotificationPreferences()
	prefs.OSNotifications = enableNotifications

	if enableNotifications {
		// Ask about notification severity
		fmt.Println("")
		fmt.Println("Which notifications would you like to receive?")
		fmt.Println("  1. All notifications (info, warnings, and critical)")
		fmt.Println("  2. Warnings and critical only")
		fmt.Println("  3. Critical only")
		fmt.Print("Choice (1-3) [3]: ")

		choice, _ := o.reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		if choice == "" {
			choice = "3"
		}

		switch choice {
		case "1":
			prefs.SeverityFilter = "all"
		case "2":
			prefs.SeverityFilter = "warning"
		default:
			prefs.SeverityFilter = "critical"
		}

		// Ask about quiet hours
		fmt.Println("")
		fmt.Print("Enable quiet hours (no notifications 22:00-08:00)? (y/N): ")
		response, _ = o.reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "y" || response == "yes" {
			prefs.QuietHours = []config.QuietHourRange{
				{Start: "22:00", End: "08:00"},
			}
			fmt.Println("Quiet hours set to 22:00 - 08:00")
		}

		prefs.RateLimitWindow = "30s"

		fmt.Println("")
		fmt.Println("✓ Notifications configured successfully!")
		fmt.Println("")
		fmt.Println("Notification preferences saved to ~/.azd/notifications.json")
		fmt.Println("You can modify this file directly or use 'azd app notifications' commands.")
		fmt.Println("")
	} else {
		fmt.Println("")
		fmt.Println("Notifications disabled.")
		fmt.Println("")
	}

	// Save configuration
	if err := config.SaveNotificationPreferences(prefs); err != nil {
		return fmt.Errorf("failed to save notification config: %w", err)
	}

	return nil
}

// ShouldRun checks if onboarding should be displayed
func (o *NotificationOnboarding) ShouldRun(ctx context.Context) (bool, error) {
	// Check if notifications config already exists
	prefsPath, err := config.GetNotificationPreferencesPath()
	if err != nil {
		return false, err
	}

	_, err = os.Stat(prefsPath)
	if os.IsNotExist(err) {
		// Config doesn't exist, should run onboarding
		return true, nil
	}

	// Config exists, skip onboarding
	return false, nil
}
