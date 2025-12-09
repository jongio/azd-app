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

### Expected behavior
1. Container services (azurite, cosmos, redis, postgres) should start as Docker containers
2. The api service should start as a process
3. Health checks should report status for all services
4. Dashboard should show container services with docker icon

### Connection strings
- **Azurite**: `UseDevelopmentStorage=true`
- **Cosmos**: `AccountEndpoint=https://localhost:8081/;AccountKey=...`
- **Redis**: `redis://localhost:6379`
- **Postgres**: `postgresql://postgres:postgres@localhost:5432/testdb`
