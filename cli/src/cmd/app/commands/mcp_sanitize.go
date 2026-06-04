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
