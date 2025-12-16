/**
 * Service utilities - backward compatibility layer
 * 
 * This file re-exports all utilities from the split modules for backward compatibility.
 * New code should import directly from the specific modules:
 * - service-status.ts - Status/lifecycle utilities
 * - service-health.ts - Health check utilities
 * - service-formatters.ts - Formatting functions
 * - service-display.ts - Display configuration
 */

// Re-export from service-status.ts
export {
  normalizeHealthStatus,
  normalizeLifecycleState,
  calculateStatusCounts,
  getEffectiveStatus,
  getServiceDisplayStatus,
  isServiceHealthy,
  type StatusCounts,
  type EffectiveDisplayStatus,
  type OperationState,
} from './service-status'

// Re-export from service-status-display.ts
export {
  getStatusDisplay,
  getLogPaneVisualStatus,
  type StatusDisplay,
  type VisualStatus,
} from './service-status-display'

// Re-export from service-health.ts
export {
  getCheckTypeDisplay,
  mergeHealthIntoService,
} from './service-health'

// Re-export from service-formatters.ts
export {
  formatRelativeTime,
  formatStartTime,
  formatLogTimestamp,
  formatResponseTime,
  formatUptime,
} from './service-formatters'

// Re-export from service-display.ts
export {
  getStatusIndicator,
  getOverallStatusIndicator,
  getStatusBadgeConfig,
  getHealthBadgeConfig,
  getServiceTypeBadgeConfig,
  getServiceModeBadgeConfig,
  getServiceTypeLabel,
  getServiceModeLabel,
  isProcessService,
  isContainerService,
  isContinuousMode,
  isOneTimeMode,
  type StatusIndicator,
  type StatusBadgeConfig,
  type HealthBadgeConfig,
  type ServiceTypeBadgeConfig,
  type ServiceModeBadgeConfig,
} from './service-display'
