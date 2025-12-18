# azure.yaml Schema v1.1 Changelog

## Overview

The v1.1 schema is now a **proper superset** of the [v1.0 schema](https://raw.githubusercontent.com/Azure/azure-dev/main/schemas/v1.0/azure.yaml.json) from Azure/azure-dev, with additional properties for local development orchestration via `azd app`.

All v1.0 azure.yaml files are **fully compatible** with v1.1, and all v1.0 properties are preserved and validated according to the original specification.

## What's New in v1.1

### Local Development Features (azd app extensions)

1. **Service Type & Mode** (`type`, `mode`)
   - Control service behavior: `http`, `tcp`, `process`, `container`
   - Run modes: `watch`, `build`, `daemon`, `task`

2. **Development Commands** (`command`, `entrypoint`)
   - Override auto-detected run commands
   - Specify custom entry points

3. **Port Mappings** (`ports`)
   - Docker Compose-style port syntax
   - Support for host binding and protocols

4. **Enhanced Environment Variables** (`environment`)
   - Array or object format (Docker Compose compatible)
   - Secret references

5. **Health Checks** (`healthcheck`)
   - Multiple check types: `http`, `tcp`, `process`, `output`
   - Docker Compose-compatible configuration
   - Pattern matching for watch mode services

6. **Prerequisites** (`reqs`)
   - Tool version requirements
   - Custom version check commands
   - Install URLs for failed checks

7. **Logging Configuration** (`logs`)
   - Project and service-level log filters
   - Log level classifications
   - Azure Log Analytics integration
   - Real-time streaming support

8. **Test Configuration** (`test`)
   - Unit, integration, and e2e test types
   - Coverage thresholds
   - Custom test commands

9. **Additional Hooks** (`hooks.prerun`, `hooks.postrun`)
   - Run hooks for `azd app run` command

## Compatibility

### From v1.0 to v1.1

All v1.0 properties are **fully supported**:

- ✅ All service properties (`host`, `project`, `language`, `docker`, `k8s`, etc.)
- ✅ All resource types and configurations
- ✅ All hooks (provision, deploy, up, down, etc.)
- ✅ Infrastructure configuration (`infra`)
- ✅ Pipeline configuration (`pipeline`)
- ✅ State management (`state`)
- ✅ Platform configuration (`platform`)
- ✅ Workflows configuration (`workflows`)
- ✅ Cloud configuration (`cloud`)
- ✅ Required versions (`requiredVersions`)

### New Properties

All v1.1 additions are **optional** and marked as "azd app extension" in descriptions. They will be ignored by standard `azd` CLI but utilized by `azd app` extension.

## Schema Validation

The v1.1 schema maintains:
- Draft-07 JSON Schema compliance
- All v1.0 validation rules
- Conditional validation (allOf, if/then/else)
- Enum constraints for resource types
- Required field validation

## Migration Guide

### For v1.0 Users

No changes needed! Your existing azure.yaml files work as-is.

To use new v1.1 features:

```yaml
name: my-app

services:
  api:
    host: containerapp
    project: ./api
    # v1.1 additions:
    ports: ["8000"]
    command: "uvicorn main:app --reload"
    healthcheck:
      type: http
      path: /health
    logs:
      filters:
        exclude: ["npm warn"]
```

### Schema ID Update

Update your azure.yaml schema reference:

```yaml
# $schema: https://raw.githubusercontent.com/Azure/azure-dev/main/schemas/v1.0/azure.yaml.json
$schema: https://raw.githubusercontent.com/jongio/azd-app/main/schemas/v1.1/azure.yaml.json
```

## Documentation

- [Full Schema Reference](../../web/src/pages/reference/azure-yaml.astro)
- [azd app CLI Reference](../../cli/docs/cli-reference.md)
- [Original v1.0 Schema](https://raw.githubusercontent.com/Azure/azure-dev/main/schemas/v1.0/azure.yaml.json)
