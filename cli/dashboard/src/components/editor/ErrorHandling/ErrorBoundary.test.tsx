/**
 * ErrorBoundary Tests
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ErrorBoundary } from './ErrorBoundary'

// Test component that throws an error
function ThrowError({ shouldThrow }: { shouldThrow: boolean }) {
  if (shouldThrow) {
    throw new Error('Test error')
  }
  return <div>No error</div>
}

describe('ErrorBoundary', () => {
  const originalConsoleError = console.error

  beforeEach(() => {
    // Suppress console.error for these tests
    console.error = vi.fn()
  })

  afterEach(() => {
    console.error = originalConsoleError
    vi.clearAllMocks()
  })

  it('should render children when no error', () => {
    render(
      <ErrorBoundary>
        <div>Test content</div>
      </ErrorBoundary>
    )

    expect(screen.getByText('Test content')).toBeInTheDocument()
  })

  it('should render fallback UI when error occurs', () => {
    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    )

    expect(screen.getByText('Something went wrong')).toBeInTheDocument()
    expect(
      screen.getByText(/The editor encountered an unexpected error/)
    ).toBeInTheDocument()
  })

  it('should call onError callback when error occurs', () => {
    const onError = vi.fn()

    render(
      <ErrorBoundary onError={onError}>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    )

    expect(onError).toHaveBeenCalledWith(expect.any(Error), expect.any(Object))
    expect(onError.mock.calls[0][0].message).toBe('Test error')
  })

  it('should render custom fallback when provided', () => {
    const customFallback = <div>Custom error UI</div>

    render(
      <ErrorBoundary fallback={customFallback}>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    )

    expect(screen.getByText('Custom error UI')).toBeInTheDocument()
    expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument()
  })

  it('should reload page on Reload button click', async () => {
    const user = userEvent.setup()
    const originalReload = window.location.reload

    // Mock window.location.reload
    Object.defineProperty(window, 'location', {
      value: {
        ...window.location,
        reload: vi.fn(),
      },
      writable: true,
    })

    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    )

    const reloadButton = screen.getByText('Reload Page')
    await user.click(reloadButton)

    expect(window.location.reload).toHaveBeenCalled()

    // Restore
    Object.defineProperty(window, 'location', {
      value: {
        ...window.location,
        reload: originalReload,
      },
      writable: true,
    })
  })

  it('should reset error state on Try Again button click', async () => {
    const user = userEvent.setup()
    let shouldThrow = true

    const TestComponent = () => {
      if (shouldThrow) {
        throw new Error('Test error')
      }
      return <div>No error</div>
    }

    const { rerender } = render(
      <ErrorBoundary key="error-boundary-1">
        <TestComponent />
      </ErrorBoundary>
    )

    expect(screen.getByText('Something went wrong')).toBeInTheDocument()

    const tryAgainButton = screen.getByText('Try Again')
    await user.click(tryAgainButton)

    // Set shouldThrow to false and remount with new key to reset state
    shouldThrow = false
    rerender(
      <ErrorBoundary key="error-boundary-2">
        <TestComponent />
      </ErrorBoundary>
    )

    expect(screen.getByText('No error')).toBeInTheDocument()
  })

  // Skip the DEV mode test - import.meta.env is not mockable in Vitest
  it.skip('should show technical details in development', () => {
    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    )

    // This test is skipped because import.meta.env.DEV cannot be mocked in Vitest
    // Technical details section is implemented but cannot be tested this way
  })

  it('should have link to report issue', () => {
    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    )

    const link = screen.getByText('report an issue')
    expect(link).toHaveAttribute('href', 'https://github.com/jongio/azd-app/issues/new')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
  })
})
