# Azure Functions & Logic Apps Support

**Status**: Implemented  
**Version**: 1.0

---

## Overview

Native support for running Azure Functions and Logic Apps Standard locally with `azd app run`.

**Supported Languages**: Logic Apps, Node.js, TypeScript, Python, .NET, Java

---

## Detection

Projects are detected when `host: function` is specified in `azure.yaml`:

### Logic Apps Standard
- `workflows/` directory with `workflow.json` files, OR
- `host.json` with `Microsoft.Azure.Functions.ExtensionBundle.Workflows`

### Node.js/TypeScript
- `package.json` + `host.json`
- Function definitions: `*/function.json` (v3) or decorators (v4)

### Python
- `function_app.py` (v2) OR `requirements.txt` + `*/function.json` (v1)
- `host.json`

### .NET
- `*.csproj` with `Microsoft.Azure.Functions.Worker` (Isolated) OR `Microsoft.NET.Sdk.Functions` (In-Process)
- `host.json`

### Java
- `pom.xml` with `azure-functions-maven-plugin` OR `build.gradle` with Azure Functions plugin
- `host.json`

---

## Configuration

```yaml
name: my-functions-app

services:
  # Logic Apps
  workflows:
    project: ./logicapp
    host: function
    # Auto-detected as Logic Apps

  # Python Functions
  api:
    project: ./functions-python
    host: function
    # Auto-detected as Python

  # .NET Functions
  dotnet-api:
    project: ./MyFunctionApp
    host: function
    ports:
      - "7073"  # Optional - defaults to 7071
```

---

## Running Locally

```bash
# Check prerequisites
azd app reqs

# Start all services
azd app run
```

**Output**:
```
🚀 Starting services

workflows    → http://localhost:7071  [Logic Apps Standard]
api          → http://localhost:7072  [Azure Functions (Python)]
dotnet-api   → http://localhost:7073  [Azure Functions (.NET)]

📊 Dashboard: http://localhost:4280
```

---

## Prerequisites

### Required for All Variants

**Azure Functions Core Tools** (`func` CLI v4.0+):
- Windows: `winget install Microsoft.Azure.FunctionsCoreTools`
- macOS: `brew install azure-functions-core-tools@4`
- Linux: `npm install -g azure-functions-core-tools@4`

### Language-Specific

**Node.js/TypeScript**:
- Node.js 18.x or 20.x
- npm (bundled with Node.js)
- TypeScript 4.x+ (for TypeScript projects)

**Python**:
- Python 3.9, 3.10, or 3.11
- pip (bundled with Python)

**.NET**:
- .NET SDK 6.0 or 8.0
- dotnet CLI (bundled with SDK)

**Java**:
- Java JDK 11 or 17
- Maven 3.6+ (for Maven projects)
- Gradle 7.0+ (for Gradle projects)

---

## Architecture

### Variant Detection

```go
type FunctionsVariant string

const (
    FunctionsVariantLogicApps  FunctionsVariant = "logicapps"
    FunctionsVariantNodeJS     FunctionsVariant = "nodejs"
    FunctionsVariantPython     FunctionsVariant = "python"
    FunctionsVariantDotNet     FunctionsVariant = "dotnet"
    FunctionsVariantJava       FunctionsVariant = "java"
    FunctionsVariantUnknown    FunctionsVariant = "unknown"
)
```

### Detection Flow

```
DetectServiceRuntime()
    ↓
[host: function?]
    ↓ YES
detectAzureFunctionsRuntime()
    ↓
detectFunctionsVariant()
    ├─ Logic Apps? (workflows/ OR extension bundle)
    ├─ .NET? (*.csproj with Functions SDK)
    ├─ Node.js? (package.json + host.json)
    ├─ Python? (function_app.py OR requirements.txt)
    └─ Java? (pom.xml OR build.gradle)
    ↓
buildFunctionsRuntime()
    ├─ Detect language
    ├─ Assign port (default 7071)
    ├─ Configure health check
    └─ Build command: ["func", "start", "--port", "7071"]
```

### Key Components

**Files**:
- `service/functions.go` - Variant detection & runtime building
- `service/detector.go` - Service detection integration
- `detector/detector.go` - Multi-project discovery
- `runner/runner.go` - Unified execution

**Functions**:
- `detectFunctionsVariant(projectDir)` - Identify variant
- `buildFunctionsRuntime(...)` - Create runtime config
- `assignFunctionsPort(...)` - Port assignment
- `FindFunctionApps(rootDir)` - Discovery

---

## Health Checks

**Logic Apps**:
- Path: `/runtime/webhooks/workflow/api/management/workflows`

**All Other Variants**:
- Path: `/admin/host/status`

---

## Error Messages

### Missing host.json

```
Error: Azure Functions project missing host.json at ./functions
Expected: host.json file in project root
Run 'func init' to initialize a new Functions project
```

### No Functions Detected

```
Error: service 'api' has host: function but no valid project detected

Expected one of:
  - Logic Apps: workflows/ directory
  - Node.js: */function.json + package.json
  - Python: function_app.py OR requirements.txt
  - .NET: *.csproj with Functions SDK
  - Java: pom.xml with azure-functions plugin

Found: host.json but no function definitions
```

### Missing Prerequisites

```
✗ Azure Functions Core Tools: not found
  Install: winget install Microsoft.Azure.FunctionsCoreTools
  Verify: func --version

✓ Node.js: v20.10.0
```

---

## Testing

### Unit Tests (15+ tests)

- Variant detection (6 variants × 3 scenarios each)
- Language detection (TypeScript vs JavaScript, etc.)
- Port assignment (default, explicit, conflicts)
- Health check configuration
- Runtime building
- Error scenarios

### Test Projects (12+ projects)

**Core Variants**:
- `functions-nodejs-v4/` - Node.js v4 model
- `functions-typescript-v4/` - TypeScript v4
- `functions-nodejs-v3/` - Node.js v3 (legacy)
- `functions-python-v2/` - Python v2 decorators
- `functions-python-v1/` - Python v1 (legacy)
- `functions-dotnet-isolated/` - .NET Isolated Worker
- `logicapp-test/` - Logic Apps Standard
- `logicapp-ai-agent-style/` - AI Agent Pattern

**Edge Cases**:
- `functions-minimal/` - Minimal valid project
- `functions-invalid-no-host/` - Missing host.json
- `functions-invalid-no-functions/` - No functions
- `functions-invalid-corrupt-host/` - Corrupt host.json

### Test Coverage

- ✅ **100% pass rate** (zero failures)
- ✅ **>80% code coverage**
- ✅ **Zero lint errors**
- ✅ **Zero regressions**

---

## Implementation Status

### Phase 1: Foundation ✅ COMPLETE
- Created `functions.go` with unified detection
- Variant detection for all languages
- Runtime configuration
- Port assignment
- Health checks
- Integration with detector

### Phase 2: Discovery ✅ COMPLETE
- `FindFunctionApps()` for multi-project discovery
- Prerequisites detection via `azd app reqs`
- Java support (Maven/Gradle)
- Updated `azd app generate`

### Phase 3: Documentation ✅ COMPLETE
- Comprehensive `azure-functions.md` (~700 lines)
- Updated `run.md` with examples
- Updated `azure.yaml.md` with 6 config examples

---

## Adding New Language Support

Example: Adding PowerShell Functions

1. Add variant:
```go
const FunctionsVariantPowerShell FunctionsVariant = "powershell"
```

2. Add detection:
```go
func isPowerShellFunctionsVariant(projectDir string) bool {
    return fileExists(projectDir, "profile.ps1") && hasFunctionJson(projectDir)
}
```

3. Add to detection chain:
```go
if isPowerShellFunctionsVariant(projectDir) {
    return FunctionsVariantPowerShell
}
```

4. Add language detection:
```go
case FunctionsVariantPowerShell:
    return "PowerShell"
```

Everything else (port, health checks, running) works automatically!

---

## References

- [Azure Functions](https://learn.microsoft.com/azure/azure-functions/)
- [Logic Apps Standard](https://learn.microsoft.com/azure/logic-apps/single-tenant-overview-compare)
- [Functions Core Tools](https://learn.microsoft.com/azure/azure-functions/functions-run-local)
- [User Documentation](../../azure-functions.md)
