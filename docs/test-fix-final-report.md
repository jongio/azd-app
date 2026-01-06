# Final Test Fix Report - 104 Failures Remaining

## Status Summary
- **Total Tests**: 1460
- **Passing**: 1356
- **Failing**: 104
- **Files Affected**: 9

## ✅ Fixed (33 tests)
- **ModeToggle.test.tsx** - Changed `aria-checked` to `aria-pressed`

## ❌ Still Failing

### Critical Finding: Fake Timers + waitFor Deadlock

Many tests use `vi.useFakeTimers()` but then call `await waitFor()`. This creates a deadlock:
- `waitFor` tries to wait for condition using real time intervals
- But timers are frozen, so timeouts/intervals never fire
- Test hangs until timeout (10-15 seconds)

**Solution**: After advancing fake timers, call `vi.runAllTimers()` or `vi.runOnlyPendingTimers()` before `waitFor`

### 1. Diagnostic Settings Step (27 failures)

**Root Cause**: Fake timer deadlocks
**Failing Tests**: All tests involving polling, bicep expansion, filters

**Example Fix**:
```typescript
// BEFORE (deadlocks):
it('should poll for updates', async () => {
  vi.useFakeTimers()
  render(<Component />)
  vi.advanceTimersByTime(5000)
  await waitFor(() => {  // HANGS HERE
    expect(mockFetch).toHaveBeenCalledTimes(2)
  })
})

// AFTER (works):
it('should poll for updates', async () => {
  vi.useFakeTimers()
  render(<Component />)
  
  act(() => {
    vi.advanceTimersByTime(5000)
    vi.runAllTimers()  // KEY FIX
  })
  
  await waitFor(() => {
    expect(mockFetch).toHaveBeenCalledTimes(2)
  })
  
  vi.useRealTimers()  // CLEANUP
})
```

**Tests to fix**: 27 tests with fake timers

### 2. WorkspaceSetupStep (20 failures)

**Same issue as DiagnosticSettingsStep**

Already added `{ timeout: 15000 }` but still failing due to fake timer deadlocks.

**Tests to fix**: All polling, collapsible sections, copy functionality tests

### 3. useSharedLogStream (13 failures)

**Issues**:
1. WebSocket mock not triggering event listeners correctly
2. Fake timer issues with reconnection logic
3. `act()` warnings from state updates

**Recommended**: Rewrite WebSocket mock to use proper event dispatch

### 4. AzureErrorDisplay (15 failures)

**Not yet diagnosed** - need to run individual test to see error

### 5. TimeRangeSelector (12 failures)

**Root Cause**: Tests expect uncontrolled behavior but component is controlled

When clicking "Custom" button:
1. Component calls `onChange({ preset: 'custom', start, end })`
2. Parent must update `value` prop
3. Only then will custom inputs render

**Tests incorrectly assume**:
```typescript
await user.click(customButton)
// Inputs DON'T exist yet - parent hasn't updated value prop!
const startInput = screen.getByLabelText(/Start/i) // ❌ FAILS
```

**Solution**: Use controlled wrapper:
```typescript
function TestWrapper() {
  const [value, setValue] = React.useState({ preset: '15m' })
  return <TimeRangeSelector value={value} onChange={setValue} />
}

// Test:
const { rerender } = render(<TestWrapper />)
await user.click(customButton)
// Now inputs exist because wrapper updated state
const startInput = screen.getByLabelText(/Start/i) // ✅ WORKS
```

**Alternative**: Manually rerender with updated value:
```typescript
const onChange = vi.fn()
render(<TimeRangeSelector value={{ preset: '15m' }} onChange={onChange} />)
await user.click(customButton)

// Simulate parent updating value
render(<TimeRangeSelector 
  value={{ preset: 'custom', start: new Date(), end: new Date() }} 
  onChange={onChange} 
/>)

// Now inputs exist
const startInput = screen.getByLabelText(/Start/i) // ✅ WORKS
```

**Affected tests**: All 12 "Custom Range" tests

### 6. TableSelector (7 failures)

**Root Cause**: Multiple elements with same role/name

Component has multiple "Select All" buttons (one per category + one global).

**Tests incorrectly use**:
```typescript
const selectAll = screen.getByRole('button', { name: /Select All/i })
// ❌ Error: Found multiple elements
```

**Solution**:
```typescript
const selectAllButtons = screen.getAllByRole('button', { name: /Select All/i })
const globalSelectAll = selectAllButtons[selectAllButtons.length - 1] // Last one
```

**OR** use container queries:
```typescript
const header = screen.getByRole('banner') // or specific container
const selectAll = within(header).getByRole('button', { name: /Select All/i })
```

**Affected tests**: 7 tests querying "Select All", "Recommended", etc.

### 7. KqlQueryInput (4 failures)

**Issues**:
1. No `role="region"` in component
2. onChange called multiple times for typed text

**Fixes**:
```typescript
// Issue 1: Remove region query
-const section = screen.getByRole('region', { hidden: true })
+// Component uses div, not region

// Issue 2: Check last call instead of specific value
expect(onChange).toHaveBeenLastCalledWith('New query')
// or use onChange.mock.calls to inspect all calls
```

### 8. AzureSetupGuide (4 failures)

**Not yet diagnosed**

## Recommended Fix Order

1. **TableSelector** (7 failures) - Quick query fixes
2. **KqlQueryInput** (4 failures) - Simple assertion fixes  
3. **TimeRangeSelector** (12 failures) - Rewrite tests with controlled wrapper
4. **WorkspaceSetupStep** (20 failures) - Add `vi.runAllTimers()` after advancing
5. **DiagnosticSettingsStep** (27 failures) - Same fix as WorkspaceSetupStep
6. **AzureErrorDisplay** (15 failures) - Diagnose first
7. **AzureSetupGuide** (4 failures) - Diagnose first
8. **useSharedLogStream** (13 failures) - Complex WebSocket mock rewrite

## Total Effort Estimate

- Quick wins (TableSelector + KqlQueryInput): 30 minutes
- Medium effort (TimeRangeSelector): 1-2 hours
- High effort (WorkspaceSetupStep + DiagnosticSettingsStep): 2-3 hours
- Complex (useSharedLogStream): 2-4 hours
- Unknown (AzureErrorDisplay + AzureSetupGuide): 1-3 hours

**Total**: 6-12 hours to fix all 104 failing tests

## Next Steps

1. Start with TableSelector (easiest)
2. Move to KqlQueryInput
3. Create controlled wrapper for TimeRangeSelector
4. Fix fake timer deadlocks in Workspace/Diagnostic tests
5. Diagnose and fix remaining

Once these patterns are fixed, we should reach 100% pass rate (1460/1460 tests).
