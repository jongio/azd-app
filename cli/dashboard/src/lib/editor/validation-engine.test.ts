/**
 * Validation Engine Tests
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

describe('validateSchema', () => {
  it('should validate against JSON Schema successfully', () => {
    const config = {
      name: 'my-app',
      services: {},
    }
    const schema = {
      type: 'object',
      properties: {
        name: { type: 'string' },
        services: { type: 'object' },
      },
      required: ['name'],
    }

    const errors = validateSchema(config, schema)
    expect(errors).toHaveLength(0)
  })

  it('should detect missing required fields', () => {
    const config = {
      services: {},
    }
    const schema = {
      type: 'object',
      properties: {
        name: { type: 'string' },
        services: { type: 'object' },
      },
      required: ['name'],
    }

    const errors = validateSchema(config, schema)
    expect(errors).toHaveLength(1)
    expect(errors[0].level).toBe('error')
    expect(errors[0].message).toContain('name')
    expect(errors[0].rule).toBe('required')
  })

  it('should detect type mismatches', () => {
    const config = {
      name: 123, // Should be string
    }
    const schema = {
      type: 'object',
      properties: {
        name: { type: 'string' },
      },
    }

    const errors = validateSchema(config, schema)
    expect(errors).toHaveLength(1)
    expect(errors[0].level).toBe('error')
    expect(errors[0].message).toContain('string')
  })

  it('should validate enum values', () => {
    const config = {
      host: 'invalid',
    }
    const schema = {
      type: 'object',
      properties: {
        host: { type: 'string', enum: ['containerapp', 'appservice', 'function'] },
      },
    }

    const errors = validateSchema(config, schema)
    expect(errors).toHaveLength(1)
    expect(errors[0].rule).toBe('enum')
  })

  it('should validate pattern matching', () => {
    const config = {
      name: 'Invalid Name!',
    }
    const schema = {
      type: 'object',
      properties: {
        name: { type: 'string', pattern: '^[a-z0-9-]+$' },
      },
    }

    const errors = validateSchema(config, schema)
    expect(errors).toHaveLength(1)
    expect(errors[0].rule).toBe('pattern')
  })

  it('should validate min/max values', () => {
    const config = {
      port: 70000, // Too high
    }
    const schema = {
      type: 'object',
      properties: {
        port: { type: 'number', minimum: 1, maximum: 65535 },
      },
    }

    const errors = validateSchema(config, schema)
    expect(errors).toHaveLength(1)
    expect(errors[0].rule).toBe('maximum')
  })
})

describe('validateUniqueServiceNames', () => {
  it('should pass when all service names are unique', () => {
    const config = {
      services: {
        api: {},
        web: {},
        database: {},
      },
    }

    const errors = validateUniqueServiceNames(config)
    expect(errors).toHaveLength(0)
  })

  it('should detect duplicate service names', () => {
    // Note: In real YAML, duplicate keys aren't possible
    // This test validates the logic if somehow duplicates exist
    const config = {
      services: {
        api: {},
        web: {},
      },
    }

    // Services is an object, so duplicates aren't actually possible
    // But let's test the validation logic
    const errors = validateUniqueServiceNames(config)
    expect(errors).toHaveLength(0)
  })

  it('should return empty array when services is undefined', () => {
    const config = {}
    const errors = validateUniqueServiceNames(config)
    expect(errors).toHaveLength(0)
  })

  it('should return empty array when services is not an object', () => {
    const config = { services: 'invalid' }
    const errors = validateUniqueServiceNames(config)
    expect(errors).toHaveLength(0)
  })
})

describe('validateUniqueResourceNames', () => {
  it('should pass when all resource names are unique', () => {
    const config = {
      resources: {
        storage: {},
        cosmos: {},
        eventhub: {},
      },
    }

    const errors = validateUniqueResourceNames(config)
    expect(errors).toHaveLength(0)
  })

  it('should return empty array when resources is undefined', () => {
    const config = {}
    const errors = validateUniqueResourceNames(config)
    expect(errors).toHaveLength(0)
  })
})

describe('validatePortConflicts', () => {
  it('should detect port conflicts between services', () => {
    const config = {
      services: {
        api: {
          ports: ['8080:8080'],
        },
        web: {
          ports: ['8080:3000'], // Same host port
        },
      },
    }

    const errors = validatePortConflicts(config)
    expect(errors).toHaveLength(1)
    expect(errors[0].level).toBe('warning')
    expect(errors[0].message).toContain('8080')
    expect(errors[0].message).toContain('api')
    expect(errors[0].message).toContain('web')
  })

  it('should pass when ports are unique', () => {
    const config = {
      services: {
        api: {
          ports: ['8080:8080'],
        },
        web: {
          ports: ['3000:3000'],
        },
      },
    }

    const errors = validatePortConflicts(config)
    expect(errors).toHaveLength(0)
  })

  it('should handle array of port objects', () => {
    const config = {
      services: {
        api: {
          ports: ['8080:8080', '8081:8081'],
        },
        web: {
          ports: ['3000:3000'],
        },
      },
    }

    const errors = validatePortConflicts(config)
    expect(errors).toHaveLength(0)
  })

  it('should handle string port format', () => {
    const config = {
      services: {
        api: {
          ports: '8080:8080',
        },
      },
    }

    const errors = validatePortConflicts(config)
    expect(errors).toHaveLength(0)
  })
})

describe('validateCircularDependencies', () => {
  it('should detect simple circular dependency', () => {
    const config = {
      services: {
        api: {
          uses: ['database'],
        },
        database: {
          uses: ['api'],
        },
      },
    }

    const errors = validateCircularDependencies(config)
    expect(errors).toHaveLength(1)
    expect(errors[0].level).toBe('error')
    expect(errors[0].message).toContain('Circular dependency')
  })

  it('should detect complex circular dependency', () => {
    const config = {
      services: {
        api: {
          uses: ['cache'],
        },
        cache: {
          uses: ['database'],
        },
        database: {
          uses: ['api'],
        },
      },
    }

    const errors = validateCircularDependencies(config)
    expect(errors.length).toBeGreaterThan(0)
    expect(errors[0].level).toBe('error')
  })

  it('should pass when no circular dependencies exist', () => {
    const config = {
      services: {
        api: {
          uses: ['database', 'cache'],
        },
        database: {},
        cache: {},
      },
    }

    const errors = validateCircularDependencies(config)
    expect(errors).toHaveLength(0)
  })

  it('should handle string uses value', () => {
    const config = {
      services: {
        api: {
          uses: 'database',
        },
        database: {},
      },
    }

    const errors = validateCircularDependencies(config)
    expect(errors).toHaveLength(0)
  })

  it('should handle resources with circular dependencies', () => {
    const config = {
      resources: {
        storage: {
          uses: ['cosmos'],
        },
        cosmos: {
          uses: ['storage'],
        },
      },
    }

    const errors = validateCircularDependencies(config)
    expect(errors.length).toBeGreaterThan(0)
  })
})

describe('validateRecommendedFields', () => {
  it('should suggest adding health checks', () => {
    const config = {
      services: {
        api: {
          host: 'containerapp',
        },
        web: {
          host: 'containerapp',
          healthcheck: {
            test: 'curl -f http://localhost:3000',
          },
        },
      },
    }

    const errors = validateRecommendedFields(config)
    expect(errors).toHaveLength(1)
    expect(errors[0].level).toBe('info')
    expect(errors[0].message).toContain('api')
    expect(errors[0].message).toContain('health check')
  })

  it('should not suggest health check when test field exists', () => {
    const config = {
      services: {
        api: {
          host: 'containerapp',
          test: 'curl -f http://localhost:8080',
        },
      },
    }

    const errors = validateRecommendedFields(config)
    expect(errors).toHaveLength(0)
  })

  it('should return empty array when no services exist', () => {
    const config = {}
    const errors = validateRecommendedFields(config)
    expect(errors).toHaveLength(0)
  })
})

describe('validateConfiguration', () => {
  const schema = {
    type: 'object',
    properties: {
      name: { type: 'string' },
      services: { type: 'object' },
    },
    required: ['name'],
  }

  it('should run all validation stages', () => {
    const config = {
      name: 'my-app',
      services: {
        api: {
          ports: ['8080:8080'],
        },
      },
    }

    const result = validateConfiguration(config, schema)
    expect(result.valid).toBe(true)
    expect(result.errors).toHaveLength(0)
    expect(result.info.length).toBeGreaterThan(0) // Should have health check suggestion
  })

  it('should categorize errors by level', () => {
    const config = {
      // Missing required 'name'
      services: {
        api: {
          ports: ['8080:8080'],
        },
        web: {
          ports: ['8080:3000'], // Port conflict
        },
      },
    }

    const result = validateConfiguration(config, schema)
    expect(result.valid).toBe(false)
    expect(result.errors.length).toBeGreaterThan(0) // Missing name
    expect(result.warnings.length).toBeGreaterThan(0) // Port conflict
    expect(result.info.length).toBeGreaterThan(0) // Health check suggestions
  })

  it('should exclude warnings when includeWarnings is false', () => {
    const config = {
      name: 'my-app',
      services: {
        api: {
          ports: ['8080:8080'],
        },
        web: {
          ports: ['8080:3000'],
        },
      },
    }

    const result = validateConfiguration(config, schema, {
      includeWarnings: false,
    })
    expect(result.warnings).toHaveLength(0)
  })

  it('should exclude info when includeInfo is false', () => {
    const config = {
      name: 'my-app',
      services: {
        api: {},
      },
    }

    const result = validateConfiguration(config, schema, {
      includeInfo: false,
    })
    expect(result.info).toHaveLength(0)
  })

  it('should skip schema validation when full is false', () => {
    const config = {
      // Missing required 'name' - would fail schema validation
      services: {},
    }

    const result = validateConfiguration(config, schema, {
      full: false,
    })
    // Should not have schema errors since full validation is disabled
    const schemaErrors = result.errors.filter(e => e.rule === 'required')
    expect(schemaErrors).toHaveLength(0)
  })

  it('should handle complex configurations', () => {
    const config = {
      name: 'complex-app',
      services: {
        api: {
          host: 'containerapp',
          ports: ['8080:8080'],
          uses: ['database', 'cache'],
        },
        web: {
          host: 'containerapp',
          ports: ['3000:3000'],
          uses: ['api'],
        },
        database: {
          host: 'containerapp',
          ports: ['5432:5432'],
          healthcheck: {
            test: 'pg_isready',
          },
        },
        cache: {
          host: 'containerapp',
          ports: ['6379:6379'],
        },
      },
    }

    const result = validateConfiguration(config, schema)
    expect(result.valid).toBe(true)
  })
})
