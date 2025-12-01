# azd-app Demo Project

This demo project showcases how to use azd-app with AI-powered debugging through the MCP (Model Context Protocol) integration with GitHub Copilot.

## Quick Start

```bash
# Install dependencies
azd-app deps

# Start the API service
azd-app run
```

## Demonstrating AI Debugging

This project includes an **intentional bug** for demonstrating AI debugging capabilities:

### The Bug

The API's `/items` endpoint doesn't validate the `price` field. When you POST an item without a price, the tax calculation fails:

```bash
# This works correctly
curl -X POST http://localhost:3000/items \
  -H "Content-Type: application/json" \
  -d '{"name": "Widget", "price": 10}'

# This triggers the bug (missing price)
curl -X POST http://localhost:3000/items \
  -H "Content-Type: application/json" \
  -d '{"name": "Broken Widget"}'
```

### Using Copilot to Debug

1. Run the service with `azd-app run`
2. Trigger the bug with the curl command above
3. Open GitHub Copilot and ask:

   > "Check the API logs for errors"

   Copilot uses the MCP tools to call `get_service_logs` and shows you the error.

4. Ask Copilot:

   > "Why is the total calculation failing?"

   Copilot analyzes the logs and code to identify the missing validation.

5. Ask Copilot:

   > "Fix the price validation bug"

   Copilot suggests adding input validation for the price field.

## MCP Configuration

The `.vscode/mcp.json` file configures the azd-app MCP server for Copilot:

```json
{
  "servers": {
    "azd-app": {
      "type": "stdio",
      "command": "azd-app",
      "args": ["mcp"]
    }
  }
}
```

## Available MCP Tools

When debugging with Copilot, these tools are available:

| Tool | Description |
|------|-------------|
| `get_services` | List all services and their status |
| `get_service_logs` | Fetch logs from a running service |
| `restart_service` | Restart a service after code changes |
| `get_project_info` | Get azure.yaml configuration |
| `get_environment_variables` | Check environment variables |

## Project Structure

```
demo/
├── azure.yaml          # Service configuration
├── .vscode/
│   └── mcp.json        # MCP server configuration
├── api/
│   ├── package.json    # Node.js dependencies
│   └── server.js       # API with intentional bug
└── README.md           # This file
```

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | /items | List all items |
| POST | /items | Create item (bug: no price validation) |
| GET | /items/:id | Get item by ID |
| DELETE | /items/:id | Delete item |

## Learn More

- [azd-app Documentation](https://jongio.github.io/azd-app/)
- [MCP Integration Guide](https://jongio.github.io/azd-app/mcp/)
- [AI Debugging Tips](https://jongio.github.io/azd-app/examples/)
