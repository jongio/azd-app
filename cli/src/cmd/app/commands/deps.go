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
			return runDepsWithServices()
		},
	}
}

// runDepsWithServices installs deps for services from azure.yaml.
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
		fmt.Println("ℹ️  No services defined in azure.yaml - skipping dependency installation")
		return nil
	}

	return installDepsFromAzureYaml(azureYaml, azureYamlPath)
}

// installDepsFromAzureYaml installs dependencies for services defined in azure.yaml.
func installDepsFromAzureYaml(azureYaml *service.AzureYaml, azureYamlPath string) error {
	if !output.IsJSON() {
		fmt.Println("🔍 Installing dependencies...")
		fmt.Println()
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
				fmt.Printf("   ⚠️  Warning: Failed to install Node.js dependencies for service %s: %v\n", serviceName, err)
			}

		case "python":
			hasProjects = true
			result, err = installPythonServiceDepsWithResult(serviceName, serviceDir)
			if err != nil && !output.IsJSON() {
				fmt.Printf("   ⚠️  Warning: Failed to install Python dependencies for service %s: %v\n", serviceName, err)
			}

		case "dotnet", ".net", "csharp", "c#":
			hasProjects = true
			result, err = installDotnetServiceDepsWithResult(serviceName, serviceDir)
			if err != nil && !output.IsJSON() {
				fmt.Printf("   ⚠️  Warning: Failed to install .NET dependencies for service %s: %v\n", serviceName, err)
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
		fmt.Println("ℹ️  No dependency-managed services found in azure.yaml")
		fmt.Println()
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

	fmt.Println("✅ Dependencies installed successfully!")
	return nil
}

// installNodeServiceDeps installs Node.js dependencies for a specific service directory.
func installNodeServiceDeps(serviceName, serviceDir string) error {
	result, err := installNodeServiceDepsWithResult(serviceName, serviceDir)
	_ = result // Ignore the result in non-JSON mode
	return err
}

// installNodeServiceDepsWithResult installs Node.js dependencies and returns structured result.
func installNodeServiceDepsWithResult(serviceName, serviceDir string) (map[string]interface{}, error) {
	// Detect package manager only within the service directory (no parent search)
	packageManager := detector.DetectNodePackageManagerWithBoundary(serviceDir, serviceDir)
	
	nodeProject := types.NodeProject{
		Dir:            serviceDir,
		PackageManager: packageManager,
	}

	if !output.IsJSON() {
		fmt.Printf("📦 Found Node.js service: %s\n", serviceName)
		fmt.Printf("   📥 Installing: %s (%s)\n", serviceDir, packageManager)
	}
	
	err := installer.InstallNodeDependencies(nodeProject)
	result := map[string]interface{}{
		"service":  serviceName,
		"type":     "node",
		"dir":      serviceDir,
		"manager":  packageManager,
		"success":  err == nil,
	}
	if err != nil {
		result["error"] = err.Error()
	}
	return result, err
}

// installPythonServiceDeps installs Python dependencies for a specific service directory.
func installPythonServiceDeps(serviceName, serviceDir string) error {
	result, err := installPythonServiceDepsWithResult(serviceName, serviceDir)
	_ = result // Ignore the result in non-JSON mode
	return err
}

// installPythonServiceDepsWithResult installs Python dependencies and returns structured result.
func installPythonServiceDepsWithResult(serviceName, serviceDir string) (map[string]interface{}, error) {
	packageManager := detector.DetectPythonPackageManager(serviceDir)
	
	pythonProject := types.PythonProject{
		Dir:            serviceDir,
		PackageManager: packageManager,
	}

	if !output.IsJSON() {
		fmt.Printf("🐍 Found Python service: %s\n", serviceName)
		fmt.Printf("   📦 %s (%s)\n", serviceDir, packageManager)
	}
	
	err := installer.SetupPythonVirtualEnv(pythonProject)
	result := map[string]interface{}{
		"service":  serviceName,
		"type":     "python",
		"dir":      serviceDir,
		"manager":  packageManager,
		"success":  err == nil,
	}
	if err != nil {
		result["error"] = err.Error()
	}
	return result, err
}

// installDotnetServiceDeps installs .NET dependencies for a specific service directory.
func installDotnetServiceDeps(serviceName, serviceDir string) error {
	result, err := installDotnetServiceDepsWithResult(serviceName, serviceDir)
	_ = result // Ignore the result in non-JSON mode
	return err
}

// installDotnetServiceDepsWithResult installs .NET dependencies and returns structured result.
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
		fmt.Printf("🔷 Found .NET service: %s\n", serviceName)
		fmt.Printf("   📦 %s\n", serviceDir)
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
