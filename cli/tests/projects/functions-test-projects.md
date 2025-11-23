# Azure Functions Test Projects

This directory contains comprehensive test projects for validating Azure Functions support in `azd app run`.

## Overview

### Prerequisites Validation Coverage

Each test project MUST validate `azd app reqs` detects correct prerequisites:

| Variant | Required Tools | Version Requirements |
|---------|---------------|---------------------|
| **All Variants** | Azure Functions Core Tools | 4.0.0+ |
| **Logic Apps** | func CLI | 4.0.0+ |
| **Logic Apps** | .NET SDK (optional) | 6.0 or 8.0 (for local debugging) |
| **Node.js v4** | func + Node.js + npm | func 4.0+, Node 18.x/20.x |
| **TypeScript v4** | func + Node.js + npm + TypeScript | func 4.0+, Node 18.x/20.x, tsc 4.x/5.x |
| **Node.js v3** | func + Node.js + npm | func 4.0+, Node 18.x/20.x |
| **Python v2** | func + Python + pip | func 4.0+, Python 3.9/3.10/3.11 |
| **Python v1** | func + Python + pip | func 4.0+, Python 3.9/3.10/3.11 |
| **.NET Isolated** | func + .NET SDK + dotnet | func 4.0+, .NET 6.0/8.0 |
| **.NET Durable** | func + .NET SDK + dotnet | func 4.0+, .NET 6.0/8.0 |
| **.NET In-Process** | func + .NET SDK + dotnet | func 4.0+, .NET 6.0 |
| **.NET 8** | func + .NET SDK + dotnet | func 4.0+, .NET 8.0 |
| **Java Maven** | func + Java + Maven | func 4.0+, Java 11/17, Maven 3.6+ |
| **Java Gradle** | func + Java + Gradle | func 4.0+, Java 11/17, Gradle 7.0+ |
| **Multi-App** | All language runtimes + func | All versions for all services |

**Prerequisites Detection Tests**:
- ✅ `azd app reqs` detects all required tools for each variant
- ✅ `azd app reqs` validates tool versions meet minimum requirements
- ✅ `azd app reqs` provides OS-specific installation instructions for missing tools
- ✅ `azd app reqs` groups prerequisites by service in multi-app workspaces
- ✅ `azd app reqs` shows clear summary (all met / X missing)

**Total Test Projects**: 17+ projects covering:

**Purpose**: Ensure `azd app` correctly detects, configures, and runs all Azure Functions project types

**Related Documentation**:
- [Azure Functions Spec](../../docs/dev/azure-functions/spec.md)
- [Azure Functions Design](../../docs/dev/azure-functions/design.md)
- [Development Tracker](../../docs/dev/azure-functions/dev-tracker.md)

---

## Test Projects by Category

### Logic Apps Standard

| Project | Programming Model | Triggers | Prerequisites |
|---------|------------------|----------|---------------|
| `logicapp-test/` | Logic Apps Standard | HTTP workflow | func CLI (+ optional .NET SDK) |
| `logicapp-ai-agent-style/` | Logic Apps + AI Foundry | HTTP workflow + AI agent | func CLI + .NET SDK (for complex infra) |

**Note**: `logicapp-ai-agent-style/` validates real-world scenario with:
- AI Foundry integration (GPT-4o deployment)
- Managed identity (user-assigned + system-assigned)
- Role-based access control (Cognitive Services Contributor, Azure AI Administrator, Azure AI User)
- Key Vault integration
- Environment-based configuration (dev/tst/acc/prd)
- Modular Bicep infrastructure
- Connection configuration for AI Foundry API

### Node.js/TypeScript Functions

| Project | Programming Model | Triggers | Prerequisites |
|---------|------------------|----------|---------------|
| `functions-nodejs-v4/` | Node.js v4 (default) | HTTP, Timer | func + Node.js 18.x/20.x + npm |
| `functions-typescript-v4/` | TypeScript v4 | HTTP (typed) | func + Node.js + npm + TypeScript |
| `functions-nodejs-v3/` | Node.js v3 (legacy) | HTTP | func + Node.js + npm |

### Python Functions

| Project | Programming Model | Triggers | Prerequisites |
|---------|------------------|----------|---------------|
| `functions-python-v2/` | Python v2 (recommended) | HTTP, Timer, Blob | func + Python 3.9/3.10/3.11 + pip |
| `functions-python-v1/` | Python v1 (legacy) | HTTP, Timer | func + Python + pip |

### .NET Functions

| Project | Programming Model | Triggers | Prerequisites |
|---------|------------------|----------|---------------|
| `functions-dotnet-isolated/` | .NET Isolated Worker | HTTP, Timer | func + .NET SDK 6.0/8.0 + dotnet |
| `functions-dotnet-isolated-durable/` | .NET Isolated + Durable | Orchestrator, Activity, HTTP | func + .NET SDK + dotnet |
| `functions-dotnet-inprocess/` | .NET In-Process (legacy) | HTTP | func + .NET SDK + dotnet |
| `functions-dotnet-8/` | .NET 8 Isolated | HTTP | func + .NET SDK 8.0 + dotnet |

### Java Functions

| Project | Programming Model | Triggers | Prerequisites |
|---------|------------------|----------|---------------|
| `functions-java-maven/` | Java + Maven | HTTP, Timer | func + Java 11/17 + Maven 3.6+ |
| `functions-java-gradle/` | Java + Gradle | HTTP | func + Java 11/17 + Gradle 7.0+ |

### Multi-Language Scenarios

| Project | Services | Prerequisites |
|---------|----------|---------------|
| `functions-multi-app/` | Node.js + Python + .NET | func + Node.js + npm + Python + pip + .NET SDK + dotnet |

### Error Scenarios

| Project | Error Type | Prerequisites |
|---------|------------|---------------|
| `functions-invalid-no-host/` | Missing host.json | N/A (error test) |
| `functions-invalid-no-functions/` | No functions defined | N/A (error test) |
| `functions-invalid-corrupt-host/` | Corrupt host.json | N/A (error test) |
| `functions-minimal/` | Minimal valid project | func + Node.js + npm |

---

## Testing Matrix

### Language Coverage

| Language | v1/Legacy Model | v2/Modern Model | Build Tool |
|----------|----------------|-----------------|------------|
| Logic Apps | ✅ logicapp-test | N/A | .NET/func |
| Logic Apps + AI | ✅ logicapp-ai-agent-style | N/A | .NET/func |
| Node.js | ✅ functions-nodejs-v3 | ✅ functions-nodejs-v4 | npm |
| TypeScript | N/A | ✅ functions-typescript-v4 | npm |
| Python | ✅ functions-python-v1 | ✅ functions-python-v2 | pip |
| .NET In-Process | ✅ functions-dotnet-inprocess | N/A | dotnet |
| .NET Isolated | N/A | ✅ functions-dotnet-isolated | dotnet |
| .NET Durable | N/A | ✅ functions-dotnet-isolated-durable | dotnet |
| .NET 8 | N/A | ✅ functions-dotnet-8 | dotnet |
| Java | N/A | ✅ functions-java-maven | Maven |
| Java | N/A | ✅ functions-java-gradle | Gradle |

### Trigger Type Coverage

| Trigger Type | Node.js | TypeScript | Python | .NET | Java |
|--------------|---------|------------|--------|------|------|
| HTTP | ✅ v4, v3 | ✅ v4 | ✅ v2, v1 | ✅ All | ✅ All |
| Timer | ✅ v4 | ❌ | ✅ v2, v1 | ✅ Isolated | ✅ Maven |
| Blob | ❌ | ❌ | ✅ v2 | ❌ | ❌ |
| Queue | ❌ | ❌ | ❌ | ❌ | ❌ |
| Orchestrator | ❌ | ❌ | ❌ | ✅ Durable | ❌ |
| Activity | ❌ | ❌ | ❌ | ✅ Durable | ❌ |

---

## Project Structure Templates

### Node.js v4 Functions
```
functions-nodejs-v4/
├── host.json                  # Functions host configuration
├── package.json               # npm dependencies (@azure/functions v4)
├── src/
│   └── functions/
│       ├── httpTrigger.js    # HTTP GET/POST handler
│       └── timerTrigger.js   # Timer trigger (every 5 min)
└── README.md                  # Project documentation
```

### TypeScript v4 Functions
```
functions-typescript-v4/
├── host.json
├── package.json               # TypeScript + @azure/functions v4
├── tsconfig.json              # TypeScript configuration
├── src/
│   └── functions/
│       └── httpTrigger.ts    # Typed HTTP handler
└── README.md
```

### Python v2 Functions
```
functions-python-v2/
├── host.json
├── requirements.txt           # azure-functions
├── function_app.py            # Decorator-based functions
│                              # - @app.route() for HTTP
│                              # - @app.schedule() for Timer
│                              # - @app.blob_trigger() for Blob
└── README.md
```

### .NET Isolated Functions
```
functions-dotnet-isolated/
├── host.json
├── FunctionApp.csproj         # Microsoft.Azure.Functions.Worker
├── Program.cs                 # Host builder setup
├── Functions.cs               # Function definitions
│                              # - HTTP trigger
│                              # - Timer trigger
└── README.md
```

### .NET Isolated Durable Functions
```
functions-dotnet-isolated-durable/
├── host.json                  # With durableTask extension config
├── DurableFunctionApp.csproj  # Durable Task extension
├── Program.cs
├── Orchestrators.cs           # [Function] orchestrator
├── Activities.cs              # [Function] activities
├── HttpStarters.cs            # HTTP starter
└── README.md
```

### Java Maven Functions
```
functions-java-maven/
├── host.json
├── pom.xml                    # azure-functions-maven-plugin
└── src/
    └── main/
        └── java/
            └── com/
                └── example/
                    ├── Function.java      # @FunctionName HTTP
                    └── TimerFunction.java # @FunctionName Timer
```

### Multi-App Workspace
```
functions-multi-app/
├── azure.yaml                 # Defines 3 services
├── functions-api/             # Node.js v4
│   ├── host.json
│   ├── package.json
│   └── src/functions/api.js
├── functions-worker/          # Python v2
│   ├── host.json
│   ├── requirements.txt
│   └── function_app.py
└── functions-processor/       # .NET Isolated
    ├── host.json
    ├── Processor.csproj
    ├── Program.cs
    └── Functions.cs
```

### Logic Apps AI Agent Pattern
```
logicapp-ai-agent-style/
├── azure.yaml                 # Service: workflows (csharp/function)
├── src/                       # Logic App workflows source
│   └── workflows/
│       └── [workflow-name]/
│           ├── workflow.json  # Workflow definition
│           └── ...
├── host.json                  # Azure Functions v4 runtime
├── connections.json           # AI Foundry API connection
└── infra/                     # Modular Bicep infrastructure
    ├── main.bicep             # Main orchestration (subscription scope)
    ├── resources.bicep        # Resource collection module
    ├── monitoring.bicep       # Application Insights + Log Analytics
    ├── abbreviations.json     # Resource naming conventions
    ├── aifoundry/
    │   ├── aifoundry.bicep    # AI Foundry + GPT-4o deployment
    │   └── role-assignment.bicep  # RBAC for Logic App
    ├── storage/
    │   └── storage.bicep      # Storage with role assignments
    ├── keyvault/
    │   └── keyvault.bicep     # Key Vault with RBAC
    └── logicapp/
        ├── plan.bicep         # App Service Plan (WS1)
        └── workflows.bicep    # Logic App site + identities
```

**Key Characteristics**:
- **AI Integration**: AI Foundry instance with GPT-4o model deployment
- **Managed Identities**: User-assigned + system-assigned identities
- **RBAC**: Cognitive Services Contributor, Azure AI Administrator, Azure AI User roles
- **Environment Support**: dev/tst/acc/prd environments
- **App Settings**: AI_FOUNDRY_NAME, AI_FOUNDRY_ENDPOINT, AI_PROJECT_NAME, AI_PROJECT_ENDPOINT
- **Connection Config**: AI Foundry API connection in connections.json
- **Modular Bicep**: Separate modules for each resource type
- **azd Workflow**: `azd provision` → `azd deploy` pattern

**Validation Points**:
- ✅ Detection works with complex app settings
- ✅ Supports external service dependencies (AI Foundry)
- ✅ Environment variables passed correctly
- ✅ Prerequisites include func CLI
- ✅ azd provision creates all infrastructure
- ✅ azd deploy works with 'azd-service-name' tag
- ✅ Managed identity authentication to AI services

---

## Test Validation Criteria

### Detection Tests

Each project must:
- ✅ Be correctly detected as an Azure Functions project
- ✅ Have correct variant identified (Logic Apps, Node.js, Python, .NET, Java)
- ✅ Have correct language detected
- ✅ Have correct programming model identified (v1/v2, In-Process/Isolated)

### Runtime Tests

Each project must:
- ✅ Successfully start with `azd app run`
- ✅ Use correct port (default 7071 or assigned port)
- ✅ Pass health check (variant-specific endpoint)
- ✅ Respond to HTTP triggers (if present)
- ✅ Show in dashboard with correct metadata

### Error Tests

Error scenario projects must:
- ✅ Produce helpful error message
- ✅ Clearly state the problem
- ✅ Suggest fix or next steps
- ✅ Not crash or produce confusing errors

---

## Prerequisites for Manual Testing

### Required Tools

| Language | Required Tools | Verify Command |
|----------|---------------|----------------|
| All | Azure Functions Core Tools v4+ | `func --version` |
| Node.js | Node.js 18+ | `node --version` |
| TypeScript | Node.js 18+ + TypeScript | `tsc --version` |
| Python | Python 3.9+ | `python --version` |
| .NET | .NET SDK 6.0+ or 8.0 | `dotnet --version` |
| Java | Java 11+ or 17 | `java -version` |
| Java (Maven) | Maven 3.6+ | `mvn --version` |
| Java (Gradle) | Gradle 7.0+ | `gradle --version` |

### Installation Commands

**Azure Functions Core Tools**:
```powershell
# Windows
winget install Microsoft.Azure.FunctionsCoreTools

# macOS
brew tap azure/functions
brew install azure-functions-core-tools@4

# Linux
curl https://packages.microsoft.com/keys/microsoft.asc | gpg --dearmor > microsoft.gpg
sudo mv microsoft.gpg /etc/apt/trusted.gpg.d/microsoft.gpg
sudo sh -c 'echo "deb [arch=amd64] https://packages.microsoft.com/repos/microsoft-ubuntu-$(lsb_release -cs)-prod $(lsb_release -cs) main" > /etc/apt/sources.list.d/dotnetdev.list'
sudo apt-get update
sudo apt-get install azure-functions-core-tools-4
```

**Language Runtimes**: See language-specific installation guides

---

## Running Manual Tests

### Prerequisites Detection Tests (`azd app reqs`)

#### Single Project - Node.js

```bash
# 1. Navigate to Node.js test project
cd cli/tests/projects/functions-nodejs-v4

# 2. Run prerequisites check
azd app reqs

# 3. Verify output shows:
# ✓ Azure Functions Core Tools: 4.x.x
# ✓ Node.js: v20.x.x (or v18.x.x)
# ✓ npm: 10.x.x
# All prerequisites met.
```

#### Single Project - Python

```bash
cd cli/tests/projects/functions-python-v2
azd app reqs

# Expected output:
# ✓ Azure Functions Core Tools: 4.x.x
# ✓ Python: 3.11.x (or 3.10.x, 3.9.x)
# ✓ pip: 23.x.x
# All prerequisites met.
```

#### Single Project - .NET

```bash
cd cli/tests/projects/functions-dotnet-isolated
azd app reqs

# Expected output:
# ✓ Azure Functions Core Tools: 4.x.x
# ✓ .NET SDK: 8.0.x (or 6.0.x)
# ✓ dotnet CLI: available
# All prerequisites met.
```

#### Single Project - Java Maven

```bash
cd cli/tests/projects/functions-java-maven
azd app reqs

# Expected output:
# ✓ Azure Functions Core Tools: 4.x.x
# ✓ Java: 17.x.x (or 11.x.x)
# ✓ Maven: 3.9.x (or 3.6+)
# All prerequisites met.
```

#### Multi-Language Workspace

```bash
cd cli/tests/projects/functions-multi-app
azd app reqs

# Expected output:
# Checking prerequisites for 3 services...
#
# Service 'functions-api' (Azure Functions - Node.js):
#   ✓ Azure Functions Core Tools: 4.x.x
#   ✓ Node.js: v20.x.x
#   ✓ npm: 10.x.x
#
# Service 'functions-worker' (Azure Functions - Python):
#   ✓ Azure Functions Core Tools: 4.x.x
#   ✓ Python: 3.11.x
#   ✓ pip: 23.x.x
#
# Service 'functions-processor' (Azure Functions - .NET):
#   ✓ Azure Functions Core Tools: 4.x.x
#   ✓ .NET SDK: 8.0.x
#
# All prerequisites met for all services.
```

#### Missing Prerequisites Test

```bash
# 1. Temporarily rename func executable (simulate not installed)
# Windows:
where func  # Note the path
ren "C:\Program Files\Microsoft\Azure Functions Core Tools\func.exe" "func.exe.bak"

# 2. Run prerequisites check
cd cli/tests/projects/functions-nodejs-v4
azd app reqs

# 3. Expected output:
# Checking prerequisites for service 'api' (Azure Functions - Node.js)...
#
# ✗ Azure Functions Core Tools: not found
#   Install: winget install Microsoft.Azure.FunctionsCoreTools
#   Verify: func --version
#
# ✓ Node.js: v20.x.x
# ✓ npm: 10.x.x
#
# 1 prerequisite missing.

# 4. Restore func executable
ren "C:\Program Files\Microsoft\Azure Functions Core Tools\func.exe.bak" "func.exe"
```

#### Old Version Test

```bash
# If you have an old version of func installed (< 4.0)
# This test validates version checking

azd app reqs

# Expected output:
# Checking prerequisites for service 'api'...
#
# ⚠ Azure Functions Core Tools: 3.0.4358 (outdated)
#   Minimum required: 4.0.0
#   Upgrade: winget install Microsoft.Azure.FunctionsCoreTools
#
# ✓ Node.js: v20.x.x
# ✓ npm: 10.x.x
#
# 1 prerequisite needs attention.
```

### Single Project Test

```bash
# 1. Navigate to test project
cd cli/tests/projects/functions-nodejs-v4

# 2. Install dependencies (if needed)
npm install

# 3. Navigate to workspace root
cd ../../..

# 4. Run azd app
azd app run

# 5. Verify
# - Check service starts in dashboard
# - Test HTTP trigger: curl http://localhost:7071/api/httpTrigger
# - Check logs in dashboard
```

### Multi-Project Test

```bash
# 1. Navigate to multi-app project
cd cli/tests/projects/functions-multi-app

# 2. Install dependencies for each service
cd functions-api && npm install && cd ..
cd functions-worker && pip install -r requirements.txt && cd ..
cd functions-processor && dotnet restore && cd ..

# 3. Navigate to workspace root
cd ../../..

# 4. Run all services
azd app run

# 5. Verify
# - All 3 services start
# - Different ports assigned (7071, 7072, 7073)
# - All show in dashboard
# - Each HTTP trigger responds
```

### Error Scenario Test

```bash
# 1. Navigate to error test project
cd cli/tests/projects/functions-invalid-no-host

# 2. Navigate to workspace root
cd ../../..

# 3. Run azd app (expect error)
azd app run

# 4. Verify
# - Error message is helpful
# - Mentions missing host.json
# - Suggests fix
```

---

## Adding New Test Projects

### Steps to Add a Test Project

1. **Create directory** under `cli/tests/projects/`
   ```bash
   mkdir cli/tests/projects/functions-{language}-{variant}
   ```

2. **Add required files**:
   - `host.json` (mandatory)
   - Language-specific files (package.json, requirements.txt, .csproj, pom.xml, etc.)
   - Function definitions
   - `README.md` documenting the project

3. **Add to this document**:
   - Update testing matrix table
   - Add to appropriate category
   - Document validation criteria

4. **Create integration test**:
   - Add test case in `cli/src/internal/service/functions_integration_test.go`
   - Test detection
   - Test runtime execution

5. **Update dev-tracker**:
   - Add checkbox in Phase 1.3
   - Add to testing checklist

### Test Project Checklist

When creating a test project:
- [ ] `host.json` is valid JSON with correct version
- [ ] Functions are properly defined (function.json or language-specific)
- [ ] Project can be built and run with `func start`
- [ ] HTTP trigger included for easy validation
- [ ] README.md explains project purpose and structure
- [ ] Required dependencies listed
- [ ] Minimal but realistic (not toy example)
- [ ] Follows language-specific best practices

---

## CI/CD Integration

### Automated Testing

Test projects are used in:
- **Unit tests**: Validate detection logic
- **Integration tests**: End-to-end detection and runtime
- **Manual tests**: Developer validation during development

### Test Execution

```bash
# Run all Functions-related tests
go test ./cli/src/internal/service/... -v -run Functions

# Run specific variant test
go test ./cli/src/internal/service/... -v -run TestDetectAndRun_NodeJSv4

# Run with coverage
go test ./cli/src/internal/service/... -v -cover -coverprofile=coverage.out
```

### CI Pipeline

GitHub Actions should:
1. Install all required tools (func, node, python, dotnet, java)
2. Install dependencies for all test projects
3. Run full test suite
4. Validate test coverage >80%
5. Report failures with logs

---

## Troubleshooting

### Common Issues

**"func: command not found"**
- Install Azure Functions Core Tools v4+
- Verify: `func --version`

**"No functions found"**
- Ensure host.json exists
- Ensure functions are defined (function.json or code-based)
- Check Functions Core Tools version

**Port conflicts**
- `azd app` should auto-assign different ports
- Verify in dashboard each service has unique port
- Check logs for port assignment messages

**Build failures**
- Install language-specific dependencies
- Node.js: `npm install`
- Python: `pip install -r requirements.txt`
- .NET: `dotnet restore`
- Java: `mvn clean package` or `gradle build`

---

## Maintenance

### When to Update

Update test projects when:
- New Azure Functions runtime version released
- New programming model available
- New trigger types supported
- Bug found in detection or runtime

### Versioning

Test projects should use:
- **Latest stable** versions of language runtimes
- **Latest stable** Azure Functions extensions
- **Current** programming models (v2 for Python/Node.js, Isolated for .NET)
- Keep **legacy** projects for backward compatibility testing

---

## Summary

**Purpose**: Comprehensive coverage of all Azure Functions variants

**Coverage**: 16+ projects × (detection + runtime + triggers) = exhaustive testing

**Benefit**: Confidence that `azd app run` supports all Azure Functions project types correctly

**Maintenance**: Keep projects up-to-date with latest Functions runtime and programming models
