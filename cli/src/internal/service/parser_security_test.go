package service

// parser_security_test.go validates that ParseAzureYaml rejects service project
// paths that escape the project root (CWE-22, SEC-012).
//
// These are internal-package tests (package service, not service_test) so that
// they can call ParseAzureYaml directly without exporting it.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeAzureYaml creates an azure.yaml with the given content in dir.
func writeAzureYaml(t *testing.T, dir, content string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	p := filepath.Join(dir, "azure.yaml")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("write azure.yaml: %v", err)
	}
	return p
}

// TestParseAzureYaml_ValidatePath_TraversalRejected tests acceptance criterion 4:
// a service project set to "../../etc/passwd" must be rejected.
func TestParseAzureYaml_ValidatePath_TraversalRejected(t *testing.T) {
	root := t.TempDir()
	writeAzureYaml(t, root, `
name: test
services:
  api:
    project: ../../etc/passwd
    host: appservice
`)

	_, err := ParseAzureYaml(root)
	if err == nil {
		t.Fatal("ParseAzureYaml() expected error for traversal path, got nil")
	}
	if !strings.Contains(err.Error(), "outside the project root") {
		t.Errorf("error = %q; want it to mention 'outside the project root'", err.Error())
	}
}

// TestParseAzureYaml_ValidatePath_AbsoluteOutsideRootRejected tests acceptance criterion 5:
// a service project set to an absolute path outside the project root must be rejected.
func TestParseAzureYaml_ValidatePath_AbsoluteOutsideRootRejected(t *testing.T) {
	root := t.TempDir()
	// Use the parent of root as the escape target — it always exists.
	outsideDir := filepath.ToSlash(filepath.Dir(root))

	content := "name: test\nservices:\n  api:\n    project: " + outsideDir + "\n    host: appservice\n"
	writeAzureYaml(t, root, content)

	_, err := ParseAzureYaml(root)
	if err == nil {
		t.Fatal("ParseAzureYaml() expected error for absolute-outside path, got nil")
	}
	if !strings.Contains(err.Error(), "outside the project root") {
		t.Errorf("error = %q; want it to mention 'outside the project root'", err.Error())
	}
}

// TestParseAzureYaml_ValidatePath_RelativeDotSlashAccepted tests acceptance criterion 6:
// "project: ./src/myapp" must be accepted when the directory exists.
func TestParseAzureYaml_ValidatePath_RelativeDotSlashAccepted(t *testing.T) {
	root := t.TempDir()
	subDir := filepath.Join(root, "src", "myapp")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", subDir, err)
	}
	writeAzureYaml(t, root, `
name: test
services:
  api:
    project: ./src/myapp
    host: appservice
`)

	got, err := ParseAzureYaml(root)
	if err != nil {
		t.Fatalf("ParseAzureYaml() unexpected error: %v", err)
	}
	svc, ok := got.Services["api"]
	if !ok {
		t.Fatal("service 'api' not found in parsed result")
	}
	if !filepath.IsAbs(svc.Project) {
		t.Errorf("service project path %q is not absolute", svc.Project)
	}
	if !strings.HasPrefix(svc.Project, root) {
		t.Errorf("service project path %q does not start with project root %q", svc.Project, root)
	}
}

// TestParseAzureYaml_ValidatePath_RelativeNoSlashAccepted tests acceptance criterion 7:
// "project: src/myapp" (no leading "./" ) must be accepted when the directory exists.
func TestParseAzureYaml_ValidatePath_RelativeNoSlashAccepted(t *testing.T) {
	root := t.TempDir()
	subDir := filepath.Join(root, "src", "myapp")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", subDir, err)
	}
	writeAzureYaml(t, root, `
name: test
services:
  api:
    project: src/myapp
    host: appservice
`)

	got, err := ParseAzureYaml(root)
	if err != nil {
		t.Fatalf("ParseAzureYaml() unexpected error: %v", err)
	}
	svc, ok := got.Services["api"]
	if !ok {
		t.Fatal("service 'api' not found in parsed result")
	}
	if !filepath.IsAbs(svc.Project) {
		t.Errorf("service project path %q is not absolute", svc.Project)
	}
	if !strings.HasPrefix(svc.Project, root) {
		t.Errorf("service project path %q does not start with project root %q", svc.Project, root)
	}
}

// TestParseAzureYaml_ValidatePath_MultipleServices_OneEscapes verifies that a
// YAML with multiple services fails fast when any service project escapes the root.
func TestParseAzureYaml_ValidatePath_MultipleServices_OneEscapes(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "web"), 0o755); err != nil {
		t.Fatalf("mkdir web: %v", err)
	}
	writeAzureYaml(t, root, `
name: test
services:
  web:
    project: ./web
    host: appservice
  evil:
    project: ../../etc/passwd
    host: appservice
`)

	_, err := ParseAzureYaml(root)
	if err == nil {
		t.Fatal("ParseAzureYaml() expected error when one service escapes root, got nil")
	}
	if !strings.Contains(err.Error(), "outside the project root") {
		t.Errorf("error = %q; want it to mention 'outside the project root'", err.Error())
	}
}

// TestParseAzureYaml_ValidatePath_NoProjectField verifies that services without
// a project field are unaffected by the path validation.
func TestParseAzureYaml_ValidatePath_NoProjectField(t *testing.T) {
	root := t.TempDir()
	writeAzureYaml(t, root, `
name: test
services:
  api:
    host: appservice
`)

	got, err := ParseAzureYaml(root)
	if err != nil {
		t.Fatalf("ParseAzureYaml() unexpected error: %v", err)
	}
	svc, ok := got.Services["api"]
	if !ok {
		t.Fatal("service 'api' not found in parsed result")
	}
	if svc.Project != "" {
		t.Errorf("expected empty project for service without project field, got %q", svc.Project)
	}
}
