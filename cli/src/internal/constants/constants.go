// Package constants provides shared constants used across the CLI.
package constants

import "time"

// Dashboard configuration
const (
	// DashboardServiceName is the service name used for port manager assignments
	DashboardServiceName = "azd-app-dashboard"
	// DashboardPortRangeMin is the minimum port for dashboard (ephemeral range)
	DashboardPortRangeMin = 40000
	// DashboardPortRangeMax is the maximum port for dashboard (ephemeral range)
	DashboardPortRangeMax = 49999
)

// Timeouts
const (
	// ServerStartupDelay is the delay to wait for HTTP server startup
	ServerStartupDelay = 100 * time.Millisecond
	// DashboardAPITimeout is the timeout for dashboard API HTTP client requests
	DashboardAPITimeout = 5 * time.Second
)

// Severity levels
const (
	SeverityCritical = "critical"
	SeverityWarning  = "warning"
	SeverityInfo     = "info"
)

// Service status values
const (
	StatusRunning    = "running"
	StatusStopped    = "stopped"
	StatusStarting   = "starting"
	StatusReady      = "ready"
	StatusNotRunning = "not-running"
	StatusError      = "error"
	StatusStopping   = "stopping"
)
