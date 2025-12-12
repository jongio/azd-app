# Azure Logs Controls Feature

## Overview

Added global interactive controls for Azure cloud logs mode in the main toolbar, allowing users to configure timeframe, refresh interval, and run diagnostics from a central location that affects all services.

## Implementation Date

December 11, 2025

## Features Added

### 1. Timeframe Selector
- **Location**: Main toolbar, next to the Azure cloud mode toggle
- **Options**: 15 minutes, 1 hour, 6 hours, 24 hours
- **Purpose**: Allows users to select the historical time range for Azure log queries
- **Default**: 15 minutes
- **Scope**: Global - applies to all services

### 2. Refresh Interval Selector
- **Location**: Main toolbar, next to the timeframe selector
- **Options**: 10s, 30s, 1m, 2m, 5m
- **Purpose**: Controls how frequently Azure logs are automatically refreshed
- **Default**: 30 seconds
- **Scope**: Global - applies to all services
- **Countdown**: Shows a countdown timer at the bottom of each log pane

### 3. Diagnostics Button
- **Location**: Main toolbar, next to the refresh interval selector
- **Icon**: Activity icon (⚡)
- **Purpose**: Opens the Azure Logs Diagnostics modal to troubleshoot connection and configuration issues
- **Scope**: Global - diagnoses the overall Azure logs configuration

### 4. Query Viewer Button (Per-Service)
- **Location**: Within each LogsPane Azure mode indicator bar
- **Icon**: Code icon (</>)
- **Purpose**: View and edit the KQL (Kusto Query Language) query for that specific service
- **Scope**: Per-service - each service can have its own custom query

## Visual Design

The Azure logs controls appear in the main toolbar when Azure mode is active:

```
┌─────────────────────────────────────────────────────────────────┐
│ Toolbar:                                                        │
│ [Pause] [Clear] [Auto-scroll] │ [Search...] │                  │
│ [🖥️Local/☁️Azure] [Timeframe: 15min▼] [Refresh: 30s▼]        │
│                   [⚡ Diagnostics] │ [Grid] [Settings]          │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ Service 1 - ☁️ Viewing Azure Logs                    [</> Query]│
├─────────────────────────────────────────────────────────────────┤
│ Log output...                                                   │
└─────────────────────────────────────────────────────────────────┘
```

## Benefits

1. **Centralized Control**: Global controls in one location for easy access
2. **Consistent Settings**: All services use the same timeframe and refresh rate
3. **Better UX**: Controls visible and accessible without scrolling
4. **Space Efficient**: Individual log panes are cleaner
5. **Clear Hierarchy**: Global settings in toolbar, per-service settings (query) in pane
6. **Visual Clarity**: Controls appear right next to mode toggle

## Technical Implementation

### Files Modified

1. **ConsoleView.tsx**
   - Added Azure control props to `LogsToolbarProps`
   - Added global state for `timeRange`, `syncInterval`, `showDiagnostics`
   - Added Azure controls in toolbar (shown when Azure mode active)
   - Integrated `DiagnosticsModal`

2. **LogsPane.tsx**
   - Removed per-service control callbacks
   - Simplified Azure indicator bar
   - Kept Query button for per-service queries

### State Management

```typescript
// Global state in ConsoleView
const [timeRange, setTimeRange] = useState({ preset: '15m' })
const [syncInterval, setSyncInterval] = useState(30000)
const [showDiagnostics, setShowDiagnostics] = useState(false)
```

## User Workflow

1. **Switch to Azure Mode**: Click Azure cloud icon → Controls appear in toolbar
2. **Adjust Timeframe**: Select from dropdown → All services refetch with new timeframe
3. **Adjust Interval**: Select refresh rate → All services update polling
4. **Run Diagnostics**: Click button → Modal shows global health checks
5. **View Service Query**: Click Query in pane → Edit service-specific query

## Testing

✅ All 645 tests pass
✅ No breaking changes
✅ 24 test files passed

## Architecture Decision

**Why Global Controls?**

- Single source of truth for all services
- Cleaner, more organized UI
- Easier to find and adjust settings
- Better alignment with mode toggle

**Why Keep Query Per-Service?**

- Each service has its own custom KQL query
- Query is specific to service's Azure resources
- Different query logic per service

## Related Components

- `DiagnosticsModal.tsx` - Azure diagnostics modal
- `ModeToggle.tsx` - Log source mode switcher
- `Select.tsx` - Native select component
- `useHistoricalLogs.ts` - Hook for querying Azure logs
