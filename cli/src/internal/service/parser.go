package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/security"

	"gopkg.in/yaml.v3"
)

// ParseAzureYaml reads and parses the azure.yaml file.
func ParseAzureYaml(workingDir string) (*AzureYaml, error) {
	// Find azure.yaml using existing detector logic
	azureYamlPath, err := detector.FindAzureYaml(workingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find azure.yaml: %w", err)
	}
	if azureYamlPath == "" {
		return nil, fmt.Errorf("azure.yaml not found in %s or parent directories", workingDir)
	}

	// Validate path
	if err := security.ValidatePath(azureYamlPath); err != nil {
		return nil, fmt.Errorf("invalid azure.yaml path: %w", err)
	}

	// Read file
	// #nosec G304 -- Path validated by security.ValidatePath
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read azure.yaml: %w", err)
	}

	// Parse YAML
	var azureYaml AzureYaml
	if err := yaml.Unmarshal(data, &azureYaml); err != nil {
		return nil, fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	// Resolve relative paths in service projects
	azureYamlDir := filepath.Dir(azureYamlPath)
	for name, svc := range azureYaml.Services {
		if svc.Project != "" {
			// Convert relative path to absolute
			if !filepath.IsAbs(svc.Project) {
				// Clean the path and join with azure.yaml directory
				absPath := filepath.Clean(filepath.Join(azureYamlDir, svc.Project))
				svc.Project = absPath
				azureYaml.Services[name] = svc
			}
		}
	}

	return &azureYaml, nil
}

// FilterServices returns only the services specified in the filter.
// If filter is empty, returns all services.
func FilterServices(azureYaml *AzureYaml, filter []string) map[string]Service {
	if len(filter) == 0 {
		return azureYaml.Services
	}

	filtered := make(map[string]Service)
	for _, name := range filter {
		if svc, exists := azureYaml.Services[name]; exists {
			filtered[name] = svc
		}
	}

	return filtered
}

// HasServices checks if azure.yaml has any services defined.
func HasServices(azureYaml *AzureYaml) bool {
	return azureYaml != nil && len(azureYaml.Services) > 0
}

// GetServiceProjectDir returns the project directory for a service.
// If the service has a project path, returns that. Otherwise, returns the working directory.
func GetServiceProjectDir(service Service, workingDir string) string {
	if service.Project != "" {
		return service.Project
	}
	return workingDir
}
