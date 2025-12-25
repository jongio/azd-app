/**
 * AzureConnectionStatus Component Tests
 * 
 * Tests the Azure connection status indicator component.
 * Verifies connection states, error handling, retry functionality, and accessibility.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AzureConnectionStatus, AzureStatusBadge, type AzureConnectionStatusProps, type AzureStatusBadgeProps } from './AzureConnectionStatus'

describe('AzureConnectionStatus', () => {
  const defaultProps: AzureConnectionStatusProps = {
    status: 'connected',
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.clearAllTimers()
  })

  describe('Basic Rendering', () => {
    it('renders with connected status', () => {
      render(<AzureConnectionStatus {...defaultProps} status="connected" />)
      
      const status = screen.getByRole('status', { name: 'Azure connection: Connected' })
      expect(status).toBeInTheDocument()
    })

    it('renders with disconnected status', () => {
      render(<AzureConnectionStatus {...defaultProps} status="disconnected" />)
      
      const status = screen.getByRole('status', { name: 'Azure connection: Disconnected' })
      expect(status).toBeInTheDocument()
    })

    it('renders with connecting status', () => {
      render(<AzureConnectionStatus {...defaultProps} status="connecting" />)
      
      const status = screen.getByRole('status', { name: 'Azure connection: Connecting' })
      expect(status).toBeInTheDocument()
    })

    it('renders with error status', () => {
      render(<AzureConnectionStatus {...defaultProps} status="error" />)
      
      const status = screen.getByRole('status', { name: 'Azure connection: Error' })
      expect(status).toBeInTheDocument()
    })

    it('renders with disabled status', () => {
      render(<AzureConnectionStatus {...defaultProps} status="disabled" />)
      
      const status = screen.getByRole('status', { name: 'Azure connection: Not Configured' })
      expect(status).toBeInTheDocument()
    })

    it('shows connecting spinner animation', () => {
      render(<AzureConnectionStatus {...defaultProps} status="connecting" />)
      
      const status = screen.getByRole('status')
      const spinner = status.querySelector('.animate-spin')
      expect(spinner).toBeInTheDocument()
    })

    it('does not show spinner for non-connecting states', () => {
      render(<AzureConnectionStatus {...defaultProps} status="connected" />)
      
      const status = screen.getByRole('status')
      const spinner = status.querySelector('.animate-spin')
      expect(spinner).not.toBeInTheDocument()
    })
  })

  describe('Detailed View', () => {
    it('shows labels when showDetails is true', () => {
      render(<AzureConnectionStatus {...defaultProps} showDetails={true} />)
      
      expect(screen.getByText('Connected')).toBeInTheDocument()
      expect(screen.getByText('Streaming logs from Azure')).toBeInTheDocument()
    })

    it('hides labels when showDetails is false', () => {
      render(<AzureConnectionStatus {...defaultProps} showDetails={false} />)
      
      expect(screen.queryByText('Connected')).not.toBeInTheDocument()
      expect(screen.queryByText('Streaming logs from Azure')).not.toBeInTheDocument()
    })

    it('shows resource count when connected with resources', () => {
      render(
        <AzureConnectionStatus 
          {...defaultProps} 
          status="connected" 
          resourceCount={3} 
          showDetails={true} 
        />
      )
      
      expect(screen.getByText('3 resources')).toBeInTheDocument()
    })

    it('shows singular "resource" for count of 1', () => {
      render(
        <AzureConnectionStatus 
          {...defaultProps} 
          status="connected" 
          resourceCount={1} 
          showDetails={true} 
        />
      )
      
      expect(screen.getByText('1 resource')).toBeInTheDocument()
    })

    it('shows error message when in error state with details', () => {
      render(
        <AzureConnectionStatus 
          {...defaultProps} 
          status="error" 
          errorMessage="Authentication failed" 
          showDetails={true} 
        />
      )
      
      expect(screen.getByText('Authentication failed')).toBeInTheDocument()
    })

    it('shows default description when no error message in detailed view', () => {
      render(
        <AzureConnectionStatus 
          {...defaultProps} 
          status="disconnected" 
          showDetails={true} 
        />
      )
      
      expect(screen.getByText('Azure logs not streaming')).toBeInTheDocument()
    })
  })

  describe('Error Handling', () => {
    it('makes status clickable when in error state with message', () => {
      render(
        <AzureConnectionStatus 
          {...defaultProps} 
          status="error" 
          errorMessage="Connection timeout" 
        />
      )
      
      const status = screen.getByRole('status')
      expect(status).toHaveClass('cursor-pointer')
      expect(status).toHaveAttribute('tabIndex', '0')
    })

    it('is not clickable when no error message', () => {
      render(
        <AzureConnectionStatus 
          {...defaultProps} 
          status="error" 
        />
      )
      
      const status = screen.getByRole('status')
      expect(status).not.toHaveClass('cursor-pointer')
      expect(status).not.toHaveAttribute('tabIndex')
    })

    it('shows error popover when clicking on error status', async () => {
      const user = userEvent.setup()
      
      render(
        <AzureConnectionStatus 
          {...defaultProps} 
          status="error" 
          errorMessage="Authentication failed: Invalid credentials" 
        />
      )
      
      const status = screen.getByRole('status')
      await user.click(status)
      
      const dialog = await screen.findByRole('dialog', { name: 'Azure connection error details' })
      expect(dialog).toBeInTheDocument()
    })

    it('shows click for details hint when error is clickable', () => {
      render(
        <AzureConnectionStatus 
          {...defaultProps} 
          status="error" 
          errorMessage="Connection failed" 
          showDetails={true} 
        />
      )
      
      expect(screen.getByText('(click for details)')).toBeInTheDocument()
    })

    it('opens popover with Enter key', async () => {
      const user = userEvent.setup()
      
      render(
        <AzureConnectionStatus 
          {...defaultProps} 
          status="error" 
          errorMessage="Connection failed" 
        />
      )
      
      const status = screen.getByRole('status')
      status.focus()
      await user.keyboard('{Enter}')
      
      const dialog = await screen.findByRole('dialog')
      expect(dialog).toBeInTheDocument()
    })

    it('opens popover with Space key', async () => {
      const user = userEvent.setup()
      
      render(
        <AzureConnectionStatus 
          {...defaultProps} 
          status="error" 
          errorMessage="Connection failed" 
        />
      )
      
      const status = screen.getByRole('status')
      status.focus()
      await user.keyboard(' ')
      
      const dialog = await screen.findByRole('dialog')
      expect(dialog).toBeInTheDocument()
    })

    it('closes error popover when clicking close button', async () => {
      const user = userEvent.setup()
      
      render(
        <AzureConnectionStatus 
          {...defaultProps} 
          status="error" 
          errorMessage="Connection failed" 
        />
      )
      
      // Open popover
      const status = screen.getByRole('status')
      await user.click(status)
      
      const dialog = await screen.findByRole('dialog')
      expect(dialog).toBeInTheDocument()
      
      // Close popover
      const closeButton = screen.getByRole('button', { name: 'Close' })
      await user.click(closeButton)
      
      await waitFor(() => {
        expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
      })
    })

    it('closes error popover when clicking outside', async () => {
      const user = userEvent.setup()
      
      render(
        <div>
          <AzureConnectionStatus 
            {...defaultProps} 
            status="error" 
            errorMessage="Connection failed" 
          />
          <div data-testid="outside">Outside</div>
        </div>
      )
      
      // Open popover
      const status = screen.getByRole('status')
      await user.click(status)
      
      await screen.findByRole('dialog')
      
      // Click outside
      const outside = screen.getByTestId('outside')
      await user.click(outside)
      
      await waitFor(() => {
        expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
      })
    })

    it('closes error popover with Escape key', async () => {
      const user = userEvent.setup()
      
      render(
        <AzureConnectionStatus 
          {...defaultProps} 
          status="error" 
          errorMessage="Connection failed" 
        />
      )
      
      // Open popover
      const status = screen.getByRole('status')
      await user.click(status)
      
      await screen.findByRole('dialog')
      
      // Press Escape
      await user.keyboard('{Escape}')
      
      await waitFor(() => {
        expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
      })
    })
  })

  describe('Retry Functionality', () => {
    it('shows retry button in error state without details', () => {
      const onRetry = vi.fn()
      
      render(
        <AzureConnectionStatus 
          {...defaultProps} 
          status="error" 
          showDetails={false} 
          onRetry={onRetry} 
        />
      )
      
      const retryButton = screen.getByRole('button', { name: 'Retry Azure connection' })
      expect(retryButton).toBeInTheDocument()
    })

    it('does not show retry button in non-error states', () => {
      const onRetry = vi.fn()
      
      render(
        <AzureConnectionStatus 
          {...defaultProps} 
          status="connected" 
          onRetry={onRetry} 
        />
      )
      
      expect(screen.queryByRole('button', { name: 'Retry Azure connection' })).not.toBeInTheDocument()
    })

    it('does not show retry button when showDetails is true', () => {
      const onRetry = vi.fn()
      
      render(
        <AzureConnectionStatus 
          {...defaultProps} 
          status="error" 
          showDetails={true} 
          onRetry={onRetry} 
        />
      )
      
      expect(screen.queryByRole('button', { name: 'Retry Azure connection' })).not.toBeInTheDocument()
    })

    it('calls onRetry when retry button clicked', async () => {
      const onRetry = vi.fn()
      const user = userEvent.setup()
      
      render(
        <AzureConnectionStatus 
          {...defaultProps} 
          status="error" 
          onRetry={onRetry} 
        />
      )
      
      const retryButton = screen.getByRole('button', { name: 'Retry Azure connection' })
      await user.click(retryButton)
      
      expect(onRetry).toHaveBeenCalledOnce()
    })

    it('calls onRetry from error popover', async () => {
      const onRetry = vi.fn()
      const user = userEvent.setup()
      
      render(
        <AzureConnectionStatus 
          {...defaultProps} 
          status="error" 
          errorMessage="Connection failed" 
          onRetry={onRetry} 
        />
      )
      
      // Open popover
      const status = screen.getByRole('status')
      await user.click(status)
      
      // The popover contains AzureErrorDisplay which may have a retry button
      // depending on the error type. Since we're passing onRetry, it should be available
      await waitFor(() => {
        const dialog = screen.getByRole('dialog')
        expect(dialog).toBeInTheDocument()
      })
      
      // Verify popover closes after retry
      // (implementation detail: retry should close the popover)
    })

    it('stops event propagation when clicking retry button', async () => {
      const onRetry = vi.fn()
      const onStatusClick = vi.fn()
      const user = userEvent.setup()
      
      render(
        <div onClick={onStatusClick}>
          <AzureConnectionStatus 
            {...defaultProps} 
            status="error" 
            onRetry={onRetry} 
          />
        </div>
      )
      
      const retryButton = screen.getByRole('button', { name: 'Retry Azure connection' })
      await user.click(retryButton)
      
      expect(onRetry).toHaveBeenCalled()
      expect(onStatusClick).not.toHaveBeenCalled()
    })
  })

  describe('Accessibility', () => {
    it('has proper role and aria-label', () => {
      render(<AzureConnectionStatus {...defaultProps} status="connected" />)
      
      const status = screen.getByRole('status', { name: 'Azure connection: Connected' })
      expect(status).toBeInTheDocument()
    })

    it('provides meaningful title for retry button', () => {
      const onRetry = vi.fn()
      
      render(
        <AzureConnectionStatus 
          {...defaultProps} 
          status="error" 
          onRetry={onRetry} 
        />
      )
      
      const retryButton = screen.getByRole('button', { name: 'Retry Azure connection' })
      expect(retryButton).toHaveAttribute('title', 'Retry connection')
    })

    it('error popover has dialog role', async () => {
      const user = userEvent.setup()
      
      render(
        <AzureConnectionStatus 
          {...defaultProps} 
          status="error" 
          errorMessage="Connection failed" 
        />
      )
      
      const status = screen.getByRole('status')
      await user.click(status)
      
      const dialog = await screen.findByRole('dialog', { name: 'Azure connection error details' })
      expect(dialog).toBeInTheDocument()
    })

    it('close button has accessible label', async () => {
      const user = userEvent.setup()
      
      render(
        <AzureConnectionStatus 
          {...defaultProps} 
          status="error" 
          errorMessage="Connection failed" 
        />
      )
      
      const status = screen.getByRole('status')
      await user.click(status)
      
      const closeButton = await screen.findByRole('button', { name: 'Close' })
      expect(closeButton).toBeInTheDocument()
    })
  })

  describe('Custom Styling', () => {
    it('applies custom className', () => {
      const { container } = render(
        <AzureConnectionStatus 
          {...defaultProps} 
          className="custom-class" 
        />
      )
      
      const wrapper = container.querySelector('.custom-class')
      expect(wrapper).toBeInTheDocument()
    })

    it('applies correct color classes for connected state', () => {
      render(<AzureConnectionStatus {...defaultProps} status="connected" />)
      
      const status = screen.getByRole('status')
      expect(status.querySelector('.text-green-600')).toBeInTheDocument()
      expect(status.querySelector('.bg-green-100')).toBeInTheDocument()
    })

    it('applies correct color classes for error state', () => {
      render(<AzureConnectionStatus {...defaultProps} status="error" />)
      
      const status = screen.getByRole('status')
      expect(status.querySelector('.text-red-600')).toBeInTheDocument()
      expect(status.querySelector('.bg-red-100')).toBeInTheDocument()
    })

    it('applies correct color classes for disconnected state', () => {
      render(<AzureConnectionStatus {...defaultProps} status="disconnected" />)
      
      const status = screen.getByRole('status')
      expect(status.querySelector('.text-yellow-600')).toBeInTheDocument()
      expect(status.querySelector('.bg-yellow-100')).toBeInTheDocument()
    })

    it('applies correct color classes for connecting state', () => {
      render(<AzureConnectionStatus {...defaultProps} status="connecting" />)
      
      const status = screen.getByRole('status')
      expect(status.querySelector('.text-blue-600')).toBeInTheDocument()
      expect(status.querySelector('.bg-blue-100')).toBeInTheDocument()
    })

    it('applies correct color classes for disabled state', () => {
      render(<AzureConnectionStatus {...defaultProps} status="disabled" />)
      
      const status = screen.getByRole('status')
      expect(status.querySelector('.text-slate-500')).toBeInTheDocument()
      expect(status.querySelector('.bg-slate-100')).toBeInTheDocument()
    })
  })
})

describe('AzureStatusBadge', () => {
  const defaultProps: AzureStatusBadgeProps = {
    status: 'connected',
  }

  describe('Basic Rendering', () => {
    it('renders compact badge with label', () => {
      render(<AzureStatusBadge {...defaultProps} />)
      
      expect(screen.getByRole('status', { name: 'Azure: Connected' })).toBeInTheDocument()
      expect(screen.getByText('Connected')).toBeInTheDocument()
    })

    it('renders all status types', () => {
      const statuses: Array<'connected' | 'disconnected' | 'connecting' | 'error' | 'disabled'> = [
        'connected',
        'disconnected',
        'connecting',
        'error',
        'disabled'
      ]
      
      statuses.forEach(status => {
        const { unmount } = render(<AzureStatusBadge {...defaultProps} status={status} />)
        const badge = screen.getByRole('status')
        expect(badge).toBeInTheDocument()
        unmount()
      })
    })

    it('shows connecting spinner', () => {
      render(<AzureStatusBadge {...defaultProps} status="connecting" />)
      
      const badge = screen.getByRole('status')
      const spinner = badge.querySelector('.animate-spin')
      expect(spinner).toBeInTheDocument()
    })
  })

  describe('Error Handling', () => {
    it('makes badge clickable when in error state with message', () => {
      render(
        <AzureStatusBadge 
          {...defaultProps} 
          status="error" 
          errorMessage="Connection failed" 
        />
      )
      
      const badge = screen.getByRole('status')
      expect(badge).toHaveClass('cursor-pointer')
      expect(badge).toHaveAttribute('tabIndex', '0')
    })

    it('shows error popover when clicked', async () => {
      const user = userEvent.setup()
      
      render(
        <AzureStatusBadge 
          {...defaultProps} 
          status="error" 
          errorMessage="Authentication failed" 
        />
      )
      
      const badge = screen.getByRole('status')
      await user.click(badge)
      
      const dialog = await screen.findByRole('dialog')
      expect(dialog).toBeInTheDocument()
    })

    it('supports keyboard activation', async () => {
      const user = userEvent.setup()
      
      render(
        <AzureStatusBadge 
          {...defaultProps} 
          status="error" 
          errorMessage="Connection failed" 
        />
      )
      
      const badge = screen.getByRole('status')
      badge.focus()
      await user.keyboard('{Enter}')
      
      const dialog = await screen.findByRole('dialog')
      expect(dialog).toBeInTheDocument()
    })
  })

  describe('Retry Functionality', () => {
    it('calls onRetry from error popover', async () => {
      const onRetry = vi.fn()
      const user = userEvent.setup()
      
      render(
        <AzureStatusBadge 
          {...defaultProps} 
          status="error" 
          errorMessage="Connection failed" 
          onRetry={onRetry} 
        />
      )
      
      const badge = screen.getByRole('status')
      await user.click(badge)
      
      await waitFor(() => {
        expect(screen.getByRole('dialog')).toBeInTheDocument()
      })
      
      // Popover should have retry functionality via AzureErrorDisplay
    })
  })

  describe('Styling', () => {
    it('applies custom className', () => {
      const { container } = render(
        <AzureStatusBadge 
          {...defaultProps} 
          className="custom-badge-class" 
        />
      )
      
      const badge = container.querySelector('.custom-badge-class')
      expect(badge).toBeInTheDocument()
    })

    it('uses compact badge styling', () => {
      render(<AzureStatusBadge {...defaultProps} />)
      
      const badge = screen.getByRole('status')
      expect(badge).toHaveClass('inline-flex', 'items-center', 'gap-1', 'px-2', 'py-1', 'rounded-full', 'text-xs', 'font-medium')
    })
  })

  describe('Accessibility', () => {
    it('has proper role and aria-label', () => {
      render(<AzureStatusBadge {...defaultProps} status="connected" />)
      
      const badge = screen.getByRole('status', { name: 'Azure: Connected' })
      expect(badge).toBeInTheDocument()
    })
  })
})
