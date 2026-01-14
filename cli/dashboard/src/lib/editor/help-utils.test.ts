/**
 * Help Utilities Tests
 */

import { describe, it, expect } from 'vitest'
import {
  getTooltipText,
  formatValidationRules,
  detectSection,
  detectSubsection,
  getHelpSectionForField,
  formatExampleValue,
  hasComplexValidation,
} from './help-utils'
import type { SchemaProperty } from '@/lib/schema'

describe('getTooltipText', () => {
  it('returns description when available', () => {
    const property: SchemaProperty = {
      name: 'test',
      type: 'string',
      description: 'Test description',
      title: 'Test Title',
      required: false,
      validation: [],
    }
    expect(getTooltipText(property)).toBe('Test description')
  })

  it('falls back to title when description is not available', () => {
    const property: SchemaProperty = {
      name: 'test',
      type: 'string',
      title: 'Test Title',
      required: false,
      validation: [],
    }
    expect(getTooltipText(property)).toBe('Test Title')
  })

  it('returns undefined when neither is available', () => {
    const property: SchemaProperty = {
      name: 'test',
      type: 'string',
      required: false,
      validation: [],
    }
    expect(getTooltipText(property)).toBeUndefined()
  })
})

describe('formatValidationRules', () => {
  it('formats required field', () => {
    const property: SchemaProperty = {
      name: 'test',
      type: 'string',
      required: true,
      validation: [],
    }
    const rules = formatValidationRules(property)
    expect(rules).toContain('Required field')
  })

  it('formats pattern validation', () => {
    const property: SchemaProperty = {
      name: 'test',
      type: 'string',
      pattern: '^[a-z]+$',
      required: false,
      validation: [],
    }
    const rules = formatValidationRules(property)
    expect(rules).toContain('Must match pattern: ^[a-z]+$')
  })

  it('formats length constraints', () => {
    const property: SchemaProperty = {
      name: 'test',
      type: 'string',
      minLength: 2,
      maxLength: 10,
      required: false,
      validation: [],
    }
    const rules = formatValidationRules(property)
    expect(rules).toContain('Minimum length: 2')
    expect(rules).toContain('Maximum length: 10')
  })

  it('formats number constraints', () => {
    const property: SchemaProperty = {
      name: 'test',
      type: 'number',
      minimum: 0,
      maximum: 100,
      required: false,
      validation: [],
    }
    const rules = formatValidationRules(property)
    expect(rules).toContain('Minimum value: 0')
    expect(rules).toContain('Maximum value: 100')
  })

  it('formats enum values', () => {
    const property: SchemaProperty = {
      name: 'test',
      type: 'enum',
      enumValues: ['option1', 'option2', 'option3'],
      required: false,
      validation: [],
    }
    const rules = formatValidationRules(property)
    expect(rules).toContain('Allowed values: option1, option2, option3')
  })
})

describe('detectSection', () => {
  it('detects top-level section', () => {
    expect(detectSection('services')).toBe('services')
    expect(detectSection('resources')).toBe('resources')
    expect(detectSection('hooks')).toBe('hooks')
  })

  it('detects section from nested path', () => {
    expect(detectSection('services.api.port')).toBe('services')
    expect(detectSection('hooks.postprovision.run')).toBe('hooks')
  })

  it('returns undefined for empty path', () => {
    expect(detectSection('')).toBeUndefined()
  })
})

describe('detectSubsection', () => {
  it('detects subsection from nested path', () => {
    expect(detectSubsection('services.api.ports')).toBe('ports')
    expect(detectSubsection('services.api.environment')).toBe('environment')
    expect(detectSubsection('services.api.healthcheck')).toBe('healthcheck')
  })

  it('returns undefined for shallow paths', () => {
    expect(detectSubsection('services')).toBeUndefined()
    expect(detectSubsection('services.api')).toBeUndefined()
  })

  it('handles deeply nested paths', () => {
    expect(detectSubsection('services.api.healthcheck.interval')).toBe('healthcheck')
  })
})

describe('getHelpSectionForField', () => {
  it('returns subsection when available', () => {
    expect(getHelpSectionForField('services.api.ports')).toBe('ports')
  })

  it('falls back to section when subsection not available', () => {
    expect(getHelpSectionForField('services.api')).toBe('services')
  })
})

describe('formatExampleValue', () => {
  it('uses default value when available', () => {
    const property: SchemaProperty = {
      name: 'test',
      type: 'string',
      defaultValue: 'default-value',
      required: false,
      validation: [],
    }
    expect(formatExampleValue(property)).toBe('default-value')
  })

  it('uses first enum value for enum type', () => {
    const property: SchemaProperty = {
      name: 'test',
      type: 'enum',
      enumValues: ['option1', 'option2'],
      required: false,
      validation: [],
    }
    expect(formatExampleValue(property)).toBe('option1')
  })

  it('uses minimum for number type', () => {
    const property: SchemaProperty = {
      name: 'test',
      type: 'number',
      minimum: 10,
      required: false,
      validation: [],
    }
    expect(formatExampleValue(property)).toBe('10')
  })

  it('returns type-appropriate defaults', () => {
    const boolProp: SchemaProperty = {
      name: 'test',
      type: 'boolean',
      required: false,
      validation: [],
    }
    expect(formatExampleValue(boolProp)).toBe('true')

    const arrayProp: SchemaProperty = {
      name: 'test',
      type: 'array',
      required: false,
      validation: [],
    }
    expect(formatExampleValue(arrayProp)).toBe('[]')
  })
})

describe('hasComplexValidation', () => {
  it('returns true for pattern validation', () => {
    const property: SchemaProperty = {
      name: 'test',
      type: 'string',
      pattern: '^[a-z]+$',
      required: false,
      validation: [],
    }
    expect(hasComplexValidation(property)).toBe(true)
  })

  it('returns true for length constraints', () => {
    const property: SchemaProperty = {
      name: 'test',
      type: 'string',
      minLength: 2,
      maxLength: 10,
      required: false,
      validation: [],
    }
    expect(hasComplexValidation(property)).toBe(true)
  })

  it('returns true for many enum values', () => {
    const property: SchemaProperty = {
      name: 'test',
      type: 'enum',
      enumValues: ['a', 'b', 'c', 'd', 'e', 'f'],
      required: false,
      validation: [],
    }
    expect(hasComplexValidation(property)).toBe(true)
  })

  it('returns true for complex types', () => {
    const objectProp: SchemaProperty = {
      name: 'test',
      type: 'object',
      required: false,
      validation: [],
    }
    expect(hasComplexValidation(objectProp)).toBe(true)

    const arrayProp: SchemaProperty = {
      name: 'test',
      type: 'array',
      required: false,
      validation: [],
    }
    expect(hasComplexValidation(arrayProp)).toBe(true)
  })

  it('returns false for simple validation', () => {
    const property: SchemaProperty = {
      name: 'test',
      type: 'string',
      required: true,
      validation: [],
    }
    expect(hasComplexValidation(property)).toBe(false)
  })
})

