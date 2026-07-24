package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJoinOpenURLPath(t *testing.T) {
	got, err := joinOpenURLPath("http://localhost:3000/api", "/health")
	if err != nil {
		t.Fatalf("joinOpenURLPath failed: %v", err)
	}
	if got != "http://localhost:3000/api/health" {
		t.Fatalf("URL = %q, want http://localhost:3000/api/health", got)
	}
}

func TestResolveOpenServiceURLFromCustomURL(t *testing.T) {
	dir := t.TempDir()
	writeOpenAzureYaml(t, dir, `
name: open-test
services:
  web:
    host: local
    project: ./web
    local:
      customUrl: https://web.example.test
`)
	mkdirOpenService(t, dir, "web")

	got, err := resolveOpenServiceURL(t.Context(), dir, "web", "/health")
	if err != nil {
		t.Fatalf("resolveOpenServiceURL failed: %v", err)
	}
	if got != "https://web.example.test/health" {
		t.Fatalf("URL = %q, want https://web.example.test/health", got)
	}
}

func TestResolveOpenServiceURLFromPorts(t *testing.T) {
	dir := t.TempDir()
	writeOpenAzureYaml(t, dir, `
name: open-test
services:
  api:
    host: local
    project: ./api
    ports:
      - "8080:80"
`)
	mkdirOpenService(t, dir, "api")

	got, err := resolveOpenServiceURL(t.Context(), dir, "api", "docs")
	if err != nil {
		t.Fatalf("resolveOpenServiceURL failed: %v", err)
	}
	if got != "http://localhost:8080/docs" {
		t.Fatalf("URL = %q, want http://localhost:8080/docs", got)
	}
}

func TestResolveOpenServiceURLMissingURL(t *testing.T) {
	dir := t.TempDir()
	writeOpenAzureYaml(t, dir, `
name: open-test
services:
  worker:
    host: local
    project: ./worker
`)
	mkdirOpenService(t, dir, "worker")

	_, err := resolveOpenServiceURL(t.Context(), dir, "worker", "")
	if err == nil {
		t.Fatal("expected missing URL error")
	}
	if !strings.Contains(err.Error(), "no known URL") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeOpenAzureYaml(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte(content), 0o600); err != nil {
		t.Fatalf("write azure.yaml: %v", err)
	}
}

func mkdirOpenService(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.Mkdir(filepath.Join(dir, name), 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", name, err)
	}
}
