// Package commands provides the command-line interface for the azd-app CLI.
package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/security"
	"github.com/jongio/azd-app/cli/src/internal/yamlutil"

	"gopkg.in/yaml.v3"
)

// DetectedRequirement represents a requirement found during project scanning.
type DetectedRequirement struct {
	Name             string // Tool identifier (e.g., "node", "docker")
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

// GenerateResult contains the outcome of reqs generation.
type GenerateResult struct {
	Reqs          []DetectedRequirement
	AzureYamlPath string
	Created       bool // True if azure.yaml was created vs updated
	Added         int  // Number of reqs added
	Skipped       int  // Number of existing reqs preserved
}

// runGenerate is the main entry point for the generate command.
func runGenerate(config GenerateConfig) error {
	output.Section("🔍", "Scanning project for dependencies")

	// Detect all reqs based on project structure
	requirements, err := detectProjectReqs(config.WorkingDir)
	if err != nil {
		return fmt.Errorf("failed to detect reqs: %w", err)
	}

	if len(requirements) == 0 {
		output.Warning("No project dependencies detected in current directory")
		output.Item("Searched: %s", config.WorkingDir)
		output.Newline()
		output.Item("Supported project types:")
		output.Item("  • Node.js (package.json)")
		output.Item("  • Python (requirements.txt, pyproject.toml)")
		output.Item("  • .NET (.csproj, .sln)")
		output.Item("  • .NET Aspire (AppHost.cs)")
		output.Item("  • Docker Compose (docker-compose.yml or package.json scripts)")
		output.Newline()
		output.Item("Make sure you're in a valid project directory.")
		return fmt.Errorf("no dependencies detected")
	}

	// Display found dependencies
	displayDetectedDependencies(requirements)

	// Display detected reqs with versions
	displayDetectedReqs(requirements)

	// Find or create azure.yaml
	azureYamlPath, created, err := findOrCreateAzureYaml(config.WorkingDir, config.DryRun)
	if err != nil {
		return fmt.Errorf("failed to find or create azure.yaml: %w", err)
	}

	if config.DryRun {
		output.Info("Would update: %s", azureYamlPath)
		output.Newline()
		output.Item("Run without --dry-run to apply changes.")
		return nil
	}

	// Merge with existing reqs
	added, skipped, err := mergeReqs(azureYamlPath, requirements)
	if err != nil {
		return fmt.Errorf("failed to merge reqs: %w", err)
	}

	output.Newline()
	if created {
		output.Success("Created azure.yaml with %d reqs", added)
	} else {
		output.Success("Updated azure.yaml with %d reqs", added)
		if skipped > 0 {
			output.Item("(%d existing reqs preserved)", skipped)
		}
	}
	output.Label("Path", azureYamlPath)
	output.Newline()
	output.Item("Run 'azd app reqs' to verify all reqs are met.")

	return nil
}

// detectProjectReqs scans the project directory for all dependencies.
func detectProjectReqs(projectDir string) ([]DetectedRequirement, error) {
	var requirements []DetectedRequirement
	foundSources := make(map[string]bool)

	// Detect Node.js projects
	if hasPackageJSON(projectDir) {
		foundSources["Node.js"] = true

		// Add Node.js
		if req := detectNode(projectDir); req.Name != "" {
			requirements = append(requirements, req)
		}

		// Add package manager
		if req := detectNodePackageManager(projectDir); req.Name != "" {
			requirements = append(requirements, req)
		}
	}

	// Detect Python projects
	if hasPythonProject(projectDir) {
		foundSources["Python"] = true

		// Add Python
		if req := detectPython(projectDir); req.Name != "" {
			requirements = append(requirements, req)
		}

		// Add package manager
		if req := detectPythonPackageManager(projectDir); req.Name != "" {
			requirements = append(requirements, req)
		}
	}

	// Detect .NET projects
	if hasDotnetProject(projectDir) {
		foundSources[".NET"] = true

		// Add .NET SDK
		if req := detectDotnet(projectDir); req.Name != "" {
			requirements = append(requirements, req)
		}

		// Check for Aspire
		if hasAspireProject(projectDir) {
			foundSources[".NET Aspire"] = true
			if req := detectAspire(projectDir); req.Name != "" {
				requirements = append(requirements, req)
			}
		}
	}

	// Detect Docker
	if hasDockerConfig(projectDir) {
		foundSources["Docker"] = true
		if req := detectDocker(projectDir); req.Name != "" {
			requirements = append(requirements, req)
		}
	}

	// Detect Azure tools
	if hasAzureYaml(projectDir) {
		if req := detectAzd(projectDir); req.Name != "" {
			requirements = append(requirements, req)
		}
	}

	// Detect Git
	if hasGit(projectDir) {
		if req := detectGit(projectDir); req.Name != "" {
			requirements = append(requirements, req)
		}
	}

	return requirements, nil
}

// File detection helpers
func hasPackageJSON(dir string) bool {
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
	if hasPackageJSON(dir) {
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
func detectNode(_ string) DetectedRequirement {
	return detectToolWithSource("node", "package.json", false)
}

func detectNodePackageManager(projectDir string) DetectedRequirement {
	// Use detector to get both package manager and source in one call
	info := detector.DetectNodePackageManagerWithSource(projectDir)
	return detectTool(info.Name, info.Source)
}

func detectPython(_ string) DetectedRequirement {
	return detectToolWithSource("python", "requirements.txt or pyproject.toml", false)
}

func detectPythonPackageManager(projectDir string) DetectedRequirement {
	// Use detector to get both package manager and source in one call
	info := detector.DetectPythonPackageManagerWithSource(projectDir)
	return detectTool(info.Name, info.Source)
}

func detectDotnet(_ string) DetectedRequirement {
	return detectToolWithSource("dotnet", ".csproj or .sln", false)
}

func detectAspire(_ string) DetectedRequirement {
	return detectToolWithSource("aspire", "AppHost.cs", false)
}

func detectDocker(_ string) DetectedRequirement {
	return detectToolWithSource("docker", "Dockerfile or docker-compose.yml", true)
}

func detectAzd(_ string) DetectedRequirement {
	return detectToolWithSource("azd", "azure.yaml", false)
}

func detectGit(_ string) DetectedRequirement {
	return detectToolWithSource("git", ".git directory", false)
}

// detectToolWithSource is a helper for detecting tools with specified source and checkRunning flag.
func detectToolWithSource(toolName, source string, checkRunning bool) DetectedRequirement {
	req := DetectedRequirement{
		Name:         toolName,
		Source:       source,
		CheckRunning: checkRunning,
	}

	installedVersion, err := getToolVersion(toolName)
	if err != nil {
		return req
	}

	req.InstalledVersion = installedVersion
	req.MinVersion = normalizeVersion(installedVersion, toolName)
	return req
}

// detectTool is a generic helper for detecting tools.
func detectTool(toolName, source string) DetectedRequirement {
	req := DetectedRequirement{
		Name:   toolName,
		Source: source,
	}

	installedVersion, err := getToolVersion(toolName)
	if err != nil {
		return req
	}

	req.InstalledVersion = installedVersion
	req.MinVersion = normalizeVersion(installedVersion, toolName)
	return req
}

// getToolVersion queries the system for the installed version of a tool.
func getToolVersion(toolName string) (string, error) {
	// Check aliases first
	if canonical, exists := toolAliases[toolName]; exists {
		toolName = canonical
	}

	// Look up tool configuration from registry
	toolConfig, exists := toolRegistry[toolName]
	if !exists {
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}

	// Execute version command directly to capture output
	// #nosec G204 -- Command and args come from toolRegistry which is a controlled map
	cmd := exec.Command(toolConfig.Command, toolConfig.Args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tool not installed: %s", toolName)
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
func normalizeVersion(installedVersion string, toolName string) string {
	parts := strings.Split(installedVersion, ".")

	switch toolName {
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
	// Validate the filename first to prevent path traversal before joining
	if err := security.ValidatePath(filename); err != nil {
		return false
	}
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
	// #nosec G304 -- Path validated by security.ValidatePath
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
			pkgMgr := req.Name
			if req.Name == "node" {
				// Look for package manager in other requirements
				for _, r := range requirements {
					if r.Name == "pnpm" || r.Name == "yarn" || r.Name == "npm" {
						pkgMgr = r.Name
						break
					}
				}
			}
			if req.Name == "node" || req.Name == "npm" || req.Name == "pnpm" || req.Name == "yarn" {
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
			pkgMgr := req.Name
			if req.Name == "python" {
				for _, r := range requirements {
					if r.Name == "poetry" || r.Name == "uv" || r.Name == "pip" || r.Name == "pipenv" {
						pkgMgr = r.Name
						break
					}
				}
			}
			if req.Name == "python" || req.Name == "pip" || req.Name == "poetry" || req.Name == "uv" || req.Name == "pipenv" {
				sources[fmt.Sprintf("Python project (%s)", pkgMgr)] = true
			}
		}
	}

	output.Item("Found:")
	for source := range sources {
		output.ItemSuccess("%s", source)
	}
	output.Newline()
}

func displayDetectedReqs(reqs []DetectedRequirement) {
	hasUninstalled := false
	installedCount := 0

	output.Section("📝", "Detected reqs")
	for _, req := range reqs {
		if req.InstalledVersion != "" {
			installedCount++
			runningNote := ""
			if req.CheckRunning {
				runningNote = ", must be running"
			}
			output.Item("%s (%s installed%s) → minVersion: \"%s\"",
				req.Name, req.InstalledVersion, runningNote, req.MinVersion)
		} else {
			hasUninstalled = true
			output.Item("%s (NOT INSTALLED) → will be added to reqs", req.Name)
		}
	}
	output.Newline()

	if hasUninstalled {
		output.Warning("Some detected dependencies are not installed:")
		output.Newline()
		for _, req := range reqs {
			if req.InstalledVersion == "" {
				output.ItemError("%s: NOT INSTALLED", req.Name)
				switch req.Name {
				case "pnpm":
					output.Item("     Install: npm install -g pnpm")
				case "poetry":
					output.Item("     Install: curl -sSL https://install.python-poetry.org | python3 -")
				case "uv":
					output.Item("     Install: curl -LsSf https://astral.sh/uv/install.sh | sh")
				}
			}
		}
		output.Newline()
		output.Item("Generating requirements anyway. Run 'azd app reqs' to check status.")
		output.Newline()
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

	// #nosec G306 -- azure.yaml is a config file, 0644 is appropriate for team access
	if err := os.WriteFile(newPath, []byte(content), 0644); err != nil {
		return "", false, fmt.Errorf("failed to create azure.yaml: %w", err)
	}

	return newPath, true, nil
}

// mergeReqs merges detected reqs into azure.yaml using text-based manipulation
// to ensure no comments, formatting, or other content is lost.
func mergeReqs(azureYamlPath string, detected []DetectedRequirement) (int, int, error) {
	// Validate path
	if err := security.ValidatePath(azureYamlPath); err != nil {
		return 0, 0, fmt.Errorf("invalid path: %w", err)
	}

	// Read existing azure.yaml as text
	// #nosec G304 -- Path validated by security.ValidatePath
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read azure.yaml: %w", err)
	}

	content := string(data)

	// Parse YAML to extract existing requirements (read-only operation)
	var azureYaml struct {
		Reqs []Prerequisite `yaml:"reqs"`
	}
	if err := yaml.Unmarshal(data, &azureYaml); err != nil {
		return 0, 0, fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	// Build map of existing requirement IDs
	existingCount := len(azureYaml.Reqs)

	// Convert detected requirements to generic map format for yamlutil
	var items []map[string]interface{}
	for _, det := range detected {
		item := map[string]interface{}{
			"name":       det.Name,
			"minVersion": det.MinVersion,
		}
		if det.CheckRunning {
			item["checkRunning"] = true
		}
		items = append(items, item)
	}

	// Use yamlutil to append new requirements
	opts := yamlutil.ArrayAppendOptions{
		SectionKey: "reqs",
		ItemIDKey:  "name",
		Items:      items,
		FormatItem: formatReqItem,
	}

	newContent, added, err := yamlutil.AppendToArraySection(content, opts)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to append reqs: %w", err)
	}

	// Write back to file
	// #nosec G306 -- azure.yaml is a config file, 0644 is appropriate for team access
	if err := os.WriteFile(azureYamlPath, []byte(newContent), 0644); err != nil {
		return 0, 0, fmt.Errorf("failed to write azure.yaml: %w", err)
	}

	return added, existingCount, nil
}

// formatReqItem formats a requirement item as YAML text.
func formatReqItem(item map[string]interface{}, arrayIndent string) string {
	var builder strings.Builder

	// Array item with Name
	builder.WriteString(arrayIndent)
	builder.WriteString("- name: ")
	builder.WriteString(item["name"].(string))
	builder.WriteString("\n")

	// MinVersion (quoted)
	builder.WriteString(arrayIndent)
	builder.WriteString("  minVersion: ")
	minVersion := item["minVersion"].(string)
	if strings.HasPrefix(minVersion, `"`) {
		builder.WriteString(minVersion)
	} else {
		builder.WriteString(`"` + minVersion + `"`)
	}
	builder.WriteString("\n")

	// CheckRunning (if present)
	if checkRunning, ok := item["checkRunning"].(bool); ok && checkRunning {
		builder.WriteString(arrayIndent)
		builder.WriteString("  checkRunning: true\n")
	}

	return builder.String()
}
