// Package fileutil provides common file system utilities for detecting project types.
package fileutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/security"
)

// File permissions
const (
	// DirPermission is the default permission for creating directories (rwxr-x---)
	DirPermission = 0750
	// FilePermission is the default permission for creating files (rw-r--r--)
	FilePermission = 0644
)

// AtomicWriteJSON writes data as JSON to a file atomically.
// It writes to a temporary file first, then renames it to the target path.
// This ensures the file is never left in a partial/corrupt state.
func AtomicWriteJSON(path string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Create a unique temp file in the same directory to avoid cross-filesystem
	// rename issues and concurrent writers clobbering the same temp filename.
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, filepath.Base(path)+".tmp.*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	// Ensure file is closed on all paths
	defer func() { _ = tmpFile.Close() }()

	if _, err := tmpFile.Write(jsonData); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Ensure data hits disk before we close/rename. This reduces races
	// where the file might not be fully flushed on platforms with delayed
	// write semantics (observed flakiness on some CI macOS runners).
	if err := tmpFile.Sync(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Set permissive mode on the temp file before rename so the final file
	// has expected permissions once moved into place.
	_ = os.Chmod(tmpPath, FilePermission)

	// Rename temp file to final file (atomic operation on most filesystems).
	// Perform a few retries to mitigate transient rename races observed on CI.
	var renameErr error
	for i := 0; i < 5; i++ {
		renameErr = os.Rename(tmpPath, path)
		if renameErr == nil {
			break
		}
		// Small backoff
		time.Sleep(20 * time.Millisecond)
	}
	if renameErr != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to rename temp file: %w", renameErr)
	}

	return nil
}

// AtomicWriteFile writes raw bytes to a file atomically.
// It writes to a temporary file first, then renames it to the target path.
// This ensures the file is never left in a partial/corrupt state.
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	// Create a unique temp file in the same directory to avoid concurrent
	// writers using the same temp filename and causing rename failures.
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, filepath.Base(path)+".tmp.*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	// Ensure file is closed on all paths
	defer func() { _ = tmpFile.Close() }()

	if _, err := tmpFile.Write(data); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Ensure temp has requested permissions before rename
	_ = os.Chmod(tmpPath, perm)

	// Rename temp file to final file (atomic operation on most filesystems).
	var renameErr2 error
	for i := 0; i < 5; i++ {
		renameErr2 = os.Rename(tmpPath, path)
		if renameErr2 == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if renameErr2 != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to rename temp file: %w", renameErr2)
	}

	// Ensure final permissions are set
	if err := os.Chmod(path, perm); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}

// ReadJSON reads JSON from a file into the target interface.
// Returns nil error if file doesn't exist (target unchanged).
func ReadJSON(path string, target interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, not an error
		}
		return fmt.Errorf("failed to read file: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	return nil
}

// EnsureDir creates a directory if it doesn't exist.
func EnsureDir(path string) error {
	if err := os.MkdirAll(path, DirPermission); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return nil
}

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
