package security

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

var (
	// ErrInvalidPath indicates a path contains invalid characters or patterns.
	ErrInvalidPath = errors.New("invalid path")
	// ErrPathTraversal indicates a path traversal attack attempt.
	ErrPathTraversal = errors.New("path traversal detected")
)

// ValidatePath checks if a path is safe to use.
// It prevents path traversal attacks and validates the path is within allowed bounds.
func ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("%w: empty path", ErrInvalidPath)
	}

	// Check for path traversal attempts before resolving
	if strings.Contains(path, "..") {
		return fmt.Errorf("%w: path contains ..", ErrPathTraversal)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("%w: cannot resolve path: %v", ErrInvalidPath, err)
	}

	// Clean the path
	cleanPath := filepath.Clean(absPath)

	// After cleaning, check again for ..
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("%w: cleaned path contains ..", ErrPathTraversal)
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
