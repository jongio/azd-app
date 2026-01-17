# Azure.yaml Schema Comparison: v1.0 (azure/azure-dev) vs v1.1 (jongio/azd-app)

**Analysis Date:** January 11, 2026  
**v1.0 Source:** https://raw.githubusercontent.com/Azure/azure-dev/main/schemas/v1.0/azure.yaml.json  
**v1.1 Source:** https://raw.githubusercontent.com/jongio/azd-app/main/schemas/v1.1/azure.yaml.json

## Executive Summary

The v1.1 schema is a **comprehensive superset** of v1.0, maintaining full feature parity for Azure deployment while adding extensive local development orchestration capabilities. The schema successfully positions azd-app as both a deployment tool and local development environment manager.

**Key Findings:**
- ✅ **100% core feature parity** - All v1.0 deployment features present in v1.1
- 🆕 **15 new major features** - Extensive local development additions
- ⚠️ **1 removed feature** - Hook secrets mapping (requires migration path)
- ⚠️ **3 compatibility concerns** - Validation relaxation, schema version, removed feature

---

## 1. Feature Parity Analysis

### ✅ All v1.0 Core Features Present in v1.1

#### Top-Level Properties (100% Parity)
| Property | v1.0 | v1.1 | Notes |
|----------|------|------|-------|
| `name` | ✅ | ✅ | Identical validation |
| `resourceGroup` | ✅ | ✅ | Identical |
| `metadata` | ✅ | ✅ | Template identifier support |
| `infra` | ✅ | ✅ | provider, path, module |
| `services` | ✅ | ✅ | Enhanced with local dev features |
| `resources` | ✅ | ✅ | All resource types preserved |
| `pipeline` | ✅ | ✅ | github, azdo support |
| `hooks` | ✅ | ✅ | All deployment hooks + new run hooks |
| `requiredVersions` | ✅ | ✅ | azd + extensions constraints |
| `state` | ✅ | ✅ | Remote state with Azure Blob Storage |
| `platform` | ✅ | ✅ | Azure Dev Center integration |
| `workflows` | ✅ | ✅ | Workflow customization |
| `cloud` | ✅ | ✅ | Sovereign clouds support |

#### Service Properties (100% Parity)
All v1.0 service properties retained:
- `apiVersion`, `resourceGroup`, `resourceName`, `project`, `image`
- `host` (appservice, containerapp, function, springapp, staticwebapp, aks, ai.endpoint, azure.ai.agent)
- `language`, `module`, `dist`, `docker`, `k8s`, `config`
- `uses`, `env` (deployment format)
- All service-level hooks (predeploy, postdeploy, prerestore, postrestore, prebuild, postbuild, prepackage, postpackage, prepublish, postpublish)

#### Resource Types (100% Parity)
All v1.0 resource types fully supported:
- Databases: `db.postgres`, `db.mysql`, `db.redis`, `db.mongo`, `db.cosmos`
- AI Services: `ai.openai.model`, `ai.project`, `ai.search`
- Compute: `host.containerapp`, `host.appservice`
- Messaging: `messaging.eventhubs`, `messaging.servicebus`
- Storage: `storage`, `keyvault`

All resource configurations preserved:
- Cosmos DB containers with partition keys
- AI project models with SKU configuration
- App Service runtime stacks
- Event Hubs hubs array
- Service Bus queues and topics

#### Infrastructure as Code (100% Parity)
- Docker: path, context, platform, registry, image, tag, buildArgs, remoteBuild
- Kubernetes: deploymentPath, namespace, deployment, service, ingress, helm, kustomize
- AI Endpoints: workspace, flow, environment, model, deployment

---

## 2. New azd-app Features (v1.1 Only)

### 2.1 Prerequisites System (`reqs`)
**Purpose:** Validate required tools before running services

```yaml
reqs:
  - name: node
    minVersion: "18.0.0"
  - name: python
    minVersion: "3.9"
  - name: docker
    checkRunning: true
  - name: mytool
    minVersion: "1.0.0"
    command: mytool
    args: ["--version"]
    versionPrefix: "v"
    versionField: 1
    installUrl: "https://example.com/install"
```

**Properties:**
- `name`: Tool name (built-in: node, python, docker, azd, dotnet, go, java)
- `minVersion`: Minimum version constraint (semver)
- `command`: Override version check command
- `args`: Override version check arguments
- `versionPrefix`: Strip prefix from version output (e.g., 'v')
- `versionField`: Which field contains version (0 = whole output)
- `checkRunning`: Verify tool is running (e.g., Docker daemon)
- `runningCheckCommand`, `runningCheckArgs`, `runningCheckExpected`, `runningCheckExitCode`
- `installUrl`: Installation URL shown when check fails

**Use Case:** Ensure development environment has correct tools before `azd app run`

---

### 2.2 Logging System (`logs`)

#### 2.2.1 Project-Level Logging
**Purpose:** Global logging configuration for all services

```yaml
logs:
  filters:
    exclude:
      - "npm warn"
      - "Debugger listening"
      - "ExperimentalWarning"
  classifications:
    - text: "DEPRECATED"
      level: "warning"
    - text: "fatal"
      level: "error"
  analytics:
    workspace: "/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.OperationalInsights/workspaces/{name}"
    pollingInterval: "10s"
    defaultTimespan: "30m"
    realtime: false
```

**Features:**
- **Filters:** Regex patterns to suppress noisy output
- **Classifications:** Override log levels based on text matching
- **Analytics:** Azure Log Analytics integration
  - Auto-detect workspace from Azure environment
  - Configurable polling and timespan
  - Realtime streaming when supported

#### 2.2.2 Service-Level Logging
**Purpose:** Override logging for specific services

```yaml
services:
  api:
    logs:
      filters:
        exclude: ["verbose debug"]
      analytics:
        tables: ["ContainerAppConsoleLogs_CL", "ContainerAppSystemLogs_CL"]
        # OR custom query:
        query: |
          FunctionAppLogs
          | where FunctionName == '{serviceName}'
          | where TimeGenerated > ago({timespan})
```

**Features:**
- Service-specific filters and classifications
- Custom Log Analytics tables
- Custom KQL queries with {serviceName} and {timespan} placeholders

---

### 2.3 Health Checks (`healthcheck`)

**Purpose:** Monitor service readiness and health

#### 2.3.1 Five Health Check Types

**1. HTTP (Default for services with ports)**
```yaml
healthcheck:
  type: http
  path: /health
  interval: 30s
  timeout: 30s
  retries: 3
  start_period: 0s
```

**2. TCP (Port connectivity)**
```yaml
healthcheck:
  type: tcp
  interval: 10s
  timeout: 5s
  retries: 3
```

**3. Process (Default for services without ports)**
```yaml
healthcheck:
  type: process
  interval: 30s
```

**4. Output (Pattern matching for watch mode)**
```yaml
healthcheck:
  type: output
  pattern: "Found 0 errors. Watching for file changes."
  interval: 5s
```

**5. None (Disabled)**
```yaml
healthcheck:
  type: none
  # OR
  disable: true
  # OR
healthcheck: false
```

#### 2.3.2 Docker Compose Compatibility
Supports Docker Compose `test` format:
```yaml
healthcheck:
  test: "http://localhost:8080/health"
  # OR
  test: "curl -f http://localhost/health || exit 1"
  # OR
  test: ["CMD", "curl", "-f", "http://localhost/health"]
  # OR
  test: ["CMD-SHELL", "curl -f http://localhost/health || exit 1"]
```

---

### 2.4 Service Types and Modes

#### 2.4.1 Service Types (`type`)
Defines how the service is accessed:

- **`http`** - HTTP/HTTPS services (default if ports defined)
- **`tcp`** - Raw TCP connections (databases, message queues)
- **`process`** - No network endpoint (default if no ports)
- **`container`** - Docker container services (auto-detected if image is set)

#### 2.4.2 Run Modes (`mode`)
For process-type services:

- **`daemon`** - Long-running background process (default)
- **`watch`** - Continuous file-watching (tsc --watch, nodemon)
- **`build`** - One-time build that exits
- **`task`** - One-time tasks run on demand

**Example:**
```yaml
services:
  tsc-watch:
    type: process
    mode: watch
    command: "tsc --watch"
    healthcheck:
      type: output
      pattern: "Found 0 errors"
```

---

### 2.5 Local Development Service Configuration

#### 2.5.1 Port Mappings (`ports`)
Docker Compose-style port definitions:

```yaml
ports:
  - "3000"                    # Container port only
  - "3000:8080"              # Host:Container
  - "127.0.0.1:3000:8080"    # IP:Host:Container
  - "8080/udp"               # UDP protocol
  - "[::1]:3000:8080"        # IPv6
```

#### 2.5.2 Entry Point and Command
```yaml
entrypoint: "main.py"                    # Entry point file
command: "uvicorn main:app --reload"     # Full run command
```

**Precedence:** `command` > auto-detected run command > default language runtime

#### 2.5.3 Environment Variables (`environment`)
Two formats supported:

**Array format (Docker Compose-compatible):**
```yaml
environment:
  - name: API_URL
    value: "http://localhost:8000"
  - name: DATABASE_URL
    secret: "POSTGRES_CONNECTION_STRING"
```

**Object format (simple):**
```yaml
environment:
  API_URL: "http://localhost:8000"
  DATABASE_URL: "${POSTGRES_CONNECTION_STRING}"
```

**Legacy `env` (deployment only):**
```yaml
env:
  VAR_NAME: "value"
```

---

### 2.6 Testing Framework (`test`)

#### 2.6.1 Global Test Configuration
```yaml
test:
  parallel: true
  failFast: false
  coverage:
    enabled: true
    threshold: 80
    exclude: ["**/vendor/**", "**/test/**"]
  outputDir: "./test-results"
  outputFormat: "junit"  # default, json, junit, github
```

#### 2.6.2 Service-Level Test Configuration
```yaml
services:
  api:
    test:
      unit:
        command: "pytest"
        path: "tests/unit"
        pattern: "test_*.py"
        env:
          TESTING: "true"
        timeout: "5m"
      integration:
        command: "pytest"
        path: "tests/integration"
        timeout: "10m"
      e2e:
        path: "tests/e2e"
        timeout: "15m"
      coverage:
        enabled: true
        threshold: 85
        exclude: ["**/migrations/**"]
        include: ["src/**"]
```

**Test Types:** `unit`, `integration`, `e2e`
**Output Formats:** default, json, junit, github

---

### 2.7 Enhanced Hooks

#### 2.7.1 New Run Hooks
```yaml
hooks:
  prerun:
    run: "./scripts/prerun.sh"
    shell: sh
  postrun:
    run: "echo 'All services ready!'"
    shell: sh
```

#### 2.7.2 Expanded Shell Types
v1.0: `sh`, `pwsh`  
v1.1: `sh`, `bash`, `pwsh`, `powershell`, `cmd`

#### 2.7.3 Hook Environment Variables
All hooks receive:
- `AZD_APP_PROJECT_DIR` - Project directory path
- `AZD_APP_PROJECT_NAME` - Project name
- `AZD_APP_SERVICE_COUNT` - Number of services

---

## 3. Missing Features (v1.0 → v1.1)

### ⚠️ Removed: Hook Secrets Mapping

**v1.0 Feature:**
```yaml
hooks:
  predeploy:
    run: "./deploy.sh"
    secrets:
      WITH_SECRET_VALUE: "ENV_VAR_WITH_SECRET"
```

**Description:** Map azd environment variables to hook secrets. If variable was set as a secret in the environment, the secret value would be passed to the hook.

**Status in v1.1:** ❌ **REMOVED**

**Impact:** 
- **Severity:** MEDIUM-HIGH
- Users relying on this for secure secret passing to hooks will need migration
- No direct replacement mechanism in v1.1

**Recommendation:**
1. **Restore feature** with deprecation warning, OR
2. **Document migration path** (e.g., use environment variables directly, use Azure Key Vault references)
3. **Consider adding** to v1.1 roadmap if community needs it

---

## 4. Implementation Differences

### 4.1 Schema Validation Strictness

| Aspect | v1.0 | v1.1 | Impact |
|--------|------|------|--------|
| Root `additionalProperties` | `false` (strict) | `true` (permissive) | ⚠️ Breaking Change |
| Service `additionalProperties` | `false` (strict) | `true` (permissive) | ⚠️ Breaking Change |
| Resource `additionalProperties` | `false` (strict) | `false` (strict) | ✅ Consistent |

**Impact:**
- **v1.0 behavior:** Unknown properties cause validation errors
- **v1.1 behavior:** Unknown properties are allowed (forward compatibility)
- **Breaking Change:** v1.0 configs with typos might pass validation in v1.1
- **Benefit:** Easier to add custom/experimental properties

**Recommendation:** Consider adding a "strict mode" flag or warning for unknown properties.

---

### 4.2 JSON Schema Version

| Schema | v1.0 | v1.1 |
|--------|------|------|
| Version | draft/2019-09 | draft-07 |
| $id | azure-dev repo | azd-app repo |
| Title | Not present | "Azure Developer CLI (azd app) Configuration" |
| Description | Not present | "Schema for azure.yaml configuration file used by azd app for local development orchestration. This is a superset of the v1.0 schema with additional properties for local development." |

**Impact:**
- **Different validators** might behave slightly differently
- **Better documentation** in v1.1 with title/description
- **Low risk** - both schema versions are compatible for features used

**Recommendation:** Test with popular JSON Schema validators (ajv, jsonschema.net)

---

### 4.3 Schema Organization

**v1.0:** Inline service and resource definitions  
**v1.1:** Uses `$ref` to `#/definitions/service` and `#/definitions/resource`

**Impact:**
- ✅ Better code organization
- ✅ Easier maintenance
- ✅ No behavioral difference

---

### 4.4 Environment Variable Formats

| Format | v1.0 | v1.1 | Purpose |
|--------|------|------|---------|
| `env` (object) | ✅ | ✅ | Azure deployment |
| `environment` (object) | ❌ | ✅ | Local dev (simple) |
| `environment` (array) | ❌ | ✅ | Local dev (with secrets) |

**v1.1 Strategy:**
- `env` - Used for Azure deployment (existing v1.0 behavior)
- `environment` - Used for local development (`azd app run`)
  - Object format: Simple key-value
  - Array format: Supports secret references

**Impact:**
- ✅ Backward compatible (env still works)
- ✅ Clear separation of concerns
- ⚠️ Potential confusion about which to use

**Recommendation:** Document when to use each format clearly.

---

### 4.5 Hook Platform Overrides

**v1.0:** `windows` and `posix` can be full hook objects (allowing nested overrides)  
**v1.1:** New `platformHookOverride` type prevents nested platform overrides

**Example:**
```yaml
# v1.1 prevents this nested structure:
hooks:
  predeploy:
    windows:
      windows:  # ❌ Not allowed in v1.1
        run: "..."
```

**Impact:**
- ✅ Cleaner schema, prevents confusion
- ⚠️ Potential edge case breaking change (unlikely anyone used nested overrides)

---

## 5. Critical Issues and Recommendations

### 🔴 Critical: Removed Hook Secrets

**Issue:** v1.0's `hooks.secrets` completely removed in v1.1  
**Severity:** MEDIUM-HIGH  
**Impact:** Silent configuration loss for users migrating from v1.0  

**Recommendations:**
1. **SHORT TERM:** Add to v1.1 with deprecation warning
2. **MEDIUM TERM:** Provide migration guide to alternatives
3. **LONG TERM:** Determine if feature is needed based on community feedback

**Alternatives:**
- Direct environment variable usage: `run: "WITH_VAR=$ENV_VAR ./script.sh"`
- Azure Key Vault integration
- Secret management service integration

---

### ⚠️ Warning: Validation Relaxation

**Issue:** `additionalProperties: false` → `true`  
**Severity:** MEDIUM  
**Impact:** Typos and mistakes might not be caught  

**Recommendations:**
1. Add lint/validation mode that warns on unknown properties
2. Document all supported properties clearly
3. Consider adding JSON Schema $comment fields for unknown properties

**Example User Impact:**
```yaml
# v1.0: Error on typo
services:
  api:
    prot: "./api"  # ❌ Unknown property error

# v1.1: Silently accepted
services:
  api:
    prot: "./api"  # ⚠️ No error, typo not caught
```

---

### ⚠️ Warning: Dual Environment Systems

**Issue:** Both `env` and `environment` exist  
**Severity:** LOW-MEDIUM  
**Impact:** User confusion  

**Recommendations:**
1. **Clear documentation** on when to use each:
   - `env` → Azure deployment (containers, functions, app services)
   - `environment` → Local development (`azd app run`)
2. Consider migration path to unify in future version
3. Add validation warning if both are specified

---

### ✅ Low Risk: JSON Schema Version

**Issue:** draft-07 vs draft/2019-09  
**Severity:** LOW  
**Impact:** Mostly compatible  

**Recommendations:**
1. Test with popular validators
2. Document schema version in README
3. Consider specifying validator version requirements

---

## 6. Migration Checklist

### For Users Moving from v1.0 to v1.1

✅ **Safe migrations:**
- All existing v1.0 configs should work
- All Azure resource types supported
- All deployment hooks preserved
- Infrastructure definitions unchanged

⚠️ **Review needed:**
1. **Hook secrets:** If using `hooks.secrets`, migrate to alternative approach
2. **Unknown properties:** v1.1 won't error on typos - validate configs carefully
3. **Test with v1.1:** Verify deployment still works

🆕 **New features to consider:**
1. Add `reqs` to validate developer environment
2. Add `healthcheck` for better local development experience
3. Add `logs.filters` to suppress noisy output
4. Add `test` configuration for automated testing
5. Use `environment` for local dev environment variables
6. Add `prerun`/`postrun` hooks for local development setup

---

## 7. Schema Enhancement Recommendations

### 7.1 Immediate Actions (Required for v1.1 Completeness)

1. **Restore Hook Secrets** (or document removal)
   ```yaml
   hooks:
     predeploy:
       run: "./deploy.sh"
       secrets:  # Add back to v1.1
         WITH_SECRET: "ENV_VAR_NAME"
   ```

2. **Add Validation Warnings** for unknown properties
   - Implement in CLI: warn when unknown properties detected
   - Help users catch typos

3. **Document Migration Guide**
   - Create v1.0 → v1.1 migration guide
   - Highlight breaking changes
   - Provide alternatives for removed features

### 7.2 Schema Documentation Improvements

1. **Add `$comment` fields** for commonly confused properties
   ```json
   "env": {
     "$comment": "For Azure deployment. Use 'environment' for local development.",
     "type": "object"
   }
   ```

2. **Improve examples** section
   - Add migration examples
   - Add common patterns
   - Add troubleshooting scenarios

3. **Create decision trees**
   - Which environment variable format to use
   - Which health check type to use
   - When to use service types

### 7.3 Future Enhancements

1. **Unify environment variable systems** (v1.2+)
   - Single `environment` property
   - Context-aware usage (deployment vs local)

2. **Add schema versioning** within azure.yaml
   ```yaml
   schema_version: "1.1"
   name: my-app
   ```

3. **Add validation modes**
   ```yaml
   validation:
     strict: true  # Error on unknown properties
     deprecation_warnings: true
   ```

4. **Enhanced cross-referencing**
   - Validate `uses` references exist
   - Validate resource types match infrastructure

---

## 8. Testing Strategy

### 8.1 Compatibility Tests

Create test suite with:

1. **v1.0 Baseline Configs**
   - Simple app service deployment
   - Container app with docker
   - AKS deployment
   - AI endpoint deployment
   - Multi-service application

2. **v1.1 Round-trip Tests**
   - Load v1.0 config → validate with v1.1 schema
   - Deploy to Azure using v1.1 tooling
   - Verify no regressions

3. **New Feature Tests**
   - Health checks (all 5 types)
   - Prerequisites validation
   - Logging configuration
   - Test framework integration

### 8.2 Schema Validation Tests

1. **Validator Compatibility**
   - Test with ajv (JavaScript)
   - Test with jsonschema (Python)
   - Test with VS Code JSON Schema validator

2. **Error Message Quality**
   - Verify helpful error messages
   - Test validation of nested properties
   - Test allOf constraints

---

## 9. Conclusion

### Summary Table

| Category | Status | Details |
|----------|--------|---------|
| **Feature Parity** | ✅ Excellent | 100% of v1.0 deployment features present |
| **New Features** | ✅ Extensive | 15+ major features for local development |
| **Missing Features** | ⚠️ 1 Removed | Hook secrets (requires migration) |
| **Breaking Changes** | ⚠️ 3 Concerns | Secrets, validation, schema version |
| **Documentation** | ✅ Good | Examples provided, needs migration guide |
| **Recommendation** | ✅ **Approve with fixes** | Address hook secrets, add warnings |

### Final Recommendation

**The v1.1 schema is production-ready with minor fixes:**

✅ **Approve after:**
1. Restore `hooks.secrets` OR document removal with migration path
2. Add CLI warnings for unknown properties
3. Create v1.0 → v1.1 migration guide
4. Test with popular JSON Schema validators

🎯 **Strategic Value:**
- v1.1 successfully extends v1.0 for local development
- Maintains backward compatibility for deployment scenarios
- Positions azd-app as comprehensive DevOps tool
- Docker Compose familiarity lowers barrier to entry

📊 **Risk Assessment:**
- **Low Risk:** Core deployment features unchanged
- **Medium Risk:** Validation relaxation might hide errors
- **Mitigated:** Documentation and CLI warnings can address concerns

---

## Appendix: Quick Reference

### v1.1 New Properties Checklist

**Top-Level:**
- [ ] `reqs` - Prerequisites
- [ ] `logs` - Logging configuration
- [ ] `test` - Test configuration

**Service-Level:**
- [ ] `entrypoint` - Entry point file
- [ ] `command` - Run command
- [ ] `type` - Service type (http/tcp/process/container)
- [ ] `mode` - Run mode (watch/build/daemon/task)
- [ ] `ports` - Port mappings
- [ ] `environment` - Environment variables (array/object)
- [ ] `healthcheck` - Health check configuration
- [ ] `logs` - Service-level logging
- [ ] `test` - Service-level testing

**Hooks:**
- [ ] `prerun` - Pre-run hook
- [ ] `postrun` - Post-run hook
- [ ] Expanded shells: bash, powershell, cmd

### Common Migration Patterns

**Environment Variables:**
```yaml
# v1.0 (deployment)
env:
  VAR: "value"

# v1.1 (local + deployment)
env:
  VAR: "value"  # Still works for deployment
environment:    # NEW: for local development
  - name: VAR
    value: "value"
```

**Health Checks:**
```yaml
# v1.1 NEW feature
healthcheck:
  type: http
  path: /health
  interval: 10s
  retries: 3
```

**Prerequisites:**
```yaml
# v1.1 NEW feature
reqs:
  - name: node
    minVersion: "18.0.0"
  - name: docker
    checkRunning: true
```

---

**Document Version:** 1.0  
**Last Updated:** January 11, 2026  
**Reviewers Needed:** azd-app team, azure-dev team, community feedback
