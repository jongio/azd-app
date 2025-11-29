# Manual Testing Guide for azd app health

This guide provides step-by-step instructions for thorough manual testing to achieve 100% confidence.

## Prerequisites

- Node.js 18+ and npm
- Python 3.9+ and pip
- Go 1.21+ (to build azd-app)
- curl (for manual testing)
- jq (optional, for JSON filtering)

## Setup Instructions

### 1. Build the azd-app CLI

```bash
cd /home/runner/work/azd-app/azd-app/cli
go build -o azd-app ./src/cmd/app
```

### 2. Navigate to the test project

```bash
cd tests/projects/health-test
```

### 3. Start all services

```bash
./start-all.sh
```

Wait 30 seconds for all services to initialize (grace periods).

## Manual Test Scenarios

### Test 1: All Services Healthy (Static Mode)

**Expected Behavior:** All 5 services report as healthy

```bash
../../../azd-app health
```

**Expected Output:**
```
Service Health Status:

✓ web       healthy    http://localhost:3000    <50ms
✓ api       healthy    http://localhost:5000    <100ms
✓ database  healthy    localhost:5432           <10ms
✓ worker    healthy    (no port)                <5ms
✓ admin     healthy    http://localhost:4000    <30ms

Summary: 5 healthy, 0 unhealthy, 0 degraded
```

**Expected Exit Code:** 0

**Manual Verification:**
```bash
echo "Exit code: $?"
# Should show: Exit code: 0
```

### Test 2: JSON Output Format

```bash
../../../azd-app health --output json
```

**Expected Output:** Valid JSON with all services

**Manual Verification:**
```bash
../../../azd-app health --output json | jq .
# Should pretty-print valid JSON

../../../azd-app health --output json | jq '.services | length'
# Should show: 5
```

### Test 3: Table Output Format

```bash
../../../azd-app health --output table
```

**Expected Output:** Tabular format with borders

**Manual Verification:**
- Check for proper alignment
- Verify all 5 services listed
- Verify columns: Service, Status, Endpoint, Response Time

### Test 4: Verbose Mode

```bash
../../../azd-app health --verbose
```

**Expected Output:** Detailed information including:
- Health check method used for each service
- Response times
- PID for worker service
- Port check details for database

**Manual Verification:**
- Web: Should show "HTTP health check to /health"
- API: Should show "HTTP health check to /healthz"
- Database: Should show "Port check on 5432"
- Worker: Should show "Process check (PID: xxxxx)"
- Admin: Should show "HTTP health check to /api/health"

### Test 5: Service Filtering

```bash
# Filter to web and api only
../../../azd-app health --service web,api
```

**Expected Output:** Only web and api services shown

**Manual Verification:**
```bash
../../../azd-app health --service web,api --output json | jq '.services | length'
# Should show: 2
```

### Test 6: Streaming Mode (Interactive)

```bash
../../../azd-app health --stream
```

**Expected Behavior:**
- Live updates every 5 seconds
- Screen clears and redraws
- Shows uptime
- Shows recent changes
- Press Ctrl+C to stop

**Manual Verification:**
1. Let it run for 15-20 seconds
2. Observe at least 3 updates
3. Press Ctrl+C
4. Verify clean shutdown with exit code 130

```bash
echo "Exit code: $?"
# Should show: Exit code: 130
```

### Test 7: Streaming Mode with JSON (Piped)

```bash
../../../azd-app health --stream --output json | head -3
```

**Expected Behavior:**
- NDJSON format (one JSON object per line)
- Each line is valid JSON
- Can be piped to jq

**Manual Verification:**
```bash
# Run for 15 seconds and capture
timeout 15 ../../../azd-app health --stream --output json > /tmp/health-stream.json

# Count lines
wc -l /tmp/health-stream.json
# Should show 3 lines (3 checks at 5s intervals)

# Verify each line is valid JSON
cat /tmp/health-stream.json | jq .
```

### Test 8: One Service Unhealthy

**Stop the web service:**
```bash
kill $(cat logs/web.pid)
```

**Run health check:**
```bash
../../../azd-app health
```

**Expected Output:**
```
Service Health Status:

✗ web       unhealthy  http://localhost:3000    (connection refused)
✓ api       healthy    http://localhost:5000    <100ms
✓ database  healthy    localhost:5432           <10ms
✓ worker    healthy    (no port)                <5ms
✓ admin     healthy    http://localhost:4000    <30ms

Summary: 4 healthy, 1 unhealthy, 0 degraded
```

**Expected Exit Code:** 1

**Manual Verification:**
```bash
echo "Exit code: $?"
# Should show: Exit code: 1
```

**Restart web service:**
```bash
cd web && nohup npm start > ../logs/web.log 2>&1 &
echo $! > ../logs/web.pid
cd ..
sleep 5  # Wait for startup
```

### Test 9: Multiple Services Unhealthy

**Stop web and API:**
```bash
kill $(cat logs/web.pid)
kill $(cat logs/api.pid)
```

**Run health check:**
```bash
../../../azd-app health
```

**Expected Output:** 2 unhealthy (web, api), 3 healthy

**Expected Exit Code:** 1

**Restart services:**
```bash
cd web && nohup npm start > ../logs/web.log 2>&1 &
echo $! > ../logs/web.pid
cd ..
cd api && nohup python app.py > ../logs/api.log 2>&1 &
echo $! > ../logs/api.pid
cd ..
sleep 5
```

### Test 10: All Services Unhealthy

**Stop all services:**
```bash
./stop-all.sh
```

**Run health check:**
```bash
../../../azd-app health
```

**Expected Output:** All 5 services unhealthy

**Expected Exit Code:** 1

**Restart all:**
```bash
./start-all.sh
sleep 30  # Wait for all services
```

### Test 11: Streaming with Service Failure During Monitoring

**Start streaming:**
```bash
../../../azd-app health --stream
```

**While running (in another terminal):**
```bash
cd /home/runner/work/azd-app/azd-app/cli/tests/projects/health-test
kill $(cat logs/api.pid)
```

**Expected Behavior:**
- Next update shows API as unhealthy
- Other services remain healthy
- Status summary updates to show 1 unhealthy

**Restart API:**
```bash
cd api && nohup python app.py > ../logs/api.log 2>&1 &
echo $! > ../logs/api.pid
cd ..
```

**Expected Behavior:**
- Next update (after grace period) shows API as healthy again

**Stop streaming:** Press Ctrl+C

### Test 12: Performance Testing

**Run health check multiple times:**
```bash
time ../../../azd-app health
time ../../../azd-app health
time ../../../azd-app health
```

**Expected Performance:**
- Total execution time: <5 seconds
- Web response: <50ms
- API response: <100ms
- Database port check: <10ms
- Worker process check: <5ms
- Admin response: <30ms

**Manual Verification:**
- Check that parallel execution is working (not sequential)
- All checks should complete in <1 second total

### Test 13: Manual HTTP Endpoint Testing

**Test each endpoint manually:**

```bash
# Web service
curl http://localhost:3000/health
# Expected: {"status":"healthy","service":"web",...}

# API service
curl http://localhost:5000/healthz
# Expected: {"status":"ok","database":"connected",...}

# Admin service (without auth - should fail)
curl http://localhost:4000/api/health
# Expected: 401 Unauthorized

# Admin service (with auth - should succeed)
curl -H "Authorization: Bearer test-token-123" http://localhost:4000/api/health
# Expected: {"status":"healthy","service":"admin","authenticated":true,...}

# Database (TCP port check)
nc -zv localhost 5432
# Expected: Connection to localhost 5432 port [tcp/*] succeeded!
```

### Test 14: Registry Verification

**Check registry file:**
```bash
cat .azure/services.json | jq .
```

**Expected Content:**
- All 5 services listed
- Health status for each service
- PIDs for each service
- URLs for services with ports
- Timestamps

### Test 15: Error Handling

**Test with no services running:**
```bash
./stop-all.sh
../../../azd-app health --verbose
```

**Expected Behavior:**
- All services show unhealthy with detailed error messages
- Exit code: 1
- No crashes or panics

**Test with invalid service filter:**
```bash
./start-all.sh
sleep 30
../../../azd-app health --service nonexistent
```

**Expected Behavior:**
- Shows "no services match filter" or empty result
- No crash
- Exit code: 0 or 2

### Test 16: Concurrent Execution

**Run multiple health checks simultaneously:**
```bash
../../../azd-app health & 
../../../azd-app health &
../../../azd-app health &
wait
```

**Expected Behavior:**
- All complete successfully
- No race conditions
- No registry corruption

### Test 17: Long-Running Streaming Test

**Run streaming for 2 minutes:**
```bash
timeout 120 ../../../azd-app health --stream
```

**Expected Behavior:**
- Continuous updates every 5 seconds
- No memory leaks
- No crashes
- Clean termination after timeout

## Cleanup

**Stop all services:**
```bash
./stop-all.sh
```

**Clean registry:**
```bash
rm -rf .azure
```

**Clean logs:**
```bash
rm -rf logs
```

## Checklist for 100% Confidence

- [ ] All services start successfully
- [ ] Test 1: All services healthy - PASS
- [ ] Test 2: JSON output format - PASS
- [ ] Test 3: Table output format - PASS
- [ ] Test 4: Verbose mode - PASS
- [ ] Test 5: Service filtering - PASS
- [ ] Test 6: Streaming interactive mode - PASS
- [ ] Test 7: Streaming NDJSON (piped) - PASS
- [ ] Test 8: One service unhealthy - PASS
- [ ] Test 9: Multiple services unhealthy - PASS
- [ ] Test 10: All services unhealthy - PASS
- [ ] Test 11: Streaming with service failure - PASS
- [ ] Test 12: Performance testing - PASS
- [ ] Test 13: Manual endpoint testing - PASS
- [ ] Test 14: Registry verification - PASS
- [ ] Test 15: Error handling - PASS
- [ ] Test 16: Concurrent execution - PASS
- [ ] Test 17: Long-running streaming - PASS

## Additional Validation

### Health Check Cascading

**Verify cascading strategy:**

1. **HTTP First:** Web, API, Admin all use HTTP
2. **Port Fallback:** Database uses port check (no HTTP endpoint)
3. **Process Last:** Worker uses process check (no port, no HTTP)

**Test each fallback:**
```bash
# Test HTTP endpoints directly
curl localhost:3000/health    # Should succeed
curl localhost:5000/healthz   # Should succeed

# Test port availability
nc -zv localhost 5432         # Should succeed

# Test process running
ps -p $(cat logs/worker.pid)  # Should show worker process
```

### Exit Codes

**Verify all exit codes:**
- `0`: All healthy (Test 1)
- `1`: One or more unhealthy (Tests 8, 9, 10)
- `2`: Error/invalid input (Test 15 with invalid filter)
- `130`: User interrupted streaming (Test 6)

### Output Formats

**Verify all formats work:**
- Text: Default, human-readable
- JSON: Valid, parseable with jq
- Table: Formatted with borders

### TTY Detection

**Verify automatic format selection:**
```bash
# Interactive (TTY) - should show formatted text
../../../azd-app health --stream

# Piped (non-TTY) - should show NDJSON
../../../azd-app health --stream | cat
```

## Success Criteria for 100% Confidence

1. ✅ All 17 tests pass
2. ✅ No crashes or panics
3. ✅ Performance meets expectations (<5s total)
4. ✅ All output formats work correctly
5. ✅ Exit codes are accurate
6. ✅ Streaming mode works in both TTY and non-TTY
7. ✅ Service filtering works correctly
8. ✅ Graceful shutdown on Ctrl+C
9. ✅ Health check cascading works (HTTP → Port → Process)
10. ✅ Registry is updated correctly
11. ✅ Concurrent execution is safe
12. ✅ Long-running streaming is stable
13. ✅ Error handling is robust
14. ✅ All services work (HTTP, TCP, Process checks)
15. ✅ Authentication works (Admin service)

## Report Issues

If any test fails:
1. Note the specific test number and failure mode
2. Check logs in `logs/` directory
3. Run with `--verbose` flag for more details
4. Document expected vs actual behavior
5. Include error messages and exit codes
