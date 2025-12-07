# pytest Python Test Framework Test Project

## Purpose

This test project validates that `azd app test` correctly **detects and runs pytest test framework** for Python applications, ensuring comprehensive test discovery and execution with the most popular Python testing tool.

## What Is Being Tested

### Test Framework Detection
When running `azd app test`:
1. pytest is correctly identified as the test runner
2. Test files are properly discovered (test_*.py, *_test.py patterns)
3. pytest configuration (pytest.ini or pyproject.toml) is parsed
4. `azd app test` executes pytest with proper reporting
5. Fixtures, markers, and parametrized tests work correctly
6. Test results are aggregated across services

### Validation Points
- ✅ pytest is the most popular Python testing framework (~90M downloads/month)
- ✅ pytest.ini or pyproject.toml configuration is recognized
- ✅ Test scripts in package.json or tox are properly detected
- ✅ Rich assertion introspection
- ✅ Fixtures for test setup/teardown
- ✅ Parametrization for test variants
- ✅ Coverage reports via pytest-cov
- ✅ Proper exit codes on test pass/fail

## Project Structure

```
pytest-svc/
├── pyproject.toml         # pytest configuration
├── requirements.txt       # Python dependencies (pytest, pytest-cov)
├── src/
│   ├── math.py          # Source code to test
│   └── string.py        # Source code to test
├── tests/
│   ├── test_math.py     # pytest test file
│   └── test_string.py   # pytest test file
└── README.md            # This file
```

## Key Configuration

In `pyproject.toml`:
```toml
[tool.pytest.ini_options]
testpaths = ["tests"]
python_files = "test_*.py"
python_classes = "Test*"
python_functions = "test_*"
addopts = "--strict-markers --tb=short"

[tool.coverage.run]
source = ["src"]
```

In `requirements.txt`:
```
pytest==7.4.0
pytest-cov==4.1.0
```

pytest is configured with test discovery patterns and coverage collection.

## Running Tests

### Manual Test
```bash
cd cli/tests/projects/test-command/python-frameworks/pytest-svc

# Install dependencies
pip install -r requirements.txt

# Run tests
pytest

# Expected output:
# tests/test_math.py::test_add PASSED                           [ 25%]
# tests/test_math.py::test_multiply PASSED                      [ 50%]
# tests/test_string.py::test_strip PASSED                       [ 75%]
# tests/test_string.py::test_upper PASSED                       [100%]
# 
# ======================== 4 passed in 0.23s ========================
```

### With azd app test
```bash
# From workspace root
azd app test --service pytest-svc

# Expected output:
# Testing pytest-svc service...
# 
# tests/test_math.py::test_add PASSED
# tests/test_math.py::test_multiply PASSED
# tests/test_string.py::test_strip PASSED
# tests/test_string.py::test_upper PASSED
# 
# Summary: 4 tests passed
```

### Coverage Report
```bash
pytest --cov=src

# Generates coverage report in terminal
# Add --cov-report=html for HTML report
```

### Automated Tests
This project is tested via:
- `cli/src/cmd/app/commands/test_command_integration_test.go` - Framework detection
- `cli/src/internal/executor/pytest_executor_test.go` - pytest execution
- CI/CD pipeline test coverage validation

## Why This Test Exists

### Problem It Solves
Without this test, we wouldn't validate:
- Correct detection of pytest as the test runner
- pytest configuration parsing
- Test file discovery patterns
- Fixture support for complex tests
- Coverage report generation
- Proper integration with azd test command
- The most popular Python testing framework (90M downloads)

### Real-World Scenario
pytest is the standard testing framework for modern Python projects. Nearly all new Python projects use pytest. This test ensures production-ready support for the most common Python testing setup.

## Test Matrix

| Aspect | Expected | Status |
|--------|----------|--------|
| Framework | pytest 7.4+ | ✅ |
| Test Files | tests/test_*.py | ✅ |
| Configuration | pyproject.toml | ✅ |
| Fixtures | Supported | ✅ |
| Parametrization | Supported | ✅ |
| Coverage | pytest-cov | ✅ |
| Markers | Custom markers | ✅ |
| Plugins | Extensible | ✅ |

## Troubleshooting

**"pytest not found"**
- Install pytest: `pip install pytest`
- Verify: `pytest --version`

**"Tests not found"**
- Ensure test files match pattern: `tests/test_*.py`
- Verify testpaths in pyproject.toml points to correct directory
- Check file locations relative to pyproject.toml

**"ImportError: cannot import module"**
- Ensure src/ is in PYTHONPATH
- Or install package in dev mode: `pip install -e .`
- Check sys.path in pytest configuration

**"Coverage not generated"**
- Install pytest-cov: `pip install pytest-cov`
- Run: `pytest --cov=src`
- Check coverage configuration in pyproject.toml

**"Fixtures not working"**
- Ensure conftest.py is in the test directory
- Check fixture scope (function, class, module, session)
- Verify fixture is used as function parameter

## Related Test Projects

- [unittest-svc/](../unittest-svc/) - unittest (standard library alternative)
- [nose2-svc/](../nose2-svc/) - nose2 (unittest extension)
- [doctest-svc/](../doctest-svc/) - doctest (inline tests)
