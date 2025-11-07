// Package service provides runtime detection and service orchestration capabilities.
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
	port, shouldUpdateAzureYaml, err := portMgr.AssignPort(serviceName, preferredPort, isExplicit, true) // isExplicit, cleanStale
	if err != nil {
		return nil, fmt.Errorf("failed to assign port: %w", err)
	}
	runtime.Port = port
	runtime.ShouldUpdateAzureYaml = shouldUpdateAzureYaml // Track if user wants azure.yaml updated
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
	// Define language detection rules in priority order
	languageRules := []struct {
		name      string
		checkFunc func() bool
	}{
		{"TypeScript", func() bool {
			return fileExists(projectDir, "package.json") && fileExists(projectDir, "tsconfig.json")
		}},
		{"JavaScript", func() bool {
			return fileExists(projectDir, "package.json")
		}},
		{"Python", func() bool {
			return fileExists(projectDir, "requirements.txt") ||
				fileExists(projectDir, "pyproject.toml") ||
				fileExists(projectDir, "poetry.lock") ||
				fileExists(projectDir, "uv.lock")
		}},
		{".NET", func() bool {
			return hasFileWithExt(projectDir, ".csproj") ||
				hasFileWithExt(projectDir, ".sln") ||
				hasFileWithExt(projectDir, ".fsproj")
		}},
		{"Java", func() bool {
			return fileExists(projectDir, "pom.xml") ||
				fileExists(projectDir, "build.gradle") ||
				fileExists(projectDir, "build.gradle.kts")
		}},
		{"Go", func() bool { return fileExists(projectDir, "go.mod") }},
		{"Rust", func() bool { return fileExists(projectDir, "Cargo.toml") }},
		{"PHP", func() bool { return fileExists(projectDir, "composer.json") }},
		{"Docker", func() bool {
			return fileExists(projectDir, "Dockerfile") || fileExists(projectDir, "docker-compose.yml")
		}},
	}

	// Check each rule in order
	for _, rule := range languageRules {
		if rule.checkFunc() {
			return rule.name, nil
		}
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
	packageManager := detector.DetectNodePackageManagerWithBoundary(projectDir, projectDir)

	// Framework detection rules in priority order
	frameworkRules := []struct {
		name      string
		checkFunc func() bool
	}{
		{"Next.js", func() bool {
			return fileExists(projectDir, "next.config.js") ||
				fileExists(projectDir, "next.config.ts") ||
				fileExists(projectDir, "next.config.mjs")
		}},
		{"Angular", func() bool { return fileExists(projectDir, "angular.json") }},
		{"Nuxt", func() bool {
			return fileExists(projectDir, "nuxt.config.ts") || fileExists(projectDir, "nuxt.config.js")
		}},
		{"React", func() bool {
			return fileExists(projectDir, "vite.config.ts") || fileExists(projectDir, "vite.config.js")
		}},
		{"SvelteKit", func() bool { return fileExists(projectDir, "svelte.config.js") }},
		{"Remix", func() bool { return fileExists(projectDir, "remix.config.js") }},
		{"Astro", func() bool { return fileExists(projectDir, "astro.config.mjs") }},
		{"NestJS", func() bool { return fileExists(projectDir, "nest-cli.json") }},
	}

	// Check each framework rule
	for _, rule := range frameworkRules {
		if rule.checkFunc() {
			return rule.name, packageManager, nil
		}
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
	packageManager := detector.DetectPythonPackageManager(projectDir)

	// Framework detection rules in priority order
	frameworkRules := []struct {
		name      string
		checkFunc func() bool
	}{
		{"Django", func() bool { return fileExists(projectDir, "manage.py") }},
		{"FastAPI", func() bool { return containsImport(projectDir, "FastAPI") }},
		{"Flask", func() bool { return containsImport(projectDir, "Flask") }},
		{"Streamlit", func() bool { return containsImport(projectDir, "streamlit") }},
		{"Gradio", func() bool { return containsImport(projectDir, "gradio") }},
	}

	// Check each framework rule
	for _, rule := range frameworkRules {
		if rule.checkFunc() {
			return rule.name, packageManager, nil
		}
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

	// Check for Spring Boot in pom.xml
	if fileExists(projectDir, "pom.xml") && containsText(filepath.Join(projectDir, "pom.xml"), "spring-boot") {
		return "Spring Boot", packageManager, nil
	}

	// Check for frameworks in build.gradle
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

// buildPythonCommand configures a Python service runtime with the appropriate command and arguments.
func buildPythonCommand(runtime *ServiceRuntime, projectDir, entrypoint, pythonCmd string) error {
	runtime.Command = pythonCmd

	switch runtime.Framework {
	case "Django":
		// Django uses manage.py - validate it exists
		managePyPath := filepath.Join(projectDir, "manage.py")
		if err := security.ValidatePath(managePyPath); err != nil {
			return fmt.Errorf("Django: invalid manage.py path: %w", err)
		}
		if _, err := os.Stat(managePyPath); err != nil {
			return fmt.Errorf("Django: manage.py not found at %s", managePyPath)
		}
		runtime.Args = []string{"manage.py", "runserver", fmt.Sprintf("0.0.0.0:%d", runtime.Port)}
		return nil

	case "FastAPI":
		appFile, err := resolvePythonEntrypoint(projectDir, entrypoint)
		if err != nil {
			return fmt.Errorf("FastAPI: %w", err)
		}
		// Use -m uvicorn to run as module from venv
		// FastAPI uses module:app format, no .py extension needed
		runtime.Args = []string{"-m", "uvicorn", appFile + ":app", "--reload", "--host", "0.0.0.0", "--port", fmt.Sprintf("%d", runtime.Port)}
		return nil

	case "Flask":
		appFile, err := resolvePythonEntrypoint(projectDir, entrypoint)
		if err != nil {
			return fmt.Errorf("Flask: %w", err)
		}
		runtime.Args = []string{"-m", "flask", "run", "--host", "0.0.0.0", "--port", fmt.Sprintf("%d", runtime.Port)}
		// Flask needs the .py extension in FLASK_APP
		if entrypoint != "" {
			runtime.Env["FLASK_APP"] = entrypoint
		} else {
			runtime.Env["FLASK_APP"] = appFile + ".py"
		}
		runtime.Env["FLASK_ENV"] = "development"
		return nil

	case "Streamlit":
		appFile, err := resolvePythonEntrypoint(projectDir, entrypoint)
		if err != nil {
			return fmt.Errorf("Streamlit: %w", err)
		}
		// Use -m streamlit to run as module from venv
		runtime.Args = []string{"-m", "streamlit", "run", appFile + ".py", "--server.port", fmt.Sprintf("%d", runtime.Port)}
		return nil

	case "Gradio", "Python":
		appFile, err := resolvePythonEntrypoint(projectDir, entrypoint)
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

// buildRunCommand builds the command and arguments to run the service.
// If entrypoint is provided (from azure.yaml), it takes precedence over auto-detection.
func buildRunCommand(runtime *ServiceRuntime, projectDir string, entrypoint string, runtimeMode string) error {
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
		return buildPythonCommand(runtime, projectDir, entrypoint, pythonCmd)
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

// configureHealthCheck sets up health check configuration based on framework.
func configureHealthCheck(runtime *ServiceRuntime) {
	healthConfigs := map[string]struct {
		path     string
		logMatch string
	}{
		"Aspire":      {"/", "Now listening on"},
		"Next.js":     {"/", "ready on"},
		"Django":      {"/", "Starting development server"},
		"Spring Boot": {"/actuator/health", "Started"},
		"FastAPI":     {"/docs", ""},
	}

	if config, exists := healthConfigs[runtime.Framework]; exists {
		runtime.HealthCheck.Path = config.path
		runtime.HealthCheck.LogMatch = config.logMatch
	} else {
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
	return containsText(packageJSONPath, fmt.Sprintf(`"%s"`, scriptName))
}

func findPythonAppFile(projectDir string) string {
	// Try common entry points (without .py extension)
	for _, filename := range []string{"main.py", "app.py", "src/main.py", "src/app.py"} {
		if fileExists(projectDir, filename) {
			// Return without .py extension for consistency
			return strings.TrimSuffix(filename, ".py")
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
