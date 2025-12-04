package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/dashboard"
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
	logsFile       string
	logsExclude    string
	logsNoBuiltins bool
)

// NewLogsCommand creates the logs command.
func NewLogsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "logs [service-name]",
		Short:        "View logs from running services",
		Long:         `Display output logs from running services for debugging and monitoring`,
		SilenceUsage: true,
		RunE:         runLogs,
	}

	cmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output (tail -f behavior)")
	cmd.Flags().StringVarP(&logsService, "service", "s", "", "Filter by service name(s) (comma-separated)")
	cmd.Flags().IntVarP(&logsTail, "tail", "n", 100, "Number of lines to show from the end")
	cmd.Flags().StringVar(&logsSince, "since", "", "Show logs since duration (e.g., 5m, 1h)")
	cmd.Flags().BoolVar(&logsTimestamps, "timestamps", true, "Show timestamps with each log entry")
	cmd.Flags().BoolVar(&logsNoColor, "no-color", false, "Disable colored output")
	cmd.Flags().StringVar(&logsLevel, "level", "all", "Filter by log level (info, warn, error, debug, all)")
	cmd.Flags().StringVar(&logsFormat, "format", "text", "Output format (text, json)")
	cmd.Flags().StringVar(&logsFile, "file", "", "Write logs to file instead of stdout")
	cmd.Flags().StringVarP(&logsExclude, "exclude", "e", "", "Regex patterns to exclude (comma-separated)")
	cmd.Flags().BoolVar(&logsNoBuiltins, "no-builtins", false, "Disable built-in filter patterns")

	return cmd
}

func runLogs(cmd *cobra.Command, args []string) error {
	output.CommandHeader("logs", "View logs from running services")
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

	// Get log manager for in-memory buffers (may be empty if called from subprocess)
	logManager := service.GetLogManager(cwd)

	// Get running services via dashboard client (works across processes)
	ctx := context.Background()
	dashboardClient, err := dashboard.NewClient(ctx, cwd)
	if err != nil {
		output.Info("No services are currently running")
		output.Item("Run 'azd app run' to start services")
		return nil
	}

	// Check if dashboard is actually responding
	if err := dashboardClient.Ping(ctx); err != nil {
		output.Info("No services are currently running")
		output.Item("Run 'azd app run' to start services")
		return nil
	}

	// Get service list from dashboard
	services, err := dashboardClient.GetServices(ctx)
	if err != nil {
		return fmt.Errorf("failed to get services from dashboard: %w", err)
	}

	// Build list of service names
	var serviceNames []string
	for _, svc := range services {
		serviceNames = append(serviceNames, svc.Name)
	}

	// Check if any services exist
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

	// Build log filter from flags and azure.yaml
	logFilter, err := buildLogFilter(cwd)
	if err != nil {
		return fmt.Errorf("failed to build log filter: %w", err)
	}

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
	outputWriter := os.Stdout
	if logsFile != "" {
		// Validate the output path to prevent path traversal attacks
		if err := security.ValidatePath(logsFile); err != nil {
			return fmt.Errorf("invalid output path: %w", err)
		}
		// #nosec G304 -- Path validated by security.ValidatePath above
		file, err := os.Create(logsFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
		outputWriter = file
	}

	// Determine which services to get logs for
	targetServices := serviceFilter
	if len(targetServices) == 0 {
		targetServices = serviceNames
	}

	// Get logs - try in-memory buffers first, fall back to log files
	var logs []service.LogEntry
	for _, serviceName := range targetServices {
		var serviceLogs []service.LogEntry

		// Try in-memory buffer first
		buffer, exists := logManager.GetBuffer(serviceName)
		if exists {
			if logsSince != "" {
				serviceLogs = buffer.GetSince(sinceTime)
			} else {
				serviceLogs = buffer.GetRecent(logsTail)
			}
		}

		// If no logs in memory, try reading from log files
		if len(serviceLogs) == 0 {
			fileLogs, err := readLogsFromFile(cwd, serviceName, logsTail, sinceTime)
			if err == nil {
				serviceLogs = fileLogs
			}
		}

		logs = append(logs, serviceLogs...)
	}

	// Sort logs by timestamp
	service.SortLogEntries(logs)

	// Filter by level
	logs = filterLogsByLevel(logs, levelFilter)

	// Filter by pattern
	logs = service.FilterLogEntries(logs, logFilter)

	// Display initial logs
	if logsFormat == "json" {
		displayLogsJSON(logs, outputWriter)
	} else {
		displayLogsText(logs, outputWriter, logsTimestamps, logsNoColor)
	}

	// Follow mode - subscribe to live logs
	if logsFollow {
		return followLogs(logManager, serviceFilter, levelFilter, logFilter, outputWriter)
	}

	return nil
}

// readLogsFromFile reads logs from the persisted log file for a service.
// This is used when the in-memory buffer is empty (e.g., when called from a subprocess).
func readLogsFromFile(projectDir, serviceName string, tail int, sinceTime time.Time) ([]service.LogEntry, error) {
	logFile := filepath.Join(projectDir, ".azure", "logs", serviceName+".log")

	file, err := os.Open(logFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []service.LogEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		entry, err := parseLogLine(line, serviceName)
		if err != nil {
			continue // Skip unparseable lines
		}

		// Apply since filter
		if !sinceTime.IsZero() && entry.Timestamp.Before(sinceTime) {
			continue
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Apply tail limit
	if tail > 0 && len(entries) > tail {
		entries = entries[len(entries)-tail:]
	}

	return entries, nil
}

// parseLogLine parses a log line from the file format:
// [2006-01-02 15:04:05.000] [LEVEL] [STREAM] message
func parseLogLine(line, serviceName string) (service.LogEntry, error) {
	entry := service.LogEntry{
		Service: serviceName,
	}

	// Parse timestamp: [2006-01-02 15:04:05.000]
	if len(line) < 25 || line[0] != '[' {
		return entry, fmt.Errorf("invalid log line format")
	}

	endTimestamp := strings.Index(line[1:], "]")
	if endTimestamp == -1 {
		return entry, fmt.Errorf("missing timestamp end bracket")
	}

	timestampStr := line[1 : endTimestamp+1]
	timestamp, err := time.Parse("2006-01-02 15:04:05.000", timestampStr)
	if err != nil {
		return entry, fmt.Errorf("failed to parse timestamp: %w", err)
	}
	entry.Timestamp = timestamp

	// Parse remaining: [LEVEL] [STREAM] message
	remaining := line[endTimestamp+3:] // Skip "] "

	// Parse level: [LEVEL]
	if len(remaining) < 3 || remaining[0] != '[' {
		entry.Message = remaining
		entry.Level = service.LogLevelInfo
		return entry, nil
	}

	endLevel := strings.Index(remaining[1:], "]")
	if endLevel == -1 {
		entry.Message = remaining
		entry.Level = service.LogLevelInfo
		return entry, nil
	}

	levelStr := remaining[1 : endLevel+1]
	entry.Level = parseLogLevelFromString(levelStr)
	remaining = remaining[endLevel+3:] // Skip "] "

	// Parse stream: [STREAM]
	if len(remaining) >= 3 && remaining[0] == '[' {
		endStream := strings.Index(remaining[1:], "]")
		if endStream != -1 {
			streamStr := remaining[1 : endStream+1]
			entry.IsStderr = streamStr == "ERR"
			remaining = remaining[endStream+3:] // Skip "] "
		}
	}

	entry.Message = remaining
	return entry, nil
}

// parseLogLevelFromString parses a log level from a string.
func parseLogLevelFromString(level string) service.LogLevel {
	switch strings.ToUpper(level) {
	case "INFO":
		return service.LogLevelInfo
	case "WARN", "WARNING":
		return service.LogLevelWarn
	case "ERROR":
		return service.LogLevelError
	case "DEBUG":
		return service.LogLevelDebug
	default:
		return service.LogLevelInfo
	}
}

// followLogs subscribes to live log streams and displays them.
func followLogs(logManager *service.LogManager, serviceFilter []string, levelFilter service.LogLevel, logFilter *service.LogFilter, output *os.File) error {
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

			// Filter by pattern
			if logFilter != nil && logFilter.ShouldFilter(entry.Message) {
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

// buildLogFilter creates a log filter from command-line flags and azure.yaml config.
// Priority: command-line flags > azure.yaml project config > built-in patterns.
func buildLogFilter(cwd string) (*service.LogFilter, error) {
	var customPatterns []string

	// Parse command-line exclude patterns
	if logsExclude != "" {
		customPatterns = service.ParseExcludePatterns(logsExclude)
	}

	// Try to load patterns from azure.yaml logs.filters section
	azureYaml, err := service.ParseAzureYaml(cwd)
	filterConfig := getFilterConfig(azureYaml, err)
	if filterConfig != nil {
		customPatterns = append(customPatterns, filterConfig.Exclude...)
	}

	// Determine if we should include built-in patterns
	includeBuiltins := !logsNoBuiltins
	if filterConfig != nil {
		// azure.yaml can override, but command-line takes precedence
		if !logsNoBuiltins {
			includeBuiltins = filterConfig.ShouldIncludeBuiltins()
		}
	}

	// Build the filter
	if includeBuiltins {
		return service.NewLogFilterWithBuiltins(customPatterns)
	}
	return service.NewLogFilter(customPatterns)
}

// getFilterConfig extracts the filter config from azure.yaml's logs section.
func getFilterConfig(azureYaml *service.AzureYaml, err error) *service.LogFilterConfig {
	if err != nil || azureYaml == nil {
		return nil
	}
	return azureYaml.Logs.GetFilters()
}
