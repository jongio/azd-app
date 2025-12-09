# Azure Error States Design Spec

Component design spec for Azure log error handling UI.

## Overview

Consistent, actionable error handling for Azure log streaming. Each error type has specific UI treatment with guidance to resolve the issue.

## Error Types

### 1. Authentication Error

**Trigger**: No credentials, expired token, invalid token

**Error indicators**:
- `azd auth login`
- `unauthorized`
- `401`
- `authentication required`
- `ErrAuthNotConfigured`

**UI Treatment**:
```
┌─────────────────────────────────────────────────────────────┐
│  🔐  Authentication Required                                 │
│                                                             │
│  Sign in to Azure to view cloud logs.                       │
│                                                             │
│  Run this command in your terminal:                         │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  azd auth login                                [📋] │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  [Retry Connection]                                         │
└─────────────────────────────────────────────────────────────┘
```

**Colors**: 
- Icon: amber-500
- Border: amber-200 dark:amber-800

---

### 2. Permission Denied

**Trigger**: User authenticated but lacks required roles

**Error indicators**:
- `403`
- `AuthorizationFailed`
- `does not have authorization`
- `permission`
- `RBAC`
- `Reader`

**UI Treatment**:
```
┌─────────────────────────────────────────────────────────────┐
│  🚫  Permission Denied                                       │
│                                                             │
│  Your account doesn't have access to query logs.           │
│                                                             │
│  Required role: Log Analytics Reader                        │
│                                                             │
│  Ask your Azure administrator to grant:                     │
│  • Log Analytics Reader on the workspace                   │
│  • Reader on the resource group                            │
│                                                             │
│  [View Azure RBAC Docs ↗]    [Retry]                       │
└─────────────────────────────────────────────────────────────┘
```

**Colors**:
- Icon: rose-500
- Border: rose-200 dark:rose-800

---

### 3. Resource Not Found

**Trigger**: Service not deployed, wrong resource group, deleted resource

**Error indicators**:
- `404`
- `ResourceNotFound`
- `not found`
- `does not exist`

**UI Treatment**:
```
┌─────────────────────────────────────────────────────────────┐
│  🔍  Resource Not Found                                      │
│                                                             │
│  Service "api" not found in Azure.                         │
│                                                             │
│  This service may not be deployed yet.                     │
│                                                             │
│  Deploy to Azure:                                           │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  azd provision                                 [📋] │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  [View Local Logs Instead]                                  │
└─────────────────────────────────────────────────────────────┘
```

**Colors**:
- Icon: slate-500
- Border: slate-200 dark:slate-700

---

### 4. Rate Limited

**Trigger**: Too many API requests

**Error indicators**:
- `429`
- `TooManyRequests`
- `rate limit`
- `throttled`

**UI Treatment**:
```
┌─────────────────────────────────────────────────────────────┐
│  ⏳  Rate Limited                                            │
│                                                             │
│  Too many requests to Azure. Please wait.                   │
│                                                             │
│  Retry in: 32s  [████████░░░░░░░░░░░░]                     │
│                                                             │
│  [Retry Now]                                                │
└─────────────────────────────────────────────────────────────┘
```

**Features**:
- Countdown timer from Retry-After header
- Progress bar
- Auto-retry when timer expires

**Colors**:
- Icon: amber-500
- Border: amber-200 dark:amber-800

---

### 5. Network Error

**Trigger**: Connection failure, timeout, DNS error

**Error indicators**:
- `network`
- `timeout`
- `ECONNREFUSED`
- `ETIMEDOUT`
- `DNS`
- `connect`

**UI Treatment**:
```
┌─────────────────────────────────────────────────────────────┐
│  🌐  Connection Failed                                       │
│                                                             │
│  Unable to reach Azure services.                           │
│                                                             │
│  Check your network connection and try again.              │
│                                                             │
│  [Retry]  [Use Local Logs]                                 │
└─────────────────────────────────────────────────────────────┘
```

**Colors**:
- Icon: sky-500
- Border: sky-200 dark:sky-800

---

### 6. Log Analytics Not Configured

**Trigger**: Workspace not found, diagnostics not enabled

**Error indicators**:
- `workspace`
- `Log Analytics`
- `diagnostic settings`
- `WorkspaceNotFound`

**UI Treatment**:
```
┌─────────────────────────────────────────────────────────────┐
│  📊  Log Analytics Not Configured                            │
│                                                             │
│  No Log Analytics workspace found for this resource.       │
│                                                             │
│  Configure logging in azure.yaml:                           │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  logs:                                              │   │
│  │    azure:                                           │   │
│  │      workspace: "your-workspace-id"                 │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  [View Setup Guide ↗]                                       │
└─────────────────────────────────────────────────────────────┘
```

**Colors**:
- Icon: violet-500
- Border: violet-200 dark:violet-800

---

### 7. Query Error

**Trigger**: Invalid KQL syntax, unsupported query

**Error indicators**:
- `query`
- `syntax`
- `BadArgumentError`
- `KQL`

**UI Treatment**:
```
┌─────────────────────────────────────────────────────────────┐
│  ⚠️  Query Error                                             │
│                                                             │
│  Invalid query syntax:                                      │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Syntax error at line 3: unexpected token '|'       │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  [Reset to Default Query]  [Edit Query]                    │
└─────────────────────────────────────────────────────────────┘
```

**Colors**:
- Icon: amber-500
- Border: amber-200 dark:amber-800

---

### 8. Generic Error (Fallback)

**Trigger**: Unrecognized error

**UI Treatment**:
```
┌─────────────────────────────────────────────────────────────┐
│  ❌  Something went wrong                                    │
│                                                             │
│  {error message}                                            │
│                                                             │
│  [Retry]  [Report Issue ↗]                                 │
└─────────────────────────────────────────────────────────────┘
```

**Colors**:
- Icon: red-500
- Border: red-200 dark:red-800

---

## Component Structure

```
AzureErrorDisplay
├── ErrorIcon (type-specific)
├── ErrorTitle
├── ErrorDescription
├── ActionableGuidance
│   ├── CommandCopy (for CLI commands)
│   ├── CodeSnippet (for config examples)
│   └── ExternalLink (for docs)
├── ErrorActions
│   ├── PrimaryAction (Retry, Login, etc.)
│   └── SecondaryAction (View Local, Report, etc.)
└── CountdownTimer (for rate limits)
```

---

## Props Interface

```typescript
type AzureErrorType = 
  | 'auth'
  | 'permission'
  | 'not-found'
  | 'rate-limit'
  | 'network'
  | 'workspace'
  | 'query'
  | 'generic'

interface AzureErrorDisplayProps {
  /** Parsed error type */
  errorType: AzureErrorType
  /** Original error message */
  message: string
  /** Service name context */
  serviceName?: string
  /** Retry callback */
  onRetry?: () => void
  /** Switch to local logs callback */
  onViewLocal?: () => void
  /** Retry-After seconds (for rate limits) */
  retryAfter?: number
  /** Additional class names */
  className?: string
}
```

---

## Error Parsing Logic

```typescript
function parseAzureError(message: string, statusCode?: number): AzureErrorType {
  const lower = message.toLowerCase()
  
  // Check status code first
  if (statusCode === 401) return 'auth'
  if (statusCode === 403) return 'permission'
  if (statusCode === 404) return 'not-found'
  if (statusCode === 429) return 'rate-limit'
  
  // Check message patterns
  if (lower.includes('azd auth login') || 
      lower.includes('authentication') ||
      lower.includes('unauthorized')) return 'auth'
  
  if (lower.includes('authorization') ||
      lower.includes('permission') ||
      lower.includes('rbac') ||
      lower.includes('does not have')) return 'permission'
  
  if (lower.includes('not found') ||
      lower.includes('does not exist')) return 'not-found'
  
  if (lower.includes('rate limit') ||
      lower.includes('throttl') ||
      lower.includes('too many')) return 'rate-limit'
  
  if (lower.includes('network') ||
      lower.includes('timeout') ||
      lower.includes('connect') ||
      lower.includes('econnrefused')) return 'network'
  
  if (lower.includes('workspace') ||
      lower.includes('log analytics') ||
      lower.includes('diagnostic')) return 'workspace'
  
  if (lower.includes('query') ||
      lower.includes('syntax') ||
      lower.includes('kql')) return 'query'
  
  return 'generic'
}
```

---

## Integration Points

### HistoricalLogPanel

Replace generic error state with `<AzureErrorDisplay>`:

```tsx
{error && !isLoading && (
  <AzureErrorDisplay
    errorType={parseAzureError(error)}
    message={error}
    serviceName={serviceName}
    onRetry={handleRetry}
    onViewLocal={onClose}
  />
)}
```

### AzureConnectionStatus (extended)

Add error details popover:

```tsx
{status === 'error' && (
  <Popover>
    <AzureErrorDisplay
      errorType={parseAzureError(errorMessage)}
      message={errorMessage}
      onRetry={onRetry}
      compact
    />
  </Popover>
)}
```

### LogsPane (Azure mode)

Show inline error banner:

```tsx
{azureError && (
  <div className="border-t border-slate-700 p-3">
    <AzureErrorDisplay
      errorType={parseAzureError(azureError)}
      message={azureError}
      serviceName={serviceName}
      onRetry={retryAzureLogs}
      inline
    />
  </div>
)}
```

---

## Accessibility

- Error containers have `role="alert"` for screen reader announcement
- Action buttons have clear, descriptive labels
- Command copy uses `aria-live="polite"` for copy confirmation
- Focus management: first action button focused on error display
- Color is not the only indicator (icons + text + border)

---

## Animations

- Fade in on error display
- Countdown timer smooth animation
- Button hover/press states
- Copy confirmation checkmark transition

---

## Test Scenarios

1. Auth error → shows login command → copy works
2. Permission error → shows required roles → external link works
3. Not found → shows provision command → local logs button works
4. Rate limit → countdown displays → auto-retry triggers
5. Network error → retry button works
6. Query error → reset button restores default query
7. Screen reader announces error type and action
8. Keyboard navigation through all actions
