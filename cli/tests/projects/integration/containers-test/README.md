# Container Services Test Project

This test project validates the well-known container services feature:

## Services

| Service | Image | Ports | Purpose |
|---------|-------|-------|---------|
| azurite | mcr.microsoft.com/azure-storage/azurite:latest | 10000, 10001, 10002 | Azure Storage emulator |
| cosmos | mcr.microsoft.com/cosmosdb/linux/azure-cosmos-emulator:latest | 8081, 10250 | Cosmos DB emulator |
| redis | redis:7-alpine | 6379 | Redis cache |
| postgres | postgres:16-alpine | 5432 | PostgreSQL database |
| api | Node.js | 3000 | Test API |

## Testing

### Start all services
```bash
azd app start
```

### Check service status
```bash
azd app health
```

### API Endpoints

The API tests actual connectivity to each container service:

| Endpoint | Description |
|----------|-------------|
| `http://localhost:3000/` | API info and available endpoints |
| `http://localhost:3000/health` | Simple health check with connection status |
| `http://localhost:3000/status` | Detailed status of all container connections |
| `http://localhost:3000/test` | Run fresh connection tests against all containers |

### What Each Service Test Does

- **Azurite**: Creates a blob container, uploads a test blob, lists blobs
- **Cosmos DB**: Creates a database/container, inserts a document, queries items
- **Redis**: Sets a key-value pair, retrieves it, counts test keys
- **PostgreSQL**: Creates a table, inserts a record, counts records

### Expected behavior
1. Container services (azurite, cosmos, redis, postgres) should start as Docker containers
2. The api service should start as a process
3. Health checks should report status for all services
4. Dashboard should show container services with docker icon
5. The `/status` endpoint should show successful connections to all services

### Connection strings (internal container network)
- **Azurite**: `http://azurite:10000/devstoreaccount1`
- **Cosmos**: `https://cosmos:8081`
- **Redis**: `redis://redis:6379`
- **Postgres**: `postgresql://postgres:postgres@postgres:5432/testdb`
