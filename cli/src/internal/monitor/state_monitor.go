// Package monitor provides state monitoring for services with transition detection.
package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/registry"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// StateMonitor monitors service state changes and detects transitions.
type StateMonitor struct {
	registry        *registry.ServiceRegistry
	interval        time.Duration
	previousStates  map[string]*ServiceState
	stateHistory    []StateTransition
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	listeners       []StateListener
	listenersMu     sync.RWMutex
	maxHistory      int
	rateLimitWindow time.Duration
	lastNotifyTime  map[string]time.Time
	rateLimitMu     sync.RWMutex
}

// ServiceState represents the state of a service at a point in time.
type ServiceState struct {
	Name        string
	Status      string // "starting", "ready", "running", "stopping", "stopped", "error", "not-running"
	Health      string // "healthy", "unhealthy", "unknown"
	PID         int
	Port        int
	PortListens bool
	PIDValid    bool
	Timestamp   time.Time
}

// StateTransition represents a change in service state.
type StateTransition struct {
	ServiceName  string
	FromState    *ServiceState
	ToState      *ServiceState
	Severity     Severity
	Description  string
	Timestamp    time.Time
	Acknowledged bool
}

// Severity represents the severity of a state transition.
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityCritical
)

// String returns the string representation of severity.
func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// StateListener is called when a state transition occurs.
type StateListener func(transition StateTransition)

// MonitorConfig contains configuration for the state monitor.
type MonitorConfig struct {
	Interval        time.Duration // Polling interval (default: 5s)
	MaxHistory      int           // Maximum transitions to keep (default: 1000)
	RateLimitWindow time.Duration // Deduplication window (default: 5m)
}

// DefaultMonitorConfig returns default monitoring configuration.
func DefaultMonitorConfig() MonitorConfig {
	return MonitorConfig{
		Interval:        5 * time.Second,
		MaxHistory:      1000,
		RateLimitWindow: 5 * time.Minute,
	}
}

// NewStateMonitor creates a new state monitor for the given registry.
func NewStateMonitor(reg *registry.ServiceRegistry, config MonitorConfig) *StateMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	if config.Interval == 0 {
		config.Interval = 5 * time.Second
	}
	if config.MaxHistory == 0 {
		config.MaxHistory = 1000
	}
	if config.RateLimitWindow == 0 {
		config.RateLimitWindow = 5 * time.Minute
	}

	return &StateMonitor{
		registry:        reg,
		interval:        config.Interval,
		previousStates:  make(map[string]*ServiceState),
		stateHistory:    make([]StateTransition, 0, config.MaxHistory),
		ctx:             ctx,
		cancel:          cancel,
		listeners:       make([]StateListener, 0),
		maxHistory:      config.MaxHistory,
		rateLimitWindow: config.RateLimitWindow,
		lastNotifyTime:  make(map[string]time.Time),
	}
}

// Start begins monitoring service states.
func (m *StateMonitor) Start() {
	m.wg.Add(1)
	go m.monitorLoop()
}

// Stop stops the state monitor.
func (m *StateMonitor) Stop() {
	m.cancel()
	m.wg.Wait()
}

// AddListener registers a listener for state transitions.
func (m *StateMonitor) AddListener(listener StateListener) {
	m.listenersMu.Lock()
	defer m.listenersMu.Unlock()
	m.listeners = append(m.listeners, listener)
}

// GetHistory returns the state transition history.
func (m *StateMonitor) GetHistory() []StateTransition {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modification
	history := make([]StateTransition, len(m.stateHistory))
	copy(history, m.stateHistory)
	return history
}

// GetCurrentState returns the current state of a service.
func (m *StateMonitor) GetCurrentState(serviceName string) (*ServiceState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.previousStates[serviceName]
	if !exists {
		return nil, false
	}

	// Return a copy
	stateCopy := *state
	return &stateCopy, true
}

// monitorLoop is the main monitoring loop.
func (m *StateMonitor) monitorLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkAllServices()
		}
	}
}

// checkAllServices checks all registered services for state changes.
func (m *StateMonitor) checkAllServices() {
	services := m.registry.ListAll()

	for _, svc := range services {
		currentState := m.captureServiceState(svc)
		m.detectTransition(currentState)
	}
}

// captureServiceState captures the current state of a service.
func (m *StateMonitor) captureServiceState(svc *registry.ServiceRegistryEntry) *ServiceState {
	state := &ServiceState{
		Name:      svc.Name,
		Status:    svc.Status,
		Health:    svc.Health,
		PID:       svc.PID,
		Port:      svc.Port,
		Timestamp: time.Now(),
	}

	// Check if PID is valid
	if svc.PID > 0 {
		state.PIDValid = isProcessRunning(svc.PID)
	}

	// Check if port is listening
	if svc.Port > 0 {
		state.PortListens = service.IsPortListening(svc.Port)
	}

	return state
}

// detectTransition detects and records state transitions.
func (m *StateMonitor) detectTransition(currentState *ServiceState) {
	// Process state update under lock, extract transition if any
	transition := m.processStateUpdate(currentState)

	// Notify listeners outside of lock to prevent deadlock
	if transition != nil {
		m.notifyListeners(*transition)
	}
}

// processStateUpdate handles the locked portion of state transition detection.
// Returns a transition to notify about, or nil if no notification needed.
func (m *StateMonitor) processStateUpdate(currentState *ServiceState) *StateTransition {
	m.mu.Lock()
	defer m.mu.Unlock()

	previousState, exists := m.previousStates[currentState.Name]

	// First time seeing this service
	if !exists {
		m.previousStates[currentState.Name] = currentState
		return nil
	}

	// Check for meaningful transitions
	transition := m.evaluateTransition(previousState, currentState)
	if transition == nil {
		// No meaningful transition, just update state
		m.previousStates[currentState.Name] = currentState
		return nil
	}

	// Check rate limiting
	if m.shouldRateLimit(currentState.Name, transition.Severity) {
		slog.Debug("Rate limiting notification",
			"service", currentState.Name,
			"severity", transition.Severity.String())
		m.previousStates[currentState.Name] = currentState
		return nil
	}

	// Record transition
	m.addTransitionLocked(transition)
	m.previousStates[currentState.Name] = currentState

	// Update rate limit timestamp (acquires its own lock)
	m.updateRateLimit(currentState.Name)

	// Return a copy for notification (caller will notify outside lock)
	transitionCopy := *transition
	return &transitionCopy
}

// evaluateTransition determines if a state change is meaningful.
func (m *StateMonitor) evaluateTransition(prev, curr *ServiceState) *StateTransition {
	// Check for critical state changes

	// Process crashed
	if prev.PIDValid && !curr.PIDValid && curr.PID > 0 {
		return &StateTransition{
			ServiceName: curr.Name,
			FromState:   prev,
			ToState:     curr,
			Severity:    SeverityCritical,
			Description: fmt.Sprintf("Process crashed - PID %d no longer exists", prev.PID),
			Timestamp:   curr.Timestamp,
		}
	}

	// Status changed to error
	if prev.Status != "error" && curr.Status == "error" {
		return &StateTransition{
			ServiceName: curr.Name,
			FromState:   prev,
			ToState:     curr,
			Severity:    SeverityCritical,
			Description: "Service entered error state",
			Timestamp:   curr.Timestamp,
		}
	}

	// Health degraded
	if prev.Health == "healthy" && curr.Health == "unhealthy" {
		return &StateTransition{
			ServiceName: curr.Name,
			FromState:   prev,
			ToState:     curr,
			Severity:    SeverityCritical,
			Description: "Health check failure - service became unhealthy",
			Timestamp:   curr.Timestamp,
		}
	}

	// Port stopped listening
	if prev.PortListens && !curr.PortListens && curr.Port > 0 {
		return &StateTransition{
			ServiceName: curr.Name,
			FromState:   prev,
			ToState:     curr,
			Severity:    SeverityCritical,
			Description: fmt.Sprintf("Port %d no longer listening", curr.Port),
			Timestamp:   curr.Timestamp,
		}
	}

	// Warning: service taking long to start
	if prev.Status == "starting" && curr.Status == "starting" {
		duration := curr.Timestamp.Sub(prev.Timestamp)
		if duration > 30*time.Second {
			return &StateTransition{
				ServiceName: curr.Name,
				FromState:   prev,
				ToState:     curr,
				Severity:    SeverityWarning,
				Description: fmt.Sprintf("Service taking longer than expected to start (%.0fs)", duration.Seconds()),
				Timestamp:   curr.Timestamp,
			}
		}
	}

	// Info: service became healthy
	if prev.Health != "healthy" && curr.Health == "healthy" {
		return &StateTransition{
			ServiceName: curr.Name,
			FromState:   prev,
			ToState:     curr,
			Severity:    SeverityInfo,
			Description: "Service became healthy",
			Timestamp:   curr.Timestamp,
		}
	}

	// Info: service started successfully
	if (prev.Status == "starting" || prev.Status == "stopped") &&
		(curr.Status == "running" || curr.Status == "ready") {
		return &StateTransition{
			ServiceName: curr.Name,
			FromState:   prev,
			ToState:     curr,
			Severity:    SeverityInfo,
			Description: "Service started successfully",
			Timestamp:   curr.Timestamp,
		}
	}

	return nil
}

// shouldRateLimit checks if notification should be rate limited.
func (m *StateMonitor) shouldRateLimit(serviceName string, severity Severity) bool {
	// Never rate limit critical events
	if severity == SeverityCritical {
		return false
	}

	m.rateLimitMu.RLock()
	defer m.rateLimitMu.RUnlock()

	key := fmt.Sprintf("%s:%s", serviceName, severity.String())
	lastTime, exists := m.lastNotifyTime[key]
	if !exists {
		return false
	}

	return time.Since(lastTime) < m.rateLimitWindow
}

// updateRateLimit updates the rate limit timestamp.
// This function manages its own locking.
func (m *StateMonitor) updateRateLimit(serviceName string) {
	m.rateLimitMu.Lock()
	defer m.rateLimitMu.Unlock()

	// Update for all severity levels to prevent notification storms
	for _, sev := range []Severity{SeverityInfo, SeverityWarning, SeverityCritical} {
		key := fmt.Sprintf("%s:%s", serviceName, sev.String())
		m.lastNotifyTime[key] = time.Now()
	}
}

// addTransitionLocked adds a transition to history (must hold mu).
func (m *StateMonitor) addTransitionLocked(transition *StateTransition) {
	m.stateHistory = append(m.stateHistory, *transition)

	// Trim history if it exceeds max
	if len(m.stateHistory) > m.maxHistory {
		// Keep most recent transitions
		copy(m.stateHistory, m.stateHistory[len(m.stateHistory)-m.maxHistory:])
		m.stateHistory = m.stateHistory[:m.maxHistory]
	}
}

// notifyListeners notifies all registered listeners.
func (m *StateMonitor) notifyListeners(transition StateTransition) {
	m.listenersMu.RLock()
	listeners := make([]StateListener, len(m.listeners))
	copy(listeners, m.listeners)
	m.listenersMu.RUnlock()

	for _, listener := range listeners {
		// Call listener in goroutine to prevent blocking
		go func(l StateListener) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("State listener panic", "error", r)
				}
			}()
			l(transition)
		}(listener)
	}
}

// isProcessRunning checks if a process with the given PID is running.
// Works cross-platform (Windows and Unix).
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, Signal(0) checks if process exists without sending a signal
	// On Windows, this always succeeds for FindProcess, so we use a different approach
	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}

	// On Windows, Signal(0) is not supported
	// On Unix, Signal(0) failing means process doesn't exist
	// Check if error contains "not supported" or similar Windows-specific messages
	errMsg := err.Error()
	if strings.Contains(errMsg, "not supported") {
		// On Windows, FindProcess succeeds if process exists
		// If we got here, process handle was created successfully
		return true
	}

	// On Unix, any other error means process doesn't exist
	return false
}
