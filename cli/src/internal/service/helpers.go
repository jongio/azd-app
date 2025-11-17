package service

import (
	"io"
	"log/slog"
)

// SafeClose is a helper function for deferred close operations that logs errors.
// Use this in defer statements to avoid ignoring close errors while keeping code clean.
//
// Example:
//
//	defer SafeClose(resp.Body, "response body")
func SafeClose(closer io.Closer, description string) {
	if closer == nil {
		return
	}
	if err := closer.Close(); err != nil {
		slog.Debug("failed to close resource", "resource", description, "error", err)
	}
}

// SafeCloseWithContext is like SafeClose but accepts additional context fields.
func SafeCloseWithContext(closer io.Closer, description string, contextFields ...any) {
	if closer == nil {
		return
	}
	if err := closer.Close(); err != nil {
		fields := []any{"resource", description, "error", err}
		fields = append(fields, contextFields...)
		slog.Debug("failed to close resource", fields...)
	}
}
