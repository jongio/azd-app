# Manual Testing Guide - `azd app health` Command

**Version:** 1.0.0  
**Last Updated:** November 14, 2025  
**Estimated Time:** 45-60 minutes

---

## Prerequisites

### Required Tools
- [x] `azd` CLI installed (build from source or use latest dev build)
- [x] Docker Desktop installed and running
- [x] `curl` or similar HTTP client
- [x] Terminal with color support (for streaming mode)

### Test Environment Setup

1. **Clone the test project:**
   ```powershell
   cd c:\code\azd-app-2\cli\tests\projects
   ```

2. **Verify Docker is running:**
   ```powershell
   docker ps
   # Should return container list without errors
   ```

3. **Build azd from source:**
   ```powershell
   cd c:\code\azd-app-2\cli
   go build -o bin/azd.exe ./src/cmd
   
   # Add to PATH or use full path
   $env:PATH = "c:\code\azd-app-2\cli\bin;$env:PATH"
   ```

---

## Test Suite Overview

| Test # | Category | Duration | Difficulty |
|--------|----------|----------|------------|
| 1 | Basic Functionality | 5 min | Easy |
| 2 | Streaming Mode | 5 min | Easy |
| 3 | Service Filtering | 5 min | Easy |
| 4 | Output Formats | 5 min | Easy |
| 5 | Circuit Breaker | 10 min | Medium |
| 6 | Rate Limiting | 5 min | Medium |
| 7 | Health Profiles | 5 min | Easy |
| 8 | Prometheus Metrics | 10 min | Medium |
| 9 | Error Scenarios | 10 min | Medium |
| 10 | Signal Handling | 5 min | Easy |

**Total:** 10 test scenarios covering all major features

---

## Test Scenario 1: Basic Health Check ✅

**Objective:** Verify basic health check functionality

### Setup
```powershell
# Navigate to test project
cd c:\code\azd-app-2\cli\tests\projects\health-test

# Start services
docker-compose up -d

# Wait for services to start (10 seconds)
Start-Sleep -Seconds 10
```

### Test Steps

1. **Run basic health check:**
   ```powershell
   azd app health
   ```

2. **Expected Output:**
   ```
   Checking health of 3 services...
   
   ✓ web-api      HEALTHY   (responded in 15ms)
   ✓ database     HEALTHY   (port 5432 is listening)
   ✓ redis        HEALTHY   (port 6379 is listening)
   
   Summary: 3/3 services healthy
   ```

3. **Verify:**
   - [x] All services show HEALTHY status
   - [x] Response times are < 100ms
   - [x] No error messages
   - [x] Summary shows correct counts

### Cleanup
```powershell
# Keep services running for next tests
```

**Status:** ⬜ PASS / ⬜ FAIL

**Notes:**
_______________________________________

---

## Test Scenario 2: Streaming Mode 🌊

**Objective:** Verify real-time streaming updates

### Test Steps

1. **Start streaming mode:**
   ```powershell
   azd app health --stream --interval 2s
   ```

2. **Expected Output:**
   ```
   Streaming health checks every 2s (press Ctrl+C to stop)...
   
   [12:34:56] ✓ web-api    HEALTHY (15ms)  ✓ database  HEALTHY (2ms)   ✓ redis     HEALTHY (1ms)
   [12:34:58] ✓ web-api    HEALTHY (14ms)  ✓ database  HEALTHY (2ms)   ✓ redis     HEALTHY (1ms)
   [12:35:00] ✓ web-api    HEALTHY (16ms)  ✓ database  HEALTHY (2ms)   ✓ redis     HEALTHY (1ms)
   ```

3. **Let it run for 30 seconds, then press Ctrl+C**

4. **Expected behavior:**
   - [x] Updates appear every 2 seconds
   - [x] Timestamps increment correctly
   - [x] Graceful shutdown on Ctrl+C
   - [x] No panic or error messages

5. **Test shorter interval:**
   ```powershell
   azd app health --stream --interval 500ms
   ```
   - [x] Updates appear every 500ms
   - [x] No lag or stuttering

**Status:** ⬜ PASS / ⬜ FAIL

**Notes:**
_______________________________________

---

## Test Scenario 3: Service Filtering 🔍

**Objective:** Verify filtering by service names

### Test Steps

1. **Filter single service:**
   ```powershell
   azd app health --services web-api
   ```

2. **Expected Output:**
   ```
   Checking health of 1 service...
   
   ✓ web-api      HEALTHY   (responded in 15ms)
   
   Summary: 1/1 services healthy
   ```

3. **Filter multiple services:**
   ```powershell
   azd app health --services "web-api,redis"
   ```

4. **Expected Output:**
   ```
   Checking health of 2 services...
   
   ✓ web-api      HEALTHY   (responded in 15ms)
   ✓ redis        HEALTHY   (port 6379 is listening)
   
   Summary: 2/2 services healthy
   ```

5. **Test non-existent service:**
   ```powershell
   azd app health --services nonexistent
   ```

6. **Expected Output:**
   ```
   Error: service 'nonexistent' not found in compose file
   Available services: web-api, database, redis
   ```

**Verify:**
- [x] Filtering works correctly
- [x] Summary counts match filtered services
- [x] Clear error for invalid service names
- [x] Helpful suggestion with available services

**Status:** ⬜ PASS / ⬜ FAIL

**Notes:**
_______________________________________

---

## Test Scenario 4: Output Formats 📊

**Objective:** Verify JSON and table output formats

### Test Steps

1. **JSON output:**
   ```powershell
   azd app health --format json
   ```

2. **Expected Output (valid JSON):**
   ```json
   {
     "services": [
       {
         "name": "web-api",
         "status": "healthy",
         "latency_ms": 15,
         "message": "HTTP 200 OK"
       },
       {
         "name": "database",
         "status": "healthy",
         "latency_ms": 2,
         "message": "Port 5432 is listening"
       },
       {
         "name": "redis",
         "status": "healthy",
         "latency_ms": 1,
         "message": "Port 6379 is listening"
       }
     ],
     "summary": {
       "total": 3,
       "healthy": 3,
       "unhealthy": 0
     },
     "timestamp": "2025-11-14T12:34:56Z"
   }
   ```

3. **Validate JSON:**
   ```powershell
   azd app health --format json | ConvertFrom-Json
   # Should parse without errors
   ```

4. **Table output:**
   ```powershell
   azd app health --format table
   ```

5. **Expected Output:**
   ```
   ┌──────────┬──────────┬────────────┬─────────────────────────┐
   │ SERVICE  │ STATUS   │ LATENCY    │ MESSAGE                 │
   ├──────────┼──────────┼────────────┼─────────────────────────┤
   │ web-api  │ HEALTHY  │ 15ms       │ HTTP 200 OK             │
   │ database │ HEALTHY  │ 2ms        │ Port 5432 is listening  │
   │ redis    │ HEALTHY  │ 1ms        │ Port 6379 is listening  │
   └──────────┴──────────┴────────────┴─────────────────────────┘
   ```

**Verify:**
- [x] JSON is valid and parseable
- [x] JSON contains all expected fields
- [x] Table has proper alignment
- [x] Table uses box-drawing characters

**Status:** ⬜ PASS / ⬜ FAIL

**Notes:**
_______________________________________

---

## Test Scenario 5: Circuit Breaker 🔌

**Objective:** Verify circuit breaker protects failing services

### Setup
```powershell
# Stop web-api to simulate failures
docker-compose stop web-api
```

### Test Steps

1. **Run health check with circuit breaker enabled:**
   ```powershell
   azd app health --circuit-breaker --circuit-breaker-threshold 3
   ```

2. **First check - Circuit CLOSED (normal operation):**
   ```
   ✗ web-api      UNHEALTHY (circuit: closed)
   ✓ database     HEALTHY
   ✓ redis        HEALTHY
   ```

3. **Run again immediately:**
   ```powershell
   azd app health --circuit-breaker --circuit-breaker-threshold 3
   ```

4. **Second check - Still CLOSED:**
   ```
   ✗ web-api      UNHEALTHY (circuit: closed)
   ```

5. **Run third time:**
   ```powershell
   azd app health --circuit-breaker --circuit-breaker-threshold 3
   ```

6. **Third check - Circuit OPENS:**
   ```
   ✗ web-api      CIRCUIT_OPEN (too many failures, circuit opened)
   ```

7. **Subsequent checks should show OPEN without attempting connection:**
   ```powershell
   azd app health --circuit-breaker --circuit-breaker-threshold 3
   ```

8. **Expected:**
   ```
   ✗ web-api      CIRCUIT_OPEN (circuit is open, not checking)
   ```

9. **Restart service:**
   ```powershell
   docker-compose start web-api
   Start-Sleep -Seconds 10
   ```

10. **Wait for circuit to reset (30 seconds), then check again:**
    ```powershell
    Start-Sleep -Seconds 30
    azd app health --circuit-breaker --circuit-breaker-threshold 3
    ```

11. **Circuit should be HALF_OPEN, then CLOSED:**
    ```
    ✓ web-api      HEALTHY (circuit: half-open → closed)
    ```

**Verify:**
- [x] Circuit opens after threshold failures
- [x] Circuit prevents checks when open
- [x] Circuit resets after timeout
- [x] Service recovers when circuit closes

**Status:** ⬜ PASS / ⬜ FAIL

**Notes:**
_______________________________________

---

## Test Scenario 6: Rate Limiting ⏱️

**Objective:** Verify rate limiting prevents service overload

### Test Steps

1. **Enable rate limiting:**
   ```powershell
   azd app health --rate-limit 1 --rate-limit-burst 1
   ```

2. **Run health check twice in quick succession:**
   ```powershell
   azd app health --rate-limit 1 --rate-limit-burst 1
   # Immediately run again
   azd app health --rate-limit 1 --rate-limit-burst 1
   ```

3. **Expected Output (second run):**
   ```
   ✗ web-api      RATE_LIMITED (rate limit exceeded, try again later)
   ✗ database     RATE_LIMITED
   ✗ redis        RATE_LIMITED
   ```

4. **Wait 2 seconds and try again:**
   ```powershell
   Start-Sleep -Seconds 2
   azd app health --rate-limit 1 --rate-limit-burst 1
   ```

5. **Expected Output:**
   ```
   ✓ web-api      HEALTHY
   ✓ database     HEALTHY
   ✓ redis        HEALTHY
   ```

6. **Test burst behavior:**
   ```powershell
   azd app health --rate-limit 10 --rate-limit-burst 3
   # Run 3 times quickly (should all succeed)
   azd app health --rate-limit 10 --rate-limit-burst 3
   azd app health --rate-limit 10 --rate-limit-burst 3
   azd app health --rate-limit 10 --rate-limit-burst 3
   # 4th should be rate limited
   azd app health --rate-limit 10 --rate-limit-burst 3
   ```

**Verify:**
- [x] Rate limiting prevents excessive checks
- [x] Clear error message when rate limited
- [x] Tokens replenish over time
- [x] Burst allows initial requests

**Status:** ⬜ PASS / ⬜ FAIL

**Notes:**
_______________________________________

---

## Test Scenario 7: Health Profiles 📋

**Objective:** Verify predefined profiles work correctly

### Test Steps

1. **Development profile:**
   ```powershell
   azd app health --profile dev
   ```

2. **Expected behavior:**
   - [x] No circuit breaker
   - [x] No rate limiting
   - [x] Short timeout (5s)
   - [x] Pretty logs

3. **Production profile:**
   ```powershell
   azd app health --profile production
   ```

4. **Expected behavior:**
   - [x] Circuit breaker enabled (threshold: 5)
   - [x] Rate limiting enabled (10 req/s, burst 20)
   - [x] Longer timeout (30s)
   - [x] JSON logs

5. **CI profile:**
   ```powershell
   azd app health --profile ci
   ```

6. **Expected behavior:**
   - [x] Circuit breaker enabled (threshold: 3)
   - [x] No rate limiting
   - [x] Medium timeout (15s)
   - [x] JSON logs

7. **Staging profile:**
   ```powershell
   azd app health --profile staging
   ```

8. **Expected behavior:**
   - [x] Circuit breaker enabled (threshold: 5)
   - [x] Rate limiting enabled (5 req/s, burst 10)
   - [x] Medium timeout (20s)
   - [x] Pretty logs

**Verify:**
- [x] Each profile applies correct settings
- [x] Profiles override individual flags
- [x] No errors loading profiles

**Status:** ⬜ PASS / ⬜ FAIL

**Notes:**
_______________________________________

---

## Test Scenario 8: Prometheus Metrics 📊

**Objective:** Verify Prometheus metrics are exposed correctly

### Test Steps

1. **Enable metrics on port 9090:**
   ```powershell
   # Start in background
   Start-Job -ScriptBlock {
     cd c:\code\azd-app-2\cli\tests\projects\health-test
     azd app health --stream --interval 5s --metrics-port 9090
   }
   
   # Wait for startup
   Start-Sleep -Seconds 3
   ```

2. **Fetch metrics:**
   ```powershell
   curl http://localhost:9090/metrics
   ```

3. **Expected metrics (verify these exist):**
   ```
   # HELP azd_health_check_duration_seconds Duration of health checks
   # TYPE azd_health_check_duration_seconds histogram
   azd_health_check_duration_seconds_bucket{service="web-api",le="0.005"} 0
   azd_health_check_duration_seconds_bucket{service="web-api",le="0.01"} 1
   azd_health_check_duration_seconds_sum{service="web-api"} 0.015
   azd_health_check_duration_seconds_count{service="web-api"} 1
   
   # HELP azd_health_check_total Total number of health checks
   # TYPE azd_health_check_total counter
   azd_health_check_total{service="web-api",status="healthy"} 1
   
   # HELP azd_health_check_errors_total Total number of health check errors
   # TYPE azd_health_check_errors_total counter
   azd_health_check_errors_total{service="web-api",error_type="none"} 0
   
   # HELP azd_service_uptime_seconds Service uptime in seconds
   # TYPE azd_service_uptime_seconds gauge
   azd_service_uptime_seconds{service="web-api"} 120.5
   
   # HELP azd_circuit_breaker_state Circuit breaker state
   # TYPE azd_circuit_breaker_state gauge
   azd_circuit_breaker_state{service="web-api",state="closed"} 1
   
   # HELP azd_health_check_http_status_total HTTP status codes
   # TYPE azd_health_check_http_status_total counter
   azd_health_check_http_status_total{service="web-api",status="200"} 1
   ```

4. **Verify metric format:**
   ```powershell
   # Should be valid Prometheus format
   curl http://localhost:9090/metrics | Select-String "azd_health"
   ```

5. **Check all 6 metric types exist:**
   - [x] `azd_health_check_duration_seconds` (histogram)
   - [x] `azd_health_check_total` (counter)
   - [x] `azd_health_check_errors_total` (counter)
   - [x] `azd_service_uptime_seconds` (gauge)
   - [x] `azd_circuit_breaker_state` (gauge)
   - [x] `azd_health_check_http_status_total` (counter)

6. **Stop background job:**
   ```powershell
   Get-Job | Stop-Job
   Get-Job | Remove-Job
   ```

**Status:** ⬜ PASS / ⬜ FAIL

**Notes:**
_______________________________________

---

## Test Scenario 9: Error Scenarios 🚨

**Objective:** Verify graceful error handling

### Test Steps

#### 9.1: Invalid Port
```powershell
azd app health --metrics-port 99999
```

**Expected:**
```
Error: invalid metrics port: must be between 1 and 65535
```

#### 9.2: Invalid Circuit Breaker Threshold
```powershell
azd app health --circuit-breaker --circuit-breaker-threshold 0
```

**Expected:**
```
Error: circuit breaker threshold must be >= 1
```

#### 9.3: Invalid Rate Limit
```powershell
azd app health --rate-limit -1
```

**Expected:**
```
Error: rate limit must be >= 0
```

#### 9.4: Missing Compose File
```powershell
cd c:\temp
azd app health
```

**Expected:**
```
Error: no docker-compose.yml file found in current directory
```

#### 9.5: Service Connection Timeout
```powershell
# Stop all services
cd c:\code\azd-app-2\cli\tests\projects\health-test
docker-compose stop

azd app health --timeout 2s
```

**Expected:**
```
✗ web-api      UNHEALTHY (connection timeout after 2s)
✗ database     UNHEALTHY (port 5432 not listening)
✗ redis        UNHEALTHY (port 6379 not listening)

Summary: 0/3 services healthy
```

#### 9.6: Invalid JSON in Compose File
```powershell
# Backup and corrupt compose file
Copy-Item docker-compose.yml docker-compose.yml.bak
"invalid yaml content" | Set-Content docker-compose.yml

azd app health
```

**Expected:**
```
Error: failed to parse docker-compose.yml: yaml: unmarshal errors:
  ...
```

```powershell
# Restore
Move-Item docker-compose.yml.bak docker-compose.yml -Force
docker-compose up -d
Start-Sleep -Seconds 10
```

**Verify:**
- [x] All error messages are clear and actionable
- [x] No panics or stack traces shown to user
- [x] Proper exit codes (non-zero for errors)
- [x] Helpful suggestions provided

**Status:** ⬜ PASS / ⬜ FAIL

**Notes:**
_______________________________________

---

## Test Scenario 10: Signal Handling 🛑

**Objective:** Verify graceful shutdown on Ctrl+C

### Test Steps

1. **Start streaming mode:**
   ```powershell
   azd app health --stream --interval 2s
   ```

2. **Let it run for ~10 seconds (5 iterations)**

3. **Press Ctrl+C**

4. **Expected behavior:**
   - [x] Immediate response (< 1 second)
   - [x] Clean shutdown message
   - [x] No panic or error messages
   - [x] No goroutine leaks (process exits cleanly)
   - [x] Terminal returns to prompt

5. **Test with metrics enabled:**
   ```powershell
   azd app health --stream --interval 2s --metrics-port 9090
   ```

6. **Press Ctrl+C**

7. **Expected behavior:**
   - [x] Metrics server shuts down cleanly
   - [x] Port 9090 is released
   - [x] No "address already in use" errors

8. **Verify port is released:**
   ```powershell
   # Should fail (port should be free)
   curl http://localhost:9090/metrics
   # Expected: Connection refused
   ```

**Status:** ⬜ PASS / ⬜ FAIL

**Notes:**
_______________________________________

---

## Final Cleanup

```powershell
# Stop all test services
cd c:\code\azd-app-2\cli\tests\projects\health-test
docker-compose down

# Kill any remaining background jobs
Get-Job | Stop-Job
Get-Job | Remove-Job

# Verify no lingering processes
Get-Process azd -ErrorAction SilentlyContinue | Stop-Process
```

---

## Test Results Summary

| Test # | Scenario | Status | Notes |
|--------|----------|--------|-------|
| 1 | Basic Health Check | ⬜ | |
| 2 | Streaming Mode | ⬜ | |
| 3 | Service Filtering | ⬜ | |
| 4 | Output Formats | ⬜ | |
| 5 | Circuit Breaker | ⬜ | |
| 6 | Rate Limiting | ⬜ | |
| 7 | Health Profiles | ⬜ | |
| 8 | Prometheus Metrics | ⬜ | |
| 9 | Error Scenarios | ⬜ | |
| 10 | Signal Handling | ⬜ | |

**Overall Result:** ⬜ PASS / ⬜ FAIL

**Total Tests:** 10  
**Passed:** _____ / 10  
**Failed:** _____ / 10  

---

## Issues Found

### Critical Issues
_None expected - all critical issues were fixed in code review_

### Medium Issues
| Issue | Description | Severity | Workaround |
|-------|-------------|----------|------------|
| | | | |

### Low Issues
| Issue | Description | Impact |
|-------|-------------|--------|
| | | |

---

## Performance Observations

| Metric | Expected | Actual | Status |
|--------|----------|--------|--------|
| Health check latency | <100ms | _____ ms | ⬜ |
| Memory usage (idle) | <100MB | _____ MB | ⬜ |
| CPU usage (streaming) | <5% | _____ % | ⬜ |
| Startup time | <1s | _____ s | ⬜ |
| Shutdown time | <1s | _____ s | ⬜ |

---

## Recommendations

### For Users
- Use `--profile production` in production environments
- Enable metrics for observability
- Set appropriate timeouts for your environment
- Use streaming mode for debugging

### For Developers
_Leave blank unless issues found_

---

## Tester Information

**Tester Name:** _________________  
**Date:** _________________  
**Environment:**
- OS: _________________
- Docker Version: _________________
- azd Version: _________________

**Signature:** _________________

---

## Quick Reference: Common Commands

```powershell
# Basic health check
azd app health

# Streaming with custom interval
azd app health --stream --interval 5s

# Production mode with metrics
azd app health --profile production --metrics-port 9090

# Debug mode with verbose logging
azd app health --log-level debug --log-format pretty

# Check specific services
azd app health --services "web-api,database"

# JSON output for CI/CD
azd app health --format json --profile ci

# Maximum safety (circuit breaker + rate limiting)
azd app health --circuit-breaker --rate-limit 10
```

---

## Troubleshooting Tips

### Services show UNHEALTHY
1. Check Docker containers are running: `docker ps`
2. Check service logs: `docker-compose logs <service>`
3. Verify ports are exposed in docker-compose.yml
4. Test connectivity manually: `curl http://localhost:<port>`

### Metrics not accessible
1. Verify port is not in use: `netstat -an | Select-String 9090`
2. Check firewall settings
3. Ensure metrics-port flag is set correctly
4. Try different port: `--metrics-port 9091`

### Circuit breaker always open
1. Reduce threshold: `--circuit-breaker-threshold 2`
2. Increase timeout: `--timeout 30s`
3. Check service is actually healthy
4. Wait for circuit reset (30s default)

### Rate limiting too aggressive
1. Increase burst: `--rate-limit-burst 50`
2. Increase rate: `--rate-limit 20`
3. Use appropriate profile: `--profile dev` (no rate limiting)

---

**End of Manual Testing Guide**
