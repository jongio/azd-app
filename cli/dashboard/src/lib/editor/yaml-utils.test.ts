import { describe, it, expect } from 'vitest'
import {
  parseYaml,
  stringifyYaml,
  validateYaml,
  safeParseYaml,
  isValidYaml,
  formatYaml,
} from './yaml-utils'

describe('yaml-utils', () => {
  describe('parseYaml', () => {
    it('should parse valid YAML', () => {
      const yamlString = `name: test-app
services:
  api:
    host: containerapp
    language: node
`

      const result = parseYaml(yamlString)

      expect(result.success).toBe(true)
      expect(result.data).toEqual({
        name: 'test-app',
        services: {
          api: {
            host: 'containerapp',
            language: 'node',
          },
        },
      })
      expect(result.error).toBeUndefined()
    })

    it('should handle parsing errors', () => {
      const invalidYaml = `this is: not: valid: yaml: : :`

      const result = parseYaml(invalidYaml)

      expect(result.success).toBe(false)
      expect(result.data).toBeUndefined()
      expect(result.error).toBeDefined()
    })

    it('should parse empty YAML', () => {
      const result = parseYaml('')

      expect(result.success).toBe(true)
      // Empty YAML can be parsed as null or undefined
      expect(result.data == null).toBe(true)
    })

    it('should parse arrays', () => {
      const yamlString = `ports:
  - "8080:8080"
  - "8081:8081"
`

      const result = parseYaml(yamlString)

      expect(result.success).toBe(true)
      expect(result.data).toEqual({
        ports: ['8080:8080', '8081:8081'],
      })
    })

    it('should parse nested objects', () => {
      const yamlString = `services:
  api:
    host: containerapp
    environment:
      NODE_ENV: production
      PORT: "8080"
`

      const result = parseYaml(yamlString)

      expect(result.success).toBe(true)
      expect(result.data).toEqual({
        services: {
          api: {
            host: 'containerapp',
            environment: {
              NODE_ENV: 'production',
              PORT: '8080',
            },
          },
        },
      })
    })
  })

  describe('stringifyYaml', () => {
    it('should stringify object to YAML', () => {
      const data = {
        name: 'test-app',
        services: {
          api: {
            host: 'containerapp',
            language: 'node',
          },
        },
      }

      const result = stringifyYaml(data)

      expect(result).toBeTruthy()
      expect(result).toContain('name: test-app')
      expect(result).toContain('services:')
      expect(result).toContain('api:')
      expect(result).toContain('host: containerapp')
      expect(result).toContain('language: node')
    })

    it('should use 2-space indentation by default', () => {
      const data = {
        services: {
          api: {
            host: 'containerapp',
          },
        },
      }

      const result = stringifyYaml(data)

      expect(result).toContain('services:\n  api:\n    host: containerapp')
    })

    it('should use custom indentation', () => {
      const data = {
        services: {
          api: {
            host: 'containerapp',
          },
        },
      }

      const result = stringifyYaml(data, { indent: 4 })

      expect(result).toContain('services:\n    api:\n        host: containerapp')
    })

    it('should stringify arrays', () => {
      const data = {
        ports: ['8080:8080', '8081:8081'],
      }

      const result = stringifyYaml(data)

      expect(result).toContain('ports:')
      // js-yaml may quote or not quote array items
      expect(result).toMatch(/8080:8080/)
      expect(result).toMatch(/8081:8081/)
    })

    it('should handle null values', () => {
      const data = {
        name: 'test',
        value: null,
      }

      const result = stringifyYaml(data)

      expect(result).toContain('name: test')
      // js-yaml uses ~ for null values
      expect(result).toMatch(/value:\s+(null|~)/)
    })
  })

  describe('validateYaml', () => {
    it('should validate valid YAML', () => {
      const yamlString = `name: test
services:
  api:
    host: containerapp
`

      const result = validateYaml(yamlString)

      expect(result.valid).toBe(true)
      expect(result.error).toBeUndefined()
    })

    it('should detect invalid YAML', () => {
      const invalidYaml = `this is: not: valid: yaml: : :`

      const result = validateYaml(invalidYaml)

      expect(result.valid).toBe(false)
      expect(result.error).toBeDefined()
    })
  })

  describe('safeParseYaml', () => {
    it('should return parsed data for valid YAML', () => {
      const yamlString = `name: test`

      const result = safeParseYaml(yamlString)

      expect(result).toEqual({ name: 'test' })
    })

    it('should return null for invalid YAML', () => {
      const invalidYaml = `this is: not: valid: yaml: : :`

      const result = safeParseYaml(invalidYaml)

      expect(result).toBeNull()
    })
  })

  describe('isValidYaml', () => {
    it('should return true for valid YAML', () => {
      const yamlString = `name: test\nservices:\n  api:\n    host: containerapp\n`

      expect(isValidYaml(yamlString)).toBe(true)
    })

    it('should return false for invalid YAML', () => {
      const invalidYaml = `this is: not: valid: yaml: : :`

      expect(isValidYaml(invalidYaml)).toBe(false)
    })
  })

  describe('formatYaml', () => {
    it('should format valid YAML consistently', () => {
      const unformattedYaml = `name: test
services:
   api:
      host: containerapp
`

      const formatted = formatYaml(unformattedYaml)

      expect(formatted).toContain('name: test')
      expect(formatted).toContain('services:')
      expect(formatted).toContain('  api:')
      expect(formatted).toContain('    host: containerapp')
    })

    it('should return original string for invalid YAML', () => {
      const invalidYaml = `this is: not: valid: yaml: : :`

      const result = formatYaml(invalidYaml)

      expect(result).toBe(invalidYaml)
    })

    it('should use custom formatting options', () => {
      const yamlString = `name: test
services:
  api:
    host: containerapp
`

      const formatted = formatYaml(yamlString, { indent: 4 })

      expect(formatted).toContain('    api:')
      expect(formatted).toContain('        host: containerapp')
    })
  })

  describe('round-trip parsing', () => {
    it('should preserve data through parse and stringify', () => {
      const original = {
        name: 'test-app',
        services: {
          api: {
            host: 'containerapp',
            language: 'node',
            ports: ['8080:8080'],
            environment: {
              NODE_ENV: 'production',
            },
          },
        },
      }

      const yamlString = stringifyYaml(original)
      const parsed = parseYaml(yamlString)

      expect(parsed.success).toBe(true)
      expect(parsed.data).toEqual(original)
    })
  })
})
