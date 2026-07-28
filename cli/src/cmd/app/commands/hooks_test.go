package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/cliout"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	got = overrideSummary(&hookOverride{
		Run:             "setup.ps1",
		ContinueOnError: boolPtr(true),
		Interactive:     boolPtr(true),
	})
	if got != "setup.ps1 (continueOnError: true) (interactive: true)" {
		t.Errorf("unexpected summary with true flags: %q", got)
	}
	got = overrideSummary(&hookOverride{
		Run:             "setup.ps1",
		ContinueOnError: boolPtr(false),
		Interactive:     boolPtr(false),
	})
	if got != "setup.ps1 (continueOnError: false) (interactive: false)" {
		t.Errorf("unexpected summary with false flags: %q", got)
	}
	got = overrideSummary(&hookOverride{
		Run:         "setup.ps1",
		Interactive: boolPtr(false),
	})
	if got != "setup.ps1 (interactive: false)" {
		t.Errorf("unexpected summary with unset continueOnError: %q", got)
	}
}

func TestNewHooksCommand(t *testing.T) {
	cmd := NewHooksCommand()

	assert.Equal(t, "hooks", cmd.Use)
	assert.Equal(t, "List the lifecycle hooks configured in azure.yaml", cmd.Short)
	assert.True(t, cmd.SilenceUsage)
	assert.NoError(t, cmd.Args(cmd, []string{}))
	assert.EqualError(t, cmd.Args(cmd, []string{"extra"}), `unknown command "extra" for "hooks"`)
	flagCount := 0
	cmd.LocalFlags().VisitAll(func(*pflag.Flag) {
		flagCount++
	})
	assert.Equal(t, 0, flagCount)
}

func TestPrintHooks(t *testing.T) {
	originalFormat := cliout.GetFormat()
	t.Cleanup(func() { _ = cliout.SetFormat(string(originalFormat)) })
	require.NoError(t, cliout.SetFormat("default"))

	tests := []struct {
		name  string
		hooks []hookInfo
		want  []string
	}{
		{
			name:  "empty",
			hooks: nil,
			want:  []string{"No lifecycle hooks are configured in azure.yaml"},
		},
		{
			name: "configured hooks",
			hooks: []hookInfo{
				{
					Name:            "prerun",
					Run:             "npm run setup",
					Shell:           "bash",
					ContinueOnError: true,
					Interactive:     true,
					Windows: &hookOverride{
						Run:             "setup.ps1",
						Shell:           "pwsh",
						ContinueOnError: boolPtr(false),
					},
				},
				{
					Name: "postrun",
					Posix: &hookOverride{
						Interactive: boolPtr(true),
					},
				},
			},
			want: []string{
				"azd app hooks",
				"prerun",
				"run",
				"npm run setup",
				"shell",
				"bash",
				"continueOnError",
				"true",
				"interactive",
				"windows",
				"setup.ps1 (shell: pwsh) (continueOnError: false)",
				"postrun",
				"(none)",
				"(default)",
				"posix",
				"(none) (interactive: true)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := captureStdout(t, func() error {
				printHooks(tt.hooks)
				return nil
			})

			require.NoError(t, err)
			for _, want := range tt.want {
				assert.Contains(t, out, want)
			}
		})
	}
}

func TestRunHooks(t *testing.T) {
	originalFormat := cliout.GetFormat()
	t.Cleanup(func() { _ = cliout.SetFormat(string(originalFormat)) })

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	chdirTemp := func(t *testing.T, dir string) {
		t.Helper()
		require.NoError(t, os.Chdir(dir))
		t.Cleanup(func() { _ = os.Chdir(originalDir) })
	}

	writeHooksAzureYaml := func(t *testing.T, dir string) {
		t.Helper()
		content := `name: hooks-command-test
hooks:
  prerun:
    run: npm run setup
    shell: bash
    continueOnError: true
    interactive: true
    windows:
      run: setup.ps1
      shell: pwsh
      continueOnError: false
  poststop:
    run: echo stopped
`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte(content), 0o600))
	}

	t.Run("text output lists hooks from azure yaml", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeHooksAzureYaml(t, tmpDir)
		chdirTemp(t, tmpDir)
		require.NoError(t, cliout.SetFormat("default"))

		out, runErr := captureStdout(t, func() error {
			return runHooks(nil, nil)
		})

		require.NoError(t, runErr)
		for _, want := range []string{
			"azd app hooks",
			"prerun",
			"npm run setup",
			"bash",
			"continueOnError",
			"interactive",
			"windows",
			"setup.ps1 (shell: pwsh) (continueOnError: false)",
			"poststop",
			"echo stopped",
		} {
			assert.Contains(t, out, want)
		}
	})

	t.Run("json output lists hooks from azure yaml", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeHooksAzureYaml(t, tmpDir)
		chdirTemp(t, tmpDir)
		require.NoError(t, cliout.SetFormat("json"))

		out, runErr := captureStdout(t, func() error {
			return runHooks(nil, nil)
		})

		require.NoError(t, runErr)
		var got []hookInfo
		require.NoError(t, json.Unmarshal([]byte(out), &got))
		require.Len(t, got, 2)
		assert.Equal(t, "prerun", got[0].Name)
		assert.Equal(t, "npm run setup", got[0].Run)
		assert.Equal(t, "bash", got[0].Shell)
		assert.True(t, got[0].ContinueOnError)
		assert.True(t, got[0].Interactive)
		require.NotNil(t, got[0].Windows)
		assert.Equal(t, "setup.ps1", got[0].Windows.Run)
		assert.Equal(t, "pwsh", got[0].Windows.Shell)
		require.NotNil(t, got[0].Windows.ContinueOnError)
		assert.False(t, *got[0].Windows.ContinueOnError)
		assert.Equal(t, "poststop", got[1].Name)
		assert.Equal(t, "echo stopped", got[1].Run)
	})

	t.Run("missing azure yaml returns load error", func(t *testing.T) {
		tmpDir := t.TempDir()
		chdirTemp(t, tmpDir)
		require.NoError(t, cliout.SetFormat("default"))

		_, runErr := captureStdout(t, func() error {
			return runHooks(nil, nil)
		})

		require.Error(t, runErr)
		assert.True(t, strings.HasPrefix(runErr.Error(), "failed to load azure.yaml:"))
	})
}
