# azd app completion

## Overview

The `completion` command generates shell autocompletion scripts for `azd app`.

This command is provided by Cobra (the CLI framework). Scripts are printed to stdout so you can redirect them into a file and source them from your shell profile.

## Usage

```bash
azd app completion [command]
```

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `bash` | Generate the autocompletion script for bash |
| `fish` | Generate the autocompletion script for fish |
| `powershell` | Generate the autocompletion script for PowerShell |
| `zsh` | Generate the autocompletion script for zsh |

## Examples

### Bash

```bash
# Generate a script file
azd app completion bash > ~/.azd-app-completion.bash

# Load for current session
source ~/.azd-app-completion.bash

# Persist (add to ~/.bashrc)
# source ~/.azd-app-completion.bash
```

### Zsh

```bash
# Generate a script file
azd app completion zsh > ~/.azd-app-completion.zsh

# Load for current session
source ~/.azd-app-completion.zsh

# Persist (add to ~/.zshrc)
# source ~/.azd-app-completion.zsh
```

### Fish

```bash
# Fish typically uses ~/.config/fish/completions/
mkdir -p ~/.config/fish/completions
azd app completion fish > ~/.config/fish/completions/azd-app.fish
```

### PowerShell

```powershell
# Load for the current session
azd app completion powershell | Out-String | Invoke-Expression

# Persist (add to your PowerShell profile)
# notepad $PROFILE
# azd app completion powershell | Out-String | Invoke-Expression
```

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--help` | `-h` | bool | `false` | Show help for completion |
