// Package testing provides test execution and coverage aggregation for multi-language projects.
package testing

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/security"
	"gopkg.in/yaml.v3"
)

// GenerateTestConfigYAML generates a YAML snippet for discovered test configurations.
// Only includes services that were auto-detected (had no config in azure.yaml).
// Returns an empty string if no auto-detected services are found.
func GenerateTestConfigYAML(validations []ServiceValidation, services []ServiceInfo) string {
	// Find services that were auto-detected (no Config in azure.yaml)
	autoDetected := make(map[string]ServiceValidation)
	for i, v := range validations {
		if v.CanTest && v.Framework != "" {
			// Check if this service had no config defined
			if i < len(services) && services[i].Config == nil {
				autoDetected[v.Name] = v
			}
		}
	}

	if len(autoDetected) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("services:\n")

	// Sort service names for consistent output
	names := make([]string, 0, len(autoDetected))
	for name := range autoDetected {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		v := autoDetected[name]
		builder.WriteString(fmt.Sprintf("  %s:\n", name))
		builder.WriteString("    test:\n")
		builder.WriteString(fmt.Sprintf("      framework: %s\n", v.Framework))
	}

	return builder.String()
}

// GetAutoDetectedServices returns services that were auto-detected (had no config in azure.yaml).
func GetAutoDetectedServices(validations []ServiceValidation, services []ServiceInfo) []ServiceValidation {
	result := make([]ServiceValidation, 0)

	for i, v := range validations {
		if v.CanTest && v.Framework != "" {
			// Check if this service had no config defined
			if i < len(services) && services[i].Config == nil {
				result = append(result, v)
			}
		}
	}

	return result
}

// SaveTestConfigToAzureYaml merges discovered test config into azure.yaml.
// Only adds test config for services that don't already have it.
// Preserves existing content and formatting as much as possible.
func SaveTestConfigToAzureYaml(azureYamlPath string, validations []ServiceValidation, services []ServiceInfo) error {
	// Validate path
	if err := security.ValidatePath(azureYamlPath); err != nil {
		return fmt.Errorf("invalid azure.yaml path: %w", err)
	}

	// Read existing azure.yaml
	// #nosec G304 -- Path validated by security.ValidatePath above
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		return fmt.Errorf("failed to read azure.yaml: %w", err)
	}

	// Parse YAML into a generic structure that preserves order
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	// Find auto-detected services
	autoDetected := make(map[string]string) // name -> framework
	for i, v := range validations {
		if v.CanTest && v.Framework != "" {
			if i < len(services) && services[i].Config == nil {
				autoDetected[v.Name] = v.Framework
			}
		}
	}

	if len(autoDetected) == 0 {
		return nil // Nothing to save
	}

	// Find the services node and add test config
	if err := addTestConfigToServices(&root, autoDetected); err != nil {
		return fmt.Errorf("failed to update azure.yaml: %w", err)
	}

	// Marshal back to YAML
	output, err := yaml.Marshal(&root)
	if err != nil {
		return fmt.Errorf("failed to serialize azure.yaml: %w", err)
	}

	// Write back to file
	// #nosec G306 -- FilePermissions constant is 0o644, standard secure permissions
	if err := os.WriteFile(azureYamlPath, output, FilePermissions); err != nil {
		return fmt.Errorf("failed to write azure.yaml: %w", err)
	}

	return nil
}

// addTestConfigToServices adds test configuration to services in the YAML node.
func addTestConfigToServices(root *yaml.Node, autoDetected map[string]string) error {
	// The root should be a document node
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return fmt.Errorf("invalid YAML document structure")
	}

	// Get the root mapping
	rootMap := root.Content[0]
	if rootMap.Kind != yaml.MappingNode {
		return fmt.Errorf("expected mapping at root level")
	}

	// Find the services key
	var servicesNode *yaml.Node
	for i := 0; i < len(rootMap.Content)-1; i += 2 {
		keyNode := rootMap.Content[i]
		if keyNode.Value == "services" {
			servicesNode = rootMap.Content[i+1]
			break
		}
	}

	if servicesNode == nil || servicesNode.Kind != yaml.MappingNode {
		return fmt.Errorf("services section not found or invalid")
	}

	// Iterate through services and add test config where needed
	for i := 0; i < len(servicesNode.Content)-1; i += 2 {
		serviceNameNode := servicesNode.Content[i]
		serviceNode := servicesNode.Content[i+1]

		if serviceNode.Kind != yaml.MappingNode {
			continue
		}

		serviceName := serviceNameNode.Value
		framework, needsConfig := autoDetected[serviceName]
		if !needsConfig {
			continue
		}

		// Check if test config already exists
		hasTestConfig := false
		for j := 0; j < len(serviceNode.Content)-1; j += 2 {
			if serviceNode.Content[j].Value == "test" {
				hasTestConfig = true
				break
			}
		}

		if hasTestConfig {
			continue
		}

		// Add test config
		testKeyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: "test",
		}

		testValueNode := &yaml.Node{
			Kind: yaml.MappingNode,
			Tag:  "!!map",
			Content: []*yaml.Node{
				{
					Kind:  yaml.ScalarNode,
					Tag:   "!!str",
					Value: "framework",
				},
				{
					Kind:  yaml.ScalarNode,
					Tag:   "!!str",
					Value: framework,
				},
			},
		}

		serviceNode.Content = append(serviceNode.Content, testKeyNode, testValueNode)
	}

	return nil
}
