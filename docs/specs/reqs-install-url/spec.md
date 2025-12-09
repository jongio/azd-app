# Reqs Install URL Enhancement

## Overview

Enhance the `azd app reqs` command to display installation URLs when requirement checks fail, and allow users to specify custom install URLs for custom requirements.

## Problem

Currently, when a requirement check fails, users see:
```
❌ mytool: NOT INSTALLED (required: 1.0.0)
```

Users must then search for installation instructions, which slows down onboarding.

## Solution

1. Add `installUrl` field to the `requirement` schema
2. Provide built-in install URLs for known tools
3. Display install URL in failure messages
4. Support custom install URLs for custom requirements

## Schema Changes

Add `installUrl` property to `requirement` definition in `schemas/v1.1/azure.yaml.json`:

```yaml
reqs:
  - name: node
    minVersion: "18.0.0"
  - name: mytool
    minVersion: "1.0.0"
    command: mytool
    args: ["--version"]
    installUrl: "https://example.com/mytool/install"
```

## Built-in Install URLs

| Tool | Install URL |
|------|-------------|
| node | https://nodejs.org/ |
| npm | https://nodejs.org/ |
| pnpm | https://pnpm.io/installation |
| yarn | https://yarnpkg.com/getting-started/install |
| python | https://www.python.org/downloads/ |
| pip | https://www.python.org/downloads/ |
| poetry | https://python-poetry.org/docs/#installation |
| uv | https://docs.astral.sh/uv/getting-started/installation/ |
| docker | https://www.docker.com/products/docker-desktop |
| git | https://git-scm.com/downloads |
| go | https://go.dev/dl/ |
| dotnet | https://dotnet.microsoft.com/download |
| aspire | https://learn.microsoft.com/dotnet/aspire/setup-tooling |
| azd | https://aka.ms/install-azd |
| az | https://aka.ms/installazurecli |
| func | https://learn.microsoft.com/azure/azure-functions/functions-run-local#install-the-azure-functions-core-tools |
| java | https://adoptium.net/ |
| mvn | https://maven.apache.org/install.html |
| gradle | https://gradle.org/install/ |

## Output Changes

### Failed Requirement (Built-in Tool)

```
❌ docker: NOT INSTALLED (required: 20.0.0)
   Install: https://www.docker.com/products/docker-desktop
```

### Failed Requirement (Custom Tool with URL)

```
❌ mytool: NOT INSTALLED (required: 1.0.0)
   Install: https://example.com/mytool/install
```

### Failed Requirement (Custom Tool without URL)

```
❌ mytool: NOT INSTALLED (required: 1.0.0)
```

### JSON Output

Add `installUrl` field to `ReqResult`:

```json
{
  "name": "docker",
  "installed": false,
  "required": "20.0.0",
  "satisfied": false,
  "message": "Not installed",
  "installUrl": "https://www.docker.com/products/docker-desktop"
}
```

## Files to Modify

1. `schemas/v1.1/azure.yaml.json` - Add `installUrl` property to requirement definition
2. `cli/src/cmd/app/commands/reqs.go` - Add `InstallUrl` to `Prerequisite` and `ReqResult`, add URL registry, update output
3. `cli/src/internal/pathutil/pathutil.go` - Consolidate install suggestions with URLs
4. `cli/src/cmd/app/commands/reqs_test.go` - Add tests for install URL display
5. `cli/docs/commands/reqs.md` - Document new `installUrl` field

## Implementation Notes

- Built-in URLs take precedence unless user specifies custom `installUrl`
- URL display only on failure (not installed or version mismatch)
- Keep output concise - single "Install:" line with clickable URL
- JSON output always includes `installUrl` when available
