package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

func TestDoctorNeedsDocker(t *testing.T) {
	tests := []struct {
		name string
		svc  service.Service
		want bool
	}{
		{"declared ports alone never require docker", service.Service{Language: "go", Ports: []string{"3000"}}, false},
		{"declared mapping alone never requires docker", service.Service{Language: "node", Ports: []string{"3000:8080"}}, false},
		{"top level image requires docker", service.Service{Image: "nginx:latest"}, true},
		{"docker image requires docker", service.Service{Docker: &service.DockerConfig{Image: "nginx:latest"}}, true},
		{"container host requires docker", service.Service{Host: "Container"}, true},
		{"container image with local command runs as a process", service.Service{Docker: &service.DockerConfig{Image: "nginx:latest"}, Command: "npm run dev"}, false},
		{"process type runs as a process", service.Service{Docker: &service.DockerConfig{Image: "nginx:latest"}, Type: "process"}, false},
		{"plain service needs nothing", service.Service{Language: "go"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := doctorNeedsDocker(tt.svc); got != tt.want {
				t.Fatalf("doctorNeedsDocker() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDoctorToolChecksDockerRequirement(t *testing.T) {
	tests := []struct {
		name       string
		svc        service.Service
		wantDocker bool
	}{
		{"ports do not require docker", service.Service{Language: "go", Ports: []string{"3000"}}, false},
		{"image requires docker", service.Service{Image: "nginx:latest", Ports: []string{"3000:80"}}, true},
		{"container host requires docker", service.Service{Host: "container"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			checks := doctorToolChecks(dir, &service.AzureYaml{Services: map[string]service.Service{"api": tt.svc}})
			if got := hasDoctorTool(checks, "docker"); got != tt.wantDocker {
				t.Fatalf("docker requirement = %v, want %v (checks: %#v)", got, tt.wantDocker, checks)
			}
		})
	}
}

func TestDoctorToolChecksTrimsProjectPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte("lock"), 0o600); err != nil {
		t.Fatalf("write lock: %v", err)
	}
	checks := doctorToolChecks(dir, &service.AzureYaml{Services: map[string]service.Service{
		"web": {Language: "node", Project: "   "},
	}})
	if !hasDoctorTool(checks, "pnpm") {
		t.Fatalf("expected pnpm requirement from trimmed project path, got %#v", checks)
	}
}

func TestDoctorToolChecksAlwaysRecordsToolName(t *testing.T) {
	checks := doctorToolChecks(t.TempDir(), &service.AzureYaml{Services: map[string]service.Service{}})
	for _, check := range checks {
		if check.CheckID != "tool.available" {
			continue
		}
		if check.Tool == "" {
			t.Fatalf("tool.available check has no Tool field: %#v", check)
		}
	}
	if !hasDoctorTool(checks, "azd") || !hasDoctorTool(checks, "git") {
		t.Fatalf("expected baseline azd and git requirements, got %#v", checks)
	}
}

func TestDoctorToolCheckJSONIncludesTool(t *testing.T) {
	data, err := json.Marshal(doctorCheck{CheckID: "tool.available", Severity: doctorFail, Message: "docker was not found on PATH", Tool: "docker"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(data), `"tool":"docker"`) {
		t.Fatalf("expected tool field in JSON, got %s", data)
	}
	data, err = json.Marshal(doctorCheck{CheckID: "port.valid", Severity: doctorPass, Message: "ok"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), `"tool"`) {
		t.Fatalf("expected tool field to be omitted, got %s", data)
	}
}

func TestDoctorPythonRequirement(t *testing.T) {
	tests := []struct {
		name           string
		files          map[string]string
		wantTool       string
		wantRequired   bool
		wantCandidates []string
	}{
		{
			name:         "uv lock selects uv",
			files:        map[string]string{"uv.lock": "lock"},
			wantTool:     "uv",
			wantRequired: true,
		},
		{
			name:         "poetry lock selects poetry",
			files:        map[string]string{"poetry.lock": "lock"},
			wantTool:     "poetry",
			wantRequired: true,
		},
		{
			name:         "pyproject without poetry does not require poetry",
			files:        map[string]string{"pyproject.toml": "[project]\nname = \"demo\"\n"},
			wantTool:     "python",
			wantRequired: true,
			// python3 must be accepted: many macOS/Linux systems ship only python3.
			wantCandidates: []string{"python", "python3"},
		},
		{
			name:           "plain project accepts python or python3",
			files:          map[string]string{"requirements.txt": "flask\n"},
			wantTool:       "python",
			wantRequired:   true,
			wantCandidates: []string{"python", "python3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
					t.Fatalf("write %s: %v", name, err)
				}
			}
			tool, req, ok := doctorPythonRequirement(dir)
			if ok != tt.wantRequired {
				t.Fatalf("required = %v, want %v", ok, tt.wantRequired)
			}
			if tool != tt.wantTool {
				t.Fatalf("tool = %q, want %q", tool, tt.wantTool)
			}
			if len(tt.wantCandidates) > 0 {
				if strings.Join(req.Candidates, ",") != strings.Join(tt.wantCandidates, ",") {
					t.Fatalf("candidates = %v, want %v", req.Candidates, tt.wantCandidates)
				}
			}
		})
	}
}

func TestDoctorPythonRequirementSkippedForVenv(t *testing.T) {
	dir := t.TempDir()
	writeDoctorVenv(t, dir, ".venv")
	if _, _, ok := doctorPythonRequirement(dir); ok {
		t.Fatalf("expected no PATH requirement when a venv supplies the interpreter")
	}
}

func TestDoctorVenvPython(t *testing.T) {
	for _, venvDir := range []string{".venv", "venv"} {
		t.Run(venvDir, func(t *testing.T) {
			dir := t.TempDir()
			if got := doctorVenvPython(dir); got != "" {
				t.Fatalf("expected no venv, got %q", got)
			}
			writeDoctorVenv(t, dir, venvDir)
			if got := doctorVenvPython(dir); got == "" {
				t.Fatalf("expected venv interpreter in %s", venvDir)
			}
		})
	}
}

func TestDoctorDetectJavaTool(t *testing.T) {
	dir := t.TempDir()
	if got := doctorDetectJavaTool(dir); got != "mvn" {
		t.Fatalf("java tool = %q, want mvn", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "build.gradle"), []byte("plugins {}"), 0o600); err != nil {
		t.Fatalf("write build.gradle: %v", err)
	}
	if got := doctorDetectJavaTool(dir); got != "gradle" {
		t.Fatalf("java tool = %q, want gradle", got)
	}
}

func TestDoctorDetectNodePackageManagerFallbacks(t *testing.T) {
	dir := t.TempDir()
	if got := doctorDetectNodePackageManager(dir); got != "npm" {
		t.Fatalf("package manager = %q, want npm", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte("lock"), 0o600); err != nil {
		t.Fatalf("write yarn.lock: %v", err)
	}
	if got := doctorDetectNodePackageManager(dir); got != "yarn" {
		t.Fatalf("package manager = %q, want yarn", got)
	}
}

func TestSortDoctorChecksOrdersFailuresFirst(t *testing.T) {
	checks := []doctorCheck{
		{CheckID: "tool.available", Severity: doctorPass, Tool: "git"},
		{CheckID: "port.declared", Severity: doctorWarn},
		{CheckID: "tool.available", Severity: doctorFail, Tool: "docker"},
		{CheckID: "tool.available", Severity: doctorFail, Tool: "azd"},
	}
	sortDoctorChecks(checks)
	if checks[0].Severity != doctorFail || checks[0].Tool != "azd" {
		t.Fatalf("expected azd failure first, got %#v", checks[0])
	}
	if checks[1].Tool != "docker" {
		t.Fatalf("expected docker failure second, got %#v", checks[1])
	}
	if checks[len(checks)-1].Severity != doctorWarn {
		t.Fatalf("expected warn last, got %#v", checks[len(checks)-1])
	}
}

func TestCountDoctorSeverity(t *testing.T) {
	checks := []doctorCheck{
		{Severity: doctorFail},
		{Severity: doctorFail},
		{Severity: doctorPass},
		{Severity: doctorWarn},
	}
	if got := countDoctorSeverity(checks, doctorFail); got != 2 {
		t.Fatalf("fail count = %d, want 2", got)
	}
	if got := countDoctorSeverity(checks, doctorWarn); got != 1 {
		t.Fatalf("warn count = %d, want 1", got)
	}
	if got := countDoctorSeverity(nil, doctorFail); got != 0 {
		t.Fatalf("nil count = %d, want 0", got)
	}
}

func TestRenderDoctorChecks(t *testing.T) {
	renderDoctorChecks([]doctorCheck{
		{CheckID: "tool.available", Severity: doctorFail, Message: "docker was not found on PATH", Hint: "Install Docker.", Tool: "docker"},
		{CheckID: "service.project", Severity: doctorPass, Message: "project path exists", Service: "api"},
	})
}

func TestRunDoctorChecksMissingAzureYaml(t *testing.T) {
	t.Chdir(t.TempDir())
	checks := runDoctorChecks()
	assertDoctorCheck(t, checks, "", "project.azure_yaml", doctorFail)
}

func TestRunDoctorChecksReportsParseFailure(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte("services: [oops\n"), 0o600); err != nil {
		t.Fatalf("write azure.yaml: %v", err)
	}
	t.Chdir(dir)
	checks := runDoctorChecks()
	assertDoctorCheck(t, checks, "", "config.parse", doctorFail)
}

func TestRunDoctorChecksEndToEnd(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "api"), 0o750); err != nil {
		t.Fatalf("mkdir api: %v", err)
	}
	yaml := "name: demo\n" +
		"services:\n" +
		"  api:\n" +
		"    project: ./api\n" +
		"    language: go\n" +
		"    host: appservice\n" +
		"    ports:\n" +
		"      - \"3000\"\n" +
		"  web:\n" +
		"    project: ./missing\n" +
		"    language: go\n" +
		"    host: appservice\n"
	if err := os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatalf("write azure.yaml: %v", err)
	}
	t.Chdir(dir)

	checks := runDoctorChecks()
	assertDoctorCheck(t, checks, "", "project.root", doctorPass)
	assertDoctorCheck(t, checks, "", "config.parse", doctorPass)
	assertDoctorCheck(t, checks, "", "services.defined", doctorPass)
	assertDoctorCheck(t, checks, "api", "service.project", doctorPass)
	assertDoctorCheck(t, checks, "web", "service.project", doctorFail)
	assertDoctorCheck(t, checks, "api", "port.valid", doctorPass)
	// Neither service is a container, so docker must not be required.
	if hasDoctorTool(checks, "docker") {
		t.Fatalf("docker must not be required for non-container services: %#v", checks)
	}
	if !hasDoctorTool(checks, "go") {
		t.Fatalf("expected go tool requirement: %#v", checks)
	}

	// Failing checks must sort ahead of passing ones.
	if checks[0].Severity != doctorFail {
		t.Fatalf("expected a failing check first, got %#v", checks[0])
	}

	data, err := json.Marshal(checks)
	if err != nil {
		t.Fatalf("marshal checks: %v", err)
	}
	if !strings.Contains(string(data), `"checkId":"service.project"`) {
		t.Fatalf("expected service.project in JSON output, got %s", data)
	}

	if err := runDoctor(); err == nil {
		t.Fatal("expected runDoctor to return an error when checks fail")
	}
}

func TestNewDoctorCommand(t *testing.T) {
	cmd := NewDoctorCommand()
	if cmd.Use != "doctor" {
		t.Fatalf("Use = %q, want doctor", cmd.Use)
	}
	if cmd.RunE == nil {
		t.Fatal("expected RunE to be wired")
	}
	t.Chdir(t.TempDir())
	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected an error when azure.yaml is missing")
	}
}

func hasDoctorTool(checks []doctorCheck, tool string) bool {
	for _, check := range checks {
		if check.CheckID == "tool.available" && check.Tool == tool {
			return true
		}
	}
	return false
}

func writeDoctorVenv(t *testing.T, dir, venvDir string) {
	t.Helper()
	binDir := "bin"
	exe := "python"
	if runtime.GOOS == "windows" {
		binDir = "Scripts"
		exe = "python.exe"
	}
	full := filepath.Join(dir, venvDir, binDir)
	if err := os.MkdirAll(full, 0o750); err != nil {
		t.Fatalf("mkdir venv: %v", err)
	}
	if err := os.WriteFile(filepath.Join(full, exe), []byte("stub"), 0o600); err != nil {
		t.Fatalf("write venv python: %v", err)
	}
}
