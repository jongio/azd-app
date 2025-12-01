package yamlutil

import (
	"fmt"
	"os"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/security"
	"gopkg.in/yaml.v3"
)

// UpdateServicePort adds or updates the ports field for a specific service in azure.yaml.
// This preserves all comments, formatting, and other content in the file.
// The port is added as a single-element ports array: ports: ["8080"]
func UpdateServicePort(azureYamlPath, serviceName string, port int) error {
	// Validate path
	if err := security.ValidatePath(azureYamlPath); err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Read existing azure.yaml
	// #nosec G304 -- Path validated by security.ValidatePath
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		return fmt.Errorf("failed to read azure.yaml: %w", err)
	}

	content := string(data)

	// Parse YAML to verify service exists
	var azureYaml struct {
		Services map[string]any `yaml:"services"`
	}
	if err := yaml.Unmarshal(data, &azureYaml); err != nil {
		return fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	if azureYaml.Services == nil {
		return fmt.Errorf("no services section found in azure.yaml")
	}

	if _, exists := azureYaml.Services[serviceName]; !exists {
		return fmt.Errorf("service '%s' not found in azure.yaml", serviceName)
	}

	// Update the ports field using text-based manipulation
	updatedContent, err := updateServicePortsInText(content, serviceName, port)
	if err != nil {
		return err
	}

	// Write back to file
	if err := os.WriteFile(azureYamlPath, []byte(updatedContent), 0600); err != nil {
		return fmt.Errorf("failed to write azure.yaml: %w", err)
	}

	return nil
}

// updateServicePortsInText adds or updates the ports field in the service definition.
func updateServicePortsInText(content, serviceName string, port int) (string, error) {
	lines := strings.Split(content, "\n")

	// Find the services section
	servicesInfo, err := findSection(lines, "services")
	if err != nil {
		return "", fmt.Errorf("services section not found")
	}

	// Find the specific service
	serviceInfo, err := findServiceInSection(lines, servicesInfo, serviceName)
	if err != nil {
		return "", err
	}

	// Check if ports field already exists
	portsLineIdx, portsIndent := findPortsLine(lines, serviceInfo)

	// Create ports array with single port
	portsLine := fmt.Sprintf("%sports:", portsIndent)
	portValueLine := fmt.Sprintf("%s  - \"%d\"", portsIndent, port)

	if portsLineIdx >= 0 {
		// Check if ports is inline format: ports: ["3000"] or ports: ["3000", "8080"]
		portsLine := lines[portsLineIdx]
		if strings.Contains(portsLine, "[") {
			// Inline array format - replace entire line
			lineIndent := getIndentation(portsLine)
			lines[portsLineIdx] = fmt.Sprintf("%sports: [\"%d\"]", lineIndent, port)
			return strings.Join(lines, "\n"), nil
		}

		// Update existing multi-line ports array - replace first port value
		// Find the first array item after ports:
		for i := portsLineIdx + 1; i < len(lines); i++ {
			line := lines[i]
			trimmed := strings.TrimSpace(line)
			lineIndent := getIndentation(line)

			// Skip empty lines and comments
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}

			// If we've left the ports array, break
			if len(lineIndent) <= len(portsIndent) {
				break
			}

			// If this is an array item, update it
			if strings.HasPrefix(trimmed, "-") {
				lines[i] = portValueLine
				return strings.Join(lines, "\n"), nil
			}
		}
		// Ports array exists but has no items - add one
		result := make([]string, 0, len(lines)+1)
		result = append(result, lines[:portsLineIdx+1]...)
		result = append(result, portValueLine)
		result = append(result, lines[portsLineIdx+1:]...)
		lines = result
	} else {
		// Insert new ports field after service name
		insertIdx := serviceInfo.lineIdx + 1

		// Insert the ports lines
		result := make([]string, 0, len(lines)+2)
		result = append(result, lines[:insertIdx]...)
		result = append(result, portsLine)
		result = append(result, portValueLine)
		result = append(result, lines[insertIdx:]...)
		lines = result
	}

	return strings.Join(lines, "\n"), nil
}

// serviceInfo holds information about a service location in YAML.
type serviceInfo struct {
	lineIdx int    // Line index where the service name appears
	indent  string // Indentation of the service properties
}

// findServiceInSection finds a specific service within the services section.
func findServiceInSection(lines []string, servicesInfo *sectionInfo, serviceName string) (*serviceInfo, error) {
	searchKey := serviceName + ":"
	serviceIndent := servicesInfo.indent + "  "

	for i := servicesInfo.lineIdx + 1; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check indentation - if less than or equal to services indent, we've left the services section
		lineIndent := getIndentation(line)
		if len(lineIndent) <= len(servicesInfo.indent) {
			break
		}

		// Check if this is our service
		if len(lineIndent) == len(serviceIndent) && (trimmed == searchKey || strings.HasPrefix(trimmed, searchKey+" ")) {
			// Found the service, now find its property indent
			propertyIndent := serviceIndent + "  "
			return &serviceInfo{
				lineIdx: i,
				indent:  propertyIndent,
			}, nil
		}
	}

	return nil, fmt.Errorf("service '%s' not found in services section", serviceName)
}

// findPortsLine looks for an existing ports field in the service definition.
func findPortsLine(lines []string, serviceInfo *serviceInfo) (int, string) {
	for i := serviceInfo.lineIdx + 1; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		lineIndent := getIndentation(line)

		// If we've left the service properties (less indented), stop
		if len(lineIndent) < len(serviceInfo.indent) {
			break
		}

		// If we're at the same level as service properties
		if len(lineIndent) == len(serviceInfo.indent) {
			// Check if this is the ports line
			if strings.HasPrefix(trimmed, "ports:") {
				return i, lineIndent
			}
		}
	}

	// Ports not found, return indent for insertion
	return -1, serviceInfo.indent
}
