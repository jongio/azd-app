package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/portmanager"
	"github.com/jongio/azd-app/cli/src/internal/security"
)

// DetectServiceRuntime determines how to run a service based on its configuration and project structure.
func DetectServiceRuntime(serviceName string, service Service, usedPorts map[int]bool, azureYamlDir string, runtimeMode string) (*ServiceRuntime, error) {
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

	runtime := &ServiceRuntime{
		Name:       serviceName,
		WorkingDir: projectDir,
		Protocol:   "http",
		Env:        make(map[string]string),
		HealthCheck: HealthCheckConfig{
			Type:     "http",
			Path:     "/",
			Timeout:  60 * time.Second,
			Interval: 2 * time.Second,
		},
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

	// Detect preferred port from config (and whether it's explicitly set in azure.yaml)
	preferredPort, isExplicit, _ := DetectPort(serviceName, service, projectDir, framework, usedPorts)

	// Use port manager to assign port (with automatic cleanup of stale processes)
	portMgr := portmanager.GetPortManager(projectDir)
	port, err := portMgr.AssignPort(serviceName, preferredPort, isExplicit, true) // isExplicit, cleanStale
	if err != nil {
		return nil, fmt.Errorf("failed to assign port: %w", err)
	}
	runtime.Port = port
	usedPorts[port] = true

	// Build command and args based on framework (AFTER port assignment)
	if err := buildRunCommand(runtime, projectDir, service.Entrypoint, runtimeMode); err != nil {
		return nil, fmt.Errorf("failed to build run command: %w", err)
	}

	// Set health check configuration based on framework
	configureHealthCheck(runtime)

	return runtime, nil
}

// detectLanguage determines the programming language used by the service.
func detectLanguage(projectDir string, host string) (string, error) {
	// Check for language indicators in priority order

	// Node.js/TypeScript
	if fileExists(projectDir, "package.json") {
		if fileExists(projectDir, "tsconfig.json") {
			return "TypeScript", nil
		}
		return "JavaScript", nil
	}

	// Python
	if fileExists(projectDir, "requirements.txt") || fileExists(projectDir, "pyproject.toml") ||
		fileExists(projectDir, "poetry.lock") || fileExists(projectDir, "uv.lock") {
		return "Python", nil
	}

	// .NET
	if hasFileWithExt(projectDir, ".csproj") || hasFileWithExt(projectDir, ".sln") ||
		hasFileWithExt(projectDir, ".fsproj") {
		return ".NET", nil
	}

	// Java
	if fileExists(projectDir, "pom.xml") || fileExists(projectDir, "build.gradle") ||
		fileExists(projectDir, "build.gradle.kts") {
		return "Java", nil
	}

	// Go
	if fileExists(projectDir, "go.mod") {
		return "Go", nil
	}

	// Rust
	if fileExists(projectDir, "Cargo.toml") {
		return "Rust", nil
	}

	// PHP
	if fileExists(projectDir, "composer.json") {
		return "PHP", nil
	}

	// Docker
	if fileExists(projectDir, "Dockerfile") || fileExists(projectDir, "docker-compose.yml") {
		return "Docker", nil
	}

	// Fallback: use host type as hint
	if host == "containerapp" || host == "aks" {
		return "Docker", nil
	}

	return "", fmt.Errorf("could not detect language in %s", projectDir)
}

// detectFrameworkAndPackageManager detects the specific framework and package manager.
func detectFrameworkAndPackageManager(projectDir string, language string) (string, string, error) {
	switch language {
	case "TypeScript", "JavaScript":
		return detectNodeFramework(projectDir)
	case "Python":
		return detectPythonFramework(projectDir)
	case ".NET":
		return detectDotNetFramework(projectDir)
	case "Java":
		return detectJavaFramework(projectDir)
	case "Go":
		return "Go", "go", nil
	case "Rust":
		return "Rust", "cargo", nil
	case "PHP":
		return detectPHPFramework(projectDir)
	case "Docker":
		return "Docker", "docker", nil
	default:
		return language, "", nil
	}
}

// detectNodeFramework detects Node.js/TypeScript framework.
func detectNodeFramework(projectDir string) (string, string, error) {
	// Detect package manager
	packageManager := detector.DetectNodePackageManagerWithBoundary(projectDir, projectDir)

	// Detect framework
	if fileExists(projectDir, "next.config.js") || fileExists(projectDir, "next.config.ts") ||
		fileExists(projectDir, "next.config.mjs") {
		return "Next.js", packageManager, nil
	}

	if fileExists(projectDir, "angular.json") {
		return "Angular", packageManager, nil
	}

	if fileExists(projectDir, "nuxt.config.ts") || fileExists(projectDir, "nuxt.config.js") {
		return "Nuxt", packageManager, nil
	}

	if fileExists(projectDir, "vite.config.ts") || fileExists(projectDir, "vite.config.js") {
		return "React", packageManager, nil // Vite is commonly used with React
	}

	if fileExists(projectDir, "svelte.config.js") {
		return "SvelteKit", packageManager, nil
	}

	if fileExists(projectDir, "remix.config.js") {
		return "Remix", packageManager, nil
	}

	if fileExists(projectDir, "astro.config.mjs") {
		return "Astro", packageManager, nil
	}

	if fileExists(projectDir, "nest-cli.json") {
		return "NestJS", packageManager, nil
	}

	// Check package.json for framework hints
	if framework := detectFrameworkFromPackageJSON(projectDir); framework != "" {
		return framework, packageManager, nil
	}

	// Default to generic Node.js
	return "Node.js", packageManager, nil
}

// detectPythonFramework detects Python framework.
func detectPythonFramework(projectDir string) (string, string, error) {
	// Detect package manager
	packageManager := detector.DetectPythonPackageManager(projectDir)

	// Detect framework
	if fileExists(projectDir, "manage.py") {
		return "Django", packageManager, nil
	}

	if containsImport(projectDir, "FastAPI") {
		return "FastAPI", packageManager, nil
	}

	if containsImport(projectDir, "Flask") {
		return "Flask", packageManager, nil
	}

	if containsImport(projectDir, "streamlit") {
		return "Streamlit", packageManager, nil
	}

	if containsImport(projectDir, "gradio") {
		return "Gradio", packageManager, nil
	}

	// Default to generic Python
	return "Python", packageManager, nil
}

// detectDotNetFramework detects .NET framework.
func detectDotNetFramework(projectDir string) (string, string, error) {
	// Check for Aspire
	if fileExists(projectDir, "AppHost.cs") {
		return "Aspire", "dotnet", nil
	}

	// Check for ASP.NET Core
	if hasFileWithExt(projectDir, ".csproj") {
		// Read csproj to detect Web SDK
		csprojFiles, _ := filepath.Glob(filepath.Join(projectDir, "*.csproj"))
		for _, csprojFile := range csprojFiles {
			if containsText(csprojFile, "Microsoft.NET.Sdk.Web") {
				return "ASP.NET Core", "dotnet", nil
			}
		}
	}

	// Default to generic .NET
	return ".NET", "dotnet", nil
}

// detectJavaFramework detects Java framework.
func detectJavaFramework(projectDir string) (string, string, error) {
	packageManager := "maven"
	if fileExists(projectDir, "build.gradle") || fileExists(projectDir, "build.gradle.kts") {
		packageManager = "gradle"
	}

	// Check for Spring Boot
	if fileExists(projectDir, "pom.xml") {
		if containsText(filepath.Join(projectDir, "pom.xml"), "spring-boot") {
			return "Spring Boot", packageManager, nil
		}
	}

	if fileExists(projectDir, "build.gradle") {
		buildGradle := filepath.Join(projectDir, "build.gradle")
		if containsText(buildGradle, "spring-boot") {
			return "Spring Boot", packageManager, nil
		}
		if containsText(buildGradle, "quarkus") {
			return "Quarkus", packageManager, nil
		}
	}

	return "Java", packageManager, nil
}

// detectPHPFramework detects PHP framework.
func detectPHPFramework(projectDir string) (string, string, error) {
	if fileExists(projectDir, "artisan") {
		return "Laravel", "composer", nil
	}

	return "PHP", "composer", nil
}

// buildRunCommand builds the command and arguments to run the service.
// If entrypoint is provided (from azure.yaml), it takes precedence over auto-detection.
func buildRunCommand(runtime *ServiceRuntime, projectDir string, entrypoint string, runtimeMode string) error {
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

	case "Express":
		runtime.Command = runtime.PackageManager
		// Try dev first, fall back to start
		if hasScript(projectDir, "dev") {
			runtime.Args = []string{"run", "dev"}
		} else {
			runtime.Args = []string{"run", "start"}
		}

	case "Node.js":
		runtime.Command = runtime.PackageManager
		// Try dev first, fall back to start
		if hasScript(projectDir, "dev") {
			runtime.Args = []string{"run", "dev"}
		} else {
			runtime.Args = []string{"run", "start"}
		}

	case "Django":
		runtime.Command = "python"
		runtime.Args = []string{"manage.py", "runserver", fmt.Sprintf("0.0.0.0:%d", runtime.Port)}

	case "FastAPI":
		runtime.Command = "uvicorn"
		// Use entrypoint if provided, otherwise find the app file
		appFile := entrypoint
		if appFile == "" {
			appFile = findPythonAppFile(projectDir)
		}
		// Validate that the entrypoint file exists
		if err := validatePythonEntrypoint(projectDir, appFile); err != nil {
			return err
		}
		runtime.Args = []string{appFile + ":app", "--reload", "--host", "0.0.0.0", "--port", fmt.Sprintf("%d", runtime.Port)}

	case "Flask":
		runtime.Command = "python"
		runtime.Args = []string{"-m", "flask", "run", "--host", "0.0.0.0", "--port", fmt.Sprintf("%d", runtime.Port)}
		// Use entrypoint if provided, otherwise find the app file
		var appFile string
		if entrypoint != "" {
			appFile = entrypoint
			runtime.Env["FLASK_APP"] = entrypoint
		} else {
			appFile = findPythonAppFile(projectDir)
			runtime.Env["FLASK_APP"] = appFile
		}
		// Validate that the entrypoint file exists
		if err := validatePythonEntrypoint(projectDir, appFile); err != nil {
			return err
		}
		runtime.Env["FLASK_ENV"] = "development"

	case "Streamlit":
		runtime.Command = "streamlit"
		// Use entrypoint if provided, otherwise find the app file
		appFile := entrypoint
		if appFile == "" {
			appFile = findPythonAppFile(projectDir)
		}
		// Validate that the entrypoint file exists
		if err := validatePythonEntrypoint(projectDir, appFile); err != nil {
			return err
		}
		runtime.Args = []string{"run", appFile + ".py", "--server.port", fmt.Sprintf("%d", runtime.Port)}

	case "Python":
		runtime.Command = "python"
		// Use entrypoint if provided, otherwise find the app file
		appFile := entrypoint
		if appFile == "" {
			appFile = findPythonAppFile(projectDir)
		}
		// Validate that the entrypoint file exists
		if err := validatePythonEntrypoint(projectDir, appFile); err != nil {
			return err
		}
		runtime.Args = []string{appFile + ".py"}

	case "Aspire":
		runtime.Command = "dotnet"
		// Find AppHost.csproj
		csprojFiles, _ := filepath.Glob(filepath.Join(projectDir, "*.csproj"))
		if len(csprojFiles) > 0 {
			// In aspire mode, use dotnet run to get native Aspire dashboard
			// In azd mode, run individual services separately
			if runtimeMode == "aspire" {
				runtime.Args = []string{"run", "--project", csprojFiles[0]}
			} else {
				// In azd mode, we run services individually, not the AppHost
				runtime.Args = []string{"run", "--project", csprojFiles[0], "--no-launch-profile"}
			}
		} else {
			runtime.Args = []string{"run"}
		}

	case "ASP.NET Core", ".NET":
		runtime.Command = "dotnet"
		// Find .csproj file
		csprojFiles, _ := filepath.Glob(filepath.Join(projectDir, "*.csproj"))
		if len(csprojFiles) > 0 {
			runtime.Args = []string{"run", "--project", csprojFiles[0]}
		} else {
			runtime.Args = []string{"run"}
		}

	case "Spring Boot":
		if runtime.PackageManager == "maven" {
			runtime.Command = "mvn"
			runtime.Args = []string{"spring-boot:run"}
		} else {
			runtime.Command = "gradle"
			runtime.Args = []string{"bootRun"}
		}

	case "Java":
		if runtime.PackageManager == "maven" {
			runtime.Command = "mvn"
			runtime.Args = []string{"exec:java"}
		} else {
			runtime.Command = "gradle"
			runtime.Args = []string{"run"}
		}

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

// configureHealthCheck sets up health check configuration based on framework.
func configureHealthCheck(runtime *ServiceRuntime) {
	switch runtime.Framework {
	case "Aspire":
		runtime.HealthCheck.Path = "/"
		runtime.HealthCheck.LogMatch = "Now listening on"
	case "Next.js":
		runtime.HealthCheck.Path = "/"
		runtime.HealthCheck.LogMatch = "ready on"
	case "Django":
		runtime.HealthCheck.Path = "/"
		runtime.HealthCheck.LogMatch = "Starting development server"
	case "Spring Boot":
		runtime.HealthCheck.Path = "/actuator/health"
		runtime.HealthCheck.LogMatch = "Started"
	case "FastAPI":
		runtime.HealthCheck.Path = "/docs"
	default:
		runtime.HealthCheck.Path = "/"
	}
}

// Helper functions

func fileExists(dir string, filename string) bool {
	path := filepath.Join(dir, filename)
	if err := security.ValidatePath(path); err != nil {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func hasFileWithExt(dir string, ext string) bool {
	pattern := filepath.Join(dir, "*"+ext)
	matches, err := filepath.Glob(pattern)
	return err == nil && len(matches) > 0
}

func containsText(filePath string, text string) bool {
	if err := security.ValidatePath(filePath); err != nil {
		return false
	}
	// #nosec G304 -- Path validated by security.ValidatePath
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), text)
}

func containsImport(projectDir string, importName string) bool {
	// Check common Python entry points
	for _, filename := range []string{"main.py", "app.py", "src/main.py", "src/app.py"} {
		filePath := filepath.Join(projectDir, filename)
		if containsText(filePath, importName) {
			return true
		}
	}
	return false
}

func detectFrameworkFromPackageJSON(projectDir string) string {
	packageJSONPath := filepath.Join(projectDir, "package.json")
	if err := security.ValidatePath(packageJSONPath); err != nil {
		return ""
	}

	// #nosec G304 -- Path validated by security.ValidatePath
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return ""
	}

	content := string(data)
	if strings.Contains(content, "\"react\"") {
		return "React"
	}
	if strings.Contains(content, "\"vue\"") {
		return "Vue"
	}
	if strings.Contains(content, "\"express\"") {
		return "Express"
	}

	return ""
}

func hasScript(projectDir string, scriptName string) bool {
	packageJSONPath := filepath.Join(projectDir, "package.json")
	if containsText(packageJSONPath, fmt.Sprintf(`"%s"`, scriptName)) {
		return true
	}
	return false
}

func findPythonAppFile(projectDir string) string {
	// Try common entry points
	for _, filename := range []string{"main", "app", "src/main", "src/app"} {
		if fileExists(projectDir, filename+".py") || fileExists(projectDir, filepath.Join("src", filename+".py")) {
			return filename
		}
	}
	return "main"
}

// validatePythonEntrypoint checks if the Python entrypoint file exists and provides helpful error messages.
func validatePythonEntrypoint(projectDir string, appFile string) error {
	// Try different file path variations
	possiblePaths := []string{
		filepath.Join(projectDir, appFile),
		filepath.Join(projectDir, appFile+".py"),
	}

	// Check if file exists
	for _, path := range possiblePaths {
		if err := security.ValidatePath(path); err == nil {
			if _, err := os.Stat(path); err == nil {
				return nil // File exists
			}
		}
	}

	// File doesn't exist - provide helpful error message
	expectedPath := filepath.Join(projectDir, appFile+".py")
	return fmt.Errorf(
		"Python entrypoint file not found: %s\n"+
			"Expected file: %s\n"+
			"Please ensure the file exists or specify the correct entrypoint in azure.yaml using:\n"+
			"  entrypoint: <filename>",
		appFile,
		expectedPath,
	)
}

func normalizeLanguage(language string) string {
	lower := strings.ToLower(language)
	switch lower {
	case "js", "javascript", "node", "nodejs", "node.js":
		return "JavaScript"
	case "ts", "typescript":
		return "TypeScript"
	case "py", "python":
		return "Python"
	case "cs", "csharp", "c#":
		return ".NET"
	case "dotnet", ".net":
		return ".NET"
	case "java":
		return "Java"
	case "go", "golang":
		return "Go"
	case "rs", "rust":
		return "Rust"
	case "php":
		return "PHP"
	case "docker":
		return "Docker"
	default:
		return language
	}
}
