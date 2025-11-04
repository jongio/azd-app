package commands

import (
	"fmt"

	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/installer"
	"github.com/jongio/azd-app/cli/src/internal/output"
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

// installNodeServiceDepsWithResult installs Node.js dependencies and returns structured result.
func _installNodeServiceDepsWithResult(serviceName, serviceDir string) (map[string]interface{}, error) {
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

// installPythonServiceDepsWithResult installs Python dependencies and returns structured result.
func _installPythonServiceDepsWithResult(serviceName, serviceDir string) (map[string]interface{}, error) {
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

// installDotnetServiceDepsWithResult installs .NET dependencies and returns structured result.
func _installDotnetServiceDepsWithResult(serviceName, serviceDir string) (map[string]interface{}, error) {
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

// Deprecated: Legacy functions kept for reference - prevents unused warnings
var (
	_ = _installNodeServiceDepsWithResult
	_ = _installPythonServiceDepsWithResult
	_ = _installDotnetServiceDepsWithResult
)
