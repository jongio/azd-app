# Azure Functions Python v1 (Legacy) Programming Model

This is a test project for Azure Functions using the legacy Python v1 programming model with `function.json` configuration files.

## Features

- **Legacy function.json model** - Uses directory-based functions with configuration files
- **HTTP Trigger** - GET/POST endpoint at `/api/HttpTrigger`
- **Timer Trigger** - Runs every 5 minutes

## Project Structure

```
functions-python-v1/
├── host.json                    # Functions runtime configuration (v3 bundle)
├── requirements.txt             # Python dependencies
├── HttpTrigger/
│   ├── function.json            # HTTP trigger configuration
│   └── __init__.py              # Handler implementation
└── TimerTrigger/
    ├── function.json            # Timer trigger configuration
    └── __init__.py              # Handler implementation
```

## Trigger Types

### HTTP Trigger (`HttpTrigger/`)

- **Methods**: GET, POST
- **Auth Level**: Anonymous
- **Endpoint**: `http://localhost:7071/api/HttpTrigger`
- **Query Param**: `?name=YourName`

**Example**:
```bash
curl http://localhost:7071/api/HttpTrigger?name=Legacy
```

**Response**:
```json
{
  "message": "Hello, Legacy! (Python v1 legacy)",
  "timestamp": "2024-01-01T12:00:00.123456",
  "method": "GET"
}
```

### Timer Trigger (`TimerTrigger/`)

- **Schedule**: Every 5 minutes (`0 */5 * * * *`)
- **Logs**: Execution time and late status

## Running Locally

### With Azure Functions Core Tools

```bash
# Create virtual environment
python -m venv .venv

# Activate (Windows)
.venv\Scripts\activate

# Activate (Linux/macOS)
source .venv/bin/activate

# Install dependencies
pip install -r requirements.txt

# Start Functions runtime
func start
```

### With azd

From workspace root:
```bash
azd app run
```

## Testing with azd

This project is designed to test backward compatibility with the legacy v1 model in `azd app run`:

1. **Detection**: Should detect as Python v1 Functions (legacy)
2. **function.json**: Should recognize function.json-based functions
3. **Health Check**: Should use HTTP trigger-based health check
4. **Port**: Should default to 7071
5. **Dashboard**: Should appear in the azd dashboard

## Legacy Model Characteristics

- Each function is in its own directory
- `function.json` defines bindings and configuration
- `__init__.py` contains the handler (or other script file)
- Extension bundle v3.x
- No decorator-based configuration

## Migration Notes

For new projects, consider migrating to the v2 programming model:
- No `function.json` files needed
- Decorator-based configuration in single `function_app.py`
- Better Python idioms
- Simplified project structure

See `functions-python-v2` for the modern approach.

## Dependencies

- `azure-functions` - Azure Functions Python library

## Notes

- This is a **legacy** programming model (v1)
- Still supported but not recommended for new projects
- Use v2 model for better developer experience
- Requires Python 3.7 or later
