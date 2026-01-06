// Package dashboard provides API endpoints for the local dashboard.
package dashboard

// Azure setup status constants
const (
	// Workspace/Service status values
	StatusNotDeployed   = "not-deployed"
	StatusNotConfigured = "not-configured"
	StatusConfigured    = "configured"
	StatusReady         = "ready"
	StatusPartial       = "partial"
	StatusMissing       = "missing"
	StatusError         = "error"
	StatusComplete      = "complete"
	StatusIncomplete    = "incomplete"

	// Auth status values
	StatusAuthenticated    = "authenticated"
	StatusUnauthenticated  = "unauthenticated"
	StatusPermissionDenied = "permission-denied"

	// Setup categories
	CategoryWorkspace          = "workspace"
	CategoryAuth               = "auth"
	CategoryDiagnosticSettings = "diagnostic-settings"
	CategoryConfig             = "config"

	// Common messages
	MsgWorkspaceNotConfigured = "Log Analytics workspace not configured"
	MsgAzureCredsNotAvailable = "Azure credentials not available. Run 'azd auth login' to authenticate."
	MsgAuthFailed             = "Authentication failed or expired. Run 'azd auth login' to re-authenticate."

	// Setup steps
	StepWorkspace          = "workspace"
	StepAuthentication     = "authentication"
	StepDiagnosticSettings = "diagnostic-settings"
	StepVerification       = "verification"
)
