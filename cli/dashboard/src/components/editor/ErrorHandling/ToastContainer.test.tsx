/**
 * ToastContainer Tests
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ToastContainer } from './ToastContainer'
import { useToast } from './toast-system'

// Mock the toast system
vi.mock('./toast-system', () => ({
  useToast: vi.fn(),
}))

describe('ToastContainer', () => {
  const mockUseToast = useToast as ReturnType<typeof vi.fn>

  it('should render nothing when no toasts', () => {
    mockUseToast.mockReturnValue({
      toasts: [],
      dismissToast: vi.fn(),
    })

    const { container } = render(<ToastContainer />)
    expect(container.firstChild).toBeNull()
  })

  it('should render success toast', () => {
    mockUseToast.mockReturnValue({
      toasts: [
        {
          id: '1',
          type: 'success',
          message: 'Success message',
        },
      ],
      dismissToast: vi.fn(),
    })

    render(<ToastContainer />)
    expect(screen.getByText('Success message')).toBeInTheDocument()
  })

  it('should render error toast', () => {
    mockUseToast.mockReturnValue({
      toasts: [
        {
          id: '1',
          type: 'error',
          message: 'Error message',
        },
      ],
      dismissToast: vi.fn(),
    })

    render(<ToastContainer />)
    expect(screen.getByText('Error message')).toBeInTheDocument()
  })

  it('should render warning toast', () => {
    mockUseToast.mockReturnValue({
      toasts: [
        {
          id: '1',
          type: 'warning',
          message: 'Warning message',
        },
      ],
      dismissToast: vi.fn(),
    })

    render(<ToastContainer />)
    expect(screen.getByText('Warning message')).toBeInTheDocument()
  })

  it('should render info toast', () => {
    mockUseToast.mockReturnValue({
      toasts: [
        {
          id: '1',
          type: 'info',
          message: 'Info message',
        },
      ],
      dismissToast: vi.fn(),
    })

    render(<ToastContainer />)
    expect(screen.getByText('Info message')).toBeInTheDocument()
  })

  it('should render toast with description', () => {
    mockUseToast.mockReturnValue({
      toasts: [
        {
          id: '1',
          type: 'success',
          message: 'Main message',
          description: 'Additional details',
        },
      ],
      dismissToast: vi.fn(),
    })

    render(<ToastContainer />)
    expect(screen.getByText('Main message')).toBeInTheDocument()
    expect(screen.getByText('Additional details')).toBeInTheDocument()
  })

  it('should render toast with action', async () => {
    const user = userEvent.setup()
    const actionFn = vi.fn()

    mockUseToast.mockReturnValue({
      toasts: [
        {
          id: '1',
          type: 'success',
          message: 'Action toast',
          action: {
            label: 'Undo',
            onClick: actionFn,
          },
        },
      ],
      dismissToast: vi.fn(),
    })

    render(<ToastContainer />)
    const actionButton = screen.getByText('Undo')
    expect(actionButton).toBeInTheDocument()

    await user.click(actionButton)
    expect(actionFn).toHaveBeenCalledTimes(1)
  })

  it('should dismiss toast on close button click', async () => {
    const user = userEvent.setup()
    const dismissToast = vi.fn()

    mockUseToast.mockReturnValue({
      toasts: [
        {
          id: '1',
          type: 'success',
          message: 'Dismissible toast',
        },
      ],
      dismissToast,
    })

    render(<ToastContainer />)
    const closeButton = screen.getByLabelText('Dismiss notification')
    await user.click(closeButton)

    expect(dismissToast).toHaveBeenCalledWith('1')
  })

  it('should render multiple toasts', () => {
    mockUseToast.mockReturnValue({
      toasts: [
        {
          id: '1',
          type: 'success',
          message: 'First toast',
        },
        {
          id: '2',
          type: 'error',
          message: 'Second toast',
        },
      ],
      dismissToast: vi.fn(),
    })

    render(<ToastContainer />)
    expect(screen.getByText('First toast')).toBeInTheDocument()
    expect(screen.getByText('Second toast')).toBeInTheDocument()
  })

  it('should have correct ARIA attributes', () => {
    mockUseToast.mockReturnValue({
      toasts: [
        {
          id: '1',
          type: 'error',
          message: 'Error toast',
        },
      ],
      dismissToast: vi.fn(),
    })

    render(<ToastContainer />)
    const notification = screen.getByRole('alert')
    expect(notification).toHaveAttribute('aria-live', 'assertive')
  })

  it('should use polite for non-error toasts', () => {
    mockUseToast.mockReturnValue({
      toasts: [
        {
          id: '1',
          type: 'success',
          message: 'Success toast',
        },
      ],
      dismissToast: vi.fn(),
    })

    render(<ToastContainer />)
    const notification = screen.getByRole('alert')
    expect(notification).toHaveAttribute('aria-live', 'polite')
  })
})
