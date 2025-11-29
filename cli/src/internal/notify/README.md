# OS Notification System

Cross-platform OS notification support for Windows, macOS, and Linux.

## Features

- **Cross-Platform**: Works on Windows 10/11, macOS, and Linux
- **Native Integration**: Uses platform-specific notification APIs
  - Windows: Toast Notifications (WinRT)
  - macOS: User Notifications (osascript)
  - Linux: libnotify (notify-send)
- **Severity Levels**: Critical, warning, and info notifications
- **Graceful Fallback**: Detects availability and handles permission denied
- **Non-Blocking**: Async notification delivery with timeout
- **Action Support**: Notification action buttons (platform dependent)

## Usage

### Basic Example

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/jongio/azd-app/cli/src/internal/notify"
)

func main() {
    // Create notifier with default config
    config := notify.DefaultConfig()
    notifier, err := notify.New(config)
    if err != nil {
        log.Fatalf("Failed to create notifier: %v", err)
    }
    defer notifier.Close()
    
    // Check if notifications are available
    if !notifier.IsAvailable() {
        log.Println("OS notifications not available, using dashboard-only mode")
        return
    }
    
    // Send notification
    ctx := context.Background()
    notification := notify.Notification{
        Title:     "API Service",
        Message:   "Service has crashed",
        Severity:  "critical",
        Timestamp: time.Now(),
    }
    
    if err := notifier.Send(ctx, notification); err != nil {
        log.Printf("Failed to send notification: %v", err)
    }
}
```

### Custom Configuration

```go
config := notify.Config{
    AppName: "My Application",
    AppID:   "com.mycompany.myapp",
    Timeout: 10 * time.Second,
}

notifier, err := notify.New(config)
```

### Notification with Actions

```go
notification := notify.Notification{
    Title:    "Service Error",
    Message:  "API service stopped responding",
    Severity: "critical",
    Actions: []notify.Action{
        {ID: "view", Label: "View Dashboard"},
        {ID: "restart", Label: "Restart Service"},
    },
    Data: map[string]string{
        "service": "api-service",
        "pid":     "12345",
    },
}

notifier.Send(ctx, notification)
```

### Request Permission

```go
// Request notification permissions (triggers OS prompt)
if err := notifier.RequestPermission(ctx); err != nil {
    if err == notify.ErrNotAvailable {
        log.Println("Notifications not available")
    } else if err == notify.ErrPermissionDenied {
        log.Println("User denied permissions")
    }
}
```

## API Reference

### Types

#### Notification
```go
type Notification struct {
    Title     string            // Notification title
    Message   string            // Notification body
    Severity  string            // "critical", "warning", "info"
    Timestamp time.Time         // When notification created
    Actions   []Action          // Action buttons
    Data      map[string]string // Arbitrary metadata
}
```

#### Action
```go
type Action struct {
    ID    string // Action identifier
    Label string // Button label
}
```

#### Config
```go
type Config struct {
    AppName string        // Application name
    AppID   string        // Platform-specific app ID
    Timeout time.Duration // Operation timeout
}
```

### Interface

#### Notifier
```go
type Notifier interface {
    Send(ctx context.Context, notification Notification) error
    IsAvailable() bool
    RequestPermission(ctx context.Context) error
    Close() error
}
```

### Functions

#### New
```go
func New(config Config) (Notifier, error)
```
Creates a new platform-specific notifier.

#### DefaultConfig
```go
func DefaultConfig() Config
```
Returns default configuration:
- AppName: "Azure Developer CLI"
- AppID: "Microsoft.AzureDeveloperCLI"
- Timeout: 5 seconds

## Platform-Specific Details

### Windows

- Uses PowerShell and WinRT Toast Notifications API
- Requires Windows 10/11
- No explicit permission request needed
- Notifications persist in Action Center
- Supports custom app ID for notification grouping

### macOS

- Uses `osascript` to trigger User Notifications
- First notification triggers permission prompt automatically
- Notifications persist in Notification Center
- Supports title and subtitle

### Linux

- Uses `notify-send` (libnotify) via D-Bus
- Requires `notify-send` to be installed
- Requires D-Bus session bus
- No explicit permission request needed
- Urgency levels: low (info), normal (warning), critical
- Critical notifications stay visible until dismissed

## Error Handling

The package provides predefined errors:

- `ErrNotAvailable`: OS notifications not available on this system
- `ErrPermissionDenied`: User denied notification permissions
- `ErrNotificationFailed`: Failed to send notification
- `ErrTimeout`: Notification operation timed out

Always check `IsAvailable()` before sending notifications and handle errors gracefully:

```go
if !notifier.IsAvailable() {
    // Fall back to dashboard-only notifications
    return
}

if err := notifier.Send(ctx, notification); err != nil {
    log.Printf("Notification failed: %v", err)
    // Continue execution, don't crash
}
```

## Testing

Run tests for current platform:
```bash
go test -v ./internal/notify
```

Run tests with coverage:
```bash
go test -cover ./internal/notify
```

## Integration Example

```go
// In state monitor callback
monitor.AddListener(func(transition StateTransition) {
    if transition.Severity == SeverityCritical {
        notification := notify.Notification{
            Title:    transition.ServiceName,
            Message:  transition.Description,
            Severity: "critical",
            Timestamp: time.Now(),
        }
        
        if err := notifier.Send(context.Background(), notification); err != nil {
            log.Printf("Failed to send notification: %v", err)
        }
    }
})
```

## Dependencies

- Windows: PowerShell (built-in on Windows 10/11)
- macOS: osascript (built-in on macOS)
- Linux: notify-send (install via package manager if needed)

## License

See LICENSE file in repository root.
