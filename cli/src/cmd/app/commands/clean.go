package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jongio/azd-core/cliout"
	"github.com/spf13/cobra"
)

// cleanCategoryBuild marks build output and cache directories, removed by default.
const cleanCategoryBuild = "build"

// cleanCategoryDeps marks dependency directories, removed only with --deps.
const cleanCategoryDeps = "deps"

// cleanArtifactNames is the full allow-list of directory names clean may remove.
// Any candidate whose base name is not in this set is rejected as a safety guard.
var cleanArtifactNames = map[string]bool{
	// Node build output and caches
	"dist": true, "build": true, ".next": true, ".nuxt": true, "out": true,
	"coverage": true, ".turbo": true,
	// Python build output and caches
	"__pycache__": true, ".pytest_cache": true, ".mypy_cache": true,
	".ruff_cache": true, "htmlcov": true, ".tox": true,
	// .NET build output
	"bin": true, "obj": true,
	// Dependency directories
	"node_modules": true, ".venv": true, "venv": true,
}

// nodeBuildArtifacts are top-level build/cache directory names in a Node project.
var nodeBuildArtifacts = []string{"dist", "build", ".next", ".nuxt", "out", "coverage", ".turbo"}

// pythonBuildArtifacts are top-level build/cache directory names in a Python project.
var pythonBuildArtifacts = []string{"__pycache__", ".pytest_cache", ".mypy_cache", ".ruff_cache", "build", "dist", "htmlcov", ".tox"}

// dotnetBuildArtifacts are top-level build directory names in a .NET project.
var dotnetBuildArtifacts = []string{"bin", "obj"}

// cleanOptions holds the options for the clean command.
type cleanOptions struct {
	deps      bool
	dryRun    bool
	olderThan time.Duration
	services  []string
	writer    io.Writer
}

// cleanTarget is a single directory clean will remove (or would remove in dry-run).
type cleanTarget struct {
	Service   string `json:"service"`
	Path      string `json:"path"`
	Category  string `json:"category"`
	SizeBytes int64  `json:"sizeBytes"`
}

// cleanResult is the machine-readable summary of a clean run.
type cleanResult struct {
	DryRun     bool          `json:"dryRun"`
	Targets    []cleanTarget `json:"targets"`
	TotalBytes int64         `json:"totalBytes"`
}

// NewCleanCommand creates the clean command.
func NewCleanCommand() *cobra.Command {
	opts := &cleanOptions{writer: os.Stdout}

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Reclaim disk space from build artifacts and caches",
		Long: `Remove build output and cache directories for the services defined in azure.yaml.

By default clean removes build artifacts and caches (dist, build, bin, obj,
__pycache__, .pytest_cache, and similar). Dependency directories such as
node_modules and .venv are left in place unless you pass --deps.

Only directories inside a detected service directory are ever removed, and only
when their name matches a known artifact. Paths outside the project are never
touched.

Examples:
  # Show what would be removed and how much space it frees
  azd app clean --dry-run

  # Remove build artifacts across all services
  azd app clean

  # Also remove dependency directories
  azd app clean --deps

  # Only remove artifacts untouched for at least 24 hours
  azd app clean --older-than 24h

  # Limit to one service
  azd app clean --service api`,
		SilenceUsage: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if flag := cmd.InheritedFlags().Lookup("output"); flag != nil && flag.Value.String() != "" {
				return cliout.SetFormat(flag.Value.String())
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClean(opts)
		},
	}

	cmd.Flags().BoolVar(&opts.deps, "deps", false, "Also remove dependency directories (node_modules, .venv)")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "List what would be removed and the reclaimable size without deleting")
	cmd.Flags().DurationVar(&opts.olderThan, "older-than", 0, "Only remove artifacts older than this duration (for example, 24h)")
	cmd.Flags().StringSliceVarP(&opts.services, "service", "s", nil, "Limit to specific services (can be specified multiple times)")

	return cmd
}

func runClean(opts *cleanOptions) error {
	if opts == nil {
		opts = &cleanOptions{}
	}
	if opts.writer == nil {
		opts.writer = os.Stdout
	}
	if opts.olderThan < 0 {
		return fmt.Errorf("--older-than must be zero or greater")
	}

	searchRoot, err := getSearchRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}
	workspaceRoot, err := resolveWorkspaceRoot(searchRoot)
	if err != nil {
		return err
	}

	nodeProjects, pythonProjects, dotnetProjects, err := detectProjectsFromAzureYaml(searchRoot)
	if err != nil {
		return err
	}
	projects := DetectedProjects{Node: nodeProjects, Python: pythonProjects, Dotnet: dotnetProjects}
	if len(opts.services) > 0 {
		projects = filterDetectedProjectsByService(projects, opts.services, searchRoot)
	}

	targets := collectCleanTargets(projects, opts.deps)
	if opts.olderThan > 0 {
		targets = filterCleanTargetsByAge(targets, opts.olderThan, time.Now())
	}

	// Compute sizes for reporting.
	var total int64
	for i := range targets {
		targets[i].SizeBytes = dirSize(targets[i].Path)
		total += targets[i].SizeBytes
	}

	result := cleanResult{DryRun: opts.dryRun, Targets: targets, TotalBytes: total}

	if opts.dryRun {
		if cliout.IsJSON() {
			return cliout.PrintJSON(result)
		}
		renderCleanText(opts.writer, result, false)
		return nil
	}

	// Remove each target, guarded by a workspace containment check.
	var freed int64
	var removed []cleanTarget
	var failures []string
	for _, t := range targets {
		if err := safeToRemove(t.Path, workspaceRoot); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", t.Path, err))
			continue
		}
		if err := os.RemoveAll(t.Path); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", t.Path, err))
			continue
		}
		freed += t.SizeBytes
		removed = append(removed, t)
	}

	result.Targets = removed
	result.TotalBytes = freed

	if cliout.IsJSON() {
		if len(failures) > 0 {
			return fmt.Errorf("failed to remove %d path(s): %v", len(failures), failures)
		}
		return cliout.PrintJSON(result)
	}

	renderCleanText(opts.writer, result, true)
	if len(failures) > 0 {
		return fmt.Errorf("failed to remove %d path(s): %v", len(failures), failures)
	}
	return nil
}

// resolveWorkspaceRoot returns the absolute, symlink-resolved project root used
// for containment checks before deleting anything.
func resolveWorkspaceRoot(searchRoot string) (string, error) {
	abs, err := filepath.Abs(searchRoot)
	if err != nil {
		return "", fmt.Errorf("failed to resolve project root: %w", err)
	}
	if resolved, symlinkErr := filepath.EvalSymlinks(abs); symlinkErr == nil {
		abs = resolved
	}
	return abs, nil
}

// collectCleanTargets returns the artifact directories that exist for the detected
// projects. When includeDeps is false, dependency directories are excluded.
func collectCleanTargets(projects DetectedProjects, includeDeps bool) []cleanTarget {
	var targets []cleanTarget
	seen := make(map[string]bool)

	add := func(service, dir, name, category string) {
		path := filepath.Join(dir, name)
		if seen[path] {
			return
		}
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			return
		}
		seen[path] = true
		targets = append(targets, cleanTarget{Service: service, Path: path, Category: category})
	}

	for _, p := range projects.Node {
		for _, name := range nodeBuildArtifacts {
			add(p.Dir, p.Dir, name, cleanCategoryBuild)
		}
		if includeDeps {
			add(p.Dir, p.Dir, "node_modules", cleanCategoryDeps)
		}
	}
	for _, p := range projects.Python {
		for _, name := range pythonBuildArtifacts {
			add(p.Dir, p.Dir, name, cleanCategoryBuild)
		}
		if includeDeps {
			add(p.Dir, p.Dir, ".venv", cleanCategoryDeps)
			add(p.Dir, p.Dir, "venv", cleanCategoryDeps)
		}
	}
	for _, p := range projects.Dotnet {
		dir := filepath.Dir(p.Path)
		for _, name := range dotnetBuildArtifacts {
			add(dir, dir, name, cleanCategoryBuild)
		}
	}

	sort.Slice(targets, func(i, j int) bool { return targets[i].Path < targets[j].Path })
	return targets
}

func filterCleanTargetsByAge(targets []cleanTarget, olderThan time.Duration, now time.Time) []cleanTarget {
	if olderThan <= 0 {
		return targets
	}
	cutoff := now.Add(-olderThan)
	filtered := make([]cleanTarget, 0, len(targets))
	for _, target := range targets {
		info, err := os.Stat(target.Path)
		if err != nil || !info.IsDir() {
			continue
		}
		if info.ModTime().Before(cutoff) || info.ModTime().Equal(cutoff) {
			filtered = append(filtered, target)
		}
	}
	return filtered
}

// safeToRemove verifies a path is a known artifact directory that lives inside the
// workspace root and is not the workspace root itself. This is the last guard
// before deletion.
func safeToRemove(path, workspaceRoot string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("cannot resolve path: %w", err)
	}
	if resolved, symlinkErr := filepath.EvalSymlinks(abs); symlinkErr == nil {
		abs = resolved
	}

	if !cleanArtifactNames[filepath.Base(abs)] {
		return fmt.Errorf("refusing to remove non-artifact directory")
	}
	if abs == workspaceRoot {
		return fmt.Errorf("refusing to remove the project root")
	}
	rel, err := filepath.Rel(workspaceRoot, abs)
	if err != nil || rel == "." || rel == ".." || filepath.IsAbs(rel) || hasParentPrefix(rel) {
		return fmt.Errorf("refusing to remove path outside the project root")
	}
	return nil
}

// hasParentPrefix reports whether a relative path escapes its base (starts with "..").
func hasParentPrefix(rel string) bool {
	return rel == ".." || (len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator))
}

// dirSize returns the total size in bytes of all regular files under path.
// Missing paths and unreadable entries contribute zero rather than failing.
func dirSize(path string) int64 {
	var total int64
	_ = filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // an unreadable entry contributes zero, it does not abort the walk
		}
		if d.IsDir() {
			return nil
		}
		if info, statErr := d.Info(); statErr == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

func renderCleanText(w io.Writer, result cleanResult, removed bool) {
	if len(result.Targets) == 0 {
		_, _ = fmt.Fprintln(w, "Nothing to clean. No build artifacts or caches found.")
		return
	}

	if result.DryRun {
		_, _ = fmt.Fprintln(w, "The following would be removed:")
	} else if removed {
		_, _ = fmt.Fprintln(w, "Removed:")
	}

	for _, t := range result.Targets {
		_, _ = fmt.Fprintf(w, "  %-6s %s (%s)\n", t.Category, t.Path, humanBytes(t.SizeBytes))
	}

	label := "Reclaimable"
	if removed {
		label = "Freed"
	}
	_, _ = fmt.Fprintf(w, "\n%s: %s across %d director%s\n", label, humanBytes(result.TotalBytes), len(result.Targets), plural(len(result.Targets)))
}

func plural(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

// humanBytes formats a byte count as a short human-readable string.
func humanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
