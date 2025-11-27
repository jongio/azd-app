// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"regexp"
	"strings"
	"sync"
)

// LogFilter provides pattern-based log filtering for service output.
type LogFilter struct {
	patterns    []*regexp.Regexp
	rawPatterns []string
	mu          sync.RWMutex
}

// BuiltInLogFilters contains patterns for common noise that doesn't indicate real problems.
// These are universal patterns that apply to most projects.
var BuiltInLogFilters = []string{
	// Electron DevTools protocol errors (universal for Electron apps)
	`Request Autofill\.enable failed`,
	`Request Autofill\.setAddresses failed`,
	`'Autofill\.\w+' wasn't found`,

	// npm registry credential warnings (environment-specific but very common)
	`npm warn Unknown env config`,

	// Node.js debugger messages (informational, not errors)
	`Debugger listening on ws://`,
	`Debugger attached`,
	`For help, see: https://nodejs.org/en/docs/inspector`,

	// Vite/esbuild common warnings
	`\[vite\] warning: .*node_modules`,

	// Common false-positive error patterns
	`ExperimentalWarning:`,
	`DeprecationWarning:`,
}

// NewLogFilter creates a new log filter with the given patterns.
// Patterns are compiled as case-insensitive regular expressions.
func NewLogFilter(patterns []string) (*LogFilter, error) {
	lf := &LogFilter{
		patterns:    make([]*regexp.Regexp, 0, len(patterns)),
		rawPatterns: make([]string, 0, len(patterns)),
	}

	for _, pattern := range patterns {
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			return nil, err
		}
		lf.patterns = append(lf.patterns, re)
		lf.rawPatterns = append(lf.rawPatterns, pattern)
	}

	return lf, nil
}

// NewLogFilterWithBuiltins creates a log filter that includes built-in patterns
// plus any additional custom patterns.
func NewLogFilterWithBuiltins(customPatterns []string) (*LogFilter, error) {
	allPatterns := make([]string, 0, len(BuiltInLogFilters)+len(customPatterns))
	allPatterns = append(allPatterns, BuiltInLogFilters...)
	allPatterns = append(allPatterns, customPatterns...)
	return NewLogFilter(allPatterns)
}

// ShouldFilter returns true if the message matches any filter pattern.
func (lf *LogFilter) ShouldFilter(message string) bool {
	if lf == nil {
		return false
	}

	lf.mu.RLock()
	defer lf.mu.RUnlock()

	for _, re := range lf.patterns {
		if re.MatchString(message) {
			return true
		}
	}
	return false
}

// AddPattern adds a new pattern to the filter.
func (lf *LogFilter) AddPattern(pattern string) error {
	lf.mu.Lock()
	defer lf.mu.Unlock()

	re, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		return err
	}

	lf.patterns = append(lf.patterns, re)
	lf.rawPatterns = append(lf.rawPatterns, pattern)
	return nil
}

// GetPatterns returns a copy of the raw pattern strings.
func (lf *LogFilter) GetPatterns() []string {
	lf.mu.RLock()
	defer lf.mu.RUnlock()

	result := make([]string, len(lf.rawPatterns))
	copy(result, lf.rawPatterns)
	return result
}

// PatternCount returns the number of patterns in the filter.
func (lf *LogFilter) PatternCount() int {
	lf.mu.RLock()
	defer lf.mu.RUnlock()
	return len(lf.patterns)
}

// LogFilterConfig represents log filter configuration from azure.yaml.
type LogFilterConfig struct {
	// Patterns to filter out (suppress) from output
	Exclude []string `yaml:"exclude,omitempty"`
	// Whether to include built-in patterns (default: true)
	IncludeBuiltins *bool `yaml:"includeBuiltins,omitempty"`
}

// LogsConfig represents the logs configuration section in azure.yaml.
// This is the root-level configuration for all logging-related settings.
type LogsConfig struct {
	// Filters for suppressing noisy log output
	Filters *LogFilterConfig `yaml:"filters,omitempty"`
	// Future: add more log-related settings here
	// Examples: level, format, output, retention, etc.
}

// GetFilters returns the filter config, or nil if not set.
func (c *LogsConfig) GetFilters() *LogFilterConfig {
	if c == nil {
		return nil
	}
	return c.Filters
}

// ShouldIncludeBuiltins returns true if built-in patterns should be included.
// Defaults to true if not specified.
func (c *LogFilterConfig) ShouldIncludeBuiltins() bool {
	if c == nil || c.IncludeBuiltins == nil {
		return true
	}
	return *c.IncludeBuiltins
}

// BuildLogFilter creates a LogFilter from the config.
func (c *LogFilterConfig) BuildLogFilter() (*LogFilter, error) {
	if c == nil {
		return NewLogFilterWithBuiltins(nil)
	}

	if c.ShouldIncludeBuiltins() {
		return NewLogFilterWithBuiltins(c.Exclude)
	}
	return NewLogFilter(c.Exclude)
}

// FilterLogEntries filters a slice of log entries based on the filter patterns.
func FilterLogEntries(entries []LogEntry, filter *LogFilter) []LogEntry {
	if filter == nil {
		return entries
	}

	result := make([]LogEntry, 0, len(entries))
	for _, entry := range entries {
		if !filter.ShouldFilter(entry.Message) {
			result = append(result, entry)
		}
	}
	return result
}

// ParseExcludePatterns parses a comma-separated string of exclude patterns.
func ParseExcludePatterns(excludeStr string) []string {
	if excludeStr == "" {
		return nil
	}

	patterns := strings.Split(excludeStr, ",")
	result := make([]string, 0, len(patterns))
	for _, p := range patterns {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
