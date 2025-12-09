# Container Services and `azd app add` Command

## Overview

This spec covers two related features:

1. **Container Services**: Enable `azd app run` to launch Docker containers defined as services in `azure.yaml`
2. **`azd app add` Command**: A new command to easily add well-known emulators and services to `azure.yaml`

## Problem Statement

**Container Services**: Currently `azd app run` orchestrates local development by running processes directly. Many Azure development scenarios require local emulators (Azurite for Azure Storage, Cosmos DB emulator) that run as Docker containers. Users must manually manage these containers outside of `azd app run`.

**Add Command**: Adding emulators to `azure.yaml` requires knowledge of correct Docker images, ports, environment variables, and connection strings. This is error-prone and time-consuming.

## Goals

1. Enable `azd app run` to launch and manage Docker containers alongside native processes
2. Use Docker Compose-compatible schema for container services (no new fields)
3. Provide `azd app add` command to easily add well-known Azure emulators
4. Auto-configure connection strings in dependent services

## Non-Goals

- Full Docker Compose feature parity (networks, volumes beyond simple mounts, etc.)
- Container building during `azd app run` (use pre-built images only)
- Kubernetes/Podman support (future consideration)
- Cloud deployment of container services

---

## Feature 1: Container Services in `azure.yaml`

### Schema Design

Container services use the **existing** schema fields, mapping to Docker Compose semantics:

```yaml
services:
  # Container service - identified by presence of 'image' field
  azurite:
    image: mcr.microsoft.com/azure-storage/azurite
    ports:
      - "10000:10000"  # Blob service
      - "10001:10001"  # Queue service
      - "10002:10002"  # Table service
    environment:
      AZURITE_ACCOUNTS: "devstoreaccount1:Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="
    healthcheck:
      type: tcp  # Container health check

  # Native process service
  api:
    language: python
    project: ./api
    ports: ["8000"]
    uses: ["azurite"]  # Dependency on container service
```

### Service Detection Logic

A service is a **container service** when:
1. Has `image` field set (direct image reference)
2. OR has `docker.image` field set (Docker config with image)

A service is a **native process service** when:
1. Has `project` field set (project directory)
2. AND does not have `image` or `docker.image`

### Container Lifecycle

**Startup:**
1. Check Docker is installed and running (add to `reqs` validation)
2. Pull image if not present locally (with progress indicator)
3. Create and start container with configured ports/environment
4. Wait for health check to pass (TCP port check for containers)

**Shutdown:**
1. Send SIGTERM to container
2. Wait up to 10 seconds for graceful shutdown
3. Force kill if timeout exceeded
4. Remove container (containers are ephemeral for dev)

### Port Mapping

Port syntax follows Docker Compose exactly:

| Format | Description |
|--------|-------------|
| `"10000"` | Container port 10000, auto-assign host port |
| `"10000:10000"` | Host port 10000 → container port 10000 |
| `"127.0.0.1:10000:10000"` | Bind to localhost only |

### Environment Variables

Use existing `environment` field with Docker Compose-compatible formats:

```yaml
environment:
  KEY: value                    # Map format
  
environment:
  - KEY=value                   # Array of strings
  
environment:
  - name: KEY
    value: value                # Array of objects
```

### Health Checks for Containers

Container services default to `type: tcp` health check (port connectivity).

```yaml
healthcheck:
  type: tcp          # Just check port is listening (default for containers)
  
healthcheck:
  type: http         # HTTP endpoint check
  test: "http://localhost:10000/"
  
healthcheck:
  test: ["CMD", "redis-cli", "ping"]  # Docker-style command check
```

### Dashboard Integration

Container services appear in the dashboard with:
- Container icon indicator
- Image name displayed
- Port mappings shown
- Health status (TCP check)
- Logs streamed from container stdout/stderr

---

## Feature 2: `azd app add` Command

### Command Syntax

```
azd app add <service-type> [options]
```

### Supported Service Types (v1)

| Type | Image | Ports | Description |
|------|-------|-------|-------------|
| `azurite` | `mcr.microsoft.com/azure-storage/azurite` | 10000, 10001, 10002 | Azure Storage emulator |
| `cosmos` | `mcr.microsoft.com/cosmosdb/linux/azure-cosmos-emulator:vnext-preview` | 8081, 1234 | Cosmos DB emulator |
| `redis` | `redis:7-alpine` | 6379 | Redis cache |
| `postgres` | `postgres:16-alpine` | 5432 | PostgreSQL database |

### Future Service Types

| Type | Image | Ports | Description |
|------|-------|-------|-------------|
| `mongodb` | `mongo:7` | 27017 | MongoDB database |
| `sqlserver` | `mcr.microsoft.com/mssql/server:2022-latest` | 1433 | SQL Server |
| `rabbitmq` | `rabbitmq:3-management-alpine` | 5672, 15672 | RabbitMQ message broker |
| `servicebus` | TBD | TBD | Service Bus emulator (when available) |

### Command Options

```
Flags:
  -n, --name string     Service name in azure.yaml (default: service-type)
  -p, --port strings    Override default ports (can specify multiple)
  --no-env              Don't add connection string environment variables
  --dry-run             Show what would be added without modifying files
  -h, --help            Help for add
```

### Workflow

1. **Validate** `azure.yaml` exists
2. **Check** service name doesn't already exist
3. **Generate** service configuration from template
4. **Prompt** for customization (interactive mode) or use defaults
5. **Update** `azure.yaml` with new service
6. **Display** connection string info for dependent services

### Example: Adding Azurite

```bash
$ azd app add azurite

Adding Azure Storage emulator (Azurite)...

✓ Added service 'azurite' to azure.yaml

Service Configuration:
  Image:  mcr.microsoft.com/azure-storage/azurite
  Ports:  10000 (Blob), 10001 (Queue), 10002 (Table)

Connection Strings for your services:
  AZURE_STORAGE_CONNECTION_STRING=DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10000/devstoreaccount1;QueueEndpoint=http://127.0.0.1:10001/devstoreaccount1;TableEndpoint=http://127.0.0.1:10002/devstoreaccount1;

Add to your service's environment:
  environment:
    AZURE_STORAGE_CONNECTION_STRING: "DefaultEndpointsProtocol=http;..."

Run 'azd app run' to start all services including Azurite.
```

### Example: Adding Cosmos DB Emulator

```bash
$ azd app add cosmos --name cosmosdb

Adding Azure Cosmos DB emulator...

✓ Added service 'cosmosdb' to azure.yaml

Service Configuration:
  Image:  mcr.microsoft.com/cosmosdb/linux/azure-cosmos-emulator:vnext-preview
  Ports:  8081 (API), 1234 (Data Explorer)

Connection String:
  COSMOS_CONNECTION_STRING=AccountEndpoint=https://localhost:8081/;AccountKey=C2y6yDjf5/R+ob0N8A7Cgv30VRDJIWEHLM+4QDU5DE2nQ9nDuVTqobD4b8mGGyPMbIZnqyMsEcaGQy67XIw/Jw==

Data Explorer:  http://localhost:1234

Note: For Java/.NET SDKs, HTTPS mode is required. Certificate setup may be needed.
```

### Generated YAML Examples

**Azurite:**
```yaml
services:
  azurite:
    image: mcr.microsoft.com/azure-storage/azurite
    ports:
      - "10000:10000"
      - "10001:10001"
      - "10002:10002"
    healthcheck:
      type: tcp
```

**Cosmos DB:**
```yaml
services:
  cosmos:
    image: mcr.microsoft.com/cosmosdb/linux/azure-cosmos-emulator:vnext-preview
    ports:
      - "8081:8081"
      - "1234:1234"
    healthcheck:
      type: tcp
```

**PostgreSQL:**
```yaml
services:
  postgres:
    image: postgres:16-alpine
    ports:
      - "5432:5432"
    environment:
      POSTGRES_PASSWORD: localdev
      POSTGRES_DB: app
    healthcheck:
      type: tcp
```

---

## Implementation Details

### Docker Integration

New package: `cli/src/internal/docker/`

```go
// docker/client.go
type Client interface {
    IsAvailable() bool
    Pull(image string) error
    Run(config ContainerConfig) (*Container, error)
    Stop(containerID string, timeout time.Duration) error
    Remove(containerID string) error
    Logs(containerID string) (io.ReadCloser, error)
    Wait(containerID string) (<-chan int, error)
}

type ContainerConfig struct {
    Name        string
    Image       string
    Ports       []PortMapping
    Environment map[string]string
    Volumes     []VolumeMount  // Future
}
```

### Service Type Detection

Update `cli/src/internal/service/detector.go`:

```go
func IsContainerService(svc Service) bool {
    return svc.Image != "" || (svc.Docker != nil && svc.Docker.Image != "")
}

func DetectServiceRuntime(name string, svc Service, ...) (*ServiceRuntime, error) {
    if IsContainerService(svc) {
        return detectContainerRuntime(name, svc, ...)
    }
    return detectProcessRuntime(name, svc, ...)
}
```

### Orchestration Changes

Update `cli/src/internal/service/orchestrator.go`:

1. Separate container and process services
2. Start containers first (dependencies typically)
3. Stream container logs to unified log system
4. Handle container health checks
5. Clean up containers on shutdown

### Well-Known Services Registry

New file: `cli/src/internal/wellknown/services.go`

```go
type WellKnownService struct {
    Name            string
    Description     string
    Image           string
    DefaultPorts    []PortSpec
    Environment     map[string]string
    HealthcheckType string
    ConnectionInfo  ConnectionInfo
}

type ConnectionInfo struct {
    EnvVarName       string
    ConnectionString string
    Notes            string
}

var Services = map[string]WellKnownService{
    "azurite": {
        Name:        "azurite",
        Description: "Azure Storage Emulator",
        Image:       "mcr.microsoft.com/azure-storage/azurite",
        DefaultPorts: []PortSpec{
            {Host: 10000, Container: 10000, Description: "Blob service"},
            {Host: 10001, Container: 10001, Description: "Queue service"},
            {Host: 10002, Container: 10002, Description: "Table service"},
        },
        HealthcheckType: "tcp",
        ConnectionInfo: ConnectionInfo{
            EnvVarName: "AZURE_STORAGE_CONNECTION_STRING",
            ConnectionString: "DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;...",
        },
    },
    // ... other services
}
```

---

## Design Decisions

### Container Volumes
**Decision**: No volume support in v1. Containers are ephemeral - data is lost on restart.

**Rationale**: Keeps implementation simple. Local dev typically doesn't need persistent emulator data. Bind mounts can be added as a fast-follow if needed.

### Container Networks
**Decision**: Use host network mode. All containers bind to localhost.

**Rationale**: Simplest approach, no DNS configuration needed, services accessible via `localhost:port`.

### Image Building
**Decision**: Out of scope for v1. Only pre-built images supported.

**Rationale**: `docker.path` exists in schema for future use. Focus on emulator use case first.

### Command Naming
**Decision**: `azd app add <service-type>`

**Rationale**: Shortest syntax, intuitive, extensible for future additions beyond emulators.

### Connection String Handling
**Decision**: Display connection string info after adding. Do not auto-inject into other services.

**Rationale**: Explicit is better than implicit. Users copy what they need. Avoids unexpected file modifications.

### Service Aliases
**Decision**: Single canonical name per service. No aliases.

**Rationale**: Simpler documentation, less confusion, easier to maintain.

### v1 Service Scope
**Decision**: Four services for v1: `azurite`, `cosmos`, `redis`, `postgres`

**Rationale**: Covers most common Azure dev scenarios (storage, database, cache). Others added based on demand.

---

## Success Criteria

### Container Services

1. Container services start when `azd app run` is executed
2. Container services appear in dashboard with health status
3. Logs from containers are streamed to unified log view
4. Containers are stopped/removed on shutdown
5. Dependencies between containers and processes work correctly

### `azd app add` Command

1. `azd app add azurite` adds working Azurite service to azure.yaml
2. `azd app add cosmos` adds working Cosmos DB emulator
3. `azd app run` successfully starts added emulator
4. Connection string info is displayed for easy integration
5. Command is idempotent (warns if service already exists)

---

## Future Considerations

- Volume persistence across runs
- Docker Compose file import
- Custom network configuration
- Container building from Dockerfile
- Podman support (drop-in Docker replacement)
- Service Bus emulator when available
- Event Hubs emulator
- SQL Edge for ARM support
