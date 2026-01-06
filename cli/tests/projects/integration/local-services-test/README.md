# Local Services Test Project

A minimal test project for testing health diagnostics and Azure Logs setup without any Azure configuration.

## Purpose

This project tests:
- **Health Diagnostic Tooltips** - Services with different health states
- **Azure Logs Setup Wizard** - All 4 steps from scratch
- **Error Handling** - No Azure resources, no credentials, etc.

## Services

### API (Port 8000)
- **Health**: Returns 503 (Unhealthy) with error details
- **Purpose**: Test unhealthy service diagnostics and error messages

### Worker (Port 8080)  
- **Health**: Returns 200 but slow (1-2s delay - Degraded)
- **Purpose**: Test degraded service diagnostics and performance warnings

### Web (Port 3000)
- **Health**: Returns 200 (Healthy)
- **Purpose**: Test healthy service display

## Running

```bash
cd cli/tests/projects/integration/local-services-test
azd app run
```

## Expected Dashboard Behavior

### Health States
- **API**: Red unhealthy icon, tooltip shows "Database connection failed"
- **Worker**: Yellow degraded icon, tooltip shows "Slow response time"  
- **Web**: Green healthy icon, tooltip shows uptime

### Azure Logs Tab
Since there's no Azure configuration:
- Step 1 (Workspace): Shows "No Azure credentials" or "No subscription"
- Step 2 (Diagnostic Settings): Shows "No services found" or "Not deployed"
- Step 3 (Verification): Can't verify without workspace
- Error recovery flows should work (Retry, Skip, etc.)

## Testing Health Diagnostics

Hover over each service's health icon to see:
- Error details and status code
- Response time
- Consecutive failures (if you wait for multiple checks)
- Suggested actions
- Copy diagnostics button

## Testing Azure Setup

Try to set up Azure logs without credentials to see:
- Authentication errors
- Missing resource errors
- Permission errors  
- All recovery paths (Retry, Skip, Back, etc.)
