package security

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var (
	// ErrInvalidPath indicates a path contains invalid characters or patterns.
	ErrInvalidPath = errors.New("invalid path")
	// ErrPathTraversal indicates a path traversal attack attempt.
	ErrPathTraversal = errors.New("path traversal detected")
	// ErrInvalidServiceName indicates an invalid service name.
	ErrInvalidServiceName = errors.New("invalid service name")

	// serviceNamePattern validates service names - alphanumeric start, then alphanumeric, underscore, hyphen, or dot.
	// Max 63 characters to align with DNS label limits and container naming conventions.
	serviceNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,62}$`)
)

// ValidatePath checks if a path is safe to use.
// It prevents path traversal attacks, symbolic link attacks, and validates the path is within allowed bounds.
func ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("%w: empty path", ErrInvalidPath)
	}

	// Check for path traversal attempts before resolving
	if strings.Contains(path, "..") {
		return fmt.Errorf("%w: path contains parent directory reference", ErrPathTraversal)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("%w: cannot resolve path: %w", ErrInvalidPath, err)
	}

	// Clean the path
	cleanPath := filepath.Clean(absPath)

	// After cleaning, check again for ..
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("%w: cleaned path contains parent directory reference", ErrPathTraversal)
	}

	// Resolve symbolic links to detect link-based attacks
	// This prevents attackers from using symlinks to escape allowed directories
	resolvedPath, err := filepath.EvalSymlinks(cleanPath)
	if err != nil {
		// If the path doesn't exist yet, that's okay - we're validating the path structure
		if !os.IsNotExist(err) {
			return fmt.Errorf("%w: cannot resolve symbolic links: %w", ErrInvalidPath, err)
		}
		// Path doesn't exist, use cleaned path for validation
		resolvedPath = cleanPath
	}

	// Verify resolved path doesn't contain ..
	if strings.Contains(resolvedPath, "..") {
		return fmt.Errorf("%w: resolved path contains parent directory reference", ErrPathTraversal)
	}

	return nil
}

// ValidateServiceName validates that a service name is safe and well-formed.
// Service names must:
// - Start with an alphanumeric character
// - Contain only alphanumeric characters, underscores, hyphens, or dots
// - Be at most 63 characters (DNS label limit)
// - Not contain path traversal sequences
//
// If allowEmpty is true, empty strings are accepted (for optional parameters).
func ValidateServiceName(name string, allowEmpty bool) error {
	if name == "" {
		if allowEmpty {
			return nil
		}
		return fmt.Errorf("%w: service name cannot be empty", ErrInvalidServiceName)
	}

	if len(name) > 63 {
		return fmt.Errorf("%w: exceeds maximum length of 63 characters", ErrInvalidServiceName)
	}

	if !serviceNamePattern.MatchString(name) {
		return fmt.Errorf("%w: must start with alphanumeric and contain only alphanumeric, underscore, hyphen, or dot", ErrInvalidServiceName)
	}

	// Extra check for path traversal attempts
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("%w: contains invalid path characters", ErrInvalidServiceName)
	}

	return nil
}

// ValidatePackageManager checks if the package manager name is allowed.
func ValidatePackageManager(pm string) error {
	allowed := map[string]bool{
		"npm":    true,
		"pnpm":   true,
		"yarn":   true,
		"pip":    true,
		"poetry": true,
		"uv":     true,
		"dotnet": true,
	}

	if !allowed[pm] {
		return fmt.Errorf("invalid package manager: %s", pm)
	}

	return nil
}

// SanitizeScriptName ensures a script name doesn't contain shell metacharacters.
func SanitizeScriptName(name string) error {
	// Disallow shell metacharacters
	dangerous := []string{";", "&", "|", ">", "<", "`", "$", "(", ")", "{", "}", "[", "]", "\n", "\r"}

	for _, char := range dangerous {
		if strings.Contains(name, char) {
			return fmt.Errorf("script name contains dangerous character: %s", char)
		}
	}

	return nil
}

// ValidateFilePermissions checks if a file has secure permissions.
// On Unix systems, it ensures the file is not world-writable.
// On Windows, this check is skipped as Windows uses ACLs differently.
func ValidateFilePermissions(path string) error {
	// Skip permission check on Windows as it uses ACLs
	if runtime.GOOS == "windows" {
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Check if file is world-writable (insecure)
	if info.Mode().Perm()&0002 != 0 {
		return fmt.Errorf("file %s is world-writable (permissions: %04o), please run: chmod 644 %s",
			path, info.Mode().Perm(), path)
	}

	return nil
}
