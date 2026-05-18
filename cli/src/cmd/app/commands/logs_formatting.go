package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

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
				fmt.Fprintf(&line, "[%s] ", timestamp)
			} else {
				line.WriteString(colorGray + "[" + timestamp + "]" + colorReset + " ")
			}
		}

		// Service name
		if noColor {
			fmt.Fprintf(&line, "[%s] ", entry.Service)
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

		_, _ = fmt.Fprintln(w, line.String())
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
			_, _ = fmt.Fprintln(w, "---")
		}

		// Show before context (if any)
		if entry.Context != nil && len(entry.Context.Before) > 0 {
			for _, line := range entry.Context.Before {
				if noColor {
					_, _ = fmt.Fprintf(w, "  %s\n", line)
				} else {
					_, _ = fmt.Fprintf(w, "  %s%s%s\n", colorGray, line, colorReset)
				}
			}
		}

		// Show the matching entry
		var line strings.Builder

		// Timestamp
		if showTimestamps {
			timestamp := entry.Timestamp.Format("15:04:05.000")
			if noColor {
				fmt.Fprintf(&line, "[%s] ", timestamp)
			} else {
				line.WriteString(colorGray + "[" + timestamp + "]" + colorReset + " ")
			}
		}

		// Service name
		if noColor {
			fmt.Fprintf(&line, "[%s] ", entry.Service)
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
			case logLevelWarn:
				line.WriteString(colorYellow + entry.Message + colorReset)
			case logLevelDebug:
				line.WriteString(colorGray + entry.Message + colorReset)
			default:
				line.WriteString(entry.Message)
			}
		}

		_, _ = fmt.Fprintln(w, line.String())

		// Show after context (if any)
		if entry.Context != nil && len(entry.Context.After) > 0 {
			for _, contextLine := range entry.Context.After {
				if noColor {
					_, _ = fmt.Fprintf(w, "  %s\n", contextLine)
				} else {
					_, _ = fmt.Fprintf(w, "  %s%s%s\n", colorGray, contextLine, colorReset)
				}
			}
		}
	}
}

// LogLevelAll is a sentinel value indicating no level filtering should be applied.
const LogLevelAll service.LogLevel = -1

// parseLogLevel parses a log level string.
