package detector

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/types"
)

// FindDotnetProjects searches for .csproj and .sln files.
// Only searches within rootDir and does not traverse outside it.
func FindDotnetProjects(rootDir string) ([]types.DotnetProject, error) {
	var dotnetProjects []types.DotnetProject
	seen := make(map[string]bool)

	// Clean the root directory path
	rootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return dotnetProjects, err
	}

	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Ensure we don't traverse outside rootDir
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil
		}
		relPath, err := filepath.Rel(rootDir, absPath)
		if err != nil || strings.HasPrefix(relPath, "..") {
			return filepath.SkipDir
		}

		if info.IsDir() {
			name := info.Name()
			if name == skipDirNodeModules || name == skipDirGit || name == skipDirBin || name == skipDirObj {
				return filepath.SkipDir
			}
		}

		if !info.IsDir() {
			ext := filepath.Ext(info.Name())
			if ext == ".csproj" || ext == ".sln" {
				// For .csproj, use the directory; for .sln, use the file itself
				if ext == ".sln" {
					if !seen[path] {
						dotnetProjects = append(dotnetProjects, types.DotnetProject{
							Path: path,
						})
						seen[path] = true
					}
				} else {
					dir := filepath.Dir(path)
					if !seen[dir] {
						dotnetProjects = append(dotnetProjects, types.DotnetProject{
							Path: path,
						})
						seen[dir] = true
					}
				}
			}
		}

		return nil
	})

	return dotnetProjects, err
}

// FindAppHost searches for AppHost.cs recursively.
// Only searches within rootDir and does not traverse outside it.
func FindAppHost(rootDir string) (*types.AspireProject, error) {
	var aspireProject *types.AspireProject

	// Clean the root directory path
	rootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		// Standard error handling: log and skip problematic paths
		if err != nil {
			slog.Debug("skipping path due to error", "path", path, "error", err)
			return nil // Skip errors but continue walking
		}

		// Ensure we don't traverse outside rootDir
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil
		}
		relPath, err := filepath.Rel(rootDir, absPath)
		if err != nil || strings.HasPrefix(relPath, "..") {
			return filepath.SkipDir
		}

		// Skip common directories
		if info.IsDir() {
			name := info.Name()
			if name == skipDirNodeModules || name == skipDirBin || name == skipDirObj || name == skipDirGit {
				return filepath.SkipDir
			}
		}

		if !info.IsDir() && (info.Name() == "AppHost.cs" || info.Name() == "Program.cs") {
			// Check if it's in a project directory (has .csproj)
			dir := filepath.Dir(path)
			matches, err := filepath.Glob(filepath.Join(dir, "*.csproj"))
			if err != nil {
				return nil // Skip on error
			}
			if len(matches) > 0 {
				aspireProject = &types.AspireProject{
					Dir:         dir,
					ProjectFile: matches[0],
				}
				return filepath.SkipAll // Found it, stop walking
			}
		}

		return nil
	})

	return aspireProject, err
}
