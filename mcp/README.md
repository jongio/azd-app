# azd app MCP Server

Model Context Protocol (MCP) server for the Azure Developer CLI App extension. This server exposes running application information and logs to AI assistants like GitHub Copilot.

## Overview

The MCP server provides AI assistants with the ability to:
- Get information about running services (status, health, URLs, ports)
- Retrieve logs from running services with filtering
- Access project metadata and configuration
- View Azure deployment information and environment variables

## Architecture

The MCP server is implemented as a TypeScript/Node.js application that integrates with the azd app CLI extension through:

1. **CLI Integration**: The `azd app mcp serve` command starts the MCP server
2. **Extension Framework**: The azd extension framework automatically invokes the MCP server when needed
3. **Tool Execution**: The MCP server executes `azd app` commands (info, logs) to retrieve data

## Available Tools

### get_services

Get comprehensive information about all running services in the current project.

**Parameters:**
- `projectDir` (optional): Project directory path

**Returns:**
- Service name, status, and health
- Local URLs and ports
- Azure deployment URLs and resource names
- Docker image information
- Environment variables
- Language and framework details

**Example:**
```json
{
  "project": "/path/to/project",
  "services": [
    {
      "name": "api",
      "language": "node",
      "framework": "express",
      "local": {
        "status": "running",
        "health": "healthy",
        "url": "http://localhost:3000",
        "port": 3000,
        "pid": 12345
      },
      "azure": {
        "url": "https://myapp-api.azurewebsites.net",
        "resourceName": "myapp-api"
      }
    }
  ]
}
```

### get_service_logs

Retrieve logs from running services with optional filtering.

**Parameters:**
- `serviceName` (optional): Filter logs by service name
- `tail` (optional): Number of recent log lines (default: 100)
- `level` (optional): Filter by log level (info, warn, error, debug, all)
- `since` (optional): Show logs since duration (e.g., "5m", "1h")

**Returns:**
Array of log entries with timestamps, service names, and messages.

**Example:**
```json
[
  {
    "timestamp": "2024-01-15T10:30:45.123Z",
    "service": "api",
    "level": "info",
    "message": "Server started on port 3000",
    "isStderr": false
  }
]
```

### get_project_info

Get project metadata and service definitions from azure.yaml.

**Parameters:**
- `projectDir` (optional): Project directory path

**Returns:**
Project name and list of defined services.

**Example:**
```json
{
  "project": "/path/to/project",
  "services": [
    {
      "name": "api",
      "language": "node",
      "framework": "express",
      "project": "./src/api"
    }
  ]
}
```

## Usage

### As Part of azd Extension

The MCP server is automatically started by azd when an AI assistant connects to the extension:

```bash
# The azd extension framework handles MCP server startup automatically
# No manual invocation needed when using with AI assistants
```

### Manual Testing

For testing purposes, you can start the MCP server directly:

```bash
# From the extension installation
azd app mcp serve

# Or directly with Node.js (for development)
cd mcp
node dist/index.js
```

## Development

### Prerequisites

- Node.js 18 or later
- npm or pnpm
- TypeScript

### Building

```bash
cd mcp
npm install
npm run build
```

### Project Structure

```
mcp/
├── src/
│   └── index.ts          # Main MCP server implementation
├── dist/                 # Compiled JavaScript output
├── package.json
├── tsconfig.json
└── README.md
```

### Adding New Tools

To add a new tool to the MCP server:

1. Define the tool in the `tools` array with name, description, and input schema
2. Add a handler case in the `CallToolRequestSchema` handler
3. Implement the tool logic (typically calling `azd app` commands)
4. Update this documentation

## Integration with azd Extension

The MCP server integrates with the azd app extension through the `extension.yaml` configuration:

```yaml
capabilities:
  - mcp-server

mcp:
  serve:
    args: ["mcp", "serve"]
```

When azd detects an AI assistant connection:
1. azd invokes the extension with `azd app mcp serve`
2. The Go CLI command starts the Node.js MCP server
3. The MCP server communicates via stdio with the AI assistant
4. Tool calls are translated to `azd app` CLI commands
5. Results are returned in JSON format

## Troubleshooting

### MCP server not found

**Error:** `MCP server not found at <path>`

**Solution:** Ensure the extension is properly installed and the MCP server is bundled:
```bash
cd mcp
npm run build
```

### Command execution errors

**Error:** `Failed to execute azd app <command>`

**Solution:** 
- Ensure `azd app` CLI is in your PATH
- Check that the project directory contains a valid azure.yaml
- Verify services are running for log commands

### No services found

**Error:** Empty services array returned

**Solution:**
- Run `azd app run` to start services
- Check that azure.yaml exists and contains service definitions
- Verify you're in the correct project directory

## Security Considerations

- The MCP server executes `azd app` commands with the same privileges as the user
- All command output is sanitized and returned as JSON
- No arbitrary command execution is allowed
- Environment variables are filtered to show only relevant Azure/service data

## License

MIT
