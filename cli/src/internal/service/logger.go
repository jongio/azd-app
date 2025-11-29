// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ServiceLogger handles multiplexed log output from multiple services.
type ServiceLogger struct {
	mu         sync.Mutex
	verbose    bool
	colors     map[string]string
	colorIndex int
}

// ANSI color codes for service output
var colorCodes = []string{
	"\033[36m", // Cyan
	"\033[33m", // Yellow
	"\033[35m", // Magenta
	"\033[32m", // Green
	"\033[34m", // Blue
	"\033[31m", // Red
	"\033[96m", // Bright Cyan
	"\033[93m", // Bright Yellow
	"\033[95m", // Bright Magenta
	"\033[92m", // Bright Green
}

const (
	colorReset = "\033[0m"
	colorBold  = "\033[1m"
	colorGray  = "\033[90m"
)

// NewServiceLogger creates a new logger for service orchestration.
func NewServiceLogger(verbose bool) *ServiceLogger {
	return &ServiceLogger{
		verbose:    verbose,
		colors:     make(map[string]string),
		colorIndex: 0,
	}
}

// getServiceColor returns a consistent color for a service.
func (l *ServiceLogger) getServiceColor(serviceName string) string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.getServiceColorUnsafe(serviceName)
}

// getServiceColorUnsafe returns a consistent color for a service without locking.
// Must be called with mutex already held.
func (l *ServiceLogger) getServiceColorUnsafe(serviceName string) string {
	if color, exists := l.colors[serviceName]; exists {
		return color
	}

	color := colorCodes[l.colorIndex%len(colorCodes)]
	l.colors[serviceName] = color
	l.colorIndex++

	return color
}

// FormatLogEntry formats a log line with service prefix and color.
func (l *ServiceLogger) FormatLogEntry(serviceName string, message string) string {
	timestamp := time.Now().Format("15:04:05")
	color := l.getServiceColor(serviceName)

	// Format: HH:MM:SS service-name │ message
	return fmt.Sprintf("%s%s%s %s%-15s%s %s│%s %s",
		colorGray, timestamp, colorReset,
		color, serviceName, colorReset,
		colorGray, colorReset,
		message)
}

// LogService logs a message from a specific service.
func (l *ServiceLogger) LogService(serviceName string, message string) {
	// Get the color first (this will lock and unlock the mutex)
	color := l.getServiceColor(serviceName)

	// Format the message without calling getServiceColor again
	timestamp := time.Now().Format("15:04:05")
	formatted := fmt.Sprintf("%s%s%s %s%-15s%s %s│%s %s",
		colorGray, timestamp, colorReset,
		color, serviceName, colorReset,
		colorGray, colorReset,
		message)

	fmt.Println(formatted)
}

// LogInfo logs an informational message (no service prefix).
func (l *ServiceLogger) LogInfo(message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("%s%s%s %s\n", colorGray, timestamp, colorReset, message)
}

// LogSuccess logs a success message with green color.
func (l *ServiceLogger) LogSuccess(serviceName string, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("15:04:05")
	color := l.getServiceColorUnsafe(serviceName)

	fmt.Printf("%s%s%s %s%-15s%s %s✓%s %s\n",
		colorGray, timestamp, colorReset,
		color, serviceName, colorReset,
		"\033[92m", colorReset,
		message)
}

// LogError logs an error message with red color.
func (l *ServiceLogger) LogError(serviceName string, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("15:04:05")
	color := l.getServiceColorUnsafe(serviceName)

	fmt.Printf("%s%s%s %s%-15s%s %s✗%s %s\n",
		colorGray, timestamp, colorReset,
		color, serviceName, colorReset,
		"\033[91m", colorReset,
		message)
}

// LogWarning logs a warning message with yellow color.
func (l *ServiceLogger) LogWarning(serviceName string, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("15:04:05")
	color := l.getServiceColorUnsafe(serviceName)

	fmt.Printf("%s%s%s %s%-15s%s %s⚠%s  %s\n",
		colorGray, timestamp, colorReset,
		color, serviceName, colorReset,
		"\033[93m", colorReset,
		message)
}

// LogVerbose logs a verbose message (only if verbose mode is enabled).
func (l *ServiceLogger) LogVerbose(serviceName string, message string) {
	if !l.verbose {
		return
	}

	l.LogService(serviceName, message)
}

// LogStartup logs the startup phase label.
func (l *ServiceLogger) LogStartup(serviceCount int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	fmt.Println("Starting services...")
}

// LogSummary logs the service URLs after startup.
func (l *ServiceLogger) LogSummary(urls map[string]string) {
	fmt.Println()
	if len(urls) > 0 {
		for name, url := range urls {
			fmt.Printf("  \033[32m✓\033[0m %-18s  %s\n", name, url)
		}
	}
}

// LogReady logs the ready phase label.
func (l *ServiceLogger) LogReady() {
	fmt.Println()
	fmt.Println("Ready")
}

// StreamLogs streams logs from multiple services to the console.
func StreamLogs(processes map[string]*ServiceProcess, logger *ServiceLogger) {
	for name, process := range processes {
		// Start goroutines to read stdout and stderr
		go func(serviceName string, proc *ServiceProcess) {
			outputChan := make(chan string, 100)

			go ReadServiceOutput(proc.Stdout, outputChan)
			go ReadServiceOutput(proc.Stderr, outputChan)

			for line := range outputChan {
				// Filter empty lines
				if strings.TrimSpace(line) != "" {
					logger.LogService(serviceName, line)
				}
			}
		}(name, process)
	}
}
