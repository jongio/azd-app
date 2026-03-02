// Package dashboard provides API endpoints for the local dashboard.
package dashboard

import (
	"strings"
)

// extractServiceNamesFromEnv parses SERVICE_*_NAME environment variables and returns service names.
// Returns azure.yaml service names (e.g., "api", "web", "worker") not Azure resource names.
func extractServiceNamesFromEnv() []string {
	serviceMap := make(map[string]bool)

	for _, line := range getEnvironment() {
		// Look for SERVICE_*_NAME pattern (e.g., SERVICE_API_NAME, SERVICE_CONTAINERAPP_API_NAME)
		if strings.HasPrefix(line, "SERVICE_") && strings.Contains(line, "_NAME=") && !strings.Contains(line, "_IMAGE_NAME=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				// Extract service name from key: SERVICE_CONTAINERAPP_API_NAME -> containerapp-api
				// or SERVICE_API_NAME -> api
				key := parts[0]
				key = strings.TrimPrefix(key, "SERVICE_")
				key = strings.TrimSuffix(key, "_NAME")

				// Convert to lowercase with hyphens (azure.yaml format)
				serviceName := strings.ToLower(strings.ReplaceAll(key, "_", "-"))

				if serviceName != "" {
					serviceMap[serviceName] = true
				}
			}
		}
	}

	// Convert map to sorted slice for consistent output
	services := make([]string, 0, len(serviceMap))
	for name := range serviceMap {
		services = append(services, name)
	}

	// Sort alphabetically for consistent ordering
	for i := 0; i < len(services); i++ {
		for j := i + 1; j < len(services); j++ {
			if services[i] > services[j] {
				services[i], services[j] = services[j], services[i]
			}
		}
	}

	return services
}

// getAllEnvironmentVars returns all environment variables as a map.
func getAllEnvironmentVars() map[string]string {
	result := make(map[string]string)
	for _, env := range getEnvironment() {
		if idx := strings.Index(env, "="); idx > 0 {
			key := env[:idx]
			value := env[idx+1:]
			result[key] = value
		}
	}
	return result
}

// substituteQueryPlaceholders replaces placeholders in a query for display.
// NOTE: This is for display only, not for query execution. The actual query
// execution path in loganalytics.go uses sanitizeKQLString for injection prevention.
func substituteQueryPlaceholders(query, serviceName, timespan string) string {
	query = strings.ReplaceAll(query, "{serviceName}", serviceName)
	query = strings.ReplaceAll(query, "{timespan}", timespan)
	return query
}

// truncateMiddle truncates a string in the middle, keeping prefix and suffix.
func truncateMiddle(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 3 {
		return s[:maxLen]
	}

	prefixLen := (maxLen - 3) / 2
	suffixLen := maxLen - 3 - prefixLen
	return s[:prefixLen] + "..." + s[len(s)-suffixLen:]
}
