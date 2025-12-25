# Service Alternate URL Configuration

## Context
Users may need to access services through alternate URLs (e.g., reverse proxies, custom domains, load balancers, or tunneling services like ngrok). Currently, `azd app` always uses the direct service URLs for launching browsers and displaying links in the console. This creates friction when services are accessed through alternate endpoints that require different CORS configurations.

## Goals
- Allow users to configure alternate URLs for each service
- Honor alternate URLs when launching services from the dashboard UI
- Display alternate URLs in console output when configured
- Support CORS configuration for services accessed via alternate URLs

## Non-Goals
- Modifying the underlying service deployment or infrastructure
- Automatic discovery or validation of alternate URLs
- Supporting multiple alternate URLs per service (single override only)
- Changing the internal service-to-service communication patterns

## Requirements

### Configuration
- Users must be able to specify an alternate URL for each service in `azure.yaml`
- Configuration format should be intuitive and follow existing `azure.yaml` conventions
- Alternate URL configuration must be optional (existing behavior is default)
- Configuration should support both Azure and local services

### Proposed Configuration Format
```yaml
services:
  web:
    project: ./src/web
    host: appservice
    language: ts
    config:
      altUrl: https://myapp.example.com
  
  api:
    project: ./src/api
    host: containerapp
    language: python
    config:
      altUrl: https://api.myapp.example.com
```

### Dashboard UI Behavior
- When a service has an `altUrl` configured, clicking "Open" in the dashboard must navigate to the alternate URL
- The service status card should indicate when an alternate URL is in use (e.g., display the alternate URL instead of or alongside the default URL)
- Hover tooltips or info icons should clarify which URL is the actual deployment and which is the alternate access point

### Console Output
- When printing service URLs (e.g., during `azd up`, `azd deploy`, or `azd app endpoints`), display the alternate URL if configured
- Console output should clearly distinguish between the deployment URL and alternate URL
- Format example:
  ```
  Service: web
    Deployment URL: https://myapp-abc123.azurewebsites.net
    Access URL: https://myapp.example.com
  ```

### CORS Handling
- For services that use CORS (typically APIs), the alternate URL origin must be included in CORS allowed origins
- CORS configuration should be updated automatically during deployment when `altUrl` is present
- This applies to both Azure App Service and Container Apps CORS settings
- Local development mode should also respect alternate URL for CORS configuration

### API and Data Model
- Extend the service configuration model to include optional `altUrl` field
- Dashboard API must return `altUrl` when available
- Browser launch logic must check for `altUrl` and prefer it over default URL
- Console formatting utilities must incorporate `altUrl` display logic

### Validation
- Alternate URLs should be valid HTTP/HTTPS URLs
- Provide warning if alternate URL is configured but appears unreachable (non-blocking)
- No validation required for URL reachability during configuration parse

## UX and Validation Notes
- Configuration parsing must fail gracefully if `altUrl` is malformed, with clear error messages
- Dashboard should handle scenarios where alternate URL is unreachable without breaking the UI
- Console output should maintain consistent formatting whether alternate URL is configured or not
- If both deployment URL and alternate URL are shown, clearly label which is which to avoid user confusion

## Implementation Considerations

### Files Likely to Change
- `cli/src/internal/appconfig/config.go` - Parse `altUrl` from `azure.yaml`
- `cli/src/internal/repository/app_config.go` - Service configuration model
- `cli/dashboard/src/types/service.ts` - TypeScript service interface
- `cli/dashboard/src/components/ServiceCard.tsx` - Display and launch logic
- `cli/src/internal/apphost/generate.go` - CORS configuration generation
- Console formatting utilities for service URL output

### CORS Configuration Updates
- Azure App Service: Update `cors.allowedOrigins` in bicep/arm templates
- Container Apps: Update ingress CORS settings
- Local development: Update development server CORS middleware

### Backward Compatibility
- Services without `altUrl` must continue to work exactly as before
- Existing `azure.yaml` files without this configuration remain valid
- Default behavior unchanged when feature is not used

## Open Questions
- Should we support environment-specific alternate URLs (e.g., different URLs for dev, staging, prod)?
- Should alternate URL override both read and write operations, or only display/navigation?
- Should we validate that the alternate URL actually reaches the service, or trust user configuration?
- How should we handle alternate URLs for services behind authentication?
- Should the dashboard show both URLs with a toggle, or only the alternate URL when configured?

## Success Criteria
- Users can configure alternate URLs in `azure.yaml` without errors
- Clicking "Open" in dashboard navigates to alternate URL when configured
- Console output displays alternate URLs clearly
- CORS configuration automatically includes alternate URL origins
- No regression in behavior for services without alternate URL configuration
- Documentation clearly explains configuration format and use cases
