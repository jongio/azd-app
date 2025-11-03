package portmanager

import (
	"bufio"
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

	managerCacheMu.Lock()
	defer managerCacheMu.Unlock()

	if mgr, exists := managerCache[absPath]; exists {
		return mgr
	}

	portsDir := filepath.Join(absPath, ".azure")
	portsFile := filepath.Join(portsDir, "ports.json")

	manager := &PortManager{
		assignments: make(map[string]*PortAssignment),
		filePath:    portsFile,
	}
	manager.portRange.start = 3000
	manager.portRange.end = 65535 // Allow full dynamic port range

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
// If isExplicit is true, the port came from azure.yaml config and MUST be used (never changed).
// If cleanStale is true, it will prompt user before killing processes on assigned ports.
func (pm *PortManager) AssignPort(serviceName string, preferredPort int, isExplicit bool, cleanStale bool) (int, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// EXPLICIT PORT MODE: Port from azure.yaml - MUST be used, never changed
	if isExplicit {
		// Validate port is in range
		if preferredPort < pm.portRange.start || preferredPort > pm.portRange.end {
			return 0, fmt.Errorf("explicit port %d for service '%s' is outside valid range %d-%d",
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
			return preferredPort, nil
		}

		// Port is in use - MUST prompt and kill (or fail)
		fmt.Fprintf(os.Stderr, "⚠️  Service '%s' requires port %d (configured in azure.yaml)\n", serviceName, preferredPort)
		if err := pm.promptAndKillProcessOnPort(serviceName, preferredPort); err != nil {
			return 0, fmt.Errorf("port %d is required for service '%s' but is in use and cannot be freed: %w",
				preferredPort, serviceName, err)
		}

		// Verify port is now available
		if !pm.isPortAvailable(preferredPort) {
			return 0, fmt.Errorf("port %d is still in use after cleanup attempt", preferredPort)
		}

		pm.assignments[serviceName] = &PortAssignment{
			ServiceName: serviceName,
			Port:        preferredPort,
			LastUsed:    time.Now(),
		}
		if err := pm.save(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save port assignment: %v\n", err)
		}
		return preferredPort, nil
	}

	// FLEXIBLE PORT MODE: Port can be changed if needed

	// Check if we already have an assignment
	if assignment, exists := pm.assignments[serviceName]; exists {
		assignment.LastUsed = time.Now()

		// Check if port is already available
		if pm.isPortAvailable(assignment.Port) {
			if err := pm.save(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save port assignment: %v\n", err)
			}
			return assignment.Port, nil
		}

		// Port is in use - prompt user if cleanStale is enabled
		if cleanStale {
			if err := pm.promptAndKillProcessOnPort(serviceName, assignment.Port); err != nil {
				// User declined or error occurred, find a new port
				fmt.Fprintf(os.Stderr, "Finding alternative port for %s...\n", serviceName)
			} else {
				// Successfully cleaned port
				if pm.isPortAvailable(assignment.Port) {
					if err := pm.save(); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: failed to save port assignment: %v\n", err)
					}
					return assignment.Port, nil
				}
			}
		}

		// Port is still in use, need to find a new one
		fmt.Fprintf(os.Stderr, "Warning: Assigned port %d for %s is in use, finding new port\n", assignment.Port, serviceName)
	}

	// Try preferred port first
	if preferredPort >= pm.portRange.start && preferredPort <= pm.portRange.end {
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
			return preferredPort, nil
		}

		// Port is in use - prompt user if cleanStale is enabled
		if cleanStale {
			if err := pm.promptAndKillProcessOnPort(serviceName, preferredPort); err == nil {
				// Successfully cleaned port
				if pm.isPortAvailable(preferredPort) {
					pm.assignments[serviceName] = &PortAssignment{
						ServiceName: serviceName,
						Port:        preferredPort,
						LastUsed:    time.Now(),
					}
					if err := pm.save(); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: failed to save port assignment: %v\n", err)
					}
					return preferredPort, nil
				}
			}
		}

		// Preferred port unavailable, will find alternative below
		fmt.Fprintf(os.Stderr, "Preferred port %d for %s is in use, finding alternative...\n", preferredPort, serviceName)
	}

	// Find an available port
	port, err := pm.findAvailablePort()
	if err != nil {
		return 0, err
	}

	pm.assignments[serviceName] = &PortAssignment{
		ServiceName: serviceName,
		Port:        port,
		LastUsed:    time.Now(),
	}
	if err := pm.save(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save port assignment: %v\n", err)
	}
	return port, nil
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

// CleanStalePorts removes assignments for ports that haven't been used in over 7 days.
func (pm *PortManager) CleanStalePorts() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	for name, assignment := range pm.assignments {
		if assignment.LastUsed.Before(cutoff) {
			delete(pm.assignments, name)
		}
	}
	if err := pm.save(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save port assignments: %v\n", err)
	}
}

// isPortAvailable checks if a port is available.
func (pm *PortManager) isPortAvailable(port int) bool {
	addr := fmt.Sprintf("localhost:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	if err := listener.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to close listener: %v\n", err)
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

// promptAndKillProcessOnPort prompts the user before killing a process on the specified port.
func (pm *PortManager) promptAndKillProcessOnPort(serviceName string, port int) error {
	// Get process info
	pid, err := pm.getProcessOnPort(port)
	if err != nil {
		// If we can't get PID, fall back to generic message
		fmt.Printf("Port %d for service '%s' is in use. Stop existing process? (y/N): ", port, serviceName)
	} else {
		fmt.Printf("Port %d for service '%s' is in use by process %d. Stop existing process? (y/N): ", port, serviceName, pid)
	}

	// Read user input
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		return fmt.Errorf("user declined to kill process on port %d", port)
	}

	// User confirmed - kill the process
	return pm.killProcessOnPort(port)
}

// killProcessOnPort kills any process listening on the specified port.
func (pm *PortManager) killProcessOnPort(port int) error {
	var cmd []string
	var args []string

	if runtime.GOOS == "windows" {
		// Windows: use netstat and taskkill
		cmd = []string{"powershell", "-Command"}
		psScript := fmt.Sprintf(`
			$connections = netstat -ano | Select-String ":%d " | Select-String "LISTENING"
			foreach ($line in $connections) {
				$parts = $line -split '\s+' | Where-Object { $_ }
				$procId = $parts[-1]
				if ($procId -match '^\d+$') {
					Write-Host "Killing process $procId on port %d"
					Stop-Process -Id $procId -Force -ErrorAction SilentlyContinue
				}
			}
		`, port, port)
		args = append(cmd, psScript)
	} else {
		// Unix: use lsof and kill
		cmd = []string{"sh", "-c"}
		shScript := fmt.Sprintf("lsof -ti:%d | xargs -r kill -9", port)
		args = append(cmd, shScript)
	}

	// Execute the kill command
	if err := executor.RunCommand(cmd[0], args[1:], "."); err != nil {
		// Ignore errors - port might not be in use
		return nil
	}

	// Wait a moment for process to die
	time.Sleep(500 * time.Millisecond)
	return nil
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
