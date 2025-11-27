# Fullstack Test Project - 5 Services

**Test Project**: fullstack-test
**Services**: 5 (api, web, worker, cache, database)
**Purpose**: Realistic multi-service logs dashboard testing

## Services Overview

### 1. api (Python/Flask) - Port 5000
**File**: `api/app.py`
**Type**: HTTP API server
**Logs**: 
- Request handling
- Health checks
- Data queries

**Status**: ✅ Existing (unchanged)

---

### 2. web (Node.js/Express) - Port 5001
**File**: `web/server.js`
**Type**: Frontend web server
**Logs**:
- HTTP requests
- API proxy calls
- Random log generation endpoint

**Status**: ✅ Existing (unchanged)

---

### 3. worker (Python) - Background Process
**File**: `worker/worker.py`
**Type**: Background job processor
**Logs**:
- Job batch processing (2-5 jobs per batch)
- Job completion/failure
- Queue status every 10 batches
- High queue depth warnings every 15 batches

**Patterns**:
```
[INFO] Processing job batch #X (Y jobs)
[INFO] Processing job #X.Y
[INFO] Job #X.Y completed successfully
[WARN] Job #X.Y completed with warnings - retry count exceeded
[ERROR] Job #X.Y failed - unable to process task data
[INFO] Queue status: X pending jobs
[WARN] High queue depth detected - scaling workers recommended
```

**Interval**: 2-5 seconds between batches

**Status**: ✅ New service

---

### 4. cache (Node.js) - Port 6379
**File**: `cache/cache.js`
**Type**: Simulated Redis cache
**Logs**:
- GET/SET/DEL/INCR/EXPIRE/EXISTS operations
- Cache hits/misses
- Memory usage tracking
- Eviction warnings
- Connection errors

**Patterns**:
```
[INFO] GET user:session:X (hit/miss)
[INFO] SET product:X (ttl: Xs)
[INFO] DEL expired_key_X
[INFO] INCR counter:views:X (value: Y)
[INFO] Memory usage: X%
[WARN] Memory usage: X% - eviction triggered
[WARN] Slow command detected: KEYS *
[ERROR] Connection refused from client X - max connections reached
```

**Interval**: 500-1500ms per operation

**Status**: ✅ New service

---

### 5. database (Python) - Port 5432
**File**: `database/database.py`
**Type**: Simulated PostgreSQL database
**Logs**:
- SQL query execution (SELECT/INSERT/UPDATE/DELETE)
- Query duration
- Connection count
- Slow query warnings
- Deadlock errors
- Replication lag
- Auto-vacuum operations

**Patterns**:
```
[INFO] SELECT * FROM users WHERE id=X (duration: Yms)
[INFO] INSERT INTO orders (...) (duration: Yms)
[INFO] UPDATE products SET status=X WHERE id=Y (duration: Zms)
[INFO] DELETE FROM sessions WHERE expires < NOW() (duration: Yms)
[INFO] Active connections: X/100
[WARN] Slow query detected: SELECT * FROM logs (Xms)
[WARN] Missing index on frequently queried column
[WARN] Replication lag: Xms
[INFO] Auto-vacuum running on table: users
[ERROR] Deadlock detected: transaction rolled back
[ERROR] Connection lost to replica database - failover initiated
```

**Interval**: 1.5-3.5 seconds per query

**Status**: ✅ New service

---

## Configuration

### azure.yaml
```yaml
services:
  api:
    language: python
    project: ./api
    ports: ["5000"]
  
  web:
    language: js
    project: ./web
    ports: ["5001"]
  
  worker:
    language: python
    project: ./worker
  
  cache:
    language: js
    project: ./cache
    ports: ["6379"]
  
  database:
    language: python
    project: ./database
    ports: ["5432"]
```

## Running the Test

### Start All Services
```bash
cd cli/tests/projects/fullstack-test
azd app run
```

### Expected Output
All 5 services should start and generate continuous logs:
- **api**: Flask startup, health checks
- **web**: Express server, proxying to API
- **worker**: Job processing every few seconds
- **cache**: Cache operations every 0.5-1.5s
- **database**: SQL queries every 1.5-3.5s

### Dashboard View
Open logs dashboard and verify:
- All 5 services appear in service selector
- Default 2-column grid shows all panes on screen
- No scrolling required to see all services
- Mix of INFO/WARN/ERROR levels across services
- Borders show service status (green/yellow/red)

## Testing Scenarios

### 1. Auto-Fit Layout
- Select all 5 services
- Try 2 columns (3 rows) - all visible
- Try 3 columns (2 rows) - all visible
- Resize window - panes adjust automatically

### 2. Log Patterns
- Watch for errors (red borders + pulse)
- Watch for warnings (yellow borders)
- Verify pattern detection works

### 3. Auto-Scroll Toggle
- Each pane should have auto-scroll button
- Default: auto-scroll ON (primary blue)
- Scroll up → auto-scroll OFF (muted gray)
- Scroll to bottom → auto-scroll ON

### 4. Pause/Resume
- Global pause stops all logs
- Auto-scroll toggle still works
- Resume continues logs from where paused

### 5. Service Filtering
- Uncheck api → only 4 services shown
- Check/uncheck any service
- Layout adjusts automatically

## File Structure

```
fullstack-test/
├── api/
│   ├── app.py (existing)
│   └── requirements.txt
├── web/
│   ├── server.js (existing)
│   └── package.json
├── worker/ (NEW)
│   ├── worker.py
│   └── requirements.txt
├── cache/ (NEW)
│   ├── cache.js
│   └── package.json
├── database/ (NEW)
│   ├── database.py
│   └── requirements.txt
├── azure.yaml (updated)
└── README.md
```

## Dependencies

### Python Services (api, worker, database)
```bash
# api
pip install flask flask-cors

# worker, database
# No dependencies (pure Python)
```

### Node Services (web, cache)
```bash
# web
npm install express

# cache
# No dependencies (pure Node)
```

## Troubleshooting

### Services Not Starting
- Check Python version (>= 3.9)
- Check Node version (>= 18)
- Install dependencies (pip/npm install)

### Logs Not Appearing
- Verify azd dashboard is running
- Check service selector (all checked)
- Check browser console for errors

### Layout Issues
- Clear browser cache
- Check viewport height (min 800px recommended)
- Try different column counts

---

**Status**: ✅ Ready for Testing
**Date**: 2025-11-23
**Test Focus**: Multi-pane auto-fit layout with 5 services
