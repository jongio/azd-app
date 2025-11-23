// Package fileutil provides common file system utilities for detecting project types.
package fileutil

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/security"
)

// FileExists checks if a file exists in a directory.
// Returns true if the file exists, false otherwise.
func FileExists(dir string, filename string) bool {
	_, err := os.Stat(filepath.Join(dir, filename))
	return err == nil
}

// HasFileWithExt checks if any file with the given extension exists in the directory.
// ext should include the dot (e.g., ".csproj")
func HasFileWithExt(dir string, ext string) bool {
	pattern := filepath.Join(dir, "*"+ext)
	matches, _ := filepath.Glob(pattern)
	return len(matches) > 0
}

// ContainsText checks if a file contains the specified text.
// Returns false if file doesn't exist, can't be read, or validation fails.
func ContainsText(filePath string, text string) bool {
	// Validate path before reading
	if err := security.ValidatePath(filePath); err != nil {
		return false
	}

	// #nosec G304 -- Path validated by security.ValidatePath
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), text)
}

// FileExistsAny checks if any of the given filenames exist in the directory.
func FileExistsAny(dir string, filenames ...string) bool {
	for _, filename := range filenames {
		if FileExists(dir, filename) {
			return true
		}
	}
	return false
}

// FilesExistAll checks if all of the given filenames exist in the directory.
func FilesExistAll(dir string, filenames ...string) bool {
	for _, filename := range filenames {
		if !FileExists(dir, filename) {
			return false
		}
	}
	return true
}

// ContainsTextInFile checks if file contains text at the specified path.
// Convenience function combining filepath.Join and ContainsText.
func ContainsTextInFile(dir string, filename string, text string) bool {
	return ContainsText(filepath.Join(dir, filename), text)
}

// HasAnyFileWithExts checks if any file with any of the given extensions exists.
func HasAnyFileWithExts(dir string, exts ...string) bool {
	for _, ext := range exts {
		if HasFileWithExt(dir, ext) {
			return true
		}
	}
	return false
}
