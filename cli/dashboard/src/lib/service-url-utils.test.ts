/**
 * Tests for service-url-utils
 *
 * Covers:
 *  - isValidUrl scheme allowlist (CWE-79 / SEC-005)
 *  - getEffectiveLocalUrl precedence + scheme filtering
 *  - getEffectiveAzureUrl precedence + scheme filtering
 *  - Badge/icon/tooltip helpers
 */

import { describe, it, expect } from 'vitest'
import {
  isValidUrl,
  getEffectiveLocalUrl,
  getEffectiveAzureUrl,
  getLocalUrlBadgeConfig,
  getAzureUrlBadgeConfig,
  getLocalUrlIconColor,
  getAzureUrlIconColor,
  getLocalUrlTooltip,
  getAzureUrlTooltip,
} from './service-url-utils'
import type { LocalServiceInfo } from '@/types'

/** Helper to create a partial LocalServiceInfo for URL-focused tests */
function localInfo(overrides: Partial<LocalServiceInfo> = {}): LocalServiceInfo {
  return { status: 'running', health: 'unknown', ...overrides }
}

// =============================================================================
// isValidUrl: scheme allowlist (CWE-79 regression suite)
// =============================================================================

describe('isValidUrl', () => {
  describe('allowed schemes', () => {
    it('accepts http:// URLs', () => {
      expect(isValidUrl('http://localhost:3000')).toBe(true)
    })

    it('accepts https:// URLs', () => {
      expect(isValidUrl('https://example.com')).toBe(true)
    })

    it('accepts https:// URLs with path and query', () => {
      expect(isValidUrl('https://api.example.com/v1/status?env=prod')).toBe(true)
    })

    it('accepts http:// URLs with port', () => {
      expect(isValidUrl('http://localhost:8080')).toBe(true)
    })
  })

  describe('blocked schemes (XSS / injection vectors)', () => {
    it('rejects javascript: URLs (CWE-79)', () => {
      expect(isValidUrl('javascript:alert(1)')).toBe(false)
    })

    it('rejects javascript: URLs with document.cookie access', () => {
      expect(isValidUrl('javascript:alert(document.cookie)')).toBe(false)
    })

    it('rejects javascript: URLs with encoded colon (%3A), URL constructor normalises it', () => {
      // new URL('javascript%3Aalert(1)') throws → returns false, which is also correct
      expect(isValidUrl('javascript%3Aalert(1)')).toBe(false)
    })

    it('rejects data: URLs', () => {
      expect(isValidUrl('data:text/html,<script>alert(1)</script>')).toBe(false)
    })

    it('rejects data: URLs with base64 payload', () => {
      expect(isValidUrl('data:text/html;base64,PHNjcmlwdD5hbGVydCgxKTwvc2NyaXB0Pg==')).toBe(false)
    })

    it('rejects vbscript: URLs', () => {
      expect(isValidUrl('vbscript:MsgBox(1)')).toBe(false)
    })

    it('rejects file: URLs', () => {
      expect(isValidUrl('file:///etc/passwd')).toBe(false)
    })

    it('rejects ftp: URLs', () => {
      expect(isValidUrl('ftp://files.example.com/secret')).toBe(false)
    })
  })

  describe('invalid / empty inputs', () => {
    it('rejects empty string', () => {
      expect(isValidUrl('')).toBe(false)
    })

    it('rejects whitespace-only string', () => {
      expect(isValidUrl('   ')).toBe(false)
    })

    it('rejects null', () => {
      expect(isValidUrl(null)).toBe(false)
    })

    it('rejects undefined', () => {
      expect(isValidUrl(undefined)).toBe(false)
    })

    it('rejects relative paths', () => {
      expect(isValidUrl('/api/health')).toBe(false)
    })

    it('rejects plain hostnames without scheme', () => {
      expect(isValidUrl('example.com')).toBe(false)
    })

    it('rejects non-URL strings', () => {
      expect(isValidUrl('not a url at all')).toBe(false)
    })

    it('rejects URLs with unbound port 0', () => {
      // port :0 is technically a valid URL but not a reachable endpoint;
      // isValidUrl only validates scheme; unbound-port filtering is
      // done separately by hasUnboundPort() in getEffectiveLocalUrl
      expect(isValidUrl('http://localhost:0')).toBe(true)
    })
  })
})

// =============================================================================
// getEffectiveLocalUrl: precedence + scheme filtering
// =============================================================================

describe('getEffectiveLocalUrl', () => {
  it('returns null when no local info provided', () => {
    expect(getEffectiveLocalUrl(undefined)).toEqual({ url: null, source: null })
  })

  it('returns null when both urls are absent', () => {
    expect(getEffectiveLocalUrl(localInfo())).toEqual({ url: null, source: null })
  })

  describe('customUrl precedence', () => {
    it('prefers customUrl over url when both valid', () => {
      const result = getEffectiveLocalUrl(localInfo({
        url: 'http://localhost:3000',
        customUrl: 'https://custom.dev',
      }))
      expect(result.url).toBe('https://custom.dev')
      expect(result.source).toBe('customUrl')
      expect(result.defaultUrl).toBe('http://localhost:3000')
    })

    it('blocks javascript: customUrl, falls back to url', () => {
      const result = getEffectiveLocalUrl(localInfo({
        url: 'http://localhost:3000',
        customUrl: 'javascript:alert(document.cookie)',
      }))
      expect(result.url).toBe('http://localhost:3000')
      expect(result.source).toBe('url')
    })

    it('blocks data: customUrl, falls back to url', () => {
      const result = getEffectiveLocalUrl(localInfo({
        url: 'http://localhost:3000',
        customUrl: 'data:text/html,<script>alert(1)</script>',
      }))
      expect(result.url).toBe('http://localhost:3000')
      expect(result.source).toBe('url')
    })

    it('blocks javascript: customUrl and invalid url, returns null', () => {
      const result = getEffectiveLocalUrl(localInfo({
        customUrl: 'javascript:alert(1)',
      }))
      expect(result.url).toBeNull()
      expect(result.source).toBeNull()
    })
  })

  describe('url fallback', () => {
    it('uses url when customUrl absent', () => {
      const result = getEffectiveLocalUrl(localInfo({ url: 'http://localhost:4000' }))
      expect(result.url).toBe('http://localhost:4000')
      expect(result.source).toBe('url')
    })

    it('blocks javascript: url', () => {
      const result = getEffectiveLocalUrl(localInfo({ url: 'javascript:void(0)' }))
      expect(result.url).toBeNull()
      expect(result.source).toBeNull()
    })

    it('filters out unbound port :0', () => {
      const result = getEffectiveLocalUrl(localInfo({ url: 'http://localhost:0' }))
      expect(result.url).toBeNull()
      expect(result.source).toBeNull()
    })
  })
})

// =============================================================================
// getEffectiveAzureUrl: precedence + scheme filtering
// =============================================================================

describe('getEffectiveAzureUrl', () => {
  it('returns null when no azure info provided', () => {
    expect(getEffectiveAzureUrl(undefined)).toEqual({ url: null, source: null })
  })

  describe('customDomain-user precedence', () => {
    it('returns user customDomain when valid', () => {
      const result = getEffectiveAzureUrl({
        url: 'https://app.azurewebsites.net',
        customDomain: 'https://api.example.com',
        customDomainSource: 'user',
      })
      expect(result.url).toBe('https://api.example.com')
      expect(result.source).toBe('customDomain-user')
    })

    it('blocks javascript: customDomain (user)', () => {
      const result = getEffectiveAzureUrl({
        url: 'https://app.azurewebsites.net',
        customDomain: 'javascript:alert(1)',
        customDomainSource: 'user',
      })
      // Falls through to url
      expect(result.url).toBe('https://app.azurewebsites.net')
      expect(result.source).toBe('url')
    })
  })

  describe('customDomain-sdk precedence', () => {
    it('returns sdk customDomain when valid', () => {
      const result = getEffectiveAzureUrl({
        url: 'https://app.azurewebsites.net',
        customDomain: 'https://custom.example.com',
        customDomainSource: 'azure-sdk',
      })
      expect(result.url).toBe('https://custom.example.com')
      expect(result.source).toBe('customDomain-sdk')
    })

    it('blocks javascript: customDomain (sdk)', () => {
      const result = getEffectiveAzureUrl({
        url: 'https://app.azurewebsites.net',
        customDomain: 'javascript:alert(1)',
        customDomainSource: 'azure-sdk',
      })
      expect(result.url).toBe('https://app.azurewebsites.net')
      expect(result.source).toBe('url')
    })
  })

  describe('customUrl precedence', () => {
    it('returns customUrl when valid and no customDomain', () => {
      const result = getEffectiveAzureUrl({
        url: 'https://app.azurewebsites.net',
        customUrl: 'https://api.contoso.com',
      })
      expect(result.url).toBe('https://api.contoso.com')
      expect(result.source).toBe('customUrl')
    })

    it('blocks javascript: customUrl', () => {
      const result = getEffectiveAzureUrl({
        url: 'https://app.azurewebsites.net',
        customUrl: 'javascript:alert(document.cookie)',
      })
      expect(result.url).toBe('https://app.azurewebsites.net')
      expect(result.source).toBe('url')
    })

    it('blocks data: customUrl', () => {
      const result = getEffectiveAzureUrl({
        url: 'https://app.azurewebsites.net',
        customUrl: 'data:text/html,<script>xss</script>',
      })
      expect(result.url).toBe('https://app.azurewebsites.net')
      expect(result.source).toBe('url')
    })
  })

  describe('url fallback', () => {
    it('returns url as last resort', () => {
      const result = getEffectiveAzureUrl({ url: 'https://app.azurewebsites.net' })
      expect(result.url).toBe('https://app.azurewebsites.net')
      expect(result.source).toBe('url')
    })

    it('blocks javascript: url', () => {
      const result = getEffectiveAzureUrl({ url: 'javascript:alert(1)' })
      expect(result.url).toBeNull()
      expect(result.source).toBeNull()
    })

    it('returns null when all fields empty', () => {
      const result = getEffectiveAzureUrl({})
      expect(result.url).toBeNull()
      expect(result.source).toBeNull()
    })
  })
})

// =============================================================================
// Badge / icon / tooltip helpers
// =============================================================================

describe('getLocalUrlBadgeConfig', () => {
  it('returns null for null source', () => {
    expect(getLocalUrlBadgeConfig(null)).toBeNull()
  })

  it('returns purple config for customUrl', () => {
    const config = getLocalUrlBadgeConfig('customUrl')
    expect(config).not.toBeNull()
    expect(config?.label).toBe('Custom URL')
    expect(config?.color).toContain('purple')
  })

  it('returns cyan config for url', () => {
    const config = getLocalUrlBadgeConfig('url')
    expect(config).not.toBeNull()
    expect(config?.label).toBe('Local URL')
    expect(config?.color).toContain('cyan')
  })
})

describe('getAzureUrlBadgeConfig', () => {
  it('returns null for null source', () => {
    expect(getAzureUrlBadgeConfig(null)).toBeNull()
  })

  it('returns purple config for customDomain-user', () => {
    const config = getAzureUrlBadgeConfig('customDomain-user')
    expect(config?.label).toBe('Custom Domain')
    expect(config?.color).toContain('purple')
  })

  it('returns amber config for customDomain-sdk', () => {
    const config = getAzureUrlBadgeConfig('customDomain-sdk')
    expect(config?.label).toContain('Azure')
    expect(config?.color).toContain('amber')
  })

  it('returns cyan config for url', () => {
    const config = getAzureUrlBadgeConfig('url')
    expect(config?.label).toBe('Deployment URL')
    expect(config?.color).toContain('cyan')
  })
})

describe('getLocalUrlIconColor', () => {
  it('returns default color for null source', () => {
    expect(getLocalUrlIconColor(null)).toContain('slate')
  })

  it('returns purple for customUrl', () => {
    expect(getLocalUrlIconColor('customUrl')).toContain('purple')
  })

  it('returns cyan for url', () => {
    expect(getLocalUrlIconColor('url')).toContain('cyan')
  })
})

describe('getAzureUrlIconColor', () => {
  it('returns default color for null source', () => {
    expect(getAzureUrlIconColor(null)).toContain('slate')
  })

  it('returns purple for customDomain-user', () => {
    expect(getAzureUrlIconColor('customDomain-user')).toContain('purple')
  })

  it('returns amber for customDomain-sdk', () => {
    expect(getAzureUrlIconColor('customDomain-sdk')).toContain('amber')
  })
})

describe('getLocalUrlTooltip', () => {
  it('returns undefined when source is null', () => {
    expect(getLocalUrlTooltip({ url: null, source: null })).toBeUndefined()
  })

  it('returns undefined when source is url', () => {
    expect(getLocalUrlTooltip({ url: 'http://localhost:3000', source: 'url' })).toBeUndefined()
  })

  it('returns tooltip with defaultUrl when customUrl with defaultUrl', () => {
    const tooltip = getLocalUrlTooltip({
      url: 'https://custom.dev',
      source: 'customUrl',
      defaultUrl: 'http://localhost:3000',
    })
    expect(tooltip).toContain('http://localhost:3000')
  })

  it('returns undefined for customUrl without defaultUrl', () => {
    expect(getLocalUrlTooltip({ url: 'https://custom.dev', source: 'customUrl' })).toBeUndefined()
  })
})

describe('getAzureUrlTooltip', () => {
  it('returns undefined when source is null', () => {
    expect(getAzureUrlTooltip({ url: null, source: null })).toBeUndefined()
  })

  it('returns tooltip for customDomain-user with deployment URL', () => {
    const tooltip = getAzureUrlTooltip({
      url: 'https://api.example.com',
      source: 'customDomain-user',
      allUrls: { url: 'https://app.azurewebsites.net' },
    })
    expect(tooltip).toContain('User-configured')
    expect(tooltip).toContain('https://app.azurewebsites.net')
  })

  it('returns tooltip for customDomain-sdk with fallback', () => {
    const tooltip = getAzureUrlTooltip({
      url: 'https://custom.example.com',
      source: 'customDomain-sdk',
      defaultUrl: 'https://app.azurewebsites.net',
    })
    expect(tooltip).toContain('Azure-discovered')
    expect(tooltip).toContain('https://app.azurewebsites.net')
  })
})
