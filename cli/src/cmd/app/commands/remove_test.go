package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-core/cliout"
)

func writeTempAzureYaml(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "azure.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write azure.yaml: %v", err)
	}
	return path
}

func TestRemoveServiceFromYaml(t *testing.T) {
	content := `name: test-app
services:
  api:
    language: python
    project: ./api
  redis:
    host: containerapp
    image: redis:7-alpine
    ports:
      - "6379:6379"
  web:
    language: js
    project: ./web
`
	path := writeTempAzureYaml(t, content)

	removed, remaining, err := removeServiceFromYaml(path, "redis")
	if err != nil {
		t.Fatalf("removeServiceFromYaml() error: %v", err)
	}
	if !removed {
		t.Fatal("removeServiceFromYaml() removed = false, want true")
	}

	wantRemaining := []string{"api", "web"}
	if strings.Join(remaining, ",") != strings.Join(wantRemaining, ",") {
		t.Errorf("remaining = %v, want %v", remaining, wantRemaining)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read azure.yaml: %v", err)
	}
	got := string(data)

	if strings.Contains(got, "redis:") {
		t.Errorf("azure.yaml still contains redis after removal:\n%s", got)
	}
	if strings.Contains(got, "6379:6379") {
		t.Errorf("azure.yaml still contains redis ports after removal:\n%s", got)
	}
	// Siblings and top-level keys are preserved.
	for _, want := range []string{"name: test-app", "api:", "web:"} {
		if !strings.Contains(got, want) {
			t.Errorf("azure.yaml missing %q after removal:\n%s", want, got)
		}
	}
}

func TestRemoveServiceFromYamlUnknown(t *testing.T) {
	content := `name: test-app
services:
  api:
    language: python
  web:
    language: js
`
	path := writeTempAzureYaml(t, content)

	removed, remaining, err := removeServiceFromYaml(path, "redis")
	if err != nil {
		t.Fatalf("removeServiceFromYaml() error: %v", err)
	}
	if removed {
		t.Error("removeServiceFromYaml() removed = true for unknown service, want false")
	}

	want := []string{"api", "web"}
	if strings.Join(remaining, ",") != strings.Join(want, ",") {
		t.Errorf("remaining = %v, want %v", remaining, want)
	}

	// The file must be untouched when nothing is removed.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read azure.yaml: %v", err)
	}
	if string(data) != content {
		t.Errorf("azure.yaml changed on a no-op removal:\n%s", string(data))
	}
}

func TestRemoveServiceFromYamlNoServicesSection(t *testing.T) {
	content := `name: test-app
`
	path := writeTempAzureYaml(t, content)

	removed, remaining, err := removeServiceFromYaml(path, "redis")
	if err != nil {
		t.Fatalf("removeServiceFromYaml() error: %v", err)
	}
	if removed {
		t.Error("removeServiceFromYaml() removed = true with no services section, want false")
	}
	if len(remaining) != 0 {
		t.Errorf("remaining = %v, want empty", remaining)
	}
}

func TestServiceNotFoundMessage(t *testing.T) {
	tests := []struct {
		name      string
		service   string
		remaining []string
		wantParts []string
	}{
		{
			name:      "with remaining services",
			service:   "redis",
			remaining: []string{"api", "web"},
			wantParts: []string{`"redis" not found`, "api, web"},
		},
		{
			name:      "no remaining services",
			service:   "redis",
			remaining: nil,
			wantParts: []string{`"redis" not found`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := serviceNotFoundMessage(tt.service, tt.remaining)
			for _, part := range tt.wantParts {
				if !strings.Contains(msg, part) {
					t.Errorf("message %q missing %q", msg, part)
				}
			}
			if tt.remaining == nil && strings.Contains(msg, "Current services") {
				t.Errorf("message should not list services when none remain: %q", msg)
			}
		})
	}
}

func TestRemoveResultJSON(t *testing.T) {
	b, err := json.Marshal(RemoveResult{
		Service: "redis",
		Removed: true,
		Message: "Removed redis from azure.yaml",
	})
	if err != nil {
		t.Fatalf("marshal RemoveResult: %v", err)
	}
	got := string(b)
	for _, want := range []string{`"service":"redis"`, `"removed":true`, `"message":"Removed redis from azure.yaml"`} {
		if !strings.Contains(got, want) {
			t.Errorf("RemoveResult JSON %s missing %q", got, want)
		}
	}
}

func TestRunRemoveNoArgs(t *testing.T) {
	err := runRemove(nil, nil)
	if err == nil {
		t.Fatal("runRemove() with no args returned nil, want error")
	}
	if !strings.Contains(err.Error(), "specify a service name") {
		t.Errorf("runRemove() error = %q, want it to mention specifying a service name", err.Error())
	}
}

func TestNewRemoveCommandUsesRequiredServiceArg(t *testing.T) {
	cmd := NewRemoveCommand()
	if cmd.Use != "remove <service>" {
		t.Fatalf("Use = %q, want remove <service>", cmd.Use)
	}
	if cmd.ValidArgsFunction == nil {
		t.Fatal("ValidArgsFunction should be set")
	}
}

func TestRemoveServiceFromYamlLastServiceKeepsEmptyServicesMapping(t *testing.T) {
	content := `name: test-app
services:
  redis:
    image: redis:7-alpine
`
	path := writeTempAzureYaml(t, content)

	removed, remaining, err := removeServiceFromYaml(path, "redis")
	if err != nil {
		t.Fatalf("removeServiceFromYaml() error: %v", err)
	}
	if !removed {
		t.Fatal("removeServiceFromYaml() removed = false, want true")
	}
	if len(remaining) != 0 {
		t.Fatalf("remaining = %v, want empty", remaining)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read azure.yaml: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "services: {}") {
		t.Fatalf("azure.yaml should keep an empty services mapping, got:\n%s", got)
	}
}

func TestRunRemoveJSONOutputRemovedAndNotFound(t *testing.T) {
	originalFormat := cliout.GetFormat()
	t.Cleanup(func() { _ = cliout.SetFormat(string(originalFormat)) })
	if err := cliout.SetFormat("json"); err != nil {
		t.Fatalf("SetFormat(json) error = %v", err)
	}

	dir := t.TempDir()
	content := `name: test-app
services:
  api:
    language: python
  redis:
    image: redis:7-alpine
`
	if err := os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write azure.yaml: %v", err)
	}
	t.Chdir(dir)

	out, err := captureStdout(t, func() error {
		return runRemove(nil, []string{"redis"})
	})
	if err != nil {
		t.Fatalf("runRemove(redis) error = %v", err)
	}
	var removed RemoveResult
	if err := json.Unmarshal([]byte(out), &removed); err != nil {
		t.Fatalf("invalid JSON output %q: %v", out, err)
	}
	if removed.Service != "redis" || !removed.Removed || !strings.Contains(removed.Message, "Removed redis") {
		t.Fatalf("removed result = %+v, want redis removed", removed)
	}

	out, err = captureStdout(t, func() error {
		return runRemove(nil, []string{"worker"})
	})
	if err != nil {
		t.Fatalf("runRemove(worker) JSON not-found error = %v, want nil", err)
	}
	var notFound RemoveResult
	if err := json.Unmarshal([]byte(out), &notFound); err != nil {
		t.Fatalf("invalid JSON output %q: %v", out, err)
	}
	if notFound.Service != "worker" || notFound.Removed || !strings.Contains(notFound.Message, `"worker" not found`) {
		t.Fatalf("not-found result = %+v, want worker not removed", notFound)
	}
}

func TestRunRemoveTextModeSuccessAndNotFound(t *testing.T) {
	originalFormat := cliout.GetFormat()
	t.Cleanup(func() { _ = cliout.SetFormat(string(originalFormat)) })
	if err := cliout.SetFormat("default"); err != nil {
		t.Fatalf("SetFormat(default) error = %v", err)
	}

	dir := t.TempDir()
	content := `name: test-app
services:
  api:
    language: python
  web:
    language: js
`
	if err := os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write azure.yaml: %v", err)
	}
	t.Chdir(dir)

	if err := runRemove(nil, []string{"web"}); err != nil {
		t.Fatalf("runRemove(web) error = %v", err)
	}
	err := runRemove(nil, []string{"worker"})
	if err == nil {
		t.Fatal("runRemove(worker) returned nil, want not-found error")
	}
	if !strings.Contains(err.Error(), `"worker" not found`) || !strings.Contains(err.Error(), "api") {
		t.Fatalf("runRemove(worker) error = %q, want not-found message with remaining services", err.Error())
	}
}

func TestRunRemoveFindAzureYamlFailure(t *testing.T) {
	originalFormat := cliout.GetFormat()
	t.Cleanup(func() { _ = cliout.SetFormat(string(originalFormat)) })
	if err := cliout.SetFormat("default"); err != nil {
		t.Fatalf("SetFormat(default) error = %v", err)
	}
	t.Chdir(t.TempDir())

	err := runRemove(nil, []string{"redis"})
	if err == nil {
		t.Fatal("runRemove() without azure.yaml returned nil, want error")
	}
	if !strings.Contains(err.Error(), "find azure.yaml") {
		t.Fatalf("runRemove() error = %q, want find azure.yaml", err.Error())
	}
}

func TestRemoveServiceFromYamlWritesAtomically(t *testing.T) {
	content := `name: test-app
services:
  api:
    language: python
  redis:
    image: redis:7-alpine
`
	path := writeTempAzureYaml(t, content)

	removed, _, err := removeServiceFromYaml(path, "redis")
	if err != nil {
		t.Fatalf("removeServiceFromYaml() error: %v", err)
	}
	if !removed {
		t.Fatal("removeServiceFromYaml() removed = false, want true")
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temporary file cleanup error = %v, want not exist", err)
	}
}
