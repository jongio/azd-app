# Dashboard Codespace URL Forwarding

## Problem

When running `azd app run` in GitHub Codespaces, the dashboard displays `localhost` URLs for services (e.g., `http://localhost:3000`). These URLs don't work when clicked because Codespaces requires forwarded URLs in the format:

```
https://{codespace-name}-{port}.app.github.dev
```

For example, if a service runs on port 3000 in a Codespace named `silver-space-xyzzy`, the accessible URL is:
```
https://silver-space-xyzzy-3000.app.github.dev
```

## Requirements

1. Detect when running in a GitHub Codespace environment
2. Transform localhost URLs to Codespace-forwarded URLs in all dashboard UI locations
3. Handle URLs in logs that reference localhost ports
4. Support both VS Code Codespaces and github.dev browser-based Codespaces

## Solution

### Environment Detection

Detect Codespace environment in the dashboard frontend via:
- `CODESPACE_NAME` - contains the Codespace name (e.g., `silver-space-xyzzy`)
- `GITHUB_CODESPACES_PORT_FORWARDING_DOMAIN` - contains the domain suffix (e.g., `app.github.dev`)

These environment variables need to be exposed to the dashboard. Two approaches:

**Option A (Recommended)**: Backend API endpoint that returns environment info
- Add `/api/environment` endpoint that returns Codespace detection info
- Frontend fetches this once on load
- More secure - backend controls what's exposed

**Option B**: Inject via HTML template at build/serve time
- Less flexible, requires rebuild

### URL Transformation Function

Create a utility function to transform localhost URLs:

```typescript
// Input: http://localhost:3000
// Output (in Codespace): https://silver-space-xyzzy-3000.app.github.dev
// Output (local): http://localhost:3000 (unchanged)
```

Transformation logic:
1. Parse the localhost URL to extract port
2. Build Codespace URL: `https://{codespaceName}-{port}.{domain}`
3. Preserve path and query string from original URL

### Affected Components

1. **ServiceCard.tsx** - Service URL link display
2. **ServiceDetailPanel.tsx** - Service detail URL display
3. **ServiceTable.tsx** - Table view URL display
4. **LogsPane.tsx / LogsView.tsx** - Linkified URLs in log output
5. **dependencies-utils.ts** - `getServiceUrl()` function

### Implementation Approach

1. Create `codespace-utils.ts` with:
   - `isCodespaceEnvironment()` - detection
   - `transformLocalhostUrl(url: string)` - URL transformation
   - `getCodespaceUrl(port: number)` - build URL from port

2. Add backend `/api/environment` endpoint returning:
   ```json
   {
     "codespace": {
       "enabled": true,
       "name": "silver-space-xyzzy",
       "domain": "app.github.dev"
     }
   }
   ```

3. Create `useCodespaceEnv()` hook to fetch/cache environment info

4. Update `getServiceUrl()` to apply transformation when in Codespace

5. Update log URL linkification to transform localhost URLs

## Edge Cases

- Multiple services on different ports
- URLs with paths (e.g., `http://localhost:3000/api/health`)
- URLs with query strings
- IPv4/IPv6 localhost variants (`127.0.0.1`, `[::1]`)
- Port 0 (dynamic/unavailable) - skip transformation
- Non-HTTP services (TCP, process) - no URL transformation needed

## Testing

- Unit tests for URL transformation function
- E2E tests with mock Codespace environment variables
- Test localhost variants: `localhost`, `127.0.0.1`, `0.0.0.0`
- Test URL with path preservation
- Test graceful fallback when not in Codespace

## Files to Create/Modify

### New Files
- `cli/dashboard/src/lib/codespace-utils.ts` - URL transformation utilities
- `cli/dashboard/src/lib/codespace-utils.test.ts` - Unit tests
- `cli/dashboard/src/hooks/useCodespaceEnv.ts` - Environment detection hook

### Modified Files
- `cli/src/internal/dashboard/server.go` - Add `/api/environment` endpoint
- `cli/dashboard/src/lib/dependencies-utils.ts` - Update `getServiceUrl()`
- `cli/dashboard/src/lib/log-utils.ts` - Update URL linkification
- `cli/dashboard/src/components/ServiceCard.tsx` - Use transformed URLs
- `cli/dashboard/src/contexts/ServicesContext.tsx` - Integrate environment context

## Success Criteria

- Clicking service URLs in Codespace opens the correct forwarded URL
- URLs in logs are correctly transformed and clickable
- Local development (non-Codespace) continues to work unchanged
- No visible delay or flicker during URL transformation
- Tests pass with >80% coverage on new code
