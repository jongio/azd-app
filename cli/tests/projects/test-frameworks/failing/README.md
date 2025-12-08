# Failing Test Projects

These projects contain **intentionally failing tests** to verify that:
1. Test runners correctly detect and report failures
2. Exit codes are non-zero when tests fail
3. Error messages are properly displayed

## Projects

| Language | Framework | Expected: 1 pass, 2 fail |
|----------|-----------|--------------------------|
| `node/` | Jest | ✓ |
| `python/` | pytest | ✓ |
| `go/` | testing | ✓ |
| `dotnet/` | xUnit | ✓ |

## Running

```bash
# Run all failing tests (should exit with error)
mage testProjectsFailing

# Run specific language
mage testProjectsFailingNode
mage testProjectsFailingPython
mage testProjectsFailingGo
mage testProjectsFailingDotnet
```

## Expected Behavior

Each project has 3 tests:
- 1 passing test
- 2 failing tests

The mage targets verify that:
1. The test runner exits with a non-zero exit code
2. At least one test failure is reported
