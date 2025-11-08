# Health Monitoring Design

## Overview

This document describes the design and architecture for the `azd app health` command, which provides comprehensive health monitoring capabilities for local development services and Azure-deployed applications.

## Goals

1. **Point-in-Time Health Checks**: Provide instant health status snapshots for all services
2. **Real-Time Streaming**: Enable continuous health monitoring with configurable intervals
3. **Intelligent Detection**: Automatically discover and use optimal health check methods
4. **Fallback Strategy**: Gracefully degrade from HTTP endpoints to port checks to process checks
5. **Integration**: Seamlessly integrate with existing service registry and dashboard
6. **Developer Experience**: Provide actionable information with clear, formatted output
7. **Automation Ready**: Support scripting and CI/CD pipelines with JSON output and exit codes

## Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│                  azd app health Command                      │
└─────────────────────────────────────────────────────────────┘
                            │
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                   Health Monitor                             │
│  - Coordinate health check execution                         │
│  - Manage streaming mode lifecycle                           │
│  - Aggregate and format results                              │
└─────────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        ↓                   ↓                   ↓
┌────────────────┐  ┌────────────────┐  ┌────────────────┐
│ Health Checker │  │ Service        │  │ Output         │
│                │  │ Discovery      │  │ Formatter      │
│ - HTTP checks  │  │                │  │                │
│ - Port checks  │  │ - Registry     │  │ - Text         │
│ - Process chks │  │ - azure.yaml   │  │ - JSON         │
│                │  │ - Filtering    │  │ - Table        │
└────────────────┘  └────────────────┘  └────────────────┘
        │                   │                   │
        └───────────────────┴───────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                  Shared Components                           │
│  - Service Registry (internal/registry)                      │
│  - Service Info (internal/serviceinfo)                       │
│  - Health Check (internal/service/health.go)                 │
│  - Output Utils (internal/output)                            │
└─────────────────────────────────────────────────────────────┘
```

### Data Flow

#### Static Mode (Point-in-Time)

```
User Command
    ↓
Parse Flags & Validate
    ↓
Load Service Registry ────┬──→ .azure/services.json
                          │
Load Service Definitions ─┴──→ azure.yaml
    ↓
Discover Services & Filter
    ↓
Detect Health Endpoints (parallel)
    ├─ Try HTTP /health, /healthz, etc.
    ├─ Fall back to port check
    └─ Fall back to process check
    ↓
Perform Health Checks (parallel)
    ├─ HTTP GET requests
    ├─ TCP connections
    └─ Process status checks
    ↓
Aggregate Results
    ├─ Calculate summary statistics
    ├─ Determine overall status
    └─ Identify issues
    ↓
Format Output
    ├─ Text (colored, formatted)
    ├─ JSON (machine-readable)
    └─ Table (compact view)
    ↓
Display to Console
    ↓
Exit with Status Code
```

#### Streaming Mode (Real-Time)

```
User Command (--stream)
    ↓
Initialize Stream
    ├─ Set up signal handlers
    ├─ Clear terminal (if TTY)
    └─ Display header
    ↓
Start Monitoring Loop
    ↓
┌─────────────────────────┐
│  Every {interval}       │
│  ├─ Perform checks      │
│  ├─ Compare with prev   │
│  ├─ Update display      │
│  ├─ Write JSON stream   │
│  └─ Wait for interval   │
└─────────────────────────┘
    ↓ (on Ctrl+C)
Graceful Shutdown
    ├─ Stop monitoring loop
    ├─ Display final summary
    └─ Exit
```

## Core Types and Interfaces

### HealthCheckResult

```go
// HealthCheckResult represents the result of a single health check
type HealthCheckResult struct {
    ServiceName  string                 `json:"serviceName"`
    Status       HealthStatus           `json:"status"`
    CheckType    HealthCheckType        `json:"checkType"`
    Endpoint     string                 `json:"endpoint,omitempty"`
    ResponseTime time.Duration          `json:"responseTime"`
    StatusCode   int                    `json:"statusCode,omitempty"`
    Error        string                 `json:"error,omitempty"`
    Timestamp    time.Time              `json:"timestamp"`
    Details      map[string]interface{} `json:"details,omitempty"`
    
    // Additional metadata
    Port         int                    `json:"port,omitempty"`
    PID          int                    `json:"pid,omitempty"`
    Uptime       time.Duration          `json:"uptime,omitempty"`
}

// HealthStatus represents the health state of a service
type HealthStatus string

const (
    HealthStatusHealthy   HealthStatus = "healthy"
    HealthStatusDegraded  HealthStatus = "degraded"
    HealthStatusUnhealthy HealthStatus = "unhealthy"
    HealthStatusStarting  HealthStatus = "starting"
    HealthStatusUnknown   HealthStatus = "unknown"
)

// HealthCheckType indicates the method used for health checking
type HealthCheckType string

const (
    HealthCheckTypeHTTP    HealthCheckType = "http"
    HealthCheckTypePort    HealthCheckType = "port"
    HealthCheckTypeProcess HealthCheckType = "process"
)
```

### HealthMonitor

```go
// HealthMonitor coordinates health checking operations
type HealthMonitor struct {
    services      []ServiceInfo
    checker       *HealthChecker
    formatter     *OutputFormatter
    interval      time.Duration
    timeout       time.Duration
    streamMode    bool
    verbose       bool
}

// Check performs health checks on all services
func (m *HealthMonitor) Check(ctx context.Context) (*HealthReport, error)

// Stream continuously monitors services and reports health
func (m *HealthMonitor) Stream(ctx context.Context, output io.Writer) error

// Stop gracefully stops the health monitor
func (m *HealthMonitor) Stop()
```

### HealthChecker

```go
// HealthChecker performs health checks on services
type HealthChecker struct {
    timeout    time.Duration
    httpClient *http.Client
}

// CheckService performs a health check on a single service
func (c *HealthChecker) CheckService(ctx context.Context, svc ServiceInfo) HealthCheckResult

// detectHealthEndpoint discovers the health endpoint for a service
func (c *HealthChecker) detectHealthEndpoint(svc ServiceInfo) (string, HealthCheckType, error)

// performHTTPCheck executes an HTTP health check
func (c *HealthChecker) performHTTPCheck(ctx context.Context, url string) HealthCheckResult

// performPortCheck executes a port health check
func (c *HealthChecker) performPortCheck(ctx context.Context, port int) HealthCheckResult

// performProcessCheck executes a process health check
func (c *HealthChecker) performProcessCheck(ctx context.Context, pid int) HealthCheckResult
```

### HealthReport

```go
// HealthReport contains aggregated health check results
type HealthReport struct {
    Timestamp time.Time           `json:"timestamp"`
    Project   string              `json:"project"`
    Services  []HealthCheckResult `json:"services"`
    Summary   HealthSummary       `json:"summary"`
}

// HealthSummary provides overall health statistics
type HealthSummary struct {
    Total      int          `json:"total"`
    Healthy    int          `json:"healthy"`
    Degraded   int          `json:"degraded"`
    Unhealthy  int          `json:"unhealthy"`
    Unknown    int          `json:"unknown"`
    Overall    HealthStatus `json:"overall"`
}
```

## Health Check Strategy

### Detection Algorithm

The health checker uses a cascading strategy to find the best health check method:

```go
func (c *HealthChecker) detectHealthEndpoint(svc ServiceInfo) (string, HealthCheckType, error) {
    // 1. Check explicit configuration in azure.yaml
    if svc.HealthCheck != nil && svc.HealthCheck.Endpoint != "" {
        return svc.HealthCheck.Endpoint, HealthCheckTypeHTTP, nil
    }
    
    // 2. Try common health endpoints (if service has HTTP port)
    if svc.Port > 0 {
        endpoints := []string{"/health", "/healthz", "/ready", "/alive", "/ping"}
        for _, path := range endpoints {
            url := fmt.Sprintf("http://localhost:%d%s", svc.Port, path)
            if c.tryHTTPEndpoint(url) {
                return path, HealthCheckTypeHTTP, nil
            }
        }
    }
    
    // 3. Fall back to port check if port is configured
    if svc.Port > 0 {
        return "", HealthCheckTypePort, nil
    }
    
    // 4. Fall back to process check
    if svc.PID > 0 {
        return "", HealthCheckTypeProcess, nil
    }
    
    // 5. No viable health check method
    return "", "", fmt.Errorf("no health check method available for service %s", svc.Name)
}
```

### HTTP Health Check Implementation

```go
func (c *HealthChecker) performHTTPCheck(ctx context.Context, url string) HealthCheckResult {
    startTime := time.Now()
    result := HealthCheckResult{
        CheckType: HealthCheckTypeHTTP,
        Endpoint:  url,
        Timestamp: startTime,
    }
    
    // Create request with context for cancellation
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        result.Status = HealthStatusUnhealthy
        result.Error = err.Error()
        return result
    }
    
    // Perform request
    resp, err := c.httpClient.Do(req)
    result.ResponseTime = time.Since(startTime)
    
    if err != nil {
        result.Status = HealthStatusUnhealthy
        result.Error = err.Error()
        return result
    }
    defer resp.Body.Close()
    
    result.StatusCode = resp.StatusCode
    
    // Read and parse response body
    body, _ := io.ReadAll(resp.Body)
    var details map[string]interface{}
    if err := json.Unmarshal(body, &details); err == nil {
        result.Details = details
        
        // Check for explicit health status in response
        if status, ok := details["status"].(string); ok {
            switch status {
            case "healthy", "ok", "up":
                result.Status = HealthStatusHealthy
            case "degraded", "warning":
                result.Status = HealthStatusDegraded
            case "unhealthy", "down", "error":
                result.Status = HealthStatusUnhealthy
            }
        }
    }
    
    // If no explicit status, use HTTP status code
    if result.Status == "" {
        switch {
        case resp.StatusCode >= 200 && resp.StatusCode < 300:
            result.Status = HealthStatusHealthy
        case resp.StatusCode >= 300 && resp.StatusCode < 400:
            result.Status = HealthStatusHealthy // Redirects acceptable
        default:
            result.Status = HealthStatusUnhealthy
        }
    }
    
    return result
}
```

### Parallel Execution

Health checks execute in parallel for performance:

```go
func (m *HealthMonitor) Check(ctx context.Context) (*HealthReport, error) {
    results := make([]HealthCheckResult, len(m.services))
    var wg sync.WaitGroup
    
    // Limit concurrency to avoid overwhelming system
    maxConcurrent := 10
    semaphore := make(chan struct{}, maxConcurrent)
    
    for i, svc := range m.services {
        wg.Add(1)
        go func(index int, service ServiceInfo) {
            defer wg.Done()
            
            // Acquire semaphore
            semaphore <- struct{}{}
            defer func() { <-semaphore }()
            
            // Perform health check with timeout
            checkCtx, cancel := context.WithTimeout(ctx, m.timeout)
            defer cancel()
            
            results[index] = m.checker.CheckService(checkCtx, service)
            results[index].ServiceName = service.Name
        }(i, svc)
    }
    
    wg.Wait()
    
    // Generate summary
    summary := m.calculateSummary(results)
    
    return &HealthReport{
        Timestamp: time.Now(),
        Project:   m.getProjectDir(),
        Services:  results,
        Summary:   summary,
    }, nil
}
```

## Streaming Mode Implementation

### Stream Manager

```go
type StreamManager struct {
    monitor   *HealthMonitor
    interval  time.Duration
    output    io.Writer
    isTTY     bool
    stopChan  chan struct{}
    doneChan  chan struct{}
}

func (s *StreamManager) Start(ctx context.Context) error {
    s.stopChan = make(chan struct{})
    s.doneChan = make(chan struct{})
    
    // Set up signal handlers
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    
    // Display initial header (if TTY)
    if s.isTTY {
        s.displayHeader()
    }
    
    ticker := time.NewTicker(s.interval)
    defer ticker.Stop()
    
    checkCount := 0
    history := NewHealthHistory(100) // Keep last 100 checks
    
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
            
        case <-sigChan:
            s.displayFinalSummary(checkCount, history)
            return nil
            
        case <-s.stopChan:
            s.displayFinalSummary(checkCount, history)
            return nil
            
        case <-ticker.C:
            report, err := s.monitor.Check(ctx)
            if err != nil {
                // Log error but continue monitoring
                s.logError(err)
                continue
            }
            
            checkCount++
            history.Add(report)
            
            // Update display
            if s.isTTY {
                s.updateDisplay(report, history)
            } else {
                // Write JSON line for non-TTY
                s.writeJSONLine(report)
            }
        }
    }
}
```

### Terminal Display

```go
func (s *StreamManager) updateDisplay(report *HealthReport, history *HealthHistory) {
    // Clear screen and move cursor to top
    fmt.Fprintf(s.output, "\033[2J\033[H")
    
    // Display header
    s.displayHeader()
    
    // Display service status with progress bars
    fmt.Fprintln(s.output, "┌─────────────────────────────────────────────────────────────┐")
    fmt.Fprintf(s.output, "│ Last Update: %s\n", report.Timestamp.Format("15:04:05"))
    fmt.Fprintf(s.output, "│ Checks Performed: %d\n", history.Count())
    fmt.Fprintln(s.output, "├─────────────────────────────────────────────────────────────┤")
    
    for _, result := range report.Services {
        uptime := history.CalculateUptime(result.ServiceName)
        s.displayServiceRow(result, uptime)
    }
    
    fmt.Fprintln(s.output, "└─────────────────────────────────────────────────────────────┘")
    
    // Display recent changes
    changes := history.GetRecentChanges(5)
    if len(changes) > 0 {
        fmt.Fprintln(s.output, "\nRecent Changes:")
        for _, change := range changes {
            fmt.Fprintf(s.output, "  %s - %s: %s → %s\n", 
                change.Timestamp.Format("15:04:05"),
                change.ServiceName,
                change.OldStatus,
                change.NewStatus)
        }
    }
}

func (s *StreamManager) displayServiceRow(result HealthCheckResult, uptime float64) {
    icon := s.getStatusIcon(result.Status)
    bar := s.getProgressBar(uptime)
    
    fmt.Fprintf(s.output, "│ %s %-12s %-10s %5dms  %s %.0f%%\n",
        icon,
        result.ServiceName,
        result.Status,
        result.ResponseTime.Milliseconds(),
        bar,
        uptime*100)
}
```

## Configuration Schema

### azure.yaml Health Check Configuration

```yaml
services:
  api:
    language: python
    project: ./api
    ports:
      - "8080"
    
    # Health check configuration
    healthCheck:
      # Type of health check
      type: http  # http, port, process
      
      # HTTP-specific settings
      endpoint: /api/health  # Path to health endpoint
      method: GET            # HTTP method (GET, POST, HEAD)
      expectedStatus: 200    # Expected HTTP status code
      
      # Headers to include in HTTP requests
      headers:
        Authorization: "Bearer ${HEALTH_CHECK_TOKEN}"
        X-Health-Check: "true"
      
      # Timing configuration
      timeout: 5s           # Timeout for each check
      interval: 10s         # Interval between checks (streaming mode)
      startPeriod: 30s      # Grace period after service start
      
      # Retry configuration
      retries: 3            # Number of retries before marking unhealthy
      retryInterval: 1s     # Time between retries
      
      # Alerting thresholds
      alerts:
        responseTime: 1000ms      # Alert if response time exceeds this
        failureThreshold: 3       # Alert after N consecutive failures
        degradedThreshold: 500ms  # Mark as degraded if response time exceeds this
```

## Output Formatting

### Text Formatter

```go
type TextFormatter struct {
    useColor bool
    verbose  bool
}

func (f *TextFormatter) Format(report *HealthReport) string {
    var buf bytes.Buffer
    
    // Header
    fmt.Fprintf(&buf, "Health Check (%s)\n", report.Timestamp.Format("2006-01-02 15:04:05"))
    fmt.Fprintln(&buf, "=====================================\n")
    
    // Services
    for _, result := range report.Services {
        icon := f.getIcon(result.Status)
        fmt.Fprintf(&buf, "%s %-25s %-12s (%s)\n",
            icon, result.ServiceName, result.Status, result.CheckType)
        
        // Details
        if result.Endpoint != "" {
            fmt.Fprintf(&buf, "  Endpoint: %s\n", result.Endpoint)
        }
        if result.ResponseTime > 0 {
            fmt.Fprintf(&buf, "  Response Time: %dms\n", result.ResponseTime.Milliseconds())
        }
        if result.StatusCode > 0 {
            fmt.Fprintf(&buf, "  Status Code: %d\n", result.StatusCode)
        }
        if result.Error != "" {
            fmt.Fprintf(&buf, "  Error: %s\n", f.colorize(result.Error, "red"))
        }
        
        // Verbose details
        if f.verbose && result.Details != nil {
            f.formatDetails(&buf, result.Details, "  ")
        }
        
        fmt.Fprintln(&buf)
    }
    
    // Summary
    fmt.Fprintln(&buf, "─────────────────────────────────────────────\n")
    fmt.Fprintf(&buf, "Summary: %d healthy, %d degraded, %d unhealthy\n",
        report.Summary.Healthy, report.Summary.Degraded, report.Summary.Unhealthy)
    fmt.Fprintf(&buf, "Overall Status: %s\n", 
        f.colorize(string(report.Summary.Overall), f.getStatusColor(report.Summary.Overall)))
    
    return buf.String()
}
```

### JSON Formatter

```go
type JSONFormatter struct {
    pretty bool
}

func (f *JSONFormatter) Format(report *HealthReport) (string, error) {
    var data []byte
    var err error
    
    if f.pretty {
        data, err = json.MarshalIndent(report, "", "  ")
    } else {
        data, err = json.Marshal(report)
    }
    
    if err != nil {
        return "", err
    }
    
    return string(data), nil
}
```

## Integration Points

### Service Registry Integration

```go
// Update registry with health status
func (m *HealthMonitor) updateRegistry(results []HealthCheckResult) error {
    reg := registry.GetRegistry(m.projectDir)
    
    for _, result := range results {
        entry := reg.Get(result.ServiceName)
        if entry != nil {
            entry.Health = string(result.Status)
            entry.LastChecked = result.Timestamp
            if result.Error != "" {
                entry.Error = result.Error
            }
            reg.Update(entry)
        }
    }
    
    return reg.Save()
}
```

### Dashboard API Integration

```go
// HTTP handlers for dashboard integration
func (h *HealthHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
    report, err := h.monitor.Check(r.Context())
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(report)
}

func (h *HealthHandler) StreamHealth(w http.ResponseWriter, r *http.Request) {
    // Set up Server-Sent Events
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-r.Context().Done():
            return
        case <-ticker.C:
            report, err := h.monitor.Check(r.Context())
            if err != nil {
                continue
            }
            
            data, _ := json.Marshal(report)
            fmt.Fprintf(w, "data: %s\n\n", data)
            w.(http.Flusher).Flush()
        }
    }
}
```

## Performance Considerations

### Concurrent Health Checks

- Maximum 10 concurrent health checks to avoid overwhelming system
- Semaphore-based rate limiting
- Context-based cancellation for clean shutdown

### Memory Management

- Circular buffer for health history (last 100 checks)
- Limit response body size when reading HTTP responses
- Clean up old entries automatically

### CPU Throttling

```go
// Monitor CPU usage and throttle if needed
func (m *HealthMonitor) shouldThrottle() bool {
    usage := getCurrentCPUUsage()
    return usage > 0.8 // 80% CPU threshold
}

func (m *HealthMonitor) Check(ctx context.Context) (*HealthReport, error) {
    if m.shouldThrottle() {
        // Add delay to reduce CPU load
        time.Sleep(time.Second)
    }
    
    // Proceed with health checks...
}
```

## Error Handling

### Graceful Degradation

When health checks fail, provide actionable information:

```go
type HealthCheckError struct {
    ServiceName string
    CheckType   HealthCheckType
    Err         error
    Suggestion  string
}

func (e *HealthCheckError) Error() string {
    return fmt.Sprintf("%s health check failed for %s: %v", 
        e.CheckType, e.ServiceName, e.Err)
}

func (e *HealthCheckError) GetSuggestion() string {
    if e.CheckType == HealthCheckTypeHTTP {
        return "Check if service is running and HTTP server is started"
    } else if e.CheckType == HealthCheckTypePort {
        return "Verify service is listening on the configured port"
    } else {
        return "Check service logs with 'azd app logs --service " + e.ServiceName + "'"
    }
}
```

## Testing Strategy

### Unit Tests

- Health checker logic (HTTP, port, process checks)
- Output formatters (text, JSON, table)
- Health report generation and aggregation
- Error handling and graceful degradation

### Integration Tests

- End-to-end health check workflows
- Streaming mode with real services
- Service registry integration
- Dashboard API integration

### Test Utilities

```go
// Mock HTTP server for testing
func newMockHealthServer(t *testing.T, statusCode int, body string) *httptest.Server {
    return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(statusCode)
        w.Write([]byte(body))
    }))
}

// Mock service info for testing
func newMockServiceInfo(name string, port int) ServiceInfo {
    return ServiceInfo{
        Name: name,
        Port: port,
        Local: &LocalServiceInfo{
            Status: "running",
        },
    }
}
```

## Security Considerations

### Authentication Headers

- Support environment variable substitution in headers
- Mask sensitive headers in verbose output
- Validate header values before sending

### Network Security

- Only allow localhost connections by default
- Support custom IP binding if needed
- Validate URLs before making requests

### Resource Limits

- Limit response body size (max 1MB)
- Timeout all network operations
- Rate limit health checks to prevent DoS

## Future Enhancements

### Phase 2: Advanced Features

1. **Health History Storage**
   - Persistent storage of health check results
   - Trend analysis and visualization
   - Query API for historical data

2. **Alerting System**
   - Email notifications on health changes
   - Webhook integration
   - Slack/Teams integration

3. **Predictive Health**
   - ML-based failure prediction
   - Anomaly detection
   - Proactive alerts

4. **Distributed Tracing**
   - Correlate health with traces
   - Dependency mapping
   - Root cause analysis

5. **Custom Health Checks**
   - Plugin system for custom logic
   - Language-specific health checkers
   - Business logic validation

## References

- Existing health check implementation: `cli/src/internal/service/health.go`
- Service registry: `cli/src/internal/registry/`
- Service info: `cli/src/internal/serviceinfo/`
- Output formatting: `cli/src/internal/output/`
- Command documentation: `cli/docs/commands/health.md`
