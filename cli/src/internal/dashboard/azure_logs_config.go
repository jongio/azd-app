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
func substituteQueryPlaceholders(query, serviceName, timespan string) string {
	query = replaceAll(query, "{serviceName}", serviceName)
	query = replaceAll(query, "{timespan}", timespan)
	return query
}

// replaceAll is a simple string replacement (avoids importing strings package).
func replaceAll(s, old, new string) string {
	result := ""
	for {
		i := indexOf(s, old)
		if i < 0 {
			return result + s
		}
		result += s[:i] + new
		s = s[i+len(old):]
	}
}

// indexOf finds the first occurrence of substr in s.
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
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
