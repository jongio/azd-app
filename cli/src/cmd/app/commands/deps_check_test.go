package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-core/cliout"
	types "github.com/jongio/azd-core/projecttype"
)

func TestNodeDepsMarker(t *testing.T) {
	tmpDir := t.TempDir()

	dir := filepath.Join(tmpDir, "installed")
	if err := os.MkdirAll(filepath.Join(dir, "node_modules"), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if marker, ok := nodeDepsMarker(types.NodeProject{Dir: dir}); !ok {
		t.Errorf("expected installed=true for %s", marker)
	}

	missing := filepath.Join(tmpDir, "missing")
	if err := os.MkdirAll(missing, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, ok := nodeDepsMarker(types.NodeProject{Dir: missing}); ok {
		t.Error("expected installed=false when node_modules is absent")
	}

	// Workspace child resolves node_modules at the workspace root.
	wsRoot := filepath.Join(tmpDir, "ws")
	child := filepath.Join(wsRoot, "packages", "child")
	if err := os.MkdirAll(child, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(wsRoot, "node_modules"), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, ok := nodeDepsMarker(types.NodeProject{Dir: child, WorkspaceRoot: wsRoot}); !ok {
		t.Error("expected installed=true via workspace root node_modules")
	}
}

func TestPythonDepsMarker(t *testing.T) {
	tmpDir := t.TempDir()

	venvDir := filepath.Join(tmpDir, "venv-style")
	if err := os.MkdirAll(filepath.Join(venvDir, ".venv"), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, ok := pythonDepsMarker(types.PythonProject{Dir: venvDir}); !ok {
		t.Error("expected installed=true for .venv")
	}

	altDir := filepath.Join(tmpDir, "alt")
	if err := os.MkdirAll(filepath.Join(altDir, "venv"), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, ok := pythonDepsMarker(types.PythonProject{Dir: altDir}); !ok {
		t.Error("expected installed=true for venv")
	}

	missing := filepath.Join(tmpDir, "missing")
	if err := os.MkdirAll(missing, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, ok := pythonDepsMarker(types.PythonProject{Dir: missing}); ok {
		t.Error("expected installed=false when no environment exists")
	}
}

func TestDotnetDepsMarker(t *testing.T) {
	tmpDir := t.TempDir()

	proj := filepath.Join(tmpDir, "api", "api.csproj")
	objDir := filepath.Join(tmpDir, "api", "obj")
	if err := os.MkdirAll(objDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(objDir, "project.assets.json"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, ok := dotnetDepsMarker(types.DotnetProject{Path: proj}); !ok {
		t.Error("expected installed=true when project.assets.json exists")
	}

	missing := filepath.Join(tmpDir, "web", "web.csproj")
	if err := os.MkdirAll(filepath.Dir(missing), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, ok := dotnetDepsMarker(types.DotnetProject{Path: missing}); ok {
		t.Error("expected installed=false without a restore marker")
	}
}

func TestBuildDepsCheckResult(t *testing.T) {
	tmpDir := t.TempDir()

	installedNode := filepath.Join(tmpDir, "web")
	if err := os.MkdirAll(filepath.Join(installedNode, "node_modules"), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	missingNode := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(missingNode, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	projects := DetectedProjects{
		Node: []types.NodeProject{
			{Dir: installedNode, PackageManager: "npm"},
			{Dir: missingNode, PackageManager: "npm"},
		},
	}

	result := buildDepsCheckResult(projects)
	if result.TotalChecked != 2 {
		t.Errorf("TotalChecked = %d, want 2", result.TotalChecked)
	}
	if result.Missing != 1 {
		t.Errorf("Missing = %d, want 1", result.Missing)
	}
	if result.AllInstalled {
		t.Error("AllInstalled should be false with one missing project")
	}

	// All installed case.
	if err := os.MkdirAll(filepath.Join(missingNode, "node_modules"), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	result = buildDepsCheckResult(projects)
	if result.Missing != 0 || !result.AllInstalled {
		t.Errorf("expected all installed, got Missing=%d AllInstalled=%v", result.Missing, result.AllInstalled)
	}
}

func TestRenderDepsCheckText(t *testing.T) {
	result := depsCheckResult{
		Projects: []depsCheckStatus{
			{Type: "node", Dir: "/root/web", Manager: "npm", Installed: true},
			{Type: "python", Dir: "/root/api", Manager: "pip", Installed: false},
		},
		TotalChecked: 2,
		Missing:      1,
	}

	var buf bytes.Buffer
	renderDepsCheckText(&buf, result, "/root")
	out := buf.String()

	for _, want := range []string{"[installed]", "[missing]", "node/npm", "python/pip", "missing dependencies"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestRunDepsCheck(t *testing.T) {
	_ = cliout.SetFormat("default")
	t.Cleanup(func() { _ = cliout.SetFormat("default") })

	tmpDir := t.TempDir()
	if resolved, err := filepath.EvalSymlinks(tmpDir); err == nil {
		tmpDir = resolved
	}

	azureYaml := `name: test-app
services:
  api:
    project: ./api
    language: nodejs
  web:
    project: ./web
    language: nodejs
`
	if err := os.WriteFile(filepath.Join(tmpDir, "azure.yaml"), []byte(azureYaml), 0o600); err != nil {
		t.Fatalf("write azure.yaml: %v", err)
	}

	for _, name := range []string{"api", "web"} {
		dir := filepath.Join(tmpDir, name)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"`+name+`"}`), 0o600); err != nil {
			t.Fatalf("write package.json: %v", err)
		}
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	// Registered after t.TempDir() so it runs first (LIFO), restoring the
	// working directory before TempDir's RemoveAll on Windows.
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	t.Run("missing dependencies returns error", func(t *testing.T) {
		// Only api has node_modules installed.
		if err := os.MkdirAll(filepath.Join(tmpDir, "api", "node_modules"), 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		_ = os.RemoveAll(filepath.Join(tmpDir, "web", "node_modules"))

		var buf bytes.Buffer
		err := runDepsCheck(&DepsOptions{}, &buf)
		if err == nil {
			t.Fatal("expected a non-nil error when a service is missing dependencies")
		}
		if !strings.Contains(buf.String(), "[missing]") {
			t.Errorf("expected report to flag a missing service, got:\n%s", buf.String())
		}
	})

	t.Run("all installed returns nil", func(t *testing.T) {
		for _, name := range []string{"api", "web"} {
			if err := os.MkdirAll(filepath.Join(tmpDir, name, "node_modules"), 0o750); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
		}

		var buf bytes.Buffer
		if err := runDepsCheck(&DepsOptions{}, &buf); err != nil {
			t.Fatalf("expected nil error when all installed, got %v", err)
		}
		if !strings.Contains(buf.String(), "installed") {
			t.Errorf("expected success report, got:\n%s", buf.String())
		}
	})

	t.Run("service filter narrows the check", func(t *testing.T) {
		// web is missing, but limiting to api (installed) should pass.
		_ = os.RemoveAll(filepath.Join(tmpDir, "web", "node_modules"))
		if err := os.MkdirAll(filepath.Join(tmpDir, "api", "node_modules"), 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		var buf bytes.Buffer
		if err := runDepsCheck(&DepsOptions{Services: []string{"api"}}, &buf); err != nil {
			t.Fatalf("expected nil error when only checking an installed service, got %v", err)
		}
	})
}
