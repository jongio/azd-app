package service_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

func TestAspireRuntimeModes(t *testing.T) {
	tests := []struct {
		name        string
		runtimeMode string
		wantArgs    func(csprojPath string) []string
	}{
		{
			name:        "Aspire mode uses dotnet run",
			runtimeMode: "aspire",
			wantArgs: func(csprojPath string) []string {
				return []string{"run", "--project", csprojPath}
			},
		},
		{
			name:        "AZD mode uses dotnet run with no-launch-profile",
			runtimeMode: "azd",
			wantArgs: func(csprojPath string) []string {
				return []string{"run", "--project", csprojPath, "--no-launch-profile"}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "aspire-runtime-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Create Aspire project structure
			csprojPath := filepath.Join(tmpDir, "TestAppHost.csproj")
			csprojContent := `<Project Sdk="Microsoft.NET.Sdk.Web">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`
			if err := os.WriteFile(csprojPath, []byte(csprojContent), 0600); err != nil {
				t.Fatalf("Failed to create csproj: %v", err)
			}

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

			// Create azure.yaml
			azureYamlContent := `name: test-aspire-app
services:
  apphost:
    project: .
    language: dotnet
    host: containerapp`
			azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
			if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0600); err != nil {
				t.Fatalf("Failed to create azure.yaml: %v", err)
			}

			// Parse azure.yaml
			azureYaml, err := service.ParseAzureYaml(azureYamlPath)
			if err != nil {
				t.Fatalf("Failed to parse azure.yaml: %v", err)
			}

			svc := azureYaml.Services["apphost"]

			// Detect runtime with specified mode
			runtime, err := service.DetectServiceRuntime("apphost", svc, map[int]bool{}, tmpDir, tt.runtimeMode)
			if err != nil {
				t.Fatalf("Failed to detect runtime: %v", err)
			}

			// Verify framework is Aspire
			if runtime.Framework != "Aspire" {
				t.Errorf("Expected framework 'Aspire', got %q", runtime.Framework)
			}

			// Verify command
			if runtime.Command != "dotnet" {
				t.Errorf("Expected command 'dotnet', got %q", runtime.Command)
			}

			// Verify args based on runtime mode
			wantArgs := tt.wantArgs(csprojPath)
			if len(runtime.Args) != len(wantArgs) {
				t.Errorf("Expected %d args, got %d: %v", len(wantArgs), len(runtime.Args), runtime.Args)
			} else {
				for i, arg := range wantArgs {
					if runtime.Args[i] != arg {
						t.Errorf("Arg[%d]: expected %q, got %q", i, arg, runtime.Args[i])
					}
				}
			}
		})
	}
}

func TestDetectServiceRuntimeWithMode(t *testing.T) {
	tests := []struct {
		name        string
		framework   string
		runtimeMode string
		checkCmd    func(runtime *service.ServiceRuntime) error
	}{
		{
			name:        "Next.js in azd mode",
			framework:   "Next.js",
			runtimeMode: "azd",
			checkCmd: func(runtime *service.ServiceRuntime) error {
				if runtime.Framework != "Next.js" {
					return fmt.Errorf("expected Next.js, got %s", runtime.Framework)
				}
				if len(runtime.Args) < 2 || runtime.Args[0] != "run" || runtime.Args[1] != "dev" {
					return fmt.Errorf("expected 'run dev' args, got %v", runtime.Args)
				}
				return nil
			},
		},
		{
			name:        "ASP.NET Core in azd mode",
			framework:   "ASP.NET Core",
			runtimeMode: "azd",
			checkCmd: func(runtime *service.ServiceRuntime) error {
				if runtime.Framework != "ASP.NET Core" {
					return fmt.Errorf("expected ASP.NET Core, got %s", runtime.Framework)
				}
				if runtime.Command != "dotnet" {
					return fmt.Errorf("expected dotnet command, got %s", runtime.Command)
				}
				return nil
			},
		},
		{
			name:        "Python FastAPI in azd mode",
			framework:   "FastAPI",
			runtimeMode: "azd",
			checkCmd: func(runtime *service.ServiceRuntime) error {
				if runtime.Framework != "FastAPI" {
					return fmt.Errorf("expected FastAPI, got %s", runtime.Framework)
				}
				if runtime.Command != "uvicorn" {
					return fmt.Errorf("expected uvicorn command, got %s", runtime.Command)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "runtime-mode-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Create project files based on framework
			switch tt.framework {
			case "Next.js":
				packageJSON := `{"name":"test","scripts":{"dev":"next dev"}}`
				if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0600); err != nil {
					t.Fatalf("Failed to create package.json: %v", err)
				}
				if err := os.WriteFile(filepath.Join(tmpDir, "next.config.js"), []byte("module.exports = {}"), 0600); err != nil {
					t.Fatalf("Failed to create next.config.js: %v", err)
				}
			case "ASP.NET Core":
				csproj := `<Project Sdk="Microsoft.NET.Sdk.Web"><PropertyGroup><TargetFramework>net8.0</TargetFramework></PropertyGroup></Project>`
				if err := os.WriteFile(filepath.Join(tmpDir, "Web.csproj"), []byte(csproj), 0600); err != nil {
					t.Fatalf("Failed to create .csproj: %v", err)
				}
			case "FastAPI":
				if err := os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte("fastapi"), 0600); err != nil {
					t.Fatalf("Failed to create requirements.txt: %v", err)
				}
				if err := os.WriteFile(filepath.Join(tmpDir, "main.py"), []byte("from fastapi import FastAPI"), 0600); err != nil {
					t.Fatalf("Failed to create main.py: %v", err)
				}
			}

			// Create azure.yaml
			azureYamlContent := `name: test-app
services:
  api:
    project: .
    host: containerapp`
			azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
			if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0600); err != nil {
				t.Fatalf("Failed to create azure.yaml: %v", err)
			}

			// Parse azure.yaml
			azureYaml, err := service.ParseAzureYaml(azureYamlPath)
			if err != nil {
				t.Fatalf("Failed to parse azure.yaml: %v", err)
			}

			svc := azureYaml.Services["api"]

			// Detect runtime with specified mode
			runtime, err := service.DetectServiceRuntime("api", svc, map[int]bool{}, tmpDir, tt.runtimeMode)
			if err != nil {
				t.Fatalf("Failed to detect runtime: %v", err)
			}

			// Run the check function
			if err := tt.checkCmd(runtime); err != nil {
				t.Errorf("Command check failed: %v", err)
			}
		})
	}
}

func TestRuntimeModeAspireVsAzd(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "runtime-mode-comparison-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create Aspire project
	csprojPath := filepath.Join(tmpDir, "AppHost.csproj")
	csprojContent := `<Project Sdk="Microsoft.NET.Sdk.Web">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`
	if err := os.WriteFile(csprojPath, []byte(csprojContent), 0600); err != nil {
		t.Fatalf("Failed to create csproj: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "AppHost.cs"), []byte("// Aspire"), 0600); err != nil {
		t.Fatalf("Failed to create AppHost.cs: %v", err)
	}

	// Create azure.yaml
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	azureYamlContent := `name: test-app
services:
  apphost:
    project: .
    host: containerapp`
	if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	azureYaml, err := service.ParseAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}

	svc := azureYaml.Services["apphost"]

	// Test aspire mode
	aspireRuntime, err := service.DetectServiceRuntime("apphost", svc, map[int]bool{}, tmpDir, "aspire")
	if err != nil {
		t.Fatalf("Failed to detect runtime in aspire mode: %v", err)
	}

	// Test azd mode
	azdRuntime, err := service.DetectServiceRuntime("apphost", svc, map[int]bool{}, tmpDir, "azd")
	if err != nil {
		t.Fatalf("Failed to detect runtime in azd mode: %v", err)
	}

	// Verify both detect Aspire framework
	if aspireRuntime.Framework != "Aspire" || azdRuntime.Framework != "Aspire" {
		t.Errorf("Expected both to detect Aspire framework")
	}

	// Verify aspire mode doesn't have --no-launch-profile
	hasNoLaunchProfile := false
	for _, arg := range aspireRuntime.Args {
		if arg == "--no-launch-profile" {
			hasNoLaunchProfile = true
			break
		}
	}
	if hasNoLaunchProfile {
		t.Error("Aspire mode should not have --no-launch-profile flag")
	}

	// Verify azd mode has --no-launch-profile
	hasNoLaunchProfile = false
	for _, arg := range azdRuntime.Args {
		if arg == "--no-launch-profile" {
			hasNoLaunchProfile = true
			break
		}
	}
	if !hasNoLaunchProfile {
		t.Error("AZD mode should have --no-launch-profile flag")
	}
}
