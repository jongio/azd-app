package testutil

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

// CreateHealthServer creates a test HTTP server with configurable response
func CreateHealthServer(statusCode int, body string, delay time.Duration) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if delay > 0 {
			time.Sleep(delay)
		}
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}))
}

// GetServerPort extracts the port from a test server
func GetServerPort(server *httptest.Server) int {
	return server.Listener.Addr().(*net.TCPAddr).Port
}

// Contains checks if string s contains substr (case-sensitive)
func Contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// ContainsLower checks if string s contains substr (case-insensitive)
func ContainsLower(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// WaitForCondition polls a condition function until it returns true or timeout expires.
// Returns true if condition was met, false if timeout occurred.
func WaitForCondition(timeout time.Duration, checkInterval time.Duration, condition func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(checkInterval)
	}
	return false
}
