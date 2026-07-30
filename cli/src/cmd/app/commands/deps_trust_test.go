package commands

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// errTrustDenied stands in for the real trust-store refusal so the tests can
// assert the gate short-circuits execute() without touching the user's home
// directory or spawning a package manager.
var errTrustDenied = errors.New("workspace not trusted")

// newTrustGateProject builds a minimal azd workspace containing one detectable
// Node project and makes it the working directory. It returns the project root.
func newTrustGateProject(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	webDir := filepath.Join(root, "web")
	if err := os.MkdirAll(webDir, 0o750); err != nil {
		t.Fatalf("create web dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "package.json"), []byte(`{"name":"web"}`), 0o600); err != nil {
		t.Fatalf("create package.json: %v", err)
	}

	content := "name: test-app\nservices:\n  web:\n    project: ./web\n    host: localhost\n"
	if err := os.WriteFile(filepath.Join(root, "azure.yaml"), []byte(content), 0o600); err != nil {
		t.Fatalf("create azure.yaml: %v", err)
	}

	// execute() resolves the search root from the process working directory.
	t.Chdir(root)

	return root
}

// TestDepsExecuteBlocksInstallWhenWorkspaceUntrusted is the core regression
// test for the deps trust gate. Installing dependencies runs arbitrary code
// through package-manager lifecycle scripts, so a refusal from the trust store
// must abort before any installer is spawned. This also covers the MCP
// install_dependencies tool, which shells out to 'azd app deps'.
func TestDepsExecuteBlocksInstallWhenWorkspaceUntrusted(t *testing.T) {
	root := newTrustGateProject(t)

	var gotPath string
	executor := newDepsExecutor(&DepsOptions{})
	executor.ensureTrusted = func(azureYamlPath string) error {
		gotPath = azureYamlPath
		return errTrustDenied
	}

	err := executor.execute()
	if !errors.Is(err, errTrustDenied) {
		t.Fatalf("execute() error = %v; want the trust refusal to propagate unwrapped", err)
	}

	if gotPath == "" {
		t.Fatal("trust gate was never consulted; deps would install without consent")
	}
	if filepath.Base(gotPath) != "azure.yaml" {
		t.Errorf("trust gate got %q; want a path ending in azure.yaml", gotPath)
	}

	// The gate must receive the azure.yaml of the workspace being installed.
	// Compare by identity so symlinked temp dirs on macOS and short paths on
	// Windows don't produce a false failure.
	want, err := os.Stat(filepath.Join(root, "azure.yaml"))
	if err != nil {
		t.Fatalf("stat expected azure.yaml: %v", err)
	}
	got, err := os.Stat(gotPath)
	if err != nil {
		t.Fatalf("stat azure.yaml passed to trust gate: %v", err)
	}
	if !os.SameFile(want, got) {
		t.Errorf("trust gate received %q, which is not the workspace azure.yaml", gotPath)
	}
}

// TestDepsExecuteDryRunSkipsTrustGate verifies --dry-run stays usable in an
// untrusted workspace. It only prints what would be installed and never spawns
// a package manager, so gating it would be a pointless prompt.
func TestDepsExecuteDryRunSkipsTrustGate(t *testing.T) {
	newTrustGateProject(t)

	called := false
	executor := newDepsExecutor(&DepsOptions{DryRun: true})
	executor.ensureTrusted = func(string) error {
		called = true
		return errTrustDenied
	}

	if err := executor.execute(); err != nil {
		t.Fatalf("execute() with --dry-run returned %v; want nil", err)
	}
	if called {
		t.Error("--dry-run consulted the trust gate; it executes no project code and must not prompt")
	}
}

// TestDepsExecuteDoesNotCleanWhenWorkspaceUntrusted proves the gate sits ahead
// of the destructive --clean step, not just the install step. An untrusted
// workspace must not have its dependency directories deleted.
func TestDepsExecuteDoesNotCleanWhenWorkspaceUntrusted(t *testing.T) {
	root := newTrustGateProject(t)

	modules := filepath.Join(root, "web", "node_modules")
	if err := os.MkdirAll(modules, 0o750); err != nil {
		t.Fatalf("create node_modules: %v", err)
	}
	marker := filepath.Join(modules, "marker.txt")
	if err := os.WriteFile(marker, []byte("keep"), 0o600); err != nil {
		t.Fatalf("create marker: %v", err)
	}

	executor := newDepsExecutor(&DepsOptions{Clean: true})
	executor.ensureTrusted = func(string) error { return errTrustDenied }

	if err := executor.execute(); !errors.Is(err, errTrustDenied) {
		t.Fatalf("execute() error = %v; want the trust refusal", err)
	}

	if _, err := os.Stat(marker); err != nil {
		t.Errorf("--clean deleted dependencies in an untrusted workspace: %v", err)
	}
}
