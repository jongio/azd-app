import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ErrorBoundary } from '@/components/ErrorBoundary'

// Component that throws an error when rendered
function ThrowingComponent({ shouldThrow = true }: { shouldThrow?: boolean }) {
  if (shouldThrow) {
    throw new Error('Test error message')
  }
  return <div>Child component rendered successfully</div>
}

describe('ErrorBoundary', () => {
  // Suppress console.error for cleaner test output during expected errors
  const originalConsoleError = console.error
  
  beforeEach(() => {
    console.error = vi.fn()
  })
  
  afterEach(() => {
    console.error = originalConsoleError
  })

  it('should render children when there is no error', () => {
    render(
      <ErrorBoundary>
        <div>Test child content</div>
      </ErrorBoundary>
    )

    expect(screen.getByText('Test child content')).toBeInTheDocument()
  })

  it('should catch errors and display fallback UI', () => {
    render(
      <ErrorBoundary>
        <ThrowingComponent />
      </ErrorBoundary>
    )

    expect(screen.getByText('Something went wrong')).toBeInTheDocument()
    expect(screen.getByText(/An unexpected error occurred/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Try Again/i })).toBeInTheDocument()
  })

  it('should display custom fallback when provided', () => {
    const customFallback = <div>Custom error message</div>

    render(
      <ErrorBoundary fallback={customFallback}>
        <ThrowingComponent />
      </ErrorBoundary>
    )

    expect(screen.getByText('Custom error message')).toBeInTheDocument()
    expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument()
  })

  it('should log errors to console for debugging', () => {
    render(
      <ErrorBoundary>
        <ThrowingComponent />
      </ErrorBoundary>
    )

    expect(console.error).toHaveBeenCalledWith(
      'ErrorBoundary caught an error:',
      expect.any(Error)
    )
    expect(console.error).toHaveBeenCalledWith(
      'Component stack:',
      expect.any(String)
    )
  })

  it('should reset error state when Try Again button is clicked', () => {
    let shouldThrow = true
    
    function ConditionalThrower() {
      if (shouldThrow) {
        throw new Error('Test error')
      }
      return <div>Recovered successfully</div>
    }

    const { rerender } = render(
      <ErrorBoundary>
        <ConditionalThrower />
      </ErrorBoundary>
    )

    // Error should be displayed
    expect(screen.getByText('Something went wrong')).toBeInTheDocument()

    // Fix the error condition
    shouldThrow = false

    // Click Try Again
    fireEvent.click(screen.getByRole('button', { name: /Try Again/i }))

    // Component should recover
    rerender(
      <ErrorBoundary>
        <ConditionalThrower />
      </ErrorBoundary>
    )
    
    expect(screen.getByText('Recovered successfully')).toBeInTheDocument()
  })

  it('should display error details in development mode', () => {
    // In test environment, NODE_ENV is 'test' which is !== 'production'
    render(
      <ErrorBoundary>
        <ThrowingComponent />
      </ErrorBoundary>
    )

    // Should show error name and message in the error details section
    // The error message appears in both the name display and stack trace
    const errorMessages = screen.getAllByText(/Error: Test error message/)
    expect(errorMessages.length).toBeGreaterThan(0)
  })

  it('should handle errors with no stack trace', () => {
    function ThrowSimpleError(): React.ReactNode {
      const error = new Error('Simple error')
      error.stack = undefined
      throw error
    }

    render(
      <ErrorBoundary>
        <ThrowSimpleError />
      </ErrorBoundary>
    )

    expect(screen.getByText('Something went wrong')).toBeInTheDocument()
    expect(screen.getByText(/Error: Simple error/)).toBeInTheDocument()
  })

  it('should render multiple children correctly', () => {
    render(
      <ErrorBoundary>
        <div>First child</div>
        <div>Second child</div>
      </ErrorBoundary>
    )

    expect(screen.getByText('First child')).toBeInTheDocument()
    expect(screen.getByText('Second child')).toBeInTheDocument()
  })

  it('should contain the AlertTriangle icon in error state', () => {
    const { container } = render(
      <ErrorBoundary>
        <ThrowingComponent />
      </ErrorBoundary>
    )

    // Check for the icon wrapper with the destructive color class
    expect(container.querySelector('.text-destructive')).toBeInTheDocument()
  })

  it('should have proper styling on the error container', () => {
    const { container } = render(
      <ErrorBoundary>
        <ThrowingComponent />
      </ErrorBoundary>
    )

    // Check for the error boundary container styling
    const errorContainer = container.querySelector('.bg-destructive\\/10')
    expect(errorContainer).toBeInTheDocument()
  })

  it('should render RefreshCw icon in Try Again button', () => {
    render(
      <ErrorBoundary>
        <ThrowingComponent />
      </ErrorBoundary>
    )

    const button = screen.getByRole('button', { name: /Try Again/i })
    expect(button).toBeInTheDocument()
    // Button should have flex layout with icon
    expect(button.className).toContain('flex')
  })

  it('should render empty fallback when provided', () => {
    // When an empty fragment is provided as fallback, nothing is rendered
    render(
      <ErrorBoundary fallback={<></>}>
        <ThrowingComponent />
      </ErrorBoundary>
    )

    expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument()
  })

  it('should handle nested ErrorBoundaries', () => {
    render(
      <ErrorBoundary fallback={<div>Outer error</div>}>
        <div>
          <ErrorBoundary fallback={<div>Inner error</div>}>
            <ThrowingComponent />
          </ErrorBoundary>
        </div>
      </ErrorBoundary>
    )

    // Inner boundary should catch the error
    expect(screen.getByText('Inner error')).toBeInTheDocument()
    expect(screen.queryByText('Outer error')).not.toBeInTheDocument()
  })

  it('should preserve error information in state', () => {
    render(
      <ErrorBoundary>
        <ThrowingComponent />
      </ErrorBoundary>
    )

    // The error details should be visible in dev mode
    // Error message appears in both the error name display and stack trace
    const errorMessages = screen.getAllByText(/Test error message/)
    expect(errorMessages.length).toBeGreaterThan(0)
  })

  it('should work with React.ReactNode fallback types', () => {
    const complexFallback = (
      <div>
        <h1>Error occurred</h1>
        <p>Please contact support</p>
        <button>Report</button>
      </div>
    )

    render(
      <ErrorBoundary fallback={complexFallback}>
        <ThrowingComponent />
      </ErrorBoundary>
    )

    expect(screen.getByText('Error occurred')).toBeInTheDocument()
    expect(screen.getByText('Please contact support')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Report' })).toBeInTheDocument()
  })
})
