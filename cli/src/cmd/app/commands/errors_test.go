package commands

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
)

// asLocalError unwraps err to an *azdext.LocalError or fails the test.
func asLocalError(t *testing.T, err error) *azdext.LocalError {
	t.Helper()

	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	var local *azdext.LocalError
	if !errors.As(err, &local) {
		t.Fatalf("expected *azdext.LocalError, got %T: %v", err, err)
	}

	return local
}

func TestErrorCodesAreDistinct(t *testing.T) {
	codes := []string{
		ErrCodeCheckFailed,
		ErrCodeInvalidFlagUsage,
		ErrCodeInvalidProjectConfig,
		ErrCodeProjectNotFound,
		ErrCodeServiceNotFound,
	}

	seen := make(map[string]bool, len(codes))
	for _, code := range codes {
		if code == "" {
			t.Fatal("error codes must not be empty")
		}
		if code != strings.ToLower(code) {
			t.Errorf("error code %q must be lowercase", code)
		}
		if strings.ContainsAny(code, " -") {
			t.Errorf("error code %q must be snake_case with no spaces or hyphens", code)
		}
		if seen[code] {
			t.Errorf("duplicate error code %q", code)
		}
		seen[code] = true
	}
}

func TestNewProjectNotFoundError(t *testing.T) {
	local := asLocalError(t, newProjectNotFoundError())

	if local.Code != ErrCodeProjectNotFound {
		t.Errorf("Code = %q, want %q", local.Code, ErrCodeProjectNotFound)
	}
	if local.Category != azdext.LocalErrorCategoryUser {
		t.Errorf("Category = %q, want %q", local.Category, azdext.LocalErrorCategoryUser)
	}
	if local.Suggestion == "" {
		t.Error("expected a suggestion telling the user how to get an azure.yaml")
	}
}

func TestNewServiceNotFoundError(t *testing.T) {
	t.Run("lists available services", func(t *testing.T) {
		local := asLocalError(t, newServiceNotFoundError("ap", []string{"api", "web"}))

		if local.Code != ErrCodeServiceNotFound {
			t.Errorf("Code = %q, want %q", local.Code, ErrCodeServiceNotFound)
		}
		if !strings.Contains(local.Message, `"ap"`) {
			t.Errorf("Message = %q, want it to name the missing service", local.Message)
		}
		if !strings.Contains(local.Suggestion, "api") || !strings.Contains(local.Suggestion, "web") {
			t.Errorf("Suggestion = %q, want it to list the available services", local.Suggestion)
		}
	})

	t.Run("handles a project with no services", func(t *testing.T) {
		local := asLocalError(t, newServiceNotFoundError("api", nil))

		if strings.Contains(local.Suggestion, "Available services") {
			t.Errorf("Suggestion = %q, want it to not offer an empty list", local.Suggestion)
		}
		if !strings.Contains(local.Suggestion, "services") {
			t.Errorf("Suggestion = %q, want it to mention the services section", local.Suggestion)
		}
	})
}

func TestNewInvalidFlagValueError(t *testing.T) {
	local := asLocalError(t, newInvalidFlagValueError("format", "xml", []string{"text", "json"}))

	if local.Code != ErrCodeInvalidFlagUsage {
		t.Errorf("Code = %q, want %q", local.Code, ErrCodeInvalidFlagUsage)
	}
	if local.Category != azdext.LocalErrorCategoryValidation {
		t.Errorf("Category = %q, want %q", local.Category, azdext.LocalErrorCategoryValidation)
	}
	if !strings.Contains(local.Message, "--format") {
		t.Errorf("Message = %q, want it to name the flag", local.Message)
	}
	if !strings.Contains(local.Message, `"xml"`) {
		t.Errorf("Message = %q, want it to quote the rejected value", local.Message)
	}
	if !strings.Contains(local.Suggestion, "text, json") {
		t.Errorf("Suggestion = %q, want it to list the allowed values", local.Suggestion)
	}
}

func TestNewInvalidFlagUsageError(t *testing.T) {
	local := asLocalError(t, newInvalidFlagUsageError("cannot combine --a with --b", "Pick one."))

	if local.Code != ErrCodeInvalidFlagUsage {
		t.Errorf("Code = %q, want %q", local.Code, ErrCodeInvalidFlagUsage)
	}
	if local.Category != azdext.LocalErrorCategoryValidation {
		t.Errorf("Category = %q, want %q", local.Category, azdext.LocalErrorCategoryValidation)
	}
}

func TestNewInvalidProjectConfigError(t *testing.T) {
	local := asLocalError(t, newInvalidProjectConfigError("azure.yaml root must be a mapping", "Fix it."))

	if local.Code != ErrCodeInvalidProjectConfig {
		t.Errorf("Code = %q, want %q", local.Code, ErrCodeInvalidProjectConfig)
	}
	if local.Category != azdext.LocalErrorCategoryUser {
		t.Errorf("Category = %q, want %q", local.Category, azdext.LocalErrorCategoryUser)
	}
}

func TestNewCheckFailedError(t *testing.T) {
	local := asLocalError(t, newCheckFailedError("2 service(s) unhealthy", "Check the logs."))

	if local.Code != ErrCodeCheckFailed {
		t.Errorf("Code = %q, want %q", local.Code, ErrCodeCheckFailed)
	}
	// A failing check is a result the user must act on, not a bug in the
	// extension, so it stays in the user category.
	if local.Category != azdext.LocalErrorCategoryUser {
		t.Errorf("Category = %q, want %q", local.Category, azdext.LocalErrorCategoryUser)
	}
}

// TestErrorsSurviveWrapping guards the contract that matters at the call site:
// callers wrap these with %w on the way up, and the azd host still has to be
// able to read the code back out.
func TestErrorsSurviveWrapping(t *testing.T) {
	wrapped := fmt.Errorf("loading project: %w", newProjectNotFoundError())

	local := asLocalError(t, wrapped)
	if local.Code != ErrCodeProjectNotFound {
		t.Errorf("Code = %q, want %q after wrapping", local.Code, ErrCodeProjectNotFound)
	}
}
