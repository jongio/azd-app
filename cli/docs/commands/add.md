# azd app add

Add a well-known container service to your azure.yaml configuration.

## Synopsis

```
azd app add [service] [flags]
```

## Description

The `add` command simplifies adding commonly-used services like Azure emulators, databases, and caches to your project. It automatically configures the Docker image, ports, environment variables, and health checks.

When you run `azd app run`, container services defined in azure.yaml will be started alongside your application services.

## Available Services

| Service | Description | Ports |
|---------|-------------|-------|
| `azurite` | Azure Storage emulator (Blob, Queue, Table) | 10000, 10001, 10002 |
| `cosmos` | Azure Cosmos DB emulator | 8081, 10250-10254 |
| `redis` | Redis in-memory cache | 6379 |
| `postgres` | PostgreSQL database | 5432 |

## Flags

| Flag | Description |
|------|-------------|
| `--list` | List all available services |
| `--show-connection` | Show connection string after adding |
| `-o, --output` | Output format (default, json) |

## Examples

### List available services

```bash
azd app add --list
```

### Add Azurite storage emulator

```bash
azd app add azurite
```

This adds the following to your azure.yaml:

```yaml
services:
  azurite:
    image: mcr.microsoft.com/azure-storage/azurite:latest
    ports:
      - "10000:10000"  # Blob
      - "10001:10001"  # Queue
      - "10002:10002"  # Table
```

### Add PostgreSQL with connection string

```bash
azd app add postgres --show-connection
```

Output includes the connection string:

```
Connection Strings
  default: postgresql://postgres:postgres@localhost:5432/app?sslmode=disable
```

### Get JSON output

```bash
azd app add redis --output json
```

```json
{
  "service": "redis",
  "added": true,
  "message": "Added Redis to azure.yaml"
}
```

## Connection Strings

After adding a service, you can use these connection strings in your application:

### Azurite

| Type | Connection String |
|------|-------------------|
| Blob | `DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=...;BlobEndpoint=http://127.0.0.1:10000/devstoreaccount1` |
| Queue | `DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=...;QueueEndpoint=http://127.0.0.1:10001/devstoreaccount1` |
| Table | `DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=...;TableEndpoint=http://127.0.0.1:10002/devstoreaccount1` |
| Default | `UseDevelopmentStorage=true` |

### Cosmos DB

```
AccountEndpoint=https://localhost:8081/;AccountKey=C2y6yDjf5/R+ob0N8A7Cgv30VRDJIWEHLM+4QDU5DE2nQ9nDuVTqobD4b8mGGyPMbIZnqyMsEcaGQy67XIw/Jw==
```

### Redis

```
localhost:6379
```

### PostgreSQL

```
postgresql://postgres:postgres@localhost:5432/app?sslmode=disable
```

## Requirements

Container services require Docker to be installed and running. The `azd app reqs` command will automatically check for Docker when container services are defined.

## See Also

- [azd app run](run.md) - Start services including containers
- [azd app reqs](reqs.md) - Check prerequisites including Docker
