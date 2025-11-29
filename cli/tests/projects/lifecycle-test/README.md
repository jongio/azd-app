# Lifecycle Test Project

This test project simulates realistic service lifecycle transitions to test the dashboard's health monitoring and state tracking capabilities.

## Services

### 1. stable-api (port 3001)
**Behavior**: Always healthy
- Consistent health checks
- Reliable baseline service
- Good for comparison with flaky services

### 2. flaky-service (port 3002)
**Behavior**: Cycles between healthy ↔ unhealthy
- **Healthy phase**: 20 seconds
- **Unhealthy phase**: 15 seconds (simulates database connection failure)
- Logs realistic error messages during unhealthy state
- Logs recovery messages when returning to healthy

### 3. slow-starter (port 3003)
**Behavior**: Unhealthy during warmup → healthy after ~25 seconds
- Simulates ML model loading, cache warming, etc.
- Shows warmup progress in logs
- Transitions to healthy once all warmup tasks complete
- Tests `start_period` health check configuration

### 4. degraded-api (port 3004)
**Behavior**: Cycles through performance phases
- **optimal**: 10ms response time
- **normal**: 50ms response time
- **slow**: 200ms response time
- **degraded**: 800ms response time (still healthy but slow)
- **critical**: 2500ms response time (unhealthy - timeout)
- Each phase lasts 15 seconds

## Running the Test

```bash
cd cli/tests/projects/lifecycle-test
azd app run
```

Or watch health in real-time:
```bash
azd app health --watch
```

## Expected Behavior Timeline

| Time | stable-api | flaky-service | slow-starter | degraded-api |
|------|------------|---------------|--------------|--------------|
| 0s   | ✅ healthy | ✅ healthy    | ❌ warmup    | ✅ optimal   |
| 15s  | ✅ healthy | ✅ healthy    | ❌ warmup    | ✅ normal    |
| 20s  | ✅ healthy | ❌ db error   | ❌ warmup    | ✅ normal    |
| 25s  | ✅ healthy | ❌ db error   | ✅ ready     | ✅ slow      |
| 35s  | ✅ healthy | ✅ recovered  | ✅ healthy   | ✅ slow      |
| 40s  | ✅ healthy | ✅ healthy    | ✅ healthy   | ⚠️ degraded |
| 55s  | ✅ healthy | ✅ healthy    | ✅ healthy   | ❌ critical  |
| 70s  | ✅ healthy | ❌ db error   | ✅ healthy   | ✅ optimal   |

## Testing Dashboard State Tracking

This project is designed to verify:

1. **State Transitions**: Dashboard correctly shows healthy → unhealthy → healthy
2. **Recovery Detection**: `hasRecovered()` returns true after services recover
3. **Change History**: `changes` array captures all state transitions
4. **Real-time Updates**: SSE stream delivers events promptly
5. **Log Correlation**: Error/recovery logs appear with correct timing
