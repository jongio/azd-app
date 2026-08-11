package commands

import (
	"errors"
	"fmt"
	"os"
	"regexp"
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
// able to read the code back out. Every constructor is covered, because call
// sites across core.go, deps_check.go, doctor.go, add.go and graph.go wrap all
// of them and the contract is worthless if it holds for only one.
func TestErrorsSurviveWrapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"check failed", newCheckFailedError("a check failed", "fix it"), ErrCodeCheckFailed},
		{"graph node not found", newGraphNodeNotFoundError("api", []string{"web"}), ErrCodeInvalidFlagUsage},
		{"invalid flag usage", newInvalidFlagUsageError("bad combination", "pick one"), ErrCodeInvalidFlagUsage},
		{"invalid flag value", newInvalidFlagValueError("output", "svg", []string{"text"}), ErrCodeInvalidFlagUsage},
		{"invalid project config", newInvalidProjectConfigError("bad azure.yaml", "fix it"), ErrCodeInvalidProjectConfig},
		{"project not found", newProjectNotFoundError(), ErrCodeProjectNotFound},
		{"service not found", newServiceNotFoundError("api", []string{"web"}), ErrCodeServiceNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wrapped := fmt.Errorf("loading project: %w", tc.err)

			local := asLocalError(t, wrapped)
			if local.Code != tc.want {
				t.Errorf("Code = %q, want %q after wrapping", local.Code, tc.want)
			}
		})
	}
}

// TestEveryErrorConstructorIsWrappingTested fails when a constructor is added
// to errors.go without a matching case above, so the contract cannot quietly
// fall behind the code.
func TestEveryErrorConstructorIsWrappingTested(t *testing.T) {
	source, err := os.ReadFile("errors.go")
	if err != nil {
		t.Fatalf("read errors.go: %v", err)
	}

	declared := regexp.MustCompile(`(?m)^func (new\w+Error)\(`).FindAllStringSubmatch(string(source), -1)
	if len(declared) == 0 {
		t.Fatal("found no error constructors in errors.go")
	}

	tested, err := os.ReadFile("errors_test.go")
	if err != nil {
		t.Fatalf("read errors_test.go: %v", err)
	}
	body := string(tested)
	start := strings.Index(body, "func TestErrorsSurviveWrapping(")
	if start < 0 {
		t.Fatal("TestErrorsSurviveWrapping not found")
	}
	end := strings.Index(body[start:], "\n}\n")
	if end < 0 {
		t.Fatal("could not delimit TestErrorsSurviveWrapping")
	}
	table := body[start : start+end]

	for _, match := range declared {
		if !strings.Contains(table, match[1]+"(") {
			t.Errorf("%s has no case in TestErrorsSurviveWrapping", match[1])
		}
	}
}
