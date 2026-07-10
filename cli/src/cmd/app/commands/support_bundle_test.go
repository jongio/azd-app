package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-core/cliout"
)

func TestNewSupportBundleCommand(t *testing.T) {
	cmd := NewSupportBundleCommand()
	if cmd == nil {
		t.Fatal("NewSupportBundleCommand returned nil")
	}
	if cmd.Use != "support-bundle" {
		t.Fatalf("Use = %q, want support-bundle", cmd.Use)
	}
	for _, name := range []string{"output", "tail", "service", "dry-run"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("%s flag not found", name)
		}
	}
}

func TestRunSupportBundleDryRunText(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte(`
	name: bundle-test
	services:
	  api:
	    host: local
	    language: go
	`), 0o600); err != nil {
		t.Fatalf("write azure.yaml: %v", err)
	}
	t.Chdir(dir)
	outDir := filepath.Join(dir, "bundle")

	if err := cliout.SetFormat("default"); err != nil {
		t.Fatalf("set output format: %v", err)
	}
	out, err := captureStdout(t, func() error {
		return runSupportBundle(t.Context(), &supportBundleOptions{
			output:  outDir,
			tail:    0,
			service: "api",
			dryRun:  true,
		})
	})
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}

	if _, statErr := os.Stat(outDir); !os.IsNotExist(statErr) {
		t.Fatalf("dry-run wrote output folder, stat error: %v", statErr)
	}
	for _, want := range []string{"Support bundle dry run", outDir, "manifest.json", "azure.yaml.redacted", "logs.json", "Service: api"} {
		if !strings.Contains(out, want) {
			t.Fatalf("dry-run output missing %q:\n%s", want, out)
		}
	}
}

func TestRunSupportBundleDryRunJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte("name: bundle-test\nservices: {}\n"), 0o600); err != nil {
		t.Fatalf("write azure.yaml: %v", err)
	}
	t.Chdir(dir)
	outDir := filepath.Join(dir, "bundle")

	originalFormat := cliout.GetFormat()
	if err := cliout.SetFormat("json"); err != nil {
		t.Fatalf("set output format: %v", err)
	}
	t.Cleanup(func() { _ = cliout.SetFormat(string(originalFormat)) })

	out, err := captureStdout(t, func() error {
		return runSupportBundle(t.Context(), &supportBundleOptions{
			output: outDir,
			tail:   5,
			dryRun: true,
		})
	})
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}

	var plan supportBundlePlan
	if err := json.Unmarshal([]byte(out), &plan); err != nil {
		t.Fatalf("dry-run JSON is invalid: %v\n%s", err, out)
	}
	if !plan.DryRun {
		t.Fatal("dryRun was false")
	}
	if plan.OutputDir != outDir {
		t.Fatalf("outputDir = %q, want %q", plan.OutputDir, outDir)
	}
	if plan.Tail != 5 {
		t.Fatalf("tail = %d, want 5", plan.Tail)
	}
	if len(plan.Files) != len(plannedSupportBundleFiles()) {
		t.Fatalf("files = %#v", plan.Files)
	}
	if _, statErr := os.Stat(outDir); !os.IsNotExist(statErr) {
		t.Fatalf("dry-run wrote output folder, stat error: %v", statErr)
	}
}

func TestRunSupportBundleDryRunValidatesInputs(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte("name: bundle-test\nservices: {}\n"), 0o600); err != nil {
		t.Fatalf("write azure.yaml: %v", err)
	}
	t.Chdir(dir)

	if err := runSupportBundle(t.Context(), &supportBundleOptions{tail: -1, dryRun: true}); err == nil || !strings.Contains(err.Error(), "--tail must be zero or greater") {
		t.Fatalf("expected tail validation error, got %v", err)
	}
	if err := runSupportBundle(t.Context(), &supportBundleOptions{tail: 0, service: "../api", dryRun: true}); err == nil || !strings.Contains(err.Error(), "invalid service name") {
		t.Fatalf("expected service validation error, got %v", err)
	}
}

func TestRedactSecretText(t *testing.T) {
	got := redactSecretText("token=abc123456 password: hunter222 normal")
	if strings.Contains(got, "abc123456") || strings.Contains(got, "hunter222") {
		t.Fatalf("secret text was not redacted: %s", got)
	}
	if !strings.Contains(got, "normal") {
		t.Fatalf("non-secret text was removed: %s", got)
	}
}

func TestWriteRedactedAzureYaml(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "azure.yaml")
	target := filepath.Join(dir, "azure.redacted.yaml")
	data := []byte(`
name: test
services:
  api:
    host: local
    environment:
      API_KEY: supersecretvalue
      PUBLIC_URL: http://localhost:3000
`)
	if err := os.WriteFile(source, data, 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if err := writeRedactedAzureYaml(source, target); err != nil {
		t.Fatalf("writeRedactedAzureYaml failed: %v", err)
	}
	out, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	text := string(out)
	if strings.Contains(text, "supersecretvalue") {
		t.Fatalf("secret value was not redacted:\n%s", text)
	}
	if !strings.Contains(text, "PUBLIC_URL") {
		t.Fatalf("non-secret value missing:\n%s", text)
	}
}

func TestRunSupportBundleCreatesFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte(`
name: bundle-test
reqs:
  - name: azd-app-missing-tool
    minVersion: 1.0.0
services:
  api:
    host: local
    language: go
    environment:
      API_TOKEN: secretvalue123
`), 0o600); err != nil {
		t.Fatalf("write azure.yaml: %v", err)
	}
	t.Chdir(dir)

	outDir := filepath.Join(dir, "bundle")
	err := runSupportBundle(t.Context(), &supportBundleOptions{output: outDir, tail: 0})
	if err != nil {
		t.Fatalf("runSupportBundle failed: %v", err)
	}

	for _, name := range []string{"manifest.json", "azure.yaml.redacted", "services.json", "requirements.json", "health.json", "logs.json"} {
		if _, err := os.Stat(filepath.Join(outDir, name)); err != nil {
			t.Fatalf("%s was not created: %v", name, err)
		}
	}
	var manifest supportBundleManifest
	data, err := os.ReadFile(filepath.Join(outDir, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("manifest is invalid JSON: %v", err)
	}
	if len(manifest.Files) == 0 {
		t.Fatal("manifest has no files")
	}
	redacted, err := os.ReadFile(filepath.Join(outDir, "azure.yaml.redacted"))
	if err != nil {
		t.Fatalf("read redacted azure.yaml: %v", err)
	}
	if strings.Contains(string(redacted), "secretvalue123") {
		t.Fatalf("support bundle contains unredacted secret:\n%s", string(redacted))
	}

	var reqs ReqsResult
	reqData, err := os.ReadFile(filepath.Join(outDir, "requirements.json"))
	if err != nil {
		t.Fatalf("read requirements: %v", err)
	}
	if err := json.Unmarshal(reqData, &reqs); err != nil {
		t.Fatalf("requirements.json is invalid JSON: %v", err)
	}
	if reqs.Satisfied {
		t.Fatal("requirements should record the failed missing-tool check")
	}
	if len(reqs.Reqs) != 1 || reqs.Reqs[0].Name != "azd-app-missing-tool" {
		t.Fatalf("unexpected requirement results: %#v", reqs.Reqs)
	}
}
