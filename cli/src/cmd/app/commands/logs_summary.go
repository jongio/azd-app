package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

// LogSummaryResult is the machine-readable shape for "logs --summary".
type LogSummaryResult struct {
	Total     int                 `json:"total"`
	StartTime *time.Time          `json:"startTime,omitempty"`
	EndTime   *time.Time          `json:"endTime,omitempty"`
	Services  []LogServiceSummary `json:"services"`
}

// LogServiceSummary contains counts for one service.
type LogServiceSummary struct {
	Service string `json:"service"`
	Info    int    `json:"info"`
	Warn    int    `json:"warn"`
	Error   int    `json:"error"`
	Debug   int    `json:"debug"`
	Unknown int    `json:"unknown,omitempty"`
	Total   int    `json:"total"`
}

type logSummaryEntry struct {
	service   string
	level     string
	timestamp time.Time
}

func buildLogSummaryFromEntries(entries []service.LogEntry) LogSummaryResult {
	summaryEntries := make([]logSummaryEntry, 0, len(entries))
	for _, entry := range entries {
		summaryEntries = append(summaryEntries, logSummaryEntry{
			service:   entry.Service,
			level:     logLevelToString(entry.Level),
			timestamp: entry.Timestamp,
		})
	}
	return buildLogSummary(summaryEntries)
}

func buildLogSummaryFromContextLogs(entries []LogEntryWithContext) LogSummaryResult {
	summaryEntries := make([]logSummaryEntry, 0, len(entries))
	for _, entry := range entries {
		summaryEntries = append(summaryEntries, logSummaryEntry{
			service:   entry.Service,
			level:     entry.Level,
			timestamp: entry.Timestamp,
		})
	}
	return buildLogSummary(summaryEntries)
}

func buildLogSummary(entries []logSummaryEntry) LogSummaryResult {
	byService := make(map[string]*LogServiceSummary)
	var start, end *time.Time

	for _, entry := range entries {
		serviceName := strings.TrimSpace(entry.service)
		if serviceName == "" {
			serviceName = "(unknown)"
		}

		row := byService[serviceName]
		if row == nil {
			row = &LogServiceSummary{Service: serviceName}
			byService[serviceName] = row
		}

		switch normalizeLogSummaryLevel(entry.level) {
		case "info":
			row.Info++
		case logLevelWarn:
			row.Warn++
		case statusError:
			row.Error++
		case logLevelDebug:
			row.Debug++
		default:
			row.Unknown++
		}
		row.Total++

		if !entry.timestamp.IsZero() {
			ts := entry.timestamp
			if start == nil || ts.Before(*start) {
				start = &ts
			}
			if end == nil || ts.After(*end) {
				end = &ts
			}
		}
	}

	services := make([]LogServiceSummary, 0, len(byService))
	for _, row := range byService {
		services = append(services, *row)
	}
	sort.Slice(services, func(i, j int) bool {
		return services[i].Service < services[j].Service
	})

	return LogSummaryResult{
		Total:     len(entries),
		StartTime: start,
		EndTime:   end,
		Services:  services,
	}
}

func normalizeLogSummaryLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "info":
		return "info"
	case logLevelWarn, "warning":
		return logLevelWarn
	case statusError:
		return statusError
	case logLevelDebug:
		return logLevelDebug
	default:
		return "unknown"
	}
}

func displayLogSummary(summary LogSummaryResult, w io.Writer, asJSON bool) {
	if asJSON {
		encoder := json.NewEncoder(w)
		if err := encoder.Encode(summary); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to encode log summary: %v\n", err)
		}
		return
	}

	_, _ = fmt.Fprintf(w, "Total log entries: %d\n", summary.Total)
	if summary.StartTime != nil && summary.EndTime != nil {
		_, _ = fmt.Fprintf(w, "Time range: %s to %s\n", summary.StartTime.Format(time.RFC3339), summary.EndTime.Format(time.RFC3339))
	}
	if len(summary.Services) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "%-20s %6s %6s %6s %6s %8s %6s\n", "Service", "Info", "Warn", "Error", "Debug", "Unknown", "Total")
	for _, row := range summary.Services {
		_, _ = fmt.Fprintf(w, "%-20s %6d %6d %6d %6d %8d %6d\n",
			row.Service, row.Info, row.Warn, row.Error, row.Debug, row.Unknown, row.Total)
	}
}
