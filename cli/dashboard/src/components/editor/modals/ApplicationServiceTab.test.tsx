/**
 * ApplicationServiceTab Tests
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ApplicationServiceTab } from './ApplicationServiceTab'
import type { ServiceFormData } from '@/lib/editor/wellknown-types'

describe('ApplicationServiceTab', () => {
  const mockOnSubmit = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  // ===========================================================================
  // Rendering Tests
  // ===========================================================================

  describe('Rendering', () => {
    it('should render all form fields', () => {
      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      expect(screen.getByLabelText(/service name/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/host type/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/project path/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/language/i)).toBeInTheDocument()
    })

    it('should render submit button', () => {
      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      expect(screen.getByRole('button', { name: /add service/i })).toBeInTheDocument()
    })

    it('should have default host type selected', () => {
      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      const hostSelect = screen.getByLabelText(/host type/i) as HTMLSelectElement
      expect(hostSelect.value).toBe('containerapp')
    })

    it('should have auto-detect language selected by default', () => {
      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      const languageSelect = screen.getByLabelText(/language/i) as HTMLSelectElement
      expect(languageSelect.value).toBe('')
    })
  })

  // ===========================================================================
  // Default Values Tests
  // ===========================================================================

  describe('Default Values', () => {
    it('should populate form with default values', () => {
      const defaultValues: Partial<ServiceFormData> = {
        name: 'my-api',
        host: 'appservice',
        project: './src/api',
        language: 'node',
      }

      render(
        <ApplicationServiceTab
          onSubmit={mockOnSubmit}
          defaultValues={defaultValues}
        />
      )

      expect(screen.getByLabelText(/service name/i)).toHaveValue('my-api')
      expect(screen.getByLabelText(/host type/i)).toHaveValue('appservice')
      expect(screen.getByLabelText(/project path/i)).toHaveValue('./src/api')
      expect(screen.getByLabelText(/language/i)).toHaveValue('node')
    })
  })

  // ===========================================================================
  // Validation Tests
  // ===========================================================================

  describe('Validation', () => {
    it('should require service name', async () => {
      const user = userEvent.setup()

      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      const submitButton = screen.getByRole('button', { name: /add service/i })
      await user.click(submitButton)

      await waitFor(() => {
        expect(screen.getByText('Service name is required')).toBeInTheDocument()
      })

      expect(mockOnSubmit).not.toHaveBeenCalled()
    })

    it('should validate service name pattern (lowercase, numbers, hyphens only)', async () => {
      const user = userEvent.setup()

      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      const nameInput = screen.getByLabelText(/service name/i)
      await user.type(nameInput, 'MyAPI_Service')

      const submitButton = screen.getByRole('button', { name: /add service/i })
      await user.click(submitButton)

      await waitFor(() => {
        expect(
          screen.getByText(/must contain only lowercase letters, numbers, and hyphens/i)
        ).toBeInTheDocument()
      })

      expect(mockOnSubmit).not.toHaveBeenCalled()
    })

    it('should accept valid service name', async () => {
      const user = userEvent.setup()

      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      const nameInput = screen.getByLabelText(/service name/i)
      await user.type(nameInput, 'my-api-service')

      const projectInput = screen.getByLabelText(/project path/i)
      await user.type(projectInput, './src/api')

      const submitButton = screen.getByRole('button', { name: /add service/i })
      await user.click(submitButton)

      await waitFor(() => {
        expect(mockOnSubmit).toHaveBeenCalled()
      })
    })

    it('should require project path', async () => {
      const user = userEvent.setup()

      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      const nameInput = screen.getByLabelText(/service name/i)
      await user.type(nameInput, 'my-api')

      const submitButton = screen.getByRole('button', { name: /add service/i })
      await user.click(submitButton)

      await waitFor(() => {
        expect(screen.getByText('Project path is required')).toBeInTheDocument()
      })

      expect(mockOnSubmit).not.toHaveBeenCalled()
    })

    it('should mark invalid fields with aria-invalid', async () => {
      const user = userEvent.setup()

      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      const submitButton = screen.getByRole('button', { name: /add service/i })
      await user.click(submitButton)

      await waitFor(() => {
        const nameInput = screen.getByLabelText(/service name/i)
        expect(nameInput).toHaveAttribute('aria-invalid', 'true')
      })
    })
  })

  // ===========================================================================
  // Form Submission Tests
  // ===========================================================================

  describe('Form Submission', () => {
    it('should submit form with required fields', async () => {
      const user = userEvent.setup()

      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      await user.type(screen.getByLabelText(/service name/i), 'my-api')
      await user.type(screen.getByLabelText(/project path/i), './src/api')

      const submitButton = screen.getByRole('button', { name: /add service/i })
      await user.click(submitButton)

      await waitFor(() => {
        expect(mockOnSubmit).toHaveBeenCalledWith(
          expect.objectContaining({
            name: 'my-api',
            host: 'containerapp',
            project: './src/api',
          })
        )
      })
    })

    it('should submit form with all fields', async () => {
      const user = userEvent.setup()

      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      await user.type(screen.getByLabelText(/service name/i), 'my-api')
      await user.selectOptions(screen.getByLabelText(/host type/i), 'appservice')
      await user.type(screen.getByLabelText(/project path/i), './src/api')
      await user.selectOptions(screen.getByLabelText(/language/i), 'node')

      const submitButton = screen.getByRole('button', { name: /add service/i })
      await user.click(submitButton)

      await waitFor(() => {
        expect(mockOnSubmit).toHaveBeenCalledWith(
          expect.objectContaining({
            name: 'my-api',
            host: 'appservice',
            project: './src/api',
            language: 'node',
          })
        )
      })
    })

    it('should omit language when auto-detect selected', async () => {
      const user = userEvent.setup()

      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      await user.type(screen.getByLabelText(/service name/i), 'my-api')
      await user.type(screen.getByLabelText(/project path/i), './src/api')

      const submitButton = screen.getByRole('button', { name: /add service/i })
      await user.click(submitButton)

      await waitFor(() => {
        expect(mockOnSubmit).toHaveBeenCalledWith(
          expect.not.objectContaining({
            language: expect.anything(),
          })
        )
      })
    })
  })

  // ===========================================================================
  // Host Type Selection Tests
  // ===========================================================================

  describe('Host Type Selection', () => {
    it('should render all host type options', () => {
      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      const hostSelect = screen.getByLabelText(/host type/i)
      const options = hostSelect.querySelectorAll('option')

      expect(options).toHaveLength(6)
      expect(options[0]).toHaveTextContent('Azure Container Apps')
      expect(options[1]).toHaveTextContent('Azure App Service')
      expect(options[2]).toHaveTextContent('Azure Functions')
      expect(options[3]).toHaveTextContent('Azure Spring Apps')
      expect(options[4]).toHaveTextContent('Azure Static Web Apps')
      expect(options[5]).toHaveTextContent('Azure Kubernetes Service')
    })

    it('should show Azure Functions hint when selected', async () => {
      const user = userEvent.setup()

      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      await user.selectOptions(screen.getByLabelText(/host type/i), 'function')

      await waitFor(() => {
        expect(
          screen.getByText(/Make sure your project has a host.json file/i)
        ).toBeInTheDocument()
      })
    })

    it('should show Static Web Apps hint when selected', async () => {
      const user = userEvent.setup()

      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      await user.selectOptions(screen.getByLabelText(/host type/i), 'staticwebapp')

      await waitFor(() => {
        expect(
          screen.getByText(/should contain static assets/i)
        ).toBeInTheDocument()
      })
    })

    it('should not show hints for other host types', async () => {
      const user = userEvent.setup()

      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      await user.selectOptions(screen.getByLabelText(/host type/i), 'containerapp')

      await waitFor(() => {
        expect(
          screen.queryByText(/Make sure your project/i)
        ).not.toBeInTheDocument()
      })
    })

    it('should update form value when host type changed', async () => {
      const user = userEvent.setup()

      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      const hostSelect = screen.getByLabelText(/host type/i) as HTMLSelectElement
      expect(hostSelect.value).toBe('containerapp')

      await user.selectOptions(hostSelect, 'aks')

      expect(hostSelect.value).toBe('aks')
    })
  })

  // ===========================================================================
  // Language Selection Tests
  // ===========================================================================

  describe('Language Selection', () => {
    it('should render all language options', () => {
      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      const languageSelect = screen.getByLabelText(/language/i)
      const options = languageSelect.querySelectorAll('option')

      expect(options.length).toBeGreaterThanOrEqual(9)
      expect(Array.from(options).map((o) => o.textContent)).toContain('Auto-detect')
      expect(Array.from(options).map((o) => o.textContent)).toContain('Node.js')
      expect(Array.from(options).map((o) => o.textContent)).toContain('Python')
      expect(Array.from(options).map((o) => o.textContent)).toContain('.NET')
    })

    it('should update form value when language changed', async () => {
      const user = userEvent.setup()

      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      const languageSelect = screen.getByLabelText(/language/i) as HTMLSelectElement
      await user.selectOptions(languageSelect, 'python')

      expect(languageSelect.value).toBe('python')
    })
  })

  // ===========================================================================
  // Submitting State Tests
  // ===========================================================================

  describe('Submitting State', () => {
    it('should disable submit button when submitting', () => {
      render(<ApplicationServiceTab onSubmit={mockOnSubmit} isSubmitting={true} />)

      const submitButton = screen.getByRole('button', { name: /adding service/i })
      expect(submitButton).toBeDisabled()
    })

    it('should show submitting text when submitting', () => {
      render(<ApplicationServiceTab onSubmit={mockOnSubmit} isSubmitting={true} />)

      expect(screen.getByText('Adding Service...')).toBeInTheDocument()
    })

    it('should enable submit button when not submitting', () => {
      render(<ApplicationServiceTab onSubmit={mockOnSubmit} isSubmitting={false} />)

      const submitButton = screen.getByRole('button', { name: /add service/i })
      expect(submitButton).not.toBeDisabled()
    })
  })

  // ===========================================================================
  // Accessibility Tests
  // ===========================================================================

  describe('Accessibility', () => {
    it('should have accessible labels for all inputs', () => {
      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      expect(screen.getByLabelText(/service name/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/host type/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/project path/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/language/i)).toBeInTheDocument()
    })

    it('should associate error messages with inputs via aria-describedby', async () => {
      const user = userEvent.setup()

      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      const submitButton = screen.getByRole('button', { name: /add service/i })
      await user.click(submitButton)

      await waitFor(() => {
        const nameInput = screen.getByLabelText(/service name/i)
        const errorId = nameInput.getAttribute('aria-describedby')
        expect(errorId).toBe('app-service-name-error')
        expect(screen.getByText('Service name is required')).toHaveAttribute('id', errorId!)
      })
    })

    it('should mark required fields with asterisk', () => {
      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      const labels = screen.getAllByText('*', { exact: false })
      expect(labels.length).toBeGreaterThanOrEqual(3) // Service Name, Host Type, Project Path
    })

    it('should have descriptive placeholders', () => {
      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      expect(screen.getByPlaceholderText('my-api')).toBeInTheDocument()
      expect(screen.getByPlaceholderText('./src/api')).toBeInTheDocument()
    })

    it('should have help text for fields', () => {
      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      expect(
        screen.getByText(/Relative path to your application code/i)
      ).toBeInTheDocument()
      expect(
        screen.getByText(/automatically detect the language/i)
      ).toBeInTheDocument()
    })
  })

  // ===========================================================================
  // Edge Cases Tests
  // ===========================================================================

  describe('Edge Cases', () => {
    it('should handle service name with numbers', async () => {
      const user = userEvent.setup()

      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      await user.type(screen.getByLabelText(/service name/i), 'api-v2')
      await user.type(screen.getByLabelText(/project path/i), './src/api-v2')

      const submitButton = screen.getByRole('button', { name: /add service/i })
      await user.click(submitButton)

      await waitFor(() => {
        expect(mockOnSubmit).toHaveBeenCalledWith(
          expect.objectContaining({
            name: 'api-v2',
          })
        )
      })
    })

    it('should handle service name with consecutive hyphens', async () => {
      const user = userEvent.setup()

      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      await user.type(screen.getByLabelText(/service name/i), 'my--api')
      await user.type(screen.getByLabelText(/project path/i), './src/api')

      const submitButton = screen.getByRole('button', { name: /add service/i })
      await user.click(submitButton)

      await waitFor(() => {
        expect(mockOnSubmit).toHaveBeenCalledWith(
          expect.objectContaining({
            name: 'my--api',
          })
        )
      })
    })

    it('should reject service name with uppercase letters', async () => {
      const user = userEvent.setup()

      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      await user.type(screen.getByLabelText(/service name/i), 'MyApi')

      const submitButton = screen.getByRole('button', { name: /add service/i })
      await user.click(submitButton)

      await waitFor(() => {
        expect(
          screen.getByText(/must contain only lowercase letters/i)
        ).toBeInTheDocument()
      })
    })

    it('should reject service name with underscores', async () => {
      const user = userEvent.setup()

      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      await user.type(screen.getByLabelText(/service name/i), 'my_api')

      const submitButton = screen.getByRole('button', { name: /add service/i })
      await user.click(submitButton)

      await waitFor(() => {
        expect(
          screen.getByText(/must contain only lowercase letters/i)
        ).toBeInTheDocument()
      })
    })

    it('should handle project path with spaces', async () => {
      const user = userEvent.setup()

      render(<ApplicationServiceTab onSubmit={mockOnSubmit} />)

      await user.type(screen.getByLabelText(/service name/i), 'my-api')
      await user.type(screen.getByLabelText(/project path/i), './src/my api')

      const submitButton = screen.getByRole('button', { name: /add service/i })
      await user.click(submitButton)

      await waitFor(() => {
        expect(mockOnSubmit).toHaveBeenCalledWith(
          expect.objectContaining({
            project: './src/my api',
          })
        )
      })
    })
  })
})
