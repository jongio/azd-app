package detector

import (
	"bufio"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// GoProject represents a detected Go project (directory with go.mod).
type GoProject struct {
	Dir    string
	Module string
}

// FindGoProjects searches for Go projects (go.mod files).
// Only searches within rootDir and does not traverse outside it.
func FindGoProjects(rootDir string) ([]GoProject, error) {
	var goProjects []GoProject
	seen := make(map[string]bool)

	// Clean the root directory path
	rootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return goProjects, err
	}

	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Debug("skipping path due to error", "path", path, "error", err)
			return nil
		}

		// Ensure we don't traverse outside rootDir
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil //nolint:nilerr // skip on error
		}
		relPath, err := filepath.Rel(rootDir, absPath)
		if err != nil || strings.HasPrefix(relPath, "..") {
			return filepath.SkipDir
		}

		if info.IsDir() {
			name := info.Name()
			if name == skipDirNodeModules || name == skipDirGit || name == skipDirBin || name == "vendor" {
				return filepath.SkipDir
			}
		}

		if !info.IsDir() && info.Name() == "go.mod" {
			dir := filepath.Dir(path)

			if seen[dir] {
				return nil
			}

			module := extractGoModule(path)
			goProjects = append(goProjects, GoProject{
				Dir:    dir,
				Module: module,
			})
			seen[dir] = true
		}

		return nil
	})

	return goProjects, err
}

// extractGoModule reads the module path from a go.mod file.
func extractGoModule(goModPath string) string {
	// #nosec G304 -- Path comes from filepath.Walk within validated rootDir
	f, err := os.Open(goModPath)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}
