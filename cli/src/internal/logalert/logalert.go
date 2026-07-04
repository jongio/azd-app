// Package logalert evaluates streaming service log lines against regex alert
// rules. A matching line produces an Alert that callers can surface through the
// notification pipeline or print directly. Rules can be scoped to a single
// service, disabled, and debounced so a noisy loop does not flood alerts.
package logalert

import (
	"fmt"
	"regexp"
	"sync"
	"time"
)

// Severity describes how important an alert is.
type Severity string

// Severity levels, aligned with the notification pipeline's severity strings.
const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// DefaultDebounce is the minimum time between alerts for the same rule and
// service when a rule does not set its own debounce window.
const DefaultDebounce = 30 * time.Second

// Rule is a single log alert rule.
type Rule struct {
	// Name identifies the rule and is used as part of the debounce key.
	Name string
	// Pattern is the regular expression matched against each log line.
	Pattern string
	// Service, when set, scopes the rule to a single service. Empty matches all.
	Service string
	// Severity is reported with the alert. Defaults to warning when empty.
	Severity Severity
	// Disabled turns the rule off without removing it.
	Disabled bool
	// Debounce overrides DefaultDebounce for this rule when greater than zero.
	Debounce time.Duration
}

// Alert is produced when a log line matches a rule.
type Alert struct {
	Rule     string
	Service  string
	Line     string
	Severity Severity
	Time     time.Time
}

// compiledRule pairs a rule with its compiled regular expression.
type compiledRule struct {
	rule Rule
	re   *regexp.Regexp
}

// Engine matches log lines against a set of rules with per-rule and per-service
// debounce. It is safe for concurrent use.
type Engine struct {
	mu              sync.Mutex
	rules           []compiledRule
	defaultDebounce time.Duration
	lastFired       map[string]time.Time
}

// DefaultRules returns built-in rules for common failure signals. They can be
// disabled by passing a rule with the same name and Disabled set to true.
func DefaultRules() []Rule {
	return []Rule{
		{Name: "panic", Pattern: `(?i)\bpanic\b`, Severity: SeverityCritical},
		{Name: "unhandled-exception", Pattern: `(?i)unhandled\s+exception`, Severity: SeverityCritical},
		{Name: "fatal", Pattern: `(?i)\bfatal\b`, Severity: SeverityCritical},
	}
}

// NewEngine compiles the given rules into an engine. When includeDefaults is
// true, DefaultRules are added first and any user rule with the same name
// overrides the default. Invalid regular expressions are reported here, at load
// time, rather than during matching.
func NewEngine(rules []Rule, includeDefaults bool) (*Engine, error) {
	var all []Rule
	if includeDefaults {
		all = append(all, DefaultRules()...)
	}
	all = append(all, rules...)

	e := &Engine{
		defaultDebounce: DefaultDebounce,
		lastFired:       make(map[string]time.Time),
	}
	index := make(map[string]int)
	for _, r := range all {
		if r.Name == "" {
			return nil, fmt.Errorf("alert rule has an empty name")
		}
		re, err := regexp.Compile(r.Pattern)
		if err != nil {
			return nil, fmt.Errorf("alert rule %q has an invalid pattern %q: %w", r.Name, r.Pattern, err)
		}
		cr := compiledRule{rule: r, re: re}
		if i, ok := index[r.Name]; ok {
			e.rules[i] = cr
			continue
		}
		index[r.Name] = len(e.rules)
		e.rules = append(e.rules, cr)
	}
	return e, nil
}

// Match evaluates one log line for a service and returns the alerts that fire at
// time now. Disabled rules, and service-scoped rules that do not match the
// service, are skipped. Repeated matches within a rule's debounce window are
// suppressed. Non-matching lines return nil.
func (e *Engine) Match(service, line string, now time.Time) []Alert {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	var alerts []Alert
	for _, cr := range e.rules {
		r := cr.rule
		if r.Disabled {
			continue
		}
		if r.Service != "" && r.Service != service {
			continue
		}
		if !cr.re.MatchString(line) {
			continue
		}

		key := r.Name + "\x00" + service
		debounce := r.Debounce
		if debounce <= 0 {
			debounce = e.defaultDebounce
		}
		if last, ok := e.lastFired[key]; ok && now.Sub(last) < debounce {
			continue
		}
		e.lastFired[key] = now

		severity := r.Severity
		if severity == "" {
			severity = SeverityWarning
		}
		alerts = append(alerts, Alert{
			Rule:     r.Name,
			Service:  service,
			Line:     line,
			Severity: severity,
			Time:     now,
		})
	}
	return alerts
}

// RuleCount returns the number of loaded rules, including disabled ones.
func (e *Engine) RuleCount() int {
	if e == nil {
		return 0
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.rules)
}
