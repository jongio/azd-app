// Package commands - MCP output sanitization to prevent ANSI/control-character
// prompt injection (CWE-150, CWE-117).
//
// Threat: a malicious service can write ANSI escape sequences to stderr that
// erase the user-visible transcript in some terminals while injecting hidden
// instructions into the LLM context window via MCP tool results.
//
// Defense: strip all ANSI/VT escape sequences and dangerous C0/C1 control
// characters from every string that exits via an MCP channel, both on the
// success path (marshalToolResult) and the error path (mcpErrorResult).
package commands

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/mark3labs/mcp-go/mcp"
)

// ansiStripper matches ANSI/VT escape sequences that must not reach the LLM:
//
//   - CSI sequences:     ESC [ <params> <final-byte>  (e.g. \x1b[2K, \x1b[1;32m)
//   - Charset sequences: ESC ( or ) <designator>      (e.g. \x1b(B)
//   - OSC sequences:     ESC ] <text> BEL             (e.g. \x1b]0;title\a)
var ansiStripper = regexp.MustCompile(
	`\x1b\[[0-9;]*[a-zA-Z]` + // CSI: ESC [ params final-byte
		`|\x1b[()][0-9A-Z]` + // Charset: ESC ( or ) designator
		`|\x1b\].*?\x07`, // OSC: ESC ] text BEL  (non-greedy)
)

// sanitizeForLLM strips ANSI escape sequences and dangerous C0/C1 control
// characters from s before it is included in an MCP tool response.
//
// Preserved: \t (0x09), \n (0x0A), \r (0x0D) — safe for LLM consumption.
// Stripped:  ANSI CSI/OSC/charset sequences; C0 control chars 0x00-0x08,
// 0x0B, 0x0C, 0x0E-0x1F (includes bare ESC); C1 control chars 0x80-0x9F.
func sanitizeForLLM(s string) string {
	// First pass: remove recognised multi-char ANSI/VT sequences.
	s = ansiStripper.ReplaceAllString(s, "")

	// Second pass: strip any remaining dangerous control characters.
	// Allocate the same capacity as the input — common case is no stripping.
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if isSafeRune(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// isSafeRune returns true if r is safe to include in an LLM-facing string.
//
//   - Allowed whitespace: \t (0x09), \n (0x0A), \r (0x0D)
//   - Stripped: C0 control chars 0x00-0x1F (minus the three above)
//   - Stripped: C1 control chars 0x80-0x9F
//   - All other runes (printable ASCII, Unicode text) pass through.
func isSafeRune(r rune) bool {
	switch r {
	case '\t', '\n', '\r':
		return true // preserve safe whitespace
	}
	switch {
	case r <= 0x1F: // C0 control chars not allowed above (includes ESC 0x1B)
		return false
	case r >= 0x80 && r <= 0x9F: // C1 control chars
		return false
	default:
		return true
	}
}

// sanitizeForAudit prepares s for inclusion in a single-line audit record.
//
// It strips everything sanitizeForLLM strips, then collapses the whitespace
// sanitizeForLLM deliberately preserves. A tool response may legitimately span
// several lines; an audit record may not, because a newline in an attacker
// controlled field is how a forged record gets appended to the log.
//
// The default slog handlers quote a value that needs it, which escapes these
// characters today. This does not rely on that: the handler is configurable,
// and a guarantee that holds only under the default configuration is not a
// guarantee.
func sanitizeForAudit(s string) string {
	return auditWhitespace.ReplaceAllString(sanitizeForLLM(s), " ")
}

// auditWhitespace matches the whitespace sanitizeForLLM preserves, which must
// not survive into a single-line audit record.
var auditWhitespace = regexp.MustCompile(`[\t\n\r]+`)

// sanitizeAny recursively sanitizes all string values in a JSON-decoded value.
//
// json.Unmarshal into any produces a fixed set of Go types: nil, bool,
// float64, string, []any, and map[string]any. Non-string scalars are returned
// unchanged; strings are passed through sanitizeForLLM; containers are walked
// recursively so every nested string is covered.
func sanitizeAny(v any) any {
	switch val := v.(type) {
	case string:
		return sanitizeForLLM(val)
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, v2 := range val {
			result[k] = sanitizeAny(v2)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, v2 := range val {
			result[i] = sanitizeAny(v2)
		}
		return result
	default:
		return val // nil, bool, float64 — no string content
	}
}

// redacted is the placeholder emitted in place of a sensitive env var value
// in MCP tool responses. Full redaction (vs. the partial masking used in CLI
// display) is required because LLM context windows must not receive any portion
// of a secret.
const redacted = "***REDACTED***"

// sensitiveSuffixes are case-folded key suffixes that indicate a secret value.
var sensitiveSuffixes = []string{
	"_TOKEN", "_SECRET", "_KEY", "_PASSWORD", "_CREDENTIAL", "_APIKEY",
}

// sensitivePrefixes are case-folded key prefixes that indicate a secret value.
var sensitivePrefixes = []string{
	"SECRET_", "TOKEN_",
}

// redactEnvVarForMCP returns redacted if key matches a known secret-naming
// pattern; otherwise it returns value unchanged.
//
// This function is used exclusively in MCP tool responses where the result
// reaches an LLM context window. Full redaction is intentional — even a
// partial leak (e.g. first/last two chars) is unacceptable for LLM output.
//
// For CLI display, see redactSecretValue in core_helpers.go which applies
// partial masking with a different UX intent.
//
// Matching is case-insensitive so both API_KEY and api_key are caught.
// Patterns checked:
//   - Suffixes: _TOKEN, _SECRET, _KEY, _PASSWORD, _CREDENTIAL, _APIKEY
//   - Prefixes: SECRET_, TOKEN_
//   - Exact:    PASSWORD, SECRET
func redactEnvVarForMCP(key, value string) string {
	upper := strings.ToUpper(key)

	// Exact matches first — cheap O(1) check.
	if upper == "PASSWORD" || upper == "SECRET" {
		return redacted
	}

	// Suffix matches.
	for _, suffix := range sensitiveSuffixes {
		if strings.HasSuffix(upper, suffix) {
			return redacted
		}
	}

	// Prefix matches.
	for _, prefix := range sensitivePrefixes {
		if strings.HasPrefix(upper, prefix) {
			return redacted
		}
	}

	return value
}

// mcpErrorResult formats an error message, sanitizes it for LLM safety, and
// returns an MCP error result with IsError=true.
//
// Use this in place of azdext.MCPErrorResult throughout the commands package.
// The sanitization prevents ANSI/control-character injection via error messages
// that may contain subprocess output (e.g. stderr captured from child processes).
func mcpErrorResult(format string, args ...any) *mcp.CallToolResult {
	msg := fmt.Sprintf(format, args...)
	return azdext.MCPErrorResult("%s", sanitizeForLLM(msg))
}
