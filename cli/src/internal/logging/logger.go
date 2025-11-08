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
)

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

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: level,
	}

	if structured {
		// JSON structured logging
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		// Text logging for human consumption
		handler = slog.NewTextHandler(os.Stderr, opts)
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

	opts := &slog.HandlerOptions{Level: slogLevel}
	handler := slog.NewTextHandler(os.Stderr, opts)
	globalLogger = slog.New(handler)
	slog.SetDefault(globalLogger)
}

// SetOutput sets the output destination for logs.
func SetOutput(w io.Writer) {
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	handler := slog.NewTextHandler(w, opts)
	globalLogger = slog.New(handler)
	slog.SetDefault(globalLogger)
}

// IsDebugEnabled returns true if debug logging is enabled.
func IsDebugEnabled() bool {
	return currentLevel == LevelDebug || os.Getenv("AZD_APP_DEBUG") == "true"
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
