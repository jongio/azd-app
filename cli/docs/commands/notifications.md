# azd app notifications

Manage process notifications for service state changes and events.

## Commands

### azd app notifications list

View notification history with filtering options.

**Usage:**
```bash
azd app notifications list [flags]
```

**Flags:**
- `--service`, `-s` - Filter by service name
- `--unread`, `-u` - Show only unread notifications
- `--limit`, `-n` - Maximum number of notifications to show (default: 50)

**Examples:**
```bash
# View all recent notifications
azd app notifications list

# View unread notifications only
azd app notifications list --unread

# View notifications for a specific service
azd app notifications list --service api

# Limit results to 10 items
azd app notifications list --limit 10
```

**Output:**
```
ID  SERVICE  SEVERITY  MESSAGE                       TIME      READ
1   api      critical  Service crashed unexpectedly  2m ago
2   web      warning   High memory usage detected    5m ago    ✓
3   db       info      Backup completed successfully 1h ago    ✓
```

---

### azd app notifications mark-read

Mark notification(s) as read.

**Usage:**
```bash
azd app notifications mark-read [id] [flags]
```

**Flags:**
- `--all`, `-a` - Mark all notifications as read

**Examples:**
```bash
# Mark specific notification as read
azd app notifications mark-read 42

# Mark all notifications as read
azd app notifications mark-read --all
```

---

### azd app notifications clear

Clear notification history.

**Usage:**
```bash
azd app notifications clear [flags]
```

**Flags:**
- `--older-than <duration>` - Clear notifications older than specified duration

**Examples:**
```bash
# Clear all notifications (prompts for confirmation)
azd app notifications clear

# Clear notifications older than 7 days
azd app notifications clear --older-than 7d

# Clear notifications older than 24 hours
azd app notifications clear --older-than 24h
```

**Duration Formats:**
- `24h` - 24 hours
- `7d` - 7 days (use `168h`)
- `30m` - 30 minutes

---

### azd app notifications stats

Show notification statistics.

**Usage:**
```bash
azd app notifications stats
```

**Output:**
```
Notification Statistics:
  Total:    156
  Unread:   12
  Critical: 3
```

---

## Database Location

Notifications are stored in:
- **Linux/macOS**: `~/.local/share/azd/notifications.db`
- **Windows**: `%LOCALAPPDATA%\azd\notifications.db`

## Configuration

Notification preferences are stored in `~/.azd/notifications.json`:

```json
{
  "osNotifications": true,
  "dashboardNotifications": true,
  "severityFilter": "critical",
  "quietHours": [
    {"start": "22:00", "end": "08:00"}
  ],
  "serviceSettings": {
    "api": {"enabled": true},
    "web": {"enabled": false}
  },
  "rateLimitWindow": "30s"
}
```

### Configuration Options

- `osNotifications` (bool) - Enable OS-level notifications
- `dashboardNotifications` (bool) - Enable dashboard toast notifications
- `severityFilter` (string) - Minimum severity level: "critical", "warning", "info", "all"
- `quietHours` (array) - Time ranges to suppress notifications (24-hour format)
- `serviceSettings` (object) - Per-service notification enable/disable
- `rateLimitWindow` (string) - Deduplication window (e.g., "30s", "5m")

## Related Commands

- `azd app run` - Start services (triggers notifications)
- `azd app info` - View service status
- `azd app logs` - View service logs

## See Also

- [Process Notifications Specification](../specs/process-notifications/README.md)
- [Integration Guide](../specs/process-notifications/INTEGRATION.md)
