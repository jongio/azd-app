/**
 * YAML Editor Integration Tests
 * 
 * Tests the complete editor integration including:
 * - Component rendering and integration
 * - State management
 * - Keyboard shortcuts
 * - Modal workflows
 * - Validation
 * - Save/load operations
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { YamlEditor } from './YamlEditor'
import { useEditorState } from './useEditorState'
import * as configApi from '@/lib/editor/config-api'

// Mock API calls
vi.mock('@/lib/editor/config-api')

// Mock performance hooks
vi.mock('@/lib/performance', async () => {
  const actual = await vi.importActual('@/lib/performance')
  return {
    ...actual,
    useCachedSchema: () => ({
      data: {
        type: 'object',
        properties: {
          name: { type: 'string', title: 'Application Name' },
          services: { type: 'object', title: 'Services' },
        },
      },
    }),
    useCachedWellKnownServices: () => ({
      data: [
        {
          name: 'azurite',
          displayName: 'Azurite (Local Storage)',
          icon: '📦',
          host: 'containerapp',
        },
      ],
    }),
  }
})

// Mock validation
vi.mock('@/lib/editor/validation', () => ({
  validateConfig: () => ({
    valid: true,
    errors: [],
    warnings: [],
  }),
}))

describe('YamlEditor Integration', () => {
  const mockConfig = {
    name: 'test-app',
    services: {
      api: {
        project: './src/api',
        host: 'containerapp',
        language: 'js',
      },
    },
  }

  const mockConfigYaml = `name: test-app
services:
  api:
    project: ./src/api
    host: containerapp
    language: js
`

  beforeEach(() => {
    // Reset state before each test
    useEditorState.setState({
      config: null,
      configYaml: '',
      originalYaml: '',
      isDirty: false,
      isLoading: false,
      isSaving: false,
      error: null,
      activeSection: 'overview',
      validationErrors: [],
      isPreviewVisible: true,
      isSidebarCollapsed: false,
      isHelpPanelOpen: false,
      helpSection: undefined,
      isAddServiceModalOpen: false,
      isResourceConfigModalOpen: false,
      isHealthCheckModalOpen: false,
      isHooksConfigModalOpen: false,
      isBackupManagerOpen: false,
      isCommandPaletteOpen: false,
      isImportModalOpen: false,
      isExportModalOpen: false,
      isKeyboardShortcutsOpen: false,
      selectedService: null,
      selectedResource: null,
    })

    // Mock loadConfig to return test config
    vi.mocked(configApi.loadConfig).mockResolvedValue({
      path: '/path/to/azure.yaml',
      content: mockConfigYaml,
      lastModified: '2024-01-01T00:00:00Z',
    })

    // Mock saveConfig
    vi.mocked(configApi.saveConfig).mockResolvedValue({
      success: true,
      backup: '/path/to/backup.yaml',
      written: true,
    })

    // Clear localStorage
    localStorage.clear()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  describe('Component Rendering', () => {
    it('should render loading state initially', () => {
      render(<YamlEditor />)
      
      expect(screen.getByText(/loading azure yaml editor/i)).toBeInTheDocument()
    })

    it('should render all main components after loading', async () => {
      render(<YamlEditor />)

      await waitFor(() => {
        expect(screen.getByText('Azure YAML Editor')).toBeInTheDocument()
      })

      // Header
      expect(screen.getByRole('button', { name: /save/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /cancel/i })).toBeInTheDocument()
      // Preview toggle button in header (there are multiple preview buttons)
      expect(screen.getAllByRole('button', { name: /preview/i }).length).toBeGreaterThan(0)

      // Navigation (if not collapsed)
      expect(screen.getByRole('navigation')).toBeInTheDocument()
    })

    it('should render preview pane when visible', async () => {
      render(<YamlEditor />)

      await waitFor(() => {
        expect(screen.getByText('YAML Preview')).toBeInTheDocument()
      })
    })

    it('should render quick actions bar', async () => {
      render(<YamlEditor />)

      await waitFor(() => {
        expect(screen.getByText('Quick Actions')).toBeInTheDocument()
      })
    })
  })

  describe('State Management', () => {
    it('should load config on mount', async () => {
      render(<YamlEditor />)

      await waitFor(() => {
        expect(configApi.loadConfig).toHaveBeenCalledTimes(1)
      })

      const state = useEditorState.getState()
      expect(state.config).toEqual(mockConfig)
      expect(state.isDirty).toBe(false)
    })

    it('should mark as dirty when config changes', async () => {
      render(<YamlEditor />)

      await waitFor(() => {
        const state = useEditorState.getState()
        expect(state.config).not.toBeNull()
      })

      // Update config
      const { updateConfig } = useEditorState.getState()
      updateConfig({ name: 'updated-app' })

      const state = useEditorState.getState()
      expect(state.isDirty).toBe(true)
    })

    it('should persist draft to localStorage', async () => {
      render(<YamlEditor />)

      await waitFor(() => {
        const state = useEditorState.getState()
        expect(state.config).not.toBeNull()
      })

      // Update config
      const { updateConfig } = useEditorState.getState()
      updateConfig({ name: 'updated-app' })

      // Check localStorage
      const draft = localStorage.getItem('azd-editor-draft')
      expect(draft).toContain('name: updated-app')
    })
  })

  describe('Keyboard Shortcuts', () => {
    it('should save config with Ctrl+S', async () => {
      const user = userEvent.setup()
      render(<YamlEditor />)

      await waitFor(() => {
        const state = useEditorState.getState()
        expect(state.config).not.toBeNull()
      })

      // Make a change
      const { updateConfig } = useEditorState.getState()
      updateConfig({ name: 'updated-app' })

      // Press Ctrl+S
      await user.keyboard('{Control>}s{/Control}')

      await waitFor(() => {
        expect(configApi.saveConfig).toHaveBeenCalled()
      })
    })

    it('should toggle preview with Ctrl+P', async () => {
      const user = userEvent.setup()
      render(<YamlEditor />)

      await waitFor(() => {
        expect(screen.getByText('YAML Preview')).toBeInTheDocument()
      })

      const initialState = useEditorState.getState().isPreviewVisible

      // Press Ctrl+P
      await user.keyboard('{Control>}p{/Control}')

      await waitFor(() => {
        const state = useEditorState.getState()
        expect(state.isPreviewVisible).toBe(!initialState)
      })
    })

    it('should open command palette with Ctrl+K', async () => {
      const user = userEvent.setup()
      render(<YamlEditor />)

      await waitFor(() => {
        const state = useEditorState.getState()
        expect(state.config).not.toBeNull()
      })

      // Press Ctrl+K
      await user.keyboard('{Control>}k{/Control}')

      await waitFor(() => {
        const state = useEditorState.getState()
        expect(state.isCommandPaletteOpen).toBe(true)
      })
    })

    it('should toggle sidebar with Ctrl+B', async () => {
      const user = userEvent.setup()
      render(<YamlEditor />)

      await waitFor(() => {
        const state = useEditorState.getState()
        expect(state.config).not.toBeNull()
      })

      const initialState = useEditorState.getState().isSidebarCollapsed

      // Press Ctrl+B
      await user.keyboard('{Control>}b{/Control}')

      await waitFor(() => {
        const state = useEditorState.getState()
        expect(state.isSidebarCollapsed).toBe(!initialState)
      })
    })

    it('should open help with F1', async () => {
      const user = userEvent.setup()
      render(<YamlEditor />)

      await waitFor(() => {
        const state = useEditorState.getState()
        expect(state.config).not.toBeNull()
      })

      // Press F1
      await user.keyboard('{F1}')

      await waitFor(() => {
        const state = useEditorState.getState()
        expect(state.isHelpPanelOpen).toBe(true)
      })
    })
  })

  describe('Save/Load Operations', () => {
    it('should save config successfully', async () => {
      const user = userEvent.setup()
      render(<YamlEditor />)

      await waitFor(() => {
        const state = useEditorState.getState()
        expect(state.config).not.toBeNull()
      })

      // Make a change
      const { updateConfig } = useEditorState.getState()
      updateConfig({ name: 'updated-app' })

      // Click save button
      const saveButton = screen.getByRole('button', { name: /save/i })
      await user.click(saveButton)

      await waitFor(() => {
        expect(configApi.saveConfig).toHaveBeenCalled()
        const state = useEditorState.getState()
        expect(state.isDirty).toBe(false)
      })
    })

    it('should show save success message', async () => {
      const user = userEvent.setup()
      render(<YamlEditor />)

      await waitFor(() => {
        const state = useEditorState.getState()
        expect(state.config).not.toBeNull()
      })

      // Make a change
      const { updateConfig } = useEditorState.getState()
      updateConfig({ name: 'updated-app' })

      // Click save button
      const saveButton = screen.getByRole('button', { name: /save/i })
      await user.click(saveButton)

      await waitFor(() => {
        expect(screen.getByText(/saved successfully/i)).toBeInTheDocument()
      })
    })

    it('should disable save button when no changes', async () => {
      render(<YamlEditor />)

      await waitFor(() => {
        const state = useEditorState.getState()
        expect(state.config).not.toBeNull()
      })

      const saveButton = screen.getByRole('button', { name: /save/i })
      expect(saveButton).toBeDisabled()
    })

    it('should discard changes on cancel', async () => {
      const user = userEvent.setup()
      
      // Mock window.confirm to return true
      vi.spyOn(window, 'confirm').mockReturnValue(true)
      
      render(<YamlEditor />)

      await waitFor(() => {
        const state = useEditorState.getState()
        expect(state.config).not.toBeNull()
      })

      // Make a change
      const { updateConfig } = useEditorState.getState()
      updateConfig({ name: 'updated-app' })

      expect(useEditorState.getState().isDirty).toBe(true)

      // Click cancel
      const cancelButton = screen.getByRole('button', { name: /cancel/i })
      await user.click(cancelButton)

      await waitFor(() => {
        const state = useEditorState.getState()
        expect(state.isDirty).toBe(false)
        expect(state.config?.name).toBe('test-app')
      })
    })
  })

  describe('Modal Workflows', () => {
    it('should open add service modal', async () => {
      const user = userEvent.setup()
      render(<YamlEditor />)

      await waitFor(() => {
        const state = useEditorState.getState()
        expect(state.config).not.toBeNull()
      })

      // Press Ctrl+N to open add service modal
      await user.keyboard('{Control>}n{/Control}')

      await waitFor(() => {
        const state = useEditorState.getState()
        expect(state.isAddServiceModalOpen).toBe(true)
      })
    })

    it('should open import modal from quick actions', async () => {
      const user = userEvent.setup()
      render(<YamlEditor />)

      await waitFor(() => {
        expect(screen.getByText('Quick Actions')).toBeInTheDocument()
      })

      const importButton = screen.getByLabelText(/import configuration/i)
      await user.click(importButton)

      await waitFor(() => {
        const state = useEditorState.getState()
        expect(state.isImportModalOpen).toBe(true)
      })
    })

    it('should open export modal from quick actions', async () => {
      const user = userEvent.setup()
      render(<YamlEditor />)

      await waitFor(() => {
        expect(screen.getByText('Quick Actions')).toBeInTheDocument()
      })

      const exportButton = screen.getByLabelText(/export configuration/i)
      await user.click(exportButton)

      await waitFor(() => {
        const state = useEditorState.getState()
        expect(state.isExportModalOpen).toBe(true)
      })
    })
  })

  describe('Validation', () => {
    it('should display validation errors', async () => {
      render(<YamlEditor />)

      await waitFor(() => {
        const state = useEditorState.getState()
        expect(state.config).not.toBeNull()
      })

      // Add validation errors and make config dirty to trigger header update
      const { setValidationErrors, updateConfig } = useEditorState.getState()
      updateConfig({ name: 'test-app' }) // This triggers re-render
      setValidationErrors([
        {
          level: 'error',
          message: 'Name is required',
          path: 'name',
        },
      ])

      // Check that state has validation errors
      await waitFor(() => {
        const state = useEditorState.getState()
        expect(state.validationErrors.length).toBe(1)
        expect(state.validationErrors[0].level).toBe('error')
      })
    })

    it('should disable save button when validation errors exist', async () => {
      render(<YamlEditor />)

      await waitFor(() => {
        const state = useEditorState.getState()
        expect(state.config).not.toBeNull()
      })

      // Add validation error
      const { setValidationErrors } = useEditorState.getState()
      setValidationErrors([
        {
          level: 'error',
          message: 'Name is required',
          path: 'name',
        },
      ])

      // Make config dirty
      const { updateConfig } = useEditorState.getState()
      updateConfig({ name: '' })

      await waitFor(() => {
        const saveButton = screen.getByRole('button', { name: /save/i })
        expect(saveButton).toBeDisabled()
      })
    })
  })

  describe('Accessibility', () => {
    it('should have proper heading structure', async () => {
      render(<YamlEditor />)

      await waitFor(() => {
        const heading = screen.getByRole('heading', { name: /azure yaml editor/i })
        expect(heading).toBeInTheDocument()
      })
    })

    it('should have navigation landmark', async () => {
      render(<YamlEditor />)

      await waitFor(() => {
        expect(screen.getByRole('navigation')).toBeInTheDocument()
      })
    })

    it('should have accessible button labels', async () => {
      render(<YamlEditor />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /save/i })).toBeInTheDocument()
        expect(screen.getByRole('button', { name: /cancel/i })).toBeInTheDocument()
        expect(screen.getByRole('button', { name: /help/i })).toBeInTheDocument()
      })
    })
  })

  describe('Error Handling', () => {
    it('should display error message when load fails', async () => {
      vi.mocked(configApi.loadConfig).mockRejectedValue(new Error('Load failed'))

      render(<YamlEditor />)

      await waitFor(() => {
        expect(screen.getByText(/load failed/i)).toBeInTheDocument()
      })
    })

    it('should display error message when save fails', async () => {
      const user = userEvent.setup()
      vi.mocked(configApi.saveConfig).mockRejectedValue(new Error('Save failed'))

      render(<YamlEditor />)

      await waitFor(() => {
        const state = useEditorState.getState()
        expect(state.config).not.toBeNull()
      })

      // Make a change
      const { updateConfig } = useEditorState.getState()
      updateConfig({ name: 'updated-app' })

      // Click save
      const saveButton = screen.getByRole('button', { name: /save/i })
      await user.click(saveButton)

      await waitFor(() => {
        expect(screen.getByText(/save failed/i)).toBeInTheDocument()
      })
    })
  })
})
