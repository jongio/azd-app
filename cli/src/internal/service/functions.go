// Package service provides Azure Functions runtime detection and configuration.
package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/portmanager"
	"github.com/jongio/azd-app/cli/src/internal/security"
)

// FunctionsVariant represents the type of Azure Functions project.
type FunctionsVariant int

const (
	// FunctionsVariantUnknown represents an unrecognized Functions variant
	FunctionsVariantUnknown FunctionsVariant = iota
	// FunctionsVariantLogicApps represents Logic Apps Standard
	FunctionsVariantLogicApps
	// FunctionsVariantNodeJS represents Node.js Functions (v3 or v4)
	FunctionsVariantNodeJS
	// FunctionsVariantPython represents Python Functions (v1 or v2)
	FunctionsVariantPython
	// FunctionsVariantDotNet represents .NET Functions (Isolated or In-Process)
	FunctionsVariantDotNet
	// FunctionsVariantJava represents Java Functions (Maven or Gradle)
	FunctionsVariantJava
)

// String returns the framework name for the FunctionsVariant.
func (v FunctionsVariant) String() string {
	switch v {
	case FunctionsVariantLogicApps:
		return "Logic Apps Standard"
	case FunctionsVariantNodeJS:
		return "Node.js Functions"
	case FunctionsVariantPython:
		return "Python Functions"
	case FunctionsVariantDotNet:
		return ".NET Functions"
	case FunctionsVariantJava:
		return "Java Functions"
	default:
		return "Unknown"
	}
}

// DefaultPort returns the default port for the FunctionsVariant.
func (v FunctionsVariant) DefaultPort() int {
	// All Azure Functions variants default to port 7071 (Azure Functions runtime default)
	return 7071
}

// HealthCheckPath returns the health check path for the FunctionsVariant.
func (v FunctionsVariant) HealthCheckPath() string {
	switch v {
	case FunctionsVariantLogicApps:
		// Logic Apps Standard exposes workflow management API
		return "/runtime/webhooks/workflow/api/management/workflows"
	default:
		// Standard Functions don't have a built-in health endpoint
		// Use admin/host/status for Functions runtime status
		return "/admin/host/status"
	}
}

// Logic Apps Detection Functions

// isLogicAppsVariant checks if the directory is a Logic Apps Standard project.
// It checks for:
// 1. workflows/ directory with workflow.json files, OR
// 2. host.json with Logic Apps extension bundle
func isLogicAppsVariant(projectDir string) bool {
	// Check for workflows folder with workflow.json files
	if hasLogicAppWorkflowsV2(projectDir) {
		return true
	}

	// Check host.json for Logic Apps extension bundle
	return hasLogicAppsExtensionBundle(projectDir)
}

// hasLogicAppWorkflowsV2 checks if the directory contains Logic Apps workflows.
// Note: Renamed to avoid conflict with existing function during Phase 1 (non-breaking).
func hasLogicAppWorkflowsV2(projectDir string) bool {
	workflowsPath := filepath.Join(projectDir, "workflows")
	if info, err := os.Stat(workflowsPath); err == nil && info.IsDir() {
		// Check for workflow.json files in subdirectories
		workflowFiles, _ := filepath.Glob(filepath.Join(workflowsPath, "*", "workflow.json"))
		return len(workflowFiles) > 0
	}
	return false
}

// hasLogicAppsExtensionBundle checks if host.json contains Logic Apps extension bundle.
func hasLogicAppsExtensionBundle(projectDir string) bool {
	hostJsonPath := filepath.Join(projectDir, "host.json")
	if err := security.ValidatePath(hostJsonPath); err != nil {
		return false
	}

	// #nosec G304 -- Path validated by security.ValidatePath
	data, err := os.ReadFile(hostJsonPath)
	if err != nil {
		return false
	}

	// Parse host.json to check for Logic Apps extension bundle
	var hostConfig struct {
		ExtensionBundle struct {
			ID string `json:"id"`
		} `json:"extensionBundle"`
	}

	if err := json.Unmarshal(data, &hostConfig); err != nil {
		return false
	}

	// Check if it's the Logic Apps Workflows extension bundle
	return hostConfig.ExtensionBundle.ID == "Microsoft.Azure.Functions.ExtensionBundle.Workflows"
}

// Node.js Detection Functions

// isNodeJSFunctionsVariant checks if the directory is a Node.js Functions project.
func isNodeJSFunctionsVariant(projectDir string) bool {
	// Node.js Functions require package.json
	if !fileExists(projectDir, "package.json") {
		return false
	}

	// Must have either:
	// 1. function.json files (v3 model), OR
	// 2. @azure/functions dependency (v4 model)
	return hasFunctionJson(projectDir) || hasAzureFunctionsDependency(projectDir)
}

// hasFunctionJson checks if the directory contains function.json files.
// This is used by both Node.js v3 and Python v1 models.
func hasFunctionJson(projectDir string) bool {
	// Check for function.json in subdirectories (v3/v1 model)
	functionJsonFiles, _ := filepath.Glob(filepath.Join(projectDir, "*", "function.json"))
	return len(functionJsonFiles) > 0
}

// hasAzureFunctionsDependency checks if package.json contains @azure/functions dependency.
func hasAzureFunctionsDependency(projectDir string) bool {
	packageJSONPath := filepath.Join(projectDir, "package.json")
	if err := security.ValidatePath(packageJSONPath); err != nil {
		return false
	}

	// #nosec G304 -- Path validated by security.ValidatePath
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return false
	}

	// Check for @azure/functions in dependencies or devDependencies
	return strings.Contains(string(data), "\"@azure/functions\"")
}

// Python Detection Functions

// isPythonFunctionsVariant checks if the directory is a Python Functions project.
func isPythonFunctionsVariant(projectDir string) bool {
	// Python Functions v2 model: function_app.py with decorators
	if fileExists(projectDir, "function_app.py") {
		return true
	}

	// Python Functions v1 model: requirements.txt + function.json files
	if fileExists(projectDir, "requirements.txt") && hasFunctionJson(projectDir) {
		return true
	}

	return false
}

// .NET Detection Functions

// isDotNetFunctionsVariant checks if the directory is a .NET Functions project.
func isDotNetFunctionsVariant(projectDir string) bool {
	// Check for .csproj files
	csprojFiles, err := filepath.Glob(filepath.Join(projectDir, "*.csproj"))
	if err != nil || len(csprojFiles) == 0 {
		return false
	}

	// Check for Azure Functions Worker (Isolated) or Azure Functions SDK (In-Process)
	for _, csprojFile := range csprojFiles {
		if containsText(csprojFile, "Microsoft.Azure.Functions.Worker") {
			return true // Isolated Worker
		}
		if containsText(csprojFile, "Microsoft.NET.Sdk.Functions") {
			return true // In-Process
		}
	}

	return false
}

// Java Detection Functions

// isJavaFunctionsVariant checks if the directory is a Java Functions project.
func isJavaFunctionsVariant(projectDir string) bool {
	// Maven project
	if fileExists(projectDir, "pom.xml") {
		pomPath := filepath.Join(projectDir, "pom.xml")
		if containsText(pomPath, "azure-functions-maven-plugin") {
			return true
		}
	}

	// Gradle project
	if fileExists(projectDir, "build.gradle") {
		buildGradlePath := filepath.Join(projectDir, "build.gradle")
		if containsText(buildGradlePath, "azurefunctions") || containsText(buildGradlePath, "azure-functions") {
			return true
		}
	}

	if fileExists(projectDir, "build.gradle.kts") {
		buildGradleKtsPath := filepath.Join(projectDir, "build.gradle.kts")
		if containsText(buildGradleKtsPath, "azurefunctions") || containsText(buildGradleKtsPath, "azure-functions") {
			return true
		}
	}

	return false
}

// Main Detection Function

// detectFunctionsVariant detects the Azure Functions variant for the given project directory.
// Returns FunctionsVariantUnknown if the project is not a recognized Functions variant.
func detectFunctionsVariant(projectDir string) FunctionsVariant {
	// Validate host.json exists (required for all Azure Functions projects)
	hostJsonPath := filepath.Join(projectDir, "host.json")
	if err := security.ValidatePath(hostJsonPath); err != nil {
		return FunctionsVariantUnknown
	}
	if _, err := os.Stat(hostJsonPath); os.IsNotExist(err) {
		return FunctionsVariantUnknown
	}

	// Check each variant in priority order
	// Logic Apps should be checked first as it has specific requirements
	if isLogicAppsVariant(projectDir) {
		return FunctionsVariantLogicApps
	}

	if isNodeJSFunctionsVariant(projectDir) {
		return FunctionsVariantNodeJS
	}

	if isPythonFunctionsVariant(projectDir) {
		return FunctionsVariantPython
	}

	if isDotNetFunctionsVariant(projectDir) {
		return FunctionsVariantDotNet
	}

	if isJavaFunctionsVariant(projectDir) {
		return FunctionsVariantJava
	}

	return FunctionsVariantUnknown
}

// Language Detection Function

// detectFunctionsLanguage detects the programming language for the Functions variant.
func detectFunctionsLanguage(variant FunctionsVariant, projectDir string) (string, error) {
	switch variant {
	case FunctionsVariantLogicApps:
		return "Logic Apps", nil

	case FunctionsVariantNodeJS:
		// Check if it's TypeScript
		if fileExists(projectDir, "tsconfig.json") {
			return "TypeScript", nil
		}
		return "JavaScript", nil

	case FunctionsVariantPython:
		return "Python", nil

	case FunctionsVariantDotNet:
		return "C#", nil

	case FunctionsVariantJava:
		return "Java", nil

	default:
		return "", fmt.Errorf("could not detect language in %s", projectDir)
	}
}

// Runtime Building Function

// buildFunctionsRuntime creates a ServiceRuntime for an Azure Functions project.
func buildFunctionsRuntime(serviceName string, service Service, projectDir string, usedPorts map[int]bool, azureYamlDir string) (*ServiceRuntime, error) {
	// Validate host.json exists
	hostJsonPath := filepath.Join(projectDir, "host.json")
	if err := security.ValidatePath(hostJsonPath); err != nil {
		return nil, fmt.Errorf("invalid host.json path: %w", err)
	}
	if _, err := os.Stat(hostJsonPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Azure Functions project missing host.json at %s", projectDir)
	}

	// Detect Functions variant
	variant := detectFunctionsVariant(projectDir)
	if variant == FunctionsVariantUnknown {
		return nil, fmt.Errorf("could not determine Azure Functions variant for project at %s (host.json exists but no function definitions found)", projectDir)
	}

	// Detect language
	language, err := detectFunctionsLanguage(variant, projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to detect language: %w", err)
	}

	// Create base ServiceRuntime
	runtime := &ServiceRuntime{
		Name:           serviceName,
		Language:       language,
		Framework:      variant.String(),
		PackageManager: "func",
		Command:        "func",
		WorkingDir:     projectDir,
		Protocol:       "http",
		Env:            make(map[string]string),
	}

	// Assign port
	port, shouldUpdateAzureYaml, err := assignFunctionsPort(serviceName, service, variant, usedPorts, azureYamlDir)
	if err != nil {
		return nil, fmt.Errorf("failed to assign port: %w", err)
	}
	runtime.Port = port
	runtime.ShouldUpdateAzureYaml = shouldUpdateAzureYaml
	usedPorts[port] = true

	// Build command args: func start --port <port>
	runtime.Args = []string{"start", "--port", fmt.Sprintf("%d", port)}

	// Configure health check
	runtime.HealthCheck = HealthCheckConfig{
		Type:     "http",
		Path:     variant.HealthCheckPath(),
		Timeout:  60000000000, // 60 seconds in nanoseconds
		Interval: 2000000000,  // 2 seconds in nanoseconds
	}

	return runtime, nil
}

// Port Assignment Function

// assignFunctionsPort assigns a port for an Azure Functions service.
// Returns the assigned port, whether azure.yaml should be updated, and any error.
func assignFunctionsPort(serviceName string, service Service, variant FunctionsVariant, usedPorts map[int]bool, azureYamlDir string) (int, bool, error) {
	// Get variant default port
	preferredPort := variant.DefaultPort()
	isExplicit := false

	// Check for explicit port in azure.yaml
	if len(service.Ports) > 0 {
		if hostPort, _, explicit := service.GetPrimaryPort(); hostPort > 0 {
			preferredPort = hostPort
			isExplicit = explicit
		}
	}

	// If preferred port is already in use and not explicit, find next available
	if !isExplicit && usedPorts[preferredPort] {
		// Find next available port starting from preferredPort + 1
		for port := preferredPort + 1; port < 65535; port++ {
			if !usedPorts[port] {
				preferredPort = port
				break
			}
		}
	}

	// Use port manager from azure.yaml directory (shared across all services)
	portMgr := portmanager.GetPortManager(azureYamlDir)
	port, shouldUpdateAzureYaml, err := portMgr.AssignPort(serviceName, preferredPort, isExplicit)
	if err != nil {
		return 0, false, fmt.Errorf("failed to assign port: %w", err)
	}

	return port, shouldUpdateAzureYaml, nil
}
