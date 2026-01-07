# Health Check Troubleshooting

## Overview

This guide helps you diagnose and fix common health check issues in `azd app`. Health checks determine whether your running services are responding correctly, and when they fail, the diagnostic information helps you understand why.

## Quick Diagnostic Steps

When a service shows as unhealthy:

1. **Check the tooltip** - Hover on the health icon in the dashboard for diagnostic details
2. **Review error message** - Understand what check failed
3. **Check consecutive failures** - Persistent (3+) vs. transient (1-2) issues
4. **View logs** - Run `azd app logs --service <name> --level error`
5. **Verify service is running** - Run `azd app info` to check process status
6. **Check configuration** - Review `azure.yaml` health check settings

## Common Health Check Errors

### HTTP Health Check Errors

#### Error: Connection Refused

**Symptom:**
```
✗ Service Health: Unhealthy

Check: HTTP GET
Endpoint: http://localhost:8080/health
Error: dial tcp [::1]:8080: connect: connection refused
```

**Causes:**
- Service not running
- Service listening on wrong port
- Service not listening on the configured address

**Solutions:**

1. **Check if service is running:**
   ```bash
   azd app info
   # Look for service status
   ```

2. **Verify port configuration:**
   ```bash
   # Check azure.yaml
   cat azure.yaml | grep -A 5 "service-name:"
   # Ensure port matches what service listens on
   ```

3. **Check service logs for startup errors:**
   ```bash
azd app logs --service api --level error --since 5m
   ```

4. **Common port mismatches:**
   ```yaml
   # azure.yaml says:
   ports: ["8080"]
   
   # But app.py has:
   app.run(port=3000)  # ❌ Mismatch
   ```

#### Error: 503 Service Unavailable

**Symptom:**
```
✗ Service Health: Unhealthy

Check: HTTP GET
Endpoint: http://localhost:8080/health
Status: 503 Service Unavailable
Response Time: 45ms

Error Details:
Database connection pool exhausted
```

**Causes:**
- Service dependencies (database, cache) are down
- Service is overloaded
- Connection pools exhausted
- Service in degraded state

**Solutions:**

1. **Check dependencies are running:**
   ```bash
   # Check all service health
   azd app health
   
   # Look for other unhealthy services
   ```

2. **Review service logs:**
   ```bash
   azd app logs --service api --level error --since 10m
   ```

3. **Restart unhealthy dependencies:**
   ```bash
   azd app restart --service database
   ```

4. **Check connection pool settings:**
   ```python
   # Increase pool size if exhausted
   engine = create_engine(
       database_url,
       pool_size=20,  # Increase from default 10
       max_overflow=10
   )
   ```

#### Error: 404 Not Found

**Symptom:**
```
✗ Service Health: Unhealthy

Check: HTTP GET
Endpoint: http://localhost:8080/health
Status: 404 Not Found
```

**Causes:**
- Health endpoint path is incorrect
- Service doesn't implement health endpoint
- Routing misconfiguration

**Solutions:**

1. **Verify health endpoint path:**
   ```bash
   # Test endpoint manually
   curl http://localhost:8080/health
   
   # Try common alternatives
   curl http://localhost:8080/healthz
   curl http://localhost:8080/api/health
   ```

2. **Update azure.yaml with correct path:**
   ```yaml
   services:
     api:
       healthcheck:
         test: "http://localhost:8080/api/health"  # Use correct path
   ```

3. **Implement health endpoint:**
   ```python
   # Flask example
   @app.route('/health')
   def health():
       return {'status': 'healthy'}, 200
   ```

4. **Use TCP check as fallback:**
   ```yaml
   services:
     api:
       healthcheck:
         type: tcp
         port: 8080
   ```

#### Error: Timeout

**Symptom:**
```
✗ Service Health: Unhealthy

Check: HTTP GET
Endpoint: http://localhost:8080/health
Error: health check timed out after 5s
```

**Causes:**
- Service is hanging/deadlocked
- Health check is too slow (blocking operations)
- Network issues
- Timeout setting too short

**Solutions:**

1. **Check if service is responsive:**
   ```bash
   # Test with longer timeout
   curl --max-time 10 http://localhost:8080/health
   ```

2. **Increase timeout in azure.yaml:**
   ```yaml
   services:
     api:
       healthcheck:
         test: "http://localhost:8080/health"
         timeout: 10s  # Increase from default 5s
   ```

3. **Optimize health endpoint (don't include slow checks):**
   ```python
   # ❌ Bad - blocks for seconds
   @app.route('/health')
   def health():
       db.execute("SELECT COUNT(*) FROM users")  # Slow query
       return {'status': 'healthy'}
   
   # ✓ Good - fast check
   @app.route('/health')
   def health():
       db.execute("SELECT 1")  # Quick connectivity check
       return {'status': 'healthy'}
   ```

4. **Check for deadlocks:**
   ```bash
   # Look for thread dumps or stack traces
   azd app logs --service api | grep -i "deadlock\|timeout"
   ```

#### Error: 500 Internal Server Error

**Symptom:**
```
✗ Service Health: Unhealthy

Check: HTTP GET
Endpoint: http://localhost:8080/health
Status: 500 Internal Server Error

Error Details:
AttributeError: 'NoneType' object has no attribute 'execute'
```

**Causes:**
- Bug in health endpoint implementation
- Unhandled exceptions
- Missing dependencies

**Solutions:**

1. **Check service logs for stack trace:**
   ```bash
   azd app logs --service api --level error --since 5m
   ```

2. **Add error handling to health endpoint:**
   ```python
   @app.route('/health')
   def health():
       try:
           # Health checks here
           db.execute("SELECT 1")
           return {'status': 'healthy'}, 200
       except Exception as e:
           return {
               'status': 'unhealthy',
               'error': str(e)
           }, 503  # Don't return 500
   ```

3. **Test health endpoint in isolation:**
   ```bash
   curl -v http://localhost:8080/health
   ```

### TCP Health Check Errors

#### Error: Connection Refused (TCP)

**Symptom:**
```
✗ Service Health: Unhealthy

Check: TCP
Port: 5432
Error: connection refused
```

**Solutions:**

1. **Verify service is listening on port:**
   ```powershell
   # Windows
   netstat -ano | findstr :5432
   
   # Shows which process is listening
   ```

2. **Check service configuration:**
   ```bash
   # PostgreSQL example - verify port in config
   cat postgresql.conf | grep port
   ```

3. **Restart service:**
   ```bash
   azd app restart --service database
   ```

#### Error: Port Already in Use

**Symptom:**
```
Service failed to start
Error: bind: address already in use
```

**Solutions:**

1. **Find process using the port:**
   ```powershell
   # Windows
   netstat -ano | findstr :8080
   # Note the PID, then:
   tasklist | findstr <PID>
   ```

2. **Kill conflicting process or change port:**
   ```yaml
   # Change port in azure.yaml
   services:
     api:
       ports: ["8081"]  # Use different port
   ```

### Process Health Check Errors

#### Error: Process Not Running

**Symptom:**
```
✗ Service Health: Unhealthy

Check: Process
PID: 12345
Error: process not found
```

**Solutions:**

1. **Check if process crashed:**
   ```bash
   azd app logs --service worker --since 10m
   ```

2. **Look for exit code in logs:**
   ```bash
   azd app info
   # Check exit code if shown
   ```

3. **Restart service:**
   ```bash
   azd app restart --service worker
   ```

4. **Fix startup issues:**
   ```bash
   # Run command manually to see errors
   cd services/worker
   python worker.py
   ```

## Interpreting Diagnostic Reports

### Sample Diagnostic Report

```markdown
# Service Health Diagnostic Report
**Service**: api
**Status**: unhealthy
**Timestamp**: 2025-12-29T10:30:45Z

## Health Check
- **Type**: HTTP GET
- **Endpoint**: http://localhost:8080/health
- **Status Code**: 503 Service Unavailable
- **Response Time**: 45ms
- **Consecutive Failures**: 3

## Error
Database connection failed: timeout after 5s

## Service Info
- **Uptime**: 15m 47s
- **PID**: 12345
- **Port**: 8080

## Suggested Actions
1. Check service logs: `azd app logs --service api`
2. Verify database is running
3. Check network connectivity
4. Review connection pool settings
```

### Key Fields Explained

| Field | What It Tells You | How to Use It |
|-------|-------------------|---------------|
| **Status Code** | HTTP response from health endpoint | 2xx = healthy, 4xx/5xx = unhealthy |
| **Response Time** | How long the check took | >1000ms indicates performance issues |
| **Consecutive Failures** | How many times it's failed in a row | 1-2 = transient, 3+ = persistent problem |
| **Uptime** | How long service has been running | Recent start = startup issue, long uptime = degradation |
| **Error** | Primary error message | Tells you what failed |
| **Error Details** | Extended information from service | Root cause from service internals |

### Consecutive Failures Guide

| Count | Interpretation | Action |
|-------|----------------|--------|
| 1 | Transient failure, likely temporary | Wait and monitor |
| 2 | Possible issue developing | Check logs |
| 3+ | Persistent problem | Immediate investigation needed |
| 5+ | Critical issue | Restart service or dependencies |

## Suggested Actions Reference

### By HTTP Status Code

| Code | Suggested Actions |
|------|------------------|
| **503** | • Check if dependencies are running<br>• Verify database/cache/queue connectivity<br>• Review connection pool settings<br>• Check for resource exhaustion |
| **500-502** | • Check service logs for errors<br>• Review recent code changes<br>• Look for unhandled exceptions<br>• Check stack traces |
| **404** | • Verify health endpoint path<br>• Check routing configuration<br>• Review health check config in azure.yaml<br>• Implement health endpoint if missing |
| **401/403** | • Check authentication configuration<br>• Verify API keys/tokens<br>• Review security settings<br>• Check if auth required for /health |
| **429** | • Reduce health check frequency<br>• Check rate limiting settings<br>• Review concurrent request limits |
| **Timeout** | • Increase timeout setting<br>• Optimize health endpoint<br>• Check for network issues<br>• Look for deadlocks/hangs |

### By Error Pattern

| Error Pattern | Suggested Actions |
|---------------|------------------|
| **Connection refused** | • Verify service is running<br>• Check port configuration<br>• Ensure service started successfully<br>• Review firewall settings |
| **Connection timeout** | • Check network connectivity<br>• Verify firewall rules<br>• Increase timeout setting<br>• Look for service hangs |
| **DNS failure** | • Check hostname resolution<br>• Verify network configuration<br>• Use IP address instead of hostname |
| **SSL/TLS error** | • Verify certificate validity<br>• Check SSL configuration<br>• Review protocol versions |
| **Process not found** | • Check if process crashed<br>• Review service logs<br>• Verify start command<br>• Check for startup errors |

## Health Check Configuration Examples

### Basic HTTP Health Check

```yaml
services:
  api:
    language: python
    project: ./api
    ports: ["8080"]
    healthcheck:
      test: "http://localhost:8080/health"
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 30s
```

### Custom Health Endpoint Path

```yaml
services:
  api:
    healthcheck:
      test: "http://localhost:8080/api/v1/health"
```

### TCP Port Check (No HTTP Endpoint)

```yaml
services:
  database:
    ports: ["5432"]
    healthcheck:
      type: tcp
      port: 5432
```

### Process Check (Background Worker)

```yaml
services:
  worker:
    healthcheck:
      type: process
      # No HTTP endpoint, just checks if process is running
```

### Disabled Health Check (Build/Watch Services)

```yaml
services:
  tsc-watch:
    healthcheck: false  # Skip health checks
```

### Advanced HTTP Health Check with Custom Command

```yaml
services:
  api:
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 5s
      timeout: 3s
      retries: 5
      start_period: 60s
```

## FAQ

### Q: Why does my service show as unhealthy even though it's running?

**A:** Running and healthy are different states:
- **Running** means the process is active (lifecycle state)
- **Healthy** means the service is responding correctly (health status)

A running service can be unhealthy if:
- Health endpoint returns non-2xx status
- Health check times out
- Service is running but not responding to requests

**Check:**
```bash
# See both running status and health
azd app info

# Test health endpoint manually
curl http://localhost:8080/health
```

### Q: Health checks work sometimes but fail intermittently. Why?

**A:** Common causes of intermittent failures:
- Service is under load (slow responses)
- Health endpoint includes flaky checks (external API calls)
- Network issues
- Resource constraints (CPU/memory)

**Solutions:**
1. Increase timeout: `timeout: 10s`
2. Increase retries: `retries: 3`
3. Simplify health endpoint (remove external dependencies)
4. Check system resources

### Q: Can I disable health checks for a service?

**A:** Yes, several ways:

```yaml
# Option 1: Boolean false
services:
  build-service:
    healthcheck: false

# Option 2: Explicit disable
services:
  build-service:
    healthcheck:
      disable: true

# Option 3: Type none
services:
  build-service:
    healthcheck:
      type: none
```

### Q: What's a good health endpoint implementation?

**A:** A good health endpoint should be:
- **Fast** (< 100ms response time)
- **Lightweight** (no heavy computation)
- **Reliable** (minimal external dependencies)
- **Informative** (return status of critical components)

**Example:**
```python
@app.route('/health')
def health():
    try:
        # Quick database connectivity check
        db.execute("SELECT 1")
        cache.ping()
        
        return {
            'status': 'healthy',
            'checks': {
                'database': 'healthy',
                'cache': 'healthy'
            }
        }, 200
    except DatabaseError as e:
        return {
            'status': 'unhealthy',
            'error': 'Database connection failed',
            'details': str(e)
        }, 503
```

### Q: How do I add suggested actions to my health endpoint?

**A:** Return a `suggestion` field in your health response:

```python
return {
    'status': 'unhealthy',
    'error': 'Database connection failed',
    'suggestion': 'Check database service and connection pool settings'
}, 503
```

This appears in the dashboard tooltip under "Suggested Actions".

### Q: What's the difference between `start_period` and `interval`?

**A:**
- **`interval`**: Time between health checks during normal operation (default: 10s)
- **`start_period`**: Grace period after startup before marking unhealthy (default: 0s)

**Example:**
```yaml
healthcheck:
  interval: 10s      # Check every 10s
  start_period: 30s  # Don't mark unhealthy for first 30s
```

This gives services time to initialize before health checks count as failures.

### Q: Can I see health check history?

**A:** Currently, consecutive failures are tracked. Full history tracking is planned for a future release.

**Workaround:**
```bash
# Follow health in streaming mode and log to file
azd app health --stream --format json > health.log

# Later, analyze:
cat health.log | jq 'select(.services[].status == "unhealthy")'
```

### Q: How do I test health checks before deploying?

**A:**
1. **Manual curl test:**
   ```bash
   curl -v http://localhost:8080/health
   ```

2. **Run health command:**
   ```bash
   azd app health --service api --verbose
   ```

3. **Check dashboard tooltip** - hover on health icon

4. **Review configuration:**
   ```bash
   cat azure.yaml | grep -A 10 "healthcheck:"
   ```

### Q: My health endpoint returns 200 but shows unhealthy. Why?

**A:** Check the response body. If your endpoint returns JSON with `"status": "unhealthy"`, the service is marked unhealthy even with HTTP 200.

**Fix:**
```python
# Return appropriate HTTP status code
if not_healthy:
    return {'status': 'unhealthy'}, 503  # Use 503, not 200
```

## Related Documentation

- [Health Check Command](../commands/health.md) - Full command reference
- [Service States and Health](../features/service-states.md) - Understanding states
- [azure.yaml Health Configuration](../schema/azure.yaml.md#healthcheck-new) - Configuration options

## Getting Help

If you're still experiencing issues:

1. **Copy diagnostic report** from dashboard tooltip
2. **Collect logs**: `azd app logs --service <name> --since 30m --file issue-logs.txt`
3. **Share configuration**: Include relevant `azure.yaml` excerpt
4. **Describe symptoms**: What you're seeing vs. what you expect
5. **File an issue**: [github.com/jongio/azd-app/issues](https://github.com/jongio/azd-app/issues)

## Notes About Running Tests on Windows

Tests were updated to bind test listeners to the loopback interface (`127.0.0.1`) by default to avoid triggering Windows Firewall prompts when tests bind to all interfaces (e.g. `:0`). If you need to run tests that intentionally exercise all-interface binds, run them on a non-Windows system or explicitly enable them and accept firewall prompts. Tests that require all-interface behavior are annotated and will be skipped on Windows.

To run integration tests that may bind all interfaces on Unix-like systems:

```bash
# Run integration tests only
go test ./cli/... -run Integration -v
```
