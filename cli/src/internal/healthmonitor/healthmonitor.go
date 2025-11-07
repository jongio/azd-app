package healthmonitor

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/registry"
	"github.com/shirou/gopsutil/v4/process"
)

// HealthMonitor manages periodic health checking for services.
type HealthMonitor struct {
	projectDir string
	registry   *registry.ServiceRegistry
	interval   time.Duration
	stopChan   chan struct{}
	running    bool
	mu         sync.Mutex
}

var (
	monitors   = make(map[string]*HealthMonitor)
	monitorsMu sync.Mutex
)

// GetMonitor returns the health monitor for a project (singleton per project).
func GetMonitor(projectDir string) *HealthMonitor {
	monitorsMu.Lock()
	defer monitorsMu.Unlock()

	if mon, exists := monitors[projectDir]; exists {
		return mon
	}

	mon := &HealthMonitor{
		projectDir: projectDir,
		registry:   registry.GetRegistry(projectDir),
		interval:   5 * time.Second,
		stopChan:   make(chan struct{}),
	}
	monitors[projectDir] = mon
	return mon
}

// Start begins health monitoring in a background goroutine.
func (hm *HealthMonitor) Start() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if hm.running {
		return fmt.Errorf("health monitor already running")
	}

	hm.running = true
	go hm.monitorLoop()
	return nil
}

// Stop terminates health monitoring.
func (hm *HealthMonitor) Stop() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if !hm.running {
		return
	}

	close(hm.stopChan)
	hm.running = false
}

// monitorLoop runs periodic health checks.
func (hm *HealthMonitor) monitorLoop() {
	ticker := time.NewTicker(hm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-hm.stopChan:
			return
		case <-ticker.C:
			hm.checkAllServices()
		}
	}
}

// checkAllServices checks health of all registered services.
func (hm *HealthMonitor) checkAllServices() {
	services := hm.registry.ListAll()

	for _, service := range services {
		status, health := hm.checkService(service)

		// Update registry if changed (triggers observers automatically)
		if status != service.Status || health != service.Health {
			if err := hm.registry.UpdateStatus(service.Name, status, health); err != nil {
				// Log error but continue checking other services
				fmt.Fprintf(os.Stderr, "Health monitor: failed to update status for %s: %v\n", service.Name, err)
			}
		}
	}
}

// checkService performs health checks on a single service.
func (hm *HealthMonitor) checkService(service *registry.ServiceRegistryEntry) (status, health string) {
	// Check 1: Process alive?
	if service.PID > 0 {
		exists, err := process.PidExists(int32(service.PID))
		if err != nil || !exists {
			return "error", "unhealthy" // Process died
		}

		// Verify process isn't zombie/dead
		p, err := process.NewProcess(int32(service.PID))
		if err == nil {
			statuses, err := p.Status()
			if err == nil && len(statuses) > 0 {
				// Check if process is zombie or dead
				for _, st := range statuses {
					if st == "Z" || st == "D" { // Z=zombie, D=dead/uninterruptible sleep
						return "error", "unhealthy"
					}
				}
			}
		}
	}

	// Check 2: Port listening?
	portListening := isPortListening(service.Port)
	
	// If service was running before (status="running") but port is no longer listening
	if service.Status == "running" && !portListening {
		return "error", "unhealthy" // Service crashed or stopped
	}
	
	// If port is not listening and service is still starting
	if !portListening {
		return "starting", "unknown" // Not ready yet
	}

	// Check 3: HTTP health (optional)
	if httpHealthy, ok := checkHTTPHealth(service.Port); ok {
		if httpHealthy {
			return "running", "healthy"
		}
		return "running", "unhealthy" // Responding but unhealthy
	}

	// Fall back: process + port = healthy
	return "running", "healthy"
}

// isPortListening checks if a port is accepting connections.
func isPortListening(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// checkHTTPHealth tries common HTTP health endpoints.
func checkHTTPHealth(port int) (healthy bool, checked bool) {
	endpoints := []string{"/health", "/healthz", "/api/health"}

	client := &http.Client{Timeout: 2 * time.Second}

	for _, endpoint := range endpoints {
		url := fmt.Sprintf("http://localhost:%d%s", port, endpoint)
		resp, err := client.Get(url)
		if err != nil {
			continue // Endpoint doesn't exist, try next
		}
		defer resp.Body.Close()

		// If we got a 404, the endpoint doesn't exist - try next
		if resp.StatusCode == http.StatusNotFound {
			continue
		}

		// Found health endpoint (got a response other than 404)
		return resp.StatusCode >= 200 && resp.StatusCode < 300, true
	}

	// No health endpoint found
	return false, false
}
