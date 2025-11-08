# Using azd app MCP Server with AI Assistants

This guide shows how to use the azd app MCP server with AI assistants like Claude Desktop and GitHub Copilot.

## What is MCP?

The Model Context Protocol (MCP) is a standard for connecting AI assistants to external tools and data sources. The azd app MCP server exposes your running application's information and logs to AI assistants, enabling them to:

- Check service status and health
- View application logs
- Access Azure deployment information
- Query project configuration

## Setup with Claude Desktop

### 1. Install the azd app extension

```bash
# Enable azd extensions
azd config set alpha.extension.enabled on

# Add the extension registry
azd extension source add -n app -t url -l "https://raw.githubusercontent.com/jongio/azd-app/refs/heads/main/registry.json"

# Install the extension
azd extension install jongio.azd.app
```

### 2. Configure Claude Desktop

Add the MCP server to your Claude Desktop configuration file:

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

### 3. Restart Claude Desktop

Close and reopen Claude Desktop to load the MCP server.

### 4. Verify the Connection

In Claude Desktop, you should see an MCP indicator showing that the azd-app server is connected. You can now ask Claude about your running services:

Example prompts:
- "What services are currently running in my azd app project?"
- "Show me the logs from my API service"
- "Check the health status of all my services"
- "What environment variables are configured for my web service?"

## Usage Examples

### Check Service Status

**Prompt:** "What services are running?"

**Claude Response:**
```
I can see 2 services running in your project:

1. **api** (Node.js/Express)
   - Status: Running (healthy)
   - Local URL: http://localhost:3000
   - Azure URL: https://myapp-api.azurewebsites.net
   - PID: 12345

2. **web** (React)
   - Status: Running (healthy)
   - Local URL: http://localhost:5173
   - Port: 5173
   - PID: 12346
```

### Get Service Logs

**Prompt:** "Show me the last 20 lines of logs from the API service"

**Claude Response:**
```
Here are the recent logs from the 'api' service:

[10:30:45] [api] Server started on port 3000
[10:30:46] [api] Connected to database
[10:31:02] [api] GET /api/users - 200 OK (15ms)
[10:31:15] [api] POST /api/users - 201 Created (8ms)
[10:32:01] [api] GET /api/products - 200 OK (12ms)
...
```

### Troubleshoot Issues

**Prompt:** "Why is my web service failing?"

**Claude Response:**
```
Let me check the status and logs of your web service.

I can see the web service is marked as "unhealthy". Looking at the error logs:

[10:35:12] [web] ERROR: Failed to connect to API at http://localhost:3000
[10:35:12] [web] Error: ECONNREFUSED

The issue is that the web service cannot connect to the API service. 
The API might not be running or the port might be blocked. 
Let me check the API service status...
```

### Check Azure Deployment

**Prompt:** "What are the Azure URLs for my deployed services?"

**Claude Response:**
```
Here are the Azure deployment URLs:

- **api**: https://myapp-api.azurewebsites.net
  Resource: myapp-api (Container App)
  
- **web**: https://myapp-web.azurewebsites.net
  Resource: myapp-web (Static Web App)

Both services are deployed and accessible.
```

## Setup with GitHub Copilot

GitHub Copilot's MCP integration is coming soon. Check back for updates.

## Available Tools

The MCP server provides six tools that enable AI assistants to both monitor and operate on your azd app projects. These complement (not duplicate) azd's MCP tools which focus on project creation and infrastructure generation.

### Our Tools

Our tools help you monitor, debug, and operate running applications:

#### Observability Tools (Read-Only)

### 1. get_services

Get comprehensive information about all running services.

**Parameters:**
- `projectDir` (optional): Project directory path

**Returns:**
- Service names, status, and health
- Local and Azure URLs
- Environment variables
- Process information (PID, start time)

### 2. get_service_logs

Retrieve logs from running services.

**Parameters:**
- `serviceName` (optional): Filter by service name
- `tail` (optional): Number of recent lines (default: 100)
- `level` (optional): Filter by level (info, warn, error, debug, all)
- `since` (optional): Show logs since duration (e.g., "5m", "1h")

**Returns:**
- Array of log entries with timestamps and messages

### 3. get_project_info

Get project metadata from azure.yaml.

**Parameters:**
- `projectDir` (optional): Project directory path

**Returns:**
- Project name and directory
- List of configured services

#### Operational Tools (Actions)

### 4. run_services

Start development services defined in azure.yaml, Aspire, or docker compose.

**Parameters:**
- `projectDir` (optional): Project directory path
- `runtime` (optional): Runtime mode ('azd', 'aspire', 'pnpm', 'docker-compose')

**Returns:**
- Status and PID of started services
- Message to use get_services for status updates

**Example:**
```
AI: "Start the application services"
→ Calls run_services tool
→ Returns: { status: "started", pid: 12345, message: "..." }
```

### 5. install_dependencies

Install dependencies for all detected projects (Node.js, Python, .NET).

**Parameters:**
- `projectDir` (optional): Project directory path

**Returns:**
- Installation status and output
- List of installed dependencies

**Example:**
```
AI: "Install all project dependencies"
→ Calls install_dependencies tool
→ Returns: { status: "completed", output: "npm install complete..." }
```

### 6. check_requirements

Check if all required prerequisites are installed and meet version requirements.

**Parameters:**
- `projectDir` (optional): Project directory path

**Returns:**
- Detailed status of each requirement
- Version information
- Missing or outdated tools

**Example:**
```
AI: "Check if I have all the required tools installed"
→ Calls check_requirements tool
→ Returns: { requirements: [...], allMet: true }
```

### azd's MCP Tools (Project Creation & Infrastructure)

For reference, azd's built-in MCP tools focus on different capabilities:
- **architecture_planning**: Plan Azure service selection and architecture design
- **discovery_analysis**: Analyze codebases to discover application components
- **azure_yaml_generation**: Generate azure.yaml configuration files
- **infrastructure_generation**: Generate Bicep/Terraform infrastructure code
- **docker_generation**: Create Dockerfiles for containerization
- **project_validation**: Validate azd project structure
- **error_troubleshooting**: Help diagnose azd command errors
- And more...

**Key Difference**: azd's MCP tools help you **create and configure** new projects, while our tools help you **monitor and debug** running applications. They work together seamlessly!

## Troubleshooting

### MCP Server Not Found

**Error:** `MCP server not found at <path>`

**Solution:** Ensure the extension is properly installed:
```bash
azd extension uninstall jongio.azd.app
azd extension install jongio.azd.app
```

### No Services Running

**Error:** Empty services list returned

**Solution:** Start your services first:
```bash
cd your-project
azd app run
```

### Connection Timeout

**Error:** MCP server times out or disconnects

**Solution:**
1. Restart Claude Desktop
2. Check Claude Desktop logs for errors
3. Try running `azd app mcp serve` manually to test
4. Ensure the azd app extension is properly installed

### Permission Errors

**Error:** Permission denied when accessing MCP server

**Solution:** Ensure the azd app CLI has execute permissions:
```bash
# Find the installation location
azd extension list --installed

# Add execute permission if needed
chmod +x <path-to-extension>/app
```

## Advanced Configuration

### Custom Project Directory

To use the MCP server with a specific project directory, you can set an environment variable in the Claude Desktop config:

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

You can configure separate MCP servers for different projects:

```json
{
  "mcpServers": {
    "azd-app-project1": {
      "command": "azd",
      "args": ["app", "mcp", "serve"],
      "env": {
        "PROJECT_DIR": "/path/to/project1"
      }
    },
    "azd-app-project2": {
      "command": "azd",
      "args": ["app", "mcp", "serve"],
      "env": {
        "PROJECT_DIR": "/path/to/project2"
      }
    }
  }
}
```

## Privacy and Security

The MCP server:
- ✅ Only accesses local project data
- ✅ Runs with the same permissions as your user
- ✅ Does not send data to external servers
- ✅ Only exposes information through the MCP protocol to authorized AI assistants

**Note:** The AI assistant (Claude, Copilot, etc.) may send the retrieved information to their servers for processing. Review your AI assistant's privacy policy for details.

## Learn More

- [MCP Documentation](https://modelcontextprotocol.io)
- [azd app CLI](../cli/README.md)
- [Azure Developer CLI](https://learn.microsoft.com/azure/developer/azure-developer-cli/)
- [MCP Server Implementation](./README.md)
