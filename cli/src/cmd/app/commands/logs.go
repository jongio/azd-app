package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/security"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/spf13/cobra"
)

var (
	logsFollow     bool
	logsService    string
	logsTail       int
	logsSince      string
	logsTimestamps bool
	logsNoColor    bool
	logsLevel      string
	logsFormat     string
	logsOutput     string
)

// NewLogsCommand creates the logs command.
func NewLogsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs [service-name]",
		Short: "View logs from running services",
		Long:  `Display output logs from running services for debugging and monitoring`,
		RunE:  runLogs,
	}

	cmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output (tail -f behavior)")
	cmd.Flags().StringVarP(&logsService, "service", "s", "", "Filter by service name(s) (comma-separated)")
	cmd.Flags().IntVarP(&logsTail, "tail", "n", 100, "Number of lines to show from the end")
	cmd.Flags().StringVar(&logsSince, "since", "", "Show logs since duration (e.g., 5m, 1h)")
	cmd.Flags().BoolVar(&logsTimestamps, "timestamps", true, "Show timestamps with each log entry")
	cmd.Flags().BoolVar(&logsNoColor, "no-color", false, "Disable colored output")
	cmd.Flags().StringVar(&logsLevel, "level", "all", "Filter by log level (info, warn, error, debug, all)")
	cmd.Flags().StringVar(&logsFormat, "format", "text", "Output format (text, json)")
	cmd.Flags().StringVar(&logsOutput, "output", "", "Write logs to file instead of stdout")

	return cmd
}

func runLogs(cmd *cobra.Command, args []string) error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Determine service filter
	var serviceFilter []string
	if len(args) > 0 {
		// Service name from positional argument
		serviceFilter = []string{args[0]}
	} else if logsService != "" {
		// Service name(s) from --service flag
		serviceFilter = strings.Split(logsService, ",")
		for i := range serviceFilter {
			serviceFilter[i] = strings.TrimSpace(serviceFilter[i])
		}
	}

	// Get log manager
	logManager := service.GetLogManager(cwd)

	// Check if any services are running
	serviceNames := logManager.GetServiceNames()
	if len(serviceNames) == 0 {
		output.Info("No services are currently running")
		output.Item("Run 'azd app run' to start services")
		return nil
	}

	// Validate service filter
	if len(serviceFilter) > 0 {
		for _, filterName := range serviceFilter {
			found := false
			for _, name := range serviceNames {
				if name == filterName {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("service '%s' not found (available: %s)", filterName, strings.Join(serviceNames, ", "))
			}
		}
	}

	// Parse log level filter
	levelFilter := parseLogLevel(logsLevel)

	// Parse since duration
	var sinceTime time.Time
	if logsSince != "" {
		duration, err := time.ParseDuration(logsSince)
		if err != nil {
			return fmt.Errorf("invalid --since duration: %w", err)
		}
		sinceTime = time.Now().Add(-duration)
	}

	// Setup output writer
	output := os.Stdout
	if logsOutput != "" {
		// Validate the output path to prevent path traversal attacks
		if err := security.ValidatePath(logsOutput); err != nil {
			return fmt.Errorf("invalid output path: %w", err)
		}
		// #nosec G304 -- Path validated by security.ValidatePath above
		file, err := os.Create(logsOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
		output = file
	}

	// Get initial logs
	var logs []service.LogEntry
	if len(serviceFilter) == 0 {
		// Get logs from all services
		if logsSince != "" {
			logs = logManager.GetAllLogsSince(sinceTime)
		} else {
			logs = logManager.GetAllLogs(logsTail)
		}
	} else {
		// Get logs from specific services
		for _, serviceName := range serviceFilter {
			buffer, exists := logManager.GetBuffer(serviceName)
			if !exists {
				continue
			}

			var serviceLogs []service.LogEntry
			if logsSince != "" {
				serviceLogs = buffer.GetSince(sinceTime)
			} else {
				serviceLogs = buffer.GetRecent(logsTail)
			}
			logs = append(logs, serviceLogs...)
		}
	}

	// Filter by level
	logs = filterLogsByLevel(logs, levelFilter)

	// Display initial logs
	if logsFormat == "json" {
		displayLogsJSON(logs, output)
	} else {
		displayLogsText(logs, output, logsTimestamps, logsNoColor)
	}

	// Follow mode - subscribe to live logs
	if logsFollow {
		return followLogs(logManager, serviceFilter, levelFilter, output)
	}

	return nil
}

// followLogs subscribes to live log streams and displays them.
func followLogs(logManager *service.LogManager, serviceFilter []string, levelFilter service.LogLevel, output *os.File) error {
	// Create subscriptions
	subscriptions := make(map[string]chan service.LogEntry)

	if len(serviceFilter) == 0 {
		// Subscribe to all services
		for serviceName, buffer := range logManager.GetAllBuffers() {
			subscriptions[serviceName] = buffer.Subscribe()
		}
	} else {
		// Subscribe to specific services
		for _, serviceName := range serviceFilter {
			buffer, exists := logManager.GetBuffer(serviceName)
			if exists {
				subscriptions[serviceName] = buffer.Subscribe()
			}
		}
	}

	if len(subscriptions) == 0 {
		return fmt.Errorf("no services to follow")
	}

	// Setup signal handling for graceful exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Merge all subscription channels
	mergedChan := make(chan service.LogEntry, 100)
	for _, ch := range subscriptions {
		go func(ch chan service.LogEntry) {
			for entry := range ch {
				mergedChan <- entry
			}
		}(ch)
	}

	// Display logs as they arrive
	for {
		select {
		case entry := <-mergedChan:
			// Filter by level
			if levelFilter != -1 && entry.Level != levelFilter {
				continue
			}

			// Display log entry
			if logsFormat == "json" {
				displayLogsJSON([]service.LogEntry{entry}, output)
			} else {
				displayLogsText([]service.LogEntry{entry}, output, logsTimestamps, logsNoColor)
			}

		case <-sigChan:
			// Cleanup subscriptions
			for serviceName, ch := range subscriptions {
				buffer, exists := logManager.GetBuffer(serviceName)
				if exists {
					buffer.Unsubscribe(ch)
				}
			}
			return nil
		}
	}
}

// displayLogsText displays logs in text format.
func displayLogsText(logs []service.LogEntry, output *os.File, showTimestamps, noColor bool) {
	for _, entry := range logs {
		var line strings.Builder

		// Timestamp
		if showTimestamps {
			timestamp := entry.Timestamp.Format("15:04:05.000")
			if noColor {
				line.WriteString(fmt.Sprintf("[%s] ", timestamp))
			} else {
				line.WriteString(fmt.Sprintf("\033[90m[%s]\033[0m ", timestamp))
			}
		}

		// Service name
		if noColor {
			line.WriteString(fmt.Sprintf("[%s] ", entry.Service))
		} else {
			line.WriteString(fmt.Sprintf("\033[36m[%s]\033[0m ", entry.Service))
		}

		// Message with color based on stderr/level
		if noColor {
			line.WriteString(entry.Message)
		} else {
			if entry.IsStderr || entry.Level == service.LogLevelError {
				line.WriteString(fmt.Sprintf("\033[31m%s\033[0m", entry.Message))
			} else if entry.Level == service.LogLevelWarn {
				line.WriteString(fmt.Sprintf("\033[33m%s\033[0m", entry.Message))
			} else if entry.Level == service.LogLevelDebug {
				line.WriteString(fmt.Sprintf("\033[90m%s\033[0m", entry.Message))
			} else {
				line.WriteString(entry.Message)
			}
		}

		fmt.Fprintln(output, line.String())
	}
}

// displayLogsJSON displays logs in JSON format.
func displayLogsJSON(logs []service.LogEntry, output *os.File) {
	encoder := json.NewEncoder(output)
	for _, entry := range logs {
		if err := encoder.Encode(entry); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to encode log entry: %v\n", err)
		}
	}
}

// parseLogLevel parses a log level string.
func parseLogLevel(level string) service.LogLevel {
	switch strings.ToLower(level) {
	case "info":
		return service.LogLevelInfo
	case "warn", "warning":
		return service.LogLevelWarn
	case "error":
		return service.LogLevelError
	case "debug":
		return service.LogLevelDebug
	case "all":
		return -1 // Special value for "all levels"
	default:
		return -1
	}
}

// filterLogsByLevel filters logs by level.
func filterLogsByLevel(logs []service.LogEntry, level service.LogLevel) []service.LogEntry {
	if level == -1 {
		return logs
	}

	filtered := make([]service.LogEntry, 0)
	for _, entry := range logs {
		if entry.Level == level {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}
