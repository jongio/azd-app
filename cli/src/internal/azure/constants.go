// Package azure provides Azure resource integration.
package azure

// Diagnostic requirement names
const (
	ReqResourceDeployment = "Resource Deployment"
	ReqDiagnosticSettings = "Diagnostic Settings"
	ReqLogFlow            = "Log Flow"
)

// Common diagnostic messages
const (
	MsgAppServiceNotDeployed      = "App Service not deployed to Azure"
	MsgContainerAppNotDeployed    = "Container App not deployed to Azure"
	MsgFunctionAppNotDeployed     = "Function App not deployed to Azure"
	MsgDiagnosticSettingsEnabled  = "Configured to send logs to Log Analytics workspace"
	MsgDiagnosticSettingsDisabled = "Not configured to send logs to Log Analytics"
	MsgNoLogsDetected             = "No logs detected in the last 15 minutes"
	MsgDiagnosticSettingsPartial  = "Diagnostic settings configured but no logs detected yet"
)

// How-to-fix messages
const (
	FixRunAzdUp                    = "Run 'azd up' to deploy your %s"
	FixConfigureDiagnosticSettings = "Configure diagnostic settings to send %s to Log Analytics"
	FixWaitForLogs                 = "Wait 5-10 minutes for logs to propagate, or ensure your app is generating logs"
)
