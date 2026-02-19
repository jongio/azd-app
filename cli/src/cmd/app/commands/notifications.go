package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"
	"unicode/utf8"

	"github.com/jongio/azd-app/cli/src/internal/config"
	"github.com/jongio/azd-app/cli/src/internal/notifications"
	"github.com/jongio/azd-core/cliout"
	"github.com/jongio/azd-core/notify"
	"github.com/spf13/cobra"
)

// NewNotificationsCommand creates the notifications command.
func NewNotificationsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notifications",
		Short: "Manage process notifications",
		Long:  "View and manage notifications for service state changes and events",
	}

	cmd.AddCommand(newNotificationsListCmd())
	cmd.AddCommand(newNotificationsMarkReadCmd())
	cmd.AddCommand(newNotificationsClearCmd())
	cmd.AddCommand(newNotificationsStatsCmd())
	cmd.AddCommand(newNotificationsTestCmd())
	cmd.AddCommand(newNotificationsEnableCmd())

	return cmd
}

func newNotificationsListCmd() *cobra.Command {
	var (
		serviceName string
		unreadOnly  bool
		limit       int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List notification history",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliout.CommandHeader("notifications list", "List notification history")
			ctx := cmd.Context()
			dbPath := getNotificationDBPath()

			db, err := notifications.NewDatabase(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() { _ = db.Close() }()

			var records []notifications.NotificationRecord
			if serviceName != "" {
				records, err = db.GetByService(ctx, serviceName, limit)
			} else if unreadOnly {
				records, err = db.GetUnread(ctx)
			} else {
				records, err = db.GetRecent(ctx, limit)
			}

			if err != nil {
				return fmt.Errorf("failed to query notifications: %w", err)
			}

			printNotifications(records)
			return nil
		},
	}

	cmd.Flags().StringVarP(&serviceName, "service", "s", "", "Filter by service name")
	cmd.Flags().BoolVarP(&unreadOnly, "unread", "u", false, "Show only unread notifications")
	cmd.Flags().IntVarP(&limit, "limit", "n", 50, "Maximum number of notifications to show")

	return cmd
}

func newNotificationsMarkReadCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "mark-read [id]",
		Short: "Mark notification(s) as read",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliout.CommandHeader("notifications mark-read", "Mark notification(s) as read")
			ctx := cmd.Context()
			dbPath := getNotificationDBPath()

			db, err := notifications.NewDatabase(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() { _ = db.Close() }()

			if all {
				if err := db.MarkAllAsRead(ctx); err != nil {
					return fmt.Errorf("failed to mark all as read: %w", err)
				}
				fmt.Println("All notifications marked as read")
				return nil
			}

			if len(args) == 0 {
				return fmt.Errorf("notification ID required (or use --all)")
			}

			var id int64
			if _, err := fmt.Sscanf(args[0], "%d", &id); err != nil {
				return fmt.Errorf("invalid notification ID: %w", err)
			}

			// Validate ID is positive
			if id <= 0 {
				return fmt.Errorf("notification ID must be positive, got %d", id)
			}

			if err := db.MarkAsRead(ctx, id); err != nil {
				return fmt.Errorf("failed to mark as read: %w", err)
			}

			fmt.Printf("Notification %d marked as read\n", id)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Mark all notifications as read")

	return cmd
}

func newNotificationsClearCmd() *cobra.Command {
	var olderThan string

	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear notification history",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliout.CommandHeader("notifications clear", "Clear notification history")
			ctx := cmd.Context()
			dbPath := getNotificationDBPath()

			db, err := notifications.NewDatabase(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() { _ = db.Close() }()

			if olderThan != "" {
				duration, err := time.ParseDuration(olderThan)
				if err != nil {
					return fmt.Errorf("invalid duration: %w", err)
				}

				if err := db.ClearOld(ctx, duration); err != nil {
					return fmt.Errorf("failed to clear old notifications: %w", err)
				}
				fmt.Printf("Cleared notifications older than %s\n", olderThan)
				return nil
			}

			// Interactive confirmation with context-aware input
			fmt.Print("Clear all notification history? (y/N): ")

			// Create a channel to read user input
			responseChan := make(chan string, 1)
			go func() {
				var response string
				_, _ = fmt.Scanln(&response)
				responseChan <- response
			}()

			// Wait for user input or context cancellation
			var response string
			select {
			case response = <-responseChan:
				// User provided input
			case <-ctx.Done():
				fmt.Println("\nCancelled by context")
				return ctx.Err()
			}

			if response != "y" && response != "Y" {
				fmt.Println("Cancelled")
				return nil
			}

			if err := db.ClearAll(ctx); err != nil {
				return fmt.Errorf("failed to clear notifications: %w", err)
			}

			fmt.Println("All notifications cleared")
			return nil
		},
	}

	cmd.Flags().StringVar(&olderThan, "older-than", "", "Clear notifications older than duration (e.g., 24h, 7d)")

	return cmd
}

func newNotificationsStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show notification statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliout.CommandHeader("notifications stats", "Show notification statistics")
			ctx := cmd.Context()
			dbPath := getNotificationDBPath()

			db, err := notifications.NewDatabase(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() { _ = db.Close() }()

			stats, err := db.GetStats(ctx)
			if err != nil {
				return fmt.Errorf("failed to get stats: %w", err)
			}

			fmt.Println("Notification Statistics:")
			fmt.Printf("  Total:    %d\n", stats["total"])
			fmt.Printf("  Unread:   %d\n", stats["unread"])
			fmt.Printf("  Critical: %d\n", stats["critical"])

			return nil
		},
	}

	return cmd
}

func printNotifications(records []notifications.NotificationRecord) {
	if len(records) == 0 {
		fmt.Println("No notifications found")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tSERVICE\tSEVERITY\tMESSAGE\tTIME\tREAD")

	for _, r := range records {
		readIcon := " "
		if r.Read {
			readIcon = "✓"
		}

		relTime := formatRelativeTime(r.Timestamp)
		message := r.Message
		if len(message) > 50 {
			message = truncateUTF8(message, 47) + "..."
		}

		_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
			r.ID,
			r.ServiceName,
			r.Severity,
			message,
			relTime,
			readIcon,
		)
	}

	_ = w.Flush()
}

func formatRelativeTime(t time.Time) string {
	diff := time.Since(t)

	if diff < time.Minute {
		return "just now"
	}
	if diff < time.Hour {
		mins := int(diff.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	}
	if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("%dh ago", hours)
	}
	days := int(diff.Hours() / 24)
	return fmt.Sprintf("%dd ago", days)
}

func getNotificationDBPath() string {
	// Use XDG_DATA_HOME on Linux, or platform-specific user data directory
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			// Fallback to temp directory if home directory is unavailable
			return filepath.Join(os.TempDir(), "azd", "notifications.db")
		}
		// Platform-specific data directory
		// Windows: %LOCALAPPDATA%\azd, Unix: ~/.local/share/azd
		switch {
		case os.Getenv("LOCALAPPDATA") != "":
			dataHome = filepath.Join(os.Getenv("LOCALAPPDATA"), "azd")
		default:
			dataHome = filepath.Join(home, ".local", "share", "azd")
		}
	} else {
		dataHome = filepath.Join(dataHome, "azd")
	}
	return filepath.Join(dataHome, "notifications.db")
}

func newNotificationsTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Send a test notification",
		Long:  "Send a test OS notification to verify notifications are working",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliout.CommandHeader("notifications test", "Send test notification")

			// Check if notifications are enabled
			prefs := config.GetGlobalNotificationPreferences()
			if !prefs.OSNotifications {
				cliout.Warning("OS notifications are disabled in preferences")
				cliout.Info("Run 'azd app notifications enable' to enable them")
				return nil
			}

			// Create notifier
			notifyConfig := notify.DefaultConfig()
			notifier, err := notify.New(notifyConfig)
			if err != nil {
				return fmt.Errorf("failed to create notifier: %w", err)
			}
			defer func() { _ = notifier.Close() }()

			// Check if available
			if !notifier.IsAvailable() {
				cliout.Error("OS notifications are not available on this system")
				cliout.Info("On Windows, ensure PowerShell is available")
				cliout.Info("On macOS, notification permissions may be needed")
				cliout.Info("On Linux, ensure notify-send is installed")
				return nil
			}

			// Send test notification with a sample URL
			notification := notify.Notification{
				Title:     "Azure Dev Test",
				Message:   "🎉 Notifications are working! Click to open the dashboard.",
				Severity:  "info",
				Timestamp: time.Now(),
				URL:       "http://localhost:5173", // Default dev dashboard URL for testing
			}

			cliout.Info("Sending test notification...")
			if err := notifier.Send(context.Background(), notification); err != nil {
				return fmt.Errorf("failed to send notification: %w", err)
			}

			cliout.Success("Test notification sent!")
			cliout.Info("Check your system notification center to verify it appeared")
			cliout.Info("Click the notification or 'View Logs' button to open browser")
			return nil
		},
	}

	return cmd
}

func newNotificationsEnableCmd() *cobra.Command {
	var disable bool

	cmd := &cobra.Command{
		Use:   "enable",
		Short: "Enable or disable OS notifications",
		Long:  "Enable or disable OS notifications for service state changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliout.CommandHeader("notifications enable", "Configure OS notifications")

			// Load current preferences
			prefs, err := config.LoadNotificationPreferences()
			if err != nil {
				return fmt.Errorf("failed to load preferences: %w", err)
			}

			if disable {
				prefs.OSNotifications = false
				cliout.Info("Disabling OS notifications...")
			} else {
				prefs.OSNotifications = true
				cliout.Info("Enabling OS notifications...")
			}

			// Save preferences
			if err := config.SaveNotificationPreferences(prefs); err != nil {
				return fmt.Errorf("failed to save preferences: %w", err)
			}

			if disable {
				cliout.Success("OS notifications disabled")
			} else {
				cliout.Success("OS notifications enabled")
				cliout.Info("Run 'azd app notifications test' to verify they work")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&disable, "disable", false, "Disable OS notifications instead of enabling")

	return cmd
}

// truncateUTF8 safely truncates a UTF-8 string to maxLen bytes,
// ensuring we don't cut in the middle of a multi-byte character.
func truncateUTF8(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	// Find the last valid UTF-8 character boundary before maxLen
	for i := maxLen; i >= 0; i-- {
		if utf8.RuneStart(s[i]) {
			return s[:i]
		}
	}
	return s[:maxLen]
}
