---
title: "azd app hooks"
description: "List the lifecycle hooks configured in azure.yaml."
command: "hooks"
category: "info"
order: 70
---

# azd app hooks

List the project-level lifecycle hooks configured in `azure.yaml`.

## Description

The `hooks` command reads `azure.yaml` and lists each configured lifecycle hook. It shows the command it runs, the shell it uses, whether it continues on error or needs user interaction, and any per-platform override for Windows or POSIX.

Hooks run around the development lifecycle:

| Hook | When it runs |
|------|--------------|
| `prerun` | before services start |
| `postrun` | after all services are ready |
| `prestop` | before services are stopped |
| `poststop` | after services are stopped |

## Usage

```bash
azd app hooks [flags]
```

## Examples

```bash
# List configured hooks
azd app hooks

# JSON array of hooks
azd app hooks --output json
```

## Flags

| Flag | Description |
|------|-------------|
| `-o, --output` | Output format (text, json) |
