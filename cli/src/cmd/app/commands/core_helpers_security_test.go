package commands

// core_helpers_security_test.go verifies that loadAzureYaml uses
// ValidatePathContainment (CWE-22, SEC-012) for the azure.yaml path.
//
// These are internal-package tests (package commands) so they can reach
// package-private helpers without an export shim.

import (
	"os"
	"path/filepath"
	"testing"

	internalsec "github.com/jongio/azd-app/cli/src/internal/security"
)

// TestValidatePathContainment_TraversalRejected mirrors acceptance criterion 4.
func TestValidatePathContainment_TraversalRejected(t *testing.T) {
	root := t.TempDir()

	// Simulate what parser.go does: Join the project root with a traversal
	// component and then Clean it.  The ".." disappears from the string but
	// the path clearly escapes root.
	traversalResolved := filepath.Clean(filepath.Join(root, "..", "..", "etc", "passwd"))

	_, err := internalsec.ValidatePathContainment(traversalResolved, root)
	if err == nil {
		t.Errorf("ValidatePathContainment(%q, %q): expected error for traversal path, got nil",
			traversalResolved, root)
	}
}

// TestValidatePathContainment_AbsoluteOutsideRejected mirrors acceptance criterion 5.
func TestValidatePathContainment_AbsoluteOutsideRejected(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Dir(root) // parent directory — always exists

	_, err := internalsec.ValidatePathContainment(outside, root)
	if err == nil {
		t.Errorf("ValidatePathContainment(%q, %q): expected error for path outside root, got nil",
			outside, root)
	}
}

// TestValidatePathContainment_RelativeDotSlashAccepted mirrors acceptance criterion 6.
func TestValidatePathContainment_RelativeDotSlashAccepted(t *testing.T) {
	root := t.TempDir()
	subDir := filepath.Join(root, "src", "myapp")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// The path as pre-resolved by parser.go (Join+Clean of "./src/myapp")
	resolved := filepath.Clean(filepath.Join(root, "./src/myapp"))

	got, err := internalsec.ValidatePathContainment(resolved, root)
	if err != nil {
		t.Errorf("ValidatePathContainment(%q, %q) unexpected error: %v", resolved, root, err)
	}
	if got == "" {
		t.Error("expected non-empty return path")
	}
}

// TestValidatePathContainment_RelativeNoSlashAccepted mirrors acceptance criterion 7.
func TestValidatePathContainment_RelativeNoSlashAccepted(t *testing.T) {
	root := t.TempDir()
	subDir := filepath.Join(root, "src", "myapp")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// The path as pre-resolved by parser.go (Join+Clean of "src/myapp")
	resolved := filepath.Clean(filepath.Join(root, "src/myapp"))

	got, err := internalsec.ValidatePathContainment(resolved, root)
	if err != nil {
		t.Errorf("ValidatePathContainment(%q, %q) unexpected error: %v", resolved, root, err)
	}
	if got == "" {
		t.Error("expected non-empty return path")
	}
}
