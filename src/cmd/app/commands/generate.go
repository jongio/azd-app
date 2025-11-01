package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"app/src/internal/detector"
	"app/src/internal/security"

	"gopkg.in/yaml.v3"
)

// DetectedRequirement represents a requirement found during project scanning.
type DetectedRequirement struct {
	ID               string // Tool identifier (e.g., "node", "docker")
	InstalledVersion string // Currently installed version (e.g., "22.3.0")
	MinVersion       string // Normalized minimum version (e.g., "22.0.0")
	CheckRunning     bool   // Whether tool must be running
	Source           string // What triggered detection (e.g., "package.json", "AppHost.cs")
}

// GenerateConfig holds configuration for requirement generation.
type GenerateConfig struct {
	DryRun     bool   // Don't write files, just show what would happen
	WorkingDir string // Directory to start search from
}

// GenerateResult contains the outcome of requirement generation.
type GenerateResult struct {
	Requirements  []DetectedRequirement
	AzureYamlPath string
	Created       bool // True if azure.yaml was created vs updated
	Added         int  // Number of requirements added
	Skipped       int  // Number of existing requirements preserved
}

// runGenerate is the main entry point for the generate command.
func runGenerate(config GenerateConfig) error {
	fmt.Println("🔍 Scanning project for dependencies...")
	fmt.Println()

	// Detect all requirements based on project structure
	requirements, err := detectProjectRequirements(config.WorkingDir)
	if err != nil {
		return fmt.Errorf("failed to detect requirements: %w", err)
	}

	if len(requirements) == 0 {
		fmt.Printf("⚠️  No project dependencies detected in current directory\n")
		fmt.Printf("   Searched: %s\n\n", config.WorkingDir)
		fmt.Println("   Supported project types:")
		fmt.Println("   • Node.js (package.json)")
		fmt.Println("   • Python (requirements.txt, pyproject.toml)")
		fmt.Println("   • .NET (.csproj, .sln)")
		fmt.Println("   • .NET Aspire (AppHost.cs)")
		fmt.Println("   • Docker Compose (docker-compose.yml or package.json scripts)")
		fmt.Println()
		fmt.Println("   Make sure you're in a valid project directory.")
		return fmt.Errorf("no dependencies detected")
	}

	// Display found dependencies
	displayDetectedDependencies(requirements)

	// Display detected requirements with versions
	displayDetectedRequirements(requirements)

	// Find or create azure.yaml
	azureYamlPath, created, err := findOrCreateAzureYaml(config.WorkingDir, config.DryRun)
	if err != nil {
		return fmt.Errorf("failed to find or create azure.yaml: %w", err)
	}

	if config.DryRun {
		fmt.Printf("\nWould update: %s\n", azureYamlPath)
		fmt.Println("\nRun without --dry-run to apply changes.")
		return nil
	}

	// Merge with existing requirements
	added, skipped, err := mergeRequirements(azureYamlPath, requirements)
	if err != nil {
		return fmt.Errorf("failed to merge requirements: %w", err)
	}

	fmt.Println()
	if created {
		fmt.Printf("✅ Created azure.yaml with %d requirements\n", added)
	} else {
		fmt.Printf("✅ Updated azure.yaml with %d requirements\n", added)
		if skipped > 0 {
			fmt.Printf("   (%d existing requirements preserved)\n", skipped)
		}
	}
	fmt.Printf("   Path: %s\n", azureYamlPath)
	fmt.Println()
	fmt.Println("Run 'azd app reqs' to verify all requirements are met.")

	return nil
}

// detectProjectRequirements scans the project directory for all dependencies.
func detectProjectRequirements(projectDir string) ([]DetectedRequirement, error) {
	var requirements []DetectedRequirement
	foundSources := make(map[string]bool)

	// Detect Node.js projects
	if hasPackageJson(projectDir) {
		foundSources["Node.js"] = true

		// Add Node.js
		if req := detectNode(projectDir); req.ID != "" {
			requirements = append(requirements, req)
		}

		// Add package manager
		if req := detectNodePackageManager(projectDir); req.ID != "" {
			requirements = append(requirements, req)
		}
	}

	// Detect Python projects
	if hasPythonProject(projectDir) {
		foundSources["Python"] = true

		// Add Python
		if req := detectPython(projectDir); req.ID != "" {
			requirements = append(requirements, req)
		}

		// Add package manager
		if req := detectPythonPackageManager(projectDir); req.ID != "" {
			requirements = append(requirements, req)
		}
	}

	// Detect .NET projects
	if hasDotnetProject(projectDir) {
		foundSources[".NET"] = true

		// Add .NET SDK
		if req := detectDotnet(projectDir); req.ID != "" {
			requirements = append(requirements, req)
		}

		// Check for Aspire
		if hasAspireProject(projectDir) {
			foundSources[".NET Aspire"] = true
			if req := detectAspire(projectDir); req.ID != "" {
				requirements = append(requirements, req)
			}
		}
	}

	// Detect Docker
	if hasDockerConfig(projectDir) {
		foundSources["Docker"] = true
		if req := detectDocker(projectDir); req.ID != "" {
			requirements = append(requirements, req)
		}
	}

	// Detect Azure tools
	if hasAzureYaml(projectDir) {
		if req := detectAzd(projectDir); req.ID != "" {
			requirements = append(requirements, req)
		}
	}

	// Detect Git
	if hasGit(projectDir) {
		if req := detectGit(projectDir); req.ID != "" {
			requirements = append(requirements, req)
		}
	}

	return requirements, nil
}

// File detection helpers
func hasPackageJson(dir string) bool {
	path := filepath.Join(dir, "package.json")
	if err := security.ValidatePath(path); err != nil {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func hasPythonProject(dir string) bool {
	files := []string{"requirements.txt", "pyproject.toml", "poetry.lock", "uv.lock", "Pipfile"}
	for _, file := range files {
		path := filepath.Join(dir, file)
		if err := security.ValidatePath(path); err != nil {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

func hasDotnetProject(dir string) bool {
	projects, _ := detector.FindDotnetProjects(dir)
	return len(projects) > 0
}

func hasAspireProject(dir string) bool {
	aspireProject, _ := detector.FindAppHost(dir)
	return aspireProject != nil
}

func hasDockerConfig(dir string) bool {
	files := []string{"Dockerfile", "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}
	for _, file := range files {
		path := filepath.Join(dir, file)
		if err := security.ValidatePath(path); err != nil {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	// Check package.json for docker scripts
	if hasPackageJson(dir) {
		if detector.HasDockerComposeScript(dir) {
			return true
		}
	}

	return false
}

func hasAzureYaml(dir string) bool {
	path, _ := detector.FindAzureYaml(dir)
	return path != ""
}

func hasGit(dir string) bool {
	path := filepath.Join(dir, ".git")
	if err := security.ValidatePath(path); err != nil {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// Tool detection functions
func detectNode(projectDir string) DetectedRequirement {
	req := DetectedRequirement{
		ID:     "node",
		Source: "package.json",
	}

	installedVersion, err := getToolVersion("node")
	if err != nil {
		return req
	}

	req.InstalledVersion = installedVersion
	req.MinVersion = normalizeVersion(installedVersion, "node")
	return req
}

func detectNodePackageManager(projectDir string) DetectedRequirement {
	// Priority: pnpm > yarn > npm
	if fileExists(projectDir, "pnpm-lock.yaml") || fileExists(projectDir, "pnpm-workspace.yaml") {
		return detectTool("pnpm", "pnpm-lock.yaml")
	}
	if fileExists(projectDir, "yarn.lock") {
		return detectTool("yarn", "yarn.lock")
	}
	if fileExists(projectDir, "package-lock.json") {
		return detectTool("npm", "package-lock.json")
	}
	// Default to npm
	return detectTool("npm", "package.json")
}

func detectPython(projectDir string) DetectedRequirement {
	req := DetectedRequirement{
		ID:     "python",
		Source: "requirements.txt or pyproject.toml",
	}

	installedVersion, err := getToolVersion("python")
	if err != nil {
		return req
	}

	req.InstalledVersion = installedVersion
	req.MinVersion = normalizeVersion(installedVersion, "python")
	return req
}

func detectPythonPackageManager(projectDir string) DetectedRequirement {
	// Priority: uv > poetry > pipenv > pip
	if fileExists(projectDir, "uv.lock") {
		return detectTool("uv", "uv.lock")
	}

	if fileExists(projectDir, "pyproject.toml") {
		content := readFileContent(filepath.Join(projectDir, "pyproject.toml"))
		if strings.Contains(content, "[tool.uv]") {
			return detectTool("uv", "pyproject.toml")
		}
		if strings.Contains(content, "[tool.poetry]") {
			return detectTool("poetry", "pyproject.toml")
		}
	}

	if fileExists(projectDir, "poetry.lock") {
		return detectTool("poetry", "poetry.lock")
	}

	if fileExists(projectDir, "Pipfile") || fileExists(projectDir, "Pipfile.lock") {
		return detectTool("pipenv", "Pipfile")
	}

	// Default to pip
	return detectTool("pip", "requirements.txt")
}

func detectDotnet(projectDir string) DetectedRequirement {
	req := DetectedRequirement{
		ID:     "dotnet",
		Source: ".csproj or .sln",
	}

	installedVersion, err := getToolVersion("dotnet")
	if err != nil {
		return req
	}

	req.InstalledVersion = installedVersion
	req.MinVersion = normalizeVersion(installedVersion, "dotnet")
	return req
}

func detectAspire(projectDir string) DetectedRequirement {
	req := DetectedRequirement{
		ID:     "aspire",
		Source: "AppHost.cs",
	}

	installedVersion, err := getToolVersion("aspire")
	if err != nil {
		return req
	}

	req.InstalledVersion = installedVersion
	req.MinVersion = normalizeVersion(installedVersion, "aspire")
	return req
}

func detectDocker(projectDir string) DetectedRequirement {
	req := DetectedRequirement{
		ID:           "docker",
		Source:       "Dockerfile or docker-compose.yml",
		CheckRunning: true,
	}

	installedVersion, err := getToolVersion("docker")
	if err != nil {
		return req
	}

	req.InstalledVersion = installedVersion
	req.MinVersion = normalizeVersion(installedVersion, "docker")
	return req
}

func detectAzd(projectDir string) DetectedRequirement {
	req := DetectedRequirement{
		ID:     "azd",
		Source: "azure.yaml",
	}

	installedVersion, err := getToolVersion("azd")
	if err != nil {
		return req
	}

	req.InstalledVersion = installedVersion
	req.MinVersion = normalizeVersion(installedVersion, "azd")
	return req
}

func detectGit(projectDir string) DetectedRequirement {
	req := DetectedRequirement{
		ID:     "git",
		Source: ".git directory",
	}

	installedVersion, err := getToolVersion("git")
	if err != nil {
		return req
	}

	req.InstalledVersion = installedVersion
	req.MinVersion = normalizeVersion(installedVersion, "git")
	return req
}

// detectTool is a generic helper for detecting tools.
func detectTool(toolID, source string) DetectedRequirement {
	req := DetectedRequirement{
		ID:     toolID,
		Source: source,
	}

	installedVersion, err := getToolVersion(toolID)
	if err != nil {
		return req
	}

	req.InstalledVersion = installedVersion
	req.MinVersion = normalizeVersion(installedVersion, toolID)
	return req
}

// getToolVersion queries the system for the installed version of a tool.
func getToolVersion(toolID string) (string, error) {
	// Check aliases first
	if canonical, exists := toolAliases[toolID]; exists {
		toolID = canonical
	}

	// Look up tool configuration from registry
	toolConfig, exists := toolRegistry[toolID]
	if !exists {
		return "", fmt.Errorf("unknown tool: %s", toolID)
	}

	// Execute version command directly to capture output
	// #nosec G204 -- Command and args come from toolRegistry which is a controlled map
	cmd := exec.Command(toolConfig.Command, toolConfig.Args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tool not installed: %s", toolID)
	}

	// Parse version from output
	version := extractVersionFromOutput(string(output), toolConfig.VersionPrefix, toolConfig.VersionField)
	return version, nil
}

// extractVersionFromOutput extracts version from command output.
func extractVersionFromOutput(output, prefix string, field int) string {
	output = strings.TrimSpace(output)

	// Remove prefix if specified
	if prefix != "" {
		output = strings.TrimPrefix(output, prefix)
		output = strings.TrimSpace(output)
	}

	// If field is specified, split and take that field
	if field > 0 {
		parts := strings.Fields(output)
		if field < len(parts) {
			output = parts[field]
		}
	}

	// Clean up version string
	output = strings.TrimSpace(output)

	// Extract just the version number (remove any trailing text)
	// Version should match pattern: X.Y.Z or vX.Y.Z
	versionRegex := regexp.MustCompile(`v?(\d+\.\d+\.\d+)`)
	if matches := versionRegex.FindStringSubmatch(output); len(matches) > 1 {
		return matches[1]
	}

	return output
}

// normalizeVersion converts installed version to minimum version constraint.
func normalizeVersion(installedVersion string, toolID string) string {
	parts := strings.Split(installedVersion, ".")

	switch toolID {
	case "node", "dotnet", "go", "rust", "docker", "git":
		// Major version only: "22.3.0" -> "22.0.0"
		if len(parts) >= 1 {
			return parts[0] + ".0.0"
		}
	case "python":
		// Major.Minor version: "3.12.5" -> "3.12.0"
		if len(parts) >= 2 {
			return parts[0] + "." + parts[1] + ".0"
		}
	case "pnpm", "npm", "yarn", "poetry", "uv", "pip", "pipenv":
		// Major version for package managers: "9.1.4" -> "9.0.0"
		if len(parts) >= 1 {
			return parts[0] + ".0.0"
		}
	case "azd", "az", "aspire":
		// Major.Minor for Azure tools: "1.5.3" -> "1.5.0"
		if len(parts) >= 2 {
			return parts[0] + "." + parts[1] + ".0"
		}
	default:
		// Default: use as-is
		return installedVersion
	}

	return installedVersion
}

// Helper functions
func fileExists(dir, filename string) bool {
	path := filepath.Join(dir, filename)
	if err := security.ValidatePath(path); err != nil {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func readFileContent(path string) string {
	if err := security.ValidatePath(path); err != nil {
		return ""
	}
	//nolint:gosec // G304: Path validated by security.ValidatePath
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// Display functions
func displayDetectedDependencies(requirements []DetectedRequirement) {
	sources := make(map[string]bool)
	for _, req := range requirements {
		// Derive project type from source
		if strings.Contains(req.Source, "package.json") {
			pkgMgr := req.ID
			if req.ID == "node" {
				// Look for package manager in other requirements
				for _, r := range requirements {
					if r.ID == "pnpm" || r.ID == "yarn" || r.ID == "npm" {
						pkgMgr = r.ID
						break
					}
				}
			}
			if req.ID == "node" || req.ID == "npm" || req.ID == "pnpm" || req.ID == "yarn" {
				sources[fmt.Sprintf("Node.js project (%s)", pkgMgr)] = true
			}
		} else if strings.Contains(req.Source, "AppHost.cs") {
			sources[".NET Aspire project"] = true
		} else if strings.Contains(req.Source, ".csproj") || strings.Contains(req.Source, ".sln") {
			if !sources[".NET Aspire project"] {
				sources[".NET project"] = true
			}
		} else if strings.Contains(req.Source, "docker") || strings.Contains(req.Source, "Dockerfile") {
			sources["Docker configuration"] = true
		} else if strings.Contains(req.Source, "requirements.txt") || strings.Contains(req.Source, "pyproject.toml") {
			pkgMgr := req.ID
			if req.ID == "python" {
				for _, r := range requirements {
					if r.ID == "poetry" || r.ID == "uv" || r.ID == "pip" || r.ID == "pipenv" {
						pkgMgr = r.ID
						break
					}
				}
			}
			if req.ID == "python" || req.ID == "pip" || req.ID == "poetry" || req.ID == "uv" || req.ID == "pipenv" {
				sources[fmt.Sprintf("Python project (%s)", pkgMgr)] = true
			}
		}
	}

	fmt.Println("Found:")
	for source := range sources {
		fmt.Printf("  ✓ %s\n", source)
	}
	fmt.Println()
}

func displayDetectedRequirements(requirements []DetectedRequirement) {
	hasUninstalled := false
	installedCount := 0

	fmt.Println("📝 Detected requirements:")
	for _, req := range requirements {
		if req.InstalledVersion != "" {
			installedCount++
			runningNote := ""
			if req.CheckRunning {
				runningNote = ", must be running"
			}
			fmt.Printf("  • %s (%s installed%s) → minVersion: \"%s\"\n",
				req.ID, req.InstalledVersion, runningNote, req.MinVersion)
		} else {
			hasUninstalled = true
			fmt.Printf("  • %s (NOT INSTALLED) → will be added to requirements\n", req.ID)
		}
	}
	fmt.Println()

	if hasUninstalled {
		fmt.Println("⚠️  Some detected dependencies are not installed:")
		fmt.Println()
		for _, req := range requirements {
			if req.InstalledVersion == "" {
				fmt.Printf("  ❌ %s: NOT INSTALLED\n", req.ID)
				if req.ID == "pnpm" {
					fmt.Println("     Install: npm install -g pnpm")
				} else if req.ID == "poetry" {
					fmt.Println("     Install: curl -sSL https://install.python-poetry.org | python3 -")
				} else if req.ID == "uv" {
					fmt.Println("     Install: curl -LsSf https://astral.sh/uv/install.sh | sh")
				}
			}
		}
		fmt.Println()
		fmt.Println("Generating requirements anyway. Run 'azd app reqs' to check status.")
		fmt.Println()
	}
}

// findOrCreateAzureYaml locates or creates azure.yaml file.
func findOrCreateAzureYaml(startDir string, dryRun bool) (string, bool, error) {
	// Try to find existing azure.yaml
	existingPath, err := detector.FindAzureYaml(startDir)
	if err == nil && existingPath != "" {
		return existingPath, false, nil
	}

	// Create new azure.yaml in current directory
	newPath := filepath.Join(startDir, "azure.yaml")
	if err := security.ValidatePath(newPath); err != nil {
		return "", false, fmt.Errorf("invalid path: %w", err)
	}

	if dryRun {
		return newPath, true, nil
	}

	// Create minimal azure.yaml
	dirName := filepath.Base(startDir)
	content := fmt.Sprintf(`# This file was auto-generated by azd app reqs --generate
# Customize as needed for your project

name: %s

# Requirements auto-generated based on detected project dependencies
reqs:
`, dirName)

	//nolint:gosec // G306: File permissions are intentionally 0644 for config files
	if err := os.WriteFile(newPath, []byte(content), 0644); err != nil {
		return "", false, fmt.Errorf("failed to create azure.yaml: %w", err)
	}

	return newPath, true, nil
}

// mergeRequirements merges detected requirements into azure.yaml.
func mergeRequirements(azureYamlPath string, detected []DetectedRequirement) (int, int, error) {
	// Validate path
	if err := security.ValidatePath(azureYamlPath); err != nil {
		return 0, 0, fmt.Errorf("invalid path: %w", err)
	}

	// Read existing azure.yaml
	//nolint:gosec // G304: Path validated by security.ValidatePath
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read azure.yaml: %w", err)
	}

	// Parse YAML
	var azureYaml map[string]interface{}
	if err := yaml.Unmarshal(data, &azureYaml); err != nil {
		return 0, 0, fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	// Get existing requirements
	var existingReqs []Prerequisite
	if reqs, ok := azureYaml["reqs"].([]interface{}); ok {
		for _, r := range reqs {
			if reqMap, ok := r.(map[string]interface{}); ok {
				prereq := Prerequisite{
					ID:         getString(reqMap, "id"),
					MinVersion: getString(reqMap, "minVersion"),
				}
				if reqMap["checkRunning"] != nil {
					prereq.CheckRunning = getBool(reqMap, "checkRunning")
				}
				existingReqs = append(existingReqs, prereq)
			}
		}
	}

	// Build map of existing IDs
	existingIDs := make(map[string]bool)
	for _, req := range existingReqs {
		existingIDs[req.ID] = true
	}

	// Add new requirements
	added := 0
	for _, detected := range detected {
		if !existingIDs[detected.ID] {
			newReq := Prerequisite{
				ID:           detected.ID,
				MinVersion:   detected.MinVersion,
				CheckRunning: detected.CheckRunning,
			}
			existingReqs = append(existingReqs, newReq)
			added++
		}
	}

	// Update azure.yaml
	azureYaml["reqs"] = existingReqs

	// Marshal back to YAML
	output, err := yaml.Marshal(azureYaml)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to marshal yaml: %w", err)
	}

	// Write back to file
	//nolint:gosec // G306: File permissions are intentionally 0644 for config files
	if err := os.WriteFile(azureYamlPath, output, 0644); err != nil {
		return 0, 0, fmt.Errorf("failed to write azure.yaml: %w", err)
	}

	skipped := len(existingIDs)
	return added, skipped, nil
}

// Helper functions for YAML parsing
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return false
}
