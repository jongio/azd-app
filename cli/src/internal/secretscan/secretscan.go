// Package secretscan provides an advisory scan that flags environment values
// which look like real secrets sitting in tracked configuration. The run
// preflight uses it to warn a developer before services start.
//
// The scan is deliberately advisory. It never blocks a run and never changes
// the environment passed to child processes, matching the orchestrator's stance
// that credential isolation belongs to the deployment environment. Its only job
// is to point out a secret that has been written as a literal value into a file
// that is tracked by source control, so it can be moved to a Key Vault reference
// or a gitignored .env file before it spreads through clones and CI logs.
package secretscan

import (
	"regexp"
	"sort"
	"strings"
)

// Finding describes one environment value that looks like a secret.
type Finding struct {
	// Source identifies where the value was read from, for example
	// "azure.yaml (service: api)" or ".env".
	Source string
	// Key is the environment variable name.
	Key string
	// Detail explains, in one short phrase, why the value was flagged.
	Detail string
}

// strongSecretKey matches variable names that strongly imply the value is a
// secret. Bare "key" is intentionally excluded because names like
// KEY_VAULT_NAME, PARTITION_KEY, or SSH_KEY_PATH are not themselves secrets.
var strongSecretKey = regexp.MustCompile(`(?i)(password|passwd|\bpwd\b|secret|client[_-]?secret|(^|_)token($|_)|credential|private[_-]?key|api[_-]?key|access[_-]?key|secret[_-]?key|connection[_-]?string|conn[_-]?str|sas[_-]?token)`)

// jwtPattern matches a JSON Web Token: three base64url segments joined by dots.
var jwtPattern = regexp.MustCompile(`^eyJ[A-Za-z0-9_-]{5,}\.[A-Za-z0-9_-]{5,}\.[A-Za-z0-9_-]{5,}$`)

// hexSecret matches a long run of hex, common for keys and hashes.
var hexSecret = regexp.MustCompile(`^[0-9a-fA-F]{32,}$`)

// base64ish matches a long base64 or base64url token.
var base64ish = regexp.MustCompile(`^[A-Za-z0-9+/=_-]{24,}$`)

// connStringSecret matches connection strings that embed a credential.
var connStringSecret = regexp.MustCompile(`(?i)(password|pwd|accountkey|sharedaccesskey|accesskey)=[^;\s]+`)

// placeholderValues are common stand-in values that are not real secrets.
var placeholderValues = map[string]bool{
	"changeme": true, "change-me": true, "changethis": true, "replaceme": true,
	"password": true, "secret": true, "token": true, "example": true,
	"examplevalue": true, "dummy": true, "placeholder": true, "todo": true,
	"tbd": true, "none": true, "null": true, "test": true, "sample": true,
}

// Inspect reports whether a single key and value look like a hardcoded secret
// and, when they do, a short reason. It returns false for empty values,
// references (Key Vault or variable indirection), and obvious placeholders.
func Inspect(key, value string) (bool, string) {
	v := strings.TrimSpace(value)
	if v == "" || isReference(v) || isPlaceholder(v) {
		return false, ""
	}

	// Value-shape detectors fire regardless of the key name.
	switch {
	case jwtPattern.MatchString(v):
		return true, "value looks like a JSON Web Token"
	case connStringSecret.MatchString(v):
		return true, "value looks like a connection string with an embedded credential"
	case hexSecret.MatchString(v):
		return true, "value looks like a long hex key"
	case base64ish.MatchString(v) && hasLetter(v) && hasDigit(v):
		return true, "value looks like a high-entropy token"
	}

	// Secret-named keys flag any remaining literal value that is not a path,
	// URL, boolean, or number.
	if strongSecretKey.MatchString(key) && looksLikeLiteralSecret(v) {
		return true, "secret-named variable holds a literal value"
	}
	return false, ""
}

// ScanEnv inspects every entry in env and returns the findings, sorted by key
// so output is stable across runs.
func ScanEnv(source string, env map[string]string) []Finding {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var findings []Finding
	for _, k := range keys {
		if flagged, detail := Inspect(k, env[k]); flagged {
			findings = append(findings, Finding{Source: source, Key: k, Detail: detail})
		}
	}
	return findings
}

// redactionKey matches variable names whose value must never be displayed.
// It is deliberately broader than strongSecretKey: bare "key" is included
// because for display redaction, over-masking is harmless while under-masking
// leaks a credential. strongSecretKey stays narrow because a false positive
// there produces a spurious warning about a file the developer must then edit.
var redactionKey = regexp.MustCompile(`(?i)(password|passwd|\bpwd\b|secret|token|key|credential|conn(ection)?[_-]?str(ing)?|sas)`)

// redactionMask is the placeholder substituted for a secret value.
const redactionMask = "***"

// RedactValue masks value when key or value indicates a credential, returning
// a short prefix and suffix so a developer can still correlate the value
// without learning it. Non-secret values are returned unchanged.
//
// Value-shape detection is limited to JWTs and connection strings with an
// embedded credential. Those two patterns have a negligible false-positive
// rate. The looser high-entropy heuristics used by Inspect are deliberately
// NOT applied here: a resource name like "rg-myapp-prod-eastus-001" trips the
// base64 shape test, and masking it would hide data the developer needs.
func RedactValue(key, value string) string {
	if value == "" {
		return value
	}
	if !redactionKey.MatchString(key) &&
		!jwtPattern.MatchString(value) &&
		!connStringSecret.MatchString(value) {
		return value
	}
	if len(value) <= 4 {
		return redactionMask
	}
	return value[:2] + redactionMask + value[len(value)-2:]
}

// RedactMap returns a copy of env with every secret-looking value masked by
// RedactValue. It always returns a non-nil map so callers can assign the
// result without a nil check. The input is never mutated.
func RedactMap(env map[string]string) map[string]string {
	out := make(map[string]string, len(env))
	for k, v := range env {
		out[k] = RedactValue(k, v)
	}
	return out
}

// isReference reports whether v is a Key Vault reference or a variable
// indirection rather than a literal value. These are the recommended way to
// carry a secret and must never be flagged.
func isReference(v string) bool {
	switch {
	case strings.HasPrefix(v, "@Microsoft.KeyVault("):
		return true
	case strings.Contains(v, "${"), strings.Contains(v, "$("):
		return true
	case strings.HasPrefix(v, "$") && !strings.ContainsAny(v, " \t"):
		return true
	case strings.HasPrefix(v, "%") && strings.HasSuffix(v, "%"):
		return true
	default:
		return false
	}
}

// isPlaceholder reports whether v is a common stand-in rather than a real
// secret, such as changeme, <your-key>, {password}, or a run of one character.
func isPlaceholder(v string) bool {
	lower := strings.ToLower(v)
	if placeholderValues[lower] {
		return true
	}
	if wrapped(v, '<', '>') || wrapped(v, '{', '}') {
		return true
	}
	if strings.HasPrefix(lower, "your-") || strings.HasPrefix(lower, "your_") || strings.HasPrefix(lower, "yourvalue") {
		return true
	}
	if len(v) > 0 && strings.Count(v, string(v[0])) == len(v) {
		return true // e.g. ***** or xxxxx
	}
	return false
}

// looksLikeLiteralSecret excludes values that a secret-named key might
// legitimately hold: filesystem paths, URLs, booleans, and plain numbers.
func looksLikeLiteralSecret(v string) bool {
	if isBoolOrNumber(v) {
		return false
	}
	if strings.Contains(v, "://") {
		return false
	}
	if isPathLike(v) {
		return false
	}
	return true
}

func wrapped(v string, open, closing byte) bool {
	return len(v) >= 2 && v[0] == open && v[len(v)-1] == closing
}

func isPathLike(v string) bool {
	switch {
	case strings.HasPrefix(v, "/"), strings.HasPrefix(v, "./"),
		strings.HasPrefix(v, "../"), strings.HasPrefix(v, "~"):
		return true
	case len(v) >= 3 && v[1] == ':' && (v[2] == '\\' || v[2] == '/'):
		return true // Windows drive path such as C:\ or C:/
	default:
		return false
	}
}

func isBoolOrNumber(v string) bool {
	switch strings.ToLower(v) {
	case "true", "false", "yes", "no", "on", "off":
		return true
	}
	for _, r := range v {
		if (r < '0' || r > '9') && r != '.' && r != '-' && r != '+' {
			return false
		}
	}
	return v != ""
}

func hasLetter(v string) bool {
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return true
		}
	}
	return false
}

func hasDigit(v string) bool {
	for _, r := range v {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}
