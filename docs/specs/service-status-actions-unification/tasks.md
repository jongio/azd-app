# Service Status and Actions Unification - Tasks

## Progress: 5/5 DONE

---

## Task 1: Consolidate Status Indicator Functions
**Agent**: Developer
**Status**: ✅ DONE

Moved `getStatusIndicator()` from `lib/dependencies-utils.ts` to `lib/service-utils.ts` as the single source of truth for status indicators. Updated dependencies-utils to re-export from service-utils.

**Files updated**:
- lib/service-utils.ts - added `getStatusIndicator()` and `StatusIndicator` interface
- lib/dependencies-utils.ts - removed function, now re-exports from service-utils

---

## Task 2: Unify ServiceStatusCard Status Calculation
**Agent**: Developer
**Status**: ✅ DONE

Added unified `calculateStatusCounts()` function to service-utils and updated ServiceStatusCard to use it instead of inline function.

**Files updated**:
- lib/service-utils.ts - added `calculateStatusCounts()` function with `StatusCounts` interface
- components/ServiceStatusCard.tsx - removed inline `calculateStatusCounts()`, imports from service-utils

---

## Task 3: Unify Sidebar Status Indicator
**Agent**: Developer
**Status**: ✅ DONE

Updated Sidebar to use unified `calculateStatusCounts()` from service-utils instead of inline logic.

**Files updated**:
- components/Sidebar.tsx - removed inline logic, uses `calculateStatusCounts()` from service-utils

---

## Task 4: Unify PerformanceMetrics StatusBadge
**Agent**: Developer
**Status**: ✅ DONE

Added `getStatusBadgeConfig()` and `getHealthBadgeConfig()` functions to service-utils. Refactored PerformanceMetrics to use these unified functions.

**Files updated**:
- lib/service-utils.ts - added `getStatusBadgeConfig()` and `getHealthBadgeConfig()` functions
- components/views/PerformanceMetrics.tsx - refactored StatusBadge and HealthBadge to use unified functions

---

## Task 5: Add Integration Tests for Status Consistency
**Agent**: Developer
**Status**: ✅ DONE

Added comprehensive tests to verify all status display functions return consistent values.

**Files updated**:
- lib/service-utils.consistency.test.ts - 30 new tests for status consistency

All 930 tests pass (3 pre-existing Azure URL styling failures unrelated to this work).

---

