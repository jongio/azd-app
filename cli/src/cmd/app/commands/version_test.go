package commands

import (
	"testing"

	"github.com/jongio/azd-core/cliout"
	"github.com/jongio/azd-core/testutil"
)

func TestVersionCommandOutput(t *testing.T) {
	// Save original values
	origVersion := Version
	origBuildTime := BuildTime
	defer func() {
		Version = origVersion
		BuildTime = origBuildTime
	}()

	// Set test values
	Version = "1.2.3"
	BuildTime = "2026-01-10T12:00:00Z"

	t.Run("default output format", func(t *testing.T) {
		// Ensure we're not in JSON mode
		cliout.SetFormat("default")

		cmd := NewVersionCommand()
		outputStr := testutil.CaptureOutput(t, func() error {
			return cmd.RunE(cmd, []string{})
		})

		// Verify output contains version information
		if !testutil.Contains(outputStr, "1.2.3") {
			t.Errorf("Expected output to contain version '1.2.3', got: %s", outputStr)
		}

		if !testutil.Contains(outputStr, "2026-01-10T12:00:00Z") {
			t.Errorf("Expected output to contain build time, got: %s", outputStr)
		}

		// Verify it contains the label format
		if !testutil.Contains(outputStr, "Version") {
			t.Error("Expected output to contain 'Version' label")
		}
	})

	t.Run("json output format", func(t *testing.T) {
		// Set JSON mode
		cliout.SetFormat("json")
		defer cliout.SetFormat("default")

		cmd := NewVersionCommand()
		outputStr := testutil.CaptureOutput(t, func() error {
			return cmd.RunE(cmd, []string{})
		})

		// Verify JSON output structure
		if !testutil.Contains(outputStr, `"version"`) {
			t.Error("Expected JSON output to contain 'version' field")
		}

		if !testutil.Contains(outputStr, `"buildTime"`) {
			t.Error("Expected JSON output to contain 'buildTime' field")
		}

		if !testutil.Contains(outputStr, "1.2.3") {
			t.Errorf("Expected JSON to contain version value, got: %s", outputStr)
		}
	})

	t.Run("dev version", func(t *testing.T) {
		Version = "dev"
		BuildTime = "unknown"
		cliout.SetFormat("default")

		cmd := NewVersionCommand()
		outputStr := testutil.CaptureOutput(t, func() error {
			return cmd.RunE(cmd, []string{})
		})

		if !testutil.Contains(outputStr, "dev") {
			t.Error("Expected output to contain 'dev' version")
		}

		if !testutil.Contains(outputStr, "unknown") {
			t.Error("Expected output to contain 'unknown' build time")
		}
	})
}
