# Azure.yaml Schema Documentation

This document describes the `azd app` extensions to the standard `azure.yaml` configuration file for local development orchestration.

## Schema Reference

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/jongio/azd-app/main/schemas/v1.1/azure.yaml.json
```

## What azd app Adds

`azd app` extends the standard `azd` azure.yaml with local development features:
- **`ports`**: Explicit port mappings (Docker Compose style)
- **`environment`**: Environment variables (Docker Compose compatible formats)
- **`entrypoint`**: Custom entry point files for Python/Node services
- **`reqs`**: Prerequisite tool validation (top-level, not per-service)

All standard `azd` fields remain fully compatible.

## Root Properties

### `name` (required)
Application name (standard `azd` field).

```yaml
name: my-app
```

### `services`
Service definitions. See [Service Object](#service-object) for `azd app` extensions.

### `resources`
Azure resources (standard `azd` field). See [Resource Object](#resource-object).

### `reqs` ⭐ NEW
Prerequisite tools required to run the application.

```yaml
reqs:
  - name: node
    minVersion: "18.0.0"
  - name: docker
    checkRunning: true
```

### `metadata`
Free-form metadata (standard `azd` field).


## Service Object

Defines a service with `azd app` local development extensions.

### Standard azd Fields

- **`host`**: Deployment target (`containerapp`, `appservice`, `function`, etc.)
- **`language`**: Programming language (auto-detected if omitted)
- **`project`**: Relative path to service directory
- **`image`**: Pre-built Docker image name
- **`docker`**: Docker build configuration (see [DockerConfig](#dockerconfig-object))

### azd app Extensions

#### `entrypoint` ⭐ NEW
**Type:** `string` (optional)

Entry point file for Python/Node.js projects with non-standard entry points.

```yaml
services:
  api:
    language: Python
    project: ./backend
    entrypoint: main.py  # Instead of default app.py
```

#### `ports` ⭐ NEW
**Type:** `array` of `string` (optional)

Port mappings in Docker Compose style. Explicit ports are mandatory - if unavailable, users are prompted.

**Formats:**
- `"8080"` - Single port (both host and container)
- `"3000:8080"` - Host port 3000 → container port 8080
- `"127.0.0.1:3000:8080"` - Bind to specific IP
- `"8080/udp"` - UDP protocol

```yaml
services:
  web:
    ports: ["3000"]
  api:
    ports: ["8080"]
  postgres:
    image: postgres:15
    ports: ["5432:5432"]
```

#### `environment` ⭐ NEW
**Type:** `map`, `array` of `string`, or `array` of `object` (optional)

Environment variables in Docker Compose-compatible formats.

```yaml
services:
  api:
    # Map format (simple)
    environment:
      DATABASE_URL: "postgresql://localhost:5432/db"
      LOG_LEVEL: debug
```

```yaml
services:
  api:
    # Array of strings
    environment:
      - DATABASE_URL=postgresql://localhost:5432/db
      - LOG_LEVEL=debug
```

```yaml
services:
  api:
    # Array of objects (with secrets)
    environment:
      - name: DATABASE_URL
        value: "postgresql://localhost:5432/db"
      - name: API_KEY
        secret: MY_SECRET  # Reference to secret
```

#### `uses`
**Type:** `array` of `string` (optional)

Service dependencies - defines startup order.

```yaml
services:
  web:
    uses: ["api"]  # Web waits for API
  api:
    uses: ["database"]  # API waits for database
```



## DockerConfig Object

Standard `azd` Docker build configuration (unchanged by `azd app`).

**Properties:** `path`, `context`, `platform`, `registry`, `image`, `tag`, `buildArgs`, `remoteBuild`

See [azd documentation](https://learn.microsoft.com/azure/developer/azure-developer-cli/azd-schema) for details.

## Resource Object

Standard `azd` resource definition with dependency support.

### Properties

- **`type`** (required): Azure resource type (e.g., `Microsoft.Storage/storageAccounts`)
- **`uses`**: Dependencies on other resources
- **`existing`**: Whether this is an existing resource (not provisioned by azd)

```yaml
resources:
  storage:
    type: Microsoft.Storage/storageAccounts
  webapp:
    type: Microsoft.Web/sites
    uses: ["storage"]
```



## Prerequisite Object

Prerequisite tool requirement.

### Properties

- **`name`** (required): Tool name (e.g., `node`, `python`, `docker`)
- **`minVersion`**: Minimum required version (semver format)
- **`command`**: Override version check command
- **`args`**: Override version check arguments
- **`versionPrefix`**: Override version prefix to strip (e.g., `v`)
- **`versionField`**: Override which output field contains version (0-based)
- **`checkRunning`**: Whether to verify the tool is running (for daemons)
- **`runningCheckCommand`**: Command to check if running
- **`runningCheckArgs`**: Arguments for running check
- **`runningCheckExpected`**: Expected substring in running check output
- **`runningCheckExitCode`**: Expected exit code for running check (default: 0)

```yaml
reqs:
  # Simple version check
  - name: node
    minVersion: "18.0.0"
  
  # Check daemon is running
  - name: docker
    minVersion: "20.0.0"
    checkRunning: true
  
  # Custom tool configuration
  - name: mytool
    minVersion: "1.0.0"
    command: mytool
    args: ["version"]
    versionField: 1
```



## Complete Example

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/jongio/azd-app/main/schemas/v1.1/azure.yaml.json

name: todo-app

reqs:
  - name: node
    minVersion: "18.0.0"
  - name: python
    minVersion: "3.9"
  - name: docker
    checkRunning: true

services:
  web:
    language: TypeScript
    project: ./frontend
    host: staticwebapp
    ports: ["3000"]
    environment:
      API_URL: "http://localhost:8000"
    uses: ["api"]
  
  api:
    language: Python
    project: ./backend
    host: containerapp
    entrypoint: main.py
    ports: ["8000"]
    environment:
      - name: DATABASE_URL
        value: "postgresql://localhost:5432/todos"
      - name: SECRET_KEY
        secret: API_SECRET
    uses: ["database"]
  
  database:
    image: postgres:15-alpine
    ports: ["5432:5432"]
    environment:
      POSTGRES_PASSWORD: localdev
      POSTGRES_DB: todos

resources:
  storage:
    type: Microsoft.Storage/storageAccounts
```

## Port Management

### Port Assignment
1. **Explicit ports** (from `azure.yaml`): Mandatory - user prompted if unavailable
2. **Framework defaults**: Auto-detected from config files if no explicit port
3. **Auto-assignment**: Finds first available port starting from 3000

### Port Conflicts
When an explicit port is in use:
```
⚠️  Service 'api' requires port 8000 (configured in azure.yaml)
This port is currently in use.

Options:
  1) Kill the process using port 8000
  2) Assign a different port automatically
  3) Cancel
```

Ports are persisted in `.azure/ports.json` for consistency.

## Environment Variables

Priority order:
1. Service-specific `environment` (from `azure.yaml`)
2. Azure environment (from `azd env`)
3. System environment

## Service Dependencies

```yaml
services:
  web:
    uses: ["api"]
  api:
    uses: ["database", "cache"]
```

**Startup order:** `database`/`cache` (parallel) → `api` → `web`

## Best Practices

```yaml
# ✅ Explicit ports prevent conflicts
services:
  api:
    ports: ["8000"]

# ✅ Define dependencies for correct startup order
services:
  web:
    uses: ["api"]

# ✅ Validate prerequisites
reqs:
  - name: docker
    checkRunning: true

# ✅ Use relative paths
services:
  web:
    project: ./frontend
```

## See Also

- [azd app CLI Reference](../cli-reference.md)
- [Port Management Design](../design/ports.md)

