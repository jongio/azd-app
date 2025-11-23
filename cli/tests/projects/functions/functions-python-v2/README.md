# Azure Functions Python v2 Programming Model

This is a test project for Azure Functions using the Python v2 programming model with decorators.

## Features

- **Python v2 decorator model** - Uses `@app.route()` and `@app.timer_trigger()` decorators
- **HTTP Trigger** - Async GET/POST endpoint at `/api/httpTrigger`
- **Timer Trigger** - Runs every 5 minutes

## Project Structure

```
functions-python-v2/
├── host.json              # Functions runtime configuration
├── requirements.txt       # Python dependencies
└── function_app.py        # All functions defined with decorators
```

## Trigger Types

### HTTP Trigger

- **Methods**: GET, POST
- **Auth Level**: Anonymous
- **Endpoint**: `http://localhost:7071/api/httpTrigger`
- **Query Param**: `?name=YourName`
- **Async**: Uses async/await pattern

**Example**:
```bash
curl http://localhost:7071/api/httpTrigger?name=Python
```

**Response**:
```json
{
  "message": "Hello, Python! (Python v2)",
  "timestamp": "2024-01-01T12:00:00.123456",
  "method": "GET"
}
```

### Timer Trigger

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

This project is designed to test the Azure Functions Python v2 detector in `azd app run`:

1. **Detection**: Should detect as Python v2 Functions
2. **Decorator Pattern**: Should recognize `function_app.py` with decorators
3. **Health Check**: Should use HTTP trigger-based health check
4. **Port**: Should default to 7071
5. **Dashboard**: Should appear in the azd dashboard

## Python v2 Model Characteristics

- Single `function_app.py` file with all functions
- Decorator-based configuration (`@app.route`, `@app.timer_trigger`, etc.)
- No `function.json` files needed
- Simplified project structure
- Better Python idioms

## Dependencies

- `azure-functions` >= 1.18.0 - Azure Functions Python library (v2 model)

## Notes

- This is the **recommended** programming model for Python
- Requires Python 3.8 or later
- Extension bundle v4.x
- For legacy v1 model, see `functions-python-v1`
