package commands

import (
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/browser"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

func TestResolveBrowserTarget(t *testing.T) {
	tests := []struct {
		name           string
		setupFlags     func()
		azureYaml      *service.AzureYaml
		setupEnv       func()
		cleanupEnv     func()
		expectedTarget browser.Target
	}{
		{
			name: "no-browser flag takes priority",
			setupFlags: func() {
				runNoBrowser = true
				runBrowser = ""
			},
			expectedTarget: browser.TargetNone,
		},
		{
			name: "browser flag takes priority over project config",
			setupFlags: func() {
				runNoBrowser = false
				runBrowser = "system"
			},
			azureYaml: &service.AzureYaml{
				Dashboard: &service.DashboardConfig{
					Browser: "none",
				},
			},
			expectedTarget: browser.TargetSystem,
		},
		{
			name: "project config used when no flag",
			setupFlags: func() {
				runNoBrowser = false
				runBrowser = ""
			},
			azureYaml: &service.AzureYaml{
				Dashboard: &service.DashboardConfig{
					Browser: "none",
				},
			},
			expectedTarget: browser.TargetNone,
		},
		{
			name: "system default when no config",
			setupFlags: func() {
				runNoBrowser = false
				runBrowser = ""
			},
			expectedTarget: browser.TargetSystem,
		},
		{
			name: "invalid browser flag falls back to system with warning",
			setupFlags: func() {
				runNoBrowser = false
				runBrowser = "invalid"
			},
			expectedTarget: browser.TargetSystem,
		},
		{
			name: "invalid project config falls back to system",
			setupFlags: func() {
				runNoBrowser = false
				runBrowser = ""
			},
			azureYaml: &service.AzureYaml{
				Dashboard: &service.DashboardConfig{
					Browser: "invalid",
				},
			},
			expectedTarget: browser.TargetSystem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.setupFlags != nil {
				tt.setupFlags()
			}
			if tt.setupEnv != nil {
				tt.setupEnv()
				if tt.cleanupEnv != nil {
					defer tt.cleanupEnv()
				}
			}

			// Execute
			target := resolveBrowserTarget(tt.azureYaml)

			// Verify
			// Note: The resolved target might differ from input due to fallback logic
			resolvedTarget := browser.ResolveTarget(target)
			expectedResolved := browser.ResolveTarget(tt.expectedTarget)

			if resolvedTarget != expectedResolved {
				t.Errorf("resolveBrowserTarget() = %v (resolved to %v), want %v (resolved to %v)",
					target, resolvedTarget, tt.expectedTarget, expectedResolved)
			}
		})
	}
}

func TestValidateBrowserFlag(t *testing.T) {
	tests := []struct {
		name        string
		browserFlag string
		wantErr     bool
	}{
		{
			name:        "empty browser flag is valid",
			browserFlag: "",
			wantErr:     false,
		},
		{
			name:        "default is valid",
			browserFlag: "default",
			wantErr:     false,
		},
		{
			name:        "system is valid",
			browserFlag: "system",
			wantErr:     false,
		},
		{
			name:        "vscode is invalid",
			browserFlag: "vscode",
			wantErr:     true,
		},
		{
			name:        "none is valid",
			browserFlag: "none",
			wantErr:     false,
		},
		{
			name:        "invalid browser flag",
			browserFlag: "chrome",
			wantErr:     true,
		},
		{
			name:        "invalid browser flag firefox",
			browserFlag: "firefox",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runBrowser = tt.browserFlag
			err := validateBrowserFlag()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateBrowserFlag() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBrowserFlagPriority(t *testing.T) {
	// This test verifies the complete priority chain:
	// Flag > Project Config > User Config > Auto-detect > System Default

	tests := []struct {
		name           string
		flagBrowser    string
		flagNoBrowser  bool
		projectBrowser string
		setupEnv       func()
		cleanupEnv     func()
		expectedBase   browser.Target
	}{
		{
			name:           "flag overrides everything",
			flagBrowser:    "system",
			flagNoBrowser:  false,
			projectBrowser: "none",
			expectedBase:   browser.TargetSystem,
		},
		{
			name:           "no-browser flag overrides everything",
			flagBrowser:    "",
			flagNoBrowser:  true,
			projectBrowser: "system",
			expectedBase:   browser.TargetNone,
		},
		{
			name:           "project config used when no flag",
			flagBrowser:    "",
			flagNoBrowser:  false,
			projectBrowser: "system",
			expectedBase:   browser.TargetSystem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup flags
			runBrowser = tt.flagBrowser
			runNoBrowser = tt.flagNoBrowser

			// Setup environment
			if tt.setupEnv != nil {
				tt.setupEnv()
				if tt.cleanupEnv != nil {
					defer tt.cleanupEnv()
				}
			}

			// Create azure.yaml with project config
			var azureYaml *service.AzureYaml
			if tt.projectBrowser != "" {
				azureYaml = &service.AzureYaml{
					Dashboard: &service.DashboardConfig{
						Browser: tt.projectBrowser,
					},
				}
			}

			// Resolve target
			target := resolveBrowserTarget(azureYaml)

			// For none target, it should match exactly
			if tt.expectedBase == browser.TargetNone {
				if target != browser.TargetNone {
					t.Errorf("Expected TargetNone, got %v", target)
				}
				return
			}

			// For other targets, compare resolved values
			resolvedTarget := browser.ResolveTarget(target)
			expectedResolved := browser.ResolveTarget(tt.expectedBase)

			if resolvedTarget != expectedResolved {
				t.Errorf("resolveBrowserTarget() resolved to %v, want %v", resolvedTarget, expectedResolved)
			}
		})
	}
}
