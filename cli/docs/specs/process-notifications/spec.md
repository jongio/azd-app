# Process State Notifications

## Overview

Provide real-time notifications to users when services or processes monitored by the azd app system enter invalid or critical states. Users should receive notifications through native OS notification systems and in-dashboard alerts without needing to actively monitor the dashboard.

## User Problem

Users running multi-service applications need to monitor service health continuously. Currently, users must:
- Keep the dashboard open and visible to notice status changes
- Manually check the dashboard or run health checks to detect failures
- Miss critical service failures when the dashboard is minimized or in the background

This creates a poor developer experience where services can fail silently while users are working in other applications.

## Goals

Enable users to receive proactive notifications when monitored processes enter critical states, allowing them to:
- Continue working in other applications while monitoring service health
- Respond quickly to service failures without manual checking
- Understand what failed and why through clear, actionable notification content

## Non-Goals

- Custom notification sounds or advanced notification customization
- Historical notification logs or notification management UI
- Notifications for non-critical events like successful starts
- Third-party notification integrations (Slack, email, etc.)
- Mobile notifications (desktop only)

## Functional Requirements

### Service State Detection

The system shall detect and classify the following invalid service states:

**Critical States (Always Notify)**:
- Service process crashed or exited unexpectedly
- Service entered error status
- Service health check failures (transition from healthy to unhealthy)
- Service port no longer listening when it should be
- Service process PID invalid or process no longer exists

**Warning States (Configurable)**:
- Service taking longer than expected to start
- Service health degraded but still responding
- Service restarting frequently (flapping detection)

**State Transitions**:
- The system shall only notify on state transitions, not continuous states
- The system shall track previous state to determine if notification is needed
- The system shall deduplicate notifications for the same service and state within a time window

### Notification Delivery

**Windows Platform**:
- Use Windows Toast Notifications (Windows 10/11 native notifications)
- Display service name, status, and timestamp
- Support notification actions (view dashboard, dismiss)
- Persist notifications in Action Center until dismissed

**macOS Platform**:
- Use macOS User Notifications framework
- Display service name, status, and timestamp
- Support notification actions (view dashboard, dismiss)
- Persist in Notification Center

**Linux Platform**:
- Use libnotify/D-Bus notifications
- Display service name, status, and timestamp
- Support basic click action to open dashboard
- Follow freedesktop.org notification specification

**Dashboard Integration**:
- Display in-dashboard toast notifications for all state changes
- Show notification badge count for unacknowledged critical states
- Persist notification list in dashboard until user dismisses
- Provide visual distinction between critical and warning notifications

### Notification Content

Each notification shall include:
- **Service Name**: Which service entered invalid state
- **State Description**: Clear, actionable description of the problem
- **Timestamp**: When the state change occurred
- **Action**: Primary action (View Dashboard, View Logs, Dismiss)
- **Severity Indicator**: Visual distinction for critical vs warning

**Example Notification Content**:
- Critical: "api-service crashed - Process exited with code 1"
- Warning: "api-service health degraded - Responded with 500 status"
- Critical: "frontend-service stopped - Port 3000 no longer listening"

### User Preferences

Users shall be able to configure:
- **Notification Enablement**: Enable/disable OS notifications globally
- **Severity Filter**: Choose which severity levels trigger notifications (critical only, critical + warning)
- **Quiet Hours**: Disable notifications during specified time ranges
- **Per-Service Settings**: Enable/disable notifications for specific services
- **Dashboard Notifications**: Enable/disable in-dashboard toasts independently from OS notifications

**Default Configuration**:
- OS notifications: Enabled for critical states only
- Dashboard notifications: Enabled for all states
- No quiet hours
- All services enabled

**Persistence**:
- Preferences stored in user configuration directory
- Settings file location: `~/.azure/azd-notifications.json`
- Changes applied immediately without restart

### Dashboard Notification UI

**Toast Notifications**:
- Appear in top-right corner of dashboard
- Auto-dismiss after 5 seconds for warnings, 10 seconds for critical
- User can dismiss manually before auto-dismiss
- Stack multiple notifications with max 3 visible
- Click to view service details or logs

**Notification Center**:
- Show notification history in collapsible panel
- Group by service
- Mark as read/unread
- Clear all action
- Filter by severity
- Timestamp relative (e.g., "2 minutes ago")

**Visual Indicators**:
- Service cards show notification badge for unacknowledged critical states
- Header shows total unacknowledged notification count
- Color coding: red for critical, yellow for warning

### CLI Integration

**Command to View Notifications**:
```
azd app notifications [--service <name>] [--severity <level>] [--since <duration>]
```

**Command to Configure**:
```
azd app notifications config --set <key>=<value>
```

**Notification State Tracking**:
- Store notification events in local database
- Track acknowledged vs unacknowledged
- Retain history for 7 days by default
- Provide JSON output format for scripting

## Service Monitoring Architecture

### State Monitoring

**Continuous Monitoring**:
- Monitor all registered services in service registry
- Poll service status every 5 seconds (configurable)
- Track state transitions between polling intervals
- Correlate with WebSocket events from dashboard server

**Health Check Integration**:
- Leverage existing health check system
- Track health check success/failure transitions
- Record health check response times
- Detect health check timeout events

**Process Monitoring**:
- Monitor process PID validity
- Track unexpected process exits
- Detect port binding changes
- Monitor process resource usage anomalies

### State Evaluation

**Transition Detection**:
- Compare current state with previous state
- Identify meaningful transitions (e.g., running → error, healthy → unhealthy)
- Filter out expected transitions (e.g., starting → running)
- Apply rate limiting to prevent notification storms

**Severity Classification**:
- Critical: Service completely unavailable or crashed
- Warning: Service degraded but operational
- Info: State change that doesn't require immediate action

### Notification Pipeline

**Event Flow**:
1. State monitor detects service state transition
2. Evaluate transition against notification rules
3. Check user preferences and filters
4. Format notification content
5. Send to OS notification system
6. Send to dashboard via WebSocket
7. Store in notification history database

**Error Handling**:
- Gracefully handle OS notification system failures
- Fall back to dashboard-only notifications if OS notifications unavailable
- Log notification delivery failures
- Retry failed OS notifications once

## User Experience

### First-Time Setup

**On First Run**:
- Request OS notification permissions
- Show onboarding modal explaining notification features
- Allow user to configure initial preferences
- Provide "Try It" button to send test notification

**Permission Handling**:
- Detect if OS notifications are blocked
- Show clear message if permissions denied
- Provide instructions to enable in OS settings
- Continue with dashboard-only mode if denied

### Notification Interaction

**Clicking OS Notification**:
- Bring dashboard window to foreground
- Navigate to failing service details
- Mark notification as acknowledged
- Scroll to service logs if applicable

**Dashboard Toast Interaction**:
- Click to view service details
- Hover to pause auto-dismiss timer
- Click X to dismiss immediately
- Click action button to view logs

### Notification Management

**Acknowledging Notifications**:
- Automatically acknowledge when user views service in dashboard
- Manual acknowledge via notification center
- Bulk acknowledge all for a service
- Auto-acknowledge when service returns to healthy state

**Notification History**:
- View all notifications from past 7 days
- Search/filter by service, severity, date range
- Export notification history as JSON or CSV
- Clear history older than N days

## Acceptance Criteria

### Critical State Notifications

- When a service process crashes, user receives OS notification within 10 seconds
- When a service enters error state, user receives OS notification within 10 seconds
- When a service health changes from healthy to unhealthy, user receives OS notification within 10 seconds
- When a service port stops listening, user receives OS notification within 10 seconds

### Dashboard Integration

- Dashboard shows toast notification for all service state changes
- Notification badge count reflects unacknowledged critical notifications
- Notification center displays notification history
- User can dismiss individual notifications or clear all

### Configuration

- User can enable/disable OS notifications via settings UI
- User can configure severity filter (critical only, critical + warning)
- User can disable notifications for specific services
- Settings persist across dashboard restarts

### Cross-Platform

- Notifications work on Windows 10/11 using native toast system
- Notifications work on macOS using native notification center
- Notifications work on Linux using libnotify
- Fallback to dashboard-only mode if OS notifications unavailable

### User Experience

- Notifications include service name, status, and timestamp
- Clicking notification opens dashboard and focuses service
- No duplicate notifications for same service state within 5 minutes
- Dashboard continues to function if notification system fails

## Technical Constraints

- Must work with existing service registry and health check systems
- Must not impact dashboard or service performance
- Must handle OS notification system unavailability gracefully
- Must support Windows, macOS, and Linux
- Notification delivery latency must be under 10 seconds from state change

## Dependencies

- Existing service monitoring and health check system
- Service registry for tracking service state
- Dashboard WebSocket infrastructure for real-time updates
- OS-specific notification libraries (Windows WinRT, macOS UserNotifications, Linux libnotify)

## Success Metrics

- Notification delivery latency: < 10 seconds from state change to notification
- False positive rate: < 5% (notifications for non-critical state changes)
- User engagement: > 60% of users enable notifications
- Notification accuracy: > 95% of critical states generate notifications

## Future Enhancements

- Notification aggregation (e.g., "3 services failed")
- Custom notification templates per service type
- Notification delivery to external systems (webhooks, Slack)
- Rich notifications with inline actions (restart service, view logs)
- Notification analytics and reporting
- Smart notification routing based on service ownership
