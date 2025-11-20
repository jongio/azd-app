# Logic Apps Test Project

This is a test project for validating Logic Apps Standard support in `azd app`.

## Structure

```
logicapp-test/
├── azure.yaml              # azd app configuration
├── host.json               # Azure Functions host configuration with workflow extension bundle
├── local.settings.json     # Local development settings (dotnet-isolated runtime)
└── workflows/
    └── TestWorkflow/
        └── workflow.json   # Logic Apps workflow definition
```

## Workflow

The test project includes a simple HTTP-triggered stateful workflow that:
- Accepts HTTP POST requests
- Returns a JSON response with a greeting message

## Testing

```bash
cd cli/tests/projects/logicapp-test
azd app run
```

The workflow should start on port 7071 and be accessible at:
```
http://localhost:7071/api/TestWorkflow/triggers/manual/invoke
```

## Requirements

- Azure Functions Core Tools v4 (`func`)
- .NET runtime (for dotnet-isolated worker)

## Configuration

This project uses:
- **Worker Runtime**: dotnet-isolated
- **Extension Bundle**: Microsoft.Azure.Functions.ExtensionBundle.Workflows
- **Framework**: Logic Apps Standard
