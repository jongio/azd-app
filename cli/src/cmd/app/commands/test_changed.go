package commands

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/testing"
)

// changedServiceFilter returns the names of services whose project directory
// contains at least one file changed since base. When requested is non-empty the
// result is the intersection of the affected services and the requested ones, so
// --changed and --service compose. Names are returned sorted.
func changedServiceFilter(services []testing.ServiceInfo, requested []string, base string) ([]string, error) {
	changed, err := gitChangedFiles(base)
	if err != nil {
		return nil, err
	}

	requestedSet := make(map[string]bool, len(requested))
	for _, r := range requested {
		if r != "" {
			requestedSet[r] = true
		}
	}

	affected := make([]string, 0, len(services))
	for _, svc := range services {
		if len(requestedSet) > 0 && !requestedSet[svc.Name] {
			continue
		}

		dir, err := filepath.Abs(svc.Dir)
		if err != nil {
			return nil, fmt.Errorf("resolve directory for service %s: %w", svc.Name, err)
		}
		if resolved, symErr := filepath.EvalSymlinks(dir); symErr == nil {
			dir = resolved
		}

		if anyPathUnder(changed, dir) {
			affected = append(affected, svc.Name)
		}
	}

	sort.Strings(affected)
	return affected, nil
}

// anyPathUnder reports whether any path in paths is dir itself or lives inside dir.
func anyPathUnder(paths []string, dir string) bool {
	prefix := dir + string(filepath.Separator)
	for _, p := range paths {
		if p == dir || strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

// gitChangedFiles returns the absolute paths of files that differ from base,
// including committed-since-base, staged, unstaged, and untracked files.
func gitChangedFiles(base string) ([]string, error) {
	root, err := gitRepoRoot()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	add := func(rel string) {
		rel = strings.TrimSpace(rel)
		if rel == "" {
			return
		}
		abs := filepath.Clean(filepath.Join(root, filepath.FromSlash(rel)))
		seen[abs] = struct{}{}
	}

	// Tracked changes relative to base (committed-since-base, staged, and unstaged).
	tracked, err := runGit(root, "diff", "--name-only", base)
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(tracked, "\n") {
		add(line)
	}

	// Untracked files that git is not yet ignoring.
	untracked, err := runGit(root, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(untracked, "\n") {
		add(line)
	}

	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out, nil
}

// gitRepoRoot returns the absolute, symlink-resolved top level of the git
// repository that contains the current working directory.
func gitRepoRoot() (string, error) {
	out, err := runGit("", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("--changed requires a git repository: %w", err)
	}
	root := strings.TrimSpace(out)
	if resolved, symErr := filepath.EvalSymlinks(root); symErr == nil {
		root = resolved
	}
	return root, nil
}

// runGit runs a git command in dir (or the current directory when dir is empty)
// and returns its stdout. On failure it returns git's stderr as the error text.
func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return stdout.String(), nil
}
