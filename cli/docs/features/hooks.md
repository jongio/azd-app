# Hooks Documentation

## Overview

Hooks are lifecycle scripts that execute automatically before and after the `azd app run` command, similar to azd's `preprovision` and `postprovision` hooks. They allow you to automate setup tasks, validation, notifications, and cleanup operations.

## Available Hooks

### `prerun`
Executes **before** starting any services. Use for:
- Database migrations
- Environment validation
- Dependency checks
- Setting up test data
- Pre-flight checks

### `postrun`
Executes **after** all services are ready. Use for:
- Notifications (e.g., Slack, email)
- Opening browser windows
- Running integration tests
- Logging startup information
- Registering services with discovery systems

## Configuration

Hooks are configured in the `azure.yaml` file under the `hooks` section:

```yaml
name: my-app

hooks:
  prerun:
    run: ./scripts/setup.sh
    shell: bash
    continueOnError: false
    interactive: false
  postrun:
    run: echo "Services are ready!"
    shell: sh

services:
  web:
    language: TypeScript
    project: ./frontend
```

## Hook Properties

### `run` (required)
The script or command to execute. Can be:
- Path to a script file: `./scripts/setup.sh`
- Inline command: `echo "Starting services"`
- Complex command: `npm run db:migrate && npm run seed`

### `shell` (optional)
The shell to use for executing the script. Defaults based on platform:
- **Windows**: `pwsh` > `powershell` > `cmd`
- **Linux/macOS**: `bash` > `sh`

Supported shells:
- `sh` - POSIX shell
- `bash` - Bash shell
- `pwsh` - PowerShell Core
- `powershell` - Windows PowerShell
- `cmd` - Windows Command Prompt

### `continueOnError` (optional, default: `false`)
Whether to continue if the hook fails:
- `false`: Stop execution and report error
- `true`: Log warning and continue

```yaml
hooks:
  prerun:
    run: ./optional-setup.sh
    continueOnError: true  # Don't fail if this script fails
```

### `interactive` (optional, default: `false`)
Whether the script requires user interaction:
- `false`: Script runs non-interactively (stdin is not connected)
- `true`: Script can prompt for user input

```yaml
hooks:
  prerun:
    run: ./interactive-setup.sh
    interactive: true  # Script can prompt user
```

## Platform-Specific Hooks

You can specify different hooks for Windows and POSIX (Linux/macOS) systems:

```yaml
hooks:
  prerun:
    run: echo "Default script"
    shell: sh
    windows:
      run: Write-Host "Windows script"
      shell: pwsh
    posix:
      run: echo "POSIX script"
      shell: bash
```

Platform-specific properties override the base configuration. You can override:
- `run`
- `shell`
- `continueOnError`
- `interactive`

## Examples

### Basic Example

```yaml
name: basic-app

hooks:
  prerun:
    run: echo "Starting application..."
  postrun:
    run: echo "Application ready!"

services:
  web:
    project: .
    ports: ["3000"]
```

### Database Migration

```yaml
name: app-with-db

hooks:
  prerun:
    run: npm run db:migrate
    shell: bash
    continueOnError: false

services:
  api:
    project: ./backend
    ports: ["8080"]
```

### Multi-Step Setup

```yaml
name: complex-app

hooks:
  prerun:
    run: |
      echo "Checking prerequisites..."
      npm run validate
      npm run db:migrate
      npm run seed
    shell: bash

services:
  web:
    project: .
    ports: ["3000"]
```

### Platform-Specific Scripts

```yaml
name: cross-platform-app

hooks:
  prerun:
    run: echo "Default setup"
    windows:
      run: |
        Write-Host "Windows setup"
        & .\scripts\setup-windows.ps1
      shell: pwsh
    posix:
      run: |
        echo "POSIX setup"
        ./scripts/setup-posix.sh
      shell: bash

services:
  web:
    project: .
```

### Notification After Startup

```yaml
name: notification-app

hooks:
  postrun:
    run: |
      curl -X POST https://hooks.slack.com/services/YOUR/WEBHOOK/URL \
        -H 'Content-Type: application/json' \
        -d '{"text":"Services are ready!"}'
    shell: bash
    continueOnError: true  # Don't fail if notification fails

services:
  web:
    project: .
    ports: ["3000"]
```

### Interactive Setup

```yaml
name: interactive-app

hooks:
  prerun:
    run: ./scripts/interactive-setup.sh
    interactive: true  # Allow user prompts
    shell: bash

services:
  web:
    project: .
```

### Conditional Execution

```yaml
name: conditional-app

hooks:
  prerun:
    run: |
      if [ ! -f .env ]; then
        echo "Creating .env file..."
        cp .env.example .env
      fi
    shell: bash

services:
  web:
    project: .
```

## Execution Context

### Working Directory
Hooks execute in the directory containing `azure.yaml`.

### Environment Variables
Hooks inherit all environment variables from the parent process, including:
- `AZURE_SUBSCRIPTION_ID`
- `AZURE_RESOURCE_GROUP_NAME`
- `AZURE_ENV_NAME`
- Any variables set in your shell
- Variables from `--env-file` (if specified)

### Exit Codes
- **0**: Success
- **Non-zero**: Failure (behavior depends on `continueOnError`)

## Security Considerations

### Trust Model

Hooks execute with the same permissions as your user account. When you run `azd app run`, you implicitly trust the commands defined in `azure.yaml`.

**This follows the same trust model as:**
- npm scripts (package.json)
- Makefile targets
- docker-compose.yml commands
- GitHub Actions workflows
- Azure Developer CLI (azd) hooks

### Security Guidance

1. **Review azure.yaml before running**: Especially in cloned/downloaded projects
2. **Treat azure.yaml like code**: It can execute arbitrary commands
3. **Use version control**: Track changes to hook commands
4. **Don't commit secrets**: Use environment variables for sensitive data

### What Hooks Can Do

Hooks have full access to:
- File system (read, write, delete)
- Network (HTTP requests, downloads)
- Other processes (start, stop)
- Environment variables

### Recommended Practices

- **Audit third-party templates**: Review azure.yaml before running
- **Use explicit script files**: Easier to review than inline commands
- **Pin dependencies**: Avoid `curl | bash` patterns in hooks
- **Principle of least privilege**: Don't run as root/admin if not needed

## Best Practices

1. **Keep hooks simple**: Complex logic should be in separate scripts
2. **Make hooks idempotent**: Safe to run multiple times
3. **Use absolute paths**: Or paths relative to azure.yaml location
4. **Handle errors gracefully**: Use `continueOnError` for optional steps
5. **Test on all platforms**: If using platform-specific hooks
6. **Log meaningful messages**: Help users understand what's happening
7. **Avoid long-running operations**: In prerun hooks to keep startup fast
8. **Use version control**: Include hook scripts in your repository

## Troubleshooting

### Hook Not Executing

**Symptom**: Hook doesn't run
**Solutions**:
- Verify `hooks` section is in `azure.yaml`
- Check YAML syntax and indentation
- Ensure `run` property is specified

### Script File Not Found

**Symptom**: Error: "script not found"
**Solutions**:
- Use absolute paths or paths relative to azure.yaml
- Verify script file exists: `ls -la scripts/setup.sh`
- Check file permissions: `chmod +x scripts/setup.sh`

### Hook Fails Every Time

**Symptom**: Hook always exits with error
**Solutions**:
- Test script independently: `bash ./scripts/setup.sh`
- Check exit code: `echo $?` (POSIX) or `$LASTEXITCODE` (PowerShell)
- Review script output for error messages
- Set `continueOnError: true` for non-critical hooks

### Platform-Specific Issues

**Symptom**: Hook works on one platform but not another
**Solutions**:
- Use platform-specific overrides
- Test on all target platforms
- Use cross-platform tools (e.g., Node.js scripts)

### Interactive Script Hangs

**Symptom**: Hook appears to hang waiting for input
**Solutions**:
- Set `interactive: true` if script needs user input
- Make scripts non-interactive by default
- Use environment variables instead of prompts

## Related Documentation

- [Azure Developer CLI Hooks](https://learn.microsoft.com/en-us/azure/developer/azure-developer-cli/azd-extensibility)
- [Environment Variables](./environment-variables.md)
- [CLI Reference](./cli-reference.md)

## Schema Reference

See the [azure.yaml schema](../schemas/v1.1/azure.yaml.json) for the complete hook configuration specification.
