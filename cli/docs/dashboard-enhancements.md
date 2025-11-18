# Dashboard Enhancements

This document describes the enhanced features added to the azd-app dashboard to make it more helpful and useful for developers.

## New Features

### 1. Environment Variables Panel

**Access:** Press `4` or navigate via sidebar

The Environment Variables panel provides comprehensive management of environment variables across all running services:

- **Search & Filter**: Search by variable name or value, filter by specific service
- **Show/Hide Values**: Toggle visibility of sensitive values with one click
- **Copy to Clipboard**: Quick copy action for any variable value
- **Multi-Service View**: See which services use each environment variable
- **Auto-Masking**: Automatically masks potentially sensitive values (keys, secrets, passwords, tokens)

**Use Cases:**
- Debug environment configuration issues
- Verify environment variables are set correctly
- Understand shared configuration across services
- Copy values for local testing

### 2. Quick Actions Panel

**Access:** Press `5` or navigate via sidebar

The Quick Actions panel provides at-a-glance service statistics and quick access to common operations:

**Stats Cards:**
- Running Services: Count of active services
- Healthy: Count of healthy services
- Errors: Count of services with errors

**Quick Actions:**
- Refresh All: Refresh status of all services
- Clear Logs: Clear all log buffers
- Export Logs: Download logs as file
- Open Terminal: Open system terminal

**Service-Specific Actions:**
- View logs for individual services
- Refresh status for specific services
- Real-time status indicators

### 3. Performance Metrics Panel

**Access:** Press `3` or navigate via sidebar

The Performance Metrics panel provides real-time insights into your development environment:

**Aggregate Metrics:**
- Active Services: Running vs total services
- Active Ports: Number of ports in use
- Average Uptime: Average service uptime across all services
- Health Score: Overall health percentage

**Service-Level Metrics Table:**
- Service name and framework
- Current status with visual indicator
- Uptime display
- Port assignment
- Health status

**Use Cases:**
- Monitor overall system health
- Identify long-running services
- Track port usage
- Quick service status overview

### 4. Service Dependencies Viewer

**Access:** Press `6` or navigate via sidebar

The Service Dependencies panel visualizes your service architecture:

**Features:**
- **Language Grouping**: Services grouped by programming language/technology
- **Status Indicators**: Visual status indicators for each service
- **Framework Display**: Shows framework and port information
- **Environment Variable Counts**: Displays count of env vars per service
- **Communication Flow**: Simple visualization of typical service flow
- **Interactive Cards**: Hover effects and clickable service URLs

**Use Cases:**
- Understand service architecture at a glance
- Identify services by technology stack
- Visualize service relationships
- Plan microservice dependencies

### 5. Keyboard Shortcuts

**Access:** Press `?` to open, `Esc` to close

Keyboard shortcuts for efficient dashboard navigation:

**Navigation:**
- `1` - Resources view
- `2` - Console view
- `3` - Metrics view
- `4` - Environment view
- `5` - Actions view
- `6` - Dependencies view

**Actions:**
- `R` - Refresh all services
- `C` - Clear console logs
- `E` - Export logs
- `/` or `Ctrl+F` - Focus search

**Views:**
- `T` - Toggle table/grid view (in Resources)
- `?` - Show keyboard shortcuts
- `Esc` - Close dialogs

### 6. Enhanced Resources View

The existing Resources view has been enhanced with:

- Persistent view mode preference (table vs grid)
- Improved search and filter controls
- Smooth transitions between views
- Better loading and error states

## Design Philosophy

The dashboard enhancements follow these principles:

1. **Developer-First**: Built for the needs of developers working with local services
2. **Minimal Overhead**: Lightweight components that don't slow down development
3. **Intuitive Navigation**: Clear visual hierarchy and keyboard shortcuts
4. **Real-Time Updates**: Live status updates via WebSocket connections
5. **Professional Design**: Modern UI with smooth animations and transitions
6. **Accessibility**: Keyboard navigation and clear visual indicators

## Comparison with Other Platforms

These enhancements bring azd-app dashboard closer to feature parity with:

- **Docker Desktop**: Service management, resource monitoring
- **Kubernetes Dashboard**: Service grouping, health indicators
- **Railway**: Environment variable management, quick actions
- **Vercel**: Performance metrics, clean UI
- **.NET Aspire**: Service dependencies, metrics display

## Future Enhancements

Potential future additions:

- Request tracing and HTTP metrics
- CPU/Memory usage graphs (requires backend integration)
- Log export with filtering
- Service restart capabilities
- Custom themes (dark/light mode toggle)
- Advanced dependency graph visualization
- Integration with Azure services metrics
- Real-time performance charts
- Alert notifications for service issues

## Testing

All new components include comprehensive test coverage:

- EnvironmentPanel: 6 tests
- QuickActions: 8 tests
- PerformanceMetrics: 10 tests
- ServiceDependencies: Covered by integration tests
- KeyboardShortcuts: Covered by integration tests

Total test suite: 206 tests (205 passing, 1 skipped)

## Implementation Notes

- **Zero Backend Changes**: All enhancements work with existing API endpoints
- **Backward Compatible**: No breaking changes to existing components
- **Minimal Dependencies**: Uses existing UI component library
- **Performance**: Optimized rendering with React hooks and memoization
- **Code Quality**: Follows existing patterns, TypeScript strict mode, ESLint compliant
