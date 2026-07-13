package commands

import (
	"fmt"
	"os"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/cliout"

	"github.com/spf13/cobra"
)

// NewHooksCommand creates the hooks command.
func NewHooksCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "List the lifecycle hooks configured in azure.yaml",
		Long: `List the project-level lifecycle hooks configured in azure.yaml.

Shows each configured hook (prerun, postrun, prestop, poststop) with the
command it runs, the shell it uses, and any per-platform override for Windows
or POSIX. Hooks run around the development lifecycle:

  prerun    before services start
  postrun   after services stop following a run
  prestop   before services are stopped
  poststop  after services are stopped

Examples:
  # List configured hooks
  azd app hooks

  # JSON object keyed by hook name
  azd app hooks --output json`,
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE:         runHooks,
	}

	return cmd
}

// hookOverride is a per-platform override of a hook.
type hookOverride struct {
	Run             string `json:"run,omitempty"`
	Shell           string `json:"shell,omitempty"`
	ContinueOnError *bool  `json:"continueOnError,omitempty"`
	Interactive     *bool  `json:"interactive,omitempty"`
}

// hookInfo is the resolved view of a single lifecycle hook.
type hookInfo struct {
	Name            string        `json:"name"`
	Run             string        `json:"run"`
	Shell           string        `json:"shell,omitempty"`
	ContinueOnError bool          `json:"continueOnError,omitempty"`
	Interactive     bool          `json:"interactive,omitempty"`
	Windows         *hookOverride `json:"windows,omitempty"`
	Posix           *hookOverride `json:"posix,omitempty"`
}

// lifecycleHookOrder is the stable display order for the four lifecycle hooks.
var lifecycleHookOrder = []string{"prerun", "postrun", "prestop", "poststop"}

func runHooks(_ *cobra.Command, _ []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	azureYaml, err := service.ParseAzureYaml(cwd)
	if err != nil {
		return fmt.Errorf("failed to load azure.yaml: %w", err)
	}

	hooks := collectHooks(azureYaml.Hooks)

	if cliout.IsJSON() {
		return cliout.PrintJSON(hooks)
	}

	printHooks(hooks)
	return nil
}

// collectHooks resolves the configured lifecycle hooks in display order. Hooks
// that are not configured are omitted.
func collectHooks(hooks *service.Hooks) []hookInfo {
	sources := map[string]*service.Hook{
		"prerun":   hooks.GetPrerun(),
		"postrun":  hooks.GetPostrun(),
		"prestop":  hooks.GetPrestop(),
		"poststop": hooks.GetPoststop(),
	}

	result := make([]hookInfo, 0, len(lifecycleHookOrder))
	for _, name := range lifecycleHookOrder {
		h := sources[name]
		if h == nil {
			continue
		}
		result = append(result, hookInfo{
			Name:            name,
			Run:             h.Run,
			Shell:           h.Shell,
			ContinueOnError: h.ContinueOnError,
			Interactive:     h.Interactive,
			Windows:         toHookOverride(h.Windows),
			Posix:           toHookOverride(h.Posix),
		})
	}
	return result
}

// toHookOverride converts a platform-specific hook to the reporting shape,
// returning nil when there is no override.
func toHookOverride(p *service.PlatformHook) *hookOverride {
	if p == nil {
		return nil
	}
	return &hookOverride{
		Run:             p.Run,
		Shell:           p.Shell,
		ContinueOnError: p.ContinueOnError,
		Interactive:     p.Interactive,
	}
}

// printHooks writes each configured hook and its details in text mode.
func printHooks(hooks []hookInfo) {
	if len(hooks) == 0 {
		cliout.Info("No lifecycle hooks are configured in azure.yaml")
		return
	}

	cliout.CommandHeader("hooks", "Configured lifecycle hooks")
	for i, h := range hooks {
		if i > 0 {
			cliout.Newline()
		}
		cliout.Info("%s", h.Name)
		cliout.Label("run", shellOrNote(h.Run))
		cliout.Label("shell", shellOrDefault(h.Shell))
		if h.ContinueOnError {
			cliout.Label("continueOnError", "true")
		}
		if h.Interactive {
			cliout.Label("interactive", "true")
		}
		if h.Windows != nil {
			cliout.Label("windows", overrideSummary(h.Windows))
		}
		if h.Posix != nil {
			cliout.Label("posix", overrideSummary(h.Posix))
		}
	}
}

// shellOrDefault returns the shell name, or a note when the default shell is used.
func shellOrDefault(shell string) string {
	if shell == "" {
		return "(default)"
	}
	return shell
}

// shellOrNote returns the command, or a note when no command is set.
func shellOrNote(run string) string {
	if run == "" {
		return "(none)"
	}
	return run
}

// overrideSummary renders a per-platform override as a short one-line summary.
func overrideSummary(o *hookOverride) string {
	summary := shellOrNote(o.Run)
	if o.Shell != "" {
		summary += fmt.Sprintf(" (shell: %s)", o.Shell)
	}
	return summary
}
