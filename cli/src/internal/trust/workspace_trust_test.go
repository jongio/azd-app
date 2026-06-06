package trust

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// writeAzureYaml writes azure.yaml with the given content into dir.
func writeAzureYaml(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write azure.yaml: %v", err)
	}
}

// newStore is a test helper that returns a TrustStore backed by a temp dir so
// tests never touch the real ~/.azd directory.
func newStore(t *testing.T) *TrustStore {
	t.Helper()
	return newTrustStoreAt(filepath.Join(t.TempDir(), "trusted-workspaces.json"))
}

// TestIsWorkspaceTrusted_NewWorkspace verifies that a workspace that has never
// been trusted is reported as untrusted without an error.
func TestIsWorkspaceTrusted_NewWorkspace(t *testing.T) {
	dir := t.TempDir()
	writeAzureYaml(t, dir, "name: test\nservices: {}\n")

	store := newStore(t)

	trusted, err := store.IsWorkspaceTrusted(dir)
	if err != nil {
		t.Fatalf("unexpected error for new workspace: %v", err)
	}
	if trusted {
		t.Error("expected new workspace to be untrusted; got trusted=true")
	}
}

// TestIsWorkspaceTrusted_AfterTrust verifies that a workspace is trusted after
// a successful TrustWorkspace call.
func TestIsWorkspaceTrusted_AfterTrust(t *testing.T) {
	dir := t.TempDir()
	writeAzureYaml(t, dir, "name: test\nservices: {}\n")

	store := newStore(t)

	if err := store.TrustWorkspace(dir); err != nil {
		t.Fatalf("TrustWorkspace failed: %v", err)
	}

	trusted, err := store.IsWorkspaceTrusted(dir)
	if err != nil {
		t.Fatalf("unexpected error after trusting: %v", err)
	}
	if !trusted {
		t.Error("expected workspace to be trusted after TrustWorkspace; got trusted=false")
	}
}

// TestIsWorkspaceTrusted_HashChanged verifies that modifying azure.yaml after
// a workspace has been trusted returns (false, ErrHashChanged).
func TestIsWorkspaceTrusted_HashChanged(t *testing.T) {
	dir := t.TempDir()
	writeAzureYaml(t, dir, "name: test\nservices: {}\n")

	store := newStore(t)

	if err := store.TrustWorkspace(dir); err != nil {
		t.Fatalf("TrustWorkspace failed: %v", err)
	}

	// Simulate a template update by changing the azure.yaml content.
	writeAzureYaml(t, dir, "name: test\nservices:\n  web:\n    language: TypeScript\n    project: .\n")

	trusted, err := store.IsWorkspaceTrusted(dir)
	if !errors.Is(err, ErrHashChanged) {
		t.Errorf("expected ErrHashChanged; got trusted=%v err=%v", trusted, err)
	}
	if trusted {
		t.Error("expected workspace to be untrusted after azure.yaml modification; got trusted=true")
	}
}

// TestRevokeWorkspace verifies that revoking trust causes the workspace to be
// reported as untrusted again.
func TestRevokeWorkspace(t *testing.T) {
	dir := t.TempDir()
	writeAzureYaml(t, dir, "name: test\nservices: {}\n")

	store := newStore(t)

	if err := store.TrustWorkspace(dir); err != nil {
		t.Fatalf("TrustWorkspace failed: %v", err)
	}

	if err := store.RevokeWorkspace(dir); err != nil {
		t.Fatalf("RevokeWorkspace failed: %v", err)
	}

	trusted, err := store.IsWorkspaceTrusted(dir)
	if err != nil {
		t.Fatalf("unexpected error after revoke: %v", err)
	}
	if trusted {
		t.Error("expected workspace to be untrusted after RevokeWorkspace; got trusted=true")
	}
}

// TestTrustStoreFilePermissions verifies that the trust-store file is written
// with 0o600 permissions so that other users on the host cannot read it.
// Skipped on Windows where Unix permission bits are not enforced.
func TestTrustStoreFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission enforcement is not available on Windows")
	}

	dir := t.TempDir()
	writeAzureYaml(t, dir, "name: test\nservices: {}\n")

	storeDir := t.TempDir()
	storePath := filepath.Join(storeDir, "trusted-workspaces.json")
	store := newTrustStoreAt(storePath)

	if err := store.TrustWorkspace(dir); err != nil {
		t.Fatalf("TrustWorkspace failed: %v", err)
	}

	info, err := os.Stat(storePath)
	if err != nil {
		t.Fatalf("failed to stat trust store: %v", err)
	}

	got := info.Mode().Perm()
	if got != storeFileMode {
		t.Errorf("trust store file permissions: got %04o, want %04o", got, storeFileMode)
	}
}
