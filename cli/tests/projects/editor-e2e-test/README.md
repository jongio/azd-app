# Azure YAML Editor E2E Test Project

This test project is designed to comprehensively exercise **every feature** of the azure.yaml 1.1 schema for end-to-end testing of the azure.yaml editor.

## Purpose

This project serves as a complete test fixture that includes:
- All top-level azure.yaml 1.1 schema properties
- All service host types (appservice, containerapp, function, springapp, staticwebapp, aks, ai.endpoint, azure.ai.agent)
- All resource types (databases, AI resources, messaging, storage, keyvault, host resources)
- All configuration options (hooks, healthchecks, test configs, logs configs, etc.)
- All service types and modes (http, tcp, process, container; watch, build, daemon, task)
- All healthcheck types (http, tcp, process, output, none)

## Project Structure

```
editor-e2e-test/
├── azure.yaml                    # Comprehensive azure.yaml exercising ALL features
├── README.md                      # This file
├── src/                          # Minimal source files for services
│   ├── web/                      # Web service (appservice)
│   ├── api/                      # API service (containerapp)
│   ├── function/                 # Function service
│   ├── containerapp/             # Container app service
│   ├── aks-service/              # AKS service
│   ├── ai-endpoint/              # AI endpoint service
│   ├── ai-agent/                 # Azure AI Agent service
│   ├── spring-service/           # Spring app service
│   ├── static-site/              # Static web app service
│   ├── typescript-watcher/       # Process service (watch mode)
│   ├── build-service/            # Process service (build mode)
│   ├── daemon-service/            # Process service (daemon mode)
│   ├── migration-task/           # Process service (task mode)
│   ├── database-proxy/           # TCP service
│   └── healthcheck-demo/         # Service demonstrating healthchecks
├── scripts/                      # Hook scripts
│   ├── pre-provision.sh
│   ├── post-provision-1.sh
│   └── post-provision-2.sh
└── infra/                        # Infrastructure templates
    └── main.bicep
```

## Features Tested

### Top-Level Properties
- ✅ `name` - Application name with pattern validation
- ✅ `resourceGroup` - Resource group with environment variable substitution
- ✅ `metadata` - Template metadata
- ✅ `infra` - Infrastructure configuration (bicep/terraform, path, module)
- ✅ `pipeline` - CI/CD pipeline configuration (github/azdo, variables, secrets)
- ✅ `requiredVersions` - azd and extension version constraints
- ✅ `state` - Remote state configuration (AzureBlobStorage)
- ✅ `platform` - Platform configuration (devcenter)
- ✅ `workflows` - Workflow overrides (custom up workflow)
- ✅ `cloud` - Cloud configuration (AzureCloud, AzureChinaCloud, AzureUSGovernment)
- ✅ `reqs` - Prerequisites/requirements with all properties
- ✅ `logs` - Global logging configuration (filters, classifications, analytics)
- ✅ `test` - Global test configuration (parallel, failFast, coverage, outputDir, outputFormat)
- ✅ `hooks` - Project-level hooks (all hook types with all properties)

### Services
All service host types are included:
- ✅ **appservice** - With all properties (language, project, module, dist, env, hooks, test, logs, local, azure)
- ✅ **containerapp** - With project and image variants, docker config, apiVersion
- ✅ **function** - With project, language, hooks
- ✅ **springapp** - With project, language
- ✅ **staticwebapp** - With project
- ✅ **aks** - With docker and k8s config (deploymentPath, namespace, deployment, service, ingress, helm, kustomize)
- ✅ **ai.endpoint** - With config (workspace, flow, environment, model, deployment)
- ✅ **azure.ai.agent** - With project and config

Service properties exercised:
- ✅ All service types: `http`, `tcp`, `process`, `container`
- ✅ All service modes: `watch`, `build`, `daemon`, `task`
- ✅ All healthcheck types: `http`, `tcp`, `process`, `output`, `none` (with all properties)
- ✅ Service-level hooks: predeploy, postdeploy, prerestore, postrestore, prebuild, postbuild, prepackage, postpackage, prepublish, postpublish
- ✅ Service-level test configs: unit, integration, e2e with all properties
- ✅ Service-level logs configs: filters, classifications, analytics
- ✅ Local and Azure custom URLs/domains
- ✅ Port configurations (single port, port mapping, host:port:container, UDP)
- ✅ Environment variables (plain values, secrets, array and object formats)
- ✅ Service dependencies (`uses`)

### Resources
All resource types are included:
- ✅ **db.postgres** - Generic database resource
- ✅ **db.mysql** - Generic database resource
- ✅ **db.redis** - Generic database resource
- ✅ **db.mongo** - Generic database resource
- ✅ **db.cosmos** - Cosmos DB with containers array
- ✅ **ai.openai.model** - AI model resource with model config
- ✅ **ai.project** - AI project with models array (name, version, format, sku)
- ✅ **ai.search** - AI search resource
- ✅ **host.containerapp** - Host containerapp with port and env
- ✅ **host.appservice** - Host appservice with port, runtime, startupCommand, env
- ✅ **messaging.eventhubs** - Event Hubs with hubs array
- ✅ **messaging.servicebus** - Service Bus with queues and topics arrays
- ✅ **storage** - Storage account with containers array
- ✅ **keyvault** - Key Vault resource
- ✅ Resources with `uses` dependencies
- ✅ Resources with `existing: true` flag

### Hooks
- ✅ All project-level hooks: preprovision, postprovision, preinfracreate, postinfracreate, preinfradelete, postinfradelete, predown, postdown, preup, postup, prepackage, postpackage, prepublish, postpublish, prerestore, postrestore, prerun, postrun
- ✅ Hook properties: run (inline and file paths), shell, continueOnError, interactive
- ✅ Platform-specific overrides: windows, posix
- ✅ Hooks as arrays and single objects

### Health Checks
- ✅ HTTP healthcheck with path, interval, timeout, retries, start_period, start_interval
- ✅ TCP healthcheck
- ✅ Process healthcheck
- ✅ Output healthcheck with pattern matching
- ✅ Disabled healthcheck
- ✅ All healthcheck properties

### Test Configuration
- ✅ Global test config: parallel, failFast, coverage (enabled, threshold, exclude, include), outputDir, outputFormat
- ✅ Service-level test config: unit, integration, e2e with command, args, path, pattern, env, timeout
- ✅ Service-level coverage config

### Logs Configuration
- ✅ Global logs config: filters (exclude patterns), classifications (text, level), analytics (workspace, pollingInterval, defaultTimespan, realtime)
- ✅ Service-level logs config: filters, classifications, analytics (tables, query)

## Running the E2E Tests

The comprehensive e2e tests are located in `cli/dashboard/e2e/azure-yaml-editor-comprehensive.spec.ts`.

### Prerequisites
- Node.js and pnpm installed
- Playwright installed (`pnpm install` in `cli/dashboard`)

### Run Tests
```bash
cd cli/dashboard
pnpm test:e2e azure-yaml-editor-comprehensive
```

Or run all e2e tests:
```bash
pnpm test:e2e
```

### Test Categories

The comprehensive test suite includes:

1. **Schema Form Generation** - All field types, validation, arrays, objects, keyboard navigation
2. **Navigation Tree** - All sections, expand/collapse, search, validation badges, keyboard navigation
3. **Service Management** - Add/edit/delete all host types, all properties, dependencies, validation
4. **Resource Management** - Add/edit/delete all resource types, all properties, dependencies
5. **Health Check Configuration** - All types (http, tcp, process, output, none), all properties
6. **Hooks Configuration** - Project-level and service-level, all hook types, all properties
7. **Environment Variables and Ports** - Add/edit/delete, validation, array vs object format
8. **YAML Editor** - Direct editing, syntax validation, error highlighting, navigation to errors
9. **Preview Pane** - Updates, download, formatting
10. **Validation** - Schema validation, business rules, validation levels, summary panel
11. **Import/Export** - Export, import from file/paste/template, merge strategies, cherry-pick
12. **Backup/Restore** - Auto-backup, view history, restore, delete
13. **Save/Load** - Save, load, auto-save, validation, unsaved changes warning
14. **Command Palette** - Open, search, execute, navigate
15. **Keyboard Shortcuts** - All shortcuts work correctly
16. **Error Handling** - Invalid YAML, schema errors, network errors, recovery
17. **Integration Tests** - Complete workflows, complex configurations, round-trip export/import

## Test Fixtures

Test fixtures are located in `cli/dashboard/e2e/fixtures/`:
- `comprehensive-azure-yaml.yaml` - The full test project azure.yaml
- `minimal-azure-yaml.yaml` - Minimal valid azure.yaml
- `invalid-azure-yaml.yaml` - Invalid YAML for error testing
- `schema-violations.yaml` - YAML with schema violations
- `service-configs.json` - Pre-built service configurations for each host type
- `resource-configs.json` - Pre-built resource configurations for each type

## Notes

- This project is **not intended for actual deployment** - it's purely for testing the editor
- Source files are minimal stubs to make the project structure valid
- The azure.yaml file is designed to exercise every schema feature, not to be a realistic project
- All tests use mocked API endpoints - no actual Azure resources are created

## Maintenance

When new features are added to azure.yaml 1.1 schema:
1. Update `azure.yaml` to include the new feature
2. Add corresponding e2e tests in `azure-yaml-editor-comprehensive.spec.ts`
3. Update this README to document the new feature

## Success Criteria

✅ Test project contains azure.yaml that exercises 100% of schema features  
✅ E2E tests cover 100% of editor features with positive and negative test cases  
✅ All tests pass and validate that the editor correctly handles each feature  
✅ Documentation explains test project structure and how to run tests  
✅ Tests are organized, readable, and easy to extend
