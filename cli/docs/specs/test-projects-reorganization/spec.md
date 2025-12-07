# Test Projects Reorganization Spec

## Overview

Reorganize test projects for the `azd app test` command to provide comprehensive coverage of popular testing frameworks for each supported language.

## Research: Top Testing Frameworks by Language

### Node.js/JavaScript/TypeScript
Based on npm downloads, State of JS survey, and industry adoption:
1. **Jest** (~95M weekly downloads) - Most popular, by Meta, great for React
2. **Vitest** (~20M weekly downloads) - Vite-native, fast, modern ESM-first
3. **Mocha** (~8M weekly downloads) - Flexible, mature, BDD/TDD style
4. **Jasmine** (~3M weekly downloads) - Standalone, no dependencies, Angular default

### Python
Based on PyPI downloads and Python Developers Survey:
1. **pytest** (~90M monthly downloads) - De facto standard, rich plugins
2. **unittest** (built-in) - Standard library, no install required
3. **nose2** (~500K monthly downloads) - unittest extension, test discovery
4. **doctest** (built-in) - Tests in docstrings, documentation-driven

### .NET (C#/F#)
Based on NuGet downloads and .NET ecosystem surveys:
1. **xUnit** (~150M downloads) - Modern, extensible, .NET Foundation
2. **NUnit** (~200M downloads) - Mature, ported from JUnit, widely adopted
3. **MSTest** (~100M downloads) - Microsoft official, VS integration
4. **bUnit** (~2M downloads) - Blazor-specific, component testing

### Go
Go has a built-in testing framework, but third-party options exist:
1. **testing** (built-in) - Standard library, idiomatic Go
2. **testify** (~15K GitHub stars) - Assertions, mocks, suites
3. **ginkgo** (~8K GitHub stars) - BDD-style, spec-based
4. **gocheck** (~2K GitHub stars) - Rich testing framework

## Directory Structure

### Current (Before)
```
cli/tests/projects/
├── node-jest-test/        # Standalone project
├── node-mocha-test/       # Standalone project  
├── python-unittest-test/  # Standalone project
├── dotnet-nunit-test/     # Standalone project
├── dotnet-mstest-test/    # Standalone project
├── polyglot-test/         # Multi-language, one framework each
├── node/                  # Package manager tests
├── python/                # Package manager tests
└── ...other test projects
```

### Proposed (After)
```
cli/tests/projects/
├── test-command/                 # All azd app test related projects
│   ├── README.md                 # Documentation for test command projects
│   ├── node-frameworks/          # Node.js frameworks (single azure.yaml, 4 services)
│   │   ├── azure.yaml
│   │   ├── jest/                 # Jest tests
│   │   ├── vitest/               # Vitest tests  
│   │   ├── mocha/                # Mocha tests
│   │   └── jasmine/              # Jasmine tests
│   ├── python-frameworks/        # Python frameworks (single azure.yaml, 4 services)
│   │   ├── azure.yaml
│   │   ├── pytest/               # pytest tests
│   │   ├── unittest/             # unittest tests
│   │   ├── nose2/                # nose2 tests
│   │   └── doctest/              # doctest tests
│   ├── dotnet-frameworks/        # .NET frameworks (single azure.yaml, 3 services)
│   │   ├── azure.yaml
│   │   ├── dotnet-frameworks.sln
│   │   ├── xunit/                # xUnit tests
│   │   ├── nunit/                # NUnit tests
│   │   └── mstest/               # MSTest tests
│   └── go-frameworks/            # Go frameworks (single azure.yaml, 3 services)
│       ├── azure.yaml
│       ├── go.mod
│       ├── testing/              # Standard testing package
│       ├── testify/              # testify assertions
│       └── ginkgo/               # BDD-style tests
├── node/                         # Package manager tests (unchanged)
├── python/                       # Package manager tests (unchanged)
└── ...other test projects        # Other command tests (unchanged)
```

## Benefits

1. **Organization**: All test-command related projects in one place
2. **Efficiency**: Single azure.yaml per language tests all frameworks
3. **Comprehensive**: Top 3-4 frameworks per language
4. **Maintainable**: Clear separation from other test projects
5. **Realistic**: Each service mimics real-world project structure

## Implementation Plan

1. Create `test-command/` directory structure
2. Create multi-framework projects for each language
3. Test each framework with `azd app test`
4. Remove old standalone framework test projects
5. Keep polyglot-test as integration reference
6. Update README documentation

## Framework Details

### Node.js Frameworks (node-frameworks/)
- **Jest**: Standard assertions, describe/it blocks, coverage built-in
- **Vitest**: Vite-compatible, ES modules, fast HMR
- **Mocha**: Flexible, works with any assertion library (chai)
- **Jasmine**: Standalone, BDD syntax, no dependencies

### Python Frameworks (python-frameworks/)
- **pytest**: Fixtures, parametrize, plugins, simple asserts
- **unittest**: TestCase classes, setUp/tearDown, standard library
- **nose2**: Plugin-based, test discovery, unittest compatible
- **doctest**: Tests in docstrings, REPL-style

### .NET Frameworks (dotnet-frameworks/)
- **xUnit**: Fact/Theory attributes, constructor injection
- **NUnit**: Test/TestCase attributes, SetUp/TearDown
- **MSTest**: TestClass/TestMethod, DataRow, VS integration

### Go Frameworks (go-frameworks/)
- **testing**: Table-driven tests, t.Run subtests
- **testify**: Assert/require packages, mock support
- **ginkgo**: Describe/It specs, BeforeEach hooks
