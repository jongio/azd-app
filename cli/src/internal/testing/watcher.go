package testing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FileWatcher monitors files for changes and triggers test re-runs
type FileWatcher struct {
	paths          []string
	ignorePatterns []string
	lastCheck      map[string]time.Time
	pollInterval   time.Duration
}

// NewFileWatcher creates a new file watcher for the given paths
func NewFileWatcher(paths []string) *FileWatcher {
	return &FileWatcher{
		paths:        paths,
		lastCheck:    make(map[string]time.Time),
		pollInterval: 500 * time.Millisecond,
		ignorePatterns: []string{
			"node_modules",
			".git",
			"__pycache__",
			"*.pyc",
			"bin",
			"obj",
			"dist",
			"build",
			"coverage",
			"test-results",
			".DS_Store",
		},
	}
}

// Watch monitors files for changes and calls the callback when changes are detected
func (w *FileWatcher) Watch(ctx context.Context, callback func() error) error {
	// Initial run
	if err := callback(); err != nil {
		fmt.Printf("Initial test run failed: %v\n", err)
	}

	// Initialize file modification times
	if err := w.scanFiles(); err != nil {
		return fmt.Errorf("failed to scan files: %w", err)
	}

	fmt.Println("\n👀 Watching for file changes... (Press Ctrl+C to stop)")

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\n👋 Stopped watching")
			return nil
		case <-ticker.C:
			changed, err := w.checkForChanges()
			if err != nil {
				fmt.Printf("Error checking for changes: %v\n", err)
				continue
			}

			if changed {
				fmt.Println("\n🔄 Changes detected, re-running tests...")
				if err := callback(); err != nil {
					fmt.Printf("Test run failed: %v\n", err)
				}
				fmt.Println("\n👀 Watching for file changes...")
			}
		}
	}
}

// scanFiles initializes the file modification times
func (w *FileWatcher) scanFiles() error {
	for _, path := range w.paths {
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip files we can't access
			}

			// Skip directories and ignored patterns
			if info.IsDir() || w.shouldIgnore(filePath) {
				if info.IsDir() && w.shouldIgnore(filePath) {
					return filepath.SkipDir
				}
				return nil
			}

			// Only track source files and test files
			if w.isRelevantFile(filePath) {
				w.lastCheck[filePath] = info.ModTime()
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}

// checkForChanges checks if any files have been modified
func (w *FileWatcher) checkForChanges() (bool, error) {
	changed := false

	for _, path := range w.paths {
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip files we can't access
			}

			// Skip directories and ignored patterns
			if info.IsDir() || w.shouldIgnore(filePath) {
				if info.IsDir() && w.shouldIgnore(filePath) {
					return filepath.SkipDir
				}
				return nil
			}

			// Only check relevant files
			if !w.isRelevantFile(filePath) {
				return nil
			}

			lastMod, exists := w.lastCheck[filePath]
			if !exists || info.ModTime().After(lastMod) {
				changed = true
				w.lastCheck[filePath] = info.ModTime()
			}

			return nil
		})

		if err != nil {
			return false, err
		}
	}

	return changed, nil
}

// shouldIgnore checks if a path should be ignored
func (w *FileWatcher) shouldIgnore(path string) bool {
	base := filepath.Base(path)
	for _, pattern := range w.ignorePatterns {
		if matched, _ := filepath.Match(pattern, base); matched {
			return true
		}
		// Also check if the path contains the pattern
		if filepath.Base(filepath.Dir(path)) == pattern {
			return true
		}
	}
	return false
}

// isRelevantFile checks if a file is relevant for test watching
func (w *FileWatcher) isRelevantFile(path string) bool {
	ext := filepath.Ext(path)
	relevantExts := map[string]bool{
		".js":   true,
		".jsx":  true,
		".ts":   true,
		".tsx":  true,
		".mjs":  true,
		".cjs":  true,
		".py":   true,
		".cs":   true,
		".go":   true,
		".java": true,
	}

	return relevantExts[ext]
}
