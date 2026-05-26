package commands

import (
	"os"
	"path/filepath"
	"testing"

	types "github.com/jongio/azd-core/projecttype"
)

func TestLinkWorkspaceChildren_EmptyProjects(t *testing.T) {
	result := linkWorkspaceChildren(nil, "/tmp")
	if result != nil {
		t.Errorf("Expected nil for nil input, got %v", result)
	}

	result = linkWorkspaceChildren([]types.NodeProject{}, "/tmp")
	if len(result) != 0 {
		t.Errorf("Expected empty slice, got %d projects", len(result))
	}
}

func TestLinkWorkspaceChildren_NoWorkspaceRoot(t *testing.T) {
	// Projects without any workspace root above them should remain unchanged
	tmpDir := t.TempDir()

	childDir := filepath.Join(tmpDir, "child")
	if err := os.MkdirAll(childDir, 0o750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(childDir, "package.json"), []byte(`{"name":"child"}`), 0o600); err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	projects := []types.NodeProject{
		{Dir: childDir, PackageManager: "npm"},
	}

	result := linkWorkspaceChildren(projects, tmpDir)
	if len(result) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(result))
	}
	if result[0].WorkspaceRoot != "" {
		t.Errorf("Expected empty WorkspaceRoot, got %q", result[0].WorkspaceRoot)
	}
}

func TestLinkWorkspaceChildren_DiscoversMissingRoot(t *testing.T) {
	// Simulate: workspace root at tmpDir (has pnpm-workspace.yaml),
	// two children in subdirs, root NOT in project list.
	tmpDir := t.TempDir()

	// Create workspace root indicators
	if err := os.WriteFile(filepath.Join(tmpDir, "pnpm-workspace.yaml"), []byte("packages:\n  - 'src/*'\n"), 0o600); err != nil {
		t.Fatalf("Failed to write pnpm-workspace.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name":"root"}`), 0o600); err != nil {
		t.Fatalf("Failed to write root package.json: %v", err)
	}

	// Create child directories
	child1 := filepath.Join(tmpDir, "src", "web")
	child2 := filepath.Join(tmpDir, "src", "api")
	for _, d := range []string{child1, child2} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatalf("Failed to create %s: %v", d, err)
		}
		if err := os.WriteFile(filepath.Join(d, "package.json"), []byte(`{"name":"child"}`), 0o600); err != nil {
			t.Fatalf("Failed to write package.json in %s: %v", d, err)
		}
	}

	projects := []types.NodeProject{
		{Dir: child1, PackageManager: "pnpm"},
		{Dir: child2, PackageManager: "pnpm"},
	}

	absRoot, _ := filepath.Abs(tmpDir)
	result := linkWorkspaceChildren(projects, absRoot)

	// Should have 3 projects: discovered root + 2 children
	if len(result) != 3 {
		t.Fatalf("Expected 3 projects (1 root + 2 children), got %d", len(result))
	}

	// First project should be the discovered workspace root
	if !result[0].IsWorkspaceRoot {
		t.Error("First project should be workspace root")
	}
	if result[0].Dir != absRoot {
		t.Errorf("Root Dir = %q, want %q", result[0].Dir, absRoot)
	}

	// Children should reference the root
	for i := 1; i < len(result); i++ {
		if result[i].WorkspaceRoot != absRoot {
			t.Errorf("Child %d WorkspaceRoot = %q, want %q", i, result[i].WorkspaceRoot, absRoot)
		}
		if result[i].IsWorkspaceRoot {
			t.Errorf("Child %d should not be workspace root", i)
		}
	}
}

func TestLinkWorkspaceChildren_ExistingRootUsesOriginalDir(t *testing.T) {
	// When workspace root IS in the project list (e.g., as a service),
	// children's WorkspaceRoot should match the root's original Dir value
	// for FilterNodeProjects compatibility.
	tmpDir := t.TempDir()

	// Create workspace root indicators
	if err := os.WriteFile(filepath.Join(tmpDir, "pnpm-workspace.yaml"), []byte("packages:\n  - 'src/*'\n"), 0o600); err != nil {
		t.Fatalf("Failed to write pnpm-workspace.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name":"root"}`), 0o600); err != nil {
		t.Fatalf("Failed to write root package.json: %v", err)
	}

	childDir := filepath.Join(tmpDir, "src", "web")
	if err := os.MkdirAll(childDir, 0o750); err != nil {
		t.Fatalf("Failed to create child dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(childDir, "package.json"), []byte(`{"name":"web"}`), 0o600); err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	// Root IS in the project list with its original Dir
	projects := []types.NodeProject{
		{Dir: tmpDir, PackageManager: "pnpm", IsWorkspaceRoot: true},
		{Dir: childDir, PackageManager: "pnpm"},
	}

	absRoot, _ := filepath.Abs(tmpDir)
	result := linkWorkspaceChildren(projects, absRoot)

	// Should still have 2 projects (root already existed, no new addition)
	if len(result) != 2 {
		t.Fatalf("Expected 2 projects, got %d", len(result))
	}

	// Find the child (non-root) project
	var child *types.NodeProject
	for i := range result {
		if !result[i].IsWorkspaceRoot {
			child = &result[i]
			break
		}
	}
	if child == nil {
		t.Fatal("Could not find child project in results")
	}

	// Child's WorkspaceRoot should match the root's Dir exactly
	if child.WorkspaceRoot != tmpDir {
		t.Errorf("Child WorkspaceRoot = %q, want %q (root's original Dir)", child.WorkspaceRoot, tmpDir)
	}
}

func TestLinkWorkspaceChildren_AllRoots(t *testing.T) {
	// If all projects are workspace roots, nothing should change
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "pnpm-workspace.yaml"), []byte("packages:\n  - '*'\n"), 0o600); err != nil {
		t.Fatalf("Failed to write pnpm-workspace.yaml: %v", err)
	}

	projects := []types.NodeProject{
		{Dir: tmpDir, PackageManager: "pnpm", IsWorkspaceRoot: true},
	}

	absRoot, _ := filepath.Abs(tmpDir)
	result := linkWorkspaceChildren(projects, absRoot)

	if len(result) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(result))
	}
	if result[0].WorkspaceRoot != "" {
		t.Errorf("Root should not have WorkspaceRoot set, got %q", result[0].WorkspaceRoot)
	}
}

func TestFindWorkspaceRootUpward_FindsPnpmWorkspace(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workspace root at tmpDir
	if err := os.WriteFile(filepath.Join(tmpDir, "pnpm-workspace.yaml"), []byte("packages:\n  - 'src/*'\n"), 0o600); err != nil {
		t.Fatalf("Failed to write pnpm-workspace.yaml: %v", err)
	}

	// Start from a nested child
	childDir := filepath.Join(tmpDir, "src", "packages", "web")
	if err := os.MkdirAll(childDir, 0o750); err != nil {
		t.Fatalf("Failed to create child dir: %v", err)
	}

	absRoot, _ := filepath.Abs(tmpDir)
	result := findWorkspaceRootUpward(childDir, absRoot)
	if result != absRoot {
		t.Errorf("findWorkspaceRootUpward() = %q, want %q", result, absRoot)
	}
}

func TestFindWorkspaceRootUpward_FindsPackageJsonWorkspaces(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workspace root via package.json workspaces field
	pkgJSON := `{"name":"root","workspaces":["packages/*"]}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(pkgJSON), 0o600); err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	childDir := filepath.Join(tmpDir, "packages", "lib")
	if err := os.MkdirAll(childDir, 0o750); err != nil {
		t.Fatalf("Failed to create child dir: %v", err)
	}

	absRoot, _ := filepath.Abs(tmpDir)
	result := findWorkspaceRootUpward(childDir, absRoot)
	if result != absRoot {
		t.Errorf("findWorkspaceRootUpward() = %q, want %q", result, absRoot)
	}
}

func TestFindWorkspaceRootUpward_StopsAtBoundary(t *testing.T) {
	tmpDir := t.TempDir()

	// Place workspace root ABOVE the boundary
	outerDir := filepath.Join(tmpDir, "outer")
	innerDir := filepath.Join(outerDir, "inner", "project")
	if err := os.MkdirAll(innerDir, 0o750); err != nil {
		t.Fatalf("Failed to create dirs: %v", err)
	}
	// Workspace root at outer level (above boundary)
	if err := os.WriteFile(filepath.Join(outerDir, "pnpm-workspace.yaml"), []byte("packages:\n  - '**'\n"), 0o600); err != nil {
		t.Fatalf("Failed to write pnpm-workspace.yaml: %v", err)
	}

	// Boundary is inner dir (below workspace root)
	absInner, _ := filepath.Abs(filepath.Join(outerDir, "inner"))
	result := findWorkspaceRootUpward(innerDir, absInner)
	if result != "" {
		t.Errorf("Expected empty (root above boundary), got %q", result)
	}
}

func TestFindWorkspaceRootUpward_NoWorkspaceFound(t *testing.T) {
	tmpDir := t.TempDir()

	childDir := filepath.Join(tmpDir, "some", "nested", "dir")
	if err := os.MkdirAll(childDir, 0o750); err != nil {
		t.Fatalf("Failed to create dirs: %v", err)
	}

	absRoot, _ := filepath.Abs(tmpDir)
	result := findWorkspaceRootUpward(childDir, absRoot)
	if result != "" {
		t.Errorf("Expected empty string, got %q", result)
	}
}

func TestFindWorkspaceRootUpward_StartDirIsRoot(t *testing.T) {
	tmpDir := t.TempDir()

	// The start dir itself is the workspace root
	if err := os.WriteFile(filepath.Join(tmpDir, "pnpm-workspace.yaml"), []byte("packages:\n  - '*'\n"), 0o600); err != nil {
		t.Fatalf("Failed to write pnpm-workspace.yaml: %v", err)
	}

	absRoot, _ := filepath.Abs(tmpDir)
	result := findWorkspaceRootUpward(absRoot, absRoot)
	if result != absRoot {
		t.Errorf("findWorkspaceRootUpward() = %q, want %q", result, absRoot)
	}
}
