package workspace

import (
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/types"
)

func TestFilterNodeProjects(t *testing.T) {
	tests := []struct {
		name     string
		projects []types.NodeProject
		expected int
		desc     string
	}{
		{
			name: "workspace root with children",
			projects: []types.NodeProject{
				{Dir: "/root", PackageManager: "npm", IsWorkspaceRoot: true},
				{Dir: "/root/api", PackageManager: "npm", WorkspaceRoot: "/root"},
				{Dir: "/root/webapp", PackageManager: "npm", WorkspaceRoot: "/root"},
			},
			expected: 1,
			desc:     "Should only return workspace root",
		},
		{
			name: "multiple independent projects",
			projects: []types.NodeProject{
				{Dir: "/project1", PackageManager: "npm"},
				{Dir: "/project2", PackageManager: "yarn"},
				{Dir: "/project3", PackageManager: "pnpm"},
			},
			expected: 3,
			desc:     "Should return all independent projects",
		},
		{
			name: "workspace children without root",
			projects: []types.NodeProject{
				{Dir: "/root/api", PackageManager: "npm", WorkspaceRoot: "/root"},
				{Dir: "/root/webapp", PackageManager: "npm", WorkspaceRoot: "/root"},
			},
			expected: 2,
			desc:     "Should return all children when root not in list",
		},
		{
			name: "mixed workspace and independent",
			projects: []types.NodeProject{
				{Dir: "/workspace", PackageManager: "npm", IsWorkspaceRoot: true},
				{Dir: "/workspace/api", PackageManager: "npm", WorkspaceRoot: "/workspace"},
				{Dir: "/independent", PackageManager: "pnpm"},
			},
			expected: 2,
			desc:     "Should return workspace root and independent project",
		},
		{
			name:     "empty list",
			projects: []types.NodeProject{},
			expected: 0,
			desc:     "Should return empty list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler()
			filtered := handler.FilterNodeProjects(tt.projects)
			if len(filtered) != tt.expected {
				t.Errorf("%s: expected %d projects, got %d", tt.desc, tt.expected, len(filtered))
			}
		})
	}
}

func TestGetWorkspaceRoots(t *testing.T) {
	projects := []types.NodeProject{
		{Dir: "/root1", PackageManager: "npm", IsWorkspaceRoot: true},
		{Dir: "/root1/api", PackageManager: "npm", WorkspaceRoot: "/root1"},
		{Dir: "/root2", PackageManager: "pnpm", IsWorkspaceRoot: true},
		{Dir: "/independent", PackageManager: "yarn"},
	}

	handler := NewHandler()
	roots := handler.GetWorkspaceRoots(projects)

	if len(roots) != 2 {
		t.Errorf("Expected 2 workspace roots, got %d", len(roots))
	}
}

func TestGetWorkspaceChildren(t *testing.T) {
	projects := []types.NodeProject{
		{Dir: "/root", PackageManager: "npm", IsWorkspaceRoot: true},
		{Dir: "/root/api", PackageManager: "npm", WorkspaceRoot: "/root"},
		{Dir: "/root/webapp", PackageManager: "npm", WorkspaceRoot: "/root"},
		{Dir: "/root/lib", PackageManager: "npm", WorkspaceRoot: "/root"},
	}

	handler := NewHandler()
	children := handler.GetWorkspaceChildren(projects, "/root")

	if len(children) != 3 {
		t.Errorf("Expected 3 children, got %d", len(children))
	}

	// Verify all children have correct workspace root
	for _, child := range children {
		if child.WorkspaceRoot != "/root" {
			t.Errorf("Child has wrong workspace root: %s", child.WorkspaceRoot)
		}
	}
}

func TestHasWorkspaces(t *testing.T) {
	tests := []struct {
		name     string
		projects []types.NodeProject
		expected bool
	}{
		{
			name: "has workspace",
			projects: []types.NodeProject{
				{Dir: "/root", IsWorkspaceRoot: true},
			},
			expected: true,
		},
		{
			name: "no workspace",
			projects: []types.NodeProject{
				{Dir: "/project", IsWorkspaceRoot: false},
			},
			expected: false,
		},
		{
			name:     "empty list",
			projects: []types.NodeProject{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler()
			result := handler.HasWorkspaces(tt.projects)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCountWorkspaces(t *testing.T) {
	projects := []types.NodeProject{
		{Dir: "/root1", IsWorkspaceRoot: true},
		{Dir: "/root1/api", WorkspaceRoot: "/root1"},
		{Dir: "/root1/webapp", WorkspaceRoot: "/root1"},
		{Dir: "/root2", IsWorkspaceRoot: true},
		{Dir: "/root2/lib", WorkspaceRoot: "/root2"},
		{Dir: "/independent", IsWorkspaceRoot: false},
	}

	handler := NewHandler()
	roots, children := handler.CountWorkspaces(projects)

	if roots != 2 {
		t.Errorf("Expected 2 roots, got %d", roots)
	}
	if children != 3 {
		t.Errorf("Expected 3 children, got %d", children)
	}
}

func TestIsWorkspaceChild(t *testing.T) {
	handler := NewHandler()

	child := types.NodeProject{Dir: "/root/api", WorkspaceRoot: "/root"}
	if !handler.IsWorkspaceChild(child) {
		t.Error("Expected IsWorkspaceChild to return true for child project")
	}

	independent := types.NodeProject{Dir: "/project", WorkspaceRoot: ""}
	if handler.IsWorkspaceChild(independent) {
		t.Error("Expected IsWorkspaceChild to return false for independent project")
	}
}
