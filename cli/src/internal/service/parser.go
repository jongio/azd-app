// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jongio/azd-app/cli/src/internal/detector"
	internalsec "github.com/jongio/azd-app/cli/src/internal/security"
	"github.com/jongio/azd-core/security"

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
	if validateErr := security.ValidatePath(azureYamlPath); validateErr != nil {
		return nil, fmt.Errorf("invalid azure.yaml path: %w", validateErr)
	}

	// Read file
	// #nosec G304 -- azureYamlPath is produced by detector.FindAzureYaml (internal) and
	// validated by security.ValidatePath immediately above.
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read azure.yaml: %w", err)
	}

	// Parse YAML
	var azureYaml AzureYaml
	if err := yaml.Unmarshal(data, &azureYaml); err != nil {
		return nil, fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	// Resolve relative paths in service projects and enforce containment within
	// the project root (CWE-22).  A naive filepath.Clean(filepath.Join(...))
	// silently removes ".." components from the string, so security.ValidatePath
	// (which searches for ".." literals) would pass even when the resolved path
	// escapes the project root.  We use ValidatePathContainment instead, which
	// uses filepath.Rel on the fully-resolved absolute paths — the only correct
	// approach.
	azureYamlDir := filepath.Dir(azureYamlPath)
	for name, svc := range azureYaml.Services {
		if svc.Project != "" {
			if !filepath.IsAbs(svc.Project) {
				// Convert relative path to absolute before the containment check.
				svc.Project = filepath.Clean(filepath.Join(azureYamlDir, svc.Project))
				azureYaml.Services[name] = svc
			}

			// Validate that the (now-absolute) project path is still within
			// azureYamlDir, whether it was originally relative (e.g. "../../etc/passwd"
			// which filepath.Clean resolves outside the root) or absolute
			// (e.g. "/etc/passwd" supplied directly in azure.yaml).
			resolved, containErr := internalsec.ValidatePathContainment(svc.Project, azureYamlDir)
			if containErr != nil {
				return nil, fmt.Errorf(
					"service %q: project path is outside the project root: %w",
					name, containErr,
				)
			}
			svc.Project = resolved
			azureYaml.Services[name] = svc
		}

		// Validate service configuration (URLs, domains, etc.)
		if err := ValidateServiceConfig(name, &svc); err != nil {
			return nil, err
		}
		azureYaml.Services[name] = svc
	}

	return &azureYaml, nil
}

// FilterServices returns only the services specified in the filter.
// If filter is empty, returns all services.
// Returns empty map if azureYaml is nil.
func FilterServices(azureYaml *AzureYaml, filter []string) map[string]Service {
	if azureYaml == nil || azureYaml.Services == nil {
		return make(map[string]Service)
	}

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
