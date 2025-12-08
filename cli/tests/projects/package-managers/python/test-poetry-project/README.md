# Poetry Package Manager Test Project

## Purpose

This test project validates that `azd app` correctly detects and uses **Poetry** for Python dependency management, the modern Python packaging and dependency management solution.

## What Is Being Tested

### Python Package Manager Detection Priority
When a Python project is detected, azd checks for package managers in this order:
1. **poetry** (if pyproject.toml with [tool.poetry]) ← This project tests this
2. uv (if pyproject.toml with [tool.uv])
3. pip (if requirements.txt exists)
4. Default to pip (if none of above)

### Validation Points
- ✅ pyproject.toml with [tool.poetry] is correctly detected
- ✅ `azd app deps` uses `poetry install`
- ✅ Poetry lockfile (poetry.lock) is respected
- ✅ `azd app run` starts the Python application
- ✅ Dependencies are installed via Poetry
- ✅ Virtual environment management works correctly

## Project Structure

```
test-poetry-project/
├── pyproject.toml         # Poetry configuration (PEP 518)
├── poetry.lock           # Poetry lockfile
├── README.md             # This file
└── server.py            # Simple Flask application
```

## Key Configuration

In `pyproject.toml`:
```toml
[tool.poetry]
name = "poetry-app"
version = "0.1.0"
description = "Test Poetry project"

[tool.poetry.dependencies]
python = "^3.9"
flask = "^3.0.0"

[build-system]
requires = ["poetry-core"]
build-backend = "poetry.core.masonry.api"
```

Poetry configuration with Flask as dependency. Poetry handles all dependency resolution.

## Running Tests

### Manual Test
```bash
cd cli/tests/projects/python/test-poetry-project

# Check detection
azd app reqs          # Should show Python + Poetry detection

# Install dependencies
azd app deps          # Should use poetry install

# Run the service
azd app run           # Should start the Flask server
```

### Expected Behavior
1. Detection identifies Python project with Poetry (pyproject.toml)
2. `azd app deps` executes: `poetry install`
3. Poetry creates/updates poetry.lock file
4. Dependencies installed in Poetry's virtual environment
5. Service starts on port 5000 (default Flask port)
6. Logs show: "Running on http://localhost:5000"

### Automated Tests
This project is tested via:
- `cli/src/internal/service/detection_test.go` - Detection logic
- `cli/src/internal/executor/poetry_executor_test.go` - Poetry execution
- Integration tests in CI/CD pipeline

## Why This Test Exists

### Problem It Solves
Without this test, we wouldn't validate:
- Correct detection of Poetry projects (pyproject.toml)
- Poetry installation and execution
- Lockfile handling and updates
- Modern Python workflow support (PEP 518/517)
- Virtual environment automatic management

### Real-World Scenario
Modern Python projects prefer Poetry for its superior dependency resolution, lockfile guarantees, and simplified dependency management. This test ensures Poetry-based projects work seamlessly.

## Test Matrix

| Aspect | Expected | Status |
|--------|----------|--------|
| Detection | Poetry (pyproject.toml) | ✅ |
| Config File | pyproject.toml | ✅ |
| Lock File | poetry.lock | ✅ |
| Command | poetry install | ✅ |
| Framework | Flask 3.0.0 | ✅ |
| Venv | Auto-created | ✅ |
| Port | 5000 | ✅ |

## Troubleshooting

**"poetry not found"**
- Install Poetry: `pip install poetry`
- Or use pipx: `pipx install poetry`
- Verify: `poetry --version`

**"poetry.lock conflicts"**
- poetry.lock is auto-managed by Poetry
- Don't edit manually
- Regenerate: `poetry update` or `poetry install`

**"Project version issue"**
- Ensure Python version matches pyproject.toml
- Check `python = "^3.9"` matches your Python
- Verify: `python --version`

**"Dependency installation fails"**
- Some packages require compilation
- Ensure build tools installed
- Try: `poetry install --no-dev` for minimal install
- Check Poetry version: `poetry --version` (should be 1.5+)

## Related Test Projects

- [test-python-project](../test-python-project/) - pip variant
- [test-uv-project](../test-uv-project/) - uv variant
