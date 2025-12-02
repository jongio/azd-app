# azd app Demo Project

This demo project showcases how to use `azd app` with AI-powered debugging through the MCP (Model Context Protocol) integration with GitHub Copilot.

> 📚 **Full Documentation**: [jongio.github.io/azd-app](https://jongio.github.io/azd-app/)

## Prerequisites

You'll need the Azure Developer CLI (azd) installed. Choose your platform:

**Windows:**
```bash
winget install microsoft.azd
```

**macOS:**
```bash
brew tap azure/azd && brew install azd
```

**Linux:**
```bash
curl -fsSL https://aka.ms/install-azd.sh | bash
```

## Install azd app

Enable the azd extensions feature and install the azd app extension:

```bash
# Enable extensions
azd config set alpha.extensions.enabled on

# Add azd app extension source
azd extension source add app https://raw.githubusercontent.com/jongio/azd-app/main/registry.json

# Install the extension
azd extension install app
```

## Quick Start

```bash
# Install dependencies
azd app deps

# Start the API service
azd app run
```

## Demonstrating AI Debugging

This project includes an **intentional bug** for demonstrating AI debugging capabilities.

### The Bug

The API's `/items` endpoint doesn't validate the `price` field. When you POST an item without a price, the tax calculation fails:

```bash
# This works correctly
curl -X POST http://localhost:3000/items -H "Content-Type: application/json" -d '{"name": "Widget", "price": 10}'

# This triggers the bug (missing price)
curl -X POST http://localhost:3000/items -H "Content-Type: application/json" -d '{"name": "Broken Widget"}'
```

### Using Copilot to Debug

1. Run the service with `azd app run`
2. Trigger the bug with the curl command above
3. Open GitHub Copilot Chat (<kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>I</kbd> on Windows/Linux or <kbd>Cmd</kbd>+<kbd>Shift</kbd>+<kbd>I</kbd> on macOS) and ask:

   > "Check the API logs for errors"

   Copilot uses the MCP tools to call `get_service_logs` and shows you the error.

4. Ask Copilot:

   > "Why is the total calculation failing?"

   Copilot analyzes the logs and code to identify the missing validation.

5. Ask Copilot:

   > "Fix the price validation bug"

   Copilot suggests adding input validation for the price field.

## MCP Configuration

The `.vscode/mcp.json` file configures the azd app MCP server for Copilot:

```json
{
  "servers": {
    "Azure Developer CLI - App Extension": {
      "command": "azd",
      "args": ["app", "mcp", "serve"]
    }
  }
}
```

## Available MCP Tools

When debugging with Copilot, 10 tools are available across three categories:

### Observe
| Tool | Description |
|------|-------------|
| `get_services` | List all services with status, ports, and health |
| `get_service_logs` | Fetch logs with optional filtering by level and time |
| `get_project_info` | Get azure.yaml configuration |

### Operate
| Tool | Description |
|------|-------------|
| `run_services` | Start all or specific services |
| `stop_services` | Stop all or specific services |
| `restart_service` | Restart a specific service |
| `install_dependencies` | Install dependencies for services |

### Configure
| Tool | Description |
|------|-------------|
| `check_requirements` | Verify prerequisites (Node.js, Python, etc.) |
| `get_environment_variables` | Get environment variables (sensitive values redacted) |
| `set_environment_variable` | Set an environment variable for a service |

See the full [MCP Tools Reference](https://jongio.github.io/azd-app/mcp/tools/) for detailed documentation.

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

- [azd app Documentation](https://jongio.github.io/azd-app/)
- [Quick Start Guide](https://jongio.github.io/azd-app/quick-start/)
- [Guided Tour](https://jongio.github.io/azd-app/tour/)
- [MCP Server & AI Debugging](https://jongio.github.io/azd-app/mcp/)
- [MCP Tools Reference](https://jongio.github.io/azd-app/mcp/tools/)
- [CLI Reference](https://jongio.github.io/azd-app/reference/cli/)
