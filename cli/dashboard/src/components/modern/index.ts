/**
 * Modern Mode Components
 * 
 * Export all components for the Modern design mode.
 * Follow design specs in: cli/dashboard/design/modern/
 */

// Main App Shell
export { ModernApp } from './ModernApp'
export type { ModernAppProps } from './ModernApp'

// Header & Navigation
export { ModernHeader } from './ModernHeader'
export type { ModernHeaderProps, ModernView } from './ModernHeader'

// Service Views
export { ModernServiceCard } from './ModernServiceCard'
export type { ModernServiceCardProps } from './ModernServiceCard'

export { ModernServiceTable } from './ModernServiceTable'
export type { ModernServiceTableProps } from './ModernServiceTable'

// Status Indicators
export {
  ModernStatusDot,
  ModernStatusBadge,
  ModernStatusIndicator,
  ModernHealthPill,
  ModernConnectionStatus,
  ModernSpinner,
} from './ModernStatusIndicator'
export type { EffectiveStatus } from './ModernStatusIndicator'

// Logs/Console View
export { ModernLogsView } from './ModernLogsView'
export type { ModernLogsViewProps } from './ModernLogsView'

// Detail Panel
export { ModernServiceDetailPanel } from './ModernServiceDetailPanel'
export type { ModernServiceDetailPanelProps } from './ModernServiceDetailPanel'

// Theme Toggle
export { ModernThemeToggle } from './ModernThemeToggle'
export type { ModernThemeToggleProps } from './ModernThemeToggle'

// Settings Dialog
export { ModernSettingsDialog } from './ModernSettingsDialog'
export type { ModernSettingsDialogProps } from './ModernSettingsDialog'

// Environment Panel
export { ModernEnvironmentPanel } from './ModernEnvironmentPanel'
export type { ModernEnvironmentPanelProps } from './ModernEnvironmentPanel'

// Metrics View
export { ModernMetricsView } from './ModernMetricsView'
export type { ModernMetricsViewProps } from './ModernMetricsView'
