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

// Constants for log streaming configuration.
const (
	// logChannelBufferSize is the buffer size for log streaming channels.
	// Set to 100 to balance memory usage with preventing blocking when logs arrive
	// faster than they can be displayed (typical burst rate is ~50 logs/second).
	logChannelBufferSize = 100

	// defaultTailLines is the default number of log lines to show.
	defaultTailLines = 100

	// maxTailLines is the maximum number of lines that can be requested.
	// Capped to prevent excessive memory usage (10K lines ≈ 1-2MB).
	maxTailLines = 10000

	// maxLogLineSize is the maximum size of a single log line (1MB).
	// This handles extremely long log lines from stack traces or JSON dumps.
	maxLogLineSize = 1 * 1024 * 1024

	// scannerInitialBufferSize is the initial buffer for the log file scanner.
	// 64KB handles most log lines without reallocation.
	scannerInitialBufferSize = 64 * 1024

	// dashboardOperationTimeout is the timeout for dashboard operations.
	// Set to 5 seconds to prevent hanging on unresponsive dashboard.
	dashboardOperationTimeout = 5 * time.Second

	// filterCapacityEstimate is the estimated match rate for level filtering.
	// Assumes ~25% of logs match a specific level filter.
	filterCapacityEstimate = 4
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

// LogEntryWithContext represents a log entry with surrounding context lines.
// Used when --context flag is specified to include lines before/after matches.
type LogEntryWithContext struct {
	Service   string      `json:"service"`
	Message   string      `json:"message"`
	Level     string      `json:"level"`
	Timestamp time.Time   `json:"timestamp"`
	IsStderr  bool        `json:"isStderr,omitempty"`
	Context   *LogContext `json:"context,omitempty"`
}

// LogContext contains log lines before and after a matching entry.
type LogContext struct {
	Before []string `json:"before,omitempty"`
	After  []string `json:"after,omitempty"`
}

// logsOptions holds the flag values for the logs command.
// Using a struct avoids global state pollution between command invocations.
type logsOptions struct {
	follow       bool
	service      string
	tail         int
	since        string
	timestamps   bool
	noColor      bool
	level        string
	format       string
	file         string
	exclude      string
	noBuiltins   bool
	contextLines int // Number of context lines before/after matching entries (0-10)
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

	// Configuration options (stored directly to avoid duplication)
	opts *logsOptions
}

// newLogsExecutor creates a logsExecutor with production dependencies.
func newLogsExecutor(opts *logsOptions) *logsExecutor {
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
		opts:          opts,
	}
}

// newLogsExecutorForTest creates a logsExecutor with custom dependencies for testing.
func newLogsExecutorForTest(
	dashboardClientFactory func(ctx context.Context, projectDir string) (DashboardClient, error),
	logManagerFactory func(projectDir string) LogManagerInterface,
	getWorkingDir func() (string, error),
	outputWriter io.Writer,
	opts *logsOptions,
) *logsExecutor {
	return &logsExecutor{
		dashboardClientFactory: dashboardClientFactory,
		logManagerFactory:      logManagerFactory,
		getWorkingDir:          getWorkingDir,
		outputWriter:           outputWriter,
		signalChan:             make(chan os.Signal, 1),
		opts:                   opts,
	}
}

// NewLogsCommand creates the logs command.
func NewLogsCommand() *cobra.Command {
	// Create options for this command invocation
	opts := &logsOptions{}

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

  # View errors with 3 lines of context before and after
  azd app logs --level error --context 3

  # View logs from the last 5 minutes
  azd app logs --since 5m

  # Export logs to a file
  azd app logs --file logs.txt

  # Output as JSON for processing
  azd app logs --format json

  # Output errors as JSON with context
  azd app logs --level error --context 3 --format json`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogsWithOptions(opts, args)
		},
	}

	cmd.Flags().BoolVarP(&opts.follow, "follow", "f", false, "Follow log output (tail -f behavior)")
	cmd.Flags().StringVarP(&opts.service, "service", "s", "", "Filter by service name(s) (comma-separated)")
	cmd.Flags().IntVarP(&opts.tail, "tail", "n", defaultTailLines, "Number of lines to show from the end")
	cmd.Flags().StringVar(&opts.since, "since", "", "Show logs since duration (e.g., 5m, 1h)")
	cmd.Flags().BoolVar(&opts.timestamps, "timestamps", true, "Show timestamps with each log entry")
	cmd.Flags().BoolVar(&opts.noColor, "no-color", false, "Disable colored output")
	cmd.Flags().StringVar(&opts.level, "level", "all", "Filter by log level (info, warn, error, debug, all)")
	cmd.Flags().StringVar(&opts.format, "format", "text", "Output format (text, json)")
	cmd.Flags().StringVar(&opts.file, "file", "", "Write logs to file instead of stdout")
	cmd.Flags().StringVarP(&opts.exclude, "exclude", "e", "", "Regex patterns to exclude (comma-separated)")
	cmd.Flags().BoolVar(&opts.noBuiltins, "no-builtins", false, "Disable built-in filter patterns")
	cmd.Flags().IntVar(&opts.contextLines, "context", 0, "Number of context lines before/after matching entries (0-10, requires --level)")

	return cmd
}

func runLogsWithOptions(opts *logsOptions, args []string) error {
	output.CommandHeader("logs", "View logs from running services")

	// Validate inputs
	if err := validateLogsOptions(opts); err != nil {
		return err
	}

	// Create executor with production dependencies
	executor := newLogsExecutor(opts)

	return executor.execute(context.Background(), args)
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

	// Add timeout for dashboard operations to prevent hanging
	dashCtx, dashCancel := context.WithTimeout(ctx, dashboardOperationTimeout)
	defer dashCancel()

	// Get running services via dashboard client (works across processes)
	dashboardClient, err := e.dashboardClientFactory(dashCtx, cwd)
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
	if err := dashboardClient.Ping(dashCtx); err != nil {
		// Debug: log actual error for troubleshooting
		if os.Getenv("AZD_APP_DEBUG") == "true" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Dashboard ping failed: %v\n", err)
		}
		output.Info("No services are currently running")
		output.Item("Run 'azd app run' to start services")
		return nil
	}

	// Get service list from dashboard
	services, err := dashboardClient.GetServices(dashCtx)
	if err != nil {
		return fmt.Errorf("failed to get services from dashboard: %w", err)
	}

	// Build list of service names
	serviceNames := make([]string, 0, len(services))
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
	levelFilter := parseLogLevel(e.opts.level)

	// Build log filter from flags and azure.yaml
	logFilter, err := e.buildLogFilterInternal(cwd)
	if err != nil {
		return fmt.Errorf("failed to build log filter: %w", err)
	}

	// Parse since duration (returns error instead of silently failing)
	sinceTime, err := e.parseSinceTime()
	if err != nil {
		return fmt.Errorf("invalid since duration: %w", err)
	}

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
	// Pass context to allow cancellation during log collection
	logs, err := e.collectLogs(ctx, cwd, targetServices, logManager, sinceTime)
	if err != nil {
		return fmt.Errorf("failed to collect logs: %w", err)
	}

	// Sort logs by timestamp
	service.SortLogEntries(logs)

	// Filter by pattern first (applies to all logs regardless of context mode)
	logs = service.FilterLogEntries(logs, logFilter)

	// Handle context mode vs regular mode
	if e.opts.contextLines > 0 && levelFilter != LogLevelAll {
		// Context mode: extract matching entries with surrounding context
		logsWithContext := e.extractLogsWithContext(logs, levelFilter, e.opts.contextLines)

		// Apply tail limit to the number of matching entries
		if e.opts.tail > 0 && len(logsWithContext) > e.opts.tail {
			logsWithContext = logsWithContext[len(logsWithContext)-e.opts.tail:]
		}

		// Display logs with context
		if e.opts.format == "json" {
			displayLogsWithContextJSON(logsWithContext, outputWriter)
		} else {
			displayLogsWithContextText(logsWithContext, outputWriter, e.opts.timestamps, e.opts.noColor)
		}
	} else {
		// Regular mode: filter by level and display
		logs = filterLogsByLevel(logs, levelFilter)

		// Apply final tail limit after all filtering (for multi-service view)
		if e.opts.tail > 0 && len(logs) > e.opts.tail {
			logs = logs[len(logs)-e.opts.tail:]
		}

		// Display initial logs
		if e.opts.format == "json" {
			displayLogsJSON(logs, outputWriter)
		} else {
			displayLogsText(logs, outputWriter, e.opts.timestamps, e.opts.noColor)
		}
	}

	// Follow mode - subscribe to live logs
	if e.opts.follow {
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
	} else if e.opts.service != "" {
		// Service name(s) from --service flag
		serviceFilter = strings.Split(e.opts.service, ",")
		for i := range serviceFilter {
			serviceFilter[i] = strings.TrimSpace(serviceFilter[i])
		}
	}
	return serviceFilter
}

// validateServiceFilter validates that all service names in the filter exist.
// Optimized with O(n) lookup using a map instead of O(n*m) nested loops.
func (e *logsExecutor) validateServiceFilter(serviceFilter, serviceNames []string) error {
	if len(serviceFilter) == 0 {
		return nil
	}

	// Build lookup map for O(1) service existence check
	serviceSet := make(map[string]struct{}, len(serviceNames))
	for _, name := range serviceNames {
		serviceSet[name] = struct{}{}
	}

	// Validate each filter service
	for _, filterName := range serviceFilter {
		if _, ok := serviceSet[filterName]; !ok {
			return fmt.Errorf("service '%s' not found (available: %s)",
				filterName, strings.Join(serviceNames, ", "))
		}
	}
	return nil
}

// parseSinceTime parses the since duration and returns the cutoff time.
// Returns error instead of silently failing when duration is invalid.
func (e *logsExecutor) parseSinceTime() (time.Time, error) {
	if e.opts.since == "" {
		return time.Time{}, nil
	}

	duration, err := time.ParseDuration(e.opts.since)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse duration '%s': %w", e.opts.since, err)
	}

	return time.Now().Add(-duration), nil
}

// setupOutputWriter creates the output writer, returning a cleanup function if a file was opened.
func (e *logsExecutor) setupOutputWriter() (io.Writer, func(), error) {
	if e.opts.file == "" {
		return e.outputWriter, nil, nil
	}

	// Validate the output path to prevent path traversal attacks
	if err := security.ValidatePath(e.opts.file); err != nil {
		return nil, nil, fmt.Errorf("invalid output path: %w", err)
	}

	// Ensure parent directory exists
	outputDir := filepath.Dir(e.opts.file)
	if outputDir != "" && outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return nil, nil, fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// #nosec G304 -- Path validated by security.ValidatePath above
	file, err := os.Create(e.opts.file)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create output file: %w", err)
	}

	// Cleanup function that properly handles close errors
	cleanup := func() {
		if err := file.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close log file: %v\n", err)
		}
	}

	return file, cleanup, nil
}

// collectLogs collects logs from all target services.
// Now accepts context to allow cancellation during log collection.
func (e *logsExecutor) collectLogs(ctx context.Context, cwd string, targetServices []string, logManager LogManagerInterface, sinceTime time.Time) ([]service.LogEntry, error) {
	// Pre-allocate with estimated capacity
	estimatedCap := len(targetServices) * e.opts.tail
	logs := make([]service.LogEntry, 0, estimatedCap)

	for _, serviceName := range targetServices {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var serviceLogs []service.LogEntry

		// Try in-memory buffer first
		buffer, exists := logManager.GetBuffer(serviceName)
		if exists {
			if e.opts.since != "" {
				serviceLogs = buffer.GetSince(sinceTime)
			} else {
				serviceLogs = buffer.GetRecent(e.opts.tail)
			}
		}

		// If no logs in memory, try reading from log files
		if len(serviceLogs) == 0 {
			fileLogs, err := readLogsFromFile(cwd, serviceName, e.opts.tail, sinceTime)
			if err == nil {
				serviceLogs = fileLogs
			}
		}

		logs = append(logs, serviceLogs...)
	}
	return logs, nil
}

// extractLogsWithContext finds log entries matching the level filter and extracts
// surrounding context lines. Handles deduplication of overlapping context ranges.
func (e *logsExecutor) extractLogsWithContext(logs []service.LogEntry, levelFilter service.LogLevel, contextLines int) []LogEntryWithContext {
	if len(logs) == 0 || contextLines <= 0 {
		return nil
	}

	// Find indices of matching entries
	var matchIndices []int
	for i, entry := range logs {
		if entry.Level == levelFilter {
			matchIndices = append(matchIndices, i)
		}
	}

	if len(matchIndices) == 0 {
		return nil
	}

	// Build entries with context, handling overlapping ranges
	result := make([]LogEntryWithContext, 0, len(matchIndices))
	usedIndices := make(map[int]bool) // Track indices already shown in context

	for _, matchIdx := range matchIndices {
		entry := logs[matchIdx]

		// Extract before context
		startBefore := matchIdx - contextLines
		if startBefore < 0 {
			startBefore = 0
		}
		before := make([]string, 0, contextLines)
		for i := startBefore; i < matchIdx; i++ {
			// Skip if this line was already shown (to avoid duplicating context)
			if !usedIndices[i] {
				before = append(before, logs[i].Message)
				usedIndices[i] = true
			}
		}

		// Extract after context
		endAfter := matchIdx + contextLines + 1
		if endAfter > len(logs) {
			endAfter = len(logs)
		}
		after := make([]string, 0, contextLines)
		for i := matchIdx + 1; i < endAfter; i++ {
			// Skip if this line was already shown (to avoid duplicating context)
			if !usedIndices[i] {
				after = append(after, logs[i].Message)
				usedIndices[i] = true
			}
		}

		// Mark the match itself as used
		usedIndices[matchIdx] = true

		// Build context only if we have any lines
		var ctx *LogContext
		if len(before) > 0 || len(after) > 0 {
			ctx = &LogContext{
				Before: before,
				After:  after,
			}
		}

		result = append(result, LogEntryWithContext{
			Service:   entry.Service,
			Message:   entry.Message,
			Level:     logLevelToString(entry.Level),
			Timestamp: entry.Timestamp,
			IsStderr:  entry.IsStderr,
			Context:   ctx,
		})
	}

	return result
}

// logLevelToString converts a LogLevel to its string representation.
func logLevelToString(level service.LogLevel) string {
	switch level {
	case service.LogLevelInfo:
		return "info"
	case service.LogLevelWarn:
		return "warn"
	case service.LogLevelError:
		return "error"
	case service.LogLevelDebug:
		return "debug"
	default:
		return "info"
	}
}

// buildLogFilterInternal creates a log filter from executor options and azure.yaml config.
func (e *logsExecutor) buildLogFilterInternal(cwd string) (*service.LogFilter, error) {
	var customPatterns []string

	// Parse command-line exclude patterns
	if e.opts.exclude != "" {
		customPatterns = service.ParseExcludePatterns(e.opts.exclude)
	}

	// Try to load patterns from azure.yaml logs.filters section
	azureYaml, err := service.ParseAzureYaml(cwd)
	filterConfig := getFilterConfig(azureYaml, err)
	if filterConfig != nil {
		customPatterns = append(customPatterns, filterConfig.Exclude...)
	}

	// Determine if we should include built-in patterns
	includeBuiltins := !e.opts.noBuiltins
	if filterConfig != nil {
		// azure.yaml can override, but command-line takes precedence
		if !e.opts.noBuiltins {
			includeBuiltins = filterConfig.ShouldIncludeBuiltins()
		}
	}

	// Build the filter
	if includeBuiltins {
		return service.NewLogFilterWithBuiltins(customPatterns)
	}
	return service.NewLogFilter(customPatterns)
}

// getOrCreateSignalChan gets or creates a signal channel with proper cleanup.
// This avoids duplication and race conditions in signal handling setup.
func (e *logsExecutor) getOrCreateSignalChan() (chan os.Signal, func()) {
	if e.signalChan != nil {
		// Test mode: return existing channel with no-op cleanup
		return e.signalChan, func() {}
	}

	// Production mode: create new channel
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	cleanup := func() {
		signal.Stop(sigChan)
	}

	return sigChan, cleanup
}

// shouldDisplayEntry checks if a log entry should be displayed based on filters.
// Extracted to avoid code duplication between follow modes.
func (e *logsExecutor) shouldDisplayEntry(entry service.LogEntry, levelFilter service.LogLevel, logFilter *service.LogFilter) bool {
	// Filter by level
	if levelFilter != LogLevelAll && entry.Level != levelFilter {
		return false
	}

	// Filter by pattern
	if logFilter != nil && logFilter.ShouldFilter(entry.Message) {
		return false
	}

	return true
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
	sigChan, cleanupSignal := e.getOrCreateSignalChan()
	defer cleanupSignal()

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

			// Use extracted filter method
			if !e.shouldDisplayEntry(entry, levelFilter, logFilter) {
				continue
			}

			// Display log entry
			if e.opts.format == "json" {
				displayLogsJSON([]service.LogEntry{entry}, outputWriter)
			} else {
				displayLogsText([]service.LogEntry{entry}, outputWriter, e.opts.timestamps, e.opts.noColor)
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
	sigChan, cleanupSignal := e.getOrCreateSignalChan()
	defer cleanupSignal()

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

	// Cleanup helper function with sync.Once to prevent double-close panics
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			close(stopChan)
			wg.Wait() // Ensure all goroutines stopped before unsubscribing
			for serviceName, ch := range subscriptions {
				buffer, exists := logManager.GetBuffer(serviceName)
				if exists {
					buffer.Unsubscribe(ch)
				}
			}
		})
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

			// Use extracted filter method
			if !e.shouldDisplayEntry(entry, levelFilter, logFilter) {
				continue
			}

			// Display log entry
			if e.opts.format == "json" {
				displayLogsJSON([]service.LogEntry{entry}, outputWriter)
			} else {
				displayLogsText([]service.LogEntry{entry}, outputWriter, e.opts.timestamps, e.opts.noColor)
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
	scanner.Buffer(make([]byte, scannerInitialBufferSize), maxLogLineSize)

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

// displayLogsWithContextJSON displays logs with context in JSON format.
// Each entry includes optional before/after context lines.
func displayLogsWithContextJSON(logs []LogEntryWithContext, w io.Writer) {
	encoder := json.NewEncoder(w)
	for _, entry := range logs {
		if err := encoder.Encode(entry); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to encode log entry: %v\n", err)
		}
	}
}

// displayLogsWithContextText displays logs with context in text format.
// Context lines are shown with indentation and separators between entries.
func displayLogsWithContextText(logs []LogEntryWithContext, w io.Writer, showTimestamps, noColor bool) {
	for i, entry := range logs {
		// Add separator between entries (not before first)
		if i > 0 {
			fmt.Fprintln(w, "---")
		}

		// Show before context (if any)
		if entry.Context != nil && len(entry.Context.Before) > 0 {
			for _, line := range entry.Context.Before {
				if noColor {
					fmt.Fprintf(w, "  %s\n", line)
				} else {
					fmt.Fprintf(w, "  %s%s%s\n", colorGray, line, colorReset)
				}
			}
		}

		// Show the matching entry
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

		// Message with color based on level
		if noColor {
			line.WriteString(entry.Message)
		} else {
			switch entry.Level {
			case "error":
				line.WriteString(colorRed + entry.Message + colorReset)
			case "warn":
				line.WriteString(colorYellow + entry.Message + colorReset)
			case "debug":
				line.WriteString(colorGray + entry.Message + colorReset)
			default:
				line.WriteString(entry.Message)
			}
		}

		fmt.Fprintln(w, line.String())

		// Show after context (if any)
		if entry.Context != nil && len(entry.Context.After) > 0 {
			for _, contextLine := range entry.Context.After {
				if noColor {
					fmt.Fprintf(w, "  %s\n", contextLine)
				} else {
					fmt.Fprintf(w, "  %s%s%s\n", colorGray, contextLine, colorReset)
				}
			}
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

// filterLogsByLevel filters logs by level with pre-allocated capacity.
func filterLogsByLevel(logs []service.LogEntry, level service.LogLevel) []service.LogEntry {
	if level == LogLevelAll {
		return logs
	}

	// Pre-allocate with estimated capacity based on typical match rate
	estimatedCap := len(logs) / filterCapacityEstimate
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

// buildLogFilter creates a log filter from options and azure.yaml config.
// This is a test helper function that wraps buildLogFilterInternal.
// Deprecated: Use executor.buildLogFilterInternal directly in new code.
func buildLogFilter(cwd string, exclude string, noBuiltins bool) (*service.LogFilter, error) {
	var customPatterns []string

	// Parse command-line exclude patterns
	if exclude != "" {
		customPatterns = service.ParseExcludePatterns(exclude)
	}

	// Try to load patterns from azure.yaml logs.filters section
	azureYaml, err := service.ParseAzureYaml(cwd)
	filterConfig := getFilterConfig(azureYaml, err)
	if filterConfig != nil {
		customPatterns = append(customPatterns, filterConfig.Exclude...)
	}

	// Determine if we should include built-in patterns
	includeBuiltins := !noBuiltins
	if filterConfig != nil {
		// azure.yaml can override, but command-line takes precedence
		if !noBuiltins {
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

// validateLogsOptions validates command-line flag values.
func validateLogsOptions(opts *logsOptions) error {
	// Validate tail is positive
	if opts.tail < 0 {
		return fmt.Errorf("--tail must be a positive number, got %d", opts.tail)
	}
	if opts.tail > maxTailLines {
		// Log warning before capping
		fmt.Fprintf(os.Stderr, "Warning: --tail value %d exceeds maximum, capping at %d\n", opts.tail, maxTailLines)
		opts.tail = maxTailLines
	}

	// Validate format
	switch opts.format {
	case "text", "json":
		// Valid formats
	default:
		return fmt.Errorf("--format must be 'text' or 'json', got '%s'", opts.format)
	}

	// Validate level
	switch strings.ToLower(opts.level) {
	case "info", "warn", "warning", "error", "debug", "all":
		// Valid levels
	default:
		return fmt.Errorf("--level must be one of: info, warn, error, debug, all; got '%s'", opts.level)
	}

	// Validate context requires level to be set (not "all")
	if opts.contextLines > 0 {
		if strings.ToLower(opts.level) == "all" {
			return fmt.Errorf("--context requires --level to be set (info, warn, error, or debug)")
		}
	}

	// Clamp context to valid range (0-MaxContextLines)
	if opts.contextLines < 0 {
		opts.contextLines = 0
	}
	if opts.contextLines > service.MaxContextLines {
		fmt.Fprintf(os.Stderr, "Warning: --context value %d exceeds maximum, capping at %d\n", opts.contextLines, service.MaxContextLines)
		opts.contextLines = service.MaxContextLines
	}

	// Validate since duration if provided
	if opts.since != "" {
		if _, err := time.ParseDuration(opts.since); err != nil {
			return fmt.Errorf("--since must be a valid duration (e.g., 5m, 1h), got '%s': %w", opts.since, err)
		}
	}

	return nil
}
