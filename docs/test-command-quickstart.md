# `azd app test` Quick Start Guide

## 5-Minute Setup

### Prerequisites
- Azure Developer CLI (azd) installed
- App extension installed: `azd extension install app`

### Quick Test

Navigate to your project and run:
```bash
azd app test
```

That's it! If your project follows standard conventions, tests will run automatically.

---

## Does It Work With My Project?

### ✅ Works out-of-the-box if you have:

**Node.js:**
- `package.json` with a `test` script
- Example: `"test": "jest"` or `"test": "vitest"`

**Python:**
- `pytest.ini`, `pyproject.toml` with pytest config, OR
- A `tests/` directory with `test_*.py` files

**.NET:**
- `*.csproj` files with test package references (xUnit, NUnit, MSTest)

**Go:**
- Files ending in `_test.go`

---

## Common Commands

```bash
# Run all tests
azd app test

# Run only unit tests
azd app test --type unit

# Run E2E tests
azd app test --type e2e

# Run with coverage
azd app test --coverage

# Run with coverage and HTML report
azd app test --coverage --coverage-format html

# Test specific project
azd app test --project ./api

# Fail fast (stop on first failure)
azd app test --fail-fast

# CI/CD: coverage threshold
azd app test --coverage --coverage-threshold 80
```

---

## Setup for Different Ecosystems

### Node.js (npm/pnpm/yarn)

**1. Add test scripts to `package.json`:**
```json
{
  "scripts": {
    "test": "jest",
    "test:unit": "jest --testPathPattern=unit",
    "test:e2e": "jest --testPathPattern=e2e"
  }
}
```

**2. Organize tests:**
```
__tests__/
├── unit/
│   └── *.test.js
└── e2e/
    └── *.test.js
```

**3. Run:**
```bash
azd app test --type unit
azd app test --type e2e
```

### Python (pytest)

**1. Structure your tests:**
```
tests/
├── unit/
│   └── test_*.py
└── e2e/
    └── test_*.py
```

**OR use markers in `pyproject.toml`:**
```toml
[tool.pytest.ini_options]
markers = [
    "unit: Unit tests",
    "e2e: End-to-end tests"
]
```

**2. Mark your tests:**
```python
import pytest

@pytest.mark.unit
def test_something():
    assert True
```

**3. Run:**
```bash
azd app test --type unit
azd app test --coverage
```

### .NET (xUnit/NUnit/MSTest)

**1. Add categories to tests:**
```csharp
[Fact]
[Trait("Category", "Unit")]
public void MyUnitTest()
{
    Assert.True(true);
}

[Fact]
[Trait("Category", "E2E")]
public void MyE2ETest()
{
    Assert.True(true);
}
```

**2. Run:**
```bash
azd app test --type unit
azd app test --coverage
```

### Go

**1. Use short flag for unit tests:**
```go
func TestUnit(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping in short mode")
    }
    // test code
}
```

**2. Name E2E tests clearly:**
```go
func TestE2E_UserFlow(t *testing.T) {
    // e2e test code
}
```

**3. Run:**
```bash
azd app test --type unit    # go test -short
azd app test --type e2e     # go test -run E2E
```

---

## Coverage Reports

### Basic Coverage
```bash
azd app test --coverage
```

Output shows terminal summary:
```
📊 Coverage Report
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ frontend     85.3% (234/274 lines)
✅ api          92.1% (156/169 lines)
✅ Overall      88.2% (390/443 lines)
```

### HTML Report
```bash
azd app test --coverage --coverage-format html
```

Opens interactive HTML report at `.test-results/coverage-report.html`

### All Formats
```bash
azd app test --coverage --coverage-format all
```

Generates:
- Terminal summary
- HTML report (`.test-results/coverage-report.html`)
- JSON export (`.test-results/coverage-report.json`)
- Cobertura XML (`.test-results/coverage-merged.xml`)

---

## Custom Configuration

If your test scripts have different names, create `azd-test.yaml`:

```yaml
version: 1

tests:
  unit:
    command: "npm run my-unit-tests"
  e2e:
    command: "npm run my-e2e-tests"
  all:
    command: "npm run all-tests"
```

---

## Troubleshooting

### "No test projects found"

**Problem:** azd can't detect your tests

**Solutions:**
1. Check you have test indicators:
   - Node.js: `package.json` with test script
   - Python: `tests/` directory or pytest config
   - .NET: `*.Tests.csproj` files
   - Go: `*_test.go` files

2. Try being explicit:
   ```bash
   azd app test --project ./your-project
   ```

### Tests don't run

**Problem:** Command runs but no tests execute

**Solution:** Check script names match conventions:
- Node.js: `test`, `test:unit`, `test:e2e`
- Or create `azd-test.yaml` (see above)

### Coverage not collected

**Problem:** Tests run but no coverage

**Solution:** Install coverage tools:
```bash
# Node.js
npm install --save-dev jest

# Python
pip install pytest-cov

# .NET
dotnet add package coverlet.collector
```

---

## CI/CD Integration

### GitHub Actions

```yaml
- name: Run tests with coverage
  run: azd app test --coverage --coverage-threshold 80 --fail-fast

- name: Upload coverage
  uses: codecov/codecov-action@v3
  with:
    files: .test-results/coverage-merged.xml
```

### Azure Pipelines

```yaml
- script: azd app test --coverage --coverage-format all
  displayName: 'Run tests with coverage'

- task: PublishCodeCoverageResults@1
  inputs:
    codeCoverageTool: 'Cobertura'
    summaryFileLocation: '.test-results/coverage-merged.xml'
```

---

## Monorepo Example

For a workspace with multiple projects:

```
my-workspace/
├── frontend/      # Node.js
├── backend/       # Python
└── services/      # .NET
```

**Run all:**
```bash
cd my-workspace
azd app test --coverage
```

**Output:**
```
📋 Found 3 test project(s):
   • frontend (node)
   • backend (python)
   • services (dotnet)

✓ frontend: 24 tests passed
✓ backend: 15 tests passed
✓ services: 8 tests passed

Overall Coverage: 86.4%
```

---

## Best Practices

1. **Organize by type**: Separate unit/e2e/integration in directories
2. **Use conventions**: Standard names = zero configuration
3. **Enable coverage**: Add `--coverage` to see what's tested
4. **Set thresholds**: Use `--coverage-threshold 80` in CI/CD
5. **Fast unit tests**: Keep unit tests under 1 second each
6. **Watch mode**: Use `--watch` during development (single project only)

---

## Getting Help

- Full documentation: See `docs/test-command-spec.md`
- Report issues: GitHub Issues
- Examples: See `tests/projects/with-tests/`

---

## Next Steps

1. ✅ Run `azd app test` in your project
2. ✅ Add coverage: `azd app test --coverage`
3. ✅ Organize tests by type (unit/e2e)
4. ✅ Set up CI/CD with coverage thresholds
5. ✅ Generate HTML reports for team review

Happy testing! 🎉
