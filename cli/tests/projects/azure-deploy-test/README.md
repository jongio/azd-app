# Azure Deployment Test Project

A minimal project for testing Azure deployment properties with `azd` and `azd app`.

## Purpose

This project tests:
- Azure deployment with `azd up`
- Local development with `azd app run`
- Azure environment variable inheritance (`AZURE_*`, `AZD_*`)
- Service URL discovery (`SERVICE_WEB_URL`)

## Structure

```
azure-deploy-test/
├── azure.yaml          # azd configuration
├── infra/
│   └── main.bicep      # Minimal Azure Container Apps deployment
└── src/
    └── web/
        ├── package.json
        └── server.js   # Express app showing Azure context
```

## Quick Start

### Local Development

```bash
cd cli/tests/projects/azure-deploy-test
azd app run
```

### Deploy to Azure

```bash
cd cli/tests/projects/azure-deploy-test
azd up
```

### Test Azure Properties

After deployment, the web app displays all `AZURE_*` and `AZD_*` environment variables to verify context inheritance.

## What It Tests

| Property | Description |
|----------|-------------|
| `AZURE_ENV_NAME` | Environment name (dev, staging, prod) |
| `AZURE_SUBSCRIPTION_ID` | Azure subscription ID |
| `AZURE_RESOURCE_GROUP_NAME` | Resource group name |
| `AZURE_LOCATION` | Azure region |
| `SERVICE_WEB_URL` | Auto-generated URL for deployed service |
| `AZD_SERVER` | gRPC server for extension communication |
| `AZD_ACCESS_TOKEN` | JWT token for extension API |

## Expected Output

When running locally with `azd app run`:
```
🚀 Server running at http://localhost:3000

Azure Environment:
  AZURE_ENV_NAME: (from azd env)
  AZURE_SUBSCRIPTION_ID: ********
  ...
```

When deployed to Azure:
```
Azure Environment:
  AZURE_ENV_NAME: dev
  AZURE_SUBSCRIPTION_ID: abc123-...
  AZURE_RESOURCE_GROUP_NAME: rg-azure-deploy-test-dev
  AZURE_LOCATION: eastus
  SERVICE_WEB_URL: https://web-xyz.azurecontainerapps.io
```
