# Service Lifecycle Management - Specification

## Overview

A comprehensive service lifecycle management feature that provides consistent start, stop, and restart controls across the dashboard UI. This enables users to control individual services and perform bulk operations from any view (grid, table, fullscreen, logs pane).

## Problem Statement

Currently, the dashboard has basic service operations but lacks:
- Bulk operations (start all, stop all, restart all)
- Consistent UI controls across all views (fullscreen, logs panes)
- Proper concurrency control to prevent race conditions
- Operation queueing to prevent conflicting concurrent operations
- Visual feedback during operations
- Proper error handling and recovery

## Requirements

### Functional Requirements

#### Individual Service Controls
- Start: Start a stopped/errored service
- Stop: Stop a running/ready/starting service  
- Restart: Stop then start a running/ready service
- Controls visible on service cards, table rows, and log panes
- Controls respect current service state (show only valid actions)

#### Bulk Operations
- Start All: Start all stopped services concurrently
- Stop All: Stop all running services concurrently
- Restart All: Restart all services sequentially with proper ordering
- Available in both regular and fullscreen modes
- Available in logs multi-pane view toolbar

#### State Display
- Running service: Show pause icon (stop) and restart option
- Stopped service: Show play icon (start)
- Starting/Stopping service: Show loading indicator, disable actions
- Error state: Show start option (allow retry)

### Non-Functional Requirements

#### Concurrency Safety
- Prevent concurrent operations on the same service
- Queue conflicting operations
- Use mutex/locking at the backend
- Handle orphaned processes gracefully

#### Performance
- Bulk operations run concurrently where safe
- No blocking of UI during operations
- Real-time status updates via WebSocket

#### Reliability
- Graceful shutdown with configurable timeout
- Force kill fallback after timeout
- Process group termination on Windows and Unix
- Registry consistency maintained through all operations

## User Interface

### Service Grid/Table View
- Each service row/card shows context-appropriate action buttons
- Compact icon buttons for inline display
- Disabled during pending operations

### Logs Multi-Pane View
- Global toolbar includes bulk operation buttons (Start All, Stop All, Restart All)
- Each log pane header shows service-specific action buttons
- Controls available in both regular and fullscreen modes

### Visual Feedback
- Loading spinner during operations
- Success/error toast notifications
- Real-time status updates via WebSocket
- Disabled state for unavailable actions

## Backend API

### Existing Endpoints (Enhanced)
- `POST /api/services/start?service={name}` - Start single service
- `POST /api/services/stop?service={name}` - Stop single service
- `POST /api/services/restart?service={name}` - Restart single service

### New Endpoints (Bulk Operations)
- `POST /api/services/start` - Start all stopped services
- `POST /api/services/stop` - Stop all running services
- `POST /api/services/restart` - Restart all services

### Response Format
```json
{
  "success": true,
  "message": "Operation completed",
  "services": [
    { "name": "api", "status": "running", "error": null },
    { "name": "web", "status": "running", "error": null }
  ]
}
```

## Concurrency Model

### Go Backend
- Use sync.Mutex for operation locking per service
- ServiceOperationManager handles operation coordination
- Prevents concurrent start/stop on same service
- Queues operations when service is in transitional state

### Operation States
- `idle`: No operation in progress, accept new operations
- `starting`: Start operation in progress
- `stopping`: Stop operation in progress
- `restarting`: Restart operation in progress

## Success Criteria

- All individual service operations work reliably
- Bulk operations complete without race conditions
- UI remains responsive during operations
- Real-time status updates via WebSocket
- Proper error handling and recovery
- Works in fullscreen and regular modes
- Works in logs pane view
- No zombie processes after stop operations
