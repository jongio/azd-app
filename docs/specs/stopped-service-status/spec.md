# Stopped Service Status

## Overview

Enhance the dashboard to properly distinguish between "stopped" (intentionally stopped) and "not-running" (never started) services, providing clear visual feedback and appropriate actions for each state.

## Problem

Currently, the dashboard treats "stopped" and "not-running" services similarly:
1. Both show the same gray/neutral visual indicators
2. Users cannot easily distinguish services they intentionally stopped vs services that never ran
3. The HealthSummary doesn't track stopped services separately
4. StatusCell and other UI components don't differentiate these states

## Solution

Introduce proper "stopped" service handling throughout the UI:
1. Add distinct visual indicators for stopped services (different from not-running)
2. Track stopped services in HealthSummary
3. Update all UI components to display stopped state appropriately
4. Ensure service actions reflect the stopped state correctly

## Functional Requirements

### 1. Process Status Values

The system supports these process status values:
- `running` - Service process is actively running
- `ready` - Service is running and ready to accept requests
- `starting` - Service is initializing
- `stopping` - Service is in the process of stopping
- `stopped` - Service was running but has been intentionally stopped
- `error` - Service encountered an error
- `not-running` - Service has never been started

### 2. Visual Differentiation

| Status | Color | Icon | Description |
|--------|-------|------|-------------|
| running/ready | Green | ● (filled circle) | Actively running |
| starting | Yellow | ◐ (half circle) | Starting up |
| stopping | Yellow | ◑ (half circle) | Shutting down |
| stopped | Gray with border | ◉ (circle with dot) | Intentionally stopped |
| error | Red | ⚠ (warning) | Error state |
| not-running | Gray | ○ (empty circle) | Never started |

### 3. HealthSummary Enhancement

Update HealthSummary to track stopped services:
- Add `stopped: number` field to HealthSummary interface
- Stopped services should not count as unhealthy
- Stopped services should be displayed separately in status counts

### 4. UI Component Updates

Components requiring updates:
- `HealthSummary` type - add stopped count
- `ServiceStatusCard` - show stopped count separately
- `StatusCell` - distinct stopped styling
- `ServiceCard` - stopped badge variant
- `PerformanceMetrics` - stopped status badge
- `panel-utils.ts` - stopped color mapping
- `dependencies-utils.ts` - stopped indicator
- `service-utils.ts` - stopped display configuration

### 5. Service Actions for Stopped Services

When a service is stopped:
- Show "Start" button enabled
- Show "Restart" button enabled
- Show "Stop" button disabled (already stopped)

## Acceptance Criteria

1. Stopped services display with distinct visual styling (gray with border/dot icon)
2. HealthSummary includes stopped count in status breakdown
3. ServiceStatusCard shows stopped count with appropriate indicator
4. StatusCell differentiates stopped from not-running visually
5. All status-related UI components handle stopped state correctly
6. Service actions reflect stopped state appropriately
7. Existing status handling remains unchanged
8. All tests pass with new stopped status handling
