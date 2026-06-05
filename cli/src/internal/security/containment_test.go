package security_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	internalsec "github.com/jongio/azd-app/cli/src/internal/security"
)

// setupProjectRoot creates a temporary directory tree for containment tests:
//
//	<root>/
//	  src/
//	    myapp/        (exists)
//	  frontend/       (exists)
func setupProjectRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, sub := range []string{
		filepath.Join("src", "myapp"),
		"frontend",
	} {
		if err := os.MkdirAll(filepath.Join(root, sub), 0o755); err != nil {
			t.Fatalf("setup: mkdir %s: %v", sub, err)
		}
	}
	return root
}

// TestValidatePathContainment_AcceptedPaths verifies paths that MUST be accepted.
func TestValidatePathContainment_AcceptedPaths(t *testing.T) {
	root := setupProjectRoot(t)

	tests := []struct {
		name    string
		path    string
		baseDir string
	}{
		{
			name:    "path equal to root",
			path:    root,
			baseDir: root,
		},
		{
			name:    "absolute subdirectory src/myapp",
			path:    filepath.Join(root, "src", "myapp"),
			baseDir: root,
		},
		{
			name:    "absolute subdirectory frontend",
			path:    filepath.Join(root, "frontend"),
			baseDir: root,
		},
		{
			// Pre-joined relative-style path (as parser.go does before calling us)
			name:    "pre-resolved dot-slash src/myapp",
			path:    filepath.Clean(filepath.Join(root, "./src/myapp")),
			baseDir: root,
		},
		{
			// Non-existent child path — creation is allowed inside root
			name:    "non-existent child allowed",
			path:    filepath.Join(root, "new-service"),
			baseDir: root,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := internalsec.ValidatePathContainment(tt.path, tt.baseDir)
			if err != nil {
				t.Errorf("ValidatePathContainment(%q, %q) unexpected error: %v", tt.path, tt.baseDir, err)
				return
			}
			if got == "" {
				t.Errorf("ValidatePathContainment(%q, %q) returned empty path on success", tt.path, tt.baseDir)
			}
		})
	}
}

// TestValidatePathContainment_RejectedPaths verifies paths that MUST be rejected (CWE-22).
func TestValidatePathContainment_RejectedPaths(t *testing.T) {
	root := setupProjectRoot(t)
	// sibling is a real directory at the same level as root, outside root.
	sibling := t.TempDir()

	tests := []struct {
		name    string
		path    string // path as it appears AFTER filepath.Clean+filepath.Join in parser.go
		baseDir string
		errMsg  string
	}{
		{
			// Acceptance criterion 4: project: ../../etc/passwd
			// In parser.go this becomes filepath.Clean(filepath.Join(root, "../../etc/passwd"))
			// which resolves two levels above root.
			name:    "traversal ../../etc/passwd resolves outside root",
			path:    filepath.Clean(filepath.Join(root, "..", "..", "etc", "passwd")),
			baseDir: root,
			errMsg:  "escapes project root",
		},
		{
			// Acceptance criterion 5: project: /etc (absolute outside root)
			name:    "absolute /etc outside root",
			path:    filepath.VolumeName(root) + string(filepath.Separator) + "etc",
			baseDir: root,
			errMsg:  "escapes project root",
		},
		{
			name:    "parent directory of root",
			path:    filepath.Dir(root),
			baseDir: root,
			errMsg:  "escapes project root",
		},
		{
			name:    "sibling directory outside root",
			path:    sibling,
			baseDir: root,
			errMsg:  "escapes project root",
		},
		{
			// Single step up — catches ../sibling style
			name:    "one level up from root",
			path:    filepath.Clean(filepath.Join(root, "..")),
			baseDir: root,
			errMsg:  "escapes project root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := internalsec.ValidatePathContainment(tt.path, tt.baseDir)
			if err == nil {
				t.Errorf("ValidatePathContainment(%q, %q) = %q, want error containing %q",
					tt.path, tt.baseDir, got, tt.errMsg)
				return
			}
			if !errors.Is(err, internalsec.ErrPathEscapesRoot) {
				t.Errorf("error %v does not wrap ErrPathEscapesRoot", err)
			}
			if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errMsg)
			}
		})
	}
}

// TestValidatePathContainment_InputValidation verifies empty-input guards.
func TestValidatePathContainment_InputValidation(t *testing.T) {
	root := setupProjectRoot(t)

	tests := []struct {
		name    string
		path    string
		baseDir string
	}{
		{name: "empty path", path: "", baseDir: root},
		{name: "empty baseDir", path: root, baseDir: ""},
		{name: "both empty", path: "", baseDir: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := internalsec.ValidatePathContainment(tt.path, tt.baseDir)
			if err == nil {
				t.Errorf("ValidatePathContainment(%q, %q) expected error, got nil", tt.path, tt.baseDir)
			}
		})
	}
}

// TestValidatePathContainment_ReturnValue verifies the returned canonical path.
func TestValidatePathContainment_ReturnValue(t *testing.T) {
	root := setupProjectRoot(t)
	sub := filepath.Join(root, "src", "myapp")

	got, err := internalsec.ValidatePathContainment(sub, root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The returned path is symlink-resolved (e.g., macOS /var → /private/var),
	// so resolve root the same way before comparing.
	resolvedRoot, evalErr := filepath.EvalSymlinks(root)
	if evalErr != nil {
		resolvedRoot = root
	}
	if !strings.HasPrefix(got, resolvedRoot) {
		t.Errorf("returned path %q does not start with root %q", got, resolvedRoot)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("returned path %q is not absolute", got)
	}
}
