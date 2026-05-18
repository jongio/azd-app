package service

import (
	"testing"
)

func TestFilterServices(t *testing.T) {
	azureYaml := &AzureYaml{
		Services: map[string]Service{
			"api":      {Host: "appservice"},
			"web":      {Host: "appservice"},
			"worker":   {Host: "containerapp"},
		},
	}

	t.Run("no filter returns all", func(t *testing.T) {
		result := FilterServices(azureYaml, nil)
		if len(result) != 3 {
			t.Errorf("got %d services, want 3", len(result))
		}
	})

	t.Run("empty filter returns all", func(t *testing.T) {
		result := FilterServices(azureYaml, []string{})
		if len(result) != 3 {
			t.Errorf("got %d services, want 3", len(result))
		}
	})

	t.Run("filter specific services", func(t *testing.T) {
		result := FilterServices(azureYaml, []string{"api", "web"})
		if len(result) != 2 {
			t.Errorf("got %d services, want 2", len(result))
		}
		if _, ok := result["api"]; !ok {
			t.Error("expected 'api' in result")
		}
		if _, ok := result["web"]; !ok {
			t.Error("expected 'web' in result")
		}
	})

	t.Run("filter nonexistent service returns empty", func(t *testing.T) {
		result := FilterServices(azureYaml, []string{"nonexistent"})
		if len(result) != 0 {
			t.Errorf("got %d services, want 0", len(result))
		}
	})

	t.Run("nil azureYaml returns empty map", func(t *testing.T) {
		result := FilterServices(nil, nil)
		if result == nil {
			t.Error("expected non-nil empty map")
		}
		if len(result) != 0 {
			t.Errorf("got %d services, want 0", len(result))
		}
	})
}

func TestHasServices(t *testing.T) {
	tests := []struct {
		name      string
		azureYaml *AzureYaml
		want      bool
	}{
		{
			name:      "nil azureYaml",
			azureYaml: nil,
			want:      false,
		},
		{
			name:      "empty services",
			azureYaml: &AzureYaml{Services: map[string]Service{}},
			want:      false,
		},
		{
			name:      "has services",
			azureYaml: &AzureYaml{Services: map[string]Service{"api": {}}},
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasServices(tt.azureYaml)
			if got != tt.want {
				t.Errorf("HasServices() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetServiceProjectDir(t *testing.T) {
	tests := []struct {
		name       string
		service    Service
		workingDir string
		want       string
	}{
		{
			name:       "service with project path",
			service:    Service{Project: "/app/src/api"},
			workingDir: "/app",
			want:       "/app/src/api",
		},
		{
			name:       "service without project path falls back to workingDir",
			service:    Service{},
			workingDir: "/app",
			want:       "/app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetServiceProjectDir(tt.service, tt.workingDir)
			if got != tt.want {
				t.Errorf("GetServiceProjectDir() = %q, want %q", got, tt.want)
			}
		})
	}
}
