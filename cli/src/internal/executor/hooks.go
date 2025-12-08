package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/output"
)

// Shell type constants for platform-specific shell detection
const (
	ShellSh         = "sh"
	ShellBash       = "bash"
	ShellPwsh       = "pwsh"
	ShellPowerShell = "powershell"
	ShellCmd        = "cmd"
	ShellZsh        = "zsh"
)

// HookConfig represents the configuration for executing a hook.
type HookConfig struct {
	Run             string   // Script or command to execute
	Shell           string   // Shell to use (sh, bash, pwsh, etc.)
	ContinueOnError bool     // Continue if hook fails
	Interactive     bool     // Requires user interaction
	Env             []string // Additional environment variables (KEY=VALUE format)
}

// ExecuteHook executes a lifecycle hook with the given configuration.
// It handles platform-specific shell selection and respects the hook's error handling settings.
func ExecuteHook(ctx context.Context, hookName string, config HookConfig, workingDir string) error {
	if config.Run == "" {
		return nil // No hook configured
	}

	// Determine shell to use
	shell := config.Shell
	if shell == "" {
		shell = getDefaultShell()
	}

	// Display hook execution start
	output.Info("🪝 Executing %s hook...", hookName)
	if !output.IsJSON() {
		output.Item("Script: %s", config.Run)
		output.Item("Shell: %s", shell)
		output.Newline()
	}

	// Prepare command with environment variables
	cmd := prepareHookCommand(ctx, shell, config.Run, workingDir, config.Env)

	// Configure stdio
	configureCommandIO(cmd, config.Interactive)

	// Execute the hook
	err := cmd.Run()

	// Handle execution result
	if err != nil {
		if config.ContinueOnError {
			output.Warning("Hook %s failed but continuing (continueOnError: true)", hookName)
			output.Item("Error: %v", err)
			output.Newline()
			return nil
		}
		return fmt.Errorf("hook %s failed: %w", hookName, err)
	}

	output.Success("Hook %s completed successfully", hookName)
	output.Newline()
	return nil
}

// prepareHookCommand prepares the command based on the shell and script.
// envVars is an optional list of additional environment variables to pass to the hook (in KEY=VALUE format)
func prepareHookCommand(ctx context.Context, shell, script, workingDir string, envVars []string) *exec.Cmd {
	var cmd *exec.Cmd

	// Determine shell arguments based on shell type
	shellLower := strings.ToLower(shell)
	switch {
	case strings.Contains(shellLower, "pwsh") || strings.Contains(shellLower, "powershell"):
		// PowerShell: use -Command for inline scripts
		// Set OutputEncoding to UTF-8 to properly display Unicode characters (emojis)
		wrappedScript := fmt.Sprintf("[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; %s", script)
		cmd = exec.CommandContext(ctx, shell, "-Command", wrappedScript)
	case strings.Contains(shellLower, "cmd"):
		// Windows CMD: use /c for commands
		cmd = exec.CommandContext(ctx, shell, "/c", script)
	default:
		// POSIX shells (sh, bash, zsh, etc.): use -c for commands
		cmd = exec.CommandContext(ctx, shell, "-c", script)
	}

	cmd.Dir = workingDir

	// Start with inherited environment variables
	cmd.Env = os.Environ()

	// Append additional hook-specific environment variables
	if len(envVars) > 0 {
		cmd.Env = append(cmd.Env, envVars...)
	}

	return cmd
}

// configureCommandIO configures stdin, stdout, and stderr for the command based on interactive mode and output format.
func configureCommandIO(cmd *exec.Cmd, interactive bool) {
	// Configure stdin based on interactive mode
	if interactive {
		cmd.Stdin = os.Stdin
	} else {
		cmd.Stdin = nil
	}

	// In JSON mode, suppress output unless interactive
	if output.IsJSON() && !interactive {
		cmd.Stdout = nil
		cmd.Stderr = nil
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
}

// getDefaultShell returns the default shell for the current platform.
func getDefaultShell() string {
	if runtime.GOOS == "windows" {
		// Check if PowerShell is available (preferred on Windows)
		if _, err := exec.LookPath(ShellPwsh); err == nil {
			return ShellPwsh
		}
		if _, err := exec.LookPath(ShellPowerShell); err == nil {
			return ShellPowerShell
		}
		return ShellCmd
	}
	// POSIX systems: prefer bash, fallback to sh
	if _, err := exec.LookPath(ShellBash); err == nil {
		return ShellBash
	}
	return ShellSh
}

// ResolveHookConfig resolves the final hook configuration, applying platform-specific overrides.
func ResolveHookConfig(hook *Hook) *HookConfig {
	if hook == nil {
		return nil
	}

	// Start with base configuration
	config := &HookConfig{
		Run:             hook.Run,
		Shell:           hook.Shell,
		ContinueOnError: hook.ContinueOnError,
		Interactive:     hook.Interactive,
	}

	// Apply platform-specific overrides
	var platformOverride *PlatformHook
	if runtime.GOOS == "windows" {
		platformOverride = hook.Windows
	} else {
		platformOverride = hook.Posix
	}

	if platformOverride != nil {
		if platformOverride.Run != "" {
			config.Run = platformOverride.Run
		}
		if platformOverride.Shell != "" {
			config.Shell = platformOverride.Shell
		}
		if platformOverride.ContinueOnError != nil {
			config.ContinueOnError = *platformOverride.ContinueOnError
		}
		if platformOverride.Interactive != nil {
			config.Interactive = *platformOverride.Interactive
		}
	}

	return config
}

// Hook represents a lifecycle hook configuration.
// Note: This type is duplicated from service.Hook to avoid circular imports.
// The service package imports executor, so executor cannot import service.
type Hook struct {
	Run             string
	Shell           string
	ContinueOnError bool
	Interactive     bool
	Windows         *PlatformHook
	Posix           *PlatformHook
}

// PlatformHook represents platform-specific hook configuration.
// Note: This type is duplicated from service.PlatformHook to avoid circular imports.
type PlatformHook struct {
	Run             string
	Shell           string
	ContinueOnError *bool
	Interactive     *bool
}
