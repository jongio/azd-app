package portmanager

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/executor"
)

// PortAssignment represents a port assignment for a service.
type PortAssignment struct {
	ServiceName string    `json:"serviceName"`
	Port        int       `json:"port"`
	LastUsed    time.Time `json:"lastUsed"`
}

// PortManager manages port assignments for services.
type PortManager struct {
	mu          sync.RWMutex
	assignments map[string]*PortAssignment // key: serviceName
	filePath    string
	portRange   struct {
		start int
		end   int
	}
	// portChecker is a function that checks if a port is available
	// This can be overridden in tests to avoid network binding
	portChecker func(port int) bool
}

var (
	managerCache   = make(map[string]*PortManager)
	managerCacheMu sync.RWMutex
)

// GetPortManager returns the port manager instance for the given project directory.
func GetPortManager(projectDir string) *PortManager {
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			projectDir = "."
		} else {
			projectDir = cwd
		}
	}

	// Normalize path
	absPath, err := filepath.Abs(projectDir)
	if err != nil {
		absPath = projectDir
	}

	if os.Getenv("AZD_APP_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "[DEBUG] GetPortManager called with: %s\n", projectDir)
		fmt.Fprintf(os.Stderr, "[DEBUG] Normalized to: %s\n", absPath)
	}

	managerCacheMu.Lock()
	defer managerCacheMu.Unlock()

	if mgr, exists := managerCache[absPath]; exists {
		if os.Getenv("AZD_APP_DEBUG") == "true" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Returning cached port manager for: %s\n", absPath)
		}
		return mgr
	}

	if os.Getenv("AZD_APP_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Creating NEW port manager for: %s\n", absPath)
	}

	portsDir := filepath.Join(absPath, ".azure")
	portsFile := filepath.Join(portsDir, "ports.json")

	manager := &PortManager{
		assignments: make(map[string]*PortAssignment),
		filePath:    portsFile,
	}
	manager.portRange.start = PortRangeStart
	manager.portRange.end = PortRangeEnd // Allow full dynamic port range

	// Set default port checker (can be overridden in tests)
	manager.portChecker = manager.defaultIsPortAvailable

	// Ensure directory exists
	if err := os.MkdirAll(portsDir, 0750); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to create ports directory: %v\n", err)
	}

	// Load existing assignments
	if err := manager.load(); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load port assignments: %v\n", err)
	}

	managerCache[absPath] = manager
	return manager
}

// AssignPort assigns or retrieves a port for a service.
//
// Parameters:
//   - serviceName: The unique name of the service (must not be empty)
//   - preferredPort: The desired port number. For explicit ports, must be 1-65535.
//   - isExplicit: If true, the port came from azure.yaml config and MUST be used (never changed).
//     If the explicit port is unavailable, the user will be prompted to either kill the
//     existing process or choose a different port.
//   - cleanStale: If true, will prompt user before killing processes on assigned ports.
//
// Returns:
//   - port: The assigned port number (guaranteed to be in the valid range 3000-65535)
//   - wasAutoAssigned: True if the user was prompted and chose to auto-assign a different port.
//     This signals that azure.yaml should be updated with the new port.
//   - error: Non-nil if the assignment failed (validation error, user cancelled, no ports available)
//
// Port range: 3000-65535
//   - Minimum 3000: Avoids well-known ports (0-1023) and registered ports (1024-2999)
//     which often require admin privileges and conflict with system services.
//   - Maximum 65535: Standard TCP/IP limit for port numbers.
//
// Behavior:
//   - EXPLICIT MODE (isExplicit=true): Port is mandatory. If unavailable, prompts user.
//   - FLEXIBLE MODE (isExplicit=false): Port is preferred but can be changed automatically.
//     If unavailable, finds an alternative port without user interaction.
//
// The assigned port is persisted to .azure/ports.json for consistency across runs.
func (pm *PortManager) AssignPort(serviceName string, preferredPort int, isExplicit bool, cleanStale bool) (int, bool, error) {
	// Validate inputs
	if serviceName == "" {
		return 0, false, fmt.Errorf("serviceName cannot be empty")
	}
	if isExplicit && (preferredPort <= 0 || preferredPort > 65535) {
		return 0, false, fmt.Errorf("explicit port must be between 1-65535, got %d", preferredPort)
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// EXPLICIT PORT MODE: Port from azure.yaml - MUST be used, prompt if in use
	if isExplicit {
		// Validate port is in range
		if preferredPort < pm.portRange.start || preferredPort > pm.portRange.end {
			return 0, false, fmt.Errorf("explicit port %d for service '%s' is outside valid range %d-%d",
				preferredPort, serviceName, pm.portRange.start, pm.portRange.end)
		}

		// Check if port is available
		if pm.isPortAvailable(preferredPort) {
			pm.assignments[serviceName] = &PortAssignment{
				ServiceName: serviceName,
				Port:        preferredPort,
				LastUsed:    time.Now(),
			}
			if err := pm.save(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save port assignment: %v\n", err)
			}
			return preferredPort, false, nil
		}

		// Port is in use - prompt user with options
		// Try to get process info to help user
		processInfo := ""
		if info, err := pm.getProcessInfoOnPort(preferredPort); err == nil {
			if info.Name != "" {
				processInfo = fmt.Sprintf(" by %s (PID %d)", info.Name, info.PID)
			} else {
				processInfo = fmt.Sprintf(" by PID %d", info.PID)
			}
		}

		fmt.Fprintf(os.Stderr, "\n⚠️  Service '%s' requires port %d (configured in azure.yaml)\n", serviceName, preferredPort)
		fmt.Fprintf(os.Stderr, "This port is currently in use%s.\n\n", processInfo)
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  1) Kill the process using port %d\n", preferredPort)
		fmt.Fprintf(os.Stderr, "  2) Assign a different port automatically\n")
		fmt.Fprintf(os.Stderr, "  3) Cancel\n\n")
		fmt.Fprintf(os.Stderr, "Choose (1/2/3): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return 0, false, fmt.Errorf("failed to read user input: %w", err)
		}

		response = strings.TrimSpace(response)
		switch response {
		case "1":
			// Kill process
			if err := pm.killProcessOnPort(preferredPort); err != nil {
				return 0, false, fmt.Errorf("failed to free port %d: %w", preferredPort, err)
			}

			// Verify port is now available
			if !pm.isPortAvailable(preferredPort) {
				return 0, false, fmt.Errorf("port %d is still in use after cleanup attempt", preferredPort)
			}

			pm.assignments[serviceName] = &PortAssignment{
				ServiceName: serviceName,
				Port:        preferredPort,
				LastUsed:    time.Now(),
			}
			if err := pm.save(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save port assignment: %v\n", err)
			}
			return preferredPort, false, nil

		case "2":
			// Assign random port
			fmt.Fprintf(os.Stderr, "\nFinding available port for '%s'...\n", serviceName)
			port, err := pm.findAvailablePort()
			if err != nil {
				return 0, false, err
			}

			pm.assignments[serviceName] = &PortAssignment{
				ServiceName: serviceName,
				Port:        port,
				LastUsed:    time.Now(),
			}
			if err := pm.save(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save port assignment: %v\n", err)
			}

			fmt.Fprintf(os.Stderr, "\n✓ Assigned port %d to service '%s'\n", port, serviceName)
			fmt.Fprintf(os.Stderr, "\n⚠️  IMPORTANT: Update your application code to use port %d\n", port)
			fmt.Fprintf(os.Stderr, "Would you like to update azure.yaml to use port %d for future runs? (y/N): ", port)

			updateResponse, err := reader.ReadString('\n')
			if err == nil {
				updateResponse = strings.TrimSpace(strings.ToLower(updateResponse))
				if updateResponse == "y" || updateResponse == "yes" {
					return port, true, nil // Signal caller to update azure.yaml
				}
			}

			return port, false, nil

		default:
			return 0, false, fmt.Errorf("operation cancelled by user")
		}
	}

	// FLEXIBLE PORT MODE: Port can be changed if needed, prompt user when conflicts detected

	// Check if we already have an assignment
	if assignment, exists := pm.assignments[serviceName]; exists {
		assignment.LastUsed = time.Now()

		if os.Getenv("AZD_APP_DEBUG") == "true" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Service '%s': Checking assigned port %d...\n", serviceName, assignment.Port)
		}

		// Check if assigned port is available
		if pm.isPortAvailable(assignment.Port) {
			if os.Getenv("AZD_APP_DEBUG") == "true" {
				fmt.Fprintf(os.Stderr, "[DEBUG] Service '%s': Port %d is available\n", serviceName, assignment.Port)
			}
			if err := pm.save(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save port assignment: %v\n", err)
			}
			return assignment.Port, false, nil
		}

		// Previously assigned port is now in use - prompt user
		if os.Getenv("AZD_APP_DEBUG") == "true" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Service '%s': Port %d is NOT available (in use)\n", serviceName, assignment.Port)
		}

		// Try to get process info to help user
		processInfo := ""
		if info, err := pm.getProcessInfoOnPort(assignment.Port); err == nil {
			if info.Name != "" {
				processInfo = fmt.Sprintf(" (PID %d: %s)", info.PID, info.Name)
			} else {
				processInfo = fmt.Sprintf(" (PID %d)", info.PID)
			}
		}

		fmt.Fprintf(os.Stderr, "\n⚠️  Service '%s' port %d is already in use%s\n", serviceName, assignment.Port, processInfo)
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  1) Kill the process using port %d\n", assignment.Port)
		fmt.Fprintf(os.Stderr, "  2) Assign a different port automatically\n")
		fmt.Fprintf(os.Stderr, "  3) Cancel\n\n")
		fmt.Fprintf(os.Stderr, "Choose (1/2/3): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return 0, false, fmt.Errorf("failed to read user input: %w", err)
		}

		response = strings.TrimSpace(response)
		switch response {
		case "1":
			// Kill process
			if err := pm.killProcessOnPort(assignment.Port); err != nil {
				return 0, false, fmt.Errorf("failed to free port %d: %w", assignment.Port, err)
			}

			// Verify port is now available
			if !pm.isPortAvailable(assignment.Port) {
				return 0, false, fmt.Errorf("port %d is still in use after cleanup attempt", assignment.Port)
			}

			// Keep the same port assignment
			if err := pm.save(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save port assignment: %v\n", err)
			}
			fmt.Fprintf(os.Stderr, "✓ Port %d freed and ready for service '%s'\n\n", assignment.Port, serviceName)
			return assignment.Port, false, nil

		case "2":
			// Find alternative port
			fmt.Fprintf(os.Stderr, "\nFinding available port for '%s'...\n", serviceName)
			port, err := pm.findAvailablePort()
			if err != nil {
				return 0, false, err
			}

			pm.assignments[serviceName] = &PortAssignment{
				ServiceName: serviceName,
				Port:        port,
				LastUsed:    time.Now(),
			}
			if err := pm.save(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save port assignment: %v\n", err)
			}

			fmt.Fprintf(os.Stderr, "✓ Assigned port %d to service '%s'\n\n", port, serviceName)
			return port, false, nil

		default:
			return 0, false, fmt.Errorf("operation cancelled by user")
		}
	}

	// Try preferred port first (if provided)
	if preferredPort >= pm.portRange.start && preferredPort <= pm.portRange.end {
		if os.Getenv("AZD_APP_DEBUG") == "true" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Service '%s': Checking preferred port %d...\n", serviceName, preferredPort)
		}

		if pm.isPortAvailable(preferredPort) {
			if os.Getenv("AZD_APP_DEBUG") == "true" {
				fmt.Fprintf(os.Stderr, "[DEBUG] Service '%s': Preferred port %d is available\n", serviceName, preferredPort)
			}
			pm.assignments[serviceName] = &PortAssignment{
				ServiceName: serviceName,
				Port:        preferredPort,
				LastUsed:    time.Now(),
			}
			if err := pm.save(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save port assignment: %v\n", err)
			}
			return preferredPort, false, nil
		}

		// Preferred port unavailable - prompt user
		if os.Getenv("AZD_APP_DEBUG") == "true" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Service '%s': Preferred port %d is NOT available (in use)\n", serviceName, preferredPort)
		}

		// Try to get process info to help user
		processInfo := ""
		if info, err := pm.getProcessInfoOnPort(preferredPort); err == nil {
			if info.Name != "" {
				processInfo = fmt.Sprintf(" (PID %d: %s)", info.PID, info.Name)
			} else {
				processInfo = fmt.Sprintf(" (PID %d)", info.PID)
			}
		}

		fmt.Fprintf(os.Stderr, "\n⚠️  Service '%s' preferred port %d is already in use%s\n", serviceName, preferredPort, processInfo)
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  1) Kill the process using port %d\n", preferredPort)
		fmt.Fprintf(os.Stderr, "  2) Assign a different port automatically\n")
		fmt.Fprintf(os.Stderr, "  3) Cancel\n\n")
		fmt.Fprintf(os.Stderr, "Choose (1/2/3): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return 0, false, fmt.Errorf("failed to read user input: %w", err)
		}

		response = strings.TrimSpace(response)
		switch response {
		case "1":
			// Kill process
			if err := pm.killProcessOnPort(preferredPort); err != nil {
				return 0, false, fmt.Errorf("failed to free port %d: %w", preferredPort, err)
			}

			// Verify port is now available
			if !pm.isPortAvailable(preferredPort) {
				return 0, false, fmt.Errorf("port %d is still in use after cleanup attempt", preferredPort)
			}

			pm.assignments[serviceName] = &PortAssignment{
				ServiceName: serviceName,
				Port:        preferredPort,
				LastUsed:    time.Now(),
			}
			if err := pm.save(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save port assignment: %v\n", err)
			}
			fmt.Fprintf(os.Stderr, "✓ Port %d freed and assigned to service '%s'\n\n", preferredPort, serviceName)
			return preferredPort, false, nil

		case "2":
			// Find alternative port (fall through to auto-assign below)
			fmt.Fprintf(os.Stderr, "\nFinding available port for '%s'...\n", serviceName)

		default:
			return 0, false, fmt.Errorf("operation cancelled by user")
		}
	}

	// Find an available port automatically
	port, err := pm.findAvailablePort()
	if err != nil {
		return 0, false, err
	}

	pm.assignments[serviceName] = &PortAssignment{
		ServiceName: serviceName,
		Port:        port,
		LastUsed:    time.Now(),
	}
	if err := pm.save(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save port assignment: %v\n", err)
	}

	// Notify user about auto-assigned port
	fmt.Fprintf(os.Stderr, "✓ Auto-assigned port %d to service '%s'\n\n", port, serviceName)

	return port, false, nil
}

// ReleasePort removes a port assignment.
func (pm *PortManager) ReleasePort(serviceName string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.assignments, serviceName)
	return pm.save()
}

// GetAssignment returns the port assignment for a service.
func (pm *PortManager) GetAssignment(serviceName string) (int, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if assignment, exists := pm.assignments[serviceName]; exists {
		return assignment.Port, true
	}
	return 0, false
}

// CleanStalePorts removes assignments for ports that haven't been used in over StalePortCleanupAge.
func (pm *PortManager) CleanStalePorts() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	cutoff := time.Now().Add(-StalePortCleanupAge)
	for name, assignment := range pm.assignments {
		if assignment.LastUsed.Before(cutoff) {
			delete(pm.assignments, name)
		}
	}
	if err := pm.save(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save port assignments: %v\n", err)
	}
}

// IsPortAvailable checks if a port is available for binding.
// This is a public wrapper around the internal port checking logic.
func (pm *PortManager) IsPortAvailable(port int) bool {
	return pm.isPortAvailable(port)
}

// isPortAvailable checks if a port is available.
func (pm *PortManager) isPortAvailable(port int) bool {
	if pm.portChecker != nil {
		return pm.portChecker(port)
	}
	return pm.defaultIsPortAvailable(port)
}

// defaultIsPortAvailable is the default implementation that actually binds to check port availability.
func (pm *PortManager) defaultIsPortAvailable(port int) bool {
	// Bind to localhost to avoid Windows Firewall prompts
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		if os.Getenv("AZD_APP_DEBUG") == "true" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Port %d bind test failed: %v\n", port, err)
		}
		return false
	}
	if err := listener.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to close listener: %v\n", err)
	}
	if os.Getenv("AZD_APP_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Port %d bind test succeeded (port is available)\n", port)
	}
	return true
}

// findAvailablePort finds an available port in the port range.
func (pm *PortManager) findAvailablePort() (int, error) {
	// Build map of assigned ports to avoid duplicates
	assignedPorts := make(map[int]bool)
	for _, assignment := range pm.assignments {
		assignedPorts[assignment.Port] = true
	}

	// Try to find an unassigned, available port
	for port := pm.portRange.start; port <= pm.portRange.end; port++ {
		if assignedPorts[port] {
			continue
		}
		if pm.isPortAvailable(port) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports in range %d-%d", pm.portRange.start, pm.portRange.end)
}

// getProcessInfo retrieves information about the process using a port.
type ProcessInfo struct {
	PID  int
	Name string
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
	var cmd string
	var args []string

	if runtime.GOOS == "windows" {
		// Windows: use netstat to find PID
		cmd = "powershell"
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
		args = []string{"-Command", psScript}
	} else {
		// Unix: use lsof to find PID
		cmd = "sh"
		args = []string{"-c", fmt.Sprintf("lsof -ti:%d | head -n 1", port)}
	}

	// Execute command to get PID
	// #nosec G204 -- cmd is either "powershell" or "sh" (hard-coded), port is validated int
	output, err := exec.Command(cmd, args...).Output()
	if err != nil {
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
	var cmd string
	var args []string

	if runtime.GOOS == "windows" {
		// Windows: use tasklist to get process name
		cmd = "powershell"
		psScript := fmt.Sprintf(`
			$proc = Get-Process -Id %d -ErrorAction SilentlyContinue
			if ($proc) {
				Write-Output $proc.ProcessName
			}
		`, pid)
		args = []string{"-Command", psScript}
	} else {
		// Unix: use ps to get process name
		cmd = "sh"
		args = []string{"-c", fmt.Sprintf("ps -p %d -o comm=", pid)}
	}

	// Execute command to get process name
	// #nosec G204 -- cmd is either "powershell" or "sh" (hard-coded), pid is validated int
	output, err := exec.Command(cmd, args...).Output()
	if err != nil {
		return "", fmt.Errorf("failed to get process name for PID %d: %w", pid, err)
	}

	name := strings.TrimSpace(string(output))
	if name == "" {
		return "", fmt.Errorf("no process name found for PID %d", pid)
	}

	return name, nil
}

// killProcessOnPort kills any process listening on the specified port.
func (pm *PortManager) killProcessOnPort(port int) error {
	if runtime.GOOS == "windows" {
		return pm.killProcessOnPortWindows(port)
	}
	return pm.killProcessOnPortUnix(port)
}

// killProcessOnPortWindows kills processes on Windows using netstat and Stop-Process.
func (pm *PortManager) killProcessOnPortWindows(port int) error {
	portStr := strconv.Itoa(port)

	// Use PowerShell with proper argument passing (no string interpolation)
	// -Command takes a ScriptBlock that we construct safely
	psScript := `
		param($Port)
		$portPattern = ":$Port "
		$connections = netstat -ano | Select-String $portPattern | Select-String "LISTENING"
		foreach ($line in $connections) {
			$parts = $line -split '\s+' | Where-Object { $_ }
			$procId = $parts[-1]
			if ($procId -match '^\d+$') {
				Write-Host "Killing process $procId on port $Port"
				Stop-Process -Id $procId -Force -ErrorAction SilentlyContinue
			}
		}
	`

	// Execute PowerShell with parameter binding
	ctx, cancel := context.WithTimeout(context.Background(), ProcessKillTimeout)
	defer cancel()

	if err := executor.RunCommand(ctx, "powershell", []string{"-Command", psScript, "-Port", portStr}, "."); err != nil {
		// Ignore errors - port might not be in use
		return nil
	}

	// Verify port is actually freed with retry loop
	return pm.verifyPortFreed(port)
}

// killProcessOnPortUnix kills processes on Unix using lsof and kill.
func (pm *PortManager) killProcessOnPortUnix(port int) error {
	portStr := strconv.Itoa(port)

	// Use proper argument array instead of shell string interpolation
	// lsof -ti:PORT returns PIDs, one per line
	ctx, cancel := context.WithTimeout(context.Background(), ProcessKillTimeout)
	defer cancel()

	// Get PIDs listening on the port
	output, err := exec.CommandContext(ctx, "lsof", "-ti:"+portStr).Output()
	if err != nil {
		// lsof exits with 1 if no processes found - this is not an error
		return nil
	}

	pids := strings.Fields(strings.TrimSpace(string(output)))
	if len(pids) == 0 {
		return nil
	}

	// Kill each process
	for _, pid := range pids {
		// Validate PID is numeric to prevent injection
		if _, err := strconv.Atoi(pid); err != nil {
			continue
		}
		_ = exec.CommandContext(ctx, "kill", "-9", pid).Run()
	}

	// Verify port is actually freed with retry loop
	return pm.verifyPortFreed(port)
}

// verifyPortFreed verifies that a port is freed after kill attempt with retry logic.
// This prevents TOCTOU race conditions where another process could bind to the port.
func (pm *PortManager) verifyPortFreed(port int) error {
	for i := 0; i < ProcessKillMaxRetries; i++ {
		if pm.isPortAvailable(port) {
			return nil
		}
		time.Sleep(ProcessKillGracePeriod)
	}

	// Port still not available after retries
	return fmt.Errorf("port %d still in use after kill attempt and %d retries", port, ProcessKillMaxRetries)
}

// load reads port assignments from disk.
func (pm *PortManager) load() error {
	data, err := os.ReadFile(pm.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &pm.assignments)
}

// save writes port assignments to disk.
func (pm *PortManager) save() error {
	data, err := json.MarshalIndent(pm.assignments, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal port assignments: %w", err)
	}

	return os.WriteFile(pm.filePath, data, 0600)
}
