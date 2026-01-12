// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"fmt"

	"github.com/jongio/azd-core/urlutil"
)

// ValidateServiceConfig validates the service configuration.
// Returns an error if the configuration is invalid.
func ValidateServiceConfig(serviceName string, svc *Service) error {
	// Validate local.customUrl if present (allows http and https)
	if svc.Local != nil && svc.Local.CustomURL != "" {
		if err := urlutil.Validate(svc.Local.CustomURL); err != nil {
			return fmt.Errorf("invalid local.customUrl for service '%s': %w", serviceName, err)
		}
	}

	// Validate azure.customUrl if present (enforces HTTPS, allows localhost)
	if svc.Azure != nil && svc.Azure.CustomURL != "" {
		if err := urlutil.ValidateHTTPSOnly(svc.Azure.CustomURL); err != nil {
			return fmt.Errorf("invalid azure.customUrl for service '%s': %w", serviceName, err)
		}
	}

	// Validate azure.customDomain if present (enforces HTTPS, allows localhost)
	if svc.Azure != nil && svc.Azure.CustomDomain != "" {
		if err := urlutil.ValidateHTTPSOnly(svc.Azure.CustomDomain); err != nil {
			return fmt.Errorf("invalid azure.customDomain for service '%s': %w", serviceName, err)
		}
	}

	return nil
}

