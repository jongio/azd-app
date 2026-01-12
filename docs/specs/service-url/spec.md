# Service URL Configuration

## Context
Users may need to access services through custom URLs in multiple scenarios:
1. **Local development**: Tunneling services (ngrok, localtunnel) for external access or webhook testing
2. **Azure deployment**: Custom domains configured on App Service or Container Apps
3. **Manual overrides**: Reverse proxies, load balancers, or other access patterns

Currently, `azd app` only uses system-generated URLs (localhost, *.azurewebsites.net). This creates friction when services are accessed through different endpoints.

## Goals
- Detect custom domains configured on Azure resources automatically
- Allow users to configure custom URL overrides for both local and Azure contexts
- Honor effective URLs (customUrl > customDomain > url) when launching services from the dashboard UI
- Display all available URLs in console output and UI when configured
- Support CORS configuration for services accessed via custom URLs

## Non-Goals
- Modifying the underlying service deployment or infrastructure
- Automatic discovery or validation of custom URL reachability
- Changing the internal service-to-service communication patterns

## Requirements

### URL Hierarchy

**Clear Separation: System-Generated vs User-Provided**

Each service context (local and Azure) has URLs from different sources:

**Local Context:**
1. **url** - System-generated (always `http://localhost:{port}`)
2. **customUrl** - User-provided in `azure.yaml` `local.customUrl` (overrides url)

**Azure Context:**
1. **url** - System-generated from environment variables (e.g., `https://myapp-abc123.azurewebsites.net`)
2. **customUrl** - User-provided in `azure.yaml` `azure.customUrl` (overrides url)
3. **customDomain** - Azure-detected OR user-provided in `azure.yaml` `azure.customDomain` (overrides customUrl and url)

**Precedence Chains:**
- **Local**: `customUrl > url`
- **Azure**: `customDomain > customUrl > url`

**Key Design Principle**: 
- `url` fields are NEVER in azure.yaml (system-generated only)
- `customUrl` and `customDomain` fields ARE in azure.yaml (user-provided)
- No naming confusion between config and runtime

### Configuration Format
Users specify custom URLs in `azure.yaml` using clearly named fields:

```yaml
services:
  web:
    project: ./src/web
    host: appservice
    language: ts
    local:
      customUrl: https://abc.ngrok-free.app  # Overrides localhost:3000
    azure:
      customUrl: https://cdn.myapp.example.com  # Overrides system Azure URL
      customDomain: https://myapp.example.com   # Optional: user can specify instead of auto-detect
  
  api:
    project: ./src/api
    host: containerapp
    language: python
    local:
      customUrl: https://api-xyz.ngrok-free.app
    # No azure customUrl/customDomain - will use auto-detected customDomain if exists, otherwise system url
```

**Note**: `azure.customDomain` can be:
- Auto-detected from Azure resource (App Service/Container App custom domain)
- User-provided in azure.yaml (takes precedence over auto-detection)
- If both exist, user-provided value wins

### Runtime API Response
The `/api/services` endpoint returns all available URLs:

```json
{
  "name": "web",
  "local": {
    "url": "http://localhost:3000",           // System-generated (never from config)
    "customUrl": "https://abc.ngrok-free.app" // From azure.yaml local.customUrl
  },
  "azure": {
    "url": "https://myapp-abc123.azurewebsites.net", // System-generated from env var
    "customUrl": "https://cdn.example.com",           // From azure.yaml azure.customUrl
    "customDomain": "https://myapp.example.com"       // From azure.yaml OR auto-detected
  }
}
```

All fields are always present (null if not applicable). Frontend applies precedence chain.

### Custom Domain Detection
For Azure services, custom domains can be:

**Auto-detected from Azure resources:**
- **App Service**: Query `properties.customDomains` via Azure SDK, use first HTTPS-enabled custom domain
- **Container Apps**: Query `properties.configuration.ingress.customDomains`, use first custom domain with valid certificate
- Detection runs during `azd app run` startup (initial load), cached for session duration

**User-provided in azure.yaml:**
- Users can explicitly set `azure.customDomain` in `azure.yaml`
- User-provided value takes precedence over auto-detected value
- Useful when Azure resource has multiple custom domains and user wants specific one

**Fallback behavior:**
- If user provides `azure.customDomain` in config → use that value
- Else if custom domain detected via Azure SDK → use detected value
- Else → use system URL from environment variables

**Detection timing:**
- During `azd app run` startup (initial load)
- Cached for session duration
- Refreshed on service restart/redeploy
- SDK failures → log warning, continue with system URL

### Dashboard UI Behavior
- **Service card**: Display effective URL (using precedence chain) as primary
- **Detail panel**: Show all available URLs with labels:
  - "System URL" - Always shown
  - "Custom Domain" - Shown when detected from Azure
  - "Custom URL (Override)" - Shown when configured in azure.yaml
- **Open in browser**: Use effective URL from precedence chain
- **URL indication**: Visual indicator (icon/badge) when using non-system URL

### Console Output
When printing service URLs (e.g., during `azd app run`), display all available URLs:

```
Service: web
  System URL: http://localhost:3000
  Custom URL: https://abc.ngrok-free.app
  → Opening: https://abc.ngrok-free.app

Service: api (Azure)
  System URL: https://api-abc123.azurewebsites.net
  Custom Domain: https://api.myapp.example.com
  → Opening: https://api.myapp.example.com

Service: cdn (Azure - all three URLs)
  System URL: https://cdn-abc123.azurewebsites.net
  Custom URL: https://api-cdn.example.com
  Custom Domain: https://cdn.example.com
  → Opening: https://cdn.example.com
```

- Show all URLs that exist
- Indicate which URL is used for "open in browser" action (→ arrow)
- Clear labeling to distinguish URL types
- Precedence: Custom Domain > Custom URL > System URL

### CORS Handling
- For services that use CORS (typically APIs), all custom URLs must be included in CORS allowed origins
- Include: customUrl (if configured) and customDomain (if detected)
- CORS configuration should be updated automatically during deployment
- This applies to both Azure App Service and Container Apps CORS settings
- Local development mode should also respect custom URLs for CORS configuration

### API and Data Model
- Extend service configuration model to include `local.customUrl`, `azure.customUrl`, and `azure.customDomain` fields in azure.yaml
- `url` fields are NEVER in azure.yaml - they are system-generated only
- Dashboard API must return `url`, `customUrl`, and `customDomain` for each context
- Custom domain merging: user-provided `azure.customDomain` takes precedence over auto-detected
- Browser launch logic must apply precedence chains:
  - Local: `customUrl || url`
  - Azure: `customDomain || customUrl || url`
- Console formatting utilities must display all available URLs

### Validation
- Custom URLs should be valid HTTP/HTTPS URLs
- Provide warning if custom URL is configured but appears unreachable (non-blocking)
- No validation required for URL reachability during configuration parse
- Custom domain detection failures should not block `azd app run` (log warning, continue with system URL)

## UX and Validation Notes
- Configuration parsing must fail gracefully if `url` is malformed, with clear error messages
- Dashboard should handle scenarios where custom URLs are unreachable without breaking the UI
- Console output should maintain consistent formatting regardless of URL availability
- Clearly label each URL type to avoid user confusion
- When customUrl overrides customDomain, consider logging an info message explaining the override

## Implementation Considerations

### Files Likely to Change
- `cli/src/internal/appconfig/config.go` - Parse `local.customUrl`, `azure.customUrl`, `azure.customDomain` from `azure.yaml`
- `cli/src/internal/repository/app_config.go` - Service configuration model with CustomURL and CustomDomain fields
- `cli/src/internal/azure/resources.go` - Custom domain detection via Azure SDK (merge with user-provided)
- `cli/src/internal/serviceinfo/serviceinfo.go` - Merge logic for user-provided vs auto-detected customDomain
- `cli/dashboard/src/types/service.ts` - TypeScript service interface with all URL fields
- `cli/dashboard/src/components/ServiceCard.tsx` - Display effective URL with updated precedence
- `cli/dashboard/src/components/ServiceDetailPanel.tsx` - Show all available URLs
- `cli/dashboard/src/components/LogsPane.tsx` - Apply precedence chain for browser launch
- `cli/src/internal/service/logger.go` - Console output for all URLs
- `cli/schemas/azure.yaml.json` - Schema update for customUrl and customDomain fields

### Custom Domain Detection Implementation
**Azure SDK Integration:**
- Use Azure SDK for Go (App Service and Container Apps clients)
- Query during service initialization in `azd app run`
- Cache results for session duration
- Handle authentication via existing Azure credentials

**App Service:**
```go
// Use azure-sdk-for-go/sdk/resourcemanager/appservice
client.Get(ctx, resourceGroup, appName)
// Extract: properties.hostNames, filter custom domains
```

**Container Apps:**
```go
// Use azure-sdk-for-go/sdk/resourcemanager/appcontainers  
client.Get(ctx, resourceGroup, containerAppName)
// Extract: properties.configuration.ingress.customDomains
```

**Error Handling:**
- SDK call failures → log warning, continue with system URL
- Network timeouts → use cached value if available, otherwise system URL
- Authentication failures → log error, continue with system URL

### Backward Compatibility
**NOT REQUIRED** - This is a breaking change.

- No support for legacy flat `url` field
- Existing `azure.yaml` files with `url` field will need to be migrated to `customUrl`
- Migration is straightforward: `url` → `local.customUrl` or `azure.customUrl`
- Clear error message if old `url` field is detected

## Open Questions
- ~~Should we support environment-specific custom URLs (e.g., different URLs for dev, staging, prod)?~~ **Resolved: Yes, via local.url and azure.url**
- ~~Should custom URL override both read and write operations, or only display/navigation?~~ **Resolved: Display/navigation only**
- ~~Should we validate that the custom URL actually reaches the service, or trust user configuration?~~ **Resolved: Trust user, warn if unreachable (non-blocking)**
- How should we handle custom URLs for services behind authentication?
- ~~Should the dashboard show both URLs with a toggle, or only the custom URL when configured?~~ **Resolved: Show all URLs with clear labels**
- Should we cache custom domain detection results to disk for faster startup?
- What happens if App Service has multiple custom domains configured? (Currently: use first HTTPS-enabled)

## Success Criteria
- Users can configure `customUrl` and `customDomain` in `azure.yaml` without errors
- Clear separation: `url` fields never in config (system-generated only)
- Custom domains can be auto-detected OR user-provided (user-provided wins)
- Clicking "Open" in dashboard navigates to effective URL with correct precedence:
  - Local: `customUrl > url`
  - Azure: `customDomain > customUrl > url`
- Console output displays all available URLs with clear labels and precedence indicator (→)
- Dashboard UI shows all URL types when they exist
- CORS configuration automatically includes all custom URLs and domains
- No confusion between config fields and runtime fields (customUrl in config, url system-only)
- Custom domain detection failures do not block `azd app run`
- Documentation clearly explains configuration format, precedence, and use cases
