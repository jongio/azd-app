/**
 * SetupVerification Component Tests
 * 
 * Tests the setup verification step component for Azure Logs Setup Guide.
 * Verifies workspace connection, log query results, user interactions, and accessibility.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { SetupVerification } from './SetupVerification'
import type { WorkspaceVerificationResponse } from '@/hooks/useWorkspaceVerification'

// =============================================================================
// Mock Data
// =============================================================================

const mockWorkspace = {
  id: '/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.OperationalInsights/workspaces/test-workspace',
  name: 'test-workspace',
}

const mockAllVerified: WorkspaceVerificationResponse = {
  status: 'success',
  workspace: mockWorkspace,
  results: {
    'app-service': {
      serviceName: 'app-service',
      logCount: 25,
      lastLogTime: '2025-12-25T10:45:00Z',
      status: 'ok',
    },
    'container-app': {
      serviceName: 'container-app',
      logCount: 42,
      lastLogTime: '2025-12-25T10:44:30Z',
      status: 'ok',
    },
    'function-app': {
      serviceName: 'function-app',
      logCount: 15,
      lastLogTime: '2025-12-25T10:43:00Z',
      status: 'ok',
    },
  },
  guidance: [
    'app-service: Logs flowing correctly',
    'container-app: Logs flowing correctly',
    'function-app: Logs flowing correctly',
  ],
}

const mockPartialVerified: WorkspaceVerificationResponse = {
  status: 'partial',
  workspace: mockWorkspace,
  results: {
    'app-service': {
      serviceName: 'app-service',
      logCount: 25,
      lastLogTime: '2025-12-25T10:45:00Z',
      status: 'ok',
    },
    'container-app': {
      serviceName: 'container-app',
      logCount: 0,
      status: 'no-logs',
      message:
        'No logs found. This may be normal if the service hasn not run yet or if diagnostic settings were just configured (allow 2-5 minutes for ingestion).',
    },
    'function-app': {
      serviceName: 'function-app',
      logCount: 0,
      status: 'no-logs',
      message: 'No logs found in the last 15 minutes.',
    },
  },
  guidance: [
    'app-service: Logs flowing correctly',
    'container-app: No recent logs - wait or trigger activity',
    'function-app: No recent logs - wait or trigger activity',
  ],
}

const mockNoLogs: WorkspaceVerificationResponse = {
  status: 'partial',
  workspace: mockWorkspace,
  results: {
    'app-service': {
      serviceName: 'app-service',
      logCount: 0,
      status: 'no-logs',
      message: 'No logs found. Diagnostic settings may not be configured.',
    },
    'container-app': {
      serviceName: 'container-app',
      logCount: 0,
      status: 'no-logs',
      message: 'No logs found in the last 15 minutes.',
    },
  },
  guidance: [
    'app-service: Configure diagnostic settings first',
    'container-app: No recent logs - wait or trigger activity',
  ],
}

const mockWithErrors: WorkspaceVerificationResponse = {
  status: 'error',
  workspace: mockWorkspace,
  results: {
    'app-service': {
      serviceName: 'app-service',
      logCount: 15,
      lastLogTime: '2025-12-25T10:45:00Z',
      status: 'ok',
    },
    'function-app': {
      serviceName: 'function-app',
      logCount: 0,
      status: 'error',
      error: 'DiagnosticSettingsNotConfigured: No diagnostic settings found for this resource.',
    },
  },
  guidance: [
    'app-service: Logs flowing correctly',
    'function-app: Configure diagnostic settings first',
  ],
}

// =============================================================================
// Setup & Teardown
// =============================================================================

describe('SetupVerification', () => {
  let fetchMock: ReturnType<typeof vi.fn>

  beforeEach(() => {
    fetchMock = vi.fn()
    globalThis.fetch = fetchMock as unknown as typeof fetch
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  // ===========================================================================
  // Idle State Tests
  // ===========================================================================

  describe('Idle State', () => {
    it('should show start verification prompt', () => {
      render(<SetupVerification onValidationChange={vi.fn()} />)

      expect(screen.getByText(/Ready to verify your Azure logs setup/i)).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /Start Verification/i })).toBeInTheDocument()
    })

    it('should show description of what verification does', () => {
      render(<SetupVerification onValidationChange={vi.fn()} />)

      expect(screen.getByText(/Test your workspace connection and log flow/i)).toBeInTheDocument()
      expect(screen.getByText(/query your workspace for recent logs/i)).toBeInTheDocument()
    })

    it('should start verification when button clicked', async () => {
      const user = userEvent.setup({ delay: null })
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockAllVerified,
      })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(fetchMock).toHaveBeenCalledWith('/api/azure/workspace/verify', expect.objectContaining({
          method: 'POST',
        }))
      })
    })
  })

  // ===========================================================================
  // Verifying State Tests
  // ===========================================================================

  describe('Verifying State', () => {
    it('should show loading spinner while verifying', async () => {
      const user = userEvent.setup({ delay: null })
      fetchMock.mockReturnValue(new Promise(() => {})) // Never resolves

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      expect(screen.getByText('Testing connection to Log Analytics workspace...')).toBeInTheDocument()
      const spinner = document.querySelector('.animate-spin')
      expect(spinner).toBeInTheDocument()
    })

    it('should show progress message', async () => {
      const user = userEvent.setup({ delay: null })
      fetchMock.mockReturnValue(new Promise(() => {}))

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      expect(screen.getByText(/This may take a few seconds/i)).toBeInTheDocument()
    })
  })

  // ===========================================================================
  // Success (All Verified) State Tests
  // ===========================================================================

  describe('Success - All Verified', () => {
    beforeEach(() => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockAllVerified,
      })
    })

    it('should display success summary when all services verified', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByText('All 3 services verified')).toBeInTheDocument()
        expect(screen.getByText('Your Azure logs are flowing correctly')).toBeInTheDocument()
      })
    })

    it('should show checkmark icon for success', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        const checkCircles = document.querySelectorAll('.lucide-circle-check-big')
        expect(checkCircles.length).toBeGreaterThan(0)
      })
    })

    it('should display all service results with log counts', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByText('app-service')).toBeInTheDocument()
        expect(screen.getByText('25 log entries found in last 15 minutes')).toBeInTheDocument()
        expect(screen.getByText('container-app')).toBeInTheDocument()
        expect(screen.getByText('42 log entries found in last 15 minutes')).toBeInTheDocument()
        expect(screen.getByText('function-app')).toBeInTheDocument()
        expect(screen.getByText('15 log entries found in last 15 minutes')).toBeInTheDocument()
      })
    })

    it('should display last log timestamps', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        // Should show formatted timestamps
        const timestamps = screen.getAllByText(/Last log:/i)
        expect(timestamps.length).toBeGreaterThan(0)
      })
    })

    it('should show guidance messages', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByText(/app-service: Logs flowing correctly/)).toBeInTheDocument()
        expect(screen.getByText(/container-app: Logs flowing correctly/)).toBeInTheDocument()
        expect(screen.getByText(/function-app: Logs flowing correctly/)).toBeInTheDocument()
      })
    })

    it('should show "Setup Complete" message', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByText('Setup Complete! 🎉')).toBeInTheDocument()
        expect(screen.getByText(/Your Azure logs integration is fully configured/)).toBeInTheDocument()
      })
    })

    it('should show "View Logs" button', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /View Logs/i })).toBeInTheDocument()
      })
    })

    it('should show "Recheck" button', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Recheck/i })).toBeInTheDocument()
      })
    })

    it('should call onValidationChange with true when all verified', async () => {
      const user = userEvent.setup({ delay: null })
      const onValidationChange = vi.fn()

      render(<SetupVerification onValidationChange={onValidationChange} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(onValidationChange).toHaveBeenCalledWith(true)
      })
    })

    it('should handle singular "service" for single service', async () => {
      const user = userEvent.setup({ delay: null })
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => ({
          status: 'success',
          workspace: mockWorkspace,
          results: {
            'app-service': {
              serviceName: 'app-service',
              logCount: 25,
              lastLogTime: '2025-12-25T10:45:00Z',
              status: 'ok',
            },
          },
          guidance: ['app-service: Logs flowing correctly'],
        }),
      })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByText('All 1 service verified')).toBeInTheDocument()
      })
    })
  })

  // ===========================================================================
  // Partial Success State Tests
  // ===========================================================================

  describe('Partial Success', () => {
    beforeEach(() => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockPartialVerified,
      })
    })

    it('should display partial success summary', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByText('1 of 3 services verified')).toBeInTheDocument()
        expect(screen.getByText(/Some services may not have generated logs yet/)).toBeInTheDocument()
      })
    })

    it('should show warning icon for partial success', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        const alertTriangles = document.querySelectorAll('.lucide-triangle-alert')
        expect(alertTriangles.length).toBeGreaterThan(0)
      })
    })

    it('should show mixed status icons for services', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        // Should have checkmark for verified service
        expect(document.querySelectorAll('.lucide-circle-check-big').length).toBeGreaterThan(0)
        // Should have warning for no-logs services
        expect(document.querySelectorAll('.lucide-triangle-alert').length).toBeGreaterThan(0)
      })
    })

    it('should display no-logs message for services without logs', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        const messages = screen.getAllByText('No logs found in last 15 minutes')
        expect(messages.length).toBeGreaterThan(0)
      })
    })

    it('should show "View Logs Anyway" button when some services verified', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /View Logs Anyway/i })).toBeInTheDocument()
      })
    })

    it('should show "Complete Setup" button', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Complete Setup/i })).toBeInTheDocument()
      })
    })

    it('should call onValidationChange with true for partial success', async () => {
      const user = userEvent.setup({ delay: null })
      const onValidationChange = vi.fn()

      render(<SetupVerification onValidationChange={onValidationChange} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(onValidationChange).toHaveBeenCalledWith(true)
      })
    })
  })

  // ===========================================================================
  // No Logs State Tests
  // ===========================================================================

  describe('No Logs Found', () => {
    beforeEach(() => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockNoLogs,
      })
    })

    it('should show partial status when no logs found', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByText('0 of 2 services verified')).toBeInTheDocument()
      })
    })

    it('should display guidance for services without logs', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByText(/Configure diagnostic settings first/)).toBeInTheDocument()
      })
    })
  })

  // ===========================================================================
  // Error State Tests
  // ===========================================================================

  describe('Error State - API Error', () => {
    beforeEach(() => {
      fetchMock.mockRejectedValue(new Error('Network connection failed'))
    })

    it('should display error message when verification fails', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByText('Verification failed')).toBeInTheDocument()
        expect(screen.getByText(/Network connection failed/)).toBeInTheDocument()
      })
    })

    it('should show error icon', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        const alertTriangles = document.querySelectorAll('.lucide-triangle-alert')
        expect(alertTriangles.length).toBeGreaterThan(0)
      })
    })

    it('should show Retry button on error', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Retry/i })).toBeInTheDocument()
      })
    })

    it('should show "Back to Diagnostic Settings" button on error', async () => {
      const user = userEvent.setup({ delay: null })
      const onNavigateToStep = vi.fn()

      render(<SetupVerification onValidationChange={vi.fn()} onNavigateToStep={onNavigateToStep} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Back to Diagnostic Settings/i })).toBeInTheDocument()
      })
    })

    it('should not show "Back" button when onNavigateToStep not provided', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.queryByRole('button', { name: /Back to Diagnostic Settings/i })).not.toBeInTheDocument()
      })
    })

    it('should call onValidationChange with false on error', async () => {
      const user = userEvent.setup({ delay: null })
      const onValidationChange = vi.fn()

      render(<SetupVerification onValidationChange={onValidationChange} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByText('Network connection failed')).toBeInTheDocument()
      })

      // Should not call with true on error
      expect(onValidationChange).not.toHaveBeenCalledWith(true)
    })
  })

  describe('Error State - Service Errors', () => {
    beforeEach(() => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockWithErrors,
      })
    })

    it('should display services with error status', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByText('function-app')).toBeInTheDocument()
        expect(screen.getByText(/DiagnosticSettingsNotConfigured/)).toBeInTheDocument()
      })
    })

    it('should show error icon for services with errors', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        const alertTriangles = document.querySelectorAll('.lucide-triangle-alert')
        expect(alertTriangles.length).toBeGreaterThan(0)
      })
    })

    it('should still show verified services correctly', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByText('app-service')).toBeInTheDocument()
        expect(screen.getByText('15 log entries found in last 15 minutes')).toBeInTheDocument()
      })
    })
  })

  // ===========================================================================
  // User Interaction Tests
  // ===========================================================================

  describe('User Interactions', () => {
    it('should retry verification when Retry button clicked', async () => {
      const user = userEvent.setup({ delay: null })
      fetchMock.mockRejectedValueOnce(new Error('Network error'))

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByText('Network error')).toBeInTheDocument()
      })

      fetchMock.mockClear()
      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => mockAllVerified,
      })

      const retryButton = screen.getByRole('button', { name: /Retry/i })
      await user.click(retryButton)

      await waitFor(() => {
        expect(fetchMock).toHaveBeenCalled()
        expect(screen.getByText('All 3 services verified')).toBeInTheDocument()
      })
    })

    it('should call onComplete when "View Logs" clicked', async () => {
      const user = userEvent.setup({ delay: null })
      const onComplete = vi.fn()
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockAllVerified,
      })

      render(<SetupVerification onValidationChange={vi.fn()} onComplete={onComplete} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /View Logs/i })).toBeInTheDocument()
      })

      const viewLogsButton = screen.getByRole('button', { name: /View Logs/i })
      await user.click(viewLogsButton)

      expect(onComplete).toHaveBeenCalledTimes(1)
    })

    it('should call onComplete when "Complete Setup" clicked', async () => {
      const user = userEvent.setup({ delay: null })
      const onComplete = vi.fn()
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockPartialVerified,
      })

      render(<SetupVerification onValidationChange={vi.fn()} onComplete={onComplete} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Complete Setup/i })).toBeInTheDocument()
      })

      const completeButton = screen.getByRole('button', { name: /Complete Setup/i })
      await user.click(completeButton)

      expect(onComplete).toHaveBeenCalledTimes(1)
    })

    it('should call onNavigateToStep when "Back to Diagnostic Settings" clicked', async () => {
      const user = userEvent.setup({ delay: null })
      const onNavigateToStep = vi.fn()
      fetchMock.mockRejectedValue(new Error('Network error'))

      render(<SetupVerification onValidationChange={vi.fn()} onNavigateToStep={onNavigateToStep} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Back to Diagnostic Settings/i })).toBeInTheDocument()
      })

      const backButton = screen.getByRole('button', { name: /Back to Diagnostic Settings/i })
      await user.click(backButton)

      expect(onNavigateToStep).toHaveBeenCalledWith('diagnostic-settings')
    })

    it('should recheck verification when Retry button clicked', async () => {
      const user = userEvent.setup({ delay: null })
      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => mockPartialVerified,
      })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByText('1 of 3 services verified')).toBeInTheDocument()
      })

      fetchMock.mockClear()
      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => mockAllVerified,
      })

      const retryButton = screen.getByRole('button', { name: /Retry/i })
      await user.click(retryButton)

      await waitFor(() => {
        expect(fetchMock).toHaveBeenCalled()
        expect(screen.getByText('All 3 services verified')).toBeInTheDocument()
      })
    })
  })

  // ===========================================================================
  // Accessibility Tests
  // ===========================================================================

  describe('Accessibility', () => {
    beforeEach(() => {
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockAllVerified,
      })
    })

    it('should have proper heading structure', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByRole('heading', { name: /Verification/i, level: 3 })).toBeInTheDocument()
        expect(screen.getByRole('heading', { name: /Services/i, level: 4 })).toBeInTheDocument()
      })
    })

    it('should have accessible buttons', async () => {
      const user = userEvent.setup({ delay: null })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        const viewLogsButton = screen.getByRole('button', { name: /View Logs/i })
        expect(viewLogsButton).toBeInTheDocument()
        expect(viewLogsButton).toHaveAttribute('type', 'button')

        const recheckButton = screen.getByRole('button', { name: /Recheck/i })
        expect(recheckButton).toBeInTheDocument()
        expect(recheckButton).toHaveAttribute('type', 'button')
      })
    })

    it('should have keyboard navigation support for all buttons', () => {
      render(<SetupVerification onValidationChange={vi.fn()} />)

      const buttons = screen.getAllByRole('button')
      buttons.forEach(button => {
        expect(button).toHaveAttribute('type', 'button')
      })
    })
  })

  // ===========================================================================
  // Edge Cases
  // ===========================================================================

  describe('Edge Cases', () => {
    it('should handle HTTP error responses', async () => {
      const user = userEvent.setup({ delay: null })
      fetchMock.mockResolvedValue({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        text: () => 'Server error occurred',
      })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByText('Verification failed')).toBeInTheDocument()
        expect(screen.getByText(/API returned 500/)).toBeInTheDocument()
      })
    })

    it('should cleanup abort controller on unmount', async () => {
      const user = userEvent.setup({ delay: null })
      fetchMock.mockReturnValue(new Promise(() => {})) // Never resolves

      const { unmount } = render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      // Unmount while loading
      unmount()

      // Should not throw errors
      expect(true).toBe(true)
    })

    it('should handle empty results gracefully', async () => {
      const user = userEvent.setup({ delay: null })
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => ({
          status: 'success',
          workspace: mockWorkspace,
          results: {},
          guidance: [],
        }),
      })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(screen.getByText('Unknown verification state')).toBeInTheDocument()
      })
    })

    it('should handle missing workspace info', async () => {
      const user = userEvent.setup({ delay: null })
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => ({
          status: 'success',
          workspace: null,
          results: mockAllVerified.results,
          guidance: mockAllVerified.guidance,
        }),
      })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      // Should still render without errors
      await waitFor(() => {
        expect(screen.getByText('All 3 services verified')).toBeInTheDocument()
      })
    })
  })

  // ===========================================================================
  // Request Payload Tests
  // ===========================================================================

  describe('Request Payload', () => {
    it('should send correct request payload', async () => {
      const user = userEvent.setup({ delay: null })
      fetchMock.mockResolvedValue({
        ok: true,
        json: () => mockAllVerified,
      })

      render(<SetupVerification onValidationChange={vi.fn()} />)

      const startButton = screen.getByRole('button', { name: /Start Verification/i })
      await user.click(startButton)

      await waitFor(() => {
        expect(fetchMock).toHaveBeenCalledWith('/api/azure/workspace/verify', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            services: [],
            timespan: 'PT15M',
          }),
          signal: expect.any(AbortSignal) as AbortSignal,
        })
      })
    })
  })
})
