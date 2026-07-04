package commands

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/logalert"
	"github.com/jongio/azd-app/cli/src/internal/notifications"
)

func TestBuildAlertEngine(t *testing.T) {
	engine, err := buildAlertEngine(false)
	if err != nil {
		t.Fatalf("buildAlertEngine(false): %v", err)
	}
	if engine != nil {
		t.Error("expected nil engine when alerts are disabled")
	}

	engine, err = buildAlertEngine(true)
	if err != nil {
		t.Fatalf("buildAlertEngine(true): %v", err)
	}
	if engine == nil || engine.RuleCount() == 0 {
		t.Error("expected a populated engine when alerts are enabled")
	}
}

func TestAlertToEvent(t *testing.T) {
	now := time.Now()
	a := logalert.Alert{Rule: "panic", Service: "web", Line: "  goroutine panic  ", Severity: logalert.SeverityCritical, Time: now}
	ev := alertToEvent(a)

	if ev.Type != notifications.EventError {
		t.Errorf("Type = %v, want EventError", ev.Type)
	}
	if ev.ServiceName != "web" {
		t.Errorf("ServiceName = %q", ev.ServiceName)
	}
	if ev.Severity != "critical" {
		t.Errorf("Severity = %q", ev.Severity)
	}
	if !strings.Contains(ev.Message, "panic") || !strings.Contains(ev.Message, "goroutine panic") {
		t.Errorf("Message = %q", ev.Message)
	}
	if ev.Metadata["rule"] != "panic" {
		t.Errorf("Metadata[rule] = %v", ev.Metadata["rule"])
	}
	if !ev.Timestamp.Equal(now) {
		t.Errorf("Timestamp = %v, want %v", ev.Timestamp, now)
	}
}

func TestRenderAlertBanner(t *testing.T) {
	var buf bytes.Buffer
	renderAlertBanner(&buf, logalert.Alert{Rule: "fatal", Service: "api", Line: "fatal error: oom", Severity: logalert.SeverityCritical})
	out := buf.String()
	for _, want := range []string{"alert", "fatal", "api", "fatal error: oom"} {
		if !strings.Contains(out, want) {
			t.Errorf("banner missing %q: %q", want, out)
		}
	}
}

func TestEmitAlertsNilEngineIsNoOp(t *testing.T) {
	e := &logsExecutor{opts: &logsOptions{}}
	var buf bytes.Buffer
	e.emitAlerts("web", "panic: boom", &buf)
	if buf.Len() != 0 {
		t.Errorf("nil engine should not write anything, got %q", buf.String())
	}
}

func TestEmitAlertsWritesBannerOnMatch(t *testing.T) {
	engine, err := logalert.NewEngine(nil, true)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	e := &logsExecutor{opts: &logsOptions{}, alertEngine: engine}

	var buf bytes.Buffer
	e.emitAlerts("worker", "panic: nil pointer dereference", &buf)
	if !strings.Contains(buf.String(), "panic") {
		t.Errorf("expected a panic alert banner, got %q", buf.String())
	}

	// A benign line produces no banner.
	buf.Reset()
	e.emitAlerts("worker", "request handled in 3ms", &buf)
	if buf.Len() != 0 {
		t.Errorf("benign line should not produce an alert, got %q", buf.String())
	}
}
