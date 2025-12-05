package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/dashboard"
	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/security"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
	"github.com/spf13/cobra"
)

// DashboardClient defines the interface for dashboard operations needed by logs.
// This interface enables testing by allowing mock implementations.
type DashboardClient interface {
	Ping(ctx context.Context) error
	GetServices(ctx context.Context) ([]*serviceinfo.ServiceInfo, error)
	StreamLogs(ctx context.Context, serviceName string, logs chan<- service.LogEntry) error
}

// LogManagerInterface defines the interface for log manager operations.
// This interface enables testing by allowing mock implementations.
type LogManagerInterface interface {
	GetBuffer(serviceName string) (*service.LogBuffer, bool)
	GetAllBuffers() map[string]*service.LogBuffer
}

// logsExecutor encapsulates the logs command execution with injectable dependencies.
// This struct enables unit testing of the logs command logic.
type logsExecutor struct {
	// Dependencies (injectable for testing)
	dashboardClientFactory func(ctx context.Context, projectDir string) (DashboardClient, error)
	logManagerFactory      func(projectDir string) LogManagerInterface
	getWorkingDir          func() (string, error)
	outputWriter           io.Writer
	signalChan             chan os.Signal

	// Options from flags
	follow     bool
	service    string
	tail       int
	since      string
	timestamps bool
	noColor    bool
	level      string
	format     string
	file       string
	exclude    string
	noBuiltins bool
}

// newLogsExecutor creates a logsExecutor with production dependencies.
func newLogsExecutor() *logsExecutor {
	return &logsExecutor{
		dashboardClientFactory: func(ctx context.Context, projectDir string) (DashboardClient, error) {
			return dashboard.NewClient(ctx, projectDir)
		},
		logManagerFactory: func(projectDir string) LogManagerInterface {
			return service.GetLogManager(projectDir)
		},
		getWorkingDir: os.Getwd,
		outputWriter:  os.Stdout,
		signalChan:    nil, // Will be created on demand
	}
}

// newLogsExecutorForTest creates a logsExecutor with custom dependencies for testing.
func newLogsExecutorForTest(
	dashboardClientFactory func(ctx context.Context, projectDir string) (DashboardClient, error),
	logManagerFactory func(projectDir string) LogManagerInterface,
	getWorkingDir func() (string, error),
	outputWriter io.Writer,
) *logsExecutor {
	return &logsExecutor{
		dashboardClientFactory: dashboardClientFactory,
		logManagerFactory:      logManagerFactory,
		getWorkingDir:          getWorkingDir,
		outputWriter:           outputWriter,
		signalChan:             make(chan os.Signal, 1),
	}
}

// logsOptions holds the flag values for the logs command.
// Using a struct avoids global state pollution between command invocations.
type logsOptions struct {
	follow     bool
	service    string
	tail       int
	since      string
	timestamps bool
	noColor    bool
	level      string
	format     string
	file       string
	exclude    string
	noBuiltins bool
}

// Constants for log streaming configuration.
const (
	// logChannelBufferSize is the buffer size for log streaming channels.
	// A larger buffer prevents blocking when logs arrive faster than they can be displayed.
	logChannelBufferSize = 100

	// defaultTailLines is the default number of log lines to show.
	defaultTailLines = 100

	// maxTailLines is the maximum number of lines that can be requested.
	maxTailLines = 10000
)

// Global variables for backward compatibility with tests.
// These are populated by the command flags.
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
		Use:   "logs [service-name]",
		Short: "View logs from running services",
		Long: `Display output logs from running services for debugging and monitoring.

Examples:
  # View last 100 lines from all services
  azd app logs

  # Follow logs in real-time (like tail -f)
  azd app logs -f

  # View logs from a specific service
  azd app logs api

  # Filter by log level
  azd app logs --level error

  # View logs from the last 5 minutes
  azd app logs --since 5m

  # Export logs to a file
  azd app logs --file logs.txt

  # Output as JSON for processing
  azd app logs --format json`,
		SilenceUsage: true,
		RunE:         runLogs,
	}

	cmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output (tail -f behavior)")
	cmd.Flags().StringVarP(&logsService, "service", "s", "", "Filter by service name(s) (comma-separated)")
	cmd.Flags().IntVarP(&logsTail, "tail", "n", defaultTailLines, "Number of lines to show from the end")
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

	// Validate inputs
	if err := validateLogsInputs(); err != nil {
		return err
	}

	// Create executor with production dependencies
	executor := newLogsExecutor()
	executor.copyFromGlobalFlags()

	return executor.execute(context.Background(), args)
}

// copyFromGlobalFlags copies global flag values to the executor.
func (e *logsExecutor) copyFromGlobalFlags() {
	e.follow = logsFollow
	e.service = logsService
	e.tail = logsTail
	e.since = logsSince
	e.timestamps = logsTimestamps
	e.noColor = logsNoColor
	e.level = logsLevel
	e.format = logsFormat
	e.file = logsFile
	e.exclude = logsExclude
	e.noBuiltins = logsNoBuiltins
}

// execute runs the logs command with the configured dependencies and options.
func (e *logsExecutor) execute(ctx context.Context, args []string) error {
	// Get current working directory
	cwd, err := e.getWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Determine service filter
	serviceFilter := e.parseServiceFilter(args)

	// Get log manager for in-memory buffers (may be empty if called from subprocess)
	logManager := e.logManagerFactory(cwd)

	// Get running services via dashboard client (works across processes)
	dashboardClient, err := e.dashboardClientFactory(ctx, cwd)
	if err != nil {
		// Debug: log actual error for troubleshooting
		if os.Getenv("AZD_APP_DEBUG") == "true" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Dashboard client creation failed: %v\n", err)
		}
		output.Info("No services are currently running")
		output.Item("Run 'azd app run' to start services")
		return nil
	}

	// Check if dashboard is actually responding
	if err := dashboardClient.Ping(ctx); err != nil {
		// Debug: log actual error for troubleshooting
		if os.Getenv("AZD_APP_DEBUG") == "true" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Dashboard ping failed: %v\n", err)
		}
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
	if err := e.validateServiceFilter(serviceFilter, serviceNames); err != nil {
		return err
	}

	// Parse log level filter
	levelFilter := parseLogLevel(e.level)

	// Build log filter from flags and azure.yaml
	logFilter, err := e.buildLogFilterInternal(cwd)
	if err != nil {
		return fmt.Errorf("failed to build log filter: %w", err)
	}

	// Parse since duration
	sinceTime := e.parseSinceTime()

	// Setup output writer
	outputWriter, cleanup, err := e.setupOutputWriter()
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Determine which services to get logs for
	targetServices := serviceFilter
	if len(targetServices) == 0 {
		targetServices = serviceNames
	}

	// Get logs - try in-memory buffers first, fall back to log files
	logs := e.collectLogs(cwd, targetServices, logManager, sinceTime)

	// Sort logs by timestamp
	service.SortLogEntries(logs)

	// Filter by level
	logs = filterLogsByLevel(logs, levelFilter)

	// Filter by pattern
	logs = service.FilterLogEntries(logs, logFilter)

	// Apply final tail limit after all filtering (for multi-service view)
	if e.tail > 0 && len(logs) > e.tail {
		logs = logs[len(logs)-e.tail:]
	}

	// Display initial logs
	if e.format == "json" {
		displayLogsJSON(logs, outputWriter)
	} else {
		displayLogsText(logs, outputWriter, e.timestamps, e.noColor)
	}

	// Follow mode - subscribe to live logs
	if e.follow {
		return e.followLogs(ctx, cwd, logManager, dashboardClient, serviceFilter, levelFilter, logFilter, outputWriter)
	}

	return nil
}

// parseServiceFilter parses service names from args and flags.
func (e *logsExecutor) parseServiceFilter(args []string) []string {
	var serviceFilter []string
	if len(args) > 0 {
		// Service name from positional argument
		serviceFilter = []string{args[0]}
	} else if e.service != "" {
		// Service name(s) from --service flag
		serviceFilter = strings.Split(e.service, ",")
		for i := range serviceFilter {
			serviceFilter[i] = strings.TrimSpace(serviceFilter[i])
		}
	}
	return serviceFilter
}

// validateServiceFilter validates that all service names in the filter exist.
func (e *logsExecutor) validateServiceFilter(serviceFilter, serviceNames []string) error {
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
	return nil
}

// parseSinceTime parses the since duration and returns the cutoff time.
func (e *logsExecutor) parseSinceTime() time.Time {
	var sinceTime time.Time
	if e.since != "" {
		duration, err := time.ParseDuration(e.since)
		if err == nil {
			sinceTime = time.Now().Add(-duration)
		}
	}
	return sinceTime
}

// setupOutputWriter creates the output writer, returning a cleanup function if a file was opened.
func (e *logsExecutor) setupOutputWriter() (io.Writer, func(), error) {
	if e.file == "" {
		return e.outputWriter, nil, nil
	}

	// Validate the output path to prevent path traversal attacks
	if err := security.ValidatePath(e.file); err != nil {
		return nil, nil, fmt.Errorf("invalid output path: %w", err)
	}

	// Ensure parent directory exists
	outputDir := filepath.Dir(e.file)
	if outputDir != "" && outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return nil, nil, fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// #nosec G304 -- Path validated by security.ValidatePath above
	file, err := os.Create(e.file)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create output file: %w", err)
	}

	return file, func() { file.Close() }, nil
}

// collectLogs collects logs from all target services.
func (e *logsExecutor) collectLogs(cwd string, targetServices []string, logManager LogManagerInterface, sinceTime time.Time) []service.LogEntry {
	var logs []service.LogEntry
	for _, serviceName := range targetServices {
		var serviceLogs []service.LogEntry

		// Try in-memory buffer first
		buffer, exists := logManager.GetBuffer(serviceName)
		if exists {
			if e.since != "" {
				serviceLogs = buffer.GetSince(sinceTime)
			} else {
				serviceLogs = buffer.GetRecent(e.tail)
			}
		}

		// If no logs in memory, try reading from log files
		if len(serviceLogs) == 0 {
			fileLogs, err := readLogsFromFile(cwd, serviceName, e.tail, sinceTime)
			if err == nil {
				serviceLogs = fileLogs
			}
		}

		logs = append(logs, serviceLogs...)
	}
	return logs
}

// buildLogFilterInternal creates a log filter from executor options and azure.yaml config.
func (e *logsExecutor) buildLogFilterInternal(cwd string) (*service.LogFilter, error) {
	var customPatterns []string

	// Parse command-line exclude patterns
	if e.exclude != "" {
		customPatterns = service.ParseExcludePatterns(e.exclude)
	}

	// Try to load patterns from azure.yaml logs.filters section
	azureYaml, err := service.ParseAzureYaml(cwd)
	filterConfig := getFilterConfig(azureYaml, err)
	if filterConfig != nil {
		customPatterns = append(customPatterns, filterConfig.Exclude...)
	}

	// Determine if we should include built-in patterns
	includeBuiltins := !e.noBuiltins
	if filterConfig != nil {
		// azure.yaml can override, but command-line takes precedence
		if !e.noBuiltins {
			includeBuiltins = filterConfig.ShouldIncludeBuiltins()
		}
	}

	// Build the filter
	if includeBuiltins {
		return service.NewLogFilterWithBuiltins(customPatterns)
	}
	return service.NewLogFilter(customPatterns)
}

// followLogs subscribes to live log streams and displays them.
func (e *logsExecutor) followLogs(ctx context.Context, projectDir string, logManager LogManagerInterface, dashboardClient DashboardClient, serviceFilter []string, levelFilter service.LogLevel, logFilter *service.LogFilter, outputWriter io.Writer) error {
	// Try in-memory subscriptions first
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

	// If no in-memory buffers, try dashboard WebSocket streaming
	if len(subscriptions) == 0 {
		return e.followLogsViaDashboard(ctx, dashboardClient, serviceFilter, levelFilter, logFilter, outputWriter)
	}

	// Use in-memory streaming
	return e.followLogsInMemory(subscriptions, logManager, levelFilter, logFilter, outputWriter)
}

// followLogsViaDashboard connects to the dashboard's WebSocket to stream logs.
func (e *logsExecutor) followLogsViaDashboard(ctx context.Context, dashboardClient DashboardClient, serviceFilter []string, levelFilter service.LogLevel, logFilter *service.LogFilter, outputWriter io.Writer) error {
	// Check if dashboard is responding
	if err := dashboardClient.Ping(ctx); err != nil {
		return fmt.Errorf("cannot follow logs: dashboard not responding (run 'azd app run' first)")
	}

	output.Info("Streaming logs from dashboard...")

	// Create context for streaming that can be cancelled
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Setup signal handling for graceful exit
	sigChan := e.signalChan
	if sigChan == nil {
		sigChan = make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(sigChan)
	}

	// Create channel for log entries
	logs := make(chan service.LogEntry, logChannelBufferSize)

	// Determine service filter (empty string for all)
	serviceName := ""
	if len(serviceFilter) == 1 {
		serviceName = serviceFilter[0]
	}

	// Start streaming in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- dashboardClient.StreamLogs(streamCtx, serviceName, logs)
	}()

	// Display logs as they arrive
	for {
		select {
		case entry := <-logs:
			// Filter by service if multiple specified
			if len(serviceFilter) > 1 {
				found := false
				for _, svc := range serviceFilter {
					if entry.Service == svc {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			// Filter by level
			if levelFilter != LogLevelAll && entry.Level != levelFilter {
				continue
			}

			// Filter by pattern
			if logFilter != nil && logFilter.ShouldFilter(entry.Message) {
				continue
			}

			// Display log entry
			if e.format == "json" {
				displayLogsJSON([]service.LogEntry{entry}, outputWriter)
			} else {
				displayLogsText([]service.LogEntry{entry}, outputWriter, e.timestamps, e.noColor)
			}

		case err := <-errChan:
			if err != nil && err != context.Canceled {
				return fmt.Errorf("log stream error: %w", err)
			}
			return nil

		case <-sigChan:
			cancel()
			return nil
		}
	}
}

// followLogsInMemory uses in-memory log buffer subscriptions.
func (e *logsExecutor) followLogsInMemory(subscriptions map[string]chan service.LogEntry, logManager LogManagerInterface, levelFilter service.LogLevel, logFilter *service.LogFilter, outputWriter io.Writer) error {
	// Setup signal handling for graceful exit
	sigChan := e.signalChan
	if sigChan == nil {
		sigChan = make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(sigChan)
	}

	// Create stop channel for goroutine cleanup
	stopChan := make(chan struct{})

	// Merge all subscription channels with WaitGroup to track completion
	mergedChan := make(chan service.LogEntry, logChannelBufferSize)
	var wg sync.WaitGroup

	for _, ch := range subscriptions {
		wg.Add(1)
		go func(ch chan service.LogEntry) {
			defer wg.Done()
			for {
				select {
				case entry, ok := <-ch:
					if !ok {
						return
					}
					select {
					case mergedChan <- entry:
					case <-stopChan:
						return
					}
				case <-stopChan:
					return
				}
			}
		}(ch)
	}

	// Close mergedChan when all goroutines complete
	go func() {
		wg.Wait()
		close(mergedChan)
	}()

	// Cleanup helper function
	cleanup := func() {
		close(stopChan)
		for serviceName, ch := range subscriptions {
			buffer, exists := logManager.GetBuffer(serviceName)
			if exists {
				buffer.Unsubscribe(ch)
			}
		}
	}

	// Display logs as they arrive
	for {
		select {
		case entry, ok := <-mergedChan:
			if !ok {
				// All sources closed
				cleanup()
				return nil
			}

			// Filter by level
			if levelFilter != LogLevelAll && entry.Level != levelFilter {
				continue
			}

			// Filter by pattern
			if logFilter != nil && logFilter.ShouldFilter(entry.Message) {
				continue
			}

			// Display log entry
			if e.format == "json" {
				displayLogsJSON([]service.LogEntry{entry}, outputWriter)
			} else {
				displayLogsText([]service.LogEntry{entry}, outputWriter, e.timestamps, e.noColor)
			}

		case <-sigChan:
			cleanup()
			return nil
		}
	}
}

// readLogsFromFile reads logs from the persisted log file for a service.
// This is used when the in-memory buffer is empty (e.g., when called from a subprocess).
// It also reads from rotated backup files (.log.1, .log.2) if needed.
func readLogsFromFile(projectDir, serviceName string, tail int, sinceTime time.Time) ([]service.LogEntry, error) {
	logsDir := filepath.Join(projectDir, ".azure", "logs")
	baseLogFile := filepath.Join(logsDir, serviceName+".log")

	var allEntries []service.LogEntry

	// Read from rotated files first (oldest to newest: .log.2, .log.1, .log)
	logFiles := []string{
		baseLogFile + ".2",
		baseLogFile + ".1",
		baseLogFile,
	}

	for _, logFile := range logFiles {
		entries, err := readSingleLogFile(logFile, serviceName, sinceTime)
		if err != nil {
			continue // File may not exist (rotated files are optional)
		}
		allEntries = append(allEntries, entries...)
	}

	if len(allEntries) == 0 {
		return nil, fmt.Errorf("no log files found for service %s", serviceName)
	}

	// Apply tail limit
	if tail > 0 && len(allEntries) > tail {
		allEntries = allEntries[len(allEntries)-tail:]
	}

	return allEntries, nil
}

// maxLogLineSize is the maximum size of a single log line (1MB).
// This handles extremely long log lines from stack traces or JSON dumps.
const maxLogLineSize = 1 * 1024 * 1024

// readSingleLogFile reads log entries from a single log file.
func readSingleLogFile(logFile, serviceName string, sinceTime time.Time) ([]service.LogEntry, error) {
	file, err := os.Open(logFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []service.LogEntry
	scanner := bufio.NewScanner(file)
	// Increase buffer size to handle long log lines (stack traces, JSON dumps)
	scanner.Buffer(make([]byte, 64*1024), maxLogLineSize)

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

// ANSI color constants for log output formatting.
// colorCyan is defined here as it's not in info.go.
// Other colors (colorGray, colorRed, colorYellow, colorReset) are in info.go.
const colorCyan = "\033[36m"

// displayLogsText displays logs in text format.
// Uses io.Writer interface for better testability and flexibility.
func displayLogsText(logs []service.LogEntry, w io.Writer, showTimestamps, noColor bool) {
	for _, entry := range logs {
		var line strings.Builder

		// Timestamp
		if showTimestamps {
			timestamp := entry.Timestamp.Format("15:04:05.000")
			if noColor {
				line.WriteString(fmt.Sprintf("[%s] ", timestamp))
			} else {
				line.WriteString(colorGray + "[" + timestamp + "]" + colorReset + " ")
			}
		}

		// Service name
		if noColor {
			line.WriteString(fmt.Sprintf("[%s] ", entry.Service))
		} else {
			line.WriteString(colorCyan + "[" + entry.Service + "]" + colorReset + " ")
		}

		// Message with color based on stderr/level
		if noColor {
			line.WriteString(entry.Message)
		} else {
			if entry.IsStderr || entry.Level == service.LogLevelError {
				line.WriteString(colorRed + entry.Message + colorReset)
			} else if entry.Level == service.LogLevelWarn {
				line.WriteString(colorYellow + entry.Message + colorReset)
			} else if entry.Level == service.LogLevelDebug {
				line.WriteString(colorGray + entry.Message + colorReset)
			} else {
				line.WriteString(entry.Message)
			}
		}

		fmt.Fprintln(w, line.String())
	}
}

// displayLogsJSON displays logs in JSON format.
// Uses io.Writer interface for better testability and flexibility.
func displayLogsJSON(logs []service.LogEntry, w io.Writer) {
	encoder := json.NewEncoder(w)
	for _, entry := range logs {
		if err := encoder.Encode(entry); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to encode log entry: %v\n", err)
		}
	}
}

// LogLevelAll is a sentinel value indicating no level filtering should be applied.
const LogLevelAll service.LogLevel = -1

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
		return LogLevelAll
	default:
		return LogLevelAll
	}
}

// filterLogsByLevel filters logs by level.
func filterLogsByLevel(logs []service.LogEntry, level service.LogLevel) []service.LogEntry {
	if level == LogLevelAll {
		return logs
	}

	// Pre-allocate with estimated capacity (assume ~25% match rate)
	estimatedCap := len(logs) / 4
	if estimatedCap < 10 {
		estimatedCap = 10
	}
	filtered := make([]service.LogEntry, 0, estimatedCap)
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

// validateLogsInputs validates command-line flag values.
func validateLogsInputs() error {
	// Validate tail is positive
	if logsTail < 0 {
		return fmt.Errorf("--tail must be a positive number, got %d", logsTail)
	}
	if logsTail > maxTailLines {
		logsTail = maxTailLines // Cap at maximum
	}

	// Validate format
	switch logsFormat {
	case "text", "json":
		// Valid formats
	default:
		return fmt.Errorf("--format must be 'text' or 'json', got '%s'", logsFormat)
	}

	// Validate level
	switch strings.ToLower(logsLevel) {
	case "info", "warn", "warning", "error", "debug", "all":
		// Valid levels
	default:
		return fmt.Errorf("--level must be one of: info, warn, error, debug, all; got '%s'", logsLevel)
	}

	// Validate since duration if provided
	if logsSince != "" {
		if _, err := time.ParseDuration(logsSince); err != nil {
			return fmt.Errorf("--since must be a valid duration (e.g., 5m, 1h), got '%s': %w", logsSince, err)
		}
	}

	return nil
}
