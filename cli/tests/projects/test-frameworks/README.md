# Testing Frameworks Test Projects

This directory contains multi-framework test projects for validating `azd app test` command functionality across different languages and testing frameworks.

## 🎯 Purpose

These test projects validate that `azd app test`:
- ✅ Correctly detects test frameworks in each language
- ✅ Properly runs tests with appropriate commands
- ✅ Aggregates results from multiple services
- ✅ Handles different test output formats
- ✅ Supports coverage reporting
- ✅ Works with popular frameworks across languages

## Directory Structure

```
test-frameworks/
├── node/                # Node.js testing frameworks (3 projects)
│   ├── jest/            # Jest (most popular, ~95M downloads)
│   ├── vitest/          # Vitest (Vite-native, ~20M downloads)
│   └── alternatives/    # Mocha + Jasmine (consolidated)
├── python/              # Python testing frameworks (2 frameworks)
│   ├── pytest-svc/      # pytest (most popular, ~90M downloads)
│   └── unittest-svc/    # unittest (built-in, xUnit style)
├── dotnet/              # .NET testing frameworks (2 frameworks)
│   ├── xunit/           # xUnit (modern, ~150M downloads)
│   └── nunit/           # NUnit (original .NET, ~200M downloads)
├── go/                  # Go testing frameworks (2 frameworks)
│   ├── testing-svc/     # Go testing (built-in standard library)
│   └── testify-svc/     # testify (assertions and mocks)
└── README.md            # This file
```

## Projects by Language

### Node.js Frameworks (`node/`)

| Service | Framework | Downloads/month | Details | Status |
|---------|-----------|-----------------|---------|--------|
| jest-service | Jest | ~95M | Most popular, all-in-one testing | See [jest/README.md](node/jest/) |
| vitest-service | Vitest | ~20M | Vite-native, fast modern testing | [vitest/README.md](node/vitest/) |
| alternatives-service | Mocha + Jasmine | ~8M + 3M | Flexible alternatives consolidated | [alternatives/README.md](node/alternatives/) |

**Run all Node.js tests:**
```bash
cd node
azd app test --all
```

### Python Frameworks (`python/`)

| Service | Framework | Downloads/month | Details | Status |
|---------|-----------|-----------------|---------|--------|
| pytest-svc | pytest | ~90M | Most popular, powerful fixtures | See [pytest-svc/README.md](python/pytest-svc/) |
| unittest-svc | unittest | Built-in | Standard library, xUnit style | [unittest-svc/README.md](python/unittest-svc/) |

**Run all Python tests:**
```bash
cd python
azd app test --all
```

### .NET Frameworks (`dotnet/`)

| Service | Framework | Downloads/month | Details | Status |
|---------|-----------|-----------------|---------|--------|
| xunit-service | xUnit | ~150M | Modern, extensible | See [xunit/README.md](dotnet/xunit/) |
| nunit-service | NUnit | ~200M | Original .NET testing | [nunit/README.md](dotnet/nunit/) |

**Run all .NET tests:**
```bash
cd dotnet
dotnet test
```

### Go Frameworks (`go/`)

| Service | Framework | GitHub Stars | Details | Status |
|---------|-----------|--------------|---------|--------|
| testing-service | testing | Built-in | Standard library | [testing-svc/README.md](go/testing-svc/) |
| testify-service | testify | ~15K | Assertions and mocks | [testify-svc/README.md](go/testify-svc/) |

**Run all Go tests:**
```bash
cd go
go test ./...
```

## Test Coverage Summary

**Total Test Frameworks**: 7 frameworks across 4 languages

| Language | Frameworks | Popular Downloads |
|----------|-----------|-------------------|
| Node.js | 3 | ~95M+ combined |
| Python | 2 | ~90M+ combined |
| .NET | 2 | ~350M+ combined |
| Go | 2 | ~15K+ stars combined |

**Test Scenarios Covered**:
- ✅ Basic assertions and expect patterns
- ✅ Parametrized/data-driven tests  
- ✅ Setup/teardown and fixtures
- ✅ BDD-style tests (Jasmine)
- ✅ Mocking and stubbing (Chai, Testify)
- ✅ Coverage reporting
- ✅ Watch/live mode
- ✅ Multiple output formats (JSON, XML, HTML)
