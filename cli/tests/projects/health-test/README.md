# Health Monitoring Test Project

Comprehensive test project for `azd app health` command demonstrating all health check types and configurations.

## Project Structure

This project contains 5 services with different health check configurations:

### 1. Web Service (HTTP Health Check with Custom Endpoint)
- **Port**: 3000
- **Health Check**: HTTP GET to `/health`
- **Interval**: 10s
- **Start Period**: 30s
- **Tech**: Node.js Express server

### 2. API Service (HTTP Health Check with Standard Endpoint)
- **Port**: 5000
- **Health Check**: HTTP GET to `/healthz`
- **Interval**: 15s
- **Start Period**: 20s
- **Tech**: Python Flask API

### 3. Database Service (TCP Port Check)
- **Port**: 5432
- **Health Check**: TCP connection to port (simulated PostgreSQL)
- **No HTTP endpoint**: Falls back to port check
- **Tech**: Node.js TCP server

### 4. Worker Service (Process-Only Check)
- **No Port**: Background worker
- **Health Check**: Process existence only
- **Tech**: Python background task

### 5. Admin Service (HTTP with Authentication)
- **Port**: 4000
- **Health Check**: HTTP GET to `/api/health` with Authorization header
- **Interval**: 5s (fast)
- **Tech**: Node.js Express with auth

## Quick Start

### Automated Test Script

**Linux/macOS:**
```bash
./quick-start.sh
```

**Windows:**
```powershell
.\quick-start.ps1
```

### Running E2E Integration Tests

The project includes comprehensive end-to-end integration tests that verify the health command works correctly across all platforms.

**Run E2E tests locally:**

```bash
# From cli directory
cd cli

# Run all E2E health tests
mage testE2E

# Or use go test directly
go test -v -tags=integration -timeout=15m ./src/cmd/app/commands -run TestHealthCommandE2E

# Run specific E2E test
go test -v -tags=integration ./src/cmd/app/commands -run TestHealthCommandE2E_FullWorkflow
```

**E2E tests include:**
- ✅ Full workflow test (install deps, start services, health checks)
- ✅ JSON output validation
- ✅ Table output formatting
- ✅ Service filtering
- ✅ Verbose mode
- ✅ Streaming mode
- ✅ Error handling (no services, invalid params)
- ✅ Cross-platform process checking

**CI/CD:**
- E2E tests run automatically on every PR affecting health command
- Tests run on Ubuntu, Windows, and macOS
- Manual workflow dispatch available: `.github/workflows/health-e2e.yml`

### Manual Setup

### 1. Start All Services

```bash
cd cli/tests/projects/health-test

# Use azd app run to start all services automatically
azd app run
```

This will:
- Automatically detect all services from `azure.yaml`
- Install dependencies for each service
- Start all services in the background
- Monitor health status
- Display service URLs and status

**Note**: The old `start-all.sh` and `stop-all.sh` scripts are deprecated. Use `azd app run` and Ctrl+C to stop.

### 2. Test Health Monitoring

In a separate terminal (while `azd app run` is running):

```bash
cd cli/tests/projects/health-test

# Static health check (all services)
azd app health

# Streaming mode with real-time updates
azd app health --stream

# JSON output for automation
azd app health --output json

# Table format
azd app health --output table

# Filter specific services
azd app health --service web,api

# Verbose mode
azd app health --verbose

# Stream with JSON output (works great with jq)
azd app health --stream --output json | jq '.services[] | select(.status != "healthy")'
```

## Expected Health Check Behaviors

### Web Service
- **Healthy**: Returns 200 OK from `/health`
- **Response Time**: <50ms
- **Status**: `{"status": "healthy", "service": "web", "version": "1.0.0"}`

### API Service
- **Healthy**: Returns 200 OK from `/healthz`
- **Response Time**: <100ms
- **Status**: `{"status": "ok", "database": "connected"}`

### Database Service
- **Healthy**: TCP connection succeeds on port 5432
- **Fallback**: No HTTP endpoint, uses port check
- **Response Time**: <10ms

### Worker Service
- **Healthy**: Process is running
- **Fallback**: No port, uses process check only
- **Check**: Verifies PID exists

### Admin Service
- **Healthy**: Returns 200 OK from `/api/health` with auth header
- **Authentication**: Requires `Authorization: Bearer test-token-123`
- **Response Time**: <30ms

## Testing Different Scenarios

### 1. All Services Healthy
```bash
# Start all services with azd app run in one terminal
azd app run

# In another terminal, check health
azd app health
# Exit code: 0
```

### 2. One Service Unhealthy
```bash
# Stop one service from the registry
azd app info  # Get the PID
kill <PID>     # Kill specific service

azd app health
# Exit code: 1 (one or more unhealthy)
# Output shows service as "unhealthy"
```

### 3. Streaming Mode
```bash
azd app health --stream
# Live updates every 5 seconds
# Press Ctrl+C to stop
# Exit code: 130 (interrupted)
```

### 4. Service Starting (Grace Period)
```bash
# Watch services start with azd app run
# During start_period, failures don't count
azd app health --verbose
# Shows "starting" status during grace period
```

### 5. Performance Testing
```bash
# All services should respond quickly
azd app health --verbose
# Check response times in verbose output
# Web: <50ms, API: <100ms, DB: <10ms, Worker: <5ms, Admin: <30ms
```

## Troubleshooting

### Services Won't Start
```bash
# Check if ports are already in use
lsof -i :3000  # Web
lsof -i :5000  # API
lsof -i :5432  # Database
lsof -i :4000  # Admin
```

### Health Checks Failing
```bash
# Test manually
curl http://localhost:3000/health
curl http://localhost:5000/healthz
nc -zv localhost 5432
curl -H "Authorization: Bearer test-token-123" http://localhost:4000/api/health

# Check logs
azd app logs --service <service-name>
azd app health --verbose
```

### Registry Issues
```bash
# Check registry and service info
azd app info
cat .azure/services.json

# Clear registry
rm -rf .azure

# Re-run services
azd app run
```

## Manual Testing Checklist

- [ ] Run `azd app run` - all 5 services start successfully
- [ ] Run `azd app health` - all services show healthy
- [ ] Run `azd app info` - verify all services registered
- [ ] Stop web service - health check shows web as unhealthy
- [ ] Restart with `azd app run` - health check shows all healthy again
- [ ] Run `azd app health --stream` - see live updates every 5s
- [ ] Test JSON output with jq filtering
- [ ] Test table format output
- [ ] Test service filtering (--service web,api)
- [ ] Test verbose mode shows response times
- [ ] Press Ctrl+C on `azd app run` - all services stop
- [ ] Performance check - all checks complete in <5s total

## Coverage Validation

This test project covers:
- ✅ HTTP health checks (web, api, admin)
- ✅ TCP port checks (database)
- ✅ Process checks (worker)
- ✅ Authentication headers (admin)
- ✅ Different health endpoints (/health, /healthz, /api/health)
- ✅ Different intervals and timeouts
- ✅ Grace periods (start_period)
- ✅ Retry logic (retries)
- ✅ All output formats (text, JSON, table)
- ✅ Static and streaming modes
- ✅ Service filtering
- ✅ Verbose logging
- ✅ Exit codes (0, 1, 2, 130)

## Next Steps

After manual testing:
1. Document any issues found
2. Add edge case tests for any bugs discovered
3. Update documentation with real-world usage patterns
4. Consider adding Docker Compose test parsing in v1.1
