/**
 * ErrorModal Tests
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ErrorModal } from './ErrorModal'

describe('ErrorModal', () => {
  const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
    title: 'Error Title',
    message: 'Error message',
  }

  let clipboardSpy: ReturnType<typeof vi.fn>

  beforeEach(() => {
    // Mock clipboard API
    clipboardSpy = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', {
      ...navigator,
      clipboard: {
        writeText: clipboardSpy,
      },
    })
  })

  afterEach(() => {
    vi.clearAllMocks()
    vi.unstubAllGlobals()
  })

  it('should not render when closed', () => {
    const { container } = render(<ErrorModal {...defaultProps} isOpen={false} />)
    expect(container.firstChild).toBeNull()
  })

  it('should render when open', () => {
    render(<ErrorModal {...defaultProps} />)
    expect(screen.getByText('Error Title')).toBeInTheDocument()
    expect(screen.getByText('Error message')).toBeInTheDocument()
  })

  it('should close on backdrop click when dismissible', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()

    render(<ErrorModal {...defaultProps} onClose={onClose} dismissible={true} />)

    const backdrop = screen.getByRole('dialog')
    await user.click(backdrop)

    expect(onClose).toHaveBeenCalled()
  })

  it('should not close on backdrop click when not dismissible', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()

    render(<ErrorModal {...defaultProps} onClose={onClose} dismissible={false} />)

    const backdrop = screen.getByRole('dialog')
    await user.click(backdrop)

    expect(onClose).not.toHaveBeenCalled()
  })

  it('should show technical details on button click', async () => {
    const user = userEvent.setup()

    render(<ErrorModal {...defaultProps} technicalDetails="Stack trace here" />)

    expect(screen.queryByText('Stack trace here')).not.toBeInTheDocument()

    const showButton = screen.getByText('Show technical details')
    await user.click(showButton)

    expect(screen.getByText('Stack trace here')).toBeInTheDocument()
    expect(screen.getByText('Hide technical details')).toBeInTheDocument()
  })

  it('should copy error details to clipboard', async () => {
    const user = userEvent.setup()

    render(<ErrorModal {...defaultProps} technicalDetails="Stack trace" />)

    // Show details first
    await user.click(screen.getByText('Show technical details'))

    // Click copy button
    const copyButton = screen.getByLabelText('Copy error details')
    await user.click(copyButton)

    expect(clipboardSpy).toHaveBeenCalledWith(
      expect.stringContaining('Error Title')
    )
    expect(clipboardSpy).toHaveBeenCalledWith(
      expect.stringContaining('Stack trace')
    )

    // Should show copied indicator
    expect(screen.getByText('Copied!')).toBeInTheDocument()
  })

  it('should render action buttons', async () => {
    const user = userEvent.setup()
    const action1 = vi.fn()
    const action2 = vi.fn()
    const onClose = vi.fn()

    render(
      <ErrorModal
        {...defaultProps}
        onClose={onClose}
        actions={[
          { label: 'Retry', onClick: action1, variant: 'primary' },
          { label: 'Cancel', onClick: action2, variant: 'secondary' },
        ]}
      />
    )

    const retryButton = screen.getByText('Retry')
    const cancelButton = screen.getByText('Cancel')

    await user.click(retryButton)
    expect(action1).toHaveBeenCalled()
    expect(onClose).toHaveBeenCalled()

    vi.clearAllMocks()

    await user.click(cancelButton)
    expect(action2).toHaveBeenCalled()
  })

  it('should render default close button when no actions', () => {
    render(<ErrorModal {...defaultProps} />)
    expect(screen.getByText('Close')).toBeInTheDocument()
  })

  it('should not render close button when not dismissible and has actions', () => {
    render(
      <ErrorModal
        {...defaultProps}
        dismissible={false}
        actions={[{ label: 'Action', onClick: vi.fn() }]}
      />
    )

    expect(screen.queryByText('Close')).not.toBeInTheDocument()
  })

  it('should have correct ARIA attributes', () => {
    render(<ErrorModal {...defaultProps} />)

    const dialog = screen.getByRole('dialog')
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(dialog).toHaveAttribute('aria-labelledby', 'error-modal-title')
  })

  it('should close on Escape key when dismissible', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()

    render(<ErrorModal {...defaultProps} onClose={onClose} dismissible={true} />)

    const dialog = screen.getByRole('dialog')
    dialog.focus()
    await user.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalled()
  })
})
