package testing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateTestConfigYAML(t *testing.T) {
	tests := []struct {
		name        string
		validations []ServiceValidation
		services    []ServiceInfo
		want        string
	}{
		{
			name: "single auto-detected service",
			validations: []ServiceValidation{
				{Name: "web", CanTest: true, Framework: "vitest"},
			},
			services: []ServiceInfo{
				{Name: "web", Config: nil},
			},
			want: `services:
  web:
    test:
      framework: vitest
`,
		},
		{
			name: "multiple auto-detected services",
			validations: []ServiceValidation{
				{Name: "api", CanTest: true, Framework: "jest"},
				{Name: "web", CanTest: true, Framework: "vitest"},
			},
			services: []ServiceInfo{
				{Name: "api", Config: nil},
				{Name: "web", Config: nil},
			},
			want: `services:
  api:
    test:
      framework: jest
  web:
    test:
      framework: vitest
`,
		},
		{
			name: "mixed: some with config, some without",
			validations: []ServiceValidation{
				{Name: "api", CanTest: true, Framework: "jest"},
				{Name: "web", CanTest: true, Framework: "vitest"},
			},
			services: []ServiceInfo{
				{Name: "api", Config: &ServiceTestConfig{Framework: "jest"}}, // Has config
				{Name: "web", Config: nil},                                   // Auto-detected
			},
			want: `services:
  web:
    test:
      framework: vitest
`,
		},
		{
			name: "no auto-detected services",
			validations: []ServiceValidation{
				{Name: "api", CanTest: true, Framework: "jest"},
			},
			services: []ServiceInfo{
				{Name: "api", Config: &ServiceTestConfig{Framework: "jest"}},
			},
			want: "",
		},
		{
			name: "non-testable services are excluded",
			validations: []ServiceValidation{
				{Name: "web", CanTest: true, Framework: "vitest"},
				{Name: "worker", CanTest: false, SkipReason: "No test files"},
			},
			services: []ServiceInfo{
				{Name: "web", Config: nil},
				{Name: "worker", Config: nil},
			},
			want: `services:
  web:
    test:
      framework: vitest
`,
		},
		{
			name:        "empty validations",
			validations: []ServiceValidation{},
			services:    []ServiceInfo{},
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateTestConfigYAML(tt.validations, tt.services)
			if got != tt.want {
				t.Errorf("GenerateTestConfigYAML() =\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}

func TestGetAutoDetectedServices(t *testing.T) {
	tests := []struct {
		name        string
		validations []ServiceValidation
		services    []ServiceInfo
		wantCount   int
		wantNames   []string
	}{
		{
			name: "all auto-detected",
			validations: []ServiceValidation{
				{Name: "web", CanTest: true, Framework: "vitest"},
				{Name: "api", CanTest: true, Framework: "jest"},
			},
			services: []ServiceInfo{
				{Name: "web", Config: nil},
				{Name: "api", Config: nil},
			},
			wantCount: 2,
			wantNames: []string{"web", "api"},
		},
		{
			name: "some have config",
			validations: []ServiceValidation{
				{Name: "web", CanTest: true, Framework: "vitest"},
				{Name: "api", CanTest: true, Framework: "jest"},
			},
			services: []ServiceInfo{
				{Name: "web", Config: nil},
				{Name: "api", Config: &ServiceTestConfig{Framework: "jest"}},
			},
			wantCount: 1,
			wantNames: []string{"web"},
		},
		{
			name: "none auto-detected",
			validations: []ServiceValidation{
				{Name: "web", CanTest: true, Framework: "vitest"},
			},
			services: []ServiceInfo{
				{Name: "web", Config: &ServiceTestConfig{Framework: "vitest"}},
			},
			wantCount: 0,
			wantNames: []string{},
		},
		{
			name: "non-testable excluded",
			validations: []ServiceValidation{
				{Name: "web", CanTest: false, SkipReason: "No tests"},
			},
			services: []ServiceInfo{
				{Name: "web", Config: nil},
			},
			wantCount: 0,
			wantNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetAutoDetectedServices(tt.validations, tt.services)
			if len(got) != tt.wantCount {
				t.Errorf("GetAutoDetectedServices() count = %d, want %d", len(got), tt.wantCount)
			}
			for i, want := range tt.wantNames {
				if i < len(got) && got[i].Name != want {
					t.Errorf("GetAutoDetectedServices()[%d].Name = %s, want %s", i, got[i].Name, want)
				}
			}
		})
	}
}

func TestSaveTestConfigToAzureYaml(t *testing.T) {
	tests := []struct {
		name           string
		initialContent string
		validations    []ServiceValidation
		services       []ServiceInfo
		wantContains   []string
		wantErr        bool
	}{
		{
			name: "add test config to service without it",
			initialContent: `name: myapp
services:
  web:
    language: typescript
    project: ./src/web
`,
			validations: []ServiceValidation{
				{Name: "web", CanTest: true, Framework: "vitest"},
			},
			services: []ServiceInfo{
				{Name: "web", Config: nil},
			},
			wantContains: []string{
				"test:",
				"framework: vitest",
			},
			wantErr: false,
		},
		{
			name: "don't modify service with existing test config",
			initialContent: `name: myapp
services:
  web:
    language: typescript
    project: ./src/web
    test:
      framework: jest
`,
			validations: []ServiceValidation{
				{Name: "web", CanTest: true, Framework: "vitest"},
			},
			services: []ServiceInfo{
				{Name: "web", Config: &ServiceTestConfig{Framework: "jest"}},
			},
			wantContains: []string{
				"framework: jest", // Should keep existing jest, not change to vitest
			},
			wantErr: false,
		},
		{
			name: "add test config to multiple services",
			initialContent: `name: myapp
services:
  web:
    language: typescript
    project: ./src/web
  api:
    language: python
    project: ./src/api
`,
			validations: []ServiceValidation{
				{Name: "web", CanTest: true, Framework: "vitest"},
				{Name: "api", CanTest: true, Framework: "pytest"},
			},
			services: []ServiceInfo{
				{Name: "web", Config: nil},
				{Name: "api", Config: nil},
			},
			wantContains: []string{
				"framework: vitest",
				"framework: pytest",
			},
			wantErr: false,
		},
		{
			name: "preserve existing fields",
			initialContent: `name: myapp
metadata:
  template: my-template
services:
  web:
    language: typescript
    project: ./src/web
    host: appservice
`,
			validations: []ServiceValidation{
				{Name: "web", CanTest: true, Framework: "vitest"},
			},
			services: []ServiceInfo{
				{Name: "web", Config: nil},
			},
			wantContains: []string{
				"name: myapp",
				"template: my-template",
				"language: typescript",
				"host: appservice",
				"framework: vitest",
			},
			wantErr: false,
		},
		{
			name: "nothing to save when all have config",
			initialContent: `name: myapp
services:
  web:
    language: typescript
    project: ./src/web
`,
			validations: []ServiceValidation{
				{Name: "web", CanTest: true, Framework: "vitest"},
			},
			services: []ServiceInfo{
				{Name: "web", Config: &ServiceTestConfig{Framework: "vitest"}},
			},
			wantContains: []string{}, // File shouldn't be modified
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()
			azureYamlPath := filepath.Join(tmpDir, "azure.yaml")

			// Write initial content
			if err := os.WriteFile(azureYamlPath, []byte(tt.initialContent), 0o644); err != nil {
				t.Fatalf("Failed to write initial azure.yaml: %v", err)
			}

			// Run SaveTestConfigToAzureYaml
			err := SaveTestConfigToAzureYaml(azureYamlPath, tt.validations, tt.services)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveTestConfigToAzureYaml() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Read back the file
			content, err := os.ReadFile(azureYamlPath)
			if err != nil {
				t.Fatalf("Failed to read azure.yaml after save: %v", err)
			}

			// Check expected content
			for _, want := range tt.wantContains {
				if !strings.Contains(string(content), want) {
					t.Errorf("SaveTestConfigToAzureYaml() result doesn't contain %q.\nContent:\n%s", want, string(content))
				}
			}
		})
	}
}

func TestSaveTestConfigToAzureYaml_InvalidPath(t *testing.T) {
	// Test with path containing traversal
	err := SaveTestConfigToAzureYaml("../../../etc/passwd", nil, nil)
	if err == nil {
		t.Error("SaveTestConfigToAzureYaml() should fail for path traversal")
	}
}

func TestSaveTestConfigToAzureYaml_NonexistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	nonexistent := filepath.Join(tmpDir, "nonexistent", "azure.yaml")

	err := SaveTestConfigToAzureYaml(nonexistent, []ServiceValidation{
		{Name: "web", CanTest: true, Framework: "vitest"},
	}, []ServiceInfo{
		{Name: "web", Config: nil},
	})

	if err == nil {
		t.Error("SaveTestConfigToAzureYaml() should fail for nonexistent file")
	}
}

func TestSaveTestConfigToAzureYaml_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")

	// Write invalid YAML
	if err := os.WriteFile(azureYamlPath, []byte("this: is: not: valid: yaml:"), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	err := SaveTestConfigToAzureYaml(azureYamlPath, []ServiceValidation{
		{Name: "web", CanTest: true, Framework: "vitest"},
	}, []ServiceInfo{
		{Name: "web", Config: nil},
	})

	if err == nil {
		t.Error("SaveTestConfigToAzureYaml() should fail for invalid YAML")
	}
}

func TestSaveTestConfigToAzureYaml_MissingServicesSection(t *testing.T) {
	tmpDir := t.TempDir()
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")

	// Write YAML without services section
	if err := os.WriteFile(azureYamlPath, []byte("name: myapp\n"), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	err := SaveTestConfigToAzureYaml(azureYamlPath, []ServiceValidation{
		{Name: "web", CanTest: true, Framework: "vitest"},
	}, []ServiceInfo{
		{Name: "web", Config: nil},
	})

	if err == nil {
		t.Error("SaveTestConfigToAzureYaml() should fail when services section is missing")
	}
}
