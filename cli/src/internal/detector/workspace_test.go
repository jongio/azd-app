package detector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasNpmWorkspaces(t *testing.T) {
	tests := []struct {
		name           string
		packageJSON    string
		expectedResult bool
	}{
		{
			name: "workspaces as array",
			packageJSON: `{
				"name": "my-workspace",
				"workspaces": ["packages/*"]
			}`,
			expectedResult: true,
		},
		{
			name: "workspaces as object with packages",
			packageJSON: `{
				"name": "my-workspace",
				"workspaces": {
					"packages": ["packages/*"]
				}
			}`,
			expectedResult: true,
		},
		{
			name: "empty workspaces array",
			packageJSON: `{
				"name": "my-workspace",
				"workspaces": []
			}`,
			expectedResult: false,
		},
		{
			name: "no workspaces field",
			packageJSON: `{
				"name": "my-project",
				"dependencies": {}
			}`,
			expectedResult: false,
		},
		{
			name: "workspaces null",
			packageJSON: `{
				"name": "my-project",
				"workspaces": null
			}`,
			expectedResult: false,
		},
		{
			name: "invalid JSON",
			packageJSON: `{
				"name": "my-project"
				"workspaces": ["packages/*"]
			}`,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()

			// Write package.json
			packageJSONPath := filepath.Join(tmpDir, "package.json")
			if err := os.WriteFile(packageJSONPath, []byte(tt.packageJSON), 0600); err != nil {
				t.Fatalf("Failed to write package.json: %v", err)
			}

			// Test HasNpmWorkspaces
			result := HasNpmWorkspaces(tmpDir)
			if result != tt.expectedResult {
				t.Errorf("HasNpmWorkspaces() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestFindNodeProjects_Workspaces(t *testing.T) {
	// Create temp directory structure:
	// workspace/
	//   package.json (with workspaces)
	//   packages/
	//     api/
	//       package.json
	//     webapp/
	//       package.json

	tmpDir := t.TempDir()

	// Create workspace root
	workspacePackageJSON := `{
		"name": "my-workspace",
		"workspaces": ["packages/*"]
	}`
	workspacePackageJSONPath := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(workspacePackageJSONPath, []byte(workspacePackageJSON), 0600); err != nil {
		t.Fatalf("Failed to create workspace package.json: %v", err)
	}

	// Create packages directory
	packagesDir := filepath.Join(tmpDir, "packages")
	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		t.Fatalf("Failed to create packages directory: %v", err)
	}

	// Create api package
	apiDir := filepath.Join(packagesDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("Failed to create api directory: %v", err)
	}
	apiPackageJSON := `{
		"name": "@workspace/api",
		"version": "1.0.0"
	}`
	if err := os.WriteFile(filepath.Join(apiDir, "package.json"), []byte(apiPackageJSON), 0600); err != nil {
		t.Fatalf("Failed to create api package.json: %v", err)
	}

	// Create webapp package
	webappDir := filepath.Join(packagesDir, "webapp")
	if err := os.MkdirAll(webappDir, 0755); err != nil {
		t.Fatalf("Failed to create webapp directory: %v", err)
	}
	webappPackageJSON := `{
		"name": "@workspace/webapp",
		"version": "1.0.0"
	}`
	if err := os.WriteFile(filepath.Join(webappDir, "package.json"), []byte(webappPackageJSON), 0600); err != nil {
		t.Fatalf("Failed to create webapp package.json: %v", err)
	}

	// Run FindNodeProjects
	projects, err := FindNodeProjects(tmpDir)
	if err != nil {
		t.Fatalf("FindNodeProjects failed: %v", err)
	}

	// Verify we found 3 projects
	if len(projects) != 3 {
		t.Fatalf("Expected 3 projects, found %d", len(projects))
	}

	// Verify workspace root is marked correctly
	var workspaceRoot *int
	for i, project := range projects {
		if project.Dir == tmpDir {
			if !project.IsWorkspaceRoot {
				t.Errorf("Workspace root should be marked as IsWorkspaceRoot")
			}
			if project.WorkspaceRoot != "" {
				t.Errorf("Workspace root should not have WorkspaceRoot set, got %s", project.WorkspaceRoot)
			}
			idx := i
			workspaceRoot = &idx
		}
	}

	if workspaceRoot == nil {
		t.Fatal("Workspace root not found in projects")
	}

	// Verify child packages are linked to workspace root
	for _, project := range projects {
		if project.Dir == apiDir || project.Dir == webappDir {
			if project.IsWorkspaceRoot {
				t.Errorf("Child project %s should not be marked as workspace root", project.Dir)
			}
			if project.WorkspaceRoot != tmpDir {
				t.Errorf("Child project %s should have WorkspaceRoot=%s, got %s", project.Dir, tmpDir, project.WorkspaceRoot)
			}
		}
	}
}

func TestFindNodeProjects_NoWorkspaces(t *testing.T) {
	// Create temp directory structure with independent projects
	tmpDir := t.TempDir()

	// Create project1
	project1Dir := filepath.Join(tmpDir, "project1")
	if err := os.MkdirAll(project1Dir, 0755); err != nil {
		t.Fatalf("Failed to create project1: %v", err)
	}
	project1PackageJSON := `{
		"name": "project1",
		"version": "1.0.0"
	}`
	if err := os.WriteFile(filepath.Join(project1Dir, "package.json"), []byte(project1PackageJSON), 0600); err != nil {
		t.Fatalf("Failed to create project1 package.json: %v", err)
	}

	// Create project2
	project2Dir := filepath.Join(tmpDir, "project2")
	if err := os.MkdirAll(project2Dir, 0755); err != nil {
		t.Fatalf("Failed to create project2: %v", err)
	}
	project2PackageJSON := `{
		"name": "project2",
		"version": "1.0.0"
	}`
	if err := os.WriteFile(filepath.Join(project2Dir, "package.json"), []byte(project2PackageJSON), 0600); err != nil {
		t.Fatalf("Failed to create project2 package.json: %v", err)
	}

	// Run FindNodeProjects
	projects, err := FindNodeProjects(tmpDir)
	if err != nil {
		t.Fatalf("FindNodeProjects failed: %v", err)
	}

	// Verify we found 2 projects
	if len(projects) != 2 {
		t.Fatalf("Expected 2 projects, found %d", len(projects))
	}

	// Verify neither is a workspace root
	for _, project := range projects {
		if project.IsWorkspaceRoot {
			t.Errorf("Project %s should not be marked as workspace root", project.Dir)
		}
		if project.WorkspaceRoot != "" {
			t.Errorf("Independent project %s should not have WorkspaceRoot set, got %s", project.Dir, project.WorkspaceRoot)
		}
	}
}

func TestFindNodeProjects_YarnWorkspaces(t *testing.T) {
	// Test Yarn workspaces format
	tmpDir := t.TempDir()

	// Create workspace root with Yarn workspaces
	workspacePackageJSON := `{
		"name": "yarn-workspace",
		"private": true,
		"workspaces": {
			"packages": ["packages/*"]
		}
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(workspacePackageJSON), 0600); err != nil {
		t.Fatalf("Failed to create workspace package.json: %v", err)
	}

	// Create a package
	pkgDir := filepath.Join(tmpDir, "packages", "pkg1")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("Failed to create package directory: %v", err)
	}
	pkgPackageJSON := `{
		"name": "pkg1",
		"version": "1.0.0"
	}`
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(pkgPackageJSON), 0600); err != nil {
		t.Fatalf("Failed to create package package.json: %v", err)
	}

	// Run FindNodeProjects
	projects, err := FindNodeProjects(tmpDir)
	if err != nil {
		t.Fatalf("FindNodeProjects failed: %v", err)
	}

	// Should find 2 projects (workspace root + child)
	if len(projects) != 2 {
		t.Fatalf("Expected 2 projects, found %d", len(projects))
	}

	// Verify workspace root detection
	foundWorkspaceRoot := false
	for _, project := range projects {
		if project.Dir == tmpDir {
			if !project.IsWorkspaceRoot {
				t.Errorf("Should detect Yarn workspaces object format")
			}
			foundWorkspaceRoot = true
		}
	}

	if !foundWorkspaceRoot {
		t.Error("Failed to find workspace root")
	}
}

func TestHasNpmWorkspaces_PnpmWorkspaceYaml(t *testing.T) {
	tests := []struct {
		name            string
		pnpmWorkspaceYAML string
		packageJSON     string
		expectedResult  bool
	}{
		{
			name: "pnpm-workspace.yaml exists",
			pnpmWorkspaceYAML: `packages:
  - 'packages/*'
  - 'apps/*'
`,
			packageJSON: `{
				"name": "pnpm-workspace",
				"version": "1.0.0"
			}`,
			expectedResult: true,
		},
		{
			name: "pnpm-workspace.yaml with package.json workspaces",
			pnpmWorkspaceYAML: `packages:
  - 'packages/*'
`,
			packageJSON: `{
				"name": "pnpm-workspace",
				"workspaces": ["packages/*"]
			}`,
			expectedResult: true,
		},
		{
			name:              "only package.json workspaces (no pnpm-workspace.yaml)",
			pnpmWorkspaceYAML: "",
			packageJSON: `{
				"name": "npm-workspace",
				"workspaces": ["packages/*"]
			}`,
			expectedResult: true,
		},
		{
			name:              "no workspace configuration",
			pnpmWorkspaceYAML: "",
			packageJSON: `{
				"name": "regular-project",
				"version": "1.0.0"
			}`,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Write pnpm-workspace.yaml if provided
			if tt.pnpmWorkspaceYAML != "" {
				pnpmWorkspacePath := filepath.Join(tmpDir, "pnpm-workspace.yaml")
				if err := os.WriteFile(pnpmWorkspacePath, []byte(tt.pnpmWorkspaceYAML), 0600); err != nil {
					t.Fatalf("Failed to write pnpm-workspace.yaml: %v", err)
				}
			}

			// Write package.json
			packageJSONPath := filepath.Join(tmpDir, "package.json")
			if err := os.WriteFile(packageJSONPath, []byte(tt.packageJSON), 0600); err != nil {
				t.Fatalf("Failed to write package.json: %v", err)
			}

			// Test HasNpmWorkspaces
			result := HasNpmWorkspaces(tmpDir)
			if result != tt.expectedResult {
				t.Errorf("HasNpmWorkspaces() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestFindNodeProjects_PnpmWorkspaces(t *testing.T) {
	// Test pnpm workspace detection with pnpm-workspace.yaml
	tmpDir := t.TempDir()

	// Create pnpm-workspace.yaml
	pnpmWorkspaceYAML := `packages:
  - 'packages/*'
`
	if err := os.WriteFile(filepath.Join(tmpDir, "pnpm-workspace.yaml"), []byte(pnpmWorkspaceYAML), 0600); err != nil {
		t.Fatalf("Failed to create pnpm-workspace.yaml: %v", err)
	}

	// Create workspace root package.json
	workspacePackageJSON := `{
		"name": "pnpm-workspace",
		"private": true
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(workspacePackageJSON), 0600); err != nil {
		t.Fatalf("Failed to create workspace package.json: %v", err)
	}

	// Create packages directory
	packagesDir := filepath.Join(tmpDir, "packages")
	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		t.Fatalf("Failed to create packages directory: %v", err)
	}

	// Create package
	pkg1Dir := filepath.Join(packagesDir, "pkg1")
	if err := os.MkdirAll(pkg1Dir, 0755); err != nil {
		t.Fatalf("Failed to create pkg1 directory: %v", err)
	}
	pkg1PackageJSON := `{
		"name": "@workspace/pkg1",
		"version": "1.0.0"
	}`
	if err := os.WriteFile(filepath.Join(pkg1Dir, "package.json"), []byte(pkg1PackageJSON), 0600); err != nil {
		t.Fatalf("Failed to create pkg1 package.json: %v", err)
	}

	// Run FindNodeProjects
	projects, err := FindNodeProjects(tmpDir)
	if err != nil {
		t.Fatalf("FindNodeProjects failed: %v", err)
	}

	// Should find 2 projects (workspace root + child)
	if len(projects) != 2 {
		t.Fatalf("Expected 2 projects, found %d", len(projects))
	}

	// Verify workspace root is detected via pnpm-workspace.yaml
	foundWorkspaceRoot := false
	for _, project := range projects {
		if project.Dir == tmpDir {
			if !project.IsWorkspaceRoot {
				t.Errorf("Should detect pnpm workspace via pnpm-workspace.yaml")
			}
			foundWorkspaceRoot = true
		}
	}

	if !foundWorkspaceRoot {
		t.Error("Failed to find pnpm workspace root")
	}

	// Verify child is linked to workspace root
	for _, project := range projects {
		if project.Dir == pkg1Dir {
			if project.WorkspaceRoot != tmpDir {
				t.Errorf("Child project should have WorkspaceRoot=%s, got %s", tmpDir, project.WorkspaceRoot)
			}
		}
	}
}
