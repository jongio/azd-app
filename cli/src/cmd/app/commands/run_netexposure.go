package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jongio/azd-app/cli/src/internal/netexposure"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/cliout"
)

// runExposureCheck warns when any service is configured to bind to every
// network interface (for example HOST=0.0.0.0 or ASPNETCORE_URLS=http://+:5000).
// Binding to all interfaces makes a local service reachable by anyone on the
// same network, which is rarely intended on a developer machine.
//
// The check is advisory: it never blocks the run or changes how a service
// starts. It can be silenced with --skip-exposure-check or by setting
// security.skipExposureCheck in azure.yaml.
func runExposureCheck(services map[string]service.Service, azureYamlDir string, azureYaml *service.AzureYaml) {
	if runSkipExposureCheck {
		return
	}
	if azureYaml != nil && azureYaml.Security != nil && azureYaml.Security.SkipExposureCheck {
		return
	}

	findings := collectExposureFindings(services, azureYamlDir)
	if len(findings) == 0 {
		return
	}
	reportExposureFindings(findings)
}

// collectExposureFindings gathers all-interface bind findings from each
// service's declared environment and from any .env files near the project or a
// service directory.
func collectExposureFindings(services map[string]service.Service, azureYamlDir string) []netexposure.Finding {
	var findings []netexposure.Finding
	seen := make(map[string]bool)

	add := func(list []netexposure.Finding) {
		for _, f := range list {
			dedupeKey := f.Source + "\x00" + f.Key
			if seen[dedupeKey] {
				continue
			}
			seen[dedupeKey] = true
			findings = append(findings, f)
		}
	}

	for _, name := range sortedExposureServiceNames(services) {
		svc := services[name]
		source := fmt.Sprintf("azure.yaml (service: %s)", name)
		add(netexposure.ScanEnv(source, svc.GetEnvironment()))
	}

	for _, path := range exposureEnvFiles(services, azureYamlDir) {
		env, err := service.LoadDotEnv(path)
		if err != nil {
			continue
		}
		add(netexposure.ScanEnv(exposureDisplayPath(azureYamlDir, path), env))
	}

	return findings
}

// exposureEnvFiles returns the set of .env files to scan: the project-root
// .env plus one per service directory. Order is deterministic.
func exposureEnvFiles(services map[string]service.Service, azureYamlDir string) []string {
	seen := make(map[string]bool)
	var paths []string

	addPath := func(p string) {
		if p == "" || seen[p] {
			return
		}
		if info, err := os.Stat(p); err != nil || info.IsDir() {
			return
		}
		seen[p] = true
		paths = append(paths, p)
	}

	addPath(filepath.Join(azureYamlDir, ".env"))
	for _, name := range sortedExposureServiceNames(services) {
		svc := services[name]
		if svc.Project != "" {
			addPath(filepath.Join(azureYamlDir, svc.Project, ".env"))
		}
	}

	return paths
}

// reportExposureFindings prints an advisory warning describing each finding and
// how to bind to loopback instead. It stays quiet in JSON output mode.
func reportExposureFindings(findings []netexposure.Finding) {
	if cliout.IsJSON() {
		return
	}

	noun := "value"
	if len(findings) != 1 {
		noun = "values"
	}
	cliout.Warning("Found %d configuration %s that bind a service to all network interfaces:", len(findings), noun)
	for _, f := range findings {
		cliout.Item("%s: %s=%s", f.Source, f.Key, f.Value)
	}
	cliout.Info("Binding to 0.0.0.0 (or ::) exposes the service to your whole network. For local development, bind to 127.0.0.1 instead.")
	cliout.Info("Silence this check with --skip-exposure-check or security.skipExposureCheck in azure.yaml.")
}

func sortedExposureServiceNames(services map[string]service.Service) []string {
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// exposureDisplayPath returns path relative to azureYamlDir when possible, so
// warnings show a short, readable location.
func exposureDisplayPath(azureYamlDir, path string) string {
	if rel, err := filepath.Rel(azureYamlDir, path); err == nil {
		return rel
	}
	return path
}
