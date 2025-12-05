/**
 * Centralized log utility functions to eliminate duplication across components.
 * This module provides consistent log level detection, ANSI conversion, and constants.
 */
import AnsiConverter from 'ansi-to-html'

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
export const MAX_LOGS_IN_MEMORY = 1000

/** Initial number of logs to fetch when a component mounts */
export const INITIAL_LOG_TAIL = 500

/** Threshold in pixels for detecting scroll position */
export const SCROLL_THRESHOLD_PX = 10

// ============================================================================
// ANSI Conversion (Single Shared Instance)
// ============================================================================

const ansiConverter = new AnsiConverter({
  fg: '#d4d4d4',
  bg: '#0d0d0d',
  newline: false,
  escapeXML: true, // CRITICAL: Must be true to prevent XSS
  stream: false,
})

/**
 * Pattern to match ANSI escape sequences for stripping.
 */
const ANSI_PATTERN = /\x1b\[[0-9;]*m|\x1b\][^\x07]*\x07|\x1b\][^\x1b]*\x1b\\/g

/**
 * Strips ANSI escape codes from text.
 */
function stripAnsi(text: string): string {
  return text.replace(ANSI_PATTERN, '')
}

/**
 * Converts ANSI escape codes to HTML for display.
 * Includes XSS sanitization for security and URL linkification.
 * 
 * URL detection is done on the stripped text first to handle cases where
 * ANSI codes might be embedded within URLs (e.g., colored port numbers).
 */
export function convertAnsiToHtml(text: string): string {
  try {
    // First, find URLs in the stripped text (without ANSI codes)
    const strippedText = stripAnsi(text)
    const urls = findUrls(strippedText)
    
    // Convert ANSI to HTML
    const html = ansiConverter.toHtml(text)
    const sanitized = sanitizeHtml(html)
    
    // Linkify URLs, handling potential HTML tags within URLs
    return linkifyUrlsWithHtmlAware(sanitized, urls)
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
 */
function linkifyUrlsWithHtmlAware(html: string, urls: string[]): string {
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
        const escaped = char.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
        // Allow optional HTML tags between characters
        return escaped + '(?:<[^>]*>)*'
      })
      .join('')
    
    // Remove the trailing tag matcher from the last character
    const pattern = new RegExp(
      flexiblePattern.replace(/\(\?:<\[\^>\]\*>\)\*$/, ''),
      'g'
    )
    
    result = result.replace(pattern, (match) => {
      // Don't double-wrap if already linkified
      if (match.includes('<a ')) return match
      return `<a href="${url}" target="_blank" rel="noopener noreferrer" class="text-cyan-400 hover:text-cyan-300 hover:underline">${match}</a>`
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
    .replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '')
    .replace(/javascript:/gi, '')
    .replace(/on\w+=/gi, '')
}

/**
 * Escapes HTML special characters for safe display.
 */
function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
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
  const hash = serviceName.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0)
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
