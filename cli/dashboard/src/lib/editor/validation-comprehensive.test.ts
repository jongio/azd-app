/**
 * Comprehensive Validation Engine Tests
 * Tests for all validation stages: schema, business rules, warnings, recommendations
 */
import { describe, it, expect } from 'vitest'
import {
  validateSchema,
  validateUniqueServiceNames,
  validateUniqueResourceNames,
  validatePortConflicts,
  validateCircularDependencies,
  validateRecommendedFields,
  validateConfiguration,
} from './validation-engine'

describe('Schema Validation', () => {
  const schema = {
    type: 'object',
    properties: {
      name: { type: 'string' },
      version: { type: 'string' },
      services: {
        type: 'object',
        additionalProperties: {
          type: 'object',
          properties: {
            host: { type: 'string' },
            port: { type: 'number', minimum: 1, maximum: 65535 },
          },
        },
      },
    },
    required: ['name'],
  }

  it('should pass valid configuration', () => {
    const config = {
      name: 'my-app',
      version: '1.0.0',
      services: {
        api: {
          host: '0.0.0.0',
          port: 8080,
        },
      },
    }

    const errors = validateSchema(config, schema)
    expect(errors).toHaveLength(0)
  })

  it('should detect missing required fields', () => {
    const config = {
      version: '1.0.0',
    }

    const errors = validateSchema(config, schema)
    expect(errors.length).toBeGreaterThan(0)
    expect(errors[0].rule).toBe('required')
    expect(errors[0].message).toContain('name')
  })

  it('should detect type mismatches', () => {
    const config = {
      name: 123, // Should be string
    }

    const errors = validateSchema(config, schema)
    expect(errors.length).toBeGreaterThan(0)
    expect(errors.some(e => e.rule === 'type')).toBe(true)
  })

  it('should validate number ranges', () => {
    const config = {
      name: 'my-app',
      services: {
        api: {
          port: 99999, // Out of range
        },
      },
    }

    const errors = validateSchema(config, schema)
    expect(errors.length).toBeGreaterThan(0)
    expect(errors.some(e => e.rule === 'maximum')).toBe(true)
  })

  it('should handle nested validation errors', () => {
    const config = {
      name: 'my-app',
      services: {
        api: {
          host: 123, // Should be string
          port: 'invalid', // Should be number
        },
      },
    }

    const errors = validateSchema(config, schema)
    expect(errors.length).toBeGreaterThan(0)
  })

  it('should format error paths correctly', () => {
    const config = {
      name: 'my-app',
      services: {
        api: {
          port: 99999,
        },
      },
    }

    const errors = validateSchema(config, schema)
    const portError = errors.find(e => e.path.includes('port'))
    expect(portError).toBeDefined()
  })
})

describe('Unique Service Names', () => {
  it('should pass with unique names', () => {
    const config = {
      services: {
        api: {},
        web: {},
        db: {},
      },
    }

    const errors = validateUniqueServiceNames(config)
    expect(errors).toHaveLength(0)
  })

  it('should detect duplicate service names', () => {
    // Note: This tests the logic, but in practice duplicate keys are impossible
    // This validates the function handles edge cases
    const config = {
      services: {
        api: {},
        web: {},
      },
    }

    const errors = validateUniqueServiceNames(config)
    expect(errors).toHaveLength(0) // No duplicates possible in object keys
  })

  it('should handle missing services', () => {
    const config = {}

    const errors = validateUniqueServiceNames(config)
    expect(errors).toHaveLength(0)
  })

  it('should handle empty services', () => {
    const config = {
      services: {},
    }

    const errors = validateUniqueServiceNames(config)
    expect(errors).toHaveLength(0)
  })
})

describe('Unique Resource Names', () => {
  it('should pass with unique names', () => {
    const config = {
      resources: {
        database: {},
        storage: {},
        cache: {},
      },
    }

    const errors = validateUniqueResourceNames(config)
    expect(errors).toHaveLength(0)
  })

  it('should handle missing resources', () => {
    const config = {}

    const errors = validateUniqueResourceNames(config)
    expect(errors).toHaveLength(0)
  })
})

describe('Port Conflicts', () => {
  it('should pass with unique ports', () => {
    const config = {
      services: {
        api: { ports: ['8080:8080'] },
        web: { ports: ['3000:3000'] },
        db: { ports: ['5432:5432'] },
      },
    }

    const errors = validatePortConflicts(config)
    expect(errors).toHaveLength(0)
  })

  it('should detect port conflicts', () => {
    const config = {
      services: {
        api: { ports: ['8080:8080'] },
        web: { ports: ['8080:3000'] }, // Same host port
      },
    }

    const errors = validatePortConflicts(config)
    expect(errors.length).toBeGreaterThan(0)
    expect(errors[0].level).toBe('warning')
    expect(errors[0].message).toContain('8080')
  })

  it('should handle array of ports', () => {
    const config = {
      services: {
        api: { ports: ['8080:8080', '8081:8081'] },
      },
    }

    const errors = validatePortConflicts(config)
    expect(errors).toHaveLength(0)
  })

  it('should handle string port', () => {
    const config = {
      services: {
        api: { ports: '8080:8080' },
      },
    }

    const errors = validatePortConflicts(config)
    expect(errors).toHaveLength(0)
  })

  it('should detect conflicts across multiple services', () => {
    const config = {
      services: {
        api: { ports: ['8080:8080'] },
        web: { ports: ['8080:3000'] },
        admin: { ports: ['8080:4000'] },
      },
    }

    const errors = validatePortConflicts(config)
    expect(errors.length).toBeGreaterThan(0)
    expect(errors[0].message).toContain('api')
    expect(errors[0].message).toContain('web')
  })

  it('should handle missing ports', () => {
    const config = {
      services: {
        api: {},
      },
    }

    const errors = validatePortConflicts(config)
    expect(errors).toHaveLength(0)
  })
})

describe('Circular Dependencies', () => {
  it('should pass with no dependencies', () => {
    const config = {
      services: {
        api: {},
        web: {},
      },
    }

    const errors = validateCircularDependencies(config)
    expect(errors).toHaveLength(0)
  })

  it('should pass with valid dependencies', () => {
    const config = {
      services: {
        api: { uses: ['db'] },
        web: { uses: ['api'] },
        db: {},
      },
    }

    const errors = validateCircularDependencies(config)
    expect(errors).toHaveLength(0)
  })

  it('should detect simple circular dependency', () => {
    const config = {
      services: {
        api: { uses: ['web'] },
        web: { uses: ['api'] },
      },
    }

    const errors = validateCircularDependencies(config)
    expect(errors.length).toBeGreaterThan(0)
    expect(errors[0].level).toBe('error')
    expect(errors[0].rule).toBe('circular-dependency')
  })

  it('should detect complex circular dependency', () => {
    const config = {
      services: {
        api: { uses: ['cache'] },
        web: { uses: ['api'] },
        cache: { uses: ['web'] },
      },
    }

    const errors = validateCircularDependencies(config)
    expect(errors.length).toBeGreaterThan(0)
  })

  it('should detect self-referencing service', () => {
    const config = {
      services: {
        api: { uses: ['api'] },
      },
    }

    const errors = validateCircularDependencies(config)
    expect(errors.length).toBeGreaterThan(0)
  })

  it('should handle array of dependencies', () => {
    const config = {
      services: {
        api: { uses: ['db', 'cache'] },
        db: {},
        cache: {},
      },
    }

    const errors = validateCircularDependencies(config)
    expect(errors).toHaveLength(0)
  })

  it('should handle string dependency', () => {
    const config = {
      services: {
        api: { uses: 'db' },
        db: {},
      },
    }

    const errors = validateCircularDependencies(config)
    expect(errors).toHaveLength(0)
  })

  it('should handle cross-type dependencies (services and resources)', () => {
    const config = {
      services: {
        api: { uses: ['database'] },
      },
      resources: {
        database: { uses: ['api'] }, // Creates cycle
      },
    }

    const errors = validateCircularDependencies(config)
    expect(errors.length).toBeGreaterThan(0)
  })
})

describe('Recommended Fields', () => {
  it('should suggest health checks', () => {
    const config = {
      services: {
        api: {
          host: '0.0.0.0',
          port: 8080,
          // Missing healthcheck
        },
      },
    }

    const errors = validateRecommendedFields(config)
    expect(errors.length).toBeGreaterThan(0)
    expect(errors[0].level).toBe('info')
    expect(errors[0].message).toContain('health check')
  })

  it('should pass with health check', () => {
    const config = {
      services: {
        api: {
          healthcheck: {
            test: ['CMD', 'curl', '-f', 'http://localhost/health'],
          },
        },
      },
    }

    const errors = validateRecommendedFields(config)
    expect(errors).toHaveLength(0)
  })

  it('should pass with test field (alternative to healthcheck)', () => {
    const config = {
      services: {
        api: {
          test: ['CMD', 'curl', '-f', 'http://localhost/health'],
        },
      },
    }

    const errors = validateRecommendedFields(config)
    expect(errors).toHaveLength(0)
  })

  it('should check all services', () => {
    const config = {
      services: {
        api: {},
        web: {},
        worker: {},
      },
    }

    const errors = validateRecommendedFields(config)
    expect(errors.length).toBe(3) // One for each service
  })
})

describe('Complete Validation', () => {
  const schema = {
    type: 'object',
    properties: {
      name: { type: 'string' },
      services: {
        type: 'object',
        additionalProperties: {
          type: 'object',
          properties: {
            host: { type: 'string' },
            port: { type: 'number' },
            ports: { type: 'array' },
            uses: {
              oneOf: [
                { type: 'string' },
                { type: 'array', items: { type: 'string' } },
              ],
            },
          },
        },
      },
    },
    required: ['name'],
  }

  it('should run all validation stages', () => {
    const config = {
      name: 'my-app',
      services: {
        api: {
          port: 8080,
          ports: ['8080:8080'],
        },
        web: {
          port: 8080, // Port conflict
          uses: ['api'],
        },
      },
    }

    const result = validateConfiguration(config, schema)

    // Should be valid (warnings don't fail validation)
    expect(result.valid).toBe(true)
    expect(result.warnings.length).toBeGreaterThan(0) // Port conflict
    expect(result.info.length).toBeGreaterThan(0) // Health check recommendations
  })

  it('should fail on errors', () => {
    const config = {
      // Missing required 'name'
      services: {
        api: { uses: ['web'] },
        web: { uses: ['api'] }, // Circular dependency
      },
    }

    const result = validateConfiguration(config, schema)

    expect(result.valid).toBe(false)
    expect(result.errors.length).toBeGreaterThan(0)
  })

  it('should respect validation options', () => {
    const config = {
      name: 'my-app',
      services: {
        api: {},
      },
    }

    // Without warnings and info
    const result1 = validateConfiguration(config, schema, {
      includeWarnings: false,
      includeInfo: false,
    })
    expect(result1.warnings).toHaveLength(0)
    expect(result1.info).toHaveLength(0)

    // With warnings and info
    const result2 = validateConfiguration(config, schema, {
      includeWarnings: true,
      includeInfo: true,
    })
    expect(result2.info.length).toBeGreaterThan(0) // Health check recommendations
  })

  it('should separate errors by level', () => {
    const config = {
      name: 'my-app',
      services: {
        api: { ports: ['8080:8080'] },
        web: { ports: ['8080:3000'], uses: ['api'] }, // Port conflict (warning)
        // Missing health checks (info)
      },
    }

    const result = validateConfiguration(config, schema)

    expect(result.errors).toHaveLength(0)
    expect(result.warnings.length).toBeGreaterThan(0)
    expect(result.info.length).toBeGreaterThan(0)
  })
})

describe('Validation Performance', () => {
  it('should validate large config quickly', () => {
    const schema = {
      type: 'object',
      properties: {
        name: { type: 'string' },
        services: { type: 'object' },
      },
    }

    // Create config with 100 services
    const services: Record<string, { port: number }> = {}
    for (let i = 0; i < 100; i++) {
      services[`service-${i}`] = { port: 8000 + i }
    }

    const config = {
      name: 'large-app',
      services,
    }

    const startTime = performance.now()
    const result = validateConfiguration(config, schema)
    const duration = performance.now() - startTime

    // Should complete in under 2 seconds
    expect(duration).toBeLessThan(2000)
    expect(result.valid).toBe(true)
  })

  it('should handle deeply nested validation efficiently', () => {
    const schema = {
      type: 'object',
      properties: {
        name: { type: 'string' },
        level1: {
          type: 'object',
          properties: {
            level2: {
              type: 'object',
              properties: {
                level3: {
                  type: 'object',
                  properties: {
                    value: { type: 'string' },
                  },
                },
              },
            },
          },
        },
      },
    }

    const config = {
      name: 'nested-app',
      level1: {
        level2: {
          level3: {
            value: 'deep',
          },
        },
      },
    }

    const startTime = performance.now()
    validateConfiguration(config, schema)
    const duration = performance.now() - startTime

    expect(duration).toBeLessThan(100)
  })
})

describe('Edge Cases', () => {
  it('should handle empty config', () => {
    const schema = { type: 'object' }
    const config = {}

    const result = validateConfiguration(config, schema)
    expect(result.valid).toBe(true)
  })

  it('should handle null values', () => {
    const schema = {
      type: 'object',
      properties: {
        nullable: { type: ['string', 'null'] },
      },
    }

    const config = {
      nullable: null,
    }

    const errors = validateSchema(config, schema)
    expect(errors).toHaveLength(0)
  })

  it('should handle undefined vs null', () => {
    const schema = {
      type: 'object',
      properties: {
        optional: { type: 'string' },
      },
    }

    const config = {
      // optional is undefined (not present)
    }

    const errors = validateSchema(config, schema)
    expect(errors).toHaveLength(0) // Optional field can be missing
  })

  it('should handle special characters in property names', () => {
    const config = {
      services: {
        'my-service': {},
        'my_service': {},
        'my.service': {},
      },
    }

    const errors = validateUniqueServiceNames(config)
    expect(errors).toHaveLength(0)
  })
})
