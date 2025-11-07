package healthmonitor

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/registry"
	"github.com/jongio/azd-app/cli/src/internal/service"
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
	// Shared HTTP client for health checks
	httpClient = &http.Client{Timeout: 2 * time.Second}
	
	// criticalPatterns contains fatal error patterns that indicate complete service failure
	criticalPatterns = []string{
		// Process crashes
		"panic:",
		"fatal error:",
		"fatal:",
		"segmentation fault",
		"core dumped",
		"stack overflow",
		
		// Port/Network binding failures
		"address already in use",
		"eaddrinuse",
		"port.*already in use",
		"failed to bind",
		"error: listen",
		"bind: address already in use",
		"listen eaddrinuse",
		
		// Application startup failures
		"application failed to start",
		"failed to start",
		"startup failed",
		"initialization failed",
		"failed to initialize",
		"bootstrap failed",
		
		// Module/Import errors (Node.js, Python, etc.)
		"cannot find module",
		"modulenotfounderror",
		"importerror:",
		"no module named",
		"module not found",
		"cannot import name",
		"could not find or load main class",
		
		// Syntax and compilation errors
		"syntaxerror",
		"unexpected token",
		"unexpected identifier",
		"parse error",
		"compilation failed",
		"build failed",
		"cannot compile",
		"syntax error",
		
		// Database connection failures
		"connection refused",
		"econnrefused",
		"cannot connect to database",
		"database connection failed",
		"failed to connect to database",
		"connection timeout",
		"no connection to server",
		
		// Authentication/Authorization failures
		"authentication failed",
		"authorization failed",
		"access denied",
		"permission denied",
		"credentials invalid",
		"unauthorized",
		
		// Out of memory
		"out of memory",
		"oom",
		"outofmemoryerror",
		"cannot allocate memory",
		"memory limit exceeded",
	}
	
	// warningPatterns contains non-fatal errors that indicate potential issues
	warningPatterns = []string{
		// Configuration issues
		"configuration error",
		"config error",
		"invalid configuration",
		"missing required",
		"environment variable.*not set",
		"missing environment variable",
		
		// Deprecation warnings
		"deprecated",
		"deprecation warning",
		
		// Performance issues
		"timeout",
		"request timeout",
		"response timeout",
		"slow query",
		"high memory usage",
		"high cpu usage",
		
		// API/Service issues
		"service unavailable",
		"503",
		"502 bad gateway",
		"504 gateway timeout",
		"api error",
		"rate limit",
		"too many requests",
		"429",
		
		// SSL/TLS issues
		"certificate",
		"ssl error",
		"tls handshake",
		"certificate verify failed",
		"certificate has expired",
		
		// File system issues
		"no such file or directory",
		"enoent",
		"file not found",
		"cannot read file",
		"permission denied",
		"disk full",
		"no space left",
		
		// Docker/Container issues
		"container.*exited",
		"container.*failed",
		"image not found",
		"pull access denied",
		
		// Cloud provider specific
		"throttling",
		"quota exceeded",
		"resource not found",
		"service limit",
		
		// Framework specific errors
		// ASP.NET
		"unhandled exception",
		"system.exception",
		
		// Express.js
		"express deprecated",
		
		// Django
		"django.core.exceptions",
		"operationalerror",
		
		// Spring Boot
		"error starting applicationcontext",
		"bean creation exception",
		
		// FastAPI
		"validation error",
		"422 unprocessable entity",
	}
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
	hm.stopChan = make(chan struct{}) // Recreate channel for potential restart
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
		status, health, errorMsg := hm.checkService(service)

		// Update registry if status/health changed, or if we have a new error message
		if status != service.Status || health != service.Health || (errorMsg != "" && errorMsg != service.Error) {
			// Update the service entry with new status
			if err := hm.registry.UpdateStatusWithError(service.Name, status, health, errorMsg); err != nil {
				// Log error but continue checking other services
				fmt.Fprintf(os.Stderr, "Health monitor: failed to update status for %s: %v\n", service.Name, err)
			}
		}
	}
}

// checkService performs health checks on a single service.
// Returns (status, health, errorMessage).
func (hm *HealthMonitor) checkService(service *registry.ServiceRegistryEntry) (status, health, errorMsg string) {
	// Check 1: Process alive?
	processAlive := false
	if service.PID > 0 {
		exists, err := process.PidExists(int32(service.PID))
		if err != nil || !exists {
			// Process definitely died or error checking
			return "error", "unhealthy", "Process not found"
		}

		// Verify process isn't zombie/dead
		p, err := process.NewProcess(int32(service.PID))
		if err != nil {
			// Can't get process info - likely dead
			return "error", "unhealthy", "Cannot access process"
		}
		
		statuses, err := p.Status()
		if err == nil && len(statuses) > 0 {
			// Check if process is zombie (Z state indicates dead process)
			for _, st := range statuses {
				if st == "Z" { // Z=zombie
					return "error", "unhealthy", "Process is zombie"
				}
			}
		}
		
		processAlive = true
	}

	// Check 2: Port listening? (Critical check - process can be alive but app crashed/failed to start)
	portListening := false
	if service.Port > 0 {
		portListening = isPortListening(service.Port)
		
		// IMPORTANT: Process being alive doesn't mean the app is healthy!
		// The app could have crashed at runtime or failed to bind to the port.
		// Port listening is the most reliable indicator of actual service health.
		
		if !portListening {
			// Port not listening - this is fishy if process is alive
			// Check logs for obvious errors to provide better diagnostics
			var logError string
			if processAlive {
				// Process running but port not listening - check for "fishy" patterns
				logError = hm.checkLogsForErrors(service.Name)
			}
			
			if logError != "" {
				// Found a fatal error in logs - service failed to start
				return "error", "unhealthy", logError
			}
			
			// No obvious error found
			if processAlive {
				// Process is running but port isn't listening
				// This means: app crashed at runtime, failed to start, or binding failed
				if service.Status == "running" || service.Status == "ready" {
					// Was running before, now not listening = runtime crash
					return "error", "unhealthy", fmt.Sprintf("Port %d not listening (runtime crash suspected)", service.Port)
				}
				// Still trying to start up (give it time to bind to port)
				return "starting", "unknown", ""
			} else {
				// Neither process nor port - completely dead
				return "error", "unhealthy", "Process and port both unavailable"
			}
		}
		// Port is listening - app successfully started and bound to port
	}

	// Check 3: HTTP health check (optional - best indicator if available)
	if service.Port > 0 && portListening {
		if httpHealthy, ok := checkHTTPHealth(service.Port); ok {
			// Service has a health endpoint and responded
			if httpHealthy {
				return "running", "healthy", ""
			}
			// Health endpoint returned error status (500, 503, etc.)
			// Process is running and port is open, but app reports unhealthy
			return "running", "unhealthy", "Health endpoint returned error status"
		}
		// No HTTP health endpoint found, fall through to port-based check
	}

	// Final decision logic:
	// For services with ports: port listening = healthy (most reliable)
	// For services without ports: process alive = healthy (best we can do)
	if service.Port > 0 {
		if portListening {
			// Port is listening, no health endpoint or health endpoint not found
			// This is normal - assume healthy
			return "running", "healthy", ""
		}
		// Port expected but not listening = unhealthy
		return "error", "unhealthy", fmt.Sprintf("Port %d not listening", service.Port)
	} else if processAlive {
		// No port configured, but process is alive
		// We have to trust the process check (less reliable but only option)
		return "running", "healthy", ""
	}

	// No reliable way to check health (no PID, no port)
	return "unknown", "unknown", "No PID or port to check"
}

// LogSeverity indicates the severity level of detected log errors.
type LogSeverity int

const (
	SeverityNone LogSeverity = iota
	SeverityWarning
	SeverityCritical
)

// LogError represents a detected error in logs with severity.
type LogError struct {
	Message  string
	Severity LogSeverity
	Pattern  string // The pattern that matched
}

// checkLogsForErrors scans recent logs for error patterns.
// Returns error message with severity prefix, or empty string if no errors found.
func (hm *HealthMonitor) checkLogsForErrors(serviceName string) string {
	logManager := service.GetLogManager(hm.projectDir)
	if logManager == nil {
		return ""
	}

	buffer, ok := logManager.GetBuffer(serviceName)
	if !ok || buffer == nil {
		return ""
	}

	// Get logs from last 30 seconds
	since := time.Now().Add(-30 * time.Second)
	logs := buffer.GetSince(since)

	// Check for critical errors first (these take precedence)
	for _, entry := range logs {
		lowerLine := strings.ToLower(entry.Message)
		for _, pattern := range criticalPatterns {
			if matchesPattern(lowerLine, pattern) {
				// Found critical error - return with severity prefix
				msg := truncateMessage(entry.Message, 150)
				return "⛔ " + msg
			}
		}
	}

	// Check for warnings if no critical errors found
	for _, entry := range logs {
		lowerLine := strings.ToLower(entry.Message)
		for _, pattern := range warningPatterns {
			if matchesPattern(lowerLine, pattern) {
				// Found warning - return with warning prefix
				msg := truncateMessage(entry.Message, 150)
				return "⚠️ " + msg
			}
		}
	}

	return ""
}

// matchesPattern checks if text matches a pattern (supports basic regex with .* wildcards).
func matchesPattern(text, pattern string) bool {
	// If pattern contains .* it's a regex pattern
	if strings.Contains(pattern, ".*") {
		// Compile and match regex
		re, err := regexp.Compile(pattern)
		if err != nil {
			// Fallback to simple contains if regex is invalid
			return strings.Contains(text, strings.ReplaceAll(pattern, ".*", ""))
		}
		return re.MatchString(text)
	}
	// Simple substring match
	return strings.Contains(text, pattern)
}

// truncateMessage truncates a message to maxLen characters.
func truncateMessage(msg string, maxLen int) string {
	// Remove leading/trailing whitespace
	msg = strings.TrimSpace(msg)
	
	if len(msg) <= maxLen {
		return msg
	}
	
	// Try to truncate at a word boundary
	truncated := msg[:maxLen]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > maxLen-20 { // Only use word boundary if it's not too far back
		truncated = truncated[:lastSpace]
	}
	
	return truncated + "..."
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
// If no health endpoint exists, tries the root endpoint to verify service responds.
func checkHTTPHealth(port int) (healthy bool, checked bool) {
	healthEndpoints := []string{"/health", "/healthz", "/api/health"}

	// First try dedicated health endpoints
	for _, endpoint := range healthEndpoints {
		url := fmt.Sprintf("http://localhost:%d%s", port, endpoint)
		resp, err := httpClient.Get(url)
		if err != nil {
			continue // Endpoint doesn't exist, try next
		}

		// If we got a 404, the endpoint doesn't exist - try next
		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			continue
		}

		// Found health endpoint (got a response other than 404)
		healthy := resp.StatusCode >= 200 && resp.StatusCode < 300
		resp.Body.Close()
		return healthy, true
	}

	// No health endpoint found - try root endpoint as fallback
	// This verifies the service is actually responding to HTTP requests
	rootURL := fmt.Sprintf("http://localhost:%d/", port)
	resp, err := httpClient.Get(rootURL)
	if err != nil {
		// Service not responding to HTTP at all
		return false, false
	}
	defer resp.Body.Close()
	
	// Service responded - consider it healthy if not a server error
	// Accept 2xx, 3xx, 4xx as "responding" (404 is fine, means server works)
	// Only 5xx indicates unhealthy
	healthy = resp.StatusCode < 500
	return healthy, true
}
