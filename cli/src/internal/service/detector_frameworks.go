// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"path/filepath"

	"github.com/jongio/azd-app/cli/src/internal/detector"
)

const (
	frameworkDocker    = "Docker"
	packageMgrDocker   = "docker"
	langNameJavaScript = "JavaScript"
	langTypeScript     = "TypeScript"
	langNamePython     = "Python"
	langNameDotNet     = ".NET"
	langNameJava       = "Java"
	langNameRust       = "Rust"
	langNamePHP        = "PHP"
	watchModeNone      = "none"
	langDotnet         = "dotnet"
)

// detectLanguage determines the programming language used by the service.
func detectLanguage(projectDir string, host string) (string, error) {
	// Define language detection rules in priority order
	languageRules := []struct {
		name      string
		checkFunc func() bool
	}{
		{langTypeScript, func() bool {
			return fileExists(projectDir, "package.json") && fileExists(projectDir, "tsconfig.json")
		}},
		{langNameJavaScript, func() bool {
			return fileExists(projectDir, "package.json")
		}},
		{langNamePython, func() bool {
			return fileExists(projectDir, "requirements.txt") ||
				fileExists(projectDir, "pyproject.toml") ||
				fileExists(projectDir, "poetry.lock") ||
				fileExists(projectDir, "uv.lock")
		}},
		{langNameDotNet, func() bool {
			return hasFileWithExt(projectDir, ".csproj") ||
				hasFileWithExt(projectDir, ".sln") ||
				hasFileWithExt(projectDir, ".fsproj")
		}},
		{langNameJava, func() bool {
			return fileExists(projectDir, "pom.xml") ||
				fileExists(projectDir, "build.gradle") ||
				fileExists(projectDir, "build.gradle.kts")
		}},
		{"Go", func() bool { return fileExists(projectDir, "go.mod") }},
		{langNameRust, func() bool { return fileExists(projectDir, "Cargo.toml") }},
		{langNamePHP, func() bool { return fileExists(projectDir, "composer.json") }},
		{frameworkDocker, func() bool {
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
		return frameworkDocker, nil
	}

	return "", errCouldNotDetectLanguage(projectDir)
}

// detectFrameworkAndPackageManager detects the specific framework and package manager.
func detectFrameworkAndPackageManager(projectDir string, language string) (string, string, error) {
	switch language {
	case langTypeScript, langNameJavaScript:
		return detectNodeFramework(projectDir)
	case langNamePython:
		return detectPythonFramework(projectDir)
	case langNameDotNet:
		return detectDotNetFramework(projectDir)
	case langNameJava:
		return detectJavaFramework(projectDir)
	case "Go":
		return "Go", "go", nil
	case langNameRust:
		return langNameRust, "cargo", nil
	case langNamePHP:
		return detectPHPFramework(projectDir)
	case frameworkDocker:
		return frameworkDocker, packageMgrDocker, nil
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
	return langNamePython, packageManager, nil
}

// detectDotNetFramework detects .NET framework.
func detectDotNetFramework(projectDir string) (string, string, error) {
	// Check for Aspire
	if fileExists(projectDir, "AppHost.cs") {
		return "Aspire", langDotnet, nil
	}

	// Check for ASP.NET Core
	if hasFileWithExt(projectDir, ".csproj") {
		// Read csproj to detect Web SDK
		csprojFiles, _ := filepath.Glob(filepath.Join(projectDir, "*.csproj"))
		for _, csprojFile := range csprojFiles {
			if containsText(csprojFile, "Microsoft.NET.Sdk.Web") {
				return "ASP.NET Core", langDotnet, nil
			}
		}
	}

	// Default to generic .NET
	return langNameDotNet, langDotnet, nil
}

// detectJavaFramework detects Java framework.
func detectJavaFramework(projectDir string) (string, string, error) {
	packageManager := "maven"
	if fileExists(projectDir, "build.gradle") || fileExists(projectDir, "build.gradle.kts") {
		packageManager = "gradle"
	}

	// Check for Spring Boot in pom.xml
	if fileExists(projectDir, "pom.xml") && containsText(filepath.Join(projectDir, "pom.xml"), "spring-boot") {
		return frameworkSpringBoot, packageManager, nil
	}

	// Check for frameworks in build.gradle
	if fileExists(projectDir, "build.gradle") {
		buildGradle := filepath.Join(projectDir, "build.gradle")
		if containsText(buildGradle, "spring-boot") {
			return frameworkSpringBoot, packageManager, nil
		}
		if containsText(buildGradle, "quarkus") {
			return "Quarkus", packageManager, nil
		}
	}

	return langNameJava, packageManager, nil
}

// detectPHPFramework detects PHP framework.
func detectPHPFramework(projectDir string) (string, string, error) {
	if fileExists(projectDir, "artisan") {
		return "Laravel", "composer", nil
	}

	return langNamePHP, "composer", nil
}
