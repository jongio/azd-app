package service

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// customRedactionMask replaces text matched by a user-supplied redaction rule.
const customRedactionMask = "***"

var (
	customRedactionMu       sync.RWMutex
	customRedactionPatterns []*regexp.Regexp
	customRedactionLiterals []string
)

// RegisterLogRedaction installs project-specific redaction rules that
// MaskSecretsInLogLine applies on top of the built-in patterns. Each entry in
// patterns is a regular expression whose matches are masked; each entry in
// literals is masked as an exact substring wherever it appears.
//
// Invalid regular expressions are skipped and returned as errors so the caller
// can warn without aborting the run. Registration replaces any rules installed
// by a previous call and is meant to run once at startup, before log streaming
// begins.
func RegisterLogRedaction(patterns, literals []string) []error {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	var errs []error
	for _, p := range patterns {
		if strings.TrimSpace(p) == "" {
			continue
		}
		re, err := regexp.Compile(p)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid redaction pattern %q: %w", p, err))
			continue
		}
		compiled = append(compiled, re)
	}

	lits := make([]string, 0, len(literals))
	for _, l := range literals {
		if l == "" {
			continue
		}
		lits = append(lits, l)
	}

	customRedactionMu.Lock()
	customRedactionPatterns = compiled
	customRedactionLiterals = lits
	customRedactionMu.Unlock()
	return errs
}

// ResetLogRedaction clears all custom redaction rules. It is intended for tests.
func ResetLogRedaction() {
	customRedactionMu.Lock()
	customRedactionPatterns = nil
	customRedactionLiterals = nil
	customRedactionMu.Unlock()
}

// applyCustomRedaction masks custom patterns and literals in message. It reads
// a snapshot of the registered rules under a read lock; the underlying slices
// are never mutated in place, so using the snapshot after unlocking is safe.
func applyCustomRedaction(message string) string {
	customRedactionMu.RLock()
	patterns := customRedactionPatterns
	literals := customRedactionLiterals
	customRedactionMu.RUnlock()

	for _, re := range patterns {
		message = re.ReplaceAllString(message, customRedactionMask)
	}
	for _, lit := range literals {
		message = strings.ReplaceAll(message, lit, customRedactionMask)
	}
	return message
}
