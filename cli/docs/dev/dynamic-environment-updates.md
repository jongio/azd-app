# Dynamic Environment Updates

## Overview

The dashboard now automatically refreshes when Azure environment variables change (e.g., after `azd provision`) using an **event-driven architecture** that requires:
- ✅ No disk I/O
- ✅ No process spawning  
- ✅ No polling
- ✅ Real-time WebSocket updates

## Architecture

```
azd provision completes
    ↓
azd updates process environment variables
    ↓
azd fires "environment updated" event
    ↓
listen command handler receives event
    ↓
RefreshEnvironmentCache() updates in-memory cache
    ↓
BroadcastServiceUpdate() pushes to WebSocket clients
    ↓
Dashboard UI auto-refreshes with new Azure URLs/names
```

## Implementation Details

### 1. Event Subscription (`listen.go`)

The `listen` command subscribes to the `"environment updated"` service event from azd:

```go
host := azdext.NewExtensionHost(azdClient).
    WithServiceEventHandler("environment updated", handleEnvironmentUpdate, nil)
```

When azd completes provisioning, it:
1. Updates environment variables in the process (`os.Environ()`)
2. Fires the `"environment updated"` event
3. Calls our `handleEnvironmentUpdate` handler

### 2. Environment Cache (`serviceinfo/serviceinfo.go`)

The serviceinfo package maintains an in-memory cache of environment variables:

```go
var (
    environmentCache   map[string]string
    environmentCacheMu sync.RWMutex
)
```

When the event fires:
- `RefreshEnvironmentCache()` reads fresh values from `os.Environ()`
- Cache is updated with the latest Azure resource URLs, names, etc.

### 3. WebSocket Broadcast (`dashboard/server.go`)

The dashboard server broadcasts updates to all connected clients:

```go
func (s *Server) BroadcastServiceUpdate(projectDir string) error {
    services, _ := serviceinfo.GetServiceInfo(projectDir) // Uses fresh cache
    
    message := map[string]interface{}{
        "type":     "services",
        "services": services,
    }
    
    // Send to all WebSocket clients
    for client := range s.clients {
        client.WriteJSON(message)
    }
}
```

### 4. Frontend Auto-Update (`useServices.ts`)

The React hook already listens to WebSocket messages:

```typescript
ws.onmessage = (event) => {
    const update = JSON.parse(event.data)
    if (update.type === 'services') {
        setServices(update.services) // Triggers re-render
    }
}
```

## Flow Example

**Scenario:** User runs `azd provision` which creates an Azure Container App with URL `https://myapp-abc123.azurecontainerapps.io`

1. **Before provision:**
   - Dashboard shows service without Azure URL (only local info)

2. **During provision:**
   - Bicep deployment creates resources
   - Azure outputs include `SERVICE_API_URL=https://myapp-abc123.azurecontainerapps.io`

3. **After provision:**
   - azd updates process environment: `os.Setenv("SERVICE_API_URL", "https://...")`
   - azd fires "environment updated" event
   - Our handler:
     - Calls `RefreshEnvironmentCache()` → cache now has fresh Azure URL
     - Calls `GetServiceInfo()` → extracts `SERVICE_API_URL` from cache
     - Calls `BroadcastServiceUpdate()` → sends to all dashboard clients
   - Dashboard WebSocket receives update
   - UI re-renders showing Azure URL in service card

4. **Result:**
   - User sees Azure deployment URL appear in dashboard **instantly**
   - No page refresh needed
   - No manual intervention required

## Benefits

### Performance
- **Zero disk I/O:** All data stays in memory
- **Zero process spawning:** No `azd env get-values` subprocess
- **Instant updates:** WebSocket push vs. polling

### Reliability
- **Type-safe:** Uses azd's official extension SDK
- **Race-free:** Synchronized cache access with RWMutex
- **Error-resilient:** Failed broadcasts don't crash the extension

### Developer Experience
- **Seamless:** Works automatically after `azd provision`
- **Real-time:** Dashboard updates the moment provision completes
- **Multi-client:** All open dashboards update simultaneously

## Testing

To test the dynamic updates:

```bash
# 1. Start the run command (which launches dashboard)
azd app run

# 2. Open dashboard in browser (shown in terminal output)
http://localhost:40031

# 3. In another terminal, provision infrastructure
azd provision

# 4. Watch dashboard auto-update with Azure URLs/resource names!
```

## Environment Variable Patterns

The system recognizes these patterns from azd provision outputs:

| Pattern | Example | Maps To |
|---------|---------|---------|
| `SERVICE_{NAME}_URL` | `SERVICE_API_URL=https://...` | `api.azure.url` |
| `{NAME}_URL` | `API_URL=https://...` | `api.azure.url` (lower priority) |
| `SERVICE_{NAME}_NAME` | `SERVICE_API_NAME=myapp-api` | `api.azure.resourceName` |
| `{NAME}_NAME` | `API_NAME=myapp-api` | `api.azure.resourceName` (lower priority) |
| `SERVICE_{NAME}_IMAGE_NAME` | `SERVICE_API_IMAGE_NAME=myimage:latest` | `api.azure.imageName` |

The `SERVICE_` prefix pattern has **higher priority** to avoid conflicts with common environment variables.

## Related Files

- **Event Handler:** `cli/src/cmd/app/commands/listen.go`
- **Environment Cache:** `cli/src/internal/serviceinfo/serviceinfo.go`
- **WebSocket Broadcast:** `cli/src/internal/dashboard/server.go`
- **Frontend Hook:** `cli/dashboard/src/hooks/useServices.ts`

## Future Enhancements

Potential improvements:

1. **Granular updates:** Only broadcast changed services, not all
2. **Event debouncing:** Batch multiple rapid updates
3. **Reconnection handling:** Auto-reconnect WebSocket on network issues
4. **Offline resilience:** Queue updates when no clients connected
