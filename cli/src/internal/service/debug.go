package service

import (
	"fmt"
	"os/exec"
	"strings"
)

// Debug port mappings for each language
var defaultDebugPorts = map[string]int{
	"node":   9229,
	"python": 5678,
	"go":     2345,
	"dotnet": 5005,
	"java":   5005,
	"rust":   4711,
}

// Debug protocol mappings for each language
var debugProtocols = map[string]string{
	"node":   "inspector",
	"python": "debugpy",
	"go":     "delve",
	"dotnet": "coreclr",
	"java":   "jdwp",
	"rust":   "lldb",
}

// GetDebugPort returns the default debug port for a language with offset.
// This is exported for use by other packages (e.g., vscode).
func GetDebugPort(language string, offset int) int {
	normalizedLang := normalizeLanguageForDebug(language)
	return getDebugPort(normalizedLang, offset)
}

// GetDebugProtocol returns the debug protocol for a language.
// This is exported for use by other packages (e.g., vscode).
func GetDebugProtocol(language string) string {
	normalizedLang := normalizeLanguageForDebug(language)
	return getDebugProtocol(normalizedLang)
}

// ConfigureDebug sets up debug configuration for a service runtime.
func ConfigureDebug(runtime *ServiceRuntime, debugEnabled bool, waitForDebugger bool, languageIndex int) {
	if !debugEnabled {
		runtime.Debug = DebugConfig{Enabled: false}
		return
	}

	normalizedLang := normalizeLanguageForDebug(runtime.Language)
	debugPort := getDebugPort(normalizedLang, languageIndex)
	protocol := getDebugProtocol(normalizedLang)
	debugURL := getDebugURL(normalizedLang, debugPort)

	runtime.Debug = DebugConfig{
		Enabled:         true,
		Port:            debugPort,
		Protocol:        protocol,
		URL:             debugURL,
		WaitForDebugger: waitForDebugger,
	}
}

// ApplyDebugFlags modifies the command to inject debug flags.
func ApplyDebugFlags(runtime *ServiceRuntime, cmd *exec.Cmd) error {
	if !runtime.Debug.Enabled {
		return nil
	}

	normalizedLang := normalizeLanguageForDebug(runtime.Language)
	debugPort := runtime.Debug.Port
	waitForDebugger := runtime.Debug.WaitForDebugger

	switch normalizedLang {
	case "node":
		flag := fmt.Sprintf("--inspect=0.0.0.0:%d", debugPort)
		if waitForDebugger {
			flag = fmt.Sprintf("--inspect-brk=0.0.0.0:%d", debugPort)
		}
		// Insert debug flag after node executable
		cmd.Args = append([]string{cmd.Args[0], flag}, cmd.Args[1:]...)

	case "python":
		// For Python, we need to use debugpy module
		debugArgs := []string{"-m", "debugpy", "--listen", fmt.Sprintf("0.0.0.0:%d", debugPort)}
		if waitForDebugger {
			debugArgs = append(debugArgs, "--wait-for-client")
		}
		// Insert debugpy args after python executable
		cmd.Args = append([]string{cmd.Args[0]}, append(debugArgs, cmd.Args[1:]...)...)

	case "go":
		// For Go, we need to replace the command with dlv
		dlvPath, err := exec.LookPath("dlv")
		if err != nil {
			return fmt.Errorf("could not find 'dlv' debugger in PATH: %w", err)
		}
		originalArgs := cmd.Args[1:] // Save original args
		cmd.Path = dlvPath
		continueFlag := "true"
		if waitForDebugger {
			continueFlag = "false"
		}
		cmd.Args = []string{
			"dlv", "debug",
			"--headless",
			"--listen", fmt.Sprintf(":%d", debugPort),
			"--api-version=2",
			"--accept-multiclient",
			fmt.Sprintf("--continue=%s", continueFlag),
		}
		// Append original args if any
		cmd.Args = append(cmd.Args, originalArgs...)

	case "dotnet":
		// .NET uses vsdbg - attach by PID, no startup flags needed
		// Debug info includes PID for process picker
		// No changes to command needed

	case "java":
		// For Java, add JDWP agent to JAVA_TOOL_OPTIONS
		suspendFlag := "n"
		if waitForDebugger {
			suspendFlag = "y"
		}
		jdwpFlag := fmt.Sprintf("-agentlib:jdwp=transport=dt_socket,server=y,suspend=%s,address=*:%d",
			suspendFlag, debugPort)

		// Check if JAVA_TOOL_OPTIONS already exists
		javaToolOptions := jdwpFlag
		for i, env := range cmd.Env {
			if strings.HasPrefix(env, "JAVA_TOOL_OPTIONS=") {
				// Append to existing JAVA_TOOL_OPTIONS
				javaToolOptions = env[len("JAVA_TOOL_OPTIONS="):] + " " + jdwpFlag
				cmd.Env[i] = "JAVA_TOOL_OPTIONS=" + javaToolOptions
				return nil
			}
		}
		// Add new JAVA_TOOL_OPTIONS
		cmd.Env = append(cmd.Env, "JAVA_TOOL_OPTIONS="+javaToolOptions)

	case "rust":
		// Rust debugging typically uses lldb/gdb
		// For now, no automatic setup - may require cargo-watch + lldb-server
		// Users will need to manually configure for Rust
	}

	return nil
}

// getDebugPort returns the default debug port for a language with offset.
func getDebugPort(language string, offset int) int {
	basePort := defaultDebugPorts[language]
	if basePort == 0 {
		basePort = 9000 // fallback
	}
	return basePort + offset
}

// getDebugProtocol returns the debug protocol for a language.
func getDebugProtocol(language string) string {
	protocol := debugProtocols[language]
	if protocol == "" {
		protocol = "unknown"
	}
	return protocol
}

// getDebugURL returns the debug URL for a language and port.
func getDebugURL(language string, port int) string {
	switch language {
	case "node":
		return fmt.Sprintf("ws://localhost:%d", port)
	case "python", "go", "java":
		return fmt.Sprintf("tcp://localhost:%d", port)
	case "dotnet":
		return "" // .NET attaches by process ID
	default:
		return fmt.Sprintf("tcp://localhost:%d", port)
	}
}
