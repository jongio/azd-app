/**
 * Codespace URL utilities for transforming localhost URLs to Codespace-forwarded URLs.
 * 
 * When running in GitHub Codespaces, localhost URLs need to be transformed to the
 * Codespace port forwarding domain for external access.
 * 
 * Example: http://localhost:3000 -> https://silver-space-xyzzy-3000.app.github.dev
 */

// =============================================================================
// Types
// =============================================================================

export interface CodespaceConfig {
  enabled: boolean
  name: string
  domain: string
}

export interface EnvironmentInfo {
  codespace: CodespaceConfig
}

// =============================================================================
// Constants
// =============================================================================

/**
 * Patterns that match localhost URLs.
 * Supports: localhost, 127.0.0.1, 0.0.0.0, [::1], [::]
 */
const LOCALHOST_PATTERNS = [
  /^https?:\/\/localhost(:\d+)?/i,
  /^https?:\/\/127\.0\.0\.1(:\d+)?/i,
  /^https?:\/\/0\.0\.0\.0(:\d+)?/i,
  /^https?:\/\/\[::1\](:\d+)?/i,
  /^https?:\/\/\[::\](:\d+)?/i,
]

/**
 * Pattern to extract port from localhost URLs.
 */
const PORT_PATTERN = /^https?:\/\/(?:localhost|127\.0\.0\.1|0\.0\.0\.0|\[::1?\]|\[::\]):(\d+)(\/.*)?$/i

/**
 * Pattern for URLs without explicit port (default to 80/443).
 */
const NO_PORT_PATTERN = /^(https?):\/\/(?:localhost|127\.0\.0\.1|0\.0\.0\.0|\[::1?\]|\[::\])(\/.*)?$/i

// =============================================================================
// URL Detection
// =============================================================================

/**
 * Checks if a URL is a localhost URL that should be transformed.
 */
export function isLocalhostUrl(url: string): boolean {
  return LOCALHOST_PATTERNS.some(pattern => pattern.test(url))
}

/**
 * Extracts the port number from a localhost URL.
 * Returns null if no port is found or port is invalid.
 */
export function extractPort(url: string): number | null {
  const match = url.match(PORT_PATTERN)
  if (match && match[1]) {
    const port = parseInt(match[1], 10)
    // Port 0 is invalid for URL transformation
    if (port > 0 && port <= 65535) {
      return port
    }
  }
  
  // Check for default ports (http=80, https=443)
  const noPortMatch = url.match(NO_PORT_PATTERN)
  if (noPortMatch) {
    return noPortMatch[1] === 'https' ? 443 : 80
  }
  
  return null
}

/**
 * Extracts the path and query string from a URL.
 */
export function extractPathAndQuery(url: string): string {
  try {
    const urlObj = new URL(url)
    return urlObj.pathname + urlObj.search + urlObj.hash
  } catch {
    // Fallback: extract everything after host:port
    const match = url.match(/^https?:\/\/[^/]+(\/.*)?$/)
    return match?.[1] || ''
  }
}

// =============================================================================
// URL Transformation
// =============================================================================

/**
 * Transforms a localhost URL to a Codespace-forwarded URL.
 * 
 * @param url - The URL to transform (e.g., http://localhost:3000/api)
 * @param config - Codespace configuration with name and domain
 * @returns Transformed URL or original if not applicable
 * 
 * @example
 * // In Codespace "silver-space-xyzzy":
 * transformLocalhostUrl('http://localhost:3000/api', config)
 * // Returns: 'https://silver-space-xyzzy-3000.app.github.dev/api'
 */
export function transformLocalhostUrl(url: string, config: CodespaceConfig | null): string {
  // Return original if not in Codespace or config is missing
  if (!config?.enabled || !config.name || !config.domain) {
    return url
  }

  // Only transform localhost URLs
  if (!isLocalhostUrl(url)) {
    return url
  }

  const port = extractPort(url)
  // Can't transform URLs without a valid port
  if (port === null) {
    return url
  }

  const pathAndQuery = extractPathAndQuery(url)
  
  // Build Codespace URL: https://{name}-{port}.{domain}{path}
  return `https://${config.name}-${port}.${config.domain}${pathAndQuery}`
}

/**
 * Builds a Codespace URL directly from a port number.
 * 
 * @param port - The port number
 * @param config - Codespace configuration
 * @param path - Optional path to append
 * @returns Codespace URL or null if not in Codespace
 */
export function buildCodespaceUrl(
  port: number,
  config: CodespaceConfig | null,
  path: string = ''
): string | null {
  if (!config?.enabled || !config.name || !config.domain) {
    return null
  }

  if (port <= 0 || port > 65535) {
    return null
  }

  return `https://${config.name}-${port}.${config.domain}${path}`
}

/**
 * Gets the appropriate service URL, transforming if in Codespace.
 * 
 * @param localUrl - The local URL (e.g., http://localhost:3000)
 * @param port - The service port (fallback if URL is missing)
 * @param config - Codespace configuration
 * @returns The appropriate URL for the current environment
 */
export function getEffectiveServiceUrl(
  localUrl: string | null | undefined,
  port: number | null | undefined,
  config: CodespaceConfig | null
): string | null {
  // If we have a URL, try to transform it
  if (localUrl) {
    // Skip invalid URLs (port 0)
    if (localUrl.match(/:0\/?$/)) {
      return null
    }
    return transformLocalhostUrl(localUrl, config)
  }

  // Fallback: build URL from port
  if (port && port > 0) {
    if (config?.enabled) {
      return buildCodespaceUrl(port, config)
    }
    return `http://localhost:${port}`
  }

  return null
}
