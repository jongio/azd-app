import { describe, it, expect } from 'vitest'
import { convertAnsiToHtml, isErrorLine, isWarningLine, getLogLevel, getServiceColor, getLogColor } from './log-utils'

describe('log-utils', () => {
  describe('convertAnsiToHtml', () => {
    it('should convert plain text', () => {
      const result = convertAnsiToHtml('Hello World')
      expect(result).toBe('Hello World')
    })

    it('should escape HTML special characters', () => {
      const result = convertAnsiToHtml('<script>alert("xss")</script>')
      expect(result).not.toContain('<script>')
      expect(result).toContain('&lt;script&gt;')
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
})
