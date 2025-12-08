# Codespace URL Forwarding Tasks

## TODO

### 1. Add backend /api/environment endpoint
- **Agent**: Developer
- **File**: `cli/src/internal/dashboard/server.go`
- **Criteria**:
  - Add `handleGetEnvironment()` handler
  - Return JSON with codespace detection: `{codespace: {enabled, name, domain}}`
  - Detect via `CODESPACE_NAME` and `GITHUB_CODESPACES_PORT_FORWARDING_DOMAIN` env vars
  - Register route in `setupRoutes()`

### 2. Create codespace-utils.ts
- **Agent**: Developer
- **File**: `cli/dashboard/src/lib/codespace-utils.ts`
- **Criteria**:
  - `transformLocalhostUrl(url, codespaceConfig)` - transform localhost to forwarded URL
  - Handle variants: `localhost`, `127.0.0.1`, `0.0.0.0`, `[::1]`
  - Preserve path and query string
  - Return original URL when not in Codespace

### 3. Create useCodespaceEnv hook
- **Agent**: Developer
- **File**: `cli/dashboard/src/hooks/useCodespaceEnv.ts`
- **Criteria**:
  - Fetch `/api/environment` once on mount
  - Cache result in state
  - Export `useCodespaceEnv()` returning `{isCodespace, codespaceName, domain}`
  - Handle loading and error states

### 4. Update getServiceUrl to transform URLs
- **Agent**: Developer
- **File**: `cli/dashboard/src/lib/dependencies-utils.ts`
- **Criteria**:
  - Import codespace utilities
  - Add optional codespaceConfig parameter
  - Apply URL transformation when in Codespace
  - Maintain backward compatibility

### 5. Update ServiceCard URL display
- **Agent**: Developer
- **File**: `cli/dashboard/src/components/ServiceCard.tsx`
- **Criteria**:
  - Use useCodespaceEnv hook
  - Transform localUrl through codespace utility
  - Display transformed URL in link

### 6. Update log URL linkification
- **Agent**: Developer
- **File**: `cli/dashboard/src/lib/log-utils.ts`
- **Criteria**:
  - Update `linkifyUrlsWithHtmlAware()` to accept codespace config
  - Transform localhost URLs in logs to Codespace URLs
  - Update `convertAnsiToHtml()` signature if needed

### 7. Unit tests for codespace-utils
- **Agent**: Tester
- **File**: `cli/dashboard/src/lib/codespace-utils.test.ts`
- **Criteria**:
  - Test localhost transformation
  - Test 127.0.0.1 transformation
  - Test URL with path preservation
  - Test URL with query string
  - Test non-Codespace passthrough
  - Test port 0 handling
  - Coverage >= 80%

### 8. E2E test for Codespace URLs
- **Agent**: Tester
- **File**: `cli/dashboard/e2e/codespace.spec.ts`
- **Criteria**:
  - Mock /api/environment response
  - Verify service URLs show Codespace domain
  - Verify clicking URL targets correct domain
