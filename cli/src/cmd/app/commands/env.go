package commands

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/cliout"

	"github.com/spf13/cobra"
)

const (
	envFormatDotenv        = "dotenv"
	envFormatShell         = "shell"
	envFormatPowerShell    = "powershell"
	envFormatJSON          = "json"
	envFormatGitHubActions = "github-actions"
)

var (
	envFormat   string
	envNoMask   bool
	envFile     string
	envAll      bool
	envExplain  bool
	envDiff     bool
	envWrite    bool
	envOut      string
	envKeys     bool
	envPrefixes []string
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

Pass --all to print the resolved environment for every service in one run. The
dotenv, shell, and powershell formats group each service under a "# <service>"
header; the json format emits an object keyed by service name; the github-actions
format emits plain KEY=value and heredoc lines with no headers so the output
stays valid for $GITHUB_ENV.

Examples:
  # Resolved environment for the api service (KEY=value lines)
  azd app env api

  # Shell export statements
  azd app env api --format shell

  # PowerShell $env: assignments (load with: azd app env api --format powershell | iex)
  azd app env api --format powershell

  # JSON object (also selected by the global --json flag)
  azd app env api --format json

  # GitHub Actions env lines for $GITHUB_ENV (multiline values use heredocs)
  azd app env api --format github-actions --no-mask >> $GITHUB_ENV

  # Raw values, no masking
  azd app env api --no-mask

  # Explain where each effective value came from
  azd app env api --explain

  # Compare the resolved environment of two services
  azd app env --diff api web

  # List variable names without values
  azd app env api --keys

  # Only print Azure variables for one service
  azd app env api --prefix AZURE_

  # Write the resolved environment to api/.env
  azd app env api --write

  # Write one .env file per service into the build/env folder
  azd app env --all --write --out build/env

  # Resolved environment for every service
  azd app env --all`,
		SilenceUsage:      true,
		Args:              cobra.MaximumNArgs(2),
		RunE:              runEnv,
		ValidArgsFunction: completeServiceArgs,
	}

	cmd.Flags().StringVar(&envFormat, "format", envFormatDotenv, "Output format: dotenv, shell, powershell, json, or github-actions")
	cmd.Flags().BoolVar(&envNoMask, "no-mask", false, "Print raw values instead of masking secret-shaped values")
	cmd.Flags().StringVar(&envFile, "env-file", "", "Path to a .env file to merge, matching azd app run")
	cmd.Flags().BoolVar(&envAll, "all", false, "Print the resolved environment for every service")
	cmd.Flags().BoolVar(&envExplain, "explain", false, "Show the source of each effective value and any sources it overrode")
	cmd.Flags().BoolVar(&envDiff, "diff", false, "Compare the resolved environment of two services (pass two service names)")
	cmd.Flags().BoolVar(&envWrite, "write", false, "Write the resolved environment to a .env file instead of printing it")
	cmd.Flags().StringVar(&envOut, "out", "", "Destination folder for --write files (writes <service>.env); defaults to each service directory")
	cmd.Flags().BoolVar(&envKeys, "keys", false, "Print variable names only")
	cmd.Flags().StringSliceVar(&envPrefixes, "prefix", nil, "Only include variables whose names start with the given prefix (repeatable)")

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
	prefixes, err := normalizeEnvPrefixes(envPrefixes)
	if err != nil {
		return err
	}

	names := make([]string, 0, len(azureYaml.Services))
	for name := range azureYaml.Services {
		names = append(names, name)
	}
	sort.Strings(names)

	// --diff compares two services and takes exactly two service names.
	if envDiff {
		if envKeys {
			return fmt.Errorf("cannot combine --keys with --diff")
		}
		if len(prefixes) > 0 {
			return fmt.Errorf("cannot combine --prefix with --diff")
		}
		return runEnvDiff(azureYaml, names, args)
	}

	// Two arguments are only valid with --diff.
	if len(args) > 1 {
		return fmt.Errorf("expected at most one service name; use --diff to compare two services")
	}

	// --all cannot be combined with a specific service name.
	if envAll && len(args) > 0 {
		return fmt.Errorf("cannot combine --all with a service name")
	}

	// --out only makes sense together with --write.
	if envOut != "" && !envWrite {
		return fmt.Errorf("--out requires --write")
	}

	if envKeys {
		if envExplain {
			return fmt.Errorf("cannot combine --keys with --explain")
		}
		if envWrite {
			return fmt.Errorf("cannot combine --keys with --write")
		}
		format, err := currentEnvFormat()
		if err != nil {
			return err
		}
		if format == envFormatGitHubActions {
			return fmt.Errorf("cannot combine --keys with --format github-actions because that format requires KEY=VALUE pairs")
		}
		if envAll {
			return runEnvAllKeys(azureYaml, names, prefixes)
		}
		if len(args) == 0 {
			return fmt.Errorf("specify a service name or --all with --keys")
		}
	}

	if envWrite {
		if !envAll && len(args) == 0 {
			return fmt.Errorf("specify a service name or --all with --write")
		}
		return runEnvWrite(azureYaml, names, args, prefixes)
	}

	if envAll {
		return runEnvAll(azureYaml, names, prefixes)
	}

	// No service name: list the available services and exit.
	if len(args) == 0 {
		return printServiceList(names)
	}

	serviceName := args[0]
	if _, ok := azureYaml.Services[serviceName]; !ok {
		return serviceNotFoundError(serviceName, names)
	}

	format, err := currentEnvFormat()
	if err != nil {
		return err
	}

	svc := azureYaml.Services[serviceName]
	mask := !envNoMask

	if envExplain {
		return runEnvExplain(serviceName, svc, mask, prefixes)
	}

	resolved, err := service.ResolveEnvironment(context.Background(), svc, getAzureEnvironmentValues(), envFile, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve environment for %q: %w", serviceName, err)
	}
	resolved = filterEnvByPrefixes(resolved, prefixes)

	if envKeys {
		return renderEnvKeys(extractEnvKeys(resolved), format, prefixes)
	}

	if format == envFormatJSON {
		return cliout.PrintJSON(maskEnv(resolved, mask))
	}
	if len(resolved) == 0 && len(prefixes) > 0 {
		fmt.Fprintf(os.Stderr, "No environment variables match prefix %s\n", formatEnvPrefixes(prefixes))
		return nil
	}

	if format == envFormatGitHubActions {
		out, err := formatGitHubEnv(resolved, mask)
		if err != nil {
			return err
		}
		fmt.Print(out)
		return nil
	}

	fmt.Print(formatEnv(resolved, format, mask))
	return nil
}

// runEnvAll resolves and prints the environment for every service. Every service
// is resolved before any output is written so a resolution failure is reported
// without emitting a partial dump.
func runEnvAll(azureYaml *service.AzureYaml, names []string, prefixes []string) error {
	format, err := currentEnvFormat()
	if err != nil {
		return err
	}

	resolvedByService := make(map[string]map[string]string, len(names))
	for _, name := range names {
		svc := azureYaml.Services[name]
		resolved, err := service.ResolveEnvironment(context.Background(), svc, getAzureEnvironmentValues(), envFile, nil, nil)
		if err != nil {
			return fmt.Errorf("failed to resolve environment for %q: %w", name, err)
		}
		resolved = filterEnvByPrefixes(resolved, prefixes)
		resolvedByService[name] = resolved
	}

	if format != envFormatJSON && len(prefixes) > 0 && len(names) > 0 && allEnvMapsEmpty(resolvedByService, names) {
		fmt.Fprintf(os.Stderr, "No environment variables match prefix %s\n", formatEnvPrefixes(prefixes))
		return nil
	}

	return renderAllEnv(resolvedByService, names, format, !envNoMask)
}

func runEnvAllKeys(azureYaml *service.AzureYaml, names []string, prefixes []string) error {
	format, err := currentEnvFormat()
	if err != nil {
		return err
	}

	keysByService := make(map[string][]string, len(names))
	for _, name := range names {
		resolved, err := service.ResolveEnvironment(context.Background(), azureYaml.Services[name], getAzureEnvironmentValues(), envFile, nil, nil)
		if err != nil {
			return fmt.Errorf("failed to resolve environment for %q: %w", name, err)
		}
		keysByService[name] = extractEnvKeys(filterEnvByPrefixes(resolved, prefixes))
	}

	if format != envFormatJSON && len(prefixes) > 0 && len(names) > 0 && allKeySlicesEmpty(keysByService, names) {
		fmt.Fprintf(os.Stderr, "No environment variables match prefix %s\n", formatEnvPrefixes(prefixes))
		return nil
	}

	return renderAllEnvKeys(keysByService, names, format)
}

func allEnvMapsEmpty(envByService map[string]map[string]string, names []string) bool {
	for _, name := range names {
		if len(envByService[name]) > 0 {
			return false
		}
	}
	return true
}

func allKeySlicesEmpty(keysByService map[string][]string, names []string) bool {
	for _, name := range names {
		if len(keysByService[name]) > 0 {
			return false
		}
	}
	return true
}

func normalizeEnvPrefixes(prefixes []string) ([]string, error) {
	normalized := make([]string, 0, len(prefixes))
	for _, prefix := range prefixes {
		prefix = strings.TrimSpace(prefix)
		if prefix == "" {
			return nil, fmt.Errorf("--prefix values cannot be empty")
		}
		normalized = append(normalized, prefix)
	}
	return normalized, nil
}

func filterEnvByPrefixes(env map[string]string, prefixes []string) map[string]string {
	if len(prefixes) == 0 {
		return env
	}
	filtered := make(map[string]string)
	for key, value := range env {
		if envKeyHasPrefix(key, prefixes) {
			filtered[key] = value
		}
	}
	return filtered
}

func envKeyHasPrefix(key string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func formatEnvPrefixes(prefixes []string) string {
	if len(prefixes) == 1 {
		return fmt.Sprintf("%q", prefixes[0])
	}
	quoted := make([]string, 0, len(prefixes))
	for _, prefix := range prefixes {
		quoted = append(quoted, fmt.Sprintf("%q", prefix))
	}
	return strings.Join(quoted, ", ")
}

func extractEnvKeys(env map[string]string) []string {
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func renderEnvKeys(keys []string, format string, prefixes []string) error {
	if format == envFormatJSON {
		return cliout.PrintJSON(keys)
	}
	if len(keys) == 0 && len(prefixes) > 0 {
		fmt.Fprintf(os.Stderr, "No environment variables match prefix %s\n", formatEnvPrefixes(prefixes))
		return nil
	}
	for _, key := range keys {
		fmt.Println(key)
	}
	return nil
}

func renderAllEnvKeys(keysByService map[string][]string, names []string, format string) error {
	if format == envFormatJSON {
		return cliout.PrintJSON(keysByService)
	}
	if len(names) == 0 {
		fmt.Println("No services are defined")
		return nil
	}
	for i, name := range names {
		if i > 0 {
			fmt.Println()
		}
		fmt.Printf("# %s\n", name)
		for _, key := range keysByService[name] {
			fmt.Println(key)
		}
	}
	return nil
}

// renderAllEnv writes the resolved environment for every service. The json format
// emits an object keyed by service name; the dotenv, shell, and powershell formats
// group each service under a "# <service>" header separated by a blank line. names
// controls the ordering. Secret-shaped values are masked when mask is true.
func renderAllEnv(resolvedByService map[string]map[string]string, names []string, format string, mask bool) error {
	if format == envFormatJSON {
		out := make(map[string]map[string]string, len(resolvedByService))
		for name, resolved := range resolvedByService {
			out[name] = maskEnv(resolved, mask)
		}
		return cliout.PrintJSON(out)
	}

	if len(names) == 0 {
		cliout.Info("No services are defined in azure.yaml")
		return nil
	}

	// GitHub Actions parses $GITHUB_ENV line by line and rejects lines that are
	// not KEY=VALUE or heredoc blocks, so the "# <service>" headers used by the
	// other formats are omitted here.
	if format == envFormatGitHubActions {
		for _, name := range names {
			block, err := formatGitHubEnv(resolvedByService[name], mask)
			if err != nil {
				return err
			}
			fmt.Print(block)
		}
		return nil
	}

	for i, name := range names {
		if i > 0 {
			fmt.Println()
		}
		fmt.Printf("# %s\n", name)
		fmt.Print(formatEnv(resolvedByService[name], format, mask))
	}
	return nil
}

// runEnvWrite resolves the environment for the selected services and writes each
// one to a .env file. Every service is resolved before any file is written so a
// resolution or path error is reported without leaving a partial set of files.
// The default destination is each service's own directory (<service>/.env); when
// --out is set the files go to <out>/<service>.env instead.
func runEnvWrite(azureYaml *service.AzureYaml, names []string, args []string, prefixes []string) error {
	format, err := resolveEnvFormat(envFormat)
	if err != nil {
		return err
	}
	mask := !envNoMask

	var selected []string
	if envAll {
		selected = names
	} else {
		serviceName := args[0]
		if _, ok := azureYaml.Services[serviceName]; !ok {
			return serviceNotFoundError(serviceName, names)
		}
		selected = []string{serviceName}
	}

	if len(selected) == 0 {
		cliout.Info("No services are defined in azure.yaml")
		return nil
	}

	type envTarget struct {
		name    string
		path    string
		content string
	}

	targets := make([]envTarget, 0, len(selected))
	for _, name := range selected {
		svc := azureYaml.Services[name]

		path, perr := envWriteTargetPath(svc, name, envOut)
		if perr != nil {
			return perr
		}

		resolved, rerr := service.ResolveEnvironment(context.Background(), svc, getAzureEnvironmentValues(), envFile, nil, nil)
		if rerr != nil {
			return fmt.Errorf("failed to resolve environment for %q: %w", name, rerr)
		}
		resolved = filterEnvByPrefixes(resolved, prefixes)

		content, cerr := envFileContent(resolved, format, mask)
		if cerr != nil {
			return cerr
		}

		targets = append(targets, envTarget{name: name, path: path, content: content})
	}

	for _, t := range targets {
		if err := writeEnvFile(t.path, t.content); err != nil {
			return err
		}
		cliout.Info("Wrote %s", t.path)
	}
	return nil
}

// envWriteTargetPath returns the file path for a service's --write output. With
// an explicit outDir the file is <outDir>/<service>.env; otherwise it is
// <service-project-dir>/.env. A service without a project directory has no
// natural destination, so the caller must supply --out.
func envWriteTargetPath(svc service.Service, name, outDir string) (string, error) {
	if outDir != "" {
		return filepath.Join(outDir, name+".env"), nil
	}
	if svc.Project == "" {
		return "", fmt.Errorf("service %q has no project directory; use --out to choose a destination", name)
	}
	return filepath.Join(svc.Project, ".env"), nil
}

// envFileContent renders the resolved environment for a --write file. The json
// format writes an indented object; dotenv and shell reuse the same rendering as
// the printed output. Secret-shaped values are masked when mask is true.
func envFileContent(resolved map[string]string, format string, mask bool) (string, error) {
	if format == envFormatJSON {
		b, err := json.MarshalIndent(maskEnv(resolved, mask), "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to encode environment as json: %w", err)
		}
		return string(b) + "\n", nil
	}
	if format == envFormatGitHubActions {
		return formatGitHubEnv(resolved, mask)
	}
	return formatEnv(resolved, format, mask), nil
}

// writeEnvFile creates the parent directory if needed and writes the file with
// owner-only permissions, since the content can include secret values.
func writeEnvFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("failed to write %q: %w", path, err)
	}
	return nil
}

// envExplainEntry is the per-variable JSON shape for "env --explain": the
// effective value plus the source that supplied it and any sources it overrode.
type envExplainEntry struct {
	Value     string              `json:"value"`
	Source    service.EnvSource   `json:"source"`
	Overrides []service.EnvSource `json:"overrides,omitempty"`
}

// runEnvExplain prints each effective variable with the source that won and,
// when a higher-priority source replaced a lower one, the sources it overrode.
func runEnvExplain(serviceName string, svc service.Service, mask bool, prefixes []string) error {
	resolved, prov, err := service.ResolveEnvironmentWithSources(context.Background(), svc, getAzureEnvironmentValues(), envFile, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve environment for %q: %w", serviceName, err)
	}
	resolved = filterEnvByPrefixes(resolved, prefixes)

	masked := maskEnv(resolved, mask)
	keys := make([]string, 0, len(masked))
	for k := range masked {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if cliout.IsJSON() {
		out := make(map[string]envExplainEntry, len(masked))
		for _, k := range keys {
			p := prov[k]
			out[k] = envExplainEntry{Value: masked[k], Source: p.Source, Overrides: p.Overrides}
		}
		return cliout.PrintJSON(out)
	}

	for _, k := range keys {
		p := prov[k]
		fmt.Printf("%s=%s\n", k, masked[k])
		if len(p.Overrides) > 0 {
			fmt.Printf("    source: %s (overrode: %s)\n", p.Source, joinSourcesHighestFirst(p.Overrides))
		} else {
			fmt.Printf("    source: %s\n", p.Source)
		}
	}
	if len(keys) == 0 && len(prefixes) > 0 {
		fmt.Fprintf(os.Stderr, "No environment variables match prefix %s\n", formatEnvPrefixes(prefixes))
	}
	return nil
}

// joinSourcesHighestFirst renders overridden sources from highest to lowest
// precedence. Overrides are recorded lowest-first, so this reverses them.
func joinSourcesHighestFirst(sources []service.EnvSource) string {
	parts := make([]string, 0, len(sources))
	for i := len(sources) - 1; i >= 0; i-- {
		parts = append(parts, string(sources[i]))
	}
	return strings.Join(parts, ", ")
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

// currentEnvFormat resolves the requested format and applies global output flags.
func currentEnvFormat() (string, error) {
	format, err := resolveEnvFormat(envFormat)
	if err != nil {
		return "", err
	}
	if cliout.IsJSON() {
		return envFormatJSON, nil
	}
	return format, nil
}

// resolveEnvFormat normalizes and validates the requested output format.
func resolveEnvFormat(format string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", envFormatDotenv:
		return envFormatDotenv, nil
	case envFormatShell:
		return envFormatShell, nil
	case envFormatPowerShell:
		return envFormatPowerShell, nil
	case envFormatJSON:
		return envFormatJSON, nil
	case envFormatGitHubActions:
		return envFormatGitHubActions, nil
	default:
		return "", fmt.Errorf("invalid --format %q: expected dotenv, shell, powershell, json, or github-actions", format)
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

// formatEnv renders the environment as sorted dotenv, shell, or powershell lines.
// Secret masking is applied first when mask is true.
func formatEnv(env map[string]string, format string, mask bool) string {
	masked := maskEnv(env, mask)

	keys := make([]string, 0, len(masked))
	for k := range masked {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		switch format {
		case envFormatShell:
			fmt.Fprintf(&b, "export %s=%s\n", k, shellQuoteDouble(masked[k]))
		case envFormatPowerShell:
			fmt.Fprintf(&b, "$env:%s = %s\n", k, powerShellQuoteSingle(masked[k]))
		default:
			fmt.Fprintf(&b, "%s=%s\n", k, masked[k])
		}
	}
	return b.String()
}

// powerShellQuoteSingle wraps a value in single quotes for a PowerShell string
// literal, escaping embedded single quotes by doubling them. Single-quoted
// PowerShell strings are literal, so no other characters need escaping and the
// result is safe to load with Invoke-Expression.
func powerShellQuoteSingle(v string) string {
	return "'" + strings.ReplaceAll(v, "'", "''") + "'"
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

// formatGitHubEnv renders the environment for appending to the GitHub Actions
// $GITHUB_ENV file. Single-line values use KEY=value; values that contain CR or
// LF use the KEY<<DELIM heredoc form with a random delimiter that does not
// appear in the value, matching how GitHub Actions expects multiline values.
// Keys are emitted in sorted order and secret-shaped values are masked when mask
// is true.
func formatGitHubEnv(env map[string]string, mask bool) (string, error) {
	masked := maskEnv(env, mask)

	keys := make([]string, 0, len(masked))
	for k := range masked {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		if k == "" || strings.ContainsAny(k, "\r\n=") || strings.Contains(k, "<<") {
			return "", fmt.Errorf("invalid GitHub Actions environment variable name %q", k)
		}

		v := masked[k]
		if strings.ContainsAny(v, "\r\n") {
			delim, err := githubEnvDelimiter(v)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&b, "%s<<%s\n%s\n%s\n", k, delim, v, delim)
			continue
		}
		fmt.Fprintf(&b, "%s=%s\n", k, v)
	}
	return b.String(), nil
}

// githubEnvDelimiter returns a random heredoc delimiter that does not appear in
// value, so a multiline value can be written to $GITHUB_ENV without the closing
// marker colliding with the content. The random suffix uses crypto/rand so the
// delimiter is not predictable from the value.
func githubEnvDelimiter(value string) (string, error) {
	buf := make([]byte, 16)
	for attempt := 0; attempt < 8; attempt++ {
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("failed to generate heredoc delimiter: %w", err)
		}
		delim := "ghadelimiter_" + hex.EncodeToString(buf)
		if !strings.Contains(value, delim) {
			return delim, nil
		}
	}
	return "", fmt.Errorf("failed to generate a unique heredoc delimiter")
}
