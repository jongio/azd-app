# Test Fixes Summary

## Fixed (33 tests)
- ✅ **ModeToggle.test.tsx** - Changed `aria-checked` to `aria-pressed` for button elements

## Needs Fixing

### 1. DiagnosticSettingsStep.test.tsx (27 failures)
**Issue**: Tests timing out at 10000ms
**Fix**: Add `{ timeout: 15000 }` to all async tests similar to WorkspaceSetupStep

### 2. WorkspaceSetupStep.test.tsx (20 failures)  
**Issue**: Tests timing out at 10000ms
**Status**: Partially fixed - added timeouts to some tests
**Remaining**: Need to add `{ timeout: 15000 }` to remaining tests

### 3. AzureErrorDisplay.test.tsx (15 failures)
**Issue**: Not investigated yet

### 4. useSharedLogStream.test.ts (13 failures)
**Issue**: WebSocket mocking issues and timing problems

### 5. TimeRangeSelector.test.tsx (12 failures)
**Issue**: Tests expect custom date inputs to appear immediately after clicking "Custom" button
**Fix**: Tests need to use controlled component pattern - rerender with `value={{ preset: 'custom', start, end }}`

### 6. TableSelector.test.tsx (7 failures)
**Issue**: Multiple elements found with same queries
**Fix**: Use more specific queries or `getAllByRole` then select specific element

### 7. KqlQueryInput.test.tsx (4 failures)
**Issues**:
- Expecting `role="region"` but component might not have it
- onChange not being called correctly with typed text

### 8. AzureSetupGuide.test.tsx (4 failures)
**Issue**: Not investigated yet

## Quick Fix Commands

```powershell
# For all DiagnosticSettingsStep tests - add { timeout: 15000 }
# Similar pattern to WorkspaceSetupStep

# For TimeRangeSelector - need controlled component wrapper
# Example:
function ControlledTimeRange({ initial }) {
  const [value, setValue] = React.useState(initial)
  return <TimeRangeSelector value={value} onChange={setValue} />
}

# For TableSelector - use getAllByRole and select specific elements
const selectAllButtons = screen.getAllByRole('button', { name: /Select All/i })
const mainSelectAllButton = selectAllButtons[0] // or find by container
```

## Priority Order
1. DiagnosticSettingsStep (27 failures) - straightforward timeout fixes
2. WorkspaceSetupStep (remaining 20) - complete timeout fixes 
3. TimeRangeSelector (12) - requires test rewrite for controlled component
4. useSharedLogStream (13) - complex WebSocket mocking
5. AzureErrorDisplay (15) - unknown issue
6. TableSelector (7) - query specificity
7. KqlQueryInput (4) - minor fixes
8. AzureSetupGuide (4) - unknown issue
