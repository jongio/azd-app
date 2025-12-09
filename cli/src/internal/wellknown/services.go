// Package wellknown provides definitions for commonly-used Azure emulator services.
//
// SECURITY NOTE: The credentials defined in this package (e.g., postgres/postgres)
// are intended ONLY for local development use. These are well-known default credentials
// for development emulators and should never be used in production environments.
package wellknown

import (
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// ServiceDefinition represents a well-known service that can be added via azd app add.
type ServiceDefinition struct {
	// Name is the canonical service name (e.g., "azurite", "cosmos")
	Name string

	// DisplayName is the human-readable name for display
	DisplayName string

	// Description provides a brief description of the service
	Description string

	// Image is the Docker image to use
	Image string

	// Ports defines the default port mappings (host:container or just container)
	Ports []string

	// Environment contains default environment variables
	Environment map[string]string

	// Healthcheck defines the default health check configuration
	Healthcheck *service.HealthcheckConfig

	// ConnectionStrings provides templates for common connection strings
	// Keys are usage names (e.g., "blob", "queue", "table")
	ConnectionStrings map[string]string

	// Category groups related services (e.g., "storage", "database", "cache")
	Category string
}

// Registry contains all well-known service definitions.
var Registry = map[string]ServiceDefinition{
	"azurite": {
		Name:        "azurite",
		DisplayName: "Azurite (Azure Storage Emulator)",
		Description: "Local Azure Storage emulator for Blob, Queue, and Table services",
		Image:       "mcr.microsoft.com/azure-storage/azurite:latest",
		Ports:       []string{"10000:10000", "10001:10001", "10002:10002"},
		Environment: map[string]string{},
		Healthcheck: &service.HealthcheckConfig{
			Test:     []string{"CMD", "curl", "-f", "http://127.0.0.1:10000/", "--connect-timeout", "5"},
			Interval: "10s",
			Timeout:  "5s",
			Retries:  3,
		},
		ConnectionStrings: map[string]string{
			"blob":    "DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10000/devstoreaccount1",
			"queue":   "DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;QueueEndpoint=http://127.0.0.1:10001/devstoreaccount1",
			"table":   "DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;TableEndpoint=http://127.0.0.1:10002/devstoreaccount1",
			"default": "UseDevelopmentStorage=true",
		},
		Category: "storage",
	},
	"cosmos": {
		Name:        "cosmos",
		DisplayName: "Azure Cosmos DB Emulator",
		Description: "Local Azure Cosmos DB emulator for NoSQL API",
		Image:       "mcr.microsoft.com/cosmosdb/linux/azure-cosmos-emulator:latest",
		Ports:       []string{"8081:8081", "10250:10250", "10251:10251", "10252:10252", "10253:10253", "10254:10254"},
		Environment: map[string]string{
			"AZURE_COSMOS_EMULATOR_PARTITION_COUNT":         "10",
			"AZURE_COSMOS_EMULATOR_ENABLE_DATA_PERSISTENCE": "true",
		},
		Healthcheck: &service.HealthcheckConfig{
			Test:     []string{"CMD", "curl", "-fk", "https://localhost:8081/_explorer/emulator.pem"},
			Interval: "30s",
			Timeout:  "10s",
			Retries:  5,
		},
		ConnectionStrings: map[string]string{
			"default": "AccountEndpoint=https://localhost:8081/;AccountKey=C2y6yDjf5/R+ob0N8A7Cgv30VRDJIWEHLM+4QDU5DE2nQ9nDuVTqobD4b8mGGyPMbIZnqyMsEcaGQy67XIw/Jw==",
		},
		Category: "database",
	},
	"redis": {
		Name:        "redis",
		DisplayName: "Redis",
		Description: "In-memory data store for caching and messaging",
		Image:       "redis:7-alpine",
		Ports:       []string{"6379:6379"},
		Environment: map[string]string{},
		Healthcheck: &service.HealthcheckConfig{
			Test:     []string{"CMD", "redis-cli", "ping"},
			Interval: "10s",
			Timeout:  "5s",
			Retries:  3,
		},
		ConnectionStrings: map[string]string{
			"default": "localhost:6379",
		},
		Category: "cache",
	},
	"postgres": {
		Name:        "postgres",
		DisplayName: "PostgreSQL",
		Description: "Open source relational database",
		Image:       "postgres:16-alpine",
		Ports:       []string{"5432:5432"},
		Environment: map[string]string{
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "postgres",
			"POSTGRES_DB":       "app",
		},
		Healthcheck: &service.HealthcheckConfig{
			Test:     []string{"CMD-SHELL", "pg_isready -U postgres"},
			Interval: "10s",
			Timeout:  "5s",
			Retries:  3,
		},
		ConnectionStrings: map[string]string{
			"default": "postgresql://postgres:postgres@localhost:5432/app?sslmode=disable",
		},
		Category: "database",
	},
}

// Categories returns all unique categories from the registry.
func Categories() []string {
	seen := make(map[string]bool)
	var categories []string
	for _, def := range Registry {
		if !seen[def.Category] {
			seen[def.Category] = true
			categories = append(categories, def.Category)
		}
	}
	return categories
}

// ByCategory returns all services in a given category.
func ByCategory(category string) []ServiceDefinition {
	var services []ServiceDefinition
	for _, def := range Registry {
		if def.Category == category {
			services = append(services, def)
		}
	}
	return services
}

// Get returns a service definition by name, or nil if not found.
func Get(name string) *ServiceDefinition {
	if def, ok := Registry[name]; ok {
		return &def
	}
	return nil
}

// Names returns all service names in the registry.
func Names() []string {
	names := make([]string, 0, len(Registry))
	for name := range Registry {
		names = append(names, name)
	}
	return names
}

// ToServiceConfig converts a ServiceDefinition to a service.Service for azure.yaml.
func (d *ServiceDefinition) ToServiceConfig() service.Service {
	env := make(service.Environment)
	for k, v := range d.Environment {
		env[k] = v
	}

	return service.Service{
		Image:       d.Image,
		Ports:       d.Ports,
		Environment: env,
		Healthcheck: d.Healthcheck,
		Type:        service.ServiceTypeContainer,
	}
}
