package commands

import (
	"context"
	"log/slog"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/require"
)

// Task 3.5 audit: every MCP tool argument that reaches the filesystem or a
// subprocess must be validated before use.
//
// The plan proposed attaching azdext.MCPSecurityPolicy to the server builder.
// That was not adopted as the gate. Two reasons, both pinned by tests below.
//
// First, the policy's URL half is inert here. No tool in this server accepts a
// URL and no tool performs outbound HTTP, so CheckURL, RequireHTTPS,
// BlockMetadataEndpoints, BlockPrivateNetworks and RedactHeaders have nothing
// to act on. TestNoToolAcceptsAURL keeps that true.
//
// Second, the policy's path half is strictly weaker than validateProjectDir.
// See TestSecurityPolicyIsWeakerThanValidateProjectDir for the specific inputs
// the SDK accepts and this extension rejects. Swapping would be a downgrade,
// and calling both would only add a redundant check that can never fire.
//
// The part of the task worth keeping is the audit itself, so it is expressed
// as a test rather than a document: TestEveryToolValidatesProjectDir drives
// every registered handler with a hostile projectDir, and
// TestEveryToolParameterIsClassified fails when a new parameter appears that
// nobody has classified as path-shaped or not.

// hostileProjectDirs are values a model could be steered into passing. Each
// must be rejected by every tool that accepts a project directory. They are
// chosen to be deterministic on any machine: the first is rejected by the
// leading-dash guard regardless of the filesystem, the second by the system
// directory denylist.
func hostileProjectDirs() []string {
	systemDir := "/etc/passwd"
	if runtime.GOOS == "windows" {
		systemDir = `C:\Windows\System32`
	}
	return []string{"-rf", systemDir}
}

// benignToolArgs supplies a valid value for every parameter this server
// declares, so a handler under audit reaches its projectDir check instead of
// short-circuiting on an unrelated missing or invalid argument.
//
// Adding a parameter without adding it here fails
// TestEveryToolParameterIsClassified, which is the point: a new parameter has
// to be consciously classified.
var benignToolArgs = map[string]any{
	"projectDir":     ".",
	"serviceName":    "api",
	"name":           "FOO_BAR",
	"value":          "example",
	"level":          "all",
	"source":         "local",
	"runtime":        "azd",
	"since":          "5m",
	"tail":           10,
	"contextLines":   2,
	"timeoutSeconds": 5,
	"wait":           false,
}

// pathShapedParams lists the parameters whose values are interpreted as
// filesystem paths. Every one of them must be validated by its handler.
//
// Membership here is what puts a tool in scope for
// TestEveryToolValidatesProjectDir. A path parameter missing from this map is
// therefore not merely undocumented, it is unaudited, which is why
// TestEveryToolParameterIsClassified refuses to accept a path-shaped name that
// is absent from both this map and knownNonPathParams.
var pathShapedParams = map[string]bool{
	"projectDir": true,
}

// pathShapedNameFragments are the substrings that make a parameter name look
// like it carries a filesystem path. The heuristic exists because the audit
// cannot tell what a handler does with a value it has never seen; it can only
// insist that a human decide. It mirrors the URL-shape heuristic in
// TestNoToolAcceptsAURL.
var pathShapedNameFragments = []string{"dir", "path", "file", "folder", "cwd"}

// knownNonPathParams records parameters whose names match a path-shaped
// fragment but whose values never reach the filesystem. Adding a name here is
// a deliberate statement, and the reason belongs beside it.
var knownNonPathParams = map[string]string{}

// looksPathShaped reports whether a parameter name suggests a filesystem path.
func looksPathShaped(param string) bool {
	lower := strings.ToLower(param)
	for _, fragment := range pathShapedNameFragments {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return false
}

// toolParamNames returns the declared input parameter names for a tool.
func toolParamNames(t *testing.T, tool *server.ServerTool) []string {
	t.Helper()
	names := make([]string, 0, len(tool.Tool.InputSchema.Properties))
	for name := range tool.Tool.InputSchema.Properties {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// callToolRequest builds the request shape a tool handler receives.
func callToolRequest(name string, args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	}
}

// resultText concatenates the textual content of a tool result.
func resultText(result *mcp.CallToolResult) string {
	return strings.Join(rawResultTexts(result), "\n")
}

func rawResultTexts(result *mcp.CallToolResult) []string {
	var out []string
	for _, c := range result.Content {
		if text, ok := c.(mcp.TextContent); ok {
			out = append(out, text.Text)
		}
	}
	return out
}

// toolResultTexts returns every text block in a tool result and fails the test
// when there is none.
//
// mcp.TextContent carries its payload in a plain Text field and implements no
// getter, so a type assertion against an inline getter interface never matches.
// Scanning loops written that way skip every element and assert nothing. This
// helper makes that failure mode impossible: a result with no text block is a
// test failure rather than a silent pass.
func toolResultTexts(t *testing.T, result *mcp.CallToolResult) []string {
	t.Helper()
	texts := rawResultTexts(result)
	require.NotEmpty(t, texts, "tool result carried no text content to inspect")
	return texts
}

// TestEveryToolParameterIsClassified fails when a tool declares a parameter
// that benignToolArgs does not cover, or a path-shaped parameter that nobody
// has classified as validated or exempt.
//
// The first check keeps TestEveryToolValidatesProjectDir honest: without it a
// new parameter would make a handler reject the call on a missing argument
// before reaching its path check, so the audit would pass while covering
// nothing.
//
// The second check closes the gap the first leaves open. Membership in
// benignToolArgs alone is enough to satisfy the first check, so a new path
// parameter added there and nowhere else would leave its handler outside the
// audit entirely, since declaresPathParam consults pathShapedParams.
func TestEveryToolParameterIsClassified(t *testing.T) {
	s := testBuildServer(t)

	var unclassified, unaudited, unjustified []string
	for toolName, tool := range s.ListTools() {
		for _, param := range toolParamNames(t, tool) {
			if _, ok := benignToolArgs[param]; !ok {
				unclassified = append(unclassified, toolName+"."+param)
			}
			if looksPathShaped(param) && !pathShapedParams[param] {
				reason, exempt := knownNonPathParams[param]
				switch {
				case !exempt:
					unaudited = append(unaudited, toolName+"."+param)
				case strings.TrimSpace(reason) == "":
					// The map is typed to carry a reason precisely so the
					// exemption is a documented decision. An empty value
					// exempts the parameter while recording nothing.
					unjustified = append(unjustified, toolName+"."+param)
				}
			}
		}
	}
	sort.Strings(unclassified)
	sort.Strings(unaudited)
	sort.Strings(unjustified)

	require.Empty(t, unclassified,
		"new MCP tool parameters are not covered by the security audit. "+
			"Add each to benignToolArgs, and to pathShapedParams if the value is used as a filesystem path: %v",
		unclassified)

	require.Empty(t, unaudited,
		"these parameters are named like filesystem paths but are not audited. "+
			"Add each to pathShapedParams so TestEveryToolValidatesProjectDir drives it, "+
			"or to knownNonPathParams with the reason it never reaches the filesystem: %v",
		unaudited)

	require.Empty(t, unjustified,
		"these parameters are exempted from the path audit with an empty reason. "+
			"knownNonPathParams carries a reason so the exemption is a decision on the record, "+
			"not an omission: %v",
		unjustified)
}

// TestEveryToolValidatesProjectDir drives every registered handler with a
// hostile project directory and requires the call to be refused. This is the
// audit that cannot rot: a tool added later that forgets to call
// validateProjectDir fails here without anyone remembering to update a list.
func TestEveryToolValidatesProjectDir(t *testing.T) {
	s := testBuildServer(t)

	covered := 0
	for toolName, tool := range s.ListTools() {
		params := toolParamNames(t, tool)
		if !declaresPathParam(params) {
			continue
		}
		covered++

		for _, hostile := range hostileProjectDirs() {
			for _, target := range params {
				if !pathShapedParams[target] {
					continue
				}
				// One hostile parameter per run. Setting every path-shaped
				// parameter at once lets a single rejection satisfy the whole
				// tool, so a second path parameter that no handler validates
				// would still pass on the strength of the first one's check.
				t.Run(toolName+"/"+target+"/"+hostile, func(t *testing.T) {
					args := map[string]any{}
					for _, p := range params {
						args[p] = benignToolArgs[p]
					}
					args[target] = hostile

					result, err := tool.Handler(context.Background(), callToolRequest(toolName, args))
					require.NoError(t, err, "handler must return a tool result, not a transport error")
					require.NotNil(t, result)
					require.True(t, result.IsError,
						"tool %s accepted hostile value %q for parameter %s", toolName, hostile, target)
					require.Contains(t, strings.ToLower(resultText(result)), "project directory",
						"tool %s rejected %q on %s for an unrelated reason, so the path check was never reached",
						toolName, hostile, target)
				})
			}
		}
	}

	require.NotZero(t, covered, "no tool declared a path-shaped parameter, so the audit covered nothing")
}

// TestNoToolAcceptsAURL pins the reason the URL half of
// azdext.MCPSecurityPolicy is not wired up. If a tool ever starts taking a URL,
// this fails and the policy (or an equivalent SSRF guard) has to be added
// before the parameter ships.
func TestNoToolAcceptsAURL(t *testing.T) {
	s := testBuildServer(t)

	var urlParams []string
	for toolName, tool := range s.ListTools() {
		for _, param := range toolParamNames(t, tool) {
			lower := strings.ToLower(param)
			if strings.Contains(lower, "url") || strings.Contains(lower, "uri") ||
				strings.Contains(lower, "endpoint") || strings.Contains(lower, "host") {
				urlParams = append(urlParams, toolName+"."+param)
			}
		}
	}

	require.Empty(t, urlParams,
		"a tool now accepts a URL-shaped parameter. Wire azdext.MCPSecurityPolicy "+
			"(or an equivalent SSRF guard) into its handler before shipping: %v", urlParams)
}

// TestSecurityPolicyIsWeakerThanValidateProjectDir records why
// azdext.MCPSecurityPolicy.CheckPath is not used as the project directory gate.
// Each case is a value the SDK policy allows and validateProjectDir refuses.
// If a future SDK closes one of these gaps, the corresponding subtest fails,
// which is the signal to reconsider adopting it.
func TestSecurityPolicyIsWeakerThanValidateProjectDir(t *testing.T) {
	home := t.TempDir()
	policy := azdext.NewMCPSecurityPolicy().ValidatePathsWithinBase(home)

	cases := []struct {
		name string
		path string
		why  string
	}{
		{
			name: "leading dash segment",
			path: filepath.Join(home, "-rf"),
			why: "the validated path is passed to a subprocess as --cwd <path>, so a segment " +
				"starting with a dash is argv injection (CWE-88). The SDK policy has no such guard",
		},
		{
			name: "path that does not exist",
			path: filepath.Join(home, "no-such-directory"),
			why:  "a project directory must exist. The SDK policy resolves the nearest existing ancestor and allows it",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, policy.CheckPath(tc.path),
				"the SDK policy was expected to allow %q. If it now rejects it, this gap has closed: %s",
				tc.path, tc.why)

			_, err := validateProjectDir(tc.path)
			require.Error(t, err, "validateProjectDir must reject %q because %s", tc.path, tc.why)
		})
	}

	t.Run("unconfigured policy allows everything", func(t *testing.T) {
		// The decisive difference: a policy with no base paths is a no-op, so
		// attaching DefaultMCPSecurityPolicy() (which configures no base paths)
		// would provide no path protection at all.
		empty := azdext.NewMCPSecurityPolicy()
		require.NoError(t, empty.CheckPath(filepath.Join(home, "..", "..", "anything")))
		require.NoError(t, azdext.DefaultMCPSecurityPolicy().CheckPath("/etc/passwd"))
	})
}

func declaresPathParam(params []string) bool {
	for _, p := range params {
		if pathShapedParams[p] {
			return true
		}
	}
	return false
}

// auditRecorder captures slog attribute values before a handler formats them.
//
// The built-in handlers quote a value containing control characters, so a
// test that inspects formatted output passes whether or not this code
// sanitizes the field. That tests slog, not us. Recording the raw attribute
// pins the property this code actually controls: what it hands to the logger.
// It also matches the threat, since the handler is configurable and a
// guarantee that holds only under the default configuration is not one.
type auditRecorder struct {
	records []map[string]string
}

func (r *auditRecorder) Enabled(context.Context, slog.Level) bool { return true }

func (r *auditRecorder) Handle(_ context.Context, rec slog.Record) error {
	fields := map[string]string{"msg": rec.Message}
	rec.Attrs(func(a slog.Attr) bool {
		fields[a.Key] = a.Value.String()
		return true
	})
	r.records = append(r.records, fields)
	return nil
}

func (r *auditRecorder) WithAttrs([]slog.Attr) slog.Handler { return r }
func (r *auditRecorder) WithGroup(string) slog.Handler      { return r }

// installAuditRecorder redirects the default logger for the duration of a test.
func installAuditRecorder(t *testing.T) *auditRecorder {
	t.Helper()

	recorder := &auditRecorder{}
	previous := slog.Default()
	slog.SetDefault(slog.New(recorder))
	t.Cleanup(func() { slog.SetDefault(previous) })

	return recorder
}

// TestRejectedProjectDirIsAudited pins that a refusal leaves a server-side
// trace. Without it the only record of a model probing the filesystem is the
// error string handed back to that same model.
func TestRejectedProjectDirIsAudited(t *testing.T) {
	recorder := installAuditRecorder(t)

	_, err := validateProjectDir("-rf")
	require.Error(t, err)

	require.Len(t, recorder.records, 1)
	require.Equal(t, "rejected project directory", recorder.records[0]["msg"])
	require.Contains(t, recorder.records[0]["path"], "-rf")
	require.NotEmpty(t, recorder.records[0]["reason"])
}

// TestAcceptedProjectDirIsNotAudited keeps the audit log signal-bearing. If
// every successful call logged, the warnings would be worthless.
func TestAcceptedProjectDirIsNotAudited(t *testing.T) {
	recorder := installAuditRecorder(t)

	dir := t.TempDir()
	if _, err := validateProjectDir(dir); err != nil {
		t.Skipf("temp dir %s is outside the allowed roots on this machine: %v", dir, err)
	}

	require.Empty(t, recorder.records, "an accepted project directory must not produce an audit warning")
}

// TestAuditedProjectDirIsSanitized pins that no attacker-supplied control
// character reaches the logger through any field of the audit record
// (CWE-117).
//
// The cases exit validateProjectDirCore through different branches. A leading
// dash is rejected before the path is resolved, and that branch formats the
// segment with %q, which escapes control characters on its own. The remaining
// cases survive the dash check and fail later, where the path is formatted
// verbatim. Only those exercise the sanitizer.
func TestAuditedProjectDirIsSanitized(t *testing.T) {
	tests := []struct {
		name string
		dir  string
	}{
		{name: "rejected at the leading dash check", dir: "-\x1b[2K\x1b[1Aevil"},
		{name: "rejected after path resolution", dir: "safe\x1b[2K\x1b[1Aevil"},
		{name: "newline forging a second record", dir: "safe\nlevel=WARN msg=\"forged\""},
		{name: "carriage return overwriting the record", dir: "safe\rlevel=WARN msg=\"forged\""},
		{name: "bell and backspace", dir: "safe\x07\x08evil"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorder := installAuditRecorder(t)

			_, err := validateProjectDir(tc.dir)
			require.Error(t, err)
			require.Len(t, recorder.records, 1, "a rejection must be audited exactly once")

			for field, value := range recorder.records[0] {
				require.False(t, containsControlChar(value),
					"audit field %q carried a control character: %q", field, value)
			}
		})
	}
}

// TestAuditedReasonIsSanitized pins the field that is easiest to overlook. The
// path is obviously caller-controlled, so it gets sanitized; the reason reads
// like server-side text, but several rejection branches format the caller's
// path straight into it, and neither filepath.Clean nor filepath.Abs strips
// control characters. Sanitizing one field and not the other protects nothing.
func TestAuditedReasonIsSanitized(t *testing.T) {
	recorder := installAuditRecorder(t)

	// Survives the dash check, so the failure is reported by a branch that
	// formats the resolved path verbatim into the error text.
	_, err := validateProjectDir("safe\x1b[2Kevil")
	require.Error(t, err)
	require.Contains(t, err.Error(), "\x1b",
		"this case is only meaningful while the raw error still carries the escape")

	require.Len(t, recorder.records, 1)
	require.False(t, containsControlChar(recorder.records[0]["reason"]),
		"the audit reason carried a control character: %q", recorder.records[0]["reason"])
}

// containsControlChar reports whether s holds any character that must not
// reach an audit record: the C0 range, DEL, and the C1 range. Tab, newline and
// carriage return are included deliberately, because an audit record is a
// single line and those are how a forged record gets appended to it.
func containsControlChar(s string) bool {
	for _, r := range s {
		if r <= 0x1F || r == 0x7F || (r >= 0x80 && r <= 0x9F) {
			return true
		}
	}
	return false
}
