package commands

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/jongio/azd-app/cli/src/internal/secretscan"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/cliout"
)

// runSecretScan performs the advisory hardcoded-secret scan for the run
// preflight. It inspects the environment blocks declared in azure.yaml and any
// tracked .env files, then prints a short advisory when a literal secret value
// is found. It never blocks the run and never mutates the child environment.
func runSecretScan(azureYaml *service.AzureYaml, services map[string]service.Service, azureYamlDir string) {
	if runSkipSecretScan {
		return
	}
	if azureYaml != nil && azureYaml.Security != nil && azureYaml.Security.SkipSecretScan {
		return
	}
	if cliout.IsJSON() {
		return
	}
	reportSecretFindings(collectSecretFindings(services, azureYamlDir))
}

// collectSecretFindings gathers findings from every service environment block
// and from each tracked .env file under azureYamlDir. It performs no output, so
// callers and tests can inspect the result directly.
func collectSecretFindings(services map[string]service.Service, azureYamlDir string) []secretscan.Finding {
	var findings []secretscan.Finding
	for _, name := range sortedServiceNames(services) {
		svc := services[name]
		findings = append(findings, secretscan.ScanEnv("azure.yaml (service: "+name+")", svc.GetEnvironment())...)
	}
	for _, path := range trackedEnvFiles(azureYamlDir) {
		env, err := service.LoadDotEnv(path)
		if err != nil {
			continue
		}
		findings = append(findings, secretscan.ScanEnv(displayPath(azureYamlDir, path), env)...)
	}
	return findings
}

// sortedServiceNames returns the service names in stable order.
func sortedServiceNames(services map[string]service.Service) []string {
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// trackedEnvFiles returns .env files under azureYamlDir that exist and are
// tracked by git. Ignored files (the recommended place for local secrets) are
// left out on purpose. If git is unavailable the result is empty.
func trackedEnvFiles(azureYamlDir string) []string {
	candidates := []string{filepath.Join(azureYamlDir, ".env")}
	if runEnvFile != "" {
		p := runEnvFile
		if !filepath.IsAbs(p) {
			p = filepath.Join(azureYamlDir, p)
		}
		candidates = append(candidates, p)
	}

	seen := make(map[string]bool, len(candidates))
	var tracked []string
	for _, p := range candidates {
		if seen[p] {
			continue
		}
		seen[p] = true
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if gitTracksFile(azureYamlDir, p) {
			tracked = append(tracked, p)
		}
	}
	return tracked
}

// gitTracksFile reports whether path is tracked by the git repository rooted at
// or above dir. Any error (git missing, not a repo, file untracked) yields false.
func gitTracksFile(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		rel = path
	}
	// Local git plumbing only, no network I/O, and no ambient context to honour
	// at any caller in this path.
	cmd := exec.Command("git", "-C", dir, "ls-files", "--error-unmatch", "--", rel) //nolint:noctx // see above
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run() == nil
}

// displayPath returns a short, workspace-relative label for a file path.
func displayPath(base, path string) string {
	if rel, err := filepath.Rel(base, path); err == nil {
		return rel
	}
	return filepath.Base(path)
}

// reportSecretFindings prints the advisory. It is a no-op when there is nothing
// to report.
func reportSecretFindings(findings []secretscan.Finding) {
	if len(findings) == 0 {
		return
	}
	cliout.Warning("Possible hardcoded secrets in tracked config (%d):", len(findings))
	for _, f := range findings {
		cliout.Item("%s: %s (%s)", f.Source, f.Key, f.Detail)
	}
	cliout.Item("Move these to a Key Vault reference or a gitignored .env file. Use --skip-secret-scan to silence this check.")
}
