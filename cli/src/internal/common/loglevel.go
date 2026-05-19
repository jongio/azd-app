package common

import (
	"github.com/jongio/azd-app/cli/src/internal/azure"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// ConvertAzureLogLevelToService converts azure log levels to service log levels.
func ConvertAzureLogLevelToService(level azure.LogLevel) service.LogLevel {
	switch level {
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
