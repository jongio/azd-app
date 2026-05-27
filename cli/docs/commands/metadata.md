# azd app metadata

## Overview

The `metadata` command is an internal command used by the Azure Developer CLI (azd) to discover extension capabilities. It outputs structured metadata about the extension's commands, flags, and supported features.

## Purpose

- **Extension Discovery**: Allows azd to introspect the extension's command tree
- **Capability Reporting**: Reports supported capabilities (custom-commands, lifecycle-events, mcp-server, etc.)
- **Integration**: Enables azd to build its command tree with extension commands

## Usage

```bash
azd app metadata
```

## Notes

This command is primarily used internally by azd and is not typically invoked directly by users. It outputs JSON metadata describing the extension's command structure and capabilities.
