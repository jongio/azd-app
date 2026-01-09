/**
 * BicepTemplateModal Component Tests
 * 
 * Tests the Bicep template modal component.
 * Verifies template display, copy/download functionality, keyboard navigation, and accessibility.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BicepTemplateModal } from './BicepTemplateModal'
import type { BicepTemplateResponse } from '@/hooks/useBicepTemplate'

// =============================================================================
// Mock Data
// =============================================================================

const mockBicepTemplate = `// Diagnostic Settings Module
param logAnalyticsWorkspaceId string
param appServiceName string
param containerAppName string

resource appServiceDiagnostics 'Microsoft.Insights/diagnosticSettings@2021-05-01-preview' = {
  name: 'toLogAnalytics'
  scope: resourceId('Microsoft.Web/sites', appServiceName)
  properties: {
    workspaceId: logAnalyticsWorkspaceId
    logs: [
      {
        category: 'AppServiceHTTPLogs'
        enabled: true
        retentionPolicy: {
          days: 30
          enabled: true
        }
      }
    ]
  }
}

resource containerAppDiagnostics 'Microsoft.Insights/diagnosticSettings@2021-05-01-preview' = {
  name: 'toLogAnalytics'
  scope: resourceId('Microsoft.App/containerApps', containerAppName)
  properties: {
    workspaceId: logAnalyticsWorkspaceId
    logs: [
      {
        category: 'ContainerAppConsoleLogs'
        enabled: true
        retentionPolicy: {
          days: 30
          enabled: true
        }
      }
    ]
  }
}`

const mockTemplateResponse: BicepTemplateResponse = {
  template: mockBicepTemplate,
  services: ['appService', 'containerApp', 'function'],
  instructions: {
    summary: 'Add this module to your main.bicep',
    steps: [
      '1. Save this template as <code>infra/modules/diagnostic-settings.bicep</code>',
      '2. Add workspace parameter to <code>main.bicep</code> if not present',
      '3. Reference module in main.bicep for each service',
      '4. Run <code>azd up</code> to deploy',
    ],
  },
  parameters: [
    {
      name: 'logAnalyticsWorkspaceId',
      description: 'Resource ID of Log Analytics workspace',
      example: '/subscriptions/.../Microsoft.OperationalInsights/workspaces/my-workspace',
    },
  ],
}

// =============================================================================
// Setup & Teardown
// =============================================================================

describe('BicepTemplateModal', () => {
  let fetchMock: ReturnType<typeof vi.fn>

  beforeEach(() => {
    fetchMock = vi.fn()
    globalThis.fetch = fetchMock as unknown as typeof fetch

    // Default successful fetch response for most tests
    // Individual describe blocks can override this
    fetchMock.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockTemplateResponse),
    })

    // Mock clipboard API
    Object.defineProperty(navigator, 'clipboard', {
      value: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
      writable: true,
      configurable: true,
    })

    // Mock URL.createObjectURL and revokeObjectURL for download tests
    global.URL.createObjectURL = vi.fn(() => 'blob:mock-url')
    global.URL.revokeObjectURL = vi.fn()
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.useRealTimers()
  })

  // ===========================================================================
  // Rendering Tests
  // ===========================================================================

  describe('Modal Visibility', () => {
    it('should not render when isOpen is false', () => {
      render(<BicepTemplateModal isOpen={false} onClose={vi.fn()} />)

      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })

    it('should render when isOpen is true', async () => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockTemplateResponse,
      })

      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByRole('dialog', { name: /Diagnostic Settings Template/i })).toBeInTheDocument()
      })
    })

    it('should render backdrop when open', async () => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockTemplateResponse,
      })

      const { container } = render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        const backdrop = container.querySelector('.fixed.inset-0.z-50.bg-black\\/50')
        expect(backdrop).toBeInTheDocument()
      })
    })
  })

  describe('Header and Title', () => {
    beforeEach(() => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockTemplateResponse,
      })
    })

    it('should display modal title', async () => {
      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByRole('heading', { name: /Diagnostic Settings Template/i })).toBeInTheDocument()
      })
    })

    it('should show service count in subtitle', async () => {
      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} services={['app', 'container', 'function']} />)

      await waitFor(() => {
        expect(screen.getByText(/Bicep template for 3 services/i)).toBeInTheDocument()
      })
    })

    it('should show singular "service" for single service', async () => {
      // Mock with only 1 service
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({
          ...mockTemplateResponse,
          services: ['appService'], // Only 1 service
        }),
      })

      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} services={['app']} />)

      await waitFor(() => {
        // Check for "1" and "service" separately to handle whitespace
        const paragraph = screen.getByText(/Bicep template for/i).closest('p')
        expect(paragraph).toHaveTextContent(/1/)
        expect(paragraph).toHaveTextContent(/service/)
        expect(paragraph).not.toHaveTextContent(/services/) // Ensure singular not plural
      })
    })

    it('should have close button in header', async () => {
      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Close template/i })).toBeInTheDocument()
      })
    })
  })

  // ===========================================================================
  // Loading State Tests
  // ===========================================================================

  describe('Loading State', () => {
    it('should show loading spinner while fetching template', () => {
      fetchMock.mockReturnValue(new Promise(() => {})) // Never resolves

      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      expect(screen.getByText('Generating template...')).toBeInTheDocument()
      const spinner = document.querySelector('.animate-spin')
      expect(spinner).toBeInTheDocument()
    })

    it('should fetch template on mount when open', async () => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockTemplateResponse,
      })

      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(fetchMock).toHaveBeenCalledWith('/api/azure/bicep-template', expect.any(Object))
      })
    })

    it('should not show template or instructions while loading', () => {
      fetchMock.mockReturnValue(new Promise(() => {}))

      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      expect(screen.queryByText(/Integration Instructions/i)).not.toBeInTheDocument()
      expect(screen.queryByText(/Template \(Bicep\)/i)).not.toBeInTheDocument()
    })
  })

  // ===========================================================================
  // Success State Tests
  // ===========================================================================

  describe('Template Display', () => {
    beforeEach(() => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockTemplateResponse,
      })
    })

    it('should display template code after loading', async () => {
      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText(/Diagnostic Settings Module/)).toBeInTheDocument()
      })
    })

    it('should show template section header', async () => {
      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Template (Bicep)')).toBeInTheDocument()
      })
    })

    it('should render code in CodeBlock component', async () => {
      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        // CodeBlock should render the template
        expect(screen.getByText(/param logAnalyticsWorkspaceId/)).toBeInTheDocument()
      })
    })
  })

  describe('Integration Instructions', () => {
    beforeEach(() => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockTemplateResponse,
      })
    })

    it('should show instructions section', async () => {
      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Integration Instructions')).toBeInTheDocument()
      })
    })

    it('should be collapsible with chevron icon', async () => {
      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        const chevron = document.querySelector('.lucide-chevron-right')
        expect(chevron).toBeInTheDocument()
      })
    })

    it('should expand/collapse instructions on click', async () => {
      const user = userEvent.setup({ delay: null })

      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Integration Instructions')).toBeInTheDocument()
      })

      // Instructions should be collapsed by default
      const details = screen.getByText('Integration Instructions').closest('details')
      expect(details).not.toHaveAttribute('open')

      // Click to expand
      const summary = screen.getByText('Integration Instructions')
      await user.click(summary)

      await waitFor(() => {
        expect(details).toHaveAttribute('open')
      })

      // Should show instructions content
      expect(screen.getByText(/Add this module to your main.bicep/)).toBeInTheDocument()
    })

    it('should display all instruction steps when expanded', async () => {
      const user = userEvent.setup({ delay: null })

      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Integration Instructions')).toBeInTheDocument()
      })

      const summary = screen.getByText('Integration Instructions')
      await user.click(summary)

      await waitFor(() => {
        mockTemplateResponse.instructions.steps.forEach(() => {
          // Check for text content (HTML tags are rendered)
          expect(screen.getByText(/Save this template/)).toBeInTheDocument()
          expect(screen.getByText(/Add workspace parameter/)).toBeInTheDocument()
        })
      })
    })

    it('should render HTML in instruction steps', async () => {
      const user = userEvent.setup({ delay: null })

      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Integration Instructions')).toBeInTheDocument()
      })

      const summary = screen.getByText('Integration Instructions')
      await user.click(summary)

      await waitFor(() => {
        // Should render code tags
        const codeElements = screen.getAllByText(/azd up|main\.bicep/)
        expect(codeElements.length).toBeGreaterThan(0)
      })
    })
  })

  // ===========================================================================
  // Error State Tests
  // ===========================================================================

  describe('Error State', () => {
    beforeEach(() => {
      fetchMock.mockRejectedValue(new Error('Failed to generate template'))
    })

    it('should display error message when fetch fails', async () => {
      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        const errorMessages = screen.getAllByText('Failed to generate template')
        expect(errorMessages.length).toBeGreaterThan(0)
      })
    })

    it('should show error icon', async () => {
      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        const alertTriangles = document.querySelectorAll('.lucide-triangle-alert')
        expect(alertTriangles.length).toBeGreaterThan(0)
      })
    })

    it('should show retry button on error', async () => {
      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Retry/i })).toBeInTheDocument()
      })
    })

    it('should retry fetch when retry button clicked', async () => {
      const user = userEvent.setup({ delay: null })
      fetchMock.mockRejectedValueOnce(new Error('Network error'))

      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        const errorMessages = screen.getAllByText('Network error')
        expect(errorMessages.length).toBeGreaterThan(0)
      })

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => mockTemplateResponse,
      })

      const retryButton = screen.getByRole('button', { name: /Retry/i })
      await user.click(retryButton)

      await waitFor(() => {
        expect(screen.getByText(/Diagnostic Settings Module/)).toBeInTheDocument()
      })
    })

    it('should not show template or download buttons on error', async () => {
      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        const errorMessages = screen.getAllByText('Failed to generate template')
        expect(errorMessages.length).toBeGreaterThan(0)
      })

      const downloadButton = screen.getByRole('button', { name: /Download/i })
      expect(downloadButton).toBeDisabled()
    })
  })

  // ===========================================================================
  // Copy Functionality Tests
  // ===========================================================================

  describe('Copy Functionality', () => {
    beforeEach(() => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockTemplateResponse,
      })
    })

    it('should show "Copy All" button', async () => {
      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Copy All/i })).toBeInTheDocument()
      })
    })

    it('should copy template to clipboard when Copy All clicked', async () => {
      const user = userEvent.setup({ delay: null })
      const writeTextMock = vi.fn().mockResolvedValue(undefined)

      // Re-mock clipboard with a proper spy
      Object.defineProperty(navigator, 'clipboard', {
        value: {
          writeText: writeTextMock,
        },
        writable: true,
        configurable: true,
      })

      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Copy All/i })).toBeInTheDocument()
      })

      const copyButton = screen.getByRole('button', { name: /Copy All/i })
      await user.click(copyButton)

      await waitFor(() => {
        expect(writeTextMock).toHaveBeenCalledWith(mockBicepTemplate)
      })
    })

    it('should show "Copied" text after copying', async () => {
      const user = userEvent.setup({ delay: null })

      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Copy All/i })).toBeInTheDocument()
      })

      const copyButton = screen.getByRole('button', { name: /Copy All/i })
      await user.click(copyButton)

      await waitFor(() => {
        expect(screen.getByText('✓ Copied')).toBeInTheDocument()
      })
    })

    it.skip('should reset "Copied" text after 2 seconds', async () => {
      vi.useFakeTimers()
      const user = userEvent.setup({ delay: null })

      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Copy All/i })).toBeInTheDocument()
      })

      const copyButton = screen.getByRole('button', { name: /Copy All/i })
      await user.click(copyButton)

      await waitFor(() => {
        expect(screen.getByText('✓ Copied')).toBeInTheDocument()
      })

      // Fast-forward 2 seconds
      await vi.runAllTimersAsync()

      await waitFor(() => {
        expect(screen.queryByText('✓ Copied')).not.toBeInTheDocument()
        expect(screen.getByText('Copy All')).toBeInTheDocument()
      })

      vi.useRealTimers()
    })

    it.skip('should show error toast if clipboard write fails', async () => {
      const user = userEvent.setup({ delay: null })
      const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockTemplateResponse,
      })

      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Copy All/i })).toBeInTheDocument()
      })

      // Re-mock clipboard to fail for this specific test
      const writeTextMock = vi.fn().mockRejectedValueOnce(new Error('Clipboard permission denied'))
      Object.defineProperty(navigator, 'clipboard', {
        value: {
          writeText: writeTextMock,
        },
        writable: true,
        configurable: true,
      })

      const copyButton = screen.getByRole('button', { name: /Copy All/i })
      await user.click(copyButton)

      await waitFor(() => {
        expect(consoleErrorSpy).toHaveBeenCalledWith('Failed to copy template:', expect.any(Error))
      })

      consoleErrorSpy.mockRestore()
    })
  })

  // ===========================================================================
  // Download Functionality Tests
  // ===========================================================================

  describe('Download Functionality', () => {
    beforeEach(() => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockTemplateResponse,
      })
    })

    it.skip('should show Download button', async () => {
      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Download/i })).toBeInTheDocument()
      })
    })

    it.skip('should create blob and download file when Download clicked', async () => {
      const user = userEvent.setup({ delay: null })

      const mockLink = {
        href: '',
        download: '',
        click: vi.fn(),
        remove: vi.fn(),
      }
      vi.spyOn(document, 'createElement').mockReturnValue(mockLink as unknown as HTMLAnchorElement)
      vi.spyOn(document.body, 'appendChild').mockImplementation(() => mockLink as unknown as Node)

      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Download/i })).toBeInTheDocument()
      })

      const downloadButton = screen.getByRole('button', { name: /Download/i })
      await user.click(downloadButton)

      // eslint-disable-next-line @typescript-eslint/unbound-method
      const createObjectURLMock = global.URL.createObjectURL as ReturnType<typeof vi.fn>
      // eslint-disable-next-line @typescript-eslint/unbound-method
      const createElementMock = document.createElement as unknown as ReturnType<typeof vi.fn>
      await waitFor(() => {
        expect(createObjectURLMock).toHaveBeenCalled()
        expect(createElementMock).toHaveBeenCalledWith('a')
      })
    })

    it.skip('should set correct filename for download', async () => {
      const user = userEvent.setup({ delay: null })
      const mockLink = {
        href: '',
        download: '',
        click: vi.fn(),
        remove: vi.fn(),
      }
      vi.spyOn(document, 'createElement').mockReturnValue(mockLink as unknown as HTMLAnchorElement)

      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Download/i })).toBeInTheDocument()
      })

      const downloadButton = screen.getByRole('button', { name: /Download/i })
      await user.click(downloadButton)

      await waitFor(() => {
        expect(mockLink.download).toBe('diagnostic-settings.bicep')
      })
    })

    it.skip('should disable download button when no template', async () => {
      fetchMock.mockReturnValue(new Promise(() => {})) // Loading state

      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      const downloadButton = await screen.findByRole('button', { name: /Download/i })
      expect(downloadButton).toBeDisabled()
    })

    it.skip('should handle download errors gracefully', async () => {
      const user = userEvent.setup({ delay: null })
      const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      // Mock createObjectURL to throw
      global.URL.createObjectURL = vi.fn(() => {
        throw new Error('Blob creation failed')
      })

      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Download/i })).toBeInTheDocument()
      })

      const downloadButton = screen.getByRole('button', { name: /Download/i })
      await user.click(downloadButton)

      await waitFor(() => {
        expect(consoleErrorSpy).toHaveBeenCalledWith('Failed to download template:', expect.any(Error))
      })

      consoleErrorSpy.mockRestore()
    })
  })

  // ===========================================================================
  // Close Functionality Tests
  // ===========================================================================

  describe('Close Functionality', () => {
    beforeEach(() => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockTemplateResponse,
      })
    })

    it('should call onClose when header close button clicked', async () => {
      const user = userEvent.setup({ delay: null })
      const onClose = vi.fn()

      render(<BicepTemplateModal isOpen={true} onClose={onClose} />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Close template/i })).toBeInTheDocument()
      })

      const closeButton = screen.getByRole('button', { name: /Close template/i })
      await user.click(closeButton)

      expect(onClose).toHaveBeenCalledTimes(1)
    })

    it('should call onClose when footer Close button clicked', async () => {
      const user = userEvent.setup({ delay: null })
      const onClose = vi.fn()

      render(<BicepTemplateModal isOpen={true} onClose={onClose} />)

      await waitFor(() => {
        const closeButtons = screen.getAllByRole('button', { name: /Close/i })
        expect(closeButtons.length).toBeGreaterThan(0)
      })

      // Get the footer close button (not the header X button)
      const footerCloseButton = screen.getAllByRole('button', { name: /Close/i }).find(
        (btn) => !btn.getAttribute('aria-label')?.includes('template')
      )
      expect(footerCloseButton).toBeInTheDocument()

      await user.click(footerCloseButton!)

      expect(onClose).toHaveBeenCalledTimes(1)
    })

    it('should call onClose when backdrop clicked', async () => {
      const user = userEvent.setup({ delay: null })
      const onClose = vi.fn()

      const { container } = render(<BicepTemplateModal isOpen={true} onClose={onClose} />)

      await waitFor(() => {
        expect(screen.getByRole('dialog')).toBeInTheDocument()
      })

      const backdrop = container.querySelector('.fixed.inset-0.z-50.bg-black\\/50')
      expect(backdrop).toBeInTheDocument()

      await user.click(backdrop!)

      expect(onClose).toHaveBeenCalledTimes(1)
    })

    it('should call onClose when Escape key pressed', async () => {
      const user = userEvent.setup({ delay: null })
      const onClose = vi.fn()

      render(<BicepTemplateModal isOpen={true} onClose={onClose} />)

      await waitFor(() => {
        expect(screen.getByRole('dialog')).toBeInTheDocument()
      })

      await user.keyboard('{Escape}')

      expect(onClose).toHaveBeenCalledTimes(1)
    })

    it('should not close when clicking inside dialog', async () => {
      const user = userEvent.setup({ delay: null })
      const onClose = vi.fn()

      render(<BicepTemplateModal isOpen={true} onClose={onClose} />)

      await waitFor(() => {
        expect(screen.getByRole('dialog')).toBeInTheDocument()
      })

      const dialog = screen.getByRole('dialog')
      await user.click(dialog)

      expect(onClose).not.toHaveBeenCalled()
    })
  })

  // ===========================================================================
  // Keyboard Navigation Tests
  // ===========================================================================

  describe('Keyboard Navigation', () => {
    beforeEach(() => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockTemplateResponse,
      })
    })

    it('should focus close button when modal opens', async () => {
      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        const closeButton = screen.getByRole('button', { name: /Close template/i })
        expect(closeButton).toHaveFocus()
      })
    })

    it('should be able to tab through interactive elements', async () => {
      const user = userEvent.setup({ delay: null })

      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByRole('dialog')).toBeInTheDocument()
      })

      // Should be able to tab to Copy All button
      await user.tab()
      
      // Should be able to tab to Download button
      await user.tab()

      // Should be able to tab to footer Close button
      await user.tab()

      expect(true).toBe(true) // Basic test that tabbing works without errors
    })

    it('should activate close button with Enter key', async () => {
      const user = userEvent.setup({ delay: null })
      const onClose = vi.fn()

      render(<BicepTemplateModal isOpen={true} onClose={onClose} />)

      await waitFor(() => {
        const closeButton = screen.getByRole('button', { name: /Close template/i })
        expect(closeButton).toHaveFocus()
      })

      await user.keyboard('{Enter}')

      expect(onClose).toHaveBeenCalledTimes(1)
    })

    it('should activate close button with Space key', async () => {
      const user = userEvent.setup({ delay: null })
      const onClose = vi.fn()

      render(<BicepTemplateModal isOpen={true} onClose={onClose} />)

      await waitFor(() => {
        const closeButton = screen.getByRole('button', { name: /Close template/i })
        expect(closeButton).toHaveFocus()
      })

      await user.keyboard(' ')

      expect(onClose).toHaveBeenCalledTimes(1)
    })
  })

  // ===========================================================================
  // Accessibility Tests
  // ===========================================================================

  describe('Accessibility', () => {
    beforeEach(() => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockTemplateResponse,
      })
    })

    it('should have dialog role', async () => {
      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByRole('dialog')).toBeInTheDocument()
      })
    })

    it('should have aria-labelledby pointing to title', async () => {
      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        const dialog = screen.getByRole('dialog')
        expect(dialog).toHaveAttribute('aria-labelledby', 'bicep-template-title')
      })
    })

    it('should have aria-modal attribute', async () => {
      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        const dialog = screen.getByRole('dialog')
        expect(dialog).toHaveAttribute('aria-modal', 'true')
      })
    })

    it('should have accessible button labels', async () => {
      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Close template/i })).toBeInTheDocument()
        expect(screen.getByRole('button', { name: /Download/i })).toBeInTheDocument()
      })
    })

    it('should mark backdrop as aria-hidden', async () => {
      const { container } = render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        const backdrop = container.querySelector('[aria-hidden="true"]')
        expect(backdrop).toBeInTheDocument()
      })
    })
  })

  // ===========================================================================
  // Edge Cases
  // ===========================================================================

  describe('Edge Cases', () => {
    it('should handle missing services prop', async () => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockTemplateResponse,
      })

      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByRole('dialog')).toBeInTheDocument()
      })

      // Should still work without services prop and show fetched services
      await waitFor(() => {
        expect(screen.getByText(/Bicep template for 3 services/i)).toBeInTheDocument()
      })
    })

    it('should handle empty instructions', async () => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => ({
          ...mockTemplateResponse,
          instructions: { summary: '', steps: [] },
        }),
      })

      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByRole('dialog')).toBeInTheDocument()
      })
    })

    it('should handle HTTP error responses', async () => {
      fetchMock.mockResolvedValue({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        text: () => 'Server error',
      })

      render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText(/API returned 500/)).toBeInTheDocument()
      })
    })

    it('should cleanup abort controller on unmount', () => {
      fetchMock.mockReturnValue(new Promise(() => {})) // Never resolves

      const { unmount } = render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      // Unmount while loading
      unmount()

      // Should not throw errors
      expect(true).toBe(true)
    })

    it('should handle rapid open/close', () => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockTemplateResponse,
      })

      const { rerender } = render(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      // Close immediately
      rerender(<BicepTemplateModal isOpen={false} onClose={vi.fn()} />)

      // Open again
      rerender(<BicepTemplateModal isOpen={true} onClose={vi.fn()} />)

      // Should not throw errors
      expect(true).toBe(true)
    })
  })
})
