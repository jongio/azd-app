// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"fmt"
	"net/url"

	"github.com/jongio/azd-core/urlutil"
)

const maxURLLength = 2048

// ValidateServiceConfig validates the service configuration.
// Returns an error if the configuration is invalid.
func ValidateServiceConfig(serviceName string, svc *Service) error {
	if svc == nil {
		return nil
	}

	// Validate deprecated root-level url field
	if svc.URL != "" {
		if err := validateURL(svc.URL, "url", serviceName); err != nil {
			return err
		}
	}

	// Validate local.customUrl if present
	if svc.Local != nil && svc.Local.CustomURL != "" {
		if err := validateURL(svc.Local.CustomURL, "local.customUrl", serviceName); err != nil {
			return err
		}
	}

	// Validate azure.customUrl if present
	if svc.Azure != nil && svc.Azure.CustomURL != "" {
		if err := validateURL(svc.Azure.CustomURL, "azure.customUrl", serviceName); err != nil {
			return err
		}
	}

	// Validate azure.customDomain if present (domain only, no protocol)
	if svc.Azure != nil && svc.Azure.CustomDomain != "" {
		if err := urlutil.ValidateDomain(svc.Azure.CustomDomain); err != nil {
			return fmt.Errorf("invalid azure.customDomain for service '%s': %w", serviceName, err)
		}
	}

	return nil
}

// validateURL performs comprehensive URL validation including security checks.
func validateURL(urlStr, fieldName, serviceName string) error {
	// Basic validation
	if err := urlutil.Validate(urlStr); err != nil {
		return fmt.Errorf("invalid %s for service '%s': %w", fieldName, serviceName, err)
	}

	// Length check
	if len(urlStr) > maxURLLength {
		return fmt.Errorf("%s for service '%s' exceeds maximum length of %d characters", fieldName, serviceName, maxURLLength)
	}

	// Parse URL for additional checks
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid %s for service '%s': %w", fieldName, serviceName, err)
	}

	// Protocol enforcement - only http and https allowed
	if parsedURL.Scheme != ServiceTypeHTTP && parsedURL.Scheme != "https" {
		return fmt.Errorf("%s for service '%s' must use http:// or https://, got %s://", fieldName, serviceName, parsedURL.Scheme)
	}

	return nil
}
