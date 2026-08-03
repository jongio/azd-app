package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/azure"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

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
		return logLevelWarn
	case service.LogLevelError:
		return statusError
	case service.LogLevelDebug:
		return logLevelDebug
	default:
		return "info"
	}
}

// convertAzureLogLevel maps azure.LogLevel to the shared service.LogLevel enum.
func convertAzureLogLevel(level azure.LogLevel) service.LogLevel {
	switch level {
	case azure.LogLevelWarn:
		return service.LogLevelWarn
	case azure.LogLevelError:
		return service.LogLevelError
	case azure.LogLevelDebug:
		return service.LogLevelDebug
	case azure.LogLevelInfo:
		fallthrough
	default:
		return service.LogLevelInfo
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
	// CLI --no-builtins flag controls this; azure.yaml always includes builtins
	includeBuiltins := !e.opts.noBuiltins

	// Build the filter
	var (
		filter *service.LogFilter
		filErr error
	)
	if includeBuiltins {
		filter, filErr = service.NewLogFilterWithBuiltins(customPatterns)
	} else {
		filter, filErr = service.NewLogFilter(customPatterns)
	}
	if filErr != nil {
		return nil, filErr
	}

	// Apply the positive include pattern from --grep, if any. Exclude patterns
	// still win, so a line matching both is dropped.
	if e.opts.grep != "" {
		if err := filter.AddIncludePattern(e.opts.grep); err != nil {
			return nil, fmt.Errorf("invalid --grep pattern: %w", err)
		}
	}

	return filter, nil
}

// getOrCreateSignalChan gets or creates a signal channel with proper cleanup.
// This avoids duplication and race conditions in signal handling setup.

func (e *logsExecutor) shouldDisplayEntry(entry service.LogEntry, levelFilter, minLevel service.LogLevel, minLevelSet bool, logFilter *service.LogFilter) bool {
	// Filter by level
	if levelFilter != LogLevelAll && entry.Level != levelFilter {
		return false
	}

	// Filter by minimum severity threshold when --min-level is set
	if minLevelSet && !meetsMinLevel(entry.Level, minLevel) {
		return false
	}

	// Filter by pattern
	if logFilter != nil && logFilter.ShouldFilter(entry.Message) {
		return false
	}

	return true
}

// followLogs subscribes to live log streams and displays them.

func parseLogLevel(level string) service.LogLevel {
	switch strings.ToLower(level) {
	case "info":
		return service.LogLevelInfo
	case logLevelWarn, "warning":
		return service.LogLevelWarn
	case "error":
		return service.LogLevelError
	case logLevelDebug:
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

// logLevelSeverity maps a log level to an ordered severity rank so thresholds can
// compare levels. The raw LogLevel iota order is not severity order, so the mapping
// is explicit: debug < info < warn < error. Unknown levels are treated as info.
func logLevelSeverity(level service.LogLevel) int {
	switch level {
	case service.LogLevelDebug:
		return 0
	case service.LogLevelInfo:
		return 1
	case service.LogLevelWarn:
		return 2
	case service.LogLevelError:
		return 3
	default:
		return 1
	}
}

// parseMinLevel parses a --min-level threshold. It returns the level and true when
// the value names a concrete severity (debug, info, warn, error). It returns false
// for empty or unrecognized values, which callers treat as "no threshold set".
func parseMinLevel(level string) (service.LogLevel, bool) {
	switch strings.ToLower(level) {
	case logLevelDebug:
		return service.LogLevelDebug, true
	case "info":
		return service.LogLevelInfo, true
	case logLevelWarn, "warning":
		return service.LogLevelWarn, true
	case "error":
		return service.LogLevelError, true
	default:
		return LogLevelAll, false
	}
}

// meetsMinLevel reports whether an entry's level is at or above the given threshold.
func meetsMinLevel(entryLevel, minLevel service.LogLevel) bool {
	return logLevelSeverity(entryLevel) >= logLevelSeverity(minLevel)
}

// filterLogsByMinLevel keeps only entries at or above the given severity threshold.
func filterLogsByMinLevel(logs []service.LogEntry, minLevel service.LogLevel) []service.LogEntry {
	estimatedCap := len(logs) / filterCapacityEstimate
	if estimatedCap < 10 {
		estimatedCap = 10
	}
	filtered := make([]service.LogEntry, 0, estimatedCap)
	for _, entry := range logs {
		if meetsMinLevel(entry.Level, minLevel) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// buildLogFilter creates a log filter from options and azure.yaml config.
// This is a test helper function that wraps buildLogFilterInternal.
//
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
	// CLI --no-builtins flag controls this; azure.yaml always includes builtins
	includeBuiltins := !noBuiltins

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
	if opts.summary && opts.follow {
		return fmt.Errorf("--summary cannot be combined with --follow")
	}

	// Validate tail is positive
	if opts.tail < 0 {
		return fmt.Errorf("--tail must be a positive number, got %d", opts.tail)
	}
	if opts.tail > maxTailLines {
		// Log warning before capping
		fmt.Fprintf(os.Stderr, "Warning: --tail value %d exceeds maximum, capping at %d\n", opts.tail, maxTailLines)
		opts.tail = maxTailLines
	}

	// Validate output format
	switch opts.output {
	case "text", jsonOutputVal:
		// Valid formats
	default:
		return fmt.Errorf("--format must be 'text' or 'json', got '%s'", opts.output)
	}

	// Validate level
	switch strings.ToLower(opts.level) {
	case "info", logLevelWarn, "warning", "error", logLevelDebug, "all":
		// Valid levels
	default:
		return fmt.Errorf("--level must be one of: info, warn, error, debug, all; got '%s'", opts.level)
	}

	// Validate min-level and its mutual exclusivity with --level and --context
	if opts.minLevel != "" {
		switch strings.ToLower(opts.minLevel) {
		case "info", logLevelWarn, "warning", "error", logLevelDebug:
			// Valid threshold levels
		default:
			return fmt.Errorf("--min-level must be one of: debug, info, warn, error; got '%s'", opts.minLevel)
		}
		if strings.ToLower(opts.level) != "all" {
			return fmt.Errorf("--min-level cannot be combined with an explicit --level (info/warn/error/debug); use one or the other")
		}
		if opts.contextLines > 0 {
			return fmt.Errorf("--min-level cannot be combined with --context")
		}
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

	// Validate source
	switch strings.ToLower(opts.source) {
	case string(LogSourceLocal), string(LogSourceAzure), string(LogSourceAll):
		// Valid sources - normalize to lowercase
		opts.source = strings.ToLower(opts.source)
	case "":
		// Default to local if not specified
		opts.source = string(LogSourceLocal)
	default:
		return fmt.Errorf("--source must be 'local', 'azure', or 'all'; got '%s'", opts.source)
	}

	return nil
}
