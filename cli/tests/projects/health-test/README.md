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

### 1. Install Dependencies

```bash
cd cli/tests/projects/health-test

# Install web service
cd web && npm install && cd ..

# Install admin service
cd admin && npm install && cd ..

# Install database service
cd database && npm install && cd ..

# Install API service
cd api && pip install -r requirements.txt && cd ..

# Install worker service
cd worker && pip install -r requirements.txt && cd ..
```

### 2. Start All Services

```bash
# Terminal 1: Web service
cd web && npm start

# Terminal 2: API service
cd api && python app.py

# Terminal 3: Database service
cd database && npm start

# Terminal 4: Worker service
cd worker && python worker.py

# Terminal 5: Admin service
cd admin && npm start
```

Or use the provided helper script:
```bash
chmod +x start-all.sh
./start-all.sh
```

### 3. Test Health Monitoring

```bash
# Build azd app CLI first (from repository root)
cd ../../..
go build -o azd-app ./src/cmd/app

# Run health checks (from health-test directory)
cd tests/projects/health-test

# Static health check (all services)
../../../azd-app health

# Streaming mode with real-time updates
../../../azd-app health --stream

# JSON output for automation
../../../azd-app health --output json

# Table format
../../../azd-app health --output table

# Filter specific services
../../../azd-app health --service web,api

# Verbose mode
../../../azd-app health --verbose

# Stream with JSON output (works great with jq)
../../../azd-app health --stream --output json | jq '.services[] | select(.status != "healthy")'
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
# Start all services, wait 30s for startup
../../../azd-app health
# Exit code: 0
```

### 2. One Service Unhealthy
```bash
# Stop the web service
# Kill web service process

../../../azd-app health
# Exit code: 1 (one or more unhealthy)
# Output shows web as "unhealthy"
```

### 3. Streaming Mode
```bash
../../../azd-app health --stream
# Live updates every 5 seconds
# Press Ctrl+C to stop
# Exit code: 130 (interrupted)
```

### 4. Service Starting (Grace Period)
```bash
# Start services one by one
# During start_period, failures don't count
../../../azd-app health --verbose
# Shows "starting" status during grace period
```

### 5. Performance Testing
```bash
# All services should respond quickly
../../../azd-app health --verbose
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
../../../azd-app health --verbose
```

### Registry Issues
```bash
# Check registry
cat .azure/services.json

# Clear registry
rm -rf .azure

# Re-run services
./start-all.sh
```

## Manual Testing Checklist

- [ ] Install all dependencies successfully
- [ ] Start all 5 services without errors
- [ ] Run `azd app health` - all services show healthy
- [ ] Stop web service - health check shows web as unhealthy
- [ ] Restart web service - health check shows web as healthy again
- [ ] Run `azd app health --stream` - see live updates every 5s
- [ ] Test JSON output with jq filtering
- [ ] Test table format output
- [ ] Test service filtering (--service web,api)
- [ ] Test verbose mode shows response times
- [ ] Kill all services - health check shows all unhealthy
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
