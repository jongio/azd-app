package healthcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ansiEscapeRegex matches ANSI/VT100 escape sequences used for terminal formatting
// (e.g., colour codes, cursor movement).  Stripped to prevent log injection (CWE-117).
var ansiEscapeRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// isLocalhostURL checks if a URL targets localhost/loopback addresses only.
// Health check URLs from azure.yaml are restricted to local services to prevent SSRF.
func isLocalhostURL(urlStr string) error {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	host := parsed.Hostname()
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "[::1]" {
		return nil
	}

	// Check if it resolves to a loopback address
	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return nil
	}

	return fmt.Errorf("health check URL must target localhost (got %q) - non-local URLs are blocked for security", host)
}

func (c *HealthChecker) performHTTPCheck(ctx context.Context, urlStr string) *httpHealthCheckResult {
	// Restrict health check URLs to localhost to prevent SSRF
	if err := isLocalhostURL(urlStr); err != nil {
		return &httpHealthCheckResult{
			Endpoint: urlStr,
			Status:   HealthStatusUnhealthy,
			Error:    err.Error(),
		}
	}

	startTime := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return &httpHealthCheckResult{
			Endpoint: urlStr,
			Status:   HealthStatusUnhealthy,
			Error:    fmt.Sprintf("failed to create request: %v", err),
		}
	}

	resp, err := c.httpClient.Do(req)
	responseTime := time.Since(startTime)

	if err != nil {
		return &httpHealthCheckResult{
			Endpoint:     urlStr,
			ResponseTime: responseTime,
			Status:       HealthStatusUnhealthy,
			Error:        fmt.Sprintf("connection failed: %v", err),
		}
	}

	// Read and close body
	limitedReader := io.LimitReader(resp.Body, maxResponseBodySize)
	body, readErr := io.ReadAll(limitedReader)
	if closeErr := resp.Body.Close(); closeErr != nil {
		slog.Warn("Failed to close response body", "err", closeErr, "url", urlStr)
	}

	result := &httpHealthCheckResult{
		Endpoint:     urlStr,
		ResponseTime: responseTime,
		StatusCode:   resp.StatusCode,
		Details:      make(map[string]any),
	}

	// Determine status based on HTTP status code
	result.Status = c.statusFromHTTPCode(resp.StatusCode)

	// Add suggestion for error responses
	if resp.StatusCode >= 400 {
		result.Details["suggestion"] = suggestHTTPErrorAction(resp.StatusCode)

		// Try to parse error details from response body
		if readErr == nil && len(body) > 0 {
			if errorDetails := parseErrorDetailsFromBody(body); errorDetails != "" {
				result.Error = errorDetails
			}
		}
	}

	// Try to parse response body for additional details
	if readErr == nil && len(body) > 0 && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		c.parseHealthResponseBody(body, result)
	}

	return result
}

// performCommandCheck executes a command for health check (CMD format).
// For container services, the command is executed inside the container using docker exec.

func (c *HealthChecker) tryHTTPHealthCheck(ctx context.Context, port int) *httpHealthCheckResult {
	cacheKey := fmt.Sprintf("port:%d", port)

	// Ensure endpointCache is initialized (for backward compatibility with tests)
	c.mu.Lock()
	if c.endpointCache == nil {
		c.endpointCache = make(map[string]string)
	}
	c.mu.Unlock()

	// Check if we have a cached endpoint for this port
	c.mu.RLock()
	cachedEndpoint, hasCached := c.endpointCache[cacheKey]
	c.mu.RUnlock()

	// If we have a cached endpoint, ONLY check that endpoint first
	if hasCached {
		// Special marker indicates no HTTP endpoint exists - skip to TCP fallback
		if cachedEndpoint == endpointCacheNone {
			slog.Debug("Skipping HTTP check - no endpoint found in previous discovery", "port", port)
			return nil
		}

		result := c.checkSingleEndpoint(ctx, port, cachedEndpoint)
		if result != nil && result.Status == HealthStatusHealthy {
			return result
		}
		// Cached endpoint failed - clear cache and rediscover
		c.mu.Lock()
		delete(c.endpointCache, cacheKey)
		c.mu.Unlock()
		slog.Debug("Cached health endpoint failed, will rediscover on next check", "port", port, "cached_endpoint", cachedEndpoint)
		// Fall through to discovery - gives one chance to find a working endpoint
	}

	// No cached endpoint - perform endpoint discovery
	slog.Debug("Discovering health endpoint (first check or cache miss)", "port", port)

	// Build list of endpoints to try, prioritizing common ones
	endpoints := []string{c.defaultEndpoint}
	for _, path := range commonHealthPaths {
		if path != c.defaultEndpoint {
			endpoints = append(endpoints, path)
		}
	}

	// Track the last non-nil result in case no healthy endpoint is found
	var lastResult *httpHealthCheckResult

	for _, endpoint := range endpoints {
		// Check context before each attempt
		if ctx.Err() != nil {
			return nil
		}

		result := c.checkSingleEndpoint(ctx, port, endpoint)
		if result != nil {
			// If healthy, cache and return immediately - stop discovery
			if result.Status == HealthStatusHealthy {
				c.mu.Lock()
				c.endpointCache[cacheKey] = endpoint
				c.mu.Unlock()
				slog.Debug("Discovered and cached health endpoint", "port", port, "endpoint", endpoint)
				return result
			}
			// Keep track of last non-nil result for fallback
			lastResult = result
		}
	}

	// No healthy endpoint found during discovery
	// Cache a marker to skip HTTP checks in future (will fall back to TCP/process checks)
	if lastResult == nil {
		c.mu.Lock()
		c.endpointCache[cacheKey] = endpointCacheNone
		c.mu.Unlock()
		slog.Debug("No HTTP health endpoint found, will use TCP fallback", "port", port)
	}

	return lastResult
}

// checkSingleEndpoint performs a single HTTP health check on a specific endpoint.
func (c *HealthChecker) checkSingleEndpoint(ctx context.Context, port int, endpoint string) *httpHealthCheckResult {
	url := fmt.Sprintf("http://localhost:%d%s", port, endpoint)

	startTime := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil
	}

	resp, err := c.httpClient.Do(req)
	responseTime := time.Since(startTime)

	if err != nil {
		// Check if error is due to context cancellation
		if ctx.Err() != nil {
			return nil
		}
		return nil
	}

	limitedReader := io.LimitReader(resp.Body, maxResponseBodySize)
	body, readErr := io.ReadAll(limitedReader)
	closeErr := resp.Body.Close()
	if closeErr != nil {
		slog.Warn("Failed to close response body", "err", closeErr, "url", url)
	}

	// Skip 404 and 400 responses - these indicate endpoint doesn't exist
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest {
		return nil
	}

	result := &httpHealthCheckResult{
		Endpoint:     url,
		ResponseTime: responseTime,
		StatusCode:   resp.StatusCode,
		Status:       c.statusFromHTTPCode(resp.StatusCode),
		Details:      make(map[string]any),
	}

	// Add suggestion for error responses
	if resp.StatusCode >= 400 {
		result.Details["suggestion"] = suggestHTTPErrorAction(resp.StatusCode)

		// Try to parse error details from response body
		if readErr == nil && len(body) > 0 {
			if errorDetails := parseErrorDetailsFromBody(body); errorDetails != "" {
				result.Error = errorDetails
			}
		}
	}

	if readErr == nil && len(body) > 0 && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		c.parseHealthResponseBody(body, result)
	}

	return result
}

// statusFromHTTPCode determines health status from HTTP status code.
func (c *HealthChecker) statusFromHTTPCode(statusCode int) HealthStatus {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return HealthStatusHealthy
	case statusCode >= 300 && statusCode < 400:
		return HealthStatusHealthy // Redirects OK
	case statusCode >= 500:
		return HealthStatusUnhealthy
	default:
		return HealthStatusDegraded
	}
}

// parseHealthResponseBody parses JSON response body for health details.
func (c *HealthChecker) parseHealthResponseBody(body []byte, result *httpHealthCheckResult) {
	var details map[string]any
	if err := json.Unmarshal(body, &details); err == nil {
		result.Details = details

		if status, ok := details["status"].(string); ok {
			switch strings.ToLower(status) {
			case "healthy", "ok", "up":
				result.Status = HealthStatusHealthy
			case "degraded", "warning":
				result.Status = HealthStatusDegraded
			case "unhealthy", "down", statusError:
				result.Status = HealthStatusUnhealthy
			}
		}
	}
}

// sanitizeResponseBody cleans HTTP response body content before it appears in error
// messages or structured logs, preventing log injection (CWE-117).
//
// It performs three transformations in order:
//  1. Strips ANSI escape sequences (colour codes, cursor movement, etc.)
//  2. Removes C0 control characters (0x00-0x1F, preserving \t, \n, \r) and
//     C1 control characters (0x80-0x9F)
//  3. Truncates to maxLen runes, appending "... (truncated)" when truncation occurs
func sanitizeResponseBody(body string, maxLen int) string {
	// 1. Strip ANSI escape sequences
	body = ansiEscapeRegex.ReplaceAllString(body, "")

	// 2. Strip C0 (except tab/newline/CR) and C1 control characters
	body = strings.Map(func(r rune) rune {
		if (r < 0x20 && r != '\t' && r != '\n' && r != '\r') || (r >= 0x80 && r <= 0x9F) {
			return -1 // drop character
		}
		return r
	}, body)

	// 3. Truncate to maxLen runes (rune-safe, avoids splitting multi-byte sequences)
	runes := []rune(body)
	if len(runes) > maxLen {
		return string(runes[:maxLen]) + "... (truncated)"
	}
	return body
}

// performProcessHealthCheck handles health checks for process-type services.
