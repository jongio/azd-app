import { describe, it, expect } from 'vitest'
import { convertAnsiToHtml, sanitizeHref, isErrorLine, isWarningLine, getLogLevel, getServiceColor, getLogColor, stripEmbeddedTimestamp } from './log-utils'

describe('log-utils', () => {
  describe('convertAnsiToHtml', () => {
    it('should convert plain text', () => {
      const result = convertAnsiToHtml('Hello World')
      expect(result).toBe('Hello World')
    })

    it('should detect theme and use appropriate colors', () => {
      // Test light mode (default, no 'dark' class)
      document.documentElement.classList.remove('dark')
      const lightResult = convertAnsiToHtml('\x1b[31mError\x1b[0m')
      expect(lightResult).toBeTruthy()
      
      // Test dark mode
      document.documentElement.classList.add('dark')
      const darkResult = convertAnsiToHtml('\x1b[31mError\x1b[0m')
      expect(darkResult).toBeTruthy()
      
      // Cleanup
      document.documentElement.classList.remove('dark')
    })

    it('should escape HTML special characters', () => {
      const result = convertAnsiToHtml('<script>alert("xss")</script>')
      expect(result).not.toContain('<script>')
      expect(result).toContain('&lt;script&gt;')
    })

    // -----------------------------------------------------------------------
    // SEC-016: XSS bypass vectors, verify the upstream security boundary
    //
    // The previous sanitizeHtml() regex blocklist was bypassable (CWE-184).
    // Security now relies solely on ansi-to-html's escapeXML:true option,
    // which calls entities.encodeXML() on every text token before any HTML
    // is assembled.  These four tests confirm that each classic bypass vector
    // is neutralised by the upstream encoding and cannot produce injectable HTML.
    // -----------------------------------------------------------------------

    it('SEC-016 bypass: whitespace-in-handler (on\\tmouseover=) is not injected as an HTML attribute', () => {
      // The old regex `on\w+=` would not match on<TAB>mouseover= because \t is not \w.
      // With escapeXML:true there are no < > to form new tags, so the text stays
      // as inert text content inside a span; event handlers in text content never fire.
      const result = convertAnsiToHtml('click on\tmouseover=alert(1)')
      // Must not appear as an attribute inside any HTML element
      expect(result).not.toMatch(/<[^>]+on[\s\t]+mouseover\s*=/i)
      // The text itself appears as content; that is safe
      expect(result).toContain('on\tmouseover=alert(1)')
    })

    it('SEC-016 bypass: tag-blocklist-miss (<svg onload=>) is entity-encoded by escapeXML:true', () => {
      // The old blocklist had no entry for <svg>. escapeXML:true converts the
      // user-supplied < to &lt; so no real SVG element is ever emitted.
      const result = convertAnsiToHtml('<svg onload=alert(document.cookie)>')
      expect(result).not.toContain('<svg')
      expect(result).toContain('&lt;svg')
    })

    it('SEC-016 bypass: javascript: scheme is not injected into any href attribute', () => {
      // URL_PATTERN only matches http:// and https://, so javascript: URIs are never
      // wrapped in an anchor tag.  The text appears as safe content, not a link.
      const result = convertAnsiToHtml('try: javascript:alert(document.cookie)')
      expect(result).not.toContain('href="javascript:')
      expect(result).not.toContain("href='javascript:")
      // Confirm no anchor element was created for it
      expect(result).not.toMatch(/<a\b[^>]*javascript:/i)
    })

    it('SEC-016 bypass: nested-tag pattern (<<script>script>) is entity-encoded, not parsed as HTML', () => {
      // Nested-tag tricks rely on a parser consuming one copy of the tag while a regex
      // misses the outer shell.  escapeXML:true converts both < characters to &lt; so
      // neither the outer nor inner angle bracket reaches the browser as HTML.
      const result = convertAnsiToHtml('<<script>script>alert(1)</script>')
      expect(result).not.toContain('<script>')
      expect(result).toContain('&lt;script&gt;')
      // The double-open bracket must also be encoded
      expect(result).not.toMatch(/<[^>]*script/i)
    })

    it('should linkify http URLs', () => {
      const result = convertAnsiToHtml('Server running at http://localhost:5173/')
      expect(result).toContain('<a href="http://localhost:5173/"')
      expect(result).toContain('target="_blank"')
      expect(result).toContain('rel="noopener noreferrer"')
    })

    it('should linkify https URLs', () => {
      const result = convertAnsiToHtml('Visit https://example.com/path')
      expect(result).toContain('<a href="https://example.com/path"')
      expect(result).toContain('target="_blank"')
    })

    it('should linkify URLs with ports', () => {
      const result = convertAnsiToHtml('Local: http://localhost:3000/')
      expect(result).toContain('<a href="http://localhost:3000/"')
    })

    it('should linkify multiple URLs in same message', () => {
      const result = convertAnsiToHtml('Local: http://localhost:3000/ Network: http://192.168.1.1:3000/')
      expect(result).toContain('href="http://localhost:3000/"')
      expect(result).toContain('href="http://192.168.1.1:3000/"')
    })

    it('should not include trailing punctuation in URLs', () => {
      const result = convertAnsiToHtml('Check http://example.com, please.')
      expect(result).toContain('href="http://example.com"')
      expect(result).not.toContain('href="http://example.com,"')
    })

    it('should handle URLs with query strings', () => {
      const result = convertAnsiToHtml('API at http://localhost:8080/api?key=value&foo=bar')
      // The href should have the raw URL (browsers handle & in href correctly)
      expect(result).toContain('href="http://localhost:8080/api?key=value&foo=bar"')
      // The display text has &amp; because it's HTML-escaped
      expect(result).toContain('>http://localhost:8080/api?key=value&amp;foo=bar</a>')
    })

    it('should add clickable link styling', () => {
      const result = convertAnsiToHtml('http://localhost:5173/')
      expect(result).toContain('class="text-cyan-400 hover:text-cyan-300 hover:underline"')
    })

    it('should preserve text before and after URL', () => {
      const result = convertAnsiToHtml('VITE v6.4.1 ready in 434 ms -> Local: http://localhost:5173/')
      expect(result).toContain('VITE v6.4.1 ready in 434 ms -&gt; Local:')
      expect(result).toContain('href="http://localhost:5173/"')
    })

    it('should linkify URLs with ANSI codes around the port', () => {
      // ANSI code around the port number
      const result = convertAnsiToHtml('http://localhost:\x1b[32m5555\x1b[0m')
      expect(result).toContain('href="http://localhost:5555"')
      expect(result).toContain('<a ')
    })

    it('should linkify URLs with ANSI codes around the colon', () => {
      // ANSI code around the colon
      const result = convertAnsiToHtml('http://localhost\x1b[36m:\x1b[0m5555')
      expect(result).toContain('href="http://localhost:5555"')
      expect(result).toContain('<a ')
    })

    it('should linkify URLs fully wrapped in ANSI codes', () => {
      // Full URL wrapped in ANSI
      const result = convertAnsiToHtml('Local: \x1b[36mhttp://localhost:5173/\x1b[0m')
      expect(result).toContain('href="http://localhost:5173/"')
    })

    it('should linkify plain URLs without ports', () => {
      const result = convertAnsiToHtml('Visit http://localhost for more info')
      expect(result).toContain('href="http://localhost"')
    })

    it('should produce an href value that contains no raw double-quotes (CWE-79 regression)', () => {
      // Normal localhost URL, verify the attribute value itself is quote-free
      const result = convertAnsiToHtml('Server at http://localhost:3000/')
      const hrefMatch = result.match(/href="([^"]*)"/)
      expect(hrefMatch).toBeTruthy()
      expect(hrefMatch![1]).not.toContain('"')
      expect(hrefMatch![1]).toBe('http://localhost:3000/')
    })

    it('should encode curly braces in URLs matched by URL_PATTERN', () => {
      // {} are allowed mid-URL by URL_PATTERN but are unsafe in href; encodeURI encodes them
      const result = convertAnsiToHtml('API at http://example.com/api?filter={name}')
      expect(result).toContain('href="http://example.com/api?filter=%7Bname%7D"')
      expect(result).not.toContain('href="http://example.com/api?filter={name}"')
    })
  })

  describe('sanitizeHref', () => {
    it('should encode double-quotes to prevent href attribute breakout (CWE-79)', () => {
      // A URL containing " would break out of href="..." if not encoded
      const xssUrl = 'http://example.com/path"onmouseover="alert(1)'
      const result = sanitizeHref(xssUrl)
      expect(result).not.toContain('"')
      expect(result).toContain('%22')
      expect(result).toMatch(/^http:\/\/example\.com\/path%22/)
    })

    it('should keep normal URLs unchanged', () => {
      expect(sanitizeHref('http://localhost:3000/')).toBe('http://localhost:3000/')
    })

    it('should preserve query-string delimiters (& = ?) so links stay functional', () => {
      const url = 'http://localhost:8080/api?key=value&foo=bar'
      expect(sanitizeHref(url)).toBe('http://localhost:8080/api?key=value&foo=bar')
    })

    it('should encode curly braces in query params', () => {
      expect(sanitizeHref('http://example.com/api?filter={name}')).toBe(
        'http://example.com/api?filter=%7Bname%7D'
      )
    })
  })

  describe('isErrorLine', () => {
    it('should detect error keywords', () => {
      expect(isErrorLine('ERROR: something failed')).toBe(true)
      expect(isErrorLine('Exception thrown')).toBe(true)
      expect(isErrorLine('FATAL error occurred')).toBe(true)
    })

    it('should not flag informational messages with error-like words', () => {
      expect(isErrorLine('Found 0 errors')).toBe(false)
      expect(isErrorLine('Debug mode:')).toBe(false)
    })

    it('should return false for normal messages', () => {
      expect(isErrorLine('Server started')).toBe(false)
      expect(isErrorLine('Request completed')).toBe(false)
    })
  })

  describe('isWarningLine', () => {
    it('should detect warning keywords', () => {
      expect(isWarningLine('WARNING: deprecated API')).toBe(true)
      expect(isWarningLine('Caution: high memory usage')).toBe(true)
    })

    it('should not flag informational messages', () => {
      expect(isWarningLine('WARNING: This is a development server')).toBe(false)
    })

    it('should return false for normal messages', () => {
      expect(isWarningLine('Server started')).toBe(false)
    })
  })

  describe('getLogLevel', () => {
    it('should return error for stderr', () => {
      expect(getLogLevel('normal message', undefined, true)).toBe('error')
    })

    it('should return error for error-level numeric', () => {
      expect(getLogLevel('normal message', 3)).toBe('error')
    })

    it('should return warning for warning-level numeric', () => {
      expect(getLogLevel('normal message', 2)).toBe('warning')
    })

    it('should return info by default', () => {
      expect(getLogLevel('normal message', 1)).toBe('info')
    })

    it('should detect error from message content', () => {
      expect(getLogLevel('ERROR: something failed')).toBe('error')
    })

    it('should detect warning from message content', () => {
      expect(getLogLevel('WARNING: deprecated')).toBe('warning')
    })
  })

  describe('getServiceColor', () => {
    it('should return a color class', () => {
      const color = getServiceColor('api')
      expect(color).toMatch(/^text-\w+-400$/)
    })

    it('should return consistent colors for same service', () => {
      const color1 = getServiceColor('web')
      const color2 = getServiceColor('web')
      expect(color1).toBe(color2)
    })

    it('should return different colors for different services', () => {
      // Note: This test might occasionally fail if two services hash to the same color
      const colors = ['api', 'web', 'worker', 'db', 'cache', 'queue'].map(getServiceColor)
      const uniqueColors = new Set(colors)
      expect(uniqueColors.size).toBeGreaterThan(1)
    })
  })

  describe('getLogColor', () => {
    it('should return red for errors', () => {
      expect(getLogColor('error')).toBe('text-red-400')
    })

    it('should return yellow for warnings', () => {
      expect(getLogColor('warning')).toBe('text-yellow-400')
    })

    it('should return tertiary for info', () => {
      expect(getLogColor('info')).toBe('text-foreground-tertiary')
    })
  })

  describe('stripEmbeddedTimestamp', () => {
    it('should strip ISO 8601 timestamps with brackets', () => {
      const result = stripEmbeddedTimestamp('[2025-12-13T05:45:49.1071934-08:00] [appservice-web] [INFO] Health endpoint hit')
      expect(result).toBe('[INFO] Health endpoint hit')
    })

    it('should strip date time timestamps with brackets', () => {
      const result = stripEmbeddedTimestamp('[2025-12-13 05:45:49] [INFO] GET / - 200')
      expect(result).toBe('[INFO] GET / - 200')
    })

    it('should strip time only timestamps with brackets', () => {
      const result = stripEmbeddedTimestamp('[05:45:49] Server started')
      expect(result).toBe('Server started')
    })

    it('should strip time with milliseconds', () => {
      const result = stripEmbeddedTimestamp('[08:20:50.670] Request received')
      expect(result).toBe('Request received')
    })

    it('should strip service name prefix', () => {
      const result = stripEmbeddedTimestamp('[appservice-web] GET / endpoint called')
      expect(result).toBe('GET / endpoint called')
    })

    it('should strip multiple nested timestamp patterns', () => {
      // This is the exact pattern from user report
      const result = stripEmbeddedTimestamp('[2025-12-13T05:45:49.1071934-08:00] [appservice-web] [2025-12-13 05:45:49] [INFO] Health endpoint hit - appservice-web is healthy')
      expect(result).toBe('[INFO] Health endpoint hit - appservice-web is healthy')
    })

    it('should preserve messages without timestamps', () => {
      const result = stripEmbeddedTimestamp('Server started successfully')
      expect(result).toBe('Server started successfully')
    })

    it('should preserve log level prefixes', () => {
      const result = stripEmbeddedTimestamp('[INFO] GET / - 200')
      expect(result).toBe('[INFO] GET / - 200')
    })

    it('should handle whitespace before timestamps', () => {
      const result = stripEmbeddedTimestamp('  [2025-12-13 05:45:49] Server started')
      expect(result).toBe('Server started')
    })

    it('should not strip service name when disabled', () => {
      const result = stripEmbeddedTimestamp('[appservice-web] GET / endpoint called', false)
      expect(result).toBe('[appservice-web] GET / endpoint called')
    })

    it('should handle empty strings', () => {
      const result = stripEmbeddedTimestamp('')
      expect(result).toBe('')
    })

    it('should handle timezone offset in ISO timestamps', () => {
      const result = stripEmbeddedTimestamp('[2025-12-13T16:20:50Z] Request processed')
      expect(result).toBe('Request processed')
    })
  })
})
