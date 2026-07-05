package commands

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/cliout"

	"github.com/spf13/cobra"
)

const (
	envFormatDotenv = "dotenv"
	envFormatShell  = "shell"
	envFormatJSON   = "json"
)

var (
	envFormat string
	envNoMask bool
	envFile   string
)

// NewEnvCommand creates the env command.
func NewEnvCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env [service]",
		Short: "Print the resolved environment for a service",
		Long: `Print the effective environment a service receives when it runs.

The output merges the process environment, the azd environment values, and the
service-specific variables from azure.yaml, the same way "azd app run" resolves
them. Pass a service name to print its environment, or run without a name to
list the available services.

Secret-shaped values are masked by default. Use --no-mask to print raw values,
for example when piping the output into another command.

Examples:
  # Resolved environment for the api service (KEY=value lines)
  azd app env api

  # Shell export statements
  azd app env api --format shell

  # JSON object (also selected by the global --json flag)
  azd app env api --format json

  # Raw values, no masking
  azd app env api --no-mask`,
		SilenceUsage:      true,
		Args:              cobra.MaximumNArgs(1),
		RunE:              runEnv,
		ValidArgsFunction: completeServiceArgs,
	}

	cmd.Flags().StringVar(&envFormat, "format", envFormatDotenv, "Output format: dotenv, shell, or json")
	cmd.Flags().BoolVar(&envNoMask, "no-mask", false, "Print raw values instead of masking secret-shaped values")
	cmd.Flags().StringVar(&envFile, "env-file", "", "Path to a .env file to merge, matching azd app run")

	return cmd
}

func runEnv(_ *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	azureYaml, err := service.ParseAzureYaml(cwd)
	if err != nil {
		return fmt.Errorf("failed to load azure.yaml: %w", err)
	}

	names := make([]string, 0, len(azureYaml.Services))
	for name := range azureYaml.Services {
		names = append(names, name)
	}
	sort.Strings(names)

	// No service name: list the available services and exit.
	if len(args) == 0 {
		return printServiceList(names)
	}

	serviceName := args[0]
	if _, ok := azureYaml.Services[serviceName]; !ok {
		if len(names) == 0 {
			return fmt.Errorf("service %q not found. No services are defined in azure.yaml", serviceName)
		}
		return fmt.Errorf("service %q not found. Available services: %s",
			serviceName, strings.Join(names, ", "))
	}

	format, err := resolveEnvFormat(envFormat)
	if err != nil {
		return err
	}
	// The global --json flag takes precedence over --format.
	if cliout.IsJSON() {
		format = envFormatJSON
	}

	svc := azureYaml.Services[serviceName]
	resolved, err := service.ResolveEnvironment(context.Background(), svc, getAzureEnvironmentValues(), envFile, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve environment for %q: %w", serviceName, err)
	}

	mask := !envNoMask
	if format == envFormatJSON {
		return cliout.PrintJSON(maskEnv(resolved, mask))
	}

	fmt.Print(formatEnv(resolved, format, mask))
	return nil
}

// printServiceList prints the available service names so the user knows the
// valid targets for "azd app env <service>".
func printServiceList(names []string) error {
	if cliout.IsJSON() {
		return cliout.PrintJSON(map[string][]string{"services": names})
	}
	if len(names) == 0 {
		cliout.Info("No services are defined in azure.yaml")
		return nil
	}
	cliout.Info("Available services (pass one to print its environment):")
	for _, name := range names {
		fmt.Printf("  %s\n", name)
	}
	return nil
}

// resolveEnvFormat normalizes and validates the requested output format.
func resolveEnvFormat(format string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", envFormatDotenv:
		return envFormatDotenv, nil
	case envFormatShell:
		return envFormatShell, nil
	case envFormatJSON:
		return envFormatJSON, nil
	default:
		return "", fmt.Errorf("invalid --format %q: expected dotenv, shell, or json", format)
	}
}

// maskEnv returns a copy of env with secret-shaped values masked when mask is
// true, leaving values untouched otherwise.
func maskEnv(env map[string]string, mask bool) map[string]string {
	out := make(map[string]string, len(env))
	for k, v := range env {
		if mask {
			out[k] = redactSecretValue(k, v)
		} else {
			out[k] = v
		}
	}
	return out
}

// formatEnv renders the environment as sorted dotenv or shell lines. Secret
// masking is applied first when mask is true.
func formatEnv(env map[string]string, format string, mask bool) string {
	masked := maskEnv(env, mask)

	keys := make([]string, 0, len(masked))
	for k := range masked {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		if format == envFormatShell {
			fmt.Fprintf(&b, "export %s=%s\n", k, shellQuoteDouble(masked[k]))
		} else {
			fmt.Fprintf(&b, "%s=%s\n", k, masked[k])
		}
	}
	return b.String()
}

// shellQuoteDouble wraps a value in double quotes, escaping the characters that
// are special inside a double-quoted shell string so the result is safe to eval.
func shellQuoteDouble(v string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		"`", "\\`",
		`$`, `\$`,
	)
	return `"` + replacer.Replace(v) + `"`
}
