package commands

import (
	"encoding/json"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

func boolPtr(b bool) *bool { return &b }

func TestCollectHooksEmpty(t *testing.T) {
	if got := collectHooks(nil); len(got) != 0 {
		t.Errorf("expected no hooks for nil, got %+v", got)
	}
	if got := collectHooks(&service.Hooks{}); len(got) != 0 {
		t.Errorf("expected no hooks for empty struct, got %+v", got)
	}
}

func TestCollectHooksOrderAndFields(t *testing.T) {
	hooks := &service.Hooks{
		Poststop: &service.Hook{Run: "echo stopped"},
		Prerun: &service.Hook{
			Run:             "npm run setup",
			Shell:           "bash",
			ContinueOnError: true,
			Interactive:     true,
		},
	}

	got := collectHooks(hooks)

	if len(got) != 2 {
		t.Fatalf("expected 2 hooks, got %d", len(got))
	}
	// prerun sorts before poststop in lifecycle order.
	if got[0].Name != "prerun" || got[1].Name != "poststop" {
		t.Fatalf("unexpected order: %s, %s", got[0].Name, got[1].Name)
	}

	prerun := got[0]
	if prerun.Run != "npm run setup" || prerun.Shell != "bash" || !prerun.ContinueOnError || !prerun.Interactive {
		t.Errorf("unexpected prerun fields: %+v", prerun)
	}
	if got[1].Run != "echo stopped" || got[1].Shell != "" {
		t.Errorf("unexpected poststop fields: %+v", got[1])
	}
}

func TestCollectHooksPlatformOverride(t *testing.T) {
	hooks := &service.Hooks{
		Prerun: &service.Hook{
			Run: "./setup.sh",
			Windows: &service.PlatformHook{
				Run:             "setup.ps1",
				Shell:           "pwsh",
				ContinueOnError: boolPtr(true),
			},
			Posix: &service.PlatformHook{
				Run:   "./setup.sh",
				Shell: "bash",
			},
		},
	}

	got := collectHooks(hooks)

	if len(got) != 1 {
		t.Fatalf("expected 1 hook, got %d", len(got))
	}
	win := got[0].Windows
	if win == nil || win.Run != "setup.ps1" || win.Shell != "pwsh" || win.ContinueOnError == nil || !*win.ContinueOnError {
		t.Errorf("unexpected windows override: %+v", win)
	}
	posix := got[0].Posix
	if posix == nil || posix.Run != "./setup.sh" || posix.Shell != "bash" {
		t.Errorf("unexpected posix override: %+v", posix)
	}
	if posix.ContinueOnError != nil {
		t.Errorf("expected nil continueOnError for posix, got %v", *posix.ContinueOnError)
	}
}

func TestToHookOverrideNil(t *testing.T) {
	if toHookOverride(nil) != nil {
		t.Error("expected nil override for nil platform hook")
	}
}

func TestHooksJSONShape(t *testing.T) {
	hooks := &service.Hooks{
		Prerun: &service.Hook{Run: "npm run setup", Shell: "bash"},
	}

	data, err := json.Marshal(collectHooks(hooks))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded []struct {
		Name  string `json:"name"`
		Run   string `json:"run"`
		Shell string `json:"shell"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(decoded) != 1 || decoded[0].Name != "prerun" || decoded[0].Run != "npm run setup" || decoded[0].Shell != "bash" {
		t.Errorf("unexpected json shape: %+v", decoded)
	}
}

func TestShellHelpers(t *testing.T) {
	if shellOrDefault("") != "(default)" {
		t.Error("empty shell should render as (default)")
	}
	if shellOrDefault("pwsh") != "pwsh" {
		t.Error("explicit shell should be preserved")
	}
	if shellOrNote("") != "(none)" {
		t.Error("empty run should render as (none)")
	}
	if shellOrNote("echo hi") != "echo hi" {
		t.Error("explicit run should be preserved")
	}
}

func TestOverrideSummary(t *testing.T) {
	got := overrideSummary(&hookOverride{Run: "setup.ps1", Shell: "pwsh"})
	if got != "setup.ps1 (shell: pwsh)" {
		t.Errorf("unexpected summary: %q", got)
	}
	got = overrideSummary(&hookOverride{Run: "setup.ps1"})
	if got != "setup.ps1" {
		t.Errorf("unexpected summary without shell: %q", got)
	}
}
