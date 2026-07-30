package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/cliout"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	validateSeverityError   = "error"
	validateSeverityWarning = "warning"
)

var serviceNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

type validateFinding struct {
	File     string `json:"file"`
	Service  string `json:"service,omitempty"`
	CheckID  string `json:"checkId"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Hint     string `json:"hint,omitempty"`
}

// NewValidateCommand creates the validate command.
func NewValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "validate",
		Short:        "Validate azure.yaml without starting services",
		Long:         `Validate azure.yaml with read-only checks for service references, project paths, ports, service types, modes, and command readiness.`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate()
		},
	}
}

func runValidate() error {
	azureYamlPath, err := findAzureYaml()
	if err != nil {
		return err
	}
	findings, err := validateAzureYamlFile(azureYamlPath)
	if err != nil {
		return err
	}
	if cliout.IsJSON() {
		if err := cliout.PrintJSON(findings); err != nil {
			return err
		}
	} else {
		renderValidateFindings(findings)
	}
	if hasValidateErrors(findings) {
		return fmt.Errorf("azure.yaml validation failed with %d error(s)", countValidateErrors(findings))
	}
	return nil
}

func validateAzureYamlFile(azureYamlPath string) ([]validateFinding, error) {
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read azure.yaml: %w", err)
	}
	var cfg service.AzureYaml
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		//nolint:nilerr // a parse failure is reported as a validation finding, not a command error
		return []validateFinding{{File: azureYamlPath, CheckID: "yaml.parse", Severity: validateSeverityError, Message: err.Error(), Hint: "Fix the YAML syntax."}}, nil
	}
	projectDir := filepath.Dir(azureYamlPath)
	findings := validateAzureYamlConfig(azureYamlPath, projectDir, &cfg)
	// ParseAzureYaml is a fail-fast backstop: it aborts on the first problem and reports it
	// as a single opaque error. Consult it only when the structured checks found no errors,
	// so one root cause (an out-of-root project path, say) yields exactly one finding
	// instead of both a precise check finding and a duplicate schema.parse finding.
	if !hasValidateErrors(findings) {
		if _, err := service.ParseAzureYaml(projectDir); err != nil {
			findings = append(findings, validateFinding{File: azureYamlPath, CheckID: "schema.parse", Severity: validateSeverityError, Message: err.Error(), Hint: "Fix the service configuration reported by the parser."})
		}
	}
	sortValidateFindings(findings)
	if findings == nil {
		// Emit [] rather than null so JSON consumers can always iterate the result.
		findings = []validateFinding{}
	}
	return findings, nil
}

func validateAzureYamlConfig(filePath, projectDir string, cfg *service.AzureYaml) []validateFinding {
	var findings []validateFinding
	if cfg == nil || len(cfg.Services) == 0 {
		return append(findings, validateFinding{File: filePath, CheckID: "services.empty", Severity: validateSeverityWarning, Message: "azure.yaml does not define any services", Hint: "Add services before running azd app run."})
	}
	known := make(map[string]struct{}, len(cfg.Services)+len(cfg.Resources))
	for name := range cfg.Services {
		known[name] = struct{}{}
	}
	for name := range cfg.Resources {
		known[name] = struct{}{}
	}

	seenPorts := map[int]string{}
	names := make([]string, 0, len(cfg.Services))
	for name := range cfg.Services {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		svc := cfg.Services[name]
		if !serviceNamePattern.MatchString(name) {
			findings = append(findings, validateFinding{File: filePath, Service: name, CheckID: "service.name", Severity: validateSeverityError, Message: fmt.Sprintf("service name %q contains unsupported characters", name), Hint: "Use letters, numbers, dot, underscore, or hyphen."})
		}
		findings = append(findings, validateProjectPath(filePath, projectDir, name, svc)...)
		for _, dep := range svc.Uses {
			if _, ok := known[dep]; !ok {
				findings = append(findings, validateFinding{File: filePath, Service: name, CheckID: "uses.unknown", Severity: validateSeverityError, Message: fmt.Sprintf("uses entry %q is not a defined service or resource", dep), Hint: "Add the dependency or remove it from uses."})
			}
		}
		for _, mapping := range svc.Ports {
			port, err := validateHostPort(mapping)
			if err != nil {
				findings = append(findings, validateFinding{File: filePath, Service: name, CheckID: "port.invalid", Severity: validateSeverityError, Message: err.Error(), Hint: "Use a host port between 1 and 65535, such as 8080:80."})
				continue
			}
			if other, ok := seenPorts[port]; ok {
				findings = append(findings, validateFinding{File: filePath, Service: name, CheckID: "port.duplicate", Severity: validateSeverityError, Message: fmt.Sprintf("host port %d is also used by service %q", port, other), Hint: "Assign a unique host port to one of the services."})
			} else {
				seenPorts[port] = name
			}
		}
		findings = append(findings, validateTypeMode(filePath, name, svc)...)
	}
	return findings
}

func validateProjectPath(filePath, projectDir, serviceName string, svc service.Service) []validateFinding {
	if strings.TrimSpace(svc.Project) == "" {
		return nil
	}
	projectPath := svc.Project
	if !filepath.IsAbs(projectPath) {
		projectPath = filepath.Clean(filepath.Join(projectDir, projectPath))
	}
	rel, err := filepath.Rel(projectDir, projectPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return []validateFinding{{File: filePath, Service: serviceName, CheckID: "project.outside-root", Severity: validateSeverityError, Message: fmt.Sprintf("project path %q is outside the project root", svc.Project), Hint: "Keep service project paths inside the repository."}}
	}
	info, err := os.Stat(projectPath)
	if err != nil {
		return []validateFinding{{File: filePath, Service: serviceName, CheckID: "project.missing", Severity: validateSeverityError, Message: fmt.Sprintf("project path %q does not exist", svc.Project), Hint: "Create the folder or update the project path."}}
	}
	if !info.IsDir() {
		return []validateFinding{{File: filePath, Service: serviceName, CheckID: "project.not-directory", Severity: validateSeverityError, Message: fmt.Sprintf("project path %q is not a directory", svc.Project), Hint: "Point project to a directory."}}
	}
	return nil
}

func validateHostPort(mapping string) (int, error) {
	mapping = strings.TrimSpace(mapping)
	if mapping == "" {
		return 0, fmt.Errorf("empty port mapping")
	}
	// The host port is the second-to-last segment for every supported form:
	// "8080:80", "127.0.0.1:8080:80", and "[::1]:8080:80" (the bracketed IPv6 host
	// splits into extra segments, but the host port still sits second from the end).
	parts := strings.Split(mapping, ":")
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

func validateTypeMode(filePath, serviceName string, svc service.Service) []validateFinding {
	var findings []validateFinding
	serviceType := strings.ToLower(strings.TrimSpace(svc.Type))
	if serviceType == "" {
		if len(svc.Ports) > 0 {
			serviceType = service.ServiceTypeHTTP
		} else {
			serviceType = service.ServiceTypeProcess
		}
	}
	switch serviceType {
	case service.ServiceTypeHTTP, service.ServiceTypeTCP, service.ServiceTypeProcess, service.ServiceTypeContainer:
	default:
		findings = append(findings, validateFinding{File: filePath, Service: serviceName, CheckID: "type.unsupported", Severity: validateSeverityError, Message: fmt.Sprintf("unsupported service type %q", svc.Type), Hint: "Use http, tcp, process, or container."})
	}
	mode := strings.ToLower(strings.TrimSpace(svc.Mode))
	if mode != "" {
		switch mode {
		case service.ServiceModeWatch, service.ServiceModeBuild, service.ServiceModeDaemon, service.ServiceModeTask:
		default:
			findings = append(findings, validateFinding{File: filePath, Service: serviceName, CheckID: "mode.unsupported", Severity: validateSeverityError, Message: fmt.Sprintf("unsupported service mode %q", svc.Mode), Hint: "Use watch, build, daemon, or task."})
		}
	}
	if serviceType == service.ServiceTypeProcess && strings.TrimSpace(svc.Command) == "" && len(svc.CommandArgs) == 0 {
		findings = append(findings, validateFinding{File: filePath, Service: serviceName, CheckID: "command.missing", Severity: validateSeverityWarning, Message: "process service has no command in azure.yaml", Hint: "Set command or make sure detection can infer how to run it."})
	}
	return findings
}

func renderValidateFindings(findings []validateFinding) {
	cliout.CommandHeader("validate", "Validate azure.yaml")
	if len(findings) == 0 {
		cliout.Success("azure.yaml is valid")
		return
	}
	for _, f := range findings {
		prefix := "WARN"
		if f.Severity == validateSeverityError {
			prefix = "FAIL"
		}
		target := f.CheckID
		if f.Service != "" {
			target = f.Service + " " + target
		}
		cliout.Item("[%s] %s: %s", prefix, target, f.Message)
		if f.Hint != "" {
			cliout.Item("      fix: %s", f.Hint)
		}
	}
}

func hasValidateErrors(findings []validateFinding) bool { return countValidateErrors(findings) > 0 }

func countValidateErrors(findings []validateFinding) int {
	count := 0
	for _, f := range findings {
		if f.Severity == validateSeverityError {
			count++
		}
	}
	return count
}

func sortValidateFindings(findings []validateFinding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Service != findings[j].Service {
			return findings[i].Service < findings[j].Service
		}
		return findings[i].CheckID < findings[j].CheckID
	})
}
