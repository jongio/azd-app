package azure

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

// FunctionValidator validates Azure Functions logging configuration.
type FunctionValidator struct {
	credential azcore.TokenCredential
	projectDir string
	laClient   *LogAnalyticsClient
}

// NewFunctionValidator creates a new Function validator.
func NewFunctionValidator(credential azcore.TokenCredential, projectDir string, workspaceID string) (*FunctionValidator, error) {
	laClient, err := NewLogAnalyticsClient(credential, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to create Log Analytics client: %w", err)
	}

	return &FunctionValidator{
		credential: credential,
		projectDir: projectDir,
		laClient:   laClient,
	}, nil
}

// Validate performs comprehensive validation of a Function App's logging configuration.
func (v *FunctionValidator) Validate(ctx context.Context, serviceName string, resource *AzureResource) (*ServiceDiagnosticResult, error) {
	result := &ServiceDiagnosticResult{
		HostType:     ResourceTypeFunction,
		Requirements: make([]Requirement, 0),
	}

	// Step 1: Check if resource is deployed
	if resource.ResourceID == "" {
		result.Status = DiagnosticStatusNotConfigured
		result.Requirements = append(result.Requirements, Requirement{
			Name:        ReqResourceDeployment,
			Status:      RequirementStatusNotMet,
			Description: MsgFunctionAppNotDeployed,
			HowToFix:    fmt.Sprintf(FixRunAzdUp, "Function App"),
		})
		result.SetupGuide = v.generateSetupGuide(serviceName, resource, false, false)
		return result, nil
	}

	// Step 2: Check Application Insights connection string
	hasAppInsights, aiErr := v.checkApplicationInsights(ctx, serviceName)
	if aiErr != nil {
		slog.Debug("error checking app insights", "service", serviceName, "error", aiErr)
	}

	if hasAppInsights {
		result.Requirements = append(result.Requirements, Requirement{
			Name:        "Application Insights",
			Status:      RequirementStatusMet,
			Description: "Connection string configured",
		})
	} else {
		result.Requirements = append(result.Requirements, Requirement{
			Name:        "Application Insights",
			Status:      RequirementStatusNotMet,
			Description: "APPLICATIONINSIGHTS_CONNECTION_STRING not configured",
			HowToFix:    "Add Application Insights connection string to function environment variables",
		})
	}

	// Step 3: Check diagnostic settings (optional for functions)
	hasDiagnosticSettings, diagErr := v.checkDiagnosticSettings(ctx, resource.ResourceID)
	if diagErr != nil {
		slog.Debug("error checking diagnostic settings", "service", serviceName, "error", diagErr)
	}

	if hasDiagnosticSettings {
		result.Requirements = append(result.Requirements, Requirement{
			Name:        "Diagnostic Settings (Optional)",
			Status:      RequirementStatusMet,
			Description: "Configured to send FunctionAppLogs to Log Analytics",
		})
	} else {
		result.Requirements = append(result.Requirements, Requirement{
			Name:        "Diagnostic Settings (Optional)",
			Status:      RequirementStatusNotMet,
			Description: "Can improve log collection performance",
			HowToFix:    "Configure diagnostic settings for FunctionAppLogs",
		})
	}

	// Step 4: Log flow check
	// Note: Currently only configuration-based validation is supported.
	// Actual log querying would require QueryLogs with correct signature.
	logCount := 0
	var lastLogTime *time.Time = nil

	result.LogCount = logCount
	result.LastLogTime = lastLogTime

	// Step 5: Determine overall status
	if logCount > 0 {
		result.Status = DiagnosticStatusHealthy
		result.Message = fmt.Sprintf("Streaming successfully • %d logs in last 15 min", logCount)
		result.Requirements = append(result.Requirements, Requirement{
			Name:        ReqLogFlow,
			Status:      RequirementStatusMet,
			Description: fmt.Sprintf("Receiving logs (last log: %s)", formatTimeSince(lastLogTime)),
		})
	} else if hasAppInsights {
		result.Status = DiagnosticStatusPartial
		result.Message = "Application Insights configured but no logs detected yet"
		result.Requirements = append(result.Requirements, Requirement{
			Name:        ReqLogFlow,
			Status:      RequirementStatusNotMet,
			Description: MsgNoLogsDetected,
			HowToFix:    "Wait 5-10 minutes for logs to propagate, or trigger a function execution",
		})
		result.SetupGuide = v.generateSetupGuide(serviceName, resource, true, false)
	} else {
		result.Status = DiagnosticStatusNotConfigured
		result.Message = "Application Insights not configured"
		result.SetupGuide = v.generateSetupGuide(serviceName, resource, false, false)
	}

	return result, nil
}

// checkApplicationInsights checks if Application Insights is configured in the environment.
func (v *FunctionValidator) checkApplicationInsights(ctx context.Context, serviceName string) (bool, error) {
	// Run azd env get-values to check for APPLICATIONINSIGHTS_CONNECTION_STRING
	cmd := exec.CommandContext(ctx, "azd", "env", "get-values")
	if v.projectDir != "" {
		cmd.Dir = v.projectDir
	}

	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	// Check if the output contains APPLICATIONINSIGHTS_CONNECTION_STRING
	return strings.Contains(string(output), "APPLICATIONINSIGHTS_CONNECTION_STRING"), nil
}

// checkDiagnosticSettings checks if diagnostic settings are configured for the Function App.
func (v *FunctionValidator) checkDiagnosticSettings(ctx context.Context, resourceID string) (bool, error) {
	checker := &DiagnosticSettingsChecker{
		credential: v.credential,
		projectDir: v.projectDir,
	}

	result := checker.checkDiagnosticSettings(ctx, "", resourceID, v.laClient.workspaceID)
	return result.Status == DiagnosticSettingsConfigured, nil
}

// generateSetupGuide creates a setup guide for Azure Functions.
func (v *FunctionValidator) generateSetupGuide(serviceName string, resource *AzureResource, hasAppInsights bool, hasLogs bool) *SetupGuide {
	if hasLogs {
		return nil
	}

	guide := &SetupGuide{
		Title:       "Azure Functions - Enable Log Streaming",
		Description: "Configure Application Insights for your Function App",
		Steps:       make([]SetupGuideStep, 0),
	}

	if !hasAppInsights {
		guide.Steps = append(guide.Steps, SetupGuideStep{
			Title:       "Add Application Insights to azure.yaml",
			Description: "Configure the connection string in your service environment",
			Code: fmt.Sprintf(`services:
  %s:
    environment:
      APPLICATIONINSIGHTS_CONNECTION_STRING: ${APPLICATIONINSIGHTS_CONNECTION_STRING}`, serviceName),
			CodeLang: "yaml",
		})

		guide.Steps = append(guide.Steps, SetupGuideStep{
			Title:       "Deploy the change",
			Description: "Deploy your function with the updated configuration",
			Command:     fmt.Sprintf("azd deploy %s", serviceName),
		})
	}

	guide.Steps = append(guide.Steps, SetupGuideStep{
		Title:       "Wait for logs",
		Description: "Logs should appear within 5-10 minutes after deployment",
	})

	guide.Steps = append(guide.Steps, SetupGuideStep{
		Title:       "Trigger a function",
		Description: "Execute your function to generate logs (via HTTP trigger or timer)",
	})

	guide.Steps = append(guide.Steps, SetupGuideStep{
		Title:       "Verify",
		Description: "Click Refresh in diagnostics to check for logs",
	})

	return guide
}
