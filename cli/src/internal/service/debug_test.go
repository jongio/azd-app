package service

import (
	"os/exec"
	"strings"
	"testing"
)

func TestConfigureDebug(t *testing.T) {
	tests := []struct {
		name            string
		runtime         *ServiceRuntime
		debugEnabled    bool
		waitForDebugger bool
		languageIndex   int
		expectedEnabled bool
		expectedPort    int
		expectedProto   string
	}{
		{
			name: "Debug disabled",
			runtime: &ServiceRuntime{
				Name:     "api",
				Language: "JavaScript",
			},
			debugEnabled:    false,
			waitForDebugger: false,
			languageIndex:   0,
			expectedEnabled: false,
		},
		{
			name: "Node.js debug enabled",
			runtime: &ServiceRuntime{
				Name:     "api",
				Language: "JavaScript",
			},
			debugEnabled:    true,
			waitForDebugger: false,
			languageIndex:   0,
			expectedEnabled: true,
			expectedPort:    9229,
			expectedProto:   "inspector",
		},
		{
			name: "Python debug enabled",
			runtime: &ServiceRuntime{
				Name:     "worker",
				Language: "Python",
			},
			debugEnabled:    true,
			waitForDebugger: false,
			languageIndex:   0,
			expectedEnabled: true,
			expectedPort:    5678,
			expectedProto:   "debugpy",
		},
		{
			name: "Go debug enabled",
			runtime: &ServiceRuntime{
				Name:     "api",
				Language: "Go",
			},
			debugEnabled:    true,
			waitForDebugger: false,
			languageIndex:   0,
			expectedEnabled: true,
			expectedPort:    2345,
			expectedProto:   "delve",
		},
		{
			name: ".NET debug enabled",
			runtime: &ServiceRuntime{
				Name:     "web",
				Language: ".NET",
			},
			debugEnabled:    true,
			waitForDebugger: false,
			languageIndex:   0,
			expectedEnabled: true,
			expectedPort:    5005,
			expectedProto:   "coreclr",
		},
		{
			name: "Multiple Node.js services - port offset",
			runtime: &ServiceRuntime{
				Name:     "api2",
				Language: "TypeScript",
			},
			debugEnabled:    true,
			waitForDebugger: false,
			languageIndex:   2,
			expectedEnabled: true,
			expectedPort:    9231, // 9229 + 2
			expectedProto:   "inspector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ConfigureDebug(tt.runtime, tt.debugEnabled, tt.waitForDebugger, tt.languageIndex)

			if tt.runtime.Debug.Enabled != tt.expectedEnabled {
				t.Errorf("Expected Enabled=%v, got %v", tt.expectedEnabled, tt.runtime.Debug.Enabled)
			}

			if tt.expectedEnabled {
				if tt.runtime.Debug.Port != tt.expectedPort {
					t.Errorf("Expected Port=%d, got %d", tt.expectedPort, tt.runtime.Debug.Port)
				}

				if tt.runtime.Debug.Protocol != tt.expectedProto {
					t.Errorf("Expected Protocol=%s, got %s", tt.expectedProto, tt.runtime.Debug.Protocol)
				}

				if tt.runtime.Debug.WaitForDebugger != tt.waitForDebugger {
					t.Errorf("Expected WaitForDebugger=%v, got %v", tt.waitForDebugger, tt.runtime.Debug.WaitForDebugger)
				}
			}
		})
	}
}

func TestApplyDebugFlags_Node(t *testing.T) {
	runtime := &ServiceRuntime{
		Name:     "api",
		Language: "JavaScript",
		Debug: DebugConfig{
			Enabled:         true,
			Port:            9229,
			Protocol:        "inspector",
			WaitForDebugger: false,
		},
	}

	cmd := exec.Command("node", "index.js")
	err := ApplyDebugFlags(runtime, cmd)

	if err != nil {
		t.Fatalf("ApplyDebugFlags failed: %v", err)
	}

	// Check that --inspect flag was added
	found := false
	for _, arg := range cmd.Args {
		if strings.Contains(arg, "--inspect=") {
			found = true
			if arg != "--inspect=0.0.0.0:9229" {
				t.Errorf("Expected --inspect=0.0.0.0:9229, got %s", arg)
			}
		}
	}

	if !found {
		t.Error("--inspect flag not found in command args")
	}
}

func TestApplyDebugFlags_NodeWaitForDebugger(t *testing.T) {
	runtime := &ServiceRuntime{
		Name:     "api",
		Language: "JavaScript",
		Debug: DebugConfig{
			Enabled:         true,
			Port:            9229,
			Protocol:        "inspector",
			WaitForDebugger: true,
		},
	}

	cmd := exec.Command("node", "index.js")
	err := ApplyDebugFlags(runtime, cmd)

	if err != nil {
		t.Fatalf("ApplyDebugFlags failed: %v", err)
	}

	// Check that --inspect-brk flag was added
	found := false
	for _, arg := range cmd.Args {
		if strings.Contains(arg, "--inspect-brk=") {
			found = true
			if arg != "--inspect-brk=0.0.0.0:9229" {
				t.Errorf("Expected --inspect-brk=0.0.0.0:9229, got %s", arg)
			}
		}
	}

	if !found {
		t.Error("--inspect-brk flag not found in command args")
	}
}

func TestApplyDebugFlags_Python(t *testing.T) {
	runtime := &ServiceRuntime{
		Name:     "worker",
		Language: "Python",
		Debug: DebugConfig{
			Enabled:         true,
			Port:            5678,
			Protocol:        "debugpy",
			WaitForDebugger: false,
		},
	}

	cmd := exec.Command("python", "app.py")
	err := ApplyDebugFlags(runtime, cmd)

	if err != nil {
		t.Fatalf("ApplyDebugFlags failed: %v", err)
	}

	// Check that debugpy module was added
	if len(cmd.Args) < 5 {
		t.Fatalf("Expected at least 5 args, got %d", len(cmd.Args))
	}

	if cmd.Args[1] != "-m" {
		t.Errorf("Expected arg[1] to be '-m', got %s", cmd.Args[1])
	}

	if cmd.Args[2] != "debugpy" {
		t.Errorf("Expected arg[2] to be 'debugpy', got %s", cmd.Args[2])
	}

	if cmd.Args[3] != "--listen" {
		t.Errorf("Expected arg[3] to be '--listen', got %s", cmd.Args[3])
	}

	if cmd.Args[4] != "0.0.0.0:5678" {
		t.Errorf("Expected arg[4] to be '0.0.0.0:5678', got %s", cmd.Args[4])
	}
}

func TestApplyDebugFlags_Java(t *testing.T) {
	runtime := &ServiceRuntime{
		Name:     "api",
		Language: "Java",
		Debug: DebugConfig{
			Enabled:         true,
			Port:            5005,
			Protocol:        "jdwp",
			WaitForDebugger: false,
		},
	}

	cmd := exec.Command("java", "-jar", "app.jar")
	cmd.Env = []string{"PATH=/usr/bin"}

	err := ApplyDebugFlags(runtime, cmd)

	if err != nil {
		t.Fatalf("ApplyDebugFlags failed: %v", err)
	}

	// Check that JAVA_TOOL_OPTIONS was added
	found := false
	for _, env := range cmd.Env {
		if strings.HasPrefix(env, "JAVA_TOOL_OPTIONS=") {
			found = true
			if !strings.Contains(env, "suspend=n") {
				t.Error("Expected suspend=n in JAVA_TOOL_OPTIONS")
			}
			if !strings.Contains(env, "address=*:5005") {
				t.Error("Expected address=*:5005 in JAVA_TOOL_OPTIONS")
			}
		}
	}

	if !found {
		t.Error("JAVA_TOOL_OPTIONS not found in environment")
	}
}

func TestNormalizeLanguageForDebug(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"JavaScript", "node"},
		{"TypeScript", "node"},
		{"Node.js", "node"},
		{"Python", "python"},
		{".NET", "dotnet"},
		{"C#", "dotnet"},
		{"Go", "go"},
		{"Java", "java"},
		{"Rust", "rust"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeLanguageForDebug(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeLanguageForDebug(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetDebugPort(t *testing.T) {
	tests := []struct {
		language string
		offset   int
		expected int
	}{
		{"node", 0, 9229},
		{"node", 1, 9230},
		{"node", 2, 9231},
		{"python", 0, 5678},
		{"python", 1, 5679},
		{"go", 0, 2345},
		{"dotnet", 0, 5005},
		{"java", 0, 5005},
		{"unknown", 0, 9000}, // fallback
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			result := getDebugPort(tt.language, tt.offset)
			if result != tt.expected {
				t.Errorf("getDebugPort(%s, %d) = %d, want %d", tt.language, tt.offset, result, tt.expected)
			}
		})
	}
}

func TestGetDebugProtocol(t *testing.T) {
	tests := []struct {
		language string
		expected string
	}{
		{"node", "inspector"},
		{"python", "debugpy"},
		{"go", "delve"},
		{"dotnet", "coreclr"},
		{"java", "jdwp"},
		{"rust", "lldb"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			result := getDebugProtocol(tt.language)
			if result != tt.expected {
				t.Errorf("getDebugProtocol(%s) = %s, want %s", tt.language, result, tt.expected)
			}
		})
	}
}

func TestGetDebugURL(t *testing.T) {
	tests := []struct {
		language string
		port     int
		expected string
	}{
		{"node", 9229, "ws://localhost:9229"},
		{"python", 5678, "tcp://localhost:5678"},
		{"go", 2345, "tcp://localhost:2345"},
		{"java", 5005, "tcp://localhost:5005"},
		{"dotnet", 5005, ""}, // .NET uses process ID
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			result := getDebugURL(tt.language, tt.port)
			if result != tt.expected {
				t.Errorf("getDebugURL(%s, %d) = %s, want %s", tt.language, tt.port, result, tt.expected)
			}
		})
	}
}
