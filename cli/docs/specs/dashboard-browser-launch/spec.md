# Dashboard Browser Launch Specification

## Overview
Automatically launch the azd dashboard in the user's preferred browser when running `azd app run`. Support multiple browser targets with configurable preferences at both machine and project levels.

## Functional Requirements

### 1. Default Behavior
- When `azd app run` starts the dashboard, automatically open it in the default system browser
- Launch URL: `http://localhost:{port}` where port is the dashboard's assigned port (default 4280)
- Launch occurs after dashboard server is confirmed running and ready to accept connections
- Launch timing: Immediately after displaying "📊 Dashboard: http://localhost:4280" message

### 2. Browser Launch Control Flag
- **Flag**: `--no-browser` or `--browser=false`
- **Short Flag**: None (explicit flag required)
- **Type**: Boolean
- **Default**: `true` (browser launch enabled by default)
- **Behavior**: 
  - When `false`: Dashboard starts but browser does not launch
  - When `true` (default): Dashboard starts and browser launches automatically
- **Example**: `azd app run --no-browser`

### 3. Browser Target Detection and Priority

Browser selection follows this priority order (highest to lowest):

#### Priority 1: Explicit Flag (Command-line Override)
- **Flag**: `--browser=<target>`
- **Values**: `default`, `vscode`, `system`, `none`
- **Examples**:
  - `azd app run --browser=vscode` - Force VS Code Simple Browser
  - `azd app run --browser=system` - Force system default browser
  - `azd app run --browser=default` - Use default system browser (same as `system`)
  - `azd app run --browser=none` - Do not launch browser (same as `--no-browser`)

#### Priority 2: Project Configuration (azure.yaml)
- **Location**: `azure.yaml` in project root
- **Configuration Section**: `dashboard` settings
- **Property**: `browser`
- **Values**: `default`, `vscode`, `system`, `none`
- **Example**:
```yaml
name: my-app

dashboard:
  browser: vscode  # Launch in VS Code Simple Browser when available
  
services:
  web:
    language: js
    project: ./src/web
```

#### Priority 3: User Configuration (azd config)
- **Scope**: Machine-level user preference
- **Config Key**: `app.dashboard.browser`
- **Values**: `default`, `vscode`, `system`, `none`
- **Set Command**: `azd config set app.dashboard.browser vscode`
- **Get Command**: `azd config get app.dashboard.browser`
- **Unset Command**: `azd config unset app.dashboard.browser`
- **Storage**: `~/.azd/config.json` (user home directory)

#### Priority 4: Environment Detection (Auto-detect VS Code)
- Detect if running inside VS Code terminal by checking environment variables:
  - `TERM_PROGRAM=vscode` (primary indicator)
  - `VSCODE_GIT_IPC_HANDLE` (secondary indicator)
  - `VSCODE_INJECTION` (tertiary indicator)
- If detected AND no higher priority setting exists: Launch in VS Code Simple Browser
- If not detected: Fall back to system default browser

#### Priority 5: System Default (Fallback)
- Use operating system's default browser
- Implementation varies by OS:
  - **Windows**: `cmd /c start <url>`
  - **macOS**: `open <url>`
  - **Linux**: `xdg-open <url>` or `sensible-browser <url>`

### 4. Browser Target Behaviors

#### `default` or `system`
- Launch in operating system's default browser
- Equivalent behaviors
- Example: Chrome if set as system default, Firefox if set as system default, etc.

#### `vscode`
- If running in VS Code: Launch VS Code Simple Browser
- If NOT running in VS Code: Fall back to system default browser
- VS Code Simple Browser opens as editor pane within VS Code
- User can move/resize pane like any VS Code editor

#### `none`
- Do not launch any browser
- Dashboard still starts and is accessible manually
- Equivalent to `--no-browser` flag

### 5. Configuration Examples

#### Machine-Level Preference
```bash
# Set user preference for all projects
azd config set app.dashboard.browser vscode

# View current setting
azd config get app.dashboard.browser

# Remove preference (use system default)
azd config unset app.dashboard.browser
```

#### Project-Level Preference
```yaml
# azure.yaml
name: team-project

dashboard:
  browser: vscode  # Everyone on team uses VS Code Simple Browser
  
services:
  api:
    language: python
    project: ./backend
```

#### Command-Line Override
```bash
# Override project/user settings for this run only
azd app run --browser=system

# Disable browser launch for this run
azd app run --no-browser
```

### 6. Priority Resolution Examples

**Example 1: All levels configured**
- Command flag: `--browser=system`
- azure.yaml: `browser: vscode`
- azd config: `app.dashboard.browser=none`
- Environment: Running in VS Code
- **Result**: System default browser (command flag wins)

**Example 2: Project and user config**
- Command flag: None
- azure.yaml: `browser: vscode`
- azd config: `app.dashboard.browser=system`
- Environment: Running in VS Code
- **Result**: VS Code Simple Browser (project config wins)

**Example 3: User config only**
- Command flag: None
- azure.yaml: No browser setting
- azd config: `app.dashboard.browser=vscode`
- Environment: Running in PowerShell (not VS Code)
- **Result**: System default browser (VS Code not detected, fallback)

**Example 4: Auto-detection**
- Command flag: None
- azure.yaml: No browser setting
- azd config: No setting
- Environment: Running in VS Code terminal
- **Result**: VS Code Simple Browser (auto-detected)

**Example 5: Pure defaults**
- Command flag: None
- azure.yaml: No browser setting
- azd config: No setting
- Environment: Running in standard terminal
- **Result**: System default browser (final fallback)

### 7. VS Code Simple Browser Integration

#### Detection Logic
Check environment variables in order:
1. `TERM_PROGRAM=vscode` - Primary and most reliable
2. `VSCODE_GIT_IPC_HANDLE` - Present in VS Code integrated terminal
3. `VSCODE_INJECTION` - VS Code shell integration marker

If ANY of these exist, VS Code is detected.

#### Launch Mechanism
When launching in VS Code Simple Browser:
1. Detect VS Code environment
2. Build `vscode://` URI: `vscode://vscode.open-simple-browser?url=<dashboard-url>`
3. Use OS to open the URI (same as opening any URL)
4. VS Code receives the URI and opens Simple Browser in current instance
5. VS Code opens Simple Browser in new editor pane within the same window
6. User can split, move, or close pane as needed

**Note**: The `vscode://` URI scheme ensures the Simple Browser opens in the **same VS Code instance** where the terminal is running, not in a new window or instance.

#### Simple Browser Features
- Embedded browser within VS Code
- Synchronized with VS Code theme
- Accessible via VS Code command palette
- Can be opened in split view alongside code
- Useful for development without switching windows

### 8. Error Handling

#### Browser Launch Failures
- If browser launch fails (process error, command not found):
  - Log warning to console
  - Display message: "⚠️ Could not open browser automatically. Dashboard available at: http://localhost:4280"
  - Dashboard continues running
  - Exit code remains 0 (not a critical error)

#### Invalid Browser Target
- If `--browser=<invalid>` specified:
  - Display error: "Invalid browser target: <invalid>. Valid options: default, vscode, system, none"
  - Exit with code 2 (validation error)
  - Do not start dashboard

#### Configuration Validation
- If `azure.yaml` contains invalid browser value:
  - Log warning: "Invalid browser setting in azure.yaml: <value>. Using default."
  - Continue with next priority level
  - Do not fail startup

### 9. User Experience

#### Console Output with Browser Launch
```bash
$ azd app run

✓ Prerequisites check passed
✓ Dependencies installed

🚀 Starting services

web              → http://localhost:3000
api              → http://localhost:3001

📊 Dashboard: http://localhost:4280
🌐 Opening dashboard in VS Code...

💡 Press Ctrl+C to stop all services
```

#### Console Output without Browser Launch
```bash
$ azd app run --no-browser

✓ Prerequisites check passed
✓ Dependencies installed

🚀 Starting services

web              → http://localhost:3000
api              → http://localhost:3001

📊 Dashboard: http://localhost:4280

💡 Press Ctrl+C to stop all services
```

#### Console Output with Launch Failure
```bash
$ azd app run

✓ Prerequisites check passed
✓ Dependencies installed

🚀 Starting services

web              → http://localhost:3000
api              → http://localhost:3001

📊 Dashboard: http://localhost:4280
⚠️ Could not open browser automatically. Dashboard available at: http://localhost:4280

💡 Press Ctrl+C to stop all services
```

### 10. Configuration Schema

#### azd config JSON Format
```json
{
  "app": {
    "dashboard": {
      "browser": "vscode"
    }
  }
}
```

#### azure.yaml Schema Addition
```yaml
dashboard:
  browser: vscode | system | default | none
  # Optional: Future enhancements
  # port: 4280
  # autoOpen: true
```

### 11. Platform-Specific Behavior

#### Windows
- System browser launch: `cmd /c start "" "<url>"`
- VS Code detection: Check `TERM_PROGRAM`, `VSCODE_GIT_IPC_HANDLE`
- VS Code launch: `code.cmd --open-url <url>` or `code --open-url <url>`

#### macOS
- System browser launch: `open "<url>"`
- VS Code detection: Check `TERM_PROGRAM`, `VSCODE_GIT_IPC_HANDLE`
- VS Code launch: `code --open-url <url>`

#### Linux
- System browser launch: `xdg-open "<url>"` (preferred) or `sensible-browser "<url>"` (fallback)
- VS Code detection: Check `TERM_PROGRAM`, `VSCODE_GIT_IPC_HANDLE`
- VS Code launch: `code --open-url <url>`

### 12. Non-Functional Requirements

#### Performance
- Browser launch should not block dashboard startup
- Launch in separate goroutine to avoid blocking main process
- Timeout for launch command: 5 seconds
- If launch times out, log warning and continue

#### Reliability
- Browser launch failure does not stop dashboard
- Dashboard remains accessible if browser launch fails
- Clear error messages for configuration issues

#### Compatibility
- Works across Windows, macOS, and Linux
- Supports all major browsers as system default
- VS Code versions: 1.60+ (when Simple Browser was introduced)

#### Security
- No sensitive data in browser launch URL
- Use localhost only (no external URLs)
- Dashboard server requires localhost binding

### 13. Future Enhancements (Not in Initial Scope)

These are potential future improvements, not included in initial implementation:

- **Custom Browser Path**: Allow specifying exact browser executable
  ```bash
  azd config set app.dashboard.browser.path "/Applications/Firefox.app"
  ```

- **Multiple Browser Launch**: Open in multiple browsers simultaneously
  ```bash
  azd app run --browser=vscode,chrome
  ```

- **Browser Profiles**: Support different browser profiles
  ```bash
  azd config set app.dashboard.browser.profile "Development"
  ```

- **Dashboard Tabs**: Open specific dashboard view/tab
  ```bash
  azd app run --browser-view=logs
  ```

- **Wait for Ready**: Advanced health check before launch
  - Wait for specific HTTP response
  - Retry logic with exponential backoff

- **Browser Arguments**: Pass custom arguments to browser
  ```yaml
  dashboard:
    browser: chrome
    browserArgs: ["--incognito", "--new-window"]
  ```

## Acceptance Criteria

### Must Have (Initial Implementation)

1. **Default Launch**: `azd app run` automatically opens dashboard in system default browser
2. **Disable Flag**: `--no-browser` flag prevents browser launch
3. **Browser Target Flag**: `--browser=<target>` flag overrides all other settings
4. **VS Code Detection**: Automatically detects VS Code environment via `TERM_PROGRAM`
5. **VS Code Launch**: Launches VS Code Simple Browser when in VS Code and configured/detected
6. **System Launch**: Launches system default browser on Windows, macOS, and Linux
7. **User Config**: `azd config set app.dashboard.browser <target>` persists machine-level preference
8. **Project Config**: `azure.yaml` supports `dashboard.browser` setting
9. **Priority Order**: Command flag > Project config > User config > Auto-detect > System default
10. **Error Handling**: Browser launch failures do not stop dashboard, show warning
11. **Validation**: Invalid browser targets show error and prevent startup
12. **Cross-Platform**: Works on Windows, macOS, and Linux

### Should Have (Nice to Have)

1. **Clear Messaging**: Console output indicates which browser target was used
2. **Config Commands**: Help text and examples for config commands
3. **Documentation**: Updated command reference with all flag options
4. **Async Launch**: Browser launch does not block dashboard startup

### Won't Have (Initial Release)

1. Custom browser paths
2. Multiple simultaneous browser launches
3. Browser profiles
4. Dashboard view/tab targeting
5. Advanced health check before launch
6. Custom browser arguments

## Testing Requirements

### Unit Tests
- Browser target priority resolution logic
- VS Code environment detection
- Configuration parsing and validation
- Platform-specific command building

### Integration Tests
- Browser launch on each platform (Windows, macOS, Linux)
- VS Code Simple Browser launch
- Flag combinations and priority
- Error scenarios (browser not found, invalid config)

### Manual Tests
- Launch in VS Code terminal → Simple Browser opens
- Launch in PowerShell → System browser opens
- Launch with `--browser=vscode` outside VS Code → System browser fallback
- Launch with `--no-browser` → No browser opens
- Set user config → Respected in subsequent runs
- Set project config → Overrides user config
- Command flag → Overrides all other settings

## Dependencies

### External Commands
- **Windows**: `cmd`, `start`, `code.cmd` or `code`
- **macOS**: `open`, `code`
- **Linux**: `xdg-open` or `sensible-browser`, `code`

### Environment Variables
- `TERM_PROGRAM` - VS Code detection
- `VSCODE_GIT_IPC_HANDLE` - VS Code detection
- `VSCODE_INJECTION` - VS Code detection
- `PATH` - Locating `code` command

### Configuration Files
- `~/.azd/config.json` - User preferences
- `azure.yaml` - Project preferences

## Impact on Existing Features

### azd app run Command
- Add `--no-browser` flag
- Add `--browser=<target>` flag
- Add browser launch after dashboard starts
- No breaking changes to existing behavior

### Dashboard Server
- No changes required
- Continues to serve on assigned port
- Browser launch is separate concern

### Configuration System
- Add new config key: `app.dashboard.browser`
- Existing config system handles new key
- No schema migration required

## Success Metrics

1. **Adoption**: >80% of users keep default auto-launch enabled
2. **VS Code Usage**: >60% of VS Code users see Simple Browser launch
3. **Configuration**: >20% of users customize browser preference
4. **Errors**: <5% of launches fail (log telemetry)
5. **User Feedback**: Positive feedback on auto-launch convenience

## Open Questions

None - all requirements defined.

## References

- VS Code Simple Browser: https://code.visualstudio.com/docs/editor/custom-layout#_simple-browser
- azd config command: Existing `azd config` implementation
- azure.yaml schema: Existing azure.yaml parsing

---

**Version**: 1.0  
**Status**: Draft  
**Last Updated**: 2025-11-23  
**Author**: Product Team
