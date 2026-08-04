package commands

import (
	"bytes"
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
var pathShapedParams = map[string]bool{
	"projectDir": true,
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
// that benignToolArgs does not cover. Without this, a new parameter would
// silently make TestEveryToolValidatesProjectDir stop exercising the handler
// it was meant to cover, because the handler would reject the call on the
// missing argument before reaching its path check.
func TestEveryToolParameterIsClassified(t *testing.T) {
	s := testBuildServer(t)

	var unclassified []string
	for toolName, tool := range s.ListTools() {
		for _, param := range toolParamNames(t, tool) {
			if _, ok := benignToolArgs[param]; !ok {
				unclassified = append(unclassified, toolName+"."+param)
			}
		}
	}
	sort.Strings(unclassified)

	require.Empty(t, unclassified,
		"new MCP tool parameters are not covered by the security audit. "+
			"Add each to benignToolArgs, and to pathShapedParams if the value is used as a filesystem path: %v",
		unclassified)
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
			t.Run(toolName+"/"+hostile, func(t *testing.T) {
				args := map[string]any{}
				for _, p := range params {
					args[p] = benignToolArgs[p]
				}
				for _, p := range params {
					if pathShapedParams[p] {
						args[p] = hostile
					}
				}

				result, err := tool.Handler(context.Background(), callToolRequest(toolName, args))
				require.NoError(t, err, "handler must return a tool result, not a transport error")
				require.NotNil(t, result)
				require.True(t, result.IsError,
					"tool %s accepted hostile project directory %q", toolName, hostile)
				require.Contains(t, strings.ToLower(resultText(result)), "project directory",
					"tool %s rejected %q for an unrelated reason, so the path check was never reached",
					toolName, hostile)
			})
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

// TestRejectedProjectDirIsAudited pins that a refusal leaves a server-side
// trace. Without it the only record of a model probing the filesystem is the
// error string handed back to that same model.
func TestRejectedProjectDirIsAudited(t *testing.T) {
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	_, err := validateProjectDir("-rf")
	require.Error(t, err)

	logged := buf.String()
	require.Contains(t, logged, "rejected project directory")
	require.Contains(t, logged, "-rf")
}

// TestAcceptedProjectDirIsNotAudited keeps the audit log signal-bearing. If
// every successful call logged, the warnings would be worthless.
func TestAcceptedProjectDirIsNotAudited(t *testing.T) {
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	dir := t.TempDir()
	if _, err := validateProjectDir(dir); err != nil {
		t.Skipf("temp dir %s is outside the allowed roots on this machine: %v", dir, err)
	}

	require.Empty(t, buf.String(), "an accepted project directory must not produce an audit warning")
}

// TestAuditedProjectDirIsSanitized pins that a hostile path cannot inject
// terminal escapes or line breaks into the log stream (CWE-117).
func TestAuditedProjectDirIsSanitized(t *testing.T) {
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	_, err := validateProjectDir("-\x1b[2K\x1b[1Aevil")
	require.Error(t, err)
	require.NotContains(t, buf.String(), "\x1b", "ANSI escape reached the audit log")
}
