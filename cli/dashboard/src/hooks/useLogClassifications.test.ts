import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { useLogClassifications } from '@/hooks/useLogClassifications'

// Helper to create mock fetch responses
const createMockFetchResponse = <T>(data: T, ok = true, status = 200) => {
  return Promise.resolve({
    ok,
    status,
    json: () => Promise.resolve(data),
    text: () => Promise.resolve(JSON.stringify(data)),
  } as Response)
}

describe('useLogClassifications', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('initial load', () => {
    it('should fetch classifications on mount', async () => {
      const mockClassifications = [
        { text: 'Connection refused', level: 'error' },
        { text: 'cache miss', level: 'info' },
      ]
      const mockFetch = vi.fn(() => 
        createMockFetchResponse({ classifications: mockClassifications })
      )
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useLogClassifications())

      expect(result.current.isLoading).toBe(true)

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(mockFetch).toHaveBeenCalledWith('/api/logs/classifications')
      expect(result.current.classifications).toEqual(mockClassifications)
      expect(result.current.error).toBeNull()
    })

    it('should handle empty classifications array', async () => {
      const mockFetch = vi.fn(() => 
        createMockFetchResponse({ classifications: [] })
      )
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useLogClassifications())

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.classifications).toEqual([])
      expect(result.current.error).toBeNull()
    })

    it('should handle fetch errors', async () => {
      const mockFetch = vi.fn(() => Promise.reject(new Error('Network error')))
      globalThis.fetch = mockFetch as unknown as typeof fetch
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      const { result } = renderHook(() => useLogClassifications())

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.classifications).toEqual([])
      expect(result.current.error).toBeInstanceOf(Error)
      expect(result.current.error?.message).toBe('Network error')

      consoleSpy.mockRestore()
    })

    it('should handle HTTP error responses', async () => {
      const mockFetch = vi.fn(() => createMockFetchResponse({}, false, 500))
      globalThis.fetch = mockFetch as unknown as typeof fetch
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      const { result } = renderHook(() => useLogClassifications())

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.classifications).toEqual([])
      expect(result.current.error).toBeInstanceOf(Error)

      consoleSpy.mockRestore()
    })
  })

  describe('addClassification', () => {
    it('should add a classification and reload', async () => {
      const initialClassifications = [{ text: 'existing', level: 'info' as const }]
      const afterAddClassifications = [
        { text: 'existing', level: 'info' as const },
        { text: 'new error', level: 'error' as const },
      ]

      let callCount = 0
      const mockFetch = vi.fn((_url: string, options?: RequestInit) => {
        if (options?.method === 'POST') {
          return createMockFetchResponse({ text: 'new error', level: 'error' }, true, 201)
        }
        // GET requests - return different data after add
        callCount++
        if (callCount === 1) {
          return createMockFetchResponse({ classifications: initialClassifications })
        }
        return createMockFetchResponse({ classifications: afterAddClassifications })
      })
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useLogClassifications())

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.classifications).toEqual(initialClassifications)

      await act(async () => {
        await result.current.addClassification('new error', 'error')
      })

      // Verify POST was called
      expect(mockFetch).toHaveBeenCalledWith('/api/logs/classifications', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ text: 'new error', level: 'error' }),
      })

      // Verify reload happened
      await waitFor(() => {
        expect(result.current.classifications).toEqual(afterAddClassifications)
      })
    })

    it('should skip reload when skipNotify is true', async () => {
      let getCallCount = 0
      const mockFetch = vi.fn((_url: string, options?: RequestInit) => {
        if (options?.method === 'POST') {
          return createMockFetchResponse({ text: 'new', level: 'error' }, true, 201)
        }
        // GET requests
        getCallCount++
        return createMockFetchResponse({ classifications: [] })
      })
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useLogClassifications())

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      const initialGetCount = getCallCount

      await act(async () => {
        await result.current.addClassification('new', 'error', true) // skipNotify = true
      })

      // Verify POST was called
      expect(mockFetch).toHaveBeenCalledWith('/api/logs/classifications', expect.objectContaining({
        method: 'POST',
      }))

      // Verify NO additional GET requests were made (no reload)
      expect(getCallCount).toBe(initialGetCount)
    })

    it('should throw on add error', async () => {
      const mockFetch = vi.fn((_url: string, options?: RequestInit) => {
        if (options?.method === 'POST') {
          return createMockFetchResponse('Invalid level', false, 400)
        }
        return createMockFetchResponse({ classifications: [] })
      })
      globalThis.fetch = mockFetch as unknown as typeof fetch
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      const { result } = renderHook(() => useLogClassifications())

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      await expect(
        act(async () => {
          await result.current.addClassification('test', 'error')
        })
      ).rejects.toThrow()

      consoleSpy.mockRestore()
    })
  })

  describe('deleteClassification', () => {
    it('should delete a classification and reload', async () => {
      const initialClassifications = [
        { text: 'first', level: 'info' as const },
        { text: 'second', level: 'error' as const },
      ]
      const afterDeleteClassifications = [{ text: 'second', level: 'error' as const }]

      let getCallCount = 0
      const mockFetch = vi.fn((_url: string, options?: RequestInit) => {
        if (options?.method === 'DELETE') {
          return createMockFetchResponse(null, true, 204)
        }
        // GET requests
        getCallCount++
        if (getCallCount === 1) {
          return createMockFetchResponse({ classifications: initialClassifications })
        }
        return createMockFetchResponse({ classifications: afterDeleteClassifications })
      })
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useLogClassifications())

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.classifications).toEqual(initialClassifications)

      await act(async () => {
        await result.current.deleteClassification(0)
      })

      // Verify DELETE was called with correct index
      expect(mockFetch).toHaveBeenCalledWith('/api/logs/classifications/0', {
        method: 'DELETE',
      })

      // Verify reload happened
      await waitFor(() => {
        expect(result.current.classifications).toEqual(afterDeleteClassifications)
      })
    })

    it('should throw on delete error', async () => {
      const mockFetch = vi.fn((_url: string, options?: RequestInit) => {
        if (options?.method === 'DELETE') {
          return createMockFetchResponse({}, false, 404)
        }
        return createMockFetchResponse({ classifications: [{ text: 'test', level: 'info' }] })
      })
      globalThis.fetch = mockFetch as unknown as typeof fetch
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      const { result } = renderHook(() => useLogClassifications())

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      await expect(
        act(async () => {
          await result.current.deleteClassification(99)
        })
      ).rejects.toThrow()

      consoleSpy.mockRestore()
    })

    it('should skip reload when skipNotify is true', async () => {
      const initialClassifications = [
        { text: 'first', level: 'info' as const },
        { text: 'second', level: 'error' as const },
      ]

      let getCallCount = 0
      const mockFetch = vi.fn((_url: string, options?: RequestInit) => {
        if (options?.method === 'DELETE') {
          return createMockFetchResponse(null, true, 204)
        }
        // GET requests
        getCallCount++
        return createMockFetchResponse({ classifications: initialClassifications })
      })
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useLogClassifications())

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      const initialGetCount = getCallCount

      await act(async () => {
        await result.current.deleteClassification(0, true) // skipNotify = true
      })

      // Verify DELETE was called
      expect(mockFetch).toHaveBeenCalledWith('/api/logs/classifications/0', {
        method: 'DELETE',
      })

      // Verify NO additional GET requests were made (no reload)
      expect(getCallCount).toBe(initialGetCount)
    })

    it('should handle batch deletes in reverse order correctly', async () => {
      // This simulates what SettingsModal does: delete indices [4, 2, 0] in reverse order
      const mockFetch = vi.fn((_url: string, options?: RequestInit) => {
        if (options?.method === 'DELETE') {
          return createMockFetchResponse(null, true, 204)
        }
        return createMockFetchResponse({ 
          classifications: [
            { text: 'item0', level: 'info' },
            { text: 'item1', level: 'info' },
            { text: 'item2', level: 'info' },
            { text: 'item3', level: 'info' },
            { text: 'item4', level: 'info' },
          ] 
        })
      })
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useLogClassifications())

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      // Simulate batch delete: delete indices 4, 2, 0 (reverse order)
      await act(async () => {
        await result.current.deleteClassification(4, true)
        await result.current.deleteClassification(2, true)
        await result.current.deleteClassification(0, true)
      })

      // Verify all deletes were called with correct indices
      expect(mockFetch).toHaveBeenCalledWith('/api/logs/classifications/4', { method: 'DELETE' })
      expect(mockFetch).toHaveBeenCalledWith('/api/logs/classifications/2', { method: 'DELETE' })
      expect(mockFetch).toHaveBeenCalledWith('/api/logs/classifications/0', { method: 'DELETE' })
    })
  })

  describe('getClassificationForText', () => {
    it('should return null for empty text', async () => {
      const mockFetch = vi.fn(() => 
        createMockFetchResponse({ classifications: [{ text: 'error', level: 'error' }] })
      )
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useLogClassifications())

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.getClassificationForText('')).toBeNull()
    })

    it('should return null when no classifications exist', async () => {
      const mockFetch = vi.fn(() => createMockFetchResponse({ classifications: [] }))
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useLogClassifications())

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.getClassificationForText('some text')).toBeNull()
    })

    it('should return null when no classification matches', async () => {
      const mockFetch = vi.fn(() => 
        createMockFetchResponse({ 
          classifications: [{ text: 'Connection refused', level: 'error' }] 
        })
      )
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useLogClassifications())

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.getClassificationForText('Server started')).toBeNull()
    })

    it('should match case-insensitively', async () => {
      const mockFetch = vi.fn(() => 
        createMockFetchResponse({ 
          classifications: [{ text: 'Connection Refused', level: 'error' }] 
        })
      )
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useLogClassifications())

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      // Should match regardless of case
      expect(result.current.getClassificationForText('connection refused')).toBe('error')
      expect(result.current.getClassificationForText('CONNECTION REFUSED')).toBe('error')
      expect(result.current.getClassificationForText('Error: Connection Refused at port 80')).toBe('error')
    })

    it('should use longest match when multiple classifications match', async () => {
      const mockFetch = vi.fn(() => 
        createMockFetchResponse({ 
          classifications: [
            { text: 'error', level: 'warning' },  // Short match
            { text: 'Connection error', level: 'error' },  // Longer match
          ] 
        })
      )
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useLogClassifications())

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      // "Connection error" is longer than "error", so should use its level
      expect(result.current.getClassificationForText('Connection error occurred')).toBe('error')
    })

    it('should return first match level when lengths are equal', async () => {
      const mockFetch = vi.fn(() => 
        createMockFetchResponse({ 
          classifications: [
            { text: 'abc', level: 'info' },
            { text: 'xyz', level: 'error' },
          ] 
        })
      )
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useLogClassifications())

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      // Only "abc" matches, should return info
      expect(result.current.getClassificationForText('abc123')).toBe('info')
      // Only "xyz" matches, should return error
      expect(result.current.getClassificationForText('xyz789')).toBe('error')
    })

    it('should handle partial matches anywhere in text', async () => {
      const mockFetch = vi.fn(() => 
        createMockFetchResponse({ 
          classifications: [{ text: 'timeout', level: 'warning' }] 
        })
      )
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useLogClassifications())

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.getClassificationForText('Request timeout after 30s')).toBe('warning')
      expect(result.current.getClassificationForText('timeout')).toBe('warning')
      expect(result.current.getClassificationForText('Connection timeout error')).toBe('warning')
    })
  })

  describe('reload', () => {
    it('should reload classifications from API', async () => {
      const initialClassifications = [{ text: 'initial', level: 'info' as const }]
      const reloadedClassifications = [
        { text: 'initial', level: 'info' as const },
        { text: 'added externally', level: 'error' as const },
      ]

      let callCount = 0
      const mockFetch = vi.fn(() => {
        callCount++
        if (callCount === 1) {
          return createMockFetchResponse({ classifications: initialClassifications })
        }
        return createMockFetchResponse({ classifications: reloadedClassifications })
      })
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useLogClassifications())

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.classifications).toEqual(initialClassifications)

      await act(async () => {
        await result.current.reload()
      })

      await waitFor(() => {
        expect(result.current.classifications).toEqual(reloadedClassifications)
      })
    })
  })
})
