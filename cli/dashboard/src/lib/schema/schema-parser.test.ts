/**
 * Tests for schema-parser.ts
 */

import { describe, it, expect, beforeEach } from 'vitest'
import { parseSchema, getPropertyByPath, type ParsedSchema } from './schema-parser'

describe('schema-parser', () => {
  describe('parseSchema', () => {
    it('should parse basic schema properties', () => {
      const schema = {
        title: 'Test Schema',
        type: 'object',
        required: ['name'],
        properties: {
          name: {
            type: 'string',
            title: 'Name',
            description: 'The name field',
            minLength: 2,
            pattern: '^[a-z]+$',
          },
          count: {
            type: 'number',
            minimum: 0,
            maximum: 100,
          },
        },
      }

      const parsed = parseSchema(schema)

      expect(parsed.name).toBe('Test Schema')
      expect(parsed.required).toEqual(['name'])
      expect(parsed.properties.name).toBeDefined()
      expect(parsed.properties.name.type).toBe('string')
      expect(parsed.properties.name.required).toBe(true)
      expect(parsed.properties.name.title).toBe('Name')
      expect(parsed.properties.name.description).toBe('The name field')
      expect(parsed.properties.name.minLength).toBe(2)
      expect(parsed.properties.name.pattern).toBe('^[a-z]+$')

      expect(parsed.properties.count).toBeDefined()
      expect(parsed.properties.count.type).toBe('number')
      expect(parsed.properties.count.required).toBe(false)
      expect(parsed.properties.count.minimum).toBe(0)
      expect(parsed.properties.count.maximum).toBe(100)
    })

    it('should parse enum properties', () => {
      const schema = {
        type: 'object',
        properties: {
          status: {
            type: 'string',
            enum: ['active', 'inactive', 'pending'],
            description: 'Status field',
          },
        },
      }

      const parsed = parseSchema(schema)

      expect(parsed.properties.status.type).toBe('enum')
      expect(parsed.properties.status.enumValues).toEqual(['active', 'inactive', 'pending'])
    })

    it('should parse object properties with nested fields', () => {
      const schema = {
        type: 'object',
        properties: {
          config: {
            type: 'object',
            required: ['host'],
            properties: {
              host: {
                type: 'string',
              },
              port: {
                type: 'number',
                default: 8080,
              },
            },
          },
        },
      }

      const parsed = parseSchema(schema)

      expect(parsed.properties.config.type).toBe('object')
      expect(parsed.properties.config.properties).toBeDefined()
      expect(parsed.properties.config.properties?.host).toBeDefined()
      expect(parsed.properties.config.properties?.host.type).toBe('string')
      expect(parsed.properties.config.properties?.host.required).toBe(true)
      expect(parsed.properties.config.properties?.port).toBeDefined()
      expect(parsed.properties.config.properties?.port.defaultValue).toBe(8080)
    })

    it('should parse array properties', () => {
      const schema = {
        type: 'object',
        properties: {
          tags: {
            type: 'array',
            items: {
              type: 'string',
            },
            minItems: 1,
            maxItems: 10,
          },
        },
      }

      const parsed = parseSchema(schema)

      expect(parsed.properties.tags.type).toBe('array')
      expect(parsed.properties.tags.items).toBeDefined()
      expect(parsed.properties.tags.items?.type).toBe('string')
      expect(parsed.properties.tags.minItems).toBe(1)
      expect(parsed.properties.tags.maxItems).toBe(10)
    })

    it('should extract validation rules', () => {
      const schema = {
        type: 'object',
        required: ['name', 'age'],
        properties: {
          name: {
            type: 'string',
            minLength: 2,
            maxLength: 50,
            pattern: '^[a-zA-Z ]+$',
          },
          age: {
            type: 'number',
            minimum: 0,
            maximum: 120,
          },
          status: {
            type: 'string',
            enum: ['active', 'inactive'],
          },
        },
      }

      const parsed = parseSchema(schema)

      // Name validations
      const nameValidation = parsed.properties.name.validation
      expect(nameValidation).toContainEqual({ type: 'required', value: true })
      expect(nameValidation).toContainEqual({ type: 'minLength', value: 2 })
      expect(nameValidation).toContainEqual({ type: 'maxLength', value: 50 })
      expect(nameValidation.find(v => v.type === 'pattern')).toBeDefined()

      // Age validations
      const ageValidation = parsed.properties.age.validation
      expect(ageValidation).toContainEqual({ type: 'required', value: true })
      expect(ageValidation).toContainEqual({ type: 'min', value: 0 })
      expect(ageValidation).toContainEqual({ type: 'max', value: 120 })

      // Status validations
      const statusValidation = parsed.properties.status.validation
      expect(statusValidation.find(v => v.type === 'enum')).toBeDefined()
    })

    it('should handle default values', () => {
      const schema = {
        type: 'object',
        properties: {
          port: {
            type: 'number',
            default: 3000,
          },
          enabled: {
            type: 'boolean',
            default: true,
          },
        },
      }

      const parsed = parseSchema(schema)

      expect(parsed.properties.port.defaultValue).toBe(3000)
      expect(parsed.properties.enabled.defaultValue).toBe(true)
    })

    it('should handle boolean properties', () => {
      const schema = {
        type: 'object',
        properties: {
          enabled: {
            type: 'boolean',
            description: 'Enable feature',
          },
        },
      }

      const parsed = parseSchema(schema)

      expect(parsed.properties.enabled.type).toBe('boolean')
      expect(parsed.properties.enabled.description).toBe('Enable feature')
    })

    it('should handle union types (array of types)', () => {
      const schema = {
        type: 'object',
        properties: {
          value: {
            type: ['string', 'null'],
          },
        },
      }

      const parsed = parseSchema(schema)

      // Should pick first non-null type
      expect(parsed.properties.value.type).toBe('string')
    })

    it('should parse definitions section', () => {
      const schema = {
        type: 'object',
        properties: {
          service: {
            $ref: '#/definitions/service',
          },
        },
        definitions: {
          service: {
            type: 'object',
            properties: {
              host: {
                type: 'string',
                enum: ['local', 'containerapp'],
              },
            },
          },
        },
      }

      const parsed = parseSchema(schema)

      expect(parsed.definitions).toBeDefined()
      expect(parsed.definitions.service).toBeDefined()
      expect(parsed.definitions.service.type).toBe('object')
      expect(parsed.definitions.service.properties?.host).toBeDefined()
    })

    it('should handle missing properties gracefully', () => {
      const schema = {
        type: 'object',
      }

      const parsed = parseSchema(schema)

      expect(parsed.properties).toEqual({})
      expect(parsed.required).toEqual([])
      expect(parsed.name).toBe('Azure YAML Configuration')
    })
  })

  describe('getPropertyByPath', () => {
    let schema: ParsedSchema

    beforeEach(() => {
      schema = parseSchema({
        type: 'object',
        properties: {
          name: {
            type: 'string',
          },
          services: {
            type: 'object',
            additionalProperties: {
              type: 'object',
              properties: {
                host: {
                  type: 'string',
                  enum: ['local', 'containerapp'],
                },
                ports: {
                  type: 'array',
                  items: {
                    type: 'string',
                  },
                },
              },
            },
          },
        },
        definitions: {
          service: {
            type: 'object',
            properties: {
              host: {
                type: 'string',
              },
            },
          },
        },
      })
    })

    it('should get top-level property', () => {
      const prop = getPropertyByPath(schema, 'name')

      expect(prop).toBeDefined()
      expect(prop?.type).toBe('string')
    })

    it('should get nested property', () => {
      const prop = getPropertyByPath(schema, 'services')

      expect(prop).toBeDefined()
      expect(prop?.type).toBe('object')
    })

    it('should get property from definitions', () => {
      const prop = getPropertyByPath(schema, 'service')

      expect(prop).toBeDefined()
      expect(prop?.type).toBe('object')
      expect(prop?.properties?.host).toBeDefined()
    })

    it('should return null for non-existent property', () => {
      const prop = getPropertyByPath(schema, 'nonexistent')

      expect(prop).toBeNull()
    })

    it('should return null for invalid nested path', () => {
      const prop = getPropertyByPath(schema, 'name.invalid.path')

      expect(prop).toBeNull()
    })
  })
})
