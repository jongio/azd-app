package vscode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// DebugConfiguration represents a VS Code debug configuration.
type DebugConfiguration struct {
	Type         string                 `json:"type"`
	Request      string                 `json:"request"`
	Name         string                 `json:"name"`
	Address      string                 `json:"address,omitempty"`
	Port         int                    `json:"port,omitempty"`
	SkipFiles    []string               `json:"skipFiles,omitempty"`
	Connect      map[string]interface{} `json:"connect,omitempty"`
	ProcessId    string                 `json:"processId,omitempty"`
	ProcessName  string                 `json:"processName,omitempty"`
	Mode         string                 `json:"mode,omitempty"`
	Host         string                 `json:"host,omitempty"`
	PathMappings []PathMapping          `json:"pathMappings,omitempty"`
	Cwd          string                 `json:"cwd,omitempty"`
	Program      string                 `json:"program,omitempty"`
}

// PathMapping represents a path mapping for remote debugging.
type PathMapping struct {
	LocalRoot  string `json:"localRoot"`
	RemoteRoot string `json:"remoteRoot"`
}

// Compound represents a compound debug configuration.
type Compound struct {
	Name           string                 `json:"name"`
	Configurations []string               `json:"configurations"`
	PreLaunchTask  string                 `json:"preLaunchTask,omitempty"`
	Presentation   map[string]interface{} `json:"presentation,omitempty"`
}

// LaunchConfig represents the launch.json file.
type LaunchConfig struct {
	Version        string               `json:"version"`
	Configurations []DebugConfiguration `json:"configurations"`
	Compounds      []Compound           `json:"compounds,omitempty"`
}

// Task represents a VS Code task.
type Task struct {
	Label          string         `json:"label"`
	Type           string         `json:"type"`
	Command        string         `json:"command"`
	IsBackground   bool           `json:"isBackground"`
	ProblemMatcher ProblemMatcher `json:"problemMatcher"`
}

// ProblemMatcher represents a VS Code problem matcher.
type ProblemMatcher struct {
	Pattern    Pattern    `json:"pattern"`
	Background Background `json:"background"`
}

// Pattern represents a problem matcher pattern.
type Pattern struct {
	Regexp string `json:"regexp"`
}

// Background represents background task configuration.
type Background struct {
	ActiveOnStart bool   `json:"activeOnStart"`
	BeginsPattern string `json:"beginsPattern"`
	EndsPattern   string `json:"endsPattern"`
}

// TasksConfig represents the tasks.json file.
type TasksConfig struct {
	Version string `json:"version"`
	Tasks   []Task `json:"tasks"`
}

// ServiceDebugInfo holds debug information for a service.
type ServiceDebugInfo struct {
	Name     string
	Language string
	Port     int
}

// GetDebugPort returns the debug port for a language with an offset for multiple services.
// This is a convenience wrapper around service.GetDebugPort.
func GetDebugPort(language string, offset int) int {
	return service.GetDebugPort(language, offset)
}

// EnsureDebugConfig generates VS Code debug configurations if they don't exist.
// Returns true if configurations were generated (first time or regenerated).
func EnsureDebugConfig(projectDir string, services []ServiceDebugInfo, force bool) (bool, error) {
	vscodeDir := filepath.Join(projectDir, ".vscode")
	launchPath := filepath.Join(vscodeDir, "launch.json")
	tasksPath := filepath.Join(vscodeDir, "tasks.json")

	// Skip if both files already exist (unless --regenerate-debug-config)
	if !force {
		launchExists := false
		tasksExists := false

		if _, err := os.Stat(launchPath); err == nil {
			launchExists = true
		}
		if _, err := os.Stat(tasksPath); err == nil {
			tasksExists = true
		}

		// Only skip if both files exist
		if launchExists && tasksExists {
			return false, nil
		}
	}

	if err := os.MkdirAll(vscodeDir, 0755); err != nil {
		return false, fmt.Errorf("failed to create .vscode directory: %w", err)
	}

	// Generate launch.json
	launch := generateLaunchJSON(services)
	if err := writeLaunchJSON(launchPath, launch); err != nil {
		return false, err
	}

	// Generate tasks.json
	tasks := generateTasksJSON()
	if err := writeTasksJSON(tasksPath, tasks); err != nil {
		return false, err
	}

	output.Success("✅ Debug configuration created!")
	return true, nil
}

// generateLaunchJSON creates the launch.json configuration.
func generateLaunchJSON(services []ServiceDebugInfo) LaunchConfig {
	configs := []DebugConfiguration{}
	compoundConfigNames := []string{}

	// Generate individual attach configs for each service
	for _, svc := range services {
		// Normalize language for debug
		normalizedLang := service.NormalizeLanguageForDebug(svc.Language)
		// Use the port that was already assigned in run.go
		debugPort := svc.Port

		configName := fmt.Sprintf("🔌 %s (%s)", svc.Name, normalizedLang)
		compoundConfigNames = append(compoundConfigNames, configName)

		config := createDebugConfig(svc.Name, normalizedLang, debugPort)
		if config.Type != "" {
			configs = append(configs, config)
		}
	}

	// Create compound configurations
	compounds := []Compound{}

	// Add "Debug ALL (already running)" compound
	if len(compoundConfigNames) > 0 {
		compounds = append(compounds, Compound{
			Name:           "🔌 Attach to ALL (already running)",
			Configurations: compoundConfigNames,
			Presentation: map[string]interface{}{
				"hidden": false,
				"group":  "",
			},
		})
	}

	// Add "Debug ALL Services" compound with preLaunchTask
	if len(compoundConfigNames) > 0 {
		compounds = append(compounds, Compound{
			Name:           "🚀 Debug ALL Services",
			Configurations: compoundConfigNames,
			PreLaunchTask:  "azd: Start Services (Debug)",
			Presentation: map[string]interface{}{
				"hidden": false,
				"group":  "",
			},
		})
	}

	return LaunchConfig{
		Version:        "0.2.0",
		Configurations: configs,
		Compounds:      compounds,
	}
}

// createDebugConfig creates a debug configuration for a specific language.
func createDebugConfig(serviceName, language string, debugPort int) DebugConfiguration {
	configName := fmt.Sprintf("🔌 %s (%s)", serviceName, language)

	switch language {
	case "node":
		return DebugConfiguration{
			Type:    "node",
			Request: "attach",
			Name:    configName,
			Address: "localhost",
			Port:    debugPort,
			SkipFiles: []string{
				"<node_internals>/**",
			},
		}

	case "python":
		return DebugConfiguration{
			Type:    "debugpy",
			Request: "attach",
			Name:    configName,
			Connect: map[string]interface{}{
				"host": "localhost",
				"port": debugPort,
			},
			PathMappings: []PathMapping{
				{
					LocalRoot:  "${workspaceFolder}",
					RemoteRoot: ".",
				},
			},
		}

	case "go":
		return DebugConfiguration{
			Type:    "go",
			Request: "attach",
			Name:    configName,
			Mode:    "remote",
			Host:    "localhost",
			Port:    debugPort,
		}

	case "dotnet":
		return DebugConfiguration{
			Type:        "coreclr",
			Request:     "attach",
			Name:        configName,
			ProcessName: serviceName,
		}

	case "java":
		return DebugConfiguration{
			Type:    "java",
			Request: "attach",
			Name:    configName,
			Host:    "localhost",
			Port:    debugPort,
		}

	default:
		// Return empty config for unsupported languages
		return DebugConfiguration{}
	}
}

// generateTasksJSON creates the tasks.json configuration.
func generateTasksJSON() TasksConfig {
	return TasksConfig{
		Version: "2.0.0",
		Tasks: []Task{
			{
				Label:        "azd: Start Services (Debug)",
				Type:         "shell",
				Command:      "azd app debug",
				IsBackground: true,
				ProblemMatcher: ProblemMatcher{
					Pattern: Pattern{
						Regexp: "^.*$",
					},
					Background: Background{
						ActiveOnStart: true,
						BeginsPattern: "🐛 Starting services in debug mode",
						EndsPattern:   "📊 Dashboard:",
					},
				},
			},
		},
	}
}

// writeLaunchJSON writes the launch.json file.
func writeLaunchJSON(path string, config LaunchConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal launch.json: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write launch.json: %w", err)
	}

	return nil
}

// writeTasksJSON writes the tasks.json file.
func writeTasksJSON(path string, config TasksConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks.json: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write tasks.json: %w", err)
	}

	return nil
}
