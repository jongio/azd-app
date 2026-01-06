/**
 * Centralized log utility functions to eliminate duplication across components.
 * This module provides consistent log level detection, ANSI conversion, and constants.
 */
import AnsiConverter from 'ansi-to-html'
import type { CodespaceConfig } from '@/lib/codespace-utils'
import { isLocalhostUrl, transformLocalhostUrl } from '@/lib/codespace-utils'
import { LOG_CONSTANTS } from '@/lib/constants'

// ============================================================================
// Constants
// ============================================================================

/** Log level constants to replace magic numbers across the codebase */
export const LOG_LEVELS = {
  INFO: 1,
  WARNING: 2,
  ERROR: 3,
} as const

export type LogLevel = (typeof LOG_LEVELS)[keyof typeof LOG_LEVELS]
export type LogLevelName = 'info' | 'warning' | 'error'

/** Maximum number of logs to keep in memory per pane */
export const MAX_LOGS_IN_MEMORY = LOG_CONSTANTS.MAX_LOGS_IN_MEMORY

/** Initial number of logs to fetch when a component mounts */
export const INITIAL_LOG_TAIL = LOG_CONSTANTS.INITIAL_LOG_TAIL

/** Threshold in pixels for detecting scroll position */
export const SCROLL_THRESHOLD_PX = LOG_CONSTANTS.SCROLL_THRESHOLD_PX

// ============================================================================
// ANSI Conversion (Dual Instances for Light/Dark Mode)
// ============================================================================

const ansiConverterDark = new AnsiConverter({
  fg: '#e2e8f0', // Light text for dark mode
  bg: '#111827',
  newline: false,
  escapeXML: true, // CRITICAL: Must be true to prevent XSS
  stream: false,
})

const ansiConverterLight = new AnsiConverter({
  fg: '#1e293b', // Dark text for light mode
  bg: '#ffffff',
  newline: false,
  escapeXML: true, // CRITICAL: Must be true to prevent XSS
  stream: false,
})

/**
 * Pattern to match ANSI escape sequences for stripping.
 */
// eslint-disable-next-line no-control-regex
const ANSI_PATTERN = /\x1b\[[0-9;]*m|\x1b\][^\x07]*\x07|\x1b\][^\x1b]*\x1b\\/g

/**
 * Strips ANSI escape codes from text.
 */
function stripAnsi(text: string): string {
  return text.replaceAll(ANSI_PATTERN, '')
}

/**
 * Converts ANSI escape codes to HTML for display.
 * Includes XSS sanitization for security and URL linkification.
 * Automatically detects light/dark theme for appropriate text colors.
 * 
 * URL detection is done on the stripped text first to handle cases where
 * ANSI codes might be embedded within URLs (e.g., colored port numbers).
 * 
 * @param text - The text with ANSI codes to convert
 * @param codespaceConfig - Optional Codespace config to transform localhost URLs
 */
export function convertAnsiToHtml(text: string, codespaceConfig?: CodespaceConfig | null): string {
  try {
    // First, find URLs in the stripped text (without ANSI codes)
    const strippedText = stripAnsi(text)
    const urls = findUrls(strippedText)
    
    // Detect current theme from document
    const isDark = typeof document !== 'undefined' && document.documentElement.classList.contains('dark')
    const converter = isDark ? ansiConverterDark : ansiConverterLight
    
    // Convert ANSI to HTML
    const html = converter.toHtml(text)
    const sanitized = sanitizeHtml(html)
    
    // Linkify URLs, handling potential HTML tags within URLs
    // Pass codespace config for localhost URL transformation
    return linkifyUrlsWithHtmlAware(sanitized, urls, codespaceConfig)
  } catch {
    // If conversion fails, escape the text for safe display
    return escapeHtml(text)
  }
}

/**
 * Pattern to match URLs (http/https) in text.
 * Matches URLs while avoiding common punctuation at the end.
 */
const URL_PATTERN = /https?:\/\/[^\s<>"'`]+[^\s<>"'`.,;:!?\])]/g

/**
 * Finds all URLs in text and returns them.
 */
function findUrls(text: string): string[] {
  const matches = text.match(URL_PATTERN)
  return matches ? [...new Set(matches)] : [] // Dedupe URLs
}

/**
 * Converts URLs in HTML to clickable anchor tags, handling cases where
 * HTML tags (from ANSI conversion) might be embedded within URLs.
 * 
 * When codespaceConfig is provided, localhost URLs are transformed to
 * Codespace-forwarded URLs in the href attribute while keeping the
 * original display text.
 */
function linkifyUrlsWithHtmlAware(
  html: string,
  urls: string[],
  codespaceConfig?: CodespaceConfig | null
): string {
  if (urls.length === 0) return html
  
  let result = html
  
  for (const url of urls) {
    // Create a pattern that matches the URL with potential HTML tags interspersed
    // and HTML entity encoding (& becomes &amp;)
    const urlChars = url.split('')
    const flexiblePattern = urlChars
      .map((char) => {
        // Handle HTML entity encoding
        if (char === '&') {
          return '(?:&amp;|&)'
        }
        // Escape special regex chars
        const escaped = char.replaceAll(/[.*+?^${}()|[\]\\]/g, String.raw`\$&`)
        // Allow optional HTML tags between characters
        return escaped + '(?:<[^>]*>)*'
      })
      .join('')
    
    // Remove the trailing tag matcher from the last character
    const pattern = new RegExp(
      flexiblePattern.replace(/\(\?:<\[\^>\]\*>\)\*$/, ''),
      'g'
    )
    
    // Determine the href URL - transform if localhost and in Codespace
    const hrefUrl = codespaceConfig && isLocalhostUrl(url)
      ? transformLocalhostUrl(url, codespaceConfig)
      : url
    
    result = result.replace(pattern, (match) => {
      // Don't double-wrap if already linkified
      if (match.includes('<a ')) return match
      return `<a href="${hrefUrl}" target="_blank" rel="noopener noreferrer" class="text-cyan-400 hover:text-cyan-300 hover:underline">${match}</a>`
    })
  }
  
  return result
}

/**
 * Sanitizes HTML to prevent XSS attacks.
 * Removes script tags and javascript: protocols.
 */
function sanitizeHtml(html: string): string {
  return html
    .replaceAll(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '')
    .replaceAll(/javascript:/gi, '')
    .replaceAll(/on\w+=/gi, '')
}

/**
 * Escapes HTML special characters for safe display.
 */
function escapeHtml(text: string): string {
  return text
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
}

// ============================================================================
// Log Level Detection (Centralized Pattern Matching)
// ============================================================================

/** Pattern for detecting error-level messages */
const ERROR_PATTERN = /\b(error|failed|failure|exception|fatal|panic|critical|crash|died)\b/i

/** Pattern for detecting warning-level messages */
const WARNING_PATTERN = /\b(warn|warning|caution|deprecated)\b/i

/** Patterns for messages that look like errors/warnings but are actually informational */
const INFORMATIONAL_PATTERNS = [
  /Debug mode:/i,
  /Development mode:/i,
  /WARNING: This is a development server/i,
  /Development mode is enabled/i,
  /Serving Flask app/i,
  /Running on (http|all addresses)/i,
  /Press CTRL\+C to quit/i,
  /Found 0 errors/i,                    // TypeScript compiler success
  /Watching for file changes/i,         // TypeScript watch mode
  /Starting compilation/i,              // TypeScript compiler starting
  /Compilation complete/i,              // TypeScript compiler success
]

/**
 * Checks if a log message contains error-level content.
 * Excludes false positives from informational messages.
 */
export function isErrorLine(message: string): boolean {
  // Exclude common informational messages that contain error-like keywords
  if (INFORMATIONAL_PATTERNS.some(pattern => pattern.test(message))) {
    return false
  }
  return ERROR_PATTERN.test(message)
}

/**
 * Checks if a log message contains warning-level content.
 * Excludes false positives from informational messages.
 */
export function isWarningLine(message: string): boolean {
  // Exclude common informational messages
  if (INFORMATIONAL_PATTERNS.some(pattern => pattern.test(message))) {
    return false
  }
  return WARNING_PATTERN.test(message)
}

/**
 * Determines the log level from a log entry.
 * Considers both the numeric level and message content.
 */
export function getLogLevel(
  message: string,
  numericLevel?: number,
  isStderr?: boolean
): LogLevelName {
  // Check stderr and explicit error level first
  if (isStderr || numericLevel === LOG_LEVELS.ERROR) return 'error'
  
  // Check message content for errors
  if (isErrorLine(message)) return 'error'
  
  // Check explicit warning level
  if (numericLevel === LOG_LEVELS.WARNING) return 'warning'
  
  // Check message content for warnings
  if (isWarningLine(message)) return 'warning'
  
  return 'info'
}

// ============================================================================
// Service Color Assignment
// ============================================================================

/** Color palette for service names (avoiding red which is reserved for errors) */
const SERVICE_COLORS = [
  'text-blue-400',
  'text-green-400',
  'text-purple-400',
  'text-cyan-400',
  'text-pink-400',
  'text-amber-400',
  'text-teal-400',
  'text-indigo-400',
  'text-lime-400',
  'text-fuchsia-400',
  'text-sky-400',
  'text-violet-400',
] as const satisfies readonly string[]

/**
 * Gets a consistent color class for a service name.
 * Uses hash-based selection for deterministic colors.
 */
export function getServiceColor(serviceName: string): string {
  let hash = 0
	for (const char of serviceName) {
		hash += char.codePointAt(0) ?? 0
	}
  return SERVICE_COLORS[hash % SERVICE_COLORS.length]
}

/**
 * Gets the appropriate text color class for a log entry based on its level.
 */
export function getLogColor(level: LogLevelName): string {
  switch (level) {
    case 'error':
      return 'text-red-400'
    case 'warning':
      return 'text-yellow-400'
    case 'info':
    default:
      return 'text-foreground-tertiary'
  }
}

// ============================================================================
// Timestamp Deduplication
// ============================================================================

/**
 * Pattern to match common embedded timestamp formats at the start of log messages:
 * - [2025-12-13 05:45:49] - bracketed date time
 * - [2025-12-13T05:45:49.123Z] - ISO format
 * - 2025-12-13 05:45:49 - plain date time
 * - [05:45:49] - bracketed time only
 * - [serviceName] - service name prefix (when already displayed in header)
 * 
 * These are commonly added by application loggers but we already display
 * the timestamp from the structured log entry, so they're redundant.
 */
const EMBEDDED_TIMESTAMP_PATTERNS = [
  // ISO 8601 with brackets: [2025-12-13T05:45:49.1071934-08:00] or [2025-12-13T05:45:49Z]
  /^\s*\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[^\]]*\]\s*/,
  // Date time with brackets: [2025-12-13 05:45:49]
  /^\s*\[\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}\]\s*/,
  // Time only with brackets at start: [05:45:49] or [08:20:50.670]
  /^\s*\[\d{2}:\d{2}:\d{2}(?:\.\d+)?\]\s*/,
]

/**
 * Pattern to match service name prefix in messages: [appservice-web]
 * These are redundant when we already show the service name in the log header.
 * Must contain lowercase letters and be at least 3 chars. Does NOT match [INFO], [WARN], etc.
 */
const SERVICE_NAME_PREFIX_PATTERN = /^\s*\[([a-z][a-z0-9_-]{2,})\]\s*/

/**
 * Known log level prefixes that should NOT be stripped (case insensitive check)
 */
const KNOWN_LOG_LEVELS = new Set(['info', 'warn', 'warning', 'error', 'debug', 'trace', 'fatal', 'critical', 'verbose'])

/**
 * Strips embedded timestamps and redundant service names from log messages.
 * This cleans up messages that already contain timestamp/source prefixes
 * when we're displaying that information in the structured log header.
 * 
 * @param message - The raw log message
 * @param stripServiceName - If true, also strips [serviceName] prefix
 * @returns The cleaned message without redundant prefixes
 */
export function stripEmbeddedTimestamp(message: string, stripServiceName = true): string {
  const stripOneTimestampPrefix = (input: string): string => {
    for (const pattern of EMBEDDED_TIMESTAMP_PATTERNS) {
      if (pattern.test(input)) {
        return input.replace(pattern, '')
      }
    }
    return input
  }

  const stripOneServicePrefix = (input: string): string => {
    const match = SERVICE_NAME_PREFIX_PATTERN.exec(input)
    if (!match) return input

    const captured = match[1].toLowerCase()
    if (KNOWN_LOG_LEVELS.has(captured)) return input
    return input.replace(SERVICE_NAME_PREFIX_PATTERN, '')
  }

  let result = message

  // Strip embedded timestamps and service names in multiple passes
  // to handle nested patterns like: [timestamp] [service] [timestamp] [level] message
  for (let i = 0; i < 5; i++) {
    const before = result
    result = stripOneTimestampPrefix(result)
    if (stripServiceName) {
      result = stripOneServicePrefix(result)
    }
    if (result === before) break
  }

  return result.trim()
}
