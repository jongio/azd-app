// Package logging provides a structured logging abstraction built on top of slog.
// It supports configurable log levels, structured JSON output, and debug mode.
// Component-based logging enables filtering by service or operation context.
package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Level represents the logging level.
type Level int

const (
	// LevelDebug is for debug messages.
	LevelDebug Level = iota
	// LevelInfo is for informational messages.
	LevelInfo
	// LevelWarn is for warnings.
	LevelWarn
	// LevelError is for errors.
	LevelError
)

var (
	// globalLogger is the default logger instance
	globalLogger *slog.Logger
	// currentLevel is the current log level
	currentLevel = LevelInfo
	// isStructured tracks whether structured JSON logging is enabled
	isStructured = false
	// outputWriter tracks the current output destination
	outputWriter io.Writer = os.Stderr
)

// Logger provides component-scoped logging with context propagation.
type Logger struct {
	slogger   *slog.Logger
	component string
}

// NewLogger creates a logger with a component context.
// The component name appears in all log output for filtering.
func NewLogger(component string) *Logger {
	return &Logger{
		slogger:   globalLogger.With("component", component),
		component: component,
	}
}

// WithService returns a new logger with service context added.
func (l *Logger) WithService(service string) *Logger {
	return &Logger{
		slogger:   l.slogger.With("service", service),
		component: l.component,
	}
}

// WithOperation returns a new logger with operation context added.
func (l *Logger) WithOperation(operation string) *Logger {
	return &Logger{
		slogger:   l.slogger.With("operation", operation),
		component: l.component,
	}
}

// WithFields returns a new logger with additional context fields.
func (l *Logger) WithFields(args ...any) *Logger {
	return &Logger{
		slogger:   l.slogger.With(args...),
		component: l.component,
	}
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, args ...any) {
	if IsDebugEnabled() {
		l.slogger.Debug(msg, args...)
	}
}

// Info logs an info message.
func (l *Logger) Info(msg string, args ...any) {
	l.slogger.Info(msg, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, args ...any) {
	l.slogger.Warn(msg, args...)
}

// Error logs an error message.
func (l *Logger) Error(msg string, args ...any) {
	l.slogger.Error(msg, args...)
}

// TestStarted logs a test execution start event.
func (l *Logger) TestStarted(service, testFile string) {
	l.slogger.Info("test started",
		"event", "test_started",
		"service", service,
		"file", testFile,
	)
}

// TestCompleted logs a test execution completion event.
func (l *Logger) TestCompleted(service string, passed, failed, skipped int, duration float64) {
	l.slogger.Info("test completed",
		"event", "test_completed",
		"service", service,
		"passed", passed,
		"failed", failed,
		"skipped", skipped,
		"duration_sec", duration,
	)
}

// CoverageCollected logs a coverage collection event.
func (l *Logger) CoverageCollected(service string, coverage float64) {
	l.slogger.Info("coverage collected",
		"event", "coverage_collected",
		"service", service,
		"coverage_pct", coverage,
	)
}

func init() {
	// Initialize with default logger (no-op for non-debug mode)
	SetupLogger(false, false)
}

// SetupLogger configures the global logger.
// debug enables debug-level logging
// structured enables structured JSON logging
func SetupLogger(debug, structured bool) {
	var level slog.Level
	if debug {
		level = slog.LevelDebug
		currentLevel = LevelDebug
	} else {
		level = slog.LevelInfo
		currentLevel = LevelInfo
	}

	// Track the handler type for later use
	isStructured = structured
	outputWriter = os.Stderr

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: level,
	}

	if structured {
		// JSON structured logging
		handler = slog.NewJSONHandler(outputWriter, opts)
	} else {
		// Text logging for human consumption
		handler = slog.NewTextHandler(outputWriter, opts)
	}

	globalLogger = slog.New(handler)
	slog.SetDefault(globalLogger)
}

// SetLevel sets the logging level.
func SetLevel(level Level) {
	currentLevel = level
	var slogLevel slog.Level
	switch level {
	case LevelDebug:
		slogLevel = slog.LevelDebug
	case LevelInfo:
		slogLevel = slog.LevelInfo
	case LevelWarn:
		slogLevel = slog.LevelWarn
	case LevelError:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	// Preserve the handler type (structured vs text) when changing levels
	opts := &slog.HandlerOptions{Level: slogLevel}
	var handler slog.Handler
	if isStructured {
		handler = slog.NewJSONHandler(outputWriter, opts)
	} else {
		handler = slog.NewTextHandler(outputWriter, opts)
	}
	globalLogger = slog.New(handler)
	slog.SetDefault(globalLogger)
}

// SetOutput sets the output destination for logs.
func SetOutput(w io.Writer) {
	outputWriter = w

	var slogLevel slog.Level
	switch currentLevel {
	case LevelDebug:
		slogLevel = slog.LevelDebug
	case LevelInfo:
		slogLevel = slog.LevelInfo
	case LevelWarn:
		slogLevel = slog.LevelWarn
	case LevelError:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: slogLevel}
	var handler slog.Handler
	if isStructured {
		handler = slog.NewJSONHandler(outputWriter, opts)
	} else {
		handler = slog.NewTextHandler(outputWriter, opts)
	}
	globalLogger = slog.New(handler)
	slog.SetDefault(globalLogger)
}

// IsDebugEnabled returns true if debug logging is enabled.
func IsDebugEnabled() bool {
	return currentLevel == LevelDebug || os.Getenv("AZD_APP_DEBUG") == "true"
}

// IsStructured returns true if structured JSON logging is enabled.
func IsStructured() bool {
	return isStructured
}

// Debug logs a debug message with optional key-value pairs.
func Debug(msg string, args ...any) {
	if IsDebugEnabled() {
		globalLogger.Debug(msg, args...)
	}
}

// Info logs an info message with optional key-value pairs.
func Info(msg string, args ...any) {
	globalLogger.Info(msg, args...)
}

// Warn logs a warning message with optional key-value pairs.
func Warn(msg string, args ...any) {
	globalLogger.Warn(msg, args...)
}

// Error logs an error message with optional key-value pairs.
func Error(msg string, args ...any) {
	globalLogger.Error(msg, args...)
}

// With creates a new logger with the given attributes.
func With(args ...any) *slog.Logger {
	return globalLogger.With(args...)
}

// ParseLevel parses a string into a Level.
func ParseLevel(s string) Level {
	switch strings.ToLower(s) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}
