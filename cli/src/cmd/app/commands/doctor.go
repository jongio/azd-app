package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/cliout"
	"github.com/spf13/cobra"
)

const (
	doctorPass = "pass"
	doctorWarn = "warn"
	doctorFail = "fail"
)

type doctorCheck struct {
	CheckID  string `json:"checkId"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Hint     string `json:"hint,omitempty"`
	Service  string `json:"service,omitempty"`
	// Tool names the executable a "tool.available" check probed, so JSON
	// consumers do not have to parse Message to learn which tool failed.
	Tool string `json:"tool,omitempty"`
}

// doctorToolRequirement describes a required executable. Candidates holds the
// acceptable executable names — the requirement is satisfied when any one of
// them resolves on PATH. An empty Candidates means the requirement name is the
// only accepted executable.
type doctorToolRequirement struct {
	Candidates []string
	Hint       string
}

// NewDoctorCommand creates the doctor command.
func NewDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "doctor",
		Short:        "Check local setup before running services",
		Long:         `Run read-only setup checks for the project root, azure.yaml, service paths, required tools, port declarations, and dashboard state.`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor()
		},
	}
}

func runDoctor() error {
	checks := runDoctorChecks()
	if cliout.IsJSON() {
		if err := cliout.PrintJSON(checks); err != nil {
			return err
		}
	} else {
		renderDoctorChecks(checks)
	}
	if fails := countDoctorSeverity(checks, doctorFail); fails > 0 {
		return fmt.Errorf("doctor found %d failing check(s)", fails)
	}
	return nil
}

func runDoctorChecks() []doctorCheck {
	azureYamlPath, err := findAzureYaml()
	if err != nil {
		return []doctorCheck{{CheckID: "project.azure_yaml", Severity: doctorFail, Message: "azure.yaml was not found", Hint: "Run this command from an azd project or create azure.yaml with azd app init."}}
	}
	projectDir := filepath.Dir(azureYamlPath)
	checks := []doctorCheck{
		{CheckID: "project.root", Severity: doctorPass, Message: fmt.Sprintf("project root: %s", projectDir)},
		{CheckID: "config.exists", Severity: doctorPass, Message: fmt.Sprintf("found %s", azureYamlPath)},
	}

	cfg, parseErr := service.ParseAzureYaml(projectDir)
	if parseErr != nil {
		checks = append(checks, doctorCheck{CheckID: "config.parse", Severity: doctorFail, Message: parseErr.Error(), Hint: "Fix azure.yaml before running services."})
		return checks
	}
	checks = append(checks, doctorCheck{CheckID: "config.parse", Severity: doctorPass, Message: "azure.yaml parsed successfully"})
	checks = append(checks, doctorServicePathChecks(projectDir, cfg)...)
	checks = append(checks, doctorToolChecks(projectDir, cfg)...)
	checks = append(checks, doctorPortChecks(cfg)...)
	checks = append(checks, doctorDashboardCheck(projectDir))
	sortDoctorChecks(checks)
	return checks
}

func doctorServicePathChecks(projectDir string, cfg *service.AzureYaml) []doctorCheck {
	if cfg == nil || len(cfg.Services) == 0 {
		return []doctorCheck{{CheckID: "services.defined", Severity: doctorWarn, Message: "no services are defined", Hint: "Add services before running azd app run."}}
	}
	checks := []doctorCheck{{CheckID: "services.defined", Severity: doctorPass, Message: fmt.Sprintf("%d service(s) defined", len(cfg.Services))}}
	names := sortedDoctorServiceNames(cfg)
	for _, name := range names {
		svc := cfg.Services[name]
		declared := strings.TrimSpace(svc.Project)
		if declared == "" {
			checks = append(checks, doctorCheck{CheckID: "service.project", Severity: doctorWarn, Service: name, Message: "service has no project path", Hint: "Set project when the service lives outside the project root."})
			continue
		}
		projectPath := declared
		if !filepath.IsAbs(projectPath) {
			projectPath = filepath.Join(projectDir, projectPath)
		}
		if info, err := os.Stat(projectPath); err != nil {
			checks = append(checks, doctorCheck{CheckID: "service.project", Severity: doctorFail, Service: name, Message: fmt.Sprintf("project path %q does not exist", declared), Hint: "Create the folder or update azure.yaml."})
		} else if !info.IsDir() {
			checks = append(checks, doctorCheck{CheckID: "service.project", Severity: doctorFail, Service: name, Message: fmt.Sprintf("project path %q is not a directory", declared), Hint: "Point project to a directory."})
		} else {
			checks = append(checks, doctorCheck{CheckID: "service.project", Severity: doctorPass, Service: name, Message: fmt.Sprintf("project path exists: %s", declared)})
		}
	}
	return checks
}

func doctorToolChecks(projectDir string, cfg *service.AzureYaml) []doctorCheck {
	required := map[string]doctorToolRequirement{
		"azd": {Hint: "Install Azure Developer CLI."},
		"git": {Hint: "Install Git."},
	}
	for _, svc := range cfg.Services {
		lang := strings.ToLower(svc.Language)
		svcDir := strings.TrimSpace(svc.Project)
		if svcDir == "" {
			svcDir = projectDir
		}
		if !filepath.IsAbs(svcDir) {
			svcDir = filepath.Join(projectDir, svcDir)
		}
		switch lang {
		case "node", "javascript", "typescript":
			required[doctorDetectNodePackageManager(svcDir)] = doctorToolRequirement{Hint: "Install the package manager used by this service."}
		case "python":
			if name, req, ok := doctorPythonRequirement(svcDir); ok {
				required[name] = req
			}
		case "dotnet", ".net", "csharp":
			required["dotnet"] = doctorToolRequirement{Hint: "Install the .NET SDK."}
		case "java":
			required[doctorDetectJavaTool(svcDir)] = doctorToolRequirement{Hint: "Install the Java build tool used by this service."}
		case "go":
			required["go"] = doctorToolRequirement{Hint: "Install Go."}
		case "rust":
			required["cargo"] = doctorToolRequirement{Hint: "Install Rust and Cargo."}
		}
		if doctorNeedsDocker(svc) {
			required["docker"] = doctorToolRequirement{Hint: "Install Docker or a compatible container runtime."}
		}
	}
	tools := make([]string, 0, len(required))
	for tool := range required {
		if tool != "" {
			tools = append(tools, tool)
		}
	}
	sort.Strings(tools)
	checks := make([]doctorCheck, 0, len(tools))
	for _, tool := range tools {
		req := required[tool]
		candidates := req.Candidates
		if len(candidates) == 0 {
			candidates = []string{tool}
		}
		found := ""
		for _, candidate := range candidates {
			if _, err := exec.LookPath(candidate); err == nil {
				found = candidate
				break
			}
		}
		if found == "" {
			checks = append(checks, doctorCheck{CheckID: "tool.available", Severity: doctorFail, Tool: tool, Message: fmt.Sprintf("%s was not found on PATH", strings.Join(candidates, " or ")), Hint: req.Hint})
			continue
		}
		checks = append(checks, doctorCheck{CheckID: "tool.available", Severity: doctorPass, Tool: tool, Message: fmt.Sprintf("%s is available", found)})
	}
	return checks
}

// doctorNeedsDocker reports whether `azd app run` actually starts the service as
// a container, mirroring service.DetectServiceRuntime. Declaring host ports is a
// first-class feature for non-container services (see service.ParsePortSpec and
// service.DetectPort priority 1), so `ports:` alone must not require Docker.
func doctorNeedsDocker(svc service.Service) bool {
	if svc.RunsAsLocalProcess() {
		return false
	}
	return svc.IsContainerService() || strings.EqualFold(svc.Host, "container")
}

func doctorPortChecks(cfg *service.AzureYaml) []doctorCheck {
	seen := map[string]string{}
	var checks []doctorCheck
	for _, name := range sortedDoctorServiceNames(cfg) {
		svc := cfg.Services[name]
		isDocker := doctorNeedsDocker(svc)
		for _, mapping := range svc.Ports {
			parsed, err := service.ParsePortSpec(mapping, isDocker)
			if err != nil {
				checks = append(checks, doctorCheck{CheckID: "port.valid", Severity: doctorFail, Service: name, Message: fmt.Sprintf("invalid port %q", mapping), Hint: "Use a host port between 1 and 65535."})
				continue
			}
			// A container service that declares only a container port gets its
			// host port auto-assigned at run time, so it can never collide.
			if isDocker && parsed.HostPort == 0 {
				checks = append(checks, doctorCheck{CheckID: "port.valid", Severity: doctorPass, Service: name, Message: fmt.Sprintf("container port %d is declared (host port auto-assigned)", parsed.ContainerPort)})
				continue
			}
			if parsed.HostPort < 1 || parsed.HostPort > 65535 {
				checks = append(checks, doctorCheck{CheckID: "port.valid", Severity: doctorFail, Service: name, Message: fmt.Sprintf("invalid host port %q", mapping), Hint: "Use a host port between 1 and 65535."})
				continue
			}
			// Only the same host port on the same protocol actually conflicts,
			// so 3000/tcp and 3000/udp coexist.
			key := fmt.Sprintf("%d/%s", parsed.HostPort, parsed.Protocol)
			if other, ok := seen[key]; ok {
				checks = append(checks, doctorCheck{CheckID: "port.unique", Severity: doctorFail, Service: name, Message: fmt.Sprintf("host port %d/%s is also declared by service %q", parsed.HostPort, parsed.Protocol, other), Hint: "Use a unique host port."})
				continue
			}
			seen[key] = name
			checks = append(checks, doctorCheck{CheckID: "port.valid", Severity: doctorPass, Service: name, Message: fmt.Sprintf("host port %d is declared", parsed.HostPort)})
		}
	}
	if len(checks) == 0 {
		return []doctorCheck{{CheckID: "port.declared", Severity: doctorWarn, Message: "no host ports are declared", Hint: "Declare ports for browser-openable HTTP services."}}
	}
	return checks
}

func doctorDashboardCheck(projectDir string) doctorCheck {
	port, err := discoverDashboardPort(projectDir)
	if err != nil {
		return doctorCheck{CheckID: "dashboard.state", Severity: doctorWarn, Message: "dashboard is not running", Hint: "Run azd app run to start the dashboard."}
	}
	return doctorCheck{CheckID: "dashboard.state", Severity: doctorPass, Message: fmt.Sprintf("dashboard is running on port %d", port)}
}

func doctorDetectNodePackageManager(dir string) string {
	switch {
	case doctorFileExists(filepath.Join(dir, "pnpm-lock.yaml")):
		return "pnpm"
	case doctorFileExists(filepath.Join(dir, "yarn.lock")):
		return "yarn"
	default:
		return "npm"
	}
}

// doctorPythonRequirement returns the tool requirement for a Python service,
// mirroring how the service is actually run. uv/poetry/pipenv projects need that
// tool on PATH; a virtual environment ships its own interpreter so nothing is
// required; otherwise either python or python3 satisfies the requirement, since
// many macOS/Linux distributions ship only python3.
func doctorPythonRequirement(dir string) (string, doctorToolRequirement, bool) {
	switch detector.DetectPythonPackageManager(dir) {
	case "uv":
		return "uv", doctorToolRequirement{Hint: "Install uv."}, true
	case "poetry":
		return "poetry", doctorToolRequirement{Hint: "Install Poetry."}, true
	case "pipenv":
		return "pipenv", doctorToolRequirement{Hint: "Install Pipenv."}, true
	}
	if doctorVenvPython(dir) != "" {
		return "", doctorToolRequirement{}, false
	}
	return "python", doctorToolRequirement{
		Candidates: []string{"python", "python3"},
		Hint:       "Install Python.",
	}, true
}

// doctorVenvPython returns the interpreter inside a service virtual environment,
// or an empty string when the service has no venv. It mirrors the venv lookup
// used by the runner.
func doctorVenvPython(dir string) string {
	for _, venvDir := range []string{".venv", "venv"} {
		candidates := []string{
			filepath.Join(dir, venvDir, "Scripts", "python.exe"),
			filepath.Join(dir, venvDir, "bin", "python"),
		}
		for _, candidate := range candidates {
			if doctorFileExists(candidate) {
				return candidate
			}
		}
	}
	return ""
}

func doctorDetectJavaTool(dir string) string {
	if doctorFileExists(filepath.Join(dir, "gradlew")) || doctorFileExists(filepath.Join(dir, "build.gradle")) || doctorFileExists(filepath.Join(dir, "build.gradle.kts")) {
		return "gradle"
	}
	return "mvn"
}

func doctorFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func sortedDoctorServiceNames(cfg *service.AzureYaml) []string {
	names := make([]string, 0, len(cfg.Services))
	for name := range cfg.Services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func renderDoctorChecks(checks []doctorCheck) {
	cliout.CommandHeader("doctor", "Check local setup")
	for _, check := range checks {
		target := check.CheckID
		if check.Service != "" {
			target = check.Service + " " + target
		}
		cliout.Item("[%s] %s: %s", strings.ToUpper(check.Severity), target, check.Message)
		if check.Hint != "" {
			cliout.Item("      fix: %s", check.Hint)
		}
	}
}

func countDoctorSeverity(checks []doctorCheck, severity string) int {
	count := 0
	for _, check := range checks {
		if check.Severity == severity {
			count++
		}
	}
	return count
}

func sortDoctorChecks(checks []doctorCheck) {
	sort.SliceStable(checks, func(i, j int) bool {
		if checks[i].Severity != checks[j].Severity {
			return checks[i].Severity < checks[j].Severity
		}
		if checks[i].Service != checks[j].Service {
			return checks[i].Service < checks[j].Service
		}
		if checks[i].CheckID != checks[j].CheckID {
			return checks[i].CheckID < checks[j].CheckID
		}
		return checks[i].Tool < checks[j].Tool
	})
}
