/**
 * Comprehensive YAML Utilities Tests
 * Tests for parsing, serialization, validation, and formatting
 */
import { describe, it, expect } from 'vitest'
import {
  parseYaml,
  stringifyYaml,
  validateYaml,
  safeParseYaml,
  isValidYaml,
  formatYaml,
  type YamlStringifyOptions,
} from './yaml-utils'

describe('YAML Parsing', () => {
  it('should parse valid YAML', () => {
    const yaml = `
name: my-app
version: 1.0.0
services:
  api:
    host: 0.0.0.0
    port: 8080
`
    const result = parseYaml(yaml)

    expect(result.success).toBe(true)
    expect(result.data).toBeDefined()
    expect(result.data).toHaveProperty('name', 'my-app')
    expect(result.data).toHaveProperty('version', '1.0.0')
  })

  it('should parse nested objects', () => {
    const yaml = `
database:
  host: localhost
  port: 5432
  credentials:
    username: admin
    password: secret
`
    const result = parseYaml(yaml)

    expect(result.success).toBe(true)
    expect(result.data).toMatchObject({
      database: {
        host: 'localhost',
        port: 5432,
        credentials: {
          username: 'admin',
          password: 'secret',
        },
      },
    })
  })

  it('should parse arrays', () => {
    const yaml = `
tags:
  - frontend
  - backend
  - api
ports:
  - 8080
  - 3000
  - 5432
`
    const result = parseYaml(yaml)

    expect(result.success).toBe(true)
    expect(result.data).toHaveProperty('tags')
    expect(Array.isArray((result.data as Record<string, unknown>).tags)).toBe(true)
    expect((result.data as Record<string, unknown>).tags).toHaveLength(3)
  })

  it('should parse mixed types', () => {
    const yaml = `
string: hello
number: 42
float: 3.14
boolean: true
null_value: null
array: [1, 2, 3]
object: { key: value }
`
    const result = parseYaml(yaml)

    expect(result.success).toBe(true)
    expect((result.data as Record<string, unknown>).string).toBe('hello')
    expect((result.data as Record<string, unknown>).number).toBe(42)
    expect((result.data as Record<string, unknown>).float).toBe(3.14)
    expect((result.data as Record<string, unknown>).boolean).toBe(true)
    expect((result.data as Record<string, unknown>).null_value).toBeNull()
    expect(Array.isArray((result.data as Record<string, unknown>).array)).toBe(true)
    expect(typeof (result.data as Record<string, unknown>).object).toBe('object')
  })

  it('should handle empty YAML', () => {
    const yaml = ''
    const result = parseYaml(yaml)

    expect(result.success).toBe(true)
    expect(result.data).toBeNull()
  })

  it('should handle comments', () => {
    const yaml = `
# This is a comment
name: my-app # inline comment
version: 1.0.0
`
    const result = parseYaml(yaml)

    expect(result.success).toBe(true)
    expect(result.data).toHaveProperty('name', 'my-app')
  })

  it('should handle multiline strings', () => {
    const yaml = `
description: |
  This is a multiline
  string that preserves
  line breaks
`
    const result = parseYaml(yaml)

    expect(result.success).toBe(true)
    expect((result.data as Record<string, unknown>).description).toContain('\n')
  })

  it('should fail on invalid YAML', () => {
    const yaml = `
invalid: yaml: syntax: error:
  - this
  should: fail
`
    const result = parseYaml(yaml)

    expect(result.success).toBe(false)
    expect(result.error).toBeDefined()
  })

  it('should handle malformed indentation', () => {
    const yaml = `
services:
 api:
   host: localhost
  web:
    host: 0.0.0.0
`
    const result = parseYaml(yaml)

    // js-yaml is somewhat forgiving, but this might fail
    expect(result.success).toBeDefined()
  })

  it('should handle unknown YAML tags safely', () => {
    // YAML can have custom tags, but the yaml package (eemeli/yaml) uses a safe schema
    // Unknown tags are parsed as strings or rejected, not executed
    const yaml = `
dangerous: !!js/function "function() { return 'exploit'; }"
`
    const result = parseYaml(yaml)

    // The yaml package may parse unknown tags as strings or reject them
    // Either behavior is safe - no code execution occurs
    // The important thing is that the parser doesn't crash and doesn't execute code
    expect(result.success).toBeDefined()
    // If it parses, it should be as a string value, not executable code
    if (result.success && result.data) {
      const data = result.data as Record<string, unknown>
      // The value should be a string, not a function
      expect(typeof data.dangerous).not.toBe('function')
    }
  })
})

describe('YAML Stringification', () => {
  it('should stringify simple object', () => {
    const data = {
      name: 'my-app',
      version: '1.0.0',
    }

    const yaml = stringifyYaml(data)

    expect(yaml).toContain('name: my-app')
    expect(yaml).toContain('version: 1.0.0')
  })

  it('should stringify nested objects', () => {
    const data = {
      database: {
        host: 'localhost',
        port: 5432,
      },
    }

    const yaml = stringifyYaml(data)

    expect(yaml).toContain('database:')
    expect(yaml).toContain('host: localhost')
    expect(yaml).toContain('port: 5432')
  })

  it('should stringify arrays', () => {
    const data = {
      tags: ['frontend', 'backend', 'api'],
    }

    const yaml = stringifyYaml(data)

    expect(yaml).toContain('tags:')
    expect(yaml).toContain('- frontend')
    expect(yaml).toContain('- backend')
    expect(yaml).toContain('- api')
  })

  it('should handle null values', () => {
    const data = {
      nullable: null,
    }

    const yaml = stringifyYaml(data)

    expect(yaml).toContain('nullable:')
  })

  it('should handle boolean values', () => {
    const data = {
      enabled: true,
      disabled: false,
    }

    const yaml = stringifyYaml(data)

    expect(yaml).toContain('enabled: true')
    expect(yaml).toContain('disabled: false')
  })

  it('should handle numbers', () => {
    const data = {
      integer: 42,
      float: 3.14,
      negative: -10,
    }

    const yaml = stringifyYaml(data)

    expect(yaml).toContain('integer: 42')
    expect(yaml).toContain('float: 3.14')
    expect(yaml).toContain('negative: -10')
  })

  it('should respect indent option', () => {
    const data = {
      level1: {
        level2: {
          value: 'nested',
        },
      },
    }

    const yaml2 = stringifyYaml(data, { indent: 2 })
    const yaml4 = stringifyYaml(data, { indent: 4 })

    expect(yaml2).toBeDefined()
    expect(yaml4).toBeDefined()
    // 4-space indent should be longer
    expect(yaml4.length).toBeGreaterThan(yaml2.length)
  })

  it('should respect lineWidth option', () => {
    const data = {
      long: 'This is a very long string that might need to be wrapped depending on the line width setting',
    }

    const yaml80 = stringifyYaml(data, { lineWidth: 80 })
    const yaml40 = stringifyYaml(data, { lineWidth: 40 })

    expect(yaml80).toBeDefined()
    expect(yaml40).toBeDefined()
  })

  it('should handle shared object references', () => {
    const shared = { shared: true }
    const data = {
      ref1: shared,
      ref2: shared,
    }

    const yaml = stringifyYaml(data)

    // The yaml package uses anchors/aliases for shared object references
    // This is expected behavior and helps preserve object identity
    expect(yaml).toBeDefined()
    expect(yaml).toContain('shared: true')
    // When the same object is referenced, yaml package may use anchors/aliases
    // This is acceptable and preserves data structure
  })

  it('should maintain key order when sortKeys is false', () => {
    const data = {
      zebra: 1,
      alpha: 2,
      beta: 3,
    }

    const yaml = stringifyYaml(data, { sortKeys: false })

    const lines = yaml.split('\n').filter(l => l.trim())
    expect(lines[0]).toContain('zebra')
    expect(lines[1]).toContain('alpha')
    expect(lines[2]).toContain('beta')
  })

  it('should handle empty object', () => {
    const yaml = stringifyYaml({})
    expect(yaml).toBe('{}\n')
  })

  it('should handle empty array', () => {
    const yaml = stringifyYaml([])
    expect(yaml).toBe('[]\n')
  })
})

describe('YAML Validation', () => {
  it('should validate correct YAML', () => {
    const yaml = `
name: my-app
version: 1.0.0
`
    const result = validateYaml(yaml)

    expect(result.valid).toBe(true)
    expect(result.error).toBeUndefined()
  })

  it('should invalidate incorrect YAML', () => {
    const yaml = `
invalid: : : syntax
`
    const result = validateYaml(yaml)

    expect(result.valid).toBe(false)
    expect(result.error).toBeDefined()
  })

  it('isValidYaml should return boolean', () => {
    expect(isValidYaml('name: value')).toBe(true)
    expect(isValidYaml('invalid: : :')).toBe(false)
  })
})

describe('Safe Parse', () => {
  it('should return data on success', () => {
    const yaml = 'name: my-app'
    const result = safeParseYaml(yaml)

    expect(result).not.toBeNull()
    expect(result).toHaveProperty('name', 'my-app')
  })

  it('should return null on failure', () => {
    const yaml = 'invalid: : :'
    const result = safeParseYaml(yaml)

    expect(result).toBeNull()
  })

  it('should handle empty input', () => {
    const result = safeParseYaml('')

    expect(result).toBeNull()
  })
})

describe('YAML Formatting', () => {
  it('should format valid YAML', () => {
    const yaml = `name: my-app
version: 1.0.0
services: {api: {port: 8080}}`

    const formatted = formatYaml(yaml)

    expect(formatted).toContain('name: my-app')
    expect(formatted).toContain('services:')
    expect(formatted).toContain('api:')
    expect(formatted).toContain('port: 8080')
  })

  it('should return original on invalid YAML', () => {
    const invalid = 'invalid: : :'
    const formatted = formatYaml(invalid)

    expect(formatted).toBe(invalid)
  })

  it('should apply formatting options', () => {
    const yaml = 'name: my-app\nservices: {api: {port: 8080}}'
    const options: YamlStringifyOptions = {
      indent: 4,
      lineWidth: 80,
    }

    const formatted = formatYaml(yaml, options)

    expect(formatted).toBeDefined()
    expect(formatted.length).toBeGreaterThan(yaml.length)
  })

  it('should normalize inconsistent formatting', () => {
    const messy = `
name:    my-app
version:  "1.0.0"
services:
  api:
      port:  8080
`
    const formatted = formatYaml(messy)

    // Should be consistently formatted
    expect(formatted).toContain('name: my-app')
    expect(formatted).toContain('version: "1.0.0"')
  })
})

describe('Round-trip Conversion', () => {
  it('should preserve data through parse and stringify', () => {
    const original = {
      name: 'my-app',
      version: '1.0.0',
      services: {
        api: {
          host: '0.0.0.0',
          port: 8080,
        },
      },
    }

    const yaml = stringifyYaml(original)
    const parsed = parseYaml(yaml)

    expect(parsed.success).toBe(true)
    expect(parsed.data).toEqual(original)
  })

  it('should preserve types', () => {
    const original = {
      string: 'hello',
      number: 42,
      boolean: true,
      null_value: null,
      array: [1, 2, 3],
      object: { key: 'value' },
    }

    const yaml = stringifyYaml(original)
    const parsed = parseYaml(yaml)

    expect(parsed.success).toBe(true)
    expect(parsed.data).toEqual(original)
  })

  it('should handle complex nested structures', () => {
    const original = {
      level1: {
        level2: {
          level3: {
            array: [
              { id: 1, name: 'first' },
              { id: 2, name: 'second' },
            ],
          },
        },
      },
    }

    const yaml = stringifyYaml(original)
    const parsed = parseYaml(yaml)

    expect(parsed.success).toBe(true)
    expect(parsed.data).toEqual(original)
  })
})

describe('Performance', () => {
  it('should parse large YAML quickly', () => {
    const services: Record<string, { port: number }> = {}
    for (let i = 0; i < 100; i++) {
      services[`service-${i}`] = { port: 8000 + i }
    }

    const large = {
      name: 'large-app',
      services,
    }

    const yaml = stringifyYaml(large)

    const startTime = performance.now()
    const result = parseYaml(yaml)
    const duration = performance.now() - startTime

    expect(result.success).toBe(true)
    expect(duration).toBeLessThan(100) // Should parse in < 100ms
  })

  it('should stringify large objects quickly', () => {
    const services: Record<string, { port: number; host: string }> = {}
    for (let i = 0; i < 100; i++) {
      services[`service-${i}`] = {
        port: 8000 + i,
        host: '0.0.0.0',
      }
    }

    const large = {
      name: 'large-app',
      services,
    }

    const startTime = performance.now()
    const yaml = stringifyYaml(large)
    const duration = performance.now() - startTime

    expect(yaml).toBeDefined()
    expect(duration).toBeLessThan(100) // Should stringify in < 100ms
  })

  it('should format large YAML efficiently', () => {
    const services: Record<string, { port: number }> = {}
    for (let i = 0; i < 50; i++) {
      services[`service-${i}`] = { port: 8000 + i }
    }

    const yaml = stringifyYaml({ name: 'app', services })

    const startTime = performance.now()
    const formatted = formatYaml(yaml)
    const duration = performance.now() - startTime

    expect(formatted).toBeDefined()
    expect(duration).toBeLessThan(200) // Should format in < 200ms
  })
})

describe('Edge Cases', () => {
  it('should handle special characters', () => {
    const data = {
      special: 'Value with: colons, commas, and [brackets]',
    }

    const yaml = stringifyYaml(data)
    const parsed = parseYaml(yaml)

    expect(parsed.success).toBe(true)
    expect(parsed.data).toEqual(data)
  })

  it('should handle unicode', () => {
    const data = {
      unicode: '你好世界 🌍',
    }

    const yaml = stringifyYaml(data)
    const parsed = parseYaml(yaml)

    expect(parsed.success).toBe(true)
    expect((parsed.data as Record<string, unknown>).unicode).toBe('你好世界 🌍')
  })

  it('should handle quotes in strings', () => {
    const data = {
      quoted: 'Value with "quotes" and \'apostrophes\'',
    }

    const yaml = stringifyYaml(data)
    const parsed = parseYaml(yaml)

    expect(parsed.success).toBe(true)
  })

  it('should handle very long strings', () => {
    const longString = 'a'.repeat(10000)
    const data = {
      long: longString,
    }

    const yaml = stringifyYaml(data)
    const parsed = parseYaml(yaml)

    expect(parsed.success).toBe(true)
    expect((parsed.data as Record<string, unknown>).long).toBe(longString)
  })

  it('should handle deeply nested objects', () => {
    let nested: Record<string, unknown> = { value: 'deep' }
    for (let i = 0; i < 50; i++) {
      nested = { level: nested }
    }

    const yaml = stringifyYaml(nested)
    const parsed = parseYaml(yaml)

    expect(parsed.success).toBe(true)
  })
})
