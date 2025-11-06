//go:build integration

package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMergeReqsPreservesStructure verifies that merging reqs preserves file structure and order.
func TestMergeReqsPreservesStructure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir, err := os.MkdirTemp("", "test-preserve-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a Node.js project
	packageJson := `{"name": "test", "version": "1.0.0"}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJson), 0600); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "package-lock.json"), []byte("{}"), 0600); err != nil {
		t.Fatalf("Failed to create package-lock.json: %v", err)
	}

	// Create azure.yaml with services before reqs
	originalYaml := `name: test-project
services:
  api:
    host: localhost
    port: 3000
    language: js
reqs:
  - name: docker
    minVersion: "20.0.0"
`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(originalYaml), 0600); err != nil {
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

	// Verify structure order is preserved: name -> services -> reqs
	nameIdx := strings.Index(contentStr, "name:")
	servicesIdx := strings.Index(contentStr, "services:")
	reqsIdx := strings.Index(contentStr, "reqs:")

	if nameIdx == -1 || servicesIdx == -1 || reqsIdx == -1 {
		t.Fatal("Missing required sections in azure.yaml")
	}

	if !(nameIdx < servicesIdx && servicesIdx < reqsIdx) {
		t.Error("Section order not preserved: expected name < services < reqs")
		t.Logf("Indices: name=%d, services=%d, reqs=%d", nameIdx, servicesIdx, reqsIdx)
	}

	// Verify services section is intact
	if !strings.Contains(contentStr, "api:") {
		t.Error("Service 'api' not found")
	}
	if !strings.Contains(contentStr, "host: localhost") {
		t.Error("Service host not preserved")
	}

	// Verify docker req was preserved
	if !strings.Contains(contentStr, "- name: docker") {
		t.Error("Existing docker requirement was not preserved")
	}

	// Verify new reqs were added
	if !strings.Contains(contentStr, "- name: node") {
		t.Error("node requirement should have been added")
	}
	if !strings.Contains(contentStr, "- name: npm") {
		t.Error("npm requirement should have been added")
	}
}

// TestMergeReqsNoDuplicates verifies that existing reqs are not duplicated.
func TestMergeReqsNoDuplicates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir, err := os.MkdirTemp("", "test-nodup-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a Node.js project
	packageJson := `{"name": "test", "version": "1.0.0"}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJson), 0600); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "package-lock.json"), []byte("{}"), 0600); err != nil {
		t.Fatalf("Failed to create package-lock.json: %v", err)
	}

	// Create azure.yaml with node already in reqs
	existingYaml := `name: test
reqs:
  - name: node
    minVersion: "18.0.0"
  - name: docker
    minVersion: "20.0.0"
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

	// Count occurrences of "- name: node" - should only be 1
	nodeCount := strings.Count(contentStr, "- name: node")
	if nodeCount != 1 {
		t.Errorf("Expected exactly 1 '- name: node', found %d", nodeCount)
	}

	// Count occurrences of "- name: docker" - should only be 1
	dockerCount := strings.Count(contentStr, "- name: docker")
	if dockerCount != 1 {
		t.Errorf("Expected exactly 1 '- name: docker', found %d", dockerCount)
	}

	// Verify npm was added (and only once)
	npmCount := strings.Count(contentStr, "- name: npm")
	if npmCount != 1 {
		t.Errorf("Expected exactly 1 '- name: npm', found %d", npmCount)
	}
}

// TestMergeReqsNoReqsSection verifies creating reqs section when it doesn't exist.
func TestMergeReqsNoReqsSection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir, err := os.MkdirTemp("", "test-noreqs-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a Node.js project
	packageJson := `{"name": "test", "version": "1.0.0"}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJson), 0600); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "package-lock.json"), []byte("{}"), 0600); err != nil {
		t.Fatalf("Failed to create package-lock.json: %v", err)
	}

	// Create azure.yaml WITHOUT reqs section
	existingYaml := `name: test
services:
  api:
    host: localhost
    port: 3000
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

	// Verify reqs section was created
	if !strings.Contains(contentStr, "reqs:") {
		t.Error("reqs section was not created")
	}

	// Verify only ONE reqs section exists
	reqsCount := strings.Count(contentStr, "reqs:")
	if reqsCount != 1 {
		t.Errorf("Expected exactly 1 'reqs:' section, found %d", reqsCount)
	}

	// Verify new reqs were added
	if !strings.Contains(contentStr, "- name: node") {
		t.Error("node requirement should have been added")
	}
	if !strings.Contains(contentStr, "- name: npm") {
		t.Error("npm requirement should have been added")
	}

	// Verify services section is still there
	if !strings.Contains(contentStr, "services:") {
		t.Error("services section was lost")
	}
}

// TestMergeReqsUserAddedCustomReq verifies that user-added custom reqs are preserved.
func TestMergeReqsUserAddedCustomReq(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir, err := os.MkdirTemp("", "test-custom-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a Node.js project
	packageJson := `{"name": "test", "version": "1.0.0"}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJson), 0600); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "package-lock.json"), []byte("{}"), 0600); err != nil {
		t.Fatalf("Failed to create package-lock.json: %v", err)
	}

	// Create azure.yaml with user-added custom requirements
	existingYaml := `name: test
reqs:
  - name: my-custom-tool
    minVersion: "1.0.0"
    command: my-tool
    args: ["--version"]
  - name: another-custom-tool
    minVersion: "2.5.0"
    checkRunning: true
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

	// Verify custom tools were preserved
	if !strings.Contains(contentStr, "- name: my-custom-tool") {
		t.Error("User-added custom tool 'my-custom-tool' was not preserved")
	}
	if !strings.Contains(contentStr, "- name: another-custom-tool") {
		t.Error("User-added custom tool 'another-custom-tool' was not preserved")
	}

	// Verify new detected reqs were added
	if !strings.Contains(contentStr, "- name: node") {
		t.Error("node requirement should have been added")
	}
	if !strings.Contains(contentStr, "- name: npm") {
		t.Error("npm requirement should have been added")
	}

	// Verify no duplicates
	customCount := strings.Count(contentStr, "- name: my-custom-tool")
	if customCount != 1 {
		t.Errorf("Expected exactly 1 '- name: my-custom-tool', found %d", customCount)
	}
}

// TestMergeReqsMultipleRuns verifies idempotency - running multiple times doesn't create duplicates.
func TestMergeReqsMultipleRuns(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir, err := os.MkdirTemp("", "test-multi-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a Node.js project
	packageJson := `{"name": "test", "version": "1.0.0"}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJson), 0600); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "package-lock.json"), []byte("{}"), 0600); err != nil {
		t.Fatalf("Failed to create package-lock.json: %v", err)
	}

	// Create initial azure.yaml
	existingYaml := `name: test
reqs:
  - name: docker
    minVersion: "20.0.0"
`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(existingYaml), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	config := GenerateConfig{
		DryRun:     false,
		WorkingDir: tmpDir,
	}

	// Run generation FIRST time
	err = runGenerate(config)
	if err != nil {
		t.Fatalf("First runGenerate failed: %v", err)
	}

	// Read after first run
	content1, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to read azure.yaml after first run: %v", err)
	}

	// Run generation SECOND time
	err = runGenerate(config)
	if err != nil {
		t.Fatalf("Second runGenerate failed: %v", err)
	}

	// Read after second run
	content2, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to read azure.yaml after second run: %v", err)
	}

	// Content should be identical after second run
	if string(content1) != string(content2) {
		t.Error("File content changed after second run (not idempotent)")
		t.Logf("After first run:\n%s", string(content1))
		t.Logf("After second run:\n%s", string(content2))
	}

	// Verify no duplicates
	contentStr := string(content2)
	nodeCount := strings.Count(contentStr, "- name: node")
	npmCount := strings.Count(contentStr, "- name: npm")
	dockerCount := strings.Count(contentStr, "- name: docker")

	if nodeCount != 1 {
		t.Errorf("Expected exactly 1 '- name: node', found %d", nodeCount)
	}
	if npmCount != 1 {
		t.Errorf("Expected exactly 1 '- name: npm', found %d", npmCount)
	}
	if dockerCount != 1 {
		t.Errorf("Expected exactly 1 '- name: docker', found %d", dockerCount)
	}
}
