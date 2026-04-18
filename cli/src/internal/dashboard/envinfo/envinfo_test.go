package envinfo

import (
	"context"
	"errors"
	"testing"
)

// withEnv sets env vars for the duration of the test and restores prior
// values via t.Cleanup, including unsetting vars that were originally
// absent. Centralized so individual tests stay terse.
func withEnv(t *testing.T, vars map[string]string) {
	t.Helper()
	for key, value := range vars {
		t.Setenv(key, value)
	}
}

// withProbe replaces the package-level vsCodeStatusProbe for the test and
// restores it on cleanup.
func withProbe(t *testing.T, probe func(ctx context.Context) ([]byte, error)) {
	t.Helper()
	prior := vsCodeStatusProbe
	vsCodeStatusProbe = probe
	t.Cleanup(func() { vsCodeStatusProbe = prior })
}

func TestDetectOutsideCodespaceReturnsZeroValues(t *testing.T) {
	withEnv(t, map[string]string{
		"CODESPACES":     "",
		"CODESPACE_NAME": "",
		"GITHUB_CODESPACES_PORT_FORWARDING_DOMAIN": "",
		"AZURE_ENV_NAME": "",
	})
	withProbe(t, func(ctx context.Context) ([]byte, error) {
		t.Fatal("probe must not be called outside a Codespace")
		return nil, nil
	})

	got := Detect(context.Background())
	if got.Codespace.Enabled {
		t.Errorf("Codespace.Enabled=true want false outside Codespaces")
	}
	if got.Codespace.Name != "" || got.Codespace.Domain != "" {
		t.Errorf("expected zero Codespace name/domain, got %+v", got.Codespace)
	}
	if got.Codespace.IsVsCodeDesktop {
		t.Error("IsVsCodeDesktop=true outside Codespaces")
	}
	if got.EnvironmentName != "" {
		t.Errorf("EnvironmentName=%q want empty", got.EnvironmentName)
	}
}

func TestDetectInBrowserCodespaceFillsDefaultDomain(t *testing.T) {
	withEnv(t, map[string]string{
		"CODESPACES":     "true",
		"CODESPACE_NAME": "test-codespace",
		// GITHUB_CODESPACES_PORT_FORWARDING_DOMAIN intentionally unset
		"GITHUB_CODESPACES_PORT_FORWARDING_DOMAIN": "",
		"AZURE_ENV_NAME": "dev",
	})
	withProbe(t, func(ctx context.Context) ([]byte, error) {
		// Browser Codespace returns the unsupported-in-browsers message.
		return []byte("The --status argument is not yet supported in browsers."), nil
	})

	got := Detect(context.Background())
	if !got.Codespace.Enabled {
		t.Fatal("Codespace.Enabled=false want true")
	}
	if got.Codespace.Name != "test-codespace" {
		t.Errorf("Codespace.Name=%q want test-codespace", got.Codespace.Name)
	}
	if got.Codespace.Domain != "app.github.dev" {
		t.Errorf("Codespace.Domain=%q want app.github.dev (default)", got.Codespace.Domain)
	}
	if got.Codespace.IsVsCodeDesktop {
		t.Error("IsVsCodeDesktop=true in browser Codespace")
	}
	if got.EnvironmentName != "dev" {
		t.Errorf("EnvironmentName=%q want dev", got.EnvironmentName)
	}
}

func TestDetectInDesktopCodespaceUsesProvidedDomain(t *testing.T) {
	withEnv(t, map[string]string{
		"CODESPACES":     "true",
		"CODESPACE_NAME": "ws-1",
		"GITHUB_CODESPACES_PORT_FORWARDING_DOMAIN": "preview.app.github.dev",
		"AZURE_ENV_NAME": "prod",
	})
	withProbe(t, func(ctx context.Context) ([]byte, error) {
		// Desktop returns normal status output (no browser sentinel).
		return []byte("Version:          Code 1.95.0\nCommit:           abc123"), nil
	})

	got := Detect(context.Background())
	if got.Codespace.Domain != "preview.app.github.dev" {
		t.Errorf("Codespace.Domain=%q want preview.app.github.dev (override)", got.Codespace.Domain)
	}
	if !got.Codespace.IsVsCodeDesktop {
		t.Error("IsVsCodeDesktop=false in desktop Codespace")
	}
}

func TestDetectFallsBackWhenProbeFails(t *testing.T) {
	withEnv(t, map[string]string{
		"CODESPACES":     "true",
		"CODESPACE_NAME": "ws-2",
		"GITHUB_CODESPACES_PORT_FORWARDING_DOMAIN": "",
		"AZURE_ENV_NAME": "",
	})
	withProbe(t, func(ctx context.Context) ([]byte, error) {
		return nil, errors.New("code: command not found")
	})

	got := Detect(context.Background())
	if got.Codespace.IsVsCodeDesktop {
		t.Error("IsVsCodeDesktop=true when probe failed; want safe fallback false")
	}
}
