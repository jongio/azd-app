# Azure Functions Node.js v3 (Legacy) Programming Model

This is a test project for Azure Functions using the legacy Node.js v3 programming model with `function.json` configuration files.

## Features

- **Legacy function.json model** - Uses directory-based functions with configuration files
- **HTTP Trigger** - GET/POST endpoint at `/api/HttpTrigger`

## Project Structure

```
functions-nodejs-v3/
├── host.json                    # Functions runtime configuration (v3 bundle)
├── package.json                 # Dependencies
└── HttpTrigger/
    ├── function.json            # Trigger configuration
    └── index.js                 # Handler implementation
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
  "message": "Hello, Legacy! (v3 legacy model)",
  "timestamp": "2024-01-01T12:00:00.000Z",
  "method": "GET"
}
```

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

This project is designed to test backward compatibility with the legacy v3 model in `azd app run`:

1. **Detection**: Should detect as Node.js Functions (v3 legacy)
2. **function.json**: Should recognize function.json-based functions
3. **Health Check**: Should use HTTP trigger-based health check
4. **Port**: Should default to 7071
5. **Dashboard**: Should appear in the azd dashboard

## Legacy Model Characteristics

- Each function is in its own directory
- `function.json` defines bindings and configuration
- `index.js` (or `index.ts`) contains the handler
- Uses `context` object for logging and output bindings
- Extension bundle v3.x

## Migration Notes

For new projects, consider migrating to the v4 programming model:
- No `function.json` files needed
- Decorator-based configuration
- Better TypeScript support
- Simplified project structure

See `functions-nodejs-v4` for the modern approach.

## Dependencies

- `azure-functions-core-tools` v4.x - Local development tools (backward compatible)

## Notes

- This is a **legacy** programming model (v3)
- Still supported but not recommended for new projects
- Use v4 model for better developer experience
