# Testing Framework Design Documents

This directory contains comprehensive design documentation for the `azd app test` command and testing framework.

## Documents

### [Testing Framework Design](testing-framework.md)

Complete architecture and design for the testing framework, including:
- High-level architecture
- Component details (orchestrator, runners, coverage, reporting)
- Data structures
- Auto-detection logic
- Coverage aggregation
- Error handling
- Performance considerations
- Security considerations
- Future enhancements

**Audience**: Developers implementing the testing framework

### [Implementation Plan](implementation-plan.md)

Detailed implementation plan with phases, timeline, and success criteria, including:
- Implementation phases and priorities
- Acceptance criteria for each component
- Integration with existing code
- Testing strategy
- Risk mitigation
- Timeline (6-10 weeks)

**Audience**: Project managers and developers

## Related Documentation

### Command Reference

- [Test Command Specification](../commands/test.md) - Complete command reference
- [CLI Reference](../cli-reference.md) - All commands reference

### Configuration

- [Test Configuration Schema](../schema/test-configuration.md) - YAML configuration reference

## Quick Links

### For Users

- **Getting Started**: See [Test Command Specification](../commands/test.md)
- **Configuration**: See [Test Configuration Schema](../schema/test-configuration.md)
- **Examples**: See [Test Command Specification](../commands/test.md#examples)

### For Contributors

- **Architecture**: See [Testing Framework Design](testing-framework.md)
- **Implementation**: See [Implementation Plan](implementation-plan.md)
- **Contributing**: See [CONTRIBUTING.md](../../../CONTRIBUTING.md)

## Overview

The testing framework provides comprehensive test execution and coverage reporting for multi-language projects:

### Key Features

1. **Multi-Language Support**: Node.js, Python, .NET
2. **Test Type Separation**: unit, integration, e2e
3. **Auto-Detection**: Automatic framework detection
4. **Aggregated Coverage**: Unified coverage across all services
5. **Parallel Execution**: Fast test execution
6. **CI/CD Integration**: Multiple output formats

### Commands

```bash
# Run all tests
azd app test

# Run with coverage
azd app test --coverage

# Run specific test type
azd app test --type unit

# Run for specific service
azd app test --service api

# Watch mode
azd app test --watch
```

### Supported Frameworks

| Language | Frameworks |
|----------|------------|
| Node.js | Jest, Vitest, Mocha, AVA, Tap |
| Python | pytest, unittest, nose2 |
| .NET | xUnit, NUnit, MSTest |

### Architecture

```
azd app test
     ↓
Test Orchestrator
     ↓
  ┌──┴──┬──────┬────────┐
  │     │      │        │
Node  Python  .NET   Coverage
Runner Runner Runner  Aggregator
  │     │      │        │
  └─────┴──────┴────────┘
           ↓
    Report Generator
```

## Implementation Status

| Phase | Status | Description |
|-------|--------|-------------|
| Phase 1 | 📝 Planned | Core infrastructure (types, orchestrator, command) |
| Phase 2 | 📝 Planned | Language-specific runners |
| Phase 3 | 📝 Planned | Coverage aggregation and reporting |
| Phase 4 | 📝 Planned | Advanced features (watch, setup/teardown) |
| Phase 5 | 📝 Planned | Testing and documentation |

Legend: 📝 Planned | 🚧 In Progress | ✅ Complete

## Design Principles

1. **Auto-Detection First**: Minimize configuration required
2. **Explicit Override**: Allow full control when needed
3. **Multi-Language**: Support all major languages
4. **CI/CD Ready**: Easy integration with pipelines
5. **Fast Feedback**: Optimize for developer workflow
6. **Consistent UX**: Follow existing command patterns

## Configuration Example

```yaml
# azure.yaml
name: fullstack-app

test:
  coverageThreshold: 80
  parallel: true

services:
  web:
    language: js
    project: ./src/web
    test:
      framework: jest
      unit:
        command: pnpm test:unit
      coverage:
        threshold: 85
  
  api:
    language: python
    project: ./src/api
    test:
      framework: pytest
      integration:
        setup:
          - docker-compose up -d postgres
        teardown:
          - docker-compose down
      coverage:
        threshold: 90
```

## Next Steps

1. Review design documents
2. Get approval from maintainer
3. Begin Phase 1 implementation
4. Iterate based on feedback

## Questions?

For questions or feedback:
- Open an issue: [GitHub Issues](https://github.com/jongio/azd-app/issues)
- Start a discussion: [GitHub Discussions](https://github.com/jongio/azd-app/discussions)
