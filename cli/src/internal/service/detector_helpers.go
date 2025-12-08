// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/fileutil"
	"github.com/jongio/azd-app/cli/src/internal/security"
)

// Helper functions for detector
// Note: fileExists, hasFileWithExt, containsText moved to internal/fileutil package

// fileExists is a convenience wrapper for fileutil.FileExists
func fileExists(dir string, filename string) bool {
	return fileutil.FileExists(dir, filename)
}

// hasFileWithExt is a convenience wrapper for fileutil.HasFileWithExt
func hasFileWithExt(dir string, ext string) bool {
	return fileutil.HasFileWithExt(dir, ext)
}

// containsText is a convenience wrapper for fileutil.ContainsText
func containsText(filePath string, text string) bool {
	return fileutil.ContainsText(filePath, text)
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
		"python entrypoint file not found: %s\n"+
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
	case "logicapp", "logicapps", "logic-app", "logic-apps":
		return "Logic Apps"
	default:
		return language
	}
}

// errCouldNotDetectLanguage returns an error when language detection fails.
func errCouldNotDetectLanguage(projectDir string) error {
	return fmt.Errorf("could not detect language in %s", projectDir)
}
