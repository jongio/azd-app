package commands

import (
	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// showServiceLogs displays the last N lines of logs for a service that failed.
func showServiceLogs(serviceName, projectDir string, maxLines int) {
	logManager := service.GetLogManager(projectDir)
	buffer, exists := logManager.GetBuffer(serviceName)
	if !exists || buffer == nil {
		return
	}

	entries := buffer.GetRecent(maxLines)
	if len(entries) == 0 {
		return
	}

	output.Newline()
	output.Error("❌ Service '%s' failed. Last %d log lines:", serviceName, len(entries))
	output.Newline()
	
	for _, entry := range entries {
		timestamp := entry.Timestamp.Format("15:04:05")
		if entry.IsStderr {
			output.Item("   [%s] %s%s%s", timestamp, output.Red, entry.Message, output.Reset)
		} else {
			output.Item("   [%s] %s", timestamp, entry.Message)
		}
	}
	output.Newline()
}
