// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"fmt"

	"github.com/jongio/azd-core/urlutil"
)

// ValidateServiceConfig validates the service configuration.
// Returns an error if the configuration is invalid.
func ValidateServiceConfig(serviceName string, url string) error {
	// Validate url if present
	if url != "" {
		if err := urlutil.Validate(url); err != nil {
			return fmt.Errorf("invalid url for service '%s': %w", serviceName, err)
		}
	}

	return nil
}
