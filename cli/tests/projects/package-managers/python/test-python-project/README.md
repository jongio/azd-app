# pip Package Manager Test Project

## Purpose

This test project validates that `azd app` correctly detects and uses **pip** for Python dependency management, the standard package manager for Python projects.

## What Is Being Tested

### Python Package Manager Detection Priority
When a Python project is detected, azd checks for package managers in this order:
1. **poetry** (if pyproject.toml with [tool.poetry])
2. **uv** (if pyproject.toml with [tool.uv])
3. **pip** (if requirements.txt exists) ← This project tests this
4. Default to pip (if none of above)

### Validation Points
- ✅ requirements.txt is correctly parsed
- ✅ `azd app deps` uses `pip install -r requirements.txt`
- ✅ `azd app run` starts the Python application
- ✅ Dependencies are installed in virtual environment or Python environment
- ✅ Standard pip workflow is properly supported
- ✅ Version constraints in requirements.txt are respected

## Project Structure

```
test-python-project/
├── requirements.txt       # pip dependencies
├── README.md             # This file
└── server.py            # Simple Flask application
```

## Key Configuration

In `requirements.txt`:
```
flask==3.0.0
```

Simple pip requirements file with Flask dependency. This is the most common Python setup.

## Running Tests

### Manual Test
```bash
cd cli/tests/projects/python/test-python-project

# Check detection
azd app reqs          # Should show Python + pip detection

# Install dependencies
azd app deps          # Should use pip install -r requirements.txt

# Run the service
azd app run           # Should start the Flask server
```

### Expected Behavior
1. Detection identifies Python project with pip
2. `azd app deps` executes: `pip install -r requirements.txt`
3. Flask and dependencies are installed
4. Service starts on port 5000 (default Flask port)
5. Logs show: "Running on http://localhost:5000"

### Automated Tests
This project is tested via:
- `cli/src/internal/service/detection_test.go` - Detection logic
- `cli/src/internal/executor/pip_executor_test.go` - pip execution
- Integration tests in CI/CD pipeline

## Why This Test Exists

### Problem It Solves
Without this test, we wouldn't validate:
- Correct detection of pip projects (requirements.txt)
- pip as the fallback package manager
- Proper dependency installation with requirements.txt
- Standard Python workflow support

### Real-World Scenario
pip is the most commonly used Python package manager. Most Python projects have a requirements.txt file. This test ensures the most common Python setup works correctly.

## Test Matrix

| Aspect | Expected | Status |
|--------|----------|--------|
| Detection | pip (requirements.txt) | ✅ |
| Dependencies File | requirements.txt | ✅ |
| Command | pip install -r requirements.txt | ✅ |
| Framework | Flask 3.0.0 | ✅ |
| Port | 5000 | ✅ |
| Virtual Env | Supported | ✅ |

## Troubleshooting

**"pip not found"**
- Install Python 3.9+ which includes pip
- Verify: `python -m pip --version`

**"No module named flask"**
- Run `azd app deps` to install dependencies
- Verify requirements.txt is in the correct location
- Check Python version: `python --version`

**"Requirements installation fails"**
- Some packages may require compilation tools
- On Windows: Install Visual C++ build tools
- On macOS: Install Xcode Command Line Tools
- On Linux: Install build-essential

## Related Test Projects

- [test-poetry-project](../test-poetry-project/) - Poetry variant
- [test-uv-project](../test-uv-project/) - uv variant (modern alternative)
