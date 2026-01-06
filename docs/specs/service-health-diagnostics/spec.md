# Service Health Diagnostics Enhancement

**Status**: Draft  
**Created**: 2025-12-29  
**Priority**: P1 (High Impact - User Experience)

## Problem Statement

Users currently see services marked as "unhealthy" in the dashboard but have no insight into **why** the service is unhealthy. The UI shows:
- Status indicator (red dot, "Unhealthy" badge)
- Generic error banner with truncated error message
- No detailed diagnostic information
- No easy way to copy diagnostic info for troubleshooting

This creates frustration and requires users to:
1. Open service detail panel manually
2. Check logs to find clues
3. Manually type or screenshot error info
4. Search documentation without context

## Goals

### Primary
1. **Visibility**: Provide detailed diagnostic information on hover
2. **Clarity**: Explain what check was performed and why it failed
3. **Actionability**: Suggest next steps and relevant commands
4. **Accessibility**: Enable copy-to-clipboard for sharing/debugging

### Secondary
1. **Consistency**: Apply to all health statuses (degraded, unhealthy, unknown)
2. **Performance**: Keep tooltips lightweight, no extra API calls
3. **Progressive Disclosure**: Show summary on card, details on hover/click

## User Experience

### Before
```
[Service Card]
🔴 api                    [Unhealthy]
Error Detected
Service Unavailable (truncated...)
```

### After
```
[Service Card - Hover on Status Icon]
┌─────────────────────────────────────────────┐
│ Service Health: Unhealthy                   │
│                                             │
│ Check Type: HTTP GET                        │
│ Endpoint: http://localhost:8080/health     │
│ Status: 503 Service Unavailable            │
│ Response Time: 45ms                         │
│                                             │
│ Error Details:                              │
│ Database connection pool exhausted          │
│                                             │
│ Suggested Actions:                          │
│ • Check service logs: azd app logs -s api  │
│ • Verify database is running                │
│ • Review connection pool settings           │
│                                             │
│ [Copy Diagnostics] [View Logs]             │
└─────────────────────────────────────────────┘
```

## Design Specifications

### 1. Health Status Tooltip

**Trigger**: Hover over DualStatusBadge health icon  
**Delay**: 400ms (follows system tooltip guidelines)  
**Position**: Auto (prefer top/right, avoid viewport overflow)

#### Tooltip Structure
```typescript
interface HealthDiagnosticTooltip {
  // Header
  status: HealthStatus
  statusLabel: string  // "Healthy", "Unhealthy", "Degraded", "Unknown"
  
  // Check Details
  checkType: HealthCheckType  // "http", "tcp", "process"
  endpoint?: string  // "http://localhost:8080/health"
  port?: number
  pid?: number
  
  // Results
  statusCode?: number  // HTTP: 200, 503, etc.
  responseTime: number  // milliseconds
  uptime?: number  // seconds since start
  
  // Error Information
  error?: string  // Primary error message
  errorDetails?: string  // Extended details if available
  consecutiveFailures?: number
  
  // Actionable Guidance
  suggestedActions: Action[]
  relevantCommands: Command[]
}

interface Action {
  text: string  // "Check service logs"
  icon?: string
  command?: string  // "azd app logs --service api"
}
```

#### Content by Health Status

##### Healthy
```
✓ Service Health: Healthy

Check: HTTP GET
Endpoint: http://localhost:8080/health
Status: 200 OK
Response Time: 12ms
Uptime: 5m 32s

Last checked: 2s ago
```

##### Degraded
```
⚠ Service Health: Degraded

Check: HTTP GET
Endpoint: http://localhost:8080/health
Status: 200 OK (slow response)
Response Time: 1,234ms
Uptime: 15m 47s

Warning: Response time exceeds threshold (>1000ms)

Suggested Actions:
• Check CPU/memory usage
• Review application performance
• Check for database query slowness

[View Metrics] [Copy Details]
```

##### Unhealthy
```
✗ Service Health: Unhealthy

Check: HTTP GET
Endpoint: http://localhost:8080/health
Status: 503 Service Unavailable
Response Time: 45ms
Consecutive Failures: 3

Error Details:
Database connection failed: timeout after 5s

Suggested Actions:
• Check service logs: azd app logs --service api
• Verify database is running
• Check network connectivity
• Review connection pool settings

[Copy Diagnostics] [View Logs]
```

##### Unknown
```
? Service Health: Unknown

No health check configured for this service.

Service Details:
Type: Process Service
Mode: daemon
PID: 12345 (running)
Uptime: 2m 15s

Note: Service is running but health status cannot be determined.
Consider adding a health check endpoint.

[Learn More] [Add Health Check]
```

### 2. Copy to Clipboard

**Format**: Structured markdown for easy pasting into issues/chat

```markdown
# Service Health Diagnostic Report
**Service**: api
**Status**: unhealthy
**Timestamp**: 2025-12-29T10:30:45Z

## Health Check
- **Type**: HTTP GET
- **Endpoint**: http://localhost:8080/health
- **Status Code**: 503 Service Unavailable
- **Response Time**: 45ms
- **Consecutive Failures**: 3

## Error
Database connection failed: timeout after 5s

## Service Info
- **Uptime**: 15m 47s
- **PID**: 12345
- **Port**: 8080

## Suggested Actions
1. Check service logs: `azd app logs --service api`
2. Verify database is running
3. Check network connectivity
4. Review connection pool settings

---
Generated by azd app health
```

### 3. Visual Design

#### Tooltip Component
```tsx
<HealthTooltip
  healthStatus={healthCheckResult}
  service={service}
  position="auto"
  maxWidth="400px"
  className="shadow-xl border-2"
/>
```

#### Color Coding
- **Healthy**: Emerald green theme
- **Degraded**: Amber/yellow theme
- **Unhealthy**: Rose/red theme
- **Unknown**: Slate/gray theme

#### Interactive Elements
- Copy button: Copies full diagnostic report
- View Logs button: Opens log panel filtered to service
- Learn More: Links to docs about health checks (unknown only)

### 4. Tooltip Implementation

#### Technology Stack
- **Library**: Radix UI Tooltip (already in use)
- **Positioning**: Floating UI (via Radix)
- **Animations**: Framer Motion (optional, or CSS)

#### Accessibility
- ARIA labels on all interactive elements
- Keyboard navigation support (Tab, Enter, Escape)
- Screen reader announcements for status changes
- Respect `prefers-reduced-motion`

## Technical Implementation

### Data Flow

```
Backend (Go)
├─ HealthChecker.CheckService()
│  ├─ Returns HealthCheckResult with:
│  │  ├─ Status, CheckType, Endpoint
│  │  ├─ StatusCode, ResponseTime
│  │  ├─ Error message
│  │  └─ Details map
│  └─ Streams via SSE
│
Frontend (React)
├─ Receives HealthCheckResult
├─ Maps to HealthDiagnostic
├─ Renders ServiceCard with DualStatusBadge
└─ Shows HealthTooltip on hover
```

### Backend Changes (Go)

#### 1. Enhanced Error Details

```go
// types.go
type HealthCheckResult struct {
    // ... existing fields ...
    
    // Enhanced diagnostic fields
    Error                string                 `json:"error,omitempty"`
    ErrorDetails         string                 `json:"errorDetails,omitempty"`  // NEW
    ConsecutiveFailures  int                    `json:"consecutiveFailures,omitempty"`  // NEW
    LastSuccessTime      *time.Time             `json:"lastSuccessTime,omitempty"`  // NEW
    Details              map[string]interface{} `json:"details,omitempty"`
}
```

#### 2. Structured Error Messages

```go
// checker.go - HTTP check failure
func (c *HealthChecker) performHTTPCheck(...) *httpHealthCheckResult {
    // ... existing code ...
    
    if statusCode >= 400 {
        result.Status = HealthStatusUnhealthy
        result.Error = fmt.Sprintf("HTTP %d: %s", statusCode, http.StatusText(statusCode))
        
        // Add detailed error if available from response body
        if body != nil && len(body) > 0 {
            result.ErrorDetails = parseHealthResponseError(body)
        }
        
        result.Details["suggestion"] = suggestHTTPErrorAction(statusCode)
    }
}

func suggestHTTPErrorAction(statusCode int) string {
    switch {
    case statusCode == 503:
        return "Service temporarily unavailable. Check if dependencies are running."
    case statusCode >= 500:
        return "Server error. Check application logs for details."
    case statusCode == 404:
        return "Health endpoint not found. Verify endpoint configuration."
    case statusCode == 401 || statusCode == 403:
        return "Authentication/authorization failed. Check credentials."
    default:
        return "HTTP request failed. Check service logs for details."
    }
}
```

#### 3. Consecutive Failure Tracking

```go
// monitor.go
type HealthMonitor struct {
    // ... existing fields ...
    failureCount map[string]int  // Track consecutive failures per service
    mu           sync.RWMutex
}

func (m *HealthMonitor) trackFailure(serviceName string, result *HealthCheckResult) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if result.Status == HealthStatusUnhealthy {
        m.failureCount[serviceName]++
        result.ConsecutiveFailures = m.failureCount[serviceName]
    } else {
        m.failureCount[serviceName] = 0
    }
}
```

### Frontend Changes (React/TypeScript)

#### 1. Update Types

```typescript
// types.ts
export interface HealthCheckResult {
  // ... existing fields ...
  
  error?: string
  errorDetails?: string  // NEW - Extended error information
  consecutiveFailures?: number  // NEW - Failure count
  lastSuccessTime?: string  // NEW - ISO timestamp of last success
  details?: Record<string, unknown>
}

export interface HealthDiagnostic {
  service: Service
  healthStatus: HealthCheckResult
  suggestedActions: HealthAction[]
  formattedReport: string  // Pre-formatted markdown for copy
}

export interface HealthAction {
  label: string
  icon?: string
  command?: string
  docsUrl?: string
}
```

#### 2. Create HealthTooltip Component

```typescript
// components/HealthTooltip.tsx
interface HealthTooltipProps {
  healthStatus: HealthCheckResult
  service: Service
  children: React.ReactNode  // Trigger element (status badge)
}

export function HealthTooltip({ healthStatus, service, children }: HealthTooltipProps) {
  const [open, setOpen] = useState(false)
  const diagnostic = useMemo(() => 
    buildHealthDiagnostic(healthStatus, service), 
    [healthStatus, service]
  )

  const handleCopy = async () => {
    await navigator.clipboard.writeText(diagnostic.formattedReport)
    toast.success('Diagnostics copied to clipboard')
  }

  return (
    <TooltipProvider>
      <Tooltip open={open} onOpenChange={setOpen} delayDuration={400}>
        <TooltipTrigger asChild>
          {children}
        </TooltipTrigger>
        <TooltipContent 
          side="top" 
          className="max-w-md p-4 space-y-3"
          sideOffset={8}
        >
          <HealthTooltipContent 
            diagnostic={diagnostic} 
            onCopy={handleCopy}
          />
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
```

#### 3. Build Diagnostic Helper

```typescript
// lib/health-diagnostics.ts
export function buildHealthDiagnostic(
  healthStatus: HealthCheckResult,
  service: Service
): HealthDiagnostic {
  const actions = getSuggestedActions(healthStatus, service)
  const report = formatDiagnosticReport(healthStatus, service, actions)
  
  return {
    service,
    healthStatus,
    suggestedActions: actions,
    formattedReport: report,
  }
}

function getSuggestedActions(
  healthStatus: HealthCheckResult,
  service: Service
): HealthAction[] {
  const actions: HealthAction[] = []
  
  // Always suggest viewing logs for unhealthy services
  if (healthStatus.status === 'unhealthy') {
    actions.push({
      label: 'Check service logs',
      icon: 'terminal',
      command: `azd app logs --service ${service.name}`,
    })
  }
  
  // Add specific suggestions based on check type and error
  if (healthStatus.checkType === 'http' && healthStatus.statusCode) {
    actions.push(...getHTTPSpecificActions(healthStatus.statusCode, service))
  }
  
  // Add suggestion from backend if available
  if (healthStatus.details?.suggestion) {
    actions.push({
      label: healthStatus.details.suggestion as string,
    })
  }
  
  return actions
}

function getHTTPSpecificActions(statusCode: number, service: Service): HealthAction[] {
  const actions: HealthAction[] = []
  
  if (statusCode >= 500) {
    actions.push(
      { label: 'Check application logs for errors' },
      { label: 'Verify service dependencies are running' },
      { label: 'Review error stack traces' }
    )
  } else if (statusCode === 404) {
    actions.push(
      { label: 'Verify health endpoint configuration' },
      { 
        label: 'View health check documentation',
        docsUrl: 'https://github.com/jongio/azd-app/docs/features/health-checks.md'
      }
    )
  }
  
  return actions
}

function formatDiagnosticReport(
  healthStatus: HealthCheckResult,
  service: Service,
  actions: HealthAction[]
): string {
  const timestamp = new Date().toISOString()
  
  return `# Service Health Diagnostic Report
**Service**: ${service.name}
**Status**: ${healthStatus.status}
**Timestamp**: ${timestamp}

## Health Check
- **Type**: ${healthStatus.checkType.toUpperCase()}${
    healthStatus.endpoint ? `\n- **Endpoint**: ${healthStatus.endpoint}` : ''
  }${
    healthStatus.statusCode ? `\n- **Status Code**: ${healthStatus.statusCode}` : ''
  }
- **Response Time**: ${formatResponseTime(healthStatus.responseTime)}${
    healthStatus.consecutiveFailures 
      ? `\n- **Consecutive Failures**: ${healthStatus.consecutiveFailures}` 
      : ''
  }

${healthStatus.error ? `## Error\n${healthStatus.error}\n` : ''}${
    healthStatus.errorDetails ? `\n${healthStatus.errorDetails}\n` : ''
  }

## Service Info
- **Uptime**: ${formatUptime(healthStatus.uptime)}${
    healthStatus.port ? `\n- **Port**: ${healthStatus.port}` : ''
  }${
    healthStatus.pid ? `\n- **PID**: ${healthStatus.pid}` : ''
  }

${actions.length > 0 ? `## Suggested Actions
${actions.map((a, i) => `${i + 1}. ${a.label}${a.command ? ` → \`${a.command}\`` : ''}`).join('\n')}
` : ''}
---
Generated by azd app health
`
}
```

#### 4. Update DualStatusBadge

```typescript
// components/StatusIndicator.tsx
export function DualStatusBadge({ status, health, service, healthStatus }: DualStatusBadgeProps) {
  // ... existing code ...
  
  return (
    <div className={cn('flex items-center gap-1.5', className)}>
      {/* Running State Icon */}
      <span className={...} title={`Process state: ${status}`}>
        {/* ... existing icon ... */}
      </span>
      
      {/* Health Status Icon - Wrapped in tooltip */}
      <HealthTooltip healthStatus={healthStatus} service={service}>
        <span className={...} title={`Service health: ${effectiveHealth}`}>
          {/* ... existing health icon ... */}
        </span>
      </HealthTooltip>
    </div>
  )
}
```

## Information Architecture

### What Error Details Are Available?

From `HealthCheckResult` we have:

| Field | Source | Example |
|-------|--------|---------|
| `error` | Go health checker | "HTTP 503: Service Unavailable" |
| `errorDetails` | Response body parsing (NEW) | "Database connection pool exhausted" |
| `checkType` | Check method | "http", "tcp", "process" |
| `endpoint` | Target URL/command | "http://localhost:8080/health" |
| `statusCode` | HTTP response | 503 |
| `responseTime` | Check duration | 45000000 (45ms in ns) |
| `consecutiveFailures` | Failure tracking (NEW) | 3 |
| `details` | Additional metadata | `{ "suggestion": "..." }` |
| `port` | Service port | 8080 |
| `pid` | Process ID | 12345 |
| `uptime` | Time since start | 947000000000 (15m 47s in ns) |

### Suggested Actions by Scenario

#### HTTP Failures

| Status Code | Primary Action | Secondary Actions |
|-------------|----------------|-------------------|
| 503 | Check dependencies | Verify database, cache, message queue |
| 500-599 | Check logs | Review stack traces, recent deployments |
| 404 | Verify endpoint | Check health check configuration |
| 401/403 | Check auth | Verify credentials, API keys |
| 429 | Rate limited | Reduce request rate, check quotas |
| Timeout | Network check | Firewall, DNS, connectivity |

#### TCP Failures

| Error | Actions |
|-------|---------|
| Connection refused | Check if service is running, verify port |
| Timeout | Network connectivity, firewall rules |
| Port not listening | Verify service started correctly |

#### Process Failures

| Error | Actions |
|-------|---------|
| Process not running | Check service logs, verify start command |
| Process crashed | Review crash logs, check exit code |
| Pattern not matched | Check output configuration, startup time |

## Files Modified/Created

### New Files
- `cli/dashboard/src/components/HealthTooltip.tsx` - Main tooltip component
- `cli/dashboard/src/components/HealthTooltipContent.tsx` - Tooltip content layout
- `cli/dashboard/src/lib/health-diagnostics.ts` - Diagnostic helpers
- `cli/dashboard/src/hooks/useHealthDiagnostic.ts` - Hook for building diagnostics

### Modified Files
- `cli/src/internal/healthcheck/types.go` - Add error detail fields
- `cli/src/internal/healthcheck/checker.go` - Enhanced error messages
- `cli/src/internal/healthcheck/monitor.go` - Failure tracking
- `cli/dashboard/src/types.ts` - Update HealthCheckResult type
- `cli/dashboard/src/components/StatusIndicator.tsx` - Wrap health icon in tooltip
- `cli/dashboard/src/components/ServiceCard.tsx` - Pass service to DualStatusBadge

## Testing Strategy

### Unit Tests
- `health-diagnostics.test.ts` - Test diagnostic building logic
- `HealthTooltip.test.tsx` - Component rendering and interactions
- Test copy-to-clipboard functionality
- Test action suggestions for different error scenarios

### Integration Tests
- Test tooltip positioning and overflow handling
- Test keyboard navigation (Tab, Enter, Escape)
- Test with long error messages (truncation)
- Test with missing/incomplete health data

### E2E Tests
- Hover on unhealthy service status icon → tooltip appears
- Click copy button → clipboard contains markdown report
- Click view logs → log panel opens with service filter
- Test all health statuses (healthy, degraded, unhealthy, unknown)

### Accessibility Tests
- Screen reader announcements
- Keyboard-only navigation
- ARIA attributes validation
- Color contrast ratios

## Metrics

### Success Criteria
1. **Discoverability**: 70%+ users hover on status icons within first session
2. **Copy Usage**: 40%+ users copy diagnostics when encountering unhealthy services
3. **Time to Resolution**: Reduce troubleshooting time by 30% (user research)
4. **Support Reduction**: Fewer "why is my service unhealthy?" support requests

### Tracking
- Event: `health_tooltip_viewed` (status, checkType)
- Event: `health_diagnostic_copied` (status, hasError)
- Event: `health_action_clicked` (action, status)
- Metric: Tooltip open duration (time spent reading)

## Documentation

### User-Facing
- Add section to `cli/docs/commands/health.md` about diagnostic tooltips
- Update dashboard screenshots to show tooltips
- Create troubleshooting guide with common errors

### Developer-Facing
- Document `HealthDiagnostic` API
- Document how to add custom health check suggestions
- Document error message best practices

## Future Enhancements

### Phase 2 (Post-MVP)
1. **Historical Health Data**: Show trend chart in tooltip (last 24h)
2. **AI-Powered Suggestions**: Use LLM to suggest fixes based on error patterns
3. **Auto-Recovery**: One-click restart/fix for common issues
4. **Health Check Configuration**: Edit health check settings from tooltip
5. **Dependency Graph**: Show which services this service depends on
6. **Alert Integration**: Link to configured alerts/notifications

### Nice-to-Have
- Export diagnostic report as file (JSON/markdown)
- Share diagnostic link (temporary URL)
- Integration with issue trackers (create GitHub issue)
- Smart retry suggestions based on failure patterns

## Open Questions

1. Should we show tooltip on service card error banner as well?
2. How long should we store consecutive failure counts? (session vs. persistent)
3. Should "Copy Diagnostics" include full service configuration?
4. Do we need rate limiting on copy-to-clipboard to prevent abuse?
5. Should we auto-expand logs panel if user clicks "View Logs"?

## References

- [Service States Documentation](../../features/service-states.md)
- [Health Check Command Reference](../../commands/health.md)
- [Status Indicator Design](../../../cli/dashboard/design/components/status-indicators.md)
- [Radix UI Tooltip Docs](https://www.radix-ui.com/docs/primitives/components/tooltip)
