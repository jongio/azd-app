/**
 * Dashboard Components
 * 
 * Export all components for the dashboard.
 * Follow design specs in: cli/dashboard/design/
 */

// Main App Shell
export { App } from './App'
export type { AppProps } from './App'

// Header & Navigation
export { Header } from './Header'
export type { HeaderProps, View } from './Header'

// Service Status Card (header status summary)
export { ServiceStatusCard } from './ServiceStatusCard'
export type { ServiceStatusCardProps } from './ServiceStatusCard'

// Service Views
export { ServiceCard } from './ServiceCard'
export type { ServiceCardProps } from './ServiceCard'

export { ServiceTable } from './ServiceTable'
export type { ServiceTableProps } from './ServiceTable'

// Status Indicators
export {
  StatusDot,
  StatusBadge,
  StatusIndicator,
  DualStatusBadge,
  HealthPill,
  ConnectionStatus,
  StatusSkeleton,
  Spinner,
} from './StatusIndicator'
export type { EffectiveStatus } from './StatusIndicator'

// Console View
export { ConsoleView } from './ConsoleView'
export type { ConsoleViewProps } from './ConsoleView'

// Detail Panel
export { ServiceDetailPanel } from './ServiceDetailPanel'
export type { ServiceDetailPanelProps } from './ServiceDetailPanel'

// Theme Toggle
export { ThemeToggle } from './ThemeToggle'
export type { ThemeToggleProps } from './ThemeToggle'

// Settings Dialog
export { SettingsDialog } from './SettingsDialog'
export type { SettingsDialogProps } from './SettingsDialog'

// Environment Panel
export { EnvironmentPanel } from './EnvironmentPanel'
export type { EnvironmentPanelProps } from './EnvironmentPanel'

// Shared Components
export { LogsPane } from './LogsPane'
export { LogsPaneGrid } from './LogsPaneGrid'
export { LogsView } from './LogsView'
export { ErrorBoundary } from './ErrorBoundary'

// Service Actions
export { ServiceActions } from './ServiceActions'

// Classifications
export { ClassificationsEditor } from './ClassificationsManager'
export type { ClassificationChange } from './ClassificationsManager'

// Notifications
export { NotificationBadge } from './NotificationBadge'
export { NotificationCenter } from './NotificationCenter'
export type { NotificationHistoryItem } from './NotificationCenter'
export { NotificationStack } from './NotificationStack'
export type { Notification } from './NotificationStack'
export { NotificationToast } from './NotificationToast'
