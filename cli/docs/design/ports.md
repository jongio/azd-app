# Port Management Design

## Overview

This document describes the port management design for `azd app`, including the rationale for aligning with Docker Compose, user flows, technical implementation, and migration guidance.

## Goals

1. **Docker Compose Alignment**: Use Docker Compose `ports` array format exclusively to enable seamless docker-compose.yml generation in the future
2. **Single Format**: Eliminate confusion by having one way to specify ports instead of multiple approaches
3. **Flexibility**: Support both explicit port mapping and flexible auto-assignment
4. **Developer Experience**: Make port configuration intuitive for Docker and non-Docker services

## Design Decisions

### Docker Compose Format

We adopted the Docker Compose `ports` specification format as our standard:

```yaml
services:
  api:
    ports:
      - "8080"          # Simple port
      - "3000:8080"     # host:container mapping
```

**Rationale:**
- Docker Compose is the industry standard for multi-service orchestration
- Developers are already familiar with this format
- Enables future docker-compose.yml generation with zero translation
- Supports advanced scenarios (IP binding, protocols, port ranges)

### Port Specification Format

The `ports` field is an array of strings supporting multiple Docker Compose formats:

| Format | Example | Meaning |
|--------|---------|---------|
| `"PORT"` (non-Docker) | `"8080"` | App listens on 8080, host also 8080 |
| `"PORT"` (Docker) | `"8080"` | Container port 8080, host auto-assigned |
| `"HOST:CONTAINER"` | `"3000:8080"` | App on 8080, host on 3000 |
| `"IP:HOST:CONTAINER"` | `"127.0.0.1:3000:8080"` | Bind to specific IP |
| `"PORT/PROTOCOL"` | `"53:53/udp"` | UDP protocol |
| `"[IPv6]:HOST:CONTAINER"` | `"[::1]:3000:8080"` | IPv6 binding |

### Behavior Differences: Docker vs Non-Docker

#### Non-Docker Services
For non-Docker services (Node.js, Python, .NET running natively):
- `"8080"` means the application listens on port 8080 AND the host port is 8080
- No port translation needed - single port serves both purposes
- Simple and intuitive for local development

#### Docker Services  
For Docker/containerized services:
- `"8080"` means container port 8080, host port auto-assigned (0)
- Follows Docker Compose convention where a single port is container-only
- Use `"3000:8080"` for explicit host:container mapping
- Docker handles the port translation between host and container

This matches Docker Compose behavior exactly, making the transition to generated docker-compose.yml seamless.

## User Flows

### Scenario 1: Simple Service (Non-Docker)

**User wants:** Run a Next.js app on port 3000

**azure.yaml:**
```yaml
services:
  web:
    language: node
    project: ./web
    ports:
      - "3000"
```

**Behavior:**
- App starts on port 3000
- Accessible at http://localhost:3000
- No port translation

### Scenario 2: Docker Service with Auto-Assigned Host Port

**User wants:** Run containerized API, don't care about host port

**azure.yaml:**
```yaml
services:
  api:
    language: python
    project: ./api
    docker:
      image: python:3.9
    ports:
      - "8080"
```

**Behavior:**
- Container listens on 8080
- Host port auto-assigned (e.g., 51234)
- Accessible at http://localhost:51234
- isExplicit = false (port was auto-assigned)

### Scenario 3: Explicit Host-Container Mapping

**User wants:** Frontend on host 3000, backend container on 8080

**azure.yaml:**
```yaml
services:
  web:
    language: node
    project: ./frontend
    ports:
      - "3000:3000"
  
  api:
    language: python
    project: ./backend
    docker:
      image: python:3.9
    ports:
      - "8080:8000"  # Host 8080 -> Container 8000
```

**Behavior:**
- Frontend: both host and app on 3000
- API: host on 8080, container on 8000
- isExplicit = true for both

### Scenario 4: Multiple Ports

**User wants:** Service exposing HTTP and gRPC

**azure.yaml:**
```yaml
services:
  api:
    language: go
    project: ./api
    ports:
      - "8080"      # HTTP
      - "9090"      # gRPC
```

**Behavior:**
- App listens on both 8080 and 9090
- First port (8080) is considered "primary" for health checks
- GetPrimaryPort() returns 8080

### Scenario 5: Localhost-Only Binding

**User wants:** Database only accessible from localhost

**azure.yaml:**
```yaml
services:
  db:
    language: other
    project: ./db
    ports:
      - "127.0.0.1:5432:5432"
```

**Behavior:**
- Binds only to 127.0.0.1
- Not accessible from network
- Secure local development

## Technical Implementation

### Type Definitions

#### Service Struct
```go
type Service struct {
    Name      string   `yaml:"name"`
    Language  string   `yaml:"language,omitempty"`
    Framework string   `yaml:"framework,omitempty"`
    Project   string   `yaml:"project,omitempty"`
    Ports     []string `yaml:"ports,omitempty"`  // Docker Compose format
    // ... other fields
}
```

#### PortMapping Struct
```go
type PortMapping struct {
    HostPort      int    // Port on host machine (0 = auto-assign)
    ContainerPort int    // Port in container/app
    BindIP        string // IP to bind to (empty = all interfaces)
    Protocol      string // "tcp" or "udp"
}
```

### Core Functions

#### ParsePortSpec
```go
func ParsePortSpec(spec string, isDocker bool) PortMapping
```

Parses a Docker Compose port specification string into a PortMapping.

**Examples:**
- `ParsePortSpec("8080", false)` → HostPort=8080, ContainerPort=8080
- `ParsePortSpec("8080", true)` → HostPort=0 (auto), ContainerPort=8080
- `ParsePortSpec("3000:8080", true)` → HostPort=3000, ContainerPort=8080
- `ParsePortSpec("127.0.0.1:3000:8080", false)` → BindIP=127.0.0.1, HostPort=3000, ContainerPort=8080
- `ParsePortSpec("53:53/udp", false)` → HostPort=53, ContainerPort=53, Protocol=udp

**Special Handling:**
- IPv6 addresses: `"[::1]:3000:8080"` or `"::1:3000:8080"`
- Protocol suffix: `"/udp"` or `"/tcp"`
- Whitespace trimming

#### GetPortMappings
```go
func (s *Service) GetPortMappings(isDocker bool) []PortMapping
```

Returns all port mappings for a service.

#### GetPrimaryPort
```go
func (s *Service) GetPrimaryPort(isDocker bool) (PortMapping, bool)
```

Returns the first port mapping (primary port) for health checks and URL generation.

### Port Detection Flow

```
DetectPort(serviceName, service, projectDir, framework, usedPorts)
    │
    ├─> Has explicit ports array in azure.yaml?
    │   ├─> YES: GetPrimaryPort() → return port, isExplicit=true
    │   └─> NO: Continue to detection
    │
    ├─> Check environment variables (PORT, SERVICE_PORT)
    │   └─> Found? Return port, isExplicit=false
    │
    ├─> Check package.json scripts (Node.js)
    │   └─> Found? Return port, isExplicit=false
    │
    ├─> Use framework defaults
    │   └─> Return default port, isExplicit=false
    │
    └─> Port conflict?
        └─> Auto-assign available port, isExplicit=false
```

### YAML Update

When `azd app run` auto-assigns a port, it updates azure.yaml:

```go
func UpdateServicePort(azureYamlPath, serviceName string, port int) error
```

**Behavior:**
- Preserves all comments and formatting
- Writes `ports:` array instead of scalar `port:` field
- Format: 
  ```yaml
  service:
    ports:
      - "8080"
    language: python
  ```

### Port Assignment

```go
func (pm *PortManager) AssignPort(serviceName string, preferredPort int) (int, bool, error)
```

**Returns:**
- `port int`: The assigned port
- `isExplicit bool`: true if port was explicitly configured, false if auto-assigned
- `error`: Any error during assignment

**Logic:**
1. Check if preferred port is available
2. If available → return preferred port, isExplicit based on source
3. If not available → find random available port, isExplicit=false
4. Update azure.yaml with assigned port

## isExplicit Flag

The `isExplicit` flag indicates whether a port was explicitly configured by the user or auto-assigned:

| Scenario | isExplicit | Explanation |
|----------|-----------|-------------|
| `ports: ["8080"]` in azure.yaml | `true` | User explicitly configured |
| `"3000:8080"` mapping | `true` | Explicit host:container |
| Framework default (no conflict) | `false` | Detected, not explicit |
| Auto-assigned due to conflict | `false` | Dynamically assigned |
| From PORT env var | `false` | Detected from environment |

**Usage:**
- Determines whether to update azure.yaml (only update if not explicit)
- Helps with messaging to users
- Used in port conflict resolution

## Migration from Old Format

### Breaking Change

This is a **breaking change**. The old `port: 8080` scalar field is replaced with `ports: ["8080"]` array.

### Migration Steps

**Before:**
```yaml
services:
  api:
    port: 8080
    language: python
```

**After:**
```yaml
services:
  api:
    ports:
      - "8080"
    language: python
```

### Automatic Migration

There is **no automatic migration** - this is a manual update. Users must update their azure.yaml files.

### Migration Tools

Users can:
1. Manually update azure.yaml (recommended for small projects)
2. Delete the `port:` field and let `azd app run` auto-detect and update

## Testing

### Unit Tests

Comprehensive unit tests in `ports_test.go`:

1. **TestParsePortSpec**: 11 test cases
   - Single port (Docker vs non-Docker)
   - Host:container mapping
   - IP:host:container (IPv4 and IPv6)
   - Protocol specification (UDP)
   - Whitespace handling

2. **TestServiceGetPortMappings**: 6 test cases
   - Single port
   - Multiple ports
   - Empty ports array
   - Mixed explicit and auto-assign

3. **TestServiceGetPrimaryPort**: 5 test cases
   - Returns first mapping
   - Handles empty array
   - Docker auto-assign behavior

4. **TestDetectPortWithPortsArray**: 4 test cases
   - Explicit port from array
   - Host:container mapping
   - Fallback to auto-assign
   - Docker container-only port

### Integration Tests

Integration tests verify end-to-end workflows:
- Port detection with azure.yaml
- YAML update after auto-assignment
- Port conflict resolution
- Multi-service orchestration

## Future Enhancements

### Docker Compose Generation

With ports in Docker Compose format, we can generate docker-compose.yml directly:

```yaml
# azure.yaml
services:
  api:
    docker:
      image: python:3.9
    ports:
      - "3000:8080"
```

↓ Generates ↓

```yaml
# docker-compose.yml
services:
  api:
    image: python:3.9
    ports:
      - "3000:8080"
```

Zero translation needed!

### Port Ranges

Docker Compose supports port ranges:
```yaml
ports:
  - "3000-3005:8000-8005"
```

Future enhancement: Support port range parsing in `ParsePortSpec`.

### Advanced Binding

Support for:
- IPv6 ranges: `"[::1]:3000-3005:8000-8005"`
- Multiple IPs per port: `["127.0.0.1:8080:8080", "192.168.1.10:8080:8080"]`

## Examples

### Example 1: Full-Stack App

```yaml
name: fullstack-app
services:
  # Frontend - explicit port
  web:
    language: node
    project: ./frontend
    ports:
      - "3000"
  
  # Backend - explicit mapping for Docker
  api:
    language: python
    project: ./backend
    docker:
      image: python:3.9
    ports:
      - "8080:8000"
  
  # Database - localhost only
  db:
    language: other
    project: ./db
    ports:
      - "127.0.0.1:5432:5432"
```

### Example 2: Microservices

```yaml
name: microservices
services:
  # API Gateway
  gateway:
    language: node
    project: ./gateway
    ports:
      - "8080"
  
  # User Service
  users:
    language: python
    project: ./services/users
    docker:
      image: python:3.9
    ports:
      - "8081:8000"
  
  # Order Service
  orders:
    language: go
    project: ./services/orders
    docker:
      image: golang:1.21
    ports:
      - "8082:8080"
```

### Example 3: Flexible Development

```yaml
name: dev-app
services:
  # Let azd detect and assign ports
  api:
    language: python
    project: ./api
    # No ports specified - will auto-detect from framework
  
  web:
    language: node
    project: ./web
    # No ports specified - will auto-detect from package.json
```

On first run, `azd app run` will:
1. Detect framework defaults
2. Assign ports (handling conflicts)
3. Update azure.yaml with assigned ports

## References

- [Docker Compose Port Specification](https://docs.docker.com/compose/compose-file/05-services/#ports)
- Implementation: `cli/src/internal/service/port.go`
- Tests: `cli/src/internal/service/ports_test.go`
- YAML Update: `cli/src/internal/yamlutil/update_service_port.go`
