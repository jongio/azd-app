package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

func newExceptTestYaml() *service.AzureYaml {
	return &service.AzureYaml{
		Services: map[string]service.Service{
			"api":    {Language: "go"},
			"web":    {Language: "js"},
			"worker": {Type: "process"},
		},
	}
}

func TestExcludeServicesRemovesNamed(t *testing.T) {
	remaining, err := excludeServices(newExceptTestYaml(), []string{"web"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(remaining) != 2 {
		t.Fatalf("expected 2 services, got %d", len(remaining))
	}
	if _, ok := remaining["web"]; ok {
		t.Error("expected web to be excluded")
	}
	if _, ok := remaining["api"]; !ok {
		t.Error("expected api to remain")
	}
	if _, ok := remaining["worker"]; !ok {
		t.Error("expected worker to remain")
	}
}

func TestExcludeServicesMultiple(t *testing.T) {
	remaining, err := excludeServices(newExceptTestYaml(), []string{"web", "worker"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected 1 service, got %d", len(remaining))
	}
	if _, ok := remaining["api"]; !ok {
		t.Error("expected api to remain")
	}
}

func TestExcludeServicesTrimsWhitespace(t *testing.T) {
	remaining, err := excludeServices(newExceptTestYaml(), []string{" web ", "", "  "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(remaining) != 2 {
		t.Fatalf("expected 2 services after trimming, got %d", len(remaining))
	}
	if _, ok := remaining["web"]; ok {
		t.Error("expected trimmed web name to be excluded")
	}
}

func TestExcludeServicesUnknown(t *testing.T) {
	_, err := excludeServices(newExceptTestYaml(), []string{"nope"})
	if err == nil {
		t.Fatal("expected error for unknown service")
	}
	msg := err.Error()
	if !strings.Contains(msg, "nope") {
		t.Errorf("error should name the unknown service, got %q", msg)
	}
	for _, name := range []string{"api", "web", "worker"} {
		if !strings.Contains(msg, name) {
			t.Errorf("error should list available service %q, got %q", name, msg)
		}
	}
}

func TestExcludeServicesAll(t *testing.T) {
	remaining, err := excludeServices(newExceptTestYaml(), []string{"api", "web", "worker"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("expected no services to remain, got %d", len(remaining))
	}
}

func TestSelectRunServicesFlags(t *testing.T) {
	savedFilter, savedExcept := runServiceFilter, runExcept
	defer func() {
		runServiceFilter = savedFilter
		runExcept = savedExcept
	}()

	yaml := newExceptTestYaml()

	// No filters: every service runs.
	runServiceFilter, runExcept = "", ""
	all, err := selectRunServices(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected all 3 services, got %d", len(all))
	}

	// --service selects only the named service.
	runServiceFilter, runExcept = "api", ""
	only, err := selectRunServices(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(only) != 1 {
		t.Fatalf("expected 1 service, got %d", len(only))
	}
	if _, ok := only["api"]; !ok {
		t.Error("expected api to be selected")
	}

	// --except skips the named service.
	runServiceFilter, runExcept = "", "api"
	rest, err := selectRunServices(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rest) != 2 {
		t.Fatalf("expected 2 services, got %d", len(rest))
	}
	if _, ok := rest["api"]; ok {
		t.Error("expected api to be excluded")
	}
}

func TestRunCommandExceptFlagMutualExclusion(t *testing.T) {
	savedFilter, savedExcept, savedRuntime := runServiceFilter, runExcept, runRuntime
	defer func() {
		runServiceFilter = savedFilter
		runExcept = savedExcept
		runRuntime = savedRuntime
	}()

	runServiceFilter = "api"
	runExcept = "web"
	runRuntime = "azd"

	err := runWithServices(context.Background(), nil, nil, nil)
	if err == nil {
		t.Fatal("expected error when --service and --except are combined")
	}
	if !strings.Contains(err.Error(), "cannot be used together") {
		t.Errorf("expected mutual-exclusion error, got %q", err)
	}
}
