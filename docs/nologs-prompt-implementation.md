# NoLogsPrompt Component Implementation Report

**Date**: December 29, 2025  
**Developer Agent**: Implementation Complete  
**Feature**: azure-logs-diagnostics  

## Overview

Implemented the NoLogsPrompt component to display when Azure services have zero logs in the selected time range. The component provides clear explanatory text and a link to open the diagnostics modal for troubleshooting.

## Implementation Details

### 1. Files Created

#### **NoLogsPrompt.tsx** 
Location: `cli/dashboard/src/components/NoLogsPrompt.tsx`

**Purpose**: Standalone component shown in log panes when Azure service has 0 logs

**Features**:
- Warning icon (AlertTriangle from lucide-react)
- Service name display
- Clear explanation of possible reasons:
  - Diagnostic settings not configured
  - Delay in log ingestion (2-5 minutes)
  - Service hasn't generated activity yet
- "View Diagnostic Details" button with Wrench icon
- Accessible with `role="status"` and `aria-label`
- Follows existing dashboard styling patterns

**Props**:
- `serviceName: string` - Name of the service with no logs
- `onOpenDiagnostics?: () => void` - Optional callback to open diagnostics modal

#### **NoLogsPrompt.test.tsx**
Location: `cli/dashboard/src/components/NoLogsPrompt.test.tsx`

**Test Coverage**: 7 tests, all passing
- Renders service name and warning message
- Displays warning icon
- Conditionally renders diagnostic button
- Calls onOpenDiagnostics when button clicked
- Has accessible status role
- Mentions all possible reasons for no logs

### 2. Files Modified

#### **LogsPaneEmptyState.tsx**
- Added import for NoLogsPrompt component
- Added `onOpenDiagnostics` prop to interface
- Updated Azure logs empty state logic:
  - Shows NoLogsPrompt when `!hasLogs && serviceName` (0 logs = potential issue)
  - Shows time range suggestion when logs exist but not in current range (expected behavior)

#### **LogsPaneContent.tsx**
- Added `onOpenDiagnostics` prop to interface
- Passed `onOpenDiagnostics` to LogsPaneEmptyState component

#### **LogsPane.tsx**
- Added `onOpenDiagnostics` prop to interface
- Passed `onOpenDiagnostics` through to LogsPaneContent

#### **ConsoleView.tsx**
- Connected `onOpenDiagnostics={() => setShowDiagnostics(true)}` to each LogsPane
- Integrated with existing DiagnosticsModal state management

### 3. Component Flow

```
ConsoleView
  └─> LogsPane (onOpenDiagnostics={() => setShowDiagnostics(true)})
       └─> LogsPaneContent (onOpenDiagnostics)
            └─> LogsPaneEmptyState (onOpenDiagnostics)
                 └─> NoLogsPrompt (onOpenDiagnostics)
                      └─> [User clicks button]
                           └─> DiagnosticsModal opens
```

## Integration Points

### Where NoLogsPrompt Appears

1. **Azure Logs Mode Only**: Only shows when `logMode === 'azure'`
2. **Zero Logs Condition**: Only shows when service has `!hasLogs` (no logs fetched)
3. **Service Name Required**: Only shows when `serviceName` is provided
4. **Time Range Agnostic**: Shows regardless of selected time range preset

### Styling

- Matches existing empty state patterns in dashboard
- Uses Tailwind CSS utility classes
- Follows dark mode support conventions
- Cyan button for primary action (matches diagnostic theme)
- Responsive and accessible design

## Build Status

✅ **Component Tests**: 7/7 passing  
✅ **TypeScript Compilation**: No errors in modified files  
⚠️ **Full Build**: Pre-existing errors in e2e/health-tooltip.spec.ts (unrelated to this work)

Note: The full build failure is due to pre-existing TypeScript errors in e2e test files that reference missing test scenario properties. These errors existed before this implementation and are not caused by the NoLogsPrompt component.

## Testing

### Manual Testing Checklist

To test the component:

1. Start `azd app run` in a test project with Azure logs configured
2. Switch to Azure logs mode in dashboard
3. View a service that has no logs in selected time range
4. Verify NoLogsPrompt appears with:
   - Service name
   - Warning icon
   - Explanatory text
   - "View Diagnostic Details" button
5. Click button → DiagnosticsModal should open
6. Close modal → NoLogsPrompt should still be visible

### Automated Tests

Run: `cd cli/dashboard; npm test -- NoLogsPrompt --run`

Expected output:
```
✓ should render service name and warning message
✓ should display warning icon  
✓ should render diagnostic button when callback provided
✓ should not render diagnostic button when callback not provided
✓ should call onOpenDiagnostics when button clicked
✓ should have accessible status role
✓ should mention all possible reasons for no logs
```

## Accessibility

- Uses semantic HTML with `role="status"` for screen readers
- Proper `aria-label` on container: "No logs available for {serviceName}"
- `aria-hidden="true"` on decorative icons
- Descriptive button `aria-label`: "View diagnostic details to troubleshoot"
- Focus management with visible focus rings
- Keyboard accessible (button can be activated with Enter/Space)

## Next Steps for Manager

✅ **COMPLETE**: Component created and integrated  
✅ **COMPLETE**: Tests written and passing  
✅ **COMPLETE**: TypeScript compilation verified  
📋 **TODO**: Manual testing in running dashboard  
📋 **TODO**: Screenshot/visual verification  
📋 **TODO**: Merge into feature branch  

## Notes

- Component is simple and self-contained
- No external dependencies beyond lucide-react icons (already in use)
- Follows existing component patterns (see HistoricalLogPanel empty state)
- Can be easily extended with additional guidance or actions if needed
- DiagnosticsModal integration already exists, just connected the callback

## Related Files

- `cli/dashboard/src/components/DiagnosticsModal.tsx` - Modal that opens when button clicked
- `cli/dashboard/src/components/HistoricalLogPanel.tsx` - Similar empty state pattern for reference
- `cli/dashboard/src/components/LogsPaneEmptyState.tsx` - Parent component that conditionally shows NoLogsPrompt

---

**Implementation Status**: ✅ COMPLETE  
**Ready for Review**: YES  
**Breaking Changes**: None  
**Backward Compatible**: Yes
