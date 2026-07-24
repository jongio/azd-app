package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

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
		if strings.TrimSpace(svc.Project) == "" {
			checks = append(checks, doctorCheck{CheckID: "service.project", Severity: doctorWarn, Service: name, Message: "service has no project path", Hint: "Set project when the service lives outside the project root."})
			continue
		}
		projectPath := svc.Project
		if !filepath.IsAbs(projectPath) {
			projectPath = filepath.Join(projectDir, projectPath)
		}
		if info, err := os.Stat(projectPath); err != nil {
			checks = append(checks, doctorCheck{CheckID: "service.project", Severity: doctorFail, Service: name, Message: fmt.Sprintf("project path %q does not exist", svc.Project), Hint: "Create the folder or update azure.yaml."})
		} else if !info.IsDir() {
			checks = append(checks, doctorCheck{CheckID: "service.project", Severity: doctorFail, Service: name, Message: fmt.Sprintf("project path %q is not a directory", svc.Project), Hint: "Point project to a directory."})
		} else {
			checks = append(checks, doctorCheck{CheckID: "service.project", Severity: doctorPass, Service: name, Message: fmt.Sprintf("project path exists: %s", svc.Project)})
		}
	}
	return checks
}

func doctorToolChecks(projectDir string, cfg *service.AzureYaml) []doctorCheck {
	required := map[string]string{"azd": "Install Azure Developer CLI.", "git": "Install Git."}
	for _, svc := range cfg.Services {
		lang := strings.ToLower(svc.Language)
		svcDir := svc.Project
		if svcDir == "" {
			svcDir = projectDir
		}
		if !filepath.IsAbs(svcDir) {
			svcDir = filepath.Join(projectDir, svcDir)
		}
		switch lang {
		case "node", "javascript", "typescript":
			required[doctorDetectNodePackageManager(svcDir)] = "Install the package manager used by this service."
		case "python":
			required[doctorDetectPythonTool(svcDir)] = "Install the Python tool used by this service."
		case "dotnet", ".net", "csharp":
			required["dotnet"] = "Install the .NET SDK."
		case "java":
			required[doctorDetectJavaTool(svcDir)] = "Install the Java build tool used by this service."
		case "go":
			required["go"] = "Install Go."
		case "rust":
			required["cargo"] = "Install Rust and Cargo."
		}
		if svc.Image != "" || len(svc.Ports) > 0 || strings.EqualFold(svc.Host, "container") {
			required["docker"] = "Install Docker or a compatible container runtime."
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
		if _, err := exec.LookPath(tool); err != nil {
			checks = append(checks, doctorCheck{CheckID: "tool.available", Severity: doctorFail, Message: fmt.Sprintf("%s was not found on PATH", tool), Hint: required[tool]})
		} else {
			checks = append(checks, doctorCheck{CheckID: "tool.available", Severity: doctorPass, Message: fmt.Sprintf("%s is available", tool)})
		}
	}
	return checks
}

func doctorPortChecks(cfg *service.AzureYaml) []doctorCheck {
	seen := map[int]string{}
	var checks []doctorCheck
	for _, name := range sortedDoctorServiceNames(cfg) {
		for _, mapping := range cfg.Services[name].Ports {
			port, err := doctorHostPort(mapping)
			if err != nil {
				checks = append(checks, doctorCheck{CheckID: "port.valid", Severity: doctorFail, Service: name, Message: err.Error(), Hint: "Use a host port between 1 and 65535."})
				continue
			}
			if other, ok := seen[port]; ok {
				checks = append(checks, doctorCheck{CheckID: "port.unique", Severity: doctorFail, Service: name, Message: fmt.Sprintf("host port %d is also declared by service %q", port, other), Hint: "Use a unique host port."})
			} else {
				seen[port] = name
				checks = append(checks, doctorCheck{CheckID: "port.valid", Severity: doctorPass, Service: name, Message: fmt.Sprintf("host port %d is declared", port)})
			}
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

func doctorDetectPythonTool(dir string) string {
	switch {
	case doctorFileExists(filepath.Join(dir, "uv.lock")):
		return "uv"
	case doctorFileExists(filepath.Join(dir, "poetry.lock")) || doctorFileExists(filepath.Join(dir, "pyproject.toml")):
		return "poetry"
	default:
		return "python"
	}
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

func doctorHostPort(mapping string) (int, error) {
	parts := strings.Split(strings.TrimSpace(mapping), ":")
	candidate := parts[0]
	if len(parts) > 1 {
		candidate = parts[len(parts)-2]
	}
	candidate = strings.TrimSuffix(strings.TrimSpace(candidate), "/tcp")
	candidate = strings.TrimSuffix(candidate, "/udp")
	port, err := strconv.Atoi(candidate)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid host port %q", mapping)
	}
	return port, nil
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
		return checks[i].CheckID < checks[j].CheckID
	})
}
