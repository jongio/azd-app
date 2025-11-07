# Python Virtual Environment Issue

## Problem

When `azd app run` executes Python services, it does not properly use the virtual environment created by `azd app deps`. This causes issues especially with frameworks that spawn subprocesses (like uvicorn's `--reload` mode).

## Current Behavior

### How Commands Are Executed

`azd app run` currently executes Python services as follows:

- **FastAPI/Uvicorn**: `uvicorn app:app --reload --host 0.0.0.0 --port <port>`
- **Flask**: `python -m flask run --host 0.0.0.0 --port <port>`
- **Django**: `python manage.py runserver 0.0.0.0:<port>`
- **Streamlit**: `streamlit run app.py --server.port <port>`
- **Generic Python**: `python app.py`

### The Issue

1. **No Virtual Environment Activation**: Commands run using system Python, not the `.venv` Python
2. **Missing Dependencies**: Packages installed in `.venv` by `azd app deps` are not accessible
3. **Subprocess Problems**: When uvicorn uses `--reload`, it spawns a subprocess using the same interpreter
   - If parent uvicorn is from system Python, subprocess also uses system Python
   - Reloaded process cannot find packages installed in `.venv`

### Code Location

The issue is in `cli/src/internal/service/detector.go` in the `buildRunCommand()` function (lines 348-400):

```go
case "FastAPI":
    runtime.Command = "uvicorn"
    // ...
    runtime.Args = []string{appFile + ":app", "--reload", "--host", "0.0.0.0", "--port", fmt.Sprintf("%d", runtime.Port)}

case "Flask":
    runtime.Command = "python"
    runtime.Args = []string{"-m", "flask", "run", "--host", "0.0.0.0", "--port", fmt.Sprintf("%d", runtime.Port)}
```

### How Services Are Started

In `cli/src/internal/service/executor.go`:

```go
func StartService(runtime *ServiceRuntime, env map[string]string, projectDir string) (*ServiceProcess, error) {
    // ...
    cmd := exec.Command(runtime.Command, args...)
    cmd.Dir = runtime.WorkingDir
    cmd.Env = os.Environ()
    // ...
    cmd.Start()
}
```

The command is executed directly without activating or using the virtual environment.

## Required Fixes

### Solution Overview

Update `buildRunCommand()` to:

1. Detect if a virtual environment exists (`.venv` or `venv`)
2. Use the venv's Python interpreter for all Python commands
3. Run framework CLIs as Python modules (e.g., `python -m uvicorn` instead of `uvicorn`)

### Specific Changes Needed

#### 1. FastAPI/Uvicorn (Most Critical)

**Current:**
```bash
uvicorn app:app --reload
```

**Should be:**
```bash
# Linux/macOS
.venv/bin/python -m uvicorn app:app --reload

# Windows
.venv\Scripts\python.exe -m uvicorn app:app --reload
```

#### 2. Flask

**Current:**
```bash
python -m flask run
```

**Should be:**
```bash
.venv/bin/python -m flask run
```

#### 3. Django

**Current:**
```bash
python manage.py runserver
```

**Should be:**
```bash
.venv/bin/python manage.py runserver
```

#### 4. Streamlit

**Current:**
```bash
streamlit run app.py
```

**Should be:**
```bash
.venv/bin/python -m streamlit run app.py
```

#### 5. Generic Python

**Current:**
```bash
python app.py
```

**Should be:**
```bash
.venv/bin/python app.py
```

## Implementation Plan

### Step 1: Add Virtual Environment Detection Helper

Add a function to detect and return the virtual environment Python path:

```go
// getPythonVenvPath returns the path to the Python interpreter in the virtual environment.
// Returns empty string if no venv is found.
func getPythonVenvPath(projectDir string) string {
    // Check for .venv first (most common)
    venvPaths := []string{
        filepath.Join(projectDir, ".venv", "Scripts", "python.exe"), // Windows
        filepath.Join(projectDir, ".venv", "bin", "python"),         // Linux/macOS
        filepath.Join(projectDir, "venv", "Scripts", "python.exe"),  // Windows (alternative)
        filepath.Join(projectDir, "venv", "bin", "python"),          // Linux/macOS (alternative)
    }
    
    for _, path := range venvPaths {
        if _, err := os.Stat(path); err == nil {
            return path
        }
    }
    
    return ""
}
```

### Step 2: Update buildRunCommand()

Modify the function to use the venv Python when available:

```go
func buildRunCommand(runtime *ServiceRuntime, projectDir string, entrypoint string, runtimeMode string) error {
    // Detect virtual environment Python
    venvPython := getPythonVenvPath(projectDir)
    pythonCmd := "python"
    if venvPython != "" {
        pythonCmd = venvPython
    }
    
    switch runtime.Framework {
    case "FastAPI":
        runtime.Command = pythonCmd
        // Use -m uvicorn to run as module
        appFile := entrypoint
        if appFile == "" {
            appFile = findPythonAppFile(projectDir)
        }
        if err := validatePythonEntrypoint(projectDir, appFile); err != nil {
            return err
        }
        runtime.Args = []string{"-m", "uvicorn", appFile + ":app", "--reload", "--host", "0.0.0.0", "--port", fmt.Sprintf("%d", runtime.Port)}
    
    case "Flask":
        runtime.Command = pythonCmd
        // Already uses -m flask, just update command
        runtime.Args = []string{"-m", "flask", "run", "--host", "0.0.0.0", "--port", fmt.Sprintf("%d", runtime.Port)}
        // ... (rest of Flask setup)
    
    case "Django":
        runtime.Command = pythonCmd
        runtime.Args = []string{"manage.py", "runserver", fmt.Sprintf("0.0.0.0:%d", runtime.Port)}
    
    case "Streamlit":
        runtime.Command = pythonCmd
        appFile := entrypoint
        if appFile == "" {
            appFile = findPythonAppFile(projectDir)
        }
        if err := validatePythonEntrypoint(projectDir, appFile); err != nil {
            return err
        }
        runtime.Args = []string{"-m", "streamlit", "run", appFile + ".py", "--server.port", fmt.Sprintf("%d", runtime.Port)}
    
    case "Python":
        runtime.Command = pythonCmd
        // ... (rest of Python setup)
    }
    
    return nil
}
```

## Benefits

After implementing these fixes:

- ✅ All Python processes use the correct virtual environment
- ✅ Subprocess spawning (like uvicorn's `--reload`) inherits the correct environment
- ✅ All installed packages from `requirements.txt` are available
- ✅ Works consistently across Windows, Linux, and macOS
- ✅ No manual activation required from users
- ✅ Aligns with how `azd app deps` creates the environment

## Testing Checklist

After implementing:

1. Create a FastAPI project with dependencies in `requirements.txt`
2. Run `azd app deps` to create venv and install dependencies
3. Run `azd app run` and verify:
   - Service starts without import errors
   - Hot reload works when editing files
   - All dependencies are accessible
4. Test on Windows, Linux, and macOS
5. Test with all supported frameworks (FastAPI, Flask, Django, Streamlit)
6. Verify behavior when no venv exists (should fall back to system Python)

## Related Files

- `cli/src/internal/service/detector.go` - Main fix location
- `cli/src/internal/service/executor.go` - Where commands are executed
- `cli/src/internal/installer/installer.go` - Where venv is created
- `cli/src/internal/runner/runner.go` - Has similar venv detection logic (lines 196-216)
