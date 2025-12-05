package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// newAzureYamlResource creates a resource for reading azure.yaml
func newAzureYamlResource() server.ServerResource {
	return server.ServerResource{
		Resource: mcp.NewResource(
			"azure://project/azure.yaml",
			"azure.yaml",
			mcp.WithResourceDescription("The azure.yaml configuration file that defines the project structure, services, and dependencies."),
			mcp.WithAnnotations([]mcp.Role{mcp.RoleUser, mcp.RoleAssistant}, 0.9),
			mcp.WithMIMEType("application/x-yaml"),
		),
		Handler: func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			// Check context
			if err := ctx.Err(); err != nil {
				return nil, fmt.Errorf("request cancelled: %w", err)
			}

			// Get and validate project directory
			projectDir := getProjectDir()
			validatedDir, err := validateProjectDir(projectDir)
			if err != nil {
				return nil, fmt.Errorf("invalid project directory: %w", err)
			}

			// Find and read azure.yaml from project directory
			azureYamlPath := filepath.Join(validatedDir, "azure.yaml")

			// Verify the file path is still within the project directory (defense in depth)
			cleanPath := filepath.Clean(azureYamlPath)
			if !strings.HasPrefix(cleanPath, validatedDir) && validatedDir != "." {
				return nil, fmt.Errorf("azure.yaml path escapes project directory")
			}

			content, err := os.ReadFile(cleanPath)
			if err != nil {
				if os.IsNotExist(err) {
					return nil, fmt.Errorf("azure.yaml not found in project directory: %s", validatedDir)
				}
				return nil, fmt.Errorf("failed to read azure.yaml: %w", err)
			}

			return []mcp.ResourceContents{
				&mcp.TextResourceContents{
					URI:      request.Params.URI,
					Text:     string(content),
					MIMEType: "application/x-yaml",
				},
			}, nil
		},
	}
}

// newServiceConfigResource creates a resource for reading service configurations
func newServiceConfigResource() server.ServerResource {
	return server.ServerResource{
		Resource: mcp.NewResource(
			"azure://project/services/configs",
			"service-configs",
			mcp.WithResourceDescription("Configuration details for all services including environment variables, ports, and runtime settings."),
			mcp.WithAnnotations([]mcp.Role{mcp.RoleAssistant}, 0.7),
			mcp.WithMIMEType("application/json"),
		),
		Handler: func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			// Check context
			if err := ctx.Err(); err != nil {
				return nil, fmt.Errorf("request cancelled: %w", err)
			}

			// Get service configurations from project directory
			var cmdArgs []string
			projectDir := getProjectDir()
			if projectDir != "." {
				validatedDir, err := validateProjectDir(projectDir)
				if err != nil {
					return nil, fmt.Errorf("invalid project directory: %w", err)
				}
				cmdArgs = append(cmdArgs, cwdFlag, validatedDir)
			}

			result, err := executeAzdAppCommand(ctx, "info", cmdArgs)
			if err != nil {
				return nil, fmt.Errorf("failed to get service configs: %w", err)
			}

			// Extract just the configuration parts (not runtime status)
			configs := make(map[string]interface{})
			if services, ok := result["services"].([]interface{}); ok {
				for _, svc := range services {
					if svcMap, ok := svc.(map[string]interface{}); ok {
						svcName, _ := svcMap["name"].(string)
						if svcName == "" {
							continue // Skip services without names
						}
						config := map[string]interface{}{
							"name":      svcMap["name"],
							"language":  svcMap["language"],
							"framework": svcMap["framework"],
							"project":   svcMap["project"],
							"env":       svcMap["env"],
						}
						configs[svcName] = config
					}
				}
			}

			jsonBytes, err := json.MarshalIndent(configs, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to marshal configs: %w", err)
			}

			return []mcp.ResourceContents{
				&mcp.TextResourceContents{
					URI:      request.Params.URI,
					Text:     string(jsonBytes),
					MIMEType: "application/json",
				},
			}, nil
		},
	}
}
