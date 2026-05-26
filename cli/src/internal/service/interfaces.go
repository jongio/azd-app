// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"context"
	"time"
)

// PortAllocator manages port assignments for services.
// The primary implementation is in the portmanager package.
type PortAllocator interface {
	// AssignPort allocates a port for a service. If preferredPort is non-zero,
	// it attempts to use that port first. Returns the assigned port, whether it
	// was auto-assigned (true) or used the preferred port (false), and any error.
	AssignPort(serviceName string, preferredPort int, isExplicit bool) (port int, autoAssigned bool, err error)

	// ReleasePort releases the port assignment for a service.
	ReleasePort(serviceName string) error

	// GetAssignment returns the currently assigned port for a service, if any.
	GetAssignment(serviceName string) (port int, exists bool)
}

// HealthChecker performs health checks on running services.
type HealthChecker interface {
	// CheckHTTP performs an HTTP health check against a service endpoint.
	CheckHTTP(ctx context.Context, port int, path string) error

	// CheckPort verifies that a TCP port is listening.
	CheckPort(ctx context.Context, port int) error

	// CheckProcess verifies that a service process is still running.
	CheckProcess(process *ServiceProcess) error
}

// LogProvider manages log buffers for services in a project.
type LogProvider interface {
	// CreateBuffer creates a log buffer for a service.
	CreateBuffer(serviceName string, maxSize int, enableFileLogging bool) (*LogBuffer, error)

	// GetBuffer retrieves the log buffer for a service.
	GetBuffer(serviceName string) (*LogBuffer, bool)

	// GetAllLogs returns logs from all services, limited to n most recent entries per service.
	GetAllLogs(n int) []LogEntry

	// GetAllLogsSince returns logs from all services since a specific time.
	GetAllLogsSince(since time.Time) []LogEntry

	// GetServiceNames returns the names of all services with log buffers.
	GetServiceNames() []string

	// RemoveBuffer removes and closes the log buffer for a service.
	RemoveBuffer(serviceName string) error

	// Clear removes all log buffers.
	Clear() error
}
