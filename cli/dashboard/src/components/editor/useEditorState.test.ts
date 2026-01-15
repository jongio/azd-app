/**
 * useEditorState Tests - Zustand Store State Management
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/react'
import { useEditorState } from './useEditorState'
import * as configApi from '@/lib/editor/config-api'

// Mock dependencies
vi.mock('@/lib/editor/config-api')
vi.mock('@/lib/editor/yaml-utils', async () => {
  const actual = await vi.importActual('@/lib/editor/yaml-utils')
  return {
    ...actual,
    parseYaml: vi.fn((yaml: string) => {
      try {
        // Simple YAML parsing mock
        if (!yaml || yaml.trim() === '') {
          return { success: false, error: 'Empty YAML' }
        }
        return { success: true, data: { name: 'test-app' } }
      } catch (e) {
        return { success: false, error: String(e) }
      }
    }),
    stringifyYaml: vi.fn((data: Record<string, unknown>) => `name: ${data.name || 'test-app'}`),
  }
})

describe('useEditorState', () => {
  beforeEach(() => {
    // Reset store state before each test
    const { result } = renderHook(() => useEditorState())
    act(() => {
      result.current.resetConfig()
    })
    
    // Clear localStorage
    localStorage.clear()
    vi.clearAllMocks()
  })

  // ===========================================================================
  // Initial State Tests
  // ===========================================================================

  describe('Initial State', () => {
    it('should initialize with null config', () => {
      const { result } = renderHook(() => useEditorState())
      
      expect(result.current.config).toBeNull()
      expect(result.current.configYaml).toBe('')
      expect(result.current.originalYaml).toBe('')
      expect(result.current.isDirty).toBe(false)
      expect(result.current.isLoading).toBe(false)
      expect(result.current.isSaving).toBe(false)
      expect(result.current.error).toBeNull()
    })

    it('should initialize with default UI state', () => {
      const { result } = renderHook(() => useEditorState())
      
      expect(result.current.isPreviewVisible).toBe(true)
      expect(result.current.isSidebarCollapsed).toBe(false)
      expect(result.current.isHelpPanelOpen).toBe(false)
      expect(result.current.activeSection).toBe('overview')
    })

    it('should initialize with all modals closed', () => {
      const { result } = renderHook(() => useEditorState())
      
      expect(result.current.isAddServiceModalOpen).toBe(false)
      expect(result.current.isResourceConfigModalOpen).toBe(false)
      expect(result.current.isHealthCheckModalOpen).toBe(false)
      expect(result.current.isHooksConfigModalOpen).toBe(false)
      expect(result.current.isBackupManagerOpen).toBe(false)
      expect(result.current.isCommandPaletteOpen).toBe(false)
      expect(result.current.isImportModalOpen).toBe(false)
      expect(result.current.isExportModalOpen).toBe(false)
      expect(result.current.isKeyboardShortcutsOpen).toBe(false)
    })

    it('should initialize with no validation errors', () => {
      const { result } = renderHook(() => useEditorState())
      
      expect(result.current.validationErrors).toEqual([])
    })
  })

  // ===========================================================================
  // Modal State Management Tests
  // ===========================================================================

  describe('Modal State Management', () => {
    it('should open and close add service modal', () => {
      const { result } = renderHook(() => useEditorState())
      
      act(() => {
        result.current.openAddServiceModal()
      })
      expect(result.current.isAddServiceModalOpen).toBe(true)
      
      act(() => {
        result.current.closeAddServiceModal()
      })
      expect(result.current.isAddServiceModalOpen).toBe(false)
    })

    it('should open and close resource config modal with context', () => {
      const { result } = renderHook(() => useEditorState())
      
      act(() => {
        result.current.openResourceConfigModal('storage')
      })
      expect(result.current.isResourceConfigModalOpen).toBe(true)
      expect(result.current.selectedResource).toBe('storage')
      
      act(() => {
        result.current.closeResourceConfigModal()
      })
      expect(result.current.isResourceConfigModalOpen).toBe(false)
      expect(result.current.selectedResource).toBeNull()
    })

    it('should open and close health check modal with context', () => {
      const { result } = renderHook(() => useEditorState())
      
      act(() => {
        result.current.openHealthCheckModal('api')
      })
      expect(result.current.isHealthCheckModalOpen).toBe(true)
      expect(result.current.selectedService).toBe('api')
      
      act(() => {
        result.current.closeHealthCheckModal()
      })
      expect(result.current.isHealthCheckModalOpen).toBe(false)
      expect(result.current.selectedService).toBeNull()
    })

    it('should open and close command palette modal', () => {
      const { result } = renderHook(() => useEditorState())
      
      act(() => {
        result.current.openCommandPalette()
      })
      expect(result.current.isCommandPaletteOpen).toBe(true)
      
      act(() => {
        result.current.closeCommandPalette()
      })
      expect(result.current.isCommandPaletteOpen).toBe(false)
    })

    it('should open and close import modal', () => {
      const { result } = renderHook(() => useEditorState())
      
      act(() => {
        result.current.openImportModal()
      })
      expect(result.current.isImportModalOpen).toBe(true)
      
      act(() => {
        result.current.closeImportModal()
      })
      expect(result.current.isImportModalOpen).toBe(false)
    })

    it('should open and close export modal', () => {
      const { result } = renderHook(() => useEditorState())
      
      act(() => {
        result.current.openExportModal()
      })
      expect(result.current.isExportModalOpen).toBe(true)
      
      act(() => {
        result.current.closeExportModal()
      })
      expect(result.current.isExportModalOpen).toBe(false)
    })

    it('should open and close backup manager', () => {
      const { result } = renderHook(() => useEditorState())
      
      act(() => {
        result.current.openBackupManager()
      })
      expect(result.current.isBackupManagerOpen).toBe(true)
      
      act(() => {
        result.current.closeBackupManager()
      })
      expect(result.current.isBackupManagerOpen).toBe(false)
    })

    it('should open and close keyboard shortcuts modal', () => {
      const { result } = renderHook(() => useEditorState())
      
      act(() => {
        result.current.openKeyboardShortcuts()
      })
      expect(result.current.isKeyboardShortcutsOpen).toBe(true)
      
      act(() => {
        result.current.closeKeyboardShortcuts()
      })
      expect(result.current.isKeyboardShortcutsOpen).toBe(false)
    })
  })

  // ===========================================================================
  // UI State Tests
  // ===========================================================================

  describe('UI State Management', () => {
    it('should toggle preview visibility', () => {
      const { result } = renderHook(() => useEditorState())
      
      const initial = result.current.isPreviewVisible
      
      act(() => {
        result.current.togglePreview()
      })
      expect(result.current.isPreviewVisible).toBe(!initial)
      
      act(() => {
        result.current.togglePreview()
      })
      expect(result.current.isPreviewVisible).toBe(initial)
    })

    it('should toggle sidebar collapse', () => {
      const { result } = renderHook(() => useEditorState())
      
      const initial = result.current.isSidebarCollapsed
      
      act(() => {
        result.current.toggleSidebar()
      })
      expect(result.current.isSidebarCollapsed).toBe(!initial)
      
      act(() => {
        result.current.toggleSidebar()
      })
      expect(result.current.isSidebarCollapsed).toBe(initial)
    })

    it('should toggle help panel and set section', () => {
      const { result } = renderHook(() => useEditorState())
      
      act(() => {
        result.current.toggleHelpPanel('services')
      })
      expect(result.current.isHelpPanelOpen).toBe(true)
      expect(result.current.helpSection).toBe('services')
      
      act(() => {
        result.current.toggleHelpPanel()
      })
      expect(result.current.isHelpPanelOpen).toBe(false)
      expect(result.current.helpSection).toBe('services') // Section persists
    })

    it('should set active section', () => {
      const { result } = renderHook(() => useEditorState())
      
      act(() => {
        result.current.setActiveSection('services')
      })
      expect(result.current.activeSection).toBe('services')
      
      act(() => {
        result.current.setActiveSection('resources')
      })
      expect(result.current.activeSection).toBe('resources')
    })
  })

  // ===========================================================================
  // Config Management Tests
  // ===========================================================================

  describe('Config Management', () => {
    it('should mark config as dirty when updating', () => {
      const { result } = renderHook(() => useEditorState())
      
      act(() => {
        result.current.updateConfig({ name: 'my-app' })
      })
      
      expect(result.current.isDirty).toBe(true)
      expect(result.current.config).toEqual({ name: 'my-app' })
    })

    it('should update nested fields', () => {
      const { result } = renderHook(() => useEditorState())
      
      act(() => {
        result.current.updateField('name', 'test-app')
      })
      
      expect(result.current.isDirty).toBe(true)
      expect(result.current.configYaml).toContain('test-app')
    })

    it('should reset config to original state', () => {
      const { result } = renderHook(() => useEditorState())
      
      // Set original state
      act(() => {
        useEditorState.setState({
          originalYaml: 'name: original-app',
          configYaml: 'name: modified-app',
          config: { name: 'modified-app' },
          isDirty: true,
        })
      })
      
      expect(result.current.isDirty).toBe(true)
      
      act(() => {
        result.current.resetConfig()
      })
      
      expect(result.current.isDirty).toBe(false)
      expect(result.current.config).toEqual({ name: 'test-app' })
    })

    it('should discard changes', () => {
      const { result } = renderHook(() => useEditorState())
      
      // Set up modified state
      act(() => {
        useEditorState.setState({
          originalYaml: 'name: original-app',
          configYaml: 'name: modified-app',
          config: { name: 'modified-app' },
          isDirty: true,
        })
      })
      
      act(() => {
        result.current.discardChanges()
      })
      
      expect(result.current.isDirty).toBe(false)
    })
  })

  // ===========================================================================
  // Validation Tests
  // ===========================================================================

  describe('Validation Management', () => {
    it('should set validation errors', () => {
      const { result } = renderHook(() => useEditorState())
      
      const errors = [
        {
          path: 'services.api.port',
          message: 'Port must be a number',
          level: 'error' as const,
        },
      ]
      
      act(() => {
        result.current.setValidationErrors(errors)
      })
      
      expect(result.current.validationErrors).toEqual(errors)
    })

    it('should clear validation errors', () => {
      const { result } = renderHook(() => useEditorState())
      
      act(() => {
        result.current.setValidationErrors([
          { path: 'test', message: 'error', level: 'error' },
        ])
      })
      
      expect(result.current.validationErrors.length).toBe(1)
      
      act(() => {
        result.current.setValidationErrors([])
      })
      
      expect(result.current.validationErrors).toEqual([])
    })
  })

  // ===========================================================================
  // Async Loading Tests
  // ===========================================================================

  describe('Async Config Loading', () => {
    it('should load config successfully', async () => {
      const mockConfigResponse = {
        content: 'name: loaded-app\nservices: {}',
        path: '/azure.yaml',
        lastModified: '2026-01-14T00:00:00Z',
      }
      
      vi.mocked(configApi.loadConfig).mockResolvedValue(mockConfigResponse)
      
      const { result } = renderHook(() => useEditorState())
      
      await act(async () => {
        await result.current.loadConfig()
      })
      
      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })
      
      expect(result.current.config).toEqual({ name: 'test-app' })
      expect(result.current.error).toBeNull()
    })

    it('should handle load errors', async () => {
      vi.mocked(configApi.loadConfig).mockRejectedValue(new Error('Network error'))
      
      const { result } = renderHook(() => useEditorState())
      
      await act(async () => {
        await result.current.loadConfig()
      })
      
      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })
      
      expect(result.current.error).toContain('Network error')
    })

    it('should save config successfully', async () => {
      const mockSaveResponse = {
        backup: '/backups/azure.yaml.bak',
        written: true,
        success: true,
        errors: [],
      }
      
      vi.mocked(configApi.saveConfig).mockResolvedValue(mockSaveResponse)
      
      const { result } = renderHook(() => useEditorState())
      
      // Set up dirty state
      act(() => {
        useEditorState.setState({
          configYaml: 'name: test-app',
          isDirty: true,
        })
      })
      
      let saveResult = false
      await act(async () => {
        saveResult = await result.current.saveConfig()
      })
      
      expect(saveResult).toBe(true)
      expect(result.current.isDirty).toBe(false)
      expect(result.current.isSaving).toBe(false)
    })

    it('should handle save errors', async () => {
      const mockSaveResponse = {
        success: false,
        backup: '',
        written: false,
        errors: ['Invalid YAML syntax'],
      }
      
      vi.mocked(configApi.saveConfig).mockResolvedValue(mockSaveResponse)
      
      const { result } = renderHook(() => useEditorState())
      
      act(() => {
        useEditorState.setState({
          configYaml: 'invalid: : yaml',
        })
      })
      
      let saveResult = true
      await act(async () => {
        saveResult = await result.current.saveConfig()
      })
      
      expect(saveResult).toBe(false)
      expect(result.current.error).toContain('Invalid YAML syntax')
    })

    it('should not save if already saving', async () => {
      const { result } = renderHook(() => useEditorState())
      
      act(() => {
        useEditorState.setState({
          isSaving: true,
        })
      })
      
      const saveResult = await result.current.saveConfig()
      
      expect(saveResult).toBe(false)
      expect(configApi.saveConfig).not.toHaveBeenCalled()
    })
  })
})
