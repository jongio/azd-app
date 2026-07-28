// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/portmanager"
	"github.com/jongio/azd-core/security"
)

const (
	// Virtual environment directory names
	venvDirPrimary   = ".venv"
	venvDirSecondary = "venv"

	// Virtual environment subdirectories
	venvBinDirWindows = "Scripts"
	venvBinDirUnix    = "bin"

	// Python executable names
	pythonExeWindows = "python.exe"
	pythonExeUnix    = "python"
)

// DetectServiceRuntime determines how to run a service based on its configuration and project structure.
func DetectServiceRuntime(serviceName string, service Service, usedPorts map[int]bool, azureYamlDir string, runtimeMode string) (*ServiceRuntime, error) {
	// Container services (image/docker.image) run as containers — UNLESS the
	// service opts into running as a local process via an explicit local
	// `command` or `type: process`. In that case its `docker.*`/image is
	// deploy-only and `azd app run` runs the command as a process.
	if service.IsContainerService() && !service.RunsAsLocalProcess() {
		return detectContainerRuntime(serviceName, service, usedPorts, azureYamlDir)
	}

	projectDir := service.Project
	if projectDir == "" {
		return nil, fmt.Errorf("service %s has no project directory", serviceName)
	}

	// Resolve relative paths against azure.yaml directory
	if !filepath.IsAbs(projectDir) {
		projectDir = filepath.Join(azureYamlDir, projectDir)
	}

	// Clean and normalize the path
	projectDir = filepath.Clean(projectDir)

	// Validate project directory
	if err := security.ValidatePath(projectDir); err != nil {
		return nil, fmt.Errorf("invalid project directory: %w", err)
	}

	// Determine default health check type based on service configuration
	defaultHealthCheckType := ServiceTypeHTTP
	if service.IsHealthcheckDisabled() {
		defaultHealthCheckType = watchModeNone
	} else if service.Healthcheck != nil && service.Healthcheck.Type != "" {
		defaultHealthCheckType = service.Healthcheck.Type
	}

	runtime := &ServiceRuntime{
		Name:       serviceName,
		WorkingDir: projectDir,
		Protocol:   "http",
		Env:        make(map[string]string),
		Restart:    service.GetRestartPolicy(),
		HealthCheck: HealthCheckConfig{
			Type:     defaultHealthCheckType,
			Path:     "/",
			Timeout:  60 * time.Second,
			Interval: 2 * time.Second,
		},
	}

	// Apply custom health check path and pattern if configured
	if service.Healthcheck != nil {
		if service.Healthcheck.Path != "" {
			runtime.HealthCheck.Path = service.Healthcheck.Path
		}
		if service.Healthcheck.Pattern != "" {
			runtime.HealthCheck.LogMatch = service.Healthcheck.Pattern
		}
	}

	// Special handling for Azure Functions (all variants including Logic Apps)
	if service.Host == "function" {
		return buildFunctionsRuntime(serviceName, service, projectDir, usedPorts, azureYamlDir)
	}

	// Static Web Apps may contain managed Azure Functions as the API backend.
	// Probe the project directory for Functions indicators (host.json + function definitions).
	if service.Host == "staticwebapp" {
		if variant := detectFunctionsVariant(projectDir); variant != FunctionsVariantUnknown {
			return buildFunctionsRuntime(serviceName, service, projectDir, usedPorts, azureYamlDir)
		}
	}

	// Detect language (use explicit language if provided)
	language := service.Language
	if language == "" {
		detectedLang, err := detectLanguage(projectDir, service.Host)
		if err != nil {
			return nil, fmt.Errorf("failed to detect language: %w", err)
		}
		language = detectedLang
	}
	runtime.Language = normalizeLanguage(language)

	// Detect framework and package manager
	framework, packageManager, err := detectFrameworkAndPackageManager(projectDir, runtime.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to detect framework: %w", err)
	}
	runtime.Framework = framework
	runtime.PackageManager = packageManager

	// Port assignment: skip for services that don't need a port (e.g., build/watch services)
	if service.NeedsPort() {
		// Detect preferred port from config (and whether it's explicitly set in azure.yaml)
		preferredPort, isExplicit, _ := DetectPort(serviceName, service, projectDir, framework, usedPorts)

		// Use port manager from azure.yaml directory (not service project dir) so all services share port assignments
		portMgr := portmanager.GetPortManager(azureYamlDir)
		port, shouldUpdateAzureYaml, err := portMgr.AssignPort(serviceName, preferredPort, isExplicit)
		if err != nil {
			return nil, fmt.Errorf("failed to assign port: %w", err)
		}
		runtime.Port = port
		runtime.ShouldUpdateAzureYaml = shouldUpdateAzureYaml // Track if user wants azure.yaml updated
		usedPorts[port] = true
	} else {
		// No port needed - service runs without HTTP endpoint (e.g., tsc --watch)
		runtime.Port = 0
		// Set health check to process-based since there's no HTTP endpoint
		if runtime.HealthCheck.Type == "http" {
			runtime.HealthCheck.Type = "process"
		}
	}

	// Build command and args based on framework (AFTER port assignment)
	// Docker Compose style: entrypoint is executable, command is args.
	// Array-form `command:` carries exact tokens; use them directly so arguments
	// containing whitespace are preserved (the string path re-splits on spaces).
	switch {
	case len(service.CommandArgs) > 0 && service.Entrypoint != "":
		runtime.Command = service.Entrypoint
		runtime.Args = append([]string(nil), service.CommandArgs...)
	case len(service.CommandArgs) > 0:
		runtime.Command = service.CommandArgs[0]
		runtime.Args = append([]string(nil), service.CommandArgs[1:]...)
	default:
		if err := buildRunCommand(runtime, projectDir, service.Entrypoint, service.Command, runtimeMode); err != nil {
			return nil, fmt.Errorf("failed to build run command: %w", err)
		}
	}

	// Set health check configuration based on framework (only if not explicitly disabled)
	if !service.IsHealthcheckDisabled() {
		configureHealthCheck(runtime)
	}

	// Detect and set service type and mode
	runtime.Type = service.GetServiceType()
	if runtime.Type == ServiceTypeProcess {
		// Detect mode from explicit config, command, or project structure
		runtime.Mode = detectServiceMode(service, runtime, projectDir)
	}

	// Copy environment variables from service config. A service run as a local
	// process — including a docker/appservice/containerapp service routed to a
	// local `command` or `type: process` (RunsAsLocalProcess) — must still
	// receive its azure.yaml `environment:` block. Orchestration resolves each
	// service's effective env from runtime.Env (see orchestrator.go), so without
	// this the block (e.g. APP_BASE_URL, NODE_ENV, POSTGRES_URL) is silently
	// dropped for every non-container service. Mirrors detectContainerRuntime.
	for key, value := range service.GetEnvironment() {
		runtime.Env[key] = value
	}

	return runtime, nil
}

// detectContainerRuntime creates a ServiceRuntime for a Docker container service.
// Container services are identified by having an `image` field set.
func detectContainerRuntime(serviceName string, service Service, usedPorts map[int]bool, azureYamlDir string) (*ServiceRuntime, error) {
	image := service.GetContainerImage()
	if image == "" {
		return nil, fmt.Errorf("container service %s has no image", serviceName)
	}

	// Determine default health check type - TCP for containers (port connectivity)
	defaultHealthCheckType := "tcp"
	if service.IsHealthcheckDisabled() {
		defaultHealthCheckType = watchModeNone
	} else if service.Healthcheck != nil && service.Healthcheck.Type != "" {
		defaultHealthCheckType = service.Healthcheck.Type
	}

	runtime := &ServiceRuntime{
		Name:       serviceName,
		WorkingDir: azureYamlDir, // Container runs from project root
		Protocol:   "tcp",
		Env:        make(map[string]string),
		Type:       ServiceTypeContainer,
		Restart:    service.GetRestartPolicy(),
		HealthCheck: HealthCheckConfig{
			Type:     defaultHealthCheckType,
			Timeout:  60 * time.Second,
			Interval: 2 * time.Second,
		},
	}

	// Store container image in the runtime
	runtime.Image = image
	runtime.Language = "container"
	runtime.Framework = packageMgrDocker

	// Copy environment variables from service config
	for key, value := range service.GetEnvironment() {
		runtime.Env[key] = value
	}

	// Container command override (string or array form) becomes the run args
	// appended after the image.
	runtime.Args = service.GetCommandArgs()

	// Image pull policy ("", "missing", "always", "never").
	runtime.PullPolicy = service.PullPolicy

	// Resolve volume mounts: named volumes pass through; bind-mount host paths
	// are resolved to absolute paths relative to the project directory.
	for _, vol := range service.Volumes {
		resolved, err := resolveVolumeSpec(vol, azureYamlDir)
		if err != nil {
			return nil, fmt.Errorf("invalid volume %q for service %s: %w", vol, serviceName, err)
		}
		runtime.Volumes = append(runtime.Volumes, resolved)
	}

	// Handle port assignment for container services
	if service.NeedsPort() {
		// Get port mappings from service config
		mappings, isExplicit := service.GetPortMappings()
		if len(mappings) > 0 {
			// Use the first port mapping's host port (or container port if host not specified)
			hostPort := mappings[0].HostPort
			containerPort := mappings[0].ContainerPort

			if hostPort == 0 {
				// Auto-assign host port using port manager
				portMgr := portmanager.GetPortManager(azureYamlDir)
				assignedPort, shouldUpdate, err := portMgr.AssignPort(serviceName, containerPort, isExplicit)
				if err != nil {
					return nil, fmt.Errorf("failed to assign port for container: %w", err)
				}
				runtime.Port = assignedPort
				runtime.ShouldUpdateAzureYaml = shouldUpdate
				// Reflect the assignment in the stored mapping so all published
				// ports (below) use the resolved host port.
				mappings[0].HostPort = assignedPort
			} else {
				runtime.Port = hostPort
			}

			// Update health check port (targets the primary/first port)
			runtime.HealthCheck.Port = runtime.Port
			usedPorts[runtime.Port] = true

			// Store ALL mappings so multi-port containers publish every port.
			runtime.Ports = mappings
			for _, m := range mappings {
				if m.HostPort > 0 {
					usedPorts[m.HostPort] = true
				}
			}
		}
	}

	return runtime, nil
}

// detectServiceMode determines the run mode for a process-type service.
// Priority: explicit config > command detection > project structure > default (daemon).
func detectServiceMode(service Service, runtime *ServiceRuntime, projectDir string) string {
	// If explicitly set in config, use that
	if service.Mode != "" {
		return service.Mode
	}

	// Build full command string for detection
	fullCmd := runtime.Command
	if len(runtime.Args) > 0 {
		fullCmd = fullCmd + " " + strings.Join(runtime.Args, " ")
	}

	// Check command for watch indicators
	if isWatchCommand(fullCmd) {
		return ServiceModeWatch
	}

	// Check command for build indicators
	if isBuildCommand(fullCmd, runtime.Language) {
		return ServiceModeBuild
	}

	// Check project structure for watch mode indicators
	if hasWatchModeIndicators(projectDir, runtime.Language, runtime.PackageManager) {
		return ServiceModeWatch
	}

	// Default to daemon for process services
	return ServiceModeDaemon
}

// isWatchCommand checks if the command indicates watch mode.
func isWatchCommand(cmd string) bool {
	cmdLower := strings.ToLower(cmd)

	// Common watch flags and commands
	watchIndicators := []string{
		"--watch",
		"-w ",
		" watch",
		"nodemon",
		"tsx watch",
		"ts-node-dev",
		"dotnet watch",
		"cargo watch",
		"air ", // Go live reload
		"reflex",
		"entr",
		"watchexec",
		"--reload", // uvicorn, flask
		"livereload",
		"browser-sync",
	}

	for _, indicator := range watchIndicators {
		if strings.Contains(cmdLower, indicator) {
			return true
		}
	}

	return false
}

// isBuildCommand checks if the command indicates a one-time build.
func isBuildCommand(cmd string, language string) bool {
	cmdLower := strings.ToLower(cmd)

	// Language-specific build commands
	buildCommands := map[string][]string{
		langTypeScript:     {"tsc", "npm run build", "pnpm build", "yarn build", "bun build"},
		langNameJavaScript: {"npm run build", "pnpm build", "yarn build", "bun build", "webpack", "rollup", "esbuild"},
		"Go":               {"go build", "go install"},
		langNameDotNet:     {"dotnet build", "dotnet publish"},
		langNameRust:       {"cargo build"},
		langNameJava:       {"mvn package", "mvn compile", "gradle build", "gradle assemble"},
		langNamePython:     {"python setup.py build", "pip wheel"},
	}

	// Check language-specific build commands
	if commands, ok := buildCommands[language]; ok {
		for _, buildCmd := range commands {
			if strings.Contains(cmdLower, buildCmd) {
				// Make sure it's not a watch variant
				if !strings.Contains(cmdLower, "watch") && !strings.Contains(cmdLower, "-w") {
					return true
				}
			}
		}
	}

	return false
}

// hasWatchModeIndicators checks project structure for watch mode configuration.
func hasWatchModeIndicators(projectDir string, language string, _ string) bool {
	switch language {
	case langTypeScript, langNameJavaScript:
		// Check package.json for watch scripts
		return hasWatchScriptInPackageJSON(projectDir)
	case "Go":
		// Check for air.toml or .air.toml (Go live reload)
		return fileExists(projectDir, "air.toml") ||
			fileExists(projectDir, ".air.toml") ||
			fileExists(projectDir, "reflex.conf")
	case langNameDotNet:
		// .NET watch is typically in command, not config
		return false
	case langNamePython:
		// Check for watchdog or watchfiles in requirements
		return containsWatchDependency(projectDir)
	default:
		return false
	}
}

// hasWatchScriptInPackageJSON checks if package.json has a watch script.
func hasWatchScriptInPackageJSON(projectDir string) bool {
	packageJSONPath := filepath.Join(projectDir, "package.json")
	if err := security.ValidatePath(packageJSONPath); err != nil {
		return false
	}

	// Check for common watch script names
	watchScripts := []string{
		`"watch"`,
		`"dev"`,
		`"start:dev"`,
		`"serve"`,
	}

	for _, script := range watchScripts {
		if containsText(packageJSONPath, script) {
			// Verify the script contains watch-like command
			// This is a heuristic - we check for common watch patterns in the file
			if containsText(packageJSONPath, "watch") ||
				containsText(packageJSONPath, "nodemon") ||
				containsText(packageJSONPath, "--reload") {
				return true
			}
		}
	}

	return false
}

// containsWatchDependency checks Python requirements for watch-related packages.
func containsWatchDependency(projectDir string) bool {
	requirementsFiles := []string{
		"requirements.txt",
		"requirements-dev.txt",
		"pyproject.toml",
	}

	watchPackages := []string{
		"watchdog",
		"watchfiles",
		"hupper",
		"reloading",
	}

	for _, reqFile := range requirementsFiles {
		filePath := filepath.Join(projectDir, reqFile)
		for _, pkg := range watchPackages {
			if containsText(filePath, pkg) {
				return true
			}
		}
	}

	return false
}
