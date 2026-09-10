package commands

import (
	"fmt"
	"strings"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
)

// Error codes reported to the azd host through azdext.LocalError.
//
// The host renders these as `ext.<category>.<code>` in telemetry, so the strings
// are effectively a public contract. They are lowercase snake_case, stable, and
// alphabetical.
//
// These cover the failure classes a user can actually act on. Infrastructure
// failures that are already wrapped with %w stay as plain errors, because a
// stable code for "the filesystem said no" carries no useful signal.
const (
	// ErrCodeCheckFailed is reported when a command whose whole job is to
	// evaluate something finds a problem. Examples are `azd app doctor` with a
	// failing check, `azd app health` with an unhealthy service, and
	// `azd app deps --check` with missing dependencies. The command ran
	// correctly, so this is a result rather than a malfunction.
	ErrCodeCheckFailed = "check_failed"

	// ErrCodeInvalidFlagUsage is reported when flags are individually valid but
	// cannot be combined, or when a flag value is outside its allowed set.
	ErrCodeInvalidFlagUsage = "invalid_flag_usage"

	// ErrCodeInvalidProjectConfig is reported when azure.yaml is found but its
	// contents cannot be used, for example a malformed document, a missing
	// services section, or a project path that points outside the project root.
	ErrCodeInvalidProjectConfig = "invalid_project_config"

	// ErrCodeProjectNotFound is reported when no azure.yaml exists in the
	// current directory or any parent. Nearly every command needs one.
	ErrCodeProjectNotFound = "project_not_found"

	// ErrCodeServiceNotFound is reported when the user names a service that is
	// not defined in azure.yaml.
	ErrCodeServiceNotFound = "service_not_found"
)

// newCheckFailedError reports that a check command found a problem. The message
// is the finding itself, so callers pass their own wording.
func newCheckFailedError(message, suggestion string) error {
	return &azdext.LocalError{
		Message:    message,
		Code:       ErrCodeCheckFailed,
		Category:   azdext.LocalErrorCategoryUser,
		Suggestion: suggestion,
	}
}

// newGraphNodeNotFoundError reports a --focus value that names no node in the
// dependency graph. Graph nodes cover services and resources alike, so this
// deliberately does not reuse newServiceNotFoundError, which would tell a user
// who typed a resource name that their resource is not a service.
func newGraphNodeNotFoundError(name string, available []string) error {
	suggestion := "The graph has no nodes. Define services or resources in azure.yaml first."
	if len(available) > 0 {
		suggestion = fmt.Sprintf("Available names: %s.", strings.Join(available, ", "))
	}

	return &azdext.LocalError{
		Message:    fmt.Sprintf("--focus value %q does not name a service or resource in the graph", name),
		Code:       ErrCodeInvalidFlagUsage,
		Category:   azdext.LocalErrorCategoryValidation,
		Suggestion: suggestion,
	}
}

// newInvalidFlagUsageError reports an unusable flag combination or value.
func newInvalidFlagUsageError(message, suggestion string) error {
	return &azdext.LocalError{
		Message:    message,
		Code:       ErrCodeInvalidFlagUsage,
		Category:   azdext.LocalErrorCategoryValidation,
		Suggestion: suggestion,
	}
}

// newInvalidFlagValueError reports a flag whose value is outside its allowed
// set. It builds the "expected one of" wording so every flag reports the same
// shape.
func newInvalidFlagValueError(flag, value string, allowed []string) error {
	return &azdext.LocalError{
		Message:  fmt.Sprintf("invalid --%s value %q", flag, value),
		Code:     ErrCodeInvalidFlagUsage,
		Category: azdext.LocalErrorCategoryValidation,
		Suggestion: fmt.Sprintf(
			"Expected one of: %s.",
			strings.Join(allowed, ", "),
		),
	}
}

// newInvalidProjectConfigError reports an azure.yaml that exists but cannot be
// used as written.
func newInvalidProjectConfigError(message, suggestion string) error {
	return &azdext.LocalError{
		Message:    message,
		Code:       ErrCodeInvalidProjectConfig,
		Category:   azdext.LocalErrorCategoryUser,
		Suggestion: suggestion,
	}
}

// newProjectNotFoundError reports a missing azure.yaml.
func newProjectNotFoundError() error {
	return &azdext.LocalError{
		Message:    "no azure.yaml found in the current directory or any parent",
		Code:       ErrCodeProjectNotFound,
		Category:   azdext.LocalErrorCategoryUser,
		Suggestion: "Change to a project directory, or run `azd app reqs --generate` to create an azure.yaml here.",
	}
}

// newServiceNotFoundError reports a service name that is not in azure.yaml. It
// lists the available names when there are any, because the usual cause is a
// typo.
func newServiceNotFoundError(name string, available []string) error {
	suggestion := "No services are defined in azure.yaml. Add a `services` section to define one."
	if len(available) > 0 {
		suggestion = fmt.Sprintf("Available services: %s.", strings.Join(available, ", "))
	}

	return &azdext.LocalError{
		Message:    fmt.Sprintf("service %q not found", name),
		Code:       ErrCodeServiceNotFound,
		Category:   azdext.LocalErrorCategoryUser,
		Suggestion: suggestion,
	}
}
