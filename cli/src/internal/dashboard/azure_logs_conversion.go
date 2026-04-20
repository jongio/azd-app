// Package dashboard provides API endpoints for the local dashboard.
package dashboard

import (
	"github.com/jongio/azd-app/cli/src/internal/azure"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// convertAzureLogLevel converts azure.LogLevel to service.LogLevel.
func convertAzureLogLevel(azLevel azure.LogLevel) service.LogLevel {
	switch azLevel {
	case azure.LogLevelInfo:
		return service.LogLevelInfo
	case azure.LogLevelWarn:
		return service.LogLevelWarn
	case azure.LogLevelError:
		return service.LogLevelError
	case azure.LogLevelDebug:
		return service.LogLevelDebug
	default:
		return service.LogLevelInfo
	}
}
