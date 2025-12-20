# Log Streaming Simplification - Tasks

<!-- NEXT: -->

## TODO

(All tasks complete)

---

## Done

### 1. Delete useAzurePollingRefreshTrigger Hook ✅
**Completed:** Removed frontend polling trigger hook.

**Files:**
- ✅ Deleted: `cli/dashboard/src/hooks/useAzurePollingRefreshTrigger.ts`
- ✅ Deleted: `cli/dashboard/src/hooks/useAzurePollingRefreshTrigger.test.tsx`
- ✅ Removed import from `cli/dashboard/src/components/LogsPane.tsx`

---

### 2. Simplify useLogsStream Hook ✅
**Completed:** Removed complex polling logic and refreshTrigger coordination.

**File:** `cli/dashboard/src/hooks/useLogsStream.ts`

**Changes:**
- ✅ Removed `refreshTrigger` from `UseLogsStreamParams` interface
- ✅ Removed `refreshTrigger` from params destructuring
- ✅ Removed `shouldPollViaHttp` logic (lines 237-270)
- ✅ Simplified effect to: initial fetch + WebSocket streaming
- ✅ Removed `refreshTrigger` from effect dependencies

---

### 3. Update LogsPane Component ✅
**Completed:** Removed syncInterval and refreshTrigger props and related state.

**File:** `cli/dashboard/src/components/LogsPane.tsx`

**Changes:**
- ✅ Removed `syncInterval` from LogsPaneProps interface
- ✅ Removed `syncInterval` prop from component
- ✅ Removed `useAzurePollingRefreshTrigger` import and usage
- ✅ Removed `secondsUntilRefresh` and `refreshTrigger` state
- ✅ Updated `useLogsStream` call to not pass `refreshTrigger`
- ✅ Updated `LogsPaneRefreshFooter` to not pass `secondsUntilRefresh`

---

### 4. Update ConsoleView Component ✅
**Completed:** Removed syncInterval state and related logic.

**File:** `cli/dashboard/src/components/ConsoleView.tsx`

**Changes:**
- ✅ Removed `syncInterval` state from ConsoleView
- ✅ Updated `useConsoleSyncSettings` to remove syncInterval management
- ✅ Removed `syncInterval` from all LogsPane calls
- ✅ Removed syncInterval from ConsoleToolbar props

---

### 5. Simplify LogsPaneRefreshFooter ✅
**Completed:** Replaced countdown timer with simple paused indicator.

**File:** `cli/dashboard/src/components/LogsPaneRefreshFooter.tsx`

**Changes:**
- ✅ Removed `secondsUntilRefresh` prop
- ✅ Removed `syncInterval` prop
- ✅ Replaced countdown UI with paused indicator: "Paused - log streaming stopped"
- ✅ Shows when isPaused && logMode === 'azure'

---

### 6. Update ConsoleToolbar Component ✅
**Completed:** Removed refresh interval selector UI.

**File:** `cli/dashboard/src/components/ConsoleToolbar.tsx`

**Changes:**
- ✅ Removed `syncInterval` prop from ConsoleToolbarProps interface
- ✅ Removed `onSyncIntervalChange` prop from interface
- ✅ Removed refresh interval dropdown UI (5s/10s/30s/1m/5m options)
- ✅ Removed unused `azureRealtime`/`onAzureRealtimeChange` props

---

### 7. Update useConsoleSyncSettings Hook ✅
**Completed:** Streamlined to only manage azureRealtime toggle.

**File:** `cli/dashboard/src/hooks/useConsoleSyncSettings.ts`

**Changes:**
- ✅ Removed `syncInterval` management
- ✅ Deleted `clampSyncInterval`, `getSavedSyncInterval`, `setSavedSyncInterval` functions
- ✅ Removed UI_CONSTANTS import
- ✅ Kept only `azureRealtime` setting

---

### 8. Update useLogsStream Tests ✅
**Completed:** Fixed tests that referenced removed polling logic.

**Files:**
- ✅ `cli/dashboard/src/hooks/useLogsStream.test.ts` - Removed refreshTrigger references
- ✅ `cli/dashboard/src/hooks/useLogsStream.polling.test.ts` - Deleted (polling-specific)
- ✅ `cli/dashboard/src/hooks/useLogsStream.flood.test.ts` - Removed refreshTrigger test, fixed unused variable

**Changes:**
- ✅ Removed refreshTrigger from test parameters
- ✅ Deleted polling.test.ts file
- ✅ Removed "refreshTrigger changes in local mode" test
- ✅ Fixed unused 'hooks' variable

---

### 9. Update Component Tests ✅
**Completed:** Fixed component tests that passed removed props.

**Files:**
- ✅ `cli/dashboard/src/components/logspane.test.tsx` - Removed polling interval test
- ✅ `cli/dashboard/src/components/LogsPane.footer.test.tsx` - Updated to test paused indicator
- ✅ `cli/dashboard/src/components/consoleview.test.tsx` - Removed sync interval clamping tests

**Changes:**
- ✅ Removed syncInterval from LogsPane test props
- ✅ Removed refreshTrigger from mock data
- ✅ Updated assertions from "Next refresh in" to "Paused" indicator
- ✅ All tests passing (33 test files, 756 tests total)

---

### 10. Verification ✅
**Completed:** All compilation and test verification complete.

**Results:**
- ✅ TypeScript compilation clean (0 errors)
- ✅ All tests passing (Test Files: 33 passed | Tests: 756 passed, 17 skipped)
- ✅ ~200 lines of code removed
- ✅ Architecture simplified: initial fetch + WebSocket only
- ✅ Backend abstracts polling (Log Analytics) vs streaming (Container Apps)

---

## Blocked

(None)

---

## Summary

**Code Reduction:** ~200 lines removed
**Tests:** 756 passing, 17 skipped (100% pass rate)
**TypeScript:** 0 compilation errors

**Architectural Change:** 
- **Before:** Frontend HTTP polling every 30s + backend WebSocket polling
- **After:** Initial HTTP fetch (one-time) + backend WebSocket (continuous)
- Backend abstracts Log Analytics polling vs Container Apps streaming

**Files Modified:** 13
**Files Deleted:** 3


