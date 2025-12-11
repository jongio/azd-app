# Testing Auto-Load Azure Logs Feature

## Implementation Summary

Successfully implemented auto-load functionality with loading state for Azure logs in the dashboard UI.

## Changes Made

### 1. Added Azure Logs State Machine (`LogsPane.tsx`)

Created a comprehensive state machine to track Azure logs loading:

```typescript
interface AzureLogsState {
  status: 'idle' | 'loading' | 'showing' | 'error'
  logs: LogEntry[]
  lastUpdated: Date | null
  error: { message: string; details?: string } | null
}
```

**States:**
- `idle`: Initial state, no logs loaded yet
- `loading`: Fetching logs from Azure API (shows spinner)
- `showing`: Logs successfully loaded and displayed
- `error`: Failed to load logs (shows error panel with retry button)

### 2. Auto-Fetch on Mode Switch

Added a `useEffect` hook that automatically triggers when `logMode` changes to `'azure'`:

- **Immediate loading state**: Sets status to `'loading'` as soon as Azure mode is selected
- **Automatic API call**: Fetches from `/api/azure/logs?service={serviceName}&tail=500`
- **Error handling**: Captures both HTTP errors and network errors
- **State updates**: Transitions to `'showing'` on success or `'error'` on failure

### 3. Loading UI

Implemented a polished loading screen that appears immediately when switching to Azure mode:

```tsx
<div className="flex flex-col items-center justify-center py-16 gap-4">
  <Loader2 className="w-8 h-8 text-azure-500 animate-spin" />
  <div className="text-center">
    <p className="text-base font-semibold text-muted-foreground">
      Loading logs from Azure...
    </p>
    <p className="text-sm text-muted-foreground/70 mt-1">
      Fetching logs for {serviceName}
    </p>
  </div>
</div>
```

**Features:**
- Azure-branded spinner with animation
- Clear loading message
- Service name context
- Centered layout with proper spacing

### 4. Error State with Retry

Added a comprehensive error display with retry functionality:

```tsx
<div className="flex flex-col items-center justify-center py-12 gap-4">
  <div className="flex items-center gap-2 text-red-500">
    <AlertTriangle className="w-6 h-6" />
    <p className="text-base font-semibold">Failed to load Azure logs</p>
  </div>
  <div className="bg-red-500/10 border border-red-500/20 rounded-lg p-4 max-w-md">
    <p className="text-sm text-red-600 dark:text-red-400 font-medium mb-2">
      {azureLogsState.error?.message}
    </p>
    {azureLogsState.error?.details && (
      <p className="text-xs text-red-600/70 dark:text-red-400/70 font-mono">
        {azureLogsState.error.details}
      </p>
    )}
  </div>
  <Button variant="outline" size="sm" onClick={retryFetch}>
    <RotateCw className="w-4 h-4 mr-2" />
    Retry
  </Button>
</div>
```

**Features:**
- Clear error icon and heading
- Error message and technical details
- Retry button that re-triggers the fetch
- Proper error styling with theme support

## Testing Instructions

### Manual Testing Steps

#### 1. Start the Application

```powershell
cd c:\code\azd-app-2\cli\tests\projects\integration\azure-logs-test
azd app run
```

Or use the demo project:

```powershell
cd c:\code\azd-app-2\cli\demo
azd app run
```

#### 2. Open the Dashboard

The dashboard URL will be shown in the terminal output, e.g., `http://localhost:40942`

#### 3. Test Loading State

**Expected Behavior:**
1. Dashboard opens showing local logs by default
2. Click the Azure mode toggle (cloud icon) in the header
3. **Immediately** see the loading spinner with text "Loading logs from Azure..."
4. After 1-3 seconds, logs should appear OR error state should show

**What to Verify:**
- [ ] Loading spinner appears instantly (no delay)
- [ ] Loading message is clear and shows service name
- [ ] Spinner has Azure blue color
- [ ] No flicker or flash of empty state

#### 4. Test Success State

**If Azure is properly configured:**
1. Switch to Azure mode
2. Wait for loading to complete
3. Verify logs appear

**What to Verify:**
- [ ] Logs display after loading completes
- [ ] Log entries have correct timestamps
- [ ] Scrolling works properly
- [ ] Can filter and search logs
- [ ] Azure mode indicator shows "Viewing Azure Logs"

#### 5. Test Error State

**To trigger an error (Azure not configured):**
1. Ensure `logs.azure.enabled: false` in `azure.yaml` OR
2. Don't have Azure credentials configured
3. Switch to Azure mode

**What to Verify:**
- [ ] Error panel appears with clear message
- [ ] Error details are shown (if available)
- [ ] Retry button is visible and clickable
- [ ] Clicking retry re-triggers the fetch (shows loading again)
- [ ] Error styling is appropriate (red theme)

#### 6. Test Mode Switching

**Test rapid switching:**
1. Switch to Azure mode
2. Immediately switch back to local mode
3. Switch to Azure again

**What to Verify:**
- [ ] No duplicate API calls
- [ ] State clears properly between switches
- [ ] Loading state resets correctly
- [ ] No memory leaks or stuck spinners

#### 7. Test Multiple Services

**If project has multiple services:**
1. Switch to Azure mode
2. Verify each service pane shows loading independently
3. Verify each service can load/error independently

**What to Verify:**
- [ ] Loading states are per-service (not global)
- [ ] One service failing doesn't affect others
- [ ] Can retry individual services

### Edge Cases to Test

#### Edge Case 1: Network Timeout
- Disconnect network before switching to Azure mode
- Should show network error with retry button

#### Edge Case 2: Empty Logs
- Switch to Azure mode when no logs exist
- Should show "No logs to display" message (not error)

#### Edge Case 3: Partial Data
- If API returns partial data, verify it displays correctly
- Should not show error state for partial success

#### Edge Case 4: Quick Mode Toggle
- Toggle between local/azure rapidly
- Should cancel in-flight requests
- Should not show stale data

### Automated Testing (Future)

Consider adding E2E tests with Playwright:

```typescript
test('Azure logs auto-load with loading state', async ({ page }) => {
  await page.goto('http://localhost:40942')
  
  // Wait for dashboard to load
  await page.waitForSelector('[data-testid="mode-toggle"]')
  
  // Switch to Azure mode
  await page.click('[aria-label="View Azure logs"]')
  
  // Verify loading state appears
  await page.waitForSelector('text=Loading logs from Azure...', { timeout: 500 })
  
  // Wait for completion
  await page.waitForSelector('[data-testid="log-entry"]', { timeout: 5000 })
    .catch(() => {
      // Or error state
      return page.waitForSelector('text=Failed to load Azure logs')
    })
})
```

## Expected Results

### ✅ Success Indicators

1. **Instant Feedback**: Loading spinner appears within 100ms of clicking Azure mode
2. **Clear Communication**: User knows what's happening ("Loading logs from Azure...")
3. **Smooth Transition**: Loading → Showing state is seamless (no flash/flicker)
4. **Error Recovery**: If fetch fails, user can retry without page reload
5. **No Manual Action**: No "Load Logs" button needed - it's automatic

### ❌ Failure Indicators

1. Delay before loading spinner appears (>500ms)
2. Empty/blank screen during fetch
3. Error state with no retry option
4. Stale data from previous mode
5. Multiple loading spinners or duplicate fetches
6. Loading state that never resolves

## Troubleshooting

### Loading Never Completes

**Check:**
- Is the backend running?
- Is Azure configuration valid?
- Check browser console for errors
- Verify `/api/azure/logs` endpoint responds

**Fix:**
- Check `azure.yaml` has `logs.azure.enabled: true`
- Verify Azure credentials are configured
- Check backend logs for API errors

### Error State Shows Immediately

**Check:**
- Azure credentials: `azd auth login`
- Project provisioned: `azd provision`
- Environment variables set correctly

### Loading Spinner Stuck

**Check:**
- Browser console for JavaScript errors
- Network tab for failed API calls
- Backend logs for request handling

**Fix:**
- Hard refresh browser (Ctrl+Shift+R)
- Restart the app
- Check for infinite loops in useEffect

## Performance Considerations

### Current Implementation

- **First Load**: 1-3 seconds (depends on Azure API latency)
- **Subsequent Switches**: Same as first load (no caching yet)
- **Memory Usage**: Minimal (state machine is lightweight)

### Future Optimizations

1. **Caching**: Cache Azure logs for 30-60 seconds to avoid re-fetching
2. **Debouncing**: Debounce rapid mode switches
3. **Prefetching**: Prefetch Azure logs in background when Azure is enabled
4. **Pagination**: Lazy-load older logs on scroll

## Related Files

- `cli/dashboard/src/components/LogsPane.tsx` - Main implementation
- `cli/dashboard/src/components/ConsoleView.tsx` - Mode switching logic
- `cli/dashboard/src/components/ModeToggle.tsx` - UI toggle component
- `cli/src/internal/service/azure_log_buffer.go` - Backend API handler

## Next Steps

1. ✅ Implement auto-load with loading state (COMPLETE)
2. [ ] Add caching to avoid redundant fetches
3. [ ] Add E2E tests for loading states
4. [ ] Add telemetry to track load times
5. [ ] Consider prefetching strategy
6. [ ] Add unit tests for state machine

## Notes

- The implementation follows React best practices with proper cleanup
- State machine makes the component behavior predictable and testable
- Error handling is comprehensive with user-friendly messages
- The UI matches the existing design system (Azure blue theme)
- Loading state is instant - no perceived delay when switching modes
