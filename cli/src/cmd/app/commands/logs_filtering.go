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
	if includeBuiltins {
		return service.NewLogFilterWithBuiltins(customPatterns)
	}
	return service.NewLogFilter(customPatterns)
}

// getOrCreateSignalChan gets or creates a signal channel with proper cleanup.
// This avoids duplication and race conditions in signal handling setup.

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
	case "text", jsonOutputVal:
		// Valid formats
	default:
		return fmt.Errorf("--format must be 'text' or 'json', got '%s'", opts.format)
	}

	// Validate level
	switch strings.ToLower(opts.level) {
	case "info", logLevelWarn, "warning", "error", logLevelDebug, "all":
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
