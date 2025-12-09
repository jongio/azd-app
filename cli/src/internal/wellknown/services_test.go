package wellknown

import (
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

func TestRegistryContainsExpectedServices(t *testing.T) {
	expectedServices := []string{"azurite", "cosmos", "redis", "postgres"}

	for _, name := range expectedServices {
		if _, ok := Registry[name]; !ok {
			t.Errorf("Registry missing expected service: %s", name)
		}
	}
}

func TestGet(t *testing.T) {
	tests := []struct {
		name     string
		want     bool
		wantName string
	}{
		{"azurite", true, "azurite"},
		{"cosmos", true, "cosmos"},
		{"redis", true, "redis"},
		{"postgres", true, "postgres"},
		{"nonexistent", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Get(tt.name)
			if tt.want && got == nil {
				t.Errorf("Get(%q) = nil, want service", tt.name)
			}
			if !tt.want && got != nil {
				t.Errorf("Get(%q) = %v, want nil", tt.name, got)
			}
			if got != nil && got.Name != tt.wantName {
				t.Errorf("Get(%q).Name = %q, want %q", tt.name, got.Name, tt.wantName)
			}
		})
	}
}

func TestNames(t *testing.T) {
	names := Names()
	if len(names) != len(Registry) {
		t.Errorf("Names() returned %d names, want %d", len(names), len(Registry))
	}

	// Check all names are in registry
	for _, name := range names {
		if _, ok := Registry[name]; !ok {
			t.Errorf("Names() returned %q which is not in Registry", name)
		}
	}
}

func TestCategories(t *testing.T) {
	categories := Categories()
	if len(categories) == 0 {
		t.Error("Categories() returned empty slice")
	}

	// Should have at least storage, database, cache
	hasStorage := false
	hasDatabase := false
	hasCache := false
	for _, cat := range categories {
		switch cat {
		case "storage":
			hasStorage = true
		case "database":
			hasDatabase = true
		case "cache":
			hasCache = true
		}
	}

	if !hasStorage {
		t.Error("Categories() missing 'storage' category")
	}
	if !hasDatabase {
		t.Error("Categories() missing 'database' category")
	}
	if !hasCache {
		t.Error("Categories() missing 'cache' category")
	}
}

func TestByCategory(t *testing.T) {
	storageServices := ByCategory("storage")
	if len(storageServices) == 0 {
		t.Error("ByCategory('storage') returned empty slice")
	}

	for _, svc := range storageServices {
		if svc.Category != "storage" {
			t.Errorf("ByCategory('storage') returned service %q with category %q", svc.Name, svc.Category)
		}
	}
}

func TestToServiceConfig(t *testing.T) {
	def := Get("azurite")
	if def == nil {
		t.Fatal("azurite service not found")
	}

	cfg := def.ToServiceConfig()

	if cfg.Image != def.Image {
		t.Errorf("ToServiceConfig().Image = %q, want %q", cfg.Image, def.Image)
	}

	if len(cfg.Ports) != len(def.Ports) {
		t.Errorf("ToServiceConfig().Ports has %d ports, want %d", len(cfg.Ports), len(def.Ports))
	}

	if cfg.Type != service.ServiceTypeContainer {
		t.Errorf("ToServiceConfig().Type = %q, want %q", cfg.Type, service.ServiceTypeContainer)
	}
}

func TestAzuriteHasConnectionStrings(t *testing.T) {
	def := Get("azurite")
	if def == nil {
		t.Fatal("azurite service not found")
	}

	// Azurite should have blob, queue, table, and default connection strings
	expected := []string{"blob", "queue", "table", "default"}
	for _, key := range expected {
		if _, ok := def.ConnectionStrings[key]; !ok {
			t.Errorf("azurite missing connection string: %s", key)
		}
	}
}

func TestAllServicesHaveRequiredFields(t *testing.T) {
	for name, def := range Registry {
		t.Run(name, func(t *testing.T) {
			if def.Name == "" {
				t.Error("Name is empty")
			}
			if def.DisplayName == "" {
				t.Error("DisplayName is empty")
			}
			if def.Description == "" {
				t.Error("Description is empty")
			}
			if def.Image == "" {
				t.Error("Image is empty")
			}
			if len(def.Ports) == 0 {
				t.Error("Ports is empty")
			}
			if def.Category == "" {
				t.Error("Category is empty")
			}
			if len(def.ConnectionStrings) == 0 {
				t.Error("ConnectionStrings is empty")
			}
			if _, ok := def.ConnectionStrings["default"]; !ok {
				t.Error("ConnectionStrings missing 'default' key")
			}
		})
	}
}
