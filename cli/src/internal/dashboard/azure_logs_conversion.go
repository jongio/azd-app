// Package dashboard provides API endpoints for the local dashboard.
package dashboard

import (
	"github.com/jongio/azd-app/cli/src/internal/azure"
	"github.com/jongio/azd-app/cli/src/internal/common"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// convertAzureLogLevel converts azure.LogLevel to service.LogLevel.
func convertAzureLogLevel(azLevel azure.LogLevel) service.LogLevel {
	return common.ConvertAzureLogLevelToService(azLevel)
}
