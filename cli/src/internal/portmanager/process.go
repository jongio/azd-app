package portmanager

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// commandTimeout is the maximum time to wait for process detection commands.
// In containerized environments (e.g., Codespaces), lsof/ss can be slow.
const commandTimeout = 5 * time.Second

// osWindows is the GOOS value for Windows.
const osWindows = "windows"

// buildGetProcessOnPortCommand returns the command and args to find a process listening on a port.
// On Linux, prefers 'ss' (faster in containers) with 'lsof' fallback.
func buildGetProcessOnPortCommand(port int) (cmd string, args []string) {
	if runtime.GOOS == osWindows {
		psScript := fmt.Sprintf(`
			$connections = netstat -ano | Select-String ":%d " | Select-String "LISTENING"
			foreach ($line in $connections) {
				$parts = $line -split '\s+' | Where-Object { $_ }
				$procId = $parts[-1]
				if ($procId -match '^\d+$') {
					Write-Output $procId
					exit 0
				}
			}
		`, port)
		return "powershell", []string{"-Command", psScript}
	}
	// Unix: Try ss first (faster in containers/Codespaces), fallback to lsof
	// ss -tlnp is much faster than lsof in containerized environments
	// The command tries ss first, and if it fails or returns empty, falls back to lsof
	script := fmt.Sprintf(`
pid=$(ss -tlnp 2>/dev/null | grep ':%d ' | sed -n 's/.*pid=\([0-9]*\).*/\1/p' | head -n1)
if [ -n "$pid" ]; then
    echo "$pid"
    exit 0
fi
# Fallback to lsof with timeout to prevent hangs
timeout 3 lsof -ti:%d 2>/dev/null | head -n 1
`, port, port)
	return "sh", []string{"-c", script}
}

// buildGetProcessNameCommand returns the command and args to get a process name by PID.
func buildGetProcessNameCommand(pid int) (cmd string, args []string) {
	if runtime.GOOS == osWindows {
		psScript := fmt.Sprintf(`
			$proc = Get-Process -Id %d -ErrorAction SilentlyContinue
			if ($proc) {
				Write-Output $proc.ProcessName
			}
		`, pid)
		return "powershell", []string{"-Command", psScript}
	}
	return "sh", []string{"-c", fmt.Sprintf("ps -p %d -o comm=", pid)}
}

// buildKillProcessCommand returns the command and args to kill a process and its children by PID.
// On Windows, uses Get-CimInstance Win32_Process to find child processes by ParentProcessId.
// On Unix, uses pkill -P to kill children first.
// In both cases, children are killed first (recursively), then the parent.
func buildKillProcessCommand(pid int) (cmd string, args []string) {
	if runtime.GOOS == osWindows {
		// PowerShell script that recursively kills child processes first, then the parent.
		// Uses Get-CimInstance Win32_Process to find children by ParentProcessId.
		psScript := fmt.Sprintf(`
			function Kill-ProcessTree {
				param([int]$ParentId)
				$children = Get-CimInstance Win32_Process -Filter "ParentProcessId = $ParentId" -ErrorAction SilentlyContinue
				foreach ($child in $children) {
					Kill-ProcessTree -ParentId $child.ProcessId
				}
				Stop-Process -Id $ParentId -Force -ErrorAction SilentlyContinue
			}
			Kill-ProcessTree -ParentId %d
		`, pid)
		return "powershell", []string{"-Command", psScript}
	}
	// Unix: Kill children first, then parent. Use timeout to prevent hangs.
	// The pkill command may not exist in all environments, so we try multiple approaches.
	script := fmt.Sprintf(`
# Kill child processes first
for child in $(pgrep -P %d 2>/dev/null); do
    kill -TERM "$child" 2>/dev/null
done
sleep 0.1
# Force kill any remaining children
for child in $(pgrep -P %d 2>/dev/null); do
    kill -9 "$child" 2>/dev/null
done
# Kill the parent process
kill -TERM %d 2>/dev/null
sleep 0.1
kill -9 %d 2>/dev/null || true
`, pid, pid, pid, pid)
	return "sh", []string{"-c", script}
}

// getProcessInfoOnPort retrieves the PID and name of the process listening on the specified port.
func (pm *PortManager) getProcessInfoOnPort(port int) (*ProcessInfo, error) {
	pid, err := pm.getProcessOnPort(port)
	if err != nil {
		return nil, err
	}

	name, _ := pm.getProcessName(pid) // Ignore error, we'll use PID only if name lookup fails
	return &ProcessInfo{PID: pid, Name: name}, nil
}

// getProcessOnPort retrieves the PID of the process listening on the specified port.
func (pm *PortManager) getProcessOnPort(port int) (int, error) {
	if port <= 0 || port > 65535 {
		return 0, fmt.Errorf("invalid port number: %d (must be 1-65535)", port)
	}

	cmd, args := buildGetProcessOnPortCommand(port)

	// Execute command with timeout to prevent hangs in containerized environments
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	// #nosec G204 -- cmd is either "powershell" or "sh" (hard-coded), port is validated int
	execCmd := exec.CommandContext(ctx, cmd, args...)
	// Don't inherit stdin - prevents blocking in non-interactive environments
	execCmd.Stdin = nil
	output, err := execCmd.Output()
	if err != nil {
		// Check if it was a timeout
		if ctx.Err() == context.DeadlineExceeded {
			return 0, fmt.Errorf("timed out getting process on port %d (this can happen in Codespaces)", port)
		}
		return 0, fmt.Errorf("failed to get process on port %d: %w", port, err)
	}

	pidStr := strings.TrimSpace(string(output))
	if pidStr == "" {
		return 0, fmt.Errorf("no process found on port %d", port)
	}

	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("invalid PID '%s': %w", pidStr, err)
	}

	return pid, nil
}

// getProcessName retrieves the process name for a given PID.
func (pm *PortManager) getProcessName(pid int) (string, error) {
	cmd, args := buildGetProcessNameCommand(pid)

	// Execute command with timeout to prevent hangs
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	// #nosec G204 -- cmd is either "powershell" or "sh" (hard-coded), pid is validated int
	execCmd := exec.CommandContext(ctx, cmd, args...)
	// Don't inherit stdin - prevents blocking in non-interactive environments
	execCmd.Stdin = nil
	output, err := execCmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("timed out getting process name for PID %d", pid)
		}
		return "", fmt.Errorf("failed to get process name for PID %d: %w", pid, err)
	}

	name := strings.TrimSpace(string(output))
	if name == "" {
		return "", fmt.Errorf("no process name found for PID %d", pid)
	}

	return name, nil
}

// KillProcessOnPort kills any process listening on the specified port.
// Returns nil if no process was using the port or if the process was successfully killed.
// Returns an error only if the process could not be terminated (e.g., protected system process).
func (pm *PortManager) KillProcessOnPort(port int) error {
	// Get the PID first so we can provide feedback
	pid, err := pm.getProcessOnPort(port)
	if err != nil {
		// Port might not be in use anymore
		return nil
	}

	// Log without exposing too much system info to prevent information disclosure
	slog.Info("terminating process on port", "port", port, "pid", pid)

	cmd, args := buildKillProcessCommand(pid)

	// Execute the kill command with timeout and without stdin inheritance
	// Using exec.CommandContext directly instead of executor.RunCommand to avoid
	// stdin inheritance which can cause hangs in Codespaces/containers
	ctx, cancel := context.WithTimeout(context.Background(), killProcessTimeout)
	defer cancel()

	// #nosec G204 -- Command injection safe: cmd is hard-coded ("powershell" or "sh"),
	// and PID is validated integer from strconv.Atoi in getProcessOnPort (no user input)
	execCmd := exec.CommandContext(ctx, cmd, args...)
	execCmd.Stdin = nil // Don't inherit stdin - prevents blocking
	if err := execCmd.Run(); err != nil {
		// Log error but don't fail - process might have already exited
		slog.Debug("kill command completed with error", "pid", pid, "error", err)
	}

	// Wait a moment for process to die
	time.Sleep(processCleanupWait)

	// Verify the process was actually killed
	// This is critical for protected/system processes that cannot be terminated
	if stillRunningPid, err := pm.getProcessOnPort(port); err == nil && stillRunningPid == pid {
		processName, _ := pm.getProcessName(pid)
		slog.Warn("process could not be terminated - likely a protected system process",
			"port", port, "pid", pid, "name", processName)
		return fmt.Errorf("process %d (%s) could not be terminated - it may be a protected system process or require administrator privileges",
			pid, processName)
	}

	return nil
}

// killProcessOnPort is an alias for KillProcessOnPort for internal use and test compatibility.
func (pm *PortManager) killProcessOnPort(port int) error {
	return pm.KillProcessOnPort(port)
}
