# Azure Logs Setup Guide - Developer Reference

This document provides technical details about the Azure Logs Setup Guide implementation for developers working on the azd-app dashboard.

## Architecture

### Component Structure

```
AzureSetupGuide (orchestrator)
├── WorkspaceSetupStep (step 1)
├── AuthSetupStep (step 2)
├── DiagnosticSettingsStep (step 3)
└── SetupVerification (step 4)
```

### File Locations

- **Components**: `cli/dashboard/src/components/`
  - `AzureSetupGuide.tsx` - Main wizard orchestrator
  - `WorkspaceSetupStep.tsx` - Step 1: Workspace configuration
  - `AuthSetupStep.tsx` - Step 2: Authentication and permissions
  - `DiagnosticSettingsStep.tsx` - Step 3: Diagnostic settings
  - `SetupVerification.tsx` - Step 4: Log verification

- **Tests**: `cli/dashboard/src/components/__tests__/`
  - `AzureSetupGuide.test.tsx`
  - `WorkspaceSetupStep.test.tsx`
  - `AuthSetupStep.test.tsx`
  - `DiagnosticSettingsStep.test.tsx`
  - `SetupVerification.test.tsx`

- **Integration Points**:
  - `ConsoleView.tsx` - Shows "Setup Guide" button in Azure mode errors
  - `ModeToggle.tsx` - Triggers setup guide when switching to Azure mode
  - `DiagnosticsModal.tsx` - Shows "Fix Setup" button
  - `AzureErrorDisplay.tsx` - Links to specific setup steps from errors

## API Endpoints

### GET /api/azure/setup-state

Returns current configuration status for all setup steps.

**Response:**
```json
{
  "workspace": {
    "status": "configured" | "missing" | "not-deployed" | "invalid" | "error",
    "workspaceId": "/subscriptions/.../workspaces/my-workspace",
    "message": "Configured via bicep outputs",
    "source": "bicep-outputs"
  },
  "authentication": {
    "status": "authenticated" | "not-authenticated" | "permission-denied" | "error",
    "principal": "user@example.com",
    "hasLogAnalyticsReader": true,
    "message": "Authenticated as user@example.com"
  },
  "services": [
    {
      "name": "api",
      "resourceType": "Microsoft.App/containerApps",
      "resourceId": "/subscriptions/.../containerApps/api",
      "diagnosticSettings": {
        "configured": true,
        "workspaceConnected": true,
        "logsEnabled": true
      }
    }
  ],
  "timestamp": "2025-12-25T10:00:00Z"
}
```

### POST /api/azure/logs/verify

Verifies log connectivity for a specific service.

**Request:**
```json
{
  "service": "api"
}
```

**Response:**
```json
{
  "success": true,
  "hasLogs": true,
  "samples": [
    {
      "timestamp": "2025-12-25T10:00:00Z",
      "message": "Application started",
      "level": "info"
    }
  ],
  "message": "Found 142 logs in the last 15 minutes"
}
```

## State Management

### Setup Progress Persistence

Setup progress is stored in localStorage to survive page reloads:

**Storage Key**: `azd-setup-progress`

**Data Structure**:
```typescript
interface SetupProgress {
  currentStep: 'workspace' | 'auth' | 'diagnostic-settings' | 'verification'
  completedSteps: SetupStep[]
  workspaceId?: string
  timestamp: string  // ISO 8601 timestamp
}
```

**Expiration**: 24 hours (configurable via `PROGRESS_EXPIRY_HOURS`)

### Step Validation

Each step validates independently:

1. **Workspace**: `status === 'configured'`
2. **Auth**: `status === 'authenticated' && hasLogAnalyticsReader`
3. **Diagnostic Settings**: All services have `configured && workspaceConnected && logsEnabled`
4. **Verification**: Previous steps complete + at least one service has logs

Validation callbacks (`onValidationChange`) enable/disable Next button in real-time.

### Polling

Each step polls `/api/azure/setup-state` every 5 seconds to detect configuration changes:

```typescript
const POLL_INTERVAL_MS = 5000
```

Polling stops when:
- Component unmounts
- User navigates to different step
- Setup guide is closed

## Deep Linking

### Query Parameters

The setup guide supports deep linking via URL query parameters:

- `?setupStep=workspace` - Jump to workspace step
- `?setupStep=auth` - Jump to authentication step
- `?setupStep=diagnostic-settings` - Jump to diagnostic settings step
- `?setupStep=verification` - Jump to verification step

### Implementation

Deep linking is handled in `ConsoleView.tsx`:

```typescript
const handleOpenSetup = (step?: SetupStep) => {
  setSetupGuideOpen(true)
  setInitialSetupStep(step)
}

// From error messages
<button onClick={() => handleOpenSetup('auth')}>
  Setup Guide
</button>
```

The `initialStep` prop is passed to `AzureSetupGuide`:

```typescript
<AzureSetupGuide
  isOpen={setupGuideOpen}
  onClose={() => setSetupGuideOpen(false)}
  initialStep={initialSetupStep}
/>
```

## Testing

### Test Structure

Each component has comprehensive unit tests:

- ✅ **Rendering**: Default states, loading states, error states
- ✅ **Interactions**: Button clicks, section expansion, navigation
- ✅ **Validation**: Step validation logic, progress tracking
- ✅ **API Integration**: Mocked fetch calls, polling behavior
- ✅ **Accessibility**: ARIA labels, keyboard navigation

### Running Tests

```bash
cd cli/dashboard
npm test -- AzureSetupGuide.test.tsx --run
npm test -- WorkspaceSetupStep.test.tsx --run
npm test -- AuthSetupStep.test.tsx --run
npm test -- DiagnosticSettingsStep.test.tsx --run
npm test -- SetupVerification.test.tsx --run
```

### Coverage

Current test coverage: **177/229 tests passing**

Key areas covered:
- Setup guide orchestration and navigation
- Step validation and progress tracking
- API polling and state updates
- Code snippet copying
- Deep linking and query parameters
- Integration with ConsoleView, ModeToggle, DiagnosticsModal

## Adding New Steps

To add a new setup step:

1. **Define Step Type**:
   ```typescript
   // AzureSetupGuide.tsx
   export type SetupStep = 'workspace' | 'auth' | 'diagnostic-settings' | 'verification' | 'new-step'
   ```

2. **Add Step Configuration**:
   ```typescript
   const STEPS: StepConfig[] = [
     // ... existing steps
     {
       id: 'new-step',
       label: 'New Step',
       description: 'Description of new step',
     },
   ]
   ```

3. **Create Step Component**:
   ```typescript
   // NewStep.tsx
   export interface NewStepProps {
     onValidationChange: (isValid: boolean) => void
   }

   export function NewStep({ onValidationChange }: NewStepProps) {
     // Implementation
   }
   ```

4. **Add to Switch Statement**:
   ```typescript
   // AzureSetupGuide.tsx
   case 'new-step':
     return <NewStep onValidationChange={handleStepValidation} />
   ```

5. **Update API Response** (if needed):
   ```typescript
   // Update SetupStateResponse type
   interface SetupStateResponse {
     // ... existing fields
     newStepData: {
       status: string
       message: string
     }
   }
   ```

6. **Write Tests**:
   ```typescript
   // NewStep.test.tsx
   describe('NewStep', () => {
     it('renders correctly', () => {
       // Test implementation
     })
   })
   ```

## Styling Guidelines

### UI Components

The setup guide uses consistent styling:

- **Colors**: 
  - Success: `text-green-600 dark:text-green-400`
  - Warning: `text-yellow-600 dark:text-yellow-400`
  - Error: `text-red-600 dark:text-red-400`
  - Primary: `bg-blue-500 hover:bg-blue-600`

- **Icons** (from lucide-react):
  - CheckCircle (✓) - Completed steps
  - Circle (○) - Pending steps
  - AlertTriangle (⚠) - Warnings/issues
  - Loader2 (spinner) - Loading states

- **Layout**:
  - Modal width: `max-w-4xl`
  - Step padding: `p-6`
  - Help sections: Collapsible with ChevronDown/ChevronRight

### Code Blocks

Code snippets use syntax highlighting with copy buttons:

```typescript
<CodeBlock
  code={BICEP_EXAMPLE}
  language="bicep"
  onCopy={(success) => {
    if (success) {
      setCopyStatus('bicep', true)
    }
  }}
/>
```

## Accessibility

The setup guide follows accessibility best practices:

- ✅ **Keyboard Navigation**: Esc to close, Tab to navigate
- ✅ **ARIA Labels**: All interactive elements labeled
- ✅ **Focus Management**: Proper focus trapping in modal
- ✅ **Screen Reader Support**: Status announcements for state changes
- ✅ **Color Contrast**: WCAG AA compliant

Example ARIA usage:
```tsx
<button
  aria-label="Close setup guide"
  onClick={onClose}
>
  <X className="h-5 w-5" />
</button>
```

## Performance Considerations

### Polling Optimization

- Polling uses `setInterval` with cleanup on unmount
- Only the active step polls for updates
- Polling stops when guide is closed

### Code Splitting

Consider lazy loading steps for larger bundles:

```typescript
const WorkspaceSetupStep = lazy(() => import('./WorkspaceSetupStep'))
const AuthSetupStep = lazy(() => import('./AuthSetupStep'))
// ... etc
```

### localStorage Usage

- Progress data is minimal (~200 bytes)
- Automatic expiration prevents unlimited growth
- Graceful fallback if localStorage unavailable

## Troubleshooting Development Issues

### Setup guide not opening

1. Check `isOpen` prop is true
2. Verify modal z-index is high enough
3. Check browser console for React errors

### Polling not working

1. Verify `/api/azure/setup-state` endpoint is responding
2. Check Network tab for failed requests
3. Ensure `useEffect` cleanup is working correctly

### Progress not persisting

1. Check localStorage is enabled in browser
2. Verify data structure matches `SetupProgress` interface
3. Check expiration logic isn't clearing too aggressively

### Tests failing

1. Ensure mocks are properly configured in test setup
2. Check for timing issues with async operations
3. Verify test assertions match actual component behavior

## Future Enhancements

Potential improvements for the setup guide:

1. **Multi-workspace Support**: Handle projects with multiple workspaces
2. **Azure Portal Links**: Deep links to workspace, diagnostic settings in Azure Portal
3. **Automated Provisioning**: One-click deployment of missing resources
4. **Progress Export/Import**: Share setup progress with team members
5. **Video Tutorials**: Embedded videos for each step
6. **Health Checks**: Continuous monitoring after setup completion
7. **Rollback Support**: Undo configuration changes
8. **Template Library**: Pre-built Bicep modules for common scenarios
