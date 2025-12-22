// Package dashboard provides API endpoints for the local dashboard.
package dashboard

import (
	"github.com/jongio/azd-app/cli/src/internal/azure"
)

// Shared constants used across Azure logs modules
const (
	errMethodNotAllowed    = "Method not allowed"
	errLoadAzureYamlFailed = "Failed to load azure.yaml"
	errSaveAzureYamlFailed = "Failed to save azure.yaml"
	logsTroubleshootURL    = "https://aka.ms/azd/app/logs/troubleshoot"
)

// Shared function variables for dependency injection and testing
var fetchAzureLogsStandalone = azure.FetchAzureLogsStandalone
var newLogAnalyticsCredential = azure.NewLogAnalyticsCredential
var validateCredentials = azure.ValidateCredentials
var getWorkspaceIDFromEnv = azure.GetWorkspaceIDFromEnv
var getOrCreateLogAnalyticsClient = azure.GetOrCreateLogAnalyticsClient
