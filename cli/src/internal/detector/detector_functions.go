package detector

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/fileutil"
	"github.com/jongio/azd-app/cli/src/internal/types"
)

// FindFunctionApps searches for Azure Functions projects (all variants including Logic Apps).
// Only searches within rootDir and does not traverse outside it.
// Returns all discovered Function Apps with their variant and language detected.
func FindFunctionApps(rootDir string) ([]types.FunctionAppProject, error) {
	var functionApps []types.FunctionAppProject
	seen := make(map[string]bool)

	// Clean the root directory path
	rootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return functionApps, err
	}

	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		// Standard error handling: log and skip problematic paths
		if err != nil {
			slog.Debug("skipping path due to error", "path", path, "error", err)
			return nil
		}

		// Ensure we don't traverse outside rootDir
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil
		}
		relPath, err := filepath.Rel(rootDir, absPath)
		if err != nil || strings.HasPrefix(relPath, "..") {
			return filepath.SkipDir
		}

		if info.IsDir() {
			name := info.Name()
			// Skip common directories
			if name == skipDirNodeModules || name == skipDirGit || name == skipDirBin || name == skipDirObj {
				return filepath.SkipDir
			}
		}

		// Look for host.json files (required for all Azure Functions projects)
		if !info.IsDir() && info.Name() == "host.json" {
			dir := filepath.Dir(path)

			// Skip if we've already processed this directory
			if seen[dir] {
				return nil
			}

			// Detect the Functions variant
			variant := detectFunctionsVariantForDiscovery(dir)
			if variant == "" {
				// host.json exists but couldn't determine variant, skip
				return nil
			}

			// Detect the language
			language := detectFunctionsLanguageForDiscovery(variant, dir)

			functionApps = append(functionApps, types.FunctionAppProject{
				Dir:      dir,
				Variant:  variant,
				Language: language,
			})
			seen[dir] = true
		}

		return nil
	})

	return functionApps, err
}

// detectFunctionsVariantForDiscovery detects the Azure Functions variant for discovery.
// Returns variant string ("logicapps", "nodejs", "python", "dotnet", "java") or empty if unknown.
func detectFunctionsVariantForDiscovery(dir string) string {
	// Check for Logic Apps Standard (workflows directory or extension bundle)
	if isLogicAppsDirectory(dir) {
		return "logicapps"
	}

	// Check for Node.js Functions (package.json + function.json or @azure/functions)
	if fileExistsInDir(dir, "package.json") {
		if hasFunctionJsonInDir(dir) || hasAzureFunctionsDependencyInDir(dir) {
			return "nodejs"
		}
	}

	// Check for Python Functions (function_app.py or requirements.txt + function.json)
	if fileExistsInDir(dir, "function_app.py") {
		return "python"
	}
	if fileExistsInDir(dir, "requirements.txt") && hasFunctionJsonInDir(dir) {
		return "python"
	}

	// Check for .NET Functions (.csproj with Azure Functions references)
	csprojFiles, err := filepath.Glob(filepath.Join(dir, "*.csproj"))
	if err == nil && len(csprojFiles) > 0 {
		for _, csprojFile := range csprojFiles {
			if containsTextInFile(csprojFile, "Microsoft.Azure.Functions.Worker") ||
				containsTextInFile(csprojFile, "Microsoft.NET.Sdk.Functions") {
				return "dotnet"
			}
		}
	}

	// Check for Java Functions (pom.xml or build.gradle with Azure Functions plugin)
	if fileExistsInDir(dir, "pom.xml") {
		if containsTextInFile(filepath.Join(dir, "pom.xml"), "azure-functions-maven-plugin") {
			return "java"
		}
	}
	if fileExistsInDir(dir, "build.gradle") {
		buildGradle := filepath.Join(dir, "build.gradle")
		if containsTextInFile(buildGradle, "azurefunctions") || containsTextInFile(buildGradle, "azure-functions") {
			return "java"
		}
	}
	if fileExistsInDir(dir, "build.gradle.kts") {
		buildGradleKts := filepath.Join(dir, "build.gradle.kts")
		if containsTextInFile(buildGradleKts, "azurefunctions") || containsTextInFile(buildGradleKts, "azure-functions") {
			return "java"
		}
	}

	return ""
}

// detectFunctionsLanguageForDiscovery detects the programming language for the Functions variant.
func detectFunctionsLanguageForDiscovery(variant string, dir string) string {
	switch variant {
	case "logicapps":
		return "Logic Apps"
	case "nodejs":
		if fileExistsInDir(dir, "tsconfig.json") {
			return "TypeScript"
		}
		return "JavaScript"
	case "python":
		return "Python"
	case "dotnet":
		return "C#"
	case "java":
		return "Java"
	default:
		return ""
	}
}

// hasFunctionJsonInDir checks if the directory contains function.json files.
func hasFunctionJsonInDir(dir string) bool {
	functionJsonFiles, _ := filepath.Glob(filepath.Join(dir, "*", "function.json"))
	return len(functionJsonFiles) > 0
}

// hasAzureFunctionsDependencyInDir checks if package.json contains @azure/functions dependency.
func hasAzureFunctionsDependencyInDir(dir string) bool {
	return fileutil.ContainsTextInFile(dir, "package.json", "\"@azure/functions\"")
}

// isLogicAppsDirectory checks if a directory is a Logic Apps Standard project.
// Returns true if the directory contains a workflows folder OR host.json with Logic Apps extension bundle.
func isLogicAppsDirectory(dir string) bool {
	// Check for workflows subdirectory
	workflowsPath := filepath.Join(dir, "workflows")
	if info, err := os.Stat(workflowsPath); err == nil && info.IsDir() {
		// Verify it has workflow.json files
		workflowFiles, _ := filepath.Glob(filepath.Join(workflowsPath, "*", "workflow.json"))
		if len(workflowFiles) > 0 {
			return true
		}
	}

	// Check host.json for Logic Apps extension bundle
	hostJsonPath := filepath.Join(dir, "host.json")
	data, err := os.ReadFile(hostJsonPath) // #nosec G304 -- Path is within project boundaries
	if err != nil {
		return false
	}

	var hostConfig struct {
		ExtensionBundle struct {
			ID string `json:"id"`
		} `json:"extensionBundle"`
	}

	if err := json.Unmarshal(data, &hostConfig); err != nil {
		return false
	}

	return hostConfig.ExtensionBundle.ID == "Microsoft.Azure.Functions.ExtensionBundle.Workflows"
}
