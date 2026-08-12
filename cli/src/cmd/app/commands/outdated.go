package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/cliout"
	"github.com/spf13/cobra"
)

// Package-manager identifiers used by the outdated command.
const (
	managerNpm    = "npm"
	managerPnpm   = "pnpm"
	managerYarn   = "yarn"
	managerPip    = "pip"
	managerDotnet = "dotnet"
	managerGo     = "go"
)

// outdatedRunTimeout bounds how long a single package-manager query may run.
const outdatedRunTimeout = 90 * time.Second

// outdatedLookPath and outdatedRunner are indirected so tests can stub the
// environment instead of shelling out to real package managers.
var (
	outdatedLookPath = exec.LookPath
	outdatedRunner   = defaultOutdatedRunner
)

// outdatedPackage describes a single dependency that has a newer version available.
type outdatedPackage struct {
	Name    string `json:"name"`
	Current string `json:"current"`
	Wanted  string `json:"wanted,omitempty"`
	Latest  string `json:"latest"`
}

// serviceOutdated groups the outdated packages found for one service.
type serviceOutdated struct {
	Service    string            `json:"service"`
	Language   string            `json:"language"`
	Manager    string            `json:"manager,omitempty"`
	Packages   []outdatedPackage `json:"packages"`
	Skipped    bool              `json:"skipped,omitempty"`
	SkipReason string            `json:"skipReason,omitempty"`
}

// outdatedResult is the machine-readable summary across all services.
type outdatedResult struct {
	Services      []serviceOutdated `json:"services"`
	TotalOutdated int               `json:"totalOutdated"`
}

// outdatedTarget is a resolved service ready to be queried.
type outdatedTarget struct {
	Service   string
	Dir       string
	Language  string
	Manager   string
	Supported bool
}

// outdatedOptions holds the options for the outdated command.
type outdatedOptions struct {
	services []string
	managers []string
	format   string
	exitCode bool
	ignore   []string
	writer   io.Writer
}

// NewOutdatedCommand creates the outdated command.
func NewOutdatedCommand() *cobra.Command {
	opts := &outdatedOptions{writer: os.Stdout}

	cmd := &cobra.Command{
		Use:   "outdated",
		Short: "Report outdated dependencies across services",
		Long: `Check each service defined in azure.yaml for outdated dependencies and print
one aggregated report.

The package manager is detected per service:
  - Node: npm, pnpm, or yarn (based on the lockfile)
  - Python: pip
  - .NET: dotnet
  - Go: go

A service whose package manager is not installed is skipped with a warning
rather than failing the whole run.

Use --ignore to drop packages you have intentionally pinned so they do not show
up as outdated or trip --exit-code. Names are matched case-insensitively; pass a
comma-separated list or repeat the flag.

Examples:
  # Report outdated dependencies for every service
  azd app outdated

  # Limit to one service
  azd app outdated --service api

  # Limit to selected package managers
  azd app outdated --manager npm,pip

  # Machine-readable output
  azd app outdated --format json

  # Fail (non-zero exit) when anything is outdated, for CI gating
  azd app outdated --exit-code

  # Ignore packages you have pinned on purpose
  azd app outdated --exit-code --ignore react,typescript`,
		SilenceUsage: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if flag := cmd.InheritedFlags().Lookup("output"); flag != nil && flag.Value.String() != "" {
				if err := cliout.SetFormat(flag.Value.String()); err != nil {
					return err
				}
			}
			if strings.EqualFold(opts.format, outputFormatJSON) {
				return cliout.SetFormat(outputFormatJSON)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOutdated(opts)
		},
	}

	cmd.Flags().StringSliceVarP(&opts.services, "service", "s", nil, "Limit to specific services (can be specified multiple times)")
	cmd.Flags().StringSliceVar(&opts.managers, "manager", nil, "Limit to package managers: npm, pnpm, yarn, pip, dotnet, or go (comma-separated)")
	cmd.Flags().StringVar(&opts.format, "format", "", "Output format: text (default) or json")
	cmd.Flags().BoolVar(&opts.exitCode, "exit-code", false, "Return a non-zero exit code when any dependency is outdated")
	cmd.Flags().StringSliceVar(&opts.ignore, "ignore", nil, "Package names to exclude from the report (comma-separated or repeated)")

	return cmd
}

func runOutdated(opts *outdatedOptions) error {
	if opts == nil {
		opts = &outdatedOptions{}
	}
	if opts.writer == nil {
		opts.writer = os.Stdout
	}

	searchRoot, err := getSearchRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	managerFilter, err := normalizeOutdatedManagerFilter(opts.managers)
	if err != nil {
		return err
	}

	targets, err := resolveOutdatedTargets(searchRoot, opts.services)
	if err != nil {
		return err
	}
	targets = filterOutdatedTargetsByManager(targets, managerFilter)

	ignored := newIgnoreSet(opts.ignore)

	result := outdatedResult{}
	for _, t := range targets {
		svc := serviceOutdated{Service: t.Service, Language: t.Language, Manager: t.Manager}

		if !t.Supported {
			svc.Skipped = true
			svc.SkipReason = "no supported package manager detected"
			result.Services = append(result.Services, svc)
			continue
		}

		bin := managerBinary(t.Manager)
		if _, lookErr := outdatedLookPath(bin); lookErr != nil {
			svc.Skipped = true
			svc.SkipReason = fmt.Sprintf("%s not installed", bin)
			result.Services = append(result.Services, svc)
			continue
		}

		pkgs, queryErr := queryOutdated(t.Dir, t.Manager)
		if queryErr != nil {
			svc.Skipped = true
			svc.SkipReason = queryErr.Error()
			result.Services = append(result.Services, svc)
			continue
		}
		pkgs = filterIgnoredPackages(pkgs, ignored)
		svc.Packages = pkgs
		result.TotalOutdated += len(pkgs)
		result.Services = append(result.Services, svc)
	}

	if cliout.IsJSON() {
		if err := cliout.PrintJSON(result); err != nil {
			return err
		}
	} else {
		renderOutdatedText(opts.writer, result)
	}

	if opts.exitCode && result.TotalOutdated > 0 {
		return fmt.Errorf("%d outdated dependenc%s found", result.TotalOutdated, pluralDeps(result.TotalOutdated))
	}
	return nil
}

func normalizeOutdatedManagerFilter(values []string) (map[string]bool, error) {
	if len(values) == 0 {
		return nil, nil
	}

	valid := map[string]bool{
		managerNpm:    true,
		managerPnpm:   true,
		managerYarn:   true,
		managerPip:    true,
		managerDotnet: true,
		managerGo:     true,
	}
	filter := make(map[string]bool)
	for _, value := range values {
		for _, raw := range strings.Split(value, ",") {
			manager := strings.ToLower(strings.TrimSpace(raw))
			if manager == "" {
				continue
			}
			if !valid[manager] {
				return nil, fmt.Errorf("invalid --manager %q: expected npm, pnpm, yarn, pip, dotnet, or go", raw)
			}
			filter[manager] = true
		}
	}
	if len(filter) == 0 {
		return nil, fmt.Errorf("--manager requires at least one package manager")
	}
	return filter, nil
}

func filterOutdatedTargetsByManager(targets []outdatedTarget, managers map[string]bool) []outdatedTarget {
	if len(managers) == 0 {
		return targets
	}
	filtered := make([]outdatedTarget, 0, len(targets))
	for _, target := range targets {
		if managers[target.Manager] {
			filtered = append(filtered, target)
		}
	}
	return filtered
}

// resolveOutdatedTargets parses azure.yaml, optionally filters to the requested
// services, and resolves each service's package manager. Unknown service names
// produce a clear error.
func resolveOutdatedTargets(searchRoot string, requested []string) ([]outdatedTarget, error) {
	azureYaml, err := service.ParseAzureYaml(searchRoot)
	if err != nil {
		return nil, err
	}

	for _, name := range requested {
		if _, ok := azureYaml.Services[name]; !ok {
			return nil, fmt.Errorf("unknown service %q", name)
		}
	}

	services := service.FilterServices(azureYaml, requested)

	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	sort.Strings(names)

	targets := make([]outdatedTarget, 0, len(names))
	for _, name := range names {
		svc := services[name]
		if svc.Project == "" {
			continue
		}
		lang, mgr, ok := resolveManager(svc.Project, svc.Language)
		targets = append(targets, outdatedTarget{
			Service:   name,
			Dir:       svc.Project,
			Language:  lang,
			Manager:   mgr,
			Supported: ok,
		})
	}
	return targets, nil
}

// newIgnoreSet builds a lookup of package names to exclude from the report. Each
// entry is trimmed and lowercased so matching is case-insensitive, and empty
// entries (from a trailing comma, say) are dropped.
func newIgnoreSet(entries []string) map[string]struct{} {
	if len(entries) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		name := strings.ToLower(strings.TrimSpace(entry))
		if name == "" {
			continue
		}
		set[name] = struct{}{}
	}
	return set
}

// filterIgnoredPackages returns the packages whose names are not in the ignore
// set, preserving order. It returns the input unchanged when nothing is ignored.
func filterIgnoredPackages(pkgs []outdatedPackage, ignored map[string]struct{}) []outdatedPackage {
	if len(ignored) == 0 || len(pkgs) == 0 {
		return pkgs
	}
	kept := make([]outdatedPackage, 0, len(pkgs))
	for _, pkg := range pkgs {
		if _, skip := ignored[strings.ToLower(strings.TrimSpace(pkg.Name))]; skip {
			continue
		}
		kept = append(kept, pkg)
	}
	return kept
}

// resolveManager determines the display language and package manager for a
// service directory, using the declared language when present and falling back
// to marker files on disk.
func resolveManager(dir, declaredLang string) (language, manager string, ok bool) {
	lang := normalizeOutdatedLanguage(declaredLang)
	if lang == "" {
		lang = inferOutdatedLanguage(dir)
	}

	switch lang {
	case "node":
		return "Node", detectNodeManager(dir), true
	case "python":
		return "Python", managerPip, true
	case "dotnet":
		return ".NET", managerDotnet, true
	case "go":
		return "Go", managerGo, true
	default:
		if declaredLang != "" {
			return declaredLang, "", false
		}
		return "unknown", "", false
	}
}

// normalizeOutdatedLanguage maps a declared azure.yaml language to one of the
// package-manager language buckets, or "" when it is not one we support.
func normalizeOutdatedLanguage(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "js", "javascript", "node", "nodejs", "node.js", "ts", "typescript":
		return "node"
	case "py", "python":
		return "python"
	case "cs", "csharp", "c#", "dotnet", ".net":
		return "dotnet"
	case "go", "golang":
		return "go"
	default:
		return ""
	}
}

// inferOutdatedLanguage guesses the language from marker files in a directory.
func inferOutdatedLanguage(dir string) string {
	if fileExists(dir, "package.json") {
		return "node"
	}
	if fileExists(dir, "go.mod") {
		return "go"
	}
	for _, marker := range []string{"requirements.txt", "pyproject.toml", "setup.py", "Pipfile"} {
		if fileExists(dir, marker) {
			return "python"
		}
	}
	if hasFileWithExt(dir, ".csproj") || hasFileWithExt(dir, ".sln") || hasFileWithExt(dir, ".fsproj") {
		return "dotnet"
	}
	return ""
}

// detectNodeManager picks the Node package manager based on the lockfile present.
func detectNodeManager(dir string) string {
	if fileExists(dir, "pnpm-lock.yaml") {
		return managerPnpm
	}
	if fileExists(dir, "yarn.lock") {
		return managerYarn
	}
	return managerNpm
}

func hasFileWithExt(dir, ext string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ext) {
			return true
		}
	}
	return false
}

// managerBinary returns the executable name for a package manager.
func managerBinary(manager string) string {
	return manager
}

// queryOutdated runs the package manager's outdated query and parses the output.
func queryOutdated(dir, manager string) ([]outdatedPackage, error) {
	args := outdatedArgs(manager)
	out, err := outdatedRunner(dir, managerBinary(manager), args)
	if len(out) == 0 && err != nil {
		return nil, fmt.Errorf("failed to run %s: %w", manager, err)
	}
	return parseOutdated(manager, out)
}

// outdatedArgs returns the arguments for a manager's outdated query.
func outdatedArgs(manager string) []string {
	switch manager {
	case managerNpm, managerPnpm:
		return []string{"outdated", "--json"}
	case managerYarn:
		return []string{"outdated", "--json"}
	case managerPip:
		return []string{"list", "--outdated", "--format=json"}
	case managerDotnet:
		return []string{"list", "package", "--outdated", "--format", "json"}
	case managerGo:
		return []string{"list", "-u", "-m", "-json", "all"}
	default:
		return nil
	}
}

// parseOutdated dispatches raw manager output to the matching parser.
func parseOutdated(manager string, data []byte) ([]outdatedPackage, error) {
	switch manager {
	case managerNpm, managerPnpm:
		return parseNpmOutdated(data)
	case managerYarn:
		return parseYarnOutdated(data)
	case managerPip:
		return parsePipOutdated(data)
	case managerDotnet:
		return parseDotnetOutdated(data)
	case managerGo:
		return parseGoOutdated(data)
	default:
		return nil, fmt.Errorf("unsupported package manager %q", manager)
	}
}

// parseNpmOutdated parses `npm outdated --json` / `pnpm outdated --json` output,
// which is a JSON object keyed by package name.
func parseNpmOutdated(data []byte) ([]outdatedPackage, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, nil
	}
	var raw map[string]struct {
		Current string `json:"current"`
		Wanted  string `json:"wanted"`
		Latest  string `json:"latest"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse npm output: %w", err)
	}
	pkgs := make([]outdatedPackage, 0, len(raw))
	for name, v := range raw {
		pkgs = append(pkgs, outdatedPackage{Name: name, Current: v.Current, Wanted: v.Wanted, Latest: v.Latest})
	}
	sortPackages(pkgs)
	return pkgs, nil
}

// parseYarnOutdated parses `yarn outdated --json` output, which is newline
// delimited JSON containing a "table" record with rows of package data.
func parseYarnOutdated(data []byte) ([]outdatedPackage, error) {
	var pkgs []outdatedPackage
	for _, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var rec struct {
			Type string `json:"type"`
			Data struct {
				Head []string   `json:"head"`
				Body [][]string `json:"body"`
			} `json:"data"`
		}
		if err := json.Unmarshal(line, &rec); err != nil {
			continue
		}
		if rec.Type != "table" {
			continue
		}
		cur, want, lat := yarnColumnIndexes(rec.Data.Head)
		for _, row := range rec.Data.Body {
			if len(row) == 0 {
				continue
			}
			pkgs = append(pkgs, outdatedPackage{
				Name:    row[0],
				Current: columnValue(row, cur),
				Wanted:  columnValue(row, want),
				Latest:  columnValue(row, lat),
			})
		}
	}
	sortPackages(pkgs)
	return pkgs, nil
}

func yarnColumnIndexes(head []string) (current, wanted, latest int) {
	current, wanted, latest = 1, 2, 3
	for i, h := range head {
		switch strings.ToLower(h) {
		case "current":
			current = i
		case "wanted":
			wanted = i
		case "latest":
			latest = i
		}
	}
	return current, wanted, latest
}

func columnValue(row []string, idx int) string {
	if idx >= 0 && idx < len(row) {
		return row[idx]
	}
	return ""
}

// parsePipOutdated parses `pip list --outdated --format=json` output.
func parsePipOutdated(data []byte) ([]outdatedPackage, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, nil
	}
	var raw []struct {
		Name          string `json:"name"`
		Version       string `json:"version"`
		LatestVersion string `json:"latest_version"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse pip output: %w", err)
	}
	pkgs := make([]outdatedPackage, 0, len(raw))
	for _, v := range raw {
		pkgs = append(pkgs, outdatedPackage{Name: v.Name, Current: v.Version, Latest: v.LatestVersion})
	}
	sortPackages(pkgs)
	return pkgs, nil
}

// parseDotnetOutdated parses `dotnet list package --outdated --format json` output.
func parseDotnetOutdated(data []byte) ([]outdatedPackage, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, nil
	}
	var raw struct {
		Projects []struct {
			Frameworks []struct {
				TopLevelPackages []struct {
					ID               string `json:"id"`
					RequestedVersion string `json:"requestedVersion"`
					ResolvedVersion  string `json:"resolvedVersion"`
					LatestVersion    string `json:"latestVersion"`
				} `json:"topLevelPackages"`
			} `json:"frameworks"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse dotnet output: %w", err)
	}
	seen := make(map[string]bool)
	var pkgs []outdatedPackage
	for _, proj := range raw.Projects {
		for _, fw := range proj.Frameworks {
			for _, p := range fw.TopLevelPackages {
				key := p.ID + "@" + p.ResolvedVersion
				if seen[key] {
					continue
				}
				seen[key] = true
				pkgs = append(pkgs, outdatedPackage{
					Name:    p.ID,
					Current: p.ResolvedVersion,
					Wanted:  p.RequestedVersion,
					Latest:  p.LatestVersion,
				})
			}
		}
	}
	sortPackages(pkgs)
	return pkgs, nil
}

// parseGoOutdated parses the streamed JSON objects from
// `go list -u -m -json all`, keeping only modules with an available update.
func parseGoOutdated(data []byte) ([]outdatedPackage, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	var pkgs []outdatedPackage
	for dec.More() {
		var mod struct {
			Path     string `json:"Path"`
			Version  string `json:"Version"`
			Main     bool   `json:"Main"`
			Indirect bool   `json:"Indirect"`
			Update   *struct {
				Version string `json:"Version"`
			} `json:"Update"`
		}
		if err := dec.Decode(&mod); err != nil {
			return nil, fmt.Errorf("failed to parse go output: %w", err)
		}
		if mod.Main || mod.Update == nil {
			continue
		}
		pkgs = append(pkgs, outdatedPackage{Name: mod.Path, Current: mod.Version, Latest: mod.Update.Version})
	}
	sortPackages(pkgs)
	return pkgs, nil
}

func sortPackages(pkgs []outdatedPackage) {
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].Name < pkgs[j].Name })
}

func renderOutdatedText(w io.Writer, result outdatedResult) {
	if len(result.Services) == 0 {
		_, _ = fmt.Fprintln(w, "No services with dependencies found.")
		return
	}

	for _, svc := range result.Services {
		header := svc.Service
		if svc.Language != "" {
			header = fmt.Sprintf("%s (%s)", svc.Service, svc.Language)
		}
		if svc.Skipped {
			_, _ = fmt.Fprintf(w, "%s: skipped - %s\n", header, svc.SkipReason)
			continue
		}
		if len(svc.Packages) == 0 {
			_, _ = fmt.Fprintf(w, "%s: up to date\n", header)
			continue
		}
		_, _ = fmt.Fprintf(w, "%s: %d outdated\n", header, len(svc.Packages))
		for _, p := range svc.Packages {
			wanted := p.Wanted
			if wanted == "" {
				wanted = "-"
			}
			_, _ = fmt.Fprintf(w, "  %-30s %s -> %s (wanted %s)\n", p.Name, p.Current, p.Latest, wanted)
		}
	}

	_, _ = fmt.Fprintf(w, "\nTotal: %d outdated dependenc%s across %d service%s\n",
		result.TotalOutdated, pluralDeps(result.TotalOutdated), len(result.Services), pluralServices(len(result.Services)))
}

func pluralDeps(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

func pluralServices(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func defaultOutdatedRunner(dir, bin string, args []string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), outdatedRunTimeout)
	defer cancel()

	// #nosec G204 -- bin is one of a fixed set of package-manager names and args
	// are constant per manager; neither is derived from untrusted input.
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	return stdout.Bytes(), err
}
