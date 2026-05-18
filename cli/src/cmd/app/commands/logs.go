package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/azure"
	"github.com/jongio/azd-app/cli/src/internal/dashboard"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
	"github.com/jongio/azd-core/cliout"
	"github.com/jongio/azd-core/security"
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
	logLevelWarn           = "warn"
	logLevelDebug          = "debug"
)

// DashboardClient defines the interface for dashboard operations needed by logs.
// This interface enables testing by allowing mock implementations.
type DashboardClient interface {
	Ping(ctx context.Context) error
	GetServices(ctx context.Context) ([]*serviceinfo.ServiceInfo, error)
	StreamLogs(ctx context.Context, serviceName string, logs chan<- service.LogEntry) error
	GetAzureLogs(ctx context.Context, services []string, tail int, since time.Time) ([]service.LogEntry, error)
	GetAzureStatus(ctx context.Context) (*service.AzureStatus, error) //nolint:staticcheck // backward-compatible API
	StreamAzureLogs(ctx context.Context, logs chan<- service.LogEntry) error
}

// Standalone Azure helpers (overridable for tests).
var (
	fetchAzureLogsStandalone  = azure.FetchAzureLogsStandalone
	streamAzureLogsStandalone = azure.StreamAzureLogsStandalone
)

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

// LogSource represents where logs are collected from.
type LogSource string

const (
	// LogSourceLocal indicates logs from locally running services.
	LogSourceLocal LogSource = "local"
	// LogSourceAzure indicates logs from Azure-deployed services.
	LogSourceAzure LogSource = "azure"
	// LogSourceAll indicates logs from both local and Azure sources.
	LogSourceAll LogSource = "all"
)

// CollectedLogs holds the result of log collection, including metadata
// for callers to make display/error decisions.
// collect() returns this; execute() and MCP handlers consume it.
type CollectedLogs struct {
	// Entries holds log entries when no context extraction is needed.
	// Initialized to empty slice (not nil) so JSON marshaling produces [] not null.
	Entries []service.LogEntry

	// EntriesWithContext holds log entries with surrounding context lines.
	// Populated when contextLines > 0 and a level filter is active.
	// Initialized to empty slice (not nil) so JSON marshaling produces [] not null.
	EntriesWithContext []LogEntryWithContext

	// HasContext indicates whether EntriesWithContext is populated (true)
	// or Entries is populated (false).
	HasContext bool

	// DashboardAvailable indicates whether the dashboard was reachable.
	DashboardAvailable bool

	// ServiceCount is the number of services discovered via dashboard.
	ServiceCount int

	// Source is the log source that was used ("local", "azure", "all").
	Source string

	// Warnings holds non-fatal warning messages (e.g., "Azure logs unavailable").
	// CLI displays these via cliout.Warning; MCP can include them in responses.
	Warnings []string
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
	contextLines int    // Number of context lines before/after matching entries (0-10)
	source       string // Log source: "local", "azure", or "all"
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

// newLogsExecutorForMCP creates a logsExecutor for MCP tool use.
// Uses production dependencies but allows specifying a project directory.
func newLogsExecutorForMCP(opts *logsOptions, projectDir string) *logsExecutor {
	return &logsExecutor{
		dashboardClientFactory: func(ctx context.Context, pd string) (DashboardClient, error) {
			return dashboard.NewClient(ctx, pd)
		},
		logManagerFactory: func(pd string) LogManagerInterface {
			return service.GetLogManager(pd)
		},
		getWorkingDir: func() (string, error) {
			if projectDir != "" {
				return projectDir, nil
			}
			return os.Getwd()
		},
		opts: opts,
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
  azd app logs --level error --context 3 --format json

  # View logs from Azure-deployed services
  azd app logs --source azure

  # View both local and Azure logs
  azd app logs --source all

  # Follow Azure logs (polling-based)
  azd app logs --source azure --follow`,
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
	cmd.Flags().StringVar(&opts.exclude, "exclude", "", "Regex patterns to exclude (comma-separated)")
	cmd.Flags().BoolVar(&opts.noBuiltins, "no-builtins", false, "Disable built-in filter patterns")
	cmd.Flags().IntVar(&opts.contextLines, "context", 0, "Number of context lines before/after matching entries (0-10, requires --level)")
	cmd.Flags().StringVar(&opts.source, "source", "local", "Log source: 'local' (default), 'azure', or 'all'")

	return cmd
}

func runLogsWithOptions(opts *logsOptions, args []string) error {
	cliout.CommandHeader("logs", "View logs from running services")

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
	// Default to local logs when source is not explicitly set (test safety)
	if e.opts.source == "" {
		e.opts.source = string(LogSourceLocal)
	}

	collected, err := e.collect(ctx, args)
	if err != nil {
		return err
	}

	// CLI-specific: emit informational messages based on collected status
	if e.opts.source == string(LogSourceLocal) {
		if !collected.DashboardAvailable || collected.ServiceCount == 0 {
			cliout.Info("No services are currently running")
			cliout.Item("Run 'azd app run' to start services")
			return nil
		}
	}
	// Emit any warnings collected during log retrieval
	for _, w := range collected.Warnings {
		cliout.Warning("%s", w)
	}

	if e.opts.source == string(LogSourceAzure) && !e.opts.follow {
		isEmpty := (!collected.HasContext && len(collected.Entries) == 0) ||
			(collected.HasContext && len(collected.EntriesWithContext) == 0)
		if isEmpty {
			cliout.Info("No Azure logs found")
			cliout.Item("Azure logs may take 1-5 minutes to appear in Log Analytics")
			cliout.Item("Use --follow (-f) to wait for new logs")
			return nil
		}
	}

	// Setup output writer
	outputWriter, cleanup, err := e.setupOutputWriter()
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Display logs
	if collected.HasContext {
		if e.opts.format == jsonOutputVal {
			displayLogsWithContextJSON(collected.EntriesWithContext, outputWriter)
		} else {
			displayLogsWithContextText(collected.EntriesWithContext, outputWriter, e.opts.timestamps, e.opts.noColor)
		}
	} else {
		if e.opts.format == jsonOutputVal {
			displayLogsJSON(collected.Entries, outputWriter)
		} else {
			displayLogsText(collected.Entries, outputWriter, e.opts.timestamps, e.opts.noColor)
		}
	}

	// Follow mode - subscribe to live logs
	if e.opts.follow {
		// Reconstruct needed state for follow mode
		cwd, err := e.getWorkingDir()
		if err != nil {
			return fmt.Errorf("failed to get working directory for follow mode: %w", err)
		}
		serviceFilter := e.parseServiceFilter(args)
		logManager := e.logManagerFactory(cwd)
		levelFilter := parseLogLevel(e.opts.level)
		logFilter, _ := e.buildLogFilterInternal(cwd)

		dashCtx, dashCancel := context.WithTimeout(ctx, dashboardOperationTimeout)
		dashboardClient, dashErr := e.dashboardClientFactory(dashCtx, cwd)
		dashCancel()
		if dashErr != nil {
			cliout.Warning("Dashboard unavailable for follow mode: %s", dashErr)
		}

		return e.followLogs(ctx, cwd, logManager, dashboardClient, serviceFilter, levelFilter, logFilter, outputWriter)
	}

	return nil
}

// collect retrieves and filters logs, returning structured data.
// Unlike execute(), it does NOT write to stdout, call cliout, or handle follow mode.
// Callers (CLI execute() or MCP handlers) use the returned CollectedLogs
// to decide what to display or return.
func (e *logsExecutor) collect(ctx context.Context, args []string) (*CollectedLogs, error) {
	// Default to local logs when source is not explicitly set
	if e.opts.source == "" {
		e.opts.source = string(LogSourceLocal)
	}

	result := &CollectedLogs{
		Source:             e.opts.source,
		Entries:            []service.LogEntry{},
		EntriesWithContext: []LogEntryWithContext{},
	}

	// Get current working directory
	cwd, err := e.getWorkingDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
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
	if err != nil && e.opts.source == string(LogSourceLocal) {
		// Local logs require dashboard; no dashboard = no services running
		return result, nil
	}

	var serviceNames []string
	if dashboardClient != nil {
		// Check if dashboard is actually responding
		if pingErr := dashboardClient.Ping(dashCtx); pingErr != nil {
			if e.opts.source == string(LogSourceLocal) {
				return result, nil
			}
			// For Azure-only flows, continue without dashboard
			dashboardClient = nil
		}
	}

	if dashboardClient != nil {
		// Get service list from dashboard
		services, svcErr := dashboardClient.GetServices(dashCtx)
		if svcErr != nil {
			return nil, fmt.Errorf("failed to get services from dashboard: %w", svcErr)
		}

		// Build list of service names
		serviceNames = make([]string, 0, len(services))
		for _, svc := range services {
			serviceNames = append(serviceNames, svc.Name)
		}

		// Check if any services exist (local mode)
		if len(serviceNames) == 0 && e.opts.source == string(LogSourceLocal) {
			result.DashboardAvailable = true
			return result, nil
		}

		// Validate service filter when we have a service list
		if valErr := e.validateServiceFilter(serviceFilter, serviceNames); valErr != nil {
			return nil, valErr
		}
	}
	result.DashboardAvailable = (dashboardClient != nil)
	result.ServiceCount = len(serviceNames)

	// Parse log level filter
	levelFilter := parseLogLevel(e.opts.level)

	// Build log filter from flags and azure.yaml
	logFilter, err := e.buildLogFilterInternal(cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to build log filter: %w", err)
	}

	// Parse since duration
	sinceTime, err := e.parseSinceTime()
	if err != nil {
		return nil, fmt.Errorf("invalid since duration: %w", err)
	}

	// Determine which services to get logs for
	targetServices := serviceFilter
	if len(targetServices) == 0 {
		targetServices = serviceNames
	}

	// Get logs based on source option
	var logs []service.LogEntry
	switch e.opts.source {
	case string(LogSourceAzure):
		logs, err = e.collectAzureLogs(ctx, cwd, dashboardClient, targetServices, sinceTime, result)
		if err != nil {
			return nil, err
		}
	case string(LogSourceAll):
		logs, err = e.collectAllLogsQuiet(ctx, cwd, dashboardClient, targetServices, logManager, sinceTime, result)
		if err != nil {
			return nil, fmt.Errorf("failed to collect logs: %w", err)
		}
	default: // "local"
		if dashboardClient == nil {
			return nil, fmt.Errorf("cannot collect local logs: dashboard not running (run 'azd app run')")
		}
		logs, err = e.collectLogs(ctx, cwd, targetServices, logManager, sinceTime)
		if err != nil {
			return nil, fmt.Errorf("failed to collect logs: %w", err)
		}
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

		result.EntriesWithContext = logsWithContext
		if result.EntriesWithContext == nil {
			result.EntriesWithContext = []LogEntryWithContext{}
		}
		result.HasContext = true
	} else {
		// Regular mode: filter by level
		logs = filterLogsByLevel(logs, levelFilter)

		// Apply final tail limit after all filtering
		if e.opts.tail > 0 && len(logs) > e.opts.tail {
			logs = logs[len(logs)-e.opts.tail:]
		}

		result.Entries = logs
		if result.Entries == nil {
			result.Entries = []service.LogEntry{}
		}
	}

	return result, nil
}

// collectAllLogsQuiet collects logs from both local and Azure sources,
// appending warnings to the CollectedLogs result instead of calling cliout.
func (e *logsExecutor) collectAllLogsQuiet(ctx context.Context, cwd string, dashboardClient DashboardClient, targetServices []string, logManager LogManagerInterface, sinceTime time.Time, result *CollectedLogs) ([]service.LogEntry, error) { //nolint:unparam // return value kept for future use/interface conformance
	var allLogs []service.LogEntry

	// Collect local logs (requires dashboard)
	if dashboardClient != nil {
		localLogs, err := e.collectLogs(ctx, cwd, targetServices, logManager, sinceTime)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to collect local logs: %s", err))
		} else {
			allLogs = append(allLogs, localLogs...)
		}
	} else {
		result.Warnings = append(result.Warnings, "Dashboard not running; local logs unavailable. Showing Azure logs only.")
	}

	// Collect Azure logs (non-fatal if not configured)
	azureLogs, err := e.collectAzureLogs(ctx, cwd, dashboardClient, targetServices, sinceTime, result)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Azure logs unavailable: %s", err))
	} else {
		allLogs = append(allLogs, azureLogs...)
	}

	return allLogs, nil
}

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
		if err := os.MkdirAll(outputDir, 0o750); err != nil {
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

// collectAzureLogs collects logs from Azure-deployed services.
// It first tries the dashboard API (if running), then returns an error with guidance.
func (e *logsExecutor) collectAzureLogs(ctx context.Context, cwd string, dashboardClient DashboardClient, targetServices []string, sinceTime time.Time, result *CollectedLogs) ([]service.LogEntry, error) {
	// Prefer dashboard when available and configured
	if dashboardClient != nil {
		status, err := dashboardClient.GetAzureStatus(ctx)
		if err == nil && status.Enabled {
			if !status.Connected {
				msg := "Azure connection not established."
				if status.ConnectionMessage != "" {
					msg = status.ConnectionMessage
				}
				return nil, fmt.Errorf("%s", msg)
			}

			logs, err := dashboardClient.GetAzureLogs(ctx, targetServices, e.opts.tail, sinceTime)
			if err == nil {
				return logs, nil
			}
			// If dashboard path fails, fall through to standalone for resiliency
			if result != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Dashboard Azure logs failed, falling back to standalone: %v", err))
			}
		}
	}

	return e.collectAzureLogsStandalone(ctx, cwd, targetServices, sinceTime)
}

func (e *logsExecutor) collectAzureLogsStandalone(ctx context.Context, cwd string, targetServices []string, sinceTime time.Time) ([]service.LogEntry, error) {
	// Convert sinceTime to duration window for standalone fetch
	since := 1 * time.Hour
	if !sinceTime.IsZero() {
		since = time.Since(sinceTime)
		if since < time.Minute {
			since = time.Minute
		}
	}

	tail := e.opts.tail
	if tail <= 0 {
		tail = 500
	}

	azLogs, err := fetchAzureLogsStandalone(ctx, azure.StandaloneLogsConfig{
		ProjectDir: cwd,
		Services:   targetServices,
		Since:      since,
		Limit:      tail,
	})
	if err != nil {
		return nil, err
	}

	logs := make([]service.LogEntry, 0, len(azLogs))
	for _, azLog := range azLogs {
		logs = append(logs, service.LogEntry{
			Service:   azLog.Service,
			Message:   azLog.Message,
			Level:     convertAzureLogLevel(azLog.Level),
			Timestamp: azLog.Timestamp,
			Source:    service.LogSourceAzure,
			AzureMetadata: &service.AzureLogMetadata{
				ResourceType:  azLog.ResourceType,
				ContainerName: azLog.ContainerName,
				InstanceID:    azLog.InstanceID,
			},
		})
	}

	return logs, nil
}

// extractLogsWithContext finds log entries matching the level filter and extracts
// surrounding context lines. Handles deduplication of overlapping context ranges.

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
