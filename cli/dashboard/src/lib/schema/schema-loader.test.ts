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
    it('should load schema from remote URL successfully', async () => {
      const mockSchema = {
        $schema: 'http://json-schema.org/draft-07/schema#',
        title: 'Test Schema',
        properties: {
          name: { type: 'string' },
        },
      }

      global.fetch = vi.fn().mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(mockSchema),
      } as Response)

      const result = await loadSchema()

      expect(result.success).toBe(true)
      expect(result.source).toBe('remote')
      expect(result.schema).toEqual(mockSchema)
      expect(result.error).toBeUndefined()
    })

    it('should fallback to bundled schema on network error', async () => {
      global.fetch = vi.fn().mockRejectedValueOnce(new Error('Network error'))

      const result = await loadSchema()

      expect(result.success).toBe(true)
      expect(result.source).toBe('bundled')
      expect(result.schema).toBeDefined()
      expect(result.schema?.$schema).toBe('http://json-schema.org/draft-07/schema#')
      expect(result.error).toBe('Network error')
    })

    it('should fallback to bundled schema on HTTP error', async () => {
      global.fetch = vi.fn().mockResolvedValueOnce({
        ok: false,
        status: 404,
        statusText: 'Not Found',
      } as Response)

      const result = await loadSchema()

      expect(result.success).toBe(true)
      expect(result.source).toBe('bundled')
      expect(result.schema).toBeDefined()
      expect(result.error).toContain('404')
    })

    it('should fallback to bundled schema on invalid JSON', async () => {
      global.fetch = vi.fn().mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.reject(new Error('Invalid JSON')),
      } as unknown as Response)

      const result = await loadSchema()

      expect(result.success).toBe(true)
      expect(result.source).toBe('bundled')
      expect(result.schema).toBeDefined()
      expect(result.error).toBe('Invalid JSON')
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
