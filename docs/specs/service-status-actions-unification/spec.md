# Service Status and Actions Unification

## Overview

Unify all service status display and service action handling across the dashboard to ensure consistency and reduce code duplication.

## Problem

The dashboard has evolved with multiple components implementing their own status calculation and display logic:

### Status Display Inconsistencies

1. **Multiple custom implementations**:
   - `ServiceStatusCard.tsx` - has custom `calculateStatusCounts()` function
   - `Sidebar.tsx` - has custom `getStatusIndicator()` function
   - `PerformanceMetrics.tsx` - has custom `StatusBadge` component with inline config
   - `ServiceDependencies.tsx` - uses `getStatusIndicator()` from dependencies-utils

2. **Some components use unified approach**:
   - `StatusCell.tsx` - uses `getStatusDisplay()` from service-utils
   - `ServiceCard.tsx` - uses `getStatusDisplay()` from service-utils
   - `ServiceDetailPanel.tsx` - uses `getStatusDisplay()` from service-utils
   - `LogsPane.tsx` - uses `getLogPaneVisualStatus()` from service-utils

3. **Utility function duplication**:
   - `lib/service-utils.ts` - has `getStatusDisplay()`, `getLogPaneVisualStatus()`, `getEffectiveStatus()`
   - `lib/dependencies-utils.ts` - has separate `getStatusIndicator()` function
   - `lib/panel-utils.ts` - has `getStatusStyles()` function

### Service Actions Inconsistencies

1. **Individual service actions**:
   - `ServiceActions.tsx` component uses `useServiceOperations` hook correctly
   - Used in: LogsPane, ServiceTableRow, ServiceCard

2. **Bulk service actions**:
   - `LogsToolbar.tsx` receives callbacks from `LogsMultiPaneView`
   - `LogsMultiPaneView` uses `useServiceOperations` hook for bulk operations

The actions implementation is more unified than status display.

## Solution

Consolidate all status display logic into `lib/service-utils.ts` and ensure all components use these unified functions.

## Functional Requirements

### 1. Unified Status Display Functions

All components must use these functions from `lib/service-utils.ts`:

| Function | Purpose | Used By |
|----------|---------|---------|
| `getStatusDisplay(status, health)` | Get text, color, icon for a status | Status cells, cards, badges |
| `getEffectiveStatus(service)` | Extract status/health from service object | All components needing status |
| `getLogPaneVisualStatus(health, fallback, process)` | Get visual status for log panes | LogsPane, ServicePanes |
| `calculateStatusCounts(services)` | Aggregate status counts for summary | Headers, sidebars |
| `getStatusIndicator(status)` | Get icon/color for status indicators | All indicator UIs |

### 2. Remove Duplicate Implementations

Components to update:
- `ServiceStatusCard.tsx` - remove inline `calculateStatusCounts()`, use unified function
- `Sidebar.tsx` - remove inline `getStatusIndicator()`, use unified function
- `PerformanceMetrics.tsx` - remove `StatusBadge` inline config, use `getStatusDisplay()`
- `ServiceDependencies.tsx` - use `getStatusIndicator()` from service-utils instead of dependencies-utils

### 3. Consolidate Utility Functions

- Move `getStatusIndicator()` from `dependencies-utils.ts` to `service-utils.ts`
- Ensure `panel-utils.ts` delegates to `service-utils.ts` for status logic
- Remove duplicate status config objects from individual components

### 4. Unified Action Handling

Ensure all service action buttons use:
- `ServiceActions` component for individual service actions
- `useServiceOperations` hook for programmatic operations
- Same visual styling and disabled states

## Acceptance Criteria

1. All status display components use functions from `lib/service-utils.ts`
2. No duplicate status configuration objects in individual components
3. `StatusBadge` in PerformanceMetrics uses unified `getStatusDisplay()` 
4. ServiceStatusCard uses unified `calculateStatusCounts()` from service-utils
5. Sidebar uses unified status indicator function from service-utils
6. All status indicators show consistent icons and colors across the UI
7. All tests pass after refactoring
8. No visual changes to end users (refactor only)

