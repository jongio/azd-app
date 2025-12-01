# Health Starting Status

## Overview

Add a "starting" health status for services during initialization, so they don't immediately show as "unhealthy" before they've had a chance to initialize.

## Problem

Currently, when services start:
1. They're registered with `health: "unknown"`
2. Health checks begin immediately and may report "unhealthy" before the service is ready
3. This creates a misleading user experience where services appear broken during normal startup

## Solution

Add a "starting" health status that:
1. Is set when services begin starting
2. Transitions to "healthy", "degraded", or "unhealthy" after health checks succeed/fail
3. Displays appropriately in the dashboard (yellow/warning indicator)

## Functional Requirements

### 1. Health Status Values

The system supports these health status values:
- `healthy` - Service is responding correctly to health checks
- `degraded` - Service is running but with reduced functionality
- `unhealthy` - Service is not responding to health checks
- `starting` - Service is initializing (new)
- `unknown` - Health status cannot be determined

### 2. Service Startup Flow

1. Service registers with status "starting" and health "starting"
2. Service process starts
3. Health checks begin with grace period
4. After first successful health check: health transitions to "healthy"
5. If health checks fail after grace period: health transitions to "unhealthy"

### 3. Dashboard Display

- "Starting" health shows with yellow/warning color
- "Starting" health shows with clock/loading indicator
- Health summary counts "starting" services appropriately

### 4. Health Summary Calculation

- Services with "starting" health are counted separately
- Overall status considers "starting" as neutral (not affecting overall health)
- "starting" does not count toward "unhealthy" totals

## Acceptance Criteria

1. New services show "starting" health instead of "unknown" during initialization
2. Dashboard displays "starting" health with appropriate visual indicators
3. Health transitions correctly from "starting" to appropriate status after checks
4. Existing "healthy", "degraded", "unhealthy", "unknown" behavior is unchanged
5. All tests pass with new health status value
