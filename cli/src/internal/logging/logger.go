// Package logging provides a structured logging abstraction built on top of logutil.
// It supports configurable log levels, structured JSON output, and debug mode.
// Component-based logging enables filtering by service or operation context.
package logging

import (
	"io"
	"log/slog"
	"os"

	"github.com/jongio/azd-core/logutil"
)

// Level re-exports the logutil level type used by the app logger.
type Level = logutil.Level

// LevelDebug and related constants re-export logutil log levels for app logging.
const (
	LevelDebug = logutil.LevelDebug
	LevelInfo  = logutil.LevelInfo
	LevelWarn  = logutil.LevelWarn
	LevelError = logutil.LevelError
)

// isStructured tracks whether structured JSON logging is enabled (app-specific).
var isStructured bool

// Logger wraps logutil.ComponentLogger with app-specific test event methods.
type Logger struct {
	*logutil.ComponentLogger
}

// NewLogger creates a logger with a component context.
// The component name appears in all log output for filtering.
func NewLogger(component string) *Logger {
	return &Logger{
		ComponentLogger: logutil.NewLogger(component),
	}
}

// TestStarted logs a test execution start event.
func (l *Logger) TestStarted(service, testFile string) {
	l.Info("test started",
		"event", "test_started",
		"service", service,
		"file", testFile,
	)
}

// TestCompleted logs a test execution completion event.
func (l *Logger) TestCompleted(service string, passed, failed, skipped int, duration float64) {
	l.Info("test completed",
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
	l.Info("coverage collected",
		"event", "coverage_collected",
		"service", service,
		"coverage_pct", coverage,
	)
}

func init() {
	SetupLogger(false, false)
}

// SetupLogger configures the global logger.
// debug enables debug-level logging
// structured enables structured JSON logging
func SetupLogger(debug, structured bool) {
	isStructured = structured
	logutil.SetupLogger(debug, structured)
}

// SetLevel sets the logging level.
func SetLevel(level Level) {
	logutil.SetLevel(level)
}

// SetOutput sets the output destination for logs.
func SetOutput(w io.Writer) {
	logutil.SetOutput(w)
}

// IsDebugEnabled returns true if debug logging is enabled.
func IsDebugEnabled() bool {
	return logutil.IsDebugEnabled() || os.Getenv("AZD_APP_DEBUG") == "true"
}

// IsStructured returns true if structured JSON logging is enabled.
func IsStructured() bool {
	return isStructured
}

// Debug logs a debug message with optional key-value pairs.
func Debug(msg string, args ...any) {
	if IsDebugEnabled() {
		logutil.Logger().Debug(msg, args...)
	}
}

// Info logs an info message with optional key-value pairs.
func Info(msg string, args ...any) {
	logutil.Info(msg, args...)
}

// Warn logs a warning message with optional key-value pairs.
func Warn(msg string, args ...any) {
	logutil.Warn(msg, args...)
}

// Error logs an error message with optional key-value pairs.
func Error(msg string, args ...any) {
	logutil.Error(msg, args...)
}

// With creates a new logger with the given attributes.
func With(args ...any) *slog.Logger {
	return logutil.Logger().With(args...)
}

// ParseLevel parses a string into a Level.
var ParseLevel = logutil.ParseLevel
