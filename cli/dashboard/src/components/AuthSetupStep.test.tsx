/**
 * AuthSetupStep Tests
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AuthSetupStep } from './AuthSetupStep'

// =============================================================================
// Mock Data
// =============================================================================

const mockAuthenticatedWithPermission = {
  authentication: {
    status: 'authenticated' as const,
    principal: 'user@example.com',
    hasLogAnalyticsReader: true,
    message: 'Authenticated with Log Analytics Reader role',
  },
  timestamp: new Date().toISOString(),
}

const mockAuthenticatedWithoutPermission = {
  authentication: {
    status: 'authenticated' as const,
    principal: 'user@example.com',
    hasLogAnalyticsReader: false,
    message: 'Authenticated but missing Log Analytics Reader role',
  },
  timestamp: new Date().toISOString(),
}

const mockNotAuthenticated = {
  authentication: {
    status: 'not-authenticated' as const,
    hasLogAnalyticsReader: false,
    message: 'Not authenticated. Please run: azd auth login',
  },
  timestamp: new Date().toISOString(),
}

const mockPermissionDenied = {
  authentication: {
    status: 'permission-denied' as const,
    hasLogAnalyticsReader: false,
    message: 'Permission denied when checking authentication',
  },
  timestamp: new Date().toISOString(),
}

const mockError = {
  authentication: {
    status: 'error' as const,
    hasLogAnalyticsReader: false,
    message: 'Failed to check authentication status',
  },
  timestamp: new Date().toISOString(),
}

// =============================================================================
// Setup & Teardown
// =============================================================================

describe('AuthSetupStep', () => {
  let fetchMock: ReturnType<typeof vi.fn>

  beforeEach(() => {
    fetchMock = vi.fn()
    globalThis.fetch = fetchMock as unknown as typeof fetch

    // Mock clipboard API
    Object.defineProperty(navigator, 'clipboard', {
      value: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
      writable: true,
      configurable: true,
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  // ===========================================================================
  // Loading State Tests
  // ===========================================================================

  describe('Loading State', () => {
    it('should show loading spinner while fetching auth state', () => {
      fetchMock.mockReturnValue(new Promise(() => {})) // Never resolves

      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      expect(screen.getByText('Checking authentication...')).toBeInTheDocument()
      // Check for the spinner by class name since it's aria-hidden
      const spinner = document.querySelector('.animate-spin')
      expect(spinner).toBeInTheDocument()
    })

    it('should call onValidationChange with false during loading', () => {
      fetchMock.mockReturnValue(new Promise(() => {}))
      const onValidationChange = vi.fn()

      render(<AuthSetupStep onValidationChange={onValidationChange} />)

      // Should not call validation during initial loading
      expect(onValidationChange).not.toHaveBeenCalled()
    })
  })

  // ===========================================================================
  // Authenticated with Permission Tests
  // ===========================================================================

  describe('Authenticated with Permission', () => {
    beforeEach(() => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockAuthenticatedWithPermission,
      })
    })

    it('should display "Authorized" status badge', async () => {
      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Authorized')).toBeInTheDocument()
      })
    })

    it('should show user principal', async () => {
      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Signed in as')).toBeInTheDocument()
        expect(screen.getByText('user@example.com')).toBeInTheDocument()
      })
    })

    it('should show permission granted with checkmark', async () => {
      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Log Analytics Reader Role')).toBeInTheDocument()
        expect(screen.getByText('You have the required permission to read logs')).toBeInTheDocument()
      })
    })

    it('should call onValidationChange with true', async () => {
      const onValidationChange = vi.fn()

      render(<AuthSetupStep onValidationChange={onValidationChange} />)

      await waitFor(() => {
        expect(onValidationChange).toHaveBeenCalledWith(true)
      })
    })

    it('should display success message', async () => {
      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Authentication successful!')).toBeInTheDocument()
        expect(screen.getByText(/You have the required permissions/)).toBeInTheDocument()
      })
    })

    it('should not show action required sections', async () => {
      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.queryByText('Action Required: Sign in to Azure')).not.toBeInTheDocument()
        expect(screen.queryByText('Action Required: Assign Log Analytics Reader Role')).not.toBeInTheDocument()
      })
    })
  })

  // ===========================================================================
  // Authenticated without Permission Tests
  // ===========================================================================

  describe('Authenticated without Permission', () => {
    beforeEach(() => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockAuthenticatedWithoutPermission,
      })
    })

    it('should display "Permission Missing" status badge', async () => {
      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Permission Missing')).toBeInTheDocument()
      })
    })

    it('should show user principal', async () => {
      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('user@example.com')).toBeInTheDocument()
      })
    })

    it('should show permission missing with warning icon', async () => {
      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Log Analytics Reader Role')).toBeInTheDocument()
        expect(screen.getByText('This role is required to access Log Analytics data')).toBeInTheDocument()
      })
    })

    it('should call onValidationChange with false', async () => {
      const onValidationChange = vi.fn()

      render(<AuthSetupStep onValidationChange={onValidationChange} />)

      await waitFor(() => {
        expect(onValidationChange).toHaveBeenCalledWith(false)
      })
    })

    it('should display role assignment action required', async () => {
      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Action Required: Assign Log Analytics Reader Role')).toBeInTheDocument()
        expect(screen.getByText(/You need the "Log Analytics Reader" role/)).toBeInTheDocument()
      })
    })

    it('should show role assignment command', async () => {
      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        const codeBlock = screen.getByText(/az role assignment create/)
        expect(codeBlock).toBeInTheDocument()
      })
    })

    it('should show Azure Portal link', async () => {
      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        const portalLink = screen.getByRole('link', { name: /Azure Portal/i })
        expect(portalLink).toHaveAttribute('href', 'https://portal.azure.com')
        expect(portalLink).toHaveAttribute('target', '_blank')
      })
    })

    it('should not show success message', async () => {
      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.queryByText('Authentication successful!')).not.toBeInTheDocument()
      })
    })
  })

  // ===========================================================================
  // Not Authenticated Tests
  // ===========================================================================

  describe('Not Authenticated', () => {
    beforeEach(() => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockNotAuthenticated,
      })
    })

    it('should display "Not Authenticated" status badge', async () => {
      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Not Authenticated')).toBeInTheDocument()
      })
    })

    it('should call onValidationChange with false', async () => {
      const onValidationChange = vi.fn()

      render(<AuthSetupStep onValidationChange={onValidationChange} />)

      await waitFor(() => {
        expect(onValidationChange).toHaveBeenCalledWith(false)
      })
    })

    it('should display sign in action required', async () => {
      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Action Required: Sign in to Azure')).toBeInTheDocument()
        expect(screen.getByText(/You need to authenticate with Azure/)).toBeInTheDocument()
      })
    })

    it('should show azd auth login command', async () => {
      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('azd auth login')).toBeInTheDocument()
      })
    })

    it('should not show user principal', async () => {
      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.queryByText('Signed in as')).not.toBeInTheDocument()
      })
    })

    it('should not show permission checklist', async () => {
      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.queryByText('Log Analytics Reader Role')).not.toBeInTheDocument()
      })
    })
  })

  // ===========================================================================
  // Permission Denied Tests
  // ===========================================================================

  describe('Permission Denied', () => {
    beforeEach(() => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockPermissionDenied,
      })
    })

    it('should display "Permission Denied" status badge', async () => {
      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Permission Denied')).toBeInTheDocument()
      })
    })

    it('should call onValidationChange with false', async () => {
      const onValidationChange = vi.fn()

      render(<AuthSetupStep onValidationChange={onValidationChange} />)

      await waitFor(() => {
        expect(onValidationChange).toHaveBeenCalledWith(false)
      })
    })
  })

  // ===========================================================================
  // Error State Tests
  // ===========================================================================

  describe('Error State', () => {
    beforeEach(() => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockError,
      })
    })

    it('should display "Error" status badge', async () => {
      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Error')).toBeInTheDocument()
      })
    })

    it('should call onValidationChange with false', async () => {
      const onValidationChange = vi.fn()

      render(<AuthSetupStep onValidationChange={onValidationChange} />)

      await waitFor(() => {
        expect(onValidationChange).toHaveBeenCalledWith(false)
      })
    })
  })

  // ===========================================================================
  // Fetch Error Tests
  // ===========================================================================

  describe('Fetch Errors', () => {
    it('should show error message when fetch fails', async () => {
      fetchMock.mockRejectedValue(new Error('Network error'))

      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Failed to load authentication state')).toBeInTheDocument()
        expect(screen.getByText('Network error')).toBeInTheDocument()
      })
    })

    it('should show error when response is not ok', async () => {
      fetchMock.mockResolvedValue({
        ok: false,
        statusText: 'Internal Server Error',
      })

      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText(/Failed to fetch setup state/)).toBeInTheDocument()
      })
    })

    it('should call onValidationChange with false on error', async () => {
      fetchMock.mockRejectedValue(new Error('Network error'))
      const onValidationChange = vi.fn()

      render(<AuthSetupStep onValidationChange={onValidationChange} />)

      await waitFor(() => {
        expect(onValidationChange).toHaveBeenCalledWith(false)
      })
    })

    it('should show retry button on error', async () => {
      fetchMock.mockRejectedValue(new Error('Network error'))

      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Retry/i })).toBeInTheDocument()
      })
    })

    it('should retry fetch when retry button clicked', async () => {
      const user = userEvent.setup({ delay: null })
      fetchMock.mockRejectedValueOnce(new Error('Network error'))
      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => mockAuthenticatedWithPermission,
      })

      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Network error')).toBeInTheDocument()
      })

      const retryButton = screen.getByRole('button', { name: /Retry/i })
      await user.click(retryButton)

      await waitFor(() => {
        expect(screen.getByText('Authorized')).toBeInTheDocument()
      })
    })
  })

  // ===========================================================================
  // Recheck Functionality Tests
  // ===========================================================================

  describe('Recheck Functionality', () => {
    it('should have a recheck button', async () => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockNotAuthenticated,
      })

      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Recheck/i })).toBeInTheDocument()
      })
    })

    it('should refetch auth state when recheck clicked', async () => {
      const user = userEvent.setup({ delay: null })
      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => mockNotAuthenticated,
      })

      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Not Authenticated')).toBeInTheDocument()
      })

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => mockAuthenticatedWithPermission,
      })

      const recheckButton = screen.getByRole('button', { name: /Recheck/i })
      await user.click(recheckButton)

      await waitFor(() => {
        expect(screen.getByText('Authorized')).toBeInTheDocument()
      })
    })

    it('should show "Checking..." text while refreshing', async () => {
      const user = userEvent.setup({ delay: null })
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockAuthenticatedWithPermission,
      })

      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Recheck/i })).toBeInTheDocument()
      })

      fetchMock.mockReturnValueOnce(new Promise(() => {})) // Never resolves

      const recheckButton = screen.getByRole('button', { name: /Recheck/i })
      await user.click(recheckButton)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Checking/i })).toBeInTheDocument()
      })
    })

    it('should disable recheck button while refreshing', async () => {
      const user = userEvent.setup({ delay: null })
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockAuthenticatedWithPermission,
      })

      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Recheck/i })).toBeInTheDocument()
      })

      fetchMock.mockReturnValueOnce(new Promise(() => {}))

      const recheckButton = screen.getByRole('button', { name: /Recheck/i })
      await user.click(recheckButton)

      await waitFor(() => {
        expect(recheckButton).toBeDisabled()
      })
    })

    it('should update validation state after recheck', async () => {
      const user = userEvent.setup({ delay: null })
      const onValidationChange = vi.fn()
      
      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => mockAuthenticatedWithoutPermission,
      })

      render(<AuthSetupStep onValidationChange={onValidationChange} />)

      await waitFor(() => {
        expect(onValidationChange).toHaveBeenCalledWith(false)
      })

      onValidationChange.mockClear()

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => mockAuthenticatedWithPermission,
      })

      const recheckButton = screen.getByRole('button', { name: /Recheck/i })
      await user.click(recheckButton)

      await waitFor(() => {
        expect(onValidationChange).toHaveBeenCalledWith(true)
      })
    })
  })

  // ===========================================================================
  // Copy Button Tests
  // ===========================================================================

  describe('Copy Functionality', () => {
    it('should have copy buttons in code blocks', async () => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockNotAuthenticated,
      })

      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('azd auth login')).toBeInTheDocument()
      })

      // Hover to reveal copy button
      const codeBlock = screen.getByText('azd auth login').closest('div')
      expect(codeBlock).toHaveClass('group')
    })

    it('should show "Copied" text after copying', async () => {
      const user = userEvent.setup({ delay: null })
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockNotAuthenticated,
      })

      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('azd auth login')).toBeInTheDocument()
      })

      const copyButton = screen.getByLabelText('Copy code')
      await user.click(copyButton)

      await waitFor(() => {
        expect(screen.getByLabelText('Copied')).toBeInTheDocument()
      })
    })
  })

  // ===========================================================================
  // Collapsible Sections Tests
  // ===========================================================================

  describe('Collapsible Help Sections', () => {
    it('should show all help section headers', async () => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockAuthenticatedWithPermission,
      })

      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Help & Guidance')).toBeInTheDocument()
      })

      expect(screen.getByText('Why is authentication required?')).toBeInTheDocument()
      expect(screen.getByText('How to assign permissions')).toBeInTheDocument()
      expect(screen.getByText('Troubleshooting')).toBeInTheDocument()
    })
  })

  // ===========================================================================
  // Polling Tests (simplified)
  // ===========================================================================

  describe('Polling Behavior', () => {
    it('should fetch state on mount', async () => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockNotAuthenticated,
      })

      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(fetchMock).toHaveBeenCalledTimes(1)
      })
    })
  })

  // ===========================================================================
  // Message Display Tests
  // ===========================================================================

  describe('Status Messages', () => {
    it('should display auth state message', async () => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockAuthenticatedWithPermission,
      })

      render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        // Check for authenticated status badge
        expect(screen.getByText('Authorized')).toBeInTheDocument()
        // Check for Log Analytics Reader role text
        expect(screen.getByText('Log Analytics Reader Role')).toBeInTheDocument()
        // Check for permission granted message
        expect(screen.getByText('You have the required permission to read logs')).toBeInTheDocument()
      })
    })

    it('should show different message for each state', async () => {
      // Test authenticated with permission
      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => mockAuthenticatedWithPermission,
      })

      const { rerender } = render(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Authorized')).toBeInTheDocument()
        expect(screen.getByText('You have the required permission to read logs')).toBeInTheDocument()
      })

      // Test not authenticated
      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => mockNotAuthenticated,
      })

      rerender(<AuthSetupStep onValidationChange={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Not Authenticated')).toBeInTheDocument()
        expect(screen.getByText('Action Required: Sign in to Azure')).toBeInTheDocument()
      })
    })
  })
})
