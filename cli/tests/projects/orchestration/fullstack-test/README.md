# Fullstack Test Project

Integration test project for validating `azd app run` with multi-service orchestration and explicit port configuration.

## Architecture

```
┌─────────────────┐         ┌─────────────────┐
│   Node.js Web   │ ──────> │  Python API     │
│   Port 3000     │  HTTP   │  Port 8000      │
│   (Express)     │         │  (Flask)        │
└─────────────────┘         └─────────────────┘
```

## Services

### API Service (Python)
- **Language**: Python 3.9+
- **Framework**: Flask 3.0.0
- **Port**: 8000 (explicit)
- **Endpoints**:
  - `GET /` - Welcome message
  - `GET /api/data` - Returns sample data
  - `GET /api/health` - Health check

### Web Service (Node.js)
- **Language**: Node.js 18+
- **Framework**: Express 4.18.2
- **Port**: 3000 (explicit)
- **Features**:
  - HTML interface with API interaction
  - Proxies requests to Python API
  - Shows API health and data

## What This Tests

✅ **Explicit Port Configuration**: Both services have `config.port` set in `azure.yaml`  
✅ **Port Manager**: Validates port assignment with `isExplicit=true` flag  
✅ **Multi-Service Orchestration**: Two services running simultaneously  
✅ **Cross-Service Communication**: Web app calls API via HTTP  
✅ **Environment Variable Injection**: `API_URL`, `PORT`, `FLASK_ENV`, `FLASK_APP`  
✅ **Requirements Validation**: All tools (node, npm, python, pip) checked before running  
✅ **Service Dashboard**: Both services appear in the dashboard  
✅ **Log Streaming**: Both services' output streams to the orchestrator

## Running with azd app

From this directory:

```bash
azd app run
```

Expected behavior:
1. Checks requirements (node 18+, npm 9+, python 3.9+, pip 20+)
2. Installs dependencies for both services
3. Assigns explicit ports: api=8000, web=3000
4. Starts both services
5. Shows dashboard with service statuses
6. Streams logs from both services

## Manual Testing

### 1. Check Requirements
```bash
azd app reqs
```

### 2. Install Dependencies
```bash
azd app deps
```

### 3. Run Services
```bash
azd app run
```

### 4. Test Endpoints

API Service:
```bash
curl http://localhost:8000/
curl http://localhost:8000/api/data
curl http://localhost:8000/api/health
```

Web Service:
- Open browser: http://localhost:3000
- Click "Check API Health"
- Click "Load Data from API"

## Expected Output

Dashboard should show:
```
┌────────┬──────────┬─────────────────────────────────┐
│ Name   │ Status   │ Endpoint                        │
├────────┼──────────┼─────────────────────────────────┤
│ api    │ Running  │ http://localhost:8000           │
│ web    │ Running  │ http://localhost:3000           │
└────────┴──────────┴─────────────────────────────────┘
```

Logs should stream from both services showing:
- API: `Running on http://127.0.0.1:8000`
- Web: `🚀 Web server started on http://localhost:3000`

## Directory Structure

```
fullstack-test/
├── azure.yaml              # Service configuration with explicit ports
├── README.md               # This file
├── api/
│   ├── app.py             # Flask API server
│   ├── requirements.txt   # Python dependencies
│   └── README.md          # API documentation
└── web/
    ├── server.js          # Express web server
    ├── package.json       # Node.js dependencies
    └── README.md          # Web app documentation
```

## Troubleshooting

**Ports already in use?**
- The port manager will detect if 8000 or 3000 are in use and fail with a clear error
- Stop existing processes on those ports before running

**Web can't reach API?**
- Check that `API_URL` is correctly set to `http://localhost:8000`
- Verify both services are running in the dashboard
- Check API logs for errors

**Dependencies not installing?**
- Run `azd app deps` manually to see detailed output
- Verify Python 3.9+ and Node.js 18+ are installed
