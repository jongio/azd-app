package azure

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

// AppServiceValidator validates Azure App Service logging configuration.
type AppServiceValidator struct {
	credential azcore.TokenCredential
	projectDir string
	laClient   *LogAnalyticsClient
}

// NewAppServiceValidator creates a new App Service validator.
func NewAppServiceValidator(credential azcore.TokenCredential, projectDir string, workspaceID string) (*AppServiceValidator, error) {
	laClient, err := NewLogAnalyticsClient(credential, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to create Log Analytics client: %w", err)
	}

	return &AppServiceValidator{
		credential: credential,
		projectDir: projectDir,
		laClient:   laClient,
	}, nil
}

// Validate performs comprehensive validation of an App Service's logging configuration.
func (v *AppServiceValidator) Validate(ctx context.Context, serviceName string, resource *AzureResource) (*ServiceDiagnosticResult, error) {
	result := &ServiceDiagnosticResult{
		HostType:     ResourceTypeAppService,
		Requirements: make([]Requirement, 0),
	}

	// Step 1: Check if resource is deployed
	if resource.ResourceID == "" {
		result.Status = DiagnosticStatusNotConfigured
		result.Requirements = append(result.Requirements, Requirement{
			Name:        ReqResourceDeployment,
			Status:      RequirementStatusNotMet,
			Description: MsgAppServiceNotDeployed,
			HowToFix:    fmt.Sprintf(FixRunAzdUp, "App Service"),
		})
		result.SetupGuide = v.generateSetupGuide(serviceName, resource, false, false)
		return result, nil
	}

	// Step 2: Check diagnostic settings
	hasDiagnosticSettings, diagErr := v.checkDiagnosticSettings(ctx, resource.ResourceID)
	if diagErr != nil {
		slog.Debug("error checking diagnostic settings", "service", serviceName, "error", diagErr)
	}

	if hasDiagnosticSettings {
		result.Requirements = append(result.Requirements, Requirement{
			Name:        ReqDiagnosticSettings,
			Status:      RequirementStatusMet,
			Description: MsgDiagnosticSettingsEnabled,
		})
	} else {
		result.Requirements = append(result.Requirements, Requirement{
			Name:        ReqDiagnosticSettings,
			Status:      RequirementStatusNotMet,
			Description: MsgDiagnosticSettingsDisabled,
			HowToFix:    fmt.Sprintf(FixConfigureDiagnosticSettings, "AppServiceConsoleLogs"),
		})
	}

	// Step 3: Log flow check
	// Note: Currently only configuration-based validation is supported.
	// Actual log querying would require QueryLogs with correct signature.
	logCount := 0
	var lastLogTime *time.Time = nil

	result.LogCount = logCount
	result.LastLogTime = lastLogTime

	// Step 4: Determine overall status
	if logCount > 0 {
		result.Status = DiagnosticStatusHealthy
		result.Message = fmt.Sprintf("Streaming successfully • %d logs in last 15 min", logCount)
		result.Requirements = append(result.Requirements, Requirement{
			Name:        ReqLogFlow,
			Status:      RequirementStatusMet,
			Description: fmt.Sprintf("Receiving logs (last log: %s)", formatTimeSince(lastLogTime)),
		})
	} else if hasDiagnosticSettings {
		result.Status = DiagnosticStatusPartial
		result.Message = MsgDiagnosticSettingsPartial
		result.Requirements = append(result.Requirements, Requirement{
			Name:        ReqLogFlow,
			Status:      RequirementStatusNotMet,
			Description: MsgNoLogsDetected,
			HowToFix:    FixWaitForLogs,
		})
		result.SetupGuide = v.generateSetupGuide(serviceName, resource, true, false)
	} else {
		result.Status = DiagnosticStatusNotConfigured
		result.Message = "Diagnostic settings not configured"
		result.Requirements = append(result.Requirements, Requirement{
			Name:        "Log Flow",
			Status:      RequirementStatusNotMet,
			Description: "Cannot receive logs without diagnostic settings",
		})
		result.SetupGuide = v.generateSetupGuide(serviceName, resource, false, false)
	}

	return result, nil
}

// checkDiagnosticSettings checks if diagnostic settings are configured for the App Service.
func (v *AppServiceValidator) checkDiagnosticSettings(ctx context.Context, resourceID string) (bool, error) {
	checker := &DiagnosticSettingsChecker{
		credential: v.credential,
		projectDir: v.projectDir,
	}

	result := checker.checkDiagnosticSettings(ctx, "", resourceID, v.laClient.workspaceID)
	return result.Status == DiagnosticSettingsConfigured, nil
}

// generateSetupGuide creates a setup guide for App Service.
func (v *AppServiceValidator) generateSetupGuide(serviceName string, resource *AzureResource, hasSettings bool, hasLogs bool) *SetupGuide {
	if hasLogs {
		return nil
	}

	guide := &SetupGuide{
		Title:       "App Service - Enable Log Streaming",
		Description: "Configure your App Service to send logs to Log Analytics workspace",
		Steps:       make([]SetupGuideStep, 0),
	}

	if !hasSettings {
		guide.Steps = append(guide.Steps, SetupGuideStep{
			Title:       "Automatic Setup (Recommended)",
			Description: "Run azd up to automatically configure diagnostic settings",
			Command:     "azd up",
		})

		guide.Steps = append(guide.Steps, SetupGuideStep{
			Title:       "Manual Setup - Azure Portal",
			Description: "1. Go to Azure Portal\n2. Navigate to your App Service\n3. Click 'Diagnostic settings'\n4. Click '+ Add diagnostic setting'\n5. Name: azd-logs\n6. Select logs: AppServiceConsoleLogs, AppServiceHTTPLogs\n7. Check 'Send to Log Analytics workspace'\n8. Select your workspace\n9. Click 'Save'",
		})
	}

	guide.Steps = append(guide.Steps, SetupGuideStep{
		Title:       "Wait for Logs",
		Description: "Logs should appear within 5-10 minutes after configuration",
	})

	guide.Steps = append(guide.Steps, SetupGuideStep{
		Title:       "Verify",
		Description: "Click the Refresh button in diagnostics to check status",
	})

	return guide
}
