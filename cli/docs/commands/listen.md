# azd app listen

## Overview

The `listen` command starts the azd extension server for this extension.

This is an internal command used by the azd extension framework to communicate with the extension over JSON-RPC on stdio. It is hidden from help output and is not intended to be run directly.

## Command Usage

```bash
azd app listen
```

## Notes

- Invoked by `azd` during extension operations.
- Provides extension framework hooks (for example, post-provision event handling).
- If you are looking to run the local development experience, use `azd app run` instead.

## Related Commands

- `azd app run` - Start services and dashboard
- `azd app mcp serve` - Start MCP server for AI tool integration
