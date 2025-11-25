package commands

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/notifications"
	"github.com/jongio/azd-app/cli/src/internal/output"
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
			output.CommandHeader("notifications list", "List notification history")
			ctx := cmd.Context()
			dbPath := getNotificationDBPath()

			db, err := notifications.NewDatabase(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer db.Close()

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
			output.CommandHeader("notifications mark-read", "Mark notification(s) as read")
			ctx := cmd.Context()
			dbPath := getNotificationDBPath()

			db, err := notifications.NewDatabase(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer db.Close()

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
			output.CommandHeader("notifications clear", "Clear notification history")
			ctx := cmd.Context()
			dbPath := getNotificationDBPath()

			db, err := notifications.NewDatabase(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer db.Close()

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

			fmt.Print("Clear all notification history? (y/N): ")
			var response string
			_, _ = fmt.Scanln(&response)

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
			output.CommandHeader("notifications stats", "Show notification statistics")
			ctx := cmd.Context()
			dbPath := getNotificationDBPath()

			db, err := notifications.NewDatabase(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer db.Close()

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
	fmt.Fprintln(w, "ID\tSERVICE\tSEVERITY\tMESSAGE\tTIME\tREAD")

	for _, r := range records {
		readIcon := " "
		if r.Read {
			readIcon = "✓"
		}

		relTime := formatRelativeTime(r.Timestamp)
		message := r.Message
		if len(message) > 50 {
			message = message[:47] + "..."
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
			r.ID,
			r.ServiceName,
			r.Severity,
			message,
			relTime,
			readIcon,
		)
	}

	w.Flush()
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
	// Use XDG_DATA_HOME or fallback to ~/.local/share
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = home + "/.local/share"
	}
	return dataHome + "/azd/notifications.db"
}
