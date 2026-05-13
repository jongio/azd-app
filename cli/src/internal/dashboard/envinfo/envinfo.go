// Package envinfo detects the dashboard runtime environment (GitHub
// Codespaces + AZD) so URL rewriting and environment-name surfacing can
// be driven from a single source of truth across transports.
//
// Lives in its own package (rather than directly under dashboard) so the
// Connect-RPC handler in cli/src/internal/rpc can consume the same logic
// without creating an import cycle with the dashboard package itself.
package envinfo

import (
	"context"
	"os"
	"os/exec"
	"strings"
)

// Info describes the GitHub Codespace + AZD environment context the
// dashboard uses to decide whether localhost URLs need to be rewritten
// for browser Codespaces, and which AZD environment is currently active.
//
// Adding a new field here automatically lights it up on every transport.
type Info struct {
	// Codespace describes the GitHub Codespace we are running inside, if
	// any. Codespace.Enabled is false outside Codespaces; the other
	// fields are zero-valued in that case.
	Codespace Codespace

	// EnvironmentName is the value of AZURE_ENV_NAME, identifying the
	// active AZD environment. Empty when no environment is selected.
	EnvironmentName string
}

// Codespace carries the subset of Codespace state the dashboard uses
// for URL rewriting. See https://docs.github.com/en/codespaces for the
// upstream environment variables consumed below.
type Codespace struct {
	// Enabled reports whether the process is running inside a Codespace.
	// Driven by the presence of CODESPACE_NAME.
	Enabled bool

	// Name is the Codespace's machine name (CODESPACE_NAME). Used as the
	// host prefix when rewriting localhost URLs.
	Name string

	// Domain is the port-forwarding domain for this Codespace
	// (GITHUB_CODESPACES_PORT_FORWARDING_DOMAIN). Defaults to
	// "app.github.dev" when running in a Codespace without an explicit
	// override, matching the upstream default.
	Domain string

	// IsVsCodeDesktop is true when running inside VS Code desktop
	// (including VS Code desktop attached to a Codespace). In that case
	// localhost URLs work natively and the dashboard skips rewriting.
	IsVsCodeDesktop bool
}

// Detect inspects process environment + the optional `code` CLI to
// populate an Info. It is safe to call from multiple goroutines but
// invokes a child process (`code --status`) when running inside a
// Codespace, so callers should plumb a cancellable context.
//
// Behavior matches the legacy /api/environment REST handler exactly so
// either transport returns the same payload byte-for-byte.
func Detect(ctx context.Context) Info {
	codespaceName := os.Getenv("CODESPACE_NAME")
	codespacePortDomain := os.Getenv("GITHUB_CODESPACES_PORT_FORWARDING_DOMAIN")

	// Default domain when running inside a Codespace without an explicit
	// override. Mirrors the upstream `app.github.dev` convention.
	if codespaceName != "" && codespacePortDomain == "" {
		codespacePortDomain = "app.github.dev"
	}

	return Info{
		Codespace: Codespace{
			Enabled:         codespaceName != "",
			Name:            codespaceName,
			Domain:          codespacePortDomain,
			IsVsCodeDesktop: runningOnVsCodeDesktop(ctx),
		},
		EnvironmentName: os.Getenv("AZURE_ENV_NAME"),
	}
}

// vsCodeStatusProbe is the production probe runningOnVsCodeDesktop calls.
// Tests inject a stub by reassigning this variable so they can exercise
// both branches without the host needing a `code` CLI installed.
var vsCodeStatusProbe = func(ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "code", "--status")
	return cmd.Output()
}

// runningOnVsCodeDesktop reports whether the dashboard is running inside
// VS Code desktop. When VS Code desktop is available (including when
// attached to a Codespace) localhost URLs work natively without
// transformation. In browser-based Codespaces, `code --status` returns
// "The --status argument is not yet supported in browsers." (see
// azure/azure-dev cli/azd/cmd/auth_login.go runningOnCodespacesBrowser).
func runningOnVsCodeDesktop(ctx context.Context) bool {
	// Outside a Codespace there is nothing to detect; the dashboard does
	// not perform URL rewriting in that case anyway.
	if os.Getenv("CODESPACES") != "true" {
		return false
	}

	output, err := vsCodeStatusProbe(ctx)
	if err != nil {
		// Failure to spawn `code` (missing CLI, sandbox, etc.) is treated
		// as "not desktop" so the dashboard falls back to URL rewriting.
		// That is the safer default — it produces working URLs in browser
		// Codespaces at the cost of an unnecessary rewrite when the CLI
		// is just absent.
		return false
	}

	return !strings.Contains(string(output), "The --status argument is not yet supported in browsers")
}
