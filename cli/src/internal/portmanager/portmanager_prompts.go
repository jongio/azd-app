// Package portmanager provides port allocation, management, and process monitoring capabilities.
package portmanager

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// PortConflictAction represents the user's chosen action for handling a port conflict.
type PortConflictAction int

const (
	// ActionKill indicates the user wants to kill the process using the port.
	ActionKill PortConflictAction = iota
	// ActionReassign indicates the user wants to be assigned a different port.
	ActionReassign
	// ActionCancel indicates the user wants to cancel the operation.
	ActionCancel
	// ActionAlwaysKill indicates the user wants to always kill processes on port conflicts.
	ActionAlwaysKill
)

// handlePortConflict prompts the user to resolve a port conflict and returns the chosen action.
// It checks the always-kill preference first and returns ActionKill if enabled.
//
// Parameters:
//   - pm: The port manager instance (used for preferences)
//   - port: The conflicting port number
//   - serviceName: The name of the service requiring the port
//   - processInfo: Human-readable info about the process using the port (e.g., " by nginx (PID 1234)")
//   - isExplicit: Whether this is an explicit port from azure.yaml (affects messaging)
//
// Returns:
//   - PortConflictAction: The user's chosen action
//   - error: Non-nil if reading user input fails or user cancels
//
// IMPORTANT: The caller MUST release the mutex before calling this function
// and re-acquire it after, as this function blocks on user input.
func handlePortConflict(pm *PortManager, port int, serviceName string, processInfo string, isExplicit bool) (PortConflictAction, error) {
	// Check if user has set preference to always kill
	if pm.getAlwaysKillPreference() {
		slog.Info("auto-killing process on port due to always-kill preference", "port", port, "service", serviceName)
		printAutoKillMessage(serviceName, port, processInfo, isExplicit)
		return ActionKill, nil
	}

	// Print the conflict message
	printConflictMessage(serviceName, port, processInfo, isExplicit)

	// Print options
	fmt.Fprintf(os.Stderr, "Options:\n")
	fmt.Fprintf(os.Stderr, "  1) Kill the process using port %d\n", port)
	fmt.Fprintf(os.Stderr, "  2) Assign a different port automatically\n")
	fmt.Fprintf(os.Stderr, "  3) Cancel\n")
	fmt.Fprintf(os.Stderr, "  4) Always kill processes (don't ask again)\n\n")
	fmt.Fprintf(os.Stderr, "Choose (1/2/3/4): ")

	// Read user input
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return ActionCancel, fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(response)
	switch response {
	case "1":
		return ActionKill, nil
	case "2":
		return ActionReassign, nil
	case "4":
		return ActionAlwaysKill, nil
	default:
		return ActionCancel, nil
	}
}

// printAutoKillMessage prints the message shown when auto-kill preference is enabled.
func printAutoKillMessage(serviceName string, port int, processInfo string, isExplicit bool) {
	if isExplicit {
		fmt.Fprintf(os.Stderr, "\n⚠️  Service '%s' requires port %d%s - auto-killing (always-kill enabled)\n", serviceName, port, processInfo)
	} else {
		// For previously assigned or preferred ports
		fmt.Fprintf(os.Stderr, "\n⚠️  Service '%s' port %d is in use%s - auto-killing (always-kill enabled)\n", serviceName, port, processInfo)
	}
}

// printConflictMessage prints the initial conflict message before showing options.
func printConflictMessage(serviceName string, port int, processInfo string, isExplicit bool) {
	if isExplicit {
		fmt.Fprintf(os.Stderr, "\n⚠️  Service '%s' requires port %d (configured in azure.yaml)\n", serviceName, port)
		fmt.Fprintf(os.Stderr, "This port is currently in use%s.\n\n", processInfo)
	} else {
		fmt.Fprintf(os.Stderr, "\n⚠️  Service '%s' port %d is already in use%s\n", serviceName, port, processInfo)
	}
}

// promptUpdateAzureYaml prompts the user to update azure.yaml with a new port.
// Returns true if user wants to update, false otherwise.
//
// IMPORTANT: The caller MUST release the mutex before calling this function
// and re-acquire it after, as this function blocks on user input.
func promptUpdateAzureYaml(port int) bool {
	fmt.Fprintf(os.Stderr, "\n⚠️  IMPORTANT: Update your application code to use port %d\n", port)
	fmt.Fprintf(os.Stderr, "Would you like to update azure.yaml to use port %d for future runs? (y/N): ", port)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// getProcessInfoString formats process info for display in messages.
// Returns a string like " by nginx (PID 1234)" or " (PID 1234)" if name is empty.
func getProcessInfoString(pm *PortManager, port int) string {
	info, err := pm.getProcessInfoOnPort(port)
	if err != nil {
		return ""
	}

	if info.Name != "" {
		return fmt.Sprintf(" by %s (PID %d)", info.Name, info.PID)
	}
	return fmt.Sprintf(" by PID %d", info.PID)
}

// printPortFreedMessage prints a message when a port is successfully freed.
func printPortFreedMessage(serviceName string, port int) {
	fmt.Fprintf(os.Stderr, "✓ Port %d freed and ready for service '%s'\n\n", port, serviceName)
}

// printPortAssignedMessage prints a message when a port is assigned to a service.
func printPortAssignedMessage(serviceName string, port int) {
	fmt.Fprintf(os.Stderr, "✓ Assigned port %d to service '%s'\n\n", port, serviceName)
}

// printPortAvailableMessage prints a message when a port becomes available.
func printPortAvailableMessage(serviceName string, port int) {
	fmt.Fprintf(os.Stderr, "✓ Port %d is now available and assigned to service '%s'\n\n", port, serviceName)
}

// printFindingPortMessage prints a message when searching for an available port.
func printFindingPortMessage(serviceName string) {
	fmt.Fprintf(os.Stderr, "\nFinding available port for '%s'...\n", serviceName)
}

// printPreferenceSavedMessage prints a message when the always-kill preference is saved.
func printPreferenceSavedMessage() {
	fmt.Fprintf(os.Stderr, "\n✓ Preference saved: will always kill processes on port conflicts\n")
}

// printKillFailedTip prints a helpful tip when killing a process fails.
func printKillFailedTip() {
	fmt.Fprintf(os.Stderr, "\nTip: Choose option 2 to find a different available port\n\n")
}

// printPortStillInUseMessage prints a message when a port is still in use after cleanup.
func printPortStillInUseMessage(port int) {
	fmt.Fprintf(os.Stderr, "\n⚠️  Port %d is still in use after cleanup\n", port)
}

// printAutoAssignedMessage prints a message when a port is auto-assigned.
func printAutoAssignedMessage(serviceName string, port int) {
	fmt.Fprintf(os.Stderr, "✓ Auto-assigned port %d to service '%s'\n\n", port, serviceName)
}
