package dashboard

import (
	"path/filepath"
	"runtime"
	"strings"
)

// normalizeProjectPath converts a project directory path to its canonical form.
// Returns both the absolute path and a normalized key for map lookups (case-insensitive on Windows).
func normalizeProjectPath(projectDir string) (string, string) {
	absPath := projectDir
	if v, err := filepath.Abs(projectDir); err == nil {
		absPath = v
	}
	if v, err := filepath.EvalSymlinks(absPath); err == nil {
		absPath = v
	}
	absPath = filepath.Clean(absPath)

	key := absPath
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}

	return absPath, key
}
