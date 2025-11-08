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
