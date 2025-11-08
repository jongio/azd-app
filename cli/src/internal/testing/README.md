# Testing Package

This package provides test execution and coverage aggregation for multi-language projects.

## Implementation Status

### Phase 1: Core Infrastructure (IN PROGRESS)

**Completed:**
- ✅ Type definitions (`types.go`)
  - `TestConfig` - Global test configuration
  - `ServiceTestConfig` - Per-service test configuration
  - `TestResult` - Test execution results
  - `CoverageData` - Coverage metrics
  - `AggregateResult` - Combined results
  - `AggregateCoverage` - Aggregated coverage

- ✅ Basic test command (`commands/test.go`)
  - All flags defined and working
  - Input validation (test type, threshold, output format)
  - Integration with command orchestrator
  - Dry-run mode

- ✅ Unit tests
  - Type definitions tested
  - Command structure tested
  - Validation logic tested

**Next Steps:**
- [ ] Test orchestrator implementation
- [ ] Framework auto-detection
- [ ] Service test execution

### Phases 2-5: Not Yet Started

See [implementation plan](../../docs/design/implementation-plan.md) for details.

## Current Functionality

The test command is currently in Phase 1 implementation. You can:

```bash
# View help and all available flags
azd app test --help

# Dry-run to see what would be tested
azd app test --dry-run --type unit --coverage --threshold 80

# Validate parameters
azd app test --type unit        # ✓ Valid
azd app test --type invalid     # ✗ Error: invalid test type
azd app test --threshold 150    # ✗ Error: threshold must be 0-100
```

## Usage

Once fully implemented (Phases 2-5), the command will:

```bash
# Run all tests
azd app test

# Run with coverage
azd app test --coverage --threshold 80

# Run specific test type
azd app test --type unit

# Watch mode
azd app test --watch --type unit

# CI/CD integration
azd app test --coverage --output-format junit
```

## Architecture

```
TestOrchestrator (TODO: Phase 1)
     ↓
  ┌──┴──┬──────┬────────┐
  │     │      │        │
Node  Python  .NET   Coverage (TODO: Phases 2-3)
Runner Runner Runner  Aggregator
```

## Contributing

When adding functionality:
1. Update type definitions as needed
2. Add unit tests for all new functions
3. Update this README with implementation status
4. Follow existing code patterns

See [implementation plan](../../docs/design/implementation-plan.md) for the complete roadmap.
