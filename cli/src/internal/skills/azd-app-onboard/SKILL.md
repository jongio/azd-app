---
name: azd-app-onboard
description: |
  Onboard any application to use 'azd app' for local multi-service development.
  Guides through project detection, azure.yaml generation, environment setup,
  health checks, hooks, Docker services, and MCP configuration.
  USE FOR: onboard to azd app, setup azd app, configure azure.yaml, get started
  with azd app, add azd app to my project, initialize azd app, new azd app project,
  migrate to azd app, convert project to azd app.
  DO NOT USE FOR: running services (use azd-app skill), Azure deployments (use azd deploy),
  infrastructure provisioning (use azd provision).
---

# Onboarding to azd app

This skill helps you onboard an existing or new application to use `azd app` for
local multi-service development orchestration. It covers the full onboarding journey
from project detection through verified running services.

## When to Use

- Setting up `azd app` in an existing project for the first time
- Creating a new `azure.yaml` from scratch based on detected project structure
- Enriching an existing `azure.yaml` with azd app extensions (ports, commands, health checks)
- Configuring multi-service monorepos
- Setting up environment variables, Key Vault references, Docker services, hooks, or MCP

## Onboarding Workflow

Follow these steps in order. Each step builds on the previous one.

### Step 1: Detect Project Structure

Run `azd app init --dry-run` to discover what azd app detects:

```bash
azd app init --dry-run
```

This scans the project directory for:
- Languages and frameworks (Node.js, Python, .NET, Java, Go, Rust, PHP)
- Package managers (npm, yarn, pnpm, pip, dotnet, maven, gradle, cargo, composer)
- Service boundaries (separate directories with their own entry points)
- Existing configuration (package.json, requirements.txt, go.mod, etc.)
- Docker files (Dockerfile, docker-compose.yml)

### Step 2: Generate azure.yaml

If no `azure.yaml` exists:
```bash
azd app init
```

If `azure.yaml` already exists but needs azd app enrichment:
```bash
azd app init  # enriches without overwriting existing config
```

To overwrite existing services section:
```bash
azd app init --force
```

### Step 3: Configure Services

The generated `azure.yaml` should have a `services` section. Each service needs:

```yaml
name: my-project
services:
  api:
    project: ./api
    language: python
    host: appservice
    command: uvicorn main:app --reload --port {port}
    ports:
      - "8080"
    healthcheck:
      type: http
      path: /health

  web:
    project: ./web
    language: js
    host: appservice
    command: npm run dev -- --port {port}
    ports:
      - "3000"
    healthcheck:
      type: http
      path: /
```

**Key fields to configure per service:**

| Field | Required | Description |
|-------|----------|-------------|
| `project` | Yes | Relative path to service directory |
| `language` | Yes | Language identifier (js, ts, python, csharp, java, go, rust, php, docker) |
| `command` | Recommended | Custom run command (auto-detected if omitted) |
| `ports` | Recommended | List of port strings (e.g., `- "8080"` or `- "8080:8080"`) |
| `healthcheck.type` | Recommended | Health check type: `http`, `tcp`, `process`, or `none` |
| `healthcheck.path` | For HTTP | Health endpoint path (used with `type: http`) |
| `host` | Optional | Azure host target (appservice, containerapp, function) |

### Step 4: Environment Variables

Configure environment variables in `azure.yaml` using three formats:

```yaml
services:
  api:
    environment:
      # Literal values
      NODE_ENV: "development"
      LOG_LEVEL: "debug"

      # Reference other environment variables
      DATABASE_URL: ${POSTGRES_CONNECTION_STRING}

      # Key Vault references (resolved at startup)
      API_KEY: keyvault://my-vault/api-key
      DB_PASSWORD: keyvault://my-vault/db-password
```

For shared environment variables, use an env file:
```bash
azd app run --env-file .env.development
```

### Step 5: Docker Services

For services running in Docker containers, use `host: containerapp` with an `image:` field:

```yaml
services:
  redis:
    host: containerapp
    image: redis:7-alpine
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 3

  postgres:
    host: containerapp
    image: postgres:16-alpine
    ports:
      - "5432:5432"
    environment:
      POSTGRES_DB: myapp
      POSTGRES_USER: dev
      POSTGRES_PASSWORD: devpassword
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U dev"]
      interval: 10s
      timeout: 5s
      retries: 3
```

Use `azd app add` for well-known container services:
```bash
azd app add redis
azd app add postgres
```

### Step 6: Lifecycle Hooks

Configure hooks for custom lifecycle behavior:

```yaml
hooks:
  # Run before services start (e.g., run migrations, seed data)
  prerun:
    shell: bash
    run: |
      echo "Running database migrations..."
      cd api && python -m alembic upgrade head

  # Run after all services are ready
  postrun:
    shell: bash
    run: |
      echo "All services running!"
      echo "API: http://localhost:8080"
      echo "Web: http://localhost:3000"

  # Run before services stop (drain connections, flush caches)
  prestop:
    shell: bash
    run: echo "Draining connections..."

  # Run after all services stopped (cleanup)
  poststop:
    shell: bash
    run: echo "Cleanup complete"
```

### Step 7: Health Checks

Configure health endpoints for service monitoring:

```yaml
services:
  api:
    healthcheck:
      type: http           # http, tcp, process, or none
      path: /health        # HTTP endpoint path (for type: http)
```

For Docker-style health checks (container services):
```yaml
services:
  redis:
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 3
```

Verify health checks work:
```bash
azd app health --all --verbose
```

For continuous monitoring:
```bash
azd app health --stream --interval 10s
```

### Step 8: MCP Server Setup

Enable AI assistant integration via Model Context Protocol:

```bash
# Start MCP server
azd app mcp serve
```

To configure your IDE (VS Code) to use the MCP server, add to `.vscode/mcp.json`:
```json
{
  "servers": {
    "azd-app": {
      "command": "azd",
      "args": ["app", "mcp", "serve"]
    }
  }
}
```

This exposes tools for AI assistants to:
- List and manage services
- View logs and errors
- Install dependencies
- Check requirements
- Manage environment variables

### Step 9: Verify Onboarding

Run the full verification sequence:

```bash
# 1. Check system requirements
azd app reqs

# 2. Install all dependencies
azd app deps

# 3. Dry-run to verify configuration
azd app run --dry-run

# 4. Run all services
azd app run

# 5. Check health (in another terminal)
azd app health --all
```

## Multi-Service Monorepo Setup

For monorepos with multiple independently-deployable services:

```yaml
name: my-monorepo
services:
  auth-service:
    project: ./services/auth
    language: ts
    command: npm run dev -- --port {port}
    ports:
      - "4001"
    healthcheck:
      type: http
      path: /health

  api-gateway:
    project: ./services/gateway
    language: ts
    command: npm run dev -- --port {port}
    ports:
      - "4000"
    healthcheck:
      type: http
      path: /health

  worker:
    project: ./services/worker
    language: python
    command: python -m celery worker
    healthcheck:
      type: process

  frontend:
    project: ./apps/web
    language: ts
    command: npm run dev -- --port {port}
    ports:
      - "3000"
    healthcheck:
      type: http
      path: /
```

Run individual services:
```bash
azd app run --service api-gateway
azd app run --service frontend
```

## Common Issues & Solutions

| Issue | Solution |
|-------|----------|
| Port conflict | Use `{port}` placeholder in command — azd app manages ports automatically |
| Service not detected | Ensure service has a recognizable entry point (package.json, main.py, go.mod, etc.) |
| Health check failing | Verify the endpoint returns HTTP 200 and the path matches `healthcheck.path` |
| Docker service won't start | Check `docker` is running with `azd app reqs` |
| Env var not resolving | Ensure the referenced variable is set in your shell or `.env` file |
| Key Vault access denied | Verify `azd auth login` and that your identity has Key Vault access |

## Complete azure.yaml Reference

```yaml
name: my-project                    # Project name

hooks:                              # Project-level lifecycle hooks
  prerun:
    shell: bash
    run: echo "starting..."
  postrun:
    shell: bash
    run: echo "ready!"
  prestop:
    shell: bash
    run: echo "stopping..."
  poststop:
    shell: bash
    run: echo "stopped"

services:
  service-name:
    project: ./path/to/service      # Relative path to service root
    language: js|ts|python|csharp|java|go|rust|php
    host: appservice|containerapp|function  # Azure host target
    command: "custom run command"   # Use {port} for dynamic port
    ports:                           # List of port strings
      - "8080"                       # Single port
      - "8080:8080"                  # host:container mapping
    healthcheck:                     # Health check configuration
      type: http|tcp|process|none    # Check type
      path: /health                  # Endpoint (for type: http)
      test: ["CMD", "curl", "-f", "http://localhost/health"]  # Docker-style
      interval: 10s                  # Check interval
      timeout: 5s                    # Check timeout
      retries: 3                     # Retry count
    environment:                     # Service environment variables
      KEY: "value"                   # Literal
      KEY: ${OTHER_VAR}              # Env var reference
      KEY: keyvault://vault/secret   # Key Vault reference

  # Container service example
  container-name:
    host: containerapp
    image: redis:7-alpine           # Docker image
    ports:
      - "6379:6379"
    environment:
      KEY: value
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 3
```
