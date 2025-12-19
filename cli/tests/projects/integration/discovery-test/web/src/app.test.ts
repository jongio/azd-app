import { describe, it, expect } from 'vitest'

describe('App', () => {
  it('should pass a basic test', () => {
    expect(1 + 1).toBe(2)
  })

  it('should handle strings', () => {
    expect('hello').toContain('ell')
  })
})
