package commands

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/logalert"
	"github.com/jongio/azd-app/cli/src/internal/notifications"
)

// buildAlertEngine returns a log alert engine when alerts are enabled, or nil
// when they are not. Invalid built-in rules would surface here at load time.
func buildAlertEngine(enabled bool) (*logalert.Engine, error) {
	if !enabled {
		return nil, nil
	}
	engine, err := logalert.NewEngine(nil, true)
	if err != nil {
		return nil, fmt.Errorf("failed to load log alert rules: %w", err)
	}
	return engine, nil
}

// alertToEvent converts a log alert into a notification pipeline event so it can
// be surfaced through the existing notifications path and the dashboard.
func alertToEvent(a logalert.Alert) notifications.Event {
	return notifications.Event{
		Type:        notifications.EventError,
		ServiceName: a.Service,
		Message:     fmt.Sprintf("[%s] %s", a.Rule, strings.TrimSpace(a.Line)),
		Severity:    string(a.Severity),
		Timestamp:   a.Time,
		Metadata: map[string]any{
			"rule": a.Rule,
			"line": a.Line,
		},
	}
}

// renderAlertBanner writes a single, easy-to-spot alert line for the CLI stream.
func renderAlertBanner(w io.Writer, a logalert.Alert) {
	_, _ = fmt.Fprintf(w, "!! alert [%s/%s] %s: %s\n",
		a.Severity, a.Rule, a.Service, strings.TrimSpace(a.Line))
}

// emitAlerts evaluates a log line against the engine and renders any alerts that
// fire. It is a no-op when the engine is nil, keeping the streaming path
// unchanged unless alerts are enabled.
func (e *logsExecutor) emitAlerts(service, message string, w io.Writer) {
	if e.alertEngine == nil {
		return
	}
	for _, alert := range e.alertEngine.Match(service, message, time.Now()) {
		renderAlertBanner(w, alert)
	}
}
