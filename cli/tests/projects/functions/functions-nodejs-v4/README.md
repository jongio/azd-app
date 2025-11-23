# Azure Functions Node.js v4 Programming Model

This is a test project for Azure Functions using the Node.js v4 programming model.

## Features

- **Modern decorator-based model** - Uses `@azure/functions` v4
- **HTTP Trigger** - GET/POST endpoint at `/api/httpTrigger`
- **Timer Trigger** - Runs every 5 minutes

## Project Structure

```
functions-nodejs-v4/
├── host.json              # Functions runtime configuration
├── package.json           # Node.js dependencies
└── src/
    └── functions/
        ├── httpTrigger.js    # HTTP GET/POST handler
        └── timerTrigger.js   # Timer trigger (every 5 min)
```

## Trigger Types

### HTTP Trigger (`httpTrigger.js`)

- **Methods**: GET, POST
- **Auth Level**: Anonymous
- **Endpoint**: `http://localhost:7071/api/httpTrigger`
- **Query Param**: `?name=YourName`

**Example**:
```bash
curl http://localhost:7071/api/httpTrigger?name=Azure
```

**Response**:
```json
{
  "message": "Hello, Azure!",
  "timestamp": "2024-01-01T12:00:00.000Z",
  "method": "GET"
}
```

### Timer Trigger (`timerTrigger.js`)

- **Schedule**: Every 5 minutes (`0 */5 * * * *`)
- **Logs**: Execution time and schedule status

## Running Locally

### With Azure Functions Core Tools

```bash
npm install
func start
```

### With azd

From workspace root:
```bash
azd app run
```

## Testing with azd

This project is designed to test the Azure Functions detector in `azd app run`:

1. **Detection**: Should detect as Node.js v4 Functions
2. **Health Check**: Should use HTTP trigger-based health check
3. **Port**: Should default to 7071
4. **Dashboard**: Should appear in the azd dashboard

## Dependencies

- `@azure/functions` v4.x - Azure Functions Node.js library
- `azure-functions-core-tools` v4.x - Local development tools

## Notes

- Uses v4 programming model (recommended)
- No `function.json` files needed (legacy v3 model)
- Functions defined using `app.http()` and `app.timer()` decorators
