package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"app/src/internal/detector"
	"app/src/internal/installer"
	"app/src/internal/service"
	"app/src/internal/types"
	"github.com/spf13/cobra"
)

// NewDepsCommand creates the deps command.
func NewDepsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "deps",
		Short: "Install dependencies for all detected projects",
		Long:  `Automatically detects and installs dependencies for Node.js (npm/pnpm/yarn), Python (uv/poetry/pip), and .NET projects`,
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
	fmt.Println("🔍 Installing dependencies...")
	fmt.Println()

	hasProjects := false

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
		switch svc.Language {
		case "node", "nodejs", "javascript", "typescript":
			hasProjects = true
			if err := installNodeServiceDeps(serviceName, serviceDir); err != nil {
				fmt.Printf("   ⚠️  Warning: Failed to install Node.js dependencies for service %s: %v\n", serviceName, err)
			}

		case "python":
			hasProjects = true
			if err := installPythonServiceDeps(serviceName, serviceDir); err != nil {
				fmt.Printf("   ⚠️  Warning: Failed to install Python dependencies for service %s: %v\n", serviceName, err)
			}

		case "dotnet", ".net", "csharp", "c#":
			hasProjects = true
			if err := installDotnetServiceDeps(serviceName, serviceDir); err != nil {
				fmt.Printf("   ⚠️  Warning: Failed to install .NET dependencies for service %s: %v\n", serviceName, err)
			}
		}
	}

	if !hasProjects {
		fmt.Println("ℹ️  No dependency-managed services found in azure.yaml")
		fmt.Println()
		return nil
	}

	fmt.Println("✅ Dependencies installed successfully!")
	return nil
}

// installNodeServiceDeps installs Node.js dependencies for a specific service directory.
func installNodeServiceDeps(serviceName, serviceDir string) error {
	// Detect package manager only within the service directory (no parent search)
	packageManager := detector.DetectNodePackageManagerWithBoundary(serviceDir, serviceDir)
	
	nodeProject := types.NodeProject{
		Dir:            serviceDir,
		PackageManager: packageManager,
	}

	fmt.Printf("📦 Found Node.js service: %s\n", serviceName)
	fmt.Printf("   📥 Installing: %s (%s)\n", serviceDir, packageManager)
	
	return installer.InstallNodeDependencies(nodeProject)
}

// installPythonServiceDeps installs Python dependencies for a specific service directory.
func installPythonServiceDeps(serviceName, serviceDir string) error {
	packageManager := detector.DetectPythonPackageManager(serviceDir)
	
	pythonProject := types.PythonProject{
		Dir:            serviceDir,
		PackageManager: packageManager,
	}

	fmt.Printf("🐍 Found Python service: %s\n", serviceName)
	fmt.Printf("   📦 %s (%s)\n", serviceDir, packageManager)
	
	return installer.SetupPythonVirtualEnv(pythonProject)
}

// installDotnetServiceDeps installs .NET dependencies for a specific service directory.
func installDotnetServiceDeps(serviceName, serviceDir string) error {
	// Find .NET projects in the service directory
	dotnetProjects, err := detector.FindDotnetProjects(serviceDir)
	if err != nil || len(dotnetProjects) == 0 {
		return fmt.Errorf("no .NET projects found in %s", serviceDir)
	}

	fmt.Printf("🔷 Found .NET service: %s\n", serviceName)
	fmt.Printf("   📦 %s\n", serviceDir)
	
	// Install dependencies for all .NET projects in the service directory
	for _, dotnetProject := range dotnetProjects {
		if err := installer.RestoreDotnetProject(dotnetProject); err != nil {
			return fmt.Errorf("failed to restore %s: %w", dotnetProject.Path, err)
		}
	}
	
	return nil
}
