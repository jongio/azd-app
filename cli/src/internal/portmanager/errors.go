package portmanager

import "fmt"

// PortInUseError represents an error when a port is already in use.
type PortInUseError struct {
	Port        int
	PID         int
	ProcessName string
	ServiceName string
}

// Error implements the error interface.
func (e *PortInUseError) Error() string {
	if e.ProcessName != "" {
		return fmt.Sprintf("port %d required by service '%s' is in use by %s (PID %d)",
			e.Port, e.ServiceName, e.ProcessName, e.PID)
	}
	return fmt.Sprintf("port %d required by service '%s' is in use by PID %d",
		e.Port, e.ServiceName, e.PID)
}

// PortRangeExhaustedError represents an error when no ports are available in the range.
type PortRangeExhaustedError struct {
	StartPort int
	EndPort   int
}

// Error implements the error interface.
func (e *PortRangeExhaustedError) Error() string {
	return fmt.Sprintf("no available ports in range %d-%d", e.StartPort, e.EndPort)
}

// InvalidPortError represents an error when a port number is invalid.
type InvalidPortError struct {
	Port   int
	Reason string
}

// Error implements the error interface.
func (e *InvalidPortError) Error() string {
	return fmt.Sprintf("invalid port %d: %s", e.Port, e.Reason)
}
