package monitor_test

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/monitor"
	"github.com/jongio/azd-app/cli/src/internal/registry"
)

// Example demonstrates how to use the StateMonitor.
func Example() {
	// Get the service registry for the current project
	reg := registry.GetRegistry(".")

	// Create monitor configuration
	config := monitor.DefaultMonitorConfig()
	config.Interval = 5 * time.Second // Poll every 5 seconds

	// Create the state monitor
	stateMonitor := monitor.NewStateMonitor(reg, config)

	// Add a listener for state transitions
	stateMonitor.AddListener(func(transition monitor.StateTransition) {
		fmt.Printf("[%s] Service %s: %s (severity: %s)\n",
			transition.Timestamp.Format("15:04:05"),
			transition.ServiceName,
			transition.Description,
			transition.Severity.String())

		// Handle critical events
		if transition.Severity == monitor.SeverityCritical {
			log.Printf("CRITICAL: %s - %s", transition.ServiceName, transition.Description)
			// Could send OS notification here
		}
	})

	// Start monitoring
	stateMonitor.Start()
	defer stateMonitor.Stop()

	// Your application runs here...
	// The monitor will detect state changes in the background

	// Example: Get history of transitions
	history := stateMonitor.GetHistory()
	fmt.Printf("Total transitions recorded: %d\n", len(history))

	// Example: Get current state of a service
	if state, exists := stateMonitor.GetCurrentState("api-service"); exists {
		fmt.Printf("API Service Status: %s, Health: %s\n", state.Status, state.Health)
	}
}

// ExampleStateMonitor_criticalOnly demonstrates monitoring only critical events.
func ExampleStateMonitor_criticalOnly() {
	reg := registry.GetRegistry(".")
	config := monitor.DefaultMonitorConfig()
	stateMonitor := monitor.NewStateMonitor(reg, config)

	// Only handle critical events
	stateMonitor.AddListener(func(transition monitor.StateTransition) {
		if transition.Severity == monitor.SeverityCritical {
			// Send OS notification
			fmt.Printf("ALERT: %s failed - %s\n",
				transition.ServiceName,
				transition.Description)
		}
	})

	stateMonitor.Start()
	defer stateMonitor.Stop()

	// Application runs...
}

// ExampleStateMonitor_withRegistry demonstrates integration with service registry.
func ExampleStateMonitor_withRegistry() {
	projectDir, _ := os.Getwd()
	reg := registry.GetRegistry(projectDir)

	// Register a service
	entry := &registry.ServiceRegistryEntry{
		Name:      "web-service",
		PID:       12345,
		Port:      8080,
		Status:    "running",
		StartTime: time.Now(),
	}
	_ = reg.Register(entry)

	// Monitor will automatically detect this service
	config := monitor.DefaultMonitorConfig()
	stateMonitor := monitor.NewStateMonitor(reg, config)

	stateMonitor.AddListener(func(transition monitor.StateTransition) {
		fmt.Printf("State change detected: %s\n", transition.Description)
	})

	stateMonitor.Start()
	defer stateMonitor.Stop()

	// Simulate state change
	time.Sleep(100 * time.Millisecond)
	_ = reg.UpdateStatus("web-service", "error")

	// Monitor will detect the transition to error state
	time.Sleep(200 * time.Millisecond)
}
