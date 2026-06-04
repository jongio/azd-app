// Package security provides path-containment validation to prevent CWE-22 path traversal.
//
// This package complements the external azd-core/security package, which uses a naive
// strings.Contains(path, "..") check that collapses after filepath.Clean.  The
// ValidatePathContainment function here uses filepath.Rel after fully resolving both
// operands, which is the only correct approach: a cleaned absolute path never contains
// ".." literals even when it references a location outside the intended base.
package security

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrPathEscapesRoot is returned when a path resolves to a location outside
// its declared base directory.
var ErrPathEscapesRoot = errors.New("path escapes project root")

// ValidatePathContainment verifies that path, once fully resolved, resides
// inside baseDir.  It defends against CWE-22 (path traversal) by applying
// filepath.Rel after resolving both operands to their canonical absolute forms,
// rather than by searching for ".." literals in the raw string.
//
// Rules enforced:
//   - Both path and baseDir must be non-empty.
//   - Relative paths are resolved against the current working directory (via
//     filepath.Abs) before the containment check; callers that have already
//     performed filepath.Join(baseDir, relPath)+filepath.Clean should pass the
//     resulting absolute path directly.
//   - Absolute paths supplied by users are accepted only when they reside inside
//     (or are equal to) baseDir.
//   - Symlinks in both path and baseDir are followed so that symlink-based
//     escapes are detected.  Non-existent paths whose parent exists are accepted
//     (creation path); if the path's parent cannot be resolved, the cleaned
//     absolute form is used.
//
// Returns the resolved absolute path on success, or an error wrapping
// ErrPathEscapesRoot / a descriptive cause on failure.
func ValidatePathContainment(path, baseDir string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("%w: path must not be empty", ErrPathEscapesRoot)
	}
	if baseDir == "" {
		return "", fmt.Errorf("%w: base directory must not be empty", ErrPathEscapesRoot)
	}

	// --- Resolve baseDir to canonical absolute form ---
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("cannot resolve base directory %q: %w", baseDir, err)
	}
	absBase = filepath.Clean(absBase)
	if realBase, symlinkErr := filepath.EvalSymlinks(absBase); symlinkErr == nil {
		absBase = realBase
	}
	// Ensure trailing separator is NOT present for consistent prefix comparison
	// via filepath.Rel (Rel handles this correctly regardless, but keep absBase clean).

	// --- Resolve path to canonical absolute form ---
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("cannot resolve path %q: %w", path, err)
	}
	absPath = filepath.Clean(absPath)

	// Follow symlinks when the path already exists so that symlink-based
	// escapes (e.g. a symlink inside baseDir pointing outside it) are caught.
	// For paths that do not yet exist (service project dirs to be created),
	// resolve symlinks on the parent directory only — this handles the macOS
	// /var → /private/var symlink that would otherwise cause a false mismatch.
	if realPath, symlinkErr := filepath.EvalSymlinks(absPath); symlinkErr == nil {
		absPath = realPath
	} else if !os.IsNotExist(symlinkErr) {
		return "", fmt.Errorf("cannot resolve symbolic links for %q: %w", path, symlinkErr)
	} else {
		// Path doesn't exist — resolve parent to normalise platform symlinks.
		parent := filepath.Dir(absPath)
		if realParent, parentErr := filepath.EvalSymlinks(parent); parentErr == nil {
			absPath = filepath.Join(realParent, filepath.Base(absPath))
		}
	}

	// --- Containment check via filepath.Rel ---
	// filepath.Rel(base, path) returns ".." or a string starting with "../"
	// ("..\" on Windows) whenever path is outside base.  This is the
	// authoritative check — it works correctly even when no ".." literals
	// appear in the individual inputs.
	rel, relErr := filepath.Rel(absBase, absPath)
	if relErr != nil {
		// On Windows, Rel returns an error when paths are on different drives.
		return "", fmt.Errorf("%w: %q and base %q are on different volumes",
			ErrPathEscapesRoot, path, baseDir)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %q resolves to %q which is outside project root %q",
			ErrPathEscapesRoot, path, absPath, absBase)
	}

	return absPath, nil
}
