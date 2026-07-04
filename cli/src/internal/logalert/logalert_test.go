package logalert

import (
	"testing"
	"time"
)

func mustEngine(t *testing.T, rules []Rule, includeDefaults bool) *Engine {
	t.Helper()
	e, err := NewEngine(rules, includeDefaults)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	return e
}

func TestNewEngineInvalidPatternAtLoad(t *testing.T) {
	_, err := NewEngine([]Rule{{Name: "bad", Pattern: "("}}, false)
	if err == nil {
		t.Fatal("expected an error for an invalid regex at load time")
	}
}

func TestNewEngineEmptyName(t *testing.T) {
	if _, err := NewEngine([]Rule{{Name: "", Pattern: "x"}}, false); err == nil {
		t.Fatal("expected an error for an empty rule name")
	}
}

func TestMatchBasic(t *testing.T) {
	e := mustEngine(t, []Rule{{Name: "err", Pattern: "ERROR", Severity: SeverityWarning}}, false)
	now := time.Now()

	alerts := e.Match("web", "some ERROR happened", now)
	if len(alerts) != 1 {
		t.Fatalf("got %d alerts, want 1", len(alerts))
	}
	a := alerts[0]
	if a.Rule != "err" || a.Service != "web" || a.Severity != SeverityWarning {
		t.Errorf("unexpected alert: %+v", a)
	}
	if a.Line != "some ERROR happened" {
		t.Errorf("alert line = %q", a.Line)
	}
}

func TestMatchNonMatchingNeverNotifies(t *testing.T) {
	e := mustEngine(t, []Rule{{Name: "err", Pattern: "ERROR"}}, false)
	if alerts := e.Match("web", "everything is fine", time.Now()); alerts != nil {
		t.Fatalf("non-matching line produced alerts: %+v", alerts)
	}
}

func TestMatchDefaultSeverity(t *testing.T) {
	e := mustEngine(t, []Rule{{Name: "err", Pattern: "ERROR"}}, false)
	alerts := e.Match("web", "ERROR", time.Now())
	if len(alerts) != 1 || alerts[0].Severity != SeverityWarning {
		t.Fatalf("expected default warning severity, got %+v", alerts)
	}
}

func TestMatchServiceScoping(t *testing.T) {
	e := mustEngine(t, []Rule{{Name: "api-only", Pattern: "boom", Service: "api"}}, false)
	now := time.Now()

	if alerts := e.Match("web", "boom", now); alerts != nil {
		t.Errorf("rule scoped to api should not fire for web: %+v", alerts)
	}
	if alerts := e.Match("api", "boom", now); len(alerts) != 1 {
		t.Errorf("rule scoped to api should fire for api, got %+v", alerts)
	}
}

func TestMatchDisabledRule(t *testing.T) {
	e := mustEngine(t, []Rule{{Name: "err", Pattern: "ERROR", Disabled: true}}, false)
	if alerts := e.Match("web", "ERROR", time.Now()); alerts != nil {
		t.Errorf("disabled rule fired: %+v", alerts)
	}
}

func TestMatchDebounce(t *testing.T) {
	e := mustEngine(t, []Rule{{Name: "err", Pattern: "ERROR", Debounce: 10 * time.Second}}, false)
	base := time.Now()

	if alerts := e.Match("web", "ERROR one", base); len(alerts) != 1 {
		t.Fatalf("first match should fire, got %+v", alerts)
	}
	// Within the debounce window: suppressed.
	if alerts := e.Match("web", "ERROR two", base.Add(5*time.Second)); alerts != nil {
		t.Fatalf("match within debounce window should be suppressed, got %+v", alerts)
	}
	// After the window: fires again.
	if alerts := e.Match("web", "ERROR three", base.Add(11*time.Second)); len(alerts) != 1 {
		t.Fatalf("match after debounce window should fire, got %+v", alerts)
	}
}

func TestMatchDebounceIsPerService(t *testing.T) {
	e := mustEngine(t, []Rule{{Name: "err", Pattern: "ERROR", Debounce: time.Minute}}, false)
	now := time.Now()

	if alerts := e.Match("web", "ERROR", now); len(alerts) != 1 {
		t.Fatalf("web should fire, got %+v", alerts)
	}
	// A different service is tracked independently, so it still fires.
	if alerts := e.Match("api", "ERROR", now); len(alerts) != 1 {
		t.Fatalf("api should fire independently of web, got %+v", alerts)
	}
}

func TestDefaultRulesFireOnPanic(t *testing.T) {
	e := mustEngine(t, nil, true)
	if e.RuleCount() != len(DefaultRules()) {
		t.Fatalf("RuleCount = %d, want %d", e.RuleCount(), len(DefaultRules()))
	}
	alerts := e.Match("worker", "goroutine panic: nil pointer", time.Now())
	if len(alerts) != 1 || alerts[0].Rule != "panic" || alerts[0].Severity != SeverityCritical {
		t.Fatalf("panic default rule did not fire as expected: %+v", alerts)
	}
}

func TestUserRuleOverridesDefault(t *testing.T) {
	// Disable the built-in panic rule by overriding it with the same name.
	e := mustEngine(t, []Rule{{Name: "panic", Pattern: `\bpanic\b`, Disabled: true}}, true)
	if alerts := e.Match("worker", "panic: boom", time.Now()); alerts != nil {
		t.Fatalf("overridden+disabled panic rule should not fire: %+v", alerts)
	}
	// Count is unchanged: the override replaced the default in place.
	if e.RuleCount() != len(DefaultRules()) {
		t.Fatalf("override should not change rule count, got %d", e.RuleCount())
	}
}

func TestNilEngineSafe(t *testing.T) {
	var e *Engine
	if alerts := e.Match("web", "ERROR", time.Now()); alerts != nil {
		t.Errorf("nil engine should return no alerts")
	}
	if e.RuleCount() != 0 {
		t.Errorf("nil engine RuleCount should be 0")
	}
}
