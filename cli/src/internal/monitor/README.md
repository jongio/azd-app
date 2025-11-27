# State Monitor

The `monitor` package provides real-time state monitoring for services with automatic transition detection and notification capabilities.

## Features

- **Continuous Monitoring**: Polls service registry at configurable intervals (default: 5 seconds)
- **State Transition Detection**: Detects meaningful changes in service state, health, and process status
- **Severity Classification**: Categorizes transitions as Critical, Warning, or Info
- **Rate Limiting**: Prevents notification storms by deduplicating events within a time window
- **Event History**: Maintains a rolling history of state transitions
- **Multi-Listener Support**: Multiple listeners can subscribe to state changes
- **Thread-Safe**: Safe for concurrent access from multiple goroutines
- **Performance Optimized**: Minimal overhead on running services

## Installation

```go
import "github.com/jongio/azd-app/cli/src/internal/monitor"
```

## Quick Start

```go
// Get service registry
reg := registry.GetRegistry(".")

// Create monitor with default config
config := monitor.DefaultMonitorConfig()
stateMonitor := monitor.NewStateMonitor(reg, config)

// Add listener for state changes
stateMonitor.AddListener(func(transition monitor.StateTransition) {
    if transition.Severity == monitor.SeverityCritical {
        log.Printf("CRITICAL: %s - %s", transition.ServiceName, transition.Description)
    }
})

// Start monitoring
stateMonitor.Start()
defer stateMonitor.Stop()
```

## Configuration

### Default Configuration

```go
config := monitor.DefaultMonitorConfig()
// Interval: 5 seconds
// MaxHistory: 1000 transitions
// RateLimitWindow: 5 minutes
```

### Custom Configuration

```go
config := monitor.MonitorConfig{
    Interval:        10 * time.Second,  // Poll every 10 seconds
    MaxHistory:      500,                 // Keep last 500 transitions
    RateLimitWindow: 2 * time.Minute,    // Deduplicate within 2 minutes
}
```

## State Detection

### Critical States (Always Notify)

The monitor detects these critical conditions:

- **Process Crashed**: Service PID no longer exists
- **Error Status**: Service entered error state
- **Health Failure**: Service changed from healthy to unhealthy
- **Port Unbound**: Service port no longer listening

### Warning States (Configurable)

- **Slow Start**: Service taking longer than expected to start (>30s)
- **Health Degraded**: Service responding but with degraded health

### Info States

- **Service Started**: Successful transition from starting to running
- **Service Healthy**: Service became healthy after being unhealthy

## Usage Examples

### Basic Monitoring

```go
reg := registry.GetRegistry(".")
config := monitor.DefaultMonitorConfig()
mon := monitor.NewStateMonitor(reg, config)

mon.AddListener(func(t monitor.StateTransition) {
    fmt.Printf("[%s] %s: %s\n", 
        t.Severity.String(), 
        t.ServiceName, 
        t.Description)
})

mon.Start()
defer mon.Stop()

// Your application runs...
```

### Critical Events Only

```go
mon.AddListener(func(t monitor.StateTransition) {
    if t.Severity == monitor.SeverityCritical {
        // Send OS notification
        sendNotification(t.ServiceName, t.Description)
    }
})
```

### Query State History

```go
// Get all transitions
history := mon.GetHistory()
for _, trans := range history {
    fmt.Printf("%s: %s\n", trans.ServiceName, trans.Description)
}

// Get current state of a service
if state, exists := mon.GetCurrentState("api-service"); exists {
    fmt.Printf("Status: %s, Health: %s, PID: %d\n", 
        state.Status, state.Health, state.PID)
}
```

### Multiple Listeners

```go
// Logger listener
mon.AddListener(func(t monitor.StateTransition) {
    log.Printf("[MONITOR] %s: %s", t.ServiceName, t.Description)
})

// Notification listener
mon.AddListener(func(t monitor.StateTransition) {
    if t.Severity == monitor.SeverityCritical {
        sendOSNotification(t)
    }
})

// Metrics listener
mon.AddListener(func(t monitor.StateTransition) {
    recordMetric("state_transition", map[string]string{
        "service": t.ServiceName,
        "severity": t.Severity.String(),
    })
})
```

## API Reference

### Types

#### `StateMonitor`

Main monitoring service.

**Methods**:
- `Start()` - Begin monitoring in background
- `Stop()` - Stop monitoring and clean up
- `AddListener(listener StateListener)` - Register state change listener
- `GetHistory() []StateTransition` - Get transition history
- `GetCurrentState(serviceName string) (*ServiceState, bool)` - Get current service state

#### `ServiceState`

Snapshot of service state at a point in time.

```go
type ServiceState struct {
    Name        string
    Status      string    // "starting", "ready", "running", "error", etc.
    Health      string    // "healthy", "unhealthy", "unknown"
    PID         int
    Port        int
    PortListens bool      // Is port actually listening?
    PIDValid    bool      // Is PID valid (process exists)?
    Timestamp   time.Time
}
```

#### `StateTransition`

Represents a detected state change.

```go
type StateTransition struct {
    ServiceName  string
    FromState    *ServiceState
    ToState      *ServiceState
    Severity     Severity
    Description  string
    Timestamp    time.Time
    Acknowledged bool
}
```

#### `Severity`

Transition severity level.

```go
const (
    SeverityInfo     Severity = iota  // Informational
    SeverityWarning                   // Warning condition
    SeverityCritical                  // Critical failure
)
```

### Functions

#### `NewStateMonitor(reg *registry.ServiceRegistry, config MonitorConfig) *StateMonitor`

Creates a new state monitor for the given service registry.

#### `DefaultMonitorConfig() MonitorConfig`

Returns default monitoring configuration.

## Rate Limiting

The monitor includes built-in rate limiting to prevent notification storms:

- **Critical events**: Never rate limited
- **Warning/Info events**: Deduplicated within rate limit window (default: 5 minutes)
- **Per-service tracking**: Rate limiting tracked separately for each service

Example:
```go
config := monitor.MonitorConfig{
    RateLimitWindow: 2 * time.Minute,  // Don't notify same service within 2 min
}
```

## Performance

- **Polling overhead**: < 1% CPU with default 5-second interval
- **Memory usage**: ~1KB per service + ~500 bytes per transition in history
- **Goroutines**: 1 monitoring goroutine + 1 per active listener callback
- **Lock contention**: Minimal; uses read locks for queries, write locks only for state updates

## Thread Safety

All public methods are thread-safe:
- Concurrent `AddListener()` calls are safe
- Concurrent `GetHistory()` / `GetCurrentState()` calls are safe
- Listeners are called in separate goroutines to prevent blocking

## Integration

### With Dashboard

```go
// In dashboard server
mon.AddListener(func(t monitor.StateTransition) {
    // Send transition to dashboard via WebSocket
    dashboardServer.BroadcastStateChange(t)
})
```

### With OS Notifications

```go
mon.AddListener(func(t monitor.StateTransition) {
    if t.Severity == monitor.SeverityCritical {
        osnotify.Send(osnotify.Notification{
            Title: fmt.Sprintf("%s Failed", t.ServiceName),
            Body: t.Description,
            Severity: osnotify.Critical,
        })
    }
})
```

### With Metrics

```go
mon.AddListener(func(t monitor.StateTransition) {
    metrics.IncrCounter("state_transitions_total", map[string]string{
        "service": t.ServiceName,
        "severity": t.Severity.String(),
        "from_status": t.FromState.Status,
        "to_status": t.ToState.Status,
    })
})
```

## Testing

Run tests:
```bash
go test ./src/internal/monitor/...
```

Run with coverage:
```bash
go test ./src/internal/monitor/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Best Practices

1. **Start Early**: Start monitoring before starting services
2. **Stop Gracefully**: Always defer `Stop()` to clean up resources
3. **Lightweight Listeners**: Keep listener callbacks fast; do heavy work in separate goroutines
4. **Error Handling**: Listeners should handle panics internally
5. **Rate Limit Awareness**: Critical events bypass rate limiting; use appropriately
6. **History Size**: Adjust `MaxHistory` based on your needs (default 1000 is suitable for most cases)

## Troubleshooting

### Monitor not detecting transitions

- Verify services are registered in service registry
- Check monitor is started with `Start()`
- Ensure polling interval is appropriate
- Check logs for detection errors

### Too many notifications

- Increase `RateLimitWindow` to deduplicate more aggressively
- Filter by severity in listener
- Check for flapping services (repeatedly changing state)

### High memory usage

- Reduce `MaxHistory` size
- Implement transition cleanup for acknowledged events

## Future Enhancements

- Flapping detection (service restarting frequently)
- Configurable transition rules
- Notification aggregation (multiple services failed)
- Export metrics to Prometheus/StatsD
- Persistent storage for transition history
