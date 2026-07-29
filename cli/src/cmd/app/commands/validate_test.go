package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateAzureYamlValid(t *testing.T) {
	dir := t.TempDir()
	writeValidateFile(t, dir, `
name: valid
services:
  api:
    host: local
    type: http
    project: ./api
    command: go run .
    ports:
      - "8080:80"
resources:
  db:
    type: postgres
`)
	mkdirValidateService(t, dir, "api")

	findings, err := validateAzureYamlFile(filepath.Join(dir, "azure.yaml"))
	if err != nil {
		t.Fatalf("validateAzureYamlFile failed: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %#v", findings)
	}
}

func TestValidateAzureYamlReportsOutOfRootProjectOnce(t *testing.T) {
	dir := t.TempDir()
	writeValidateFile(t, dir, `
name: invalid
services:
  api:
    host: local
    project: ../outside
    command: go run .
`)

	findings := mustValidateFindings(t, dir)
	if len(findings) != 1 {
		t.Fatalf("expected exactly 1 finding for a single root cause, got %d: %#v", len(findings), findings)
	}
	assertValidateFinding(t, findings, "api", "project.outside-root")
}

func TestValidateHostPort(t *testing.T) {
	tests := []struct {
		name    string
		mapping string
		want    int
		wantErr bool
	}{
		{name: "bare port", mapping: "8080", want: 8080},
		{name: "host and container port", mapping: "8080:80", want: 8080},
		{name: "bind address", mapping: "127.0.0.1:8080:80", want: 8080},
		{name: "ipv6 bind address", mapping: "[::1]:8080:80", want: 8080},
		{name: "protocol suffix", mapping: "8080/tcp", want: 8080},
		{name: "surrounding whitespace", mapping: " 8080:80 ", want: 8080},
		{name: "empty mapping", mapping: "   ", wantErr: true},
		{name: "port above range", mapping: "70000:80", wantErr: true},
		{name: "port below range", mapping: "0:80", wantErr: true},
		{name: "non-numeric port", mapping: "abc:80", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateHostPort(tt.mapping)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got port %d", tt.mapping, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.mapping, err)
			}
			if got != tt.want {
				t.Fatalf("expected port %d for %q, got %d", tt.want, tt.mapping, got)
			}
		})
	}
}

func TestValidateAzureYamlFindsMissingDependency(t *testing.T) {
	dir := t.TempDir()
	writeValidateFile(t, dir, `
name: invalid
services:
  api:
    host: local
    project: ./api
    command: go run .
    uses:
      - missing-db
`)
	mkdirValidateService(t, dir, "api")

	findings := mustValidateFindings(t, dir)
	assertValidateFinding(t, findings, "api", "uses.unknown")
}

func TestValidateAzureYamlFindsInvalidAndDuplicatePorts(t *testing.T) {
	dir := t.TempDir()
	writeValidateFile(t, dir, `
name: invalid
services:
  api:
    host: local
    project: ./api
    command: go run .
    ports:
      - "70000:80"
  web:
    host: local
    project: ./web
    command: npm run dev
    ports:
      - "8080:80"
  admin:
    host: local
    project: ./admin
    command: npm run dev
    ports:
      - "8080:80"
`)
	for _, name := range []string{"api", "web", "admin"} {
		mkdirValidateService(t, dir, name)
	}

	findings := mustValidateFindings(t, dir)
	assertValidateFinding(t, findings, "api", "port.invalid")
	assertValidateCheck(t, findings, "port.duplicate")
}

func TestValidateAzureYamlFindsMissingProjectPath(t *testing.T) {
	dir := t.TempDir()
	writeValidateFile(t, dir, `
name: invalid
services:
  api:
    host: local
    project: ./missing
    command: go run .
`)

	findings := mustValidateFindings(t, dir)
	assertValidateFinding(t, findings, "api", "project.missing")
}

func TestValidateAzureYamlFindsUnsupportedMode(t *testing.T) {
	dir := t.TempDir()
	writeValidateFile(t, dir, `
name: invalid
services:
  worker:
    host: local
    project: ./worker
    type: process
    mode: forever
    command: go run .
`)
	mkdirValidateService(t, dir, "worker")

	findings := mustValidateFindings(t, dir)
	assertValidateFinding(t, findings, "worker", "mode.unsupported")
}

func mustValidateFindings(t *testing.T, dir string) []validateFinding {
	t.Helper()
	findings, err := validateAzureYamlFile(filepath.Join(dir, "azure.yaml"))
	if err != nil {
		t.Fatalf("validateAzureYamlFile failed: %v", err)
	}
	return findings
}

func assertValidateFinding(t *testing.T, findings []validateFinding, serviceName, checkID string) {
	t.Helper()
	for _, finding := range findings {
		if finding.Service == serviceName && finding.CheckID == checkID {
			return
		}
	}
	t.Fatalf("missing finding %s/%s in %#v", serviceName, checkID, findings)
}

func assertValidateCheck(t *testing.T, findings []validateFinding, checkID string) {
	t.Helper()
	for _, finding := range findings {
		if finding.CheckID == checkID {
			return
		}
	}
	t.Fatalf("missing finding %s in %#v", checkID, findings)
}

func writeValidateFile(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte(content), 0o600); err != nil {
		t.Fatalf("write azure.yaml: %v", err)
	}
}

func mkdirValidateService(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.Mkdir(filepath.Join(dir, name), 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", name, err)
	}
}
