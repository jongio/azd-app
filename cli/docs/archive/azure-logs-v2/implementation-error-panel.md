# Azure Logs Error Panel Implementation

**Date**: December 10, 2025  
**Status**: ✅ Completed

## Overview

Enhanced the Azure logs error handling in the dashboard UI to provide actionable guidance with copyable commands, retry functionality, and documentation links.

## Changes Made

### 1. Enhanced `AzureErrorDisplay.tsx`

**Location**: `cli/dashboard/src/components/AzureErrorDisplay.tsx`

#### Key Features Added:

1. **ErrorInfo Support**
   - Added `errorInfo?: ErrorInfo` prop to accept structured errors from API
   - Automatically maps error codes to display types
   - Prioritizes ErrorInfo fields over static config

2. **Copyable Command Box**
   - `CommandCopy` component with one-click copy functionality
   - Visual feedback with checkmark animation
   - Accessible with ARIA labels
   - Supports commands from both ErrorInfo and static config

3. **Action Message Display**
   - Shows `ErrorInfo.action` field with prominent styling
   - Provides context-specific guidance

4. **Retry Button**
   - Prominent cyan primary button for retry action
   - Integrates with existing `onRetry` callback
   - Displays appropriate action text from config

5. **Documentation Links**
   - Opens in new tab with security attributes
   - Prioritizes `ErrorInfo.docsUrl` over static config
   - External link icon for clarity

6. **Error Type Mapping**
   - Helper function `mapErrorCodeToType()` converts API error codes to display types
   - Supports: AUTH, PERMISSION, NOT_FOUND, RATE_LIMIT, NETWORK, WORKSPACE, QUERY, GENERIC

#### Props Interface:

```typescript
export interface AzureErrorDisplayProps {
  errorType?: AzureErrorType
  message?: string
  onRetry?: () => void
  onViewLocal?: () => void
  onResetQuery?: () => void
  retryAfter?: number
  compact?: boolean
  className?: string
  errorInfo?: ErrorInfo  // NEW: Structured error from API
}
```

#### UI Structure (Full Display):

```
┌─────────────────────────────────────┐
│          [Error Icon]               │
│                                     │
│        Error Title                  │
│    Error Message                    │
│    Action Message (if present)      │
│                                     │
│  ┌──────────────────────────────┐  │
│  │ $ command-to-run      [Copy] │  │
│  └──────────────────────────────┘  │
│                                     │
│  [Docs Link] [Secondary] [Retry]   │
└─────────────────────────────────────┘
```

### 2. Updated `LogsPane.tsx`

**Location**: `cli/dashboard/src/components/LogsPane.tsx`

#### Changes:

1. **Simplified Error Display**
   - Removed local `mapErrorCodeToType` function (moved to AzureErrorDisplay)
   - Pass `errorInfo` directly to AzureErrorDisplay
   - Removed unused `serviceName` prop

2. **Integration with Azure Logs State**
   - Displays error panel when `azureLogsState.status === 'error'`
   - Passes `azureLogsState.error` (ErrorInfo) to component
   - Wires up retry button to `handleRetryAzureLogs`

#### Code Example:

```tsx
{logMode === 'azure' && azureLogsState.status === 'error' && azureLogsState.error ? (
  <AzureErrorDisplay
    errorInfo={azureLogsState.error}
    onRetry={handleRetryAzureLogs}
  />
) : /* ... other states ... */}
```

### 3. Type Definitions

**Location**: `cli/dashboard/src/types.ts`

The `ErrorInfo` interface was already defined:

```typescript
export interface ErrorInfo {
  message: string   // Human-readable error message
  code: string      // Error code: "AUTH_REQUIRED", "NOT_DEPLOYED", etc.
  action: string    // What the user should do
  command?: string  // CLI command to run (optional)
  docsUrl?: string  // Documentation URL
}
```

## Error Types Supported

| Error Code | Display Type | Icon | Primary Action | Features |
|-----------|--------------|------|----------------|----------|
| AUTH_REQUIRED | auth | KeyRound | Retry | Command copy |
| PERMISSION | permission | ShieldOff | Retry | Permission details, docs link |
| NOT_DEPLOYED, NOT_FOUND | not-found | Search | Retry | Command copy, local fallback |
| RATE_LIMIT | rate-limit | Clock | Retry | Countdown timer |
| NETWORK, TIMEOUT | network | Wifi | Retry | Local fallback |
| WORKSPACE | workspace | Database | - | Config snippet, docs link |
| QUERY | query | AlertTriangle | Retry | Query reset option |
| UNKNOWN | generic | XCircle | Retry | Report issue link |

## User Experience Flow

### Example: Authentication Error

1. User switches to Azure logs mode
2. API returns error:
   ```json
   {
     "status": "error",
     "error": {
       "message": "Not authenticated with Azure",
       "code": "AUTH_REQUIRED",
       "action": "Sign in to Azure to view cloud logs",
       "command": "azd auth login",
       "docsUrl": "https://aka.ms/azd/auth"
     }
   }
   ```

3. Dashboard displays:
   - 🔑 Authentication Required
   - "Not authenticated with Azure"
   - "Sign in to Azure to view cloud logs"
   - Copyable command box: `azd auth login` with copy button
   - [View Setup Guide] [Retry Now] buttons

4. User clicks Copy → Command copied to clipboard with visual feedback
5. User runs command in terminal
6. User clicks Retry Now → Logs load successfully

### Example: Permission Error

1. User authenticated but lacks permissions
2. API returns PERMISSION error
3. Dashboard shows:
   - 🛡️ Permission Denied
   - Required permissions list:
     - Log Analytics Reader on workspace
     - Reader on resource group
   - [View Azure RBAC Docs] [Retry] buttons

## Testing

### Manual Testing Steps:

1. **Test AUTH_REQUIRED Error**:
   ```powershell
   # Ensure not logged in
   azd auth logout
   # Start app and switch to Azure mode
   azd app run
   # Expected: Auth error with "azd auth login" command
   ```

2. **Test Copy Functionality**:
   - Click copy button on command
   - Verify checkmark appears
   - Paste in terminal to verify clipboard

3. **Test Retry Button**:
   - Click Retry Now
   - Verify loading state appears
   - Verify error clears or new error shows

4. **Test Docs Link**:
   - Click View Setup Guide
   - Verify opens in new tab
   - Verify URL matches docsUrl field

5. **Test Different Error Types**:
   - Trigger NOT_DEPLOYED (no resources)
   - Trigger NETWORK (disconnect internet)
   - Verify appropriate icons and messages

## Visual Design

- **Error Colors**: Matches error type (amber for auth/warning, red for critical)
- **Icons**: Contextual Lucide icons for each error type
- **Copy Button**: Hover state with transition
- **Retry Button**: Prominent cyan with white text
- **Spacing**: Consistent padding and gaps
- **Dark Mode**: Full support with theme-aware colors

## Accessibility

- ✅ ARIA labels on all interactive elements
- ✅ Screen reader announcements for copy feedback
- ✅ Keyboard navigation support
- ✅ Focus ring on all focusable elements
- ✅ Semantic HTML with role="alert"
- ✅ Sufficient color contrast (WCAG AA)

## Performance

- ✅ No unnecessary re-renders
- ✅ Memoized callbacks where needed
- ✅ Efficient clipboard API usage
- ✅ Fast error type mapping

## Future Enhancements

1. **Telemetry**: Track which errors occur most frequently
2. **Auto-retry**: Countdown timer for transient errors
3. **Quick Actions**: In-dashboard auth flow
4. **Error History**: Show previous errors in a timeline
5. **Smart Suggestions**: ML-based fix recommendations

## Files Modified

1. `cli/dashboard/src/components/AzureErrorDisplay.tsx` - Enhanced error panel
2. `cli/dashboard/src/components/LogsPane.tsx` - Integrated error display
3. `cli/dashboard/src/types.ts` - Already had ErrorInfo type (no changes needed)

## Build Verification

```bash
cd c:\code\azd-app-2\cli
mage build
# ✅ SUCCESS: Build completed successfully!
```

## Conclusion

The enhanced Azure error panel provides users with:

- **Clear Error Messages**: Understand what went wrong
- **Actionable Guidance**: Know exactly what to do next
- **One-Click Solutions**: Copy commands instantly
- **Quick Recovery**: Retry with a single click
- **Additional Help**: Access documentation when needed

This implementation significantly improves the developer experience when encountering Azure log errors, reducing friction and time to resolution.

---

**Implementation Complete** ✅
