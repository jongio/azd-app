import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  loadConfig,
  saveConfig,
  listBackups,
  getBackup,
  restoreBackup,
  deleteBackup,
} from './config-api'

describe('config-api', () => {
  beforeEach(() => {
    global.fetch = vi.fn()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('loadConfig', () => {
    it('should load configuration successfully', async () => {
      const mockResponse = {
        path: '/workspace/azure.yaml',
        content: 'name: test-app\nservices:\n  api:\n    host: containerapp\n',
        lastModified: '2026-01-11T10:00:00Z',
      }

      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      })

      const result = await loadConfig()

      expect(global.fetch).toHaveBeenCalledWith('/api/editor/config', {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      })

      expect(result).toEqual(mockResponse)
    })

    it('should throw error on failed request', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: false,
        json: async () => ({ error: 'File not found' }),
      })

      await expect(loadConfig()).rejects.toThrow('File not found')
    })
  })

  describe('saveConfig', () => {
    it('should save configuration successfully', async () => {
      const content = 'name: updated-app\nservices:\n  web:\n    host: appservice\n'
      const mockResponse = {
        success: true,
        backup: '/workspace/azure.yaml.backup.2026-01-11T100000Z',
        written: true,
        errors: [],
      }

      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      })

      const result = await saveConfig(content)

      expect(global.fetch).toHaveBeenCalledWith('/api/editor/config', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ content }),
      })

      expect(result).toEqual(mockResponse)
    })

    it('should throw error on invalid YAML', async () => {
      const invalidContent = 'this is: not: valid: yaml: : :'

      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: false,
        json: async () => ({ error: 'Invalid YAML content' }),
      })

      await expect(saveConfig(invalidContent)).rejects.toThrow('Invalid YAML content')
    })
  })

  describe('listBackups', () => {
    it('should list backups successfully', async () => {
      const mockResponse = {
        backups: [
          {
            timestamp: '2026-01-11T120000Z',
            path: '/workspace/azure.yaml.backup.2026-01-11T120000Z',
            size: 1024,
          },
          {
            timestamp: '2026-01-11T110000Z',
            path: '/workspace/azure.yaml.backup.2026-01-11T110000Z',
            size: 1020,
          },
        ],
      }

      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      })

      const result = await listBackups()

      expect(global.fetch).toHaveBeenCalledWith('/api/editor/backups', {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      })

      expect(result).toEqual(mockResponse)
      expect(result.backups).toHaveLength(2)
      // Should be sorted newest first
      expect(result.backups[0].timestamp > result.backups[1].timestamp).toBe(true)
    })
  })

  describe('getBackup', () => {
    it('should get backup content successfully', async () => {
      const timestamp = '2026-01-11T120000Z'
      const mockResponse = {
        content: 'name: backup-content\nservices:\n  api:\n    host: containerapp\n',
        timestamp,
      }

      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      })

      const result = await getBackup(timestamp)

      expect(global.fetch).toHaveBeenCalledWith(`/api/editor/backups/${timestamp}`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      })

      expect(result).toEqual(mockResponse)
    })

    it('should throw error for non-existent backup', async () => {
      const timestamp = '2026-01-01T000000Z'

      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: false,
        json: async () => ({ error: 'Backup not found' }),
      })

      await expect(getBackup(timestamp)).rejects.toThrow('Backup not found')
    })
  })

  describe('restoreBackup', () => {
    it('should restore backup successfully', async () => {
      const timestamp = '2026-01-11T120000Z'
      const mockResponse = {
        success: true,
        restoredFrom: '/workspace/azure.yaml.backup.2026-01-11T120000Z',
        backupCreated: '/workspace/azure.yaml.backup.2026-01-11T130000Z',
      }

      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      })

      const result = await restoreBackup(timestamp)

      expect(global.fetch).toHaveBeenCalledWith(`/api/editor/backups/${timestamp}/restore`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      })

      expect(result).toEqual(mockResponse)
      expect(result.success).toBe(true)
    })
  })

  describe('deleteBackup', () => {
    it('should delete backup successfully', async () => {
      const timestamp = '2026-01-11T120000Z'

      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: true,
      })

      await deleteBackup(timestamp)

      expect(global.fetch).toHaveBeenCalledWith(`/api/editor/backups/${timestamp}`, {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
        },
      })
    })

    it('should throw error on failed delete', async () => {
      const timestamp = '2026-01-11T120000Z'

      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: false,
        json: async () => ({ error: 'Failed to delete backup' }),
      })

      await expect(deleteBackup(timestamp)).rejects.toThrow('Failed to delete backup')
    })
  })
})
