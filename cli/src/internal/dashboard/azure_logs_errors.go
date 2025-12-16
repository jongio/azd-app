// Package dashboard provides API endpoints for the local dashboard.
package dashboard

import (
	"github.com/jongio/azd-app/cli/src/internal/azure"
)

// Azure Logs error codes
const (
	ErrorCodeAuthExpired  = "AUTH_EXPIRED"
	ErrorCodeAuthRequired = "AUTH_REQUIRED"
	ErrorCodeNotDeployed  = "NOT_DEPLOYED"
	ErrorCodeNoWorkspace  = "NO_WORKSPACE"
	ErrorCodeNoPermission = "NO_PERMISSION"
	ErrorCodeUnknown      = "UNKNOWN"
)

// ErrorInfo provides actionable error information with documentation links.
type ErrorInfo struct {
	Message string `json:"message"` // Human-readable error message
	Code    string `json:"code"`    // Error code: "AUTH_EXPIRED", "NOT_DEPLOYED", etc.
	Action  string `json:"action"`  // What the user should do
	Command string `json:"command"` // CLI command to run (optional)
	DocsURL string `json:"docsUrl"` // Documentation URL
}

// mapAzureErrorToInfo converts Azure errors to structured ErrorInfo with docs links.
func mapAzureErrorToInfo(err error) *ErrorInfo {
	if azErr, ok := err.(*azure.AzureLogsError); ok {
		info := &ErrorInfo{
			Message: azErr.Message,
			Code:    azErr.Code,
			Action:  azErr.Action,
			Command: azErr.Command,
		}

		// Add documentation URLs based on error code
		switch azErr.Code {
		case ErrorCodeAuthExpired, ErrorCodeAuthRequired:
			info.DocsURL = "https://aka.ms/azd/app/logs/troubleshoot#auth"
		case ErrorCodeNotDeployed:
			info.DocsURL = "https://aka.ms/azd/app/logs/setup"
		case ErrorCodeNoWorkspace:
			info.DocsURL = "https://aka.ms/azd/app/logs/configure"
		case ErrorCodeNoPermission:
			info.DocsURL = "https://aka.ms/azd/app/logs/troubleshoot#permissions"
		default:
			info.DocsURL = logsTroubleshootURL
		}

		return info
	}

	// Generic error
	return &ErrorInfo{
		Message: err.Error(),
		Code:    ErrorCodeUnknown,
		Action:  "Check logs for more details",
		DocsURL: logsTroubleshootURL,
	}
}
