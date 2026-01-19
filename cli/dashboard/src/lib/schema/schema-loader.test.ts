/**
 * Tests for schema-loader.ts
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { loadSchema, getBundledSchema } from './schema-loader'

describe('schema-loader', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('loadSchema', () => {
    it('should load bundled schema by default', async () => {
      const result = await loadSchema()

      expect(result.success).toBe(true)
      expect(result.source).toBe('bundled')
      expect(result.schema).toBeDefined()
      expect(result.schema?.$schema).toBe('http://json-schema.org/draft-07/schema#')
    })
  })

  describe('getBundledSchema', () => {
    it('should return bundled schema synchronously', () => {
      const schema = getBundledSchema()

      expect(schema).toBeDefined()
      expect(schema.$schema).toBe('http://json-schema.org/draft-07/schema#')
      expect(schema.title).toBeDefined()
      expect(schema.properties).toBeDefined()
    })

    it('should return valid azure.yaml schema', () => {
      const schema = getBundledSchema()

      expect(schema.properties).toHaveProperty('name')
      expect(schema.properties).toHaveProperty('services')
      expect(schema.required).toContain('name')
    })
  })
})
