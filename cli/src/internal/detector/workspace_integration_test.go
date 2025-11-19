package detector

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNpmWorkspaceIntegration tests the npm workspace test project.
// This validates the complete workspace detection and handling workflow.
func TestNpmWorkspaceIntegration(t *testing.T) {
	// Path to test workspace project
	workspaceDir := filepath.Join("..", "..", "..", "tests", "projects", "node", "test-npm-workspace")
	
	// Check if test project exists
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		t.Skipf("Test workspace project not found at %s", workspaceDir)
	}
	
	// Find all Node.js projects
	projects, err := FindNodeProjects(workspaceDir)
	if err != nil {
		t.Fatalf("FindNodeProjects failed: %v", err)
	}
	
	// Should find exactly 3 projects: root + 2 children
	if len(projects) != 3 {
		t.Fatalf("Expected 3 projects (root + api + webapp), found %d", len(projects))
	}
	
	// Verify workspace root
	var workspaceRoot *int
	for i, p := range projects {
		absWorkspaceDir, _ := filepath.Abs(workspaceDir)
		if p.Dir == absWorkspaceDir {
			if !p.IsWorkspaceRoot {
				t.Errorf("Root project should be marked as workspace root")
			}
			if p.WorkspaceRoot != "" {
				t.Errorf("Workspace root should not have WorkspaceRoot field set, got %s", p.WorkspaceRoot)
			}
			idx := i
			workspaceRoot = &idx
		}
	}
	
	if workspaceRoot == nil {
		t.Fatal("Workspace root not found in detected projects")
	}
	
	// Verify child packages
	foundApi := false
	foundWebapp := false
	
	for _, p := range projects {
		baseName := filepath.Base(p.Dir)
		
		if baseName == "api" {
			foundApi = true
			if p.IsWorkspaceRoot {
				t.Errorf("API package should not be marked as workspace root")
			}
			absWorkspaceDir, _ := filepath.Abs(workspaceDir)
			if p.WorkspaceRoot != absWorkspaceDir {
				t.Errorf("API package should have WorkspaceRoot=%s, got %s", absWorkspaceDir, p.WorkspaceRoot)
			}
		}
		
		if baseName == "webapp" {
			foundWebapp = true
			if p.IsWorkspaceRoot {
				t.Errorf("Webapp package should not be marked as workspace root")
			}
			absWorkspaceDir, _ := filepath.Abs(workspaceDir)
			if p.WorkspaceRoot != absWorkspaceDir {
				t.Errorf("Webapp package should have WorkspaceRoot=%s, got %s", absWorkspaceDir, p.WorkspaceRoot)
			}
		}
	}
	
	if !foundApi {
		t.Error("API package not found in detected projects")
	}
	if !foundWebapp {
		t.Error("Webapp package not found in detected projects")
	}
	
	// Verify all use npm package manager
	for _, p := range projects {
		if p.PackageManager != "npm" {
			t.Errorf("Expected npm package manager, got %s for %s", p.PackageManager, p.Dir)
		}
	}
}

// TestNpmWorkspaceHasWorkspaces verifies the workspace detection function.
func TestNpmWorkspaceHasWorkspaces(t *testing.T) {
	workspaceDir := filepath.Join("..", "..", "..", "tests", "projects", "node", "test-npm-workspace")
	
	// Convert to absolute path
	absWorkspaceDir, err := filepath.Abs(workspaceDir)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	
	if _, err := os.Stat(absWorkspaceDir); os.IsNotExist(err) {
		t.Skipf("Test workspace project not found at %s", absWorkspaceDir)
	}
	
	// Test root workspace
	if !HasNpmWorkspaces(absWorkspaceDir) {
		t.Errorf("Root project at %s should have workspaces defined", absWorkspaceDir)
	}
	
	// Test child packages (should not have workspaces)
	apiDir := filepath.Join(absWorkspaceDir, "packages", "api")
	if HasNpmWorkspaces(apiDir) {
		t.Error("API package should not have workspaces defined")
	}
	
	webappDir := filepath.Join(absWorkspaceDir, "packages", "webapp")
	if HasNpmWorkspaces(webappDir) {
		t.Error("Webapp package should not have workspaces defined")
	}
}

// TestPnpmWorkspaceIntegration tests the pnpm workspace test project.
// This validates pnpm-workspace.yaml detection and complete workflow.
func TestPnpmWorkspaceIntegration(t *testing.T) {
	// Path to test pnpm workspace project
	workspaceDir := filepath.Join("..", "..", "..", "tests", "projects", "node", "test-pnpm-workspace")
	
	// Convert to absolute path
	absWorkspaceDir, err := filepath.Abs(workspaceDir)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	
	// Check if test project exists
	if _, err := os.Stat(absWorkspaceDir); os.IsNotExist(err) {
		t.Skipf("Test pnpm workspace project not found at %s", absWorkspaceDir)
	}
	
	// Find all Node.js projects
	projects, err := FindNodeProjects(absWorkspaceDir)
	if err != nil {
		t.Fatalf("FindNodeProjects failed: %v", err)
	}
	
	// Should find exactly 3 projects: root + 2 children
	if len(projects) != 3 {
		t.Fatalf("Expected 3 projects (root + api + webapp), found %d", len(projects))
	}
	
	// Verify workspace root
	var workspaceRoot *int
	for i, p := range projects {
		if p.Dir == absWorkspaceDir {
			if !p.IsWorkspaceRoot {
				t.Errorf("Root project should be marked as workspace root (detected via pnpm-workspace.yaml)")
			}
			if p.WorkspaceRoot != "" {
				t.Errorf("Workspace root should not have WorkspaceRoot field set, got %s", p.WorkspaceRoot)
			}
			if p.PackageManager != "pnpm" {
				t.Errorf("Expected pnpm package manager, got %s", p.PackageManager)
			}
			idx := i
			workspaceRoot = &idx
		}
	}
	
	if workspaceRoot == nil {
		t.Fatal("pnpm workspace root not found in detected projects")
	}
	
	// Verify child packages
	foundApi := false
	foundWebapp := false
	
	for _, p := range projects {
		baseName := filepath.Base(p.Dir)
		
		if baseName == "api" {
			foundApi = true
			if p.IsWorkspaceRoot {
				t.Errorf("API package should not be marked as workspace root")
			}
			if p.WorkspaceRoot != absWorkspaceDir {
				t.Errorf("API package should have WorkspaceRoot=%s, got %s", absWorkspaceDir, p.WorkspaceRoot)
			}
		}
		
		if baseName == "webapp" {
			foundWebapp = true
			if p.IsWorkspaceRoot {
				t.Errorf("Webapp package should not be marked as workspace root")
			}
			if p.WorkspaceRoot != absWorkspaceDir {
				t.Errorf("Webapp package should have WorkspaceRoot=%s, got %s", absWorkspaceDir, p.WorkspaceRoot)
			}
		}
	}
	
	if !foundApi {
		t.Error("API package not found in detected projects")
	}
	if !foundWebapp {
		t.Error("Webapp package not found in detected projects")
	}
}

// TestPnpmWorkspaceHasWorkspaces verifies pnpm-workspace.yaml detection.
func TestPnpmWorkspaceHasWorkspaces(t *testing.T) {
	workspaceDir := filepath.Join("..", "..", "..", "tests", "projects", "node", "test-pnpm-workspace")
	
	// Convert to absolute path
	absWorkspaceDir, err := filepath.Abs(workspaceDir)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	
	if _, err := os.Stat(absWorkspaceDir); os.IsNotExist(err) {
		t.Skipf("Test pnpm workspace project not found at %s", absWorkspaceDir)
	}
	
	// Test root workspace (should detect pnpm-workspace.yaml)
	if !HasNpmWorkspaces(absWorkspaceDir) {
		t.Errorf("Root project at %s should have pnpm-workspace.yaml detected", absWorkspaceDir)
	}
	
	// Test child packages (should not have workspaces)
	apiDir := filepath.Join(absWorkspaceDir, "packages", "api")
	if HasNpmWorkspaces(apiDir) {
		t.Error("API package should not have workspaces defined")
	}
	
	webappDir := filepath.Join(absWorkspaceDir, "packages", "webapp")
	if HasNpmWorkspaces(webappDir) {
		t.Error("Webapp package should not have workspaces defined")
	}
}
