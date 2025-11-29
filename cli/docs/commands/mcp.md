# azd app mcp

## Overview

The `mcp` command provides Model Context Protocol (MCP) server functionality, enabling AI assistants like Claude Desktop and GitHub Copilot to interact with your azd app projects. The MCP server exposes tools and resources that allow AI assistants to monitor services, read logs, check requirements, and manage your development environment.

## Purpose

- **AI Integration**: Connect AI assistants to your development environment
- **Service Monitoring**: Expose service status, health, and logs to AI
- **Project Introspection**: Provide project configuration and metadata to AI
- **Operations**: Enable AI-assisted service management (start, stop, restart)
- **Configuration**: Allow AI to query and guide environment variable setup

## Command Usage

```bash
azd app mcp <subcommand> [flags]
```

### Subcommands

| Subcommand | Description |
|------------|-------------|
| `serve` | Start the MCP server for AI assistant integration |

## Subcommand: serve

Starts the Model Context Protocol server using stdio transport, allowing AI assistants to communicate with your azd app project.

```bash
azd app mcp serve
```

### How It Works

The MCP server:
1. Starts listening on stdio (standard input/output)
2. Registers tools, resources, and system instructions
3. Waits for MCP protocol messages from AI assistants
4. Executes tool calls and returns results
5. Provides read access to project resources

### Tools Provided

The MCP server exposes 10 tools organized into three categories:

#### Observability Tools (Read-Only)

| Tool | Description |
|------|-------------|
| `get_services` | Get comprehensive information about all running services including status, health, URLs, ports, and environment variables |
| `get_service_logs` | Retrieve logs from running services with filtering by service name, log level, and time range |
| `get_project_info` | Get project metadata and configuration from azure.yaml |

#### Operational Tools

| Tool | Description |
|------|-------------|
| `run_services` | Start development services defined in azure.yaml, Aspire, or docker compose |
| `stop_services` | Get guidance on stopping running development services |
| `restart_service` | Get guidance on restarting a specific service |
| `install_dependencies` | Install dependencies for all detected projects (Node.js, Python, .NET) |
| `check_requirements` | Check if all required prerequisites are installed and meet version requirements |

#### Configuration Tools

| Tool | Description |
|------|-------------|
| `get_environment_variables` | Get environment variables configured for services |
| `set_environment_variable` | Get guidance on setting environment variables |

### Resources Provided

The MCP server exposes 2 resources:

| Resource URI | Name | Description |
|--------------|------|-------------|
| `azure://project/azure.yaml` | azure.yaml | The project's azure.yaml configuration file |
| `azure://project/services/configs` | service-configs | Consolidated service configurations including environment variables |

### System Instructions

The MCP server includes built-in guidance for AI assistants:

```
Best Practices:
1. Always use get_services to check current state before starting/stopping services
2. Use check_requirements before installing dependencies to see what's needed
3. Use get_service_logs to diagnose issues when services fail to start
4. Read azure.yaml resource to understand project structure before operations
```

## Integration with VS Code / GitHub Copilot

VS Code supports MCP servers through dedicated configuration files. You can configure the azd app MCP server at workspace level or user level.

### Workspace Configuration (Recommended for Projects)

Create a `.vscode/mcp.json` file in your project root. This allows team members to share the same MCP configuration:

```json
{
  "servers": {
    "azd-app": {
      "type": "stdio",
      "command": "azd",
      "args": ["app", "mcp", "serve"]
    }
  }
}
```

### User Configuration (Global)

For personal configuration across all workspaces, use the **MCP: Open User Configuration** command in VS Code, or add to your user-level `mcp.json`:

**Location:**
- **Windows:** `%APPDATA%\Code\User\mcp.json`
- **macOS:** `~/Library/Application Support/Code/User/mcp.json`
- **Linux:** `~/.config/Code/User/mcp.json`

```json
{
  "servers": {
    "azd-app": {
      "type": "stdio",
      "command": "azd",
      "args": ["app", "mcp", "serve"]
    }
  }
}
```

### With Custom Project Directory

Use the `env` property to specify a project directory:

```json
{
  "servers": {
    "azd-app": {
      "type": "stdio",
      "command": "azd",
      "args": ["app", "mcp", "serve"],
      "env": {
        "PROJECT_DIR": "${workspaceFolder}"
      }
    }
  }
}
```

> **Note:** VS Code supports [predefined variables](https://code.visualstudio.com/docs/reference/variables-reference) like `${workspaceFolder}` in the configuration.

### With Environment File

Load environment variables from a `.env` file:

```json
{
  "servers": {
    "azd-app": {
      "type": "stdio",
      "command": "azd",
      "args": ["app", "mcp", "serve"],
      "envFile": "${workspaceFolder}/.env"
    }
  }
}
```

### Multiple Projects

Configure separate servers for different projects in your workspace:

```json
{
  "servers": {
    "azd-app-frontend": {
      "type": "stdio",
      "command": "azd",
      "args": ["app", "mcp", "serve"],
      "env": {
        "PROJECT_DIR": "${workspaceFolder}/frontend"
      }
    },
    "azd-app-backend": {
      "type": "stdio",
      "command": "azd",
      "args": ["app", "mcp", "serve"],
      "env": {
        "PROJECT_DIR": "${workspaceFolder}/backend"
      }
    }
  }
}
```

### Dev Container Configuration

For [Dev Containers](https://code.visualstudio.com/docs/devcontainers/containers), add the MCP server to your `devcontainer.json`:

```json
{
  "image": "mcr.microsoft.com/devcontainers/base:ubuntu",
  "customizations": {
    "vscode": {
      "mcp": {
        "servers": {
          "azd-app": {
            "command": "azd",
            "args": ["app", "mcp", "serve"]
          }
        }
      }
    }
  }
}
```

### VS Code MCP Configuration Properties

| Property | Required | Description | Example |
|----------|----------|-------------|---------|
| `type` | Yes | Transport type (always `stdio` for azd app) | `"stdio"` |
| `command` | Yes | Command to run | `"azd"` |
| `args` | No | Arguments array | `["app", "mcp", "serve"]` |
| `env` | No | Environment variables | `{"PROJECT_DIR": "/path"}` |
| `envFile` | No | Path to .env file | `"${workspaceFolder}/.env"` |

### Managing MCP Servers in VS Code

VS Code provides several commands to manage MCP servers:

| Command | Description |
|---------|-------------|
| `MCP: List Servers` | View all configured servers and their status |
| `MCP: Open User Configuration` | Edit user-level mcp.json |
| `MCP: Open Workspace Folder Configuration` | Edit workspace mcp.json |
| `MCP: Reset Cached Tools` | Clear cached tool definitions |
| `MCP: Reset Trust` | Reset trust settings for servers |

## Integration with Claude Desktop

### Configuration

Add the MCP server to your Claude Desktop configuration:

**macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`
**Linux:** `~/.config/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "azd-app": {
      "command": "azd",
      "args": ["app", "mcp", "serve"]
    }
  }
}
```

### With Custom Project Directory

```json
{
  "mcpServers": {
    "azd-app": {
      "command": "azd",
      "args": ["app", "mcp", "serve"],
      "env": {
        "PROJECT_DIR": "/path/to/your/project"
      }
    }
  }
}
```

### Multiple Projects

```json
{
  "mcpServers": {
    "azd-app-frontend": {
      "command": "azd",
      "args": ["app", "mcp", "serve"],
      "env": {
        "PROJECT_DIR": "/path/to/frontend"
      }
    },
    "azd-app-backend": {
      "command": "azd",
      "args": ["app", "mcp", "serve"],
      "env": {
        "PROJECT_DIR": "/path/to/backend"
      }
    }
  }
}
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PROJECT_DIR` | Project directory to use for operations | Current directory (`.`) |

## Tool Parameters

### get_services

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `projectDir` | string | No | Project directory path. Defaults to current directory. |

### get_service_logs

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `serviceName` | string | No | Filter logs to a specific service |
| `tail` | number | No | Number of recent log lines to retrieve (default: 100) |
| `level` | string | No | Filter by log level: `info`, `warn`, `error`, `debug`, or `all` (default: `all`) |
| `since` | string | No | Show logs since duration (e.g., `5m`, `1h`, `30s`) |

### get_project_info

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `projectDir` | string | No | Project directory path. Defaults to current directory. |

### run_services

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `projectDir` | string | No | Project directory path. Defaults to current directory. |
| `runtime` | string | No | Runtime mode: `azd`, `aspire`, `pnpm`, or `docker-compose` (default: `azd`) |

### stop_services

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `projectDir` | string | No | Project directory path. Defaults to current directory. |

### restart_service

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `serviceName` | string | **Yes** | Name of the service to restart |
| `projectDir` | string | No | Project directory path. Defaults to current directory. |

### install_dependencies

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `projectDir` | string | No | Project directory path. Defaults to current directory. |

### check_requirements

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `projectDir` | string | No | Project directory path. Defaults to current directory. |

### get_environment_variables

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `serviceName` | string | No | Filter to a specific service |
| `projectDir` | string | No | Project directory path. Defaults to current directory. |

### set_environment_variable

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | **Yes** | Name of the environment variable |
| `value` | string | **Yes** | Value of the environment variable |
| `serviceName` | string | No | Service to apply the variable to |

## Technical Details

### Protocol

- **Transport**: stdio (standard input/output)
- **Protocol**: Model Context Protocol (MCP)
- **Server Name**: `azd-app-mcp-server`
- **Version**: `0.1.0`

### Capabilities

| Capability | Enabled | Details |
|------------|---------|---------|
| Tools | Yes | 10 tools for monitoring and operations |
| Resources | Yes | 2 resources (subscribe=false, listChanged=true) |
| Prompts | No | Not currently implemented |
| Instructions | Yes | Built-in best practices guidance |

### Timeouts

| Operation | Timeout |
|-----------|---------|
| Default command execution | 30 seconds |
| Dependency installation | 5 minutes |
| Background process start | No timeout (runs in background) |

### Error Handling

All tool handlers return structured error responses:
- Context cancellation is properly detected
- Timeout errors include duration information
- Command failures include stderr output for debugging
- Parameter validation errors are descriptive

## Comparison with azd's MCP Tools

The azd app MCP server complements (does not duplicate) azd's built-in MCP capabilities:

| azd app MCP (this server) | azd MCP (built-in) |
|---------------------------|-------------------|
| Monitor running services | Plan architecture |
| View application logs | Discover components |
| Start/stop services | Generate azure.yaml |
| Check requirements | Generate infrastructure |
| Manage environment vars | Create Dockerfiles |
| **Focus: Runtime Operations** | **Focus: Project Setup** |

## Examples

### Testing the Server Manually

```bash
# Start the server to test it
azd app mcp serve

# The server will wait for JSON-RPC messages on stdin
# Use Ctrl+C to stop
```

### Example AI Conversations

**User:** "What services are running in my project?"
**AI:** Uses `get_services` tool to list running services with their status.

**User:** "Show me the error logs from the API"
**AI:** Uses `get_service_logs` with `serviceName=api` and `level=error`.

**User:** "Install all my project dependencies"
**AI:** Uses `install_dependencies` tool and reports results.

**User:** "Check if I have all the required tools"
**AI:** Uses `check_requirements` and lists any missing prerequisites.

## Troubleshooting

### Server Not Starting

```bash
# Verify the extension is installed
azd extension list --installed

# Reinstall if needed
azd extension uninstall jongio.azd.app
azd extension install jongio.azd.app
```

### Connection Issues in Claude Desktop

1. Restart Claude Desktop after config changes
2. Check Claude Desktop logs for errors
3. Test the server manually: `azd app mcp serve`
4. Verify the path to `azd` is in your system PATH

### Connection Issues in VS Code

1. Check the MCP server status: Run **MCP: List Servers** from the Command Palette
2. View server logs: Select the server → **Show Output**
3. Restart the server: Select the server → **Restart**
4. Reset cached tools: Run **MCP: Reset Cached Tools** if tools aren't appearing
5. Verify `azd` is in your PATH by running `azd version` in the VS Code terminal

### VS Code MCP Configuration Errors

**Server not appearing:**
- Ensure `mcp.json` is in `.vscode/` folder (workspace) or the correct user config location
- Verify JSON syntax is valid (VS Code provides IntelliSense in `mcp.json` files)
- Check that `type` is set to `"stdio"` for local servers

**Tools not showing in chat:**
- Open the Chat view and click the **Tools** button to verify the server is listed
- Try running **MCP: Reset Cached Tools** to refresh tool discovery
- Restart VS Code if the server was recently added

**Trust dialog not appearing:**
- Run **MCP: Reset Trust** to reset trust settings
- Start the server from the MCP server list (not directly from `mcp.json`)

### Permission Errors

```bash
# Ensure azd app has execute permissions
chmod +x $(which azd)

# On Windows, run as administrator if needed
```

### No Services Found

Ensure services are running before querying:
```bash
azd app run
```

## See Also

- [MCP Usage Guide](../../docs/mcp-usage.md) - User-friendly setup guide
- [run Command](run.md) - Start development services
- [logs Command](logs.md) - View service logs
- [info Command](info.md) - Get project information
- [reqs Command](reqs.md) - Check requirements
- [deps Command](deps.md) - Install dependencies
- [MCP Documentation](https://modelcontextprotocol.io) - Official MCP specification
- [VS Code MCP Servers](https://code.visualstudio.com/docs/copilot/customization/mcp-servers) - VS Code MCP documentation
- [Claude Desktop MCP](https://modelcontextprotocol.io/quickstart/user) - Claude Desktop MCP setup
