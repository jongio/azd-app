# Service URL Fields - Multi-Field Design

## Context

Services need multiple URL configurations to support various access patterns across local development and Azure deployment scenarios. The current single-field approach (`azure.url`) is insufficient for complex workflows where services have:

- Auto-discovered local URLs (localhost with dynamic ports)
- Custom local URLs (via reverse proxy, ngrok, custom DNS)
- Auto-discovered Azure deployment URLs (from environment variables)
- Custom Azure URLs (user-configured custom domains, CDN endpoints)
- Azure-managed custom domains (retrieved via Azure SDK)

## Goals

- Provide a complete, hierarchical URL configuration system
- Support both auto-discovery and user customization at each layer
- Enable Azure SDK integration for custom domain discovery
- Maintain clear precedence rules for URL selection
- Preserve backward compatibility with existing single-field implementation

## Non-Goals

- Automatic DNS configuration or custom domain provisioning
- SSL certificate management
- Service mesh or traffic routing configuration
- Multi-region URL management
- Health checking or URL validation at runtime

## Requirements

### Field Definitions

#### 1. `local.url` (Auto-Generated)
- **Type:** `string` (read-only, system-generated)
- **Source:** Auto-discovered from running service (e.g., `http://localhost:3000`)
- **Purpose:** Default local development URL
- **User Control:** None (system-managed)
- **Example:** `http://localhost:3000`, `http://localhost:5173`

#### 2. `local.customUrl` (User-Configured)
- **Type:** `string` (optional, user-supplied in `azure.yaml`)
- **Source:** User configuration in `azure.yaml`
- **Purpose:** Override local URL for custom access patterns
- **User Control:** Full (configured in `azure.yaml`)
- **Use Cases:**
  - Reverse proxy (nginx, Caddy)
  - Tunneling services (ngrok, Cloudflare Tunnel, localtunnel)
  - Custom local DNS (`.local`, `.test` domains)
  - Load balancers
- **Precedence:** Overrides `local.url`
- **Example:** `https://myapp.ngrok.io`, `http://myapp.local:8080`

#### 3. `azure.url` (Auto-Discovered)
- **Type:** `string` (read-only, system-discovered)
- **Source:** Environment variables (e.g., `SERVICE_WEB_URL`, `AZURE_WEB_URL`)
- **Purpose:** Azure deployment URL (container app, app service, etc.)
- **User Control:** None (discovered from Azure deployment)
- **Example:** `https://web-abc123.azurecontainerapps.io`

#### 4. `azure.customUrl` (User-Configured)
- **Type:** `string` (optional, user-supplied in `azure.yaml`)
- **Source:** User configuration in `azure.yaml`
- **Purpose:** User-specified custom access URL for Azure deployments
- **User Control:** Full (configured in `azure.yaml`)
- **Use Cases:**
  - Custom domain (e.g., `https://www.mycompany.com`)
  - CDN endpoint (e.g., `https://cdn.mycompany.com`)
  - API gateway (e.g., `https://api.mycompany.com`)
  - Reverse proxy or load balancer in front of Azure service
- **Precedence:** Overrides `azure.url`
- **Example:** `https://www.mycompany.com`, `https://api.mycompany.com`

#### 5. `azure.customDomain` (Azure SDK Retrieved)
- **Type:** `string` (read-only, Azure SDK-discovered)
- **Source:** Azure SDK queries (custom domains configured in Azure Portal)
- **Purpose:** Custom domains configured in Azure (App Service, Container Apps, Front Door)
- **User Control:** Indirect (configured via Azure Portal, discovered by SDK)
- **Use Cases:**
  - App Service custom domains
  - Container Apps custom domains
  - Azure Front Door custom domains
  - Azure CDN custom domains
- **Precedence:** Overrides both `azure.customUrl` and `azure.url`
- **Example:** `https://www.mycompany.com`, `https://app.example.com`

### URL Precedence (Highest to Lowest)

#### Local Development (Dashboard/CLI)
1. `local.customUrl` (user override)
2. `local.url` (auto-discovered)

#### Azure Deployment (Console/Dashboard)
1. `azure.customDomain` (user-configured in azure.yaml, highest priority)
2. `azure.customDomain` (Azure SDK-discovered, if not user-configured)
3. `azure.customUrl` (user configuration)
4. `azure.url` (auto-discovered from env vars)

### Configuration Format

```yaml
# azure.yaml
services:
  web:
    project: ./src/web
    host: containerapp
    language: ts
    local:
      customUrl: https://myapp.ngrok.io  # Optional: local development override
    azure:
      customUrl: https://www.mycompany.com  # Optional: Azure deployment override
  
  api:
    project: ./src/api
    host: appservice
    language: python
    local:
      customUrl: http://api.local:8080  # Optional
    azure:
      customUrl: https://api.mycompany.com  # Optional
      customDomain: https://app.example.com  # Optional: override Azure SDK discovery
```

**Note:** 
- `local.url` and `azure.url` are NOT configured in `azure.yaml` - they are auto-discovered at runtime.
- `azure.customDomain` is optional in `azure.yaml` - if not configured, Azure SDK will attempt to discover it from Azure Portal. User configuration overrides SDK discovery.

### Data Model

#### TypeScript Interface

```typescript
export interface Service {
  name: string
  host: string
  language?: string
  
  local?: LocalServiceInfo
  azure?: AzureServiceInfo
}

export interface LocalServiceInfo {
  url?: string         // Auto-discovered (e.g., http://localhost:3000)
  customUrl?: string   // User-configured in azure.yaml
  // ... other local fields
}

export interface AzureServiceInfo {
  url?: string                // Auto-discovered from env vars
  customUrl?: string          // User-configured in azure.yaml
  customDomain?: string       // User-configured (overrides SDK) OR SDK-discovered
  customDomainSource?: 'user' | 'azure-sdk' // Indicates source of customDomain
  
  resourceName?: string
  resourceType?: string
  subscriptionId?: string
  resourceGroup?: string
  // ... other Azure fields
}
```

#### Go Model

```go
type Service struct {
    Name     string              `json:"name"`
    Host     string              `json:"host"`
    Language string              `json:"language,omitempty"`
    Local    *LocalServiceInfo   `json:"local,omitempty"`
    Azure    *AzureServiceInfo   `json:"azure,omitempty"`
}

type LocalServiceInfo struct {
    URL       string `json:"url,omitempty"`        // Auto-discovered
    CustomURL string `json:"customUrl,omitempty"`  // From azure.yaml
    // ... other fields
}

type AzureServiceInfo struct {
    URL                string `json:"url,omitempty"`                // From env vars
    CustomURL          string `json:"customUrl,omitempty"`          // From azure.yaml
    CustomDomain       string `json:"customDomain,omitempty"`       // User-configured (overrides SDK) OR SDK-discovered
    CustomDomainSource string `json:"customDomainSource,omitempty"` // "user" or "azure-sdk"
    
    ResourceName   string `json:"resourceName,omitempty"`
    ResourceType   string `json:"resourceType,omitempty"`
    SubscriptionID string `json:"subscriptionId,omitempty"`
    ResourceGroup  string `json:"resourceGroup,omitempty"`
    // ... other fields
}
```

### Dashboard UI Behavior

#### Local Development Tab
- **Display URL:** `local.customUrl` OR `local.url` (in that order)
- **Visual Indicator:** Purple badge if `local.customUrl` is active
- **Label:** 
  - "URL (Custom)" if using `local.customUrl`
  - "URL" if using `local.url`
- **Tooltip:** Show default URL when custom URL is used
  - Example: "Custom URL configured (default: http://localhost:3000)"
- **Navigation:** Click opens the effective URL

#### Azure Deployment Tab
- **Display URL:** `azure.customDomain` OR `azure.customUrl` OR `azure.url` (in that order)
- **Visual Indicators:**
  - Purple badge for `azure.customDomain` when user-configured
  - Gold badge for `azure.customDomain` when Azure SDK-discovered
  - Purple badge for `azure.customUrl` (user-configured)
  - Cyan badge for `azure.url` (auto-discovered)
- **Labels:**
  - "Custom Domain" if using user-configured `azure.customDomain`
  - "Custom Domain (Azure)" if using SDK-discovered `azure.customDomain`
  - "Custom URL" if using `azure.customUrl`
  - "Deployment URL" if using `azure.url`
- **Tooltip:** Show all URLs when overrides are active
  - Example (user-configured): "Custom domain (deployment: https://web-abc123.azurecontainerapps.io)"
  - Example (SDK-discovered): "Azure custom domain (Azure: https://www.mycompany.com, deployment: https://web-abc123.azurecontainerapps.io)"
- **Show All URLs:** Display all available URLs in detail panel
  - Deployment URL: `azure.url`
  - Custom URL: `azure.customUrl` (if configured)
  - Custom Domain: `azure.customDomain` (show source: user or Azure SDK)

### Console Output

```
Service: web
  Local URL: http://localhost:3000
  Custom Local URL: https://myapp.ngrok.io
  
  Azure Deployment URL: https://web-abc123.azurecontainerapps.io
  Azure Custom URL: https://www.mycompany.com
  Azure Custom Domain: https://app.mycompany.com
  
  Access URL: https://app.mycompany.com (custom domain)
```

**Simplified Output (when no overrides):**
```
Service: web
  Local URL: http://localhost:3000
  Azure URL: https://web-abc123.azurecontainerapps.io
```

### Validation Rules

#### `local.customUrl`
- Must be valid HTTP/HTTPS URL
- Must not be empty string
- Must have protocol (`http://` or `https://`)
- Must have host
- Length limit: 2048 characters

#### `azure.customUrl`
- Must be valid HTTP/HTTPS URL
- Recommend HTTPS for production
- Must not be empty string
- Must have protocol (`http://` or `https://`)
- Must have host
- Length limit: 2048 characters

#### `azure.customDomain`
- Must be valid HTTP/HTTPS URL (when user-configured)
- Recommend HTTPS for production
- Must not be empty string
- Must have protocol (`http://` or `https://`)
- Must have host
- Length limit: 2048 characters
- No validation when Azure SDK-discovered (trust Azure Portal config)

#### Read-Only Fields (No Validation)
- `local.url` - system-generated
- `azure.url` - from environment

### Azure SDK Integration

#### Custom Domain Discovery (Fallback)

**Precedence:** User-configured `azure.customDomain` in `azure.yaml` takes precedence. Azure SDK discovery is used only as fallback when not user-configured.

When `azure.customDomain` is NOT set in `azure.yaml`, the system queries Azure SDK to discover custom domain configurations:

**App Service:**
```go
import "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice"

// Get custom domains for App Service
client := armappservice.NewWebAppsClient(subscriptionID, credential, nil)
site, err := client.Get(ctx, resourceGroup, appName, nil)
if err == nil && site.Properties != nil {
    if site.Properties.HostNames != nil {
        for _, hostname := range site.Properties.HostNames {
            if !strings.HasSuffix(*hostname, ".azurewebsites.net") {
                customDomain = "https://" + *hostname
                break
            }
        }
    }
}
```

**Container Apps:**
```go
import "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers"

// Get custom domains for Container App
client := armappcontainers.NewContainerAppsClient(subscriptionID, credential, nil)
app, err := client.Get(ctx, resourceGroup, containerAppName, nil)
if err == nil && app.Properties != nil && app.Properties.Configuration != nil {
    if app.Properties.Configuration.Ingress != nil && app.Properties.Configuration.Ingress.CustomDomains != nil {
        for _, domain := range app.Properties.Configuration.Ingress.CustomDomains {
            if domain.Name != nil {
                customDomain = "https://" + *domain.Name
                break
            }
        }
    }
}
```

**Caching Strategy:**
- Cache Azure SDK-discovered `azure.customDomain` for 5 minutes
- User-configured `azure.customDomain` does not require caching (read from config)
- Refresh SDK cache on demand (user-initiated)
- Background refresh on service status changes
- Skip SDK query entirely if user has configured `azure.customDomain`

### CORS Considerations

When multiple URLs are configured, CORS must include all relevant origins:

```javascript
// Node.js example
const allowedOrigins = [
  process.env.LOCAL_URL,              // http://localhost:3000
  process.env.LOCAL_CUSTOM_URL,       // https://myapp.ngrok.io
  process.env.AZURE_URL,              // https://web-abc.azurecontainerapps.io
  process.env.AZURE_CUSTOM_URL,       // https://www.mycompany.com
  process.env.AZURE_CUSTOM_DOMAIN,    // https://app.mycompany.com (user-configured or SDK-discovered)
].filter(Boolean);

app.use(cors({
  origin: function(origin, callback) {
    if (!origin || allowedOrigins.includes(origin)) {
      callback(null, true);
    } else {
      callback(new Error('Not allowed by CORS'));
    }
  },
  credentials: true
}));
```

### Backward Compatibility

**Migration from Single-Field Design:**

Current implementation uses `azure.url` as user-configured field. To maintain compatibility:

1. **Reading existing configs:**
   - If `azure.url` is found in `azure.yaml` → treat as `azure.customUrl`
   - Emit deprecation warning: "azure.url is deprecated, use azure.customUrl instead"
   
2. **Migration path:**
   ```yaml
   # Old (deprecated but still works)
   services:
     web:
       url: https://www.mycompany.com  # Root-level
   
   # New (recommended)
   services:
     web:
       azure:
         customUrl: https://www.mycompany.com
   ```

3. **Gradual transition:**
   - Phase 1: Support both formats, emit warnings
   - Phase 2: Update all docs to new format
   - Phase 3 (future): Remove old format support (breaking change)

## Implementation Considerations

### Files to Modify

#### Configuration Parsing
- `cli/src/internal/appconfig/config.go` - Parse `local.customUrl`, `azure.customUrl`, and `azure.customDomain`
- `schemas/v1.1/azure.yaml.json` - Add schema definitions for new fields

#### Data Model
- `cli/dashboard/src/types.ts` - Update TypeScript interfaces
- `cli/src/internal/repository/app_config.go` - Update Go structs

#### Dashboard UI
- `cli/dashboard/src/components/ServiceCard.tsx` - Implement precedence logic
- `cli/dashboard/src/components/ServiceDetailPanel.tsx` - Show all URLs
- `cli/dashboard/src/components/ServiceTable.tsx` - Update table display

#### Azure SDK Integration
- `cli/src/internal/azure/customdomain.go` - NEW: Custom domain discovery
- `cli/src/internal/serviceinfo/serviceinfo.go` - Integrate SDK calls

#### Console Output
- `cli/src/internal/cmd/info.go` - Update formatting to show all URLs

### Performance Considerations

**Azure SDK Calls:**
- Make SDK calls asynchronous (don't block startup)
- Cache results (5-minute TTL)
- Fail gracefully if SDK unavailable
- Show spinner/loading state in dashboard
- Use parallel goroutines for multi-service discovery

**Dashboard Loading:**
- Load `local.url`, `local.customUrl`, `azure.url`, `azure.customUrl`, and user-configured `azure.customDomain` immediately
- Load SDK-discovered `azure.customDomain` asynchronously (only if not user-configured)
- Show placeholder while loading SDK-discovered custom domain
- Progressive enhancement (UI works without SDK-discovered custom domain)

### Error Handling

**Azure SDK Errors:**
- Log error but don't block app functionality
- Display fallback to `azure.customUrl` or `azure.url`
- Show warning indicator in dashboard if SDK call fails
- Retry logic with exponential backoff

**Validation Errors:**
- Fail fast on `azure.yaml` parse
- Clear error messages for invalid URLs
- Suggest corrections (e.g., "Did you mean https://...?")

### Testing Strategy

#### Unit Tests
- URL precedence logic (all 5 combinations, including user-configured vs SDK-discovered customDomain)
- Validation for `local.customUrl`, `azure.customUrl`, and `azure.customDomain`
- Azure SDK mock responses
- Error handling (SDK failures, invalid URLs)
- CustomDomainSource field tracking

#### Integration Tests
- Multi-field configuration parsing
- Dashboard URL display with all fields
- Console output formatting
- CORS configuration generation

#### Visual Tests
- Badge colors (purple, gold, cyan)
- Tooltip display
- Detail panel layout
- Loading states

## Open Questions

1. **Multiple Custom Domains:**
   - If Azure has multiple custom domains, which one do we select?
   - Options: First, primary, user-selectable
   - **Decision:** Use first custom domain, add UI to select if multiple exist

2. **Environment-Specific URLs:**
   - Should we support per-environment custom URLs (dev, staging, prod)?
   - **Decision:** Out of scope for v1, revisit in future

3. **URL Health Checks:**
   - Should we validate that custom URLs are reachable?
   - **Decision:** No automatic validation, trust user configuration

4. **Azure SDK Credentials:**
   - What credentials to use for SDK calls?
   - Options: User's Azure CLI credentials, Managed Identity, explicit auth
   - **Decision:** Use Azure CLI credentials (same as azd)

5. **Custom Domain SSL:**
   - Should we indicate HTTPS/TLS status for custom domains?
   - **Decision:** Display protocol as-is from Azure, no validation

6. **User Override of SDK-Discovered Domain:**
   - If user configures `azure.customDomain`, should we still query Azure SDK?
   - **Decision:** No, skip SDK query entirely to improve performance. User configuration takes full precedence.

7. **Distinguishing User vs SDK Source:**
   - How to visually indicate if customDomain is user-configured vs SDK-discovered?
   - **Decision:** Use purple badge for user-configured, gold badge for SDK-discovered. Include source in tooltip.

## Success Criteria

- ✅ All 5 URL fields can be configured/discovered independently
- ✅ `azure.customDomain` can be user-configured in `azure.yaml` OR auto-discovered via Azure SDK
- ✅ User-configured `azure.customDomain` takes precedence over SDK-discovered value
- ✅ SDK query is skipped when `azure.customDomain` is user-configured (performance optimization)
- ✅ Precedence rules enforced consistently (local and Azure)
- ✅ Dashboard displays correct URL based on precedence
- ✅ Dashboard indicates source of `azure.customDomain` (user vs Azure SDK) with badges
- ✅ Console output shows all available URLs clearly
- ✅ Azure SDK integration retrieves custom domains successfully (when not user-configured)
- ✅ Backward compatible with existing single-field implementation
- ✅ CORS guide updated for multi-URL scenarios
- ✅ Clear visual indicators for each URL type
- ✅ Tests cover all precedence combinations (including user vs SDK customDomain)
- ✅ Performance: SDK calls don't block startup, skipped when user-configured
- ✅ Error handling: Graceful degradation if SDK unavailable

## Documentation Requirements

### User Documentation
- **azure-yaml Reference:** Document all 5 fields with examples, emphasizing dual nature of `azure.customDomain` (user-configured OR SDK-discovered)
- **CORS Guide:** Multi-URL CORS configuration patterns
- **Custom Domain Guide:** NEW - How Azure custom domains work (user override vs SDK discovery)
- **Migration Guide:** Upgrade from single-field to multi-field design

### Developer Documentation
- **Architecture:** URL precedence flow diagrams
- **Azure SDK Integration:** Custom domain discovery patterns
- **Testing Guide:** How to test multi-URL scenarios

## Risks & Mitigation

### Risk 1: Azure SDK Dependency
- **Impact:** App depends on Azure SDK availability
- **Mitigation:** Graceful degradation, cache results, async loading

### Risk 2: Performance Impact
- **Impact:** SDK calls could slow down dashboard
- **Mitigation:** Async loading, caching, parallel requests

### Risk 3: Complexity
- **Impact:** 5 fields is complex for users
- **Mitigation:** Progressive disclosure, good defaults, clear docs

### Risk 4: Breaking Changes
- **Impact:** Migration from single-field could break existing configs
- **Mitigation:** Backward compat layer, deprecation warnings, migration guide

## Timeline

### Phase 1: Schema & Data Model (Week 1)
- Update `azure.yaml` schema
- Update TypeScript and Go types
- Configuration parsing

### Phase 2: Azure SDK Integration (Week 2)
- Implement custom domain discovery
- Add caching layer
- Error handling

### Phase 3: Dashboard UI (Week 2-3)
- Update ServiceCard, ServiceDetailPanel, ServiceTable
- Visual indicators (badges, colors)
- Loading states

### Phase 4: Console & CORS (Week 3)
- Update console output formatting
- Update CORS guide
- Environment variable handling

### Phase 5: Testing & Docs (Week 4)
- Unit and integration tests
- Visual tests
- Documentation updates
- Migration guide

## Alternatives Considered

### Alternative 1: Single Computed Field
- **Approach:** Keep single `url` field, compute internally
- **Rejected:** Lacks transparency, hard to debug

### Alternative 2: Array of URLs
- **Approach:** `urls: [{type: 'local', url: '...'}, ...]`
- **Rejected:** Complex configuration, unclear precedence

### Alternative 3: Environment-Scoped URLs
- **Approach:** `urls: {dev: '...', prod: '...'}`
- **Rejected:** Out of scope, adds environment concept

## Appendix

### Example Configurations

#### Minimal (No Overrides)
```yaml
services:
  web:
    project: ./web
    host: containerapp
```

**Result:**
- `local.url`: `http://localhost:3000` (auto-discovered)
- `azure.url`: `https://web-abc.azurecontainerapps.io` (from env)

#### Local Development Override
```yaml
services:
  web:
    project: ./web
    host: containerapp
    local:
      customUrl: https://myapp.ngrok.io
```

**Result:**
- `local.url`: `http://localhost:3000`
- `local.customUrl`: `https://myapp.ngrok.io` ✅ **Used**
- `azure.url`: `https://web-abc.azurecontainerapps.io`

#### Full Configuration
```yaml
services:
  web:
    project: ./web
    host: containerapp
    local:
      customUrl: https://myapp.ngrok.io
    azure:
      customUrl: https://www.mycompany.com
```

**Result:**
- `local.url`: `http://localhost:3000`
- `local.customUrl`: `https://myapp.ngrok.io` ✅ **Local access**
- `azure.url`: `https://web-abc.azurecontainerapps.io`
- `azure.customUrl`: `https://www.mycompany.com` ✅ **Azure access (if no customDomain)**
- `azure.customDomain`: `https://app.mycompany.com` ✅ **Azure access (if exists)**

---

**Spec Version:** 1.0  
**Date:** January 12, 2026  
**Status:** Draft  
**Authors:** Product Team  
**Reviewers:** TBD
