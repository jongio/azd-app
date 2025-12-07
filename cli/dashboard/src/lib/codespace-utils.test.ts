/**
 * Tests for codespace-utils.ts
 */
import { describe, it, expect } from 'vitest'
import {
  isLocalhostUrl,
  extractPort,
  extractPathAndQuery,
  transformLocalhostUrl,
  buildCodespaceUrl,
  getEffectiveServiceUrl,
  type CodespaceConfig,
} from './codespace-utils'

// =============================================================================
// Test Fixtures
// =============================================================================

const codespaceConfig: CodespaceConfig = {
  enabled: true,
  name: 'silver-space-xyzzy',
  domain: 'app.github.dev',
}

const disabledConfig: CodespaceConfig = {
  enabled: false,
  name: '',
  domain: '',
}

// =============================================================================
// isLocalhostUrl Tests
// =============================================================================

describe('isLocalhostUrl', () => {
  it('detects localhost URLs', () => {
    expect(isLocalhostUrl('http://localhost:3000')).toBe(true)
    expect(isLocalhostUrl('https://localhost:3000')).toBe(true)
    expect(isLocalhostUrl('http://localhost')).toBe(true)
    expect(isLocalhostUrl('http://localhost/')).toBe(true)
  })

  it('detects 127.0.0.1 URLs', () => {
    expect(isLocalhostUrl('http://127.0.0.1:3000')).toBe(true)
    expect(isLocalhostUrl('https://127.0.0.1:8080')).toBe(true)
    expect(isLocalhostUrl('http://127.0.0.1')).toBe(true)
  })

  it('detects 0.0.0.0 URLs', () => {
    expect(isLocalhostUrl('http://0.0.0.0:3000')).toBe(true)
    expect(isLocalhostUrl('https://0.0.0.0:8080')).toBe(true)
  })

  it('detects IPv6 localhost URLs', () => {
    expect(isLocalhostUrl('http://[::1]:3000')).toBe(true)
    expect(isLocalhostUrl('http://[::]:3000')).toBe(true)
  })

  it('returns false for non-localhost URLs', () => {
    expect(isLocalhostUrl('http://example.com')).toBe(false)
    expect(isLocalhostUrl('https://api.github.com')).toBe(false)
    expect(isLocalhostUrl('http://192.168.1.1:3000')).toBe(false)
  })

  it('is case-insensitive', () => {
    expect(isLocalhostUrl('HTTP://LOCALHOST:3000')).toBe(true)
    expect(isLocalhostUrl('Http://LocalHost:8080')).toBe(true)
  })
})

// =============================================================================
// extractPort Tests
// =============================================================================

describe('extractPort', () => {
  it('extracts port from localhost URLs', () => {
    expect(extractPort('http://localhost:3000')).toBe(3000)
    expect(extractPort('http://localhost:8080')).toBe(8080)
    expect(extractPort('http://localhost:5173/path')).toBe(5173)
  })

  it('extracts port from 127.0.0.1 URLs', () => {
    expect(extractPort('http://127.0.0.1:3000')).toBe(3000)
    expect(extractPort('https://127.0.0.1:443')).toBe(443)
  })

  it('extracts port from 0.0.0.0 URLs', () => {
    expect(extractPort('http://0.0.0.0:8000')).toBe(8000)
  })

  it('extracts port from IPv6 localhost URLs', () => {
    expect(extractPort('http://[::1]:3000')).toBe(3000)
    expect(extractPort('http://[::]:5000')).toBe(5000)
  })

  it('returns default port for URLs without explicit port', () => {
    expect(extractPort('http://localhost')).toBe(80)
    expect(extractPort('https://localhost')).toBe(443)
    expect(extractPort('http://localhost/')).toBe(80)
  })

  it('returns null for port 0', () => {
    expect(extractPort('http://localhost:0')).toBe(null)
  })

  it('returns null for invalid ports', () => {
    expect(extractPort('http://localhost:99999')).toBe(null)
  })
})

// =============================================================================
// extractPathAndQuery Tests
// =============================================================================

describe('extractPathAndQuery', () => {
  it('extracts path from URL', () => {
    expect(extractPathAndQuery('http://localhost:3000/api')).toBe('/api')
    expect(extractPathAndQuery('http://localhost:3000/api/users')).toBe('/api/users')
  })

  it('extracts path with query string', () => {
    expect(extractPathAndQuery('http://localhost:3000/api?key=value')).toBe('/api?key=value')
    expect(extractPathAndQuery('http://localhost:3000/api?a=1&b=2')).toBe('/api?a=1&b=2')
  })

  it('extracts path with hash', () => {
    expect(extractPathAndQuery('http://localhost:3000/page#section')).toBe('/page#section')
  })

  it('returns empty string for URLs without path', () => {
    expect(extractPathAndQuery('http://localhost:3000')).toBe('/')
  })

  it('handles complex paths', () => {
    expect(extractPathAndQuery('http://localhost:3000/api/v2/users?page=1#top')).toBe('/api/v2/users?page=1#top')
  })
})

// =============================================================================
// transformLocalhostUrl Tests
// =============================================================================

describe('transformLocalhostUrl', () => {
  it('transforms localhost URL to Codespace URL', () => {
    expect(transformLocalhostUrl('http://localhost:3000', codespaceConfig))
      .toBe('https://silver-space-xyzzy-3000.app.github.dev/')
  })

  it('transforms 127.0.0.1 URL to Codespace URL', () => {
    expect(transformLocalhostUrl('http://127.0.0.1:8080', codespaceConfig))
      .toBe('https://silver-space-xyzzy-8080.app.github.dev/')
  })

  it('transforms 0.0.0.0 URL to Codespace URL', () => {
    expect(transformLocalhostUrl('http://0.0.0.0:5000', codespaceConfig))
      .toBe('https://silver-space-xyzzy-5000.app.github.dev/')
  })

  it('preserves path in transformed URL', () => {
    expect(transformLocalhostUrl('http://localhost:3000/api/health', codespaceConfig))
      .toBe('https://silver-space-xyzzy-3000.app.github.dev/api/health')
  })

  it('preserves query string in transformed URL', () => {
    expect(transformLocalhostUrl('http://localhost:3000/api?key=value', codespaceConfig))
      .toBe('https://silver-space-xyzzy-3000.app.github.dev/api?key=value')
  })

  it('preserves hash in transformed URL', () => {
    expect(transformLocalhostUrl('http://localhost:3000/page#section', codespaceConfig))
      .toBe('https://silver-space-xyzzy-3000.app.github.dev/page#section')
  })

  it('returns original URL when not in Codespace', () => {
    expect(transformLocalhostUrl('http://localhost:3000', disabledConfig))
      .toBe('http://localhost:3000')
    expect(transformLocalhostUrl('http://localhost:3000', null))
      .toBe('http://localhost:3000')
  })

  it('returns original URL for non-localhost URLs', () => {
    expect(transformLocalhostUrl('http://example.com', codespaceConfig))
      .toBe('http://example.com')
    expect(transformLocalhostUrl('https://api.github.com', codespaceConfig))
      .toBe('https://api.github.com')
  })

  it('returns original URL for localhost without port', () => {
    // URLs without explicit port get default port (80/443)
    expect(transformLocalhostUrl('http://localhost', codespaceConfig))
      .toBe('https://silver-space-xyzzy-80.app.github.dev/')
  })
})

// =============================================================================
// buildCodespaceUrl Tests
// =============================================================================

describe('buildCodespaceUrl', () => {
  it('builds Codespace URL from port', () => {
    expect(buildCodespaceUrl(3000, codespaceConfig))
      .toBe('https://silver-space-xyzzy-3000.app.github.dev')
  })

  it('builds Codespace URL with path', () => {
    expect(buildCodespaceUrl(3000, codespaceConfig, '/api'))
      .toBe('https://silver-space-xyzzy-3000.app.github.dev/api')
  })

  it('returns null when not in Codespace', () => {
    expect(buildCodespaceUrl(3000, disabledConfig)).toBe(null)
    expect(buildCodespaceUrl(3000, null)).toBe(null)
  })

  it('returns null for invalid ports', () => {
    expect(buildCodespaceUrl(0, codespaceConfig)).toBe(null)
    expect(buildCodespaceUrl(-1, codespaceConfig)).toBe(null)
    expect(buildCodespaceUrl(99999, codespaceConfig)).toBe(null)
  })
})

// =============================================================================
// getEffectiveServiceUrl Tests
// =============================================================================

describe('getEffectiveServiceUrl', () => {
  it('transforms localhost URL when in Codespace', () => {
    expect(getEffectiveServiceUrl('http://localhost:3000', 3000, codespaceConfig))
      .toBe('https://silver-space-xyzzy-3000.app.github.dev/')
  })

  it('returns localhost URL when not in Codespace', () => {
    expect(getEffectiveServiceUrl('http://localhost:3000', 3000, disabledConfig))
      .toBe('http://localhost:3000')
    expect(getEffectiveServiceUrl('http://localhost:3000', 3000, null))
      .toBe('http://localhost:3000')
  })

  it('builds URL from port when URL is missing', () => {
    expect(getEffectiveServiceUrl(null, 3000, codespaceConfig))
      .toBe('https://silver-space-xyzzy-3000.app.github.dev')
    expect(getEffectiveServiceUrl(undefined, 3000, disabledConfig))
      .toBe('http://localhost:3000')
  })

  it('returns null for invalid port 0 URL', () => {
    expect(getEffectiveServiceUrl('http://localhost:0', 0, codespaceConfig)).toBe(null)
  })

  it('returns null when no URL and no port', () => {
    expect(getEffectiveServiceUrl(null, null, codespaceConfig)).toBe(null)
    expect(getEffectiveServiceUrl(null, 0, codespaceConfig)).toBe(null)
  })

  it('prefers URL over port when both provided', () => {
    expect(getEffectiveServiceUrl('http://localhost:3000/api', 8080, codespaceConfig))
      .toBe('https://silver-space-xyzzy-3000.app.github.dev/api')
  })
})
