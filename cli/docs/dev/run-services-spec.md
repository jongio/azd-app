# Run Services Feature Specification

## Overview

Enhance the `azd app run` command to support running multiple services defined in `azure.yaml`. The command will automatically detect each service's language, framework, and configuration, start all services concurrently, and display their URLs with live log streaming.

## User Story

**As a developer**, I want to run all services defined in my `azure.yaml` file with a single command, so I can quickly start my entire multi-service application locally for development and testing without manually starting each service individually.

## Command Syntax

```bash
# Run all services defined in azure.yaml
azd app run

# Run specific services only
azd app run --service web --service api
azd app run -s web -s api

# Run services with environment variables from .env file
azd app run --env-file .env.local

# Run with verbose output (show detailed logs)
azd app run --verbose
azd app run -v

# Dry-run mode: show what would be executed without starting services
azd app run --dry-run
```

## Azure.yaml Schema Integration

The command uses the `services` section of `azure.yaml` as defined in the [official schema](https://raw.githubusercontent.com/Azure/azure-dev/main/schemas/v1.0/azure.yaml.json).

### Relevant Service Properties

| Property | Type | Required | Description | Used By Run Command |
|----------|------|----------|-------------|---------------------|
| `host` | string | Yes | Azure resource type (appservice, containerapp, function, staticwebapp, aks, ai.endpoint) | Framework detection hint |
| `language` | string | No | Implementation language (dotnet, csharp, fsharp, py, python, js, ts, java, docker) | Language/runtime detection |
| `project` | string | No | Path to service source directory | Service location |
| `docker` | object | No | Docker build configuration | Container-based services |
| `config` | object | No | Service-specific configuration | Port, environment vars |
| `env` | array | No | Environment variables (name, value, secret) | Runtime environment |
| `uses` | array | No | Dependencies on other services/resources | Startup order |

### Example azure.yaml

```yaml
name: my-app

services:
  web:
    host: staticwebapp
    language: ts
    project: ./frontend
    
  api:
    host: appservice
    language: python
    project: ./backend
    config:
      port: 8000
    env:
      - name: DATABASE_URL
        value: ${DATABASE_URL}
    uses:
      - db
      
  worker:
    host: containerapp
    language: dotnet
    project: ./worker
    docker:
      path: ./Dockerfile
      context: .
    uses:
      - api
      
  apphost:
    host: containerapp
    language: dotnet
    project: ./AppHost
    # Aspire AppHost project

resources:
  db:
    type: db.postgres
```

## Detection & Execution Strategy

### 1. Service Discovery

The command reads `azure.yaml` and extracts all services defined in the `services` section:

1. **Find azure.yaml**: Use existing `FindAzureYaml()` from detector package
2. **Parse services**: Extract service definitions with their properties
3. **Resolve paths**: Convert relative paths in `project` to absolute paths
4. **Filter services**: If `--service` flag used, only process specified services

### 2. Language & Framework Detection

For each service, detect the appropriate runtime and framework using a priority-based detection strategy:

#### Detection Priority

1. **Explicit `language` property** in azure.yaml
2. **Host type** (`host` property) provides hints
3. **File-based detection** in the `project` directory
4. **Fallback to generic detection** from existing detector package

#### Detection Matrix by Language

| Language | Detection Criteria | Package Manager | Run Command | Port Detection |
|----------|-------------------|-----------------|-------------|----------------|
| **Node.js** | package.json | pnpm > yarn > npm | `{pm} run dev` or `{pm} start` | package.json scripts or default 3000 |
| **TypeScript** | tsconfig.json + package.json | pnpm > yarn > npm | `{pm} run dev` or `{pm} start` | package.json scripts or default 3000 |
| **Python** | requirements.txt, pyproject.toml | uv > poetry > pip | `{pm} run {entry}` or `python {entry}` | config.port or default 8000 |
| **.NET** | *.csproj, *.sln | dotnet | `dotnet run --project {proj}` | launchSettings.json or default 5000 |
| **.NET Aspire** | AppHost.cs | dotnet | `dotnet run --project {apphost}` | Aspire dashboard (default 15888) |
| **Java** | pom.xml, build.gradle | maven, gradle | `mvn spring-boot:run` or `gradle bootRun` | application.properties or default 8080 |
| **Go** | go.mod | go | `go run .` or `go run main.go` | Code analysis or default 8080 |
| **Docker** | Dockerfile or docker-compose.yml | docker | `docker compose up` | docker-compose.yml ports or Dockerfile EXPOSE |
| **Rust** | Cargo.toml | cargo | `cargo run` | Code analysis or default 8000 |
| **PHP** | composer.json | composer | `php artisan serve` (Laravel) or `php -S localhost:8000` | Framework-specific or default 8000 |

#### Framework-Specific Detection

Within each language, detect specific frameworks for optimized run commands:

**Node.js/TypeScript Frameworks:**
- Next.js: `package.json` contains "next" → `{pm} run dev` (port 3000)
- React: CRA, Vite → `{pm} run dev` or `{pm} run start`
- Vue: Vite, Vue CLI → `{pm} run dev`
- Angular: `angular.json` → `ng serve` (port 4200)
- Express: Detect in package.json or main file
- NestJS: `nest-cli.json` → `{pm} run start:dev`
- Nuxt: `nuxt.config.ts` → `{pm} run dev`
- SvelteKit: `svelte.config.js` → `{pm} run dev`
- Remix: `remix.config.js` → `{pm} run dev`
- Astro: `astro.config.mjs` → `{pm} run dev`

**Python Frameworks:**
- Django: `manage.py` → `python manage.py runserver` (port 8000)
- Flask: Detect from imports → `flask run` (port 5000)
- FastAPI: Detect from imports → `uvicorn {module}:app --reload` (port 8000)
- Streamlit: Detect from imports → `streamlit run {entry}` (port 8501)
- Gradio: Detect from imports → `python {entry}` (auto-assigns port)

**.NET Frameworks:**
- ASP.NET Core: *.csproj with SDK="Microsoft.NET.Sdk.Web"
- Blazor: Detect from project SDK and dependencies
- Aspire: AppHost.cs or Aspire.Hosting package reference

**Java Frameworks:**
- Spring Boot: pom.xml or build.gradle with spring-boot
- Quarkus: pom.xml with quarkus
- Micronaut: Detect from dependencies

### 3. Port Detection Strategy

Ports are detected in the following priority order:

#### Priority 1: azure.yaml Configuration
```yaml
services:
  api:
    config:
      port: 8000  # Explicit port in azure.yaml
```

#### Priority 2: Framework Configuration Files

| Framework | Configuration File | Port Location |
|-----------|-------------------|---------------|
| Node.js | package.json | `scripts.dev` or `scripts.start` with --port |
| .NET | Properties/launchSettings.json | `profiles.{profile}.applicationUrl` |
| Python/Django | settings.py | `PORT` or default 8000 |
| Python/FastAPI | Code analysis | `uvicorn.run()` port parameter |
| Java/Spring | application.properties or application.yml | `server.port` |
| Docker | docker-compose.yml | `ports` mapping |
| Docker | Dockerfile | `EXPOSE` directive |
| Angular | angular.json | `projects.{name}.architect.serve.options.port` |

#### Priority 3: Environment Variables
- PORT
- HTTP_PORT
- WEB_PORT
- SERVICE_PORT
- {SERVICE_NAME}_PORT

#### Priority 4: Default Ports by Framework

| Framework | Default Port | Protocol |
|-----------|--------------|----------|
| Next.js | 3000 | HTTP |
| React (Vite/CRA) | 3000/5173 | HTTP |
| Angular | 4200 | HTTP |
| Vue | 5173 | HTTP |
| .NET (Kestrel) | 5000 | HTTP |
| .NET (Kestrel HTTPS) | 5001 | HTTPS |
| Aspire Dashboard | 15888 | HTTP |
| Django | 8000 | HTTP |
| FastAPI | 8000 | HTTP |
| Flask | 5000 | HTTP |
| Express | 3000 | HTTP |
| Spring Boot | 8080 | HTTP |
| Go | 8080 | HTTP |
| PHP | 8000 | HTTP |

#### Priority 5: Dynamic Port Assignment
If no port is detected or there's a conflict, assign dynamically:
- Find available port starting from 3000
- Increment until free port found
- Set as environment variable for the service

### 4. Environment Variables & Configuration

#### Loading Environment Variables

1. **Azure Environment Variables**: Inherit from azd environment context (AZD_SERVER, AZD_ACCESS_TOKEN, Azure resources)
2. **Service env from azure.yaml**: Merge env variables from service definition
3. **Optional .env file**: Load from `--env-file` if specified
4. **Computed variables**: Auto-generated service URLs and ports

#### Environment Variable Resolution

Variables support substitution from multiple sources:

```yaml
services:
  api:
    env:
      - name: DATABASE_URL
        value: ${DATABASE_URL}  # From azd environment or .env file
      - name: FRONTEND_URL
        value: ${SERVICE_URL_WEB}  # Auto-generated from web service URL
      - name: API_KEY
        secret: ${API_KEY}  # Mark as secret (don't log)
```

**Auto-generated Service Variables:**
- `SERVICE_URL_{SERVICE_NAME}`: Full URL (e.g., `http://localhost:3000`)
- `SERVICE_PORT_{SERVICE_NAME}`: Port number only (e.g., `3000`)
- `SERVICE_HOST_{SERVICE_NAME}`: Host only (e.g., `localhost`)

#### Secret Handling

Variables marked with `secret` property:
- Not logged to console
- Masked in output (show as `***`)
- Passed securely to service process

### 5. Dependency Resolution & Startup Order

Services may depend on other services via the `uses` property:

```yaml
services:
  web:
    uses:
      - api  # web depends on api
  api:
    uses:
      - db  # api depends on db resource
  db:
    type: db.postgres
```

**Startup Algorithm:**

1. **Build dependency graph**: Parse `uses` relationships
2. **Topological sort**: Determine startup order
3. **Cycle detection**: Error if circular dependencies found
4. **Resource filtering**: Skip resources (db, storage, etc.) - only start services
5. **Parallel startup**: Start services at same dependency level concurrently
6. **Health checks**: Wait for service readiness before starting dependents

**Example Startup Sequence:**

```
Level 0 (no dependencies): [db (skip - resource)]
Level 1 (depends on Level 0): [api] → Start api
  Wait for api health check...
  ✓ api ready at http://localhost:8000
Level 2 (depends on Level 1): [web, worker] → Start in parallel
  Wait for web health check...
  ✓ web ready at http://localhost:3000
  Wait for worker health check...
  ✓ worker ready at http://localhost:8080
```

### 6. Health Checks

After starting each service, verify it's ready:

#### Health Check Strategies

**HTTP-based services:**
1. Try `GET /health` endpoint (common convention)
2. Try `GET /api/health`
3. Try `GET /` (root)
4. Timeout after 60 seconds
5. Retry every 2 seconds

**Non-HTTP services:**
1. Check if process is running
2. Check if port is listening
3. Wait for log indicators (e.g., "Server started")

**Framework-Specific Health Checks:**
- Aspire: Check dashboard availability at configured port
- Next.js: Wait for "ready on" log message
- Django: Check for "Starting development server" log
- Spring Boot: Check for "Started {Application}" log

#### Configuration

```yaml
services:
  api:
    config:
      healthCheck:
        path: /health  # Custom health endpoint
        timeout: 120  # Seconds to wait
        interval: 5  # Retry interval in seconds
```

### 7. Docker & Container Support

Services with `docker` configuration are run using Docker:

```yaml
services:
  worker:
    host: containerapp
    project: ./worker
    docker:
      path: ./Dockerfile
      context: .
      platform: amd64
      buildArgs:
        - NODE_ENV=development
```

**Docker Run Strategy:**

1. **Build image** (if Dockerfile present):
   ```bash
   docker build -t {service-name}:dev \
     --platform {platform} \
     --build-arg {args} \
     -f {dockerfile-path} \
     {context}
   ```

2. **Run container**:
   ```bash
   docker run -d \
     --name {service-name}-dev \
     -p {detected-port}:{container-port} \
     --env-file {generated-env-file} \
     {service-name}:dev
   ```

3. **Alternative: Docker Compose**
   If `docker-compose.yml` or `compose.yml` exists in project:
   ```bash
   docker compose -f {compose-file} up -d {service-name}
   ```

### 8. Process Management

All services run as child processes, managed by the azd app run command:

**Process Lifecycle:**
1. **Start**: Spawn process with inherited environment
2. **Monitor**: Capture stdout/stderr
3. **Health check**: Verify service is ready
4. **Log streaming**: Multiplex logs from all services
5. **Graceful shutdown**: On Ctrl+C, stop all services
6. **Cleanup**: Kill orphaned processes, remove temp files

**Process Tracking:**
- Track PIDs for all spawned processes
- Monitor process exit codes
- Restart on failure (optional with `--restart` flag)
- Log aggregation with service name prefix

## Output & User Experience

### Startup Output

```
🔍 Loading services from azure.yaml...

Found 4 services: web, api, worker, apphost

📋 Dependency order: db → api → web, worker → apphost

🚀 Starting services...

[db] ⏭️  Skipping resource (not a service)

[api] 📁 ./backend (Python/FastAPI)
[api] 🔧 Starting uvicorn app:app --reload --port 8000
[api] ⏳ Waiting for health check...
[api] ✅ Ready at http://localhost:8000

[web] 📁 ./frontend (TypeScript/Next.js)
[web] 🔧 Starting pnpm run dev --port 3000
[web] ⏳ Waiting for health check...
[web] ✅ Ready at http://localhost:3000

[worker] 📁 ./worker (.NET/Worker Service)
[worker] 🔧 Starting dotnet run --project worker.csproj
[worker] ⏳ Waiting for health check...
[worker] ✅ Ready (background service)

[apphost] 📁 ./AppHost (.NET/Aspire)
[apphost] 🔧 Starting dotnet run --project AppHost.csproj
[apphost] ⏳ Waiting for Aspire dashboard...
[apphost] ✅ Dashboard ready at http://localhost:15888

✨ All services started successfully!

📊 Service URLs:
  web:     http://localhost:3000
  api:     http://localhost:8000
  apphost: http://localhost:15888 (Aspire Dashboard)

🌐 Auto-generated environment variables:
  SERVICE_URL_WEB=http://localhost:3000
  SERVICE_URL_API=http://localhost:8000
  SERVICE_PORT_WEB=3000
  SERVICE_PORT_API=8000

📝 Streaming logs (Ctrl+C to stop all services)...
```

### Live Log Output

Logs from all services are streamed with colored prefixes:

```
[web]     ▲ Next.js 14.2.0
[web]     - Local:        http://localhost:3000
[api]     INFO:     Uvicorn running on http://0.0.0.0:8000
[api]     INFO:     Application startup complete.
[worker]  info: Microsoft.Hosting.Lifetime[0]
[worker]       Worker running at: 11/01/2025 10:30:00
[apphost] info: Aspire.Hosting.Dashboard[0]
[apphost]      Now listening on: http://localhost:15888
```

**Log Features:**
- Color-coded by service (different color per service)
- Timestamp prefix (optional with `--timestamps`)
- Log level detection and highlighting (ERROR, WARN, INFO)
- Search/filter logs (with `--filter` flag)

### Dry-Run Output

```
🔍 Loading services from azure.yaml...

Found 4 services: web, api, worker, apphost

📋 Execution plan:

[api] Would execute in ./backend:
  Command: uvicorn app:app --reload --port 8000
  Env vars:
    DATABASE_URL=postgresql://localhost:5432/mydb
    SERVICE_URL_WEB=http://localhost:3000 (auto-generated)
  
[web] Would execute in ./frontend:
  Command: pnpm run dev --port 3000
  Env vars:
    API_URL=http://localhost:8000
    SERVICE_URL_API=http://localhost:8000 (auto-generated)

[worker] Would execute in ./worker:
  Command: dotnet run --project worker.csproj
  Env vars:
    API_URL=http://localhost:8000
    SERVICE_URL_API=http://localhost:8000 (auto-generated)

[apphost] Would execute in ./AppHost:
  Command: dotnet run --project AppHost.csproj
  Env vars:
    (Inherits all Azure environment variables)

Run without --dry-run to start services.
```

### Error Handling

**Service-specific errors:**
```
[api] ❌ Failed to start
[api] Error: Port 8000 already in use
[api] 
[api] Suggestions:
[api]   1. Stop the process using port 8000
[api]   2. Use a different port in azure.yaml config.port
[api]   3. Let azd assign a port automatically (remove config.port)

⚠️  Stopping other services due to failure...
[web] 🛑 Stopped
```

**Dependency errors:**
```
❌ Circular dependency detected: api → web → api

Dependency chain: api uses web, web uses api

Please fix the 'uses' relationships in azure.yaml.
```

**Missing configuration:**
```
[worker] ⚠️  No language specified and could not detect automatically
[worker] 
[worker] Please add to azure.yaml:
  services:
    worker:
      language: dotnet  # or python, js, ts, etc.
```

## Implementation Plan

### Phase 1: Core Service Orchestration

**New Files:**
- `src/internal/service/parser.go` - Parse azure.yaml services section
- `src/internal/service/detector.go` - Detect language/framework for each service
- `src/internal/service/executor.go` - Run individual services
- `src/internal/service/orchestrator.go` - Manage multi-service execution
- `src/internal/service/types.go` - Service-related types

**Key Functions:**
```go
// ParseAzureYaml reads azure.yaml and extracts service definitions
func ParseAzureYaml(path string) (*AzureYaml, error)

// DetectServiceRuntime determines how to run a service
func DetectServiceRuntime(service Service) (*ServiceRuntime, error)

// StartService starts a single service
func StartService(service Service, runtime ServiceRuntime, env map[string]string) (*ServiceProcess, error)

// OrchestratServices manages multi-service startup with dependencies
func OrchestrateServices(services []Service, options OrchestrationOptions) error
```

**Data Structures:**
```go
type AzureYaml struct {
    Name     string
    Services map[string]Service
    Resources map[string]Resource
}

type Service struct {
    Host     string
    Language string
    Project  string
    Docker   *DockerConfig
    Config   map[string]interface{}
    Env      []EnvVar
    Uses     []string
}

type ServiceRuntime struct {
    Language      string
    Framework     string
    PackageManager string
    Command       string
    Args          []string
    WorkingDir    string
    Port          int
    HealthCheck   HealthCheckConfig
}

type ServiceProcess struct {
    Name       string
    PID        int
    Port       int
    URL        string
    Process    *os.Process
    Stdout     io.ReadCloser
    Stderr     io.ReadCloser
    HealthCheck chan error
}
```

### Phase 2: Language & Framework Detection

**Enhance existing detector package:**
- `src/internal/detector/nodejs.go` - Enhanced Node.js/TypeScript detection
- `src/internal/detector/python.go` - Enhanced Python framework detection
- `src/internal/detector/dotnet.go` - .NET and Aspire detection
- `src/internal/detector/java.go` - Java framework detection (new)
- `src/internal/detector/go.go` - Go detection (new)
- `src/internal/detector/docker.go` - Docker/Compose detection (new)

**Port Detection:**
- `src/internal/detector/port.go` - Port detection logic
- Support for launchSettings.json, package.json, config files

**Framework-Specific Detection:**
```go
// DetectNodeFramework identifies Next.js, React, Vue, etc.
func DetectNodeFramework(projectDir string) (string, error)

// DetectPythonFramework identifies Django, FastAPI, Flask
func DetectPythonFramework(projectDir string) (string, error)

// DetectJavaFramework identifies Spring Boot, Quarkus, Micronaut
func DetectJavaFramework(projectDir string) (string, error)
```

### Phase 3: Environment & Configuration Management

**New Files:**
- `src/internal/service/env.go` - Environment variable resolution
- `src/internal/service/config.go` - Configuration merging

**Key Functions:**
```go
// ResolveEnvironment merges env vars from multiple sources
func ResolveEnvironment(service Service, azureEnv, dotEnv map[string]string, serviceURLs map[string]string) map[string]string

// GenerateServiceURLs creates SERVICE_URL_* variables
func GenerateServiceURLs(services map[string]*ServiceProcess) map[string]string

// LoadDotEnv loads variables from .env file
func LoadDotEnv(path string) (map[string]string, error)
```

### Phase 4: Dependency Graph & Startup Order

**New Files:**
- `src/internal/service/graph.go` - Dependency graph construction
- `src/internal/service/topology.go` - Topological sort for startup order

**Key Functions:**
```go
// BuildDependencyGraph creates graph from service 'uses' relationships
func BuildDependencyGraph(services map[string]Service) (*DependencyGraph, error)

// TopologicalSort returns services in startup order
func TopologicalSort(graph *DependencyGraph) ([][]string, error)

// DetectCycles identifies circular dependencies
func DetectCycles(graph *DependencyGraph) error
```

### Phase 5: Health Checks & Monitoring

**New Files:**
- `src/internal/service/health.go` - Health check implementations
- `src/internal/service/monitor.go` - Process monitoring

**Key Functions:**
```go
// PerformHealthCheck verifies service is ready
func PerformHealthCheck(process *ServiceProcess, config HealthCheckConfig) error

// HTTPHealthCheck attempts HTTP requests to health endpoints
func HTTPHealthCheck(url string, timeout time.Duration) error

// PortHealthCheck verifies port is listening
func PortHealthCheck(port int, timeout time.Duration) error

// MonitorProcess tracks process status and logs
func MonitorProcess(process *ServiceProcess, output chan LogEntry) error
```

### Phase 6: Log Streaming & Output

**New Files:**
- `src/internal/service/logger.go` - Multi-service log aggregation
- `src/internal/service/output.go` - Formatted output

**Key Functions:**
```go
// StreamLogs multiplexes logs from all services
func StreamLogs(processes []*ServiceProcess, output io.Writer) error

// FormatLogEntry adds service prefix and color
func FormatLogEntry(service string, message string, level LogLevel) string

// DisplayServiceURLs shows startup summary
func DisplayServiceURLs(processes []*ServiceProcess) error
```

### Phase 7: Docker Integration

**New Files:**
- `src/internal/service/docker.go` - Docker build and run

**Key Functions:**
```go
// BuildDockerImage builds image from Dockerfile
func BuildDockerImage(service Service, config DockerConfig) error

// RunDockerContainer starts container from image
func RunDockerContainer(service Service, image string, env map[string]string) (*ServiceProcess, error)

// RunDockerCompose uses docker-compose.yml if present
func RunDockerCompose(service Service, composeFile string) (*ServiceProcess, error)
```

### Phase 8: Command Integration & Flags

**Update existing file:**
- `src/cmd/app/commands/run.go` - Add service orchestration logic

**New Command Flags:**
```go
var (
    serviceFilter  []string  // --service, -s
    envFile        string    // --env-file
    verbose        bool      // --verbose, -v
    dryRun         bool      // --dry-run
    noHealthCheck  bool      // --no-health-check
    timestamps     bool      // --timestamps
    restart        bool      // --restart
    logFilter      string    // --filter
)
```

### Phase 9: Testing

**Test Files:**
- `src/internal/service/parser_test.go`
- `src/internal/service/detector_test.go`
- `src/internal/service/graph_test.go`
- `src/internal/service/env_test.go`
- `src/internal/service/health_test.go`
- `src/cmd/app/commands/run_integration_test.go`

**Test Scenarios:**
- Parse azure.yaml with various service configurations
- Detect languages and frameworks correctly
- Build dependency graphs and detect cycles
- Resolve environment variables with substitution
- Perform health checks for different service types
- Handle errors gracefully (port conflicts, missing files)
- Integration tests with real azure.yaml examples

**Test Projects:**
- `tests/projects/multi-service/` - Multi-service test app
  - Node.js frontend
  - Python API backend
  - .NET worker service
  - Aspire AppHost
  - Docker-based service

### Phase 10: Documentation

**Documentation Updates:**
- `README.md` - Add run services example
- `docs/run-command.md` - Comprehensive guide (new)
- `docs/service-detection.md` - Detection strategies (new)
- `docs/azure-yaml-services.md` - Service configuration guide (new)
- `docs/quickstart.md` - Update with multi-service example
- `CHANGELOG.md` - Document new feature

## Advanced Features (Future Enhancements)

### 1. Service Restart on File Changes

Watch for file changes and auto-restart services:

```bash
azd app run --watch
```

**Implementation:**
- Use file watchers (fsnotify) for each service project directory
- Debounce changes (wait 500ms after last change)
- Restart affected service only
- Show rebuild progress

### 2. Service Scaling

Run multiple instances of a service:

```yaml
services:
  api:
    config:
      replicas: 3  # Run 3 instances
```

**Implementation:**
- Assign different ports to each instance (8000, 8001, 8002)
- Load balance via nginx or built-in proxy
- Update SERVICE_URL_API to load balancer URL

### 3. Remote Service Connections

Connect to remote Azure services during local development:

```yaml
services:
  api:
    config:
      remote: true  # Don't run locally, use Azure deployment
```

**Implementation:**
- Fetch Azure service URL from deployment
- Set SERVICE_URL_API to Azure URL
- Skip local startup for this service

### 4. Debug Mode

Attach debuggers to services:

```bash
azd app run --debug api
```

**Implementation:**
- Start service with debug flags enabled
- Node.js: `--inspect`
- Python: Wait for debugpy
- .NET: `--debugger` or environment variable
- Display debug connection info

### 5. Service Templates

Pre-configured service detection templates:

```yaml
services:
  api:
    template: python-fastapi  # Use template instead of detection
```

**Templates:**
- `node-nextjs`
- `node-express`
- `python-fastapi`
- `python-django`
- `dotnet-webapi`
- `dotnet-aspire`
- `java-springboot`

### 6. Interactive Service Selection

Prompt user to choose which services to run:

```bash
azd app run --interactive

? Select services to run: (Use arrow keys and space to select)
  [x] web
  [x] api
  [ ] worker
  [x] apphost
```

### 7. Parallel Health Checks

Optimize startup by checking health in parallel for same-level dependencies:

```
Level 0: [db] → Skip
Level 1: [api, auth] → Start both, health check in parallel
Level 2: [web] → Start after both api and auth are healthy
```

### 8. Resource Provisioning

Automatically provision local resources (databases, etc.):

```yaml
resources:
  db:
    type: db.postgres
    config:
      local: true  # Start local postgres via Docker
```

**Implementation:**
- Detect resources with `local: true`
- Start Docker containers for databases
- Generate connection strings
- Inject into dependent services

## Security Considerations

1. **Path Validation**: All paths from azure.yaml validated with `security.ValidatePath()`
2. **Environment Variable Injection**: Sanitize and validate environment variables
3. **Process Isolation**: Services run in separate processes with restricted permissions
4. **Secret Handling**: Mask secrets in logs and output
5. **Port Binding**: Only bind to localhost by default (no external exposure)
6. **Docker Security**: Use least-privilege containers, no privileged mode
7. **Input Validation**: Validate all azure.yaml input with JSON schema

## Error Recovery & Resilience

1. **Graceful Degradation**: If one service fails, show error but keep others running
2. **Retry Logic**: Retry failed health checks with exponential backoff
3. **Timeout Protection**: Kill services that take too long to start
4. **Resource Cleanup**: Always cleanup processes and temp files, even on error
5. **Signal Handling**: Proper handling of Ctrl+C, SIGTERM, SIGINT
6. **Zombie Prevention**: Reap child processes correctly

## Performance Considerations

1. **Lazy Loading**: Only parse azure.yaml when needed
2. **Parallel Execution**: Start services at same dependency level concurrently
3. **Efficient Health Checks**: Use lightweight HTTP HEAD requests
4. **Log Buffering**: Buffer logs before writing to reduce I/O
5. **Resource Limits**: Respect system limits (max processes, max file descriptors)

## Compatibility & Migration

### Backward Compatibility

The enhanced `azd app run` command maintains backward compatibility:

1. **No azure.yaml**: Falls back to current behavior (detect Aspire, pnpm, docker)
2. **Empty services section**: Same fallback
3. **Legacy project structure**: Works without azure.yaml

### Migration Path

For existing projects:

1. **Step 1**: Continue using `azd app run` without azure.yaml (works as before)
2. **Step 2**: Generate azure.yaml with `azd init` or manually create
3. **Step 3**: Add services section incrementally
4. **Step 4**: Enjoy multi-service orchestration

## Success Metrics

- **Adoption**: 70%+ of projects with multiple services use azure.yaml services
- **Accuracy**: 95%+ correct language/framework detection
- **Performance**: Services start within 30 seconds for typical projects
- **Reliability**: 99%+ successful startups without errors
- **Coverage**: 80%+ test coverage for service package

## Open Questions

1. **Q**: Should we support non-HTTP services (gRPC, message queues)?
   **A**: Yes, but initial implementation focuses on HTTP. Add gRPC health checks in future enhancement.

2. **Q**: How to handle services that don't have health endpoints?
   **A**: Fall back to port listening check + process status check. Allow custom health check commands in config.

3. **Q**: Should we support running services on remote machines (SSH)?
   **A**: Not in MVP. Consider for future enhancement with `remote: ssh://...` configuration.

4. **Q**: How to handle services that require compilation/build steps?
   **A**: Detect build requirements and run build commands automatically (e.g., `tsc` for TypeScript, `npm run build`). Show build progress.

5. **Q**: Should we cache detection results to speed up subsequent runs?
   **A**: Yes, cache detection results in `.azd/.cache/service-runtime.json`. Invalidate on azure.yaml change.

6. **Q**: How to handle different environments (dev, staging, prod)?
   **A**: Support `--env {name}` flag to load environment-specific configuration. Future enhancement: azure.yaml environment overrides.

## Timeline Estimate

- **Phase 1** (Core Orchestration): 5 days
- **Phase 2** (Language Detection): 5 days
- **Phase 3** (Environment Management): 3 days
- **Phase 4** (Dependency Graph): 3 days
- **Phase 5** (Health Checks): 4 days
- **Phase 6** (Log Streaming): 3 days
- **Phase 7** (Docker Integration): 4 days
- **Phase 8** (Command Integration): 2 days
- **Phase 9** (Testing): 6 days
- **Phase 10** (Documentation): 3 days

**Total**: ~38 development days

## Example Workflows

### Multi-Service App Development

```bash
# Clone project with azure.yaml
git clone https://github.com/org/multi-service-app
cd multi-service-app

# Generate requirements
azd app reqs --gen

# Check all requirements are met
azd app reqs

# Install dependencies for all services
azd app deps

# Run all services
azd app run

# Visit app at http://localhost:3000
# API available at http://localhost:8000
# Aspire dashboard at http://localhost:15888
```

### Selective Service Testing

```bash
# Run only API and database for API testing
azd app run --service api

# Run frontend and API, skip worker
azd app run --service web --service api

# Dry-run to see what would be executed
azd app run --dry-run
```

### Development with Custom Environment

```bash
# Load environment from local .env file
azd app run --env-file .env.local

# Run with verbose logging
azd app run --verbose

# Watch for changes and auto-restart
azd app run --watch
```

## Appendix: Language Detection Examples

### Example 1: Next.js Frontend

**Project Structure:**
```
frontend/
  package.json
  next.config.js
  tsconfig.json
```

**Detection Result:**
```yaml
Runtime:
  Language: TypeScript
  Framework: Next.js
  PackageManager: pnpm
  Command: pnpm run dev
  Port: 3000 (from package.json or default)
```

### Example 2: FastAPI Backend

**Project Structure:**
```
backend/
  pyproject.toml
  main.py
  requirements.txt
```

**Detection Result:**
```yaml
Runtime:
  Language: Python
  Framework: FastAPI
  PackageManager: uv
  Command: uvicorn main:app --reload --port 8000
  Port: 8000
```

### Example 3: Aspire AppHost

**Project Structure:**
```
AppHost/
  AppHost.csproj
  AppHost.cs
  Properties/
    launchSettings.json
```

**Detection Result:**
```yaml
Runtime:
  Language: .NET
  Framework: Aspire
  Command: dotnet run --project AppHost.csproj
  Port: 15888 (Aspire dashboard from launchSettings.json)
```

### Example 4: Spring Boot API

**Project Structure:**
```
api/
  pom.xml
  src/
    main/
      java/
      resources/
        application.properties
```

**Detection Result:**
```yaml
Runtime:
  Language: Java
  Framework: Spring Boot
  Command: mvn spring-boot:run
  Port: 8080 (from application.properties or default)
```
