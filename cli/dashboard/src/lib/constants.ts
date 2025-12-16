/**
 * Centralized constants for the dashboard application.
 * This module consolidates all magic numbers to improve maintainability and consistency.
 */

// =============================================================================
// Log Management Constants
// =============================================================================

/**
 * Constants for log buffer management and display.
 */
export const LOG_CONSTANTS = {
  /**
   * Maximum number of logs to keep in memory per pane.
   * Prevents memory issues with long-running services.
   */
  MAX_LOGS_IN_MEMORY: 1000,
  
  /**
   * Initial number of logs to fetch when a component mounts.
   * Provides sufficient context without overwhelming the UI.
   */
  INITIAL_LOG_TAIL: 500,
  
  /**
   * Threshold in pixels for detecting scroll position.
   * Used to determine if user is at bottom of log view.
   */
  SCROLL_THRESHOLD_PX: 10,
} as const

// =============================================================================
// WebSocket Connection Constants
// =============================================================================

/**
 * Constants for WebSocket connection management and reconnection logic.
 */
export const WEBSOCKET_CONSTANTS = {
  /**
   * Initial delay in milliseconds before first WebSocket reconnection attempt.
   * Uses exponential backoff from this base value.
   */
  WS_INITIAL_RETRY_DELAY_MS: 1000,
  
  /**
   * Maximum delay in milliseconds between WebSocket reconnection attempts.
   * Caps the exponential backoff to prevent excessive wait times.
   */
  WS_MAX_RETRY_DELAY_MS: 30000,
  
  /**
   * Maximum number of WebSocket reconnection attempts.
   * After this limit, the connection is considered failed.
   */
  WS_MAX_RETRIES: 10,
  
  /**
   * Default delay in milliseconds before SSE health stream reconnection.
   * Used after connection loss to avoid overwhelming the server.
   */
  DEFAULT_RECONNECT_DELAY: 3000,
  
  /**
   * Maximum number of SSE health stream reconnection attempts.
   * After this limit, user must manually refresh.
   */
  MAX_RECONNECT_ATTEMPTS: 5,
} as const

// =============================================================================
// UI Interaction Constants
// =============================================================================

/**
 * Constants for UI timing, animations, and user feedback.
 */
export const UI_CONSTANTS = {
  /**
   * Minimum sync interval in milliseconds for Azure log polling.
   * Prevents excessive API requests.
   */
  MIN_SYNC_INTERVAL: 5000,
  
  /**
   * Maximum sync interval in milliseconds for Azure log polling.
   * Ensures logs are eventually refreshed even with long intervals.
   */
  MAX_SYNC_INTERVAL: 300000,
  
  /**
   * Duration in milliseconds to show copy feedback message.
   * Long enough to be noticeable but not obtrusive.
   */
  COPY_FEEDBACK_DURATION_MS: 1500,
  
  /**
   * Minimum height in pixels for log pane content area.
   * Ensures usability when pane is expanded.
   */
  MIN_PANE_HEIGHT_PX: 150,
  
  /**
   * Minimum width in pixels for a log pane in grid layout.
   * Prevents panes from becoming too narrow to read.
   */
  MIN_PANE_WIDTH: 250,
  
  /**
   * Minimum height in pixels for a log pane in grid layout.
   * Ensures enough space for log content and controls.
   */
  MIN_PANE_HEIGHT: 120,
  
  /**
   * Gap between grid items in pixels.
   * Provides visual separation between log panes.
   */
  GRID_GAP: 16,
  
  /**
   * Padding on the grid container in pixels.
   * Creates space around the edge of the log pane grid.
   */
  GRID_PADDING: 16,
  
  /**
   * Height in pixels of a collapsed pane header.
   * Used for grid layout calculations.
   */
  COLLAPSED_PANE_HEIGHT: 48,
} as const

// =============================================================================
// Health Check Constants
// =============================================================================

/**
 * Constants for health check intervals and timing.
 */
export const HEALTH_CONSTANTS = {
  /**
   * Default interval in seconds between health checks.
   * Balances responsiveness with resource usage.
   */
  DEFAULT_INTERVAL: 5,
  
  /**
   * Maximum number of health changes to keep in memory.
   * Prevents unbounded memory growth with history.
   */
  MAX_CHANGES_TO_KEEP: 50,
} as const
