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
	if hasValidateErrors(findings) {
		t.Fatalf("expected no errors, got %#v", findings)
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
