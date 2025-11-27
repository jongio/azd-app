// Package browser provides utilities for launching URLs in the user's web browser.
// It supports Windows, macOS, and Linux with automatic detection of the system
// default browser. The package handles cross-platform differences in browser
// launch commands and provides options for controlling launch behavior.
package browser

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Target represents the browser target for launching URLs.
type Target string

const (
	// TargetDefault uses the system default browser
	TargetDefault Target = "default"
	// TargetSystem uses the system default browser (alias for TargetDefault)
	TargetSystem Target = "system"
	// TargetNone disables browser launching
	TargetNone Target = "none"
)

// ValidTargets returns all valid browser target values.
func ValidTargets() []Target {
	return []Target{TargetDefault, TargetSystem, TargetNone}
}

// IsValid checks if a target string is valid.
func IsValid(target string) bool {
	t := Target(target)
	for _, valid := range ValidTargets() {
		if t == valid {
			return true
		}
	}
	return false
}

// ResolveTarget determines the actual browser target to use.
// Converts "default" to "system", and respects "none".
func ResolveTarget(target Target) Target {
	// If target is none, respect that
	if target == TargetNone {
		return TargetNone
	}

	// Convert default to system (they're aliases)
	return TargetSystem
}

// LaunchOptions contains options for launching a browser.
type LaunchOptions struct {
	// URL to open
	URL string
	// Target browser to use
	Target Target
	// Timeout for the launch command (default 5 seconds)
	Timeout time.Duration
}

// Launch opens the specified URL in the browser determined by the target.
// Returns an error if the launch fails, but this is not critical.
// The function is non-blocking and launches the browser in a separate goroutine.
func Launch(opts LaunchOptions) error {
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Second
	}

	// Validate URL - must be http or https
	if !strings.HasPrefix(opts.URL, "http://") && !strings.HasPrefix(opts.URL, "https://") {
		return fmt.Errorf("invalid URL scheme: URL must start with http:// or https://")
	}

	// Resolve the actual target
	target := ResolveTarget(opts.Target)

	// If target is none, don't launch
	if target == TargetNone {
		return nil
	}

	// Launch in goroutine to avoid blocking
	go func() {
		if err := launchSync(opts.URL, target, opts.Timeout); err != nil {
			// Log error but don't fail - this is non-critical
			fmt.Fprintf(os.Stderr, "⚠️  Could not open browser automatically: %v\n", err)
		}
	}()

	return nil
}

// launchSync performs the actual browser launch synchronously.
func launchSync(url string, target Target, timeout time.Duration) error {
	// Create context with timeout for proper cancellation
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var cmd *exec.Cmd

	switch target {
	case TargetSystem, TargetDefault:
		cmd = buildSystemCommandContext(ctx, url)
	default:
		return fmt.Errorf("unsupported browser target: %s", target)
	}

	// Execute command - context handles timeout automatically
	return cmd.Run()
}

// buildSystemCommandContext builds the command to launch the system default browser with context.
func buildSystemCommandContext(ctx context.Context, url string) *exec.Cmd {
	switch runtime.GOOS {
	case "windows":
		// Use 'start' command with empty title to avoid issues with URLs
		return exec.CommandContext(ctx, "cmd", "/c", "start", "", url)
	case "darwin":
		// macOS
		return exec.CommandContext(ctx, "open", url)
	case "linux":
		// Try xdg-open first (most common)
		if _, err := exec.LookPath("xdg-open"); err == nil {
			return exec.CommandContext(ctx, "xdg-open", url)
		}
		// Fallback to sensible-browser
		return exec.CommandContext(ctx, "sensible-browser", url)
	default:
		// Unknown OS - try xdg-open as a guess
		return exec.CommandContext(ctx, "xdg-open", url)
	}
}

// GetTargetDisplayName returns a human-readable name for the browser target.
func GetTargetDisplayName(target Target) string {
	resolved := ResolveTarget(target)

	switch resolved {
	case TargetSystem, TargetDefault:
		return "default browser"
	case TargetNone:
		return "none"
	default:
		return string(resolved)
	}
}

// FormatValidTargets returns a comma-separated list of valid targets.
func FormatValidTargets() string {
	targets := ValidTargets()
	strs := make([]string, len(targets))
	for i, t := range targets {
		strs[i] = string(t)
	}
	return strings.Join(strs, ", ")
}
