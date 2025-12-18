# Local Services (host: local)

## Overview

Services with `host: local` are **development-only** containers or emulators that run locally but are **automatically skipped** during Azure deployment. Common examples include:

- **azurite** - Azure Storage emulator
- **postgres** - PostgreSQL database
- **redis** - Redis cache
- **mongodb** - MongoDB database
- **cosmosdb emulator** - Azure Cosmos DB emulator

## Configuration

Define local services in `azure.yaml`:

```yaml
services:
  # Deployable services
  api:
    host: containerapp
    language: python
    project: ./src/api
  
  # Local-only services (not deployed)
  azurite:
    host: local
    image: mcr.microsoft.com/azure-storage/azurite:latest
    ports:
      - 10000:10000
      - 10001:10001
      - 10002:10002
  
  postgres:
    host: local
    image: postgres:16
    ports:
      - 5432:5432
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: myapp
```

## Behavior

### During `azd app run`

Local services **will start** along with your other services:

```bash
azd app run
```

```
✓ Starting services...
  - api (containerapp)
  - azurite (local)
  - postgres (local)
```

### During `azd deploy`

Local services are **automatically skipped** - no error, no deployment:

```bash
azd deploy
```

```
✓ Done: Deploying service api
  - Endpoint: https://api-xyz.azurewebsites.net/

✓ Done: Deploying service azurite
  - No artifacts were found (local-only service)

SUCCESS: Your application was deployed to Azure
```

The extension gracefully handles mixed services, allowing you to keep local development services in your azure.yaml without affecting production deployments.

## Best Practices

### Keep Local Services in azure.yaml

**Recommended**: Keep all services in one azure.yaml file for simplicity:

```yaml
services:
  api:
    host: containerapp
    project: ./src/api
    environment:
      # Development: uses azurite (via azd app run)
      # Production: uses real Azure Storage (via azd deploy)
      AZURE_STORAGE_CONNECTION_STRING: ${AZURE_STORAGE_CONNECTION_STRING}

# Development only - remove or comment out for production
# azurite:
#   host: local
#   image: mcr.microsoft.com/azure-storage/azurite:latest
```

### Option 3: Use Infrastructure as Code

name: my-app
services:
  # Deployable service
  api:
    host: containerapp
    project: ./src/api
    environment:
      # Use local emulator in dev, real Azure Storage in production
      AZURE_STORAGE_CONNECTION_STRING: ${AZURE_STORAGE_CONNECTION_STRING}
  
  # Local-only emulator (automatically skipped during azd deploy)
  azurite:
    host: local
    image: mcr.microsoft.com/azure-storage/azurite:latest
    ports:
      - 10000:10000
      - 10001:10001
      - 10002:10002
```

### Use Environment-Specific Configuration

Use Azure resources that are provisioned during `azd provision`:

```yaml
services:
  api:
    host: containerapp
    project: ./src/api
  
  # Local emulator for development
  azurite:
    host: local
    image: mcr.microsoft.com/azure-storage/azurite:latest

# Infrastructure as Code provisions real Azure Storage
resources:
  storage:all services (including local emulators)
azd app run

# 2. Make changes and test locally

# 3. Deploy to Azure (local services automatically skipped)
azd deploy
```

The workflow is seamless - no need to modify azure.yaml between environments.**During `azd app run`**: Starts local containers normally
2. **During `azd deploy`**: Returns empty artifacts (no-op) allowing deployment to continue for other services
3. **Logging**: Records that the service was skipped in deployment logs

This seamless handling means you can use one azure.yaml for both development and production.

## Development Workflows local services), `azd deploy` (skips local services)
- **Schema**: See [azure.yaml.json](../../../schemas/v1.1/azure.yaml.json) for full schema

## See Also

- [Container Services](../containers.md)
- [Run Command](../commands/run.md)
- [Azure Deployment](../azure-deployment.md)
