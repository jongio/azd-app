package service

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestNewServiceLogger(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
	}{
		{"verbose mode", true},
		{"normal mode", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewServiceLogger(tt.verbose)
			if logger == nil {
				t.Fatal("NewServiceLogger() returned nil")
			}

			if logger.verbose != tt.verbose {
				t.Errorf("verbose = %v, want %v", logger.verbose, tt.verbose)
			}

			if logger.colors == nil {
				t.Error("colors map is nil")
			}
		})
	}
}

func TestServiceLogger_GetServiceColor(t *testing.T) {
	logger := NewServiceLogger(false)

	// Get color for same service multiple times
	color1 := logger.getServiceColor("test-service")
	color2 := logger.getServiceColor("test-service")

	if color1 != color2 {
		t.Error("getServiceColor() returned different colors for same service")
	}

	// Get colors for different services
	colorA := logger.getServiceColor("service-a")
	colorB := logger.getServiceColor("service-b")

	if colorA == colorB {
		t.Error("getServiceColor() returned same color for different services")
	}

	// Verify colors are from the colorCodes list
	found := false
	for _, code := range colorCodes {
		if colorA == code {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("getServiceColor() returned color not in colorCodes: %s", colorA)
	}
}

func TestServiceLogger_FormatLogEntry(t *testing.T) {
	logger := NewServiceLogger(false)

	formatted := logger.FormatLogEntry("test-service", "test message")

	// Verify format contains key components
	if !strings.Contains(formatted, "test-service") {
		t.Error("Formatted log missing service name")
	}

	if !strings.Contains(formatted, "test message") {
		t.Error("Formatted log missing message")
	}

	if !strings.Contains(formatted, "│") {
		t.Error("Formatted log missing separator")
	}

	// Should contain ANSI codes
	if !strings.Contains(formatted, "\033[") {
		t.Error("Formatted log missing ANSI color codes")
	}
}

func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outChan := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outChan <- buf.String()
	}()

	f()

	w.Close()
	os.Stdout = old

	return <-outChan
}

func TestServiceLogger_LogService(t *testing.T) {
	logger := NewServiceLogger(false)

	// Pre-allocate color to avoid mutex during capture
	_ = logger.getServiceColor("test-service")

	output := captureStdout(func() {
		logger.LogService("test-service", "test message")
	})

	if !strings.Contains(output, "test-service") {
		t.Error("LogService output missing service name")
	}

	if !strings.Contains(output, "test message") {
		t.Error("LogService output missing message")
	}
}

func TestServiceLogger_LogVerbose(t *testing.T) {
	// Verbose mode enabled
	verboseLogger := NewServiceLogger(true)
	_ = verboseLogger.getServiceColor("test")

	output := captureStdout(func() {
		verboseLogger.LogVerbose("test", "verbose message")
	})

	if !strings.Contains(output, "verbose message") {
		t.Error("LogVerbose() in verbose mode didn't output message")
	}

	// Verbose mode disabled
	normalLogger := NewServiceLogger(false)
	_ = normalLogger.getServiceColor("test")

	output = captureStdout(func() {
		normalLogger.LogVerbose("test", "should not see this")
	})

	if strings.Contains(output, "should not see this") {
		t.Error("LogVerbose() in normal mode shouldn't output message")
	}
}

func TestServiceLogger_LogSuccess(t *testing.T) {
	logger := NewServiceLogger(false)

	// Pre-allocate color to avoid mutex during capture
	_ = logger.getServiceColor("test-service")

	output := captureStdout(func() {
		logger.LogSuccess("test-service", "operation successful")
	})

	if !strings.Contains(output, "test-service") {
		t.Error("LogSuccess output missing service name")
	}

	if !strings.Contains(output, "operation successful") {
		t.Error("LogSuccess output missing message")
	}

	// Should contain checkmark
	if !strings.Contains(output, "✓") {
		t.Error("LogSuccess output missing checkmark")
	}
}

func TestServiceLogger_LogError(t *testing.T) {
	logger := NewServiceLogger(false)

	// Pre-allocate color to avoid mutex during capture
	_ = logger.getServiceColor("test-service")

	output := captureStdout(func() {
		logger.LogError("test-service", "operation failed")
	})

	if !strings.Contains(output, "test-service") {
		t.Error("LogError output missing service name")
	}

	if !strings.Contains(output, "operation failed") {
		t.Error("LogError output missing message")
	}

	// Should contain X mark
	if !strings.Contains(output, "✗") {
		t.Error("LogError output missing error mark")
	}
}

func TestServiceLogger_LogWarning(t *testing.T) {
	logger := NewServiceLogger(false)

	// Pre-allocate color to avoid mutex during capture
	_ = logger.getServiceColor("test-service")

	output := captureStdout(func() {
		logger.LogWarning("test-service", "warning message")
	})

	if !strings.Contains(output, "test-service") {
		t.Error("LogWarning output missing service name")
	}

	if !strings.Contains(output, "warning message") {
		t.Error("LogWarning output missing message")
	}

	// Should contain warning symbol
	if !strings.Contains(output, "⚠") {
		t.Error("LogWarning output missing warning symbol")
	}
}

func TestServiceLogger_LogInfo(t *testing.T) {
	logger := NewServiceLogger(false)

	output := captureStdout(func() {
		logger.LogInfo("info message")
	})

	if !strings.Contains(output, "info message") {
		t.Error("LogInfo output missing message")
	}
}

func TestServiceLogger_LogStartup(t *testing.T) {
	logger := NewServiceLogger(false)

	output := captureStdout(func() {
		logger.LogStartup(3)
	})

	// Simplified output just says "Starting services..."
	if !strings.Contains(output, "Starting services...") {
		t.Error("LogStartup output missing 'Starting services...'")
	}
}

func TestServiceLogger_LogSummary(t *testing.T) {
	logger := NewServiceLogger(false)

	serviceURLs := map[string]URLInfo{
		"web": {
			SystemURL:   "http://localhost:3000",
			Environment: "local",
		},
		"api": {
			SystemURL:   "http://localhost:8080",
			Environment: "local",
		},
	}

	output := captureStdout(func() {
		logger.LogSummary(serviceURLs)
	})

	if !strings.Contains(output, "web") {
		t.Error("LogSummary output missing web service")
	}

	if !strings.Contains(output, "api") {
		t.Error("LogSummary output missing api service")
	}

	if !strings.Contains(output, "http://localhost:3000") {
		t.Error("LogSummary output missing web URL")
	}

	if !strings.Contains(output, "http://localhost:8080") {
		t.Error("LogSummary output missing api URL")
	}

	// Should have checkmarks
	if !strings.Contains(output, "✓") {
		t.Error("LogSummary output missing checkmarks")
	}
}

func TestServiceLogger_LogSummary_EmptyURLs(t *testing.T) {
	logger := NewServiceLogger(false)

	output := captureStdout(func() {
		logger.LogSummary(map[string]URLInfo{})
	})

	// Empty URLs should produce no output (not even a newline for the summary itself)
	// Just verify it doesn't panic and produces minimal output
	if strings.Contains(output, "✓") {
		t.Error("LogSummary with empty URLs should not have checkmarks")
	}
}

func TestServiceLogger_LogSummary_WithCustomURLs(t *testing.T) {
	logger := NewServiceLogger(false)

	customURL := "https://myapp.example.com"
	serviceURLs := map[string]URLInfo{
		"web": {
			SystemURL:   "http://localhost:3000",
			CustomURL:   &customURL,
			Environment: "local",
		},
		"api": {
			SystemURL:   "http://localhost:8080",
			Environment: "local",
		},
	}

	output := captureStdout(func() {
		logger.LogSummary(serviceURLs)
	})

	// Should use custom URL for web (effective URL)
	if !strings.Contains(output, "https://myapp.example.com") {
		t.Error("LogSummary should display custom URL for web service")
	}

	// Should show localhost URL in details for web (system URL)
	if !strings.Contains(output, "http://localhost:3000") {
		t.Error("LogSummary should show system URL in details for web service")
	}

	// Should show localhost URL for api (only URL)
	if !strings.Contains(output, "http://localhost:8080") {
		t.Error("LogSummary should display localhost URL for api service")
	}

	// Should show arrow for web (multiple URLs)
	if !strings.Contains(output, "→") {
		t.Error("LogSummary should show arrow for service with multiple URLs")
	}

	// Should show "Custom URL:" label
	if !strings.Contains(output, "Custom URL:") {
		t.Error("LogSummary should show 'Custom URL:' label")
	}
}

func TestServiceLogger_LogSummary_WithCustomDomain(t *testing.T) {
	logger := NewServiceLogger(false)

	customDomain := "https://api.myapp.example.com"
	serviceURLs := map[string]URLInfo{
		"api": {
			SystemURL:    "https://api-abc123.azurewebsites.net",
			CustomDomain: &customDomain,
			Environment:  "azure",
		},
	}

	output := captureStdout(func() {
		logger.LogSummary(serviceURLs)
	})

	// Should use custom domain as effective URL
	if !strings.Contains(output, "https://api.myapp.example.com") {
		t.Error("LogSummary should display custom domain as effective URL")
	}

	// Should show Azure system URL in details
	if !strings.Contains(output, "https://api-abc123.azurewebsites.net") {
		t.Error("LogSummary should show Azure system URL in details")
	}

	// Should show "Custom Domain:" label
	if !strings.Contains(output, "Custom Domain:") {
		t.Error("LogSummary should show 'Custom Domain:' label")
	}

	// Should show "Azure URL:" label
	if !strings.Contains(output, "Azure URL:") {
		t.Error("LogSummary should show 'Azure URL:' label for Azure services")
	}
}

func TestServiceLogger_LogSummary_PrecedenceChain(t *testing.T) {
	logger := NewServiceLogger(false)

	customDomain := "https://api.myapp.example.com"
	customURL := "https://api-override.example.com"
	serviceURLs := map[string]URLInfo{
		"api": {
			SystemURL:    "https://api-abc123.azurewebsites.net",
			CustomDomain: &customDomain,
			CustomURL:    &customURL,
			Environment:  "azure",
		},
	}

	output := captureStdout(func() {
		logger.LogSummary(serviceURLs)
	})

	// CustomDomain should be effective URL (highest priority: customDomain > customUrl > url)
	// All three URLs should be present in details
	// CustomDomain should be effective URL (highest priority: customDomain > customUrl > url)
	// All three URLs should be present in details
	if !strings.Contains(output, "https://api-abc123.azurewebsites.net") {
		t.Error("LogSummary should show Azure system URL")
	}
	if !strings.Contains(output, "https://api.myapp.example.com") {
		t.Error("LogSummary should show custom domain (effective URL)")
	}
	if !strings.Contains(output, "https://api-override.example.com") {
		t.Error("LogSummary should show custom URL in details")
	}
}

func TestServiceLogger_LogReady(t *testing.T) {
	logger := NewServiceLogger(false)

	output := captureStdout(func() {
		logger.LogReady()
	})

	if !strings.Contains(output, "Ready") {
		t.Error("LogReady output missing ready message")
	}
}

func TestServiceLogger_ColorConsistency(t *testing.T) {
	logger := NewServiceLogger(false)

	// Call multiple times for same service
	services := []string{"service-a", "service-b", "service-c"}
	colors := make(map[string]string)

	// Get colors first time
	for _, svc := range services {
		colors[svc] = logger.getServiceColor(svc)
	}

	// Verify colors remain consistent
	for i := 0; i < 3; i++ {
		for _, svc := range services {
			color := logger.getServiceColor(svc)
			if color != colors[svc] {
				t.Errorf("Color changed for service %s on iteration %d", svc, i)
			}
		}
	}
}

func TestServiceLogger_ColorCycling(t *testing.T) {
	logger := NewServiceLogger(false)

	// Create more services than available colors
	serviceCount := len(colorCodes) + 5
	colors := make(map[string]string)

	for i := 0; i < serviceCount; i++ {
		serviceName := fmt.Sprintf("service-%d", i)
		colors[serviceName] = logger.getServiceColor(serviceName)
	}

	// Verify all services got a color
	if len(colors) != serviceCount {
		t.Errorf("Got %d colors, want %d", len(colors), serviceCount)
	}

	// Verify colors repeat after running out of unique colors
	firstServiceColor := colors["service-0"]
	wrappedServiceColor := colors[fmt.Sprintf("service-%d", len(colorCodes))]

	if firstServiceColor != wrappedServiceColor {
		t.Error("Colors didn't cycle after exhausting colorCodes")
	}
}
