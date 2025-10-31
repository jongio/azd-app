# DevStack Extension for Azure Developer CLI

A collection of developer productivity commands and utilities for the Azure Developer CLI (azd).

## Overview

DevStack Extension enhances your azd experience with additional commands and workflows designed to boost developer productivity.

## Installation

### For Development & Testing

**Quick Start - Recommended Method:**

```powershell
# Clone and navigate to the project
cd c:\code\devstackazdextension

# Run the installation script (uses azd x build internally)
.\install-local.ps1

# Test it!
azd devstack hi
```

**Manual Method using azd developer tools:**

```powershell
# Build and install automatically
azd x build

# Then register in azd config (done automatically by install-local.ps1)
# Now you can use it
azd devstack hi
```

**Development Workflow:**

After making changes to your code:

```powershell
# Option 1: Quick rebuild with azd tools
azd x build

# Option 2: Use watch mode for automatic rebuilds
azd x watch

# Option 3: Run the install script again
.\install-local.ps1
```

### Uninstalling

```powershell
.\install-local.ps1 -Uninstall
```

### For Production Use

For production deployment to publish your extension to a registry:

1. Build for all platforms:
   ```bash
   azd x build --all
   ```

2. Package the extension:
   ```bash
   azd x pack
   ```

3. Create a GitHub release:
   ```bash
   azd x release --repo your-org/your-repo
   ```

4. Publish to registry:
   ```bash
   azd x publish --repo your-org/your-repo
   ```

See the [extension framework documentation](https://github.com/Azure/azure-dev/blob/main/cli/azd/docs/extension-framework.md) for details.

### Prerequisites

- [Azure Developer CLI (azd)](https://learn.microsoft.com/azure/developer/azure-developer-cli/install-azd) installed
- AZD Extensions Developer Kit installed: `azd extension install microsoft.azd.extensions`
- Go 1.21 or later (for building from source)
- PowerShell (for Windows development)

### Building the Extension

1. Clone this repository:
   ```bash
   git clone https://github.com/devstack/azd-devstack
   cd azd-devstack
   ```

2. Build for your current platform:
   ```bash
   # On Windows (PowerShell)
   .\build.ps1

   # On Unix/Linux/macOS
   chmod +x build.sh
   ./build.sh
   ```

3. Build for all platforms:
   ```bash
   # On Windows (PowerShell)
   .\build.ps1 -All

   # On Unix/Linux/macOS
   ./build.sh --all
   ```

## Commands

### `azd devstack hi`

Displays a friendly greeting and confirms the extension is working correctly.

```bash
azd devstack hi
```

**Output:**
```
👋 Hi from DevStack Extension!

🚀 DevStack Extension v0.1.0
   A collection of developer productivity commands for Azure Developer CLI

   Ready to help you build amazing things! 💡
```

## Development

### Project Structure

```
devstackazdextension/
├── extension.yaml      # Extension manifest
├── go.mod             # Go module definition
├── main.go            # Main entry point
├── cmd_hi.go          # Hi command implementation
├── build.sh           # Unix build script
├── build.ps1          # Windows build script
├── README.md          # This file
└── CHANGELOG.md       # Version history
```

### Adding New Commands

1. Create a new file for your command (e.g., `cmd_yourcommand.go`)
2. Implement the command following the cobra pattern:

```go
package main

import (
    "fmt"
    "github.com/spf13/cobra"
)

func newYourCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "yourcommand",
        Short: "Description of your command",
        RunE: func(cmd *cobra.Command, args []string) error {
            fmt.Println("Your command logic here")
            return nil
        },
    }
}
```

3. Register the command in `main.go`:

```go
rootCmd.AddCommand(newYourCommand())
```

4. Update `extension.yaml` to include the new command in the examples section

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - See LICENSE file for details

## Resources

- [Azure Developer CLI Documentation](https://learn.microsoft.com/azure/developer/azure-developer-cli/)
- [azd Extension Framework](https://github.com/Azure/azure-dev/blob/main/cli/azd/docs/extension-framework.md)
