# Test Projects

This directory contains comprehensive test projects used to validate `azd app` commands across all supported languages, frameworks, and scenarios.

## Directory Structure

```
projects/
├── 📦 package-managers/     (7 projects) - Dependency detection & installation
│   ├── node/               (5 projects) - npm, pnpm, yarn, override, workspaces
│   └── python/             (2 projects) - pip, poetry
│
├── 🧪 test-frameworks/      (9 projects) - Test runner discovery & execution
│   ├── node/               (3 projects) - jest, vitest, alternatives
│   ├── python/             (2 projects) - pytest, unittest
│   ├── dotnet/             (2 projects) - xunit, nunit
│   └── go/                 (2 projects) - testing, testify
│
├── ⚡ functions/            (12 projects) - Azure Functions variants
│
├── 🔗 orchestration/        (3 projects) - Multi-service configuration
│   ├── azure-deploy-test/
│   ├── fullstack-test/
│   └── process-services-test/
│
└── 🔧 integration/          (9 projects) - Advanced features & edge cases
    ├── aspire-test/
    ├── azure/
    ├── boundary-test/
    ├── env-formats-test/
    ├── go-api/
    ├── health-test/
    ├── hooks-test/
    ├── lifecycle-test/
    └── polyglot-test/
```

## Complete Test Project Map

### 📦 Package Manager Detection (`package-managers/`)

**Purpose**: Validate dependency detection and installation across different package managers and versions.

#### Node.js (`package-managers/node/`)
- **test-npm-project/** - npm with explicit packageManager field and default fallback
- **test-pnpm-project/** - pnpm with explicit packageManager field
- **test-yarn-project/** - yarn with explicit packageManager field
- **test-package-manager-override/** - packageManager field overrides lock files
- **test-npm-workspace/** - npm workspaces with monorepo race condition fix

#### Python (`package-managers/python/`)
- **test-python-project/** - pip package manager (default)
- **test-poetry-project/** - Poetry dependency management

### 🧪 Testing Framework Support (`test-frameworks/`)

**Purpose**: Validate `azd app test` command detects and runs tests across all popular testing frameworks.

#### Node.js (`test-frameworks/node/`)
- **jest/** - Jest (most popular, ~95M downloads)
- **vitest/** - Vitest (Vite-native, ~20M downloads)
- **alternatives/** - Mocha + Jasmine (consolidated)

#### Python (`test-frameworks/python/`)
- **pytest-svc/** - pytest (most popular, ~90M downloads)
- **unittest-svc/** - unittest (built-in, xUnit style)

#### .NET (`test-frameworks/dotnet/`)
- **xunit/** - xUnit (modern, ~150M downloads)
- **nunit/** - NUnit (original .NET, ~200M downloads)

#### Go (`test-frameworks/go/`)
- **testing-svc/** - Go testing (built-in standard library)
- **testify-svc/** - testify (assertions and mocks, ~15K stars)

### 🚀 Service Orchestration (`orchestration/`)

**Purpose**: Validate multi-service scenarios, port management, environment variables, and deployment.

- **azure-deploy-test/** - Minimal Azure deployment with Container Apps
  - Tests: Azure environment variable inheritance, AZURE_*, AZD_*, SERVICE_* variables
  
- **fullstack-test/** - Multi-service orchestration
  - Tests: Explicit port configuration, cross-service HTTP communication, multi-language services

- **process-services-test/** - Service types and modes
  - Tests: HTTP services, TCP services, process services, watch mode, build mode, daemon mode

### 🔌 Azure Functions Support

**Purpose**: Validate Azure Functions detection and execution for all languages and models.

**Total Projects**: 11 core projects covering all major scenarios

#### Logic Apps (2 projects)
| Project | Model | Focus | Why Needed |
|---------|-------|-------|-----------|
| `logicapp-test/` | Logic Apps Standard | Basic workflows | Validate HTTP-triggered workflows with func CLI |
| `logicapp-ai-agent-style/` | Logic Apps + AI Foundry | Real-world complexity | Test complex infra, managed identities, AI integration |

#### Node.js/TypeScript (2 projects)
| Project | Version | Focus | Why Needed |
|---------|---------|-------|-----------|
| `functions-nodejs-v4/` | v4 (current + TS) | HTTP + Timer, TypeScript | Validate modern Node.js model with type support |
| `functions-nodejs-v3/` | v3 (legacy) | Legacy support | Ensure backward compatibility |

#### Python (2 projects)
| Project | Version | Focus | Why Needed |
|---------|---------|-------|-----------|
| `functions-python-v2/` | v2 (current) | Decorator model | Validate modern Python model |
| `functions-python-v1/` | v1 (legacy) | Function.json | Test backward compatibility |

#### .NET (3 projects)
| Project | Model | Focus | Why Needed |
|---------|-------|-------|-----------|
| `functions-dotnet-isolated/` | Isolated Worker | .NET 6.0+ | Validate isolated model (recommended) |
| `functions-dotnet-isolated-durable/` | Durable Functions | Orchestration | Test stateful workflows |
| `functions-dotnet-inprocess/` | In-Process (legacy) | Legacy .NET | Backward compatibility |

#### Java (1 project)
| Project | Build Tool | Focus | Why Needed |
|---------|-----------|-------|-----------|
| `functions-java-maven/` | Maven 3.6+ | Standard build | Most common Java project setup |

#### Multi-Language (1 project)
| Project | Languages | Focus | Why Needed |
|---------|-----------|-------|-----------|
| `functions-multi-app/` | Node.js + Python + .NET | Workspace | Test multiple runtimes in one workspace |

#### Error Scenarios (3 projects)
| Project | Error Type | Focus | Why Needed |
|---------|-----------|-------|-----------|
| `functions-invalid-no-host/` | Missing host.json | Error handling | Validate helpful error messages |
| `functions-invalid-no-functions/` | No functions defined | Missing config | Test edge case detection |
| `functions-minimal/` | Minimal but valid | Baseline | Simplest valid project |

#### Coverage Matrix

**Language Support**:
- ✅ Node.js (v3, v4) - Legacy and modern models
- ✅ TypeScript - v4 type-safe variant
- ✅ Python (v1, v2) - Legacy and modern models
- ✅ .NET (In-Process, Isolated, Durable) - All variants
- ✅ Java (Maven, Gradle) - Multiple build tools
- ✅ Logic Apps - Workflow-based serverless

**Trigger Type Coverage**:
- ✅ HTTP triggers (all languages)
- ✅ Timer/Schedule (Node.js, Python, .NET, Java)
- ✅ Blob storage (Python)
- ✅ Durable orchestration (.NET)

**Prerequisites Validation**:
- ✅ `azd app reqs` detects all required tools for each variant
- ✅ `azd app reqs` validates tool versions meet minimum requirements (func 4.0+)
- ✅ `azd app reqs` provides OS-specific installation instructions
- ✅ Detection: Correct variant and language identification
- ✅ Runtime: Successful execution with `azd app run`
- ✅ Health checks: Proper endpoint responses
- ✅ Errors: Helpful messages for common issues

### 🩺 Health & Monitoring (`integration/`)

**Purpose**: Validate health monitoring, state tracking, and service lifecycle management.

- **health-test/** - Comprehensive health check validation
  - Tests: HTTP health checks, TCP port checks, process checks, streaming mode, JSON output, filtering, authentication
  
- **lifecycle-test/** - Service state transitions
  - Tests: State change detection, recovery tracking, change history, real-time updates, log correlation

### 🔧 Advanced Features (`integration/`)

**Purpose**: Validate advanced configuration and edge cases.

- **boundary-test/** - Workspace boundary checking
  - Tests: Only detect services within azure.yaml workspace, don't traverse parent directories
  
- **hooks-test/** - Hook execution (basic and platform-specific)
  - Tests: Simple prerun/postrun hooks, shell execution, platform detection (Windows/POSIX), external scripts
  
- **env-formats-test/** - Environment variable handling
  - Tests: .env file parsing, variable interpolation, override behavior
  
- **aspire-test/** - .NET Aspire integration
  - Tests: Aspire manifest parsing, service discovery
  
- **go-api/** - Go language support
  - Tests: Go project detection, dependency management
  
- **polyglot-test/** - Mixed language monorepo
  - Tests: Multiple languages in one workspace, independent service runs
  
- **azure/** - Configuration file variants
  - Tests: azure.yaml parsing, backup/recovery, error handling

## Quick Reference by Use Case

### Testing Dependencies & Installation
```bash
cd package-managers/node/test-npm-project && azd app deps
cd package-managers/python/test-python-project && azd app deps
```

### Testing Test Runner Discovery
```bash
cd test-frameworks/node/jest && azd app test
cd test-frameworks/python/pytest-svc && azd app test
cd test-frameworks/dotnet && dotnet test
```

### Testing Azure Functions
```bash
cd functions/functions-nodejs-v4 && azd app run
cd functions/functions-python-v2 && azd app run
cd functions/functions-dotnet-isolated && azd app run
```

### Testing Multi-Service Orchestration
```bash
cd orchestration/fullstack-test && azd app run
cd orchestration/process-services-test && azd app run
cd integration/polyglot-test && azd app run
```

### Testing Health Monitoring
```bash
cd integration/health-test && azd app run        # Start services
cd integration/health-test && azd app health      # Check health
```

### Testing Advanced Features
```bash
cd integration/boundary-test/workspace && azd app run
cd integration/hooks-test && azd app run --dry-run
cd orchestration/azure-deploy-test && azd up
```

## Running Tests

From the root directory:
```bash
# Test deps command
azd app deps

# Test run command
azd app run

# Test health command
azd app health

# Test test command
azd app test

# Test on specific project
cd cli/tests/projects/orchestration/fullstack-test
azd app run
```

## Test Project Statistics

| Category | Count | Purpose |
|----------|-------|---------|
| Package Manager | 7 | Detect and install dependencies |
| Testing Frameworks | 9 | Discover and run tests |
| Azure Functions | 12 | Function variants and languages |
| Service Orchestration | 3 | Multi-service configuration |
| Health & Monitoring | 2 | Service health and lifecycle |
| Advanced Features | 7 | Edge cases and integrations |
| **Total** | **40** | Comprehensive coverage |

## Key Testing Principles

1. **Detection First**: Each project validates detection before runtime
2. **Realistic Scenarios**: Projects represent real-world use cases
3. **Edge Cases**: Boundary, error, and version compatibility tests included
4. **Multi-Language**: Node.js, Python, .NET, Go, Java coverage
5. **Platform Support**: Windows, Linux, macOS compatibility verified
6. **Performance**: Tests include realistic timing and scale scenarios
