package workspace

import (
	"github.com/jongio/azd-app/cli/src/internal/types"
)

// Handler manages workspace detection and filtering for package managers.
// It prevents race conditions by ensuring only workspace roots are installed
// when workspace children are detected.
type Handler struct {
	workspaceRoots map[string]bool
}

// NewHandler creates a new workspace handler.
func NewHandler() *Handler {
	return &Handler{
		workspaceRoots: make(map[string]bool),
	}
}

// FilterNodeProjects filters Node.js projects to prevent duplicate installations
// in workspace scenarios. Returns only the projects that should be installed.
//
// Logic:
// - Workspace roots: Always included
// - Workspace children: Skipped if their workspace root is in the list
// - Independent projects: Always included
func (h *Handler) FilterNodeProjects(projects []types.NodeProject) []types.NodeProject {
	var filtered []types.NodeProject
	workspaceHandled := make(map[string]bool)

	for _, project := range projects {
		// If this is a workspace root, add it and mark as handled
		if project.IsWorkspaceRoot {
			filtered = append(filtered, project)
			workspaceHandled[project.Dir] = true
		} else if project.WorkspaceRoot != "" {
			// This is a workspace child - skip if workspace root is already handled
			if !workspaceHandled[project.WorkspaceRoot] {
				// Workspace root not found in our projects, install this child project
				filtered = append(filtered, project)
			}
			// If workspace root exists, it will handle this child's dependencies
		} else {
			// Independent project, not part of any workspace
			filtered = append(filtered, project)
		}
	}

	return filtered
}

// GetWorkspaceRoots returns all detected workspace root directories.
func (h *Handler) GetWorkspaceRoots(projects []types.NodeProject) []string {
	roots := make([]string, 0)
	for _, project := range projects {
		if project.IsWorkspaceRoot {
			roots = append(roots, project.Dir)
		}
	}
	return roots
}

// GetWorkspaceChildren returns all child projects for a given workspace root.
func (h *Handler) GetWorkspaceChildren(projects []types.NodeProject, workspaceRoot string) []types.NodeProject {
	children := make([]types.NodeProject, 0)
	for _, project := range projects {
		if project.WorkspaceRoot == workspaceRoot {
			children = append(children, project)
		}
	}
	return children
}

// IsWorkspaceChild checks if a project is a workspace child.
func (h *Handler) IsWorkspaceChild(project types.NodeProject) bool {
	return project.WorkspaceRoot != ""
}

// HasWorkspaces checks if any projects define workspaces.
func (h *Handler) HasWorkspaces(projects []types.NodeProject) bool {
	for _, project := range projects {
		if project.IsWorkspaceRoot {
			return true
		}
	}
	return false
}

// CountWorkspaces returns the number of workspace roots and children.
func (h *Handler) CountWorkspaces(projects []types.NodeProject) (roots int, children int) {
	for _, project := range projects {
		if project.IsWorkspaceRoot {
			roots++
		} else if project.WorkspaceRoot != "" {
			children++
		}
	}
	return roots, children
}
