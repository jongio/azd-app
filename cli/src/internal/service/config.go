// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"fmt"
	"strings"
)

// ValidateServiceConfig validates the service configuration.
// Returns an error if the configuration is invalid.
func ValidateServiceConfig(serviceName string, url string) error {
	// Validate url if present
	if url != "" {
		if err := ValidateURL(url); err != nil {
			return fmt.Errorf("invalid url for service '%s': %w", serviceName, err)
		}
	}

	return nil
}

// ValidateURL validates that a custom URL is a valid HTTP/HTTPS URL.
// Returns an error if the URL is malformed or doesn't use http:// or https://.
func ValidateURL(url string) error {
	if url == "" {
		return fmt.Errorf("url cannot be empty")
	}

	// Normalize the URL for validation
	url = strings.TrimSpace(url)

	// Check for http:// or https:// prefix
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("url must start with http:// or https://, got: %s", url)
	}

	// Basic validation: ensure there's something after the protocol
	if url == "http://" {
		return fmt.Errorf("url missing domain after http://")
	}
	if url == "https://" {
		return fmt.Errorf("url missing domain after https://")
	}

	// Check for minimum URL length (protocol + at least one character for domain)
	// http://a is the shortest valid URL (8 chars)
	if len(url) < 8 {
		return fmt.Errorf("url is too short to be valid: %s", url)
	}

	return nil
}
