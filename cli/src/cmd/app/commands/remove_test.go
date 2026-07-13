package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
