/**
 * Well-Known Services API Tests
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { fetchWellKnownServices, fetchWellKnownService } from './wellknown'

describe('Well-Known Services API', () => {
  const mockServices = [
    {
      name: 'azurite',
      displayName: 'Azurite (Azure Storage Emulator)',
      description: 'Local Azure Storage emulator',
      category: 'storage',
      icon: '📦',
      host: 'containerapp',
      image: 'mcr.microsoft.com/azure-storage/azurite:latest',
      ports: ['10000:10000'],
      connectionStrings: {
        default: 'UseDevelopmentStorage=true',
      },
    },
    {
      name: 'cosmos',
      displayName: 'Azure Cosmos DB Emulator',
      description: 'Local Azure Cosmos DB emulator',
      category: 'database',
      icon: '🌍',
      host: 'containerapp',
      image: 'mcr.microsoft.com/cosmosdb/linux/azure-cosmos-emulator:latest',
      ports: ['8081:8081'],
      connectionStrings: {
        default: 'AccountEndpoint=https://localhost:8081/',
      },
    },
  ]

  beforeEach(() => {
    global.fetch = vi.fn()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('fetchWellKnownServices', () => {
    it('fetches all well-known services successfully', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ services: mockServices }),
      })

      const result = await fetchWellKnownServices()

      expect(global.fetch).toHaveBeenCalledWith('/api/editor/wellknown')
      expect(result).toEqual(mockServices)
    })

    it('returns empty array when services field is missing', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: true,
        json: async () => ({}),
      })

      const result = await fetchWellKnownServices()

      expect(result).toEqual([])
    })

    it('throws error when response is not ok', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
      })

      await expect(fetchWellKnownServices()).rejects.toThrow(
        'Failed to fetch well-known services: HTTP 500: Internal Server Error'
      )
    })

    it('throws error when fetch fails', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockRejectedValueOnce(new Error('Network error'))

      await expect(fetchWellKnownServices()).rejects.toThrow('Network error')
    })
  })

  describe('fetchWellKnownService', () => {
    it('fetches specific service by name successfully', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: true,
        json: async () => mockServices[0],
      })

      const result = await fetchWellKnownService('azurite')

      expect(global.fetch).toHaveBeenCalledWith('/api/editor/wellknown/azurite')
      expect(result).toEqual(mockServices[0])
    })

    it('returns null when service is not found (404)', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: false,
        status: 404,
      })

      const result = await fetchWellKnownService('nonexistent')

      expect(result).toBeNull()
    })

    it('throws error when response is not ok (non-404 errors)', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
      })

      await expect(fetchWellKnownService('azurite')).rejects.toThrow(
        'Failed to fetch well-known service azurite: HTTP 500: Internal Server Error'
      )
    })

    it('throws error when fetch fails', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockRejectedValueOnce(new Error('Network error'))

      await expect(fetchWellKnownService('azurite')).rejects.toThrow('Network error')
    })
  })
})
