package commands

import (
	"fmt"
	"regexp"
	"strings"
)

// extractVersion extracts version from command cliout.
func extractVersion(config ToolConfig, output string) string {
	// Handle azd special case first (multi-line output)
	if strings.Contains(output, "azd version") {
		return extractAzdVersion(output)
	}

	// Handle Podman aliased to Docker (multi-line output with "Podman Engine")
	if strings.Contains(output, "Podman Engine") {
		return extractPodmanVersion(output)
	}

	// Extract specific field BEFORE stripping prefix (field extraction first)
	if config.VersionField > 0 {
		parts := strings.Fields(output)
		if len(parts) > config.VersionField {
			output = parts[config.VersionField]
		}
	}

	// Strip prefix if configured (after field extraction)
	if config.VersionPrefix != "" {
		output = strings.TrimPrefix(output, config.VersionPrefix)
	}

	return extractFirstVersion(output)
}

// extractAzdVersion extracts version from azd multi-line cliout.
func extractAzdVersion(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "azd version") {
			parts := strings.Fields(line)
			for _, part := range parts {
				if v := extractFirstVersion(part); v != "" && v != "version" {
					return v
				}
			}
		}
	}
	return ""
}

// extractPodmanVersion extracts version from Podman multi-line cliout.
// Podman output format when aliased to docker:
//
// Client:       Podman Engine
// Version:      5.7.0
// API Version:  5.7.0
// ...
func extractPodmanVersion(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Version:") {
			// Extract version from "Version:      5.7.0"
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return extractFirstVersion(parts[1])
			}
		}
	}
	return ""
}

// Compiled regex patterns for version extraction (package-level for performance)
var (
	semanticVersionRegex = regexp.MustCompile(`(\d+\.\d+\.\d+)`)
	simpleVersionRegex   = regexp.MustCompile(`(\d+\.\d+)`)
)

// extractFirstVersion finds the first semantic version in a string.
func extractFirstVersion(s string) string {
	// Match semantic version pattern (e.g., 1.2.3, 20.0.0, etc.)
	matches := semanticVersionRegex.FindStringSubmatch(s)
	if len(matches) > 1 {
		return matches[1]
	}

	// Try simpler pattern (e.g., 1.2)
	matches = simpleVersionRegex.FindStringSubmatch(s)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// compareVersions compares installed version against required version.
// Returns true if installed >= required.
// Missing version parts are treated as 0 (e.g., "1.2" is equivalent to "1.2.0").
func compareVersions(installed, required string) bool {
	installedParts := parseVersion(installed)
	requiredParts := parseVersion(required)

	// Get the maximum length to compare all parts
	maxLen := len(requiredParts)
	if len(installedParts) > maxLen {
		maxLen = len(installedParts)
	}

	// Compare each part left to right, treating missing parts as 0
	for i := 0; i < maxLen; i++ {
		installedPart := 0
		if i < len(installedParts) {
			installedPart = installedParts[i]
		}

		requiredPart := 0
		if i < len(requiredParts) {
			requiredPart = requiredParts[i]
		}

		if installedPart > requiredPart {
			return true
		}
		if installedPart < requiredPart {
			return false
		}
		// Equal, continue to next part
	}

	return true // All parts equal
}

// parseVersion parses a version string into numeric parts.
func parseVersion(version string) []int {
	parts := strings.Split(version, ".")
	result := make([]int, 0, len(parts))

	for _, part := range parts {
		var num int
		_, _ = fmt.Sscanf(part, "%d", &num)
		result = append(result, num)
	}

	return result
}
