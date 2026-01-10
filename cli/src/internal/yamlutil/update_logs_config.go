// Package yamlutil provides utilities for manipulating YAML files while preserving
// formatting, comments, and structure. It uses text-based manipulation to guarantee
// zero data loss when updating YAML configuration files.
package yamlutil

import (
	"fmt"
	"os"
	"strings"

	"github.com/jongio/azd-core/security"
	"gopkg.in/yaml.v3"
)

// UpdateServiceLogsConfig updates the logs configuration for a specific service in azure.yaml.
// This preserves all comments, formatting, schema fields, and other content in the file.
// It only modifies the specific analytics configuration (tables or query) for the given service.
func UpdateServiceLogsConfig(azureYamlPath, serviceName string, tables []string, query string) error {
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
	if parseErr := yaml.Unmarshal(data, &azureYaml); parseErr != nil {
		return fmt.Errorf("failed to parse azure.yaml: %w", parseErr)
	}

	if azureYaml.Services == nil {
		return fmt.Errorf("no services section found in azure.yaml")
	}

	if _, exists := azureYaml.Services[serviceName]; !exists {
		return fmt.Errorf("service '%s' not found in azure.yaml", serviceName)
	}

	// Update the logs config using text-based manipulation
	updatedContent, err := updateLogsConfigInText(content, serviceName, tables, query)
	if err != nil {
		return err
	}

	// Write back to file
	if err := os.WriteFile(azureYamlPath, []byte(updatedContent), 0600); err != nil {
		return fmt.Errorf("failed to write azure.yaml: %w", err)
	}

	return nil
}

// updateLogsConfigInText adds or updates the logs.analytics configuration in the service definition.
func updateLogsConfigInText(content, serviceName string, tables []string, query string) (string, error) {
	lines := strings.Split(content, "\n")

	// Find the services section
	servicesInfo, err := findSection(lines, "services")
	if err != nil {
		return "", fmt.Errorf("services section not found")
	}

	// Find the specific service
	serviceInfo, err := FindServiceInSection(lines, servicesInfo, serviceName)
	if err != nil {
		return "", err
	}

	// Find or create logs section
	logs := findOrCreateLogsSection(&lines, serviceInfo)

	// Find or create analytics section within logs
	analytics := findOrCreateAnalyticsSection(&lines, logs)

	// Now update the analytics content - remove old tables/query and add new ones
	err = updateAnalyticsContent(&lines, analytics, tables, query)
	if err != nil {
		return "", err
	}

	return strings.Join(lines, "\n"), nil
}

// logsSection holds information about the logs section and whether it was created.
type logsSection struct {
	idx     int
	indent  string
	created bool
}

// analyticsSection holds information about the analytics section and whether it was created.
type analyticsSection struct {
	idx     int
	indent  string
	created bool
}

// findOrCreateLogsSection finds or creates the logs section within a service.
func findOrCreateLogsSection(lines *[]string, serviceInfo *serviceInfo) logsSection {
	serviceIndent := serviceInfo.indent
	logsIndent := serviceIndent

	lastPropertyIdx := serviceInfo.lineIdx

	// Look for existing logs section and find the last property line
	for i := serviceInfo.lineIdx + 1; i < len(*lines); i++ {
		line := (*lines)[i]
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		lineIndent := getIndentation(line)

		// If we've left the service properties, stop
		if len(lineIndent) < len(serviceIndent) {
			break
		}

		// If we're at the same level as service properties
		if len(lineIndent) == len(serviceIndent) {
			if strings.HasPrefix(trimmed, "logs:") {
				return logsSection{idx: i, indent: lineIndent, created: false}
			}
			// Track the last property at this level
			lastPropertyIdx = i
		} else if len(lineIndent) > len(serviceIndent) {
			// This is a nested property - keep track of it
			lastPropertyIdx = i
		}
	}

	// Logs section not found - insert it after the last property
	insertIdx := lastPropertyIdx + 1
	logsLine := logsIndent + "logs:"

	result := make([]string, 0, len(*lines)+1)
	result = append(result, (*lines)[:insertIdx]...)
	result = append(result, logsLine)
	result = append(result, (*lines)[insertIdx:]...)
	*lines = result

	return logsSection{idx: insertIdx, indent: logsIndent, created: true}
}

// findOrCreateAnalyticsSection finds or creates the analytics section within logs.
func findOrCreateAnalyticsSection(lines *[]string, logs logsSection) analyticsSection {
	analyticsIndent := logs.indent + "    "

	// Look for existing analytics section
	for i := logs.idx + 1; i < len(*lines); i++ {
		line := (*lines)[i]
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		lineIndent := getIndentation(line)

		// If we've left the logs section, stop
		if len(lineIndent) <= len(logs.indent) {
			break
		}

		// Check for analytics section
		if strings.HasPrefix(trimmed, "analytics:") {
			return analyticsSection{idx: i, indent: lineIndent, created: false}
		}
	}

	// Analytics section not found - insert it after logs:
	insertIdx := logs.idx + 1
	analyticsLine := logs.indent + "    analytics:"

	result := make([]string, 0, len(*lines)+1)
	result = append(result, (*lines)[:insertIdx]...)
	result = append(result, analyticsLine)
	result = append(result, (*lines)[insertIdx:]...)
	*lines = result

	return analyticsSection{idx: insertIdx, indent: analyticsIndent, created: true}
}

// updateAnalyticsContent removes old tables/query and adds new ones.
func updateAnalyticsContent(lines *[]string, analytics analyticsSection, tables []string, query string) error {
	// Find the range of content to replace (everything under analytics)
	startIdx := analytics.idx + 1
	endIdx := startIdx

	// Find where analytics content ends
	foundContent := false
	for i := startIdx; i < len(*lines); i++ {
		line := (*lines)[i]
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments initially to find content
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		lineIndent := getIndentation(line)

		// If we've left the analytics section, stop
		if len(lineIndent) <= len(analytics.indent) {
			endIdx = i
			break
		}

		// We're still in analytics section
		foundContent = true
		endIdx = i + 1
	}

	// If we didn't find any content and didn't break, we're at the end
	// In this case, endIdx should be startIdx (insert at current position)
	if !foundContent && endIdx != startIdx {
		endIdx = startIdx
	}

	// Build new analytics content
	var newContent []string
	contentIndent := analytics.indent + "    "

	if len(tables) > 0 {
		// Add tables array
		newContent = append(newContent, contentIndent+"tables:")
		for _, table := range tables {
			newContent = append(newContent, contentIndent+fmt.Sprintf("    - %s", table))
		}
	} else if query != "" {
		// Add custom query (multi-line string)
		newContent = append(newContent, contentIndent+"query: |")
		// Split query into lines and indent each
		queryLines := strings.Split(strings.TrimSpace(query), "\n")
		for _, qLine := range queryLines {
			newContent = append(newContent, contentIndent+"    "+qLine)
		}
	}

	// Replace the old content with new content
	result := make([]string, 0, len(*lines)-endIdx+startIdx+len(newContent))
	result = append(result, (*lines)[:startIdx]...)
	result = append(result, newContent...)
	result = append(result, (*lines)[endIdx:]...)
	*lines = result

	return nil
}
