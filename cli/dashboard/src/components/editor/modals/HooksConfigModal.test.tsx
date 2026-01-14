/**
 * Tests for HooksConfigModal component
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HooksConfigModal } from './HooksConfigModal'
import type { HookConfig } from '@/lib/editor/hooks-types'

describe('HooksConfigModal', () => {
  const mockOnClose = vi.fn()
  const mockOnSave = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('Rendering', () => {
    it('should not render when closed', () => {
      render(
        <HooksConfigModal
          isOpen={false}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      expect(screen.queryByText('Add Lifecycle Hook')).not.toBeInTheDocument()
    })

    it('should render when open', () => {
      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      expect(screen.getByText('Add Lifecycle Hook')).toBeInTheDocument()
      expect(screen.getByLabelText(/Lifecycle Event/i)).toBeInTheDocument()
    })

    it('should show "Edit" title when editing existing hook', () => {
      const config: HookConfig = {
        run: './setup.sh',
        shell: 'bash',
      }

      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
          initialEvent="preprovision"
          initialConfig={config}
        />
      )

      expect(screen.getByText('Edit Lifecycle Hook')).toBeInTheDocument()
    })

    it('should display event selector in add mode', () => {
      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const eventSelect = screen.getByLabelText(/Lifecycle Event/i)
      expect(eventSelect).toBeInTheDocument()
      expect(eventSelect.tagName).toBe('SELECT')
    })

    it('should display event info in edit mode', () => {
      const config: HookConfig = {
        run: './deploy.sh',
      }

      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
          initialEvent="predeploy"
          initialConfig={config}
        />
      )

      expect(screen.getByText('Pre Deploy')).toBeInTheDocument()
      expect(screen.getByText(/Before application deployment/i)).toBeInTheDocument()
    })
  })

  describe('Enable/Disable Toggle', () => {
    it('should show toggle to enable hook', () => {
      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      expect(screen.getByLabelText(/Enable this hook/i)).toBeInTheDocument()
    })

    it('should hide configuration fields when disabled', () => {
      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      // Hook is disabled by default
      expect(screen.queryByLabelText(/Script Command/i)).not.toBeInTheDocument()
    })

    it('should show configuration fields when enabled', async () => {
      const user = userEvent.setup()

      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const enableToggle = screen.getByLabelText(/Enable this hook/i)
      await user.click(enableToggle)

      await waitFor(() => {
        expect(screen.getByLabelText(/Script Command/i)).toBeInTheDocument()
      })
    })
  })

  describe('Base Configuration', () => {
    it('should pre-fill fields with initial config', () => {
      const config: HookConfig = {
        run: './my-script.sh',
        shell: 'bash',
        continueOnError: true,
        interactive: false,
      }

      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
          initialEvent="postprovision"
          initialConfig={config}
        />
      )

      const scriptInput = screen.getByDisplayValue('./my-script.sh')
      expect(scriptInput).toBeInTheDocument()

      const continueOnErrorCheckbox = screen.getByLabelText(/Continue on error/i)
      expect(continueOnErrorCheckbox).toBeChecked()
    })

    it('should allow editing script command', async () => {
      const user = userEvent.setup()

      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const enableToggle = screen.getByLabelText(/Enable this hook/i)
      await user.click(enableToggle)

      const scriptInput = screen.getByLabelText(/Script Command/i)
      await user.type(scriptInput, './deploy.sh')

      expect(scriptInput).toHaveValue('./deploy.sh')
    })

    it('should allow selecting shell type', async () => {
      const user = userEvent.setup()

      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const enableToggle = screen.getByLabelText(/Enable this hook/i)
      await user.click(enableToggle)

      const shellSelect = screen.getByLabelText(/Shell/i)
      await user.selectOptions(shellSelect, 'bash')

      expect(shellSelect).toHaveValue('bash')
    })
  })

  describe('Platform Overrides', () => {
    it('should show Windows override section when checked', async () => {
      const user = userEvent.setup()

      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const enableToggle = screen.getByLabelText(/Enable this hook/i)
      await user.click(enableToggle)

      const windowsToggle = screen.getByLabelText(/Windows Override/i)
      await user.click(windowsToggle)

      await waitFor(() => {
        const windowsInputs = screen.getAllByPlaceholderText(/setup\.ps1/i)
        expect(windowsInputs.length).toBeGreaterThan(0)
      })
    })

    it('should show POSIX override section when checked', async () => {
      const user = userEvent.setup()

      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const enableToggle = screen.getByLabelText(/Enable this hook/i)
      await user.click(enableToggle)

      const posixToggle = screen.getByLabelText(/POSIX Override/i)
      await user.click(posixToggle)

      await waitFor(() => {
        const posixInputs = screen.getAllByPlaceholderText(/setup\.sh/i)
        expect(posixInputs.length).toBeGreaterThan(0)
      })
    })

    it('should pre-fill platform overrides from initial config', () => {
      const config: HookConfig = {
        run: './default.sh',
        windows: {
          run: '.\\windows.ps1',
          shell: 'pwsh',
        },
        posix: {
          run: './posix.sh',
          shell: 'bash',
        },
      }

      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
          initialEvent="prerun"
          initialConfig={config}
        />
      )

      expect(screen.getByDisplayValue('.\\windows.ps1')).toBeInTheDocument()
      expect(screen.getByDisplayValue('./posix.sh')).toBeInTheDocument()
    })
  })

  describe('Platform Coverage Warning', () => {
    it('should show warning when only Windows override exists', async () => {
      const user = userEvent.setup()

      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const enableToggle = screen.getByLabelText(/Enable this hook/i)
      await user.click(enableToggle)

      const windowsToggle = screen.getByLabelText(/Windows Override/i)
      await user.click(windowsToggle)

      const windowsInput = screen.getByPlaceholderText(/setup\.ps1/i)
      await user.type(windowsInput, '.\\setup.ps1')

      await waitFor(() => {
        expect(screen.getByText(/only runs on Windows/i)).toBeInTheDocument()
      })
    })

    it('should show success message for cross-platform coverage', async () => {
      const user = userEvent.setup()

      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const enableToggle = screen.getByLabelText(/Enable this hook/i)
      await user.click(enableToggle)

      const scriptInput = screen.getByLabelText(/Script Command/i)
      await user.type(scriptInput, './setup.sh')

      await waitFor(() => {
        expect(screen.getByText(/cross-platform coverage/i)).toBeInTheDocument()
      })
    })
  })

  describe('Form Submission', () => {
    it('should call onSave with null when hook is disabled', async () => {
      const user = userEvent.setup()

      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const saveButton = screen.getByRole('button', { name: /Save Hook/i })
      await user.click(saveButton)

      await waitFor(() => {
        expect(mockOnSave).toHaveBeenCalledWith('preprovision', null)
      })
    })

    it('should call onSave with hook config when enabled', async () => {
      const user = userEvent.setup()

      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const enableToggle = screen.getByLabelText(/Enable this hook/i)
      await user.click(enableToggle)

      const scriptInput = screen.getByLabelText(/Script Command/i)
      await user.type(scriptInput, './build.sh')

      const saveButton = screen.getByRole('button', { name: /Save Hook/i })
      await user.click(saveButton)

      await waitFor(() => {
        expect(mockOnSave).toHaveBeenCalledWith(
          'preprovision',
          expect.objectContaining({
            run: './build.sh',
            shell: 'sh',
            continueOnError: false,
            interactive: false,
          })
        )
      })
    })

    it('should call onClose after successful save', async () => {
      const user = userEvent.setup()

      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const saveButton = screen.getByRole('button', { name: /Save Hook/i })
      await user.click(saveButton)

      await waitFor(() => {
        expect(mockOnClose).toHaveBeenCalled()
      })
    })

    it('should handle save errors gracefully', async () => {
      const user = userEvent.setup()
      const errorSave = vi.fn().mockRejectedValue(new Error('Save failed'))
      const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {})

      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={errorSave}
        />
      )

      const saveButton = screen.getByRole('button', { name: /Save Hook/i })
      await user.click(saveButton)

      await waitFor(() => {
        expect(alertSpy).toHaveBeenCalledWith(expect.stringContaining('Failed to save'))
      })

      alertSpy.mockRestore()
    })

    it('should disable save button while submitting', async () => {
      const user = userEvent.setup()
      const slowSave = vi.fn().mockImplementation(
        () => new Promise((resolve) => setTimeout(resolve, 100))
      )

      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={slowSave}
        />
      )

      const saveButton = screen.getByRole('button', { name: /Save Hook/i })
      await user.click(saveButton)

      expect(saveButton).toBeDisabled()
      expect(screen.getByText('Saving...')).toBeInTheDocument()
    })
  })

  describe('Validation', () => {
    it('should require script command when no platform overrides', async () => {
      const user = userEvent.setup()

      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const enableToggle = screen.getByLabelText(/Enable this hook/i)
      await user.click(enableToggle)

      // Leave script empty
      const saveButton = screen.getByRole('button', { name: /Save Hook/i })
      await user.click(saveButton)

      await waitFor(() => {
        expect(screen.getByText(/Script command is required/i)).toBeInTheDocument()
      })

      expect(mockOnSave).not.toHaveBeenCalled()
    })

    it('should not require base script when both platforms have overrides', async () => {
      const user = userEvent.setup()

      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const enableToggle = screen.getByLabelText(/Enable this hook/i)
      await user.click(enableToggle)

      // Enable both platform overrides
      const windowsToggle = screen.getByLabelText(/Windows Override/i)
      await user.click(windowsToggle)

      const posixToggle = screen.getByLabelText(/POSIX Override/i)
      await user.click(posixToggle)

      // Fill in platform scripts - use getAllByPlaceholderText since there are multiple
      const windowsInput = screen.getByPlaceholderText(/setup\.ps1/i)
      await user.type(windowsInput, '.\\setup.ps1')

      const posixInputs = screen.getAllByPlaceholderText(/setup\.sh/i)
      // The second one is the POSIX override input (first is base config)
      await user.type(posixInputs[1], './setup.sh')

      // Base script is optional now
      const saveButton = screen.getByRole('button', { name: /Save Hook/i })
      await user.click(saveButton)

      await waitFor(() => {
        expect(mockOnSave).toHaveBeenCalled()
      })
    })
  })

  describe('Cancel Button', () => {
    it('should call onClose when cancel is clicked', async () => {
      const user = userEvent.setup()

      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const cancelButton = screen.getByRole('button', { name: /Cancel/i })
      await user.click(cancelButton)

      expect(mockOnClose).toHaveBeenCalled()
    })

    it('should not call onSave when cancel is clicked', async () => {
      const user = userEvent.setup()

      render(
        <HooksConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const cancelButton = screen.getByRole('button', { name: /Cancel/i })
      await user.click(cancelButton)

      expect(mockOnSave).not.toHaveBeenCalled()
    })
  })
})
