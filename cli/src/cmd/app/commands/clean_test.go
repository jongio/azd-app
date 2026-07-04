package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	types "github.com/jongio/azd-core/projecttype"
)

func TestNewCleanCommand(t *testing.T) {
	cmd := NewCleanCommand()
	if cmd == nil {
		t.Fatal("NewCleanCommand returned nil")
	}
	if cmd.Use != "clean" {
		t.Fatalf("Use = %q, want clean", cmd.Use)
	}
	for _, name := range []string{"deps", "dry-run", "service"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("flag %q not found", name)
		}
	}
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	mkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestCollectCleanTargets(t *testing.T) {
	root := t.TempDir()
	nodeDir := filepath.Join(root, "web")
	pyDir := filepath.Join(root, "api")
	netDir := filepath.Join(root, "svc")

	mkdirAll(t, filepath.Join(nodeDir, "dist"))
	mkdirAll(t, filepath.Join(nodeDir, "node_modules"))
	mkdirAll(t, filepath.Join(pyDir, "__pycache__"))
	mkdirAll(t, filepath.Join(pyDir, ".venv"))
	mkdirAll(t, filepath.Join(netDir, "bin"))
	mkdirAll(t, filepath.Join(netDir, "obj"))

	projects := DetectedProjects{
		Node:   []types.NodeProject{{Dir: nodeDir}},
		Python: []types.PythonProject{{Dir: pyDir}},
		Dotnet: []types.DotnetProject{{Path: filepath.Join(netDir, "svc.csproj")}},
	}

	// Default: build artifacts only, no dependency directories.
	got := collectCleanTargets(projects, false)
	paths := targetPaths(got)
	for _, want := range []string{
		filepath.Join(nodeDir, "dist"),
		filepath.Join(pyDir, "__pycache__"),
		filepath.Join(netDir, "bin"),
		filepath.Join(netDir, "obj"),
	} {
		if !paths[want] {
			t.Errorf("expected build target %s to be collected", want)
		}
	}
	if paths[filepath.Join(nodeDir, "node_modules")] {
		t.Error("node_modules should not be collected without --deps")
	}
	if paths[filepath.Join(pyDir, ".venv")] {
		t.Error(".venv should not be collected without --deps")
	}

	// With deps: dependency directories are included and tagged accordingly.
	got = collectCleanTargets(projects, true)
	paths = targetPaths(got)
	if !paths[filepath.Join(nodeDir, "node_modules")] {
		t.Error("node_modules should be collected with --deps")
	}
	if !paths[filepath.Join(pyDir, ".venv")] {
		t.Error(".venv should be collected with --deps")
	}
	for _, tgt := range got {
		if filepath.Base(tgt.Path) == "node_modules" && tgt.Category != cleanCategoryDeps {
			t.Errorf("node_modules category = %q, want %q", tgt.Category, cleanCategoryDeps)
		}
		if filepath.Base(tgt.Path) == "dist" && tgt.Category != cleanCategoryBuild {
			t.Errorf("dist category = %q, want %q", tgt.Category, cleanCategoryBuild)
		}
	}
}

func targetPaths(targets []cleanTarget) map[string]bool {
	m := make(map[string]bool, len(targets))
	for _, t := range targets {
		m[t.Path] = true
	}
	return m
}

func TestDirSize(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), "12345")       // 5 bytes
	writeFile(t, filepath.Join(dir, "sub", "b.txt"), "1234") // 4 bytes

	if got := dirSize(dir); got != 9 {
		t.Fatalf("dirSize = %d, want 9", got)
	}
	if got := dirSize(filepath.Join(dir, "does-not-exist")); got != 0 {
		t.Fatalf("dirSize(missing) = %d, want 0", got)
	}
}

func TestSafeToRemove(t *testing.T) {
	root := t.TempDir()
	resolved, err := resolveWorkspaceRoot(root)
	if err != nil {
		t.Fatalf("resolveWorkspaceRoot: %v", err)
	}

	valid := filepath.Join(root, "dist")
	mkdirAll(t, valid)
	if err := safeToRemove(valid, resolved); err != nil {
		t.Errorf("safeToRemove(valid) = %v, want nil", err)
	}

	// Non-artifact directory names are rejected.
	nonArtifact := filepath.Join(root, "src")
	mkdirAll(t, nonArtifact)
	if err := safeToRemove(nonArtifact, resolved); err == nil {
		t.Error("safeToRemove(src) should reject non-artifact directory")
	}

	// Artifact-named directory outside the workspace root is rejected.
	outside := t.TempDir()
	outsideArtifact := filepath.Join(outside, "dist")
	mkdirAll(t, outsideArtifact)
	if err := safeToRemove(outsideArtifact, resolved); err == nil {
		t.Error("safeToRemove(outside) should reject path outside workspace")
	}
}

func TestHumanBytes(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1048576, "1.0 MiB"},
	}
	for _, tt := range tests {
		if got := humanBytes(tt.in); got != tt.want {
			t.Errorf("humanBytes(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// setupCleanProject writes an azure.yaml with a single Node service and returns the
// service directory. The caller chdirs into the project root.
func setupCleanProject(t *testing.T) (root, serviceDir string) {
	t.Helper()
	root = t.TempDir()
	azureYaml := "name: clean-test\nservices:\n  web:\n    host: local\n    language: js\n    project: ./web\n"
	writeFile(t, filepath.Join(root, "azure.yaml"), azureYaml)
	serviceDir = filepath.Join(root, "web")
	writeFile(t, filepath.Join(serviceDir, "package.json"), "{\"name\":\"web\"}")
	writeFile(t, filepath.Join(serviceDir, "dist", "bundle.js"), "console.log(1)")
	writeFile(t, filepath.Join(serviceDir, "node_modules", "dep", "index.js"), "module.exports = 1")
	return root, serviceDir
}

func TestRunCleanDryRun(t *testing.T) {
	root, serviceDir := setupCleanProject(t)
	t.Chdir(root)

	var buf bytes.Buffer
	if err := runClean(&cleanOptions{dryRun: true, writer: &buf}); err != nil {
		t.Fatalf("runClean dry-run failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Reclaimable") {
		t.Fatalf("dry-run output should report reclaimable size:\n%s", out)
	}
	if !strings.Contains(out, filepath.Join(serviceDir, "dist")) {
		t.Fatalf("dry-run should list dist:\n%s", out)
	}
	// Dry-run must not delete anything.
	if _, err := os.Stat(filepath.Join(serviceDir, "dist")); err != nil {
		t.Fatalf("dry-run deleted dist: %v", err)
	}
}

func TestRunCleanRemovesBuildKeepsDeps(t *testing.T) {
	root, serviceDir := setupCleanProject(t)
	t.Chdir(root)

	var buf bytes.Buffer
	if err := runClean(&cleanOptions{writer: &buf}); err != nil {
		t.Fatalf("runClean failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Freed") {
		t.Fatalf("output should report freed space:\n%s", buf.String())
	}
	if _, err := os.Stat(filepath.Join(serviceDir, "dist")); !os.IsNotExist(err) {
		t.Errorf("dist should be removed, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(serviceDir, "node_modules")); err != nil {
		t.Errorf("node_modules should be kept without --deps, stat err = %v", err)
	}
}

func TestRunCleanWithDepsRemovesNodeModules(t *testing.T) {
	root, serviceDir := setupCleanProject(t)
	t.Chdir(root)

	var buf bytes.Buffer
	if err := runClean(&cleanOptions{deps: true, writer: &buf}); err != nil {
		t.Fatalf("runClean --deps failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(serviceDir, "node_modules")); !os.IsNotExist(err) {
		t.Errorf("node_modules should be removed with --deps, stat err = %v", err)
	}
}
