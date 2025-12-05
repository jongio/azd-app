# azd app version

## Overview

The `version` command displays the current version of the `azd app` extension.

## Purpose

- **Version Information**: Show installed extension version
- **Compatibility Checking**: Verify version for troubleshooting
- **Update Detection**: Compare with latest available version (future)

## Command Usage

```bash
azd app version
```

### Flags

None. This command has no additional flags.

## Execution Flow

```
┌─────────────────────────────────────────────────────────────┐
│                 azd app version                              │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Read Version from Build-Time Variable                       │
│  - Embedded during compilation                               │
│  - Located in version package                                │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Display Version String                                      │
│  Format: "azd app extension version X.Y.Z"                   │
└─────────────────────────────────────────────────────────────┘
```

## Version Format

The version follows **semantic versioning** (semver):

```
MAJOR.MINOR.PATCH

Example: 0.5.1
  │    │   │
  │    │   └─ Patch: Bug fixes
  │    └───── Minor: New features (backward compatible)
  └────────── Major: Breaking changes
```

## Output

### Standard Output

```bash
$ azd app version

  version
  Show version information

  Version  0.5.1
  Built    2024-11-04T10:00:00Z
```

### JSON Output

```bash
$ azd app version --output json
{
  "version": "0.5.1",
  "buildTime": "2024-11-04T10:00:00Z"
}
```

## Common Use Cases

### 1. Check Installed Version

```bash
$ azd app version
azd app extension
Version: 0.5.1
Built: 2024-11-04T10:00:00Z
```

### 2. Verify Installation

```bash
$ which azd-app
/usr/local/bin/azd-app

$ azd app version
azd app extension
Version: 0.5.1
Built: 2024-11-04T10:00:00Z
```

### 3. Troubleshooting

When reporting issues, include version information:

```bash
$ azd app version
azd app extension
Version: 0.5.1
Built: 2024-11-04T10:00:00Z

# Include in bug report
```

## Integration with azd

The `azd app` extension integrates with the main `azd` CLI:

```bash
# Check azd version
$ azd version
azd version 1.5.0

# Check azd app version
$ azd app version
azd app extension
Version: 0.5.1
Built: 2024-11-04T10:00:00Z
```

## Version Compatibility

The extension is designed to be compatible with:

| Component | Minimum Version |
|-----------|----------------|
| azd CLI | 1.0.0 |
| Go runtime | 1.21 |
| Node.js (for services) | 18.0.0 |
| Python (for services) | 3.11.0 |
| .NET (for services) | 8.0.0 |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |

## Related Commands

- `azd version` - Show main azd CLI version
- `azd app reqs` - Check tool versions for your project

## Examples

### Example 1: Simple Version Check

```bash
$ azd app version
azd app extension
Version: 0.5.1
Built: 2024-11-04T10:00:00Z
```

### Example 2: In Scripts

```bash
#!/bin/bash

# Check if azd app is installed
if ! command -v azd &> /dev/null; then
    echo "azd app not found"
    exit 1
fi

# Check version
VERSION=$(azd app version | awk '{print $NF}')
echo "Using azd app version: $VERSION"
```

### Example 3: Compare Versions

```bash
$ azd version && azd app version
azd version 1.5.0
azd app extension
Version: 0.5.1
Built: 2024-11-04T10:00:00Z
```
