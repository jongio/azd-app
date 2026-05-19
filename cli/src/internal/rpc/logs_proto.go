package rpc

import (
	"sort"
	"strings"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// clampGetLogsTail mirrors handleGetLogs' tail-parameter handling:
// zero/negative -> 500, above 10k -> 10k. Returns the adjusted value
// and a flag indicating whether adjustment happened.
func clampGetLogsTail(requested int32) (int, bool) {
	if requested <= 0 {
		// Zero is the proto default - the caller didn't ask for
		// anything specific, so substituting the default is not a
		// clamp. Negative values are nonsensical but we treat them as
		// "use default" rather than erroring; flagging them as clamped
		// would be technically true but useless to clients (they can't
		// retry with a smaller positive value to "uncramp").
		return defaultGetLogsTail, false
	}
	if requested > maxGetLogsTail {
		return maxGetLogsTail, true
	}
	return int(requested), false
}

// fromProtoLogLevel maps the wire enum to the string ValidateClassificationLevel
// expects. Unknown / unspecified collapses to "" so validation rejects it
// rather than silently coercing to a default.
func fromProtoLogLevel(l v1.LogLevel) string {
	switch l {
	case v1.LogLevel_LOG_LEVEL_INFO:
		return "info"
	case v1.LogLevel_LOG_LEVEL_WARN:
		return "warning"
	case v1.LogLevel_LOG_LEVEL_ERROR:
		return "error"
	case v1.LogLevel_LOG_LEVEL_DEBUG:
		// Debug is a valid LogLevel for log entries but NOT a valid
		// classification level (ValidateClassificationLevel only
		// accepts info/warning/error). Return "" so AddClassification
		// rejects it with InvalidArgument instead of silently saving
		// an unparseable rule.
		return ""
	default:
		return ""
	}
}

// toProtoLogLevel maps service.LogLevel (int) to the wire enum.
func toProtoLogLevel(l service.LogLevel) v1.LogLevel {
	switch l {
	case service.LogLevelInfo:
		return v1.LogLevel_LOG_LEVEL_INFO
	case service.LogLevelWarn:
		return v1.LogLevel_LOG_LEVEL_WARN
	case service.LogLevelError:
		return v1.LogLevel_LOG_LEVEL_ERROR
	case service.LogLevelDebug:
		return v1.LogLevel_LOG_LEVEL_DEBUG
	default:
		return v1.LogLevel_LOG_LEVEL_UNSPECIFIED
	}
}

// classificationLevelToProto maps the string-typed classification level
// to the wire enum. Unknown strings collapse to UNSPECIFIED so a future
// stored value doesn't masquerade as info.
func classificationLevelToProto(level string) v1.LogLevel {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "info":
		return v1.LogLevel_LOG_LEVEL_INFO
	case "warning":
		return v1.LogLevel_LOG_LEVEL_WARN
	case "error":
		return v1.LogLevel_LOG_LEVEL_ERROR
	default:
		return v1.LogLevel_LOG_LEVEL_UNSPECIFIED
	}
}

// toProtoLogEntry converts a single service.LogEntry to its proto
// representation. Maps IsStderr to LogStream STDOUT/STDERR; sets
// LogSource based on service.LogEntry.Source ("local"/"azure"/empty);
// includes timestamp, service, level, and message.
//
// service.LogEntry.Sequence and AzureMetadata are intentionally NOT
// surfaced here: this RPC is for LOCAL logs only (StreamLocalLogs /
// GetLogs scope), so Azure-only fields would be permanently zero on the
// wire and confuse consumers. The Azure surface gets its own RPC.
func toProtoLogEntry(e service.LogEntry) *v1.LogEntry {
	stream := v1.LogStream_LOG_STREAM_STDOUT
	if e.IsStderr {
		stream = v1.LogStream_LOG_STREAM_STDERR
	}
	src := v1.LogSource_LOG_SOURCE_UNSPECIFIED
	switch e.Source {
	case service.LogSourceLocal, "":
		src = v1.LogSource_LOG_SOURCE_LOCAL
	case service.LogSourceAzure:
		src = v1.LogSource_LOG_SOURCE_AZURE
	}
	return &v1.LogEntry{
		Service:   e.Service,
		Message:   e.Message,
		Level:     toProtoLogLevel(e.Level),
		Timestamp: timestamppb.New(e.Timestamp),
		Stream:    stream,
		Source:    src,
	}
}

// toProtoLogEntries converts a slice of LogEntry. Sorted by timestamp
// for determinism (LogManager.GetAllLogs already sorts; per-buffer
// GetRecent is naturally ordered, but the merged path benefits from a
// belt-and-braces sort here).
func toProtoLogEntries(entries []service.LogEntry) []*v1.LogEntry {
	out := make([]*v1.LogEntry, len(entries))
	// Stable sort to preserve relative order of entries with identical
	// timestamps (e.g. burst-logged events from the same service).
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})
	for i := range entries {
		out[i] = toProtoLogEntry(entries[i])
	}
	return out
}

// toProtoClassification maps a single service.LogClassification.
func toProtoClassification(c service.LogClassification) *v1.Classification {
	return &v1.Classification{
		Text:  c.Text,
		Level: classificationLevelToProto(c.Level),
	}
}

// toProtoClassifications maps a slice of service.LogClassification.
func toProtoClassifications(in []service.LogClassification) []*v1.Classification {
	out := make([]*v1.Classification, len(in))
	for i := range in {
		out[i] = toProtoClassification(in[i])
	}
	return out
}

// defaultProtoPreferences returns the default Preferences value, mirroring
// dashboard.getDefaultPreferences (logs_config.go) exactly. Drift between
// these two is a bug; the test harness asserts equality of the relevant
// fields.
func defaultProtoPreferences() *v1.Preferences {
	return &v1.Preferences{
		Version: "1.0",
		// theme: empty string ("system") is the documented default in
		// the dashboard hook; absence of a stored value falls back to
		// the user-agent's preferred colour scheme. Leaving it empty
		// here matches that behaviour without forcing a value the
		// legacy code never wrote.
		Theme: "",
		Ui: &v1.UIPreferences{
			GridColumns:      2,
			GridAutoFit:      false,
			ViewMode:         "grid",
			SelectedServices: []string{},
		},
		Behavior: &v1.BehaviorPreferences{
			AutoScroll:      true,
			PauseOnScroll:   true,
			TimestampFormat: "hh:mm:ss.sss",
		},
		Copy: &v1.CopyPreferences{
			DefaultFormat:    "plaintext",
			IncludeTimestamp: true,
			IncludeService:   true,
		},
	}
}
