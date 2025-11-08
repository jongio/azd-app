#!/usr/bin/env node

/**
 * Model Context Protocol server for Azure Developer CLI App extension.
 * Exposes running application information and logs to AI assistants.
 */

import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
  Tool,
} from "@modelcontextprotocol/sdk/types.js";
import { exec } from "child_process";
import { promisify } from "util";

const execAsync = promisify(exec);

// Server configuration
const SERVER_NAME = "azd-app-mcp-server";
const SERVER_VERSION = "0.1.0";

// Create MCP server instance
const server = new Server(
  {
    name: SERVER_NAME,
    version: SERVER_VERSION,
  },
  {
    capabilities: {
      tools: {},
    },
  }
);

/**
 * Execute azd app CLI command and return JSON output
 */
async function executeAzdAppCommand(
  command: string,
  args: string[] = []
): Promise<any> {
  const allArgs = [command, ...args, "--output", "json"];
  const cmd = `azd app ${allArgs.join(" ")}`;

  try {
    const { stdout, stderr } = await execAsync(cmd, {
      maxBuffer: 10 * 1024 * 1024, // 10MB buffer for large logs
    });

    if (stderr && !stdout) {
      throw new Error(stderr);
    }

    // Parse JSON output
    if (stdout.trim()) {
      return JSON.parse(stdout);
    }

    return {};
  } catch (error: any) {
    // Try to parse error output as JSON if possible
    if (error.stdout) {
      try {
        return JSON.parse(error.stdout);
      } catch {
        // If not JSON, throw the original error
      }
    }

    throw new Error(
      `Failed to execute azd app ${command}: ${error.message || error}`
    );
  }
}

/**
 * Define available tools
 */
const tools: Tool[] = [
  {
    name: "get_services",
    description:
      "Get information about all running services in the current azd app project. " +
      "Returns comprehensive details including service status, health, URLs, ports, " +
      "Azure deployment information, and environment variables.",
    inputSchema: {
      type: "object",
      properties: {
        projectDir: {
          type: "string",
          description:
            "Optional project directory path. If not provided, uses current directory.",
        },
      },
    },
  },
  {
    name: "get_service_logs",
    description:
      "Get logs from running services. Can filter by service name, log level, " +
      "and time range. Supports both recent logs and live streaming.",
    inputSchema: {
      type: "object",
      properties: {
        serviceName: {
          type: "string",
          description:
            "Optional service name to filter logs. If not provided, shows logs from all services.",
        },
        tail: {
          type: "number",
          description: "Number of recent log lines to retrieve. Default is 100.",
        },
        level: {
          type: "string",
          description:
            "Filter by log level: 'info', 'warn', 'error', 'debug', or 'all'. Default is 'all'.",
          enum: ["info", "warn", "error", "debug", "all"],
        },
        since: {
          type: "string",
          description:
            "Show logs since duration (e.g., '5m', '1h', '30s'). If provided, overrides tail parameter.",
        },
      },
    },
  },
  {
    name: "get_project_info",
    description:
      "Get project metadata and configuration from azure.yaml. " +
      "Returns project name, directory, and service definitions.",
    inputSchema: {
      type: "object",
      properties: {
        projectDir: {
          type: "string",
          description:
            "Optional project directory path. If not provided, uses current directory.",
        },
      },
    },
  },
];

/**
 * Handler for listing available tools
 */
server.setRequestHandler(ListToolsRequestSchema, async () => {
  return { tools };
});

/**
 * Handler for tool execution
 */
server.setRequestHandler(CallToolRequestSchema, async (request) => {
  const { name, arguments: args } = request.params;

  try {
    switch (name) {
      case "get_services": {
        const projectDir = (args as any)?.projectDir;
        const cmdArgs = projectDir ? ["--project", projectDir] : [];
        const result = await executeAzdAppCommand("info", cmdArgs);

        return {
          content: [
            {
              type: "text",
              text: JSON.stringify(result, null, 2),
            },
          ],
        };
      }

      case "get_service_logs": {
        const serviceName = (args as any)?.serviceName;
        const tail = (args as any)?.tail;
        const level = (args as any)?.level;
        const since = (args as any)?.since;

        const cmdArgs: string[] = [];

        if (serviceName) {
          cmdArgs.push(serviceName);
        }

        if (tail && !since) {
          cmdArgs.push("--tail", tail.toString());
        }

        if (level && level !== "all") {
          cmdArgs.push("--level", level);
        }

        if (since) {
          cmdArgs.push("--since", since);
        }

        // Add format flag for JSON output
        cmdArgs.push("--format", "json");

        // Execute the logs command
        const { stdout } = await execAsync(
          `azd app logs ${cmdArgs.join(" ")}`,
          {
            maxBuffer: 10 * 1024 * 1024, // 10MB buffer for large logs
          }
        );

        // Parse the line-by-line JSON output
        const logEntries = stdout
          .trim()
          .split("\n")
          .filter((line) => line.trim())
          .map((line) => {
            try {
              return JSON.parse(line);
            } catch {
              return null;
            }
          })
          .filter((entry) => entry !== null);

        return {
          content: [
            {
              type: "text",
              text: JSON.stringify(logEntries, null, 2),
            },
          ],
        };
      }

      case "get_project_info": {
        const projectDir = (args as any)?.projectDir;
        const cmdArgs = projectDir ? ["--project", projectDir] : [];
        const result = await executeAzdAppCommand("info", cmdArgs);

        // Extract just project-level info
        const projectInfo = {
          project: result.project || "unknown",
          services: result.services?.map((s: any) => ({
            name: s.name,
            language: s.language,
            framework: s.framework,
            project: s.project,
          })),
        };

        return {
          content: [
            {
              type: "text",
              text: JSON.stringify(projectInfo, null, 2),
            },
          ],
        };
      }

      default:
        throw new Error(`Unknown tool: ${name}`);
    }
  } catch (error: any) {
    return {
      content: [
        {
          type: "text",
          text: `Error: ${error.message || error}`,
        },
      ],
      isError: true,
    };
  }
});

/**
 * Start the MCP server
 */
async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);

  // Log startup to stderr (stdout is reserved for MCP protocol)
  console.error(`${SERVER_NAME} v${SERVER_VERSION} started`);
}

main().catch((error) => {
  console.error("Fatal error:", error);
  process.exit(1);
});
