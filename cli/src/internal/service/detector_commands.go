// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/security"
)

// buildRunCommand builds the command and arguments to run the service.
//
// Priority:
//  1. command: Full shell command (e.g., "uvicorn main:app --reload") - PRIMARY
//  2. entrypoint + command: Advanced Docker Compose style (rarely needed)
//  3. Neither: Auto-detect based on framework
func buildRunCommand(runtime *ServiceRuntime, projectDir, entrypoint, command, runtimeMode string) error {
	// Primary: command alone (most common case)
	if command != "" && entrypoint == "" {
		return parseShellCommand(runtime, command)
	}

	// Advanced: entrypoint + command (Docker Compose style)
	if entrypoint != "" {
		if command != "" {
			// Both provided: entrypoint is executable, command is args
			runtime.Command = entrypoint
			runtime.Args = strings.Fields(command)
		} else {
			// Only entrypoint: split it as full command
			return parseShellCommand(runtime, entrypoint)
		}
		return nil
	}

	// Neither provided: use framework-specific defaults
	return buildFrameworkCommand(runtime, projectDir, runtimeMode)
}

// parseShellCommand parses a user-provided shell command into command and args.
// Handles both simple commands ("node server.js") and complex ones ("uvicorn main:app --reload").
func parseShellCommand(runtime *ServiceRuntime, command string) error {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}
	runtime.Command = parts[0]
	if len(parts) > 1 {
		runtime.Args = parts[1:]
	} else {
		runtime.Args = nil
	}
	return nil
}

// buildFrameworkCommand builds framework-specific commands using intelligent defaults.
func buildFrameworkCommand(runtime *ServiceRuntime, projectDir, runtimeMode string) error {
	// Handle Python frameworks with venv support
	pythonFrameworks := map[string]struct{}{
		"Django": {}, "FastAPI": {}, "Flask": {},
		"Streamlit": {}, "Gradio": {}, "Python": {},
	}

	if _, isPython := pythonFrameworks[runtime.Framework]; isPython {
		pythonCmd := "python"
		if venvPython := getPythonVenvPath(projectDir); venvPython != "" {
			pythonCmd = venvPython
		}
		return buildPythonDefaultCommand(runtime, projectDir, pythonCmd)
	}

	switch runtime.Framework {
	case "Next.js", "React", "Vue", "Svelte", "SvelteKit", "Remix", "Astro", "Nuxt":
		runtime.Command = runtime.PackageManager
		runtime.Args = []string{"run", "dev"}

	case "Angular":
		runtime.Command = "ng"
		runtime.Args = []string{"serve", "--port", fmt.Sprintf("%d", runtime.Port)}

	case "NestJS":
		runtime.Command = runtime.PackageManager
		runtime.Args = []string{"run", "start:dev"}

	case "Express", "Node.js":
		runtime.Command = runtime.PackageManager
		// Try dev first, fall back to start
		if hasScript(projectDir, "dev") {
			runtime.Args = []string{"run", "dev"}
		} else {
			runtime.Args = []string{"run", "start"}
		}

	case "Logic Apps Standard":
		// Command already set in detectLogicAppRuntime
		return nil

	case "Aspire":
		return buildDotNetCommand(runtime, projectDir, runtimeMode, true)

	case "ASP.NET Core", ".NET":
		return buildDotNetCommand(runtime, projectDir, runtimeMode, false)

	case "Spring Boot":
		buildJavaCommand(runtime, true)
		return nil

	case "Java":
		buildJavaCommand(runtime, false)
		return nil

	case "Go":
		runtime.Command = "go"
		runtime.Args = []string{"run", "."}

	case "Rust":
		runtime.Command = "cargo"
		runtime.Args = []string{"run"}

	case "Laravel":
		runtime.Command = "php"
		runtime.Args = []string{"artisan", "serve", "--host=0.0.0.0", "--port=" + fmt.Sprintf("%d", runtime.Port)}

	case "PHP":
		runtime.Command = "php"
		runtime.Args = []string{"-S", fmt.Sprintf("0.0.0.0:%d", runtime.Port)}

	default:
		return fmt.Errorf("unsupported framework: %s", runtime.Framework)
	}

	return nil
}

// buildDotNetCommand configures a .NET service runtime command.
func buildDotNetCommand(runtime *ServiceRuntime, projectDir, runtimeMode string, isAspire bool) error {
	runtime.Command = "dotnet"

	csprojFiles, _ := filepath.Glob(filepath.Join(projectDir, "*.csproj"))
	if len(csprojFiles) > 0 {
		if isAspire && runtimeMode == "aspire" {
			// In aspire mode, use dotnet run to get native Aspire dashboard
			runtime.Args = []string{"run", "--project", csprojFiles[0]}
		} else if isAspire {
			// In azd mode, run individual services separately
			runtime.Args = []string{"run", "--project", csprojFiles[0], "--no-launch-profile"}
		} else {
			runtime.Args = []string{"run", "--project", csprojFiles[0]}
		}
	} else {
		runtime.Args = []string{"run"}
	}
	return nil
}

// buildJavaCommand configures a Java service runtime command.
func buildJavaCommand(runtime *ServiceRuntime, isSpringBoot bool) {
	if runtime.PackageManager == "maven" {
		runtime.Command = "mvn"
		if isSpringBoot {
			runtime.Args = []string{"spring-boot:run"}
		} else {
			runtime.Args = []string{"exec:java"}
		}
	} else {
		runtime.Command = "gradle"
		if isSpringBoot {
			runtime.Args = []string{"bootRun"}
		} else {
			runtime.Args = []string{"run"}
		}
	}
}

// getPythonVenvPath returns the path to the Python interpreter in the virtual environment.
// Returns empty string if no venv is found.
func getPythonVenvPath(projectDir string) string {
	// Check for .venv first (most common), then venv (alternative)
	venvPaths := []string{
		filepath.Join(projectDir, venvDirPrimary, venvBinDirWindows, pythonExeWindows),   // Windows
		filepath.Join(projectDir, venvDirPrimary, venvBinDirUnix, pythonExeUnix),         // Linux/macOS
		filepath.Join(projectDir, venvDirSecondary, venvBinDirWindows, pythonExeWindows), // Windows (alternative)
		filepath.Join(projectDir, venvDirSecondary, venvBinDirUnix, pythonExeUnix),       // Linux/macOS (alternative)
	}

	for _, path := range venvPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// resolvePythonEntrypoint resolves and validates the Python entrypoint file.
// Returns the entrypoint filename, with fallback to auto-detection if not provided.
func resolvePythonEntrypoint(projectDir, entrypoint string) (string, error) {
	appFile := entrypoint
	if appFile == "" {
		appFile = findPythonAppFile(projectDir)
	}
	if err := validatePythonEntrypoint(projectDir, appFile); err != nil {
		return "", err
	}
	return appFile, nil
}

// buildPythonDefaultCommand configures a Python service runtime with framework-specific defaults.
// Auto-detects the app file based on framework conventions.
func buildPythonDefaultCommand(runtime *ServiceRuntime, projectDir, pythonCmd string) error {
	runtime.Command = pythonCmd

	switch runtime.Framework {
	case "Django":
		// Django uses manage.py - validate it exists
		managePyPath := filepath.Join(projectDir, "manage.py")
		if err := security.ValidatePath(managePyPath); err != nil {
			return fmt.Errorf("django: invalid manage.py path: %w", err)
		}
		if _, err := os.Stat(managePyPath); err != nil {
			return fmt.Errorf("django: manage.py not found at %s", managePyPath)
		}
		runtime.Args = []string{"manage.py", "runserver", fmt.Sprintf("0.0.0.0:%d", runtime.Port)}
		return nil

	case "FastAPI":
		appFile, err := resolvePythonEntrypoint(projectDir, "") // Auto-detect
		if err != nil {
			return fmt.Errorf("FastAPI: %w", err)
		}
		// Use -m uvicorn to run as module from venv
		// FastAPI uses module:app format, no .py extension needed
		runtime.Args = []string{"-m", "uvicorn", appFile + ":app", "--reload", "--host", "0.0.0.0", "--port", fmt.Sprintf("%d", runtime.Port)}
		return nil

	case "Flask":
		appFile, err := resolvePythonEntrypoint(projectDir, "") // Auto-detect
		if err != nil {
			return fmt.Errorf("flask: %w", err)
		}
		runtime.Args = []string{"-m", "flask", "run", "--host", "0.0.0.0", "--port", fmt.Sprintf("%d", runtime.Port)}
		runtime.Env["FLASK_APP"] = appFile + ".py"
		runtime.Env["FLASK_ENV"] = "development"
		return nil

	case "Streamlit":
		appFile, err := resolvePythonEntrypoint(projectDir, "") // Auto-detect
		if err != nil {
			return fmt.Errorf("streamlit: %w", err)
		}
		// Use -m streamlit to run as module from venv
		runtime.Args = []string{"-m", "streamlit", "run", appFile + ".py", "--server.port", fmt.Sprintf("%d", runtime.Port)}
		return nil

	case "Gradio", "Python":
		appFile, err := resolvePythonEntrypoint(projectDir, "") // Auto-detect
		if err != nil {
			return fmt.Errorf("%s: %w", runtime.Framework, err)
		}
		// Run as regular Python script
		runtime.Args = []string{appFile + ".py"}
		return nil

	default:
		return fmt.Errorf("unsupported Python framework: %s", runtime.Framework)
	}
}

// configureHealthCheck sets up health check configuration based on framework.
func configureHealthCheck(runtime *ServiceRuntime) {
	healthConfigs := map[string]struct {
		path     string
		logMatch string
	}{
		"Logic Apps Standard": {"/runtime/webhooks/workflow/api/management/workflows", "Host started"},
		"Aspire":              {"/", "Now listening on"},
		"Next.js":             {"/", "ready on"},
		"Django":              {"/", "Starting development server"},
		"Spring Boot":         {"/actuator/health", "Started"},
		"FastAPI":             {"/docs", ""},
	}

	if config, exists := healthConfigs[runtime.Framework]; exists {
		runtime.HealthCheck.Path = config.path
		runtime.HealthCheck.LogMatch = config.logMatch
	} else {
		runtime.HealthCheck.Path = "/"
	}
}
