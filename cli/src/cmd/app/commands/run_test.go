package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/spf13/cobra"
)

func TestRunCommandFlags(t *testing.T) {
	cmd := NewRunCommand()

	tests := []struct {
		name        string
		args        []string
		wantRuntime string
		wantError   bool
	}{
		{
			name:        "Default runtime should be azd",
			args:        []string{},
			wantRuntime: "azd",
			wantError:   false,
		},
		{
			name:        "Valid runtime azd",
			args:        []string{"--runtime", "azd"},
			wantRuntime: "azd",
			wantError:   false,
		},
		{
			name:        "Valid runtime aspire",
			args:        []string{"--runtime", "aspire"},
			wantRuntime: "aspire",
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags for each test
			cmd = NewRunCommand()
			runRuntime = "azd" // reset to default

			// Parse flags
			cmd.ParseFlags(tt.args)

			// Check runtime value
			if runRuntime != tt.wantRuntime {
				t.Errorf("Expected runtime %q, got %q", tt.wantRuntime, runRuntime)
			}
		})
	}
}

func TestRunCommandRuntimeValidation(t *testing.T) {
	tests := []struct {
		name      string
		runtime   string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "Valid runtime azd",
			runtime:   runtimeModeAzd,
			wantError: false,
		},
		{
			name:      "Valid runtime aspire",
			runtime:   runtimeModeAspire,
			wantError: false,
		},
		{
			name:      "Invalid runtime foo",
			runtime:   "foo",
			wantError: true,
			errorMsg:  "invalid --runtime value",
		},
		{
			name:      "Invalid runtime bar",
			runtime:   "bar",
			wantError: true,
			errorMsg:  "invalid --runtime value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the runtime variable
			runRuntime = tt.runtime

			// Create a minimal mock command for testing
			cmd := &cobra.Command{
				RunE: func(cmd *cobra.Command, args []string) error {
					// Only test the validation logic
					return validateRuntimeMode(runRuntime)
				},
			}

			err := cmd.RunE(cmd, []string{})

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestRunCommandFlagDefaults(t *testing.T) {
	cmd := NewRunCommand()

	// Check that flags exist with correct defaults
	runtimeFlag := cmd.Flags().Lookup("runtime")
	if runtimeFlag == nil {
		t.Fatal("--runtime flag not found")
	}

	if runtimeFlag.DefValue != "azd" {
		t.Errorf("Expected default runtime to be 'azd', got %q", runtimeFlag.DefValue)
	}

	serviceFlag := cmd.Flags().Lookup("service")
	if serviceFlag == nil {
		t.Fatal("--service flag not found")
	}

	verboseFlag := cmd.Flags().Lookup("verbose")
	if verboseFlag == nil {
		t.Fatal("--verbose flag not found")
	}

	dryRunFlag := cmd.Flags().Lookup("dry-run")
	if dryRunFlag == nil {
		t.Fatal("--dry-run flag not found")
	}

	envFileFlag := cmd.Flags().Lookup("env-file")
	if envFileFlag == nil {
		t.Fatal("--env-file flag not found")
	}
}

func TestRunAspireMode(t *testing.T) {
	// Create temporary directory with Aspire project
	tmpDir, err := os.MkdirTemp("", "aspire-mode-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create AppHost.csproj
	csprojPath := filepath.Join(tmpDir, "AppHost.csproj")
	csprojContent := `<Project Sdk="Microsoft.NET.Sdk.Web">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`
	if err := os.WriteFile(csprojPath, []byte(csprojContent), 0600); err != nil {
		t.Fatalf("Failed to create csproj: %v", err)
	}

	// Create AppHost.cs
	appHostPath := filepath.Join(tmpDir, "AppHost.cs")
	appHostContent := `// Aspire AppHost
namespace TestAppHost;
public class Program {
    public static void Main(string[] args) {
        var builder = DistributedApplication.CreateBuilder(args);
        builder.Build().Run();
    }
}`
	if err := os.WriteFile(appHostPath, []byte(appHostContent), 0600); err != nil {
		t.Fatalf("Failed to create AppHost.cs: %v", err)
	}

	// Test that runAspireMode can find the project
	// Note: We can't actually run it in tests, but we can verify the function doesn't error on setup
	aspireProject, err := detector.FindAppHost(tmpDir)
	if err != nil {
		t.Fatalf("Failed to find AppHost: %v", err)
	}

	if aspireProject == nil {
		t.Fatal("Expected to find Aspire project, got nil")
	}

	if aspireProject.Dir != tmpDir {
		t.Errorf("Expected dir %q, got %q", tmpDir, aspireProject.Dir)
	}

	if aspireProject.ProjectFile != csprojPath {
		t.Errorf("Expected project file %q, got %q", csprojPath, aspireProject.ProjectFile)
	}
}
