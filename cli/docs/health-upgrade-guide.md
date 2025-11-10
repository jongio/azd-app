# Health Command Upgrade Guide

## Overview

The `azd app health` command has been upgraded with production-grade features. This guide helps you migrate from basic health checks to the enhanced version.

## What's New?

- **Circuit Breaker** - Prevents cascading failures
- **Rate Limiting** - Protects services from overload
- **Result Caching** - Reduces redundant checks
- **Prometheus Metrics** - Full observability
- **Structured Logging** - JSON/pretty/text formats
- **Health Profiles** - Environment-specific configs

## Quick Start

### 1. Generate Profiles (One-Time Setup)

```bash
azd app health --save-profiles
```

This creates `.azd/health-profiles.yaml` with 4 default profiles: development, production, ci, staging.

### 2. Use a Profile

```bash
# Development (verbose, no caching)
azd app health --profile development

# Production (circuit breaker, metrics, caching)
azd app health --profile production

# CI/CD (long timeouts, many retries)
azd app health --profile ci

# Staging (balanced settings)
azd app health --profile staging
```

## Migration Scenarios

### Scenario 1: Basic Health Check → Development Profile

**Before:**
```bash
azd app health
```

**After:**
```bash
azd app health --profile development
```

**What Changed:**
- Now logs with structured format (pretty colored output)
- Debug logging enabled by default
- No caching (immediate feedback)
- Verbose details shown

**Benefits:**
- Better debugging with structured logs
- Clearer error messages
- More context in output

### Scenario 2: Production Monitoring → Production Profile

**Before:**
```bash
azd app health --stream --interval 30s
```

**After:**
```bash
azd app health --profile production --stream
```

**What Changed:**
- Circuit breaker enabled (prevents hammering failing services)
- Rate limiting (10 checks/sec per service)
- Result caching (5s TTL)
- Prometheus metrics enabled
- JSON logging for log aggregation
- 30s interval (from profile)

**Benefits:**
- Services protected from health check storms
- Metrics for monitoring and alerting
- Reduced load with caching
- Production-ready logging

### Scenario 3: CI/CD Health Gate → CI Profile

**Before:**
```bash
azd app health --timeout 30s --output json
```

**After:**
```bash
azd app health --profile ci --output json
```

**What Changed:**
- Long timeout (30s) from profile
- 5 retries before failure
- No caching (accurate status)
- JSON structured logging

**Benefits:**
- More resilient to slow startup times
- Better failure tolerance
- Accurate status for gates

### Scenario 4: Custom Configuration

**Before:**
```bash
azd app health --timeout 10s --interval 15s --stream
```

**After (Option 1 - Use Profile + Override):**
```bash
# Start with production profile, override timeout
azd app health --profile production --timeout 10s --stream
```

**After (Option 2 - Full Custom):**
```bash
# Manual configuration
azd app health \
  --timeout 10s \
  --interval 15s \
  --circuit-breaker \
  --rate-limit 10 \
  --cache-ttl 5s \
  --metrics \
  --log-level info \
  --log-format json \
  --stream
```

**After (Option 3 - Custom Profile):**

1. Generate profiles:
   ```bash
   azd app health --save-profiles
   ```

2. Edit `.azd/health-profiles.yaml`:
   ```yaml
   profiles:
     my-custom:
       name: my-custom
       interval: 15s
       timeout: 10s
       retries: 3
       circuitBreaker: true
       rateLimit: 10
       cacheTTL: 5s
       metrics: true
       logLevel: info
       logFormat: json
   ```

3. Use custom profile:
   ```bash
   azd app health --profile my-custom --stream
   ```

## Backward Compatibility

All existing commands work without changes:

```bash
# These still work exactly as before
azd app health
azd app health --service web
azd app health --stream --interval 10s
azd app health --output json
azd app health --verbose
```

**Default Behavior:**
- No circuit breaker
- No rate limiting
- No caching
- No metrics
- Simple console logging
- Info log level

**To Opt-In to New Features:**
- Use `--profile` flag
- Or manually enable features with individual flags

## Common Upgrade Patterns

### Pattern 1: Enable Metrics for Existing Checks

```bash
# Before
azd app health --stream

# After
azd app health --stream --metrics
```

Then scrape with Prometheus:
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'azd_health'
    static_configs:
      - targets: ['localhost:9090']
```

### Pattern 2: Add Circuit Breaker to Protect Services

```bash
# Before
azd app health --stream --interval 5s

# After
azd app health --stream --interval 5s --circuit-breaker
```

**Effect:** After 5 consecutive failures, circuit opens and health checks are skipped for 60s.

### Pattern 3: Add Rate Limiting for High-Frequency Checks

```bash
# Before (may overwhelm services)
azd app health --stream --interval 1s

# After (max 10 checks/sec per service)
azd app health --stream --interval 1s --rate-limit 10
```

### Pattern 4: Add Caching to Reduce Load

```bash
# Before (checks every request)
azd app health --cache-ttl 0

# After (cache for 30s)
azd app health --cache-ttl 30s
```

### Pattern 5: Switch to Structured Logging

```bash
# Before (simple console output)
azd app health --stream

# After (JSON for log aggregation)
azd app health --stream --log-format json > health.log
```

## Environment-Specific Recommendations

### Development

**Goal:** Fast feedback, verbose output, no caching

```bash
azd app health --profile development
```

**What You Get:**
- Debug logging with pretty colors
- Verbose output
- No caching (always fresh)
- No metrics overhead
- No circuit breaker (fail fast)

### Production

**Goal:** Reliability, observability, efficiency

```bash
azd app health --profile production --stream
```

**What You Get:**
- Circuit breaker (prevents cascading failures)
- Rate limiting (10/sec per service)
- Caching (5s TTL, reduces load)
- Prometheus metrics (observability)
- JSON logging (log aggregation)
- Info log level (not verbose)

**Add to your infrastructure:**
```bash
# Start health monitoring
nohup azd app health --profile production --stream > /dev/null 2>&1 &

# Configure Prometheus to scrape localhost:9090/metrics
```

### CI/CD

**Goal:** Accurate status, failure tolerance, long timeouts

```bash
azd app health --profile ci --output json
```

**What You Get:**
- 30s timeout (handles slow startup)
- 5 retries (tolerates transient failures)
- No caching (accurate status)
- JSON output (parseable)
- JSON logging (CI log aggregation)

**Pipeline Integration:**
```yaml
# GitHub Actions
- name: Health Check
  run: azd app health --profile ci --output json || exit 1
  
# Azure Pipelines
- script: azd app health --profile ci --output json
  displayName: 'Health Check'
  failOnStderr: true
```

### Staging

**Goal:** Production-like with more debugging

```bash
azd app health --profile staging --stream
```

**What You Get:**
- Circuit breaker enabled
- Rate limiting (20/sec, higher than prod)
- Caching (3s TTL, shorter than prod)
- Metrics enabled
- Debug logging (more verbose than prod)
- JSON format (log aggregation)

## Troubleshooting

### "Health profile not found"

**Error:**
```
Error: health profile 'production' not found
```

**Solution:**
```bash
# Generate sample profiles
azd app health --save-profiles

# Verify file exists
cat .azd/health-profiles.yaml
```

### "Metrics server failed"

**Error:**
```
Warning: Metrics server failed: listen tcp :9090: bind: address already in use
```

**Solution:**
```bash
# Use different port
azd app health --metrics --metrics-port 9091
```

### "Circuit breaker open - service unavailable"

**Status:**
```
Service: web
Status: unhealthy
Error: circuit breaker open - service unavailable
```

**Meaning:** Service has failed repeatedly and circuit breaker is protecting it.

**Solution:**
1. Check service logs for root cause
2. Fix the service issue
3. Wait for circuit breaker timeout (60s by default)
4. Circuit will enter half-open and test service
5. If successful, circuit closes and checks resume

**Manual Reset:**
```bash
# Restart health monitoring to reset circuits
azd app health --stream
```

### "Rate limit exceeded"

**Status:**
```
Service: api
Status: unhealthy
Error: rate limit exceeded
```

**Solutions:**
```bash
# Option 1: Increase rate limit
azd app health --rate-limit 20 --stream

# Option 2: Increase interval
azd app health --rate-limit 10 --interval 15s --stream

# Option 3: Disable rate limiting (development only)
azd app health --rate-limit 0 --stream
```

## Feature Toggle Reference

| Feature | Enable Flag | Disable/Default |
|---------|------------|-----------------|
| Circuit Breaker | `--circuit-breaker` | Off by default |
| Rate Limiting | `--rate-limit 10` | `--rate-limit 0` (off) |
| Caching | `--cache-ttl 30s` | `--cache-ttl 0` (off) |
| Metrics | `--metrics` | Off by default |
| Debug Logging | `--log-level debug` | `--log-level info` |
| JSON Logging | `--log-format json` | `--log-format pretty` |

## Profile Customization

### Example: Custom Production Profile

1. Generate base profiles:
   ```bash
   azd app health --save-profiles
   ```

2. Edit `.azd/health-profiles.yaml`:
   ```yaml
   profiles:
     my-production:
       name: my-production
       interval: 60s          # Check every minute
       timeout: 3s            # Short timeout
       retries: 2             # 2 retries
       circuitBreaker: true
       circuitBreakerFailures: 3   # Open after 3 failures
       circuitBreakerTimeout: 120s # 2 min recovery
       rateLimit: 5           # Conservative rate limit
       verbose: false
       logLevel: warn         # Only warnings and errors
       logFormat: json
       metrics: true
       metricsPort: 9090
       cacheTTL: 10s          # Cache for 10 seconds
   ```

3. Use your profile:
   ```bash
   azd app health --profile my-production --stream
   ```

## Metrics Integration

### Prometheus

**1. Start health monitoring with metrics:**
```bash
azd app health --profile production --stream &
```

**2. Configure Prometheus:**
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'azd_health'
    static_configs:
      - targets: ['localhost:9090']
    scrape_interval: 15s
```

**3. Useful queries:**
```promql
# Health check success rate
sum(rate(azd_health_check_total{status="healthy"}[5m])) by (service) 
/ 
sum(rate(azd_health_check_total[5m])) by (service)

# Average check latency
rate(azd_health_check_duration_seconds_sum[5m]) 
/ 
rate(azd_health_check_duration_seconds_count[5m])

# Circuit breaker open services
azd_circuit_breaker_state == 2

# Error rate by type
sum(rate(azd_health_check_errors_total[5m])) by (error_type)
```

### Grafana

Import dashboard with panels:
- Health check success rate (gauge)
- Service uptime (time series)
- Circuit breaker states (state timeline)
- Error types (pie chart)
- Check latency (histogram)

## FAQ

**Q: Do I need to use profiles?**  
A: No. All existing commands work. Profiles are opt-in for convenience.

**Q: Can I mix profile and individual flags?**  
A: Yes. Individual flags override profile settings.

**Q: Are profiles stored in version control?**  
A: You can, but we recommend adding `.azd/` to `.gitignore` and generating profiles per environment.

**Q: What happens if profile file doesn't exist?**  
A: Default profiles are used if file doesn't exist. Only errors if you explicitly request a profile and file is missing.

**Q: Do I need Prometheus to use metrics?**  
A: No. Metrics endpoint works standalone. Prometheus is optional for scraping and visualization.

**Q: Is there performance overhead?**  
A: Minimal. Circuit breaker check: <1ms. Metrics recording: <1ms. Rate limiting: 0-1000ms backpressure. Caching: <1ms on hit.

**Q: Can I use multiple profiles?**  
A: No, one profile at a time. But you can create custom profiles combining features.

**Q: What about backward compatibility?**  
A: 100% backward compatible. All existing commands work exactly as before.

## Summary

**Easiest Migration:**
```bash
# Development
azd app health --profile development

# Production  
azd app health --profile production --stream

# CI/CD
azd app health --profile ci --output json
```

**For Maximum Control:**
```bash
azd app health \
  --circuit-breaker \
  --rate-limit 10 \
  --cache-ttl 5s \
  --metrics \
  --log-level info \
  --log-format json \
  --stream
```

**Remember:**
- Profiles are opt-in
- Individual flags override profiles
- All features are backward compatible
- Generate sample profiles: `azd app health --save-profiles`
- Metrics endpoint: `http://localhost:9090/metrics`
