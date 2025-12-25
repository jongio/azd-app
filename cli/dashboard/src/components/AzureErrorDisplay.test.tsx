/**
 * AzureErrorDisplay Component Tests
 * 
 * Tests the Azure error display component with various error types,
 * retry functionality, and accessibility features.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AzureErrorDisplay, type AzureErrorDisplayProps } from './AzureErrorDisplay'
import type { ErrorInfo } from '@/types'

describe('AzureErrorDisplay', () => {
  const defaultProps: AzureErrorDisplayProps = {
    errorType: 'generic',
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('Error Type Rendering', () => {
    it('renders authentication error', () => {
      render(<AzureErrorDisplay {...defaultProps} errorType="auth" />)
      
      expect(screen.getByRole('alert')).toBeInTheDocument()
      expect(screen.getByText('Authentication Required')).toBeInTheDocument()
      expect(screen.getByText(/Sign in to Azure to view cloud logs/i)).toBeInTheDocument()
    })

    it('renders permission denied error', () => {
      render(<AzureErrorDisplay {...defaultProps} errorType="permission" />)
      
      expect(screen.getByText('Permission Denied')).toBeInTheDocument()
      expect(screen.getByText(/doesn't have access to query logs/i)).toBeInTheDocument()
    })

    it('renders not-found error', () => {
      render(<AzureErrorDisplay {...defaultProps} errorType="not-found" />)
      
      expect(screen.getByText('Resource Not Found')).toBeInTheDocument()
      expect(screen.getByText(/requested resource was not found/i)).toBeInTheDocument()
    })

    it('renders not-found error with service name', () => {
      render(<AzureErrorDisplay {...defaultProps} errorType="not-found" serviceName="my-api" />)
      
      expect(screen.getByText(/Service "my-api" not found in Azure/i)).toBeInTheDocument()
    })

    it('renders rate-limit error', () => {
      render(<AzureErrorDisplay {...defaultProps} errorType="rate-limit" />)
      
      expect(screen.getByText('Rate Limited')).toBeInTheDocument()
      expect(screen.getByText(/Too many requests to Azure/i)).toBeInTheDocument()
    })

    it('renders network error', () => {
      render(<AzureErrorDisplay {...defaultProps} errorType="network" />)
      
      expect(screen.getByText('Connection Failed')).toBeInTheDocument()
      expect(screen.getByText(/Unable to reach Azure services/i)).toBeInTheDocument()
    })

    it('renders workspace error', () => {
      render(<AzureErrorDisplay {...defaultProps} errorType="workspace" />)
      
      expect(screen.getByText('Log Analytics Not Configured')).toBeInTheDocument()
      expect(screen.getByText(/No Log Analytics workspace found/i)).toBeInTheDocument()
    })

    it('renders query error', () => {
      render(<AzureErrorDisplay {...defaultProps} errorType="query" />)
      
      expect(screen.getByText('Query Error')).toBeInTheDocument()
      expect(screen.getByText(/Invalid query syntax/i)).toBeInTheDocument()
    })

    it('renders generic error as fallback', () => {
      render(<AzureErrorDisplay {...defaultProps} errorType="generic" />)
      
      expect(screen.getByText('Something went wrong')).toBeInTheDocument()
      expect(screen.getByText(/An error occurred/i)).toBeInTheDocument()
    })
  })

  describe('Compact Mode', () => {
    it('renders compact variant with minimal UI', () => {
      render(<AzureErrorDisplay {...defaultProps} errorType="auth" compact={true} />)
      
      // Should show title and message in compact form
      expect(screen.getByText('Authentication Required')).toBeInTheDocument()
    })

    it('shows retry button in compact mode', () => {
      const onRetry = vi.fn()
      render(<AzureErrorDisplay {...defaultProps} compact={true} onRetry={onRetry} />)
      
      const retryButton = screen.getByRole('button', { name: 'Retry' })
      expect(retryButton).toBeInTheDocument()
    })

    it('calls onRetry when compact retry button clicked', async () => {
      const onRetry = vi.fn()
      const user = userEvent.setup()
      
      render(<AzureErrorDisplay {...defaultProps} compact={true} onRetry={onRetry} />)
      
      const retryButton = screen.getByRole('button', { name: 'Retry' })
      await user.click(retryButton)
      
      expect(onRetry).toHaveBeenCalledOnce()
    })
  })

  describe('Error Messages', () => {
    it('displays custom error message', () => {
      render(
        <AzureErrorDisplay 
          {...defaultProps} 
          errorType="auth" 
          message="Custom authentication error message" 
        />
      )
      
      expect(screen.getByText('Custom authentication error message')).toBeInTheDocument()
    })

    it('shows query error details', () => {
      const errorMessage = 'Syntax error at line 5: unexpected token'
      render(
        <AzureErrorDisplay 
          {...defaultProps} 
          errorType="query" 
          message={errorMessage} 
        />
      )
      
      // The error message appears in both the description and the details pre block
      const allMatches = screen.getAllByText(errorMessage)
      expect(allMatches.length).toBeGreaterThan(0)
    })
  })

  describe('Command Display', () => {
    it('shows command copy button for auth error', () => {
      render(<AzureErrorDisplay {...defaultProps} errorType="auth" />)
      
      expect(screen.getByText('azd auth login')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'Copy command' })).toBeInTheDocument()
    })

    it('copies command to clipboard', async () => {
      const user = userEvent.setup()
      
      // Mock clipboard API
      const writeText = vi.fn().mockResolvedValue(undefined)
      Object.defineProperty(navigator, 'clipboard', {
        value: { writeText },
        configurable: true,
      })
      
      render(<AzureErrorDisplay {...defaultProps} errorType="auth" />)
      
      const copyButton = screen.getByRole('button', { name: 'Copy command' })
      await user.click(copyButton)
      
      await waitFor(() => {
        expect(writeText).toHaveBeenCalledWith('azd auth login')
      })
      
      // Should show copied indicator
      expect(screen.getByRole('button', { name: 'Copied' })).toBeInTheDocument()
    })

    it('resets copied state after timeout', async () => {
      vi.useFakeTimers()
      const user = userEvent.setup({ delay: null })
      
      const writeText = vi.fn().mockResolvedValue(undefined)
      Object.defineProperty(navigator, 'clipboard', {
        value: { writeText },
        configurable: true,
      })
      
      render(<AzureErrorDisplay {...defaultProps} errorType="auth" />)
      
      const copyButton = screen.getByRole('button', { name: 'Copy command' })
      await user.click(copyButton)
      
      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Copied' })).toBeInTheDocument()
      })
      
      // Fast-forward time
      vi.advanceTimersByTime(2000)
      
      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Copy command' })).toBeInTheDocument()
      })
      
      vi.useRealTimers()
    })
  })

  describe('Code Snippets', () => {
    it('shows code snippet for workspace error', () => {
      render(<AzureErrorDisplay {...defaultProps} errorType="workspace" />)
      
      expect(screen.getByText('logs:')).toBeInTheDocument()
      expect(screen.getByText('azure:')).toBeInTheDocument()
      expect(screen.getByText(/workspace: "your-workspace-id"/)).toBeInTheDocument()
    })
  })

  describe('External Links', () => {
    it('shows external documentation link', () => {
      render(<AzureErrorDisplay {...defaultProps} errorType="permission" />)
      
      const link = screen.getByRole('link', { name: /View Azure RBAC Docs/i })
      expect(link).toBeInTheDocument()
      expect(link).toHaveAttribute('href', expect.stringContaining('learn.microsoft.com'))
      expect(link).toHaveAttribute('target', '_blank')
      expect(link).toHaveAttribute('rel', 'noopener noreferrer')
    })
  })

  describe('Rate Limit with Countdown', () => {
    it('shows countdown timer for rate limit', () => {
      vi.useFakeTimers()
      
      render(<AzureErrorDisplay {...defaultProps} errorType="rate-limit" retryAfter={30} />)
      
      expect(screen.getByText('Retry in:')).toBeInTheDocument()
      expect(screen.getByText('30s')).toBeInTheDocument()
      
      vi.useRealTimers()
    })

    it('decrements countdown timer', async () => {
      vi.useFakeTimers()
      
      render(<AzureErrorDisplay {...defaultProps} errorType="rate-limit" retryAfter={5} />)
      
      expect(screen.getByText(/5/)).toBeInTheDocument()
      
      await vi.advanceTimersByTimeAsync(1000)
      expect(screen.getByText(/4/)).toBeInTheDocument()
      
      await vi.advanceTimersByTimeAsync(1000)
      expect(screen.getByText(/3/)).toBeInTheDocument()
      
      vi.useRealTimers()
    })

    it('calls onRetry when countdown completes', async () => {
      vi.useFakeTimers()
      const onRetry = vi.fn()
      
      render(
        <AzureErrorDisplay 
          {...defaultProps} 
          errorType="rate-limit" 
          retryAfter={2} 
          onRetry={onRetry} 
        />
      )
      
      expect(onRetry).not.toHaveBeenCalled()
      
      await vi.advanceTimersByTimeAsync(2000)
      
      expect(onRetry).toHaveBeenCalledOnce()
      
      vi.useRealTimers()
    })
  })

  describe('Retry Functionality', () => {
    it('shows retry button when onRetry provided', () => {
      const onRetry = vi.fn()
      render(<AzureErrorDisplay {...defaultProps} onRetry={onRetry} />)
      
      const retryButton = screen.getByRole('button', { name: /Retry/i })
      expect(retryButton).toBeInTheDocument()
    })

    it('calls onRetry when retry button clicked', async () => {
      const onRetry = vi.fn()
      const user = userEvent.setup()
      
      render(<AzureErrorDisplay {...defaultProps} onRetry={onRetry} />)
      
      const retryButton = screen.getByRole('button', { name: /Retry/i })
      await user.click(retryButton)
      
      expect(onRetry).toHaveBeenCalledOnce()
    })

    it('does not show retry button when onRetry not provided', () => {
      render(<AzureErrorDisplay {...defaultProps} />)
      
      expect(screen.queryByRole('button', { name: /Retry/i })).not.toBeInTheDocument()
    })
  })

  describe('Secondary Actions', () => {
    it('shows View Local Logs button for network error', () => {
      const onViewLocal = vi.fn()
      
      render(
        <AzureErrorDisplay 
          {...defaultProps} 
          errorType="network" 
          onViewLocal={onViewLocal} 
        />
      )
      
      expect(screen.getByRole('button', { name: /View Local Logs/i })).toBeInTheDocument()
    })

    it('calls onViewLocal when View Local Logs is clicked', async () => {
      const onViewLocal = vi.fn()
      const user = userEvent.setup()
      
      render(
        <AzureErrorDisplay 
          {...defaultProps} 
          errorType="network" 
          onViewLocal={onViewLocal} 
        />
      )
      
      const localButton = screen.getByRole('button', { name: /View Local Logs/i })
      await user.click(localButton)
      expect(onViewLocal).toHaveBeenCalledOnce()
    })

    it('shows Reset to Default Query button for query error', () => {
      const onResetQuery = vi.fn()
      
      render(
        <AzureErrorDisplay 
          {...defaultProps} 
          errorType="query" 
          onResetQuery={onResetQuery} 
        />
      )
      
      expect(screen.getByRole('button', { name: /Reset to Default Query/i })).toBeInTheDocument()
    })

    it('calls onResetQuery when Reset to Default Query is clicked', async () => {
      const onResetQuery = vi.fn()
      const user = userEvent.setup()
      
      render(
        <AzureErrorDisplay 
          {...defaultProps} 
          errorType="query" 
          onResetQuery={onResetQuery} 
        />
      )
      
      const resetButton = screen.getByRole('button', { name: /Reset to Default Query/i })
      await user.click(resetButton)
      expect(onResetQuery).toHaveBeenCalledOnce()
    })

    it('shows Report Issue link for generic error', () => {
      render(<AzureErrorDisplay {...defaultProps} errorType="generic" />)
      
      const reportLink = screen.getByRole('link', { name: /Report Issue/i })
      expect(reportLink).toBeInTheDocument()
      expect(reportLink).toHaveAttribute('href', expect.stringContaining('github.com'))
      expect(reportLink).toHaveAttribute('target', '_blank')
    })
  })

  describe('ErrorInfo Integration', () => {
    it('uses ErrorInfo when provided', () => {
      const errorInfo: ErrorInfo = {
        code: 'AUTH_REQUIRED',
        message: 'Authentication is required to access Azure logs',
        action: 'Please sign in using Azure CLI',
        command: 'az login',
        docsUrl: 'https://docs.example.com/auth',
      }
      
      render(<AzureErrorDisplay {...defaultProps} errorInfo={errorInfo} />)
      
      expect(screen.getByText('Authentication is required to access Azure logs')).toBeInTheDocument()
      expect(screen.getByText('Please sign in using Azure CLI')).toBeInTheDocument()
      expect(screen.getByText('az login')).toBeInTheDocument()
    })

    it('maps error code to error type correctly', () => {
      const errorInfo: ErrorInfo = {
        code: 'PERMISSION_DENIED',
        message: 'Access denied',
        action: 'Check permissions',
      }
      
      render(<AzureErrorDisplay {...defaultProps} errorInfo={errorInfo} />)
      
      // Should map PERMISSION_DENIED to permission error type
      expect(screen.getByText('Permission Denied')).toBeInTheDocument()
    })

    it('prioritizes errorInfo command over default command', () => {
      const errorInfo: ErrorInfo = {
        code: 'AUTH_REQUIRED',
        message: 'Auth required',
        action: 'Login',
        command: 'custom auth command',
      }
      
      render(<AzureErrorDisplay {...defaultProps} errorInfo={errorInfo} />)
      
      expect(screen.getByText('custom auth command')).toBeInTheDocument()
      expect(screen.queryByText('azd auth login')).not.toBeInTheDocument()
    })

    it('prioritizes errorInfo docsUrl over default external link', () => {
      const errorInfo: ErrorInfo = {
        code: 'PERMISSION_DENIED',
        message: 'Access denied',
        action: 'Check permissions',
        docsUrl: 'https://custom-docs.example.com',
      }
      
      render(<AzureErrorDisplay {...defaultProps} errorInfo={errorInfo} />)
      
      const link = screen.getByRole('link', { name: /View Setup Guide/i })
      expect(link).toHaveAttribute('href', 'https://custom-docs.example.com')
    })
  })

  describe('Diagnostics', () => {
    it('shows Run Diagnostics button when callback provided', () => {
      const onRunDiagnostics = vi.fn()
      
      render(<AzureErrorDisplay {...defaultProps} onRunDiagnostics={onRunDiagnostics} />)
      
      expect(screen.getByRole('button', { name: /Run Diagnostics/i })).toBeInTheDocument()
    })

    it('calls onRunDiagnostics when button clicked', async () => {
      const onRunDiagnostics = vi.fn()
      const user = userEvent.setup()
      
      render(<AzureErrorDisplay {...defaultProps} onRunDiagnostics={onRunDiagnostics} />)
      
      const diagnosticsButton = screen.getByRole('button', { name: /Run Diagnostics/i })
      await user.click(diagnosticsButton)
      expect(onRunDiagnostics).toHaveBeenCalledOnce()
    })

    it('does not show diagnostics button when callback not provided', () => {
      render(<AzureErrorDisplay {...defaultProps} />)
      
      expect(screen.queryByRole('button', { name: /Run Diagnostics/i })).not.toBeInTheDocument()
    })
  })

  describe('Permission Details', () => {
    it('shows permission details for permission error', () => {
      render(<AzureErrorDisplay {...defaultProps} errorType="permission" />)
      
      expect(screen.getByText(/Required permissions:/i)).toBeInTheDocument()
      expect(screen.getByText(/Log Analytics Reader on the workspace/i)).toBeInTheDocument()
      expect(screen.getByText(/Reader on the resource group/i)).toBeInTheDocument()
    })
  })

  describe('Accessibility', () => {
    it('has alert role for screen readers', () => {
      render(<AzureErrorDisplay {...defaultProps} />)
      
      const alert = screen.getByRole('alert')
      expect(alert).toBeInTheDocument()
    })

    it('provides accessible labels for interactive elements', () => {
      const onRetry = vi.fn()
      render(<AzureErrorDisplay {...defaultProps} onRetry={onRetry} />)
      
      const retryButton = screen.getByRole('button', { name: /Retry/i })
      expect(retryButton).toBeInTheDocument()
    })

    it('includes screen reader only text for copy feedback', async () => {
      const user = userEvent.setup()
      
      const writeText = vi.fn().mockResolvedValue(undefined)
      Object.defineProperty(navigator, 'clipboard', {
        value: { writeText },
        configurable: true,
      })
      
      render(<AzureErrorDisplay {...defaultProps} errorType="auth" />)
      
      const copyButton = screen.getByRole('button', { name: 'Copy command' })
      await user.click(copyButton)
      
      await waitFor(() => {
        const srText = screen.getByText('Copied to clipboard', { selector: '.sr-only' })
        expect(srText).toBeInTheDocument()
      })
    })
  })

  describe('Custom Styling', () => {
    it('applies custom className', () => {
      const { container } = render(
        <AzureErrorDisplay {...defaultProps} className="custom-error-class" />
      )
      
      const element = container.querySelector('.custom-error-class')
      expect(element).toBeInTheDocument()
    })
  })
})
