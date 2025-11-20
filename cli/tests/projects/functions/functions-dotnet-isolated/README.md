# Azure Functions .NET Isolated Worker

This is a test project for Azure Functions using the .NET Isolated Worker model.

## Features

- **.NET 8 Isolated Worker** - Runs in a separate process from the Functions host
- **HTTP Trigger** - GET/POST endpoint at `/api/HttpTrigger`
- **Timer Trigger** - Runs every 5 minutes
- **Dependency Injection** - Full DI support with `ILogger` injection

## Project Structure

```
functions-dotnet-isolated/
├── host.json                      # Functions runtime configuration
├── local.settings.json            # Local development settings
├── FunctionApp.csproj             # Project file with NuGet packages
├── Program.cs                     # Host builder configuration
├── HttpTriggerFunction.cs         # HTTP trigger implementation
└── TimerTriggerFunction.cs        # Timer trigger implementation
```

## Trigger Types

### HTTP Trigger (`HttpTriggerFunction.cs`)

- **Methods**: GET, POST
- **Auth Level**: Anonymous
- **Endpoint**: `http://localhost:7071/api/HttpTrigger`
- **Query Param**: `?name=YourName`

**Example**:
```bash
curl http://localhost:7071/api/HttpTrigger?name=.NET
```

**Response**:
```json
{
  "message": "Hello, .NET! (.NET Isolated)",
  "timestamp": "2024-01-01T12:00:00.0000000Z",
  "method": "GET"
}
```

### Timer Trigger (`TimerTriggerFunction.cs`)

- **Schedule**: Every 5 minutes (`0 */5 * * * *`)
- **Logs**: Execution time and next scheduled run

## Running Locally

### With Azure Functions Core Tools

```bash
dotnet restore
dotnet build
func start
```

### With azd

From workspace root:
```bash
azd app run
```

## Testing with azd

This project is designed to test the Azure Functions .NET Isolated detector in `azd app run`:

1. **Detection**: Should detect as .NET Isolated Worker Functions
2. **Worker Runtime**: Should recognize `dotnet-isolated` runtime
3. **Build**: Should compile .NET project before running
4. **Health Check**: Should use HTTP trigger-based health check
5. **Port**: Should default to 7071
6. **Dashboard**: Should appear in the azd dashboard

## .NET Isolated Worker Characteristics

- Runs in a separate process from the Functions host
- Full .NET runtime features available
- Dependency injection using `Microsoft.Extensions.DependencyInjection`
- Uses `Microsoft.Azure.Functions.Worker` packages
- Better isolation and testability
- Supports .NET 6, 7, 8+

## Dependencies

- `Microsoft.Azure.Functions.Worker` - Core worker functionality
- `Microsoft.Azure.Functions.Worker.Sdk` - Build tools
- `Microsoft.Azure.Functions.Worker.Extensions.Http` - HTTP bindings
- `Microsoft.Azure.Functions.Worker.Extensions.Timer` - Timer bindings

## Notes

- This is the **recommended** model for .NET Functions
- Requires .NET 6.0 or later
- For legacy in-process model, see `functions-dotnet-inprocess`
- Worker runtime: `dotnet-isolated`
