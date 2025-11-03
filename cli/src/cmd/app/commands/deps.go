package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/installer"
	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/types"
	"github.com/spf13/cobra"
)

// NewDepsCommand creates the deps command.
func NewDepsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "deps",
		Short: "Install dependencies for all detected projects",
		Long:  `Automatically detects and installs dependencies for Node.js (npm/pnpm/yarn), Python (uv/poetry/pip), and .NET projects`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Try to get the output flag from parent or self
			var formatValue string
			if flag := cmd.InheritedFlags().Lookup("output"); flag != nil {
				formatValue = flag.Value.String()
			} else if flag := cmd.Flags().Lookup("output"); flag != nil {
				formatValue = flag.Value.String()
			}
			if formatValue != "" {
				return output.SetFormat(formatValue)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use orchestrator to run deps (which will automatically run reqs first)
			return cmdOrchestrator.Run("deps")
		},
	}
}

// runDepsWithServices installs deps for services from azure.yaml.
//
//nolint:unused // Legacy function - kept for potential future use
func runDepsWithServices() error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Try to find azure.yaml
	azureYamlPath, err := detector.FindAzureYaml(cwd)
	if err != nil {
		return fmt.Errorf("error searching for azure.yaml: %w", err)
	}

	// Require azure.yaml
	if azureYamlPath == "" {
		return fmt.Errorf("azure.yaml not found - create one with 'services' section to define your development environment")
	}

	azureYaml, err := service.ParseAzureYaml(azureYamlPath)
	if err != nil {
		return fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	if !service.HasServices(azureYaml) {
		output.Info("No services defined in azure.yaml - skipping dependency installation")
		return nil
	}

	return installDepsFromAzureYaml(azureYaml, azureYamlPath)
}

// installDepsFromAzureYaml installs dependencies for services defined in azure.yaml.
//
//nolint:unused // Legacy function - kept for potential future use
func installDepsFromAzureYaml(azureYaml *service.AzureYaml, azureYamlPath string) error {
	if !output.IsJSON() {
		output.Section("🔍", "Installing dependencies")
	}

	hasProjects := false
	var results []map[string]interface{}

	// Track unique project directories to avoid duplicate installations
	installedDirs := make(map[string]bool)

	// Install dependencies for each service
	for serviceName, svc := range azureYaml.Services {
		// Convert relative project path to absolute path
		serviceDir := svc.Project
		if !filepath.IsAbs(serviceDir) {
			baseDir := filepath.Dir(azureYamlPath)
			serviceDir = filepath.Join(baseDir, serviceDir)
		}

		// Skip if we already installed dependencies for this directory
		if installedDirs[serviceDir] {
			continue
		}
		installedDirs[serviceDir] = true

		// Install dependencies based on service language
		var err error
		var result map[string]interface{}

		switch svc.Language {
		case "node", "nodejs", "javascript", "typescript":
			hasProjects = true
			result, err = installNodeServiceDepsWithResult(serviceName, serviceDir)
			if err != nil && !output.IsJSON() {
				output.ItemWarning("Failed to install Node.js dependencies for service %s: %v", serviceName, err)
			}

		case "python":
			hasProjects = true
			result, err = installPythonServiceDepsWithResult(serviceName, serviceDir)
			if err != nil && !output.IsJSON() {
				output.ItemWarning("Failed to install Python dependencies for service %s: %v", serviceName, err)
			}

		case "dotnet", ".net", "csharp", "c#":
			hasProjects = true
			result, err = installDotnetServiceDepsWithResult(serviceName, serviceDir)
			if err != nil && !output.IsJSON() {
				output.ItemWarning("Failed to install .NET dependencies for service %s: %v", serviceName, err)
			}
		}

		if result != nil {
			results = append(results, result)
		}
	}

	if !hasProjects {
		if output.IsJSON() {
			return output.PrintJSON(map[string]interface{}{
				"success":  true,
				"services": []interface{}{},
				"message":  "No dependency-managed services found in azure.yaml",
			})
		}
		output.Info("No dependency-managed services found in azure.yaml")
		return nil
	}

	if output.IsJSON() {
		// Check if any service failed
		allSuccess := true
		for _, result := range results {
			if success, ok := result["success"].(bool); ok && !success {
				allSuccess = false
				break
			}
		}
		return output.PrintJSON(map[string]interface{}{
			"success":  allSuccess,
			"services": results,
		})
	}

	output.Success("Dependencies installed successfully!")
	return nil
}

// installNodeServiceDeps installs Node.js dependencies for a specific service directory.
//
//nolint:unused // Legacy function - kept for potential future use
func installNodeServiceDeps(serviceName, serviceDir string) error {
	result, err := installNodeServiceDepsWithResult(serviceName, serviceDir)
	_ = result // Ignore the result in non-JSON mode
	return err
}

// installNodeServiceDepsWithResult installs Node.js dependencies and returns structured result.
//
//nolint:unused // Legacy function - kept for potential future use
func installNodeServiceDepsWithResult(serviceName, serviceDir string) (map[string]interface{}, error) {
	// Detect package manager only within the service directory (no parent search)
	packageManager := detector.DetectNodePackageManagerWithBoundary(serviceDir, serviceDir)

	nodeProject := types.NodeProject{
		Dir:            serviceDir,
		PackageManager: packageManager,
	}

	if !output.IsJSON() {
		output.Step("📦", "Found Node.js service: %s", serviceName)
		output.Item("Installing: %s (%s)", serviceDir, packageManager)
	}

	err := installer.InstallNodeDependencies(nodeProject)
	result := map[string]interface{}{
		"service": serviceName,
		"type":    "node",
		"dir":     serviceDir,
		"manager": packageManager,
		"success": err == nil,
	}
	if err != nil {
		result["error"] = err.Error()
	}
	return result, err
}

// installPythonServiceDeps installs Python dependencies for a specific service directory.
//
//nolint:unused // Legacy function - kept for potential future use
func installPythonServiceDeps(serviceName, serviceDir string) error {
	result, err := installPythonServiceDepsWithResult(serviceName, serviceDir)
	_ = result // Ignore the result in non-JSON mode
	return err
}

// installPythonServiceDepsWithResult installs Python dependencies and returns structured result.
//
//nolint:unused // Legacy function - kept for potential future use
func installPythonServiceDepsWithResult(serviceName, serviceDir string) (map[string]interface{}, error) {
	packageManager := detector.DetectPythonPackageManager(serviceDir)

	pythonProject := types.PythonProject{
		Dir:            serviceDir,
		PackageManager: packageManager,
	}

	if !output.IsJSON() {
		output.Step("🐍", "Found Python service: %s", serviceName)
		output.Item("%s (%s)", serviceDir, packageManager)
	}

	err := installer.SetupPythonVirtualEnv(pythonProject)
	result := map[string]interface{}{
		"service": serviceName,
		"type":    "python",
		"dir":     serviceDir,
		"manager": packageManager,
		"success": err == nil,
	}
	if err != nil {
		result["error"] = err.Error()
	}
	return result, err
}

// installDotnetServiceDeps installs .NET dependencies for a specific service directory.
//
//nolint:unused // Legacy function - kept for potential future use
func installDotnetServiceDeps(serviceName, serviceDir string) error {
	result, err := installDotnetServiceDepsWithResult(serviceName, serviceDir)
	_ = result // Ignore the result in non-JSON mode
	return err
}

// installDotnetServiceDepsWithResult installs .NET dependencies and returns structured result.
//
//nolint:unused // Legacy function - kept for potential future use
func installDotnetServiceDepsWithResult(serviceName, serviceDir string) (map[string]interface{}, error) {
	// Find .NET projects in the service directory
	dotnetProjects, err := detector.FindDotnetProjects(serviceDir)
	if err != nil || len(dotnetProjects) == 0 {
		errMsg := fmt.Errorf("no .NET projects found in %s", serviceDir)
		return map[string]interface{}{
			"service": serviceName,
			"type":    "dotnet",
			"dir":     serviceDir,
			"success": false,
			"error":   errMsg.Error(),
		}, errMsg
	}

	if !output.IsJSON() {
		output.Step("🔷", "Found .NET service: %s", serviceName)
		output.Item("%s", serviceDir)
	}

	// Install dependencies for all .NET projects in the service directory
	for _, dotnetProject := range dotnetProjects {
		if err := installer.RestoreDotnetProject(dotnetProject); err != nil {
			errResult := fmt.Errorf("failed to restore %s: %w", dotnetProject.Path, err)
			return map[string]interface{}{
				"service": serviceName,
				"type":    "dotnet",
				"dir":     serviceDir,
				"path":    dotnetProject.Path,
				"success": false,
				"error":   errResult.Error(),
			}, errResult
		}
	}

	return map[string]interface{}{
		"service": serviceName,
		"type":    "dotnet",
		"dir":     serviceDir,
		"success": true,
	}, nil
}
