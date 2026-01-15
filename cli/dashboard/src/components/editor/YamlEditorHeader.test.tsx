/**
 * YamlEditorHeader Tests
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { YamlEditorHeader } from './YamlEditorHeader'
import { useEditorState } from './useEditorState'

// Mock theme provider
vi.mock('@/lib/editor/theme-provider', () => ({
  ThemeToggle: () => <button>Theme Toggle</button>,
}))

describe('YamlEditorHeader', () => {
  beforeEach(() => {
    // Reset state before each test
    useEditorState.setState({
      isDirty: false,
      isSaving: false,
      isLoading: false,
      error: null,
      isPreviewVisible: true,
      validationErrors: [],
      isAddServiceModalOpen: false,
    })
    
    vi.clearAllMocks()
  })

  // ===========================================================================
  // Basic Rendering Tests
  // ===========================================================================

  describe('Rendering', () => {
    it('should render header with title', () => {
      render(<YamlEditorHeader />)
      
      expect(screen.getByText('Azure YAML Editor')).toBeInTheDocument()
    })

    it('should render all action buttons', () => {
      render(<YamlEditorHeader />)
      
      expect(screen.getByRole('button', { name: /add service/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /preview/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /help/i })).toBeInTheDocument()
      // Keyboard button has no accessible name, check by title
      const keyboardButton = screen.getAllByRole('button').find(
        btn => btn.getAttribute('title')?.includes('Keyboard shortcuts')
      )
      expect(keyboardButton).toBeDefined()
      expect(screen.getByRole('button', { name: /cancel/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /save/i })).toBeInTheDocument()
    })

    it('should render theme toggle', () => {
      render(<YamlEditorHeader />)
      
      expect(screen.getByText('Theme Toggle')).toBeInTheDocument()
    })
  })

  // ===========================================================================
  // Status Indicators Tests
  // ===========================================================================

  describe('Status Indicators', () => {
    it('should show loading state', () => {
      useEditorState.setState({ isLoading: true })
      
      render(<YamlEditorHeader />)
      
      expect(screen.getByText('Loading...')).toBeInTheDocument()
    })

    it('should show unsaved changes badge when dirty', () => {
      useEditorState.setState({ isDirty: true })
      
      render(<YamlEditorHeader />)
      
      expect(screen.getByText('Unsaved changes')).toBeInTheDocument()
    })

    it('should show error message when present', () => {
      useEditorState.setState({ error: 'Failed to save configuration' })
      
      render(<YamlEditorHeader />)
      
      expect(screen.getByText('Failed to save configuration')).toBeInTheDocument()
    })

    it('should show validation error count', () => {
      useEditorState.setState({
        validationErrors: [
          { path: 'services.api.port', message: 'Invalid port', level: 'error' },
          { path: 'services.web.port', message: 'Invalid port', level: 'error' },
        ],
      })
      
      render(<YamlEditorHeader />)
      
      expect(screen.getByText('2 errors')).toBeInTheDocument()
    })

    it('should show validation warning count when no errors', () => {
      useEditorState.setState({
        validationErrors: [
          { path: 'services.api.name', message: 'Name should be lowercase', level: 'warning' },
          { path: 'services.web.name', message: 'Name should be lowercase', level: 'warning' },
          { path: 'services.db.name', message: 'Name should be lowercase', level: 'warning' },
        ],
      })
      
      render(<YamlEditorHeader />)
      
      expect(screen.getByText('3 warnings')).toBeInTheDocument()
    })

    it('should show error count even when warnings exist', () => {
      useEditorState.setState({
        validationErrors: [
          { path: 'services.api.port', message: 'Invalid port', level: 'error' },
          { path: 'services.api.name', message: 'Name should be lowercase', level: 'warning' },
        ],
      })
      
      render(<YamlEditorHeader />)
      
      expect(screen.getByText('1 error')).toBeInTheDocument()
      expect(screen.queryByText(/warning/i)).not.toBeInTheDocument()
    })

    it('should use singular form for single error', () => {
      useEditorState.setState({
        validationErrors: [
          { path: 'services.api.port', message: 'Invalid port', level: 'error' },
        ],
      })
      
      render(<YamlEditorHeader />)
      
      expect(screen.getByText('1 error')).toBeInTheDocument()
    })
  })

  // ===========================================================================
  // Button State Tests
  // ===========================================================================

  describe('Button States', () => {
    it('should disable save button when no changes', () => {
      useEditorState.setState({ isDirty: false })
      
      render(<YamlEditorHeader />)
      
      const saveButton = screen.getByRole('button', { name: /save/i })
      expect(saveButton).toBeDisabled()
    })

    it('should enable save button when dirty', () => {
      useEditorState.setState({ isDirty: true })
      
      render(<YamlEditorHeader />)
      
      const saveButton = screen.getByRole('button', { name: /save/i })
      expect(saveButton).not.toBeDisabled()
    })

    it('should disable save button when saving', () => {
      useEditorState.setState({ isDirty: true, isSaving: true })
      
      render(<YamlEditorHeader />)
      
      const saveButton = screen.getByRole('button', { name: /saving/i })
      expect(saveButton).toBeDisabled()
    })

    it('should disable save button when loading', () => {
      useEditorState.setState({ isDirty: true, isLoading: true })
      
      render(<YamlEditorHeader />)
      
      const saveButton = screen.getByRole('button', { name: /save/i })
      expect(saveButton).toBeDisabled()
    })

    it('should disable save button when validation errors exist', () => {
      useEditorState.setState({
        isDirty: true,
        validationErrors: [
          { path: 'services.api.port', message: 'Invalid port', level: 'error' },
        ],
      })
      
      render(<YamlEditorHeader />)
      
      const saveButton = screen.getByRole('button', { name: /save/i })
      expect(saveButton).toBeDisabled()
    })

    it('should enable save button with warnings but no errors', () => {
      useEditorState.setState({
        isDirty: true,
        validationErrors: [
          { path: 'services.api.name', message: 'Name should be lowercase', level: 'warning' },
        ],
      })
      
      render(<YamlEditorHeader />)
      
      const saveButton = screen.getByRole('button', { name: /save/i })
      expect(saveButton).not.toBeDisabled()
    })

    it('should disable cancel button when no changes', () => {
      useEditorState.setState({ isDirty: false })
      
      render(<YamlEditorHeader />)
      
      const cancelButton = screen.getByRole('button', { name: /cancel/i })
      expect(cancelButton).toBeDisabled()
    })

    it('should enable cancel button when dirty', () => {
      useEditorState.setState({ isDirty: true })
      
      render(<YamlEditorHeader />)
      
      const cancelButton = screen.getByRole('button', { name: /cancel/i })
      expect(cancelButton).not.toBeDisabled()
    })

    it('should disable cancel button when saving', () => {
      useEditorState.setState({ isDirty: true, isSaving: true })
      
      render(<YamlEditorHeader />)
      
      const cancelButton = screen.getByRole('button', { name: /cancel/i })
      expect(cancelButton).toBeDisabled()
    })

    it('should show saving state with spinner', () => {
      useEditorState.setState({ isDirty: true, isSaving: true })
      
      render(<YamlEditorHeader />)
      
      expect(screen.getByText('Saving...')).toBeInTheDocument()
    })
  })

  // ===========================================================================
  // Save Functionality Tests
  // ===========================================================================

  describe('Save Functionality', () => {
    it('should call saveConfig when save button clicked', async () => {
      const user = userEvent.setup()
      const mockSaveConfig = vi.fn().mockResolvedValue(true)
      
      useEditorState.setState({
        isDirty: true,
        saveConfig: mockSaveConfig,
      })
      
      render(<YamlEditorHeader />)
      
      const saveButton = screen.getByRole('button', { name: /save/i })
      await user.click(saveButton)
      
      expect(mockSaveConfig).toHaveBeenCalledTimes(1)
    })

    it('should show success message after successful save', async () => {
      const user = userEvent.setup()
      const mockSaveConfig = vi.fn().mockResolvedValue(true)
      
      useEditorState.setState({
        isDirty: true,
        saveConfig: mockSaveConfig,
      })
      
      render(<YamlEditorHeader />)
      
      const saveButton = screen.getByRole('button', { name: /save/i })
      await user.click(saveButton)
      
      await waitFor(() => {
        expect(screen.getByText('Saved successfully')).toBeInTheDocument()
      })
    })

    it('should hide success message after 2 seconds', async () => {
      vi.useFakeTimers()
      const mockSaveConfig = vi.fn().mockResolvedValue(true)
      
      useEditorState.setState({
        isDirty: true,
        saveConfig: mockSaveConfig,
      })
      
      render(<YamlEditorHeader />)
      
      const saveButton = screen.getByRole('button', { name: /save/i })
      
      fireEvent.click(saveButton)
      
      // Flush all promises to let the async saveConfig complete
      await act(async () => {
        await Promise.resolve()
      })
      
      // Now success message should be visible
      expect(screen.getByText('Saved successfully')).toBeInTheDocument()
      
      // Advance timers to trigger the setTimeout(2000)
      act(() => {
        vi.advanceTimersByTime(2000)
      })
      
      // Success message should be gone
      expect(screen.queryByText('Saved successfully')).not.toBeInTheDocument()
      
      vi.useRealTimers()
    })

    it('should not show success message on failed save', async () => {
      const mockSaveConfig = vi.fn().mockResolvedValue(false)
      
      useEditorState.setState({
        isDirty: true,
        saveConfig: mockSaveConfig,
      })
      
      render(<YamlEditorHeader />)
      
      const saveButton = screen.getByRole('button', { name: /save/i })
      
      fireEvent.click(saveButton)
      
      // Flush promises to let the async saveConfig complete
      await act(async () => {
        await Promise.resolve()
      })
      
      // Verify no success message appears
      expect(screen.queryByText('Saved successfully')).not.toBeInTheDocument()
    })
  })

  // ===========================================================================
  // Cancel Functionality Tests
  // ===========================================================================

  describe('Cancel Functionality', () => {
    it('should call discardChanges when cancel confirmed', () => {
      const mockDiscardChanges = vi.fn()
      window.confirm = vi.fn(() => true)
      
      useEditorState.setState({
        isDirty: true,
        discardChanges: mockDiscardChanges,
      })
      
      render(<YamlEditorHeader />)
      
      const cancelButton = screen.getByRole('button', { name: /cancel/i })
      fireEvent.click(cancelButton)
      
      expect(window.confirm).toHaveBeenCalledWith('Discard all changes?')
      expect(mockDiscardChanges).toHaveBeenCalledTimes(1)
    })

    it('should not discard changes when cancel declined', () => {
      const mockDiscardChanges = vi.fn()
      window.confirm = vi.fn(() => false)
      
      useEditorState.setState({
        isDirty: true,
        discardChanges: mockDiscardChanges,
      })
      
      render(<YamlEditorHeader />)
      
      const cancelButton = screen.getByRole('button', { name: /cancel/i })
      fireEvent.click(cancelButton)
      
      expect(window.confirm).toHaveBeenCalled()
      expect(mockDiscardChanges).not.toHaveBeenCalled()
    })
  })

  // ===========================================================================
  // Preview Toggle Tests
  // ===========================================================================

  describe('Preview Toggle', () => {
    it('should show preview visible state', () => {
      useEditorState.setState({ isPreviewVisible: true })
      
      render(<YamlEditorHeader />)
      
      const previewButton = screen.getByRole('button', { name: /preview/i })
      expect(previewButton).toHaveAttribute('title', 'Hide preview (Ctrl+P)')
    })

    it('should show preview hidden state', () => {
      useEditorState.setState({ isPreviewVisible: false })
      
      render(<YamlEditorHeader />)
      
      const previewButton = screen.getByRole('button', { name: /preview/i })
      expect(previewButton).toHaveAttribute('title', 'Show preview (Ctrl+P)')
    })

    it('should toggle preview when clicked', () => {
      const mockTogglePreview = vi.fn()
      
      useEditorState.setState({
        isPreviewVisible: true,
        togglePreview: mockTogglePreview,
      })
      
      render(<YamlEditorHeader />)
      
      const previewButton = screen.getByRole('button', { name: /preview/i })
      fireEvent.click(previewButton)
      
      expect(mockTogglePreview).toHaveBeenCalledTimes(1)
    })
  })

  // ===========================================================================
  // Help and Shortcuts Tests
  // ===========================================================================

  describe('Help and Shortcuts', () => {
    it('should open help panel when help button clicked', () => {
      const mockToggleHelpPanel = vi.fn()
      
      useEditorState.setState({
        toggleHelpPanel: mockToggleHelpPanel,
      })
      
      render(<YamlEditorHeader />)
      
      const helpButton = screen.getByRole('button', { name: /help/i })
      fireEvent.click(helpButton)
      
      expect(mockToggleHelpPanel).toHaveBeenCalledTimes(1)
    })

    it('should open keyboard shortcuts when keyboard button clicked', () => {
      const mockOpenKeyboardShortcuts = vi.fn()
      
      useEditorState.setState({
        openKeyboardShortcuts: mockOpenKeyboardShortcuts,
      })
      
      render(<YamlEditorHeader />)
      
      // Keyboard button has no text, only title - query by title attribute
      const keyboardButton = screen.getAllByRole('button').find(
        btn => btn.getAttribute('title')?.includes('Keyboard shortcuts')
      )
      expect(keyboardButton).toBeDefined()
      fireEvent.click(keyboardButton!)
      
      expect(mockOpenKeyboardShortcuts).toHaveBeenCalledTimes(1)
    })
  })

  // ===========================================================================
  // Add Service Modal Tests
  // ===========================================================================

  describe('Add Service Modal', () => {
    it('should call onOpenAddService when add service button clicked', () => {
      const mockOnOpenAddService = vi.fn()
      
      render(<YamlEditorHeader onOpenAddService={mockOnOpenAddService} />)
      
      const addServiceButton = screen.getByRole('button', { name: /add service/i })
      fireEvent.click(addServiceButton)
      
      expect(mockOnOpenAddService).toHaveBeenCalledTimes(1)
    })

    it('should hide add service button when modal is open', () => {
      useEditorState.setState({ isAddServiceModalOpen: true })
      
      render(<YamlEditorHeader />)
      
      // Button has aria-hidden when modal is open - can't use getByRole
      const addServiceButton = screen.getByText('Add Service').closest('button')
      expect(addServiceButton).toHaveAttribute('aria-hidden', 'true')
    })
  })
})
