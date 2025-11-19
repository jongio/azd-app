# Azure Functions TypeScript v4 Programming Model

This is a test project for Azure Functions using TypeScript with the v4 programming model.

## Features

- **TypeScript with full type safety** - Uses `@azure/functions` v4 with TypeScript
- **HTTP Trigger** - Typed GET/POST endpoint at `/api/httpTrigger`
- **Timer Trigger** - Strongly-typed timer that runs every 5 minutes

## Project Structure

```
functions-typescript-v4/
├── host.json                 # Functions runtime configuration
├── package.json              # Dependencies
├── tsconfig.json             # TypeScript configuration
└── src/
    └── functions/
        ├── httpTrigger.ts    # Typed HTTP handler
        └── timerTrigger.ts   # Typed timer trigger
```

## Trigger Types

### HTTP Trigger (`httpTrigger.ts`)

- **Methods**: GET, POST
- **Auth Level**: Anonymous
- **Endpoint**: `http://localhost:7071/api/httpTrigger`
- **Query Param**: `?name=YourName`
- **Type Safety**: Full TypeScript types for request/response

**Example**:
```bash
curl http://localhost:7071/api/httpTrigger?name=TypeScript
```

**Response**:
```json
{
  "message": "Hello, TypeScript!",
  "timestamp": "2024-01-01T12:00:00.000Z",
  "method": "GET",
  "name": "TypeScript"
}
```

### Timer Trigger (`timerTrigger.ts`)

- **Schedule**: Every 5 minutes (`0 */5 * * * *`)
- **Type Safety**: Strongly-typed Timer object

## Running Locally

### With Azure Functions Core Tools

```bash
npm install
npm run build
func start
```

### With azd

From workspace root:
```bash
azd app run
```

## Development

### Watch mode (auto-rebuild on changes)
```bash
npm run watch
```

### Build only
```bash
npm run build
```

## Testing with azd

This project is designed to test the Azure Functions TypeScript detector in `azd app run`:

1. **Detection**: Should detect as TypeScript Functions (Node.js variant)
2. **Build**: Should handle TypeScript compilation
3. **Health Check**: Should use HTTP trigger-based health check
4. **Port**: Should default to 7071
5. **Dashboard**: Should appear in the azd dashboard

## TypeScript Features

- **Strict mode enabled** - Maximum type safety
- **Interface definitions** - Typed request/response models
- **Type inference** - IntelliSense support in VS Code
- **Module resolution** - CommonJS with ES2021 target

## Dependencies

- `@azure/functions` v4.x - Azure Functions library with TypeScript types
- `typescript` v5.x - TypeScript compiler
- `@types/node` v20.x - Node.js type definitions
- `azure-functions-core-tools` v4.x - Local development tools

## Notes

- TypeScript files are compiled to `dist/` directory
- Uses v4 programming model (recommended)
- No `function.json` files needed
- Full IntelliSense and type checking support
