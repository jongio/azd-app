# Python Entry Point Configuration

## Overview

The `azd app` extension automatically detects Python entry point files when running services. You can also explicitly specify the entry point in your `azure.yaml` file for full control.

## Automatic Detection

When no `entrypoint` is specified, the extension searches for common Python entry point files in the following order:

### Entry Point Files (in order of preference)
1. `main.py`
2. `app.py`
3. `agent.py`
4. `__main__.py`
5. `run.py`
6. `server.py`

### Search Directories (in order)
1. Root directory (e.g., `main.py`)
2. `src/` (e.g., `src/main.py`)
3. `src/app/` (e.g., `src/app/main.py`)
4. `src/agent/` (e.g., `src/agent/agent.py`)
5. `app/` (e.g., `app/main.py`)
6. `agent/` (e.g., `agent/main.py`)

### Examples

The auto-detection will find entry points in various project structures:

```
# Simple project
myproject/
  ├── main.py          ✅ Found
  └── requirements.txt

# Src layout
myproject/
  ├── src/
  │   └── app.py       ✅ Found
  └── requirements.txt

# Deep structure
myproject/
  ├── src/
  │   └── agent/
  │       └── agent.py ✅ Found
  └── pyproject.toml
```

## Explicit Configuration

For projects with non-standard entry points or when you want explicit control, specify the `entrypoint` field in `azure.yaml`:

```yaml
services:
  api:
    language: python
    project: ./api
    entrypoint: src/custom/my_app.py  # Relative to project directory
    host: localhost
    config:
      port: 5000
```

### Benefits of Explicit Configuration

1. **Clarity**: Makes the entry point obvious to all developers
2. **Control**: Use any file name or location you prefer
3. **Speed**: Skips auto-detection, faster startup
4. **Consistency**: Same behavior across different project structures

## Package Manager Integration

The entry point works with all supported Python package managers:

### uv
```bash
uv run python your_entrypoint.py
```

### poetry
```bash
poetry run python your_entrypoint.py
```

### pip (with virtual environment)
```bash
.venv/Scripts/python.exe your_entrypoint.py  # Windows
.venv/bin/python your_entrypoint.py          # Linux/Mac
```

## Framework-Specific Behavior

For certain frameworks, the entry point is used differently:

### Flask
```yaml
services:
  api:
    language: python
    project: ./api
    entrypoint: app.py  # Sets FLASK_APP=app.py
```

### FastAPI
```yaml
services:
  api:
    language: python
    project: ./api
    entrypoint: main  # Used as: uvicorn main:app
```

### Django
Django uses `manage.py` automatically, so `entrypoint` is typically not needed.

### Generic Python
```yaml
services:
  script:
    language: python
    project: ./scripts
    entrypoint: src/process_data.py  # Runs directly: python src/process_data.py
```

## Error Messages

If no entry point is found, you'll see a helpful error message:

```
❌ No Python entry point found. Searched for: [main.py, app.py, agent.py, __main__.py, run.py, server.py] 
   in directories: [root, src/, src/app/, src/agent/, app/, agent/]

To fix this, you can:
  1. Create one of the expected entry point files (e.g., main.py, app.py, agent.py)
  2. OR specify a custom entry point in azure.yaml:
     services:
       yourservice:
         language: python
         project: ./path/to/service
         entrypoint: path/to/your/entrypoint.py
```

## Best Practices

1. **Use standard names** when possible (`main.py` or `app.py`) to avoid needing configuration
2. **Specify entrypoint** if your project has an unusual structure or multiple Python files
3. **Use relative paths** for entrypoint values (relative to the `project` directory)
4. **Include `.py` extension** in the entrypoint path for clarity

## Examples

### AI Agent Project
```yaml
services:
  agent:
    language: python
    project: ./foundry-agent
    entrypoint: src/agent/agent.py
    config:
      port: 8000
```

### Multi-Service API
```yaml
services:
  users-api:
    language: python
    project: ./services/users
    entrypoint: src/main.py
    config:
      port: 5000
  
  orders-api:
    language: python
    project: ./services/orders
    entrypoint: src/main.py
    config:
      port: 5001
```

### Script Runner
```yaml
services:
  processor:
    language: python
    project: ./batch
    entrypoint: scripts/process.py
```
