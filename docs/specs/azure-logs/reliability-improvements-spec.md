# Azure Logs Reliability & Performance Improvements

## Overview

Address critical design flaws discovered in the Azure logs implementation related to connection management, error handling, polling efficiency, and data reliability.

## Problems Identified

### P0 - Critical (Data Loss & UX)

**1. No Backpressure Handling**
- Silent data loss when frontend can't keep up with log stream
- No sequence numbers or acknowledgment mechanism
- Frontend has no way to detect missing logs
- **Impact:** Users miss critical error logs without knowing it

**2. Inconsistent Error Handling**
- Azure polling continues silently on repeated failures
- No exponential backoff for transient errors
- Dashboard shows stale "connected" status
- No automatic reconnection strategy
- **Impact:** Users don't know when Azure logs stopped working

**3. Connection Pooling Missing**
- New Log Analytics client created per request
- No HTTP connection reuse
- Excessive TCP handshakes and TLS overhead
- **Impact:** Higher latency, potential rate limiting, resource waste

### P1 - Correctness & Performance

**4. Workspace GUID Race Condition**
- Multiple concurrent goroutines can all make ARM API calls before cache is populated
- Mutex only protects cache check, not the resolution operation
- **Impact:** Redundant ARM API calls, slower startup

**5. Polling Inefficiency**
- Queries `ago(1m)` repeatedly, fetching overlapping data
- No reliable `lastSeenTimestamp` tracking
- Wastes Log Analytics query units (costs money)
- **Impact:** Higher costs, duplicate data, increased latency

**6. Log Level Inference Fragile**
- "0 errors found" classified as ERROR
- No word boundary checking
- `infoOverridePatterns` only applied to local logs
- **Impact:** False positive errors in UI

**7. Streaming Window Alignment**
- Initial fetch uses 24h default regardless of user-requested time range
- Frontend shows more logs than expected
- **Impact:** Confusing UX, slower initial load

**8. Resource Discovery Brittle**
- Service name mapping fails for multi-word names
- `SERVICE_MY_API_NAME` becomes "my-api" instead of "my_api"
- No validation or warnings for unmapped services
- **Impact:** Azure logs fail silently for some services

### P2 - Minor Issues

**9. File Rotation Race Condition**
- Size check and rotation not atomic with write
- Potential for corrupted log files
- **Impact:** Rare log file corruption

**10. Authentication Scope Workaround**
- Using `DefaultAzureCredential` instead of extension framework token
- Requires Azure CLI to be installed and logged in
- **Impact:** Fragile auth, extra dependency (documented as known issue)

## Solution Design

### 1. Backpressure Handling

**Add Sequence Numbers:**
```go
type LogEntry struct {
    Sequence  uint64    `json:"sequence,omitempty"`
    // ... existing fields
}
```

**Frontend Acknowledgment:**
- Frontend tracks last received sequence number
- Periodically sends ack via WebSocket
- Backend buffers unacked entries (limited size)

**Catch-up API:**
```
GET /api/logs/catchup?service={name}&fromSeq={seq}
```

**UI Indicator:**
- Show "catching up... (123 logs behind)" when sequence gap detected
- Auto-trigger catch-up query

### 2. Error Handling & Reconnection

**Exponential Backoff:**
```go
type PollingState struct {
    failureCount  int
    backoffDelay  time.Duration
    maxBackoff    time.Duration
}

func (p *PollingState) NextDelay() time.Duration {
    if p.failureCount == 0 {
        return baseInterval
    }
    delay := baseInterval * time.Duration(math.Pow(2, float64(p.failureCount)))
    if delay > p.maxBackoff {
        delay = p.maxBackoff
    }
    return delay
}
```

**Health Monitoring:**
```go
type ConnectionHealth struct {
    Status         string    // "connected", "degraded", "disconnected"
    LastSuccess    time.Time
    ConsecutiveFails int
    LastError      string
}
```

**WebSocket Status Messages:**
```json
{
    "type": "status",
    "status": "disconnected",
    "error": "Log Analytics query failed",
    "retrying": true,
    "nextRetry": "2025-12-21T10:30:15Z"
}
```

### 3. Connection Pooling

**Singleton Client Cache:**
```go
var (
    clientCache   = make(map[string]*LogAnalyticsClient)
    clientCacheMu sync.RWMutex
)

func GetOrCreateClient(workspaceID string) (*LogAnalyticsClient, error) {
    clientCacheMu.RLock()
    if client, exists := clientCache[workspaceID]; exists {
        clientCacheMu.RUnlock()
        return client, nil
    }
    clientCacheMu.RUnlock()
    
    clientCacheMu.Lock()
    defer clientCacheMu.Unlock()
    
    // Double-check after acquiring write lock
    if client, exists := clientCache[workspaceID]; exists {
        return client, nil
    }
    
    client, err := NewLogAnalyticsClient(cred, workspaceID)
    if err != nil {
        return nil, err
    }
    clientCache[workspaceID] = client
    return client, nil
}
```

### 4. Workspace GUID Resolution

**Use sync.Once:**
```go
type LogAnalyticsClient struct {
    client        *azlogs.Client
    workspaceID   string
    credential    azcore.TokenCredential
    workspaceGUID string
    resolveOnce   sync.Once
    resolveErr    error
}

func (c *LogAnalyticsClient) getWorkspaceGUID(ctx context.Context) (string, error) {
    c.resolveOnce.Do(func() {
        c.workspaceGUID, c.resolveErr = NormalizeWorkspaceID(ctx, c.credential, c.workspaceID)
    })
    return c.workspaceGUID, c.resolveErr
}
```

### 5. Polling Efficiency

**Track Last Seen per Service:**
```go
type StreamingState struct {
    LastSeenTimestamp time.Time
    LastSeenHash      string  // Deduplication
}

// In query
query := fmt.Sprintf(`
| where TimeGenerated > datetime('%s')
| order by TimeGenerated asc
`, state.LastSeenTimestamp.Format(time.RFC3339Nano))
```

**Update on Each Entry:**
```go
for _, entry := range entries {
    if entry.Timestamp.After(state.LastSeenTimestamp) {
        state.LastSeenTimestamp = entry.Timestamp
        // Send to channel
    }
}
```

### 6. Log Level Inference

**Apply Overrides Consistently:**
```go
func inferLogLevelFromAzure(message string) LogLevel {
    lowerMsg := strings.ToLower(message)
    
    // Check override patterns first (same as local)
    for _, pattern := range infoOverridePatterns {
        if strings.Contains(lowerMsg, pattern) {
            return LogLevelInfo
        }
    }
    
    // Use word boundaries for error detection
    if containsWord(lowerMsg, "error") || containsWord(lowerMsg, "exception") {
        return LogLevelError
    }
    
    // ... rest of logic
}

func containsWord(s, word string) bool {
    re := regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\b`)
    return re.MatchString(s)
}
```

### 7. Streaming Window Alignment

**Pass Time Range to Backend:**
```typescript
// Frontend
const params = new URLSearchParams({
    service: serviceName,
    since: sinceParam,  // "1h", "30m", etc.
})
```

**Backend Respects Parameter:**
```go
since := r.URL.Query().Get("since")
initialWindow := parseDuration(since)
if initialWindow == 0 {
    initialWindow = 1 * time.Hour  // Default
}
```

### 8. Resource Discovery

**Improve Name Normalization:**
```go
func normalizeServiceName(envVarName string) string {
    // SERVICE_MY_API_NAME -> my_api (preserve underscores)
    name := strings.TrimPrefix(envVarName, "SERVICE_")
    name = strings.TrimSuffix(name, "_NAME")
    name = strings.ToLower(name)
    // Don't replace underscores - they're intentional
    return name
}
```

**Validate Mappings:**
```go
// Warn when azure.yaml service has no environment variable
for _, svc := range azureYAMLServices {
    if _, found := serviceNameMap[svc.Name]; !found {
        slog.Warn("No Azure resource found for service", "service", svc.Name)
    }
}
```

### 9. File Rotation Atomicity

**Lock Entire Operation:**
```go
func (lb *LogBuffer) Add(entry LogEntry) {
    // ... buffer logic
    
    if lb.fileWriter != nil {
        lb.fileMu.Lock()
        // Check size and rotate atomically
        if lb.currentFileSize >= MaxLogFileSize {
            lb.rotateLogFileUnsafe()  // Caller holds lock
        }
        lb.writeToFileUnsafe(entry)
        lb.fileMu.Unlock()
    }
}
```

## Implementation Plan

### Phase 1: Critical Fixes (P0)
**Effort:** 1 day
- Task 1: Implement exponential backoff and health monitoring
- Task 2: Add connection pooling for Log Analytics clients
- Task 3: Implement backpressure with sequence numbers

### Phase 2: Performance & Correctness (P1)
**Effort:** 1 day
- Task 4: Fix workspace GUID race with sync.Once
- Task 5: Improve polling efficiency with lastSeen tracking
- Task 6: Fix log level inference for Azure logs
- Task 7: Align streaming window with user intent
- Task 8: Improve resource discovery robustness

### Phase 3: Polish (P2)
**Effort:** 0.5 day
- Task 9: Fix file rotation race condition
- Task 10: Document auth scope limitation

**Total Effort:** 2.5 days

## Testing Strategy

### Unit Tests
- Mock Azure SDK responses
- Test exponential backoff logic
- Test sequence number generation
- Test client cache concurrency
- Test name normalization edge cases

### Integration Tests
- Test WebSocket reconnection with mock failures
- Test catch-up query with sequence gaps
- Test polling with various failure scenarios
- Test connection pool reuse

### Manual Testing
- Simulate Azure API failures (mock server)
- Test slow frontend with backpressure
- Verify UI shows correct status indicators
- Test multi-word service names

## Success Criteria

- ✅ No silent data loss during slow frontend scenarios
- ✅ Clear connection status shown in UI
- ✅ Automatic reconnection with exponential backoff
- ✅ Reduced Log Analytics costs (no duplicate queries)
- ✅ No redundant ARM API calls on startup
- ✅ Accurate log level classification
- ✅ All services mapped correctly from environment

## Out of Scope

- Authentication scope issue (requires upstream azd framework change)
- Remains documented as known limitation using DefaultAzureCredential workaround

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Sequence number overflow | Use uint64 (584 billion years at 1 log/sec) |
| Breaking changes to WebSocket protocol | Version message format, backward compatible |
| Performance regression | Benchmark before/after, profile connection pooling |
| Complex state management | Clear state machine documentation, extensive tests |

## Rollback Plan

All changes are additive or internal refactoring:
- Sequence numbers optional (backward compatible)
- Connection pooling is transparent
- Error handling is progressive enhancement
- Can revert individual commits without breaking functionality
