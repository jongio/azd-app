package healthcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/service"

	"github.com/jongio/azd-core/procutil"
)

func (c *HealthChecker) buildResultFromHTTPCheck(result HealthCheckResult, httpResult *httpHealthCheckResult, port int, isInStartupGracePeriod bool) HealthCheckResult {
	result.CheckType = HealthCheckTypeHTTP
	result.Endpoint = httpResult.Endpoint
	result.ResponseTime = httpResult.ResponseTime
	result.StatusCode = httpResult.StatusCode
	result.Status = httpResult.Status
	result.Details = httpResult.Details
	result.Error = httpResult.Error
	// Store detailed error information separately if available
	if httpResult.Error != "" && len(httpResult.Error) > 100 {
		result.ErrorDetails = httpResult.Error
		result.Error = httpResult.Error[:100] + "..." // Truncate main error field
	}
	if port > 0 {
		result.Port = port
	}
	// If check failed but we're in startup grace period, keep "starting" status
	if isInStartupGracePeriod && result.Status != HealthStatusHealthy {
		result.Status = HealthStatusStarting
	}
	return result
}

// tryCustomHealthCheck performs a health check using custom configuration from azure.yaml.
// For container services (svc.Type == "container"), CMD and CMD-SHELL health checks
// are executed inside the container using docker exec.

func (c *HealthChecker) performProcessHealthCheck(_ context.Context, svc serviceInfo, isInStartupGracePeriod bool) HealthCheckResult {
	result := HealthCheckResult{
		ServiceName: svc.Name,
		Timestamp:   time.Now(),
		CheckType:   HealthCheckTypeProcess,
		ServiceMode: svc.Mode,
	}

	if !svc.StartTime.IsZero() {
		if !svc.EndTime.IsZero() {
			result.Uptime = svc.EndTime.Sub(svc.StartTime)
		} else {
			result.Uptime = time.Since(svc.StartTime)
		}
	}

	if svc.Mode == ServiceModeBuild || svc.Mode == ServiceModeTask {
		return c.performBuildTaskHealthCheck(svc, isInStartupGracePeriod, result)
	}

	if svc.HealthCheck != nil && svc.HealthCheck.Type == "output" && svc.HealthCheck.Pattern != "" {
		return c.performOutputHealthCheck(svc, isInStartupGracePeriod, result)
	}

	if svc.PID > 0 {
		result.PID = svc.PID
		if result.Details == nil {
			result.Details = make(map[string]any)
		}

		isRunning := isProcessRunning(svc.PID)
		if isRunning {
			result.Status = HealthStatusHealthy
			result.Details["pid"] = svc.PID
		} else {
			if isInStartupGracePeriod {
				result.Status = HealthStatusStarting
			} else {
				result.Status = HealthStatusUnhealthy
			}
			result.Error = fmt.Sprintf("process %d not running", svc.PID)
			// Add actionable suggestion
			result.Details["suggestion"] = suggestProcessErrorAction(svc.PID, isRunning, svc.Mode)
			result.Details["pid"] = svc.PID
		}
		return result
	}

	if isInStartupGracePeriod {
		result.Status = HealthStatusStarting
	} else {
		result.Status = HealthStatusUnknown
	}
	result.Error = "no process ID available for health check"

	return result
}

// performBuildTaskHealthCheck handles health checks for build and task mode services.
func (c *HealthChecker) performBuildTaskHealthCheck(svc serviceInfo, isInStartupGracePeriod bool, result HealthCheckResult) HealthCheckResult {
	result.PID = svc.PID

	if svc.PID > 0 && isProcessRunning(svc.PID) {
		if isInStartupGracePeriod {
			result.Status = HealthStatusStarting
		} else {
			result.Status = HealthStatusHealthy
		}
		if svc.Mode == ServiceModeBuild {
			result.Details = map[string]any{"state": "building"}
		} else {
			result.Details = map[string]any{"state": "running"}
		}
		return result
	}

	if svc.ExitCode != nil {
		if *svc.ExitCode == 0 {
			result.Status = HealthStatusHealthy
			if svc.Mode == ServiceModeBuild {
				result.Details = map[string]any{"state": "built", "exitCode": 0}
			} else {
				result.Details = map[string]any{"state": statusCompleted, "exitCode": 0}
			}
		} else {
			result.Status = HealthStatusUnhealthy
			result.Error = fmt.Sprintf("process exited with code %d", *svc.ExitCode)
			result.Details = map[string]any{"state": "failed", "exitCode": *svc.ExitCode}
		}
		return result
	}

	if svc.PID > 0 {
		result.Status = HealthStatusHealthy
		if svc.Mode == ServiceModeBuild {
			result.Details = map[string]any{"state": "built", "note": "exit code not captured"}
		} else {
			result.Details = map[string]any{"state": statusCompleted, "note": "exit code not captured"}
		}
		return result
	}

	if isInStartupGracePeriod {
		result.Status = HealthStatusStarting
		return result
	}

	result.Status = HealthStatusUnknown
	result.Error = "no process information available"
	return result
}

// performOutputHealthCheck handles health checks for services using output pattern matching.
func (c *HealthChecker) performOutputHealthCheck(svc serviceInfo, isInStartupGracePeriod bool, result HealthCheckResult) HealthCheckResult {
	pattern := svc.HealthCheck.Pattern
	result.PID = svc.PID
	result.Details = map[string]any{
		"checkType": "output",
		"pattern":   pattern,
	}

	if svc.PID > 0 && !isProcessRunning(svc.PID) {
		if svc.ExitCode != nil {
			if *svc.ExitCode == 0 {
				result.Status = HealthStatusHealthy
				result.Details["state"] = statusCompleted
				return result
			}
			result.Status = HealthStatusUnhealthy
			result.Error = fmt.Sprintf("process exited with code %d before pattern matched", *svc.ExitCode)
			result.Details["state"] = "failed"
			return result
		}
		if isInStartupGracePeriod {
			result.Status = HealthStatusStarting
		} else {
			result.Status = HealthStatusUnhealthy
			result.Error = "process not running"
		}
		return result
	}

	projectDir, _ := os.Getwd()
	logManager := service.GetLogManager(projectDir)
	buffer, exists := logManager.GetBuffer(svc.Name)

	if !exists {
		if isInStartupGracePeriod {
			result.Status = HealthStatusStarting
			result.Details["state"] = "waiting_for_logs"
		} else {
			result.Status = HealthStatusUnknown
			result.Error = "log buffer not available"
		}
		return result
	}

	if buffer.ContainsPattern(pattern) {
		result.Status = HealthStatusHealthy
		result.Details["state"] = "pattern_matched"
		return result
	}

	if isInStartupGracePeriod {
		result.Status = HealthStatusStarting
		result.Details["state"] = "waiting_for_pattern"
	} else {
		if svc.Mode == ServiceModeWatch {
			result.Status = HealthStatusHealthy
			result.Details["state"] = "watching"
		} else {
			result.Status = HealthStatusUnhealthy
			result.Error = fmt.Sprintf("pattern %q not found in output", pattern)
			result.Details["state"] = "pattern_not_matched"
			result.Details["suggestion"] = "Check output pattern configuration. Service may still be starting or pattern may be incorrect."
		}
	}

	return result
}

// checkPort checks if a TCP port is listening.
func (c *HealthChecker) checkPort(ctx context.Context, port int) bool {
	address := fmt.Sprintf("localhost:%d", port)
	dialer := net.Dialer{Timeout: defaultPortCheckTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// suggestTCPErrorAction provides actionable suggestions for TCP connection errors.
func suggestTCPErrorAction(err error, port int) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()
	if strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "actively refused") {
		return fmt.Sprintf("Port %d connection refused. Verify service is running and port is correct.", port)
	}
	if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "i/o timeout") {
		return fmt.Sprintf("Port %d connection timeout. Check network connectivity and firewall rules.", port)
	}
	if strings.Contains(errMsg, "no route to host") {
		return "Network unreachable. Check network configuration."
	}
	return fmt.Sprintf("Port %d connection failed. Verify service is running.", port)
}

// suggestProcessErrorAction provides actionable suggestions for process check errors.
func suggestProcessErrorAction(pid int, isRunning bool, _ string) string {
	if !isRunning {
		return fmt.Sprintf("Process %d not running. Check service logs and verify start command.", pid)
	}
	return ""
}

// isProcessRunning delegates to procutil.IsProcessRunning for cross-platform process detection.
func isProcessRunning(pid int) bool {
	return procutil.IsProcessRunning(pid)
}

// suggestHTTPErrorAction provides actionable suggestions based on HTTP status code.
func suggestHTTPErrorAction(statusCode int) string {
	switch statusCode {
	case 503:
		return "Service temporarily unavailable. Check if dependencies are running."
	case 500, 501, 502, 504, 505, 506, 507, 508, 509, 510, 511:
		return "Server error. Check application logs for details."
	case 404:
		return "Health endpoint not found. Verify endpoint configuration."
	case 401:
		return "Authentication failed. Check credentials."
	case 403:
		return "Authorization failed. Check permissions."
	case 429:
		return "Rate limited. Reduce request rate or check quotas."
	case 408:
		return "Request timeout. Check network connectivity and service performance."
	default:
		if statusCode >= 500 && statusCode < 600 {
			return "Server error. Check application logs for details."
		}
		return "HTTP request failed. Check service logs for details."
	}
}

// parseErrorDetailsFromBody attempts to extract error details from HTTP response body.
func parseErrorDetailsFromBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}

	// Try to parse as JSON
	var jsonData map[string]any
	if err := json.Unmarshal(body, &jsonData); err == nil {
		// Look for common error fields
		for _, key := range []string{statusError, "message", "detail", "details", "error_description"} {
			if val, ok := jsonData[key]; ok {
				if str, ok := val.(string); ok && str != "" {
					return str
				}
			}
		}
	}

	// If JSON parsing failed or no error field found, return truncated body (first 200 chars)
	bodyStr := string(body)
	if len(bodyStr) > 200 {
		return bodyStr[:200] + "..."
	}
	return bodyStr
}
