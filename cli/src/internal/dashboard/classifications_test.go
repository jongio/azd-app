package dashboard

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

// TestSaveAzureYaml_FilePermissions verifies that saveAzureYaml writes
// azure.yaml with mode 0o600 (owner read-write only), not 0o644. This guards
// against SEC-004: azure.yaml contains potentially sensitive configuration
// (workspace IDs, tenant refs) that should not be world-readable.
//
// On Windows, os.Chmod maps only the read-only bit, so the test skips the
// exact-mode assertion and only verifies the file was written successfully.
func TestSaveAzureYaml_FilePermissions(t *testing.T) {
	tmp := t.TempDir()

	ay := &service.AzureYaml{Name: "perm-test"}
	if err := saveAzureYaml(tmp, ay); err != nil {
		t.Fatalf("saveAzureYaml: %v", err)
	}

	path := filepath.Join(tmp, "azure.yaml")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat(%q): %v", path, err)
	}

	if runtime.GOOS == "windows" {
		// Windows ACLs control effective access; os.Chmod only toggles the
		// read-only attribute. Skip the exact mode assertion and verify the
		// file is writable (not accidentally marked read-only).
		if info.Mode().Perm()&0o200 == 0 {
			t.Errorf("azure.yaml is read-only on Windows (mode=%o), want writable", info.Mode().Perm())
		}
		return
	}

	const want = os.FileMode(0o600)
	if got := info.Mode().Perm(); got != want {
		t.Errorf("azure.yaml permissions=%o, want %o (owner read-write only)", got, want)
	}
}
