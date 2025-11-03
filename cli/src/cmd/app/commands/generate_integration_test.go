//go:build integration

package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name         string
		projectPath  string
		expectedReqs []string
		minReqs      int // Minimum number of requirements expected
	}{
		{
			name:         "Node.js npm project",
			projectPath:  "../../../../tests/projects/node/test-npm-project",
			expectedReqs: []string{"node", "npm"},
			minReqs:      2,
		},
		{
			name:         "Node.js pnpm project",
			projectPath:  "../../../../tests/projects/node/test-pnpm-project",
			expectedReqs: []string{"node", "pnpm"},
			minReqs:      2,
		},
		{
			name:         "Python project",
			projectPath:  "../../../../tests/projects/python/test-python-project",
			expectedReqs: []string{"python", "pip"},
			minReqs:      2,
		},
		{
			name:         "Python poetry project",
			projectPath:  "../../../../tests/projects/python/test-poetry-project",
			expectedReqs: []string{"python", "poetry"},
			minReqs:      2,
		},
		{
			name:         "Python uv project",
			projectPath:  "../../../../tests/projects/python/test-uv-project",
			expectedReqs: []string{"python", "uv"},
			minReqs:      2,
		},
		{
			name:         "Aspire project",
			projectPath:  "../../../../tests/projects/aspire-test",
			expectedReqs: []string{"dotnet", "aspire"},
			minReqs:      2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get absolute path
			absPath, err := filepath.Abs(tt.projectPath)
			if err != nil {
				t.Fatalf("Failed to get absolute path: %v", err)
			}

			// Skip if project directory doesn't exist
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				t.Skipf("Project directory does not exist: %s", absPath)
			}

			// Clean up azure.yaml if it exists
			azureYamlPath := filepath.Join(absPath, "azure.yaml")
			os.Remove(azureYamlPath)
			defer os.Remove(azureYamlPath)

			// Run generation
			config := GenerateConfig{
				DryRun:     false,
				WorkingDir: absPath,
			}

			err = runGenerate(config)
			if err != nil {
				t.Fatalf("runGenerate failed: %v", err)
			}

			// Verify azure.yaml was created
			if _, err := os.Stat(azureYamlPath); os.IsNotExist(err) {
				t.Fatalf("azure.yaml was not created")
			}

			// Read azure.yaml content
			content, err := os.ReadFile(azureYamlPath)
			if err != nil {
				t.Fatalf("Failed to read azure.yaml: %v", err)
			}

			contentStr := string(content)

			// Verify expected requirements are present
			for _, req := range tt.expectedReqs {
				if !strings.Contains(contentStr, "id: "+req) {
					t.Errorf("Expected requirement %q not found in azure.yaml", req)
				}
			}

			// Count requirements
			reqCount := strings.Count(contentStr, "id:")
			if reqCount < tt.minReqs {
				t.Errorf("Expected at least %d requirements, found %d", tt.minReqs, reqCount)
			}

			// Verify all requirements have minVersion
			if strings.Count(contentStr, "minVersion:") != reqCount {
				t.Errorf("Not all requirements have minVersion specified")
			}
		})
	}
}

func TestGenerateDryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "test-dryrun-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple Node.js project
	packageJson := `{"name": "test", "version": "1.0.0"}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJson), 0600); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	// Run with dry-run
	config := GenerateConfig{
		DryRun:     true,
		WorkingDir: tmpDir,
	}

	err = runGenerate(config)
	if err != nil {
		t.Fatalf("runGenerate with dry-run failed: %v", err)
	}

	// Verify azure.yaml was NOT created
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if _, err := os.Stat(azureYamlPath); !os.IsNotExist(err) {
		t.Errorf("azure.yaml should not be created in dry-run mode")
	}
}

func TestGenerateMerge(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "test-merge-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a Node.js project
	packageJson := `{"name": "test", "version": "1.0.0"}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJson), 0600); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	// Create existing azure.yaml with custom requirement
	existingYaml := `name: test
reqs:
  - id: custom-tool
    minVersion: 1.0.0
  - id: node
    minVersion: 18.0.0
`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(existingYaml), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	// Run generation
	config := GenerateConfig{
		DryRun:     false,
		WorkingDir: tmpDir,
	}

	err = runGenerate(config)
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	// Read updated azure.yaml
	content, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to read azure.yaml: %v", err)
	}

	contentStr := string(content)

	// Verify custom-tool was preserved
	if !strings.Contains(contentStr, "id: custom-tool") {
		t.Errorf("Existing custom-tool requirement was not preserved")
	}

	// Verify node still exists (should be preserved, not added again)
	nodeCount := strings.Count(contentStr, "id: node")
	if nodeCount != 1 {
		t.Errorf("Expected exactly 1 node requirement, found %d", nodeCount)
	}

	// Verify npm was added
	if !strings.Contains(contentStr, "id: npm") {
		t.Errorf("npm requirement should have been added")
	}
}

func TestGenerateNoProject(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary directory with no project files
	tmpDir, err := os.MkdirTemp("", "test-noproject-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Run generation on empty directory
	config := GenerateConfig{
		DryRun:     false,
		WorkingDir: tmpDir,
	}

	err = runGenerate(config)
	if err == nil {
		t.Errorf("Expected error when no project is detected, got nil")
	}

	// Verify error message mentions no dependencies detected
	if err != nil && !strings.Contains(err.Error(), "no dependencies detected") {
		t.Errorf("Expected 'no dependencies detected' error, got: %v", err)
	}
}
